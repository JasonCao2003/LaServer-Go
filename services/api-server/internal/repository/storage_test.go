package repository

import (
	"net/url"
	"testing"
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
