package handler

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAuthLoginAcceptsLegacyForm(t *testing.T) {
	api, _ := newAPI(t, true)
	a := create(t, api)
	form := url.Values{"user": {a["user"].(string)}, "device": {a["device"].(string)}}
	req := httptest.NewRequest("POST", "/account/auth/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("legacy form login: %d %s", w.Code, w.Body.String())
	}
}

func TestAuthLoginPrefersJSONDespiteFormHeader(t *testing.T) {
	api, _ := newAPI(t, true)
	a := create(t, api)
	req := httptest.NewRequest("POST", "/account/auth/login", strings.NewReader(`{"user":"`+a["user"].(string)+`"}`))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("JSON priority: %d %s", w.Code, w.Body.String())
	}
}

func TestAuthLoginUnauthorizedEnvelope(t *testing.T) {
	api, _ := newAPI(t, false)
	a := create(t, api)
	out := call(t, api, "/account/auth/login", map[string]any{"user": a["user"]}, 406)
	if out["user"] != a["user"] || out["end_date"] != float64(0) || out["message"] != "账号未授权" {
		t.Fatalf("legacy error fields lost: %v", out)
	}
}
