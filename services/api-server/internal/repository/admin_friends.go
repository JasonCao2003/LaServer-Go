package repository

import (
	"context"
	"database/sql"
)

type AdminFriendRow struct {
	FriendID, Status, ReverseStatus string
	Nickname                        string
	Given, Received                 int64
	Remark, DeviceName              string
	TwoWay                          bool
}

func (s *Store) AdminFriendIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.QueryContext(ctx, `SELECT friend_id FROM friendship WHERE user_id=? UNION SELECT user_id FROM friendship WHERE friend_id=?`, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	seen := map[string]bool{}
	out := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		if id != "" && id != userID && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out, rows.Err()
}

func (s *Store) AdminFriendRow(ctx context.Context, userID, friendID string) (AdminFriendRow, error) {
	var row AdminFriendRow
	var forward, reverse sql.NullString
	err := s.QueryRowContext(ctx, `SELECT COALESCE(f.status,''),COALESCE(r.status,''),COALESCE(fs.nickname,''),COALESCE(fs.given,0),COALESCE(fs.recvd,0),COALESCE(u.remark,''),COALESCE(u.device_name,''),f.id,r.id FROM friendship f LEFT JOIN friendship r ON r.user_id=f.friend_id AND r.friend_id=f.user_id LEFT JOIN friend_status fs ON fs.user_id=f.user_id AND fs.friend_id=f.friend_id LEFT JOIN user u ON u.id=f.friend_id WHERE f.user_id=? AND f.friend_id=?`, userID, friendID).Scan(&row.Status, &row.ReverseStatus, &row.Nickname, &row.Given, &row.Received, &row.Remark, &row.DeviceName, &forward, &reverse)
	if err == sql.ErrNoRows {
		return row, nil
	}
	if err != nil {
		return row, err
	}
	row.FriendID = friendID
	row.TwoWay = forward.Valid && reverse.Valid
	return row, nil
}

func (s *Store) AdminSetFriendship(ctx context.Context, userID, friendID, state, adminIP, auditAction string) (bool, string, error) {
	existsA, err := s.HasUser(ctx, userID)
	if err != nil {
		return false, "", err
	}
	existsB, err := s.HasUser(ctx, friendID)
	if err != nil {
		return false, "", err
	}
	if !existsA || !existsB {
		return false, "账号不存在", nil
	}
	pairs := [][2]string{{userID, friendID}, {friendID, userID}}
	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return false, "", err
	}
	defer tx.Rollback()
	changed := int64(0)
	if state == "remove" {
		for _, pair := range pairs {
			result, err := tx.ExecContext(ctx, `INSERT INTO friendship(user_id,friend_id,status) VALUES(?,?,'removed') ON CONFLICT(user_id,friend_id) DO UPDATE SET status='removed',updated_at=CURRENT_TIMESTAMP`, pair[0], pair[1])
			if err != nil {
				return false, "", err
			}
			if n, _ := result.RowsAffected(); n > 0 {
				changed++
			}
			if _, err = tx.ExecContext(ctx, `UPDATE friend_status SET local_soft_deleted=1,updated_at=CURRENT_TIMESTAMP WHERE user_id=? AND friend_id=?`, pair[0], pair[1]); err != nil {
				return false, "", err
			}
		}
	} else {
		stamp := nowEpoch()
		for _, pair := range pairs {
			if err = ensureOneRelationship(ctx, tx, pair[0], pair[1], stamp); err != nil {
				return false, "", err
			}
			result, err := tx.ExecContext(ctx, `UPDATE friendship SET status='active',updated_at=CURRENT_TIMESTAMP WHERE user_id=? AND friend_id=?`, pair[0], pair[1])
			if err != nil {
				return false, "", err
			}
			if n, _ := result.RowsAffected(); n > 0 {
				changed++
			}
			if _, err = tx.ExecContext(ctx, `UPDATE friend_status SET local_soft_deleted=0,updated_at=CURRENT_TIMESTAMP WHERE user_id=? AND friend_id=?`, pair[0], pair[1]); err != nil {
				return false, "", err
			}
		}
	}
	detail := "双向置为好友"
	if state == "remove" {
		detail = "双向置为已删除"
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO admin_audit(admin_ip,action,user_id,friend_id,detail,created_at) VALUES(?,?,?,?,?,?)`, adminIP, auditAction, userID, friendID, detail, nowEpoch()); err != nil {
		return false, "", err
	}
	if err = tx.Commit(); err != nil {
		return false, "", err
	}
	return true, detail, nil
}
