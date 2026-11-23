package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validConfig = `{
  "udp_listen": "0.0.0.0:25565",
  "management_listen": "127.0.0.1:8080",
  "max_peers": 32,
  "max_players": 8,
  "join_timeout_ms": 6000,
  "max_frame_bytes": 16384
}`

func writeConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "realtime.json")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad(t *testing.T) {
	cfg, err := Load(writeConfig(t, validConfig))
	if err != nil {
		t.Fatal(err)
	}
	want := Config{
		UDPListen:        "0.0.0.0:25565",
		ManagementListen: "127.0.0.1:8080",
		MaxPeers:         32,
		MaxPlayers:       8,
		JoinTimeoutMS:    6000,
		MaxFrameBytes:    16384,
	}
	if cfg != want {
		t.Fatalf("configuration = %+v, want %+v", cfg, want)
	}
}

func TestLoadRejectsInvalidConfiguration(t *testing.T) {
	for _, test := range []struct {
		name string
		old  string
		new  string
		want string
	}{
		{"invalid-udp-address", "0.0.0.0:25565", "0.0.0.0", "udp_listen"},
		{"zero-udp-port", "0.0.0.0:25565", "0.0.0.0:0", "udp_listen"},
		{"public-management", "127.0.0.1:8080", "0.0.0.0:8080", "management_listen"},
		{"zero-peers", "32", "0", "max_peers"},
		{"too-many-players", "\"max_players\": 8", "\"max_players\": 33", "max_players"},
		{"zero-timeout", "6000", "0", "join_timeout_ms"},
		{"zero-frame-size", "16384", "0", "max_frame_bytes"},
		{"unknown-key", "udp_listen", "udp_address", "unknown field"},
		{"trailing-object", "\n}", "\n} {}", "after"},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := strings.Replace(validConfig, test.old, test.new, 1)
			_, err := Load(writeConfig(t, body))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}
