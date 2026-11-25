// 验证管理面板封禁规则影响游戏请求且可立即撤销。
package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/handler"
)

func TestAdminIPBanAppliesAndPersists(t *testing.T) {
	_, db := newAPI(t, true)
	cfg := handler.Config{InstanceDir: t.TempDir(), AdminToken: "test-admin", RoomHost: "127.0.0.1", RoomPort: 25565}
	makeAPI := func() *handler.API {
		api, err := New(db, cfg)
		if err != nil {
			t.Fatal(err)
		}
		return api
	}
	const path = "/admin/api/ip_bans"
	api := makeAPI()
	args := map[string]any{"action": "add", "entry": "192.0.2.9/24"}
	dCall(t, api, "POST", path, "127.0.0.1:1000", "", args, 403)
	dCall(t, api, "POST", path, "127.0.0.1:1000", "test-admin", args, 200)
	check := func(api *handler.API, peer, header string, want int) {
		req := httptest.NewRequest("POST", "/account/get_motd", strings.NewReader("{}"))
		req.RemoteAddr = peer
		req.Header.Set("X-Forwarded-For", header)
		result := httptest.NewRecorder()
		api.ServeHTTP(result, req)
		if result.Code != want {
			t.Fatalf("peer %s: got %d, want %d", peer, result.Code, want)
		}
	}
	check(api, "192.0.2.1:1000", "198.51.100.1", http.StatusMethodNotAllowed)
	check(api, "198.51.100.1:1000", "192.0.2.1", http.StatusOK)
	cfg.IPBanTrustedProxies = []string{"198.51.100.1/32"}
	api = makeAPI()
	check(api, "198.51.100.1:1000", "198.51.100.2, 192.0.2.1", http.StatusMethodNotAllowed)
	dCall(t, api, "POST", path, "127.0.0.1:1000", "test-admin", map[string]any{"action": "remove", "entry": "192.0.2.0/24"}, 200)
	check(api, "192.0.2.1:1000", "", http.StatusOK)
}
