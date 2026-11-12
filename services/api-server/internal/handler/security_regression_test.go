package handler

import "testing"

func TestFeedMutationRequiresOwner(t *testing.T) {
	api, _ := newAPI(t, true)
	owner, other := create(t, api), create(t, api)
	post := call(t, api, "/service/social-feed/api/v1/create", map[string]any{"user": owner["user"], "session": owner["session"], "pool_type": "test", "pool_name": "test", "message": "original"}, 200)
	id := post["social_feed_id"].(string)
	call(t, api, "/service/social-feed/api/v1/edit", map[string]any{"user": other["user"], "session": other["session"], "social_feed_id": id, "message": "changed"}, 403)
	call(t, api, "/service/social-feed/api/v1/delete", map[string]any{"user": other["user"], "session": other["session"], "social_feed_id": id}, 403)
	call(t, api, "/service/social-feed/api/v1/edit", map[string]any{"user": owner["user"], "session": "wrong", "social_feed_id": id, "message": "changed"}, 401)
}

func TestStageMutationRequiresOwner(t *testing.T) {
	api, _ := newAPI(t, true)
	owner, other := create(t, api), create(t, api)
	props := []map[string]any{{"name": "private"}}
	call(t, api, "/service/stage/api/v1/set", map[string]any{"user": other["user"], "session": other["session"], "user_id": owner["user"], "stage_id": "test", "props": props}, 403)
	call(t, api, "/service/stage/api/v1/set", map[string]any{"user": owner["user"], "session": "wrong", "stage_id": "test", "props": props}, 401)
	call(t, api, "/service/stage/api/v1/set", map[string]any{"user": owner["user"], "session": owner["session"], "stage_id": "test", "props": props}, 200)
}

func TestRecordingMutationRequiresSession(t *testing.T) {
	api, _ := newAPI(t, true)
	owner := create(t, api)
	call(t, api, "/service/recording/api/v2/create", map[string]any{"user": owner["user"], "recording_data": map[string]any{}}, 400)
	call(t, api, "/service/recording/api/v2/update", map[string]any{"recording_id": "missing", "chunk": 1}, 400)
}
