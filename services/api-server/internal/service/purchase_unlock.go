package service

import (
	"database/sql"
	"errors"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

func (a *Service) PurchaseUnlock(r Request, m Object) (Object, error) {
	missing := []string{}
	for _, key := range []string{"user", "session", "cost"} {
		if _, ok := m[key]; !ok {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return nil, &Error{Status: 400, Body: Object{"result": "missing_parameters", "missing": missing}}
	}
	uid, err := a.magicUser(r, m, true)
	if err != nil {
		return nil, &Error{Status: 401, Body: Object{"result": "unauthorized"}}
	}
	name, kind := text(m, "name"), text(m, "type")
	ack, _ := m["ack"].(bool)
	field := "candles"
	var cost int64
	if name != "" {
		if _, ok := m["ack"].(bool); !ok || kind == "" {
			return nil, &Error{Status: 400, Body: Object{"result": "invalid_request"}}
		}
	}
	if name != "" && kind != "" {
		node := a.spiritNode(name)
		if node == nil {
			return nil, &Error{Status: 400, Body: Object{"result": "invalid_request"}}
		}
		field = text(node, "ctyp")
		if field == "" {
			field = "candles"
		}
		cost = defInt(node["cst"])
		if cost == 0 && text(node, "typ") != "quest" {
			cost = 1
		}
	}
	if name == "" {
		id := number(m, "unlock_id", 0)
		if id == 0 {
			return nil, &Error{Status: 400, Body: Object{"result": "invalid_request"}}
		}
		if err = a.seedShopDefaults(r); err != nil {
			return nil, err
		}
		err = a.store.QueryRowContext(r.Context, repository.ReadShopPurchaseItem, id).Scan(&name, &field, &cost)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &Error{Status: 404, Body: Object{"result": "item_not_found"}}
		}
		if err != nil {
			return nil, err
		}
		kind = "shop"
	}
	if !currencyField(field) || cost < 0 {
		return nil, &Error{Status: 400, Body: Object{"result": "invalid_price"}}
	}
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(r.Context, repository.EnsureWingCurrency, uid); err != nil {
		return nil, err
	}
	var existing bool
	err = tx.QueryRowContext(r.Context, repository.ReadOwnedUnlock, uid, name).Scan(&existing)
	if err == nil {
		return Object{"result": "already"}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	paid, err := tx.SpendCurrency(r.Context, uid, field, cost)
	if err != nil {
		return nil, err
	}
	if !paid {
		return nil, &Error{Status: 400, Body: Object{"result": "insufficient_currency"}}
	}
	stamp := now()
	updates := []Object{}
	insert := func(item, typ string, acknowledged bool) error {
		res, e := tx.ExecContext(r.Context, repository.InsertOwnedUnlock, uid, item, typ, acknowledged, stamp)
		if e != nil {
			return e
		}
		n, e := res.RowsAffected()
		if e != nil {
			return e
		}
		if n > 0 {
			updates = append(updates, Object{"name": item, "created_at": stamp, "ack": acknowledged, "type": typ})
		}
		return nil
	}
	if err = insert(name, kind, ack); err != nil {
		return nil, err
	}
	for _, candidate := range a.emoteFamilyLevels(name) {
		if candidate != name {
			if err = insert(candidate, kind, ack); err != nil {
				return nil, err
			}
		}
	}
	if name == "cape" {
		if err = insert("GainedFlame", "level", false); err != nil {
			return nil, err
		}
	}
	if name == "SkyHubFirstArrival" {
		if err = insert("HubStatueForm", "level", false); err != nil {
			return nil, err
		}
	}
	currency, err := currencySnapshot(r.Context, tx, uid)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return Object{"result": "ok", "currency": currency, "update_unlocks": updates}, nil
}
