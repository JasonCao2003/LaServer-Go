package handler

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func credentialAPI(t *testing.T) *API {
	t.Helper()
	_, db := newAPI(t, true)
	api, err := New(db, Config{AutoAuthorize: true, RoomHost: "192.168.1.10", RoomPort: 25565})
	if err != nil {
		t.Fatal(err)
	}
	return api
}

func authCreate(t *testing.T, api *API) map[string]any {
	t.Helper()
	return call(t, api, "/account/auth/create", map[string]any{
		"type": "device", "device_name": "test", "device_token": "", "production": false,
		"tos_version": 1, "device_key": "", "sig_ts": 0, "sig": "", "hashes": []any{},
	}, 200)
}

func TestCredentialLoginRejectsIDAndDeviceOnlyWithoutRotatingSession(t *testing.T) {
	api := credentialAPI(t)
	call(t, api, "/account/create", map[string]any{"device_name": "legacy"}, 403)
	created := authCreate(t, api)
	info := created["authinfo"].(map[string]any)
	user, device, session := info["user"].(string), info["device"].(string), created["session"].(string)
	call(t, api, "/account/auth/login", map[string]any{"user": user, "device": device, "key": info["key"]}, 401)
	call(t, api, "/account/auth/login", map[string]any{"device": device, "key": info["key"]}, 401)
	call(t, api, "/account/login", map[string]any{"user": user, "device": device, "key": info["key"]}, 403)
	call(t, api, "/account/get_currency", map[string]any{"user": user, "session": session}, 200)
}

func TestCredentialLoginRejectsCrossAccountAndReplayedToken(t *testing.T) {
	api := credentialAPI(t)
	a := authCreate(t, api)
	b := authCreate(t, api)
	ai := a["authinfo"].(map[string]any)
	bi := b["authinfo"].(map[string]any)
	user := ai["user"].(string)
	old := a["session"].(string)
	credential := ai["recovery_token"].(string)
	if credential == "" {
		t.Fatal("create did not issue a recovery credential")
	}
	call(t, api, "/account/auth/login", map[string]any{"user": user, "recovery_token": bi["recovery_token"], "device": "attacker"}, 401)
	call(t, api, "/account/get_currency", map[string]any{"user": user, "session": old}, 200)
	login := call(t, api, "/account/auth/login", map[string]any{"user": user, "recovery_token": credential}, 200)
	newToken := login["authinfo"].(map[string]any)["recovery_token"].(string)
	if newToken == credential || login["session"] == old {
		t.Fatal("credential or session was not rotated")
	}
	call(t, api, "/account/get_currency", map[string]any{"user": user, "session": old}, 401)
	call(t, api, "/account/auth/login", map[string]any{"user": user, "recovery_token": credential}, 401)
	call(t, api, "/account/get_currency", map[string]any{"user": user, "session": login["session"]}, 200)
	call(t, api, "/account/login", map[string]any{"user": user, "recovery_token": newToken}, 403)
	call(t, api, "/account/auth/login", map[string]any{"user": user, "recovery_token": newToken}, 200)
}

func TestCredentialLoginAcceptsFormAndBOMJSON(t *testing.T) {
	api := credentialAPI(t)
	created := authCreate(t, api)["authinfo"].(map[string]any)
	user, credential := created["user"].(string), created["recovery_token"].(string)
	form := url.Values{"user": {user}, "recovery_token": {credential}}
	req := httptest.NewRequest("POST", "/account/auth/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("form login: %d %s", w.Code, w.Body.String())
	}
	// A failed retry cannot revoke the session issued by the form request.
	call(t, api, "/account/auth/login", map[string]any{"user": user, "recovery_token": credential}, 401)
	created = authCreate(t, api)["authinfo"].(map[string]any)
	req = httptest.NewRequest("POST", "/account/auth/login", strings.NewReader("\ufeff"+`{"user":"`+created["user"].(string)+`","recovery_token":"`+created["recovery_token"].(string)+`"}`))
	req.Header.Set("Content-Type", "text/plain")
	w = httptest.NewRecorder()
	api.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("BOM JSON login: %d %s", w.Code, w.Body.String())
	}
}

func TestCredentialModeRejectsLegacyIssuedSessionAfterModeSwitch(t *testing.T) {
	dev, db := newAPI(t, true)
	created := authCreate(t, dev)
	info := created["authinfo"].(map[string]any)
	user := info["user"].(string)
	stolen := call(t, dev, "/account/auth/login", map[string]any{"user": user}, 200)
	oldSession := stolen["session"].(string)
	stolenToken := stolen["authinfo"].(map[string]any)["recovery_token"]
	call(t, dev, "/account/get_currency", map[string]any{"user": user, "session": oldSession}, 200)
	strict, err := New(db, Config{AutoAuthorize: true, RoomHost: "192.168.1.10", RoomPort: 25565})
	if err != nil {
		t.Fatal(err)
	}
	call(t, strict, "/account/get_currency", map[string]any{"user": user, "session": oldSession}, 401)
	call(t, strict, "/starwatch/auth/api/v1/auth", map[string]any{"sessionToken": oldSession}, 400)
	call(t, strict, "/account/auth/login", map[string]any{"user": user, "recovery_token": stolenToken}, 401)
	call(t, strict, "/account/auth/login", map[string]any{"user": user, "recovery_token": info["recovery_token"]}, 401)
	fresh := authCreate(t, strict)
	freshInfo := fresh["authinfo"].(map[string]any)
	loggedIn := call(t, strict, "/account/auth/login", map[string]any{"user": freshInfo["user"], "recovery_token": freshInfo["recovery_token"]}, 200)
	call(t, strict, "/account/get_currency", map[string]any{"user": freshInfo["user"], "session": loggedIn["session"]}, 200)
}

func TestLegacyLoginRevokesCredentialModeProvenanceEvenWithoutTokenRotation(t *testing.T) {
	dev, db := newAPI(t, true)
	strict, err := New(db, Config{AutoAuthorize: true, RoomHost: "192.168.1.10", RoomPort: 25565})
	if err != nil {
		t.Fatal(err)
	}
	created := authCreate(t, strict)
	info := created["authinfo"].(map[string]any)
	user, credential := info["user"].(string), info["recovery_token"].(string)
	call(t, dev, "/account/login", map[string]any{"user": user}, 200)
	call(t, strict, "/account/auth/login", map[string]any{"user": user, "recovery_token": credential}, 401)
}
