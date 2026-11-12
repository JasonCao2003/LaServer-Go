package service

import (
	"database/sql"
	"errors"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
	"strings"
)

func spiritUnlockID(name string) int64 {
	hash := uint32(2166136261)
	for _, b := range []byte(name) {
		hash ^= uint32(b)
		hash *= 16777619
	}
	return int64(hash)
}

func spiritFailure(reason string) Object {
	return Object{"result": "error", "error": reason, "currency": Object{}, "update_unlocks": []Object{}, "get_lootboxes": []Object{}}
}
func (a *Service) PurchaseSpiritShopItem(r Request, m Object) (Object, error) {
	uid, err := a.magicUser(r, m, true)
	if err != nil {
		return spiritFailure("Invalid session or user not found"), nil
	}
	name := text(m, "item_name")
	if name == "" {
		return spiritFailure("Missing required fields"), nil
	}
	node := a.spiritNode(name)
	if node == nil {
		return spiritFailure("Unknown spirit shop item"), nil
	}
	field := text(node, "ctyp")
	if field == "" {
		field = "candles"
	}
	cost := defInt(node["cst"])
	if cost < 0 {
		return spiritFailure("Invalid price"), nil
	}
	if cost == 0 && text(node, "typ") != "quest" && !strings.HasPrefix(text(node, "ta"), "zero_trust") && !strings.HasPrefix(text(node, "spirit"), "questap") {
		cost = 1
	}
	if field == "season_heart" {
		field = "heart"
	}
	if !currencyField(field) {
		if cost > 0 {
			return spiritFailure("Unsupported currency"), nil
		}
		field = "candles"
	}
	if id := number(m, "unlock_id", 0); id != 0 && id == spiritUnlockID(name) {
		var catalogField string
		var catalogCost int64
		lookup := a.store.QueryRowContext(r.Context, repository.ReadShopPrice, id).Scan(&catalogField, &catalogCost)
		if lookup == nil && currencyField(catalogField) && catalogCost >= 0 {
			field, cost = catalogField, catalogCost
		} else if lookup != nil && !errors.Is(lookup, sql.ErrNoRows) {
			return nil, lookup
		}
	}
	box := text(node, "typ") == "lootbox" || strings.HasPrefix(name, "lootbox")
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(r.Context, repository.EnsureWingCurrency, uid); err != nil {
		return nil, err
	}
	var ack bool
	ownedErr := tx.QueryRowContext(r.Context, repository.ReadOwnedUnlock, uid, name).Scan(&ack)
	if ownedErr != nil && !errors.Is(ownedErr, sql.ErrNoRows) {
		return nil, ownedErr
	}
	if ownedErr == nil && !box {
		currency, err := currencySnapshot(r.Context, tx, uid)
		if err != nil {
			return nil, err
		}
		if err = tx.Commit(); err != nil {
			return nil, err
		}
		update := []Object{}
		if !ack {
			update = append(update, Object{"type": "spiritshop", "name": name, "created_at": 0, "ack": false})
		}
		loot, err := a.openLootboxes(r, uid)
		return Object{"result": "ok", "currency": currency, "update_unlocks": update, "get_lootboxes": loot}, err
	}
	paid, err := tx.SpendCurrency(r.Context, uid, field, cost)
	if err != nil {
		return nil, err
	}
	if !paid {
		currency, e := currencySnapshot(r.Context, tx, uid)
		if e != nil {
			return nil, e
		}
		var available int64
		if e = tx.QueryRowContext(r.Context, repository.CurrencyRead(field), uid).Scan(&available); e != nil {
			return nil, e
		}
		_ = tx.Rollback()
		loot, e := a.openLootboxes(r, uid)
		if e != nil {
			return nil, e
		}
		return Object{"result": "error", "code": "INSUFFICIENT_CURRENCY", "error": "Insufficient " + field, "currency_type": field, "required": cost, "available": available, "currency": currency, "update_unlocks": []Object{}, "get_lootboxes": loot}, nil
	}
	updates := []Object{}
	if ownedErr != nil {
		if _, err = tx.ExecContext(r.Context, repository.InsertOwnedUnlock, uid, name, "spiritshop", false, now()); err != nil {
			return nil, err
		}
		updates = append(updates, Object{"type": "spiritshop", "name": name, "created_at": 0, "ack": false})
	} else if !ack {
		updates = append(updates, Object{"type": "spiritshop", "name": name, "created_at": 0, "ack": false})
	}
	if box {
		res, e := tx.ExecContext(r.Context, repository.IncrementOpenLootbox, uid, name)
		if e != nil {
			return nil, e
		}
		n, e := res.RowsAffected()
		if e != nil {
			return nil, e
		}
		if n == 0 {
			if _, e = tx.ExecContext(r.Context, repository.UpsertLootbox, uid, name, "lootbox", now()); e != nil {
				return nil, e
			}
		}
	} else {
		if ownedErr != nil {
			for _, level := range a.emoteFamilyLevels(name) {
				if level == name {
					continue
				}
				res, e := tx.ExecContext(r.Context, repository.InsertOwnedUnlock, uid, level, "spiritshop", false, now())
				if e != nil {
					return nil, e
				}
				n, e := res.RowsAffected()
				if e != nil {
					return nil, e
				}
				if n > 0 {
					updates = append(updates, Object{"type": "spiritshop", "name": level, "created_at": 0, "ack": false})
				}
			}
		}
		if text(node, "typ") == "heart" {
			if err = tx.GrantCurrency(r.Context, uid, "heart", 1); err != nil {
				return nil, err
			}
		} else if text(node, "typ") == "season_heart" {
			if err = tx.GrantCurrency(r.Context, uid, "season_heart", 1); err != nil {
				return nil, err
			}
		}
	}
	currency, err := currencySnapshot(r.Context, tx, uid)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	loot, err := a.openLootboxes(r, uid)
	return Object{"result": "ok", "currency": currency, "update_unlocks": updates, "get_lootboxes": loot}, err
}
