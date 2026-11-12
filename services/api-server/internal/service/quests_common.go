package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

func (a *Service) EmptyDailyQuests(_ Request, _ Object) (Object, error) {
	return Object{"daily_quests": []any{}}, nil
}
func (a *Service) EmptyWeeklyQuests(_ Request, _ Object) (Object, error) {
	return Object{"weekly_quests": []any{}}, nil
}
func (a *Service) EmptySeasonQuestsV2(_ Request, _ Object) (Object, error) {
	return Object{"season_quests": []any{}, "has_season_quests_rewards": []any{}}, nil
}
func (a *Service) EmptySeasonPassDefs(_ Request, _ Object) (Object, error) {
	return Object{"season_pass_defs": []any{}}, nil
}
func (a *Service) EmptyEventCurrency(_ Request, _ Object) (Object, error) {
	return Object{"event_currency": []any{}}, nil
}

func (a *Service) questUser(r Request, m Object) (string, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	if uid == "" || text(m, "session") == "" {
		return "", fail(400, "Invalid request body")
	}
	u, err := a.findUser(r.Context, "id", uid)
	if errors.Is(err, sql.ErrNoRows) || err == nil && stringValue(u.Session) != text(m, "session") {
		return "", fail(401, "Invalid session or user not found")
	}
	if err != nil {
		return "", err
	}
	return uid, nil
}
func (a *Service) gameControl() Object {
	if a.cfg.GameConfigDir == "" {
		return Object{}
	}
	b, err := os.ReadFile(filepath.Join(a.cfg.GameConfigDir, "game_control.json"))
	if err != nil {
		return Object{}
	}
	var out Object
	if json.Unmarshal(b, &out) != nil || out == nil {
		return Object{}
	}
	return out
}
func nestedObject(m Object, k string) Object {
	v, _ := m[k].(map[string]any)
	if v == nil {
		return Object{}
	}
	return v
}
func questPool(v any) []Object {
	raw, _ := v.([]any)
	out := make([]Object, 0, len(raw))
	for _, item := range raw {
		q, ok := item.(map[string]any)
		if !ok || text(q, "daily_quest_def_id") == "" {
			continue
		}
		out = append(out, q)
	}
	return out
}
func (a *Service) dailyQuestPools() (season, normal []Object) {
	cfg := a.gameControl()
	dq := nestedObject(cfg, "daily_quests")
	season = questPool(nestedObject(dq, "season")["pool"])
	normal = questPool(nestedObject(dq, "normal")["pool"])
	if len(season)+len(normal) == 0 {
		season = questPool(nestedObject(cfg, "season_quests")["daily_pool"])
	}
	if len(season)+len(normal) == 0 {
		for _, pair := range [][2]string{{"fetchlight_mind", "fetchlight_mind_count"}, {"pickup_30_wax", "wax_pickup_count"}, {"hug_someone", "non_unique_hug_count"}, {"save_a_spirit_in_night", "night_collectibles_collected_count"}} {
			season = append(season, Object{"daily_quest_def_id": pair[0], "stat_type": pair[1]})
		}
	}
	return
}
func (a *Service) questReward(season bool) int64 {
	key := "reward_candle"
	if season {
		key = "reward_season_candle"
	}
	n := number(nestedObject(a.gameControl(), "season_quests"), key, 1)
	if n < 0 {
		return 0
	}
	return n
}
func (a *Service) seasonNumber() int64 {
	n := number(nestedObject(a.gameControl(), "season"), "current_season", int64(a.cfg.CurrentSeason))
	if n < 0 {
		return 0
	}
	return n
}
func (a *Service) seasonDisabled(name string) bool {
	if a.cfg.GameConfigDir == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(a.cfg.GameConfigDir, name+"_disabled"))
	return err == nil
}
func pacificDay(epoch int64) string {
	zone, _ := time.LoadLocation("America/Los_Angeles")
	return time.Unix(epoch, 0).In(zone).Format("2006-01-02")
}
func localDay() string { return pacificDay(now()) }
func questNames(m Object) []string {
	raw, _ := m["quest_names"].([]any)
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}
func questID(m Object) string                   { return text(m, "quest_id") }
func insertionCount(r sql.Result) (bool, error) { n, e := r.RowsAffected(); return n > 0, e }
func questStats(ctx context.Context, reader interface {
	QueryContext(context.Context, repository.Statement, ...any) (*sql.Rows, error)
}, id string) (Object, error) {
	out := Object{"init_stats": []Object{}, "completion_requirements": []Object{}, "target_values": []Object{}}
	rows, err := reader.QueryContext(ctx, `SELECT category,stat_type,value FROM quest_stat WHERE quest_id=? ORDER BY stat_type,category`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var category, st string
		var value float64
		if err = rows.Scan(&category, &st, &value); err != nil {
			return nil, err
		}
		if list, ok := out[category].([]Object); ok {
			out[category] = append(list, Object{"stat_type": st, "value": value})
		}
	}
	return out, rows.Err()
}
func questDefinition(a *Service, name string) Object {
	for _, q := range a.magicDefinitions("get_quest_defs", "account__quest__get_quest_defs.json") {
		if text(q, "id") == name {
			return q
		}
	}
	configured, err := a.configuredObject("quest.json")
	if err == nil {
		if defs, ok := configured["get_quest_defs"].([]any); ok {
			for _, v := range defs {
				if q, ok := v.(map[string]any); ok && text(q, "id") == name {
					return q
				}
			}
		}
	}
	return Object{}
}
