package service

import "github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"

func genericFailure(message string) Object {
	return Object{"generic_shop_purchase": "failed", "error": message, "generic_shop_item_tracking_info": []Object{}, "get_consumables": []Object{}, "currency": Object{}}
}
func (a *Service) PurchaseGenericShopItem(r Request, m Object) (Object, error) {
	uid, err := a.magicUser(r, m, true)
	if err != nil {
		return genericFailure("Invalid session or user not found"), nil
	}
	d := a.genericShopDef(m)
	if d == nil {
		return genericFailure("Unknown shop item"), nil
	}
	if text(d, "item_type") != "consumable" {
		return genericFailure("Unsupported shop item type"), nil
	}
	name := text(d, "item_name")
	if name == "" {
		name = text(m, "item_name")
	}
	var cid int64
	for _, c := range a.versionMagicDefinitions("get_consumable_defs", "account__consumable__get_consumable_defs.json", r.UserAgent) {
		if text(c, "name") == name || defInt(c["id"]) == number(m, "consumable_id", 0) && number(m, "consumable_id", 0) != 0 {
			cid = defInt(c["id"])
			break
		}
	}
	if cid == 0 {
		return genericFailure("Unknown consumable"), nil
	}
	field := text(d, "currency_type")
	if field == "" {
		field = "candles"
	}
	cost := defInt(d["cost"])
	if cost < 0 || !currencyField(field) {
		return genericFailure("Invalid shop price"), nil
	}
	count := defInt(d["item_count"])
	if count <= 0 {
		count = 1
	}
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(r.Context, repository.EnsureWingCurrency, uid); err != nil {
		return nil, err
	}
	ok, err := tx.SpendCurrency(r.Context, uid, field, cost)
	if err != nil {
		return nil, err
	}
	if !ok {
		currency, e := currencySnapshot(r.Context, tx, uid)
		if e != nil {
			return nil, e
		}
		var available int64
		if e = tx.QueryRowContext(r.Context, repository.CurrencyRead(field), uid).Scan(&available); e != nil {
			return nil, e
		}
		_ = tx.Rollback()
		items, e := a.consumables(r, uid, 0)
		if e != nil {
			return nil, e
		}
		return Object{"generic_shop_purchase": "failed", "code": "INSUFFICIENT_CURRENCY", "currency_type": field, "required": cost, "available": available, "generic_shop_item_tracking_info": []Object{}, "get_consumables": items, "currency": currency}, nil
	}
	if _, err = tx.ExecContext(r.Context, repository.GiveConsumable, uid, cid, count); err != nil {
		return nil, err
	}
	currency, err := currencySnapshot(r.Context, tx, uid)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	items, err := a.consumables(r, uid, 0)
	if err != nil {
		return nil, err
	}
	id := defInt(d["id"])
	tracking := []Object{}
	if id != 0 {
		tracking = append(tracking, Object{"count": 1, "id": id, "ttl": 86400 - now()%86400})
	}
	return Object{"generic_shop_purchase": "success", "generic_shop_item_tracking_info": tracking, "get_consumables": items, "currency": currency}, nil
}
