package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

func (a *Service) statusConfig(filename string, key string) []Object {
	if a.cfg.GameConfigDir == "" {
		return nil
	}
	raw, err := os.ReadFile(filepath.Join(a.cfg.GameConfigDir, filename))
	if err != nil {
		return nil
	}
	var envelope map[string][]Object
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&envelope) == nil {
		return envelope[key]
	}
	var list []Object
	decoder = json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&list) == nil {
		return list
	}
	return nil
}
func (a *Service) StatusUnlocks(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	base := a.statusConfig("all_unlock_status.json", "status_unlocks")
	merged := map[string]Object{}
	order := []string{}
	add := func(item Object, replace bool) {
		name := text(item, "name")
		if name == "" {
			return
		}
		if _, ok := merged[name]; !ok {
			order = append(order, name)
		} else if !replace {
			return
		}
		merged[name] = item
	}
	if uid != "" && text(m, "session") != "" {
		var err error
		uid, err = a.magicUser(r, m, true)
		if err != nil {
			return nil, err
		}
		u, err := a.findUser(r.Context, "id", uid)
		if err != nil {
			return nil, err
		}
		full, err := a.policy("auto_full_unlock.json", integer(u.CreatedAt), false)
		if err != nil {
			return nil, err
		}
		if full {
			for _, v := range a.statusConfig("kaiquantu.json", "status_unlocks") {
				add(v, true)
			}
		}
	}
	for _, v := range base {
		add(v, false)
	}
	if uid != "" && text(m, "session") != "" {
		rows, err := a.store.QueryContext(r.Context, repository.ReadStatusUnlocks, uid)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var name, kind string
			var ack bool
			var stamp int64
			if err = rows.Scan(&name, &kind, &ack, &stamp); err != nil {
				rows.Close()
				return nil, err
			}
			add(Object{"name": name, "type": kind, "ack": ack, "unlocked_at": stamp}, true)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
		deleted, err := a.store.QueryContext(r.Context, repository.ReadStatusTombstones, uid)
		if err != nil {
			return nil, err
		}
		for deleted.Next() {
			var name string
			if err = deleted.Scan(&name); err != nil {
				deleted.Close()
				return nil, err
			}
			delete(merged, name)
		}
		err = deleted.Err()
		deleted.Close()
		if err != nil {
			return nil, err
		}
	}
	out := []Object{}
	for _, name := range order {
		if v, ok := merged[name]; ok {
			out = append(out, v)
		}
	}
	// Shared-space props are available to clients even before authentication.
	for _, entry := range a.mergeFullUnlocks(nil, r.UserAgent) {
		name := entry.Name
		if !strings.HasPrefix(name, "CharSkyKid_") || !strings.Contains(name, "_Prop_") {
			continue
		}
		if _, ok := merged[name]; ok {
			continue
		}
		merged[name] = Object{"name": name}
		out = append(out, Object{"name": name, "type": "level", "ack": true, "unlocked_at": entry.CreatedAt})
	}
	return Object{"status_unlocks": out, "status_unlocks_total_count": len(out)}, nil
}
func (a *Service) AddStatusUnlock(r Request, m Object) (Object, error) {
	if err := require(m, "user", "session", "name", "ack"); err != nil {
		return nil, err
	}
	uid, err := a.magicUser(r, m, true)
	if err != nil {
		return nil, err
	}
	name := text(m, "name")
	if name == "" {
		return nil, fail(400, "Invalid name")
	}
	ack, ok := m["ack"].(bool)
	if !ok {
		return nil, fail(400, "Invalid ack")
	}
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	stamp := now()
	res, err := tx.ExecContext(r.Context, repository.AddStatusUnlock, uid, name, ack, stamp)
	if err != nil {
		return nil, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	response := Object{"result": "ok"}
	if affected == 0 {
		response["status"] = "already"
		var kind string
		if err = tx.QueryRowContext(r.Context, repository.ReadStatusUnlock, uid, name).Scan(&kind, &ack, &stamp); err != nil {
			return nil, err
		}
		response["update_status_unlocks"] = []Object{{"name": name, "type": kind, "ack": ack, "unlocked_at": stamp}}
	} else {
		response["update_status_unlocks"] = []Object{{"name": name, "type": "level", "ack": ack, "unlocked_at": stamp}}
	}
	if _, err = tx.ExecContext(r.Context, repository.ClearStatusTombstone, uid, name); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return response, nil
}
func (a *Service) DeleteStatusUnlock(r Request, m Object) (Object, error) {
	if err := require(m, "user", "session", "name"); err != nil {
		return nil, err
	}
	uid, err := a.magicUser(r, m, true)
	if err != nil {
		return nil, err
	}
	name := text(m, "name")
	if name == "" {
		return nil, fail(400, "Invalid name")
	}
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(r.Context, repository.DeleteStatusUnlock, uid, name); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(r.Context, repository.AddStatusTombstone, uid, name, now()); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return Object{"result": "ok", "delete_status_unlocks": []string{name}}, nil
}
func (a *Service) RemoveDye(r Request, m Object) (Object, error) {
	uid, err := a.magicUser(r, m, true)
	if err != nil {
		var e *Error
		if errors.As(err, &e) {
			return nil, &Error{Status: e.Status, Body: Object{"status": "ERROR", "result": "error"}}
		}
		return nil, err
	}
	if name := text(m, "unlock_name"); name != "" {
		if _, err = a.store.ExecContext(r.Context, repository.DeleteItemDye, uid, name); err != nil {
			return nil, err
		}
	}
	return Object{"status": "OK", "result": "ok"}, nil
}

func (a *Service) CollectCollectible(r Request, m Object) (Object, error) {
	uid, err := a.magicUser(r, m, true)
	if err != nil {
		return nil, err
	}
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var count int64
	if err = tx.QueryRowContext(r.Context, repository.CountCollectibles, uid).Scan(&count); err != nil {
		return nil, err
	}
	if count == 0 {
		for _, d := range a.statusConfig("collectibles.json", "collectibles") {
			id := text(d, "id")
			if id == "" {
				continue
			}
			if _, err = tx.ExecContext(r.Context, repository.SeedCollectible, uid, id, d["used"], d["candle_space"], d["carrying"], number(d, "level", 0), number(d, "permanent_level", 0)); err != nil {
				return nil, err
			}
		}
	}
	if name := text(m, "name"); name != "" {
		carrying, _ := m["carrying"].(bool)
		if _, err = tx.ExecContext(r.Context, repository.CollectCollectible, uid, name, carrying); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	rows, err := a.store.QueryContext(r.Context, repository.ReadCollectibles, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Object{}
	for rows.Next() {
		var id string
		var used, space, carrying bool
		var level, permanent int64
		if err = rows.Scan(&id, &used, &space, &carrying, &level, &permanent); err != nil {
			return nil, err
		}
		out = append(out, Object{"id": id, "used": used, "candle_space": space, "carrying": carrying, "level": level, "permanent_level": permanent})
	}
	return Object{"collectibles": out}, rows.Err()
}
func (a *Service) AckCollectible(_ Request, _ Object) (Object, error) { return Object{}, nil }
