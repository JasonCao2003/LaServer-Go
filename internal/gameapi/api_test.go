package gameapi

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func newAPI(t *testing.T, authorized bool) (*API, *sql.DB) {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	api, err := New(db, Config{AutoAuthorize: authorized, RoomHost: "192.168.1.10", RoomPort: 25565})
	if err != nil {
		t.Fatal(err)
	}
	return api, db
}

func call(t *testing.T, h http.Handler, path string, body any, want int) map[string]any {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("User-Agent", "Sky-Live-ios/0.3.11.100")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != want {
		t.Fatalf("%s: status %d want %d: %s", path, w.Code, want, w.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func create(t *testing.T, h http.Handler) map[string]any {
	return call(t, h, "/account/create", map[string]any{"device_name": "test-device", "language": "zh"}, 200)
}

func TestLegacyLoginRotatesSession(t *testing.T) {
	api, _ := newAPI(t, true)
	a := create(t, api)
	if len(a["key"].(string)) != 64 {
		t.Fatal("invalid login key")
	}
	b := call(t, api, "/account/login", map[string]any{"device": a["device"], "key": a["key"], "language": "zh"}, 200)
	if a["user"] != b["user"] || a["session"] == b["session"] {
		t.Fatal("identity or session rotation broken")
	}
	call(t, api, "/account/get_currency", map[string]any{"user": a["user"], "session": a["session"]}, 401)
	r := call(t, api, "/account/get_currency", map[string]any{"user": b["user"], "session": b["session"]}, 200)
	if r["currency"].(map[string]any)["wax"] != float64(3) {
		t.Fatalf("currency default: %v", r)
	}
}

func TestAuthorizationAndBan(t *testing.T) {
	api, db := newAPI(t, false)
	a := create(t, api)
	req := map[string]any{"user": a["user"]}
	call(t, api, "/account/login", req, 406)
	if _, err := db.Exec("UPDATE user SET is_authorized=1,is_banned=1,ban_end_time=0 WHERE id=?", a["user"]); err != nil {
		t.Fatal(err)
	}
	call(t, api, "/account/login", req, 406)
	if _, err := db.Exec("UPDATE user SET ban_end_time=1 WHERE id=?", a["user"]); err != nil {
		t.Fatal(err)
	}
	call(t, api, "/account/login", req, 200)
}

func TestUserdataPersistenceAndSessionIsolation(t *testing.T) {
	api, db := newAPI(t, true)
	a := create(t, api)
	b := create(t, api)
	req := map[string]any{"user": a["user"], "session": a["session"], "data": "eyJ4IjoxfQ==", "time": 123, "version": 3}
	call(t, api, "/account/sync_user_data", req, 200)
	req["session"] = b["session"]
	call(t, api, "/account/userdata", req, 401)
	req["session"] = a["session"]
	req["data"] = ""
	api2, err := New(db, Config{RoomHost: "192.168.1.10", RoomPort: 25565})
	if err != nil {
		t.Fatal(err)
	}
	out := call(t, api2, "/account/userdata", req, 200)["userdata"].(map[string]any)
	if out["data"] != "eyJ4IjoxfQ==" || out["time"] != float64(123) {
		t.Fatalf("saved data overwritten: %v", out)
	}
}

func TestRoomDiscoveryWithoutRealtime(t *testing.T) {
	api, _ := newAPI(t, true)
	a := create(t, api)
	out := call(t, api, "/account/find_previous_or_empty", map[string]any{"user": a["user"], "session": a["session"]}, 200)
	if out["uri"] != "192.168.1.10:25565" {
		t.Fatalf("wrong room: %v", out)
	}
	if _, err := hex.DecodeString(out["signature"].(string)); err != nil {
		t.Fatal(err)
	}
}

func TestMalformedAndUnavailableRoutes(t *testing.T) {
	api, _ := newAPI(t, true)
	for _, body := range []string{"", "null", "[]", "{", "{} {}", `{"device_name":12}`} {
		w := httptest.NewRecorder()
		api.ServeHTTP(w, httptest.NewRequest("POST", "/account/create", strings.NewReader(body)))
		if w.Code != 400 {
			t.Fatalf("body %q status %d: %s", body, w.Code, w.Body.String())
		}
	}
	call(t, api, "/account/purchase_spirit_shop_item", map[string]any{}, 501)
	call(t, api, "/not-a-route", map[string]any{}, 404)
}

func TestExplicitMigrationAndOldDatabaseProtection(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "empty.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = New(db, Config{RoomHost: "127.0.0.1", RoomPort: 25565}); err == nil {
		t.Fatal("uninitialized DB accepted")
	}
	if _, err = db.Exec("CREATE TABLE user(id TEXT PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	if err = Migrate(context.Background(), db); err == nil {
		t.Fatal("legacy DB was silently modified")
	}
}

func TestVIPDailyCurrencyBonusIsAtomic(t *testing.T) {
	api, db := newAPI(t, true)
	a := create(t, api)
	if _, err := db.Exec("UPDATE user_currency SET vip=1 WHERE user_id=?", a["user"]); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/account/get_currency", strings.NewReader(`{"user":"`+a["user"].(string)+`","session":"`+a["session"].(string)+`"}`))
			w := httptest.NewRecorder()
			api.ServeHTTP(w, req)
			if w.Code != 200 {
				t.Errorf("currency: %d %s", w.Code, w.Body.String())
			}
		}()
	}
	wg.Wait()
	var candles int
	if err := db.QueryRow("SELECT season_candle FROM user_currency WHERE user_id=?", a["user"]).Scan(&candles); err != nil {
		t.Fatal(err)
	}
	if candles != 1 {
		t.Fatalf("got %d VIP daily candles, want exactly 1", candles)
	}
}

func TestNumericCheckpointUpdatesAndFurthestProgress(t *testing.T) {
	api, _ := newAPI(t, true)
	a := create(t, api)
	req := map[string]any{"user": a["user"], "session": a["session"], "level_id": 3526133726, "level_hash": 10, "bst_id": 1, "states": []int{1}}
	call(t, api, "/account/set_checkpoint", req, 200)
	req["level_id"] = 3928396665
	call(t, api, "/account/set_checkpoint", req, 200)
	req["bst_id"] = 2
	result := call(t, api, "/account/set_checkpoint", req, 200)
	points := result["checkpoints"].([]any)
	if len(points) != 2 {
		t.Fatalf("same level duplicated: %v", points)
	}
	first := points[0].(map[string]any)
	second := points[1].(map[string]any)
	if first["arc_order"] != float64(2) || first["is_furthest"] != true || second["is_furthest"] != false || first["bst_id"] != float64(2) {
		t.Fatalf("progress lost: %v", points)
	}
}

func TestBootstrapVersionAndDeviceCapabilities(t *testing.T) {
	api, _ := newAPI(t, true)
	req := httptest.NewRequest("POST", "/account/get_latest_build_version", strings.NewReader("{}"))
	req.Header.Set("User-Agent", "Sky-Live-ios/0.13.4.170388")
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("build endpoint: %d %s", w.Code, w.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["latest_build_version"].(float64) < 170388 {
		t.Fatalf("client would reject old build: %v", result)
	}
	result = call(t, api, "/account/device_capabilities/lookup", map[string]any{}, 200)
	if result["device_capabilities"].(map[string]any)["gpu_rating"] != float64(10003) {
		t.Fatalf("capabilities: %v", result)
	}
	call(t, api, "/account/get_vars", map[string]any{}, 200)
}
