package service

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (a *Service) configuredObject(name string) (Object, error) {
	if a.cfg.GameConfigDir == "" {
		return nil, fail(503, "Game configuration unavailable")
	}
	b, err := os.ReadFile(filepath.Join(a.cfg.GameConfigDir, name))
	if err != nil {
		return nil, fail(503, "Game configuration unavailable")
	}
	b = bytes.TrimPrefix(b, []byte{0xef, 0xbb, 0xbf})
	var result Object
	if err = json.Unmarshal(b, &result); err != nil || result == nil {
		return nil, fail(503, "Invalid game configuration")
	}
	return result, nil
}

func (a *Service) QuestDefinitions(_ Request, _ Object) (Object, error) {
	return a.configuredObject("quest.json")
}
func (a *Service) IAPList(_ Request, _ Object) (Object, error) {
	return a.configuredObject("iaplist.json")
}
func (a *Service) QuestionnaireGroups(_ Request, _ Object) (Object, error) {
	return Object{"get_questionnaire_groups": []any{}}, nil
}

func (a *Service) Invites(r Request, m Object) (Object, error) {
	userID, session := text(m, "user"), text(m, "session")
	if userID == "" || session == "" {
		return nil, fail(400, "Invalid request body")
	}
	u, err := a.findUser(r.Context, "id", userID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != session) {
		return nil, fail(401, "Invalid session or user not found")
	}
	if err != nil {
		return nil, err
	}
	rows, err := a.store.Invites(r.Context, u.ID)
	if err != nil {
		return nil, err
	}
	out := make([]Object, 0, len(rows))
	for _, row := range rows {
		var friend any
		if row.FriendID != nil {
			friend = *row.FriendID
		}
		out = append(out, Object{"inviter_user_id": row.InviterUserID, "token_id": row.TokenID, "nickname": row.Nickname, "friend_id": friend, "created": row.Created})
	}
	return Object{"invites": out}, nil
}

func (a *Service) CheckInvite(r Request, m Object) (Object, error) {
	userID, session, tokenID := text(m, "user"), text(m, "session"), text(m, "token_id")
	if userID == "" {
		userID = text(m, "user_id")
	}
	if userID == "" || session == "" || tokenID == "" {
		return nil, &Error{Status: 400, Body: Object{"error": "Missing required fields", "invite_result": "error", "invite_exists": false}}
	}
	invite, err := a.store.InviteByToken(r.Context, tokenID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &Error{Status: 404, Body: Object{"error": "Invite token not found", "invite_result": "error", "invite_exists": false}}
	}
	if err != nil {
		return nil, err
	}
	checkedAt := now()
	var code any
	if invite.FormattedCode != nil {
		code = *invite.FormattedCode
	}
	details := Object{"inviter_user_id": invite.InviterUserID, "formatted_code": code, "nickname": invite.Nickname, "created_at": invite.Created, "is_available": true, "invite_age_seconds": checkedAt - invite.Created}
	if user, findErr := a.findUser(r.Context, "id", userID); findErr == nil && stringValue(user.Session) == session && invite.InviterUserID == user.ID {
		details["is_own_invite"] = true
	}
	return Object{"invite_token_id": invite.TokenID, "invite_exists": true, "invite_result": "ok", "invite_details": details, "checked_at": checkedAt}, nil
}

func (a *Service) DeleteInvite(r Request, m Object) (Object, error) {
	userID, session, tokenID := text(m, "user"), text(m, "session"), text(m, "token_id")
	if userID == "" {
		userID = text(m, "user_id")
	}
	if userID == "" || session == "" || tokenID == "" {
		return nil, &Error{Status: 400, Body: Object{"error": "Missing required fields", "message": "缺少必需字段"}}
	}
	u, err := a.findUser(r.Context, "id", userID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != session) {
		return nil, &Error{Status: 401, Body: Object{"error": "Invalid user session", "message": "无效的用户会话"}}
	}
	if err != nil {
		return nil, err
	}
	invite, err := a.store.InviteByToken(r.Context, tokenID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &Error{Status: 404, Body: Object{"error": "Invite record not found", "message": "邀请记录不存在"}}
	}
	if err != nil {
		return nil, err
	}
	if invite.InviterUserID != u.ID {
		return nil, &Error{Status: 403, Body: Object{"error": "Unauthorized to delete this invite", "message": "无权删除此邀请"}}
	}
	if _, err = a.store.DeleteInvite(r.Context, tokenID, u.ID); err != nil {
		return nil, err
	}
	remaining, err := a.store.Invites(r.Context, u.ID)
	if err != nil {
		return nil, err
	}
	list := make([]Object, 0, len(remaining))
	for _, row := range remaining {
		var friend any
		if row.FriendID != nil {
			friend = *row.FriendID
		}
		list = append(list, Object{"inviter_user_id": row.InviterUserID, "token_id": row.TokenID, "nickname": row.Nickname, "friend_id": friend, "created": row.Created})
	}
	return Object{"invite_token_id": tokenID, "invite_deleted": true, "invites": list}, nil
}

func formatInviteCode(tokenID string) (string, error) {
	b, err := hex.DecodeString(strings.ReplaceAll(tokenID, "-", ""))
	if err != nil || len(b) != 16 {
		return "", errors.New("invalid invite token")
	}
	v := uint64(b[8]&7)<<56 | uint64(b[9])<<48 | uint64(b[10])<<40 | uint64(b[11])<<32 | uint64(b[12])<<24 | uint64(b[13])<<16 | uint64(b[14])<<8 | uint64(b[15])
	const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	var code [12]byte
	for i := 0; i < 12; i++ {
		code[i] = alphabet[(v>>uint(55-5*i))&31]
	}
	return string(code[:4]) + "-" + string(code[4:8]) + "-" + string(code[8:]), nil
}

func (a *Service) CreateInvite(r Request, m Object) (Object, error) {
	userID, session, nickname := text(m, "user"), text(m, "session"), text(m, "invite_nickname")
	if userID == "" {
		userID = text(m, "user_id")
	}
	if userID == "" {
		return nil, &Error{Status: 400, Body: Object{"error": "Missing required field: user or user_id", "code": "MISSING_USER"}}
	}
	if session == "" {
		return nil, &Error{Status: 400, Body: Object{"error": "Missing required field: session", "code": "MISSING_SESSION"}}
	}
	if nickname == "" {
		return nil, &Error{Status: 400, Body: Object{"error": "Missing required field: invite_nickname", "code": "MISSING_NICKNAME"}}
	}
	u, err := a.findUser(r.Context, "id", userID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != session) {
		return nil, &Error{Status: 401, Body: Object{"error": "Invalid user session", "code": "INVALID_SESSION"}}
	}
	if err != nil {
		return nil, err
	}
	if integer(u.IsBanned) == 1 && (integer(u.BanEndTime) == 0 || now() < integer(u.BanEndTime)) {
		return nil, &Error{Status: 403, Body: Object{"error": "User account is banned", "code": "USER_BANNED"}}
	}
	var tokenID, formatted string
	created := now()
	for attempt := 0; attempt < 5; attempt++ {
		tokenID, err = uuid()
		if err != nil {
			return nil, err
		}
		formatted, err = formatInviteCode(tokenID)
		if err != nil {
			return nil, err
		}
		err = a.store.CreateInvite(r.Context, u.ID, tokenID, formatted, nickname, created)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, &Error{Status: 500, Body: Object{"error": "Failed to generate unique invite code", "code": "CODE_GENERATION_FAILED"}}
	}
	return Object{"invite_nickname": nickname, "invite_token_id": tokenID, "invite_formatted_code": formatted, "invite_created_at": created, "inviter_user_id": u.ID, "result": "success"}, nil
}

func (a *Service) AcceptInvite(r Request, m Object) (Object, error) {
	userID, session, tokenID := text(m, "user"), text(m, "session"), text(m, "token_id")
	if userID == "" {
		userID = text(m, "user_id")
	}
	if userID == "" || session == "" || tokenID == "" {
		return nil, &Error{Status: 400, Body: Object{"error": "Missing required fields", "invite_result": "error"}}
	}
	u, err := a.findUser(r.Context, "id", userID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != session) {
		return nil, &Error{Status: 401, Body: Object{"error": "Invalid user session", "invite_result": "error"}}
	}
	if err != nil {
		return nil, err
	}
	if integer(u.IsBanned) == 1 && (integer(u.BanEndTime) == 0 || now() < integer(u.BanEndTime)) {
		return nil, &Error{Status: 403, Body: Object{"error": "User account is banned", "invite_result": "error"}}
	}
	invite, err := a.store.InviteByToken(r.Context, tokenID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &Error{Status: 404, Body: Object{"error": "Invite token not found", "invite_result": "error"}}
	}
	if err != nil {
		return nil, err
	}
	if invite.FriendID != nil {
		return nil, &Error{Status: 400, Body: Object{"error": "Invite has already been used", "invite_result": "error"}}
	}
	if invite.InviterUserID == u.ID {
		return nil, &Error{Status: 400, Body: Object{"error": "You cannot accept your own invite", "invite_result": "error"}}
	}
	acceptedAt := now()
	if acceptedAt-invite.Created > 7*24*60*60 {
		return nil, &Error{Status: 400, Body: Object{"error": "Invite has expired", "invite_result": "error"}}
	}
	inviter, err := a.findUser(r.Context, "id", invite.InviterUserID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &Error{Status: 404, Body: Object{"error": "Inviter user not found", "invite_result": "error"}}
	}
	if err != nil {
		return nil, err
	}
	nickname := text(m, "invite_nickname")
	if strings.TrimSpace(nickname) == "" {
		nickname = invite.Nickname
	}
	if strings.TrimSpace(nickname) == "" {
		nickname = "Friend"
	}
	created := now()
	accepted, err := a.store.AcceptInvite(r.Context, tokenID, u.ID, inviter.ID, nickname, created)
	if err != nil {
		return nil, err
	}
	if !accepted {
		return nil, &Error{Status: 400, Body: Object{"error": "Invite has already been used", "invite_result": "error"}}
	}
	friendCount, err := a.store.FriendCount(r.Context, u.ID)
	if err != nil {
		return nil, err
	}
	abilities := []int{19, 6, 9, 2, 27, 1, 10}
	var code any
	if invite.FormattedCode != nil {
		code = *invite.FormattedCode
	}
	createdISO := time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")
	update := []Object{{"unlinked_star_sku": "", "given": 0, "recvd": 1, "level": 1, "abilities": abilities, "when_created": createdISO, "friend_id": inviter.ID, "last_seen_outfit": nil, "nickname": nickname, "soft_deleted": nil, "they_soft_deleted": nil}}
	details := Object{"inviter_user_id": invite.InviterUserID, "formatted_code": code, "original_nickname": invite.Nickname, "custom_nickname": nickname, "created_at": invite.Created, "friend_created_at": created, "is_used": true}
	return Object{"invite_token_id": tokenID, "invite_accepted": true, "invite_result": "ok", "invite_friend_id": inviter.ID, "update_friends": update, "set_friend_count": friendCount, "update_online_friends": []any{}, "invite_details": details, "accepted_at": acceptedAt, "server_time": acceptedAt}, nil
}
