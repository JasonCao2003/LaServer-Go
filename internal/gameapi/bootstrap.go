package gameapi

import (
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
)

var buildPattern = regexp.MustCompile(`/[0-9.]*\.([0-9]+)`)

func (a *API) latestBuild(r *http.Request, _ object) (object, error) {
	build := int64(0)
	if m := buildPattern.FindStringSubmatch(r.UserAgent()); len(m) > 1 {
		if n, e := strconv.ParseInt(m[1], 10, 64); e == nil {
			build = n
		}
	}
	_, err := a.db.ExecContext(r.Context(), "INSERT INTO version(id,version) VALUES(1,?) ON CONFLICT(id) DO UPDATE SET version=MAX(COALESCE(version.version,0),excluded.version)", build)
	if err != nil {
		return nil, err
	}
	if err = a.db.QueryRowContext(r.Context(), "SELECT version FROM version WHERE id=1").Scan(&build); err != nil {
		return nil, err
	}
	return object{"latest_build_version": build, "min_gpu_rating": 540, "maintenance_mode": false, "maintenance_msg": "", "app_store_uri": "", "tos": object{"version": 4, "tos_url": "", "pp_url": ""}, "cs_enabled": true, "cs_db_enabled": false, "country_code": "CN"}, nil
}

func (a *API) capabilities(r *http.Request, m object) (object, error) {
	if id := text(m, "user"); id != "" {
		u, err := a.findUser(r.Context(), "id", id)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fail(404, "User not found")
		}
		if err != nil {
			return nil, err
		}
		if err = a.checkAllowed(r.Context(), &u, 503); err != nil {
			return nil, err
		}
	}
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return nil, err
	}
	return object{"device_capabilities": object{"gpu_rating": 10003, "hash_id": binary.BigEndian.Uint32(b[:])}}, nil
}

func (a *API) variables(_ *http.Request, _ object) (object, error) {
	var out object
	err := json.Unmarshal([]byte(variablesJSON), &out)
	return out, err
}
func (a *API) motd(_ *http.Request, _ object) (object, error) {
	stamp := now()
	return object{"motd": "欢迎来到 LaServer", "motd_title": "LaServer", "motd_timestamp": stamp, "motd_version": 0, "motd_buttons": []any{}, "motd_season_start": stamp, "motd_season_end": stamp + 30*86400}, nil
}
func (a *API) support(r *http.Request, m object) (object, error) {
	if _, err := a.authenticated(r, m); err != nil {
		return nil, err
	}
	return object{"result": "ok", "has_reply": false}, nil
}
