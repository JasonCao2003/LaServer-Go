package panel

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	for _, tc := range []struct {
		name, extra, api string
		missing          string
		valid            bool
	}{
		{"defaults", "", "http://127.0.0.1:8443", "", true},
		{"https", "", "https://api.example.org", "", true},
		{"bad timeout", `,"upstream_timeout_ms":0`, "http://127.0.0.1", "", false},
		{"wildcard", `,"listen":"0.0.0.0:18080"`, "http://127.0.0.1", "", false},
		{"remote http", "", "http://api.example.org", "", false},
		{"userinfo", "", "https://a:b@api.example.org", "", false},
		{"path", "", "https://api.example.org/admin", "", false},
		{"query", "", "https://api.example.org?x=1", "", false},
		{"fragment", "", "https://api.example.org#admin", "", false},
		{"no password", "", "http://127.0.0.1", "LASERVER_PANEL_PASSWORD", false},
		{"no token", "", "http://127.0.0.1", "LASERVER_PANEL_API_TOKEN", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, key := range []string{"LASERVER_PANEL_PASSWORD", "LASERVER_PANEL_API_TOKEN"} {
				value := "test-" + key
				if key == tc.missing {
					value = ""
				}
				t.Setenv(key, value)
			}
			file := filepath.Join(t.TempDir(), "panel.json")
			if err := os.WriteFile(file, []byte(`{"api_url":"`+tc.api+`","room_url":"http://[::1]:8080"`+tc.extra+`}`), 0600); err != nil {
				t.Fatal(err)
			}
			cfg, err := LoadConfig(file)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v", tc.valid, err)
			}
			if tc.valid && (cfg.Listen != "127.0.0.1:18080" || cfg.UpstreamTimeout != 2*time.Second) {
				t.Fatal("unexpected defaults")
			}
		})
	}
}
