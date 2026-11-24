package panel

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"html/template"
	"net/http"
	"sync"
	"time"
)

const cookieName = "laserver_panel_session"
const sessionTTL = 8 * time.Hour
const maxSessions = 64

type session struct {
	csrf    string
	expires time.Time
}

type panelServer struct {
	cfg      Config
	pages    *template.Template
	client   *http.Client
	mu       sync.Mutex
	sessions map[string]session
}

func newPanel(cfg Config, pages *template.Template) *panelServer {
	return &panelServer{cfg: cfg, pages: pages, sessions: make(map[string]session), client: &http.Client{
		Timeout:       cfg.UpstreamTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

func (p *panelServer) render(w http.ResponseWriter, name string, status int, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = p.pages.ExecuteTemplate(w, name, data)
}

func (p *panelServer) loginPage(w http.ResponseWriter, _ *http.Request) {
	p.render(w, "login.html", http.StatusOK, "")
}

func (p *panelServer) parsePost(w http.ResponseWriter, r *http.Request) bool {
	origins := r.Header.Values("Origin")
	if len(origins) != 1 || origins[0] != "http://"+p.cfg.Listen {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return false
	}
	return true
}

func randomToken() (string, error) {
	var raw [32]byte
	_, err := rand.Read(raw[:])
	return hex.EncodeToString(raw[:]), err
}

func (p *panelServer) login(w http.ResponseWriter, r *http.Request) {
	if !p.parsePost(w, r) {
		return
	}
	entered := sha256.Sum256([]byte(r.PostForm.Get("password")))
	expected := sha256.Sum256([]byte(p.cfg.password))
	if p.cfg.password == "" || subtle.ConstantTimeCompare(entered[:], expected[:]) != 1 {
		p.render(w, "login.html", http.StatusUnauthorized, "密码不正确。")
		return
	}
	id, err := randomToken()
	if err != nil {
		http.Error(w, "Session unavailable", 500)
		return
	}
	csrf, err := randomToken()
	if err != nil {
		http.Error(w, "Session unavailable", 500)
		return
	}
	now := time.Now()
	s := session{csrf: csrf, expires: now.Add(sessionTTL)}
	previous, _ := r.Cookie(cookieName)
	p.mu.Lock()
	for key, value := range p.sessions {
		if !now.Before(value.expires) {
			delete(p.sessions, key)
		}
	}
	if previous != nil {
		delete(p.sessions, previous.Value)
	}
	if len(p.sessions) >= maxSessions {
		p.mu.Unlock()
		http.Error(w, "Too many sessions", http.StatusServiceUnavailable)
		return
	}
	p.sessions[id] = s
	p.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: id, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, Expires: s.expires})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (p *panelServer) activeSession(r *http.Request) (string, session, bool) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return "", session{}, false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	s, ok := p.sessions[cookie.Value]
	if ok && !time.Now().Before(s.expires) {
		delete(p.sessions, cookie.Value)
		ok = false
	}
	return cookie.Value, s, ok
}

func (p *panelServer) logout(w http.ResponseWriter, r *http.Request) {
	if !p.parsePost(w, r) {
		return
	}
	id, s, ok := p.activeSession(r)
	if !ok || subtle.ConstantTimeCompare([]byte(r.PostForm.Get("csrf")), []byte(s.csrf)) != 1 {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	p.mu.Lock()
	delete(p.sessions, id)
	p.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: cookieName, Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
