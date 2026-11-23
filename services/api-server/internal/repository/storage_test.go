package repository

import (
	"context"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/migrations"
)

func TestDriveLetterFileURIHasNoAuthority(t *testing.T) {
	u := url.URL{Scheme: "file", Path: fileURIPath("C:/Users/test data/game.db")}
	parsed, err := url.Parse(u.String())
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Host != "" || u.String() != "file:///C:/Users/test%20data/game.db" {
		t.Fatalf("drive parsed as authority: %s", u.String())
	}
	if fileURIPath("/home/test/game.db") != "/home/test/game.db" {
		t.Fatal("Linux path changed")
	}
}

func TestMigrateV17PreservesAccountAndMarksOldSessionUnverified(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "v17.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, script := range []string{
		migrations.Initial, migrations.Profile, migrations.Social, migrations.Constellations,
		migrations.PlayerState, migrations.Invites, migrations.Unlocks, migrations.Resources,
		migrations.MagicStatus, migrations.Commerce, migrations.Quests, migrations.Relationships,
		migrations.Messages, migrations.Geonotes, migrations.SocialFeed, migrations.Stage,
		migrations.AdminAudit,
	} {
		if _, err := db.Exec(script); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec("UPDATE laserver_schema SET version=17 WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO user(id,session,recovery_token) VALUES('old','old-session','old-secret')"); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := CheckSchema(db); err != nil {
		t.Fatal(err)
	}
	var session, recovery string
	var verified int
	if err := db.QueryRow("SELECT session,recovery_token,credential_mode_issued FROM user WHERE id='old'").Scan(&session, &recovery, &verified); err != nil {
		t.Fatal(err)
	}
	if session != "old-session" || recovery != "old-secret" || verified != 0 {
		t.Fatalf("old account changed: session=%q recovery=%q verified=%d", session, recovery, verified)
	}
	if err := Migrate(context.Background(), db); err != nil {
		t.Fatalf("v18 migration is not idempotent: %v", err)
	}
}
