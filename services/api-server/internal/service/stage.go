package service

import (
	"encoding/json"
	"strings"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

const nilUserID = "00000000-0000-0000-0000-000000000000"

func stageIDFor(m Object) (string, string) {
	stageID := strings.TrimSpace(text(m, "stage_id"))
	levelID := strings.TrimSpace(text(m, "level_id"))
	if stageID == "" && levelID != "" {
		stageID = levelID + "_CNDL"
	}
	return stageID, levelID
}

func normalizeStageProp(raw any) (Object, bool) {
	prop, ok := raw.(map[string]any)
	if !ok {
		return nil, false
	}
	pos := text(prop, "pos")
	if pos == "" {
		pos = text(prop, "position")
	}
	if pos == "" {
		pos = "(0,0,0)"
	}
	ori := text(prop, "ori")
	if ori == "" {
		ori = text(prop, "rotation")
	}
	if ori == "" {
		ori = text(prop, "rot")
	}
	if ori == "" {
		ori = "(0,0,0,1)"
	}
	scale := text(prop, "scale")
	if scale == "" {
		scale = "(1,1,1)"
	}
	name := text(prop, "name")
	if name == "" {
		name = text(prop, "prop_name")
	}
	if name == "" {
		name = text(prop, "prop")
	}
	out := Object{"name": name, "pos": pos, "ori": ori, "scale": scale, "type": "unlock", "userDataBool": boolValue(prop["userDataBool"]), "version": number(prop, "version", 0)}
	if value := text(prop, "type"); value != "" {
		out["type"] = value
	}
	for key, value := range prop {
		if _, exists := out[key]; !exists {
			out[key] = value
		}
	}
	return out, true
}

func normalizeStageProps(list []any) []Object {
	out := []Object{}
	for _, raw := range list {
		if prop, ok := normalizeStageProp(raw); ok {
			out = append(out, prop)
		}
	}
	return out
}

func stagePropsFromRequest(m Object) ([]any, bool) {
	keys := []string{"props", "prop", "items", "placements", "stage_props", "props_list", "objects", "props_data"}
	dictList := func(v any) ([]any, bool) {
		list, ok := v.([]any)
		if !ok {
			return nil, false
		}
		for _, item := range list {
			if _, ok := item.(map[string]any); !ok {
				return nil, false
			}
		}
		return list, true
	}
	for _, key := range keys {
		if list, ok := dictList(m[key]); ok {
			return list, true
		}
	}
	for _, key := range keys {
		if nested, ok := m[key].(map[string]any); ok {
			for _, inner := range keys {
				if list, ok := dictList(nested[inner]); ok {
					return list, true
				}
			}
		}
	}
	skip := map[string]bool{"user": true, "session": true, "user_id": true, "stage_id": true, "level_id": true, "sequence": true}
	var best []any
	for key, value := range m {
		if skip[key] {
			continue
		}
		if list, ok := dictList(value); ok && (best == nil || len(list) > len(best)) {
			best = list
		}
	}
	return best, best != nil
}

func (a *Service) StageGet(r Request, m Object) (Object, error) {
	stageID, levelID := stageIDFor(m)
	caller := text(m, "user")
	target := text(m, "user_id")
	if target == nilUserID {
		target = ""
	}
	owner := target
	if owner == "" {
		owner = caller
	}
	state, found, err := a.store.StageState(r.Context, stageID, owner)
	if err != nil {
		return Object{"level_id": "", "props": []Object{}, "sequence": 1, "stage_id": "", "status": "OK", "user_id": nilUserID}, nil
	}
	props := []Object{}
	if found {
		var raw []any
		if json.Unmarshal([]byte(state.Props), &raw) == nil {
			props = normalizeStageProps(raw)
		}
	}
	sequence := int64(1)
	if found && state.Sequence > 0 {
		sequence = state.Sequence
	}
	respLevel := levelID
	if respLevel == "" && found {
		respLevel = state.LevelID
	}
	respUser := owner
	if found && state.UserID != "" {
		respUser = state.UserID
	}
	if respUser == "" {
		respUser = nilUserID
	}
	return Object{"level_id": respLevel, "props": props, "sequence": sequence, "stage_id": stageID, "status": "OK", "user_id": respUser, "updated_at": state.UpdatedAt}, nil
}

func (a *Service) StageSet(r Request, m Object) (Object, error) {
	stageID, levelID := stageIDFor(m)
	owner, err := a.questUser(r, m)
	if err != nil {
		return nil, err
	}
	if claimed := text(m, "user_id"); claimed != "" && claimed != nilUserID && claimed != owner {
		return nil, fail(403, "Stage owner mismatch")
	}
	propsList, ok := stagePropsFromRequest(m)
	if !ok {
		return nil, fail(400, "Missing stage props")
	}
	if stageID == "" {
		return nil, fail(400, "Missing stage_id")
	}
	props := normalizeStageProps(propsList)
	encoded, err := json.Marshal(props)
	if err != nil {
		return nil, err
	}
	old, found, err := a.store.StageState(r.Context, stageID, owner)
	if err != nil {
		return nil, err
	}
	sequence := number(m, "sequence", 0)
	if sequence <= 0 {
		sequence = 1
		if found && old.Sequence > 0 {
			sequence = old.Sequence + 1
		}
	}
	storedLevel := levelID
	if storedLevel == "" && found {
		storedLevel = old.LevelID
	}
	storedUser := owner
	if storedUser == "" && found {
		storedUser = old.UserID
	}
	state := repository.StageState{StageID: stageID, OwnerID: owner, Props: string(encoded), Sequence: sequence, LevelID: storedLevel, UserID: storedUser, UpdatedAt: now()}
	if err = a.store.SaveStageState(r.Context, state); err != nil {
		return nil, err
	}
	return Object{"status": "OK"}, nil
}

func (a *Service) FriendsInApartments(_ Request, _ Object) (Object, error) {
	return Object{"friends": []Object{}}, nil
}

func (a *Service) LeaveApartment(_ Request, _ Object) (Object, error) {
	return Object{"result": "success"}, nil
}

func (a *Service) UpdateApartmentStatus(_ Request, _ Object) (Object, error) {
	return Object{"result": "success"}, nil
}

func (a *Service) Reserve(_ Request, _ Object) (Object, error) {
	return Object{"status": true, "msg": "OK", "timestamp": now(), "hmac": strings.Repeat("0", 64), "bench_id": 5, "seat_id": 1, "room_ip": a.cfg.RoomHost, "room_port": a.cfg.RoomPort, "platform_filter": 0, "event_id": 8, "base_event_id": 8, "subevent_id": 0}, nil
}
