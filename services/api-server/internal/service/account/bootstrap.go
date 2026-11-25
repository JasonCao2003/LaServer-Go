// 客户端启动资源（版本、变量、MOTD 等）

package account

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

// 获取最新构建版本
func (a *Service) LatestBuild(r Request, _ object) (object, error) {
	build := int64(0)
	if m := buildPattern.FindStringSubmatch(r.UserAgent); len(m) > 1 {
		if n, e := strconv.ParseInt(m[1], 10, 64); e == nil {
			build = n
		}
	}
	_, err := a.Store.ExecContext(r.Context, repository.AdvanceLatestBuild, build)
	if err != nil {
		return nil, err
	}
	if err = a.Store.QueryRowContext(r.Context, repository.ReadLatestBuild).Scan(&build); err != nil {
		return nil, err
	}
	return object{"latest_build_version": build, "min_gpu_rating": 540, "maintenance_mode": false, "maintenance_msg": "", "app_store_uri": "", "tos": object{"version": 4, "tos_url": "", "pp_url": ""}, "cs_enabled": true, "cs_db_enabled": false, "country_code": "CN"}, nil
}

// 获取设备能力配置
func (a *Service) Capabilities(r Request, m object) (object, error) {
	if id := text(m, "user"); id != "" {
		u, err := a.FindUser(r.Context, "id", id)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fail(404, "User not found")
		}
		if err != nil {
			return nil, err
		}
		if err = a.CheckAllowed(r.Context, &u, 503); err != nil {
			return nil, err
		}
	}
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return nil, err
	}
	return object{"device_capabilities": object{"gpu_rating": 10003, "hash_id": binary.BigEndian.Uint32(b[:])}}, nil
}

// 获取客户端变量
func (a *Service) Variables(_ Request, _ object) (object, error) {
	var out object
	err := json.Unmarshal([]byte(variablesJSON), &out)
	return out, err
}

// 获取公告信息
func (a *Service) Motd(_ Request, _ object) (object, error) {
	stamp := now()
	defaults := object{"motd": "欢迎来到 LaServer", "motd_title": "LaServer", "motd_timestamp": stamp, "motd_version": 0, "motd_buttons": []any{}, "motd_season_start": stamp, "motd_season_end": stamp + 30*86400}
	data, err := a.localMotd()
	if err == nil && data == nil {
		data, err = a.LegacyObject("motd_config.json")
	}
	if err != nil {
		return nil, err
	}
	if data != nil {
		if _, ok := data["motd"].(string); !ok {
			data["motd"] = defaults["motd"]
		}
		if _, ok := data["motd_title"].(string); !ok {
			data["motd_title"] = defaults["motd_title"]
		}
		if _, ok := data["motd_buttons"].([]any); !ok {
			data["motd_buttons"] = defaults["motd_buttons"]
		}
		if _, ok := data["motd_version"]; !ok {
			data["motd_version"] = defaults["motd_version"]
		}
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
	return defaults, nil
}

// 查询客服回复
func (a *Service) Support(r Request, m object) (object, error) {
	if _, err := a.Authenticated(r, m); err != nil {
		return nil, err
	}
	return object{"result": "ok", "has_reply": false}, nil
}
