package gameapi

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigPathsAndInvalidValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.json")
	good := `{"listen":"127.0.0.1:0","database":"data/test.db","room_host":"127.0.0.1","room_port":25565}`
	if err := os.WriteFile(path, []byte(good), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database != filepath.Join(dir, "data/test.db") {
		t.Fatalf("relative database path: %s", cfg.Database)
	}
	for _, bad := range []string{`{}`, good + ` {}`, strings.Replace(good, `"database"`, `"unknown"`, 1), strings.Replace(good, `"room_port":25565`, `"room_port":25565,"tls_cert":"cert.pem"`, 1)} {
		if err := os.WriteFile(path, []byte(bad), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadConfig(path); err == nil {
			t.Fatalf("invalid config accepted: %s", bad)
		}
	}
}

func TestBodyLimitAndMethod(t *testing.T) {
	api, _ := newAPI(t, true)
	for _, tc := range []struct {
		method, body string
		status       int
	}{{"GET", "{}", 405}, {"POST", strings.Repeat(" ", 2<<20) + "{}", 413}} {
		req := httptest.NewRequest(tc.method, "/account/create", strings.NewReader(tc.body))
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)
		if w.Code != tc.status {
			t.Fatalf("status %d want %d", w.Code, tc.status)
		}
	}
}

func TestAuthCreateAndLoginEnvelope(t *testing.T) {
	api, _ := newAPI(t, true)
	a := call(t, api, "/account/auth/create", map[string]any{"type": "device", "device_name": "test", "device_token": "", "production": false, "tos_version": 4, "device_key": "", "sig_ts": 0, "sig": "", "hashes": []string{}}, 200)
	auth := a["authinfo"].(map[string]any)
	b := call(t, api, "/account/auth/login", map[string]any{"user": auth["user"], "device": auth["device"]}, 200)
	next := b["authinfo"].(map[string]any)
	if auth["user"] != next["user"] || len(next["key"].(string)) != 64 || a["session"] == b["session"] {
		t.Fatalf("auth envelope: %v", b)
	}
	if b["websocket_auth"] == "" {
		t.Fatal("missing websocket authorization")
	}
}
