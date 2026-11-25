package repository

import "context"

type AdminAccount struct {
	ID         string `json:"id"`
	Device     string `json:"device"`
	Platform   string `json:"platform"`
	Version    string `json:"version"`
	Authorized bool   `json:"authorized"`
	Banned     bool   `json:"banned"`
	BanEnd     int64  `json:"ban_end"`
	Created    int64  `json:"created"`
}

func (s *Store) AdminAccounts(ctx context.Context, keyword string, offset int) ([]AdminAccount, int, int, int, int, error) {
	const match = `(?1 = '' OR instr(lower(id),lower(?1)) > 0 OR instr(lower(remark),lower(?1)) > 0 OR instr(lower(device_name),lower(?1)) > 0 OR instr(lower(device),lower(?1)) > 0 OR instr(lower(platform),lower(?1)) > 0)`
	const banned = `(COALESCE(is_banned,0) != 0 AND (COALESCE(ban_end_time,0)=0 OR ban_end_time>unixepoch()))`
	var total, blocked, matched, unauthorized int
	rows, err := s.QueryContext(ctx, `SELECT count(*), count(*) FILTER (WHERE `+banned+`), count(*) FILTER (WHERE `+match+`), count(*) FILTER (WHERE `+match+` AND NOT `+banned+` AND COALESCE(is_authorized,0)=0) FROM user`, keyword)
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}
	if rows.Next() {
		err = rows.Scan(&total, &blocked, &matched, &unauthorized)
	} else {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}
	rows, err = s.QueryContext(ctx, `SELECT id, COALESCE(NULLIF(device_name,''),device,''), COALESCE(platform,''), COALESCE(CAST(version AS TEXT),''), COALESCE(is_authorized,0)!=0, `+banned+`, COALESCE(ban_end_time,0), COALESCE(created_at,0) FROM user WHERE `+match+` ORDER BY created_at DESC,id LIMIT 50 OFFSET ?2`, keyword, offset)
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}
	defer rows.Close()
	items := []AdminAccount{}
	for rows.Next() {
		var a AdminAccount
		if err := rows.Scan(&a.ID, &a.Device, &a.Platform, &a.Version, &a.Authorized, &a.Banned, &a.BanEnd, &a.Created); err != nil {
			return nil, 0, 0, 0, 0, err
		}
		items = append(items, a)
	}
	return items, total, blocked, matched, unauthorized, rows.Err()
}

func (s *Store) SetAdminAccountAuthorization(ctx context.Context, id string, authorized bool) (int64, error) {
	result, err := s.ExecContext(ctx, `UPDATE user SET is_authorized=? WHERE id=?`, authorized, id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *Store) SetAdminAccountBan(ctx context.Context, id string, banned bool, duration, start, end int64) (int64, error) {
	result, err := s.ExecContext(ctx, `UPDATE user SET is_banned=?,ban_duration=?,ban_start_time=?,ban_end_time=? WHERE id=?`, banned, duration, start, end, id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
