package repository

import (
	"context"
	"database/sql"
)

type Invite struct {
	InviterUserID string
	TokenID       string
	FormattedCode *string
	Nickname      string
	FriendID      *string
	Created       int64
}

func (s *Store) InviteByToken(ctx context.Context, tokenID string) (Invite, error) {
	var item Invite
	var code, friend sql.NullString
	err := s.QueryRowContext(ctx, Statement(`SELECT inviter_user_id,token_id,formatted_code,nickname,friend_id,created FROM invite WHERE token_id=?`), tokenID).Scan(&item.InviterUserID, &item.TokenID, &code, &item.Nickname, &friend, &item.Created)
	if code.Valid {
		item.FormattedCode = &code.String
	}
	if friend.Valid {
		item.FriendID = &friend.String
	}
	return item, err
}

func (s *Store) DeleteInvite(ctx context.Context, tokenID, ownerID string) (bool, error) {
	result, err := s.ExecContext(ctx, Statement(`DELETE FROM invite WHERE token_id=? AND inviter_user_id=?`), tokenID, ownerID)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n > 0, err
}

func (s *Store) CreateInvite(ctx context.Context, inviterID, tokenID, formattedCode, nickname string, created int64) error {
	_, err := s.ExecContext(ctx, Statement(`INSERT INTO invite(inviter_user_id,token_id,formatted_code,nickname,created) VALUES(?,?,?,?,?)`), inviterID, tokenID, formattedCode, nickname, created)
	return err
}

func (s *Store) AcceptInvite(ctx context.Context, tokenID, userID, inviterID, nickname string, created int64) (bool, error) {
	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, Statement(`UPDATE invite SET friend_id=?,updated_at=CURRENT_TIMESTAMP WHERE token_id=? AND friend_id IS NULL`), userID, tokenID)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil
	}
	const abilities = `[19,6,9,2,27,1,10]`
	insertStatus := Statement(`INSERT INTO friend_status(user_id,friend_id,given,recvd,nickname,when_created,level,abilities,hints,outfit,player_badge_type,local_soft_deleted,attitude) VALUES(?,?,?,?,?,?,1,?,'[]','{}',0,0,'0')`)
	if _, err = tx.ExecContext(ctx, insertStatus, userID, inviterID, 0, 1, nickname, created, abilities); err != nil {
		return false, err
	}
	defaultNickname := userID
	if len(defaultNickname) > 8 {
		defaultNickname = defaultNickname[:8]
	}
	if _, err = tx.ExecContext(ctx, insertStatus, inviterID, userID, 1, 0, defaultNickname, created, abilities); err != nil {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, Statement(`INSERT INTO friendship(user_id,friend_id,status) VALUES(?,?,'active')`), userID, inviterID); err != nil {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, Statement(`INSERT INTO friendship(user_id,friend_id,status) VALUES(?,?,'active')`), inviterID, userID); err != nil {
		return false, err
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) FriendCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := s.QueryRowContext(ctx, Statement(`SELECT count(*) FROM friend_status WHERE user_id=?`), userID).Scan(&n)
	return n, err
}

func (s *Store) Invites(ctx context.Context, userID string) ([]Invite, error) {
	rows, err := s.QueryContext(ctx, Statement(`SELECT inviter_user_id,token_id,nickname,friend_id,created FROM invite WHERE inviter_user_id=? ORDER BY id`), userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Invite{}
	for rows.Next() {
		var item Invite
		var friend sql.NullString
		if err = rows.Scan(&item.InviterUserID, &item.TokenID, &item.Nickname, &friend, &item.Created); err != nil {
			return nil, err
		}
		if friend.Valid {
			item.FriendID = &friend.String
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
