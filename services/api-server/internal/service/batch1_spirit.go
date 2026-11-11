package service

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func spiritNodeID(node Object, key string) int64 { return number(node, key, 0) }

func spiritTreeLimit(ua string) int {
	major, minor, _, known := clientVersion(ua)
	if !known || major > 0 || minor >= 26 {
		return 0
	}
	if minor >= 20 {
		return 500
	}
	if minor >= 10 {
		return 250
	}
	return 120
}

func limitSpiritTree(nodes []Object, max int) []Object {
	if max <= 0 || len(nodes) <= max {
		return nodes
	}
	byID := map[int64]Object{}
	roots := []int64{}
	children := map[int64][]int64{}
	for _, node := range nodes {
		id := spiritNodeID(node, "id")
		dep := spiritNodeID(node, "dep")
		byID[id] = node
		if dep == 0 {
			roots = append(roots, id)
		} else {
			children[dep] = append(children[dep], id)
		}
	}
	owner := map[int64]int64{}
	for _, node := range nodes {
		id := spiritNodeID(node, "id")
		if _, ok := owner[id]; ok {
			continue
		}
		chain := []int64{}
		seen := map[int64]bool{}
		current := id
		root := int64(0)
		for current != 0 && !seen[current] {
			if known, ok := owner[current]; ok {
				root = known
				break
			}
			seen[current] = true
			chain = append(chain, current)
			parent, ok := byID[current]
			if !ok {
				break
			}
			dep := spiritNodeID(parent, "dep")
			if dep == 0 {
				root = current
				break
			}
			current = dep
		}
		for _, member := range chain {
			owner[member] = root
		}
	}
	keep := map[int64]bool{}
	total := 0
	for _, root := range roots {
		count := 0
		stack := []int64{root}
		visited := map[int64]bool{}
		for len(stack) > 0 {
			id := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if visited[id] {
				continue
			}
			visited[id] = true
			count++
			stack = append(stack, children[id]...)
		}
		if len(keep) > 0 && total+count > max {
			break
		}
		keep[root] = true
		total += count
	}
	out := make([]Object, 0, total)
	for _, node := range nodes {
		if keep[owner[spiritNodeID(node, "id")]] {
			out = append(out, node)
		}
	}
	return out
}

func neutralizeSpiritNodes(nodes []Object) {
	for _, node := range nodes {
		guide := strings.HasPrefix(text(node, "spirit"), "questap")
		action := text(node, "ta")
		if guide {
			node["cst"] = 0
			if strings.HasPrefix(action, "spirit_unlock") {
				node["ta"] = "zero_trust_unlock"
			}
			continue
		}
		if number(node, "cst", -1) == 0 && text(node, "typ") != "quest" && !strings.HasPrefix(action, "zero_trust") {
			node["cst"] = 1
		}
	}
}

func (a *Service) SpiritShops(r Request, m Object) (Object, error) {
	if a.cfg.GameConfigDir == "" {
		return nil, fail(503, "Game configuration unavailable")
	}
	library := filepath.Join(filepath.Dir(a.cfg.GameConfigDir), "resources", "static_responses", "account__get_spirit_shops.json")
	raw, err := os.ReadFile(library)
	if err != nil {
		raw, err = os.ReadFile(filepath.Join(a.cfg.GameConfigDir, "spirit_shops.json"))
	}
	if err != nil {
		return nil, fail(503, "Game configuration unavailable")
	}
	var data Object
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err = decoder.Decode(&data); err != nil {
		return nil, fail(503, "Invalid game configuration")
	}
	values, ok := data["spirit_shops"].([]any)
	if !ok {
		return nil, fail(503, "Invalid game configuration")
	}
	if control, readErr := a.legacyObject("game_control.json"); readErr == nil {
		if override := asObject(control["spirit_shops"]); override != nil && boolSetting(override["enabled"], false) {
			if shops, valid := override["shops"].([]any); valid && len(shops) > 0 {
				values = shops
			}
		}
	}
	nodes := make([]Object, 0, len(values))
	for _, value := range values {
		node, ok := value.(map[string]any)
		if ok {
			nodes = append(nodes, node)
		}
	}
	neutralizeSpiritNodes(nodes)
	nodes = limitSpiritTree(nodes, spiritTreeLimit(r.UserAgent))
	total := len(nodes)
	limit := number(m, "l", int64(total))
	offset := number(m, "o", 0)
	if limit <= 0 {
		limit = int64(total)
	}
	if offset < 0 {
		offset = 0
	}
	if offset > int64(total) {
		offset = int64(total)
	}
	end := int64(total)
	if limit < end-offset {
		end = offset + limit
	}
	return Object{"spirit_shops": nodes[int(offset):int(end)], "spirit_shops_total_count": total, "spirit_shops_version": number(data, "spirit_shops_version", 0)}, nil
}
