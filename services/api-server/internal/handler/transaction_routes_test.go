// 交易相关 HTTP 入口契约测试

package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Every batch-B entry is called through HTTP. These are entry contract checks;
// transaction, persistence, and reward edge cases live in batch_b_test.go.
func TestBatchBDirectHTTPEntries(t *testing.T) {
	api, _ := newAPI(t, true)
	account := create(t, api)
	cases := []struct {
		path string
		key  string
	}{
		{"/account/get_shop", "get_shop"},
		{"/account/lootboxes/get", "get_lootboxes"},
		{"/account/lootboxes/redeem_box", "result"},
		{"/account/redeem_reward", "result"},
		{"/account/commerce/receipt", "result"},
		{"/account/commerce/fulfill", "result"},
		{"/account/commerce/update", "result"},
		{"/account/commerce/product_info", "product_info"},
		{"/account/purchase_spirit_shop_item", "result"},
		{"/account/purchase_event_buff", "result"},
		{"/account/purchase_generic_shop_item", "generic_shop_purchase"},
		{"/account/purchase_unlock", "result"},
		{"/account/ack_forge_rate_reset", "update_forge_rates"},
		{"/account/buy_candle_wax", "result"},
		{"/account/code/verify", "result"},
		{"/account/purchase_dye", "error"},
		{"/account/set_achievement_stats", "error"},
		{"/account/get_daily_quests", "daily_quests"},
		{"/account/get_weekly_quests", "weekly_quests"},
		{"/account/get_season_quests_v2", "season_quests"},
		{"/account/get_season_pass_defs", "season_pass_defs"},
		{"/account/get_event_currency", "event_currency"},
		{"/account/season_finalize", "currency"},
		{"/account/season_start", "result"},
		{"/account/activate_season_quest", "error"},
		{"/account/claim_season_quest_reward", "error"},
		{"/account/get_season_quests", "season_quests"},
		{"/account/get_account_world_quests", "set_world_quests"},
		{"/account/claim_quest_reward", "error"},
		{"/account/collect_pickup_batch", "error"},
		{"/service/quest/api/v1/check_for_previous_completion", "results"},
		{"/service/quest/api/v1/get_player_quests", "active"},
		{"/service/quest/api/v1/get_player_quests_by_name", "results"},
		{"/service/quest/api/v1/get_quest_schedule", "quest_schedule"},
		{"/service/quest/api/v1/create", "error"},
		{"/service/quest/api/v1/update_quest", "error"},
	}
	if len(cases) != 36 {
		t.Fatalf("batch B route count = %d, want 36", len(cases))
	}
	badRequest := map[string]bool{
		"/account/commerce/fulfill":          true,
		"/account/purchase_unlock":           true,
		"/account/code/verify":               true,
		"/account/purchase_dye":              true,
		"/account/set_achievement_stats":     true,
		"/account/activate_season_quest":     true,
		"/account/claim_season_quest_reward": true,
		"/account/claim_quest_reward":        true,
		"/account/collect_pickup_batch":      true,
		"/service/quest/api/v1/create":       true,
		"/service/quest/api/v1/update_quest": true,
	}
	sessionRejected := map[string]bool{
		"/account/get_shop":                  true,
		"/account/lootboxes/get":             true,
		"/account/commerce/receipt":          true,
		"/account/commerce/fulfill":          true,
		"/account/purchase_unlock":           true,
		"/account/code/verify":               true,
		"/account/purchase_dye":              true,
		"/account/set_achievement_stats":     true,
		"/account/season_finalize":           true,
		"/account/season_start":              true,
		"/account/activate_season_quest":     true,
		"/account/claim_season_quest_reward": true,
		"/account/get_season_quests":         true,
		"/account/get_account_world_quests":  true,
		"/account/claim_quest_reward":        true,
		"/account/collect_pickup_batch":      true,
		"/service/quest/api/v1/create":       true,
		"/service/quest/api/v1/update_quest": true,
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			body := map[string]any{"user": account["user"], "session": account["session"]}
			encoded, err := json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
			post := httptest.NewRecorder()
			api.ServeHTTP(post, httptest.NewRequest(http.MethodPost, tc.path, bytes.NewReader(encoded)))
			wantStatus := 200
			if badRequest[tc.path] {
				wantStatus = 400
			} else if tc.path == "/account/commerce/receipt" {
				wantStatus = 501 // Real receipt verification is deliberately unavailable.
			}
			if post.Code != wantStatus {
				t.Fatalf("POST status=%d want=%d body=%s", post.Code, wantStatus, post.Body.String())
			}
			var response map[string]any
			if err := json.Unmarshal(post.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if _, ok := response[tc.key]; !ok {
				t.Fatalf("POST status=%d missing %q: %v", post.Code, tc.key, response)
			}
			t.Logf("POST status=%d key=%s", post.Code, tc.key)

			invalid := httptest.NewRecorder()
			api.ServeHTTP(invalid, httptest.NewRequest(http.MethodPost, tc.path, bytes.NewBufferString("{")))
			if invalid.Code != 400 {
				t.Fatalf("invalid JSON status=%d, want 400", invalid.Code)
			}
			get := httptest.NewRecorder()
			api.ServeHTTP(get, httptest.NewRequest(http.MethodGet, tc.path, nil))
			wantMethod := 405
			if tc.path == "/account/commerce/product_info" {
				wantMethod = 200
			}
			if get.Code != wantMethod {
				t.Fatalf("GET status=%d, want %d", get.Code, wantMethod)
			}
			authBody := map[string]any{"user": account["user"], "session": "invalid-session"}
			switch tc.path {
			case "/account/lootboxes/redeem_box":
				authBody["box_id"] = "missing-box"
			case "/account/purchase_unlock":
				authBody["cost"] = 0
			case "/account/code/verify":
				authBody["code"] = "AAAA-BBBB-CCCC"
			}
			badSessionBody, err := json.Marshal(authBody)
			if err != nil {
				t.Fatal(err)
			}
			badSession := httptest.NewRecorder()
			api.ServeHTTP(badSession, httptest.NewRequest(http.MethodPost, tc.path, bytes.NewReader(badSessionBody)))
			if sessionRejected[tc.path] {
				if badSession.Code != 401 {
					t.Fatalf("invalid session status=%d want=401 body=%s", badSession.Code, badSession.Body.String())
				}
			} else if badSession.Code >= 500 {
				t.Fatalf("invalid session caused server error: status=%d body=%s", badSession.Code, badSession.Body.String())
			}
		})
	}
}
