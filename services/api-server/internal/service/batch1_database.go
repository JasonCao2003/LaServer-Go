package service

import (
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

func isoDate(value string) string { return strings.Replace(value, " ", "T", 1) }

func (a *Service) Surveys(r Request, m Object) (Object, error) {
	userID, err := a.batch1User(r, m)
	if err != nil {
		return nil, err
	}
	rows, err := a.store.QueryContext(r.Context, repository.ActiveSurveys, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Object{}
	for rows.Next() {
		var id int64
		var title, kind, start, end string
		var description sql.NullString
		if err = rows.Scan(&id, &title, &description, &kind, &start, &end); err != nil {
			return nil, err
		}
		var desc any
		if description.Valid {
			desc = description.String
		}
		out = append(out, Object{"id": id, "title": title, "description": desc, "type": kind, "start_date": isoDate(start), "end_date": isoDate(end)})
	}
	return Object{"surveys": out}, rows.Err()
}

func (a *Service) VideoDefinitions(r Request, m Object) (Object, error) {
	if _, err := a.batch1User(r, m); err != nil {
		return nil, err
	}
	defs := []Object{}
	rows, err := a.store.QueryContext(r.Context, repository.VideoDefinitions)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var name, location, subtitle string
		var duration int64
		var translated bool
		if err = rows.Scan(&name, &duration, &location, &subtitle, &translated); err != nil {
			rows.Close()
			return nil, err
		}
		defs = append(defs, Object{"name": name, "duration": duration, "location": location, "subtitle_base": subtitle, "has_translations": translated})
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	schedules := []Object{}
	rows, err = a.store.QueryContext(r.Context, repository.VideoSchedules)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var channel, event, raw string
		var pre, gap, post int64
		var loop, silent bool
		if err = rows.Scan(&channel, &event, &pre, &raw, &loop, &gap, &post, &silent); err != nil {
			return nil, err
		}
		var playlist any = []any{}
		if json.Valid([]byte(raw)) {
			_ = json.Unmarshal([]byte(raw), &playlist)
		}
		schedules = append(schedules, Object{"channel": channel, "event": event, "preshow_seconds": pre, "playlist": playlist, "loop": loop, "gap_seconds": gap, "postshow_seconds": post, "silence_please": silent})
	}
	return Object{"video_defs": defs, "video_schedule": schedules}, rows.Err()
}

func (a *Service) TrustActivityDefinitions(r Request, m Object) (Object, error) {
	if _, err := a.batch1User(r, m); err != nil {
		return nil, err
	}
	rows, err := a.store.QueryContext(r.Context, repository.TrustActivities)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Object{}
	for rows.Next() {
		var id, name, kind string
		var description sql.NullString
		var level, points int64
		if err = rows.Scan(&id, &name, &description, &kind, &level, &points); err != nil {
			return nil, err
		}
		var desc any
		if description.Valid {
			desc = description.String
		}
		out = append(out, Object{"activity_id": id, "activity_name": name, "description": desc, "activity_type": kind, "required_level": level, "reward_points": points})
	}
	return Object{"get_trust_activity_defs": out}, rows.Err()
}

func (a *Service) TrustGroupDefinitions(r Request, m Object) (Object, error) {
	userID, err := a.batch1User(r, m)
	if err != nil {
		return nil, err
	}
	rows, err := a.store.QueryContext(r.Context, repository.UserTrustGroups, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Object{}
	for rows.Next() {
		var id, name string
		var description, created, joined sql.NullString
		var min, max, members, level int64
		if err = rows.Scan(&id, &name, &description, &min, &max, &members, &level, &created, &joined); err != nil {
			return nil, err
		}
		var desc, createdAt, joinedAt any
		if description.Valid {
			desc = description.String
		}
		if created.Valid {
			createdAt = isoDate(created.String)
		}
		if joined.Valid {
			joinedAt = isoDate(joined.String)
		}
		out = append(out, Object{"group_id": id, "group_name": name, "description": desc, "min_trust_level": min, "max_members": max, "current_members": members, "user_trust_level": level, "created_at": createdAt, "joined_at": joinedAt})
	}
	return Object{"get_trust_group_defs": out}, rows.Err()
}

func (a *Service) InfractionStatus(r Request, m Object) (Object, error) {
	userID, err := a.batch1User(r, m)
	if err != nil {
		return nil, err
	}
	rows, err := a.store.QueryContext(r.Context, repository.ActiveInfractions, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	count, top, rank := 0, "low", 0
	for rows.Next() {
		var severity string
		if err = rows.Scan(&severity); err != nil {
			return nil, err
		}
		count++
		weight := map[string]int{"low": 1, "medium": 2, "high": 3}[severity]
		if weight == 0 {
			weight = 1
		}
		if count == 1 || weight > rank {
			rank, top = weight, severity
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	var access any
	if count > 0 {
		access = Object{"restricted": true, "reason": "Account has active infractions", "infractions_count": count, "severity": top}
	}
	return Object{"status": "OK", "access_status": access}, nil
}
