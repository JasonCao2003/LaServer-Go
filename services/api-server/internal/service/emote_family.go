package service

import (
	"sort"
	"strconv"
	"strings"
)

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
func (a *Service) emoteFamilyLevels(name string) []string {
	family, ok := emoteFamily(name)
	if !ok {
		return nil
	}
	type level struct {
		name   string
		number int
	}
	levels := []level{}
	for _, node := range a.catalog("account__get_spirit_shops.json", "spirit_shops.json", "spirit_shops") {
		candidate := text(node, "nm")
		prefix, match := emoteFamily(candidate)
		if !match || prefix != family || text(node, "typ") != "spirit_upgrade" {
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
