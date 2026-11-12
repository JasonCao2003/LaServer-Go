package repository

import (
	"context"
	"database/sql"
)

type Geonote struct {
	ID                   int64
	GeonoteID            string
	PoolType, PoolName   string
	Message              string
	AuthorID, AuthorName string
	CreatedAt            int64
	EditedAt             sql.NullInt64
	LevelID, LevelHash   int64
	PosX, PosY, PosZ     float64
	ParentGUID           int64
	ParentRelPos, CamDir string
	Likes, EmojiID       int64
	Liked, Visible       bool
	Reported, Bug        bool
	ServerTime           int64
}

const geonoteColumns = `id,geonote_id,pool_type,pool_name,message,author_id,author_name,created_at,edited_at,level_id,level_hash,pos_x,pos_y,pos_z,parent_guid,parent_rel_pos,cam_dir,likes,liked,visible,reported,emoji_id,bug,server_time`

func scanGeonote(scanner rowScanner) (Geonote, error) {
	var n Geonote
	err := scanner.Scan(&n.ID, &n.GeonoteID, &n.PoolType, &n.PoolName, &n.Message, &n.AuthorID, &n.AuthorName, &n.CreatedAt, &n.EditedAt, &n.LevelID, &n.LevelHash, &n.PosX, &n.PosY, &n.PosZ, &n.ParentGUID, &n.ParentRelPos, &n.CamDir, &n.Likes, &n.Liked, &n.Visible, &n.Reported, &n.EmojiID, &n.Bug, &n.ServerTime)
	return n, err
}

func (s *Store) CreateGeonote(ctx context.Context, n Geonote) error {
	_, err := s.ExecContext(ctx, `INSERT INTO geonote(geonote_id,pool_type,pool_name,message,author_id,author_name,created_at,level_id,level_hash,pos_x,pos_y,pos_z,parent_guid,parent_rel_pos,cam_dir,likes,liked,visible,reported,emoji_id,bug,server_time) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,0,0,1,0,0,?,?)`, n.GeonoteID, n.PoolType, n.PoolName, n.Message, n.AuthorID, n.AuthorName, n.CreatedAt, n.LevelID, n.LevelHash, n.PosX, n.PosY, n.PosZ, n.ParentGUID, n.ParentRelPos, n.CamDir, n.Bug, n.ServerTime)
	return err
}

func (s *Store) Geonotes(ctx context.Context, levelID int64, filterLevel, bug bool, limit int64) ([]Geonote, error) {
	query := `SELECT ` + geonoteColumns + ` FROM geonote WHERE bug=?`
	args := []any{bug}
	if filterLevel {
		query += ` AND level_id=?`
		args = append(args, levelID)
	}
	query += ` ORDER BY created_at DESC,id DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.QueryContext(ctx, Statement(query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Geonote{}
	for rows.Next() {
		n, err := scanGeonote(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *Store) GeonoteByID(ctx context.Context, geonoteID string) (Geonote, bool, error) {
	n, err := scanGeonote(s.QueryRowContext(ctx, `SELECT `+geonoteColumns+` FROM geonote WHERE geonote_id=?`, geonoteID))
	if err == sql.ErrNoRows {
		return n, false, nil
	}
	return n, err == nil, err
}

func (s *Store) DeleteGeonote(ctx context.Context, geonoteID, authorID string, bugOnly bool) (bool, error) {
	query := `DELETE FROM geonote WHERE geonote_id=? AND author_id=?`
	if bugOnly {
		query += ` AND bug=1`
	}
	result, err := s.ExecContext(ctx, Statement(query), geonoteID, authorID)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n > 0, err
}

func (s *Store) EditGeonote(ctx context.Context, geonoteID, authorID, message string, stamp int64) (Geonote, bool, error) {
	result, err := s.ExecContext(ctx, `UPDATE geonote SET message=?,edited_at=? WHERE geonote_id=? AND author_id=?`, message, stamp, geonoteID, authorID)
	if err != nil {
		return Geonote{}, false, err
	}
	n, err := result.RowsAffected()
	if err != nil || n == 0 {
		return Geonote{}, false, err
	}
	return s.GeonoteByID(ctx, geonoteID)
}

// MutateGeonote applies global like/visibility/report/emoji changes. Likes only
// move on a false->true or true->false transition, matching the legacy store.
func (s *Store) MutateGeonote(ctx context.Context, geonoteID string, liked, visible, reported *bool, emojiID *int64, stamp int64) (Geonote, bool, error) {
	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return Geonote{}, false, err
	}
	defer tx.Rollback()
	n, err := scanGeonote(tx.QueryRowContext(ctx, `SELECT `+geonoteColumns+` FROM geonote WHERE geonote_id=?`, geonoteID))
	if err == sql.ErrNoRows {
		return Geonote{}, false, nil
	}
	if err != nil {
		return Geonote{}, false, err
	}
	if liked != nil {
		if *liked && !n.Liked {
			n.Likes++
		} else if !*liked && n.Liked && n.Likes > 0 {
			n.Likes--
		}
		n.Liked = *liked
	}
	if visible != nil {
		n.Visible = *visible
	}
	if reported != nil {
		n.Reported = *reported
	}
	if emojiID != nil {
		n.EmojiID = *emojiID
	}
	if _, err = tx.ExecContext(ctx, `UPDATE geonote SET likes=?,liked=?,visible=?,reported=?,emoji_id=?,edited_at=? WHERE id=?`, n.Likes, n.Liked, n.Visible, n.Reported, n.EmojiID, stamp, n.ID); err != nil {
		return Geonote{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return Geonote{}, false, err
	}
	n.EditedAt.Int64 = stamp
	n.EditedAt.Valid = true
	return n, true, nil
}
