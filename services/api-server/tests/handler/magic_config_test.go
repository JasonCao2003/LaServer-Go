// 魔法/增益静态配置回退测试

package handler_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTask3MagicStaticOverrideFallsBackToEmbeddedOnCorruption(t *testing.T) {
	api, _ := newAPI(t, true)
	account := create(t, api)
	body := map[string]any{"user": account["user"], "session": account["session"]}
	root := t.TempDir()
	api.RuntimeConfig().GameConfigDir = filepath.Join(root, "config")
	static := filepath.Join(root, "resources", "static_responses")
	if err := os.MkdirAll(static, 0700); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path, filename, key, fixture string
	}{
		{"/account/consumable/get_consumable_defs", "account__consumable__get_consumable_defs.json", "get_consumable_defs", `{"get_consumable_defs":[{"id":123,"name":"message_fixture"}]}`},
		{"/account/buff/get_buff_defs", "account__buff__get_buff_defs.json", "get_buff_defs", `{"get_buff_defs":[{"id":456,"name":"fixture_buff"}]}`},
	} {
		t.Run(tc.key, func(t *testing.T) {
			for _, ua := range []string{"Sky-Live-ios/0.1.8.100", "Sky-Live-ios/0.3.11.100"} {
				missing := task3CallWithUA(t, api, tc.path, ua, body, 200)
				if len(missing[tc.key].([]any)) == 0 {
					t.Fatalf("%s: embedded definitions unavailable", ua)
				}
			}
			path := filepath.Join(static, tc.filename)
			if err := os.WriteFile(path, []byte(tc.fixture), 0600); err != nil {
				t.Fatal(err)
			}
			configured := task3CallWithUA(t, api, tc.path, "Sky-Live-ios/0.26.0.100", body, 200)
			if len(configured[tc.key].([]any)) != 1 {
				t.Fatalf("valid override ignored: %v", configured)
			}
			for _, bad := range []string{"{", tc.fixture + "garbage", `{"` + tc.key + `":[null]}`, `{"` + tc.key + `":[{}]}`, `{"` + tc.key + `":[{"unexpected":1}]}`} {
				if err := os.WriteFile(path, []byte(bad), 0600); err != nil {
					t.Fatal(err)
				}
				corrupt := task3CallWithUA(t, api, tc.path, "Sky-Live-ios/0.26.0.100", body, 200)
				if len(corrupt[tc.key].([]any)) < 2 {
					t.Fatalf("corrupt override %q replaced embedded definitions: %v", bad, corrupt)
				}
			}
		})
	}
}
