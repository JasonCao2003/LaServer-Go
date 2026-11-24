// 管理后台好友增删与审计

package relationships

import (
	"log/slog"
	"net"
	"sort"
	"strings"
)

// 判断管理员请求是否允许
func (a *Service) adminAllowed(r Request, m Object) bool {
	if a.Cfg.AdminToken != "" {
		token := r.Header.Get("X-Admin-Token")
		if token == "" {
			token = text(m, "admin_token")
		}
		return token != "" && token == a.Cfg.AdminToken
	}
	ip := net.ParseIP(r.RemoteIP)
	return ip != nil && ip.IsLoopback()
}

// 返回管理员拒绝错误
func adminForbidden() error {
	return &Error{Status: 403, Body: Object{"ok": false, "error": "forbidden"}}
}

// 列出管理员好友
func (a *Service) AdminFriendsList(r Request, m Object) (Object, error) {
	if !a.adminAllowed(r, m) {
		return nil, adminForbidden()
	}
	uid := strings.TrimSpace(text(m, "user"))
	if uid == "" {
		return Object{"ok": false, "error": "缺少 user"}, nil
	}
	ids, err := a.Store.AdminFriendIDs(r.Context, uid)
	if err != nil {
		return Object{"ok": false, "error": "读取失败"}, nil
	}
	keyword := strings.ToLower(strings.TrimSpace(text(m, "keyword")))
	rows := []Object{}
	for _, friendID := range ids {
		row, err := a.Store.AdminFriendRow(r.Context, uid, friendID)
		if err != nil {
			return Object{"ok": false, "error": "读取失败"}, nil
		}
		item := Object{"friend_id": friendID, "status": row.Status, "reverse_status": row.ReverseStatus, "two_way": row.TwoWay, "nickname": row.Nickname, "given": row.Given, "recvd": row.Received, "remark": row.Remark, "device_name": row.DeviceName}
		if keyword != "" {
			haystack := strings.ToLower(friendID + row.Nickname + row.Remark + row.DeviceName)
			if !strings.Contains(haystack, keyword) {
				continue
			}
		}
		rows = append(rows, item)
	}
	sort.Slice(rows, func(i, j int) bool {
		left, right := rows[i], rows[j]
		leftActive := text(left, "status") == "active"
		rightActive := text(right, "status") == "active"
		if leftActive != rightActive {
			return leftActive
		}
		leftName := text(left, "nickname")
		if leftName == "" {
			leftName = text(left, "friend_id")
		}
		rightName := text(right, "nickname")
		if rightName == "" {
			rightName = text(right, "friend_id")
		}
		return leftName < rightName
	})
	return Object{"ok": true, "user": uid, "count": len(rows), "friends": rows}, nil
}

// 设置管理员好友关系
func (a *Service) AdminFriendsSet(r Request, m Object) (Object, error) {
	if !a.adminAllowed(r, m) {
		return nil, adminForbidden()
	}
	uid := strings.TrimSpace(text(m, "user"))
	friendID := strings.TrimSpace(text(m, "friend"))
	if friendID == "" {
		friendID = strings.TrimSpace(text(m, "friend_id"))
	}
	action := strings.ToLower(strings.TrimSpace(text(m, "action")))
	if action == "" {
		action = strings.ToLower(strings.TrimSpace(text(m, "state")))
	}
	if action == "" {
		action = "add"
	}
	switch action {
	case "del", "delete":
		action = "remove"
	case "add", "create", "new":
		action = "add"
	}
	if uid == "" || friendID == "" {
		return Object{"ok": false, "detail": "缺少账号 id", "action": action, "user": uid, "friend": friendID, "pushed": false}, nil
	}
	if uid == friendID {
		return Object{"ok": false, "detail": "同一个账号: 不能和自己建立好友关系", "action": action, "user": uid, "friend": friendID, "pushed": false}, nil
	}
	if action != "add" && action != "remove" && action != "restore" {
		return Object{"ok": false, "detail": "未知操作: " + action, "action": action, "user": uid, "friend": friendID, "pushed": false}, nil
	}
	ok, detail, err := a.Store.AdminSetFriendship(r.Context, uid, friendID, action, r.RemoteIP, action)
	if err != nil {
		return Object{"ok": false, "detail": "执行失败", "action": action, "user": uid, "friend": friendID, "pushed": false}, nil
	}
	if ok {
		slog.Info("admin friends change", "action", action, "user", uid, "friend", friendID, "ip", r.RemoteIP, "detail", detail)
	}
	return Object{"ok": ok, "detail": detail, "action": action, "user": uid, "friend": friendID, "pushed": false}, nil
}

// 清空管理员好友关系
func (a *Service) AdminFriendsClear(r Request, m Object) (Object, error) {
	if !a.adminAllowed(r, m) {
		return nil, adminForbidden()
	}
	uid := strings.TrimSpace(text(m, "user"))
	if uid == "" {
		return Object{"ok": false, "error": "缺少 user"}, nil
	}
	ids, err := a.Store.AdminFriendIDs(r.Context, uid)
	if err != nil {
		return Object{"ok": false, "error": "执行失败"}, nil
	}
	removed := []string{}
	failed := []string{}
	for _, friendID := range ids {
		ok, _, err := a.Store.AdminSetFriendship(r.Context, uid, friendID, "remove", r.RemoteIP, "clear")
		if err != nil || !ok {
			failed = append(failed, friendID)
			continue
		}
		removed = append(removed, friendID)
	}
	slog.Info("admin friends clear", "user", uid, "removed", len(removed), "failed", len(failed), "ip", r.RemoteIP)
	return Object{"ok": len(failed) == 0, "user": uid, "removed": removed, "removed_count": len(removed), "failed": failed}, nil
}
