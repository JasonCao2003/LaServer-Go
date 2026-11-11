package service

import "time"

// Until a live room registry is configured, authenticated sessions are not presence evidence.
func (a *Service) OnlineFriends(r Request, m object) (object, error) {
	u, err := a.authenticated(r, m)
	if err != nil {
		return nil, err
	}
	records, err := a.store.Statues(r.Context, u.ID)
	if err != nil {
		return nil, err
	}
	total := 0
	for _, f := range records {
		added := f.Given > 0 || f.Received > 0
		for _, id := range intList(f.Abilities) {
			if id == 1 {
				added = true
			}
		}
		if added {
			total++
		}
	}
	return object{"online_friends": []string{}, "refresh_online_friends": 30, "total_online": 0, "total_friends": total, "timestamp": now(), "user_id": u.ID}, nil
}

func (a *Service) FriendStatues(r Request, m object) (object, error) {
	u, err := a.authenticated(r, m)
	if err != nil {
		return nil, err
	}
	records, err := a.store.Statues(r.Context, u.ID)
	if err != nil {
		return nil, err
	}
	items := []object{}
	included := map[string]bool{}
	for _, f := range records {
		abilities := intList(f.Abilities)
		added := f.Given > 0 || f.Received > 0
		for _, id := range abilities {
			if id == 1 {
				added = true
			}
		}
		if !added && !f.Blocked && !f.BlockedByFriend {
			continue
		}
		included[f.ID] = true
		var created any
		if f.WhenCreated > 0 {
			created = time.Unix(f.WhenCreated, 0).UTC().Format("2006-01-02T15:04:05")
		}
		items = append(items, object{"friend_id": f.ID, "recvd": f.Received, "nickname": f.Nickname, "when_created": f.WhenCreated, "level": f.Level, "abilities": abilities, "hints": intList(f.Hints), "created_at": created, "soft_deleted": nil, "they_soft_deleted": nil, "unlinked_star_sku": "", "blocked": f.Blocked, "local_blocked": f.BlockedByFriend, "local_soft_deleted": f.LocalDeleted})
	}
	slots, err := a.store.Constellations(r.Context, u.ID)
	if err != nil {
		return nil, err
	}
	pages := []object{}
	for _, slot := range slots {
		if !included[slot.FriendID] {
			continue
		}
		for int64(len(pages)) <= slot.Page {
			pages = append(pages, object{"name": "", "friends": []object{}})
		}
		page := pages[slot.Page]
		page["friends"] = append(page["friends"].([]object), object{"friend_id": slot.FriendID, "constellation_node_index": slot.Node})
	}
	return object{"set_friend_statues": items, "constellation_friend_pages": pages, "result": "success", "user_id": u.ID, "total_friends": len(items), "total_pages": len(pages), "timestamp": now() * 1000}, nil
}
