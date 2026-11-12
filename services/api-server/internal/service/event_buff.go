package service

import (
	"fmt"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

func (a *Service) PurchaseEventBuff(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	base := func(reason string) (Object, error) {
		return Object{"result": "error", "update_buffs": []Object{}, "error": reason, "get_buffs": []Object{}, "get_buffs_sign": ""}, nil
	}
	if uid == "" || text(m, "session") == "" {
		return base("Invalid request body")
	}
	if _, err := a.magicUser(r, m, true); err != nil {
		return base("Invalid session or user not found")
	}
	var identifier any
	for _, key := range []string{"buff_id", "event_buff_id", "buff_name", "item_name", "name"} {
		if value := m[key]; value != nil && fmt.Sprint(value) != "" {
			identifier = value
			break
		}
	}
	defs := a.versionMagicDefinitions("get_buff_defs", "account__buff__get_buff_defs.json", r.UserAgent)
	var found Object
	for _, d := range defs {
		if fmt.Sprint(d["id"]) == fmt.Sprint(identifier) || text(d, "name") == fmt.Sprint(identifier) {
			found = d
			break
		}
	}
	if found == nil {
		for _, d := range a.versionMagicDefinitions("get_consumable_defs", "account__consumable__get_consumable_defs.json", r.UserAgent) {
			if text(d, "name") == fmt.Sprint(identifier) || fmt.Sprint(d["id"]) == fmt.Sprint(identifier) {
				for _, b := range defs {
					if defInt(b["id"]) == defInt(d["buff_id"]) {
						found = b
						break
					}
				}
				break
			}
		}
	}
	if found != nil {
		stamp := now()
		duration := defSeconds(found["duration"])
		if duration <= 0 {
			duration = 1200
		}
		cooldown := defSeconds(found["cooldown"])
		if _, err := a.store.ExecContext(r.Context, repository.UpsertBuff, uid, fmt.Sprint(found["id"]), stamp+duration, stamp+cooldown, uid); err != nil {
			return nil, err
		}
	}
	buffs, err := a.buffs(r, uid)
	if err != nil {
		return nil, err
	}
	buffs["result"] = "ok"
	buffs["update_buffs"] = []Object{}
	return buffs, nil
}
