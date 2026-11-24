package panel

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestFriendsHTTPContract(t *testing.T) {
	calls := make(chan string, 8)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if r.Method != "POST" || r.Header.Get("X-Admin-Token") != "private-token" || json.NewDecoder(r.Body).Decode(&body) != nil || body["user"] != "alice" {
			t.Error("unexpected upstream request")
		}
		calls <- r.URL.Path
		switch r.URL.Path {
		case "/admin/api/friends_list":
			if body["keyword"] != "test" {
				t.Error("missing keyword")
			}
			w.Write([]byte(`{"ok":true,"friends":[{"friend_id":"bob","nickname":"<script>","remark":"<b>"}]}`))
		case "/admin/api/friends_set":
			if body["friend"] != "bob" || body["action"] != "restore" {
				t.Error("unexpected friend mutation")
			}
			w.Write([]byte(`{"ok":false,"detail":"业务失败"}`))
		case "/admin/api/friends_clear":
			w.Write([]byte(`{"ok":false,"removed_count":1,"failed":["bob"]}`))
		default:
			t.Error("unexpected endpoint")
		}
	}))
	defer upstream.Close()
	api, _ := url.Parse(upstream.URL)
	h, err := New(Config{Listen: "127.0.0.1:18080", APIURL: api, UpstreamTimeout: time.Second, password: "secret", apiToken: "private-token"})
	if err != nil {
		t.Fatal(err)
	}
	const origin = "http://127.0.0.1:18080"
	request := func(method, path, body string, c *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, origin+path, strings.NewReader(body))
		r.Header.Set("Origin", origin)
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if c != nil {
			r.AddCookie(c)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := request("POST", "/friends/set", "", nil); w.Code != 403 {
		t.Fatalf("unauthenticated: %d", w.Code)
	}
	cookie := request("POST", "/login", "password=secret", nil).Result().Cookies()[0]
	w := request("GET", "/friends?user=alice&keyword=test", "", cookie)
	body := w.Body.String()
	if w.Code != 200 || !strings.Contains(body, "&lt;script&gt;") || !strings.Contains(body, "&lt;b&gt;") || strings.Contains(body, "private-token") {
		t.Fatalf("list: %d %s", w.Code, body)
	}
	_, tail, ok := strings.Cut(body, `name="csrf" value="`)
	if !ok {
		t.Fatal("missing csrf")
	}
	csrf, _, _ := strings.Cut(tail, `"`)
	for _, row := range []struct {
		path, form string
		want       int
	}{
		{"set", "csrf=bad&user=alice&friend=bob&action=restore", 403},
		{"set", "csrf=" + csrf + "&user=alice&friend=bob&action=bad", 400},
		{"clear", "csrf=" + csrf + "&user=alice&confirm=bob", 400},
	} {
		if w := request("POST", "/friends/"+row.path, row.form, cookie); w.Code != row.want {
			t.Fatal(w.Code)
		}
	}
	if len(calls) != 1 {
		t.Fatal("invalid submission reached upstream")
	}
	w = request("POST", "/friends/set", "csrf="+csrf+"&user=alice&friend=bob&action=restore", cookie)
	if !strings.Contains(w.Body.String(), "业务失败") {
		t.Fatal(w.Body.String())
	}
	w = request("POST", "/friends/clear", "csrf="+csrf+"&user=alice&confirm=alice", cookie)
	if !strings.Contains(w.Body.String(), "已移除 1") || !strings.Contains(w.Body.String(), "bob") {
		t.Fatal(w.Body.String())
	}
	for _, want := range []string{"friends_list", "friends_set", "friends_clear"} {
		select {
		case got := <-calls:
			if got != "/admin/api/"+want {
				t.Fatal(got)
			}
		default:
			t.Fatal("missing call", want)
		}
	}
}

func TestFriendWriteTimeoutIsNotRetried(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		<-r.Context().Done()
	}))
	defer upstream.Close()
	api, _ := url.Parse(upstream.URL)
	p := newPanel(Config{APIURL: api, UpstreamTimeout: 250 * time.Millisecond, apiToken: "test"}, nil)
	data := friendsPage{}
	got := p.loadFriendResult(t.Context(), &data, "set", map[string]string{"user": "alice", "friend": "bob", "action": "add"})
	if got != "unconfirmed" || !strings.Contains(data.Message, "待确认") || calls.Load() != 1 {
		t.Fatalf("got %s, message %q, calls %d", got, data.Message, calls.Load())
	}
}
