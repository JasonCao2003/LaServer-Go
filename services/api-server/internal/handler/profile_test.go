package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestUpgradeV1PreservesAccounts(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "old-go.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(schema); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("INSERT INTO user(id,session) VALUES('keep','session')"); err != nil {
		t.Fatal(err)
	}
	if err = Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	var id string
	if err = db.QueryRow("SELECT id FROM user WHERE session='session'").Scan(&id); err != nil || id != "keep" {
		t.Fatalf("account lost: %s %v", id, err)
	}
	if err = checkSchema(db); err != nil {
		t.Fatal(err)
	}
}

func TestMigrationFailureRollsBackNewTables(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "collision.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(schema); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("CREATE TABLE app_badge(id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	if err = Migrate(context.Background(), db); err == nil {
		t.Fatal("expected migration collision")
	}
	var version, count int
	if err = db.QueryRow("SELECT version FROM laserver_schema WHERE id=1").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name='achievement_def'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if version != 1 || count != 0 {
		t.Fatalf("partial migration: version=%d created=%d", version, count)
	}
}

func TestMigrationRejectsFutureVersion(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "future.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(schema); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("UPDATE laserver_schema SET version=99"); err != nil {
		t.Fatal(err)
	}
	if err = Migrate(context.Background(), db); err == nil {
		t.Fatal("future version accepted")
	}
	if err = checkSchema(db); err == nil {
		t.Fatal("future version started")
	}
}

func TestAchievementDefinitionsFilterInactive(t *testing.T) {
	api, db := newAPI(t, true)
	a := create(t, api)
	if _, err := db.Exec(`INSERT INTO achievement_def(achievement_id,name,is_active,is_secret) VALUES('visible','Visible',1,0),('hidden','Hidden',0,1)`); err != nil {
		t.Fatal(err)
	}
	out := call(t, api, "/account/get_achievement_defs", map[string]any{"user": a["user"], "session": a["session"]}, 200)
	rows := out["achievement_defs"].([]any)
	if len(rows) != 1 || rows[0].(map[string]any)["id"] != "visible" || rows[0].(map[string]any)["is_secret"] != false {
		t.Fatalf("bad definitions: %v", out)
	}
}

func TestProfileReadsAreAccountScoped(t *testing.T) {
	api, db := newAPI(t, true)
	a := create(t, api)
	b := create(t, api)
	if _, err := db.Exec(`INSERT INTO achievement(user_id,achievement_id,achievement_name,unlocked_at,progress,completed) VALUES(?,?,?,?,?,?)`, a["user"], "first", "First", 123, 7, true); err != nil {
		t.Fatal(err)
	}
	req := map[string]any{"user": a["user"], "session": a["session"]}
	out := call(t, api, "/account/get_achievements", req, 200)
	rows := out["achievements"].([]any)
	if len(rows) != 1 || rows[0].(map[string]any)["progress"] != float64(7) {
		t.Fatalf("achievement missing: %v", out)
	}
	out = call(t, api, "/account/get_achievements", map[string]any{"user": b["user"], "session": b["session"]}, 200)
	if len(out["achievements"].([]any)) != 0 {
		t.Fatalf("cross-account leak: %v", out)
	}
	out = call(t, api, "/account/get_app_badge_number", req, 200)
	if out["set_app_badge_number"] != float64(0) {
		t.Fatal(out)
	}
	if _, err := db.Exec("UPDATE app_badge SET badge_number=6 WHERE user_id=?", a["user"]); err != nil {
		t.Fatal(err)
	}
	if out = call(t, api, "/account/get_app_badge_number", req, 200); out["set_app_badge_number"] != float64(6) {
		t.Fatal(out)
	}
}

func TestCollectiblesMergeAndMissingConfiguration(t *testing.T) {
	api, db := newAPI(t, true)
	a := create(t, api)
	req := map[string]any{"user": a["user"], "session": a["session"]}
	call(t, api, "/account/get_collectibles", req, 503)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "collectibles.json"), []byte(`{"collectibles":[{"id":"wave","level":1},{"id":"bow","level":1}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	api.cfg.GameConfigDir = dir
	if err := os.WriteFile(filepath.Join(dir, "map.json"), []byte(`{"get_map_defs":[{"id":42}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	mapResponse := call(t, api, "/account/get_map_defs", map[string]any{}, 200)
	if mapResponse["get_map_defs"].([]any)[0].(map[string]any)["id"] != float64(42) {
		t.Fatal(mapResponse)
	}
	if _, err := db.Exec("INSERT INTO collectible(id,user_id,level,used) VALUES(?,?,?,?)", "wave", a["user"], 3, true); err != nil {
		t.Fatal(err)
	}
	out := call(t, api, "/account/get_collectibles", req, 200)
	rows := out["collectibles"].([]any)
	if len(rows) != 2 || rows[0].(map[string]any)["level"] != float64(3) || rows[1].(map[string]any)["id"] != "bow" {
		t.Fatalf("merge broken: %v", out)
	}
	alias := call(t, api, "/account/get_collectibles", map[string]any{"user_id": a["user"], "session": a["session"]}, 200)
	if alias["collectibles"].([]any)[0].(map[string]any)["level"] != float64(3) {
		t.Fatal(alias)
	}
	base := call(t, api, "/account/get_collectibles", map[string]any{}, 200)
	if base["collectibles"].([]any)[0].(map[string]any)["level"] != float64(1) {
		t.Fatalf("anonymous leaked inventory: %v", base)
	}
	b := create(t, api)
	other := call(t, api, "/account/get_collectibles", map[string]any{"user": b["user"], "session": b["session"]}, 200)
	if other["collectibles"].([]any)[0].(map[string]any)["level"] != float64(1) {
		t.Fatalf("cross-account collectible leak: %v", other)
	}
}
