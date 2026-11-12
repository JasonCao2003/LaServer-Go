"""Local api-server smoke test against a running instance.

Usage: python scripts/verify/local_smoke.py https://127.0.0.1:18443
"""
import json
import ssl
import sys
import urllib.request

BASE = sys.argv[1] if len(sys.argv) > 1 else "https://127.0.0.1:18443"
CTX = ssl._create_unverified_context()
results = []


def request(path, body=None, method="POST", headers=None, raw=False):
    data = None
    hdrs = {"Content-Type": "application/json"}
    if headers:
        hdrs.update(headers)
    if body is not None:
        data = json.dumps(body).encode("utf-8")
    req = urllib.request.Request(BASE + path, data=data, headers=hdrs, method=method)
    try:
        with urllib.request.urlopen(req, context=CTX, timeout=15) as resp:
            payload = resp.read()
            if raw:
                return resp.status, payload, dict(resp.headers)
            return resp.status, json.loads(payload or b"{}"), dict(resp.headers)
    except urllib.error.HTTPError as exc:
        payload = exc.read()
        try:
            parsed = json.loads(payload or b"{}")
        except Exception:
            parsed = {"raw": payload[:200].decode("utf-8", "replace")}
        return exc.code, parsed, dict(exc.headers)


def check(name, condition, detail=""):
    results.append((name, bool(condition), detail))
    print(("PASS " if condition else "FAIL ") + name + ("" if condition else " :: " + str(detail)))


def create(name):
    status, body, _ = request("/account/create", {"device_name": name})
    check("create " + name, status == 200 and body.get("user") and body.get("session"), (status, body))
    return body


a = create("local-a")
b = create("local-b")

status, body, _ = request("/account/get_currency", {"user": a["user"], "session": a["session"]})
check("get_currency", status == 200 and "currency" in body, (status, body))

for path, key in [
    ("/account/get_unlocks", "unlocks"),
    ("/account/get_friends", "set_friends"),
    ("/account/get_daily_quests", "daily_quests"),
    ("/account/get_event_schedule", "event_schedule"),
    ("/account/get_spirit_shops", "get_spirit_shops"),
    ("/account/get_collectibles", "collectibles"),
]:
    status, body, _ = request(path, {"user": a["user"], "session": a["session"]})
    check(path, status == 200 and isinstance(body, dict), (status, body if status != 200 else ""))

status, body, _ = request("/account/translate", {})
check("translate", status == 200 and body.get("result") == "ok", (status, body))

status, body, _ = request("/account/set_device_token", {})
check("set_device_token", status == 200 and body == {}, (status, body))

chat = "local smoke chat"
status, body, _ = request("/account/chat/send", {"user": a["user"], "session": a["session"], "msg": chat})
check("chat/send", status == 200 and body.get("result") == "success", (status, body))
status, body, _ = request("/account/get_pending_messages", {"user": b["user"], "session": b["session"], "wait": 2})
check("get_pending_messages", status == 200 and any(m.get("msg") == chat for m in body.get("set_recvd_messages", [])), (status, body))

status, body, _ = request("/account/social_feed/create", {"user": a["user"], "session": a["session"], "pool_type": "MessageBoat", "pool_name": "local", "message": "feed smoke", "extra_data": {"pos": "(1,2,3)"}})
feed_id = body.get("social_feed_id")
check("social_feed/create", status == 200 and feed_id, (status, body))
status, body, _ = request("/account/social_feed/get_user_feeds", {"user": a["user"], "session": a["session"], "pool_type": "MessageBoat"})
check("social_feed/get_user_feeds", status == 200 and body.get("result") == "ok", (status, body))
status, body, _ = request("/account/social_feed/like", {"user": b["user"], "session": b["session"], "social_feed_id": feed_id, "liked": True})
check("social_feed/like", status == 200 and body.get("result") == "ok", (status, body))
status, body, _ = request("/service/social-feed/api/v1/get_by_ids", {"social_feed_ids": [feed_id]})
check("social_feed/get_by_ids", status == 200 and body.get("result") == "ok" and len(body.get("feeds", [])) == 1, (status, body))

status, body, _ = request("/account/geonote/create_geonote2", {"user": a["user"], "session": a["session"], "message": "note smoke", "level_id": 7, "pos": "[1,2,3]"})
note_id = body.get("geonote", {}).get("geonote_id")
check("geonote/create", status == 200 and body.get("result") == "ok" and note_id, (status, body))
status, body, _ = request("/account/geonote/get_user_geonotes", {"user": a["user"], "session": a["session"], "level_id": 7})
check("geonote/get", status == 200 and any(n.get("geonote_id") == note_id for n in body.get("geonotes", [])), (status, body))
status, body, _ = request("/account/geonote/social_geonote2", {"user": b["user"], "session": b["session"], "geonote_id": note_id, "liked": True})
check("geonote/social", status == 200 and body.get("result") == "ok", (status, body))
status, body, _ = request("/account/geonote/edit_message_geonote", {"user": a["user"], "session": a["session"], "geonote_id": note_id, "message": "edited"})
check("geonote/edit", status == 200 and body.get("result") == "ok", (status, body))

status, body, _ = request("/service/stage/api/v1/set", {"user": a["user"], "session": a["session"], "stage_id": "CandleSpace_CNDL", "level_id": "CandleSpace", "props": [{"name": "smoke_prop", "pos": "(1,2,3)"}]})
check("stage/set", status == 200 and body.get("status") == "OK", (status, body))
status, body, _ = request("/service/stage/api/v1/get", {"user": a["user"], "session": a["session"], "stage_id": "CandleSpace_CNDL", "level_id": "CandleSpace", "user_id": "00000000-0000-0000-0000-000000000000"})
check("stage/get", status == 200 and body.get("props") and body.get("props")[0].get("name") == "smoke_prop", (status, body))

status, body, _ = request("/service/recording/api/v2/create", {"user": a["user"], "session": a["session"], "level_id": "CandleSpace", "pool_type": "RecordingCandle", "recording_data": {"frames": [1], "chet": 1}})
recording_id = body.get("recording_id")
check("recording/v2/create", status == 200 and body.get("status") == "OK" and recording_id, (status, body))
status, body, _ = request("/service/recording/api/v2/update", {"user": a["user"], "session": a["session"], "recording_id": recording_id, "chunk": 1, "recording_data": {"frames": [2]}})
check("recording/v2/update", status == 200 and body.get("status") == "OK", (status, body))
status, body, _ = request("/service/recording/api/v2/get", {"recording_id": recording_id})
check("recording/v2/get", status == 200 and body.get("seek_table"), (status, body))
link = body.get("link", "")
if link:
    status, payload, headers = request(link.replace(BASE, ""), method="GET", raw=True, headers={"Range": "bytes=0-3"})
    check("ugc/range", status == 206 and len(payload) == 4, (status, payload[:40], headers.get("Content-Range")))

status, body, _ = request("/service/relationship/api/v1/free_gifts/send", {"user": a["user"], "session": a["session"], "target": b["user"], "gift_type": "gift_heart_wax"})
gift_id = body.get("sent_free_gift", {}).get("gift_id")
check("free_gifts/send", status == 200 and body.get("result") == "ok" and gift_id, (status, body))
status, body, _ = request("/service/relationship/api/v1/free_gifts/get_pending", {"user": b["user"], "session": b["session"]})
check("free_gifts/get_pending", status == 200 and len(body.get("set_recvd_free_gifts", [])) >= 1, (status, body))
status, body, _ = request("/service/relationship/api/v1/free_gifts/claim", {"user": b["user"], "session": b["session"], "gift_id": gift_id})
check("free_gifts/claim", status == 200 and body.get("result") == "ok", (status, body))

status, body, _ = request("/admin/api/friends_set", {"user": a["user"], "friend": b["user"], "action": "add"})
check("admin/friends_set", status == 200 and body.get("ok") is True, (status, body))
status, body, _ = request("/admin/api/friends_list?user=" + a["user"], method="GET")
check("admin/friends_list", status == 200 and body.get("ok") is True and body.get("count", 0) >= 1, (status, body))
status, body, _ = request("/admin/api/friends_clear", {"user": a["user"]})
check("admin/friends_clear", status == 200 and body.get("ok") is True, (status, body))

status, body, _ = request("/service/reservation/api/v1/reserve", {})
check("reservation", status == 200 and body.get("status") is True, (status, body))
status, body, _ = request("/account/find_previous_or_empty", {"user": b["user"], "session": b["session"]})
check("find_previous_or_empty", status == 200 and body.get("uri"), (status, body))
status, body, _ = request("/account/join_friend_game", {"user": a["user"], "session": a["session"], "friend_id": b["user"]})
check("join_friend_game", status == 200 and body.get("move_to_game"), (status, body))

status, body, _ = request("/service/message/api/v2/get_all", {"user": a["user"], "max_count": 5})
check("mailbox/get_all", status == 200 and body.get("status") == "OK", (status, body))

status, body, _ = request("/admin/api/friends_set", {"user": a["user"], "friend": b["user"], "action": "add"})
check("admin/friends_set (re-add)", status == 200 and body.get("ok") is True, (status, body))

status, body, _ = request("/account/set_friend_name", {"user": a["user"], "session": a["session"], "friend": b["user"], "name": "Buddy"})
check("set_friend_name", status == 200 and body.get("result") is True, (status, body))

status, body, _ = request("/account/set_friend_block", {"user": a["user"], "session": a["session"], "friend": b["user"], "blocked": True})
check("set_friend_block", status == 200 and body.get("result") is True, (status, body))
status, body, _ = request("/account/get_blocked_friends", {"user": a["user"], "session": a["session"]})
check("get_blocked_friends", status == 200 and any(f.get("friend_id") == b["user"] for f in body.get("set_blocked_friends", {}).get("friends", [])), (status, body))
status, body, _ = request("/account/set_friend_block", {"user": a["user"], "session": a["session"], "friend": b["user"], "blocked": False})
check("set_friend_block (unblock)", status == 200 and body.get("result") is True, (status, body))

status, body, _ = request("/account/set_outfit", {"user": a["user"], "session": a["session"], "outfit": {"wing": {"id": 1, "tex": 2}, "height": 1.5, "scale": 0.1}})
check("set_outfit", status == 200 and body.get("result") == "ok", (status, body))
status, body, _ = request("/account/get_remote_outfit", {"user": b["user"], "session": b["session"], "target": a["user"]})
check("get_remote_outfit", status == 200 and body.get("get_remote_outfit", {}).get("height") == 1.5, (status, body))

status, body, _ = request("/account/give_candle", {"user": a["user"], "session": a["session"], "recv_user": b["user"], "ability_id": 26, "ability_cost": 0})
check("give_candle", status == 200 and body.get("result") is True, (status, body))

status, body, _ = request("/account/set_joinable", {"joinable": True})
check("set_joinable", status == 200 and body.get("joinable") is True, (status, body))

status, body, _ = request("/account/star/get", {"user": a["user"], "session": a["session"]})
check("star/get", status == 200 and "get_star_tag_defs" in body, (status, body))

status, body, _ = request("/service/relationship/api/v1/update_friend_constellation_pages", {"user": a["user"], "session": a["session"], "pages": [{"friends": [{"friend_id": b["user"], "constellation_node_index": 1}]}]})
check("constellation/update", status == 200 and body.get("result") is True, (status, body))
status, body, _ = request("/account/get_friend_statues", {"user": a["user"], "session": a["session"]})
check("get_friend_statues", status == 200 and body.get("result") == "success", (status, body))

status, body, _ = request("/account/send", {"user": b["user"], "session": b["session"], "target": a["user"], "gift_type": "gift_heart_wax"})
account_gift_id = body.get("sent_message", {}).get("gift_id")
check("account/send gift", status == 200 and body.get("result") == "ok" and account_gift_id, (status, body))
status, body, _ = request("/account/claim_message_gift", {"user": a["user"], "session": a["session"], "gift_id": account_gift_id})
check("claim_message_gift", status == 200 and body.get("result") == "ok", (status, body))

failed = [name for name, ok, _ in results if not ok]
print("\nSUMMARY: %d passed, %d failed" % (len(results) - len(failed), len(failed)))
if failed:
    print("FAILED:", ", ".join(failed))
    sys.exit(1)
