package gameapi

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	entity "github.com/JasonCao2003/LaServer-Go/services/api-server/internal/model"
)

func token(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
func uuid() (string, error) {
	s, e := token(16)
	if e != nil {
		return "", e
	}
	return s[:8] + "-" + s[8:12] + "-4" + s[13:16] + "-8" + s[17:20] + "-" + s[20:], nil
}

var versionPattern = regexp.MustCompile(`Sky[^/]*/([0-9]+)\.([0-9]+)\.([0-9]+)`)

func legacyVersion(ua string) bool {
	m := versionPattern.FindStringSubmatch(ua)
	if len(m) == 0 {
		return false
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	return major == 0 && minor < 5
}

func (a *API) findUser(ctx context.Context, key, value string) (entity.User, error) {
	var u entity.User
	// key is chosen by the caller from this allowlist, never from request input.
	if key != "id" && key != "device_identifier" {
		return u, errors.New("invalid lookup")
	}
	err := a.db.QueryRowContext(ctx, "SELECT id,device_identifier,session,recovery_token,is_authorized,is_banned,ban_end_time,platform FROM user WHERE "+key+"=?", value).Scan(&u.ID, &u.DeviceIdentifier, &u.Session, &u.RecoveryToken, &u.IsAuthorized, &u.IsBanned, &u.BanEndTime, &u.Platform)
	return u, err
}

func (a *API) createUser(r *http.Request, name, id string, authCreate bool) (entity.User, error) {
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
	if authCreate {
		device, err = uuid()
		if err != nil {
			return u, err
		}
		if !a.cfg.AutoAuthorize {
			session = "UNAUTH_" + session
		}
	}
	authorized := 0
	if a.cfg.AutoAuthorize {
		authorized = 1
	}
	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		return u, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(r.Context(), `INSERT INTO user(id,device,device_identifier,session,recovery_token,is_authorized,ip_address,created_at,device_name,user_agent) VALUES(?,?,?,?,?,?,?,?,?,?)`, id, name, device, session, recovery, authorized, remoteIP(r), now(), name, r.UserAgent())
	if err != nil {
		return u, err
	}
	if _, err = tx.ExecContext(r.Context(), "INSERT INTO user_currency(user_id) VALUES(?)", id); err != nil {
		return u, err
	}
	for k, v := range a.cfg.DefaultCurrency {
		if !currencyField(k) {
			return u, fmt.Errorf("unknown default currency %q", k)
		}
		if _, err = tx.ExecContext(r.Context(), "UPDATE user_currency SET "+k+"=? WHERE user_id=?", v, id); err != nil {
			return u, err
		}
	}
	if err = tx.Commit(); err != nil {
		return u, err
	}
	return a.findUser(r.Context(), "id", id)
}

func (a *API) create(r *http.Request, m object) (object, error) {
	auth := r.URL.Path == "/account/auth/create"
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
		if !legacyUA(r) && !looksLegacy {
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
		return object{"authinfo": object{"user": u.ID, "device": stringValue(u.DeviceIdentifier), "key": key, "recovery_token": stringValue(u.RecoveryToken)}, "session": stringValue(u.Session), "resource_list": authCreateResources, "authorization_required": !a.cfg.AutoAuthorize, "message": "账号已创建"}, nil
	}
	result := object{"user": u.ID, "device": stringValue(u.DeviceIdentifier), "key": key, "session": stringValue(u.Session), "resource_list": createResources}
	if legacyUA(r) {
		augmentLegacy(result, u)
	}
	return result, nil
}

func banError(u entity.User, status int) error {
	info := object{"user_id": u.ID, "end_date": integer(u.BanEndTime), "type": "account"}
	if until := integer(u.BanEndTime); until > now() {
		remaining := until - now()
		info["remaining_seconds"] = remaining
		info["remaining_days"] = remaining / 86400
	}
	return &apiError{status, object{"error": "Account banned", "baninfo": info}}
}
func (a *API) checkAllowed(ctx context.Context, u *entity.User, status int) error {
	if integer(u.IsBanned) == 1 {
		if integer(u.BanEndTime) == 0 || integer(u.BanEndTime) > now() {
			return banError(*u, status)
		}
		if _, err := a.db.ExecContext(ctx, "UPDATE user SET is_banned=0,ban_duration=0,ban_start_time=0,ban_end_time=0 WHERE id=?", u.ID); err != nil {
			return err
		}
	}
	if integer(u.IsAuthorized) == 0 {
		return &apiError{status, object{"error": "Account not authorized", "baninfo": object{"user_id": u.ID, "end_date": 0, "type": "unauthorized", "message": "账号尚未授权，请联系管理员"}}}
	}
	return nil
}

func (a *API) login(r *http.Request, m object) (object, error) {
	id := strings.TrimSpace(text(m, "user"))
	var u entity.User
	var err error
	if _, hasUser := m["user"]; !hasUser && text(m, "device") != "" {
		u, err = a.findUser(r.Context(), "device_identifier", text(m, "device"))
	} else {
		if id == "" {
			return nil, fail(400, "Missing required field: user")
		}
		u, err = a.findUser(r.Context(), "id", id)
		if errors.Is(err, sql.ErrNoRows) && a.cfg.RecreateMissingAccount {
			u, err = a.createUser(r, text(m, "device_name"), id, false)
		}
	}
	auth := r.URL.Path == "/account/auth/login"
	if errors.Is(err, sql.ErrNoRows) {
		if auth {
			return object{"result": "not_found", "message": "用户不存在", "user_id": id}, nil
		}
		return nil, fail(404, "User not found")
	}
	if err != nil {
		return nil, err
	}
	if err = a.checkAllowed(r.Context(), &u, 406); err != nil {
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
	if d := text(m, "device"); d != "" {
		device = d
	}
	if _, err = a.db.ExecContext(r.Context(), "UPDATE user SET session=?,recovery_token=?,device_identifier=?,ip_address=?,user_agent=? WHERE id=?", session, recovery, device, remoteIP(r), r.UserAgent(), u.ID); err != nil {
		return nil, err
	}
	u.DeviceIdentifier = &device
	if auth {
		key, err := token(32)
		if err != nil {
			return nil, err
		}
		scheme := "ws"
		if r.TLS != nil {
			scheme = "wss"
		}
		return object{"authinfo": object{"user": u.ID, "device": device, "key": key, "recovery_token": recovery}, "session": session, "resource_list": authLoginResources, "external_links": []string{}, "websocket_url": scheme + "://" + r.Host + "/account/ws", "websocket_auth": "Basic " + base64.StdEncoding.EncodeToString([]byte(u.ID+":"+session))}, nil
	}
	result := object{"user": u.ID, "device": nil, "key": strings.Repeat("0", 64), "session": session, "resource_list": loginResources}
	if legacyUA(r) {
		augmentLegacy(result, u)
	}
	return result, nil
}

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

func (a *API) authenticated(r *http.Request, m object) (entity.User, error) {
	var u entity.User
	if text(m, "user") == "" || text(m, "session") == "" {
		return u, fail(400, "Invalid request body")
	}
	u, err := a.findUser(r.Context(), "id", text(m, "user"))
	if errors.Is(err, sql.ErrNoRows) {
		return u, fail(401, "Invalid session or user not found")
	}
	if err != nil {
		return u, err
	}
	if stringValue(u.Session) != text(m, "session") {
		return u, fail(401, "Invalid session or user not found")
	}
	return u, a.checkAllowed(r.Context(), &u, 406)
}
