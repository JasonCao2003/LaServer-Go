package handler_test

import "testing"

func TestAdminAccountSearch(t *testing.T) {
	api, _ := newAPI(t, true)
	a := create(t, api)
	create(t, api)
	dCall(t, api, "POST", "/admin/api/accounts_list", "192.0.2.3:12345", "", nil, 403)
	out := dCall(t, api, "POST", "/admin/api/accounts_list", "127.0.0.1:12345", "", map[string]any{"keyword": a["user"]}, 200)
	if out["total"] != float64(2) || out["matched"] != float64(1) {
		t.Fatalf("unexpected counts: %v", out)
	}
	rows, ok := out["accounts"].([]any)
	if !ok || len(rows) != 1 {
		t.Fatalf("unexpected accounts: %v", out)
	}
	row := rows[0].(map[string]any)
	if row["id"] != a["user"] || row["session"] != nil || row["recovery_token"] != nil {
		t.Fatal("unexpected account projection")
	}
}

func TestAdminAccountModeration(t *testing.T) {
	api, db := newAPI(t, true)
	a := create(t, api)
	body := map[string]any{"user": a["user"], "action": "ban", "seconds": "3600"}
	const path = "/admin/api/accounts_change"
	dCall(t, api, "POST", path, "192.0.2.3:12345", "", body, 403)
	dCall(t, api, "POST", path, "127.0.0.1:12345", "", body, 200)
	var banned int
	var deadline int64
	if err := db.QueryRow("SELECT is_banned,ban_end_time FROM user WHERE id=?", a["user"]).Scan(&banned, &deadline); err != nil || banned != 1 || deadline <= 0 {
		t.Fatalf("ban not persisted: %d %d %v", banned, deadline, err)
	}
	body["action"], body["seconds"] = "unban", ""
	dCall(t, api, "POST", path, "127.0.0.1:12345", "", body, 200)
	if err := db.QueryRow("SELECT is_banned,ban_end_time FROM user WHERE id=?", a["user"]).Scan(&banned, &deadline); err != nil || banned != 0 || deadline != 0 {
		t.Fatalf("ban not cleared: %d %d %v", banned, deadline, err)
	}
}
