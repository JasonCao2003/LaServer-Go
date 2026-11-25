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
	mux.HandleFunc("GET /announcement", p.announcement)
	mux.HandleFunc("POST /announcement", p.announcement)
	mux.HandleFunc("GET /ip-bans", p.ipBans)
	mux.HandleFunc("POST /ip-bans", p.ipBans)
	mux.HandleFunc("GET /operations", p.operations)
	mux.HandleFunc("POST /operations", p.operations)
	mux.HandleFunc("GET /magic", p.catalog)
	mux.HandleFunc("POST /magic", p.catalog)
	mux.HandleFunc("GET /ugc", p.catalog)
	mux.HandleFunc("POST /ugc", p.catalog)
	mux.HandleFunc("GET /other", p.catalog)
	mux.HandleFunc("POST /other", p.catalog)
	mux.HandleFunc("GET /traffic/http", p.traffic)
	mux.HandleFunc("GET /traffic/udp", p.traffic)
	mux.HandleFunc("GET /updates", p.updates)
	mux.HandleFunc("POST /updates", p.updates)
	mux.HandleFunc("GET /accounts", p.accounts)
	mux.HandleFunc("POST /accounts/change", p.changeAccount)
	mux.HandleFunc("POST /accounts/batch", p.batchAccounts)
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
		if cfg.password == "" && r.Method == http.MethodGet {
			if r.URL.Path == "/login" {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
			if _, _, ok := p.activeSession(r); !ok {
				// Replace only the expired panel cookie before issuing its successor.
				cookies := r.Cookies()
				r.Header.Del("Cookie")
				for _, c := range cookies {
					if c.Name != cookieName {
						r.AddCookie(c)
					}
				}
				if !p.issueSession(w, r) {
					return
				}
			}
		}
		mux.ServeHTTP(w, r)
	}), nil
}
