package repository

import (
	"context"
	"database/sql"
)

type StageState struct {
	StageID, OwnerID string
	Props            string
	Sequence         int64
	LevelID, UserID  string
	UpdatedAt        int64
}

func (s *Store) StageState(ctx context.Context, stageID, ownerID string) (StageState, bool, error) {
	var state StageState
	err := s.QueryRowContext(ctx, `SELECT stage_id,owner_id,props,sequence,level_id,user_id,updated_at FROM stage_state WHERE stage_id=? AND owner_id=?`, stageID, ownerID).Scan(&state.StageID, &state.OwnerID, &state.Props, &state.Sequence, &state.LevelID, &state.UserID, &state.UpdatedAt)
	if err == sql.ErrNoRows {
		return state, false, nil
	}
	return state, err == nil, err
}

func (s *Store) SaveStageState(ctx context.Context, state StageState) error {
	_, err := s.ExecContext(ctx, `INSERT INTO stage_state(stage_id,owner_id,props,sequence,level_id,user_id,updated_at) VALUES(?,?,?,?,?,?,?) ON CONFLICT(stage_id,owner_id) DO UPDATE SET props=excluded.props,sequence=excluded.sequence,level_id=excluded.level_id,user_id=excluded.user_id,updated_at=excluded.updated_at`, state.StageID, state.OwnerID, state.Props, state.Sequence, state.LevelID, state.UserID, state.UpdatedAt)
	return err
}
