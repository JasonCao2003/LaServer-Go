package repository

import (
	"context"
	"database/sql"
)

type Statement string
type Store struct{ db *sql.DB }

func New(db *sql.DB) *Store { return &Store{db: db} }
func (s *Store) ExecContext(c context.Context, q Statement, args ...any) (sql.Result, error) {
	return s.db.ExecContext(c, string(q), args...)
}
func (s *Store) QueryContext(c context.Context, q Statement, args ...any) (*sql.Rows, error) {
	return s.db.QueryContext(c, string(q), args...)
}
func (s *Store) QueryRowContext(c context.Context, q Statement, args ...any) *sql.Row {
	return s.db.QueryRowContext(c, string(q), args...)
}

type Tx struct{ tx *sql.Tx }

func (s *Store) BeginTx(c context.Context, o *sql.TxOptions) (*Tx, error) {
	t, e := s.db.BeginTx(c, o)
	if e != nil {
		return nil, e
	}
	return &Tx{t}, nil
}
func (t *Tx) ExecContext(c context.Context, q Statement, args ...any) (sql.Result, error) {
	return t.tx.ExecContext(c, string(q), args...)
}
func (t *Tx) QueryRowContext(c context.Context, q Statement, args ...any) *sql.Row {
	return t.tx.QueryRowContext(c, string(q), args...)
}
func (t *Tx) Commit() error   { return t.tx.Commit() }
func (t *Tx) Rollback() error { return t.tx.Rollback() }
func UserLookup(key string) Statement {
	if key != "id" && key != "device_identifier" {
		panic("invalid lookup column")
	}
	return Statement("SELECT id,device_identifier,session,recovery_token,is_authorized,is_banned,ban_end_time,platform,created_at FROM user WHERE " + key + "=?")
}
func CurrencyUpdate(key string) Statement {
	switch key {
	case "candles", "heart", "heart_wax", "prestige", "prestige_vip", "prestige_wax", "season_candle", "season_wax", "season_wax_14", "season_heart", "season_pass_token", "vip", "wax":
		return Statement("UPDATE user_currency SET " + key + "=? WHERE user_id=?")
	}
	panic("invalid currency column")
}

const (
	CreateUser                 Statement = `INSERT INTO user(id,device,device_identifier,session,recovery_token,is_authorized,ip_address,created_at,device_name,user_agent) VALUES(?,?,?,?,?,?,?,?,?,?)`
	CreateCurrency             Statement = "INSERT INTO user_currency(user_id) VALUES(?)"
	ClearExpiredBan            Statement = "UPDATE user SET is_banned=0,ban_duration=0,ban_start_time=0,ban_end_time=0 WHERE id=?"
	RotateSession              Statement = "UPDATE user SET session=?,recovery_token=?,device_identifier=?,ip_address=?,user_agent=? WHERE id=?"
	ReadCurrency               Statement = `SELECT candles,heart,heart_wax,prestige,prestige_vip,prestige_wax,season_candle,season_wax,season_wax_14,season_heart,season_pass_token,vip,wax FROM user_currency WHERE user_id=?`
	MarkDailyBonus             Statement = `INSERT INTO season_pass_daily(user_id,day) SELECT user_id,? FROM user_currency WHERE user_id=? AND vip>0 ON CONFLICT(user_id,day) DO NOTHING`
	GrantDailyBonus            Statement = "UPDATE user_currency SET season_candle=COALESCE(season_candle,0)+1,updated_at=CURRENT_TIMESTAMP WHERE user_id=?"
	SaveUserdata               Statement = `INSERT INTO user_data(user_id,data,data_time,data_version,updated_at) VALUES(?,?,?,?,?) ON CONFLICT(user_id) DO UPDATE SET data=excluded.data,data_time=excluded.data_time,data_version=excluded.data_version,updated_at=excluded.updated_at`
	ReadUserdata               Statement = "SELECT data,data_time,data_version FROM user_data WHERE user_id=?"
	EnsureRank                 Statement = "INSERT INTO user_rank(user_id) VALUES(?) ON CONFLICT(user_id) DO NOTHING"
	ReadRank                   Statement = "SELECT COALESCE(social_rank,0),COALESCE(is_shepherd,0),COALESCE(is_beginner,0) FROM user_rank WHERE user_id=?"
	ReadCheckpoints            Statement = "SELECT checkpoints FROM user_checkpoint WHERE user_id=?"
	SaveCheckpoints            Statement = "INSERT INTO user_checkpoint(user_id,checkpoints) VALUES(?,?) ON CONFLICT(user_id) DO UPDATE SET checkpoints=excluded.checkpoints"
	ReadAchievementDefinitions Statement = `SELECT achievement_id,name,description,icon_url,category,points,is_secret FROM achievement_def WHERE is_active=1 ORDER BY id`
	ReadAchievements           Statement = `SELECT achievement_id,achievement_name,achievement_description,unlocked_at,COALESCE(progress,0),COALESCE(completed,0) FROM achievement WHERE user_id=? ORDER BY unlocked_at DESC,id`
	EnsureBadge                Statement = "INSERT INTO app_badge(user_id,badge_type) VALUES(?,'general') ON CONFLICT(user_id,badge_type) DO NOTHING"
	ReadBadge                  Statement = "SELECT COALESCE(badge_number,0) FROM app_badge WHERE user_id=? AND badge_type='general'"
	ReadCollectibles           Statement = "SELECT id,COALESCE(used,0),COALESCE(candle_space,0),COALESCE(carrying,0),COALESCE(level,1),COALESCE(permanent_level,0) FROM collectible WHERE user_id=? ORDER BY row_id"
	AdvanceLatestBuild         Statement = "INSERT INTO version(id,version) VALUES(1,?) ON CONFLICT(id) DO UPDATE SET version=MAX(COALESCE(version.version,0),excluded.version)"
	ReadLatestBuild            Statement = "SELECT version FROM version WHERE id=1"
)
