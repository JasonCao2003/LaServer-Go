package repository

import (
	"context"
	"database/sql"
	"strings"
)

type FriendRecord struct {
	ID, Nickname, Platform, Device, Abilities, Hints, Attitude string
	Created                                                    sql.NullString
	Given, Received, Level, WhenCreated, Badge                 int64
	BlockedByFriend, Removed, LocalDeleted                     bool
}

func (s *Store) Friends(ctx context.Context, userID string, players []string, limit, offset int64) ([]FriendRecord, error) {
	query := `SELECT u.id,COALESCE(fs.nickname,''),COALESCE(NULLIF(u.platform,''),'unknown'),COALESCE(u.device,''),COALESCE(fs.abilities,'[]'),COALESCE(fs.hints,'[]'),COALESCE(fs.attitude,'0'),strftime('%Y-%m-%dT%H:%M:%S',f.created_at),COALESCE(fs.given,0),COALESCE(fs.recvd,0),COALESCE(fs.level,1),COALESCE(fs.when_created,0),COALESCE(fs.player_badge_type,0),COALESCE(reverse.status='blocked',0),COALESCE(reverse.status='removed',0),COALESCE(fs.local_soft_deleted,0) FROM friendship f JOIN user u ON u.id=f.friend_id LEFT JOIN friend_status fs ON fs.user_id=f.user_id AND fs.friend_id=f.friend_id LEFT JOIN friendship reverse ON reverse.user_id=f.friend_id AND reverse.friend_id=f.user_id WHERE f.user_id=? AND f.status='active'`
	args := []any{userID}
	if len(players) > 0 {
		query += " AND f.friend_id IN (" + strings.TrimSuffix(strings.Repeat("?,", len(players)), ",") + ")"
		for _, id := range players {
			args = append(args, id)
		}
	}
	query += " ORDER BY f.updated_at DESC,f.id LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := []FriendRecord{}
	for rows.Next() {
		var f FriendRecord
		if err = rows.Scan(&f.ID, &f.Nickname, &f.Platform, &f.Device, &f.Abilities, &f.Hints, &f.Attitude, &f.Created, &f.Given, &f.Received, &f.Level, &f.WhenCreated, &f.Badge, &f.BlockedByFriend, &f.Removed, &f.LocalDeleted); err != nil {
			return nil, err
		}
		records = append(records, f)
	}
	return records, rows.Err()
}
