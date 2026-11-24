// 路由与 HTTP 方法兼容性测试

package handler

import (
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// These responses distinguish a method mismatch from an unknown path.
// In particular, the legacy API never redirects a POST to a normalized URL.
func TestExactRouteAndMethodResponses(t *testing.T) {
	api, _ := newAPI(t, false)
	for _, tc := range []struct {
		method, path string
		status       int
		allow        string
	}{
		{"POST", "/healthz", 405, ""},
		{"HEAD", "/healthz", 405, ""},
		{"GET", "/account/create", 405, "POST"},
		{"PATCH", "/account/commerce/product_info", 405, "GET, POST"},
		{"PUT", "/admin/api/friends_list", 405, "GET, POST"},
		{"GET", "/account/get_friend_statues", 405, "POST, OPTIONS"},
		{"POST", "/ugc/file", 405, "GET"},
		{"POST", "/account/create/", 404, ""},
		{"GET", "/HEALTHZ", 404, ""},
		{"GET", "/ugc", 404, ""},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			api.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			body := "{\"error\":\"Not Found\"}\n"
			if tc.status == http.StatusMethodNotAllowed {
				body = "{\"error\":\"Method not allowed\"}\n"
			}
			if w.Code != tc.status || w.Header().Get("Allow") != tc.allow || w.Body.String() != body ||
				w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Location") != "" {
				t.Fatalf("status=%d headers=%v body=%q", w.Code, w.Header(), w.Body.String())
			}
		})
	}
}

func TestGinRoutesMatchLegacyInventory(t *testing.T) {
	raw, err := os.ReadFile("../../../../json/api/legacy-routes.json")
	if err != nil {
		t.Fatal(err)
	}
	var inventory []struct {
		Path    string
		Methods []string
	}
	if err := json.Unmarshal(raw, &inventory); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"GET /healthz": true, "POST /internal/v1/room/admissions/verify": true}
	for _, entry := range inventory {
		path := entry.Path
		if path == "/ugc/<path:subpath>" {
			path = "/ugc/*subpath"
		}
		for _, method := range entry.Methods {
			want[method+" "+path] = true
		}
	}
	api, _ := newAPI(t, false)
	got := map[string]bool{}
	for _, route := range api.router.Routes() {
		got[route.Method+" "+route.Path] = true
	}
	if !maps.Equal(got, want) {
		for key := range want {
			if !got[key] {
				t.Errorf("missing route %s", key)
			}
		}
		for key := range got {
			if !want[key] {
				t.Errorf("unexpected route %s", key)
			}
		}
	}
}
