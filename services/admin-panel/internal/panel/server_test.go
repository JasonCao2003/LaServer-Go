package panel

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoginPageHostAndMethod(t *testing.T) {
	handler, err := New(Config{Listen: "127.0.0.1:18080"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, url string
		status      int
	}{
		{"GET", "http://127.0.0.1:18080/login", 200},
		{"GET", "http://evil.example:18080/login", 403},
		{"PUT", "http://127.0.0.1:18080/login", 405},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(tc.method, tc.url, nil))
		if w.Code != tc.status {
			t.Fatalf("%s %s status=%d want %d", tc.method, tc.url, w.Code, tc.status)
		}
		if tc.status == 200 && (!strings.Contains(w.Body.String(), "管理面板") || w.Header().Get("Cache-Control") != "no-store") {
			t.Fatal("missing login page or cache policy")
		}
	}
}
