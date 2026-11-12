package service

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
	"strings"
)

var forgeTargets = map[string][2]string{"wax": {"wax", "candles"}, "heart_wax": {"heart_wax", "heart"}, "season_wax": {"season_wax", "season_candle"}, "season_wax_14": {"season_wax_14", "season_heart"}, "prestige_wax": {"prestige_wax", "prestige"}, "prestige_vip": {"prestige_vip", "prestige"}}
var forgeSourceHints = map[string]string{"candles": "wax", "candle": "wax", "heart": "heart_wax", "season_candle": "season_wax", "season_candles": "season_wax", "season_heart": "season_wax_14", "prestige": "prestige_wax"}

func forgeSource(m Object) string {
	for _, key := range []string{"forge_currency", "source_type", "source", "forge_type", "forge_source", "wax_type", "wax", "currency", "currency_type", "type", "kind", "item_type", "resource", "target"} {
		value := strings.ToLower(text(m, key))
		if _, ok := forgeTargets[value]; ok {
			return value
		}
		if source, ok := forgeSourceHints[value]; ok {
			return source
		}
	}
	return "wax"
}
func (a *Service) BuyCandleWax(r Request, m Object) (Object, error) {
	failed := func(message string) Object {
		return Object{"result": "error", "message": message, "currency": Object{}, "update_forge_rates": []Object{}}
	}
	uid, err := a.magicUser(r, m, true)
	if err != nil {
		return failed("Invalid session or user not found"), nil
	}
	cost := number(m, "cost", 0)
	if cost <= 0 {
		cost = 1
	}
	want := number(m, "count", 1)
	if want <= 0 {
		want = 1
	}
	if want > 1000000 || cost > 1000000000 {
		return failed("Invalid forge amount"), nil
	}
	source := forgeSource(m)
	fields := forgeTargets[source]
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(r.Context, repository.EnsureWingCurrency, uid); err != nil {
		return nil, err
	}
	var have int64
	if err = tx.QueryRowContext(r.Context, repository.CurrencyRead(fields[0]), uid).Scan(&have); err != nil {
		return nil, err
	}
	if have <= 0 {
		return failed("wax不足"), nil
	}
	afford := have / cost
	count := want
	charge := count * cost
	if afford < count {
		count = afford
		charge = count * cost
	}
	if count == 0 {
		count = 1
		charge = have
	}
	if count > 1000000 {
		return failed("Invalid count"), nil
	}
	paid, err := tx.SpendCurrency(r.Context, uid, fields[0], charge)
	if err != nil {
		return nil, err
	}
	if !paid {
		return failed("wax不足"), nil
	}
	if err = tx.GrantCurrency(r.Context, uid, fields[1], count); err != nil {
		return nil, err
	}
	rates := []Object{}
	if source == "wax" {
		if _, err = tx.ExecContext(r.Context, repository.EnsureWaxForgeRate, uid); err != nil {
			return nil, err
		}
		var tier, level, tierIndex, tierCount int64
		var tierCompletion, levelCompletion float64
		var ack, enabled bool
		if err = tx.QueryRowContext(r.Context, repository.ReadWaxForgeRate, uid).Scan(&tier, &level, &tierCompletion, &levelCompletion, &tierIndex, &tierCount, &ack, &enabled); err != nil {
			return nil, err
		}
		rates = append(rates, Object{"source_type": "wax", "current_tier": tier, "current_level": level, "tier_completion": tierCompletion, "level_completion": levelCompletion, "level_tier_index": tierIndex, "level_tier_count": tierCount, "reset_acknowledged": ack, "enabled": enabled})
	}
	currency, err := currencySnapshot(r.Context, tx, uid)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return Object{"result": "ok", "currency": currency, "update_forge_rates": rates}, nil
}
