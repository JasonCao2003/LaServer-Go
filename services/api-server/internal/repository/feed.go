package repository

import (
	"context"
	"database/sql"
	"strings"
)

type FeedPost struct {
	ID                                       int64
	UserID, SocialFeedID, PoolType, PoolName string
	Message                                  string
	CreatedAt, ExpireAt                      int64
	Likes, Comments, Reports, Hides          int64
	Attempts, Succeeds                       int64
	Poisoned                                 bool
}

type FeedExtra struct {
	SocialFeedID    string
	ExtraData       string
	ResourceID      string
	CommentsEnabled bool
	Tags            string
	Private         bool
}

const feedColumns = `id,user_id,social_feed_id,pool_type,pool_name,message,created_at,expire_at,likes_count,comments_count,report_count,hide_count,attempt_count,succeed_count,is_poisoned`

func scanFeed(scanner rowScanner) (FeedPost, error) {
	var p FeedPost
	err := scanner.Scan(&p.ID, &p.UserID, &p.SocialFeedID, &p.PoolType, &p.PoolName, &p.Message, &p.CreatedAt, &p.ExpireAt, &p.Likes, &p.Comments, &p.Reports, &p.Hides, &p.Attempts, &p.Succeeds, &p.Poisoned)
	return p, err
}

func (s *Store) CreateFeed(ctx context.Context, p FeedPost) error {
	_, err := s.ExecContext(ctx, `INSERT INTO social_feed_post(user_id,social_feed_id,pool_type,pool_name,message,created_at,expire_at,likes_count,comments_count,report_count,hide_count,attempt_count,succeed_count,is_poisoned) VALUES(?,?,?,?,?,?,?,0,0,0,0,0,0,0)`, p.UserID, p.SocialFeedID, p.PoolType, p.PoolName, p.Message, p.CreatedAt, p.ExpireAt)
	return err
}

func (s *Store) FeedByID(ctx context.Context, socialFeedID string) (FeedPost, bool, error) {
	p, err := scanFeed(s.QueryRowContext(ctx, `SELECT `+feedColumns+` FROM social_feed_post WHERE social_feed_id=?`, socialFeedID))
	if err == sql.ErrNoRows {
		return p, false, nil
	}
	return p, err == nil, err
}

func (s *Store) Feeds(ctx context.Context, poolTypes, poolNames []string, skip, limit int64, includeZeroExpiry bool, stamp int64) ([]FeedPost, int64, error) {
	where := []string{}
	args := []any{}
	if includeZeroExpiry {
		where = append(where, `(expire_at=0 OR expire_at>?)`)
	} else {
		where = append(where, `expire_at>?`)
	}
	args = append(args, stamp)
	if len(poolTypes) > 0 {
		where = append(where, `pool_type IN (`+placeholders(len(poolTypes))+`)`)
		for _, v := range poolTypes {
			args = append(args, v)
		}
	}
	if len(poolNames) > 0 {
		where = append(where, `pool_name IN (`+placeholders(len(poolNames))+`)`)
		for _, v := range poolNames {
			args = append(args, v)
		}
	}
	clause := strings.Join(where, " AND ")
	var total int64
	if err := s.QueryRowContext(ctx, Statement(`SELECT count(*) FROM social_feed_post WHERE `+clause), args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	query := `SELECT ` + feedColumns + ` FROM social_feed_post WHERE ` + clause + ` ORDER BY created_at DESC,id DESC`
	queryArgs := append([]any{}, args...)
	if limit > 0 {
		query += ` LIMIT ? OFFSET ?`
		queryArgs = append(queryArgs, limit, skip)
	}
	rows, err := s.QueryContext(ctx, Statement(query), queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []FeedPost{}
	for rows.Next() {
		p, err := scanFeed(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}

func (s *Store) FeedsByIDs(ctx context.Context, ids []string) ([]FeedPost, error) {
	if len(ids) == 0 {
		return []FeedPost{}, nil
	}
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := s.QueryContext(ctx, Statement(`SELECT `+feedColumns+` FROM social_feed_post WHERE social_feed_id IN (`+placeholders(len(ids))+`)`), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FeedPost{}
	for rows.Next() {
		p, err := scanFeed(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) DeleteFeed(ctx context.Context, socialFeedID string) (bool, error) {
	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `DELETE FROM social_feed_like WHERE social_feed_id=?`, socialFeedID); err != nil {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM social_feed_extra WHERE social_feed_id=?`, socialFeedID); err != nil {
		return false, err
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM social_feed_post WHERE social_feed_id=?`, socialFeedID)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *Store) UpdateFeed(ctx context.Context, id, message, poolType, poolName string, expireAt int64) error {
	_, err := s.ExecContext(ctx, `UPDATE social_feed_post SET message=COALESCE(NULLIF(?,''),message),pool_type=COALESCE(NULLIF(?,''),pool_type),pool_name=COALESCE(NULLIF(?,''),pool_name),expire_at=? WHERE social_feed_id=?`, message, poolType, poolName, expireAt, id)
	return err
}

func (s *Store) UpdateFeedCounters(ctx context.Context, id string, likes, hides int64) error {
	_, err := s.ExecContext(ctx, `UPDATE social_feed_post SET likes_count=?,hide_count=? WHERE social_feed_id=?`, likes, hides, id)
	return err
}

func (s *Store) FeedLikeExists(ctx context.Context, userID, socialFeedID string) (bool, error) {
	var n int
	err := s.QueryRowContext(ctx, `SELECT 1 FROM social_feed_like WHERE user_id=? AND social_feed_id=?`, userID, socialFeedID).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (s *Store) AddFeedLike(ctx context.Context, userID, socialFeedID string, stamp int64) error {
	_, err := s.ExecContext(ctx, `INSERT INTO social_feed_like(user_id,social_feed_id,created_at) VALUES(?,?,?) ON CONFLICT(user_id,social_feed_id) DO NOTHING`, userID, socialFeedID, stamp)
	return err
}

func (s *Store) RemoveFeedLike(ctx context.Context, userID, socialFeedID string) error {
	_, err := s.ExecContext(ctx, `DELETE FROM social_feed_like WHERE user_id=? AND social_feed_id=?`, userID, socialFeedID)
	return err
}

func (s *Store) FeedExtra(ctx context.Context, socialFeedID string) (FeedExtra, bool, error) {
	var e FeedExtra
	var tags sql.NullString
	err := s.QueryRowContext(ctx, `SELECT social_feed_id,extra_data,resource_id,comments_enabled,tags,private FROM social_feed_extra WHERE social_feed_id=?`, socialFeedID).Scan(&e.SocialFeedID, &e.ExtraData, &e.ResourceID, &e.CommentsEnabled, &tags, &e.Private)
	if err == sql.ErrNoRows {
		return e, false, nil
	}
	if err != nil {
		return e, false, err
	}
	if tags.Valid {
		e.Tags = tags.String
	}
	return e, true, nil
}

func (s *Store) SaveFeedExtra(ctx context.Context, e FeedExtra) error {
	var tags any
	if e.Tags != "" {
		tags = e.Tags
	}
	_, err := s.ExecContext(ctx, `INSERT INTO social_feed_extra(social_feed_id,extra_data,resource_id,comments_enabled,tags,private) VALUES(?,?,?,?,?,?) ON CONFLICT(social_feed_id) DO UPDATE SET extra_data=excluded.extra_data,resource_id=excluded.resource_id,comments_enabled=excluded.comments_enabled,tags=excluded.tags,private=excluded.private,updated_at=CURRENT_TIMESTAMP`, e.SocialFeedID, e.ExtraData, e.ResourceID, e.CommentsEnabled, tags, e.Private)
	return err
}

func (s *Store) FeedIDForResource(ctx context.Context, resourceID string) (string, bool, error) {
	var id string
	err := s.QueryRowContext(ctx, `SELECT social_feed_id FROM social_feed_extra WHERE resource_id=? ORDER BY updated_at DESC LIMIT 1`, resourceID).Scan(&id)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	return id, err == nil, err
}

func (s *Store) ResourceIDForFeed(ctx context.Context, socialFeedID string) (string, bool, error) {
	var resource string
	err := s.QueryRowContext(ctx, `SELECT resource_id FROM social_feed_extra WHERE social_feed_id=?`, feedKey(socialFeedID)).Scan(&resource)
	if err == sql.ErrNoRows || resource == "" {
		return "", false, nil
	}
	return resource, err == nil, err
}

func feedKey(id string) string {
	if index := strings.IndexByte(id, ':'); index >= 0 {
		return id[:index]
	}
	return id
}

func (s *Store) AreFriends(ctx context.Context, a, b string) (bool, error) {
	var n int
	err := s.QueryRowContext(ctx, `SELECT 1 FROM friendship WHERE status='active' AND ((user_id=? AND friend_id=?) OR (user_id=? AND friend_id=?)) LIMIT 1`, a, b, b, a).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}
