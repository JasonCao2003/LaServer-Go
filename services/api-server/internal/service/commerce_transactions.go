package service

import (
	"database/sql"
	"errors"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
	"strings"
)

func commerceFailure(status int, result string) error {
	return &Error{Status: status, Body: Object{"errorcode": 1, "result": result}}
}
func firstText(m Object, keys ...string) string {
	for _, key := range keys {
		if value := text(m, key); value != "" {
			return value
		}
	}
	return ""
}
func commerceCurrencyField(raw string) string {
	switch raw {
	case "hearts":
		return "heart"
	case "candle":
		return "candles"
	case "season_candles":
		return "season_candle"
	case "season_hearts":
		return "season_heart"
	case "pass_token":
		return "season_pass_token"
	}
	return raw
}
func (a *Service) CommerceReceipt(r Request, m Object) (Object, error) {
	uid, err := a.magicUser(r, m, true)
	if err != nil {
		return nil, commerceFailure(401, "invalid_session")
	}
	if strings.ToLower(text(m, "platform")) != "fake" {
		return nil, commerceFailure(501, "receipt_verification_unavailable")
	}
	if !a.cfg.AllowFakeCommerce {
		return nil, commerceFailure(403, "fake_commerce_disabled")
	}
	if target := firstText(m, "target_uid", "target_user"); target != "" && target != uid {
		return nil, commerceFailure(400, "gift_not_supported")
	}
	pid := firstText(m, "target_pid", "target_product_id", "product_id", "productId")
	// A fake receipt is constant per product in the old client, so it cannot identify a purchase.
	// Require an explicit unique transaction ID to make grants replay-safe.
	tid := firstText(m, "transaction_id", "transactionId")
	if raw, ok := m["transactions"].([]any); ok && len(raw) == 1 {
		if tx, ok := raw[0].(map[string]any); ok {
			pid = firstText(tx, "product_id", "productId", "id")
			tid = firstText(tx, "transaction_id", "transactionId", "tid")
		}
	}
	if tx, ok := m["transactions"].(map[string]any); ok {
		pid = firstText(tx, "product_id", "productId", "id")
		tid = firstText(tx, "transaction_id", "transactionId", "tid")
	}
	if pid == "" || tid == "" {
		return nil, commerceFailure(400, "unique_transaction_id_required")
	}
	var product Object
	for _, p := range a.statusConfig("iaplist.json", "iaplist") {
		if text(p, "id") == pid {
			product = p
			break
		}
	}
	if product == nil {
		return nil, commerceFailure(404, "unknown_product")
	}
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var priorUser, priorProduct, priorStatus string
	lookup := tx.QueryRowContext(r.Context, repository.ReadCommerceOrder, tid).Scan(&priorUser, &priorProduct, &priorStatus)
	if lookup != nil && !errors.Is(lookup, sql.ErrNoRows) {
		return nil, lookup
	}
	if lookup == nil && (priorUser != uid || priorProduct != pid) {
		return nil, commerceFailure(409, "transaction_conflict")
	}
	granted := []Object{}
	duplicate := lookup == nil
	if !duplicate {
		if _, err = tx.ExecContext(r.Context, repository.EnsureWingCurrency, uid); err != nil {
			return nil, err
		}
		for _, pair := range [][2]string{{"currencyType", "currencyCount"}, {"currencyType2", "currencyCount2"}} {
			raw := text(product, pair[0])
			count := defInt(product[pair[1]])
			if raw == "" || count <= 0 {
				continue
			}
			field := commerceCurrencyField(raw)
			if !currencyField(field) {
				return nil, commerceFailure(503, "unsupported_product_currency")
			}
			if err = tx.GrantCurrency(r.Context, uid, field, count); err != nil {
				return nil, err
			}
			var total int64
			if err = tx.QueryRowContext(r.Context, repository.CurrencyRead(field), uid).Scan(&total); err != nil {
				return nil, err
			}
			granted = append(granted, Object{"currency": raw, "count": count, "total": total})
		}
		if strings.HasPrefix(strings.ToUpper(pid), "SPASS") {
			if _, err = tx.ExecContext(r.Context, repository.CurrencyUpdate("vip"), 1, uid); err != nil {
				return nil, err
			}
			granted = append(granted, Object{"currency": "vip", "count": 1, "total": 1})
		}
		if unlocks, ok := product["unlocks"].([]any); ok {
			for _, v := range unlocks {
				name, ok := v.(string)
				if !ok || name == "" {
					continue
				}
				if strings.HasPrefix(name, "{{") {
					granted = append(granted, Object{"unlock": name})
					continue
				}
				res, e := tx.ExecContext(r.Context, repository.InsertOwnedUnlock, uid, name, "spiritshop", false, now())
				if e != nil {
					return nil, e
				}
				n, e := res.RowsAffected()
				if e != nil {
					return nil, e
				}
				item := Object{"unlock": name}
				if n == 0 {
					item["already_owned"] = true
				}
				granted = append(granted, item)
			}
		}
		if _, err = tx.ExecContext(r.Context, repository.InsertCommerceOrder, uid, pid, tid, now()); err != nil {
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
	item := Object{"product_id": pid, "transaction_id": tid, "result": "ok", "status": "simulated", "duplicate": duplicate, "granted": granted}
	return Object{"errorcode": 0, "result": "ok", "status": "simulated", "transactions": []Object{item}, "verified": []Object{}, "granted": granted, "currency": currency}, nil
}
func (a *Service) CommerceFulfill(r Request, m Object) (Object, error) {
	uid, err := a.magicUser(r, m, true)
	if err != nil {
		return nil, commerceFailure(401, "invalid_session")
	}
	tid := firstText(m, "target_tid", "transaction_id", "transactionId", "tid")
	if tid == "" {
		return nil, commerceFailure(400, "missing_transaction")
	}
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var owner, pid, status string
	err = tx.QueryRowContext(r.Context, repository.ReadCommerceOrder, tid).Scan(&owner, &pid, &status)
	if errors.Is(err, sql.ErrNoRows) || err == nil && owner != uid {
		return nil, commerceFailure(404, "unknown_transaction")
	}
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(r.Context, repository.FulfillCommerceOrder, uid, tid); err != nil {
		return nil, err
	}
	currency, err := currencySnapshot(r.Context, tx, uid)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return Object{"errorcode": 0, "result": "ok", "status": "fulfilled", "transaction_id": tid, "product_id": pid, "fulfilled": true, "currency": currency}, nil
}
