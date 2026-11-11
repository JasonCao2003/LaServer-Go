package service

import (
	"database/sql"
	"errors"
)

var defaultForgeRates = []Object{
	{"source_type": "wax", "current_tier": 26, "current_level": 4, "tier_completion": 0.0, "level_completion": 0.0, "level_tier_index": 0, "level_tier_count": 5, "reset_acknowledged": false, "enabled": true},
	{"source_type": "color_yellow_wax", "current_tier": 4, "current_level": 1, "tier_completion": 0.0, "level_completion": 0.0, "level_tier_index": 0, "level_tier_count": 3, "reset_acknowledged": false, "enabled": true},
	{"source_type": "color_white_wax", "current_tier": 2, "current_level": 1, "tier_completion": 0.0, "level_completion": 0.0, "level_tier_index": 0, "level_tier_count": 1, "reset_acknowledged": false, "enabled": true},
	{"source_type": "color_red_wax", "current_tier": 5, "current_level": 1, "tier_completion": 0.0, "level_completion": 0.0, "level_tier_index": 0, "level_tier_count": 4, "reset_acknowledged": false, "enabled": true},
	{"source_type": "color_magenta_wax", "current_tier": 4, "current_level": 1, "tier_completion": 0.0, "level_completion": 0.0, "level_tier_index": 0, "level_tier_count": 3, "reset_acknowledged": false, "enabled": true},
	{"source_type": "color_green_wax", "current_tier": 5, "current_level": 1, "tier_completion": 0.0, "level_completion": 0.0, "level_tier_index": 0, "level_tier_count": 4, "reset_acknowledged": false, "enabled": true},
	{"source_type": "color_cyan_wax", "current_tier": 4, "current_level": 1, "tier_completion": 0.0, "level_completion": 0.0, "level_tier_index": 0, "level_tier_count": 3, "reset_acknowledged": false, "enabled": true},
	{"source_type": "color_blue_wax", "current_tier": 5, "current_level": 1, "tier_completion": 0.0, "level_completion": 0.0, "level_tier_index": 0, "level_tier_count": 4, "reset_acknowledged": false, "enabled": true},
	{"source_type": "color_black_wax", "current_tier": 2, "current_level": 1, "tier_completion": 0.0, "level_completion": 0.0, "level_tier_index": 0, "level_tier_count": 1, "reset_acknowledged": false, "enabled": true},
}

func (a *Service) ForgeRates(r Request, m Object) (Object, error) {
	userID := text(m, "user")
	if userID == "" {
		userID = text(m, "user_id")
	}
	rows, err := a.store.ForgeRates(r.Context, userID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return Object{"forge_rates": defaultForgeRates}, nil
	}
	out := make([]Object, 0, len(rows))
	for _, v := range rows {
		out = append(out, Object{"source_type": v.SourceType, "current_tier": v.CurrentTier, "current_level": v.CurrentLevel, "tier_completion": v.TierCompletion, "level_completion": v.LevelCompletion, "level_tier_index": v.LevelTierIndex, "level_tier_count": v.LevelTierCount, "reset_acknowledged": v.ResetAcknowledged, "enabled": v.Enabled})
	}
	return Object{"forge_rates": out}, nil
}

func (a *Service) Outfit(r Request, m Object) (Object, error) {
	userID := text(m, "user")
	if userID == "" {
		return nil, fail(401, "User not found")
	}
	exists, err := a.store.HasUser(r.Context, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fail(401, "User not found")
	}
	row, err := a.store.GetOrCreateOutfit(r.Context, userID)
	if err != nil {
		return nil, err
	}
	result := Object{"result": "ok", "set_outfit": Object{"voice": row.Voice, "seed": row.Seed, "scale": row.Scale, "refreshversion": row.RefreshVersion, "height": row.Height, "attitude": row.Attitude}}
	set := result["set_outfit"].(Object)
	for name, part := range row.Parts {
		set[name] = part
	}
	set["arms"] = Object{"id": row.ArmsID}
	return result, nil
}

func (a *Service) Questionnaires(r Request, m Object) (Object, error) {
	if text(m, "user") == "" || text(m, "session") == "" {
		return nil, &Error{Status: 400, Body: Object{"result": "missing_parameters"}}
	}
	u, err := a.findUser(r.Context, "id", text(m, "user"))
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != text(m, "session")) {
		return nil, &Error{Status: 401, Body: Object{"result": "invalid_session"}}
	}
	if err != nil {
		return nil, err
	}
	return Object{"result": "ok"}, nil
}

func (a *Service) AchievementStats(r Request, m Object) (Object, error) {
	u, err := a.authenticated(r, m)
	if err != nil {
		return nil, err
	}
	rows, err := a.store.AchievementStats(r.Context, u.ID, now())
	if err != nil {
		return nil, err
	}
	out := make([]Object, 0, len(rows))
	for _, v := range rows {
		out = append(out, Object{"type": v.Type, "value": v.Value, "update": v.Update})
	}
	return Object{"achievement_stats": out}, nil
}

func (a *Service) AchievementStatsTracking(_ Request, m Object) (Object, error) {
	if err := require(m, "user"); err != nil {
		return nil, err
	}
	return Object{"achievement_stats": []Object{{"type": "visit_night_vase_count", "value": int64(40000000000), "update": int64(1692419146)}}}, nil
}

func (a *Service) EventCurrencyDefinitions(_ Request, m Object) (Object, error) {
	if err := require(m, "user"); err != nil {
		return nil, err
	}
	var result Object
	if err := legacyResponse("event_currency_defs", &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (a *Service) RelationshipDefinitions(r Request, m Object) (Object, error) {
	if err := require(m, "user"); err != nil {
		return nil, err
	}
	var result struct {
		Definitions []map[string]any `json:"relationship_defs"`
	}
	if err := legacyResponse("relationship_abilities", &result); err != nil {
		return nil, err
	}
	return Object{"relationship_defs": legacyRelationshipDefs(r.UserAgent, result.Definitions)}, nil
}
