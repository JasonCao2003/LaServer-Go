// Package legacyimport copies compatible legacy SQLite rows into a new Go API database.
// The legacy database is attached read-only; unsupported tables and orphan rows are reported.
package legacyimport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
	_ "modernc.org/sqlite"
)

type TableReport struct {
	Name         string `json:"name"`
	SourceRows   int64  `json:"source_rows"`
	ImportedRows int64  `json:"imported_rows"`
	SkippedRows  int64  `json:"skipped_rows"`
	Status       string `json:"status"`
	Detail       string `json:"detail,omitempty"`
}

type ValueCheck struct {
	Source int64 `json:"source"`
	Target int64 `json:"target"`
	Match  bool  `json:"match"`
}

type SidecarReport struct {
	Name          string `json:"name"`
	Files         int    `json:"files"`
	ImportedFiles int    `json:"imported_files"`
	Bytes         int64  `json:"bytes"`
}

type Report struct {
	Source    string                `json:"source"`
	Target    string                `json:"target,omitempty"`
	Partial   bool                  `json:"partial"`
	Tables    []TableReport         `json:"tables"`
	KeyChecks map[string]ValueCheck `json:"key_checks"`
	Sidecars  []SidecarReport       `json:"sidecars"`
}

type Options struct {
	AllowPartial  bool
	RecordingsDir string
}

type column struct {
	name, typ string
	notNull   bool
	defaulted bool
	pk        bool
}

type foreignKey struct {
	id, seq  int
	parent   string
	from, to string
}

// Keys are Go column names; values are their legacy equivalents.
var legacyAliases = map[string]map[string]string{
	"quest_stat":          {"category": "stat_category"},
	"season_quest":        {"quest_id": "daily_quest_def_id"},
	"season_quest_reward": {"quest_id": "daily_quest_def_id"},
}

func quoted(name string) string { return `"` + strings.ReplaceAll(name, `"`, `""`) + `"` }

func readonlyURI(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("source must be a regular SQLite file")
	}
	p := filepath.ToSlash(abs)
	if len(p) >= 3 && p[1] == ':' && p[2] == '/' {
		p = "/" + p
	}
	u := url.URL{Scheme: "file", Path: p}
	q := u.Query()
	q.Set("mode", "ro")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func schemaTables(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, schema string) ([]string, error) {
	rows, err := q.QueryContext(ctx, "SELECT name FROM "+schema+".sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

func tableColumns(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, schema, table string) ([]column, error) {
	rows, err := q.QueryContext(ctx, "PRAGMA "+schema+".table_info("+quoted(table)+")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []column{}
	for rows.Next() {
		var ordinal, required, pk int
		var name, typ string
		var fallback sql.NullString
		if err := rows.Scan(&ordinal, &name, &typ, &required, &fallback, &pk); err != nil {
			return nil, err
		}
		out = append(out, column{name, typ, required != 0, fallback.Valid, pk != 0})
	}
	return out, rows.Err()
}

func tableForeignKeys(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, table string) ([]foreignKey, error) {
	rows, err := q.QueryContext(ctx, "PRAGMA main.foreign_key_list("+quoted(table)+")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []foreignKey{}
	for rows.Next() {
		var key foreignKey
		var update, remove, match string
		if err := rows.Scan(&key.id, &key.seq, &key.parent, &key.from, &key.to, &update, &remove, &match); err != nil {
			return nil, err
		}
		out = append(out, key)
	}
	return out, rows.Err()
}

func sidecars(source string) ([]SidecarReport, error) {
	out := []SidecarReport{}
	rootDir := filepath.Dir(source)
	for _, name := range []string{"recordings", "feed_extra", "stage"} {
		root := filepath.Join(rootDir, name)
		var files int
		var bytes int64
		err := filepath.WalkDir(root, func(path string, item os.DirEntry, walkErr error) error {
			if errors.Is(walkErr, os.ErrNotExist) && path == root {
				return nil
			}
			if walkErr != nil {
				return walkErr
			}
			if !item.IsDir() {
				info, infoErr := item.Info()
				if infoErr != nil {
					return infoErr
				}
				if !info.Mode().IsRegular() {
					return fmt.Errorf("unsupported sidecar file: %s", path)
				}
				files++
				bytes += info.Size()
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		if files > 0 {
			out = append(out, SidecarReport{Name: name, Files: files, Bytes: bytes})
		}
	}
	info, err := os.Stat(filepath.Join(rootDir, "magic_defs_config.json"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err == nil && !info.Mode().IsRegular() {
		return nil, errors.New("magic_defs_config.json is not a regular file")
	}
	if err == nil {
		out = append(out, SidecarReport{Name: "magic_defs_config.json", Files: 1, Bytes: info.Size()})
	}
	return out, nil
}

func ensureOutsideLegacy(sourceDB, target string) error {
	realSource, err := filepath.EvalSymlinks(sourceDB)
	if err != nil {
		return err
	}
	sourceRoot := filepath.Dir(realSource)
	targetRoot, err := filepath.EvalSymlinks(filepath.Dir(target))
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(sourceRoot, targetRoot)
	if err != nil || rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return errors.New("destination must be outside legacy instance")
	}
	return nil
}

func migrateMemory(ctx context.Context) (*sql.DB, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, err
	}
	if err := repository.Migrate(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// Preflight reads the source through SQLite's read-only URI and writes only to an in-memory target.
func Preflight(ctx context.Context, source string) (Report, error) {
	realSource, err := filepath.EvalSymlinks(source)
	if err != nil {
		return Report{}, err
	}
	db, err := migrateMemory(ctx)
	if err != nil {
		return Report{}, err
	}
	defer db.Close()
	return copyInto(ctx, db, realSource, "")
}

// Import requires a destination path that does not exist, including an empty existing database.
func Import(ctx context.Context, source, target string) (Report, error) {
	return ImportWithOptions(ctx, source, target, Options{})
}

// ImportWithOptions requires an explicit opt-in when rows or sidecar files cannot be migrated.
func ImportWithOptions(ctx context.Context, source, target string, options Options) (Report, error) {
	sourceAbs, err := filepath.Abs(source)
	if err != nil {
		return Report{}, err
	}
	sourceAbs, err = filepath.EvalSymlinks(sourceAbs)
	if err != nil {
		return Report{}, err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return Report{}, err
	}
	if sourceAbs == targetAbs {
		return Report{}, errors.New("source and target must differ")
	}
	if err := ensureOutsideLegacy(sourceAbs, targetAbs); err != nil {
		return Report{}, err
	}
	if _, err := readonlyURI(sourceAbs); err != nil {
		return Report{}, err
	}
	preflight, err := Preflight(ctx, sourceAbs)
	if err != nil {
		return preflight, err
	}
	if !options.AllowPartial {
		for _, row := range preflight.Tables {
			if row.SkippedRows > 0 {
				return preflight, errors.New("preflight found unmapped rows; partial import requires explicit AllowPartial")
			}
		}
		for _, sidecar := range preflight.Sidecars {
			if sidecar.ImportedFiles < sidecar.Files && (sidecar.Name != "recordings" || options.RecordingsDir == "") {
				return preflight, errors.New("preflight found unmapped sidecars; provide RecordingsDir or AllowPartial")
			}
		}
	}
	file, err := os.OpenFile(targetAbs, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return Report{}, fmt.Errorf("destination must not exist: %w", err)
	}
	file.Close()
	complete := false
	defer func() {
		if !complete {
			_ = os.Remove(targetAbs)
		}
	}()
	db, err := repository.Open(targetAbs)
	if err != nil {
		return Report{}, err
	}
	defer db.Close()
	if err := repository.Migrate(ctx, db); err != nil {
		return Report{}, err
	}
	report, err := copyInto(ctx, db, sourceAbs, targetAbs)
	if err != nil {
		return report, err
	}
	if !options.AllowPartial && reportHasLossExceptPlannedRecordings(report, options.RecordingsDir != "") {
		return report, errors.New("source changed or has unmapped content; partial import requires explicit AllowPartial")
	}
	if options.RecordingsDir != "" {
		expected := 0
		for _, sidecar := range report.Sidecars {
			if sidecar.Name == "recordings" {
				expected = sidecar.Files
			}
		}
		count, err := copyRecordings(sourceAbs, options.RecordingsDir, expected)
		if err != nil {
			return report, err
		}
		for i := range report.Sidecars {
			if report.Sidecars[i].Name == "recordings" {
				report.Sidecars[i].ImportedFiles = count
			}
		}
		report.Partial = reportHasLoss(report)
	}
	if !options.AllowPartial && report.Partial {
		return report, errors.New("actual import is partial; explicit AllowPartial required")
	}
	complete = true
	return report, nil
}

func copyInto(ctx context.Context, db *sql.DB, source, target string) (Report, error) {
	uri, err := readonlyURI(source)
	if err != nil {
		return Report{}, err
	}
	source, _ = filepath.Abs(source)
	files, err := sidecars(source)
	if err != nil {
		return Report{}, err
	}
	report := Report{Source: source, Target: target, Tables: []TableReport{}, KeyChecks: map[string]ValueCheck{}, Sidecars: files}
	if _, err := db.ExecContext(ctx, "ATTACH DATABASE ? AS legacy", uri); err != nil {
		return report, fmt.Errorf("attach read-only source: %w", err)
	}
	defer db.ExecContext(context.Background(), "DETACH DATABASE legacy")
	mainTables, err := schemaTables(ctx, db, "main")
	if err != nil {
		return report, err
	}
	legacyTables, err := schemaTables(ctx, db, "legacy")
	if err != nil {
		return report, err
	}
	available := map[string]bool{}
	for _, table := range legacyTables {
		available[table] = true
	}
	targetTables := map[string]bool{}
	for _, table := range mainTables {
		targetTables[table] = true
	}
	ordered := []string{}
	state := map[string]int{}
	var visit func(string) error
	visit = func(table string) error {
		if state[table] == 2 {
			return nil
		}
		if state[table] == 1 {
			return fmt.Errorf("cyclic target foreign keys at %s", table)
		}
		state[table] = 1
		keys, err := tableForeignKeys(ctx, db, table)
		if err != nil {
			return err
		}
		for _, key := range keys {
			if key.parent != table && targetTables[key.parent] {
				if err := visit(key.parent); err != nil {
					return err
				}
			}
		}
		state[table] = 2
		ordered = append(ordered, table)
		return nil
	}
	for _, table := range mainTables {
		if table != "laserver_schema" {
			if err := visit(table); err != nil {
				return report, err
			}
		}
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return report, err
	}
	defer tx.Rollback()
	for _, table := range ordered {
		if !available[table] {
			continue
		}
		row, err := copyTable(ctx, tx, table)
		report.Tables = append(report.Tables, row)
		if err != nil {
			return report, fmt.Errorf("import %s: %w", table, err)
		}
	}
	for _, table := range legacyTables {
		if targetTables[table] {
			continue
		}
		var count int64
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM legacy."+quoted(table)).Scan(&count); err != nil {
			return report, err
		}
		report.Tables = append(report.Tables, TableReport{Name: table, SourceRows: count, SkippedRows: count, Status: "unsupported", Detail: "no Go table"})
	}
	sort.Slice(report.Tables, func(i, j int) bool { return report.Tables[i].Name < report.Tables[j].Name })
	if err := importFeedExtras(ctx, tx, &report); err != nil {
		return report, err
	}
	report.Partial = reportHasLoss(report)
	if err := compareKeyValues(ctx, tx, &report); err != nil {
		return report, err
	}
	for key, check := range report.KeyChecks {
		if !check.Match {
			return report, fmt.Errorf("key value mismatch: %s", key)
		}
	}
	violations, err := tx.QueryContext(ctx, "PRAGMA main.foreign_key_check")
	if err != nil {
		return report, err
	}
	invalid := violations.Next()
	if err := violations.Err(); err != nil {
		violations.Close()
		return report, err
	}
	if err := violations.Close(); err != nil {
		return report, err
	}
	if invalid {
		return report, errors.New("imported target has foreign key violations")
	}
	if err := tx.Commit(); err != nil {
		return report, err
	}
	return report, nil
}

func reportHasLoss(report Report) bool {
	for _, row := range report.Tables {
		if row.SkippedRows > 0 {
			return true
		}
	}
	for _, sidecar := range report.Sidecars {
		if sidecar.ImportedFiles < sidecar.Files {
			return true
		}
	}
	return false
}

func reportHasLossExceptPlannedRecordings(report Report, recordingsPlanned bool) bool {
	for _, row := range report.Tables {
		if row.SkippedRows > 0 {
			return true
		}
	}
	for _, sidecar := range report.Sidecars {
		if sidecar.Name == "recordings" && recordingsPlanned {
			continue
		}
		if sidecar.ImportedFiles < sidecar.Files {
			return true
		}
	}
	return false
}

func importFeedExtras(ctx context.Context, tx *sql.Tx, report *Report) error {
	root := filepath.Join(filepath.Dir(report.Source), "feed_extra")
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	count := 0
	for _, entry := range entries {
		if !entry.Type().IsRegular() || filepath.Ext(entry.Name()) != ".json" {
			return fmt.Errorf("unsupported feed_extra sidecar: %s", entry.Name())
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		if id == "" {
			return errors.New("empty feed_extra ID")
		}
		raw, err := os.ReadFile(filepath.Join(root, entry.Name()))
		if err != nil {
			return err
		}
		var value struct {
			ExtraData       map[string]any `json:"extra_data"`
			CommentsEnabled *bool          `json:"comments_enabled"`
			Tags            any            `json:"tags"`
			Private         bool           `json:"private"`
		}
		if err := json.Unmarshal(raw, &value); err != nil {
			return fmt.Errorf("feed_extra %s: %w", entry.Name(), err)
		}
		if value.ExtraData == nil {
			value.ExtraData = map[string]any{}
		}
		encoded, err := json.Marshal(value.ExtraData)
		if err != nil {
			return err
		}
		var tags any
		if value.Tags != nil {
			encodedTags, marshalErr := json.Marshal(value.Tags)
			err = marshalErr
			if err != nil {
				return err
			}
			tags = string(encodedTags)
		}
		comments := true
		if value.CommentsEnabled != nil {
			comments = *value.CommentsEnabled
		}
		resource, _ := value.ExtraData["resource_id"].(string)
		var exists bool
		if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM main.social_feed_post WHERE social_feed_id=?)", id).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			continue
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO main.social_feed_extra(social_feed_id,extra_data,resource_id,comments_enabled,tags,private) VALUES(?,?,?,?,?,?)", id, string(encoded), resource, comments, tags, value.Private); err != nil {
			return err
		}
		count++
	}
	for i := range report.Sidecars {
		if report.Sidecars[i].Name == "feed_extra" {
			report.Sidecars[i].ImportedFiles = count
		}
	}
	return nil
}

func copyRecordings(sourceDB, targetDir string, expected int) (int, error) {
	sourceDir := filepath.Join(filepath.Dir(sourceDB), "recordings")
	entries, err := os.ReadDir(sourceDir)
	if errors.Is(err, os.ErrNotExist) {
		if expected != 0 {
			return 0, errors.New("recording sidecar directory disappeared during import")
		}
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		return 0, err
	}
	if err := ensureOutsideLegacy(sourceDB, absTarget); err != nil {
		return 0, err
	}
	if _, err := os.Stat(absTarget); !errors.Is(err, os.ErrNotExist) {
		return 0, errors.New("recording target already exists or cannot be checked")
	}
	stage, err := os.MkdirTemp(filepath.Dir(absTarget), ".recordings-import-")
	if err != nil {
		return 0, err
	}
	created := []string{}
	done := false
	defer func() {
		if !done {
			for _, path := range created {
				_ = os.Remove(path)
			}
			_ = os.Remove(stage)
		}
	}()
	for _, entry := range entries {
		ext := filepath.Ext(entry.Name())
		if !entry.Type().IsRegular() || ext != ".json" && ext != ".ndjson" {
			return 0, fmt.Errorf("unsupported recording sidecar: %s", entry.Name())
		}
		input, err := os.Open(filepath.Join(sourceDir, entry.Name()))
		if err != nil {
			return 0, err
		}
		path := filepath.Join(stage, entry.Name())
		output, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			input.Close()
			return 0, err
		}
		created = append(created, path)
		_, copyErr := io.Copy(output, input)
		closeErr := output.Close()
		input.Close()
		if copyErr != nil {
			return 0, copyErr
		}
		if closeErr != nil {
			return 0, closeErr
		}
	}
	if len(entries) != expected {
		return 0, fmt.Errorf("recording sidecar count changed during import: expected=%d actual=%d", expected, len(entries))
	}
	if err := os.Rename(stage, absTarget); err != nil {
		return 0, err
	}
	done = true
	return len(entries), nil
}

func copyTable(ctx context.Context, tx *sql.Tx, table string) (TableReport, error) {
	row := TableReport{Name: table}
	if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM legacy."+quoted(table)).Scan(&row.SourceRows); err != nil {
		return row, err
	}
	legacyCols, err := tableColumns(ctx, tx, "legacy", table)
	if err != nil {
		return row, err
	}
	mainCols, err := tableColumns(ctx, tx, "main", table)
	if err != nil {
		return row, err
	}
	sourceCols := map[string]bool{}
	for _, col := range legacyCols {
		sourceCols[col.name] = true
	}
	type selectedColumn struct{ target, source string }
	selected := []selectedColumn{}
	for _, col := range mainCols {
		from := col.name
		if alias := legacyAliases[table][col.name]; alias != "" {
			from = alias
		}
		if sourceCols[from] {
			selected = append(selected, selectedColumn{col.name, from})
		} else if col.notNull && !col.defaulted && !(col.pk && strings.EqualFold(col.typ, "INTEGER")) {
			row.Status, row.SkippedRows, row.Detail = "unsupported", row.SourceRows, "missing required column "+col.name
			return row, nil
		}
	}
	if len(selected) == 0 {
		row.Status, row.SkippedRows, row.Detail = "unsupported", row.SourceRows, "no common columns"
		return row, nil
	}
	keys, err := tableForeignKeys(ctx, tx, table)
	if err != nil {
		return row, err
	}
	conditions := []string{}
	for _, key := range keys {
		if key.to == "" || !sourceCols[key.from] {
			continue
		}
		child := "legacy." + quoted(table) + "." + quoted(key.from)
		parent := "main." + quoted(key.parent)
		conditions = append(conditions, "("+child+" IS NULL OR EXISTS(SELECT 1 FROM "+parent+" WHERE "+parent+"."+quoted(key.to)+"="+child+"))")
	}
	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}
	var eligible int64
	if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM legacy."+quoted(table)+where).Scan(&eligible); err != nil {
		return row, err
	}
	cols := make([]string, len(selected))
	values := make([]string, len(selected))
	for i, col := range selected {
		cols[i] = quoted(col.target)
		values[i] = quoted(col.source)
	}
	statement := "INSERT INTO main." + quoted(table) + "(" + strings.Join(cols, ",") + ") SELECT " + strings.Join(values, ",") + " FROM legacy." + quoted(table) + where
	if _, err := tx.ExecContext(ctx, statement); err != nil {
		return row, err
	}
	if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM main."+quoted(table)).Scan(&row.ImportedRows); err != nil {
		return row, err
	}
	row.SkippedRows = row.SourceRows - row.ImportedRows
	row.Status = "copied"
	if row.ImportedRows != eligible {
		return row, fmt.Errorf("eligible=%d imported=%d", eligible, row.ImportedRows)
	}
	return row, nil
}

func compareKeyValues(ctx context.Context, tx *sql.Tx, report *Report) error {
	for _, row := range report.Tables {
		if row.Status != "copied" || row.Name != "user_currency" && row.Name != "user_data" {
			continue
		}
		legacyCols, err := tableColumns(ctx, tx, "legacy", row.Name)
		if err != nil {
			return err
		}
		targetCols, err := tableColumns(ctx, tx, "main", row.Name)
		if err != nil {
			return err
		}
		available := map[string]bool{}
		for _, col := range targetCols {
			available[col.name] = true
		}
		matches := []string{}
		for _, col := range legacyCols {
			if available[col.name] {
				matches = append(matches, "source."+quoted(col.name)+" IS dest."+quoted(col.name))
			}
		}
		var expected, equal int64
		base := " FROM legacy." + quoted(row.Name) + " AS source JOIN main.user AS owner ON owner.id=source.user_id"
		if err := tx.QueryRowContext(ctx, "SELECT count(*)"+base).Scan(&expected); err != nil {
			return err
		}
		query := "SELECT count(*)" + base + " JOIN main." + quoted(row.Name) + " AS dest ON dest.user_id=source.user_id WHERE " + strings.Join(matches, " AND ")
		if err := tx.QueryRowContext(ctx, query).Scan(&equal); err != nil {
			return err
		}
		report.KeyChecks[row.Name+".rows"] = ValueCheck{Source: expected, Target: equal, Match: expected == equal && expected == row.ImportedRows}
		if row.Name != "user_currency" {
			continue
		}
		for _, field := range []string{"candles", "heart", "wax", "season_candle", "season_heart", "vip"} {
			found := false
			for _, col := range legacyCols {
				found = found || col.name == field
			}
			if !found {
				continue
			}
			var from, to int64
			query := "SELECT coalesce(sum(" + quoted(field) + "),0) FROM legacy.user_currency WHERE EXISTS(SELECT 1 FROM main.user WHERE id=legacy.user_currency.user_id)"
			if err := tx.QueryRowContext(ctx, query).Scan(&from); err != nil {
				return err
			}
			if err := tx.QueryRowContext(ctx, "SELECT coalesce(sum("+quoted(field)+"),0) FROM main.user_currency").Scan(&to); err != nil {
				return err
			}
			report.KeyChecks["user_currency."+field] = ValueCheck{Source: from, Target: to, Match: from == to}
		}
	}
	return nil
}
