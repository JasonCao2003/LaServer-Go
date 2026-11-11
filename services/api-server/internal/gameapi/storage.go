package gameapi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "modernc.org/sqlite"
	"net/url"
	"path/filepath"
)

// Open configures every SQLite connection. It does not initialize tables.
func Open(path string) (*sql.DB, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	uri := url.URL{Scheme: "file", Path: filepath.ToSlash(absolute)}
	q := uri.Query()
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "foreign_keys(1)")
	uri.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", uri.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// Migrate initializes only a new database; legacy instances require explicit import.
func Migrate(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var count int
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		if _, err = tx.ExecContext(ctx, schema); err != nil {
			return fmt.Errorf("initialize schema: %w", err)
		}
	} else {
		var version int
		if err = tx.QueryRowContext(ctx, "SELECT version FROM laserver_schema WHERE id=1").Scan(&version); err != nil {
			return errors.New("unrecognized database; refusing migration")
		}
		if version != 1 {
			return fmt.Errorf("no migration from schema %d", version)
		}
	}
	if _, err = tx.ExecContext(ctx, profileSchema); err != nil {
		return fmt.Errorf("migrate profile schema: %w", err)
	}
	if _, err = tx.ExecContext(ctx, "UPDATE laserver_schema SET version=2 WHERE id=1"); err != nil {
		return err
	}
	return tx.Commit()
}

func checkSchema(db *sql.DB) error {
	var version int
	if err := db.QueryRow("SELECT version FROM laserver_schema WHERE id=1").Scan(&version); err != nil {
		return errors.New("database not initialized; run api-server migrate on a new database")
	}
	if version != 2 {
		return fmt.Errorf("unsupported schema version %d", version)
	}
	return nil
}

const profileSchema = `
CREATE TABLE achievement_def(id INTEGER PRIMARY KEY,achievement_id TEXT NOT NULL UNIQUE,name TEXT NOT NULL,description TEXT,icon_url TEXT,category TEXT,points INTEGER DEFAULT 0,is_secret BOOLEAN DEFAULT 0,is_active BOOLEAN DEFAULT 1,created_at DATETIME DEFAULT CURRENT_TIMESTAMP,updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE achievement(id INTEGER PRIMARY KEY,user_id TEXT NOT NULL REFERENCES user(id),achievement_id TEXT NOT NULL,achievement_name TEXT NOT NULL,achievement_description TEXT,unlocked_at INTEGER NOT NULL,progress INTEGER DEFAULT 0,completed BOOLEAN DEFAULT 0,created_at DATETIME DEFAULT CURRENT_TIMESTAMP,UNIQUE(user_id,achievement_id));
CREATE TABLE app_badge(id INTEGER PRIMARY KEY,user_id TEXT NOT NULL REFERENCES user(id),badge_number INTEGER DEFAULT 0,badge_type TEXT DEFAULT 'general',created_at DATETIME DEFAULT CURRENT_TIMESTAMP,updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,UNIQUE(user_id,badge_type));
CREATE TABLE collectible(row_id INTEGER PRIMARY KEY,id TEXT NOT NULL,user_id TEXT NOT NULL REFERENCES user(id),used BOOLEAN DEFAULT 0,candle_space BOOLEAN DEFAULT 0,carrying BOOLEAN DEFAULT 0,level INTEGER DEFAULT 1,permanent_level INTEGER DEFAULT 0,created_at DATETIME DEFAULT CURRENT_TIMESTAMP,updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,UNIQUE(user_id,id));
`

const schema = `
CREATE TABLE laserver_schema(id INTEGER PRIMARY KEY CHECK(id=1),version INTEGER NOT NULL);
INSERT INTO laserver_schema VALUES(1,1);
CREATE TABLE user (
 id TEXT PRIMARY KEY NOT NULL, device TEXT DEFAULT '', device_identifier TEXT,
 session TEXT UNIQUE, recovery_token TEXT, version INTEGER, platform TEXT DEFAULT '',
 is_banned INTEGER DEFAULT 0,ban_duration INTEGER DEFAULT 0,ban_start_time INTEGER DEFAULT 0,ban_end_time INTEGER DEFAULT 0,
 is_authorized INTEGER DEFAULT 0,ip_address TEXT DEFAULT '0.0.0.0',created_at INTEGER,remark TEXT DEFAULT '',
 device_name TEXT DEFAULT '',device_type TEXT DEFAULT '',device_model TEXT DEFAULT '',device_os TEXT DEFAULT '',user_agent TEXT DEFAULT ''
);
CREATE TABLE user_currency (
 id INTEGER PRIMARY KEY, user_id TEXT NOT NULL UNIQUE REFERENCES user(id),
 candles INTEGER DEFAULT 0,heart INTEGER DEFAULT 0,heart_wax INTEGER DEFAULT 0,
 prestige INTEGER DEFAULT 0,prestige_vip INTEGER DEFAULT 0,prestige_wax INTEGER DEFAULT 0,
 season_candle INTEGER DEFAULT 0,season_wax INTEGER DEFAULT 0,season_wax_14 INTEGER DEFAULT 0,
 season_heart INTEGER DEFAULT 0,season_pass_token INTEGER DEFAULT 0,vip INTEGER DEFAULT 0,wax INTEGER DEFAULT 3,
 updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE user_data(user_id TEXT PRIMARY KEY NOT NULL REFERENCES user(id),data TEXT NOT NULL DEFAULT '',data_time INTEGER DEFAULT 0,data_version INTEGER DEFAULT 3,updated_at INTEGER DEFAULT 0);
CREATE TABLE user_rank(id INTEGER PRIMARY KEY,user_id TEXT NOT NULL UNIQUE REFERENCES user(id),social_rank REAL DEFAULT 0,is_shepherd BOOLEAN DEFAULT 0,is_beginner BOOLEAN DEFAULT 0,created_at DATETIME DEFAULT CURRENT_TIMESTAMP,updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE user_checkpoint(user_id TEXT PRIMARY KEY NOT NULL REFERENCES user(id),checkpoints TEXT DEFAULT '[]');
CREATE TABLE season_pass_daily(user_id TEXT NOT NULL REFERENCES user(id),day TEXT NOT NULL,PRIMARY KEY(user_id,day));
CREATE TABLE version(id INTEGER PRIMARY KEY,version INTEGER);
`
