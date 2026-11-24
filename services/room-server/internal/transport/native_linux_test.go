//go:build linux && cgo

// Linux 原生传输测试

package transport

import (
	"bytes"
	"net/netip"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func buildABIFixture(t *testing.T, variant string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "bridge.so")
	args := []string{"-shared", "-fPIC", "-Wall", "-Wextra", "-o", path, "testdata/bridge_fixture.c"}
	if variant != "" {
		args = append(args, "-D"+variant)
	}
	if out, err := exec.Command("cc", args...).CombinedOutput(); err != nil {
		t.Fatalf("compile ABI fixture: %v: %s", err, out)
	}
	return path
}

func TestLinuxNativeLoaderErrors(t *testing.T) {
	addr := netip.MustParseAddrPort("127.0.0.1:23456")
	for _, tc := range []struct{ name, variant, want string }{
		{"ABI", "BAD_ABI", "ABI 99"},
		{"symbol", "MISSING_SEND", "skyla_enet_send"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := OpenNative(buildABIFixture(t, tc.variant), addr, 2, 1024)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %s", err, tc.want)
			}
		})
	}
	if _, err := OpenNative(filepath.Join(t.TempDir(), "absent.so"), addr, 2, 1024); err == nil {
		t.Fatal("missing library accepted")
	}
}

func TestLinuxNativeValidation(t *testing.T) {
	for _, tc := range []struct {
		name, path, addr string
		peers, frame     int
	}{
		{"relative", "bridge.so", "127.0.0.1:23456", 2, 1024},
		{"ipv6", "/missing.so", "[::1]:23456", 2, 1024},
		{"zero peers", "/missing.so", "127.0.0.1:23456", 0, 1024},
		{"excess peers", "/missing.so", "127.0.0.1:23456", 4096, 1024},
		{"zero frame", "/missing.so", "127.0.0.1:23456", 2, 0},
		{"excess frame", "/missing.so", "127.0.0.1:23456", 2, 32<<20 + 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := OpenNative(tc.path, netip.MustParseAddrPort(tc.addr), tc.peers, tc.frame)
			if err == nil || strings.Contains(err.Error(), "load SkyENet") {
				t.Fatalf("validation error = %v", err)
			}
		})
	}
}

func TestLinuxNativePacketOwnershipAndLifecycle(t *testing.T) {
	path := buildABIFixture(t, "")
	addr := netip.MustParseAddrPort("127.0.0.1:23456")
	if _, err := OpenNative(path, netip.MustParseAddrPort("127.0.0.1:23457"), 2, 1024); err == nil {
		t.Fatal("native open failure ignored")
	}
	host, err := OpenNative(path, addr, 2, 1024)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = host.Close() })
	if _, ok, err := host.Poll(time.Millisecond); ok || err != nil {
		t.Fatalf("idle = %v, %v", ok, err)
	}
	for _, timeout := range []time.Duration{-1, time.Duration(1<<32) * time.Millisecond} {
		if _, _, err := host.Poll(timeout); err == nil {
			t.Fatal("invalid timeout accepted")
		}
	}
	payload := []byte{0, 1, 255}
	if err := host.Send(7, 1, true, payload); err != nil {
		t.Fatal(err)
	}
	first, ok, err := host.Poll(0)
	if err != nil || !ok || first.Kind != Received || first.Peer != 7 || first.Channel != 1 || !first.Reliable || !bytes.Equal(first.Data, payload) {
		t.Fatalf("receive = %+v, %v, %v", first, ok, err)
	}
	if err := host.Send(7, 0, false, []byte{9, 8, 7}); err != nil {
		t.Fatal(err)
	}
	second, ok, err := host.Poll(0)
	if err != nil || !ok || second.Channel != 0 || second.Reliable || !bytes.Equal(second.Data, []byte{9, 8, 7}) || !bytes.Equal(first.Data, payload) {
		t.Fatalf("ownership/metadata: first=%+v second=%+v err=%v", first, second, err)
	}
	if err := host.Send(7, 0, false, nil); err != nil {
		t.Fatal(err)
	}
	empty, ok, err := host.Poll(0)
	if err != nil || !ok || len(empty.Data) != 0 {
		t.Fatalf("empty packet = %+v %v %v", empty, ok, err)
	}
	if err := host.Send(8, 0, false, payload); err == nil {
		t.Fatal("native send failure ignored")
	}
	if err := host.Send(7, 0, false, payload); err != nil {
		t.Fatal(err)
	}
	host.Disconnect(7, 42)
	if _, ok, err := host.Poll(0); ok || err != nil {
		t.Fatalf("disconnect did not reach native host: %v %v", ok, err)
	}
	if err := host.Close(); err != nil {
		t.Fatal(err)
	}
	if err := host.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := host.Poll(0); err == nil {
		t.Fatal("closed poll accepted")
	}
	if err := host.Send(7, 0, true, payload); err == nil {
		t.Fatal("closed send accepted")
	}
	host.Disconnect(7, 0)
}

func TestLinuxNativeMalformedPoll(t *testing.T) {
	for _, variant := range []string{"BAD_LENGTH", "BAD_KIND"} {
		t.Run(variant, func(t *testing.T) {
			host, err := OpenNative(buildABIFixture(t, variant), netip.MustParseAddrPort("127.0.0.1:23456"), 2, 1024)
			if err != nil {
				t.Fatal(err)
			}
			defer host.Close()
			if _, _, err := host.Poll(0); err == nil {
				t.Fatal("invalid native result accepted")
			}
		})
	}
}
