package gameapi

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
)

func (a *API) achievementDefinitions(r *http.Request, m object) (object, error) {
	if _, err := a.authenticated(r, m); err != nil {
		return nil, err
	}
	rows, err := a.db.QueryContext(r.Context(), `SELECT achievement_id,name,description,icon_url,category,points,is_secret FROM achievement_def WHERE is_active=1 ORDER BY id`)
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

func (a *API) mapDefinitions(r *http.Request, m object) (object, error) {
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

func (a *API) achievements(r *http.Request, m object) (object, error) {
	u, err := a.authenticated(r, m)
	if err != nil {
		return nil, err
	}
	rows, err := a.db.QueryContext(r.Context(), `SELECT achievement_id,achievement_name,achievement_description,unlocked_at,COALESCE(progress,0),COALESCE(completed,0) FROM achievement WHERE user_id=? ORDER BY unlocked_at DESC,id`, u.ID)
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

func (a *API) badge(r *http.Request, m object) (object, error) {
	u, err := a.authenticated(r, m)
	if err != nil {
		return nil, err
	}
	if _, err = a.db.ExecContext(r.Context(), "INSERT INTO app_badge(user_id,badge_type) VALUES(?,'general') ON CONFLICT(user_id,badge_type) DO NOTHING", u.ID); err != nil {
		return nil, err
	}
	var n int64
	err = a.db.QueryRowContext(r.Context(), "SELECT COALESCE(badge_number,0) FROM app_badge WHERE user_id=? AND badge_type='general'", u.ID).Scan(&n)
	return object{"set_app_badge_number": n}, err
}

func (a *API) collectibles(r *http.Request, m object) (object, error) {
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
	rows, err := a.db.QueryContext(r.Context(), "SELECT id,COALESCE(used,0),COALESCE(candle_space,0),COALESCE(carrying,0),COALESCE(level,1),COALESCE(permanent_level,0) FROM collectible WHERE user_id=? ORDER BY row_id", u.ID)
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
