package service

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

const (
	relationshipMaxPages = 64
	remoteOutfitZeroID   = "000000000000000000000000"
)

func relationTime(epoch int64) string {
	return time.Unix(epoch, 0).UTC().Format("2006-01-02T15:04:05.000Z")
}

func compactRelation(e repository.RelationEntry, nickname bool) Object {
	out := Object{"abilities": intList(e.Abilities), "created_at": relationTime(e.WhenCreated), "friend_id": e.ID, "hints": intList(e.Hints), "last_seen_outfit": nil, "soft_deleted": nil, "they_soft_deleted": nil}
	if nickname {
		out["nickname"] = e.Nickname
	}
	return out
}

func outfitPayload(o repository.Outfit) Object {
	out := Object{"voice": o.Voice, "seed": o.Seed, "scale": o.Scale, "refreshversion": o.RefreshVersion, "height": o.Height, "attitude": o.Attitude, "arms": Object{"id": o.ArmsID}}
	for name, part := range o.Parts {
		out[name] = Object{"tex": part["tex"], "pat": part["pat"], "mask": part["mask"], "id": part["id"]}
	}
	return out
}

func candidateFriendPayload(f repository.CandidateFriend, outfit *repository.Outfit) Object {
	var created any
	if f.CreatedAt.Valid {
		created = f.CreatedAt.String
	}
	stamp := f.WhenCreated
	if stamp == 0 {
		stamp = now()
	}
	out := Object{"friend_id": f.ID, "user_id": f.ID, "nickname": f.Nickname, "platform": f.Platform, "device": f.Device, "is_online": false, "last_seen": nil, "created_at": created, "relationship": "friend", "status": "active", "given": f.Given, "recvd": f.Received, "level": f.Level, "when_created": stamp, "player_badge_type": f.Badge, "attitude": "0", "abilities": intList(f.Abilities), "hints": intList(f.Hints), "soft_deleted": false, "they_soft_deleted": false, "local_soft_deleted": f.LocalDeleted, "blocked": true, "local_blocked": false, "they_blocked": f.TheyBlocked}
	if outfit != nil {
		out["outfit"] = outfitPayload(*outfit)
		out["last_seen_outfit"] = outfitPayload(*outfit)
	}
	return out
}

// relationshipUser resolves user/user_id plus session for the service routes.
func (a *Service) relationshipUser(r Request, m Object) (string, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	if uid == "" || text(m, "session") == "" {
		return "", fail(400, "Invalid request body")
	}
	u, err := a.findUser(r.Context, "id", uid)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fail(401, "Invalid session or user not found")
	}
	if err != nil {
		return "", err
	}
	if stringValue(u.Session) != text(m, "session") {
		return "", fail(401, "Invalid session or user not found")
	}
	if err = a.checkAllowed(r.Context, &u, 406); err != nil {
		return "", err
	}
	return uid, nil
}

func boolValue(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return x != "" && x != "false"
	case json.Number:
		n, err := x.Float64()
		return err == nil && n != 0
	case float64:
		return x != 0
	default:
		return false
	}
}

func floatValue(v any) (float64, bool) {
	switch x := v.(type) {
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	case float64:
		return x, true
	case int64:
		return float64(x), true
	case string:
		var f float64
		if err := json.Unmarshal([]byte(x), &f); err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

func valueOr(m Object, key string, fallback any) any {
	if v, ok := m[key]; ok && v != nil {
		return v
	}
	return fallback
}

func (a *Service) GetBlockedFriends(r Request, m Object) (Object, error) {
	u, err := a.authenticated(r, m)
	if err != nil {
		return nil, err
	}
	limit := number(m, "page_max", 150)
	if limit <= 0 {
		limit = 150
	}
	offset := number(m, "page_offset", 0)
	if offset < 0 {
		offset = 0
	}
	records, total, err := a.store.BlockedFriends(r.Context, u.ID, limit, offset)
	if err != nil {
		return emptyBlockedFriends(), nil
	}
	ids := make([]string, 0, len(records))
	for _, f := range records {
		ids = append(ids, f.ID)
	}
	outfits, err := a.store.Outfits(r.Context, ids)
	if err != nil {
		return emptyBlockedFriends(), nil
	}
	items := []Object{}
	for _, f := range records {
		var outfit *repository.Outfit
		if row, ok := outfits[f.ID]; ok {
			outfit = &row
		}
		items = append(items, candidateFriendPayload(f, outfit))
	}
	next := int64(-1)
	if offset+int64(len(items)) < total {
		next = offset + int64(len(items))
	}
	prev := int64(-1)
	if offset > 0 {
		prev = offset - limit
	}
	return Object{"set_blocked_friends": Object{"friends": items, "page_next": next, "page_prev": prev}}, nil
}

func emptyBlockedFriends() Object {
	return Object{"set_blocked_friends": Object{"friends": []Object{}, "page_next": -1, "page_prev": -1}}
}

func missingFields(m Object, keys ...string) string {
	missing := []string{}
	for _, key := range keys {
		if v, ok := m[key]; !ok || v == nil {
			missing = append(missing, key)
		}
	}
	return strings.Join(missing, ", ")
}

func (a *Service) SetFriendName(r Request, m Object) (Object, error) {
	if missing := missingFields(m, "user", "session", "friend", "name"); missing != "" {
		return Object{"result": "missing_parameters: " + missing}, nil
	}
	u, err := a.findUser(r.Context, "id", text(m, "user"))
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != text(m, "session")) {
		return Object{"result": "unauthorized"}, nil
	}
	if err != nil {
		return Object{"result": "internal_error"}, nil
	}
	name := []rune(text(m, "name"))
	if len(name) > 100 {
		name = name[:100]
	}
	entry, found, err := a.store.SaveFriendName(r.Context, u.ID, text(m, "friend"), string(name))
	if err != nil {
		return Object{"result": "internal_error"}, nil
	}
	if !found {
		return Object{"result": "friend_not_found"}, nil
	}
	return Object{"result": true, "update_friends": []Object{compactRelation(entry, true)}}, nil
}

func (a *Service) SetFriendBlock(r Request, m Object) (Object, error) {
	if missing := missingFields(m, "user", "session", "friend", "blocked"); missing != "" {
		return Object{"result": "missing_parameters: " + missing}, nil
	}
	u, err := a.findUser(r.Context, "id", text(m, "user"))
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != text(m, "session")) {
		return Object{"result": "unauthorized"}, nil
	}
	if err != nil {
		return Object{"result": "internal_error"}, nil
	}
	blocked := boolValue(m["blocked"])
	entry, found, err := a.store.SaveFriendBlock(r.Context, u.ID, text(m, "friend"), blocked)
	if err != nil {
		return Object{"result": "internal_error"}, nil
	}
	var item Object
	if found {
		item = compactRelation(entry, true)
	} else {
		item = Object{"abilities": []int64{}, "created_at": nil, "friend_id": text(m, "friend"), "hints": []int64{}, "last_seen_outfit": nil, "nickname": "", "soft_deleted": nil, "they_soft_deleted": nil}
	}
	item["blocked"] = blocked
	item["local_blocked"] = false
	item["they_blocked"] = entry.TheyBlocked
	return Object{"result": true, "update_friends": []Object{item}}, nil
}

func (a *Service) GetExternalFriends(r Request, m Object) (Object, error) {
	u, err := a.authenticated(r, m)
	if err != nil {
		return nil, err
	}
	limit := number(m, "max_count", 64)
	if limit < 0 {
		limit = 0
	}
	records, err := a.store.ExternalFriends(r.Context, u.ID, limit)
	if err != nil {
		return nil, err
	}
	items := []Object{}
	for _, f := range records {
		item := Object{"external_account_id": f.ExternalAccountID, "platform": f.Platform, "nickname": f.Nickname, "is_connected": f.IsConnected}
		if f.Avatar.Valid && f.Avatar.String != "" {
			item["avatar"] = f.Avatar.String
		}
		if f.IsConnected && f.ConnectedAt.Valid && f.ConnectedAt.Int64 != 0 {
			item["connected_at"] = f.ConnectedAt.Int64
		}
		if f.FriendUserID.Valid && f.FriendUserID.String != "" {
			item["game_user_id"] = f.FriendUserID.String
		}
		items = append(items, item)
	}
	return Object{"external_account_friends": items}, nil
}

func (a *Service) Serendipity(r Request, m Object) (Object, error) {
	u, err := a.authenticated(r, m)
	if err != nil {
		return nil, err
	}
	limit := number(m, "max_count", 4)
	if limit < 0 {
		limit = 0
	}
	records, err := a.store.SerendipityMatches(r.Context, u.ID, limit, now())
	if err != nil {
		return nil, err
	}
	items := []Object{}
	for _, match := range records {
		item := Object{"match_id": match.ID, "matched_user_id": match.MatchedUserID, "matched_user_nickname": "", "match_type": match.MatchType, "match_score": match.MatchScore, "matched_at": match.MatchedAt, "status": match.Status}
		if match.ExpiresAt.Valid && match.ExpiresAt.Int64 != 0 {
			item["expires_at"] = match.ExpiresAt.Int64
		}
		items = append(items, item)
	}
	return Object{"serendipity_matches": items}, nil
}

func (a *Service) StarTags(r Request, m Object) (Object, error) {
	u, err := a.authenticated(r, m)
	if err != nil {
		return nil, err
	}
	if major, minor, _, known := clientVersion(r.UserAgent); known && (major > 0 || (major == 0 && minor >= 26)) {
		var static Object
		if err := legacyResponse("star_tags", &static); err == nil {
			return static, nil
		}
	}
	defs, links, err := a.store.StarTags(r.Context, u.ID)
	if err != nil {
		return nil, err
	}
	defItems := []Object{}
	for _, d := range defs {
		batches := any([]any{})
		if strings.TrimSpace(d.Batches) != "" {
			var parsed any
			if json.Unmarshal([]byte(d.Batches), &parsed) == nil {
				batches = parsed
			}
		}
		defItems = append(defItems, Object{"name": d.Name, "sku": d.SKU, "part": d.Part, "consumable_name": d.ConsumableName, "group_name": d.GroupName, "batches": batches, "can_link_with": d.CanLinkWith})
	}
	linkItems := []Object{}
	for _, l := range links {
		linkItems = append(linkItems, Object{"tag_name": l.TagName, "linked_item": l.LinkedItem, "linked_at": l.LinkedAt})
	}
	return Object{"get_star_tag_defs": defItems, "star_tag_links": linkItems}, nil
}

func joinError(status int, message, code string) error {
	return &Error{status, Object{"result": "error", "message": message, "code": code}}
}

func (a *Service) JoinFriendGame(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	if uid == "" {
		return nil, joinError(400, "缺少用户ID字段", "MISSING_USER_ID")
	}
	if text(m, "session") == "" {
		return nil, joinError(400, "缺少会话令牌", "MISSING_SESSION_TOKEN")
	}
	friendID := text(m, "friend_id")
	if friendID == "" {
		return nil, joinError(400, "缺少好友ID", "MISSING_FRIEND_ID")
	}
	u, err := a.findUser(r.Context, "id", uid)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != text(m, "session")) {
		return nil, joinError(401, "无效的会话或用户不存在", "INVALID_SESSION")
	}
	if err != nil {
		return nil, err
	}
	exists, err := a.store.HasUser(r.Context, friendID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, joinError(404, "好友不存在", "FRIEND_NOT_FOUND")
	}
	assignment, found, err := a.store.LatestAssignment(r.Context, friendID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, joinError(404, "好友当前不在游戏中", "FRIEND_NOT_IN_GAME")
	}
	assignment.MoveTS = now() * 1000
	if err = a.store.SaveJoinAssignment(r.Context, uid, assignment, friendID); err != nil {
		return nil, err
	}
	move := Object{"signature": assignment.Signature, "sig": assignment.Sig, "target_id": friendID, "client_changelist": valueOr(m, "client_changelist", "282913"), "ps_only": assignment.PSOnly, "platform_filter": assignment.PlatformFilter, "public_ip": valueOr(m, "public_ip", "0.0.0.0"), "private_ip": valueOr(m, "private_ip", "127.0.0.1"), "port": valueOr(m, "port", 42336), "level": valueOr(m, "level", 3526133726), "level_hash": valueOr(m, "level_hash", 30890), "client_version": valueOr(m, "client_version", 5023), "players": []string{friendID}, "move_ts": assignment.MoveTS, "event_id": valueOr(m, "event_id", 0)}
	return Object{"move_to_game": move, "is_joinable": true, "is_dnd": false}, nil
}

var outfitPartDefaults = map[string]int64{
	"body": 3691343236, "wing": 2245279351, "hair": 1229053584, "mask": 3663390882, "neck": 3800884691,
	"feet": 1532093990, "horn": 3680499229, "prop": 2035109393, "hat": 1470462579, "face": 4267756678,
}

func (a *Service) SetOutfit(r Request, m Object) (Object, error) {
	if len(m) == 0 {
		return nil, fail(400, "Request body is empty or not valid JSON")
	}
	if missing := missingFields(m, "user", "session", "outfit"); missing != "" {
		return nil, fail(400, "Missing required fields: user_id, session, or outfit")
	}
	u, err := a.findUser(r.Context, "id", text(m, "user"))
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != text(m, "session")) {
		return nil, fail(401, "Invalid session or user not found")
	}
	if err != nil {
		return nil, err
	}
	outfit, ok := m["outfit"].(map[string]any)
	if !ok {
		return nil, fail(400, "Invalid outfit data format")
	}
	if _, err = a.store.GetOrCreateOutfit(r.Context, u.ID); err != nil {
		return nil, err
	}
	parts := map[string]any{}
	for name, fallbackID := range outfitPartDefaults {
		raw, exists := outfit[name]
		if !exists {
			continue
		}
		part, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		for _, key := range []string{"tex", "pat", "mask"} {
			if _, present := part[key]; present {
				parts[name+"_"+key] = number(part, key, 0)
			}
		}
		parts[name+"_id"] = number(part, "id", fallbackID)
	}
	if raw, exists := outfit["arms"].(map[string]any); exists {
		parts["arms_id"] = number(raw, "id", 0)
	}
	if _, exists := outfit["height"]; exists {
		if v, ok := floatValue(outfit["height"]); ok {
			parts["height"] = v
		}
	}
	if _, exists := outfit["scale"]; exists {
		if v, ok := floatValue(outfit["scale"]); ok {
			parts["scale"] = v
		}
	}
	for _, key := range []string{"voice", "seed"} {
		if _, exists := outfit[key]; exists {
			parts[key] = number(outfit, key, 0)
		}
	}
	if _, exists := outfit["attitude"]; exists {
		parts["attitude"] = text(outfit, "attitude")
	}
	if err = a.store.SetOutfitParts(r.Context, u.ID, parts); err != nil {
		return nil, err
	}
	return Object{"result": "ok", "message": "Outfit updated successfully"}, nil
}

func (a *Service) SetJoinable(_ Request, m Object) (Object, error) {
	if len(m) == 0 {
		return nil, fail(400, "Request body is empty or not valid JSON")
	}
	if _, ok := m["joinable"]; !ok {
		return nil, fail(400, "Missing required parameter: joinable")
	}
	return Object{"joinable": m["joinable"]}, nil
}

func defaultRemoteOutfit(userID string) repository.Outfit {
	o := repository.Outfit{Parts: map[string]map[string]int64{}, Voice: 9, Seed: 16515, Scale: 0.22549559, Height: 1.93536222, Attitude: "0"}
	for name, id := range map[string]int64{"wing": 2245279351, "prop": 2035109393, "neck": 3800884691, "mask": 3663390882, "horn": 3680499229, "hat": 1470462579, "hair": 1229053584, "feet": 1532093990, "face": 4267756678, "body": 3691343236} {
		o.Parts[name] = map[string]int64{"id": id}
	}
	return o
}

func (a *Service) GetRemoteOutfit(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	if uid == "" || text(m, "session") == "" {
		return nil, fail(400, "Invalid request body")
	}
	u, err := a.findUser(r.Context, "id", uid)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != text(m, "session")) {
		return nil, fail(401, "invalid session")
	}
	if err != nil {
		return nil, err
	}
	target := text(m, "target")
	if target == "" {
		target = text(m, "target_user")
	}
	if target == "" {
		target = text(m, "remote_user")
	}
	if target == "" {
		return nil, fail(400, "Missing target")
	}
	resolved := target
	exists, err := a.store.HasUser(r.Context, resolved)
	if err != nil {
		return nil, err
	}
	if !exists {
		if base := repository.CleanCreator(target); base != resolved {
			resolved = base
			exists, err = a.store.HasUser(r.Context, resolved)
			if err != nil {
				return nil, err
			}
		}
	}
	var outfit repository.Outfit
	if exists {
		outfit, err = a.store.GetOrCreateOutfit(r.Context, resolved)
		if err != nil {
			return nil, err
		}
	} else {
		outfit = defaultRemoteOutfit(resolved)
	}
	return Object{"result": "ok", "get_remote_outfit": outfitPayload(outfit), "serendipity_types": []any{}}, nil
}

func (a *Service) RemoveRelationshipHint(r Request, m Object) (Object, error) {
	failure := Object{"result": false, "update_friends": []Object{}}
	if missingFields(m, "user", "session", "friend") != "" {
		return failure, nil
	}
	u, err := a.findUser(r.Context, "id", text(m, "user"))
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != text(m, "session")) || err != nil {
		return failure, nil
	}
	entry, found, err := a.store.RemoveFriendHint(r.Context, u.ID, text(m, "friend"), number(m, "hint", -1))
	if err != nil || !found {
		return failure, nil
	}
	return Object{"result": true, "update_friends": []Object{compactRelation(entry, true)}}, nil
}

func (a *Service) GetAllFriendGist(_ Request, _ Object) (Object, error) {
	return Object{"get_all_friend_gist": []any{}}, nil
}

func (a *Service) GetFollowerCount(r Request, m Object) (Object, error) {
	uid, err := a.relationshipUser(r, m)
	if err != nil {
		return nil, err
	}
	count, err := a.store.FollowerCount(r.Context, uid)
	if err != nil {
		return nil, err
	}
	return Object{"count": count}, nil
}

func constellationFailure(status string) Object {
	return Object{"res": "ok", "result": false, "status": status, "constellation_friend_pages": []Object{}, "pages": []Object{}, "total_pages": 0, "total_friends": 0}
}

func (a *Service) UpdateFriendConstellationPages(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	if uid == "" || text(m, "session") == "" {
		return constellationFailure("missing_parameters"), nil
	}
	u, err := a.findUser(r.Context, "id", uid)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != text(m, "session")) || err != nil {
		return constellationFailure("unauthorized"), nil
	}
	sentPages, hasPages := m["pages"].([]any)
	if !hasPages {
		sentPages = []any{}
	}
	if len(sentPages) > relationshipMaxPages {
		sentPages = sentPages[:relationshipMaxPages]
	}
	pages := make([][]repository.ConstellationItem, 0, len(sentPages))
	for _, rawPage := range sentPages {
		page, ok := rawPage.(map[string]any)
		if !ok {
			pages = append(pages, []repository.ConstellationItem{})
			continue
		}
		friendsRaw, _ := page["friends"].([]any)
		friends := []repository.ConstellationItem{}
		seen := map[string]bool{}
		for _, rawFriend := range friendsRaw {
			friend, ok := rawFriend.(map[string]any)
			if !ok {
				continue
			}
			id := text(friend, "friend_id")
			if id == "" {
				id = text(friend, "friend")
			}
			if id == "" {
				continue
			}
			id = strings.TrimSpace(repository.CleanCreator(id))
			if id == "" || seen[id] {
				continue
			}
			if len(id) > 36 {
				id = id[:36]
			}
			seen[id] = true
			friends = append(friends, repository.ConstellationItem{FriendID: id, Node: number(friend, "constellation_node_index", 0)})
		}
		pages = append(pages, friends)
	}
	if err = a.store.SaveConstellationPages(r.Context, u.ID, pages); err != nil {
		return constellationFailure("server_error"), nil
	}
	slots, err := a.store.Constellations(r.Context, u.ID)
	if err != nil {
		return constellationFailure("server_error"), nil
	}
	total := int64(len(pages))
	for _, slot := range slots {
		if slot.Page+1 > total {
			total = slot.Page + 1
		}
	}
	outPages := make([]Object, total)
	for i := range outPages {
		outPages[i] = Object{"name": "", "page_index": i, "friends": []Object{}}
	}
	friendCount := 0
	for _, slot := range slots {
		page := outPages[slot.Page]
		page["friends"] = append(page["friends"].([]Object), Object{"friend_id": slot.FriendID, "constellation_node_index": slot.Node})
		friendCount++
	}
	return Object{"res": "ok", "result": true, "status": "ok", "constellation_friend_pages": outPages, "pages": outPages, "total_pages": len(outPages), "total_friends": friendCount}, nil
}

func (a *Service) TrustGet(_ Request, _ Object) (Object, error) {
	return Object{"npcs": []any{}, "players": []any{}}, nil
}

func (a *Service) TrustGetCapped(_ Request, _ Object) (Object, error) {
	// The legacy handler auto-created an account for any supplied user id. The
	// Go service keeps the fixed response but does not create unauthenticated
	// accounts; account creation remains on /account/create.
	return Object{"capped_activities": []any{}, "capped": []any{}}, nil
}

func followResponse(userID, creator, nickname string) Object {
	return Object{"result": "ok", "follower": Object{"id": remoteOutfitZeroID, "follower": userID, "creator": repository.CleanCreator(creator), "last_modified": relationTime(now()), "nickname": nickname}}
}

func (a *Service) AddFollow(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	if uid == "" || text(m, "session") == "" {
		return nil, &Error{Status: 400, Body: Object{"result": "error"}}
	}
	creator := text(m, "creator")
	if creator == "" {
		return nil, &Error{Status: 400, Body: Object{"result": "error"}}
	}
	u, err := a.findUser(r.Context, "id", uid)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != text(m, "session")) {
		return nil, &Error{Status: 401, Body: Object{"result": "error"}}
	}
	if err != nil {
		return nil, err
	}
	nickname := text(m, "nickname")
	if err = a.store.UpsertBoatFollow(r.Context, u.ID, repository.CleanCreator(creator), nickname, relationTime(now())); err != nil {
		return nil, err
	}
	return followResponse(u.ID, creator, nickname), nil
}

func (a *Service) SetFollowName(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	if uid == "" || text(m, "session") == "" {
		return nil, &Error{Status: 400, Body: Object{"result": "error"}}
	}
	u, err := a.findUser(r.Context, "id", uid)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != text(m, "session")) {
		return nil, &Error{Status: 401, Body: Object{"result": "error"}}
	}
	if err != nil {
		return nil, err
	}
	creator, nickname := text(m, "creator"), text(m, "nickname")
	if err = a.store.UpsertBoatFollow(r.Context, u.ID, repository.CleanCreator(creator), nickname, relationTime(now())); err != nil {
		return nil, err
	}
	return followResponse(u.ID, creator, nickname), nil
}

func (a *Service) DeleteFollow(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	if uid == "" || text(m, "session") == "" {
		return nil, &Error{Status: 400, Body: Object{"result": "error"}}
	}
	u, err := a.findUser(r.Context, "id", uid)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != text(m, "session")) {
		return nil, &Error{Status: 401, Body: Object{"result": "error"}}
	}
	if err != nil {
		return nil, err
	}
	deleted, err := a.store.DeleteBoatFollow(r.Context, u.ID, repository.CleanCreator(text(m, "creator")))
	if err != nil {
		return nil, err
	}
	return Object{"number_deleted": deleted}, nil
}
