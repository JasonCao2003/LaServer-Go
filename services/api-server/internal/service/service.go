package service

import (
	"context"
	"encoding/json"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/config"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Object = map[string]any
type object = Object
type Request struct {
	Context                         context.Context
	Path, UserAgent, RemoteIP, Host string
	Secure                          bool
}
type Service struct {
	store *repository.Store
	cfg   *config.Config
}

func New(store *repository.Store, cfg *config.Config) *Service {
	return &Service{store: store, cfg: cfg}
}

type Error struct {
	Status int
	Body   Object
}

func (e *Error) Error() string              { return http.StatusText(e.Status) }
func fail(status int, message string) error { return &Error{status, object{"error": message}} }
func text(m object, k string) string        { s, _ := m[k].(string); return s }
func number(m object, k string, def int64) int64 {
	v := m[k]
	var s string
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
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
func remoteIP(r Request) string { return r.RemoteIP }
func legacyUA(r Request) bool   { return legacyVersion(r.UserAgent) }
func now() int64                { return time.Now().Unix() }
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
