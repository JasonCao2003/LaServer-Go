// 会话与用户校验

package core

import (
	"context"
	"database/sql"
	"errors"

	entity "github.com/JasonCao2003/LaServer-Go/services/api-server/internal/model"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

// 按条件查找用户
func (a *Base) FindUser(ctx context.Context, key, value string) (entity.User, error) {
	var u entity.User
	var verified int
	// key is chosen by the caller from this allowlist, never from request input.
	if key != "id" && key != "device_identifier" {
		return u, errors.New("invalid lookup")
	}
	err := a.Store.QueryRowContext(ctx, repository.UserLookup(key), value).Scan(&u.ID, &u.DeviceIdentifier, &u.Session, &u.RecoveryToken, &u.IsAuthorized, &u.IsBanned, &u.BanEndTime, &u.Platform, &u.CreatedAt, &verified)
	if err == nil && a.Cfg.LoginMode != "legacy_dev" && verified != 1 {
		u.Session = nil
	}
	return u, err
}

// 构造封禁错误
func BanError(u entity.User, status int) error {
	info := object{"user_id": u.ID, "end_date": Integer(u.BanEndTime), "type": "account"}
	if until := Integer(u.BanEndTime); until > Now() {
		remaining := until - Now()
		info["remaining_seconds"] = remaining
		info["remaining_days"] = remaining / 86400
	}
	return &Error{status, object{"error": "Account banned", "baninfo": info}}
}

// 校验账号是否允许登录
func (a *Base) CheckAllowed(ctx context.Context, u *entity.User, status int) error {
	if Integer(u.IsBanned) == 1 {
		if Integer(u.BanEndTime) == 0 || Integer(u.BanEndTime) > Now() {
			return BanError(*u, status)
		}
		if _, err := a.Store.ExecContext(ctx, repository.ClearExpiredBan, u.ID); err != nil {
			return err
		}
	}
	if Integer(u.IsAuthorized) == 0 {
		return &Error{status, object{"error": "Account not authorized", "baninfo": object{"user_id": u.ID, "end_date": 0, "type": "unauthorized", "message": "账号尚未授权，请联系管理员"}}}
	}
	return nil
}

// 校验会话并返回用户
func (a *Base) Authenticated(r Request, m object) (entity.User, error) {
	var u entity.User
	if Text(m, "user") == "" || Text(m, "session") == "" {
		return u, Fail(400, "Invalid request body")
	}
	u, err := a.FindUser(r.Context, "id", Text(m, "user"))
	if errors.Is(err, sql.ErrNoRows) {
		return u, Fail(401, "Invalid session or user not found")
	}
	if err != nil {
		return u, err
	}
	if StringValue(u.Session) != Text(m, "session") {
		return u, Fail(401, "Invalid session or user not found")
	}
	return u, a.CheckAllowed(r.Context, &u, 406)
}

// 校验任务请求用户
func (a *Base) QuestUser(r Request, m Object) (string, error) {
	uid := Text(m, "user")
	if uid == "" {
		uid = Text(m, "user_id")
	}
	if uid == "" || Text(m, "session") == "" {
		return "", Fail(400, "Invalid request body")
	}
	u, err := a.FindUser(r.Context, "id", uid)
	if errors.Is(err, sql.ErrNoRows) || err == nil && StringValue(u.Session) != Text(m, "session") {
		return "", Fail(401, "Invalid session or user not found")
	}
	if err != nil {
		return "", err
	}
	return uid, nil
}
