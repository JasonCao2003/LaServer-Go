// 成就、地图、徽章与收集品档案

package account

import (
	"database/sql"
	"encoding/json"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"

	"os"
	"path/filepath"
)

// 获取成就定义
func (a *Service) AchievementDefinitions(r Request, m object) (object, error) {
	if _, err := a.Authenticated(r, m); err != nil {
		return nil, err
	}
	rows, err := a.Store.QueryContext(r.Context, repository.ReadAchievementDefinitions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []object{}
	for rows.Next() {
		var id, name string
		var description, icon, category sql.NullString
		var points sql.NullInt64
		var secret sql.NullBool
		if err = rows.Scan(&id, &name, &description, &icon, &category, &points, &secret); err != nil {
			return nil, err
		}
		out = append(out, object{"id": id, "name": name, "description": nullableString(description), "icon_url": nullableString(icon), "category": nullableString(category), "points": nullableInt(points), "is_secret": nullableBool(secret)})
	}
	return object{"achievement_defs": out}, rows.Err()
}

// 可空字符串转接口值
func nullableString(v sql.NullString) any {
	if v.Valid {
		return v.String
	}
	return nil
}

// 可空整数转接口值
func nullableInt(v sql.NullInt64) any {
	if v.Valid {
		return v.Int64
	}
	return nil
}

// 可空布尔转接口值
func nullableBool(v sql.NullBool) any {
	if v.Valid {
		return v.Bool
	}
	return nil
}

// 获取地图定义
func (a *Service) MapDefinitions(r Request, m object) (object, error) {
	if a.Cfg.GameConfigDir == "" {
		return nil, fail(503, "Game configuration unavailable")
	}
	data, err := os.ReadFile(filepath.Join(a.Cfg.GameConfigDir, "map.json"))
	if err != nil {
		return nil, fail(503, "Map configuration unavailable")
	}
	var result object
	if json.Unmarshal(data, &result) != nil || result == nil {
		return nil, fail(503, "Invalid map configuration")
	}
	if !validConfiguredArray(result["get_map_defs"], "name") {
		return nil, fail(503, "Invalid map configuration")
	}
	return result, nil
}

// 获取成就列表
func (a *Service) Achievements(r Request, m object) (object, error) {
	u, err := a.Authenticated(r, m)
	if err != nil {
		return nil, err
	}
	rows, err := a.Store.QueryContext(r.Context, repository.ReadAchievements, u.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []object{}
	for rows.Next() {
		var id, name string
		var description sql.NullString
		var stamp, progress int64
		var completed bool
		if err = rows.Scan(&id, &name, &description, &stamp, &progress, &completed); err != nil {
			return nil, err
		}
		var desc any
		if description.Valid {
			desc = description.String
		}
		out = append(out, object{"id": id, "name": name, "description": desc, "unlocked_at": stamp, "progress": progress, "completed": completed})
	}
	return object{"achievements": out}, rows.Err()
}

// 获取徽章数量
func (a *Service) Badge(r Request, m object) (object, error) {
	u, err := a.Authenticated(r, m)
	if err != nil {
		return nil, err
	}
	if _, err = a.Store.ExecContext(r.Context, repository.EnsureBadge, u.ID); err != nil {
		return nil, err
	}
	var n int64
	err = a.Store.QueryRowContext(r.Context, repository.ReadBadge, u.ID).Scan(&n)
	return object{"set_app_badge_number": n}, err
}

// 获取收集品列表
func (a *Service) Collectibles(r Request, m object) (object, error) {
	if text(m, "user") == "" {
		m["user"] = text(m, "user_id")
	}
	if a.Cfg.GameConfigDir == "" {
		return nil, fail(503, "Game configuration unavailable")
	}
	raw, err := os.ReadFile(filepath.Join(a.Cfg.GameConfigDir, "collectibles.json"))
	if err != nil {
		return nil, fail(503, "Collectibles configuration unavailable")
	}
	var envelope object
	var defaults []object
	if json.Unmarshal(raw, &defaults) != nil {
		if json.Unmarshal(raw, &envelope) != nil {
			return nil, fail(503, "Invalid collectibles configuration")
		}
		data, err := json.Marshal(envelope["collectibles"])
		if err != nil || json.Unmarshal(data, &defaults) != nil {
			return nil, fail(503, "Invalid collectibles configuration")
		}
	}
	if defaults == nil {
		return nil, fail(503, "Invalid collectibles configuration")
	}
	for _, item := range defaults {
		if len(item) == 0 || text(item, "id") == "" {
			return nil, fail(503, "Invalid collectibles configuration")
		}
	}
	out := []object{}
	positions := map[string]int{}
	for _, item := range defaults {
		key := text(item, "id")
		if key == "" {
			continue
		}
		if _, ok := positions[key]; !ok {
			positions[key] = len(out)
			out = append(out, item)
		}
	}
	if text(m, "user") == "" || text(m, "session") == "" {
		return object{"collectibles": out}, nil
	}
	u, err := a.Authenticated(r, m)
	if err != nil {
		return nil, err
	}
	full, err := a.Policy("auto_full_unlock.json", integer(u.CreatedAt), false)
	if err != nil {
		return nil, err
	}
	if full {
		all := a.optionalFullCollectibles()
		combined := []object{}
		positions = map[string]int{}
		for _, item := range append(all, out...) {
			id := text(item, "id")
			if id == "" {
				continue
			}
			if _, ok := positions[id]; !ok {
				positions[id] = len(combined)
				combined = append(combined, item)
			}
		}
		out = combined
	}
	rows, err := a.Store.QueryContext(r.Context, repository.ReadCollectibles, u.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var used, space, carrying bool
		var level, permanent int64
		if err = rows.Scan(&id, &used, &space, &carrying, &level, &permanent); err != nil {
			return nil, err
		}
		item := object{"id": id, "used": used, "candle_space": space, "carrying": carrying, "level": level, "permanent_level": permanent}
		if pos, ok := positions[id]; ok {
			out[pos] = item
		} else {
			positions[id] = len(out)
			out = append(out, item)
		}
	}
	return object{"collectibles": out}, rows.Err()
}

// 辅助函数：可选完整收集品
func (a *Service) optionalFullCollectibles() []object {
	raw, err := os.ReadFile(filepath.Join(a.Cfg.GameConfigDir, "quandongzuo.json"))
	if err != nil {
		return nil
	}
	var all []object
	var envelope object
	if json.Unmarshal(raw, &all) == nil {
		return all
	}
	if json.Unmarshal(raw, &envelope) != nil {
		return nil
	}
	encoded, err := json.Marshal(envelope["collectibles"])
	if err != nil || json.Unmarshal(encoded, &all) != nil {
		return nil
	}
	return all
}
