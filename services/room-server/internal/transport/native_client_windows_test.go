//go:build windows && amd64

package transport

import (
	"net/netip"
	"os"
	"testing"
	"time"

	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/protocol"
)

func TestNativeLegacyClientJoin(t *testing.T) {
	path := os.Getenv("SKYLA_ENET_BRIDGE_TEST_DLL")
	target := os.Getenv("SKYLA_ENET_TEST_REAL_ADDR")
	if path == "" || target == "" {
		t.Skip("requires a manually prepared 0.1.8 test client and advertised address")
	}
	addr, err := netip.ParseAddrPort(target)
	if err != nil {
		t.Fatal(err)
	}
	host, err := OpenNative(path, addr, 8, 16384)
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	t.Logf("listening on %s", addr)

	connected := make(map[uint64]struct{})
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		event, ok, err := host.Poll(50 * time.Millisecond)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			continue
		}
		switch event.Kind {
		case Connected:
			connected[event.Peer] = struct{}{}
			t.Log("ENet connect")
		case Disconnected:
			delete(connected, event.Peer)
		case Received:
			if _, ok := connected[event.Peer]; !ok || len(event.Data) == 0 || event.Data[0] != 3 {
				continue
			}
			join, err := protocol.DecodeJoinGameFingerprint(event.Data)
			if err != nil {
				t.Fatal(err)
			}
			if join.Engine != "sky0134" || join.ServerVersion != 284 || join.ClientVersion != 542 {
				t.Fatalf("unexpected profile %s/%d/%d", join.Engine, join.ServerVersion, join.ClientVersion)
			}
			t.Logf("JoinGame engine=%s version=%d/%d build=%d channel=%d reliable=%t bytes=%d",
				join.Engine, join.ServerVersion, join.ClientVersion, join.Build,
				event.Channel, event.Reliable, len(event.Data))
			return
		}
	}
	t.Fatal("no compatible JoinGame arrived during observation window")
}
