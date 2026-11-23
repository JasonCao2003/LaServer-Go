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

func TestAuthLoginRequiresUserAndPreservesDevice(t *testing.T) {
	api, db := newAPI(t, true)
	a := create(t, api)
	call(t, api, "/account/auth/login", map[string]any{"device": a["device"], "key": a["key"]}, 400)
	clientDevice := strings.Repeat("d", 32)
	login := call(t, api, "/account/auth/login", map[string]any{"user": a["user"], "device": clientDevice}, 200)
	if login["authinfo"].(map[string]any)["device"] != clientDevice {
		t.Fatal("auth/login did not echo the supplied client device")
	}
	var storedDevice string
	if err := db.QueryRow("SELECT device_identifier FROM user WHERE id=?", a["user"]).Scan(&storedDevice); err != nil {
		t.Fatal(err)
	}
	if storedDevice != a["device"] {
		t.Fatal("auth/login changed the registered device")
	}
	login = call(t, api, "/account/auth/login", map[string]any{"user": a["user"], "device": "short"}, 200)
	if got := login["authinfo"].(map[string]any)["device"].(string); len(got) != 36 || got == storedDevice {
		t.Fatalf("short device should receive a new UUID, got %q", got)
	}
	byDevice := call(t, api, "/account/login", map[string]any{"device": a["device"], "key": a["key"]}, 200)
	if byDevice["user"] != a["user"] {
		t.Fatal("registered device no longer identifies the account")
	}
}
