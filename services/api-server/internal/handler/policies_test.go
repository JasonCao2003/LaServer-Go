package handler

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAutoAuthorizationHonorsLegacySince(t *testing.T) {
	api, _ := newAPI(t, true)
	dir := t.TempDir()
	api.cfg.GameConfigDir = dir
	if err := os.WriteFile(filepath.Join(dir, "auto_auth.json"), []byte(`{"enabled":true,"since":4102444800}`), 0600); err != nil {
		t.Fatal(err)
	}
	a := create(t, api)
	call(t, api, "/account/login", map[string]any{"user": a["user"]}, 406)
	if err := os.WriteFile(filepath.Join(dir, "auto_auth.json"), []byte(`{"enabled":true,"since":0}`), 0600); err != nil {
		t.Fatal(err)
	}
	b := create(t, api)
	call(t, api, "/account/login", map[string]any{"user": b["user"]}, 200)
	// Enabling new-account authorization must not authorize older rows retroactively.
	call(t, api, "/account/login", map[string]any{"user": a["user"]}, 406)
}

func TestMalformedAuthorizationPolicyFailsClosed(t *testing.T) {
	api, _ := newAPI(t, true)
	dir := t.TempDir()
	api.cfg.GameConfigDir = dir
	if err := os.WriteFile(filepath.Join(dir, "auto_auth.json"), []byte(`{"enabled":true,"since":"oops"}`), 0600); err != nil {
		t.Fatal(err)
	}
	a := create(t, api)
	call(t, api, "/account/login", map[string]any{"user": a["user"]}, 406)
}

func TestMOTDUsesLegacyFixedAndLiveTime(t *testing.T) {
	api, _ := newAPI(t, true)
	dir := t.TempDir()
	api.cfg.GameConfigDir = dir
	p := filepath.Join(dir, "motd_config.json")
	if err := os.WriteFile(p, []byte(`{"motd":"hello","motd_title":"test","motd_timestamp_mode":"fixed","motd_timestamp":123}`), 0600); err != nil {
		t.Fatal(err)
	}
	out := call(t, api, "/account/get_motd", map[string]any{}, 200)
	if out["motd_timestamp"] != float64(123) || out["motd"] != "hello" || out["motd_season_start"] != float64(123) {
		t.Fatalf("fixed MOTD: %v", out)
	}
	if _, ok := out["motd_timestamp_mode"]; ok {
		t.Fatal("configuration-only field leaked")
	}
	if err := os.WriteFile(p, []byte(`{"motd_timestamp_mode":"now","motd_timestamp":123}`), 0600); err != nil {
		t.Fatal(err)
	}
	out = call(t, api, "/account/get_motd", map[string]any{}, 200)
	if out["motd_timestamp"].(float64) <= 123 {
		t.Fatal("stale live timestamp")
	}
}

func TestFullCollectiblesOnlyForEligibleAccounts(t *testing.T) {
	api, db := newAPI(t, true)
	a := create(t, api)
	b := create(t, api)
	dir := t.TempDir()
	api.cfg.GameConfigDir = dir
	files := map[string]string{"collectibles.json": `{"collectibles":[{"id":"wave","level":1}]}`, "quandongzuo.json": `{"collectibles":[{"id":"dance","level":4}]}`, "auto_full_unlock.json": `{"enabled":true,"since":100}`}
	for name, value := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec("UPDATE user SET created_at=1 WHERE id=?", a["user"]); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE user SET created_at=100 WHERE id=?", b["user"]); err != nil {
		t.Fatal(err)
	}
	old := call(t, api, "/account/get_collectibles", map[string]any{"user": a["user"], "session": a["session"]}, 200)
	if len(old["collectibles"].([]any)) != 1 {
		t.Fatal("old account received automatic full actions")
	}
	eligible := call(t, api, "/account/get_collectibles", map[string]any{"user": b["user"], "session": b["session"]}, 200)
	if len(eligible["collectibles"].([]any)) != 2 {
		t.Fatalf("eligible account missing actions: %v", eligible)
	}
	public := call(t, api, "/account/get_collectibles", map[string]any{}, 200)
	if len(public["collectibles"].([]any)) != 1 {
		t.Fatal("public response received automatic full actions")
	}
	if err := os.Remove(filepath.Join(dir, "quandongzuo.json")); err != nil {
		t.Fatal(err)
	}
	fallback := call(t, api, "/account/get_collectibles", map[string]any{"user": b["user"], "session": b["session"]}, 200)
	if len(fallback["collectibles"].([]any)) != 1 {
		t.Fatal("optional full template failure blocked basic actions")
	}
}

func TestVersionOverrideControlsLegacyResponse(t *testing.T) {
	api, _ := newAPI(t, true)
	dir := t.TempDir()
	api.cfg.GameConfigDir = dir
	if err := os.WriteFile(filepath.Join(dir, "client_version_overrides.json"), []byte(`{"by_full":{"0.23.6.354648":"0.3.11"},"by_build":{"354648":"0.22.0"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/account/create", strings.NewReader(`{"device_name":"test","language":"zh"}`))
	req.Header.Set("User-Agent", "Sky-Live-ios/0.23.6.354648")
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || out["player_badge_type"] != float64(-1) {
		t.Fatalf("full override not applied: %d %v", w.Code, out)
	}
}
