package gameapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Listen                 string           `json:"listen"`
	Database               string           `json:"database"`
	TLSCert                string           `json:"tls_cert"`
	TLSKey                 string           `json:"tls_key"`
	AutoAuthorize          bool             `json:"auto_authorize"`
	RecreateMissingAccount bool             `json:"recreate_missing_account"`
	RoomHost               string           `json:"room_host"`
	RoomPort               int              `json:"room_port"`
	CurrentSeason          int              `json:"current_season"`
	DefaultCurrency        map[string]int64 `json:"default_currency"`
}

type API struct {
	db     *sql.DB
	cfg    Config
	routes map[string]endpoint
	legacy map[string]bool
}
type object = map[string]any
type endpoint func(*http.Request, object) (object, error)
type apiError struct {
	status int
	body   object
}

func (e *apiError) Error() string           { return http.StatusText(e.status) }
func fail(status int, message string) error { return &apiError{status, object{"error": message}} }

func New(db *sql.DB, cfg Config) (*API, error) {
	if err := checkSchema(db); err != nil {
		return nil, err
	}
	ip := net.ParseIP(cfg.RoomHost)
	if ip == nil || ip.To4() == nil || cfg.RoomPort < 1 || cfg.RoomPort > 65535 {
		return nil, errors.New("room_host must be IPv4 and room_port must be 1..65535")
	}
	a := &API{db: db, cfg: cfg, routes: map[string]endpoint{}, legacy: map[string]bool{}}
	for _, path := range legacyPaths {
		a.legacy[path] = true
	}
	a.routes["/account/create"] = a.create
	a.routes["/account/auth/create"] = a.create
	a.routes["/account/login"] = a.login
	a.routes["/account/auth/login"] = a.login
	a.routes["/account/userdata"] = a.userdata
	a.routes["/account/sync_user_data"] = a.userdata
	a.routes["/account/get_currency"] = a.currency
	a.routes["/account/get_rank"] = a.rank
	a.routes["/account/find_previous_or_empty"] = a.discover
	a.routes["/account/get_checkpoints"] = a.checkpoints
	a.routes["/account/set_checkpoint"] = a.checkpoints
	a.routes["/account/get_latest_build_version"] = a.latestBuild
	a.routes["/account/device_capabilities/lookup"] = a.capabilities
	a.routes["/account/get_vars"] = a.variables
	a.routes["/account/get_motd"] = a.motd
	a.routes["/account/support/has_reply"] = a.support
	return a, nil
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if r.URL.Path == "/healthz" {
		if r.Method != "GET" {
			write(w, 405, object{"error": "Method not allowed"})
			return
		}
		write(w, 200, object{"status": "ok"})
		return
	}
	handler, ok := a.routes[r.URL.Path]
	if !ok {
		status := 404
		if a.legacy[r.URL.Path] {
			status = 501
		}
		write(w, status, object{"error": http.StatusText(status)})
		return
	}
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		write(w, 405, object{"error": "Method not allowed"})
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 2<<20))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			write(w, 413, object{"error": "Request body too large"})
		} else {
			write(w, 400, object{"error": "Invalid request body"})
		}
		return
	}
	raw = bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf})
	var data object
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if dec.Decode(&data) != nil || data == nil {
		write(w, 400, object{"error": "Invalid JSON format"})
		return
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		write(w, 400, object{"error": "Invalid JSON format"})
		return
	}
	result, err := handler(r, data)
	if err != nil {
		var e *apiError
		if errors.As(err, &e) {
			write(w, e.status, e.body)
		} else {
			slog.Error("request failed", "path", r.URL.Path, "error_type", errorType(err))
			write(w, 500, object{"error": "Internal server error"})
		}
		return
	}
	write(w, 200, result)
}
func errorType(err error) string {
	if errors.Is(err, sql.ErrNoRows) {
		return "missing_row"
	}
	return "internal"
}
func write(w http.ResponseWriter, status int, body object) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
func text(m object, k string) string { s, _ := m[k].(string); return s }
func number(m object, k string, def int64) int64 {
	v := m[k]
	var s string
	switch x := v.(type) {
	case json.Number:
		s = x.String()
	case string:
		s = x
	default:
		return def
	}
	n, e := strconv.ParseInt(s, 10, 64)
	if e != nil {
		return def
	}
	return n
}
func integer(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}
func stringValue(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
func remoteIP(r *http.Request) string {
	h, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return h
}
func legacyUA(r *http.Request) bool { return legacyVersion(r.UserAgent()) }
func now() int64                    { return time.Now().Unix() }
func require(m object, keys ...string) error {
	var missing []string
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return fail(400, "Missing required keys: "+strings.Join(missing, ", "))
	}
	return nil
}
