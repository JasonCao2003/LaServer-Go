package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"testing"
)

func dCall(t *testing.T, api *API, method, path, remote, token string, body map[string]any, want int) map[string]any {
	t.Helper()
	var payload []byte
	if method == "GET" {
		query := url.Values{}
		for key, value := range body {
			query.Set(key, fmt.Sprint(value))
		}
		path += "?" + query.Encode()
	} else {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.RemoteAddr = remote
	req.Header.Set("Content-Type", "text/plain")
	if token != "" {
		req.Header.Set("X-Admin-Token", token)
	}
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)
	if w.Code != want {
		t.Fatalf("%s %s: status %d want %d: %s", method, path, w.Code, want, w.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestBatchDStageOwnershipSequenceAndPlaceholders(t *testing.T) {
	api, db := newAPI(t, true)
	a, b := create(t, api), create(t, api)
	base := map[string]any{"user": a["user"], "session": a["session"], "user_id": a["user"], "stage_id": "test-stage", "level_id": "test-level", "props": []any{map[string]any{"name": "chair", "pos": "(1,2,3)"}}}
	call(t, api, "/service/stage/api/v1/set", map[string]any{"user": a["user"], "session": b["session"], "stage_id": "test-stage", "props": []any{}}, 401)
	call(t, api, "/service/stage/api/v1/set", map[string]any{"user": a["user"], "session": a["session"], "user_id": b["user"], "stage_id": "test-stage", "props": []any{}}, 403)
	call(t, api, "/service/stage/api/v1/set", map[string]any{"user": a["user"], "session": a["session"], "stage_id": "test-stage"}, 400)
	call(t, api, "/service/stage/api/v1/set", base, 200)
	read := call(t, api, "/service/stage/api/v1/get", map[string]any{"user": b["user"], "user_id": a["user"], "stage_id": "test-stage"}, 200)
	if read["status"] != "OK" || read["sequence"] != float64(1) || len(read["props"].([]any)) != 1 {
		t.Fatal(read)
	}
	call(t, api, "/service/stage/api/v1/set", base, 200)
	read = call(t, api, "/service/stage/api/v1/get", map[string]any{"user_id": a["user"], "stage_id": "test-stage"}, 200)
	if read["sequence"] != float64(2) {
		t.Fatal(read)
	}
	base["sequence"] = 9
	call(t, api, "/service/stage/api/v1/set", base, 200)
	read = call(t, api, "/service/stage/api/v1/get", map[string]any{"user_id": a["user"], "stage_id": "test-stage"}, 200)
	if read["sequence"] != float64(9) {
		t.Fatal(read)
	}
	if n := dCount(t, db, "SELECT COUNT(*) FROM stage_state WHERE stage_id='test-stage' AND owner_id=?", a["user"]); n != 1 {
		t.Fatal("stage upsert made duplicate rows")
	}
	other := call(t, api, "/service/stage/api/v1/get", map[string]any{"user_id": b["user"], "stage_id": "test-stage"}, 200)
	if len(other["props"].([]any)) != 0 {
		t.Fatal("stage leaked to another owner", other)
	}
	if out := call(t, api, "/service/reservation/api/v1/reserve", map[string]any{}, 200); out["room_ip"] != "192.168.1.10" || out["room_port"] != float64(25565) {
		t.Fatal(out)
	}
	if out := call(t, api, "/service/stage/api/v1/friends_in_apartments", map[string]any{}, 200); len(out["friends"].([]any)) != 0 {
		t.Fatal(out)
	}
	for _, path := range []string{"/service/stage/api/v1/leave_apartment", "/service/stage/api/v1/update_apartment_status"} {
		if out := call(t, api, path, map[string]any{}, 200); out["result"] != "success" {
			t.Fatal(path, out)
		}
	}
}

func dCount(t *testing.T, db interface{ QueryRow(string, ...any) *sql.Row }, query string, args ...any) int {
	t.Helper()
	var count int
	if err := db.QueryRow(query, args...).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func TestBatchDStageWriteFailurePreservesStoredState(t *testing.T) {
	api, db := newAPI(t, true)
	a := create(t, api)
	request := map[string]any{"user": a["user"], "session": a["session"], "stage_id": "failure-stage", "props": []any{map[string]any{"name": "old"}}}
	call(t, api, "/service/stage/api/v1/set", request, 200)
	if _, err := db.Exec(`CREATE TRIGGER stage_test_failure BEFORE UPDATE ON stage_state BEGIN SELECT RAISE(ABORT,'stage write failure'); END`); err != nil {
		t.Fatal(err)
	}
	request["props"] = []any{map[string]any{"name": "new"}}
	call(t, api, "/service/stage/api/v1/set", request, 500)
	read := call(t, api, "/service/stage/api/v1/get", map[string]any{"user_id": a["user"], "stage_id": "failure-stage"}, 200)
	if read["sequence"] != float64(1) || read["props"].([]any)[0].(map[string]any)["name"] != "old" {
		t.Fatal(read)
	}
}

func TestBatchDStageReadStorageFailureIsReported(t *testing.T) {
	api, db := newAPI(t, true)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	call(t, api, "/service/stage/api/v1/get", map[string]any{"stage_id": "unavailable", "user_id": "test"}, 500)
}

func TestBatchDAdminAccessMethodsAndFriendLifecycle(t *testing.T) {
	api, db := newAPI(t, true)
	a, b := create(t, api), create(t, api)
	for _, path := range []string{"/admin/api/friends_list", "/admin/api/friends_set", "/admin/api/friends_clear"} {
		dCall(t, api, "POST", path, "198.51.100.10:1234", "", map[string]any{"user": a["user"], "friend": b["user"]}, 403)
		dCall(t, api, "GET", path, "198.51.100.10:1234", "", map[string]any{"user": a["user"], "friend": b["user"]}, 403)
	}
	admin, err := New(db, Config{AutoAuthorize: true, RoomHost: "192.168.1.10", RoomPort: 25565, AdminToken: "test-admin-token"})
	if err != nil {
		t.Fatal(err)
	}
	set := map[string]any{"user": a["user"], "friend": b["user"], "action": "add"}
	dCall(t, admin, "POST", "/admin/api/friends_set", "127.0.0.1:1234", "wrong", set, 403)
	if out := dCall(t, admin, "POST", "/admin/api/friends_set", "198.51.100.10:1234", "test-admin-token", set, 200); out["ok"] != true {
		t.Fatal(out)
	}
	if n := dCount(t, db, "SELECT COUNT(*) FROM friendship WHERE status='active' AND ((user_id=? AND friend_id=?) OR (user_id=? AND friend_id=?))", a["user"], b["user"], b["user"], a["user"]); n != 2 {
		t.Fatal("admin add not bidirectional")
	}
	for _, method := range []string{"GET", "POST"} {
		out := dCall(t, admin, method, "/admin/api/friends_list", "198.51.100.10:1234", "test-admin-token", map[string]any{"user": a["user"]}, 200)
		if out["ok"] != true || out["count"] != float64(1) {
			t.Fatal(method, out)
		}
	}
	remove := map[string]any{"user": a["user"], "friend": b["user"], "action": "remove"}
	if out := dCall(t, admin, "GET", "/admin/api/friends_set", "198.51.100.10:1234", "test-admin-token", remove, 200); out["ok"] != true {
		t.Fatal(out)
	}
	if n := dCount(t, db, "SELECT COUNT(*) FROM friendship WHERE status='removed' AND ((user_id=? AND friend_id=?) OR (user_id=? AND friend_id=?))", a["user"], b["user"], b["user"], a["user"]); n != 2 {
		t.Fatal("admin remove not bidirectional")
	}
	set["action"] = "restore"
	set["admin_token"] = "test-admin-token"
	if out := dCall(t, admin, "POST", "/admin/api/friends_set", "198.51.100.10:1234", "", set, 200); out["ok"] != true {
		t.Fatal(out)
	}
	if out := dCall(t, admin, "GET", "/admin/api/friends_clear", "198.51.100.10:1234", "test-admin-token", map[string]any{"user": a["user"]}, 200); out["removed_count"] != float64(1) {
		t.Fatal(out)
	}
	if out := dCall(t, admin, "POST", "/admin/api/friends_clear", "198.51.100.10:1234", "test-admin-token", map[string]any{"user": a["user"]}, 200); out["ok"] != true {
		t.Fatal(out)
	}
	if n := dCount(t, db, "SELECT COUNT(*) FROM admin_audit WHERE user_id=?", a["user"]); n < 4 {
		t.Fatalf("admin audit rows=%d", n)
	}
}

func TestBatchDAdminLoopbackAndValidation(t *testing.T) {
	api, db := newAPI(t, true)
	a, b := create(t, api), create(t, api)
	if out := dCall(t, api, "POST", "/admin/api/friends_set", "127.0.0.1:1234", "", map[string]any{"user": a["user"], "friend": a["user"]}, 200); out["ok"] != false {
		t.Fatal(out)
	}
	if out := dCall(t, api, "POST", "/admin/api/friends_set", "[::1]:1234", "", map[string]any{"user": a["user"], "friend": b["user"], "action": "unknown"}, 200); out["ok"] != false {
		t.Fatal(out)
	}
	if out := dCall(t, api, "POST", "/admin/api/friends_set", "127.0.0.1:1234", "", map[string]any{"user": a["user"], "friend": "missing-user"}, 200); out["ok"] != false {
		t.Fatal(out)
	}
	if n := dCount(t, db, "SELECT COUNT(*) FROM admin_audit"); n != 0 {
		t.Fatal("invalid actions were audited")
	}
}

func TestBatchDAdminAuditFailureRollsBackFriendChange(t *testing.T) {
	api, db := newAPI(t, true)
	a, b, c := create(t, api), create(t, api), create(t, api)
	if _, err := db.Exec(`CREATE TRIGGER audit_test_failure BEFORE INSERT ON admin_audit WHEN NEW.friend_id='` + b["user"].(string) + `' BEGIN SELECT RAISE(ABORT,'audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	add := map[string]any{"user": a["user"], "friend": b["user"], "action": "add"}
	out := dCall(t, api, "POST", "/admin/api/friends_set", "127.0.0.1:1234", "", add, 200)
	if out["ok"] != false {
		t.Fatal("audit failure reported success", out)
	}
	if n := dCount(t, db, "SELECT COUNT(*) FROM friendship WHERE user_id=? AND friend_id=?", a["user"], b["user"]); n != 0 {
		t.Fatal("friend change committed without audit")
	}
	add["friend"] = c["user"]
	if out := dCall(t, api, "POST", "/admin/api/friends_set", "127.0.0.1:1234", "", add, 200); out["ok"] != true {
		t.Fatal(out)
	}
	if _, err := db.Exec("INSERT INTO friendship(user_id,friend_id,status) VALUES(?,?,'active'),(?,?,'active')", a["user"], b["user"], b["user"], a["user"]); err != nil {
		t.Fatal(err)
	}
	clear := dCall(t, api, "POST", "/admin/api/friends_clear", "127.0.0.1:1234", "", map[string]any{"user": a["user"]}, 200)
	if clear["ok"] != false || len(clear["failed"].([]any)) != 1 {
		t.Fatal("partial clear was reported as complete", clear)
	}
	if n := dCount(t, db, "SELECT COUNT(*) FROM friendship WHERE user_id=? AND friend_id=? AND status='active'", a["user"], b["user"]); n != 1 {
		t.Fatal("failed clear changed unaudited friendship")
	}
	if n := dCount(t, db, "SELECT COUNT(*) FROM friendship WHERE user_id=? AND friend_id=? AND status='removed'", a["user"], c["user"]); n != 1 {
		t.Fatal("successful clear did not commit")
	}
}
