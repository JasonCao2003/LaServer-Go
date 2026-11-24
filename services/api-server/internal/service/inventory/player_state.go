// 玩家锻造、装扮、问卷与成就统计

package inventory

import (
	"database/sql"
	"errors"
)

// 获取装扮
func (a *Service) Outfit(r Request, m Object) (Object, error) {
	userID := text(m, "user")
	if userID == "" {
		return nil, fail(401, "User not found")
	}
	exists, err := a.Store.HasUser(r.Context, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fail(401, "User not found")
	}
	row, err := a.Store.GetOrCreateOutfit(r.Context, userID)
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

// 获取问卷
func (a *Service) Questionnaires(r Request, m Object) (Object, error) {
	if text(m, "user") == "" || text(m, "session") == "" {
		return nil, &Error{Status: 400, Body: Object{"result": "missing_parameters"}}
	}
	u, err := a.FindUser(r.Context, "id", text(m, "user"))
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != text(m, "session")) {
		return nil, &Error{Status: 401, Body: Object{"result": "invalid_session"}}
	}
	if err != nil {
		return nil, err
	}
	return Object{"result": "ok"}, nil
}

// 读取成就统计
func (a *Service) AchievementStats(r Request, m Object) (Object, error) {
	u, err := a.Authenticated(r, m)
	if err != nil {
		return nil, err
	}
	rows, err := a.Store.AchievementStats(r.Context, u.ID, now())
	if err != nil {
		return nil, err
	}
	out := make([]Object, 0, len(rows))
	for _, v := range rows {
		out = append(out, Object{"type": v.Type, "value": v.Value, "update": v.Update})
	}
	return Object{"achievement_stats": out}, nil
}

// 获取成就统计跟踪
func (a *Service) AchievementStatsTracking(_ Request, m Object) (Object, error) {
	if err := require(m, "user"); err != nil {
		return nil, err
	}
	return Object{"achievement_stats": []Object{{"type": "visit_night_vase_count", "value": int64(40000000000), "update": int64(1692419146)}}}, nil
}

// 获取活动货币定义
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

// 获取关系能力定义
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
