// 关系定义版本过滤

package core

// 判断关系事件是否允许
func RelationshipEventAllowed(event string, version [3]int, known bool) bool {
	if event == "SocialIAP" {
		return false
	}
	minimum := map[string][3]int{"SocialWarpBlock": {0, 11, 0}, "SocialLink": {0, 12, 0}, "SocialUnlink": {0, 12, 0}, "SocialIAP": {0, 19, 0}}
	min, gated := minimum[event]
	if !gated {
		return true
	}
	if !known {
		return false
	}
	for i := range version {
		if version[i] > min[i] {
			return true
		}
		if version[i] < min[i] {
			return false
		}
	}
	return true
}

// 按版本过滤关系定义
func LegacyRelationshipDefs(ua string, defs []map[string]any) []map[string]any {
	major, minor, patch, known := ClientVersion(ua)
	version := [3]int{major, minor, patch}
	legacy := major == 0 && minor < 5
	allowed := map[string]bool{"SocialOffer": true}
	if known && (minor > 3 || (minor == 3 && patch >= 0)) {
		allowed = map[string]bool{"SocialOffer": true, "SocialHandHold": true, "SocialChat": true, "SocialFreeChat": true, "SocialKingHug": true, "SocialTeleportUnlock": true, "RelationshipGift": true, "SocialShop": true}
	}
	kept := make([]map[string]any, 0, len(defs))
	for _, def := range defs {
		event, _ := def["event"].(string)
		if legacy && !allowed[event] {
			continue
		}
		if !RelationshipEventAllowed(event, version, known) {
			continue
		}
		kept = append(kept, def)
	}
	for changed := true; changed; {
		changed = false
		ids := map[any]bool{}
		for _, d := range kept {
			ids[d["id"]] = true
		}
		next := kept[:0]
		for _, d := range kept {
			reqs, _ := d["pre_reqs"].([]any)
			valid := true
			for _, id := range reqs {
				if id != nil && !ids[id] {
					valid = false
					break
				}
			}
			if valid {
				next = append(next, d)
			} else {
				changed = true
			}
		}
		kept = next
	}
	return kept
}
