// 静态资源与配置读取

package core

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// 读取静态资源目录中的列表
func (a *Base) Catalog(filename, fallback, key string) []Object {
	if a.Cfg.GameConfigDir == "" {
		return nil
	}
	path := filepath.Join(filepath.Dir(a.Cfg.GameConfigDir), "resources", "static_responses", filename)
	raw, err := os.ReadFile(path)
	if err != nil && fallback != "" {
		raw, err = os.ReadFile(filepath.Join(a.Cfg.GameConfigDir, fallback))
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

// 读取配置对象
func (a *Base) ConfiguredObject(name string) (Object, error) {
	if a.Cfg.GameConfigDir == "" {
		return nil, Fail(503, "Game configuration unavailable")
	}
	b, err := os.ReadFile(filepath.Join(a.Cfg.GameConfigDir, name))
	if err != nil {
		return nil, Fail(503, "Game configuration unavailable")
	}
	b = bytes.TrimPrefix(b, []byte{0xef, 0xbb, 0xbf})
	var result Object
	if err = json.Unmarshal(b, &result); err != nil || result == nil {
		return nil, Fail(503, "Invalid game configuration")
	}
	return result, nil
}

// 读取状态配置列表
func (a *Base) StatusConfig(filename string, key string) []Object {
	if a.Cfg.GameConfigDir == "" {
		return nil
	}
	raw, err := os.ReadFile(filepath.Join(a.Cfg.GameConfigDir, filename))
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

// 解析表情动作系列
func emoteFamily(name string) (string, bool) {
	if !strings.HasPrefix(name, "emote_upgrade_") {
		return "", false
	}
	cut := strings.LastIndex(name, "_")
	if cut <= len("emote_upgrade_") {
		return "", false
	}
	if _, err := strconv.Atoi(name[cut+1:]); err != nil {
		return "", false
	}
	return name[:cut], true
}

// 查找精灵节点
func (a *Base) SpiritNode(name string) Object {
	for _, node := range a.Catalog("account__get_spirit_shops.json", "spirit_shops.json", "spirit_shops") {
		if Text(node, "nm") == name {
			return node
		}
	}
	return nil
}

// 查找通用商店定义
func (a *Base) GenericShopDef(m Object) Object {
	defs := a.Catalog("account__get_generic_shops.json", "generic_shops.json", "generic_shop_defs")
	for _, key := range []string{"id", "item_id", "shop_item_id", "generic_shop_item_id", "unlock_id"} {
		if id := Number(m, key, 0); id != 0 {
			for _, d := range defs {
				if DefInt(d["id"]) == id {
					return d
				}
			}
		}
	}
	for _, key := range []string{"name", "shop_item_name", "def_name", "item_name", "consumable_name"} {
		if name := Text(m, key); name != "" {
			for _, d := range defs {
				if Text(d, "name") == name || Text(d, "item_name") == name {
					return d
				}
			}
		}
	}
	return nil
}

// 获取表情动作升级系列
func (a *Base) EmoteFamilyLevels(name string) []string {
	family, ok := emoteFamily(name)
	if !ok {
		return nil
	}
	type level struct {
		name   string
		number int
	}
	levels := []level{}
	for _, node := range a.Catalog("account__get_spirit_shops.json", "spirit_shops.json", "spirit_shops") {
		candidate := Text(node, "nm")
		prefix, match := emoteFamily(candidate)
		if !match || prefix != family || Text(node, "typ") != "spirit_upgrade" {
			continue
		}
		cut := strings.LastIndex(candidate, "_")
		n, _ := strconv.Atoi(candidate[cut+1:])
		levels = append(levels, level{candidate, n})
	}
	sort.Slice(levels, func(i, j int) bool { return levels[i].number < levels[j].number })
	out := make([]string, 0, len(levels))
	for _, v := range levels {
		out = append(out, v.name)
	}
	return out
}
