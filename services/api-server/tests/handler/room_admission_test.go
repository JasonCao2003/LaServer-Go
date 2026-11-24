// 进房准入令牌校验测试

package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/handler"
)

func TestRoomAdmissionBindsDiscoveryToAccountSessionAndRoom(t *testing.T) {
	_, db := newAPI(t, true)
	cfg := handler.Config{
		AutoAuthorize:       true,
		LoginMode:           "legacy_dev",
		Listen:              "127.0.0.1:8443",
		RoomHost:            "192.168.1.10",
		RoomPort:            25565,
		RoomAdmissionKey:    strings.Repeat("ab", 32),
		RoomAdmissionBearer: strings.Repeat("x", 32),
	}
	api, err := New(db, cfg)
	if err != nil {
		t.Fatal(err)
	}
	player := create(t, api)
	other := create(t, api)
	discovery := call(t, api, "/account/find_previous_or_empty",
		map[string]any{"user": player["user"], "session": player["session"]}, 200)

	verify := func(account any, signature any, room, bearer string, want int) map[string]any {
		t.Helper()
		request, err := json.Marshal(map[string]any{
			"account_id": account,
			"signature":  signature,
			"room_uri":   room,
		})
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/internal/v1/room/admissions/verify", bytes.NewReader(request))
		req.RemoteAddr = "127.0.0.1:30000"
		req.Header.Set("Authorization", "Bearer "+bearer)
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)
		if w.Code != want {
			t.Fatalf("verify status = %d, want %d", w.Code, want)
		}
		var result map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}

	room := discovery["uri"].(string)
	signature := discovery["signature"].(string)
	if len(signature) != 64 {
		t.Fatalf("signature length = %d", len(signature))
	}
	verify(player["user"], signature, room, "wrong", 401)
	verify(other["user"], signature, room, cfg.RoomAdmissionBearer, 401)
	verify(player["user"], signature, "192.168.1.11:25565", cfg.RoomAdmissionBearer, 401)
	verify(player["user"], strings.Repeat("0", 64), room, cfg.RoomAdmissionBearer, 401)
	valid := verify(player["user"], signature, room, cfg.RoomAdmissionBearer, 200)
	if valid["account_id"] != player["user"] || valid["expires_at"] == nil {
		t.Fatalf("unexpected verified identity: %+v", valid)
	}

	join := call(t, api, "/account/join_friend_game", map[string]any{
		"user": other["user"], "session": other["session"], "friend_id": player["user"],
	}, 200)
	friendSignature := join["move_to_game"].(map[string]any)["signature"].(string)
	verify(other["user"], friendSignature, room, cfg.RoomAdmissionBearer, 200)
	verify(player["user"], friendSignature, room, cfg.RoomAdmissionBearer, 401)
	for _, assignment := range []struct {
		name    string
		account any
		token   string
	}{
		{"discovery", player["user"], signature},
		{"friend_join", other["user"], friendSignature},
	} {
		t.Run(assignment.name, func(t *testing.T) {
			var stored string
			if err := db.QueryRow("SELECT signature FROM server_assignment WHERE user_id=?", assignment.account).Scan(&stored); err != nil {
				t.Fatal(err)
			}
			if stored == assignment.token {
				t.Fatal("API persisted a live admission credential")
			}
		})
	}

	if _, err := db.Exec("UPDATE user SET is_banned=1,ban_end_time=0 WHERE id=?", player["user"]); err != nil {
		t.Fatal(err)
	}
	verify(player["user"], signature, room, cfg.RoomAdmissionBearer, 403)
	if _, err := db.Exec("UPDATE user SET is_banned=0 WHERE id=?", player["user"]); err != nil {
		t.Fatal(err)
	}
	call(t, api, "/account/login", map[string]any{"device": player["device"], "key": player["key"]}, 200)
	verify(player["user"], signature, room, cfg.RoomAdmissionBearer, 401)

	if _, err := db.Exec("UPDATE server_assignment SET server_uri=? WHERE user_id=?", "192.168.1.11:25565", player["user"]); err != nil {
		t.Fatal(err)
	}
	stale := call(t, api, "/account/join_friend_game", map[string]any{
		"user": other["user"], "session": other["session"], "friend_id": player["user"],
	}, 404)
	if stale["code"] != "FRIEND_NOT_IN_GAME" {
		t.Fatalf("unexpected stale room response: %v", stale["code"])
	}
	var assignments int
	if err := db.QueryRow("SELECT COUNT(*) FROM server_assignment WHERE user_id=?", other["user"]).Scan(&assignments); err != nil {
		t.Fatal(err)
	}
	if assignments != 1 {
		t.Fatalf("joining a stale room wrote an assignment, found %d assignments", assignments)
	}
}
