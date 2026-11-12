package service

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
	"strings"
)

func dyeSlot(value any) (string, []string) {
	raw, _ := value.([]any)
	parts := []string{}
	colors := []string{}
	for _, v := range raw {
		item, ok := v.(string)
		if !ok || item == "" {
			continue
		}
		parts = append(parts, item)
		color := strings.SplitN(item, "_", 2)[0]
		switch color {
		case "white", "black", "red", "blue", "cyan", "yellow", "green", "magenta":
			colors = append(colors, "color_"+color)
		}
	}
	if len(parts) == 0 {
		return "none", colors
	}
	return strings.Join(parts, "_"), colors
}
func (a *Service) PurchaseDye(r Request, m Object) (Object, error) {
	uid, err := a.magicUser(r, m, true)
	if err != nil {
		return nil, err
	}
	name := text(m, "unlock_name")
	if name == "" {
		return nil, fail(400, "Missing unlock_name")
	}
	primary, colors1 := dyeSlot(m["primary_slot"])
	secondary, colors2 := dyeSlot(m["secondary_slot"])
	dye := "(" + primary + "," + secondary + ")"
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	stamp := now()
	if _, err = tx.ExecContext(r.Context, repository.InsertDyeUnlock, uid, name, stamp); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(r.Context, repository.DeleteItemDye, uid, name); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(r.Context, repository.SetItemDye, uid, name, dye, stamp, stamp); err != nil {
		return nil, err
	}
	for _, color := range append(colors1, colors2...) {
		if _, err = tx.ExecContext(r.Context, repository.EnsureColorDye, uid, color); err != nil {
			return nil, err
		}
	}
	var kind string
	var ack bool
	var created int64
	if err = tx.QueryRowContext(r.Context, repository.ReadDyeUnlock, uid, name).Scan(&kind, &ack, &created); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(r.Context, repository.EnsureWingCurrency, uid); err != nil {
		return nil, err
	}
	currency, err := currencySnapshot(r.Context, tx, uid)
	if err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(r.Context, repository.ReadColorDyes, uid)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var color string
		var value int64
		if err = rows.Scan(&color, &value); err != nil {
			rows.Close()
			return nil, err
		}
		currency[color] = value
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for _, color := range []string{"white", "black", "red", "blue", "cyan", "yellow", "green", "magenta"} {
		key := "color_" + color
		if _, ok := currency[key]; !ok {
			currency[key] = int64(0)
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	item := Object{"upgrades": Object{"types": []any{}, "dyes": []any{}, "count": 0}, "type": kind, "name": name, "created_at": created, "ack": ack}
	return Object{"purchase_result": []any{"success", "undefined", []Object{item}}, "currency": currency}, nil
}
