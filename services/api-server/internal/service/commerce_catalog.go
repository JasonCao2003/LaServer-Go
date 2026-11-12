package service

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
)

func (a *Service) catalog(filename, fallback, key string) []Object {
	if a.cfg.GameConfigDir == "" {
		return nil
	}
	path := filepath.Join(filepath.Dir(a.cfg.GameConfigDir), "resources", "static_responses", filename)
	raw, err := os.ReadFile(path)
	if err != nil && fallback != "" {
		raw, err = os.ReadFile(filepath.Join(a.cfg.GameConfigDir, fallback))
	}
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
func (a *Service) spiritNode(name string) Object {
	for _, node := range a.catalog("account__get_spirit_shops.json", "spirit_shops.json", "spirit_shops") {
		if text(node, "nm") == name {
			return node
		}
	}
	return nil
}
func (a *Service) genericShopDef(m Object) Object {
	defs := a.catalog("account__get_generic_shops.json", "generic_shops.json", "generic_shop_defs")
	for _, key := range []string{"id", "item_id", "shop_item_id", "generic_shop_item_id", "unlock_id"} {
		if id := number(m, key, 0); id != 0 {
			for _, d := range defs {
				if defInt(d["id"]) == id {
					return d
				}
			}
		}
	}
	for _, key := range []string{"name", "shop_item_name", "def_name", "item_name", "consumable_name"} {
		if name := text(m, key); name != "" {
			for _, d := range defs {
				if text(d, "name") == name || text(d, "item_name") == name {
					return d
				}
			}
		}
	}
	return nil
}
