// 魔法/消耗品/增益共享逻辑

package core

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/legacydata"
)

var magicEventMinimum = map[string][3]int{"ShoutBuffEvent": {0, 11, 0}, "SocialMessageBoats": {0, 11, 0}, "TeleportBuffEvent": {0, 11, 0}, "BirthdayHatBuffEvent": {0, 19, 5}, "DropWingConsumableEvent": {0, 19, 5}, "RecordingCandleEvent": {0, 19, 5}, "SharedSpaceCandleEvent": {0, 19, 5}, "SkyFigurineConsumableEvent": {0, 19, 5}, "TimelineBuffEvent": {0, 19, 5}, "ArtBookBuffEvent": {0, 33, 7}, "SocialMessageStones": {0, 33, 7}}
var magicEffectMinimum = map[string][3]int{"height_mult": {0, 10, 1}, "outfit_body": {0, 11, 0}, "outfit_hair": {0, 11, 0}, "outfit_mask": {0, 11, 0}, "outfit_neck": {0, 11, 0}, "outfit_prop": {0, 11, 0}, "outfit_wing": {0, 11, 0}, "relationship_allow": {0, 11, 0}, "outfit_horn": {0, 12, 2}, "relationship_flair": {0, 12, 3}, "trail_rainbow": {0, 13, 4}, "candle_trick": {0, 19, 5}, "cat_curse": {0, 19, 5}, "crab_rock_trick": {0, 19, 5}, "placeable": {0, 19, 5}, "repellant_krill": {0, 19, 5}, "dnd": {0, 22, 0}, "birthdayhat": {0, 33, 7}, "crab_curse": {0, 33, 7}, "currency_trick": {0, 33, 7}, "dragon_dance": {0, 33, 7}, "dye": {0, 33, 7}, "emitter_follow": {0, 33, 7}, "glow_rainbow": {0, 33, 7}}

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

// 判断魔法定义是否版本可见
func magicVersionAllowed(ua string, d Object, buff bool) bool {
	major, minor, patch, known := ClientVersion(ua)
	if !known {
		return true
	}
	v := major*1000000 + minor*1000 + patch
	if min, ok := magicEventMinimum[Text(d, "event")]; ok && v < min[0]*1000000+min[1]*1000+min[2] {
		return false
	}
	if buff && v < 17000 {
		if min, ok := magicEffectMinimum[Text(d, "effect")]; ok && v < min[0]*1000000+min[1]*1000+min[2] {
			return false
		}
	}
	return true
}

// 判断增益是否版本可见
func (a *Base) BuffIDAllowed(ua string, id int64) bool {
	for _, d := range a.MagicDefinitions("get_buff_defs", "account__buff__get_buff_defs.json") {
		if DefInt(d["id"]) == id {
			return magicVersionAllowed(ua, d, true)
		}
	}
	return true
}

// 判断是否为 UGC 魔法
func IsUGCMagic(d Object) bool {
	name, event := Text(d, "name"), Text(d, "event")
	return Text(d, "subcategory") == "UGC" || event == "SocialMessageBoats" || event == "SocialMessageStones" || event == "RecordingCandleEvent" || event == "SharedSpaceCandleEvent" || name == "memory_candle" || name == "sharedspace_candle" || strings.HasPrefix(name, "message_") || strings.HasPrefix(name, "personality_message_")
}

// 读取魔法定义
func (a *Base) MagicDefinitions(key, filename string) []Object {
	decode := func(raw []byte) []Object {
		var envelope map[string][]Object
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		var extra any
		if decoder.Decode(&envelope) != nil || decoder.Decode(&extra) != io.EOF {
			return nil
		}
		defs := envelope[key]
		for _, def := range defs {
			if len(def) == 0 || DefInt(def["id"]) == 0 {
				return nil
			}
		}
		return defs
	}
	if a.Cfg.GameConfigDir != "" {
		path := filepath.Join(filepath.Dir(a.Cfg.GameConfigDir), "resources", "static_responses", filename)
		if raw, err := os.ReadFile(path); err == nil {
			if defs := decode(raw); defs != nil {
				return defs
			}
		}
	}
	raw, err := legacydata.Read(filename)
	if err != nil {
		return []Object{}
	}
	if defs := decode(raw); defs != nil {
		return defs
	}
	return []Object{}
}

// 读取按版本过滤的魔法定义
func (a *Base) VersionMagicDefinitions(key, filename, ua string) []Object {
	defs := a.MagicDefinitions(key, filename)
	buff := key == "get_buff_defs"
	out := make([]Object, 0, len(defs))
	allowedBuff := map[int64]bool{}
	if !buff {
		for _, d := range a.MagicDefinitions("get_buff_defs", "account__buff__get_buff_defs.json") {
			allowedBuff[DefInt(d["id"])] = magicVersionAllowed(ua, d, true)
		}
	}
	for _, d := range defs {
		if !magicVersionAllowed(ua, d, buff) {
			continue
		}
		if !buff {
			bid := DefInt(d["buff_id"])
			if allowed, known := allowedBuff[bid]; bid != 0 && known && !allowed {
				continue
			}
		}
		out = append(out, d)
	}
	return out
}

// 校验魔法请求用户
func (a *Base) MagicUser(r Request, m Object, strict bool) (string, error) {
	uid := Text(m, "user")
	if uid == "" {
		uid = Text(m, "user_id")
	}
	if uid == "" || strict && Text(m, "session") == "" {
		return "", Fail(400, "Invalid request body")
	}
	if session := Text(m, "session"); session != "" {
		u, err := a.FindUser(r.Context, "id", uid)
		if errors.Is(err, sql.ErrNoRows) || err == nil && StringValue(u.Session) != session {
			return "", Fail(401, "Invalid session or user not found")
		}
		if err != nil {
			return "", err
		}
	}
	return uid, nil
}

// 读取消耗品列表
func (a *Base) Consumables(r Request, uid string, includeID int64) ([]Object, error) {
	defs := a.VersionMagicDefinitions("get_consumable_defs", "account__consumable__get_consumable_defs.json", r.UserAgent)
	known := map[int64]Object{}
	for _, d := range defs {
		known[DefInt(d["id"])] = d
	}
	rows, err := a.Store.QueryContext(r.Context, repository.ReadConsumables, uid)
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
		if IsUGCMagic(d) {
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

// 读取增益列表
func (a *Base) Buffs(r Request, uid string) (Object, error) {
	rows, err := a.Store.QueryContext(r.Context, repository.ReadBuffs, uid)
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
		if active > 0 && active <= Now() {
			continue
		}
		if !a.BuffIDAllowed(r.UserAgent, DefInt(id)) {
			continue
		}
		out = append(out, Object{"active_until": active, "buff_id": DefInt(id), "cooldown_until": cooldown, "giver_user_id": giver})
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return DefInt(out[i]["buff_id"]) < DefInt(out[j]["buff_id"]) })
	encoded, _ := json.Marshal(out)
	sum := sha256.Sum256(append(encoded, []byte(uid)...))
	return Object{"get_buffs": out, "get_buffs_sign": hex.EncodeToString(sum[:])}, nil
}

// 读取锻造概率
func (a *Base) ForgeRates(r Request, m Object) (Object, error) {
	userID := Text(m, "user")
	if userID == "" {
		userID = Text(m, "user_id")
	}
	rows, err := a.Store.ForgeRates(r.Context, userID)
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
