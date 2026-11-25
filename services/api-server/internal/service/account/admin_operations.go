package account

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var operationsWrite sync.Mutex

func (a *Service) AdminOperations(r Request, m object) (object, error) {
	token := a.Cfg.AdminToken
	if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(r.Header.Get("X-Admin-Token"))) != 1 {
		return nil, &Error{Status: 403, Body: object{"error": "forbidden"}}
	}
	section := text(m, "section")
	switch section {
	case "season", "tcandles", "events", "daily_quests", "season_quests", "spirit_shops":
	default:
		return nil, &Error{Status: 400, Body: object{"error": "invalid section"}}
	}
	operationsWrite.Lock()
	defer operationsWrite.Unlock()
	cfg, err := a.LegacyObject("game_control.json")
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		cfg = object{}
	}
	value := cfg[section]
	if value == nil {
		value = object{}
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	revision := fmt.Sprintf("%x", sha256.Sum256(raw))
	switch text(m, "action") {
	case "read":
		return object{"ok": true, "value": string(raw), "revision": revision}, nil
	case "save":
	default:
		return nil, &Error{Status: 400, Body: object{"error": "invalid action"}}
	}
	if text(m, "revision") != revision {
		return nil, &Error{Status: http.StatusConflict, Body: object{"error": "configuration changed"}}
	}
	input := text(m, "value")
	var next object
	decoder := json.NewDecoder(strings.NewReader(input))
	decoder.UseNumber()
	if len(input) > 256<<10 || !json.Valid([]byte(input)) || decoder.Decode(&next) != nil || next == nil {
		return nil, &Error{Status: 400, Body: object{"error": "invalid object"}}
	}
	cfg[section] = next
	dir := a.Cfg.InstanceDir
	if dir == "" {
		return nil, fmt.Errorf("instance directory missing")
	}
	if err = os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	f, err := os.CreateTemp(dir, "operations-*.tmp")
	if err != nil {
		return nil, err
	}
	defer os.Remove(f.Name())
	err = json.NewEncoder(f).Encode(cfg)
	closeErr := f.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if err = os.Rename(f.Name(), filepath.Join(dir, "game_control.json")); err != nil {
		return nil, err
	}
	return object{"ok": true}, nil
}
