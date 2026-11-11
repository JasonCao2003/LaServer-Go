package service

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

const (
	defaultUnlockTime = int64(1654570099)
	legacyUnlockLimit = 142952
	modernUnlockLimit = 262144
)

var emoteUnlockPattern = regexp.MustCompile(`^emote_upgrade_([a-zA-Z0-9]+)_([0-9]+)$`)

func (a *Service) defaultUnlockNames() map[string]bool {
	allowed := map[string]bool{"CharSkyKid_Prop_None": true}
	if a.cfg.GameConfigDir == "" {
		return allowed
	}
	raw, err := os.ReadFile(filepath.Join(a.cfg.GameConfigDir, "spirit_shops.json"))
	if err != nil {
		return allowed
	}
	var data struct {
		Shops []struct {
			Name string `json:"nm"`
			Type string `json:"typ"`
		} `json:"spirit_shops"`
	}
	if json.Unmarshal(raw, &data) != nil {
		return allowed
	}
	for _, item := range data.Shops {
		if item.Type != "spirit_upgrade" {
			continue
		}
		match := emoteUnlockPattern.FindStringSubmatch(item.Name)
		if len(match) == 3 && match[1] == "wave" {
			allowed[item.Name] = true
		}
	}
	return allowed
}

func isDefaultUnlock(name string, allowed map[string]bool) bool {
	if !allowed[name] {
		return false
	}
	return name == "CharSkyKid_Prop_None" || emoteUnlockPattern.MatchString(name)
}

func (a *Service) seedDefaultUnlocks(r Request, userID string, allowed map[string]bool) error {
	items := make([]repository.Unlock, 0, len(allowed))
	for name := range allowed {
		ack := name == "CharSkyKid_Prop_None"
		stamp := defaultUnlockTime
		if name == "emote_upgrade_wave_01" {
			stamp = 1654571824
		}
		items = append(items, repository.Unlock{Name: name, Type: "spiritshop", Ack: &ack, CreatedAt: stamp})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return a.store.SeedDefaultUnlocks(r.Context, userID, items)
}

func (a *Service) unlockConfigLimit(key string, fallback int) int {
	if a.cfg.GameConfigDir == "" {
		return fallback
	}
	raw, err := os.ReadFile(filepath.Join(a.cfg.GameConfigDir, "unlock_limits.json"))
	if err != nil {
		return fallback
	}
	var values map[string]json.Number
	if json.Unmarshal(raw, &values) != nil {
		return fallback
	}
	if n, err := values[key].Int64(); err == nil && n >= 4096 {
		return int(n)
	}
	return fallback
}

func unlockPriority(item repository.Unlock) int {
	name := item.Name
	switch {
	case len(name) >= len("emote_upgrade_") && name[:len("emote_upgrade_")] == "emote_upgrade_":
		return 0
	case bytes.Contains([]byte(name), []byte("_Wing_")) || bytes.Contains([]byte(name), []byte("_Cape")) || len(name) >= 4 && name[len(name)-4:] == "Wing":
		return 1
	case len(name) >= len("CharSkyKid_") && name[:len("CharSkyKid_")] == "CharSkyKid_":
		return 2
	case len(name) >= 4 && name[:4] == "Char" || len(name) >= 4 && name[:4] == "NPC_" || len(name) >= 5 && name[:5] == "Nest_":
		return 4
	case item.Type == "level":
		return 5
	default:
		return 3
	}
}

func (a *Service) finalizeUnlocks(items []repository.Unlock, ua string) []repository.Unlock {
	sort.SliceStable(items, func(i, j int) bool {
		pi, pj := unlockPriority(items[i]), unlockPriority(items[j])
		if pi != pj {
			return pi < pj
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
	limit := a.unlockConfigLimit("legacy_max_bytes", legacyUnlockLimit)
	major, minor, patch, known := clientVersion(ua)
	if known && (major > 0 || minor > 26 || minor == 26 && patch >= 0) {
		limit = a.unlockConfigLimit("modern_max_bytes", modernUnlockLimit)
	}
	for len(items) > 0 {
		response := Object{"unlocks": unlockPayload(items), "unlocks_total_count": len(items)}
		encoded, _ := json.Marshal(response)
		// Reserve two bytes per field/list separator for the legacy encoder's spaces.
		if len(encoded)+len(items)*2 <= limit {
			break
		}
		items = items[:len(items)-1]
	}
	return items
}

func unlockPayload(items []repository.Unlock) []Object {
	out := make([]Object, 0, len(items))
	for _, item := range items {
		out = append(out, Object{"name": item.Name, "type": item.Type, "ack": item.Ack, "created_at": item.CreatedAt})
	}
	return out
}

func (a *Service) maxCountUnlocks(r Request, userID string, allowed map[string]bool) (Object, error) {
	if a.cfg.GameConfigDir == "" {
		return nil, fail(404, "Unlock configuration not found")
	}
	raw, err := os.ReadFile(filepath.Join(a.cfg.GameConfigDir, "0.6.5_unlock.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || a.cfg.GameConfigDir == "" {
			return nil, fail(404, "Unlock configuration not found")
		}
		return nil, fail(500, "Failed to read unlock configuration")
	}
	var source any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&source) != nil {
		return nil, fail(500, "Invalid JSON in unlock configuration")
	}
	var configured []any
	switch value := source.(type) {
	case []any:
		configured = value
	case map[string]any:
		configured, _ = value["unlocks"].([]any)
	}
	items := make([]repository.Unlock, 0, len(configured))
	seen := map[string]bool{}
	for _, value := range configured {
		entry, ok := value.(map[string]any)
		if !ok {
			continue
		}
		name, _ := entry["name"].(string)
		if !isDefaultUnlock(name, allowed) || seen[name] {
			continue
		}
		seen[name] = true
		item := repository.Unlock{Name: name, Type: text(entry, "type"), CreatedAt: number(entry, "created_at", 0)}
		if ack, ok := entry["ack"].(bool); ok {
			item.Ack = &ack
		}
		items = append(items, item)
	}
	if userID != "" {
		rows, queryErr := a.store.Unlocks(r.Context, userID)
		if queryErr == nil {
			for _, item := range rows {
				if !seen[item.Name] {
					items = append(items, item)
					seen[item.Name] = true
				}
			}
		}
	}
	items = a.finalizeUnlocks(items, r.UserAgent)
	return Object{"unlocks": unlockPayload(items), "unlocks_total_count": len(items)}, nil
}

func (a *Service) Unlocks(r Request, m Object) (Object, error) {
	userID, session := text(m, "user"), text(m, "session")
	if userID == "" || session == "" {
		return nil, fail(400, "Invalid request body")
	}
	allowed := a.defaultUnlockNames()
	u, err := a.findUser(r.Context, "id", userID)
	if errors.Is(err, sql.ErrNoRows) || err == nil && stringValue(u.Session) != session {
		return nil, fail(401, "Invalid session or user not found")
	}
	if err != nil {
		return nil, err
	}
	if _, ok := m["max_count"]; ok {
		return a.maxCountUnlocks(r, userID, allowed)
	}
	if err = a.seedDefaultUnlocks(r, u.ID, allowed); err != nil {
		return nil, err
	}
	items, err := a.store.Unlocks(r.Context, u.ID)
	if err != nil {
		return nil, err
	}
	total := len(items)
	sort.SliceStable(items, func(i, j int) bool {
		pi, pj := unlockPriority(items[i]), unlockPriority(items[j])
		if pi != pj {
			return pi < pj
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
	paginated := false
	if _, paginated = m["l"]; paginated {
		offset, limit := int(number(m, "o", 0)), int(number(m, "l", 1000))
		if offset < 0 {
			offset = 0
		}
		if limit < 0 {
			limit = 0
		}
		if offset > len(items) {
			offset = len(items)
		}
		end := offset + limit
		if end > len(items) {
			end = len(items)
		}
		items = items[offset:end]
	} else {
		items = a.finalizeUnlocks(items, r.UserAgent)
		total = len(items)
	}
	return Object{"unlocks": unlockPayload(items), "unlocks_total_count": total}, nil
}

func (a *Service) AckUnlock(r Request, m Object) (Object, error) {
	for _, key := range []string{"user", "session", "name"} {
		if _, ok := m[key]; !ok {
			return Object{"result": "Missing required parameters: " + key}, nil
		}
	}
	u, err := a.findUser(r.Context, "id", text(m, "user"))
	if errors.Is(err, sql.ErrNoRows) || err == nil && stringValue(u.Session) != text(m, "session") {
		return Object{"result": "unauthorized"}, nil
	}
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(text(m, "name"))
	if name == "" {
		return Object{"result": "missing_parameters: name"}, nil
	}
	item, err := a.store.AckUnlock(r.Context, u.ID, name, now())
	if err != nil {
		return nil, err
	}
	return Object{"result": true, "update_unlocks": unlockPayload([]repository.Unlock{item})}, nil
}
