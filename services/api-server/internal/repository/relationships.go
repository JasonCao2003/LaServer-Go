package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

func nowEpoch() int64 { return time.Now().Unix() }

const defaultFriendAbilities = "[1,2,9,10,19]"

// RelationEntry is one owner-scoped friendship row joined with the reverse
// direction, matching the fields the client reads in compact friend payloads.
type RelationEntry struct {
	ID, Nickname, Abilities, Hints      string
	Given, Received, Level, WhenCreated int64
	Badge                               int64
	SelfBlocked, TheyBlocked, Removed   bool
	LocalDeleted                        bool
	Blocked                             bool
}

type CandidateFriend struct {
	RelationEntry
	Platform, Device string
	CreatedAt        sql.NullString
}

const relationFields = `fs.friend_id,COALESCE(fs.nickname,''),COALESCE(fs.abilities,'[]'),COALESCE(fs.hints,'[]'),COALESCE(fs.given,0),COALESCE(fs.recvd,0),COALESCE(fs.level,1),COALESCE(fs.when_created,0),COALESCE(fs.player_badge_type,0),COALESCE(f.status='blocked',0),COALESCE(r.status='blocked',0),COALESCE(f.status='removed',0),COALESCE(fs.local_soft_deleted,0)`

const relationSelect = `SELECT ` + relationFields + ` FROM friend_status fs LEFT JOIN friendship f ON f.user_id=fs.user_id AND f.friend_id=fs.friend_id LEFT JOIN friendship r ON r.user_id=fs.friend_id AND r.friend_id=fs.user_id`

func scanRelation(scanner rowScanner) (RelationEntry, error) {
	var e RelationEntry
	err := scanner.Scan(&e.ID, &e.Nickname, &e.Abilities, &e.Hints, &e.Given, &e.Received, &e.Level, &e.WhenCreated, &e.Badge, &e.SelfBlocked, &e.TheyBlocked, &e.Removed, &e.LocalDeleted)
	return e, err
}

func (s *Store) Relation(ctx context.Context, userID, friendID string) (RelationEntry, bool, error) {
	e, err := scanRelation(s.QueryRowContext(ctx, Statement(relationSelect+` WHERE fs.user_id=? AND fs.friend_id=?`), userID, friendID))
	if err == sql.ErrNoRows {
		return RelationEntry{}, false, nil
	}
	if err != nil {
		return RelationEntry{}, false, err
	}
	return e, true, nil
}

func (s *Store) BlockedFriends(ctx context.Context, userID string, limit, offset int64) ([]CandidateFriend, int64, error) {
	var total int64
	if err := s.QueryRowContext(ctx, `SELECT count(*) FROM friendship WHERE user_id=? AND status='blocked'`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.QueryContext(ctx, Statement(`SELECT `+relationFields+`,COALESCE(NULLIF(u.platform,''),'unknown'),COALESCE(u.device,''),strftime('%Y-%m-%dT%H:%M:%S',f.created_at) FROM friend_status fs JOIN friendship f ON f.user_id=fs.user_id AND f.friend_id=fs.friend_id JOIN user u ON u.id=fs.friend_id LEFT JOIN friendship r ON r.user_id=fs.friend_id AND r.friend_id=fs.user_id WHERE fs.user_id=? AND f.status='blocked' ORDER BY f.updated_at DESC LIMIT ? OFFSET ?`), userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []CandidateFriend{}
	for rows.Next() {
		var e CandidateFriend
		if err = rows.Scan(&e.ID, &e.Nickname, &e.Abilities, &e.Hints, &e.Given, &e.Received, &e.Level, &e.WhenCreated, &e.Badge, &e.SelfBlocked, &e.TheyBlocked, &e.Removed, &e.LocalDeleted, &e.Platform, &e.Device, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}

// ensureOneRelationship creates the owner-scoped friendship and status rows in
// one direction. A blocked or removed row is never promoted to active.
func ensureOneRelationship(ctx context.Context, tx *Tx, owner, friend string, stamp int64) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO friendship(user_id,friend_id,status) VALUES(?,?,'active') ON CONFLICT(user_id,friend_id) DO UPDATE SET status='active',updated_at=CURRENT_TIMESTAMP WHERE friendship.status='pending'`, owner, friend); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO friend_status(user_id,friend_id,given,recvd,nickname,when_created,level,abilities,hints,outfit,player_badge_type,local_soft_deleted,attitude) VALUES(?,?,0,0,'',?,1,?,'[]','{}',0,0,'0') ON CONFLICT(user_id,friend_id) DO NOTHING`, owner, friend, stamp, defaultFriendAbilities)
	return err
}

func addHints(raw string) []int64 {
	var values []json.RawMessage
	if json.Unmarshal([]byte(raw), &values) != nil {
		return []int64{}
	}
	out := []int64{}
	for _, v := range values {
		s := string(v)
		var text string
		if json.Unmarshal(v, &text) == nil {
			s = text
		}
		var n int64
		if json.Unmarshal([]byte(s), &n) == nil {
			out = append(out, n)
		}
	}
	return out
}

func (s *Store) SaveFriendName(ctx context.Context, userID, friendID, name string) (RelationEntry, bool, error) {
	exists, err := s.HasUser(ctx, friendID)
	if err != nil || !exists {
		return RelationEntry{}, false, err
	}
	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return RelationEntry{}, false, err
	}
	defer tx.Rollback()
	stamp := nowEpoch()
	if err = ensureOneRelationship(ctx, tx, userID, friendID, stamp); err != nil {
		return RelationEntry{}, false, err
	}
	if err = ensureOneRelationship(ctx, tx, friendID, userID, stamp); err != nil {
		return RelationEntry{}, false, err
	}
	var abilities, hints string
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(abilities,'[]'),COALESCE(hints,'[]') FROM friend_status WHERE user_id=? AND friend_id=?`, userID, friendID).Scan(&abilities, &hints); err != nil {
		return RelationEntry{}, false, err
	}
	if len(addHints(abilities)) == 0 {
		abilities = defaultFriendAbilities
	}
	kept := []int64{}
	for _, h := range addHints(hints) {
		if h != 1 {
			kept = append(kept, h)
		}
	}
	encoded, _ := json.Marshal(kept)
	if _, err = tx.ExecContext(ctx, `UPDATE friend_status SET nickname=?,abilities=?,hints=?,local_soft_deleted=0,updated_at=CURRENT_TIMESTAMP WHERE user_id=? AND friend_id=?`, name, abilities, string(encoded), userID, friendID); err != nil {
		return RelationEntry{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return RelationEntry{}, false, err
	}
	e, found, err := s.Relation(ctx, userID, friendID)
	return e, found, err
}

func (s *Store) SaveFriendBlock(ctx context.Context, userID, friendID string, blocked bool) (RelationEntry, bool, error) {
	exists, err := s.HasUser(ctx, friendID)
	if err != nil || !exists {
		return RelationEntry{ID: friendID}, false, err
	}
	status := "active"
	if blocked {
		status = "blocked"
	}
	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return RelationEntry{}, false, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO friendship(user_id,friend_id,status) VALUES(?,?,?) ON CONFLICT(user_id,friend_id) DO UPDATE SET status=excluded.status,updated_at=CURRENT_TIMESTAMP`, userID, friendID, status); err != nil {
		return RelationEntry{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return RelationEntry{}, false, err
	}
	e, found, err := s.Relation(ctx, userID, friendID)
	return e, found, err
}

func (s *Store) RemoveFriendHint(ctx context.Context, userID, friendID string, hint int64) (RelationEntry, bool, error) {
	e, found, err := s.Relation(ctx, userID, friendID)
	if err != nil || !found {
		return e, found, err
	}
	hints := addHints(e.Hints)
	kept := make([]int64, 0, len(hints))
	removed := false
	for _, h := range hints {
		if h == hint {
			removed = true
			continue
		}
		kept = append(kept, h)
	}
	if !removed {
		return e, true, nil
	}
	encoded, _ := json.Marshal(kept)
	if _, err = s.ExecContext(ctx, `UPDATE friend_status SET hints=?,updated_at=CURRENT_TIMESTAMP WHERE user_id=? AND friend_id=?`, string(encoded), userID, friendID); err != nil {
		return e, false, err
	}
	e, found, err = s.Relation(ctx, userID, friendID)
	return e, found, err
}

type ExternalFriendRecord struct {
	ExternalAccountID, Platform, Nickname string
	FriendUserID, Avatar                  sql.NullString
	IsConnected                           bool
	ConnectedAt                           sql.NullInt64
}

func (s *Store) ExternalFriends(ctx context.Context, userID string, limit int64) ([]ExternalFriendRecord, error) {
	rows, err := s.QueryContext(ctx, `SELECT external_account_id,external_platform,friend_nickname,friend_user_id,friend_avatar,COALESCE(is_connected,0),connected_at FROM external_account_friend WHERE user_id=? LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ExternalFriendRecord{}
	for rows.Next() {
		var r ExternalFriendRecord
		if err = rows.Scan(&r.ExternalAccountID, &r.Platform, &r.Nickname, &r.FriendUserID, &r.Avatar, &r.IsConnected, &r.ConnectedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type SerendipityRecord struct {
	ID                               int64
	MatchedUserID, MatchType, Status string
	MatchScore                       float64
	MatchedAt                        int64
	ExpiresAt                        sql.NullInt64
}

func (s *Store) SerendipityMatches(ctx context.Context, userID string, limit, stamp int64) ([]SerendipityRecord, error) {
	rows, err := s.QueryContext(ctx, `SELECT sm.id,sm.matched_user_id,sm.match_type,COALESCE(sm.match_score,0),sm.matched_at,COALESCE(sm.status,'active'),sm.expires_at FROM serendipity_match sm JOIN user u ON u.id=sm.matched_user_id WHERE sm.user_id=? AND sm.status='active' AND (sm.expires_at IS NULL OR sm.expires_at>?) ORDER BY sm.match_score DESC,sm.id LIMIT ?`, userID, stamp, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SerendipityRecord{}
	for rows.Next() {
		var r SerendipityRecord
		if err = rows.Scan(&r.ID, &r.MatchedUserID, &r.MatchType, &r.MatchScore, &r.MatchedAt, &r.Status, &r.ExpiresAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type StarTagDefinition struct {
	Name, SKU, Part, ConsumableName, GroupName, Batches, CanLinkWith string
}
type StarTagLink struct {
	TagName, LinkedItem string
	LinkedAt            int64
}

func (s *Store) StarTags(ctx context.Context, userID string) ([]StarTagDefinition, []StarTagLink, error) {
	rows, err := s.QueryContext(ctx, `SELECT name,sku,COALESCE(part,''),consumable_name,COALESCE(group_name,''),COALESCE(batches,''),COALESCE(can_link_with,'') FROM star_tag_def ORDER BY id`)
	if err != nil {
		return nil, nil, err
	}
	defs := []StarTagDefinition{}
	for rows.Next() {
		var d StarTagDefinition
		if err = rows.Scan(&d.Name, &d.SKU, &d.Part, &d.ConsumableName, &d.GroupName, &d.Batches, &d.CanLinkWith); err != nil {
			rows.Close()
			return nil, nil, err
		}
		defs = append(defs, d)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, nil, err
	}
	rows.Close()
	rows, err = s.QueryContext(ctx, `SELECT tag_name,linked_item,linked_at FROM star_tag_link WHERE user_id=? ORDER BY id`, userID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	links := []StarTagLink{}
	for rows.Next() {
		var l StarTagLink
		if err = rows.Scan(&l.TagName, &l.LinkedItem, &l.LinkedAt); err != nil {
			return nil, nil, err
		}
		links = append(links, l)
	}
	return defs, links, rows.Err()
}

type ServerAssignment struct {
	ServerURI, Signature, Sig string
	MoveTS                    int64
	PSOnly                    bool
	PlatformFilter            int64
}

func (s *Store) LatestAssignment(ctx context.Context, userID string) (ServerAssignment, bool, error) {
	var a ServerAssignment
	err := s.QueryRowContext(ctx, `SELECT server_uri,move_ts,signature,sig,COALESCE(ps_only,0),COALESCE(platform_filter,0) FROM server_assignment WHERE user_id=? ORDER BY move_ts DESC,id DESC LIMIT 1`, userID).Scan(&a.ServerURI, &a.MoveTS, &a.Signature, &a.Sig, &a.PSOnly, &a.PlatformFilter)
	if err == sql.ErrNoRows {
		return a, false, nil
	}
	return a, err == nil, err
}

// AssignServer records the room a player was last sent to so friends can join.
func (s *Store) AssignServer(ctx context.Context, userID string, a ServerAssignment) error {
	_, err := s.ExecContext(ctx, `INSERT INTO server_assignment(user_id,server_uri,move_ts,signature,sig,ps_only,platform_filter) VALUES(?,?,?,?,?,?,?)`, userID, a.ServerURI, a.MoveTS, a.Signature, a.Sig, a.PSOnly, a.PlatformFilter)
	return err
}

func (s *Store) SaveJoinAssignment(ctx context.Context, userID string, a ServerAssignment, friendID string) error {
	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO server_assignment(user_id,server_uri,move_ts,signature,sig,ps_only,platform_filter) VALUES(?,?,?,?,?,?,?)`, userID, a.ServerURI, a.MoveTS, a.Signature, a.Sig, a.PSOnly, a.PlatformFilter); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO user_online_status(user_id,friend_id,is_online,last_seen) VALUES(?,?,1,CURRENT_TIMESTAMP) ON CONFLICT(user_id,friend_id) DO UPDATE SET is_online=1,last_seen=CURRENT_TIMESTAMP`, userID, friendID); err != nil {
		return err
	}
	return tx.Commit()
}

type ConstellationItem struct {
	FriendID string
	Node     int64
}

// SaveConstellationPages replaces only the page indexes present in pages.
func (s *Store) SaveConstellationPages(ctx context.Context, userID string, pages [][]ConstellationItem) error {
	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for index, friends := range pages {
		if _, err = tx.ExecContext(ctx, `DELETE FROM constellation_friend_page WHERE user_id=? AND page_index=?`, userID, index); err != nil {
			return err
		}
		for _, f := range friends {
			if _, err = tx.ExecContext(ctx, `INSERT INTO constellation_friend_page(user_id,page_index,friend_id,constellation_node_index,created_at,updated_at) VALUES(?,?,?,?,?,?)`, userID, index, f.FriendID, f.Node, nowEpoch(), nowEpoch()); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (s *Store) FollowerCount(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := s.QueryRowContext(ctx, `SELECT count(*) FROM follow_relationship WHERE following_id=? AND status='active'`, userID).Scan(&count)
	return count, err
}

func (s *Store) UpsertBoatFollow(ctx context.Context, userID, creator, nickname, lastModified string) error {
	_, err := s.ExecContext(ctx, `INSERT INTO boat_follow(user_id,creator,nickname,last_modified) VALUES(?,?,?,?) ON CONFLICT(user_id,creator) DO UPDATE SET nickname=excluded.nickname,last_modified=excluded.last_modified`, userID, creator, nickname, lastModified)
	return err
}

func (s *Store) DeleteBoatFollow(ctx context.Context, userID, creator string) (int64, error) {
	result, err := s.ExecContext(ctx, `DELETE FROM boat_follow WHERE user_id=? AND creator=?`, userID, creator)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// cleanCreator mirrors the legacy split(":")[0] behavior for creator ids.
func CleanCreator(creator string) string {
	if index := strings.IndexByte(creator, ':'); index >= 0 {
		return creator[:index]
	}
	return creator
}
