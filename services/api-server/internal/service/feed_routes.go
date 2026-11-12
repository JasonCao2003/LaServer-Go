package service

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

const feedTTLSeconds = 14 * 24 * 3600

func (a *Service) socialCreate(r Request, m Object, account bool) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	if uid == "" || text(m, "session") == "" {
		if account {
			return Object{"result": "error", "message": "Missing fields"}, nil
		}
		return nil, &Error{Status: 400, Body: Object{"result": "error", "error": "Invalid request body"}}
	}
	u, err := a.findUser(r.Context, "id", uid)
	if err != nil || stringValue(u.Session) != text(m, "session") {
		if account {
			return Object{"result": "error", "message": "Invalid session"}, nil
		}
		return nil, &Error{Status: 401, Body: Object{"result": "error", "error": "Invalid session"}}
	}
	poolType, poolName := text(m, "pool_type"), text(m, "pool_name")
	if poolType == "" || poolName == "" {
		if account {
			return Object{"result": "error", "message": "Missing fields"}, nil
		}
		return nil, &Error{Status: 400, Body: Object{"result": "error", "error": "Missing fields"}}
	}
	fid, err := uuid()
	if err != nil {
		return Object{"result": "error"}, nil
	}
	stamp := now()
	post := repository.FeedPost{UserID: uid, SocialFeedID: fid, PoolType: poolType, PoolName: poolName, Message: text(m, "message"), CreatedAt: stamp, ExpireAt: stamp + feedTTLSeconds}
	if err = a.store.CreateFeed(r.Context, post); err != nil {
		if account {
			return Object{"result": "error", "message": "Internal error"}, nil
		}
		return Object{"result": "error", "error": "internal_error"}, nil
	}
	if extra, ok := m["extra_data"].(map[string]any); ok {
		var comments, private *bool
		if v, exists := m["comments_enabled"]; exists {
			value := boolValue(v)
			comments = &value
		}
		if v, exists := m["private"]; exists {
			value := boolValue(v)
			private = &value
		}
		a.storeFeedExtra(r, fid, extra, comments, m["tags"], private)
	}
	if account {
		return Object{"result": "success", "social_feed_id": fid}, nil
	}
	return Object{"result": "success", "social_feed_id": feedPublicID(post)}, nil
}

func (a *Service) storeFeedExtra(r Request, socialFeedID string, extra map[string]any, comments *bool, tags any, private *bool) {
	merge := Object{}
	if old, found, err := a.store.FeedExtra(r.Context, feedBareID(socialFeedID)); err == nil && found {
		_ = json.Unmarshal([]byte(old.ExtraData), &merge)
	}
	for k, v := range extra {
		merge[k] = v
	}
	encoded, _ := json.Marshal(merge)
	record := repository.FeedExtra{SocialFeedID: feedBareID(socialFeedID), ExtraData: string(encoded), ResourceID: text(merge, "resource_id"), CommentsEnabled: true}
	if old, found, err := a.store.FeedExtra(r.Context, feedBareID(socialFeedID)); err == nil && found {
		record.CommentsEnabled = old.CommentsEnabled
		record.Private = old.Private
		record.Tags = old.Tags
	}
	if comments != nil {
		record.CommentsEnabled = *comments
	}
	if tags != nil {
		encodedTags, _ := json.Marshal(tags)
		record.Tags = string(encodedTags)
	}
	if private != nil {
		record.Private = *private
	}
	_ = a.store.SaveFeedExtra(r.Context, record)
}

func (a *Service) SocialFeedCreate(r Request, m Object) (Object, error) {
	return a.socialCreate(r, m, true)
}

func (a *Service) ServiceSocialFeedCreate(r Request, m Object) (Object, error) {
	return a.socialCreate(r, m, false)
}

func (a *Service) AccountGetUserFeeds(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	if uid == "" || text(m, "session") == "" {
		return Object{"feeds": []Object{}, "result": "ok", "total_available_results": 0}, nil
	}
	if u, err := a.findUser(r.Context, "id", uid); err != nil || stringValue(u.Session) != text(m, "session") {
		return Object{"feeds": []Object{}, "result": "ok", "total_available_results": 0}, nil
	}
	skip := number(m, "skip", 0)
	if skip < 0 {
		skip = 0
	}
	limit := clampBatch(number(m, "batch_size", 0), 10, 1, 200)
	posts, total, err := a.store.Feeds(r.Context, feedPoolTypes(m), feedPoolNames(m), skip, limit, true, now())
	if err != nil {
		return Object{"feeds": []Object{}, "result": "ok", "total_available_results": 0}, nil
	}
	items := []Object{}
	for _, p := range posts {
		items = append(items, a.feedItemAccount(r, p, uid, false))
	}
	return Object{"feeds": items, "result": "ok", "total_available_results": total}, nil
}

func (a *Service) AccountGetCuratedFeeds(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	viewer := ""
	if uid != "" && text(m, "session") != "" {
		if u, err := a.findUser(r.Context, "id", uid); err == nil && stringValue(u.Session) == text(m, "session") {
			viewer = uid
		}
	}
	posts, _, err := a.store.Feeds(r.Context, feedPoolTypes(m), feedPoolNames(m), 0, 0, true, now())
	if err != nil {
		return emptyCurated(), nil
	}
	nPublic := clampBatch(number(m, "n_public", 0), 32, 1, 200)
	nFriends := clampBatch(number(m, "n_friends", 0), 32, 0, 200)
	nRandom := clampBatch(number(m, "n_random", 0), 6, 0, 200)
	friends := []repository.FeedPost{}
	public := []repository.FeedPost{}
	for _, p := range posts {
		if viewer != "" {
			if ok, _ := a.store.AreFriends(r.Context, viewer, p.UserID); ok {
				friends = append(friends, p)
				continue
			}
		}
		public = append(public, p)
	}
	selected := []repository.FeedPost{}
	if int64(len(friends)) > nFriends {
		friends = friends[:nFriends]
	}
	selected = append(selected, friends...)
	publicSelection := public
	if int64(len(publicSelection)) > nPublic {
		publicSelection = publicSelection[:nPublic]
	}
	selected = append(selected, publicSelection...)
	if nRandom > 0 && int64(len(public)) > int64(len(publicSelection)) {
		extra := public[len(publicSelection):]
		if int64(len(extra)) > nRandom {
			extra = extra[:nRandom]
		}
		selected = append(selected, extra...)
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].CreatedAt > selected[j].CreatedAt })
	items := []Object{}
	for _, p := range selected {
		friend := false
		for _, f := range friends {
			if f.SocialFeedID == p.SocialFeedID {
				friend = true
			}
		}
		items = append(items, a.feedItemAccount(r, p, viewer, friend))
	}
	local := []Object{}
	friendItems := []Object{}
	for i, item := range items {
		if item["local_creation"] == true {
			local = append(local, item)
		}
		if i < len(selected) {
			// friend list is the first len(friends) selected entries
		}
	}
	for _, p := range selected {
		for _, f := range friends {
			if f.SocialFeedID == p.SocialFeedID {
				friendItems = append(friendItems, a.feedItemAccount(r, p, viewer, true))
			}
		}
	}
	inner := Object{"followed": nil, "friends": nil, "local": nil, "public": items, "total_available_results": len(items), "total_followed": 0, "total_friends": len(friends)}
	return Object{"feeds": items, "local": local, "random": []Object{}, "friends": friendItems, "followed": []Object{}, "total_friends": len(friends), "total_available_results": len(items), "inner_ring": inner, "outer_ring": nil, "result": "ok"}, nil
}

func emptyCurated() Object {
	inner := Object{"followed": nil, "friends": nil, "local": nil, "public": []Object{}, "total_available_results": 0, "total_followed": 0, "total_friends": 0}
	return Object{"feeds": []Object{}, "local": []Object{}, "random": []Object{}, "friends": []Object{}, "followed": []Object{}, "total_friends": 0, "total_available_results": 0, "inner_ring": inner, "outer_ring": nil, "result": "ok"}
}

func (a *Service) AccountGetFeedReactions(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	if uid == "" || text(m, "session") == "" || text(m, "social_feed_id") == "" {
		return nil, &Error{Status: 400, Body: Object{"error": "Missing required fields"}}
	}
	u, err := a.findUser(r.Context, "id", uid)
	if err != nil || stringValue(u.Session) != text(m, "session") {
		return nil, &Error{Status: 401, Body: Object{"error": "Invalid session"}}
	}
	p, found := a.feedByAnyID(r, text(m, "social_feed_id"))
	if !found {
		return nil, &Error{Status: 404, Body: Object{"error": "Social feed post not found"}}
	}
	return Object{"reactions": feedReactions9(p), "user_reactions": nil}, nil
}

func (a *Service) AccountFeedLike(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	if uid == "" || text(m, "session") == "" || text(m, "social_feed_id") == "" {
		return Object{"result": "error", "message": "Missing fields: user, session, social_feed_id, liked"}, nil
	}
	if _, ok := m["liked"]; !ok {
		return Object{"result": "error", "message": "Missing fields: liked"}, nil
	}
	u, err := a.findUser(r.Context, "id", uid)
	if err != nil || stringValue(u.Session) != text(m, "session") {
		return Object{"result": "error", "message": "Invalid session"}, nil
	}
	rawID := text(m, "social_feed_id")
	p, found := a.feedByAnyID(r, rawID)
	if !found {
		return Object{"result": "error", "message": "Social feed post not found"}, nil
	}
	liked := boolValue(m["liked"])
	exists, err := a.store.FeedLikeExists(r.Context, uid, rawID)
	if err != nil {
		return Object{"result": "error", "message": "Database error"}, nil
	}
	likes := p.Likes
	if liked && !exists {
		if err = a.store.AddFeedLike(r.Context, uid, rawID, now()); err == nil {
			likes++
		}
	} else if !liked && exists {
		if err = a.store.RemoveFeedLike(r.Context, uid, rawID); err == nil && likes > 0 {
			likes--
		}
	}
	_ = a.store.UpdateFeedCounters(r.Context, p.SocialFeedID, likes, p.Hides)
	return Object{"result": "ok"}, nil
}

func (a *Service) AccountDeleteFeed(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	if uid == "" || text(m, "session") == "" || text(m, "social_feed_id") == "" || text(m, "author_id") == "" {
		return Object{"result": "error", "message": "Missing fields: user, session, social_feed_id, author_id"}, nil
	}
	u, err := a.findUser(r.Context, "id", uid)
	if err != nil || stringValue(u.Session) != text(m, "session") {
		return Object{"result": "error", "message": "Invalid session"}, nil
	}
	p, found := a.feedByAnyID(r, text(m, "social_feed_id"))
	if !found {
		return Object{"result": "error", "message": "Social feed post not found"}, nil
	}
	if p.UserID != uid {
		return Object{"result": "error", "message": "Permission denied"}, nil
	}
	if _, err = a.store.DeleteFeed(r.Context, p.SocialFeedID); err != nil {
		return Object{"result": "error", "message": "Database error"}, nil
	}
	return Object{"result": "ok"}, nil
}

func (a *Service) ServiceFeedEdit(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	if uid != "" && text(m, "session") != "" {
		if u, err := a.findUser(r.Context, "id", uid); err != nil || stringValue(u.Session) != text(m, "session") {
			uid = ""
		}
	} else {
		uid = ""
	}
	rawID := text(m, "social_feed_id")
	if rawID == "" {
		rawID = text(m, "feed_id")
	}
	if rawID == "" {
		rawID = text(m, "id")
	}
	bare := feedBareID(rawID)
	post, found := a.feedByAnyID(r, rawID)
	stamp := now()
	if found {
		message := text(m, "message")
		poolType, poolName := text(m, "pool_type"), text(m, "pool_name")
		_ = a.store.UpdateFeed(r.Context, post.SocialFeedID, message, poolType, poolName, stamp+feedTTLSeconds)
		if message != "" {
			post.Message = message
		}
		if poolType != "" {
			post.PoolType = poolType
		}
		if poolName != "" {
			post.PoolName = poolName
		}
		post.ExpireAt = stamp + feedTTLSeconds
	} else {
		fid := bare
		if len(fid) != 36 || strings.Count(fid, "-") != 4 || fid == "00000000-0000-0000-0000-000000000000" {
			generated, err := uuid()
			if err != nil {
				return Object{"result": "error"}, nil
			}
			fid = generated
		}
		post = repository.FeedPost{UserID: uid, SocialFeedID: fid, PoolType: text(m, "pool_type"), PoolName: text(m, "pool_name"), Message: text(m, "message"), CreatedAt: stamp, ExpireAt: stamp + feedTTLSeconds}
		if err := a.store.CreateFeed(r.Context, post); err != nil {
			return Object{"result": "error"}, nil
		}
	}
	if extra, ok := m["extra_data"].(map[string]any); ok && len(extra) > 0 {
		if resource := text(m, "resource_id"); resource != "" {
			if _, exists := extra["resource_id"]; !exists {
				extra["resource_id"] = resource
			}
		}
		var comments, private *bool
		if v, exists := m["comments_enabled"]; exists {
			value := boolValue(v)
			comments = &value
		}
		if v, exists := m["private"]; exists {
			value := boolValue(v)
			private = &value
		}
		a.storeFeedExtra(r, post.SocialFeedID, extra, comments, m["tags"], private)
	}
	publicID := feedPublicID(post)
	if strings.Contains(rawID, ":") && feedBareID(rawID) == post.SocialFeedID {
		publicID = rawID
	}
	return Object{"result": "success", "social_feed_id": publicID}, nil
}

func (a *Service) ServiceFeedHide(r Request, m Object) (Object, error) {
	if _, err := a.feedAuth(r, m); err != nil {
		return nil, &Error{Status: 401, Body: Object{"result": "error"}}
	}
	p, found := a.feedByAnyID(r, text(m, "social_feed_id"))
	if found {
		_ = a.store.UpdateFeedCounters(r.Context, p.SocialFeedID, p.Likes, p.Hides+1)
	}
	return Object{"result": ""}, nil
}

func (a *Service) ServiceFeedLike(r Request, m Object) (Object, error) {
	uid, err := a.feedAuth(r, m)
	if err != nil {
		return nil, &Error{Status: 401, Body: Object{"result": "error"}}
	}
	bare := feedBareID(text(m, "social_feed_id"))
	p, found, err := a.store.FeedByID(r.Context, bare)
	if err != nil {
		return Object{"result": "error", "error": "internal_error"}, nil
	}
	liked := boolValue(m["liked"])
	exists, _ := a.store.FeedLikeExists(r.Context, uid, bare)
	likes := p.Likes
	if liked && !exists {
		_ = a.store.AddFeedLike(r.Context, uid, bare, now())
		likes++
	} else if !liked && exists {
		_ = a.store.RemoveFeedLike(r.Context, uid, bare)
		if likes > 0 {
			likes--
		}
	}
	if found {
		_ = a.store.UpdateFeedCounters(r.Context, p.SocialFeedID, likes, p.Hides)
	}
	return Object{"result": "created"}, nil
}

func (a *Service) ServiceFeedSuperLike(r Request, m Object) (Object, error) {
	uid, err := a.feedAuth(r, m)
	if err != nil {
		return nil, &Error{Status: 401, Body: Object{"result": "error"}}
	}
	bare := feedBareID(text(m, "social_feed_id"))
	p, found, _ := a.store.FeedByID(r.Context, bare)
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return Object{"result": "error", "error": "internal_error"}, nil
	}
	defer tx.Rollback()
	if err = tx.EnsureCurrency(r.Context, uid); err != nil {
		return Object{"result": "error", "error": "internal_error"}, nil
	}
	balance, err := tx.CurrencyBalance(r.Context, uid, "candles")
	if err != nil {
		return Object{"result": "error", "error": "internal_error"}, nil
	}
	if boolValue(m["super_liked"]) {
		newBalance := balance - 3
		if newBalance < 0 {
			newBalance = 0
		}
		if err = tx.SetCurrency(r.Context, uid, "candles", newBalance); err != nil {
			return Object{"result": "error", "error": "internal_error"}, nil
		}
		balance = newBalance
		if exists, _ := a.store.FeedLikeExists(r.Context, uid, bare); !exists {
			_, _ = tx.ExecContext(r.Context, repository.Statement(`INSERT INTO social_feed_like(user_id,social_feed_id,created_at) VALUES(?,?,?) ON CONFLICT(user_id,social_feed_id) DO NOTHING`), uid, bare, now())
		}
		if found {
			_, _ = tx.ExecContext(r.Context, repository.Statement(`UPDATE social_feed_post SET likes_count=COALESCE(likes_count,0)+1 WHERE social_feed_id=?`), p.SocialFeedID)
		}
	}
	if err = tx.Commit(); err != nil {
		return Object{"result": "error", "error": "internal_error"}, nil
	}
	return Object{"result": "success", "update_currency": Object{"candles": balance}}, nil
}

func (a *Service) ServiceFeedDelete(r Request, m Object) (Object, error) {
	if _, err := a.feedAuth(r, m); err != nil {
		return nil, &Error{Status: 401, Body: Object{"result": "error"}}
	}
	if p, found := a.feedByAnyID(r, text(m, "social_feed_id")); found {
		_, _ = a.store.DeleteFeed(r.Context, p.SocialFeedID)
	}
	return Object{"result": "ok"}, nil
}

func (a *Service) SocialFeedComments(_ Request, _ Object) (Object, error) {
	return Object{"comments": []Object{}, "local": []Object{}}, nil
}

func (a *Service) SocialFeedReport(_ Request, _ Object) (Object, error) {
	return Object{"result": ""}, nil
}

func (a *Service) ServiceGetCuratedFeeds(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	viewer := ""
	if uid != "" && text(m, "session") != "" {
		if u, err := a.findUser(r.Context, "id", uid); err == nil && stringValue(u.Session) == text(m, "session") {
			viewer = uid
		}
	}
	limit := clampBatch(number(m, "batch_size", 0), 64, 1, 200)
	posts, _, err := a.store.Feeds(r.Context, feedPoolTypes(m), feedPoolNames(m), 0, limit, false, now())
	if err != nil {
		return emptyServiceCurated(), nil
	}
	items := []Object{}
	for _, p := range posts {
		items = append(items, a.feedItemV2(r, p, viewer))
	}
	inner := Object{"followed": nil, "friends": nil, "local": nil, "public": items, "total_available_results": len(items), "total_followed": 0, "total_friends": 0}
	return Object{"inner_ring": inner, "outer_ring": nil}, nil
}

func emptyServiceCurated() Object {
	inner := Object{"followed": nil, "friends": nil, "local": nil, "public": []Object{}, "total_available_results": 0, "total_followed": 0, "total_friends": 0}
	return Object{"inner_ring": inner, "outer_ring": nil}
}

func (a *Service) ServiceGetUserFeeds(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	viewer := ""
	if uid != "" && text(m, "session") != "" {
		if u, err := a.findUser(r.Context, "id", uid); err == nil && stringValue(u.Session) == text(m, "session") {
			viewer = uid
		}
	}
	skip := number(m, "skip", 0)
	limit := clampBatch(number(m, "batch_size", 0), 20, 1, 200)
	posts, _, err := a.store.Feeds(r.Context, feedPoolTypes(m), feedPoolNames(m), skip, limit, false, now())
	if err != nil {
		return Object{"feeds": []Object{}}, nil
	}
	items := []Object{}
	for _, p := range posts {
		items = append(items, a.feedItemV1(r, p))
	}
	a.hints.remember(viewer, feedPoolTypes(m), feedPoolNames(m), len(posts))
	return Object{"feeds": items}, nil
}

func (a *Service) ServiceGetByIDs(r Request, m Object) (Object, error) {
	ids := []string{}
	add := func(v any) {
		for _, id := range asStringList(v) {
			ids = append(ids, feedBareID(id))
		}
	}
	for _, key := range []string{"social_feed_ids", "feed_ids", "ids", "social_feed_id", "feed_id"} {
		if v, ok := m[key]; ok {
			add(v)
		}
	}
	if len(ids) == 0 {
		if list, ok := m["messages"].([]any); ok {
			for _, item := range list {
				if obj, ok := item.(map[string]any); ok {
					add(obj["social_feed_id"])
				}
			}
		}
	}
	resolved := []string{}
	for _, id := range ids {
		if mapped, found, err := a.store.FeedIDForResource(r.Context, id); err == nil && found {
			resolved = append(resolved, mapped)
		} else {
			resolved = append(resolved, id)
		}
	}
	posts, err := a.store.FeedsByIDs(r.Context, resolved)
	if err != nil {
		return Object{"feeds": []Object{}, "result": "ok"}, nil
	}
	byID := map[string]repository.FeedPost{}
	for _, p := range posts {
		byID[p.SocialFeedID] = p
	}
	items := []Object{}
	seen := map[string]bool{}
	for _, id := range resolved {
		if p, ok := byID[id]; ok && !seen[p.SocialFeedID] {
			seen[p.SocialFeedID] = true
			items = append(items, a.feedItemV1(r, p))
		}
	}
	return Object{"feeds": items, "result": "ok"}, nil
}

func (a *Service) ServiceGetFeedReactions(r Request, m Object) (Object, error) {
	p, found := a.feedByAnyID(r, text(m, "social_feed_id"))
	if !found {
		return Object{"reactions": feedReactions8(repository.FeedPost{}), "user_reactions": nil}, nil
	}
	return Object{"reactions": feedReactions8(p), "user_reactions": nil}, nil
}

func (a *Service) ServiceGetPenpals(_ Request, _ Object) (Object, error) {
	return Object{"penpals": []Object{}}, nil
}

func (a *Service) ServiceGetPenpalByID(r Request, m Object) (Object, error) {
	p, found := a.feedByAnyID(r, text(m, "social_feed_id"))
	if !found || p.PoolType != "SocialFeedPenpal" {
		return Object{"feeds": []Object{}, "result": "ok"}, nil
	}
	return Object{"feeds": []Object{a.feedItemV1(r, p)}, "result": "ok"}, nil
}
