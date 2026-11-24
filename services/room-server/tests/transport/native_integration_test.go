//go:build (windows && amd64) || (linux && cgo)

// 原生传输集成测试

package transport_test

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/transport"
)

func TestNativeOpenAndPoll(t *testing.T) {
	path := nativeIntegrationLibrary()
	if path == "" {
		t.Skip("set SKYLA_ENET_BRIDGE_TEST_LIBRARY to the built native bridge")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	host, err := transport.OpenNative(absolute, netip.MustParseAddrPort("127.0.0.1:0"), 2, 1024)
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	event, ok, err := host.Poll(time.Millisecond)
	if err != nil || ok {
		t.Fatalf("idle poll = %+v, %v, %v", event, ok, err)
	}
	if err := host.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := host.Poll(time.Millisecond); err == nil {
		t.Fatal("poll on closed host succeeded")
	}
}

func TestNativeRoundTripWithSkyENet(t *testing.T) {
	path := nativeIntegrationLibrary()
	module := os.Getenv("SKYLA_ENET_TEST_NODE_MODULE")
	if path == "" || module == "" {
		t.Skip("set the native bridge and Node sky-enet module paths")
	}

	reservation, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	address := reservation.LocalAddr().(*net.UDPAddr).AddrPort().Addr().Unmap()
	port := reservation.LocalAddr().(*net.UDPAddr).Port
	if err := reservation.Close(); err != nil {
		t.Fatal(err)
	}

	host, err := transport.OpenNative(path, netip.AddrPortFrom(address, uint16(port)), 2, 8192)
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	const script = `
const assert = require('node:assert/strict');
const { once } = require('node:events');
const { Client } = require(process.env.SKYLA_ENET_TEST_NODE_MODULE);
const client = new Client({ ip: '127.0.0.1', port: +process.argv[1], checksum: true, channelLimit: 2 });
(async () => {
    try {
        await client.connect({ timeoutMs: 2500 });
        for (const [id, reliable, size] of [[0, true, 4096], [1, false, 9]]) {
            const payload = Buffer.alloc(size, id + 1);
            const next = once(client, 'receive');
            assert.equal(client.send(id, payload, reliable), 0);
            client.flush();
            const [event] = await next;
            assert.equal(event.channelID, id);
            assert.deepEqual(event.data, payload);
        }
        const disconnected = once(client, 'disconnect');
        client.disconnect(7);
        client.flush();
        await disconnected;
    } finally {
        client.stop();
        client.destroy();
    }
})().catch(err => { console.error(err); process.exitCode = 1; });
`
	cmd := exec.CommandContext(ctx, "node", "--eval", script, fmt.Sprint(port))
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if cmd.ProcessState == nil {
			cancel()
			_ = cmd.Wait()
		}
	}()

	var state int
	for ctx.Err() == nil {
		event, ok, err := host.Poll(25 * time.Millisecond)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			continue
		}
		switch event.Kind {
		case transport.Connected:
			if state != 0 || event.Channel != 0 {
				t.Fatalf("unexpected connect in state %d on channel %d", state, event.Channel)
			}
			state++
		case transport.Received:
			if state != 1 && state != 2 {
				t.Fatalf("unexpected receive in state %d", state)
			}
			channel := uint8(state - 1)
			size := 4096
			if channel == 1 {
				size = 9
			}
			if event.Channel != channel || event.Reliable != (channel == 0) ||
				!bytes.Equal(event.Data, bytes.Repeat([]byte{channel + 1}, size)) {
				t.Fatalf("unexpected receive on channel %d, reliable=%v, size=%d", event.Channel, event.Reliable, len(event.Data))
			}
			if err := host.Send(event.Peer, event.Channel, event.Reliable, event.Data); err != nil {
				t.Fatal(err)
			}
			state++
		case transport.Disconnected:
			if state != 3 || event.Channel != 0 {
				t.Fatalf("disconnected in state %d on channel %d", state, event.Channel)
			}
			if err := cmd.Wait(); err != nil {
				t.Fatalf("client error = %v, output = %s", err, output.String())
			}
			return
		}
	}
	_ = cmd.Wait()
	t.Fatalf("SkyENet client timeout in state %d, output = %s", state, output.String())
}

func nativeIntegrationLibrary() string {
	if path := os.Getenv("SKYLA_ENET_BRIDGE_TEST_LIBRARY"); path != "" {
		return path
	}
	return os.Getenv("SKYLA_ENET_BRIDGE_TEST_DLL")
}
