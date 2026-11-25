package relationships

import (
	"log/slog"
	"strconv"
	"strings"
	"time"
)

func (a *Service) AdminAccountsList(r Request, m Object) (Object, error) {
	if !a.adminAllowed(r, m) {
		return nil, adminForbidden()
	}
	keyword := strings.TrimSpace(text(m, "keyword"))
	page := 1
	if value := text(m, "page"); value != "" {
		var err error
		page, err = strconv.Atoi(value)
		if err != nil || page < 1 || page > 100000 {
			return nil, &Error{Status: 400, Body: Object{"ok": false, "error": "invalid page"}}
		}
	}
	if len(keyword) > 256 {
		return nil, &Error{Status: 400, Body: Object{"ok": false, "error": "keyword too long"}}
	}
	items, total, banned, matched, unauthorized, err := a.Store.AdminAccounts(r.Context, keyword, (page-1)*50)
	if err != nil {
		return nil, &Error{Status: 500, Body: Object{"ok": false, "error": "读取失败"}}
	}
	return Object{"ok": true, "accounts": items, "total": total, "banned": banned, "matched": matched, "unauthorized": unauthorized}, nil
}

func (a *Service) AdminAccountsChange(r Request, m Object) (Object, error) {
	if !a.adminAllowed(r, m) {
		return nil, adminForbidden()
	}
	id, action := strings.TrimSpace(text(m, "user")), text(m, "action")
	if id == "" || len(id) > 256 {
		return nil, &Error{Status: 400, Body: Object{"ok": false, "error": "invalid user"}}
	}
	var count int64
	var err error
	switch action {
	case "authorize", "revoke":
		count, err = a.Store.SetAdminAccountAuthorization(r.Context, id, action == "authorize")
	case "unban":
		count, err = a.Store.SetAdminAccountBan(r.Context, id, false, 0, 0, 0)
	case "ban":
		duration, parseErr := strconv.ParseInt(text(m, "seconds"), 10, 64)
		if parseErr != nil || duration < 0 || duration > 315360000 {
			return nil, &Error{Status: 400, Body: Object{"ok": false, "error": "invalid duration"}}
		}
		start, end := time.Now().Unix(), int64(0)
		if duration > 0 {
			end = start + duration
		}
		count, err = a.Store.SetAdminAccountBan(r.Context, id, true, duration, start, end)
	default:
		return nil, &Error{Status: 400, Body: Object{"ok": false, "error": "invalid action"}}
	}
	if err != nil {
		return nil, &Error{Status: 500, Body: Object{"ok": false, "error": "写入失败"}}
	}
	if count == 0 {
		return nil, &Error{Status: 404, Body: Object{"ok": false, "error": "账号不存在"}}
	}
	slog.Info("admin account change", "user", id, "action", action)
	return Object{"ok": true}, nil
}
