// 数据库重开后的幂等与持久化测试

package handler_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/handler"
)

func TestTask4PurchaseAndRewardStayIdempotentAfterDatabaseReopen(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "api.db")
	if err := os.WriteFile(filepath.Join(root, "iaplist.json"), []byte(`{"iaplist":[{"id":"TEST_CANDLE","currencyType":"candles","currencyCount":5}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	config := handler.Config{AutoAuthorize: true, LoginMode: "legacy_dev", Listen: "127.0.0.1:8443", GameConfigDir: root, AllowFakeCommerce: true, RoomHost: "127.0.0.1", RoomPort: 25565}
	api, err := New(db, config)
	if err != nil {
		t.Fatal(err)
	}
	a := create(t, api)
	if _, err := db.Exec("INSERT INTO shop_item(name,unlockid,unlock_cost,currency_type) VALUES('restart_unlock',990121,2,'candles')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE user_currency SET candles=2 WHERE user_id=?", a["user"]); err != nil {
		t.Fatal(err)
	}
	buy := map[string]any{"user": a["user"], "session": a["session"], "cost": 2, "unlock_id": 990121}
	if out := call(t, api, "/account/purchase_unlock", buy, 200); out["result"] != "ok" {
		t.Fatal(out)
	}
	receipt := map[string]any{"user": a["user"], "session": a["session"], "platform": "fake", "product_id": "TEST_CANDLE", "transaction_id": "restart-receipt"}
	call(t, api, "/account/commerce/receipt", receipt, 200)
	call(t, api, "/account/get_season_quests", map[string]any{"user": a["user"], "session": a["session"]}, 200)
	quest := map[string]any{"user": a["user"], "session": a["session"], "quest_id": "fetchlight_mind", "in_season": 1}
	call(t, api, "/account/activate_season_quest", quest, 200)
	call(t, api, "/account/set_achievement_stats", map[string]any{"user": a["user"], "session": a["session"], "achievement_stats": []any{map[string]any{"type": "fetchlight_mind_count", "value": 1}}}, 200)
	call(t, api, "/account/claim_season_quest_reward", quest, 200)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	restarted, err := New(db, config)
	if err != nil {
		t.Fatal(err)
	}
	if out := call(t, restarted, "/account/purchase_unlock", buy, 200); out["result"] != "already" {
		t.Fatal("duplicate unlock purchase after reopen", out)
	}
	if out := call(t, restarted, "/account/commerce/receipt", receipt, 200); out["transactions"].([]any)[0].(map[string]any)["duplicate"] != true {
		t.Fatal("duplicate receipt after reopen", out)
	}
	call(t, restarted, "/account/claim_season_quest_reward", quest, 200)
	if n := cQueryInt(t, db, "SELECT candles FROM user_currency WHERE user_id=?", a["user"]); n != 5 {
		t.Fatalf("candles after replay=%d", n)
	}
	if n := cQueryInt(t, db, "SELECT season_candle FROM user_currency WHERE user_id=?", a["user"]); n != 1 {
		t.Fatalf("season reward after replay=%d", n)
	}
}

// A fresh connection must retain durable state, while chat's process-local queue expires.
func TestTask4SocialStateAfterDatabaseReopen(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "api.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	config := handler.Config{AutoAuthorize: true, LoginMode: "legacy_dev", Listen: "127.0.0.1:8443", InstanceDir: root, RoomHost: "127.0.0.1", RoomPort: 25565}
	api, err := New(db, config)
	if err != nil {
		t.Fatal(err)
	}
	a, b := create(t, api), create(t, api)
	call(t, api, "/service/message/api/v1/mark_received", map[string]any{"user": b["user"], "session": b["session"], "messages": []any{map[string]any{"id": "task4-mail"}}}, 200)
	call(t, api, "/service/relationship/api/v1/free_gifts/send", map[string]any{"user": a["user"], "session": a["session"], "target": b["user"], "gift_type": "gift_scandle"}, 200)
	feed := call(t, api, "/service/social-feed/api/v1/create", map[string]any{"user": a["user"], "session": a["session"], "pool_type": "test", "pool_name": "test", "message": "task4-feed"}, 200)
	clip := call(t, api, "/service/recording/api/v2/create", map[string]any{"user": a["user"], "session": a["session"], "user_id": a["user"], "recording_data": map[string]any{"chet": 1}}, 200)
	call(t, api, "/account/chat/send", map[string]any{"user": a["user"], "session": a["session"], "msg": "transient"}, 200)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	restarted, err := New(db, config)
	if err != nil {
		t.Fatal(err)
	}
	if n := cQueryInt(t, db, "SELECT COUNT(*) FROM message WHERE user_id=? AND id='task4-mail'", b["user"]); n != 1 {
		t.Fatalf("message state after reopen=%d", n)
	}
	if n := cQueryInt(t, db, "SELECT COUNT(*) FROM message WHERE user_id=? AND id='task4-mail'", a["user"]); n != 0 {
		t.Fatalf("cross-account message state=%d", n)
	}
	pending := call(t, restarted, "/service/relationship/api/v1/free_gifts/get_pending", map[string]any{"user": b["user"], "session": b["session"]}, 200)
	if len(pending["set_recvd_free_gifts"].([]any)) != 1 {
		t.Fatal("gift lost after reopen", pending)
	}
	feeds := call(t, restarted, "/service/social-feed/api/v1/get_by_ids", map[string]any{"social_feed_ids": []string{feed["social_feed_id"].(string)}}, 200)
	if len(feeds["feeds"].([]any)) != 1 {
		t.Fatal("feed lost after reopen", feeds)
	}
	recording := call(t, restarted, "/service/recording/api/v2/get", map[string]any{"recording_id": clip["recording_id"]}, 200)
	if recording["status"] != "OK" || len(recording["seek_table"].(map[string]any)) != 1 {
		t.Fatal("recording lost after reopen", recording)
	}
	chat := call(t, restarted, "/account/get_pending_messages", map[string]any{"user": b["user"], "session": b["session"]}, 200)
	entries := chat["set_recvd_messages"].([]any)
	if len(entries) != 1 || entries[0].(map[string]any)["type"] != "gift_scandle" {
		t.Fatal("durable gift or transient chat after restart differs", chat)
	}
}
