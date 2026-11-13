package handler

import (
	"database/sql"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func cAuth(a map[string]any) map[string]any {
	return map[string]any{"user": a["user"], "session": a["session"]}
}

func TestBatchCFriendRelationAndRoomBoundary(t *testing.T) {
	api, db := newAPI(t, true)
	a, b, c := create(t, api), create(t, api), create(t, api)
	name := map[string]any{"user": a["user"], "session": a["session"], "friend": b["user"], "name": "伙伴"}
	if got := call(t, api, "/account/set_friend_name", name, 200); got["result"] != true {
		t.Fatal(got)
	}
	var reverse int
	if err := db.QueryRow("SELECT COUNT(*) FROM friendship WHERE user_id=? AND friend_id=?", b["user"], a["user"]).Scan(&reverse); err != nil {
		t.Fatal(err)
	}
	if reverse != 1 {
		t.Fatal("friendship missing reverse row")
	}
	name["session"] = c["session"]
	if got := call(t, api, "/account/set_friend_name", name, 200); got["result"] != "unauthorized" {
		t.Fatal(got)
	}
	name["session"] = a["session"]
	block := map[string]any{"user": a["user"], "session": a["session"], "friend": b["user"], "blocked": true}
	if got := call(t, api, "/account/set_friend_block", block, 200); got["result"] != true {
		t.Fatal(got)
	}
	blocked := call(t, api, "/account/get_blocked_friends", cAuth(a), 200)
	if len(blocked["set_blocked_friends"].(map[string]any)["friends"].([]any)) != 1 {
		t.Fatal(blocked)
	}
	other := call(t, api, "/account/get_blocked_friends", cAuth(c), 200)
	if len(other["set_blocked_friends"].(map[string]any)["friends"].([]any)) != 0 {
		t.Fatal(other)
	}
	join := map[string]any{"user": a["user"], "session": a["session"], "friend_id": b["user"]}
	if got := call(t, api, "/account/join_friend_game", join, 404); got["code"] != "FRIEND_NOT_IN_GAME" {
		t.Fatal(got)
	}
	call(t, api, "/account/get_remote_outfit", map[string]any{"user": a["user"], "session": c["session"], "target": b["user"]}, 401)
	call(t, api, "/account/set_outfit", map[string]any{"user": a["user"], "session": c["session"], "outfit": map[string]any{"height": 2}}, 401)
	call(t, api, "/account/set_outfit", map[string]any{"user": b["user"], "session": b["session"], "outfit": map[string]any{"height": 2}}, 200)
	outfit := call(t, api, "/account/get_remote_outfit", map[string]any{"user": a["user"], "session": a["session"], "target": b["user"]}, 200)
	if outfit["get_remote_outfit"].(map[string]any)["height"] != float64(2) {
		t.Fatal(outfit)
	}
}

func TestBatchCRelationshipRemainingEntries(t *testing.T) {
	api, db := newAPI(t, true)
	a, b := create(t, api), create(t, api)
	request := cAuth(a)
	for _, path := range []string{"/account/get_external_friends", "/account/serendipity/get", "/account/star/get"} {
		out := call(t, api, path, request, 200)
		if len(out) == 0 {
			t.Fatal(path, out)
		}
	}
	if out := call(t, api, "/account/set_joinable", map[string]any{"joinable": false}, 200); out["joinable"] != false {
		t.Fatal(out)
	}
	if out := call(t, api, "/account/remove_relationship_hint", map[string]any{"user": a["user"], "session": b["session"], "friend": b["user"], "hint": 1}, 200); out["result"] != false {
		t.Fatal(out)
	}
	if out := call(t, api, "/service/relationship/api/v1/get_all_friend_gist", request, 200); len(out["get_all_friend_gist"].([]any)) != 0 {
		t.Fatal(out)
	}
	if out := call(t, api, "/service/relationship/api/v1/get_follower_count", request, 200); out["count"] != float64(0) {
		t.Fatal(out)
	}
	pages := map[string]any{"user": a["user"], "session": a["session"], "pages": []any{map[string]any{"friends": []any{map[string]any{"friend_id": b["user"], "constellation_node_index": 2}, map[string]any{"friend_id": b["user"], "constellation_node_index": 2}}}}}
	if out := call(t, api, "/service/relationship/api/v1/update_friend_constellation_pages", pages, 200); out["total_friends"] != float64(1) {
		t.Fatal(out)
	}
	if n := cQueryInt(t, db, "SELECT COUNT(*) FROM constellation_friend_page WHERE user_id=?", a["user"]); n != 1 {
		t.Fatal("duplicate constellation entry")
	}
	pages["session"] = b["session"]
	if out := call(t, api, "/service/relationship/api/v1/update_friend_constellation_pages", pages, 200); out["status"] != "unauthorized" {
		t.Fatal(out)
	}
	for _, path := range []string{"/service/trust/api/v1/get_capped", "/service/trust/api/v1/get"} {
		if out := call(t, api, path, request, 200); len(out) == 0 {
			t.Fatal(path, out)
		}
	}
}

func TestBatchCMessageReceivedScopedAndStable(t *testing.T) {
	api, db := newAPI(t, true)
	a, b := create(t, api), create(t, api)
	message := []any{map[string]any{"id": "mail-1"}}
	call(t, api, "/service/message/api/v1/mark_received", map[string]any{"user": a["user"], "session": b["session"], "messages": message}, 401)
	request := map[string]any{"user": a["user"], "session": a["session"], "messages": message}
	first := call(t, api, "/service/message/api/v1/mark_received", request, 200)
	if first["status"] != "OK" {
		t.Fatal(first)
	}
	var stamp int64
	if err := db.QueryRow("SELECT received_at FROM message WHERE user_id=? AND id='mail-1'", a["user"]).Scan(&stamp); err != nil {
		t.Fatal(err)
	}
	call(t, api, "/service/message/api/v1/mark_received", request, 200)
	var again int64
	if err := db.QueryRow("SELECT received_at FROM message WHERE user_id=? AND id='mail-1'", a["user"]).Scan(&again); err != nil {
		t.Fatal(err)
	}
	if again != stamp {
		t.Fatalf("received timestamp changed: %d -> %d", stamp, again)
	}
	var other int
	if err := db.QueryRow("SELECT COUNT(*) FROM message WHERE user_id=?", b["user"]).Scan(&other); err != nil {
		t.Fatal(err)
	}
	if other != 0 {
		t.Fatal("cross-account message state")
	}
	seen := call(t, api, "/service/message/api/v1/mark_seen", request, 200)
	if seen["status"] != "OK" || len(seen["messages"].([]any)) != 1 {
		t.Fatal(seen)
	}
}

func TestBatchCMailboxVersionsAndRemainingMessageEntries(t *testing.T) {
	_, db := newAPI(t, true)
	dir := t.TempDir()
	mailbox := `{"mails":[{"id":"mail-early","type":"mail","subject":"early","body":"hello"},{"id":"mail-late","type":"mail","subject":"late","body":"future","min_version":"0.4.0"}]}`
	if err := os.WriteFile(filepath.Join(dir, "mailbox.json"), []byte(mailbox), 0600); err != nil {
		t.Fatal(err)
	}
	api, err := New(db, Config{AutoAuthorize: true, GameConfigDir: dir, RoomHost: "192.168.1.10", RoomPort: 25565})
	if err != nil {
		t.Fatal(err)
	}
	a, b := create(t, api), create(t, api)
	for _, path := range []string{"/service/message/api/v1/get_all", "/service/message/api/v2/get_all"} {
		out := call(t, api, path, cAuth(a), 200)
		if out["status"] != "OK" || len(out["messages"].([]any)) == 0 {
			t.Fatal(path, out)
		}
	}
	got := call(t, api, "/service/message/api/v1/get", map[string]any{"user": a["user"], "session": a["session"], "messages": []any{map[string]any{"id": "mail-early"}}}, 200)
	if got["status"] != "OK" || len(got["update_messages"].([]any)) != 1 {
		t.Fatal(got)
	}
	for _, path := range []string{"/account/translate", "/account/set_device_token"} {
		out := call(t, api, path, cAuth(a), 200)
		if path == "/account/translate" && out["result"] != "ok" {
			t.Fatal(out)
		}
	}
	call(t, api, "/account/give_candle", map[string]any{"user": a["user"], "session": b["session"], "recv_user": b["user"], "ability_id": 1}, 401)
	if out := call(t, api, "/account/give_candle", cAuth(a), 200); out["result"] != "missing_target_or_ability" {
		t.Fatal(out)
	}
	call(t, api, "/service/relationship/api/v1/free_gifts/send", map[string]any{"user": a["user"], "session": a["session"], "target": b["user"], "gift_type": "gift_scandle"}, 200)
	pending := call(t, api, "/service/relationship/api/v1/free_gifts/get_pending", cAuth(b), 200)
	if len(pending["set_recvd_free_gifts"].([]any)) != 1 {
		t.Fatal(pending)
	}
	call(t, api, "/service/relationship/api/v1/free_gifts/get_pending", map[string]any{"user": b["user"], "session": a["session"]}, 401)
}

func TestBatchCExpiredGiftCannotBeClaimed(t *testing.T) {
	api, db := newAPI(t, true)
	a, b := create(t, api), create(t, api)
	call(t, api, "/service/relationship/api/v1/free_gifts/send", map[string]any{"user": a["user"], "session": a["session"], "target": b["user"], "gift_type": "gift_scandle"}, 200)
	var id string
	if err := db.QueryRow("SELECT gift_id FROM free_gift WHERE receiver_id=?", b["user"]).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE free_gift SET expires_at=? WHERE gift_id=?", time.Now().Unix()-1, id); err != nil {
		t.Fatal(err)
	}
	claim := call(t, api, "/service/relationship/api/v1/free_gifts/claim", map[string]any{"user": b["user"], "session": b["session"], "gift_id": id}, 200)
	if claim["result"] != "invalid" {
		t.Fatal(claim)
	}
	for _, path := range []string{"/account/claim_message_gift", "/account/claim_message_gifts"} {
		out := call(t, api, path, map[string]any{"user": b["user"], "session": b["session"], "gift_id": id, "from_id": a["user"]}, 200)
		if out["result"] != "ok" || len(out["set_recvd_messages"].([]any)) != 0 {
			t.Fatal(path, out)
		}
	}
	var balance int
	if err := db.QueryRow("SELECT season_candle FROM user_currency WHERE user_id=?", b["user"]).Scan(&balance); err != nil {
		t.Fatal(err)
	}
	if balance != 0 {
		t.Fatal("expired gift paid reward")
	}
}

func TestBatchCGiftClaimSurvivesAPIRecreation(t *testing.T) {
	api, db := newAPI(t, true)
	a, b := create(t, api), create(t, api)
	call(t, api, "/service/relationship/api/v1/free_gifts/send", map[string]any{"user": a["user"], "session": a["session"], "target": b["user"], "gift_type": "gift_scandle"}, 200)
	api2, err := New(db, Config{AutoAuthorize: true, RoomHost: "192.168.1.10", RoomPort: 25565})
	if err != nil {
		t.Fatal(err)
	}
	pending := call(t, api2, "/service/relationship/api/v1/free_gifts/get_pending", cAuth(b), 200)
	if len(pending["set_recvd_free_gifts"].([]any)) != 1 {
		t.Fatal(pending)
	}
	var id string
	if err := db.QueryRow("SELECT gift_id FROM free_gift WHERE receiver_id=?", b["user"]).Scan(&id); err != nil {
		t.Fatal(err)
	}
	claim := map[string]any{"user": b["user"], "session": b["session"], "gift_id": id}
	call(t, api2, "/service/relationship/api/v1/free_gifts/claim", claim, 200)
	call(t, api, "/account/claim_message_gift", claim, 200)
	if n := cQueryInt(t, db, "SELECT season_candle FROM user_currency WHERE user_id=?", b["user"]); n != 1 {
		t.Fatalf("cross-instance claim paid %d", n)
	}
}

func TestBatchCGeonoteAliasesBugAndLimit(t *testing.T) {
	api, db := newAPI(t, true)
	a := create(t, api)
	for _, path := range []string{"/account/create_geonote2", "/account/geonote/create_geonote2"} {
		out := call(t, api, path, map[string]any{"user": a["user"], "session": a["session"], "message": path, "level_id": 91}, 200)
		if out["result"] != "ok" {
			t.Fatal(path, out)
		}
	}
	for _, path := range []string{"/account/geonote/request_geonotes_for_n_friends", "/account/get_user_geonotes", "/account/geonote/get_user_geonotes"} {
		out := call(t, api, path, map[string]any{"user": a["user"], "session": a["session"], "level_id": 91, "n": 1, "limit": 1}, 200)
		if len(out["geonotes"].([]any)) != 1 {
			t.Fatal(path, out)
		}
	}
	read := call(t, api, "/account/get_user_geonotes", map[string]any{"user": a["user"], "session": a["session"], "level_id": 91}, 200)
	items := read["geonotes"].([]any)
	if len(items) != 2 {
		t.Fatal(read)
	}
	id := items[0].(map[string]any)["geonote_id"]
	for _, path := range []string{"/account/social_geonote2", "/account/geonote/social_geonote2"} {
		out := call(t, api, path, map[string]any{"user": a["user"], "session": a["session"], "geonote_id": id, "liked": true}, 200)
		if out["result"] != "ok" {
			t.Fatal(path, out)
		}
	}
	if n := cQueryInt(t, db, "SELECT COUNT(*) FROM geonote WHERE geonote_id=? AND liked=1", id); n != 1 {
		t.Fatal("geonote social flag missing")
	}
	for _, path := range []string{"/account/create_bug_report_geonote", "/account/geonote/create_bug_report_geonote"} {
		out := call(t, api, path, map[string]any{"user": a["user"], "session": a["session"], "message": path, "level_id": 91}, 200)
		if out["result"] != "ok" {
			t.Fatal(path, out)
		}
	}
	for _, path := range []string{"/account/request_n_bug_report_geonotes", "/account/geonote/request_n_bug_report_geonotes"} {
		out := call(t, api, path, map[string]any{"user": a["user"], "session": a["session"], "level_id": 91}, 200)
		if len(out["geonotes"].([]any)) != 2 {
			t.Fatal(path, out)
		}
	}
	for _, path := range []string{"/account/delete_bug_report_geonote", "/account/geonote/delete_bug_report_geonote"} {
		out := call(t, api, "/account/request_n_bug_report_geonotes", map[string]any{"user": a["user"], "session": a["session"], "level_id": 91}, 200)
		id := out["geonotes"].([]any)[0].(map[string]any)["geonote_id"]
		call(t, api, path, map[string]any{"user": a["user"], "session": a["session"], "geonote_id": id}, 200)
	}
	if n := cQueryInt(t, db, "SELECT COUNT(*) FROM geonote WHERE author_id=? AND bug=1", a["user"]); n != 0 {
		t.Fatal("bug reports remain")
	}
}

func TestBatchCFeedAliasesPaginationAndLikes(t *testing.T) {
	api, db := newAPI(t, true)
	a, b := create(t, api), create(t, api)
	ids := []string{}
	for _, path := range []string{"/account/social_feed/create", "/service/social-feed/api/v1/create", "/account/social_feed/create"} {
		out := call(t, api, path, map[string]any{"user": a["user"], "session": a["session"], "pool_type": "test", "pool_name": "test", "message": path}, 200)
		if out["result"] != "success" {
			t.Fatal(path, out)
		}
		ids = append(ids, out["social_feed_id"].(string))
	}
	for _, path := range []string{"/account/social_feed/get_user_feeds", "/service/social-feed/api/v1/get_user_feeds"} {
		out := call(t, api, path, map[string]any{"user": b["user"], "session": b["session"], "skip": 0, "batch_size": 2}, 200)
		if len(out["feeds"].([]any)) != 2 {
			t.Fatal(path, out)
		}
		out = call(t, api, path, map[string]any{"user": b["user"], "session": b["session"], "skip": 2, "batch_size": 2}, 200)
		if len(out["feeds"].([]any)) != 1 {
			t.Fatal(path, out)
		}
	}
	for _, path := range []string{"/account/social_feed/get_curated_feeds", "/service/social-feed/api/v2/get_curated_feeds"} {
		out := call(t, api, path, map[string]any{"user": b["user"], "session": b["session"]}, 200)
		if _, ok := out["inner_ring"].(map[string]any); !ok {
			t.Fatal(path, out)
		}
	}
	byID := call(t, api, "/service/social-feed/api/v1/get_by_ids", map[string]any{"social_feed_ids": []string{ids[0], ids[0], ids[1]}}, 200)
	if len(byID["feeds"].([]any)) != 2 {
		t.Fatal(byID)
	}
	for _, path := range []string{"/account/social_feed/like", "/service/social-feed/api/v1/like"} {
		for i := 0; i < 2; i++ {
			out := call(t, api, path, map[string]any{"user": b["user"], "session": b["session"], "social_feed_id": ids[0], "liked": true}, 200)
			if out["result"] != "ok" && out["result"] != "created" {
				t.Fatal(path, out)
			}
		}
	}
	if n := cQueryInt(t, db, "SELECT likes_count FROM social_feed_post WHERE user_id=? AND message=?", a["user"], "/account/social_feed/create"); n != 1 {
		t.Fatalf("like counted %d times", n)
	}
	for _, path := range []string{"/account/social_feed/get_reactions", "/service/social-feed/api/v1/get_reactions"} {
		out := call(t, api, path, map[string]any{"user": b["user"], "session": b["session"], "social_feed_id": ids[0]}, 200)
		if _, ok := out["reactions"]; !ok {
			t.Fatal(path, out)
		}
	}
	if out := call(t, api, "/service/social-feed/api/v1/hide", map[string]any{"user": b["user"], "session": b["session"], "social_feed_id": ids[0]}, 200); out["result"] != "" {
		t.Fatal(out)
	}
	if out := call(t, api, "/service/social-feed/api/v2/comments", map[string]any{}, 200); len(out["comments"].([]any)) != 0 {
		t.Fatal(out)
	}
	if out := call(t, api, "/service/social-feed/api/v1/report", map[string]any{}, 200); out["result"] != "" {
		t.Fatal(out)
	}
}

func TestBatchCPenpalPlaceholderAndAccountDelete(t *testing.T) {
	api, db := newAPI(t, true)
	a, b := create(t, api), create(t, api)
	created := call(t, api, "/account/social_feed/create", map[string]any{"user": a["user"], "session": a["session"], "pool_type": "SocialFeedPenpal", "pool_name": "test", "message": "penpal"}, 200)
	id := created["social_feed_id"]
	if out := call(t, api, "/service/social-feed/api/v2/get_penpals", cAuth(a), 200); len(out["penpals"].([]any)) != 0 {
		t.Fatal(out)
	}
	if out := call(t, api, "/service/social-feed/api/v2/get_penpal_by_id", map[string]any{"social_feed_id": id}, 200); len(out["feeds"].([]any)) != 1 {
		t.Fatal(out)
	}
	deleteRequest := map[string]any{"user": b["user"], "session": b["session"], "social_feed_id": id, "author_id": a["user"]}
	if out := call(t, api, "/account/social_feed/delete", deleteRequest, 200); out["message"] != "Permission denied" {
		t.Fatal(out)
	}
	if n := cQueryInt(t, db, "SELECT COUNT(*) FROM social_feed_post WHERE user_id=?", a["user"]); n != 1 {
		t.Fatal("other account deleted feed")
	}
	deleteRequest["user"], deleteRequest["session"] = a["user"], a["session"]
	if out := call(t, api, "/account/social_feed/delete", deleteRequest, 200); out["result"] != "ok" {
		t.Fatal(out)
	}
	if n := cQueryInt(t, db, "SELECT COUNT(*) FROM social_feed_post WHERE user_id=?", a["user"]); n != 0 {
		t.Fatal("owner delete failed")
	}
}

func TestBatchCOptionsMethodOnRelationshipRoutes(t *testing.T) {
	api, _ := newAPI(t, true)
	for _, path := range []string{"/service/relationship/api/v1/update_friend_constellation_pages", "/service/relationship/api/v1/free_gifts/send"} {
		req := httptest.NewRequest("OPTIONS", path, nil)
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)
		if w.Code != 200 && w.Code != 204 {
			t.Fatalf("%s: OPTIONS %d: %s", path, w.Code, w.Body.String())
		}
	}
}

func TestBatchCRecordingRestartAndFailedChunk(t *testing.T) {
	api, db, dir := cNewFileAPI(t)
	a := create(t, api)
	base := map[string]any{"user": a["user"], "session": a["session"], "user_id": a["user"], "recording_data": map[string]any{"chet": 1}}
	created := call(t, api, "/service/recording/api/v2/create", base, 200)
	id := created["recording_id"].(string)
	path := cRecordingPath(dir, id)
	before := cFileSize(t, path)
	bad := map[string]any{"user": a["user"], "session": a["session"], "user_id": a["user"], "recording_id": id, "chunk": 3, "recording_data": map[string]any{"chet": 3}}
	call(t, api, "/service/recording/api/v2/update", bad, 409)
	if cFileSize(t, path) != before {
		t.Fatal("out-of-order chunk wrote data")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	bad["chunk"] = 1
	call(t, api, "/service/recording/api/v2/update", bad, 500)
	got := call(t, api, "/service/recording/api/v2/get", map[string]any{"recording_id": id}, 200)
	if len(got["seek_table"].(map[string]any)) != 1 {
		t.Fatal("failed chunk changed metadata", got)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"chet":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	api2, err := New(db, Config{AutoAuthorize: true, InstanceDir: dir, RoomHost: "192.168.1.10", RoomPort: 25565})
	if err != nil {
		t.Fatal(err)
	}
	got = call(t, api2, "/service/recording/api/v2/get", map[string]any{"recording_id": id}, 200)
	if got["status"] != "OK" || len(got["seek_table"].(map[string]any)) != 1 {
		t.Fatal(got)
	}
	if w := cGet(t, api2, "/ugc/"+a["user"].(string)+"/"+id+".ndjson", "bytes=0-3"); w.Code != 206 {
		t.Fatal(w.Code)
	}
	v1 := call(t, api2, "/service/recording/api/v1/create", base, 200)
	v1got := call(t, api2, "/service/recording/api/v1/get", map[string]any{"recording_id": v1["recording_id"]}, 200)
	if v1got["status"] != "OK" {
		t.Fatal(v1got)
	}
}

func cQueryInt(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func cGet(t *testing.T, api *API, path, rangeHeader string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest("GET", path, nil)
	if rangeHeader != "" {
		r.Header.Set("Range", rangeHeader)
	}
	w := httptest.NewRecorder()
	api.ServeHTTP(w, r)
	return w
}

func cNewFileAPI(t *testing.T) (*API, *sql.DB, string) {
	t.Helper()
	_, db := newAPI(t, true)
	dir := t.TempDir()
	api, err := New(db, Config{AutoAuthorize: true, InstanceDir: dir, RoomHost: "192.168.1.10", RoomPort: 25565})
	if err != nil {
		t.Fatal(err)
	}
	return api, db, dir
}

func cFileSize(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Size()
}

func cRecordingPath(dir, id string) string { return filepath.Join(dir, "recordings", id+".ndjson") }
