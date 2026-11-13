package handler

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBatchCGeonoteAliasesAndOwnership(t *testing.T) {
	api, _ := newAPI(t, true)
	a, b := create(t, api), create(t, api)
	auth := map[string]any{"user": a["user"], "session": a["session"]}
	created := call(t, api, "/account/geonote/create_geonote2", map[string]any{"user": a["user"], "session": a["session"], "message": "hello", "level_id": 42}, 200)
	if created["result"] != "ok" {
		t.Fatal(created)
	}
	id := created["geonote"].(map[string]any)["geonote_id"]
	read := call(t, api, "/account/get_user_geonotes", map[string]any{"user": b["user"], "session": b["session"], "level_id": 42}, 200)
	if len(read["geonotes"].([]any)) != 1 {
		t.Fatal(read)
	}
	call(t, api, "/account/edit_message_geonote", map[string]any{"user": b["user"], "session": b["session"], "geonote_id": id, "message": "stolen", "level_id": 42}, 200)
	call(t, api, "/account/geonote/delete_geonote", map[string]any{"user": b["user"], "session": b["session"], "geonote_id": id}, 200)
	read = call(t, api, "/account/geonote/get_user_geonotes", map[string]any{"user": a["user"], "session": a["session"], "level_id": 42}, 200)
	items := read["geonotes"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["message"] != "hello" {
		t.Fatal(read)
	}
	call(t, api, "/account/geonote/edit_message_geonote", map[string]any{"user": a["user"], "session": a["session"], "geonote_id": id, "message": "updated", "level_id": 42}, 200)
	call(t, api, "/account/delete_geonote", map[string]any{"user": a["user"], "session": a["session"], "geonote_id": id}, 200)
	read = call(t, api, "/account/get_user_geonotes", map[string]any{"user": a["user"], "session": a["session"], "level_id": 42}, 200)
	if len(read["geonotes"].([]any)) != 0 {
		t.Fatal(read)
	}
	auth["session"] = b["session"]
	call(t, api, "/account/geonote/create_geonote2", auth, 401)
}

func TestBatchCFollowIsAccountScoped(t *testing.T) {
	api, db := newAPI(t, true)
	a, b := create(t, api), create(t, api)
	creator := "test-creator"
	call(t, api, "/service/relationship/api/v1/add_follow", map[string]any{"user": a["user"], "session": b["session"], "creator": creator}, 401)
	call(t, api, "/service/relationship/api/v1/add_follow", map[string]any{"user": a["user"], "session": a["session"], "creator": creator, "nickname": "one"}, 200)
	call(t, api, "/service/relationship/api/v1/set_follow_name", map[string]any{"user": a["user"], "session": a["session"], "creator": creator, "nickname": "two"}, 200)
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM boat_follow WHERE user_id=?", a["user"]).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected one follow, got %d", n)
	}
	call(t, api, "/service/relationship/api/v1/delete_follow", map[string]any{"user": b["user"], "session": b["session"], "creator": creator}, 200)
	if err := db.QueryRow("SELECT COUNT(*) FROM boat_follow WHERE user_id=?", a["user"]).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal("cross-account delete removed follow")
	}
	call(t, api, "/service/relationship/api/v1/delete_follow", map[string]any{"user": a["user"], "session": a["session"], "creator": creator}, 200)
	if err := db.QueryRow("SELECT COUNT(*) FROM boat_follow WHERE user_id=?", a["user"]).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("follow remains after delete")
	}
}

func TestBatchCFeedOwnershipAndSuperLikeIdempotency(t *testing.T) {
	api, db := newAPI(t, true)
	a, b := create(t, api), create(t, api)
	created := call(t, api, "/service/social-feed/api/v1/create", map[string]any{"user": a["user"], "session": a["session"], "pool_type": "test", "pool_name": "test", "message": "original"}, 200)
	if created["result"] != "success" {
		t.Fatal(created)
	}
	id := created["social_feed_id"]
	call(t, api, "/service/social-feed/api/v1/edit", map[string]any{"user": b["user"], "session": b["session"], "social_feed_id": id, "message": "stolen"}, 403)
	call(t, api, "/service/social-feed/api/v1/delete", map[string]any{"user": b["user"], "session": b["session"], "social_feed_id": id}, 403)
	var message string
	if err := db.QueryRow("SELECT message FROM social_feed_post WHERE user_id=?", a["user"]).Scan(&message); err != nil {
		t.Fatal(err)
	}
	if message != "original" {
		t.Fatal(message)
	}
	if _, err := db.Exec("UPDATE user_currency SET candles=5 WHERE user_id=?", b["user"]); err != nil {
		t.Fatal(err)
	}
	request := map[string]any{"user": b["user"], "session": b["session"], "social_feed_id": id, "super_liked": true}
	first := call(t, api, "/service/social-feed/api/v1/super_like", request, 200)
	second := call(t, api, "/service/social-feed/api/v1/super_like", request, 200)
	if first["result"] != "success" || second["result"] != "success" {
		t.Fatal(first, second)
	}
	var candles, likes int
	if err := db.QueryRow("SELECT candles FROM user_currency WHERE user_id=?", b["user"]).Scan(&candles); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT likes_count FROM social_feed_post WHERE user_id=?", a["user"]).Scan(&likes); err != nil {
		t.Fatal(err)
	}
	if candles != 2 || likes != 1 {
		t.Fatalf("duplicate super like charged or counted again: candles=%d likes=%d", candles, likes)
	}
	call(t, api, "/service/social-feed/api/v1/edit", map[string]any{"user": a["user"], "session": a["session"], "social_feed_id": id, "message": "updated"}, 200)
	call(t, api, "/service/social-feed/api/v1/delete", map[string]any{"user": a["user"], "session": a["session"], "social_feed_id": id}, 200)
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM social_feed_post WHERE user_id=?", a["user"]).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("feed remains after owner deletion")
	}
}

func TestBatchCFreeGiftClaimIsScopedAndIdempotent(t *testing.T) {
	api, db := newAPI(t, true)
	a, b, c := create(t, api), create(t, api), create(t, api)
	sent := call(t, api, "/service/relationship/api/v1/free_gifts/send", map[string]any{"user": a["user"], "session": a["session"], "target": b["user"], "gift_type": "gift_scandle"}, 200)
	if sent["result"] != "ok" {
		t.Fatal(sent)
	}
	var giftID string
	if err := db.QueryRow("SELECT gift_id FROM free_gift WHERE sender_id=? AND receiver_id=?", a["user"], b["user"]).Scan(&giftID); err != nil {
		t.Fatal(err)
	}
	bad := call(t, api, "/service/relationship/api/v1/free_gifts/claim", map[string]any{"user": c["user"], "session": c["session"], "gift_id": giftID}, 200)
	if bad["result"] != "invalid" {
		t.Fatal(bad)
	}
	request := map[string]any{"user": b["user"], "session": b["session"], "gift_id": giftID}
	first := call(t, api, "/service/relationship/api/v1/free_gifts/claim", request, 200)
	second := call(t, api, "/service/relationship/api/v1/free_gifts/claim", request, 200)
	if first["result"] != "ok" || second["result"] != "ok" {
		t.Fatal(first, second)
	}
	var balance int
	if err := db.QueryRow("SELECT season_candle FROM user_currency WHERE user_id=?", b["user"]).Scan(&balance); err != nil {
		t.Fatal(err)
	}
	if balance != 1 {
		t.Fatalf("repeated gift claim balance=%d", balance)
	}
}

func TestBatchCRecordingChunksAndUGCRanges(t *testing.T) {
	_, db := newAPI(t, true)
	api, err := New(db, Config{AutoAuthorize: true, InstanceDir: t.TempDir(), RoomHost: "192.168.1.10", RoomPort: 25565})
	if err != nil {
		t.Fatal(err)
	}
	a, b := create(t, api), create(t, api)
	base := map[string]any{"user": a["user"], "session": a["session"], "user_id": a["user"], "recording_data": map[string]any{"chet": 1}}
	created := call(t, api, "/service/recording/api/v2/create", base, 200)
	if created["status"] != "OK" {
		t.Fatal(created)
	}
	id := created["recording_id"]
	call(t, api, "/service/recording/api/v2/update", map[string]any{"user": b["user"], "session": b["session"], "user_id": b["user"], "recording_id": id, "chunk": 1, "recording_data": map[string]any{"chet": 2}}, 403)
	call(t, api, "/service/recording/api/v2/update", map[string]any{"user": a["user"], "session": a["session"], "user_id": a["user"], "recording_id": id, "chunk": 2, "recording_data": map[string]any{"chet": 2}}, 409)
	update := map[string]any{"user": a["user"], "session": a["session"], "user_id": a["user"], "recording_id": id, "chunk": 1, "recording_data": map[string]any{"chet": 2}}
	call(t, api, "/service/recording/api/v2/update", update, 200)
	call(t, api, "/service/recording/api/v2/update", update, 200)
	got := call(t, api, "/service/recording/api/v2/get", map[string]any{"recording_id": id}, 200)
	if len(got["seek_table"].(map[string]any)) != 2 {
		t.Fatal(got)
	}
	path := "/ugc/" + a["user"].(string) + "/" + id.(string) + ".ndjson"
	request := func(path, rng string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", path, nil)
		if rng != "" {
			req.Header.Set("Range", rng)
		}
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)
		return w
	}
	full := request(path, "")
	if full.Code != 200 || !strings.Contains(full.Body.String(), "chet") {
		t.Fatal(full.Code, full.Body.String())
	}
	part := request(path, "bytes=0-3")
	if part.Code != 206 || part.Body.String() != full.Body.String()[:4] {
		t.Fatal(part.Code, part.Body.String())
	}
	if w := request(path, "bytes=999999-"); w.Code != 416 {
		t.Fatal(w.Code)
	}
	if w := request("/ugc/../"+id.(string)+".ndjson", ""); w.Code != 404 {
		t.Fatal("traversal served recording", w.Code)
	}
}

func TestBatchCChatAliasesAndPolling(t *testing.T) {
	api, _ := newAPI(t, true)
	a := create(t, api)
	for _, path := range []string{"/account/chat/send", "/account/send", "/account/send_message"} {
		out := call(t, api, path, map[string]any{"user": a["user"], "session": a["session"], "msg": path}, 200)
		if out["result"] != "success" {
			t.Fatal(path, out)
		}
	}
	first := call(t, api, "/account/get_pending_messages", map[string]any{"user": a["user"], "session": a["session"]}, 200)
	second := call(t, api, "/account/get_pending_messages", map[string]any{"user": a["user"], "session": a["session"]}, 200)
	if len(first["set_recvd_messages"].([]any)) != 3 || len(second["set_recvd_messages"].([]any)) != 0 {
		t.Fatal(first, second)
	}
	call(t, api, "/account/get_pending_messages", map[string]any{"user": a["user"], "session": "wrong"}, 401)
}

func TestBatchCMessageGiftClaimAliases(t *testing.T) {
	api, db := newAPI(t, true)
	a, b := create(t, api), create(t, api)
	sent := call(t, api, "/account/send", map[string]any{"user": a["user"], "session": a["session"], "target": b["user"], "gift_type": "gift_scandle"}, 200)
	if sent["result"] != "ok" {
		t.Fatal(sent)
	}
	var giftID string
	if err := db.QueryRow("SELECT gift_id FROM free_gift WHERE sender_id=? AND receiver_id=?", a["user"], b["user"]).Scan(&giftID); err != nil {
		t.Fatal(err)
	}
	claim := map[string]any{"user": b["user"], "session": b["session"], "gift_id": giftID}
	if out := call(t, api, "/account/claim_message_gift", claim, 200); out["result"] != "ok" {
		t.Fatal(out)
	}
	if out := call(t, api, "/account/claim_message_gifts", claim, 200); out["result"] != "ok" {
		t.Fatal(out)
	}
	var balance int
	if err := db.QueryRow("SELECT season_candle FROM user_currency WHERE user_id=?", b["user"]).Scan(&balance); err != nil {
		t.Fatal(err)
	}
	if balance != 1 {
		t.Fatalf("gift paid twice: %d", balance)
	}
}

func TestBatchCRecordingStorageFailureIsReported(t *testing.T) {
	_, db := newAPI(t, true)
	root := t.TempDir()
	blocked := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(blocked, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	api, err := New(db, Config{AutoAuthorize: true, InstanceDir: blocked, RoomHost: "192.168.1.10", RoomPort: 25565})
	if err != nil {
		t.Fatal(err)
	}
	a := create(t, api)
	for _, path := range []string{"/service/recording/api/v1/create", "/service/recording/api/v2/create"} {
		call(t, api, path, map[string]any{"user": a["user"], "session": a["session"], "user_id": a["user"], "recording_data": map[string]any{"chet": 1}}, 500)
	}
}
