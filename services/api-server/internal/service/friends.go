package service

import (
	"encoding/json"
	"strconv"
	"strings"
)

func (a *Service) Friends(r Request, m object) (object, error) {
	u, err := a.authenticated(r, m)
	if err != nil {
		return nil, err
	}
	limit, offset := number(m, "page_max", 50), number(m, "page_offset", 0)
	if limit < 0 || offset < 0 || limit > 1000 {
		return nil, fail(400, "Invalid pagination")
	}
	players := []string{}
	if v, exists := m["players"]; exists {
		list, ok := v.([]any)
		if !ok || len(list) > 1000 {
			return nil, fail(400, "Invalid players")
		}
		for _, v := range list {
			if id, ok := v.(string); ok && strings.TrimSpace(id) != "" {
				players = append(players, strings.TrimSpace(id))
			}
		}
	}
	records, err := a.store.Friends(r.Context, u.ID, players, limit, offset)
	if err != nil {
		return nil, err
	}
	items := []object{}
	for _, f := range records {
		var created any
		if f.Created.Valid {
			created = f.Created.String
		}
		stamp := f.WhenCreated
		if stamp == 0 {
			stamp = now()
		}
		items = append(items, object{"friend_id": f.ID, "user_id": f.ID, "nickname": f.Nickname, "platform": f.Platform, "device": f.Device, "is_online": false, "last_seen": nil, "created_at": created, "relationship": "friend", "status": "active", "given": f.Given, "recvd": f.Received, "level": f.Level, "when_created": stamp, "player_badge_type": f.Badge, "attitude": f.Attitude, "abilities": intList(f.Abilities), "hints": intList(f.Hints), "soft_deleted": f.Removed, "they_soft_deleted": f.Removed, "local_soft_deleted": f.Removed || f.LocalDeleted, "blocked": false, "local_blocked": f.BlockedByFriend, "they_blocked": f.BlockedByFriend})
	}
	return object{"set_friends": items, "set_friend_count": len(items), "result": "success", "user_id": u.ID, "total_friends": len(items), "page_max": limit, "page_offset": offset, "timestamp": now() * 1000}, nil
}
func intList(raw string) []int64 {
	var values []json.RawMessage
	if json.Unmarshal([]byte(raw), &values) != nil {
		return []int64{}
	}
	out := []int64{}
	for _, v := range values {
		s := string(v)
		var text string
		if json.Unmarshal(v, &text) == nil {
			s = text
		}
		if n, e := strconv.ParseInt(s, 10, 64); e == nil {
			out = append(out, n)
		}
	}
	return out
}
