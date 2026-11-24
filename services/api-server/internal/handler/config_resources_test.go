// 配置资源缺失/损坏回退测试

package handler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTask3ConfiguredLoginResourcesReportMissingAndCorruptSources(t *testing.T) {
	for _, tc := range []struct {
		path, filename, key, valid string
	}{
		{"/account/get_map_defs", "map.json", "get_map_defs", `{"get_map_defs":[]}`},
		{"/account/quest/get_quest_defs", "quest.json", "get_quest_defs", `{"get_quest_defs":[]}`},
		{"/account/iaplist", "iaplist.json", "iaplist", `{"iaplist":[]}`},
		{"/account/get_collectibles", "collectibles.json", "collectibles", `{"collectibles":[]}`},
		{"/account/get_spirit_shops", "spirit_shops.json", "spirit_shops", `{"spirit_shops":[],"spirit_shops_version":1}`},
	} {
		t.Run(tc.filename, func(t *testing.T) {
			api, _ := newAPI(t, true)
			account := create(t, api)
			request := map[string]any{"user": account["user"], "session": account["session"]}
			dir := t.TempDir()
			api.cfg.GameConfigDir = dir
			call(t, api, tc.path, request, 503)
			path := filepath.Join(dir, tc.filename)
			for _, malformed := range []string{"{", "{}", `{"` + tc.key + `":[null]}`, `{"` + tc.key + `":[{}]}`, `{"` + tc.key + `":[{"unexpected":1}]}`} {
				if err := os.WriteFile(path, []byte(malformed), 0600); err != nil {
					t.Fatal(err)
				}
				call(t, api, tc.path, request, 503)
			}
			if err := os.WriteFile(path, []byte(tc.valid), 0600); err != nil {
				t.Fatal(err)
			}
			out := call(t, api, tc.path, request, 200)
			if _, ok := out[tc.key]; !ok {
				t.Fatalf("missing resource key %q in %v", tc.key, out)
			}
		})
	}
}

func TestTask3SpiritShopsUsesValidConfigWhenStaticOverrideIsCorrupt(t *testing.T) {
	api, _ := newAPI(t, true)
	root := t.TempDir()
	dir := filepath.Join(root, "config")
	static := filepath.Join(root, "resources", "static_responses")
	for _, path := range []string{dir, static} {
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	api.cfg.GameConfigDir = dir
	if err := os.WriteFile(filepath.Join(dir, "spirit_shops.json"), []byte(`{"spirit_shops":[{"id":7,"dep":0,"nm":"fixture"}],"spirit_shops_version":3}`), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(static, "account__get_spirit_shops.json")
	for _, fixture := range []string{"{", `{"spirit_shops":[1]}`, `{"spirit_shops":[]}garbage`} {
		if err := os.WriteFile(path, []byte(fixture), 0600); err != nil {
			t.Fatal(err)
		}
		for _, ua := range []string{"Sky-Live-ios/0.1.8.100", "Sky-Live-ios/0.3.11.100"} {
			out := task3CallWithUA(t, api, "/account/get_spirit_shops", ua, map[string]any{}, 200)
			if out["spirit_shops_version"] != float64(3) || len(out["spirit_shops"].([]any)) != 1 {
				t.Fatalf("%s: did not use valid config after corrupt static override %q: %v", ua, fixture, out)
			}
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "game_control.json"), []byte(`{"spirit_shops":{"enabled":true,"shops":[1]}}`), 0600); err != nil {
		t.Fatal(err)
	}
	control := task3CallWithUA(t, api, "/account/get_spirit_shops", "Sky-Live-ios/0.1.8.100", map[string]any{}, 200)
	if control["spirit_shops_version"] != float64(3) || len(control["spirit_shops"].([]any)) != 1 {
		t.Fatalf("invalid control override hid valid config: %v", control)
	}
}

func TestTask3ShopUnlocksAndOutfitConfigAndVersion(t *testing.T) {
	api, _ := newAPI(t, true)
	account := create(t, api)
	request := map[string]any{"user": account["user"], "session": account["session"]}
	dir := t.TempDir()
	api.cfg.GameConfigDir = dir
	shop := call(t, api, "/account/get_shop", request, 200)
	if len(shop["get_shop"].([]any)) != 1 {
		t.Fatalf("invalid shop contract: %v", shop)
	}
	outfit := call(t, api, "/account/get_outfit", request, 200)
	if _, ok := outfit["set_outfit"].(map[string]any); !ok {
		t.Fatalf("invalid outfit contract: %v", outfit)
	}
	base := task3CallWithUA(t, api, "/account/get_unlocks", "Sky-Live-ios/0.1.8.100", request, 200)
	if !task3HasUnlock(base, "CharSkyKid_Prop_None") {
		t.Fatalf("missing default unlock without optional files: %v", base)
	}
	fixtures := map[string]string{
		"auto_full_unlock.json":      `{"enabled":true,"since":0}`,
		"full_unlocks_template.json": `{"unlocks":[{"name":"CharNew","type":"spiritshop"},{"name":"ordinary","type":"spiritshop"}]}`,
		"outfit_version_map.json":    `[{"major":0,"minor":1,"patch":8,"chars":["CharOld"]},{"major":0,"minor":3,"patch":11,"chars":["CharNew"]}]`,
	}
	for name, value := range fixtures {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct{ ua, want, reject string }{
		{"Sky-Live-ios/0.1.8.100", "CharOld", "CharNew"},
		{"Sky-Live-ios/0.3.11.100", "CharNew", "CharOld"},
	} {
		out := task3CallWithUA(t, api, "/account/get_unlocks", tc.ua, request, 200)
		if !task3HasUnlock(out, tc.want) || task3HasUnlock(out, tc.reject) || !task3HasUnlock(out, "ordinary") {
			t.Fatalf("%s: wrong versioned unlocks: %v", tc.ua, out)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "auto_full_unlock.json"), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	fallback := task3CallWithUA(t, api, "/account/get_unlocks", "Sky-Live-ios/0.1.8.100", request, 200)
	if task3HasUnlock(fallback, "CharOld") || !task3HasUnlock(fallback, "CharSkyKid_Prop_None") {
		t.Fatalf("corrupt full-unlock policy did not fail closed: %v", fallback)
	}
}

func task3HasUnlock(response map[string]any, name string) bool {
	for _, raw := range response["unlocks"].([]any) {
		if raw.(map[string]any)["name"] == name {
			return true
		}
	}
	return false
}

func TestTask3MotdPreservesRequiredFieldsForPartialConfig(t *testing.T) {
	api, _ := newAPI(t, true)
	dir := t.TempDir()
	api.cfg.GameConfigDir = dir
	path := filepath.Join(dir, "motd_config.json")
	for _, fixture := range []string{"", "{}", `{"motd":123,"motd_title":null,"motd_buttons":"bad"}`} {
		if fixture != "" {
			if err := os.WriteFile(path, []byte(fixture), 0600); err != nil {
				t.Fatal(err)
			}
		}
		out := call(t, api, "/account/get_motd", map[string]any{}, 200)
		if _, ok := out["motd"].(string); !ok {
			t.Fatalf("fixture %q has no motd text: %v", fixture, out)
		}
		if _, ok := out["motd_title"].(string); !ok {
			t.Fatalf("fixture %q has no motd title: %v", fixture, out)
		}
		if _, ok := out["motd_buttons"].([]any); !ok {
			t.Fatalf("fixture %q has no motd buttons: %v", fixture, out)
		}
	}
	if err := os.WriteFile(path, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	call(t, api, "/account/get_motd", map[string]any{}, 503)
}
