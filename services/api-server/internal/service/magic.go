package service

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

var magicEventMinimum = map[string][3]int{"ShoutBuffEvent": {0, 11, 0}, "SocialMessageBoats": {0, 11, 0}, "TeleportBuffEvent": {0, 11, 0}, "BirthdayHatBuffEvent": {0, 19, 5}, "DropWingConsumableEvent": {0, 19, 5}, "RecordingCandleEvent": {0, 19, 5}, "SharedSpaceCandleEvent": {0, 19, 5}, "SkyFigurineConsumableEvent": {0, 19, 5}, "TimelineBuffEvent": {0, 19, 5}, "ArtBookBuffEvent": {0, 33, 7}, "SocialMessageStones": {0, 33, 7}}
var magicEffectMinimum = map[string][3]int{"height_mult": {0, 10, 1}, "outfit_body": {0, 11, 0}, "outfit_hair": {0, 11, 0}, "outfit_mask": {0, 11, 0}, "outfit_neck": {0, 11, 0}, "outfit_prop": {0, 11, 0}, "outfit_wing": {0, 11, 0}, "relationship_allow": {0, 11, 0}, "outfit_horn": {0, 12, 2}, "relationship_flair": {0, 12, 3}, "trail_rainbow": {0, 13, 4}, "candle_trick": {0, 19, 5}, "cat_curse": {0, 19, 5}, "crab_rock_trick": {0, 19, 5}, "placeable": {0, 19, 5}, "repellant_krill": {0, 19, 5}, "dnd": {0, 22, 0}, "birthdayhat": {0, 33, 7}, "crab_curse": {0, 33, 7}, "currency_trick": {0, 33, 7}, "dragon_dance": {0, 33, 7}, "dye": {0, 33, 7}, "emitter_follow": {0, 33, 7}, "glow_rainbow": {0, 33, 7}}

func magicVersionAllowed(ua string, d Object, buff bool) bool {
	major, minor, patch, known := clientVersion(ua)
	if !known {
		return true
	}
	v := major*1000000 + minor*1000 + patch
	if min, ok := magicEventMinimum[text(d, "event")]; ok && v < min[0]*1000000+min[1]*1000+min[2] {
		return false
	}
	if buff && v < 17000 {
		if min, ok := magicEffectMinimum[text(d, "effect")]; ok && v < min[0]*1000000+min[1]*1000+min[2] {
			return false
		}
	}
	return true
}

func (a *Service) versionMagicDefinitions(key, filename, ua string) []Object {
	defs := a.magicDefinitions(key, filename)
	buff := key == "get_buff_defs"
	out := make([]Object, 0, len(defs))
	allowedBuff := map[int64]bool{}
	if !buff {
		for _, d := range a.magicDefinitions("get_buff_defs", "account__buff__get_buff_defs.json") {
			allowedBuff[defInt(d["id"])] = magicVersionAllowed(ua, d, true)
		}
	}
	for _, d := range defs {
		if !magicVersionAllowed(ua, d, buff) {
			continue
		}
		if !buff {
			bid := defInt(d["buff_id"])
			if allowed, known := allowedBuff[bid]; bid != 0 && known && !allowed {
				continue
			}
		}
		out = append(out, d)
	}
	return out
}
func (a *Service) buffIDAllowed(ua string, id int64) bool {
	for _, d := range a.magicDefinitions("get_buff_defs", "account__buff__get_buff_defs.json") {
		if defInt(d["id"]) == id {
			return magicVersionAllowed(ua, d, true)
		}
	}
	return true
}
func isUGCMagic(d Object) bool {
	name, event := text(d, "name"), text(d, "event")
	return text(d, "subcategory") == "UGC" || event == "SocialMessageBoats" || event == "SocialMessageStones" || event == "RecordingCandleEvent" || event == "SharedSpaceCandleEvent" || name == "memory_candle" || name == "sharedspace_candle" || strings.HasPrefix(name, "message_") || strings.HasPrefix(name, "personality_message_")
}

func (a *Service) magicDefinitions(key, filename string) []Object {
	var raw []byte
	if a.cfg.GameConfigDir != "" {
		path := filepath.Join(filepath.Dir(a.cfg.GameConfigDir), "resources", "static_responses", filename)
		raw, _ = os.ReadFile(path)
	}
	if len(raw) == 0 {
		raw, _ = legacyData.ReadFile("legacydata/" + filename)
	}
	var envelope map[string][]Object
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&envelope) != nil {
		return []Object{}
	}
	defs := envelope[key]
	if defs == nil {
		return []Object{}
	}
	return defs
}

func defInt(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case json.Number:
		v, _ = n.Int64()
	case string:
		parsed, _ := strconv.ParseInt(n, 10, 64)
		return parsed
	}
	if n, ok := v.(int64); ok {
		return n
	}
	return 0
}
func defSeconds(v any) int64 {
	var n float64
	switch x := v.(type) {
	case float64:
		n = x
	case json.Number:
		n, _ = x.Float64()
	case int64:
		n = float64(x)
	}
	if math.IsNaN(n) || math.IsInf(n, 0) || n < 0 {
		return 0
	}
	return int64(n * 60)
}

func (a *Service) ConsumableDefinitions(r Request, m Object) (Object, error) {
	defs := a.versionMagicDefinitions("get_consumable_defs", "account__consumable__get_consumable_defs.json", r.UserAgent)
	return Object{"get_consumable_defs": a.boundMagicDefinitions(r, m, "get_consumable_defs", defs, 86195)}, nil
}
func (a *Service) BuffDefinitions(r Request, m Object) (Object, error) {
	defs := a.versionMagicDefinitions("get_buff_defs", "account__buff__get_buff_defs.json", r.UserAgent)
	return Object{"get_buff_defs": a.boundMagicDefinitions(r, m, "get_buff_defs", defs, 86016)}, nil
}

func (a *Service) boundMagicDefinitions(r Request, m Object, key string, defs []Object, limit int) []Object {
	major, minor, _, known := clientVersion(r.UserAgent)
	if known && (major > 0 || minor >= 26) {
		return defs
	}
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	priority := map[int64]bool{}
	if key == "get_consumable_defs" {
		for _, d := range defs {
			if isUGCMagic(d) {
				priority[defInt(d["id"])] = true
			}
		}
	}
	if uid != "" {
		if key == "get_consumable_defs" {
			rows, err := a.store.QueryContext(r.Context, repository.ReadConsumables, uid)
			if err == nil {
				for rows.Next() {
					var id, cooldown, qty int64
					if rows.Scan(&id, &cooldown, &qty) == nil && qty > 0 {
						priority[id] = true
					}
				}
				rows.Close()
			}
		} else {
			consumables, err := a.store.QueryContext(r.Context, repository.ReadConsumables, uid)
			if err == nil {
				owned := map[int64]bool{}
				for consumables.Next() {
					var id, cooldown, quantity int64
					if consumables.Scan(&id, &cooldown, &quantity) == nil && quantity > 0 {
						owned[id] = true
					}
				}
				consumables.Close()
				for _, d := range a.versionMagicDefinitions("get_consumable_defs", "account__consumable__get_consumable_defs.json", r.UserAgent) {
					if owned[defInt(d["id"])] {
						priority[defInt(d["buff_id"])] = true
					}
				}
			}
			rows, err := a.store.QueryContext(r.Context, repository.ReadBuffs, uid)
			if err == nil {
				for rows.Next() {
					var id, giver string
					var active, cooldown int64
					if rows.Scan(&id, &active, &cooldown, &giver) == nil {
						priority[defInt(id)] = true
					}
				}
				rows.Close()
			}
		}
	}
	ordered := make([]Object, 0, len(defs))
	for _, d := range defs {
		if priority[defInt(d["id"])] {
			ordered = append(ordered, d)
		}
	}
	for _, d := range defs {
		if !priority[defInt(d["id"])] {
			ordered = append(ordered, d)
		}
	}
	out := []Object{}
	for _, d := range ordered {
		candidate := d
		if key == "get_consumable_defs" {
			candidate = Object{}
			for _, field := range []string{"id", "name", "buff_id", "cooldown", "event", "icon", "is_toy", "param", "type"} {
				if value, ok := d[field]; ok {
					candidate[field] = value
				}
			}
		}
		next := append(out, candidate)
		encoded, _ := json.Marshal(Object{key: next})
		if len(encoded) > limit {
			break
		}
		out = next
	}
	return out
}

func (a *Service) magicUser(r Request, m Object, strict bool) (string, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	if uid == "" || strict && text(m, "session") == "" {
		return "", fail(400, "Invalid request body")
	}
	if session := text(m, "session"); session != "" {
		u, err := a.findUser(r.Context, "id", uid)
		if errors.Is(err, sql.ErrNoRows) || err == nil && stringValue(u.Session) != session {
			return "", fail(401, "Invalid session or user not found")
		}
		if err != nil {
			return "", err
		}
	}
	return uid, nil
}

func (a *Service) consumables(r Request, uid string, includeID int64) ([]Object, error) {
	defs := a.versionMagicDefinitions("get_consumable_defs", "account__consumable__get_consumable_defs.json", r.UserAgent)
	known := map[int64]Object{}
	for _, d := range defs {
		known[defInt(d["id"])] = d
	}
	rows, err := a.store.QueryContext(r.Context, repository.ReadConsumables, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Object{}
	for rows.Next() {
		var id, cooldown, quantity int64
		if err = rows.Scan(&id, &cooldown, &quantity); err != nil {
			return nil, err
		}
		d, ok := known[id]
		if !ok {
			continue
		}
		if isUGCMagic(d) {
			if quantity < 99 {
				quantity = 99
			}
		} else if quantity <= 0 && id != includeID {
			continue
		}
		out = append(out, Object{"consumable_id": id, "cooldown_until": cooldown, "quantity": quantity})
	}
	return out, rows.Err()
}
func (a *Service) GetConsumables(r Request, m Object) (Object, error) {
	uid, err := a.magicUser(r, m, false)
	if err != nil {
		return nil, err
	}
	items, err := a.consumables(r, uid, 0)
	return Object{"get_consumables": items}, err
}
func (a *Service) GetConsumables1(r Request, m Object) (Object, error) {
	uid, err := a.magicUser(r, m, true)
	if err != nil {
		return nil, err
	}
	items, err := a.consumables(r, uid, 0)
	return Object{"get_consumables": items}, err
}

func (a *Service) buffs(r Request, uid string) (Object, error) {
	rows, err := a.store.QueryContext(r.Context, repository.ReadBuffs, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Object{}
	for rows.Next() {
		var id, giver string
		var active, cooldown int64
		if err = rows.Scan(&id, &active, &cooldown, &giver); err != nil {
			return nil, err
		}
		if active > 0 && active <= now() {
			continue
		}
		if !a.buffIDAllowed(r.UserAgent, defInt(id)) {
			continue
		}
		out = append(out, Object{"active_until": active, "buff_id": defInt(id), "cooldown_until": cooldown, "giver_user_id": giver})
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return defInt(out[i]["buff_id"]) < defInt(out[j]["buff_id"]) })
	encoded, _ := json.Marshal(out)
	sum := sha256.Sum256(append(encoded, []byte(uid)...))
	return Object{"get_buffs": out, "get_buffs_sign": hex.EncodeToString(sum[:])}, nil
}
func (a *Service) GetBuffs(r Request, m Object) (Object, error) {
	uid, err := a.magicUser(r, m, false)
	if err != nil {
		return nil, err
	}
	return a.buffs(r, uid)
}
func (a *Service) GetBuffs1(r Request, m Object) (Object, error) {
	uid, err := a.magicUser(r, m, true)
	if err != nil {
		return nil, err
	}
	return a.buffs(r, uid)
}
func (a *Service) RemoveBuff(r Request, m Object) (Object, error) {
	uid, err := a.magicUser(r, m, true)
	if err != nil {
		return nil, err
	}
	id := text(m, "buff_id")
	if id == "" {
		id = text(m, "id")
	}
	if id == "" {
		id = text(m, "buff_name")
	}
	if id != "" {
		if _, err = a.store.ExecContext(r.Context, repository.DeleteBuff, uid, id); err != nil {
			return nil, err
		}
	}
	result, err := a.buffs(r, uid)
	if err != nil {
		return nil, err
	}
	result["remove_buff"] = "success"
	return result, nil
}
func (a *Service) ConsumeItem(r Request, m Object) (Object, error) {
	uid, err := a.magicUser(r, m, true)
	if err != nil {
		return nil, err
	}
	defs := a.versionMagicDefinitions("get_consumable_defs", "account__consumable__get_consumable_defs.json", r.UserAgent)
	var definition Object
	for _, candidate := range []any{m["consumable_id"], m["consumable_name"], m["item_name"], m["name"]} {
		for _, d := range defs {
			if fmt.Sprint(d["id"]) == fmt.Sprint(candidate) || text(d, "name") == fmt.Sprint(candidate) {
				definition = d
				break
			}
		}
		if definition != nil {
			break
		}
	}
	status := "failed"
	var definitionID int64
	if definition != nil {
		id := defInt(definition["id"])
		definitionID = id
		count := number(m, "consumable_count", 1)
		if count <= 0 {
			count = 1
		}
		tx, e := a.store.BeginTx(r.Context, nil)
		if e != nil {
			return nil, e
		}
		defer tx.Rollback()
		query := repository.ConsumeItem
		if isUGCMagic(definition) {
			query = repository.ConsumeUGCItem
		}
		args := []any{count, now() + defSeconds(definition["cooldown"]), uid, id}
		if query == repository.ConsumeItem {
			args = append(args, count)
		}
		res, e := tx.ExecContext(r.Context, query, args...)
		if e != nil {
			return nil, e
		}
		changed, e := res.RowsAffected()
		if e != nil {
			return nil, e
		}
		if changed > 0 {
			status = "success"
			buffID := defInt(definition["buff_id"])
			if buffID != 0 {
				stamp := now()
				buffDefs := a.magicDefinitions("get_buff_defs", "account__buff__get_buff_defs.json")
				active, cooldown := int64(1200), int64(0)
				for _, bd := range buffDefs {
					if defInt(bd["id"]) == buffID {
						if duration := defSeconds(bd["duration"]); duration > 0 {
							active = duration
						}
						cooldown = defSeconds(bd["cooldown"])
						break
					}
				}
				_, e = tx.ExecContext(r.Context, repository.UpsertBuff, uid, fmt.Sprint(buffID), stamp+active, stamp+cooldown, uid)
				if e != nil {
					return nil, e
				}
			} else if text(definition, "name") == "resize_potion" {
				n, e := rand.Int(rand.Reader, big.NewInt(300001))
				if e != nil {
					return nil, e
				}
				height := 0.85 + float64(n.Int64())/1000000
				if _, e = tx.ExecContext(r.Context, repository.SetOutfitHeight, height, uid); e != nil {
					return nil, e
				}
			}
		}
		if e = tx.Commit(); e != nil {
			return nil, e
		}
	}
	b, err := a.buffs(r, uid)
	if err != nil {
		return nil, err
	}
	items, err := a.consumables(r, uid, definitionID)
	if err != nil {
		return nil, err
	}
	b["consume_item"] = status
	b["get_consumables"] = items
	b["shared_results"] = []Object{}
	height := 1.0
	if err = a.store.QueryRowContext(r.Context, repository.ReadOutfitHeight, uid).Scan(&height); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	b["new_height"] = height
	return b, nil
}
