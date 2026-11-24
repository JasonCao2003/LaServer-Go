// 事务失败回滚测试

package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/handler"
)

func TestTask4ConcurrentUnlockPurchaseChargesOnce(t *testing.T) {
	api, db := newAPI(t, true)
	account := create(t, api)
	if _, err := db.Exec("INSERT INTO shop_item(name,unlockid,unlock_cost,currency_type) VALUES('parallel_unlock',990114,2,'candles')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE user_currency SET candles=2 WHERE user_id=?", account["user"]); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{"user": account["user"], "session": account["session"], "cost": 2, "unlock_id": 990114})
	var wg sync.WaitGroup
	results := make(chan string, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/account/purchase_unlock", bytes.NewReader(body))
			w := httptest.NewRecorder()
			api.ServeHTTP(w, req)
			var result map[string]any
			if json.Unmarshal(w.Body.Bytes(), &result) != nil {
				results <- "invalid_json"
				return
			}
			if w.Code != 200 {
				results <- "status_error"
				return
			}
			results <- result["result"].(string)
		}()
	}
	wg.Wait()
	close(results)
	seen := map[string]int{}
	for result := range results {
		seen[result]++
	}
	if seen["ok"] != 1 || seen["already"] != 1 {
		t.Fatalf("concurrent results: %v", seen)
	}
	var candles, owned int
	if err := db.QueryRow("SELECT candles FROM user_currency WHERE user_id=?", account["user"]).Scan(&candles); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT count(*) FROM unlock_item WHERE user_id=? AND name='parallel_unlock'", account["user"]).Scan(&owned); err != nil {
		t.Fatal(err)
	}
	if candles != 0 || owned != 1 {
		t.Fatalf("partial or duplicate purchase: candles=%d owned=%d", candles, owned)
	}
}

func TestTask4UnlockInsertFailureRollsBackDebit(t *testing.T) {
	api, db := newAPI(t, true)
	account := create(t, api)
	if _, err := db.Exec("INSERT INTO shop_item(name,unlockid,unlock_cost,currency_type) VALUES('rollback_unlock',990115,2,'candles')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE user_currency SET candles=2 WHERE user_id=?", account["user"]); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TRIGGER task4_fail_unlock BEFORE INSERT ON unlock_item WHEN NEW.name='rollback_unlock' BEGIN SELECT RAISE(ABORT,'injected failure'); END"); err != nil {
		t.Fatal(err)
	}
	request := map[string]any{"user": account["user"], "session": account["session"], "cost": 2, "unlock_id": 990115}
	call(t, api, "/account/purchase_unlock", request, 500)
	var candles, owned int
	if err := db.QueryRow("SELECT candles FROM user_currency WHERE user_id=?", account["user"]).Scan(&candles); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT count(*) FROM unlock_item WHERE user_id=? AND name='rollback_unlock'", account["user"]).Scan(&owned); err != nil {
		t.Fatal(err)
	}
	if candles != 2 || owned != 0 {
		t.Fatalf("failed purchase partly wrote: candles=%d owned=%d", candles, owned)
	}
}

func TestTask4ReceiptOrderFailureRollsBackGrant(t *testing.T) {
	api, db := newAPI(t, true)
	account := create(t, api)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "iaplist.json"), []byte(`{"iaplist":[{"id":"TEST_CANDLE","currencyType":"candles","currencyCount":5}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	enabled, err := New(db, handler.Config{LoginMode: "legacy_dev", Listen: "127.0.0.1:8443", RoomHost: "127.0.0.1", RoomPort: 25565, GameConfigDir: dir, AllowFakeCommerce: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TRIGGER task4_fail_order BEFORE INSERT ON commerce_order BEGIN SELECT RAISE(ABORT,'injected failure'); END"); err != nil {
		t.Fatal(err)
	}
	request := map[string]any{"user": account["user"], "session": account["session"], "platform": "fake", "product_id": "TEST_CANDLE", "transaction_id": "rollback-order"}
	call(t, enabled, "/account/commerce/receipt", request, 500)
	var candles, orders int
	if err := db.QueryRow("SELECT candles FROM user_currency WHERE user_id=?", account["user"]).Scan(&candles); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT count(*) FROM commerce_order WHERE transaction_id='rollback-order'").Scan(&orders); err != nil {
		t.Fatal(err)
	}
	if candles != 0 || orders != 0 {
		t.Fatalf("failed receipt partly wrote: candles=%d orders=%d", candles, orders)
	}
}

func TestTask4SeasonRewardFailureRollsBackClaimMarker(t *testing.T) {
	api, db := newAPI(t, true)
	account := create(t, api)
	auth := map[string]any{"user": account["user"], "session": account["session"]}
	call(t, api, "/account/get_season_quests", auth, 200)
	call(t, api, "/account/activate_season_quest", map[string]any{"user": account["user"], "session": account["session"], "quest_id": "fetchlight_mind"}, 200)
	call(t, api, "/account/set_achievement_stats", map[string]any{"user": account["user"], "session": account["session"], "achievement_stats": []any{map[string]any{"type": "fetchlight_mind_count", "value": 1}}}, 200)
	if _, err := db.Exec("CREATE TRIGGER task4_fail_reward BEFORE UPDATE OF season_candle ON user_currency BEGIN SELECT RAISE(ABORT,'injected failure'); END"); err != nil {
		t.Fatal(err)
	}
	claim := map[string]any{"user": account["user"], "session": account["session"], "quest_id": "fetchlight_mind", "in_season": 1}
	call(t, api, "/account/claim_season_quest_reward", claim, 500)
	var marked, balance int
	if err := db.QueryRow("SELECT rewarded FROM season_quest_reward WHERE user_id=? AND quest_id='fetchlight_mind'", account["user"]).Scan(&marked); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT season_candle FROM user_currency WHERE user_id=?", account["user"]).Scan(&balance); err != nil {
		t.Fatal(err)
	}
	if marked != 0 || balance != 0 {
		t.Fatalf("failed reward partly wrote: marked=%d balance=%d", marked, balance)
	}
	if _, err := db.Exec("DROP TRIGGER task4_fail_reward"); err != nil {
		t.Fatal(err)
	}
	call(t, api, "/account/claim_season_quest_reward", claim, 200)
	call(t, api, "/account/claim_season_quest_reward", claim, 200)
	if err := db.QueryRow("SELECT season_candle FROM user_currency WHERE user_id=?", account["user"]).Scan(&balance); err != nil || balance != 1 {
		t.Fatalf("reward after retry=%d err=%v", balance, err)
	}
}

func TestTask4RedeemBoxFailureRollsBackWalletAndBox(t *testing.T) {
	api, db := newAPI(t, true)
	account := create(t, api)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "spirit_shops.json"), []byte(`{"spirit_shops":[{"id":11,"nm":"lootbox_fixture","typ":"lootbox","ctyp":"candles","cst":1}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	enabled, err := New(db, handler.Config{LoginMode: "legacy_dev", Listen: "127.0.0.1:8443", RoomHost: "127.0.0.1", RoomPort: 25565, GameConfigDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE user_currency SET candles=1 WHERE user_id=?", account["user"]); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO lootbox(user_id,lootbox_id,quantity) VALUES(?,'lootbox_fixture',1)", account["user"]); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TRIGGER task4_fail_consumable BEFORE INSERT ON user_consumable BEGIN SELECT RAISE(ABORT,'injected failure'); END"); err != nil {
		t.Fatal(err)
	}
	request := map[string]any{"user": account["user"], "session": account["session"], "box_id": "lootbox_fixture"}
	call(t, enabled, "/account/lootboxes/redeem_box", request, 500)
	var candles, quantity, consumables int
	if err := db.QueryRow("SELECT candles FROM user_currency WHERE user_id=?", account["user"]).Scan(&candles); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT quantity FROM lootbox WHERE user_id=? AND lootbox_id='lootbox_fixture'", account["user"]).Scan(&quantity); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT count(*) FROM user_consumable WHERE user_id=?", account["user"]).Scan(&consumables); err != nil {
		t.Fatal(err)
	}
	if candles != 1 || quantity != 1 || consumables != 0 {
		t.Fatalf("failed redeem partly wrote: candles=%d quantity=%d consumables=%d", candles, quantity, consumables)
	}
	if _, err := db.Exec("DROP TRIGGER task4_fail_consumable"); err != nil {
		t.Fatal(err)
	}
	if out := call(t, enabled, "/account/lootboxes/redeem_box", request, 200); out["result"] != "ok" {
		t.Fatal(out)
	}
	if out := call(t, enabled, "/account/lootboxes/redeem_box", request, 200); out["result"] == "ok" {
		t.Fatal("redeemed empty box twice")
	}
}

func TestWingConversionRollbackAndRetry(t *testing.T) {
	api, db := newAPI(t, true)
	owner := create(t, api)
	req := map[string]any{"user": owner["user"], "session": owner["session"], "name": "baseline-wing"}
	call(t, api, "/account/wing_buffs/collect", req, 200)
	if _, err := db.Exec("CREATE TRIGGER reject_prestige BEFORE UPDATE OF prestige ON user_currency BEGIN SELECT RAISE(ABORT,'test rollback'); END"); err != nil {
		t.Fatal(err)
	}
	failure := call(t, api, "/account/wing_buffs/convert", req, 500)
	if failure["error"] != "Internal server error" {
		t.Fatalf("unexpected conversion error: %v", failure)
	}
	var deposited, stamp int64
	if err := db.QueryRow("SELECT deposited,last_conversion FROM user_wing_buff WHERE user_id=? AND name=?", owner["user"], req["name"]).Scan(&deposited, &stamp); err != nil || deposited != 0 || stamp != 0 {
		t.Fatalf("after rollback: deposited=%d stamp=%d err=%v", deposited, stamp, err)
	}
	if _, err := db.Exec("DROP TRIGGER reject_prestige"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []float64{1, 0} {
		got := call(t, api, "/account/wing_buffs/convert", req, 200)
		if got["converted"] != want || got["currency"].(map[string]any)["prestige"] != float64(1) {
			t.Fatalf("retry: converted=%v currency=%v", got["converted"], got["currency"])
		}
	}
}
