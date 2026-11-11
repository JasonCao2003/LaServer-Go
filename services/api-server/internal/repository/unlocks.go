package repository

import (
	"context"
	"database/sql"
)

type Unlock struct {
	Name      string
	Type      string
	Ack       *bool
	CreatedAt int64
}

func (s *Store) SeedDefaultUnlocks(ctx context.Context, userID string, defaults []Unlock) error {
	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var count int
	if err = tx.QueryRowContext(ctx, Statement(`SELECT count(*) FROM unlock_item WHERE user_id=?`), userID).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return tx.Commit()
	}
	for _, item := range defaults {
		if _, err = tx.ExecContext(ctx, Statement(`INSERT INTO unlock_item(user_id,name,unlock_type,ack,created_at) VALUES(?,?,?,?,?)`), userID, item.Name, item.Type, item.Ack, item.CreatedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) Unlocks(ctx context.Context, userID string) ([]Unlock, error) {
	rows, err := s.QueryContext(ctx, Statement(`SELECT name,unlock_type,ack,created_at FROM unlock_item WHERE user_id=? ORDER BY id`), userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Unlock{}
	for rows.Next() {
		var item Unlock
		var ack sql.NullBool
		if err = rows.Scan(&item.Name, &item.Type, &ack, &item.CreatedAt); err != nil {
			return nil, err
		}
		if ack.Valid {
			item.Ack = &ack.Bool
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) AckUnlock(ctx context.Context, userID, name string, createdAt int64) (Unlock, error) {
	var item Unlock
	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return item, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, Statement(`INSERT INTO unlock_item(user_id,name,unlock_type,ack,created_at) VALUES(?,?,'spiritshop',1,?) ON CONFLICT(user_id,name) DO UPDATE SET ack=1`), userID, name, createdAt)
	if err != nil {
		return item, err
	}
	var ack bool
	err = tx.QueryRowContext(ctx, Statement(`SELECT name,unlock_type,ack,created_at FROM unlock_item WHERE user_id=? AND name=?`), userID, name).Scan(&item.Name, &item.Type, &ack, &item.CreatedAt)
	if err != nil {
		return item, err
	}
	item.Ack = &ack
	return item, tx.Commit()
}
