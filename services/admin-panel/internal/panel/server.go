package panel

import (
	"embed"
	"html/template"
	"net/http"
	"net/netip"
)

//go:embed templates/*.html
var templates embed.FS

func New(cfg Config) (http.Handler, error) {
	allowed, err := listenAddress(cfg.Listen)
	if err != nil {
		return nil, err
	}
	pages, err := template.ParseFS(templates, "templates/*.html")
	if err != nil {
		return nil, err
	}
	p := newPanel(cfg, pages)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", p.overview)
	mux.HandleFunc("GET /login", p.loginPage)
	mux.HandleFunc("POST /login", p.login)
	mux.HandleFunc("POST /logout", p.logout)
	mux.HandleFunc("GET /friends", p.friendsList)
	mux.HandleFunc("POST /friends/set", p.friendsSet)
	mux.HandleFunc("POST /friends/clear", p.friendsClear)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		host, err := netip.ParseAddrPort(r.Host)
		if err != nil || host != allowed {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		mux.ServeHTTP(w, r)
	}), nil
}
