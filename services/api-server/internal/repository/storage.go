package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/migrations"
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
	uri := url.URL{Scheme: "file", Path: fileURIPath(filepath.ToSlash(absolute))}
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

// A drive-letter path must be absolute in a file URI, not an authority (file://C:/...).
func fileURIPath(path string) string {
	if len(path) >= 3 && path[1] == ':' && path[2] == '/' {
		return "/" + path
	}
	return path
}

// Migrate initializes only a new database; legacy instances require explicit import.
func Migrate(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var count, version int
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		if _, err = tx.ExecContext(ctx, schema); err != nil {
			return fmt.Errorf("initialize schema: %w", err)
		}
		version = 1
	} else {
		if err = tx.QueryRowContext(ctx, "SELECT version FROM laserver_schema WHERE id=1").Scan(&version); err != nil {
			return errors.New("unrecognized database; refusing migration")
		}
	}
	if version < 1 || version > 10 {
		return fmt.Errorf("no migration from schema %d", version)
	}
	if version == 10 {
		return tx.Commit()
	}
	if version == 1 {
		if _, err = tx.ExecContext(ctx, profileSchema); err != nil {
			return fmt.Errorf("migrate profile schema: %w", err)
		}
		if _, err = tx.ExecContext(ctx, "UPDATE laserver_schema SET version=2 WHERE id=1"); err != nil {
			return err
		}
	}
	if version <= 2 {
		if _, err = tx.ExecContext(ctx, migrations.Social); err != nil {
			return fmt.Errorf("migrate social schema: %w", err)
		}
		if _, err = tx.ExecContext(ctx, "UPDATE laserver_schema SET version=3 WHERE id=1"); err != nil {
			return err
		}
	}
	if version <= 3 {
		if _, err = tx.ExecContext(ctx, migrations.Constellations); err != nil {
			return fmt.Errorf("migrate constellation schema: %w", err)
		}
		if _, err = tx.ExecContext(ctx, "UPDATE laserver_schema SET version=4 WHERE id=1"); err != nil {
			return err
		}
	}
	if version <= 4 {
		if _, err = tx.ExecContext(ctx, migrations.PlayerState); err != nil {
			return fmt.Errorf("migrate player state schema: %w", err)
		}
		if _, err = tx.ExecContext(ctx, "UPDATE laserver_schema SET version=5 WHERE id=1"); err != nil {
			return err
		}
	}
	if version <= 5 {
		if _, err = tx.ExecContext(ctx, migrations.Invites); err != nil {
			return fmt.Errorf("migrate invite schema: %w", err)
		}
		if _, err = tx.ExecContext(ctx, "UPDATE laserver_schema SET version=6 WHERE id=1"); err != nil {
			return err
		}
	}
	if version <= 6 {
		if _, err = tx.ExecContext(ctx, migrations.Unlocks); err != nil {
			return fmt.Errorf("migrate unlock schema: %w", err)
		}
		if _, err = tx.ExecContext(ctx, "UPDATE laserver_schema SET version=7 WHERE id=1"); err != nil {
			return err
		}
	}
	if version <= 7 {
		if _, err = tx.ExecContext(ctx, migrations.Resources); err != nil {
			return fmt.Errorf("migrate resource schema: %w", err)
		}
		if _, err = tx.ExecContext(ctx, "UPDATE laserver_schema SET version=8 WHERE id=1"); err != nil {
			return err
		}
	}
	if version <= 8 {
		if _, err = tx.ExecContext(ctx, migrations.MagicStatus); err != nil {
			return fmt.Errorf("migrate magic and status schema: %w", err)
		}
		if _, err = tx.ExecContext(ctx, "UPDATE laserver_schema SET version=9 WHERE id=1"); err != nil {
			return err
		}
	}
	if version <= 9 {
		if _, err = tx.ExecContext(ctx, migrations.Commerce); err != nil {
			return fmt.Errorf("migrate commerce schema: %w", err)
		}
		if _, err = tx.ExecContext(ctx, "UPDATE laserver_schema SET version=10 WHERE id=1"); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func CheckSchema(db *sql.DB) error {
	var version int
	if err := db.QueryRow("SELECT version FROM laserver_schema WHERE id=1").Scan(&version); err != nil {
		return errors.New("database not initialized; run api-server migrate on a new database")
	}
	if version != 10 {
		return fmt.Errorf("unsupported schema version %d", version)
	}
	return nil
}

var profileSchema = migrations.Profile

var schema = migrations.Initial
