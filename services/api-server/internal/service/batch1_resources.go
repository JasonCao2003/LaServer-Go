package service

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

func (a *Service) CurrencyDefinitions(_ Request, _ Object) (Object, error) {
	return Object{"currency_defs": []any{}}, nil
}

func (a *Service) TrustTierDefinitions(_ Request, _ Object) (Object, error) {
	return Object{"get_trust_tier_defs": []any{}}, nil
}

func (a *Service) RadianceSpawnDefinitions(_ Request, _ Object) (Object, error) {
	return a.configuredObject("radiance_spawn.json")
}

func (a *Service) batch1User(r Request, m Object) (string, error) {
	userID, session := text(m, "user"), text(m, "session")
	if userID == "" || session == "" {
		return "", fail(400, "Invalid request body")
	}
	u, err := a.findUser(r.Context, "id", userID)
	if errors.Is(err, sql.ErrNoRows) || err == nil && stringValue(u.Session) != session {
		return "", fail(401, "Invalid session or user not found")
	}
	if err != nil {
		return "", err
	}
	return u.ID, nil
}

func (a *Service) RemoteConfigUpload(r Request, m Object) (Object, error) {
	if len(m) == 0 {
		return nil, &Error{Status: 400, Body: Object{"result": "error", "message": "请求体不能为空"}}
	}
	for _, key := range []string{"user", "session"} {
		if _, ok := m[key]; !ok {
			return nil, &Error{Status: 400, Body: Object{"result": "error", "message": "缺少必需参数: " + key}}
		}
	}
	u, err := a.findUser(r.Context, "id", text(m, "user"))
	if errors.Is(err, sql.ErrNoRows) || err == nil && stringValue(u.Session) != text(m, "session") {
		return nil, &Error{Status: 401, Body: Object{"result": "error", "message": "无效的会话或用户不存在"}}
	}
	if err != nil {
		return nil, err
	}
	if integer(u.IsBanned) == 1 && (integer(u.BanEndTime) == 0 || now() < integer(u.BanEndTime)) {
		return nil, &Error{Status: 403, Body: Object{"result": "banned", "message": "账号已被封禁"}}
	}
	// The legacy handler only acknowledged the request; it did not persist a file.
	return Object{"result": "ok"}, nil
}

func (a *Service) GenericShopTracking(r Request, m Object) (Object, error) {
	if _, err := a.batch1User(r, m); err != nil {
		return nil, err
	}
	if a.cfg.GameConfigDir == "" {
		return Object{"generic_shop_item_tracking_info": []any{}}, nil
	}
	raw, err := os.ReadFile(filepath.Join(a.cfg.GameConfigDir, "generic_shops.json"))
	if errors.Is(err, os.ErrNotExist) {
		return Object{"generic_shop_item_tracking_info": []any{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var source any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err = decoder.Decode(&source); err != nil {
		return nil, fail(503, "Invalid game configuration")
	}
	var shops []any
	if container, ok := source.(map[string]any); ok {
		shops, _ = container["generic_shop_defs"].([]any)
	} else {
		shops, _ = source.([]any)
	}
	info := []Object{}
	for _, value := range shops {
		shop, ok := value.(map[string]any)
		if !ok {
			continue
		}
		name := text(shop, "name")
		if name == "" {
			name = text(shop, "shop_name")
		}
		if requested := text(m, "shop_name"); requested != "" && requested != name {
			continue
		}
		items, _ := shop["items"].([]any)
		for _, value := range items {
			item, ok := value.(map[string]any)
			if !ok {
				continue
			}
			id := number(item, "id", 0)
			if id == 0 {
				id = number(item, "unlockid", 0)
			}
			if id == 0 {
				id = number(item, "item_id", 0)
			}
			if id == 0 {
				continue
			}
			info = append(info, Object{"id": id, "count": number(item, "count", 1), "ttl": number(item, "ttl", 1645257)})
		}
	}
	return Object{"generic_shop_item_tracking_info": info}, nil
}

func (a *Service) LevelPickups(r Request, m Object) (Object, error) {
	if _, ok := m["level"]; !ok {
		return nil, fail(400, "Invalid request body")
	}
	userID, err := a.batch1User(r, m)
	if err != nil {
		return nil, err
	}
	var defaults []repository.PickupEmitter
	if err = legacyResponse("level_pickup_emitters", &defaults); err != nil {
		return nil, err
	}
	level := number(m, "level", 0)
	emitters, err := a.store.LevelPickupEmitters(r.Context, userID, level, defaults)
	if err != nil {
		return nil, err
	}
	return Object{"level": m["level"], "pickups": []any{}, "global_pickups": []any{}, "pickup_emitters": emitters}, nil
}
