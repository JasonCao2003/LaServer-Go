// 消耗品、增益定义与使用逻辑

package inventory

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

// 获取消耗品定义
func (a *Service) ConsumableDefinitions(r Request, m Object) (Object, error) {
	defs := a.VersionMagicDefinitions("get_consumable_defs", "account__consumable__get_consumable_defs.json", r.UserAgent)
	return Object{"get_consumable_defs": a.boundMagicDefinitions(r, m, "get_consumable_defs", defs, 86195)}, nil
}

// 获取增益定义
func (a *Service) BuffDefinitions(r Request, m Object) (Object, error) {
	defs := a.VersionMagicDefinitions("get_buff_defs", "account__buff__get_buff_defs.json", r.UserAgent)
	return Object{"get_buff_defs": a.boundMagicDefinitions(r, m, "get_buff_defs", defs, 86016)}, nil
}

// 绑定并筛选魔法定义
func (a *Service) boundMagicDefinitions(r Request, m Object, key string, defs []Object, limit int) []Object {
	major, minor, _, known := clientVersion(r.UserAgent)
	if known && (major > 0 || minor >= 26) {
		return defs
	}
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	priority := map[int64]bool{}
	if key == "get_consumable_defs" {
		for _, d := range defs {
			if isUGCMagic(d) {
				priority[defInt(d["id"])] = true
			}
		}
	}
	if uid != "" {
		if key == "get_consumable_defs" {
			rows, err := a.Store.QueryContext(r.Context, repository.ReadConsumables, uid)
			if err == nil {
				for rows.Next() {
					var id, cooldown, qty int64
					if rows.Scan(&id, &cooldown, &qty) == nil && qty > 0 {
						priority[id] = true
					}
				}
				rows.Close()
			}
		} else {
			consumables, err := a.Store.QueryContext(r.Context, repository.ReadConsumables, uid)
			if err == nil {
				owned := map[int64]bool{}
				for consumables.Next() {
					var id, cooldown, quantity int64
					if consumables.Scan(&id, &cooldown, &quantity) == nil && quantity > 0 {
						owned[id] = true
					}
				}
				consumables.Close()
				for _, d := range a.VersionMagicDefinitions("get_consumable_defs", "account__consumable__get_consumable_defs.json", r.UserAgent) {
					if owned[defInt(d["id"])] {
						priority[defInt(d["buff_id"])] = true
					}
				}
			}
			rows, err := a.Store.QueryContext(r.Context, repository.ReadBuffs, uid)
			if err == nil {
				for rows.Next() {
					var id, giver string
					var active, cooldown int64
					if rows.Scan(&id, &active, &cooldown, &giver) == nil {
						priority[defInt(id)] = true
					}
				}
				rows.Close()
			}
		}
	}
	ordered := make([]Object, 0, len(defs))
	for _, d := range defs {
		if priority[defInt(d["id"])] {
			ordered = append(ordered, d)
		}
	}
	for _, d := range defs {
		if !priority[defInt(d["id"])] {
			ordered = append(ordered, d)
		}
	}
	out := []Object{}
	for _, d := range ordered {
		candidate := d
		if key == "get_consumable_defs" {
			candidate = Object{}
			for _, field := range []string{"id", "name", "buff_id", "cooldown", "event", "icon", "is_toy", "param", "type"} {
				if value, ok := d[field]; ok {
					candidate[field] = value
				}
			}
		}
		next := append(out, candidate)
		encoded, _ := json.Marshal(Object{key: next})
		if len(encoded) > limit {
			break
		}
		out = next
	}
	return out
}

// 获取消耗品（新版）
func (a *Service) GetConsumables(r Request, m Object) (Object, error) {
	uid, err := a.MagicUser(r, m, false)
	if err != nil {
		return nil, err
	}
	items, err := a.Consumables(r, uid, 0)
	return Object{"get_consumables": items}, err
}

// 获取消耗品（旧版）
func (a *Service) GetConsumables1(r Request, m Object) (Object, error) {
	uid, err := a.MagicUser(r, m, true)
	if err != nil {
		return nil, err
	}
	items, err := a.Consumables(r, uid, 0)
	return Object{"get_consumables": items}, err
}

// 获取增益（新版）
func (a *Service) GetBuffs(r Request, m Object) (Object, error) {
	uid, err := a.MagicUser(r, m, false)
	if err != nil {
		return nil, err
	}
	return a.Buffs(r, uid)
}

// 获取增益（旧版）
func (a *Service) GetBuffs1(r Request, m Object) (Object, error) {
	uid, err := a.MagicUser(r, m, true)
	if err != nil {
		return nil, err
	}
	return a.Buffs(r, uid)
}

// 移除增益
func (a *Service) RemoveBuff(r Request, m Object) (Object, error) {
	uid, err := a.MagicUser(r, m, true)
	if err != nil {
		return nil, err
	}
	id := text(m, "buff_id")
	if id == "" {
		id = text(m, "id")
	}
	if id == "" {
		id = text(m, "buff_name")
	}
	if id != "" {
		if _, err = a.Store.ExecContext(r.Context, repository.DeleteBuff, uid, id); err != nil {
			return nil, err
		}
	}
	result, err := a.Buffs(r, uid)
	if err != nil {
		return nil, err
	}
	result["remove_buff"] = "success"
	return result, nil
}

// 使用消耗品
func (a *Service) ConsumeItem(r Request, m Object) (Object, error) {
	uid, err := a.MagicUser(r, m, true)
	if err != nil {
		return nil, err
	}
	defs := a.VersionMagicDefinitions("get_consumable_defs", "account__consumable__get_consumable_defs.json", r.UserAgent)
	var definition Object
	for _, candidate := range []any{m["consumable_id"], m["consumable_name"], m["item_name"], m["name"]} {
		for _, d := range defs {
			if fmt.Sprint(d["id"]) == fmt.Sprint(candidate) || text(d, "name") == fmt.Sprint(candidate) {
				definition = d
				break
			}
		}
		if definition != nil {
			break
		}
	}
	status := "failed"
	var definitionID int64
	if definition != nil {
		id := defInt(definition["id"])
		definitionID = id
		count := number(m, "consumable_count", 1)
		if count <= 0 {
			count = 1
		}
		tx, e := a.Store.BeginTx(r.Context, nil)
		if e != nil {
			return nil, e
		}
		defer tx.Rollback()
		query := repository.ConsumeItem
		if isUGCMagic(definition) {
			query = repository.ConsumeUGCItem
		}
		args := []any{count, now() + defSeconds(definition["cooldown"]), uid, id}
		if query == repository.ConsumeItem {
			args = append(args, count)
		}
		res, e := tx.ExecContext(r.Context, query, args...)
		if e != nil {
			return nil, e
		}
		changed, e := res.RowsAffected()
		if e != nil {
			return nil, e
		}
		if changed > 0 {
			status = "success"
			buffID := defInt(definition["buff_id"])
			if buffID != 0 {
				stamp := now()
				buffDefs := a.MagicDefinitions("get_buff_defs", "account__buff__get_buff_defs.json")
				active, cooldown := int64(1200), int64(0)
				for _, bd := range buffDefs {
					if defInt(bd["id"]) == buffID {
						if duration := defSeconds(bd["duration"]); duration > 0 {
							active = duration
						}
						cooldown = defSeconds(bd["cooldown"])
						break
					}
				}
				_, e = tx.ExecContext(r.Context, repository.UpsertBuff, uid, fmt.Sprint(buffID), stamp+active, stamp+cooldown, uid)
				if e != nil {
					return nil, e
				}
			} else if text(definition, "name") == "resize_potion" {
				n, e := rand.Int(rand.Reader, big.NewInt(300001))
				if e != nil {
					return nil, e
				}
				height := 0.85 + float64(n.Int64())/1000000
				if _, e = tx.ExecContext(r.Context, repository.SetOutfitHeight, height, uid); e != nil {
					return nil, e
				}
			}
		}
		if e = tx.Commit(); e != nil {
			return nil, e
		}
	}
	b, err := a.Buffs(r, uid)
	if err != nil {
		return nil, err
	}
	items, err := a.Consumables(r, uid, definitionID)
	if err != nil {
		return nil, err
	}
	b["consume_item"] = status
	b["get_consumables"] = items
	b["shared_results"] = []Object{}
	height := 1.0
	if err = a.Store.QueryRowContext(r.Context, repository.ReadOutfitHeight, uid).Scan(&height); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	b["new_height"] = height
	return b, nil
}
