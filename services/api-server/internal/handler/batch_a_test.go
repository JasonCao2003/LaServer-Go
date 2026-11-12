package handler

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestBatchAResourceFailureAndAuth(t *testing.T) {
	api, _ := newAPI(t, true)
	account := create(t, api)
	valid := map[string]any{"user": account["user"], "session": account["session"]}
	invalid := map[string]any{"user": account["user"], "session": "wrong"}
	call(t, api, "/account/get_radiance_spawn_defs", valid, 503)
	call(t, api, "/service/auth/api/v1/token", valid, 503)
	call(t, api, "/account/remote_config/upload", map[string]any{}, 400)
	call(t, api, "/account/remote_config/upload", invalid, 401)
	ok := call(t, api, "/account/remote_config/upload", valid, 200)
	if ok["result"] != "ok" {
		t.Fatalf("upload acknowledgement: %v", ok)
	}
	call(t, api, "/account/get_level_pickups", valid, 400)
	invalid["level"] = 1
	call(t, api, "/account/get_level_pickups", invalid, 401)
	valid["level"] = 1
	result := call(t, api, "/account/get_level_pickups", valid, 200)
	if result["level"] != float64(1) || result["pickup_emitters"] == nil {
		t.Fatalf("pickups shape: %v", result)
	}
}

func TestBatchAResourceRouteContracts(t *testing.T) {
	api, _ := newAPI(t, true)
	account := create(t, api)
	auth := map[string]any{"user": account["user"], "session": account["session"]}
	for _, tc := range []struct {
		path   string
		body   map[string]any
		status int
		key    string
	}{
		{"/account/get_surveys", auth, 200, "surveys"},
		{"/account/get_event_schedule", auth, 200, "event_schedule"},
		{"/account/get_spirit_shops", auth, 503, "error"},
		{"/account/get_currency_defs", auth, 200, "currency_defs"},
		{"/account/trust/get_trust_tier_defs", auth, 200, "get_trust_tier_defs"},
		{"/account/get_video_defs", auth, 200, "video_defs"},
		{"/account/get_radiance_spawn_defs", auth, 503, "error"},
		{"/account/get_level_pickups", map[string]any{"user": account["user"], "session": account["session"], "level": 1}, 200, "pickup_emitters"},
		{"/account/trust/get_trust_group_defs", auth, 200, "get_trust_group_defs"},
		{"/account/remote_config/upload", auth, 200, "result"},
		{"/account/trust/get_trust_activity_defs", auth, 200, "get_trust_activity_defs"},
		{"/account/get_generic_shop_item_tracking_info", auth, 200, "generic_shop_item_tracking_info"},
		{"/service/auth/api/v1/token", auth, 503, "error"},
		{"/service/infraction/api/v1/get_infractions_status_decorated", auth, 200, "status"},
		{"/starwatch/auth/api/v1/auth", map[string]any{"sessionToken": account["session"], "deviceId": "test"}, 503, "error"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			out := call(t, api, tc.path, tc.body, tc.status)
			if _, ok := out[tc.key]; !ok {
				t.Fatalf("missing %s: %v", tc.key, out)
			}
		})
	}
}

func TestBatchAStatusUnlockLifecycle(t *testing.T) {
	api, _ := newAPI(t, true)
	owner, other := create(t, api), create(t, api)
	req := map[string]any{"user": owner["user"], "session": owner["session"], "name": "test_level", "ack": false}
	call(t, api, "/service/status/api/v1/add_unlock", req, 200)
	call(t, api, "/service/status/api/v1/add_unlock", req, 200)
	status := call(t, api, "/service/status/api/v1/get_unlocks", map[string]any{"user": owner["user"], "session": owner["session"]}, 200)
	if status["status_unlocks_total_count"] != float64(1) {
		t.Fatalf("owner unlocks: %v", status)
	}
	otherStatus := call(t, api, "/service/status/api/v1/get_unlocks", map[string]any{"user": other["user"], "session": other["session"]}, 200)
	if otherStatus["status_unlocks_total_count"] != float64(0) {
		t.Fatalf("cross-account unlock: %v", otherStatus)
	}
	call(t, api, "/service/status/api/v1/delete_unlock", req, 200)
	status = call(t, api, "/service/status/api/v1/get_unlocks", map[string]any{"user": owner["user"], "session": owner["session"]}, 200)
	if status["status_unlocks_total_count"] != float64(0) {
		t.Fatalf("unlock not removed: %v", status)
	}
	req["session"] = other["session"]
	call(t, api, "/service/status/api/v1/add_unlock", req, 401)
}

func TestBatchAWingBuffLifecycle(t *testing.T) {
	api, db := newAPI(t, true)
	owner, other := create(t, api), create(t, api)
	req := map[string]any{"user": owner["user"], "session": owner["session"], "name": "test_wing"}
	call(t, api, "/account/wing_buffs/collect", req, 200)
	wings := call(t, api, "/account/wing_buffs/get", req, 200)
	if len(wings["wing_buffs"].([]any)) != 1 {
		t.Fatalf("collected wing: %v", wings)
	}
	otherWings := call(t, api, "/account/wing_buffs/get", map[string]any{"user": other["user"], "session": other["session"]}, 200)
	if len(otherWings["wing_buffs"].([]any)) != 0 {
		t.Fatalf("cross-account wing: %v", otherWings)
	}
	call(t, api, "/account/wing_buffs/drop", req, 200)
	call(t, api, "/account/wing_buffs/collect", req, 200)
	call(t, api, "/account/wing_buffs/convert", req, 200)
	call(t, api, "/account/wing_buffs/convert", req, 200)
	var prestige int64
	if err := db.QueryRow("SELECT prestige FROM user_currency WHERE user_id=?", owner["user"]).Scan(&prestige); err != nil {
		t.Fatal(err)
	}
	if prestige != 1 {
		t.Fatalf("repeated conversion granted %d prestige", prestige)
	}
	req["session"] = other["session"]
	call(t, api, "/account/wing_buffs/drop", req, 401)
}

func TestBatchAMagicDefinitionsAndAccountIsolation(t *testing.T) {
	api, _ := newAPI(t, true)
	owner, other := create(t, api), create(t, api)
	req := map[string]any{"user": owner["user"], "session": owner["session"]}
	defs := call(t, api, "/account/consumable/get_consumable_defs", req, 200)
	if defs["get_consumable_defs"] == nil {
		t.Fatalf("missing consumable definitions: %v", defs)
	}
	buffs := call(t, api, "/account/buff/get_buff_defs", req, 200)
	if buffs["get_buff_defs"] == nil {
		t.Fatalf("missing buff definitions: %v", buffs)
	}
	call(t, api, "/account/consumable/get_consumables1", req, 200)
	call(t, api, "/account/buff/get_buffs1", req, 200)
	req["session"] = other["session"]
	call(t, api, "/account/consumable/get_consumables1", req, 401)
	call(t, api, "/account/buff/get_buffs1", req, 401)
}

func TestBatchAStateRouteContracts(t *testing.T) {
	api, _ := newAPI(t, true)
	account := create(t, api)
	base := map[string]any{"user": account["user"], "session": account["session"]}
	with := func(key string, value any) map[string]any {
		m := map[string]any{"user": account["user"], "session": account["session"]}
		m[key] = value
		return m
	}
	for _, tc := range []struct {
		path   string
		body   map[string]any
		status int
		key    string
	}{
		{"/account/consumable/get_consumable_defs", base, 200, "get_consumable_defs"},
		{"/account/consumable/get_consumables", base, 200, "get_consumables"},
		{"/account/consumable/consume_item1", base, 200, "consume_item"},
		{"/account/wing_buffs/get", base, 200, "wing_buffs"},
		{"/account/wing_buffs/collect", with("name", "route_wing"), 200, "update_wing_buffs"},
		{"/account/collect_collectible", with("name", "route_collectible"), 200, "collectibles"},
		{"/account/wing_buffs/drop", with("name", "route_wing"), 200, "update_wing_buffs"},
		{"/account/wing_buffs/deposit", with("name_deposit_id_pairs", []any{[]any{"route_wing", 1}}), 200, "update_wing_buffs"},
		{"/account/wing_buffs/convert", base, 200, "wing_buffs"},
		{"/account/buff/get_buffs1", base, 200, "get_buffs"},
		{"/account/buff/get_buffs", base, 200, "get_buffs"},
		{"/account/buff/remove_buff1", base, 200, "remove_buff"},
		{"/account/buff/remove_buff", base, 200, "remove_buff"},
		{"/account/consumable/get_consumables1", base, 200, "get_consumables"},
		{"/account/ack_collectible", base, 200, ""},
		{"/account/buff/get_buff_defs", base, 200, "get_buff_defs"},
		{"/service/status/api/v1/add_unlock", map[string]any{"user": account["user"], "session": account["session"], "name": "route_unlock", "ack": false}, 200, "update_status_unlocks"},
		{"/service/status/api/v1/delete_unlock", with("name", "route_unlock"), 200, "delete_status_unlocks"},
		{"/service/status/api/v1/get_unlocks", base, 200, "status_unlocks"},
		{"/service/inventory/api/v1/unlocks/remove_dye", base, 200, "status"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			out := call(t, api, tc.path, tc.body, tc.status)
			if tc.key != "" {
				if _, ok := out[tc.key]; !ok {
					t.Fatalf("missing %s: %v", tc.key, out)
				}
			}
		})
	}
}

func TestBatchABaselineReadContracts(t *testing.T) {
	api, _ := newAPI(t, true)
	account := create(t, api)
	auth := map[string]any{"user": account["user"], "session": account["session"]}
	for _, tc := range []struct {
		path   string
		status int
		key    string
	}{
		{"/account/get_rank", 200, "set_rank"},
		{"/account/support/has_reply", 200, "has_reply"},
		{"/account/get_achievement_stats_tracking", 200, "achievement_stats"},
		{"/account/get_event_currency_defs", 200, "event_currency_defs"},
		{"/account/get_questionnaires", 200, "result"},
		{"/account/get_generic_shops", 200, "generic_shop_defs"},
		{"/account/get_forge_rates", 200, "forge_rates"},
		{"/account/get_achievement_stats", 200, "achievement_stats"},
		{"/account/get_relationship_abilities", 200, "relationship_defs"},
		{"/account/get_unlocks", 200, "unlocks"},
		{"/account/iaplist", 503, "error"},
		{"/account/get_questionnaire_groups", 200, "get_questionnaire_groups"},
		{"/account/get_checkpoints", 200, "checkpoints"},
		{"/account/get_invites", 200, "invites"},
		{"/account/get_outfit", 200, "set_outfit"},
		{"/account/quest/get_quest_defs", 503, "error"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			out := call(t, api, tc.path, auth, tc.status)
			if _, ok := out[tc.key]; !ok {
				t.Fatalf("missing %s: %v", tc.key, out)
			}
		})
	}
}

func TestBatchAInviteLifecycleAndAck(t *testing.T) {
	api, _ := newAPI(t, true)
	a, b := create(t, api), create(t, api)
	created := call(t, api, "/account/create_invite", map[string]any{"user": a["user"], "session": a["session"], "invite_nickname": "friend"}, 200)
	token := created["invite_token_id"].(string)
	check := call(t, api, "/account/check_invite", map[string]any{"user": b["user"], "session": b["session"], "token_id": token}, 200)
	if check["invite_exists"] != true {
		t.Fatalf("invite missing: %v", check)
	}
	accepted := call(t, api, "/account/accept_invite", map[string]any{"user": b["user"], "session": b["session"], "token_id": token}, 200)
	if accepted["invite_accepted"] != true {
		t.Fatalf("invite not accepted: %v", accepted)
	}
	call(t, api, "/account/accept_invite", map[string]any{"user": b["user"], "session": b["session"], "token_id": token}, 400)
	invites := call(t, api, "/account/get_invites", map[string]any{"user": a["user"], "session": a["session"]}, 200)
	if len(invites["invites"].([]any)) != 1 {
		t.Fatalf("invites not persisted: %v", invites)
	}
	call(t, api, "/account/delete_invite", map[string]any{"user": b["user"], "session": b["session"], "token_id": token}, 403)
	call(t, api, "/account/delete_invite", map[string]any{"user": a["user"], "session": a["session"], "token_id": token}, 200)
	ack := call(t, api, "/account/ack_unlock", map[string]any{"user": a["user"], "session": a["session"], "name": "test_unlock"}, 200)
	if ack["result"] != true {
		t.Fatalf("ack failed: %v", ack)
	}
	bad := call(t, api, "/account/ack_unlock", map[string]any{"user": a["user"], "session": b["session"], "name": "test_unlock"}, 200)
	if bad["result"] != "unauthorized" {
		t.Fatalf("cross-account ack: %v", bad)
	}
}

func TestBatchAConsumableRejectsInsufficientStock(t *testing.T) {
	api, db := newAPI(t, true)
	owner, other := create(t, api), create(t, api)
	const itemID = 1737402210
	if _, err := db.Exec("INSERT INTO user_consumable(user_id,consumable_id,quantity) VALUES(?,?,1)", owner["user"], itemID); err != nil {
		t.Fatal(err)
	}
	request := map[string]any{"user": owner["user"], "session": owner["session"], "consumable_id": itemID, "consumable_count": 2}
	response := call(t, api, "/account/consumable/consume_item1", request, 200)
	if response["consume_item"] != "failed" {
		t.Fatalf("overdraw accepted: %v", response)
	}
	var quantity int
	if err := db.QueryRow("SELECT quantity FROM user_consumable WHERE user_id=? AND consumable_id=?", owner["user"], itemID).Scan(&quantity); err != nil {
		t.Fatal(err)
	}
	if quantity != 1 {
		t.Fatalf("stock changed after failed consume: %d", quantity)
	}
	request["consumable_count"] = 1
	response = call(t, api, "/account/consumable/consume_item1", request, 200)
	if response["consume_item"] != "success" {
		t.Fatalf("owned item not consumed: %v", response)
	}
	response = call(t, api, "/account/consumable/consume_item1", request, 200)
	if response["consume_item"] != "failed" {
		t.Fatalf("repeat consume accepted: %v", response)
	}
	request["session"] = other["session"]
	call(t, api, "/account/consumable/consume_item1", request, 401)
}

func TestBatchAConfiguredResourceRejectsCorruption(t *testing.T) {
	api, db := newAPI(t, true)
	account := create(t, api)
	dir := t.TempDir()
	configured, err := New(db, Config{RoomHost: "192.168.1.10", RoomPort: 25565, GameConfigDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	request := map[string]any{"user": account["user"], "session": account["session"]}
	call(t, configured, "/account/get_radiance_spawn_defs", request, 503)
	path := filepath.Join(dir, "radiance_spawn.json")
	if err = os.WriteFile(path, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	call(t, configured, "/account/get_radiance_spawn_defs", request, 503)
	if err = os.WriteFile(path, []byte(`{"radiance_spawn_defs":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	response := call(t, configured, "/account/get_radiance_spawn_defs", request, 200)
	if _, ok := response["radiance_spawn_defs"]; !ok {
		t.Fatalf("configured resource missing: %v", response)
	}
	spirit := filepath.Join(dir, "spirit_shops.json")
	if err = os.WriteFile(spirit, []byte(`{"spirit_shops":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	response = call(t, configured, "/account/get_spirit_shops", request, 200)
	if _, ok := response["spirit_shops"]; !ok {
		t.Fatalf("configured spirit shop missing: %v", response)
	}
}

func TestBatchAMagicDefinitionsFilterByClientVersion(t *testing.T) {
	api, _ := newAPI(t, true)
	account := create(t, api)
	body, _ := json.Marshal(map[string]any{"user": account["user"], "session": account["session"]})
	contains := func(agent string) bool {
		req := httptest.NewRequest("POST", "/account/consumable/get_consumable_defs", bytes.NewReader(body))
		req.Header.Set("User-Agent", agent)
		resp := httptest.NewRecorder()
		api.ServeHTTP(resp, req)
		if resp.Code != 200 {
			t.Fatalf("%s: %d %s", agent, resp.Code, resp.Body.String())
		}
		var result map[string]any
		if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		for _, raw := range result["get_consumable_defs"].([]any) {
			if raw.(map[string]any)["id"] == float64(1954164294) {
				return true
			}
		}
		return false
	}
	if contains("Sky-Live-ios/0.1.8.100") {
		t.Fatal("new consumable leaked to 0.1.8")
	}
	if contains("Sky-Live-ios/0.3.11.100") {
		t.Fatal("new consumable leaked to 0.3.11")
	}
	if !contains("Sky-Live-ios/0.19.5.100") {
		t.Fatal("supported consumable missing from 0.19.5")
	}
}
