package service

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

func (a *Service) SeasonQuests(r Request, m Object) (Object, error) {
	uid, err := a.questUser(r, m)
	if err != nil {
		return nil, err
	}
	day := localDay()
	season, normal := a.dailyQuestPools()
	pool := append(season, normal...)
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	keep := map[string]bool{}
	for _, q := range pool {
		id := text(q, "daily_quest_def_id")
		if keep[id] {
			continue
		}
		keep[id] = true
		st := text(q, "stat_type")
		if st == "" {
			st = id + "_count"
		}
		if _, err = tx.ExecContext(r.Context, `INSERT INTO season_quest(user_id,date,quest_id,stat_type) VALUES(?,?,?,?) ON CONFLICT(user_id,date,quest_id) DO UPDATE SET stat_type=excluded.stat_type`, uid, day, id, st); err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(r.Context, `INSERT INTO season_quest_reward(user_id,date,quest_id) VALUES(?,?,?) ON CONFLICT DO NOTHING`, uid, day, id); err != nil {
			return nil, err
		}
	}
	quests := []Object{}
	rewards := []Object{}
	rows, err := tx.QueryContext(r.Context, `SELECT quest_id,stat_type,start_value,activated FROM season_quest WHERE user_id=? AND date=? ORDER BY rowid`, uid, day)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id, st string
		var start float64
		var active bool
		if err = rows.Scan(&id, &st, &start, &active); err != nil {
			rows.Close()
			return nil, err
		}
		if keep[id] {
			quests = append(quests, Object{"date": day, "daily_quest_def_id": id, "stat_type": st, "start_value": start, "activated": active})
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = tx.QueryContext(r.Context, `SELECT quest_id,rewarded FROM season_quest_reward WHERE user_id=? AND date=? ORDER BY rowid`, uid, day)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id string
		var rewarded bool
		if err = rows.Scan(&id, &rewarded); err != nil {
			rows.Close()
			return nil, err
		}
		if keep[id] {
			rewards = append(rewards, Object{"date": day, "daily_quest_def_id": id, "rewarded": rewarded})
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return Object{"season_quests": quests, "has_season_quests_rewards": rewards}, nil
}
func (a *Service) ActivateSeasonQuest(r Request, m Object) (Object, error) {
	uid, err := a.questUser(r, m)
	if err != nil {
		return nil, err
	}
	id := questID(m)
	if id == "" {
		return nil, fail(400, "Missing quest_id")
	}
	season, normal := a.dailyQuestPools()
	statType := ""
	for _, q := range append(season, normal...) {
		if text(q, "daily_quest_def_id") == id {
			statType = text(q, "stat_type")
			if statType == "" {
				statType = id + "_count"
			}
			break
		}
	}
	if statType == "" {
		return nil, fail(404, "Quest not scheduled")
	}
	var baseline float64
	err = a.store.QueryRowContext(r.Context, `SELECT value FROM achievement_stat WHERE user_id=? AND stat_type=?`, uid, statType).Scan(&baseline)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	_, err = a.store.ExecContext(r.Context, `INSERT INTO season_quest(user_id,date,quest_id,stat_type,start_value,activated) VALUES(?,?,?,?,?,1) ON CONFLICT(user_id,date,quest_id) DO UPDATE SET start_value=excluded.start_value,activated=1 WHERE activated=0`, uid, localDay(), id, statType, baseline)
	if err != nil {
		return nil, err
	}
	return Object{"season_quest_activated": []Object{{"activated": true, "quest_id": id, "start_value": baseline}}}, nil
}
func (a *Service) ClaimSeasonQuestReward(r Request, m Object) (Object, error) {
	uid, err := a.questUser(r, m)
	if err != nil {
		return nil, err
	}
	id := questID(m)
	if id == "" {
		return nil, fail(400, "Missing quest_id")
	}
	day := localDay()
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var rewarded bool
	err = tx.QueryRowContext(r.Context, `SELECT rewarded FROM season_quest_reward WHERE user_id=? AND date=? AND quest_id=?`, uid, day, id).Scan(&rewarded)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fail(404, "Quest not found")
	}
	if err != nil {
		return nil, err
	}
	if !rewarded {
		eligible, e := a.questEligible(r, tx, uid, day, id)
		if e != nil {
			return nil, e
		}
		if !eligible {
			return nil, fail(409, "Quest not completed")
		}
	}
	result, err := tx.ExecContext(r.Context, `UPDATE season_quest_reward SET rewarded=1 WHERE user_id=? AND date=? AND quest_id=? AND rewarded=0`, uid, day, id)
	if err != nil {
		return nil, err
	}
	first, err := insertionCount(result)
	if err != nil {
		return nil, err
	}
	if first {
		field := "candles"
		inSeason := defInt(m["in_season"]) != 0
		if inSeason {
			field = "season_candle"
		}
		if err = tx.GrantCurrency(r.Context, uid, field, a.questReward(inSeason)); err != nil {
			return nil, err
		}
	}
	currency, err := currencySnapshot(r.Context, tx, uid)
	if err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(r.Context, `SELECT quest_id,rewarded FROM season_quest_reward WHERE user_id=? AND date=? ORDER BY rowid`, uid, day)
	if err != nil {
		return nil, err
	}
	rewards := []Object{}
	for rows.Next() {
		var id string
		var rewarded bool
		if err = rows.Scan(&id, &rewarded); err != nil {
			rows.Close()
			return nil, err
		}
		rewards = append(rewards, Object{"daily_quest_def_id": id, "date": day, "rewarded": rewarded})
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return Object{"currency": currency, "has_season_quests_rewards": rewards, "season_quest_claim_result": "ok"}, nil
}

func (a *Service) questEligible(r Request, tx *repository.Tx, uid, day, id string) (bool, error) {
	var count int
	err := tx.QueryRowContext(r.Context, `SELECT COUNT(*) FROM player_quest WHERE user_id=? AND quest_name=? AND status IN ('succeeded','collected') AND expiry_time>=?`, uid, id, now()).Scan(&count)
	if err != nil || count > 0 {
		return count > 0, err
	}
	var statType string
	var start, current float64
	err = tx.QueryRowContext(r.Context, `SELECT stat_type,start_value FROM season_quest WHERE user_id=? AND date=? AND quest_id=? AND activated=1`, uid, day, id).Scan(&statType, &start)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	err = tx.QueryRowContext(r.Context, `SELECT value FROM achievement_stat WHERE user_id=? AND stat_type=?`, uid, statType).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	needed := float64(1)
	if values, ok := questDefinition(a, id)["completion_stats_values"].([]any); ok && len(values) > 0 {
		if n := defInt(values[0]); n > 0 {
			needed = float64(n)
		}
	}
	return current-start >= needed, nil
}
func (a *Service) SeasonStart(r Request, m Object) (Object, error) {
	if a.seasonDisabled("season_start") {
		return Object{}, nil
	}
	uid, err := a.questUser(r, m)
	if err != nil {
		return nil, err
	}
	n := a.seasonNumber()
	res, err := a.store.ExecContext(r.Context, `INSERT INTO season_marker(user_id,name,created_at) VALUES(?,?,0) ON CONFLICT DO NOTHING`, uid, fmt.Sprintf("season_start_%d", n))
	if err != nil {
		return nil, err
	}
	first, err := insertionCount(res)
	if err != nil {
		return nil, err
	}
	result := []string{}
	if first {
		result = append(result, fmt.Sprintf("season_%d", n))
	}
	return Object{"result": result}, nil
}
func (a *Service) SeasonFinalize(r Request, m Object) (Object, error) {
	if a.seasonDisabled("season_finalize") {
		return Object{}, nil
	}
	uid, err := a.questUser(r, m)
	if err != nil {
		return nil, err
	}
	sc := nestedObject(a.gameControl(), "season")
	n := a.seasonNumber()
	start := number(sc, "start_time", 0)
	duration := number(sc, "duration_days", 70)
	ended := sc["force_end"] == true || start > 0 && now() >= start+duration*86400
	name := text(sc, "season_unlock_name")
	if name == "" {
		name = fmt.Sprintf("season_finish_%d", n)
	}
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	first := false
	if !ended {
		_, err = tx.ExecContext(r.Context, `DELETE FROM season_marker WHERE user_id=? AND name=?`, uid, name)
		if err != nil {
			return nil, err
		}
	} else {
		res, e := tx.ExecContext(r.Context, `INSERT INTO season_marker(user_id,name,created_at) VALUES(?,?,0) ON CONFLICT DO NOTHING`, uid, name)
		if e != nil {
			return nil, e
		}
		first, e = insertionCount(res)
		if e != nil {
			return nil, e
		}
		if first {
			ratio := number(sc, "convert_ratio", 1)
			if ratio < 0 {
				ratio = 0
			}
			_, err = tx.ExecContext(r.Context, `UPDATE user_currency SET candles=COALESCE(candles,0)+COALESCE(season_candle,0)*?,season_candle=0,season_heart=0,season_wax=0 WHERE user_id=?`, ratio, uid)
			if err != nil {
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
	rates, err := a.ForgeRates(r, Object{"user": uid})
	if err != nil {
		return nil, err
	}
	unlocks := []Object{}
	result := []string{}
	if first {
		result = append(result, fmt.Sprintf("season_%d", n))
		unlocks = append(unlocks, Object{"type": "season", "name": name, "created_at": 0, "ack": false})
	}
	return Object{"result": result, "currency": currency, "update_forge_rates": rates["forge_rates"], "update_unlocks": unlocks}, nil
}
func (a *Service) WorldQuests(r Request, m Object) (Object, error) {
	uid, err := a.questUser(r, m)
	if err != nil {
		return nil, err
	}
	rows, err := a.store.QueryContext(r.Context, `SELECT name,cooldown_over_time,ever_completed FROM world_quest WHERE user_id=? ORDER BY name`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Object{}
	for rows.Next() {
		var name string
		var cooldown int64
		var completed bool
		if err = rows.Scan(&name, &cooldown, &completed); err != nil {
			return nil, err
		}
		out = append(out, Object{"world_quest_def_id": name, "cooldown_over_time": cooldown, "ever_completed": completed})
	}
	return Object{"set_world_quests": out}, rows.Err()
}
func (a *Service) ClaimWorldQuestReward(r Request, m Object) (Object, error) {
	uid, err := a.questUser(r, m)
	if err != nil {
		return nil, err
	}
	name := text(m, "name")
	if name == "" {
		return nil, fail(400, "Missing name")
	}
	pct := 0.0
	switch v := m["bonus_percent"].(type) {
	case float64:
		pct = v
	case json.Number:
		pct, _ = v.Float64()
	case string:
		pct, _ = strconv.ParseFloat(v, 64)
	}
	if math.IsNaN(pct) || math.IsInf(pct, 0) {
		pct = 0
	}
	amount := int64(15 + math.RoundToEven(pct*20))
	if amount < 0 {
		amount = 0
	}
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	cooldown := now() + 86400
	res, err := tx.ExecContext(r.Context, `INSERT INTO world_quest(user_id,name,cooldown_over_time) VALUES(?,?,?) ON CONFLICT(user_id,name) DO UPDATE SET cooldown_over_time=excluded.cooldown_over_time WHERE world_quest.cooldown_over_time<=?`, uid, name, cooldown, now())
	if err != nil {
		return nil, err
	}
	first, err := insertionCount(res)
	if err != nil {
		return nil, err
	}
	if first {
		if err = tx.GrantCurrency(r.Context, uid, "candles", amount); err != nil {
			return nil, err
		}
		if err = tx.GrantCurrency(r.Context, uid, "wax", amount); err != nil {
			return nil, err
		}
	} else {
		amount = 0
		err = tx.QueryRowContext(r.Context, `SELECT cooldown_over_time FROM world_quest WHERE user_id=? AND name=?`, uid, name).Scan(&cooldown)
		if err != nil {
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
	return Object{"bonus_amount": amount, "bonus_wax_amount": amount, "currency": currency, "result": "ok", "update_forge_rates": []any{}, "update_world_quest": Object{"cooldown_over_time": cooldown, "ever_completed": true, "world_quest_def_id": name}}, nil
}
func (a *Service) CollectPickupBatch(r Request, m Object) (Object, error) {
	uid, err := a.questUser(r, m)
	if err != nil {
		return nil, err
	}
	level := text(m, "level_id")
	if level == "" {
		level = strconv.FormatInt(number(m, "level_id", 0), 10)
	}
	if _, ok := m["pickup_ids"].([]any); !ok {
		return nil, fail(400, "Missing pickup_ids")
	}
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	day := pacificDay(now())
	for _, kind := range []string{"pickup_ids", "global_pickup_ids"} {
		ids, _ := m[kind].([]any)
		for _, raw := range ids {
			id := fmt.Sprint(raw)
			res, e := tx.ExecContext(r.Context, `INSERT INTO collected_pickup(user_id,level_id,pickup_id,kind,date) VALUES(?,?,?,?,?) ON CONFLICT DO NOTHING`, uid, level, id, kind, day)
			if e != nil {
				return nil, e
			}
			first, e := insertionCount(res)
			if e != nil {
				return nil, e
			}
			if first {
				field, amount := "wax", int64(4)
				if kind == "global_pickup_ids" && id == "4180017154" {
					field, amount = "season_candle", 1
				}
				if err = tx.GrantCurrency(r.Context, uid, field, amount); err != nil {
					return nil, err
				}
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
	rates, err := a.ForgeRates(r, Object{"user": uid})
	if err != nil {
		return nil, err
	}
	return Object{"currency": currency, "level_id": m["level_id"], "update_forge_rates": rates["forge_rates"]}, nil
}
