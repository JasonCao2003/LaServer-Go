package service

import (
	"database/sql"
	"encoding/json"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"

	"os"
	"path/filepath"
)

func (a *Service) AchievementDefinitions(r Request, m object) (object, error) {
	if _, err := a.authenticated(r, m); err != nil {
		return nil, err
	}
	rows, err := a.store.QueryContext(r.Context, repository.ReadAchievementDefinitions)
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
func nullableString(v sql.NullString) any {
	if v.Valid {
		return v.String
	}
	return nil
}
func nullableInt(v sql.NullInt64) any {
	if v.Valid {
		return v.Int64
	}
	return nil
}
func nullableBool(v sql.NullBool) any {
	if v.Valid {
		return v.Bool
	}
	return nil
}

func (a *Service) MapDefinitions(r Request, m object) (object, error) {
	if a.cfg.GameConfigDir == "" {
		return nil, fail(503, "Game configuration unavailable")
	}
	data, err := os.ReadFile(filepath.Join(a.cfg.GameConfigDir, "map.json"))
	if err != nil {
		return nil, fail(503, "Map configuration unavailable")
	}
	var result object
	if json.Unmarshal(data, &result) != nil || result == nil {
		return nil, fail(503, "Invalid map configuration")
	}
	return result, nil
}

func (a *Service) Achievements(r Request, m object) (object, error) {
	u, err := a.authenticated(r, m)
	if err != nil {
		return nil, err
	}
	rows, err := a.store.QueryContext(r.Context, repository.ReadAchievements, u.ID)
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

func (a *Service) Badge(r Request, m object) (object, error) {
	u, err := a.authenticated(r, m)
	if err != nil {
		return nil, err
	}
	if _, err = a.store.ExecContext(r.Context, repository.EnsureBadge, u.ID); err != nil {
		return nil, err
	}
	var n int64
	err = a.store.QueryRowContext(r.Context, repository.ReadBadge, u.ID).Scan(&n)
	return object{"set_app_badge_number": n}, err
}

func (a *Service) Collectibles(r Request, m object) (object, error) {
	if text(m, "user") == "" {
		m["user"] = text(m, "user_id")
	}
	if a.cfg.GameConfigDir == "" {
		return nil, fail(503, "Game configuration unavailable")
	}
	raw, err := os.ReadFile(filepath.Join(a.cfg.GameConfigDir, "collectibles.json"))
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
	u, err := a.authenticated(r, m)
	if err != nil {
		return nil, err
	}
	full, err := a.policy("auto_full_unlock.json", integer(u.CreatedAt), false)
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
	rows, err := a.store.QueryContext(r.Context, repository.ReadCollectibles, u.ID)
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

// Full actions are optional; preserve base and owned items if the auxiliary file is unavailable.
func (a *Service) optionalFullCollectibles() []object {
	raw, err := os.ReadFile(filepath.Join(a.cfg.GameConfigDir, "quandongzuo.json"))
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
