package repository

import (
	"context"
	"database/sql"
)

type ForgeRate struct {
	SourceType                      string
	CurrentTier, CurrentLevel       int64
	TierCompletion, LevelCompletion float64
	LevelTierIndex, LevelTierCount  int64
	ResetAcknowledged, Enabled      bool
}

type Outfit struct {
	Parts                       map[string]map[string]int64
	Voice, Seed, RefreshVersion int64
	Scale, Height               float64
	Attitude                    string
	ArmsID                      int64
}

const (
	readForgeRates                Statement = `SELECT source_type,COALESCE(current_tier,0),COALESCE(current_level,0),COALESCE(tier_completion,0),COALESCE(level_completion,0),COALESCE(level_tier_index,0),COALESCE(level_tier_count,0),COALESCE(reset_acknowledged,0),COALESCE(enabled,0) FROM forge_rate WHERE user_id=? ORDER BY id`
	readOutfit                    Statement = `SELECT wing_tex,wing_pat,wing_mask,wing_id,prop_tex,prop_pat,prop_mask,prop_id,neck_tex,neck_pat,neck_mask,neck_id,mask_tex,mask_pat,mask_mask,mask_id,horn_tex,horn_pat,horn_mask,horn_id,hat_tex,hat_pat,hat_mask,hat_id,hair_tex,hair_pat,hair_mask,hair_id,feet_tex,feet_pat,feet_mask,feet_id,face_tex,face_pat,face_mask,face_id,body_tex,body_pat,body_mask,body_id,voice,seed,scale,refreshversion,height,attitude,arms_id FROM user_outfit WHERE user_id=?`
	createDefaultOutfit           Statement = `INSERT INTO user_outfit(user_id,wing_tex,wing_pat,wing_mask,wing_id,voice,seed,scale,refreshversion,prop_tex,prop_pat,prop_mask,prop_id,neck_tex,neck_pat,neck_mask,neck_id,mask_tex,mask_pat,mask_mask,mask_id,horn_tex,horn_pat,horn_mask,horn_id,height,hat_tex,hat_pat,hat_mask,hat_id,hair_tex,hair_pat,hair_mask,hair_id,feet_tex,feet_pat,feet_mask,feet_id,face_tex,face_pat,face_mask,face_id,body_tex,body_pat,body_mask,body_id,attitude,arms_id) VALUES(?,0,0,0,2245279351,9,16515,0.22549559,0,0,0,0,2035109393,0,0,0,3800884691,0,0,0,3663390882,0,0,0,3680499229,1.93536222,0,0,0,1470462579,0,0,0,1229053584,0,0,0,1532093990,0,0,0,4267756678,0,0,0,3691343236,'0',0) ON CONFLICT(user_id) DO NOTHING`
	userExists                    Statement = `SELECT 1 FROM user WHERE id=?`
	insertDefaultAchievementStats Statement = `INSERT INTO achievement_stat(user_id,stat_type,value,updated_at) VALUES(?,?,40000000000,?) ON CONFLICT(user_id,stat_type) DO NOTHING`
	readAchievementStats          Statement = `SELECT stat_type,value,updated_at FROM achievement_stat WHERE user_id=? ORDER BY id`
)

func (s *Store) HasUser(ctx context.Context, userID string) (bool, error) {
	var exists int
	err := s.QueryRowContext(ctx, userExists, userID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (s *Store) ForgeRates(ctx context.Context, userID string) ([]ForgeRate, error) {
	rows, err := s.QueryContext(ctx, readForgeRates, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ForgeRate
	for rows.Next() {
		var row ForgeRate
		if err = rows.Scan(&row.SourceType, &row.CurrentTier, &row.CurrentLevel, &row.TierCompletion, &row.LevelCompletion, &row.LevelTierIndex, &row.LevelTierCount, &row.ResetAcknowledged, &row.Enabled); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) GetOrCreateOutfit(ctx context.Context, userID string) (Outfit, error) {
	var row Outfit
	row.Parts = make(map[string]map[string]int64, 10)
	read := func() error {
		var values [40]int64
		err := s.QueryRowContext(ctx, readOutfit, userID).Scan(
			&values[0], &values[1], &values[2], &values[3], &values[4], &values[5], &values[6], &values[7],
			&values[8], &values[9], &values[10], &values[11], &values[12], &values[13], &values[14], &values[15],
			&values[16], &values[17], &values[18], &values[19], &values[20], &values[21], &values[22], &values[23],
			&values[24], &values[25], &values[26], &values[27], &values[28], &values[29], &values[30], &values[31],
			&values[32], &values[33], &values[34], &values[35], &values[36], &values[37], &values[38], &values[39],
			&row.Voice, &row.Seed, &row.Scale, &row.RefreshVersion, &row.Height, &row.Attitude, &row.ArmsID)
		if err != nil {
			return err
		}
		names := []string{"wing", "prop", "neck", "mask", "horn", "hat", "hair", "feet", "face", "body"}
		for i, name := range names {
			row.Parts[name] = map[string]int64{"tex": values[i*4], "pat": values[i*4+1], "mask": values[i*4+2], "id": values[i*4+3]}
		}
		return nil
	}
	if err := read(); err == nil {
		return row, nil
	} else if err != sql.ErrNoRows {
		return row, err
	}
	if _, err := s.ExecContext(ctx, createDefaultOutfit, userID); err != nil {
		return row, err
	}
	return row, read()
}

type AchievementStat struct {
	Type          string
	Value, Update int64
}

func (s *Store) AchievementStats(ctx context.Context, userID string, stamp int64) ([]AchievementStat, error) {
	rows, err := s.QueryContext(ctx, readAchievementStats, userID)
	if err != nil {
		return nil, err
	}
	var out []AchievementStat
	for rows.Next() {
		var item AchievementStat
		if err = rows.Scan(&item.Type, &item.Value, &item.Update); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, item)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	if len(out) > 0 {
		return out, nil
	}
	for _, kind := range []string{"world_quest_prairie_laugh_count", "spirit_cold_count"} {
		if _, err = s.ExecContext(ctx, insertDefaultAchievementStats, userID, kind, stamp); err != nil {
			return nil, err
		}
	}
	rows, err = s.QueryContext(ctx, readAchievementStats, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out = nil
	for rows.Next() {
		var item AchievementStat
		if err = rows.Scan(&item.Type, &item.Value, &item.Update); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
