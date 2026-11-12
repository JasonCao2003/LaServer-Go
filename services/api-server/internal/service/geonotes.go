package service

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

const geonoteIDAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

var geonotePosRE = regexp.MustCompile(`[-0-9.]+`)

func newGeonoteID() string {
	buf := make([]byte, 6)
	_, _ = rand.Read(buf)
	suffix := make([]byte, 6)
	for i, b := range buf {
		suffix[i] = geonoteIDAlphabet[int(b)%len(geonoteIDAlphabet)]
	}
	return fmt.Sprintf("gn_%x_%s", nowMillis(), suffix)
}

func geonoteError(status int, code string) error {
	return &Error{Status: status, Body: Object{"result": code}}
}

func (a *Service) geonoteReadUser(r Request, m Object) (string, error) {
	if _, ok := m["user"]; !ok {
		return "", geonoteError(400, "invalid_params")
	}
	if _, ok := m["session"]; !ok {
		return "", geonoteError(400, "invalid_params")
	}
	uid := text(m, "user")
	u, err := a.findUser(r.Context, "id", uid)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != text(m, "session")) {
		return "", geonoteError(401, "invalid_session")
	}
	if err != nil {
		return "", geonoteError(500, "internal_error")
	}
	if integer(u.IsBanned) == 1 {
		end := integer(u.BanEndTime)
		if end == 0 {
			return "", geonoteError(403, "user_permanently_banned")
		}
		if now() < end {
			return "", geonoteError(403, "user_banned")
		}
		if _, err = a.store.ExecContext(r.Context, repository.ClearExpiredBan, uid); err != nil {
			return "", geonoteError(500, "internal_error")
		}
	}
	return uid, nil
}

func (a *Service) geonoteWriteUser(r Request, m Object) (string, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	if uid == "" || text(m, "session") == "" {
		return "", geonoteError(400, "invalid_params")
	}
	u, err := a.findUser(r.Context, "id", uid)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != text(m, "session")) {
		return "", geonoteError(401, "invalid_session")
	}
	if err != nil {
		return "", geonoteError(500, "internal_error")
	}
	return uid, nil
}

func vec3(v any, fallback [3]float64) [3]float64 {
	out := fallback
	switch x := v.(type) {
	case []any:
		for i := 0; i < 3 && i < len(x); i++ {
			if f, ok := floatValue(x[i]); ok {
				out[i] = f
			}
		}
	case string:
		matches := geonotePosRE.FindAllString(x, 3)
		for i, match := range matches {
			if f, err := strconv.ParseFloat(match, 64); err == nil {
				out[i] = f
			}
		}
	}
	return out
}

func geonoteFlag(v any) *bool {
	if v == nil {
		return nil
	}
	var value bool
	switch x := v.(type) {
	case bool:
		value = x
	case string:
		switch strings.ToLower(strings.TrimSpace(x)) {
		case "1", "true", "yes":
			value = true
		case "0", "false", "no", "":
			value = false
		default:
			if n, err := strconv.Atoi(strings.TrimSpace(x)); err == nil {
				value = n != 0
			} else {
				value = x != ""
			}
		}
	case json.Number:
		n, _ := x.Int64()
		value = n != 0
	case float64:
		value = x != 0
	default:
		value = true
	}
	return &value
}

func geonoteClient(n repository.Geonote, viewer string) Object {
	pos := []float64{n.PosX, n.PosY, n.PosZ}
	parent := vec3(n.ParentRelPos, [3]float64{})
	cam := vec3(n.CamDir, [3]float64{})
	return Object{
		"geonote_id": n.GeonoteID, "pool_type": n.PoolType, "pool_name": n.PoolName,
		"message": n.Message, "author_id": n.AuthorID, "author_name": n.AuthorName,
		"created_at": n.CreatedAt, "level_id": n.LevelID, "pos_x": n.PosX, "pos_y": n.PosY, "pos_z": n.PosZ,
		"id": n.GeonoteID, "creator": n.AuthorID, "creator_nickname": n.AuthorName,
		"created_time": n.CreatedAt, "pos": pos, "level_hash": n.LevelHash,
		"parent_guid": n.ParentGUID, "parent_rel_pos": parent, "cam_dir": cam,
		"likes": n.Likes, "liked": n.Liked, "visible": n.Visible, "reported": n.Reported,
		"emoji_id": n.EmojiID, "bug": n.Bug, "is_mine": viewer != "" && n.AuthorID == viewer,
	}
}

func geonotePayload(notes []repository.Geonote, viewer string) Object {
	items := []Object{}
	likes := []Object{}
	social := []Object{}
	for _, n := range notes {
		item := geonoteClient(n, viewer)
		items = append(items, item)
		likes = append(likes, Object{"geonote_id": n.GeonoteID, "id": n.GeonoteID, "likes": n.Likes})
		liked, visible, reported := int64(0), int64(0), int64(0)
		if n.Liked {
			liked = 1
		}
		if n.Visible {
			visible = 1
		}
		if n.Reported {
			reported = 1
		}
		social = append(social, Object{"geonote_id": n.GeonoteID, "id": n.GeonoteID, "liked": liked, "visible": visible, "reported": reported, "emoji_id": n.EmojiID})
	}
	return Object{"geonotes": items, "likes": likes, "emojis": []Object{}, "social": social}
}

func geonoteLevel(m Object) int64 {
	if n := number(m, "level_id", -1); n >= 0 {
		return n
	}
	return number(m, "level", 0)
}

func geonoteLimit(value int64) int64 {
	if value <= 0 {
		return 100
	}
	if value > 200 {
		return 200
	}
	return value
}

func (a *Service) GetUserGeonotes(r Request, m Object) (Object, error) {
	uid, err := a.geonoteReadUser(r, m)
	if err != nil {
		return nil, err
	}
	level := geonoteLevel(m)
	limit := number(m, "limit", 0)
	if limit == 0 {
		limit = number(m, "n", 0)
	}
	if limit == 0 {
		limit = 100
	}
	notes, err := a.store.Geonotes(r.Context, level, level != 0, false, geonoteLimit(limit))
	if err != nil {
		return nil, err
	}
	return geonotePayload(notes, uid), nil
}

func (a *Service) RequestGeonotesForNFriends(r Request, m Object) (Object, error) {
	uid, err := a.geonoteReadUser(r, m)
	if err != nil {
		return nil, err
	}
	level := geonoteLevel(m)
	limit := number(m, "n", 0)
	if limit == 0 {
		limit = number(m, "limit", 0)
	}
	if limit == 0 {
		limit = 50
	}
	notes, err := a.store.Geonotes(r.Context, level, level != 0, false, geonoteLimit(limit))
	if err != nil {
		return nil, err
	}
	return geonotePayload(notes, uid), nil
}

func (a *Service) RequestNBugReportGeonotes(r Request, m Object) (Object, error) {
	uid, err := a.geonoteWriteUser(r, m)
	if err != nil {
		return nil, err
	}
	level := geonoteLevel(m)
	limit := number(m, "n", 0)
	if limit == 0 {
		limit = number(m, "limit", 0)
	}
	if limit == 0 {
		limit = 50
	}
	notes, err := a.store.Geonotes(r.Context, level, level != 0, true, geonoteLimit(limit))
	if err != nil {
		return Object{"result": "error", "error": "internal_error"}, nil
	}
	return geonotePayload(notes, uid), nil
}

func runeLimit(s string, n int) string {
	runes := []rune(s)
	if len(runes) > n {
		runes = runes[:n]
	}
	return string(runes)
}

func (a *Service) CreateGeonote(r Request, m Object) (Object, error) {
	return a.createGeonote(r, m, false)
}

func (a *Service) CreateBugReportGeonote(r Request, m Object) (Object, error) {
	return a.createGeonote(r, m, true)
}

func (a *Service) createGeonote(r Request, m Object, bug bool) (Object, error) {
	uid, err := a.geonoteWriteUser(r, m)
	if err != nil {
		return nil, err
	}
	message := strings.TrimSpace(firstText(m, "message", "msg", "content", "text"))
	if message == "" {
		return Object{"result": "error", "geonote": Object{}}, nil
	}
	level := number(m, "level_id", 0)
	pos := [3]float64{}
	if raw, ok := m["pos"]; ok {
		pos = vec3(raw, [3]float64{})
	} else if raw, ok := m["position"]; ok {
		pos = vec3(raw, [3]float64{})
	} else if raw, ok := m["location"]; ok {
		pos = vec3(raw, [3]float64{})
	} else {
		pos = [3]float64{floatOr(m, "pos_x"), floatOr(m, "pos_y"), floatOr(m, "pos_z")}
	}
	authorName := firstText(m, "author_name", "name", "creator_nickname")
	if authorName == "" {
		if len(uid) > 8 {
			authorName = uid[:8]
		} else {
			authorName = uid
		}
	}
	poolType := text(m, "pool_type")
	if poolType == "" {
		poolType = "SeasonMeditationShrineNormal"
	}
	parentRel := vec3(m["parent_rel_pos"], [3]float64{})
	cam := vec3(m["cam_dir"], [3]float64{})
	parentJSON, _ := json.Marshal(parentRel)
	camJSON, _ := json.Marshal(cam)
	stamp := now()
	n := repository.Geonote{
		GeonoteID: newGeonoteID(), PoolType: poolType, PoolName: text(m, "pool_name"),
		Message: runeLimit(message, 500), AuthorID: uid, AuthorName: authorName,
		CreatedAt: stamp, LevelID: level, LevelHash: number(m, "level_hash", 0),
		PosX: pos[0], PosY: pos[1], PosZ: pos[2], ParentGUID: number(m, "parent_guid", 0),
		ParentRelPos: string(parentJSON), CamDir: string(camJSON), Visible: true, Bug: bug, ServerTime: stamp,
	}
	if err = a.store.CreateGeonote(r.Context, n); err != nil {
		return Object{"result": "error", "geonote": Object{}, "error": "internal_error"}, nil
	}
	notes, err := a.store.Geonotes(r.Context, level, level != 0, bug, 0)
	if err != nil {
		return Object{"result": "error", "geonote": Object{}, "error": "internal_error"}, nil
	}
	result := geonotePayload(notes, uid)
	result["result"] = "ok"
	result["geonote"] = geonoteClient(n, uid)
	return result, nil
}

func floatOr(m Object, key string) float64 {
	if f, ok := floatValue(m[key]); ok {
		return f
	}
	return 0
}

func (a *Service) DeleteGeonote(r Request, m Object) (Object, error) {
	return a.deleteGeonote(r, m, false)
}

func (a *Service) DeleteBugReportGeonote(r Request, m Object) (Object, error) {
	return a.deleteGeonote(r, m, true)
}

func (a *Service) deleteGeonote(r Request, m Object, bugOnly bool) (Object, error) {
	uid, err := a.geonoteWriteUser(r, m)
	if err != nil {
		return nil, err
	}
	gid := text(m, "geonote_id")
	if gid == "" {
		gid = text(m, "id")
	}
	if gid != "" {
		if _, err = a.store.DeleteGeonote(r.Context, gid, uid, bugOnly); err != nil {
			return Object{"result": "error", "error": "internal_error"}, nil
		}
	}
	return Object{"result": "ok"}, nil
}

func (a *Service) EditMessageGeonote(r Request, m Object) (Object, error) {
	uid, err := a.geonoteWriteUser(r, m)
	if err != nil {
		return nil, err
	}
	gid := text(m, "geonote_id")
	if gid == "" {
		gid = text(m, "id")
	}
	message := strings.TrimSpace(firstText(m, "message", "msg"))
	if gid == "" || message == "" {
		level := geonoteLevel(m)
		notes, _ := a.store.Geonotes(r.Context, level, level != 0, false, 0)
		result := geonotePayload(notes, uid)
		result["result"] = "ok"
		result["geonote"] = Object{}
		return result, nil
	}
	n, found, err := a.store.EditGeonote(r.Context, gid, uid, runeLimit(message, 500), now())
	if err != nil {
		return Object{"result": "error", "geonote": Object{}, "error": "internal_error"}, nil
	}
	level := geonoteLevel(m)
	if found {
		level = n.LevelID
	}
	notes, err := a.store.Geonotes(r.Context, level, level != 0, found && n.Bug, 0)
	if err != nil {
		return Object{"result": "error", "geonote": Object{}, "error": "internal_error"}, nil
	}
	result := geonotePayload(notes, uid)
	result["result"] = "ok"
	if found {
		result["geonote"] = geonoteClient(n, uid)
	} else {
		result["geonote"] = Object{}
	}
	return result, nil
}

func (a *Service) SocialGeonote(r Request, m Object) (Object, error) {
	uid, err := a.geonoteWriteUser(r, m)
	if err != nil {
		return nil, err
	}
	gid := text(m, "geonote_id")
	if gid == "" {
		gid = text(m, "id")
	}
	var liked, visible, reported *bool
	if v, ok := m["liked"]; ok {
		liked = geonoteFlag(v)
	}
	if v, ok := m["visible"]; ok {
		visible = geonoteFlag(v)
	}
	if v, ok := m["reported"]; ok {
		reported = geonoteFlag(v)
	}
	var emoji *int64
	if v, ok := m["emoji_id"]; ok && v != nil && scalarString(v) != "" {
		n := number(m, "emoji_id", 0)
		emoji = &n
	}
	level := geonoteLevel(m)
	bug := false
	if gid != "" {
		n, found, err := a.store.MutateGeonote(r.Context, gid, liked, visible, reported, emoji, now())
		if err != nil {
			return Object{"result": "error", "error": "internal_error"}, nil
		}
		if found {
			level = n.LevelID
			bug = n.Bug
		}
	}
	notes, err := a.store.Geonotes(r.Context, level, level != 0, bug, 0)
	if err != nil {
		return Object{"result": "error", "error": "internal_error"}, nil
	}
	result := geonotePayload(notes, uid)
	result["result"] = "ok"
	return result, nil
}
