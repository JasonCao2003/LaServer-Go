package service

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

var feedExtraDefaults = map[string]any{
	"chunk_count": 0, "ori": "(0,0,0,1)", "pos": "(-10,0,-10)", "prompt_used": "",
	"resource_id": "", "scale": "(1,1,1)", "sub_type": 0,
}

type recentHint struct {
	types, names []string
	hits         int
	ts           time.Time
}

type recentHints struct {
	mu      sync.Mutex
	entries map[string][]recentHint
}

func newRecentHints() *recentHints {
	return &recentHints{entries: map[string][]recentHint{}}
}

func (h *recentHints) remember(key string, types, names []string, hits int) {
	if key == "" || (len(types) == 0 && len(names) == 0) {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	cutoff := time.Now().Add(-600 * time.Second)
	for k, list := range h.entries {
		live := list[:0]
		for _, entry := range list {
			if entry.ts.After(cutoff) {
				live = append(live, entry)
			}
		}
		if len(live) == 0 {
			delete(h.entries, k)
		} else {
			h.entries[k] = live
		}
	}
	list := append(h.entries[key], recentHint{types: types, names: names, hits: hits, ts: time.Now()})
	if len(list) > 8 {
		list = list[len(list)-8:]
	}
	h.entries[key] = list
}

func (h *recentHints) recall(key string, hits int) ([]string, []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	list := h.entries[key]
	cutoff := time.Now().Add(-600 * time.Second)
	var fallback *recentHint
	for i := len(list) - 1; i >= 0; i-- {
		entry := list[i]
		if !entry.ts.After(cutoff) {
			continue
		}
		if entry.hits == hits {
			return entry.types, entry.names
		}
		if fallback == nil {
			copyEntry := entry
			fallback = &copyEntry
		}
	}
	if fallback != nil {
		return fallback.types, fallback.names
	}
	return nil, nil
}

func feedBareID(id string) string {
	if index := strings.IndexByte(id, ':'); index >= 0 {
		return id[:index]
	}
	return id
}

func feedPublicID(p repository.FeedPost) string {
	if strings.Contains(p.SocialFeedID, ":") {
		return p.SocialFeedID
	}
	sum := md5.Sum([]byte(p.SocialFeedID + p.Message))
	return p.SocialFeedID + ":" + hex.EncodeToString(sum[:])[:8]
}

func feedISO(ts int64, millis bool) string {
	if ts <= 0 {
		return ""
	}
	t := time.Unix(ts, 0).UTC()
	if millis {
		return t.Format("2006-01-02T15:04:05.000Z")
	}
	return t.Format("2006-01-02T15:04:05Z")
}

func (a *Service) feedExtra(r Request, socialFeedID string) (Object, repository.FeedExtra) {
	extra := Object{}
	for k, v := range feedExtraDefaults {
		extra[k] = v
	}
	record, found, err := a.store.FeedExtra(r.Context, feedBareID(socialFeedID))
	if err == nil && found {
		var stored map[string]any
		if json.Unmarshal([]byte(record.ExtraData), &stored) == nil {
			for k := range feedExtraDefaults {
				if v, ok := stored[k]; ok && v != nil {
					extra[k] = v
				}
			}
			for _, key := range []string{"posted_to", "read_at", "user_id"} {
				if v, ok := stored[key]; ok && v != nil {
					extra[key] = v
				}
			}
		}
		return extra, record
	}
	return extra, repository.FeedExtra{CommentsEnabled: true}
}

func (a *Service) feedTags(r Request, socialFeedID string) any {
	_, record := a.feedExtra(r, socialFeedID)
	if record.Tags == "" {
		return nil
	}
	var tags any
	if json.Unmarshal([]byte(record.Tags), &tags) != nil {
		return nil
	}
	return tags
}

func feedReactions8(p repository.FeedPost) Object {
	return Object{"is_poisoned": p.Poisoned, "total_comment_count": p.Comments, "total_hide_count": p.Hides, "total_like_count": p.Likes, "total_like_reward_message_id": 0, "total_report_count": p.Reports, "total_reported_comment_count": 0, "total_super_like_count": 0}
}

func feedReactions9(p repository.FeedPost) Object {
	return Object{"score": 0, "is_poisoned": p.Poisoned, "total_hide_count": p.Hides, "total_like_count": p.Likes, "total_report_count": p.Reports, "total_attempt_count": p.Attempts, "total_comment_count": p.Comments, "total_succeed_count": p.Succeeds, "total_like_reward_message_id": 0}
}

func (a *Service) feedItemV1(r Request, p repository.FeedPost) Object {
	extra, record := a.feedExtra(r, p.SocialFeedID)
	return Object{
		"comments_enabled": record.CommentsEnabled,
		"create_at":        p.CreatedAt * 1000,
		"expire_at":        p.ExpireAt * 1000,
		"extra_data":       extra,
		"message":          p.Message,
		"pool_name":        p.PoolName,
		"pool_type":        p.PoolType,
		"reactions":        feedReactions8(p),
		"social_feed_id":   feedPublicID(p),
		"state":            "normal",
		"tags":             a.feedTags(r, p.SocialFeedID),
		"user_id":          p.UserID,
	}
}

func (a *Service) feedItemV2(r Request, p repository.FeedPost, viewer string) Object {
	extra, record := a.feedExtra(r, p.SocialFeedID)
	friend := false
	if viewer != "" {
		if ok, err := a.store.AreFriends(r.Context, viewer, p.UserID); err == nil {
			friend = ok
		}
	}
	return Object{
		"comments_enabled": record.CommentsEnabled,
		"create_at":        feedISO(p.CreatedAt, true),
		"expire_at":        feedISO(p.ExpireAt, true),
		"extra_data":       extra,
		"followed":         false,
		"friend":           friend,
		"local_creation":   viewer != "" && p.UserID == viewer,
		"message":          p.Message,
		"pool_name":        p.PoolName,
		"pool_type":        p.PoolType,
		"reactions":        feedReactions8(p),
		"social_feed_id":   feedPublicID(p),
		"state":            "normal",
		"tags":             a.feedTags(r, p.SocialFeedID),
		"user_id":          p.UserID,
	}
}

func (a *Service) feedItemAccount(r Request, p repository.FeedPost, viewer string, friend bool) Object {
	extra, record := a.feedExtra(r, p.SocialFeedID)
	reactions := feedReactions8(p)
	reactions["score"] = 0
	reactions["total_attempt_count"] = p.Attempts
	reactions["total_succeed_count"] = p.Succeeds
	var created, expires any
	if p.CreatedAt > 0 {
		created = feedISO(p.CreatedAt, true)
	}
	if p.ExpireAt > 0 {
		expires = feedISO(p.ExpireAt, false)
	}
	return Object{
		"social_feed_id": p.SocialFeedID, "user_id": p.UserID, "author_id": p.UserID,
		"pool_type": p.PoolType, "pool_name": p.PoolName, "create_at": created, "expire_at": expires,
		"message": p.Message, "reactions": reactions, "state": "normal",
		"tags": a.feedTags(r, p.SocialFeedID), "extra_data": extra,
		"local_creation": viewer != "" && viewer == p.UserID, "followed": false, "friend": friend,
		"comments_enabled": record.CommentsEnabled, "total_like_count": p.Likes,
		"total_hide_count": p.Hides, "total_report_count": p.Reports, "user_reactions": []Object{},
	}
}

func asStringList(v any) []string {
	switch x := v.(type) {
	case string:
		if x == "" {
			return nil
		}
		return []string{x}
	case []any:
		out := []string{}
		for _, item := range x {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func feedPoolTypes(m Object) []string {
	if list := asStringList(m["pool_types"]); len(list) > 0 {
		return list
	}
	return asStringList(m["pool_type"])
}

func feedPoolNames(m Object) []string {
	for _, key := range []string{"pool_names", "pool_name", "level_id"} {
		if list := asStringList(m[key]); len(list) > 0 {
			return list
		}
	}
	return nil
}

func clampBatch(v int64, def, min, max int64) int64 {
	if v == 0 {
		return def
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func (a *Service) feedAuth(r Request, m Object) (string, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	if uid == "" || text(m, "session") == "" {
		return "", fail(401, "Invalid session")
	}
	u, err := a.findUser(r.Context, "id", uid)
	if err != nil || stringValue(u.Session) != text(m, "session") {
		return "", fail(401, "Invalid session")
	}
	return uid, nil
}

func (a *Service) feedByAnyID(r Request, id string) (repository.FeedPost, bool) {
	bare := feedBareID(id)
	p, found, err := a.store.FeedByID(r.Context, bare)
	if err != nil {
		return p, false
	}
	if found {
		return p, true
	}
	if bare != id {
		p, found, _ = a.store.FeedByID(r.Context, id)
	}
	return p, found
}
