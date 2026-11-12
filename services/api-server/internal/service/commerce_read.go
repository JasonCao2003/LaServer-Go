package service

import (
	"database/sql"
	"errors"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

func (a *Service) seedShopDefaults(r Request) error {
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var count int64
	if err = tx.QueryRowContext(r.Context, repository.CountShopItems).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return tx.Commit()
	}
	var items []Object
	if err = legacyResponse("shop_defaults", &items); err != nil {
		return err
	}
	for _, v := range items {
		if _, err = tx.ExecContext(r.Context, repository.InsertShopItem, text(v, "name"), defInt(v["unlockid"]), defInt(v["unlock_cost"]), text(v, "currency_type"), v["listed"], defInt(v["unlock_dep"]), defInt(v["visual_dep"]), defInt(v["x"]), defInt(v["y"])); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (a *Service) GetShop(r Request, m Object) (Object, error) {
	if _, err := a.magicUser(r, m, true); err != nil {
		return nil, err
	}
	if err := a.seedShopDefaults(r); err != nil {
		return nil, err
	}
	rows, err := a.store.QueryContext(r.Context, repository.ReadShopItems)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Object{}
	for rows.Next() {
		var name, currency string
		var id, cost, unlockDep, visualDep, x, y int64
		var listed bool
		if err = rows.Scan(&name, &id, &cost, &currency, &listed, &unlockDep, &visualDep, &x, &y); err != nil {
			return nil, err
		}
		out = append(out, Object{"name": name, "unlockid": id, "unlock_cost": cost, "currency_type": currency, "listed": listed, "unlock_dep": unlockDep, "visual_dep": visualDep, "x": x, "y": y})
	}
	return Object{"get_shop": []any{out}}, rows.Err()
}
func (a *Service) openLootboxes(r Request, uid string) ([]Object, error) {
	rows, err := a.store.QueryContext(r.Context, repository.ReadOpenLootboxes, uid, now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Object{}
	for rows.Next() {
		var id, kind string
		var quantity, acquired int64
		if err = rows.Scan(&id, &kind, &quantity, &acquired); err != nil {
			return nil, err
		}
		if kind == "" {
			kind = "lootbox"
		}
		out = append(out, Object{"lootbox_id": id, "type": kind, "quantity": quantity, "acquired_at": acquired})
	}
	return out, rows.Err()
}
func (a *Service) GetLootboxes(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	if uid != "" && text(m, "session") != "" {
		u, err := a.findUser(r.Context, "id", uid)
		if errors.Is(err, sql.ErrNoRows) || err == nil && stringValue(u.Session) != text(m, "session") {
			return nil, fail(401, "Invalid session or user not found")
		}
		if err != nil {
			return nil, err
		}
	}
	items, err := a.openLootboxes(r, uid)
	return Object{"get_lootboxes": items}, err
}
