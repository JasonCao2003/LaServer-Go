package handler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// Set LASERVER_GAME_CONFIG_DIR to run the read-only old-config smoke test locally.
func TestTask3LegacyConfigCandidateResources(t *testing.T) {
	dir := os.Getenv("LASERVER_GAME_CONFIG_DIR")
	if dir == "" {
		t.Skip("LASERVER_GAME_CONFIG_DIR not set")
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("legacy config directory unavailable: %q", dir)
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "json", "api", "task1-login-resources.json"))
	if err != nil {
		t.Fatal(err)
	}
	var inventory struct {
		Resources []struct {
			Resource string `json:"resource"`
			Path     string `json:"candidate_go_path"`
			Kind     string `json:"candidate_kind"`
		} `json:"resources"`
	}
	if err := json.Unmarshal(raw, &inventory); err != nil {
		t.Fatal(err)
	}
	api, _ := newAPI(t, true)
	account := create(t, api)
	api.cfg.GameConfigDir = dir
	login := call(t, api, "/account/auth/login", map[string]any{"user": account["user"]}, 200)
	if _, ok := login["external_links"].([]any); !ok {
		t.Fatalf("login omitted embedded external_links: %v", sortedObjectKeys(login))
	}
	info := login["authinfo"].(map[string]any)
	body := map[string]any{"user": info["user"], "user_id": info["user"], "session": login["session"], "level": 0}
	paths := map[string][]string{}
	responseKeys := map[string][]string{}
	aliases := map[string][]string{
		"free_gifts":        {"set_recvd_free_gifts", "set_sent_free_gifts"},
		"gift_messages":     {"set_recvd_messages", "set_sent_messages"},
		"infractions":       {"status", "access_status"},
		"invite_list":       {"invites"},
		"questionnaires":    {"result"},
		"set_friend_status": {"set_friend_statues"},
		"set_random":        {"set_rank"},
	}
	for _, resource := range inventory.Resources {
		if resource.Path != "" && resource.Path != "/account/auth/login" {
			paths[resource.Path] = append(paths[resource.Path], resource.Resource)
			responseKeys[resource.Path] = append(responseKeys[resource.Path], aliases[resource.Resource]...)
			if resource.Kind == "response_key_match" {
				key := resource.Resource
				// The old dynamic tracking handler and iaplist configuration use these keys.
				// The Task 1 candidate name is not itself a verified response key.
				switch key {
				case "achievement_stats_tracking":
					key = "achievement_stats"
				case "iap_list":
					key = "iaplist"
				}
				responseKeys[resource.Path] = append(responseKeys[resource.Path], key)
			}
		}
	}
	ordered := make([]string, 0, len(paths))
	for path := range paths {
		ordered = append(ordered, path)
	}
	if len(inventory.Resources) != 48 || len(ordered) != 42 {
		t.Fatalf("resource inventory changed: resources=%d candidate_paths=%d; update this audit and its documentation", len(inventory.Resources), len(ordered))
	}
	sort.Strings(ordered)
	for _, ua := range []string{"Sky-Live-ios/0.1.8.100", "Sky-Live-ios/0.3.11.100", "Sky-Live-ios/0.13.4.170388"} {
		for _, path := range ordered {
			response := task3CallWithUA(t, api, path, ua, body, 200)
			if response == nil {
				t.Fatalf("%s %s returned no JSON object", ua, path)
			}
			for _, key := range responseKeys[path] {
				if _, ok := response[key]; !ok {
					t.Fatalf("%s %s missing response key %q; got %v", ua, path, key, sortedObjectKeys(response))
				}
			}
			t.Logf("ua=%s path=%s resources=%v status=200 keys=%v", ua, path, paths[path], sortedObjectKeys(response))
		}
	}
}

func sortedObjectKeys(value map[string]any) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
