package service

import (
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"

	"regexp"
	"strconv"
)

var buildPattern = regexp.MustCompile(`/[0-9.]*\.([0-9]+)`)

func (a *Service) LatestBuild(r Request, _ object) (object, error) {
	build := int64(0)
	if m := buildPattern.FindStringSubmatch(r.UserAgent); len(m) > 1 {
		if n, e := strconv.ParseInt(m[1], 10, 64); e == nil {
			build = n
		}
	}
	_, err := a.store.ExecContext(r.Context, repository.AdvanceLatestBuild, build)
	if err != nil {
		return nil, err
	}
	if err = a.store.QueryRowContext(r.Context, repository.ReadLatestBuild).Scan(&build); err != nil {
		return nil, err
	}
	return object{"latest_build_version": build, "min_gpu_rating": 540, "maintenance_mode": false, "maintenance_msg": "", "app_store_uri": "", "tos": object{"version": 4, "tos_url": "", "pp_url": ""}, "cs_enabled": true, "cs_db_enabled": false, "country_code": "CN"}, nil
}

func (a *Service) Capabilities(r Request, m object) (object, error) {
	if id := text(m, "user"); id != "" {
		u, err := a.findUser(r.Context, "id", id)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fail(404, "User not found")
		}
		if err != nil {
			return nil, err
		}
		if err = a.checkAllowed(r.Context, &u, 503); err != nil {
			return nil, err
		}
	}
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return nil, err
	}
	return object{"device_capabilities": object{"gpu_rating": 10003, "hash_id": binary.BigEndian.Uint32(b[:])}}, nil
}

func (a *Service) Variables(_ Request, _ object) (object, error) {
	var out object
	err := json.Unmarshal([]byte(variablesJSON), &out)
	return out, err
}
func (a *Service) Motd(_ Request, _ object) (object, error) {
	stamp := now()
	data, err := a.legacyObject("motd_config.json")
	if err != nil {
		return nil, err
	}
	if data != nil {
		if text(data, "motd_timestamp_mode") != "fixed" || number(data, "motd_timestamp", 0) == 0 {
			data["motd_timestamp"] = stamp
		}
		delete(data, "motd_timestamp_mode")
		if _, ok := data["motd_season_start"]; !ok {
			data["motd_season_start"] = number(data, "motd_timestamp", stamp)
		}
		if _, ok := data["motd_season_end"]; !ok {
			data["motd_season_end"] = number(data, "motd_season_start", stamp) + 30*86400
		}
		return data, nil
	}
	return object{"motd": "欢迎来到 LaServer", "motd_title": "LaServer", "motd_timestamp": stamp, "motd_version": 0, "motd_buttons": []any{}, "motd_season_start": stamp, "motd_season_end": stamp + 30*86400}, nil
}
func (a *Service) Support(r Request, m object) (object, error) {
	if _, err := a.authenticated(r, m); err != nil {
		return nil, err
	}
	return object{"result": "ok", "has_reply": false}, nil
}
