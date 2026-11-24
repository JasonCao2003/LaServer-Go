// 账号创建、登录与会话鉴权

package account

import (
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"

	"regexp"
	"strings"

	entity "github.com/JasonCao2003/LaServer-Go/services/api-server/internal/model"
)

var versionPattern = regexp.MustCompile(`Sky[^/]*/([0-9]+)\.([0-9]+)\.([0-9]+)`)

// 创建用户记录
func (a *Service) createUser(r Request, name, id string, authCreate bool) (entity.User, error) {
	var u entity.User
	var err error
	if id == "" {
		id, err = uuid()
		if err != nil {
			return u, err
		}
	}
	device, err := token(16)
	if err != nil {
		return u, err
	}
	session, err := token(16)
	if err != nil {
		return u, err
	}
	recovery, err := token(32)
	if err != nil {
		return u, err
	}
	stamp := now()
	autoAuthorize, err := a.Policy("auto_auth.json", stamp, a.Cfg.AutoAuthorize)
	if err != nil {
		return u, err
	}
	if authCreate {
		device, err = uuid()
		if err != nil {
			return u, err
		}
		if !autoAuthorize {
			session = "UNAUTH_" + session
		}
	}
	authorized := 0
	if autoAuthorize {
		authorized = 1
	}
	tx, err := a.Store.BeginTx(r.Context, nil)
	if err != nil {
		return u, err
	}
	defer tx.Rollback()
	verified := 0
	if authCreate && a.Cfg.LoginMode != "legacy_dev" {
		verified = 1
	}
	_, err = tx.ExecContext(r.Context, repository.CreateUser, id, name, device, session, recovery, authorized, remoteIP(r), stamp, name, r.UserAgent, verified)
	if err != nil {
		return u, err
	}
	if _, err = tx.ExecContext(r.Context, repository.CreateCurrency, id); err != nil {
		return u, err
	}
	for k, v := range a.Cfg.DefaultCurrency {
		if !currencyField(k) {
			return u, fmt.Errorf("unknown default currency %q", k)
		}
		if _, err = tx.ExecContext(r.Context, repository.CurrencyUpdate(k), v, id); err != nil {
			return u, err
		}
	}
	if err = tx.Commit(); err != nil {
		return u, err
	}
	return a.FindUser(r.Context, "id", id)
}

// 创建账号或资源
func (a *Service) Create(r Request, m object) (object, error) {
	auth := r.Path == "/account/auth/create"
	if !auth && a.Cfg.LoginMode != "legacy_dev" {
		return nil, fail(403, "Legacy account protocol requires login_mode legacy_dev")
	}
	name, ok := m["device_name"].(string)
	if !ok {
		return nil, fail(400, "Invalid device_name")
	}
	if auth {
		if err := require(m, "type", "device_name", "device_token", "production", "tos_version", "device_key", "sig_ts", "sig", "hashes"); err != nil {
			return nil, err
		}
	} else {
		looksLegacy := name != "" && text(m, "sig") == "" && text(m, "device_key") == "" && m["hashes"] == nil
		if !a.LegacyUA(r) && !looksLegacy {
			if err := require(m, "device_name", "language", "tos_version", "link_external", "device_key", "sig_ts", "sig", "hashes"); err != nil {
				return nil, err
			}
		}
	}
	runes := []rune(name)
	if len(runes) > 255 {
		name = string(runes[:255])
	}
	u, err := a.createUser(r, name, "", auth)
	if err != nil {
		return nil, err
	}
	key := strings.Repeat("0", 64)
	if auth {
		key, err = token(32)
		if err != nil {
			return nil, err
		}
	}
	if auth {
		return object{"authinfo": object{"user": u.ID, "device": stringValue(u.DeviceIdentifier), "key": key, "recovery_token": stringValue(u.RecoveryToken)}, "session": stringValue(u.Session), "resource_list": authCreateResources, "authorization_required": integer(u.IsAuthorized) == 0, "message": "账号已创建"}, nil
	}
	result := object{"user": u.ID, "device": stringValue(u.DeviceIdentifier), "key": key, "session": stringValue(u.Session), "resource_list": createResources}
	if a.LegacyUA(r) {
		augmentLegacy(result, u)
	}
	return result, nil
}

// 执行账号登录
func (a *Service) Login(r Request, m object) (object, error) {
	credentialMode := a.Cfg.LoginMode != "legacy_dev"
	auth := r.Path == "/account/auth/login"
	if credentialMode && r.Path != "/account/auth/login" {
		return nil, fail(403, "Legacy account protocol requires login_mode legacy_dev")
	}
	id := strings.TrimSpace(text(m, "user"))
	providedCredential := text(m, "recovery_token")
	if credentialMode && (id == "" || providedCredential == "") {
		return nil, fail(401, "Invalid credentials")
	}
	var u entity.User
	var err error
	if _, hasUser := m["user"]; !hasUser && text(m, "device") != "" && !credentialMode && !auth {
		u, err = a.FindUser(r.Context, "device_identifier", text(m, "device"))
	} else {
		if id == "" {
			return nil, fail(400, "Missing required field: user")
		}
		u, err = a.FindUser(r.Context, "id", id)
		if errors.Is(err, sql.ErrNoRows) && a.Cfg.RecreateMissingAccount && !credentialMode {
			u, err = a.createUser(r, text(m, "device_name"), id, false)
		}
	}
	if errors.Is(err, sql.ErrNoRows) {
		if credentialMode {
			return nil, fail(401, "Invalid credentials")
		}
		if auth {
			return object{"result": "not_found", "message": "用户不存在", "user_id": id}, nil
		}
		return nil, fail(404, "User not found")
	}
	if err != nil {
		return nil, err
	}
	if credentialMode && subtle.ConstantTimeCompare([]byte(providedCredential), []byte(stringValue(u.RecoveryToken))) != 1 {
		return nil, fail(401, "Invalid credentials")
	}
	if credentialMode && u.Session == nil {
		return nil, fail(401, "Invalid credentials")
	}
	if err = a.CheckAllowed(r.Context, &u, 406); err != nil {
		if auth {
			if e, ok := err.(*Error); ok {
				info, _ := e.Body["baninfo"].(object)
				message := "账号已被封禁"
				if info["type"] == "unauthorized" {
					message = "账号未授权"
				}
				return nil, &Error{Status: e.Status, Body: object{"baninfo": info, "user": u.ID, "end_date": integer(u.BanEndTime), "message": message}}
			}
		}
		return nil, err
	}
	session, err := token(16)
	if err != nil {
		return nil, err
	}
	recovery := stringValue(u.RecoveryToken)
	if auth || m["request_recovery_token"] == true {
		recovery, err = token(32)
		if err != nil {
			return nil, err
		}
	}
	device := stringValue(u.DeviceIdentifier)
	if d := text(m, "device"); d != "" && !auth {
		device = d
	}
	query := repository.RotateSession
	args := []any{session, recovery, device, remoteIP(r), r.UserAgent, u.ID}
	if credentialMode {
		query = repository.RotateSessionWithCredential
		args = append(args, providedCredential)
	}
	updated, err := a.Store.ExecContext(r.Context, query, args...)
	if err != nil {
		return nil, err
	}
	if credentialMode {
		rows, err := updated.RowsAffected()
		if err != nil {
			return nil, err
		}
		if rows != 1 {
			return nil, fail(401, "Invalid credentials")
		}
	}
	u.DeviceIdentifier = &device
	if auth {
		responseDevice := text(m, "device")
		if len([]rune(responseDevice)) < 32 {
			responseDevice, err = uuid()
			if err != nil {
				return nil, err
			}
		}
		key, err := token(32)
		if err != nil {
			return nil, err
		}
		scheme := "ws"
		if r.Secure {
			scheme = "wss"
		}
		return object{"authinfo": object{"user": u.ID, "device": responseDevice, "key": key, "recovery_token": recovery}, "session": session, "resource_list": authLoginResources, "external_links": []string{}, "websocket_url": scheme + "://" + r.Host + "/account/ws", "websocket_auth": "Basic " + base64.StdEncoding.EncodeToString([]byte(u.ID+":"+session))}, nil
	}
	result := object{"user": u.ID, "device": nil, "key": strings.Repeat("0", 64), "session": session, "resource_list": loginResources}
	if a.LegacyUA(r) {
		augmentLegacy(result, u)
	}
	return result, nil
}

// 为旧版响应补充字段
func augmentLegacy(result object, u entity.User) {
	result["device"] = stringValue(u.DeviceIdentifier)
	result["tos_url"] = "https://www.thatgamecompany.com/terms-of-service/"
	result["pp_url"] = "https://www.thatgamecompany.com/privacy-policy/"
	result["external_links"] = []string{}
	result["player_badge_type"] = -1
	result["motd_title"] = "你知道吗？"
	result["motd"] = "在遇境的天上，有许多星星。这些星星来自先灵或你的友人，点亮他们吧！"
	result["motd_timestamp"] = now()
	result["motd_version"] = 0
	result["motd_buttons"] = []any{}
	result["motd_season_end"] = 0
}
