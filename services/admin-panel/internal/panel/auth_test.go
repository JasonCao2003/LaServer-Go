package panel

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestLoginOverviewAndLogout(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Admin-Token") != "" {
			t.Error("probe sent token")
		}
		if r.URL.Path == "/readyz" {
			w.WriteHeader(503)
			return
		}
		if r.URL.Path == "/healthz" {
			w.WriteHeader(204)
		}
	}))
	defer upstream.Close()
	room, _ := url.Parse(upstream.URL)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer api.Close()
	apiURL, _ := url.Parse(api.URL)
	h, err := New(Config{Listen: "127.0.0.1:18080", APIURL: apiURL, RoomURL: room, UpstreamTimeout: 50 * time.Millisecond, password: "secret", apiToken: "token"})
	if err != nil {
		t.Fatal(err)
	}
	call := func(method, path, body, origin string, c *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://127.0.0.1:18080"+path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Origin", origin)
		if c != nil {
			r.AddCookie(c)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	origin := "http://127.0.0.1:18080"
	if w := call("GET", "/", "", "", nil); w.Code != 303 {
		t.Fatal(w.Code)
	}
	if w := call("POST", "/login", "password=secret", "http://evil.example", nil); w.Code != 403 {
		t.Fatal(w.Code)
	}
	if w := call("POST", "/login", "password=bad", origin, nil); w.Code != 401 {
		t.Fatal(w.Code)
	}
	w := call("POST", "/login", "password=secret", origin, nil)
	if w.Code != 303 || len(w.Result().Cookies()) != 1 {
		t.Fatalf("login status %d", w.Code)
	}
	c := w.Result().Cookies()[0]
	if !c.HttpOnly || c.SameSite != http.SameSiteStrictMode {
		t.Fatal("cookie flags")
	}
	w = call("GET", "/", "", "", c)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "超时") || !strings.Contains(w.Body.String(), "尚未就绪") {
		t.Fatalf("overview status %d: %s", w.Code, w.Body)
	}
	if w := call("POST", "/logout", "csrf=bad", origin, c); w.Code != 403 {
		t.Fatal(w.Code)
	}
	marker := `name="csrf" value="`
	_, tail, ok := strings.Cut(w.Body.String(), marker)
	if !ok {
		t.Fatal("missing logout token")
	}
	csrf, _, _ := strings.Cut(tail, `"`)
	if w := call("POST", "/logout", url.Values{"csrf": {csrf}}.Encode(), origin, c); w.Code != 303 {
		t.Fatal(w.Code)
	}
	if w := call("GET", "/", "", "", c); w.Code != 303 {
		t.Fatal("session still active")
	}
}

func TestExpiredSessionIsRejectedAndRemoved(t *testing.T) {
	p := newPanel(Config{}, nil)
	p.sessions["expired"] = session{expires: time.Now().Add(-time.Second)}
	r := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
	r.AddCookie(&http.Cookie{Name: cookieName, Value: "expired"})
	if _, _, ok := p.activeSession(r); ok {
		t.Fatal("expired session accepted")
	}
	if len(p.sessions) != 0 {
		t.Fatal("expired session retained")
	}
}

func TestProbeStatusAndRedirect(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.WriteHeader(http.StatusOK)
		case "/readyz":
			http.Redirect(w, r, "/healthz", http.StatusFound)
		}
	}))
	origin, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	p := newPanel(Config{UpstreamTimeout: time.Second}, nil)
	ctx := t.Context()
	if got := p.probe(ctx, origin, "/healthz", 200); got != "在线" {
		t.Fatal(got)
	}
	if got := p.probe(ctx, origin, "/readyz", 204); got != "响应异常（HTTP 302）" {
		t.Fatal(got)
	}
	upstream.Close()
	if got := p.probe(ctx, origin, "/healthz", 200); got != "不可达" {
		t.Fatal(got)
	}
}
