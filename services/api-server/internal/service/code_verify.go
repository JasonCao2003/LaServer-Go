package service

import (
	"database/sql"
	"errors"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
	"regexp"
	"strings"
)

var inviteCodeFormat = regexp.MustCompile(`^[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}$`)

func inviteError(status int, message string) error {
	return &Error{Status: status, Body: Object{"error": message, "result": "error"}}
}
func (a *Service) VerifyCode(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	code := strings.ToUpper(strings.TrimSpace(text(m, "code")))
	if uid == "" || text(m, "session") == "" || code == "" {
		return nil, inviteError(400, "Missing required fields")
	}
	u, err := a.findUser(r.Context, "id", uid)
	if errors.Is(err, sql.ErrNoRows) || err == nil && stringValue(u.Session) != text(m, "session") {
		return nil, inviteError(401, "User authentication failed")
	}
	if err != nil {
		return nil, err
	}
	if !inviteCodeFormat.MatchString(code) {
		return nil, inviteError(400, "Invalid invite code format")
	}
	var inviter, token, formatted, nickname string
	var created int64
	err = a.store.QueryRowContext(r.Context, repository.ReadInviteByFormattedCode, code).Scan(&inviter, &token, &formatted, &nickname, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, inviteError(404, "Invite code not found")
	}
	if err != nil {
		return nil, err
	}
	if inviter == uid {
		return nil, inviteError(400, "You cannot use your own invite code")
	}
	return Object{"result": "ok", "type": "i", "code": token, "invite_info": Object{"inviter_user_id": inviter, "formatted_code": formatted, "nickname": nickname, "created_at": created}}, nil
}
