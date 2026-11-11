package handler

import "testing"

func TestFreshAccountFriendsAreAnArray(t *testing.T) {
	api, _ := newAPI(t, true)
	a := create(t, api)
	out := call(t, api, "/account/get_friends", map[string]any{"user": a["user"], "session": a["session"]}, 200)
	if items, ok := out["set_friends"].([]any); !ok || len(items) != 0 {
		t.Fatalf("expected empty array for actual empty inventory: %v", out)
	}
	if out["set_friend_count"] != float64(0) {
		t.Fatal(out)
	}
}

func TestFriendsAreScopedAndPreserveIntArrays(t *testing.T) {
	api, db := newAPI(t, true)
	a := create(t, api)
	b := create(t, api)
	c := create(t, api)
	if _, err := db.Exec("INSERT INTO friendship(user_id,friend_id,status) VALUES(?,?,'active')", a["user"], b["user"]); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO friend_status(user_id,friend_id,nickname,abilities,hints,when_created) VALUES(?,?,?,?,?,?)", a["user"], b["user"], "Friend", "[1,2]", "[9]", 123); err != nil {
		t.Fatal(err)
	}
	out := call(t, api, "/account/get_friends", map[string]any{"user": a["user"], "session": a["session"]}, 200)
	rows := out["set_friends"].([]any)
	if len(rows) != 1 {
		t.Fatal(out)
	}
	item := rows[0].(map[string]any)
	if item["friend_id"] != b["user"] || item["nickname"] != "Friend" || item["abilities"].([]any)[0] != float64(1) {
		t.Fatal(item)
	}
	other := call(t, api, "/account/get_friends", map[string]any{"user": c["user"], "session": c["session"]}, 200)
	if len(other["set_friends"].([]any)) != 0 {
		t.Fatal("cross-account friendship leak")
	}
}

func TestFriendStatuesRetainConstellationPagePositions(t *testing.T) {
	api, db := newAPI(t, true)
	a := create(t, api)
	b := create(t, api)
	c := create(t, api)
	if _, err := db.Exec("INSERT INTO friend_status(user_id,friend_id,nickname,abilities,when_created) VALUES(?,?,?,?,?)", a["user"], b["user"], "Friend", "[1]", 123); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO friend_status(user_id,friend_id,nickname,abilities,when_created) VALUES(?,?,?,?,?)", a["user"], c["user"], "", "[]", 123); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO constellation_friend_page(user_id,page_index,friend_id,constellation_node_index) VALUES(?,2,?,4)", a["user"], b["user"]); err != nil {
		t.Fatal(err)
	}
	out := call(t, api, "/account/get_friend_statues", map[string]any{"user": a["user"], "session": a["session"]}, 200)
	if len(out["set_friend_statues"].([]any)) != 1 {
		t.Fatalf("lit stranger became full friend: %v", out)
	}
	pages := out["constellation_friend_pages"].([]any)
	if len(pages) != 3 || len(pages[0].(map[string]any)["friends"].([]any)) != 0 || len(pages[2].(map[string]any)["friends"].([]any)) != 1 {
		t.Fatal(out)
	}
}

func TestOfflineRoomDoesNotInventOnlineFriends(t *testing.T) {
	api, _ := newAPI(t, true)
	a := create(t, api)
	out := call(t, api, "/account/get_online_friends", map[string]any{"user": a["user"], "session": a["session"]}, 200)
	if len(out["online_friends"].([]any)) != 0 || out["refresh_online_friends"] != float64(30) {
		t.Fatal(out)
	}
}
