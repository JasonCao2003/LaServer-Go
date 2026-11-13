package handler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBatchBReceiptIsExplicitAndIdempotent(t *testing.T) {
	api, db := newAPI(t, true)
	owner, other := create(t, api), create(t, api)
	base := map[string]any{"user": owner["user"], "session": owner["session"], "platform": "fake", "product_id": "TEST_CANDLE", "transaction_id": "purchase-1"}
	call(t, api, "/account/commerce/receipt", base, 403)
	real := map[string]any{"user": owner["user"], "session": owner["session"], "platform": "android", "product_id": "TEST_CANDLE", "transaction_id": "purchase-1"}
	call(t, api, "/account/commerce/receipt", real, 501)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "iaplist.json"), []byte(`{"iaplist":[{"id":"TEST_CANDLE","currencyType":"candles","currencyCount":5}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	enabled, err := New(db, Config{RoomHost: "192.168.1.10", RoomPort: 25565, GameConfigDir: dir, AllowFakeCommerce: true})
	if err != nil {
		t.Fatal(err)
	}
	var before, after int64
	if err = db.QueryRow("SELECT candles FROM user_currency WHERE user_id=?", owner["user"]).Scan(&before); err != nil {
		t.Fatal(err)
	}
	first := call(t, enabled, "/account/commerce/receipt", base, 200)
	if first["result"] != "ok" {
		t.Fatalf("receipt: %v", first)
	}
	second := call(t, enabled, "/account/commerce/receipt", base, 200)
	if second["transactions"].([]any)[0].(map[string]any)["duplicate"] != true {
		t.Fatalf("duplicate receipt: %v", second)
	}
	if err = db.QueryRow("SELECT candles FROM user_currency WHERE user_id=?", owner["user"]).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after-before != 5 {
		t.Fatalf("grant repeated or missing: before=%d after=%d", before, after)
	}
	cross := map[string]any{"user": other["user"], "session": other["session"], "platform": "fake", "product_id": "TEST_CANDLE", "transaction_id": "purchase-1"}
	call(t, enabled, "/account/commerce/receipt", cross, 409)
	call(t, enabled, "/account/commerce/fulfill", map[string]any{"user": other["user"], "session": other["session"], "transaction_id": "purchase-1"}, 404)
	call(t, enabled, "/account/commerce/fulfill", map[string]any{"user": owner["user"], "session": owner["session"], "transaction_id": "purchase-1"}, 200)
}

func TestBatchBQuestCompletionAndSharedRewardMarker(t *testing.T) {
	api, db := newAPI(t, true)
	owner, other := create(t, api), create(t, api)
	auth := map[string]any{"user": owner["user"], "session": owner["session"]}
	call(t, api, "/account/get_season_quests", auth, 200)
	created := call(t, api, "/service/quest/api/v1/create", map[string]any{"user": owner["user"], "session": owner["session"], "quest_name": "fetchlight_mind"}, 200)
	id := created["quest_id"].(string)
	call(t, api, "/service/quest/api/v1/update_quest", map[string]any{"user": owner["user"], "session": owner["session"], "quest_id": id, "status": "succeeded"}, 409)
	call(t, api, "/service/quest/api/v1/update_quest", map[string]any{"user": owner["user"], "session": owner["session"], "quest_id": id, "collect": true}, 409)
	call(t, api, "/service/quest/api/v1/update_quest", map[string]any{"user": other["user"], "session": other["session"], "quest_id": id, "collect": true}, 404)
	call(t, api, "/account/set_achievement_stats", map[string]any{"user": owner["user"], "session": owner["session"], "achievement_stats": []any{map[string]any{"type": "fetchlight_mind_count", "value": 1}}}, 200)
	var before, after int64
	if err := db.QueryRow("SELECT season_candle FROM user_currency WHERE user_id=?", owner["user"]).Scan(&before); err != nil {
		t.Fatal(err)
	}
	claim := map[string]any{"user": owner["user"], "session": owner["session"], "quest_id": id, "collect": true}
	first := call(t, api, "/service/quest/api/v1/update_quest", claim, 200)
	if first["status"] != "collected" {
		t.Fatalf("quest not collected: %v", first)
	}
	call(t, api, "/service/quest/api/v1/update_quest", claim, 200)
	call(t, api, "/account/claim_season_quest_reward", map[string]any{"user": owner["user"], "session": owner["session"], "quest_id": "fetchlight_mind", "in_season": 1}, 200)
	if err := db.QueryRow("SELECT season_candle FROM user_currency WHERE user_id=?", owner["user"]).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after-before != 1 {
		t.Fatalf("quest reward duplicated: before=%d after=%d", before, after)
	}
}

func TestBatchBUnlockPurchaseDoesNotDoubleCharge(t *testing.T) {
	api, db := newAPI(t, true)
	owner, other := create(t, api), create(t, api)
	if _, err := db.Exec("INSERT INTO shop_item(name,unlockid,unlock_cost,currency_type) VALUES('test_unlock',990001,2,'candles')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE user_currency SET candles=5 WHERE user_id=?", owner["user"]); err != nil {
		t.Fatal(err)
	}
	buy := map[string]any{"user": owner["user"], "session": owner["session"], "cost": 2, "unlock_id": 990001}
	first := call(t, api, "/account/purchase_unlock", buy, 200)
	if first["result"] != "ok" {
		t.Fatalf("first purchase: %v", first)
	}
	second := call(t, api, "/account/purchase_unlock", buy, 200)
	if second["result"] != "already" {
		t.Fatalf("duplicate purchase: %v", second)
	}
	var balance int64
	if err := db.QueryRow("SELECT candles FROM user_currency WHERE user_id=?", owner["user"]).Scan(&balance); err != nil {
		t.Fatal(err)
	}
	if balance != 3 {
		t.Fatalf("unexpected balance: %d", balance)
	}
	poor := map[string]any{"user": other["user"], "session": other["session"], "cost": 2, "unlock_id": 990001}
	call(t, api, "/account/purchase_unlock", poor, 400)
	var owned int
	if err := db.QueryRow("SELECT COUNT(*) FROM unlock_item WHERE user_id=? AND name='test_unlock'", other["user"]).Scan(&owned); err != nil {
		t.Fatal(err)
	}
	if owned != 0 {
		t.Fatal("insufficient purchase granted unlock")
	}
}

func TestBatchBPickupAndWorldRewardRepeatProtection(t *testing.T) {
	api, db := newAPI(t, true)
	owner := create(t, api)
	pickup := map[string]any{"user": owner["user"], "session": owner["session"], "level_id": "test_level", "pickup_ids": []any{101, 101}, "global_pickup_ids": []any{4180017154}}
	var waxBefore, seasonBefore int64
	if err := db.QueryRow("SELECT wax,season_candle FROM user_currency WHERE user_id=?", owner["user"]).Scan(&waxBefore, &seasonBefore); err != nil {
		t.Fatal(err)
	}
	call(t, api, "/account/collect_pickup_batch", pickup, 200)
	call(t, api, "/account/collect_pickup_batch", pickup, 200)
	var waxAfter, seasonAfter int64
	if err := db.QueryRow("SELECT wax,season_candle FROM user_currency WHERE user_id=?", owner["user"]).Scan(&waxAfter, &seasonAfter); err != nil {
		t.Fatal(err)
	}
	if waxAfter-waxBefore != 4 || seasonAfter-seasonBefore != 1 {
		t.Fatalf("pickup repeat grant: wax %d->%d season %d->%d", waxBefore, waxAfter, seasonBefore, seasonAfter)
	}
	world := map[string]any{"user": owner["user"], "session": owner["session"], "name": "test_world", "bonus_percent": 0}
	first := call(t, api, "/account/claim_quest_reward", world, 200)
	second := call(t, api, "/account/claim_quest_reward", world, 200)
	if first["bonus_amount"] != float64(15) || second["bonus_amount"] != float64(0) {
		t.Fatalf("world reward repeat: %v %v", first, second)
	}
}

func TestBatchBSeasonAccountClaimRequiresCompletion(t *testing.T) {
	api, db := newAPI(t, true)
	owner, other := create(t, api), create(t, api)
	auth := map[string]any{"user": owner["user"], "session": owner["session"]}
	call(t, api, "/account/get_season_quests", auth, 200)
	claim := map[string]any{"user": owner["user"], "session": owner["session"], "quest_id": "fetchlight_mind", "in_season": 1}
	call(t, api, "/account/claim_season_quest_reward", claim, 409)
	call(t, api, "/account/claim_season_quest_reward", map[string]any{"user": other["user"], "session": other["session"], "quest_id": "fetchlight_mind", "in_season": 1}, 404)
	call(t, api, "/account/activate_season_quest", claim, 200)
	call(t, api, "/account/claim_season_quest_reward", claim, 409)
	call(t, api, "/account/set_achievement_stats", map[string]any{"user": owner["user"], "session": owner["session"], "achievement_stats": []any{map[string]any{"type": "fetchlight_mind_count", "value": 1}}}, 200)
	var before, after int64
	if err := db.QueryRow("SELECT season_candle FROM user_currency WHERE user_id=?", owner["user"]).Scan(&before); err != nil {
		t.Fatal(err)
	}
	call(t, api, "/account/claim_season_quest_reward", claim, 200)
	call(t, api, "/account/claim_season_quest_reward", claim, 200)
	if err := db.QueryRow("SELECT season_candle FROM user_currency WHERE user_id=?", owner["user"]).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after-before != 1 {
		t.Fatalf("season reward repeated: %d -> %d", before, after)
	}
}
