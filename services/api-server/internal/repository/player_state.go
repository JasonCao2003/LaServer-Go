package repository

import (
	"context"
	"database/sql"
	"strings"
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

type rowScanner interface {
	Scan(dest ...any) error
}

func scanOutfit(scanner rowScanner) (Outfit, error) {
	var row Outfit
	row.Parts = make(map[string]map[string]int64, 10)
	var values [40]int64
	err := scanner.Scan(
		&values[0], &values[1], &values[2], &values[3], &values[4], &values[5], &values[6], &values[7],
		&values[8], &values[9], &values[10], &values[11], &values[12], &values[13], &values[14], &values[15],
		&values[16], &values[17], &values[18], &values[19], &values[20], &values[21], &values[22], &values[23],
		&values[24], &values[25], &values[26], &values[27], &values[28], &values[29], &values[30], &values[31],
		&values[32], &values[33], &values[34], &values[35], &values[36], &values[37], &values[38], &values[39],
		&row.Voice, &row.Seed, &row.Scale, &row.RefreshVersion, &row.Height, &row.Attitude, &row.ArmsID)
	if err != nil {
		return row, err
	}
	names := []string{"wing", "prop", "neck", "mask", "horn", "hat", "hair", "feet", "face", "body"}
	for i, name := range names {
		row.Parts[name] = map[string]int64{"tex": values[i*4], "pat": values[i*4+1], "mask": values[i*4+2], "id": values[i*4+3]}
	}
	return row, nil
}

func (s *Store) GetOrCreateOutfit(ctx context.Context, userID string) (Outfit, error) {
	row, err := scanOutfit(s.QueryRowContext(ctx, readOutfit, userID))
	if err == nil {
		return row, nil
	}
	if err != sql.ErrNoRows {
		return row, err
	}
	if _, err = s.ExecContext(ctx, createDefaultOutfit, userID); err != nil {
		return row, err
	}
	return scanOutfit(s.QueryRowContext(ctx, readOutfit, userID))
}

// Outfits loads the outfit rows that exist for the given users. Friends without
// a stored outfit are intentionally absent so callers can emit null.
func (s *Store) Outfits(ctx context.Context, userIDs []string) (map[string]Outfit, error) {
	out := make(map[string]Outfit, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	query := `SELECT user_id,wing_tex,wing_pat,wing_mask,wing_id,prop_tex,prop_pat,prop_mask,prop_id,neck_tex,neck_pat,neck_mask,neck_id,mask_tex,mask_pat,mask_mask,mask_id,horn_tex,horn_pat,horn_mask,horn_id,hat_tex,hat_pat,hat_mask,hat_id,hair_tex,hair_pat,hair_mask,hair_id,feet_tex,feet_pat,feet_mask,feet_id,face_tex,face_pat,face_mask,face_id,body_tex,body_pat,body_mask,body_id,voice,seed,scale,refreshversion,height,attitude,arms_id FROM user_outfit WHERE user_id IN (` + strings.TrimSuffix(strings.Repeat("?,", len(userIDs)), ",") + `)`
	args := make([]any, len(userIDs))
	for i, id := range userIDs {
		args[i] = id
	}
	rows, err := s.QueryContext(ctx, Statement(query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		outfit, err := scanOutfit(outfitPrefixScanner{rows, &id})
		if err != nil {
			return nil, err
		}
		out[id] = outfit
	}
	return out, rows.Err()
}

// outfitPrefixScanner lets scanOutfit consume a leading user_id column.
type outfitPrefixScanner struct {
	rows *sql.Rows
	id   *string
}

func (s outfitPrefixScanner) Scan(dest ...any) error {
	return s.rows.Scan(append([]any{s.id}, dest...)...)
}

var outfitColumns = map[string]bool{
	"wing_tex": true, "wing_pat": true, "wing_mask": true, "wing_id": true,
	"prop_tex": true, "prop_pat": true, "prop_mask": true, "prop_id": true,
	"neck_tex": true, "neck_pat": true, "neck_mask": true, "neck_id": true,
	"mask_tex": true, "mask_pat": true, "mask_mask": true, "mask_id": true,
	"horn_tex": true, "horn_pat": true, "horn_mask": true, "horn_id": true,
	"hat_tex": true, "hat_pat": true, "hat_mask": true, "hat_id": true,
	"hair_tex": true, "hair_pat": true, "hair_mask": true, "hair_id": true,
	"feet_tex": true, "feet_pat": true, "feet_mask": true, "feet_id": true,
	"face_tex": true, "face_pat": true, "face_mask": true, "face_id": true,
	"body_tex": true, "body_pat": true, "body_mask": true, "body_id": true,
	"voice": true, "seed": true, "scale": true, "refreshversion": true,
	"height": true, "attitude": true, "arms_id": true,
}

// SetOutfitParts updates only the supplied columns. Column names are checked
// against a fixed allowlist and never come from request input directly.
func (s *Store) SetOutfitParts(ctx context.Context, userID string, parts map[string]any) error {
	if len(parts) == 0 {
		return nil
	}
	names := make([]string, 0, len(parts))
	args := make([]any, 0, len(parts)+1)
	for name, value := range parts {
		if !outfitColumns[name] {
			continue
		}
		names = append(names, name+"=?")
		args = append(args, value)
	}
	if len(names) == 0 {
		return nil
	}
	args = append(args, userID)
	_, err := s.ExecContext(ctx, Statement("UPDATE user_outfit SET "+strings.Join(names, ",")+" WHERE user_id=?"), args...)
	return err
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
