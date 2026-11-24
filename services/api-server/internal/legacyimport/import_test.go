// 旧数据库导入与安全检查测试

package legacyimport

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func legacyFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "legacy.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
		CREATE TABLE user(id TEXT PRIMARY KEY,device TEXT,session TEXT);
		CREATE TABLE user_currency(id INTEGER PRIMARY KEY,user_id TEXT,candles INTEGER,heart INTEGER,wax INTEGER);
		CREATE TABLE user_data(user_id TEXT PRIMARY KEY,data TEXT,data_time INTEGER,data_version INTEGER,updated_at INTEGER);
		CREATE TABLE player_quest(quest_id TEXT PRIMARY KEY,user_id TEXT,quest_name TEXT,start_time INTEGER,expiry_time INTEGER,status TEXT);
		CREATE TABLE quest_stat(id INTEGER PRIMARY KEY,quest_id TEXT,stat_type TEXT,stat_category TEXT,value INTEGER);
		CREATE TABLE season_quest(id INTEGER PRIMARY KEY,user_id TEXT,date TEXT,daily_quest_def_id TEXT,stat_type TEXT,start_value REAL,activated INTEGER);
		CREATE TABLE season_quest_reward(id INTEGER PRIMARY KEY,user_id TEXT,date TEXT,daily_quest_def_id TEXT,rewarded INTEGER);
		CREATE TABLE social_feed_post(id INTEGER PRIMARY KEY,user_id TEXT,social_feed_id TEXT,pool_type TEXT,pool_name TEXT,message TEXT);
		INSERT INTO user(id,device,session) VALUES('kept','test-device','old-session');
		INSERT INTO user_currency(id,user_id,candles,heart,wax) VALUES(1,'kept',7,2,9),(2,'orphan',99,99,99);
		INSERT INTO user_data(user_id,data,data_time,data_version,updated_at) VALUES('kept','save',3,3,4),('orphan','lost',5,3,6);
		INSERT INTO player_quest VALUES('quest-1','kept','fetchlight_mind',10,20,'active');
		INSERT INTO quest_stat(quest_id,stat_type,stat_category,value) VALUES('quest-1','fetchlight_mind_count','target_values',1);
		INSERT INTO season_quest(user_id,date,daily_quest_def_id,stat_type,start_value,activated) VALUES('kept','2026-11-24','fetchlight_mind','fetchlight_mind_count',0,1);
		INSERT INTO season_quest_reward(user_id,date,daily_quest_def_id,rewarded) VALUES('kept','2026-11-24','fetchlight_mind',1);
		INSERT INTO social_feed_post(user_id,social_feed_id,pool_type,pool_name,message) VALUES('kept','feed-1','test','test','visible');
	`)
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range [][3]string{
		{"recordings", "clip-1.json", `{"recording_id":"clip-1","user_id":"kept"}`},
		{"feed_extra", "feed-1.json", `{"extra_data":{"resource_id":"resource-1"},"comments_enabled":false,"tags":["test"],"private":true}`},
		{"feed_extra", "orphan-feed.json", `{"extra_data":{"resource_id":"orphan-resource"}}`},
	} {
		name, file, body := fixture[0], fixture[1], fixture[2]
		dir := filepath.Join(filepath.Dir(path), name)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, file), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "magic_defs_config.json"), []byte(`{"magic_defs":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func tableReport(t *testing.T, report Report, name string) TableReport {
	t.Helper()
	for _, row := range report.Tables {
		if row.Name == name {
			return row
		}
	}
	t.Fatalf("missing table report %s", name)
	return TableReport{}
}

func TestPreflightIsReadOnlyAndReportsOrphans(t *testing.T) {
	source := legacyFixture(t)
	before, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	report, err := Preflight(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("preflight modified source")
	}
	for _, name := range []string{"user_currency", "user_data"} {
		row := tableReport(t, report, name)
		if row.SourceRows != 2 || row.ImportedRows != 1 || row.SkippedRows != 1 {
			t.Fatalf("%s: %+v", name, row)
		}
	}
	for _, name := range []string{"quest_stat", "season_quest", "season_quest_reward"} {
		row := tableReport(t, report, name)
		if row.ImportedRows != 1 || row.Status != "copied" {
			t.Fatalf("legacy column mapping %s: %+v", name, row)
		}
	}
	if len(report.Sidecars) != 3 {
		t.Fatalf("sidecar inventory: %+v", report.Sidecars)
	}
	foundMagic := false
	for _, sidecar := range report.Sidecars {
		if sidecar.Name == "magic_defs_config.json" {
			foundMagic = sidecar.Files == 1 && sidecar.ImportedFiles == 0
		}
	}
	if !foundMagic {
		t.Fatal("unmapped configuration sidecar omitted from report")
	}
	for _, sidecar := range report.Sidecars {
		if sidecar.Name == "feed_extra" && (sidecar.Files != 2 || sidecar.ImportedFiles != 1) {
			t.Fatalf("orphan feed sidecar not reported: %+v", sidecar)
		}
	}
}

func TestImportRequiresNewTargetAndPreservesKeyValues(t *testing.T) {
	source := legacyFixture(t)
	target := filepath.Join(t.TempDir(), "new.db")
	if _, err := Import(context.Background(), source, target); err == nil {
		t.Fatal("partial legacy source imported without explicit opt-in")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("strict rejection created target: %v", err)
	}
	recordings := filepath.Join(filepath.Dir(target), "recordings")
	report, err := ImportWithOptions(context.Background(), source, target, Options{AllowPartial: true, RecordingsDir: recordings})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Partial {
		t.Fatal("skipped source rows were not reported as partial")
	}
	if raw, err := os.ReadFile(filepath.Join(recordings, "clip-1.json")); err != nil || len(raw) == 0 {
		t.Fatalf("recording sidecar not copied: %v", err)
	}
	if report.KeyChecks["user_currency.candles"].Source != 7 || !report.KeyChecks["user_currency.candles"].Match {
		t.Fatal(report.KeyChecks)
	}
	for _, name := range []string{"user_currency.rows", "user_data.rows"} {
		check, ok := report.KeyChecks[name]
		if !ok || check.Source != 1 || check.Target != 1 || !check.Match {
			t.Fatalf("key row check %s: %+v", name, check)
		}
	}
	db, err := sql.Open("sqlite", target)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var candles int64
	if err := db.QueryRow("SELECT candles FROM user_currency WHERE user_id='kept'").Scan(&candles); err != nil || candles != 7 {
		t.Fatalf("imported candles=%d err=%v", candles, err)
	}
	var count int
	if err := db.QueryRow("SELECT count(*) FROM user_currency WHERE user_id='orphan'").Scan(&count); err != nil || count != 0 {
		t.Fatalf("orphan imported: count=%d err=%v", count, err)
	}
	var category, seasonID string
	if err := db.QueryRow("SELECT category FROM quest_stat WHERE quest_id='quest-1'").Scan(&category); err != nil || category != "target_values" {
		t.Fatalf("quest category=%q err=%v", category, err)
	}
	if err := db.QueryRow("SELECT quest_id FROM season_quest WHERE user_id='kept'").Scan(&seasonID); err != nil || seasonID != "fetchlight_mind" {
		t.Fatalf("season quest ID=%q err=%v", seasonID, err)
	}
	var resourceID string
	if err := db.QueryRow("SELECT resource_id FROM social_feed_extra WHERE social_feed_id='feed-1'").Scan(&resourceID); err != nil || resourceID != "resource-1" {
		t.Fatalf("feed sidecar resource=%q err=%v", resourceID, err)
	}
	if err := db.QueryRow("SELECT count(*) FROM social_feed_extra WHERE social_feed_id='orphan-feed'").Scan(&count); err != nil || count != 0 {
		t.Fatalf("orphan feed sidecar imported: count=%d err=%v", count, err)
	}
	var tagsType string
	if err := db.QueryRow("SELECT typeof(tags) FROM social_feed_extra WHERE social_feed_id='feed-1'").Scan(&tagsType); err != nil || tagsType != "text" {
		t.Fatalf("feed tags type=%q err=%v", tagsType, err)
	}
	rows, err := db.Query("PRAGMA foreign_key_check")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("imported target has foreign key violations")
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if _, err := Import(context.Background(), source, target); err == nil {
		t.Fatal("existing destination was overwritten")
	}
	if err := db.QueryRow("SELECT candles FROM user_currency WHERE user_id='kept'").Scan(&candles); err != nil || candles != 7 {
		t.Fatal("rejected import changed target")
	}
}

func TestImportRejectsTargetInsideLegacyInstance(t *testing.T) {
	source := legacyFixture(t)
	target := filepath.Join(filepath.Dir(source), "new-api.db")
	if _, err := ImportWithOptions(context.Background(), source, target, Options{AllowPartial: true}); err == nil {
		t.Fatal("import wrote new target into legacy instance")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("legacy instance target exists: %v", err)
	}
}

func TestImportRejectsSymlinkIntoLegacyInstance(t *testing.T) {
	source := legacyFixture(t)
	link := filepath.Join(t.TempDir(), "old-instance-link")
	if err := os.Symlink(filepath.Dir(source), link); err != nil {
		t.Skipf("directory symlink unavailable: %v", err)
	}
	target := filepath.Join(link, "new-api.db")
	if _, err := ImportWithOptions(context.Background(), source, target, Options{AllowPartial: true}); err == nil {
		t.Fatal("symlink target wrote inside legacy instance")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(source), "new-api.db")); !os.IsNotExist(err) {
		t.Fatalf("legacy instance was modified through symlink: %v", err)
	}
}

func TestImportRejectsTargetBesideSymlinkedSourceFile(t *testing.T) {
	realSource := legacyFixture(t)
	link := filepath.Join(t.TempDir(), "users-link.db")
	if err := os.Symlink(realSource, link); err != nil {
		t.Skipf("file symlink unavailable: %v", err)
	}
	target := filepath.Join(filepath.Dir(realSource), "new-api.db")
	if _, err := ImportWithOptions(context.Background(), link, target, Options{AllowPartial: true}); err == nil {
		t.Fatal("symlinked source allowed target in actual legacy instance")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("actual legacy instance was modified: %v", err)
	}
}

func TestPreflightFailsOnUnreadableSidecarShape(t *testing.T) {
	source := legacyFixture(t)
	path := filepath.Join(filepath.Dir(source), "magic_defs_config.json")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := Preflight(context.Background(), source); err == nil {
		t.Fatal("preflight silently omitted invalid sidecar")
	}
}
