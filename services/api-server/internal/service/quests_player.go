package service

import (
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

var questStatuses = []string{"inactive", "active", "failed", "succeeded", "collected", "canceled", "suspended"}

func emptyQuestBuckets() Object {
	out := Object{}
	for _, status := range questStatuses {
		out[status] = []Object{}
	}
	return out
}
func (a *Service) questRow(r Request, id string) (Object, error) {
	var name, status string
	var start, expiry int64
	err := a.store.QueryRowContext(r.Context, `SELECT quest_name,status,start_time,expiry_time FROM player_quest WHERE quest_id=?`, id).Scan(&name, &status, &start, &expiry)
	if err != nil {
		return nil, err
	}
	stats, err := questStats(r.Context, a.store, id)
	if err != nil {
		return nil, err
	}
	return Object{"quest_name": name, "quest_id": id, "start_time": start, "expiry_time": expiry, "stats": stats, "status": status}, nil
}
func (a *Service) PlayerQuests(r Request, m Object) (Object, error) {
	out := emptyQuestBuckets()
	uid, err := a.questUser(r, m)
	if err != nil {
		return out, nil
	}
	rows, err := a.store.QueryContext(r.Context, `SELECT quest_id,status FROM player_quest WHERE user_id=? ORDER BY start_time`, uid)
	if err != nil {
		return nil, err
	}
	type pair struct{ id, status string }
	pairs := []pair{}
	for rows.Next() {
		var p pair
		if err = rows.Scan(&p.id, &p.status); err != nil {
			rows.Close()
			return nil, err
		}
		pairs = append(pairs, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for _, p := range pairs {
		q, e := a.questRow(r, p.id)
		if e != nil {
			return nil, e
		}
		if list, ok := out[p.status].([]Object); ok {
			out[p.status] = append(list, q)
		}
	}
	return out, nil
}
func (a *Service) PlayerQuestsByName(r Request, m Object) (Object, error) {
	result := Object{}
	uid, authErr := a.questUser(r, m)
	for _, name := range questNames(m) {
		entry := Object{"quest_id": nil, "quest_name": "", "start_time": 0, "stats": Object{}, "status": ""}
		if authErr == nil {
			var id string
			err := a.store.QueryRowContext(r.Context, `SELECT quest_id FROM player_quest WHERE user_id=? AND quest_name=? ORDER BY start_time DESC LIMIT 1`, uid, name).Scan(&id)
			if err == nil {
				entry, err = a.questRow(r, id)
				if err != nil {
					return nil, err
				}
			} else if !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}
		}
		result[name] = entry
	}
	return Object{"results": result}, nil
}
func (a *Service) CheckPreviousCompletion(r Request, m Object) (Object, error) {
	uid, authErr := a.questUser(r, m)
	out := []Object{}
	for _, name := range questNames(m) {
		today, previous := false, false
		if authErr == nil {
			var count int
			err := a.store.QueryRowContext(r.Context, `SELECT COUNT(*) FROM season_quest_reward WHERE user_id=? AND quest_id=? AND rewarded=1`, uid, name).Scan(&count)
			if err != nil {
				return nil, err
			}
			previous = count > 0
			err = a.store.QueryRowContext(r.Context, `SELECT COUNT(*) FROM season_quest_reward WHERE user_id=? AND quest_id=? AND date=? AND rewarded=1`, uid, name, localDay()).Scan(&count)
			if err != nil {
				return nil, err
			}
			today = count > 0
		}
		out = append(out, Object{"quest_name": name, "completed_today": today, "previously_completed": previous})
	}
	return Object{"results": out}, nil
}
func (a *Service) CreateQuest(r Request, m Object) (Object, error) {
	uid, err := a.questUser(r, m)
	if err != nil {
		return nil, err
	}
	name := text(m, "quest_name")
	if name == "" {
		return nil, fail(400, "Missing quest_name")
	}
	seasonPool, normalPool := a.dailyQuestPools()
	allowed := false
	for _, q := range append(seasonPool, normalPool...) {
		if text(q, "daily_quest_def_id") == name {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, fail(404, "Quest not scheduled")
	}
	current := number(m, "current_time", now())
	zone, _ := time.LoadLocation("America/Los_Angeles")
	t := time.Unix(current, 0).In(zone)
	expiry := time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, zone).Unix() - 1
	kind := "daily"
	season, _ := a.dailyQuestPools()
	for _, q := range season {
		if text(q, "daily_quest_def_id") == name {
			sc := nestedObject(a.gameControl(), "season")
			start := number(sc, "start_time", 0)
			end := start + number(sc, "duration_days", 70)*86400
			if start > 0 && end > current {
				kind = "season"
				expiry = end
			}
			break
		}
	}
	def := questDefinition(a, name)
	statsTypes := []string{}
	if raw, ok := def["completion_stats_types"].([]any); ok {
		for _, v := range raw {
			if s, ok := v.(string); ok && s != "" {
				statsTypes = append(statsTypes, s)
			}
		}
	}
	if len(statsTypes) == 0 {
		statsTypes = []string{name + "_count"}
	}
	vals, _ := def["completion_stats_values"].([]any)
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var id string
	var start int64
	var oldExpiry int64
	var status string
	err = tx.QueryRowContext(r.Context, `SELECT quest_id,start_time,expiry_time,status FROM player_quest WHERE user_id=? AND quest_name=? AND expiry_time>=? AND status IN ('active','succeeded') ORDER BY start_time DESC LIMIT 1`, uid, name, current).Scan(&id, &start, &oldExpiry, &status)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if errors.Is(err, sql.ErrNoRows) {
		id, err = uuid()
		if err != nil {
			return nil, err
		}
		start = current
		oldExpiry = expiry
		status = "active"
		_, err = tx.ExecContext(r.Context, `INSERT INTO player_quest(quest_id,user_id,quest_name,start_time,expiry_time,status) VALUES(?,?,?,?,?,'active')`, id, uid, name, start, expiry)
		if err != nil {
			return nil, err
		}
		for i, st := range statsTypes {
			req := int64(1)
			if i < len(vals) {
				req = defInt(vals[i])
				if req < 1 {
					req = 1
				}
			}
			var base float64
			err = tx.QueryRowContext(r.Context, `SELECT value FROM achievement_stat WHERE user_id=? AND stat_type=?`, uid, st).Scan(&base)
			if errors.Is(err, sql.ErrNoRows) {
				base = 0
			} else if err != nil {
				return nil, err
			}
			for _, row := range []struct {
				category string
				value    float64
			}{{"baseline", base}, {"init_stats", 0}, {"completion_requirements", float64(req)}, {"target_values", float64(req)}} {
				_, err = tx.ExecContext(r.Context, `INSERT INTO quest_stat(quest_id,stat_type,category,value) VALUES(?,?,?,?)`, id, st, row.category, row.value)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	stats, err := questStats(r.Context, tx, id)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return Object{"expiry_time": oldExpiry, "is_scheduled_quest": def["scheduled_event"] == true, "previously_completed": false, "quest_giver": text(def, "npc"), "quest_id": id, "quest_name": name, "quest_type": kind, "start_time": start, "stats": stats, "status": status, "success": true}, nil
}
func (a *Service) UpdateQuest(r Request, m Object) (Object, error) {
	uid, err := a.questUser(r, m)
	if err != nil {
		return nil, err
	}
	id := questID(m)
	if id == "" {
		return nil, fail(400, "Missing quest_id")
	}
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var name, status string
	err = tx.QueryRowContext(r.Context, `SELECT quest_name,status FROM player_quest WHERE user_id=? AND quest_id=?`, uid, id).Scan(&name, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fail(404, "Quest not found")
	}
	if err != nil {
		return nil, err
	}
	if m["collect"] == true {
		reward := int64(0)
		if status != "collected" {
			if status != "succeeded" {
				return nil, fail(409, "Quest not completed")
			}
			_, err = tx.ExecContext(r.Context, `UPDATE player_quest SET status='collected' WHERE user_id=? AND quest_id=?`, uid, id)
			if err != nil {
				return nil, err
			}
			marker, markErr := tx.ExecContext(r.Context, `INSERT INTO season_quest_reward(user_id,date,quest_id,rewarded) VALUES(?,?,?,1) ON CONFLICT(user_id,date,quest_id) DO UPDATE SET rewarded=1 WHERE rewarded=0`, uid, localDay(), name)
			err = markErr
			if err != nil {
				return nil, err
			}
			first, countErr := insertionCount(marker)
			if countErr != nil {
				return nil, countErr
			}
			if first {
				reward = a.questReward(true)
				if err = tx.GrantCurrency(r.Context, uid, "season_candle", reward); err != nil {
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
		return Object{"success": true, "status": "collected", "rewards": Object{"currency": Object{"season_candle": reward}, "season_reward_id": Object{"quest_season_reward": 1}}, "fulfillment": Object{"currency": currency}, "purchase_fx": Object{"currency_count": reward, "currency_type": "season_candle", "sound_id": 1}, "fulfilled_types": []string{"currency"}}, nil
	}
	next := text(m, "status")
	if next == "" {
		next = "active"
	}
	valid := false
	for _, s := range questStatuses {
		if next == s {
			valid = true
		}
	}
	if !valid {
		return nil, fail(400, "Invalid quest status")
	}
	if (next == "succeeded" && status != "succeeded") || (next == "collected" && status != "collected") {
		return nil, fail(409, "Quest status is server controlled")
	}
	if status != "collected" {
		_, err = tx.ExecContext(r.Context, `UPDATE player_quest SET status=? WHERE user_id=? AND quest_id=?`, next, uid, id)
		if err != nil {
			return nil, err
		}
	} else {
		next = status
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return Object{"status": next, "success": true}, nil
}
func (a *Service) SetAchievementStats(r Request, m Object) (Object, error) {
	uid, err := a.questUser(r, m)
	if err != nil {
		return nil, err
	}
	raw, ok := m["achievement_stats"].([]any)
	if !ok {
		return nil, fail(400, "Invalid achievement_stats")
	}
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	stamp := now()
	for _, v := range raw {
		item, ok := v.(map[string]any)
		if !ok {
			continue
		}
		st := text(item, "type")
		if st == "" {
			continue
		}
		val, ok := numericStat(item["value"])
		if !ok {
			continue
		}
		_, err = tx.ExecContext(r.Context, `INSERT INTO achievement_stat(user_id,stat_type,value,updated_at) VALUES(?,?,?,?) ON CONFLICT(user_id,stat_type) DO UPDATE SET value=excluded.value,updated_at=excluded.updated_at`, uid, st, val, stamp)
		if err != nil {
			return nil, err
		}
	}
	rows, err := tx.QueryContext(r.Context, `SELECT quest_id FROM player_quest WHERE user_id=? AND status='active'`, uid)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		reqRows, e := tx.QueryContext(r.Context, `SELECT q.stat_type,q.value,COALESCE(b.value,0),COALESCE(a.value,0) FROM quest_stat q LEFT JOIN quest_stat b ON b.quest_id=q.quest_id AND b.stat_type=q.stat_type AND b.category='baseline' LEFT JOIN achievement_stat a ON a.user_id=? AND a.stat_type=q.stat_type WHERE q.quest_id=? AND q.category='completion_requirements'`, uid, id)
		if e != nil {
			return nil, e
		}
		all, has := true, false
		type progress struct {
			st    string
			value float64
		}
		updates := []progress{}
		for reqRows.Next() {
			var st string
			var needed, base, cur float64
			if e = reqRows.Scan(&st, &needed, &base, &cur); e != nil {
				reqRows.Close()
				return nil, e
			}
			has = true
			p := math.Max(0, cur-base)
			updates = append(updates, progress{st, p})
			if p < needed {
				all = false
			}
		}
		e = reqRows.Err()
		reqRows.Close()
		if e != nil {
			return nil, e
		}
		for _, p := range updates {
			_, err = tx.ExecContext(r.Context, `UPDATE quest_stat SET value=? WHERE quest_id=? AND stat_type=? AND category='init_stats'`, p.value, id, p.st)
			if err != nil {
				return nil, err
			}
		}
		if has && all {
			_, err = tx.ExecContext(r.Context, `UPDATE player_quest SET status='succeeded' WHERE quest_id=? AND status='active'`, id)
			if err != nil {
				return nil, err
			}
		}
	}
	out := []Object{}
	rows, err = tx.QueryContext(r.Context, `SELECT stat_type,value,updated_at FROM achievement_stat WHERE user_id=? ORDER BY stat_type`, uid)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var st string
		var val float64
		var stamp int64
		if err = rows.Scan(&st, &val, &stamp); err != nil {
			rows.Close()
			return nil, err
		}
		var value any = val
		if val == math.Trunc(val) {
			value = int64(val)
		}
		out = append(out, Object{"type": st, "value": value, "update": stamp})
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return Object{"achievement_stats": out}, nil
}
func numericStat(v any) (float64, bool) {
	switch x := v.(type) {
	case bool:
		return 0, false
	case float64:
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return 0, false
		}
		return x, true
	case json.Number:
		n, err := x.Float64()
		return n, err == nil && !math.IsNaN(n) && !math.IsInf(n, 0)
	case int64:
		return float64(x), true
	case string:
		n, e := strconv.ParseFloat(x, 64)
		return n, e == nil && !math.IsNaN(n) && !math.IsInf(n, 0)
	default:
		return 0, false
	}
}
func (a *Service) QuestSchedule(_ Request, m Object) (Object, error) {
	zone, _ := time.LoadLocation("America/Los_Angeles")
	base := time.Unix(number(m, "current_time", now()), 0).In(zone)
	season, normal := a.dailyQuestPools()
	pool := append(season, normal...)
	renderable := map[string]bool{}
	for _, q := range a.magicDefinitions("get_quest_defs", "account__quest__get_quest_defs.json") {
		renderable[text(q, "id")] = true
	}
	catalog := []string{}
	if a.cfg.GameConfigDir != "" {
		if raw, err := os.ReadFile(filepath.Join(a.cfg.GameConfigDir, "daily_quest_catalog.json")); err == nil {
			var data Object
			if json.Unmarshal(raw, &data) == nil {
				for _, q := range questPool(data["daily_quest_catalog"]) {
					id := text(q, "daily_quest_def_id")
					if len(renderable) == 0 || renderable[id] {
						catalog = append(catalog, id)
					}
				}
			}
		}
	}
	out := []Object{}
	for delta := -1; delta <= 6; delta++ {
		d := time.Date(base.Year(), base.Month(), base.Day()+delta, 0, 0, 0, 0, zone)
		epoch := d.Unix()
		ids := []string{}
		if delta == 0 {
			seen := map[string]bool{}
			for _, q := range pool {
				id := text(q, "daily_quest_def_id")
				if id != "" && !seen[id] && (len(renderable) == 0 || renderable[id]) {
					ids = append(ids, id)
					seen[id] = true
				}
			}
		}
		if delta != 0 || len(ids) == 0 {
			ids = previewQuestIDs(d.Format("2006-01-02"), catalog)
		}
		for _, id := range ids {
			out = append(out, Object{"epoch": epoch, "quest_id": id})
		}
	}
	return Object{"quest_schedule": out}, nil
}

func previewQuestIDs(day string, catalog []string) []string {
	if len(catalog) == 0 {
		return []string{}
	}
	digest := md5.Sum([]byte(day))
	seed := uint32(digest[12])<<24 | uint32(digest[13])<<16 | uint32(digest[14])<<8 | uint32(digest[15])
	count := 4
	if len(catalog) < count {
		count = len(catalog)
	}
	out := make([]string, 0, count)
	used := map[int]bool{}
	for len(out) < count {
		seed = (seed*1103515245 + 12345) & 0x7fffffff
		j := int(seed % uint32(len(catalog)))
		for used[j] {
			j = (j + 1) % len(catalog)
		}
		used[j] = true
		out = append(out, catalog[j])
	}
	return out
}
