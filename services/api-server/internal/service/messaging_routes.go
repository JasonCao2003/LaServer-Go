package service

import (
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

func nowMillis() int64 { return time.Now().UnixMilli() }

func scalarString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case json.Number:
		return x.String()
	case float64:
		return strconvFormatFloat(x)
	default:
		return ""
	}
}

func strconvFormatFloat(v float64) string {
	encoded, _ := json.Marshal(v)
	return string(encoded)
}

func (a *Service) messageAuth(r Request, m Object) (string, error) {
	uid, session := text(m, "user"), text(m, "session")
	if uid == "" || session == "" {
		return "", fail(400, "Missing user or session")
	}
	u, err := a.findUser(r.Context, "id", uid)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != session) {
		return "", fail(401, "Invalid session")
	}
	if err != nil {
		return "", err
	}
	return uid, nil
}

func chatPayload(uid, msg, ch, id string, stamp int64) Object {
	return Object{"type": "chat", "sender_id": uid, "msg_id": id, "msg": msg, "ch": ch, "timestamp": stamp, "result": "success"}
}

func (a *Service) GetPendingMessages(r Request, m Object) (Object, error) {
	uid, err := a.messageAuth(r, m)
	if err != nil {
		return nil, err
	}
	wait, _ := floatValue(m["wait"])
	if wait > 30 {
		wait = 30
	}
	if wait > 0 {
		deadline := time.Now().Add(time.Duration(wait * float64(time.Second)))
		for time.Now().Before(deadline) && a.bus.empty(uid) {
			if r.Context.Err() != nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	recvd := []Object{}
	for _, entry := range a.bus.drain(uid) {
		recvd = append(recvd, Object{"type": entry["type"], "sender_id": entry["sender_id"], "msg_id": entry["msg_id"], "msg": entry["msg"], "ch": entry["ch"], "timestamp": entry["timestamp"], "result": "success"})
	}
	stamp := now()
	receivedGifts, err := a.store.ReceivedGifts(r.Context, uid, stamp)
	if err != nil {
		return nil, err
	}
	for _, gift := range receivedGifts {
		recvd = append(recvd, recvdMessageEntry(gift))
	}
	sentGifts, err := a.store.SentGifts(r.Context, uid, stamp)
	if err != nil {
		return nil, err
	}
	sent := []Object{}
	for _, gift := range sentGifts {
		sent = append(sent, sentGiftEntry(gift))
	}
	return Object{"set_sent_messages": sent, "set_recvd_messages": recvd}, nil
}

func (a *Service) ChatSend(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" || text(m, "session") == "" {
		return Object{"result": "error", "message": "Missing parameters: user, session"}, nil
	}
	u, err := a.findUser(r.Context, "id", uid)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != text(m, "session")) {
		return Object{"result": "error", "message": "Invalid session"}, nil
	}
	if err != nil {
		return Object{"result": "error", "message": "server_error"}, nil
	}
	msg := text(m, "msg")
	if msg == "" {
		msg = text(m, "chat_msg")
	}
	if msg == "" {
		msg = text(m, "message")
	}
	ch := text(m, "ch")
	if ch == "" {
		ch = text(m, "channel")
	}
	if ch == "" {
		ch = "global"
	}
	id, err := uuid()
	if err != nil {
		return Object{"result": "error", "message": "server_error"}, nil
	}
	stamp := nowMillis()
	entry := chatPayload(uid, msg, ch, id, stamp)
	if err = a.broadcastChat(entry); err != nil {
		a.pushChat(uid, entry)
	}
	return Object{"result": "success", "msg_id": id, "timestamp": stamp}, nil
}

// AccountSend handles /account/send and /account/send_message: chat-shaped
// bodies delegate to chat, everything else sends a gift.
func (a *Service) AccountSend(r Request, m Object) (Object, error) {
	if text(m, "msg") != "" || text(m, "chat_msg") != "" || text(m, "message") != "" {
		return a.ChatSend(r, m)
	}
	return a.accountGiftSend(r, m)
}

func (a *Service) accountGiftSend(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	if uid == "" || text(m, "session") == "" {
		return Object{"result": "error", "error": "Invalid request body"}, nil
	}
	u, err := a.findUser(r.Context, "id", uid)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != text(m, "session")) {
		return Object{"result": "error", "error": "Invalid session"}, nil
	}
	if err != nil {
		return Object{"result": "error", "error": "server_error"}, nil
	}
	target := text(m, "target")
	if target == "" {
		target = text(m, "recv_user")
	}
	if target == "" {
		target = text(m, "friend_id")
	}
	if target == "" {
		return Object{"result": "error", "error": "Missing target"}, nil
	}
	target = repository.CleanCreator(target)
	giftType := normalizeGiftType(m["gift_type"])
	if giftType == "" {
		giftType = normalizeGiftType(m["type"])
	}
	if giftType == "" {
		return Object{"result": "error", "error": "Unknown gift_type", "currency": a.currencySnapshot(uid)}, nil
	}
	exists, err := a.store.HasUser(r.Context, target)
	if err != nil {
		return nil, err
	}
	if !exists {
		return Object{"result": "error", "error": "receiver_not_found", "currency": a.currencySnapshot(uid)}, nil
	}
	spec := giftSpecs[giftType]
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if spec.pay > 0 {
		if err = tx.EnsureCurrency(r.Context, uid); err != nil {
			return nil, err
		}
		balance, err := tx.CurrencyBalance(r.Context, uid, spec.payType)
		if err != nil {
			return nil, err
		}
		if balance < spec.pay {
			return Object{"result": "error", "error": "Not enough currency", "currency": a.currencySnapshot(uid)}, nil
		}
		ok, err := tx.SpendCurrency(r.Context, uid, spec.payType, spec.pay)
		if err != nil || !ok {
			return Object{"result": "error", "error": "Not enough currency", "currency": a.currencySnapshot(uid)}, nil
		}
	}
	stamp := now()
	gift := repository.FreeGift{SenderID: uid, ReceiverID: target, GiftType: giftType, GiftID: strconv.FormatInt(newGiftID(), 10), SentAt: stamp}
	gift.ExpiresAt.Int64 = stamp + giftTTLSeconds
	gift.ExpiresAt.Valid = true
	if err = tx.InsertGift(r.Context, gift); err != nil {
		return Object{"result": "error", "error": "server_error", "currency": a.currencySnapshot(uid)}, nil
	}
	if err = tx.Commit(); err != nil {
		return Object{"result": "error", "error": "server_error", "currency": a.currencySnapshot(uid)}, nil
	}
	entry := sentGiftEntry(gift)
	entry["timestamp"] = nowMillis()
	return Object{"res": "ok", "result": "ok", "currency": a.currencySnapshot(uid), "sent_message": entry, "timestamp": nowMillis()}, nil
}

func (a *Service) Translate(_ Request, _ Object) (Object, error) {
	return Object{"result": "ok"}, nil
}

func (a *Service) SetDeviceToken(_ Request, _ Object) (Object, error) {
	return Object{}, nil
}

type relationAbility struct {
	ID            int64   `json:"id"`
	Cost          int64   `json:"cost"`
	CurrencyType  string  `json:"currency_type"`
	Permanent     *bool   `json:"permanent"`
	RewardType    string  `json:"reward_type"`
	RewardAmount  int64   `json:"reward_amount"`
	GiveAlso      []int64 `json:"give_also"`
	TrustActivity string  `json:"trust_activity"`
}

func (a *Service) relationAbilities() []relationAbility {
	var defs struct {
		Defs []relationAbility `json:"relationship_defs"`
	}
	if err := legacyResponse("relationship_abilities", &defs); err != nil {
		return []relationAbility{}
	}
	return defs.Defs
}

func abilityContains(ids []int64, id int64) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

func unionAbilities(base, extra []int64) []int64 {
	out := append([]int64{}, base...)
	for _, id := range extra {
		if !abilityContains(out, id) {
			out = append(out, id)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func removeHint(hints []int64, hint int64) []int64 {
	out := []int64{}
	for _, h := range hints {
		if h != hint {
			out = append(out, h)
		}
	}
	return out
}

func giveFailure(code string, currency Object, count int64) Object {
	return Object{"res": "ok", "result": code, "currency": currency, "update_friends": []Object{}, "set_sent_messages": []Object{}, "set_friend_count": count, "update_online_friends": []Object{}, "trust_update": Object{"result": code, "applied": false}}
}

func friendUpdateEntry(e repository.RelationEntry) Object {
	return Object{"abilities": intList(e.Abilities), "friend_id": e.ID, "given": e.Given, "hints": intList(e.Hints), "last_seen_outfit": nil, "level": e.Level, "nickname": e.Nickname, "recvd": e.Received, "soft_deleted": nil, "they_soft_deleted": nil, "unlinked_star_sku": "", "when_created": relationTime(e.WhenCreated)}
}

func (a *Service) GiveCandle(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	if uid == "" || text(m, "session") == "" {
		return nil, fail(400, "Invalid request body")
	}
	u, err := a.findUser(r.Context, "id", uid)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != text(m, "session")) {
		return nil, fail(401, "invalid session")
	}
	if err != nil {
		return nil, err
	}
	target := text(m, "recv_user")
	if target == "" {
		target = text(m, "receiver")
	}
	if target == "" {
		target = text(m, "to")
	}
	target = repository.CleanCreator(target)
	abilityID := number(m, "ability_id", 0)
	abilityCost := number(m, "ability_cost", 0)
	count, _ := a.store.CountAddedFriends(r.Context, uid)
	if target == "" || abilityID == 0 {
		return giveFailure("missing_target_or_ability", a.currencySnapshot(uid), count), nil
	}
	exists, err := a.store.HasUser(r.Context, target)
	if err != nil {
		return nil, err
	}
	if !exists {
		return giveFailure("receiver_not_found", a.currencySnapshot(uid), count), nil
	}
	def := relationAbility{CurrencyType: "candles", Permanent: nil}
	for _, candidate := range a.relationAbilities() {
		if candidate.ID == abilityID {
			def = candidate
			break
		}
	}
	cost := abilityCost
	if cost <= 0 {
		cost = def.Cost
	}
	if cost < 0 {
		cost = 0
	}
	isWax := cost <= 0
	curType := def.CurrencyType
	if !currencyField(curType) {
		curType = "candles"
	}
	permanent := true
	if def.Permanent != nil {
		permanent = *def.Permanent
	}
	ids := []int64{abilityID}
	ids = append(ids, def.GiveAlso...)
	if abilityContains(ids, 1) {
		ids = append(ids, 1, 2, 9, 10, 19)
	}
	ids = sortedUnique(ids)

	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	stamp := now()
	if err = tx.EnsureRelationPairTx(r.Context, uid, target, intsJSON(ids), stamp); err != nil {
		return giveFailure("server_error", a.currencySnapshot(uid), count), nil
	}
	own, found, err := tx.RelationTx(r.Context, uid, target)
	if err != nil || !found {
		return giveFailure("not_friends", a.currencySnapshot(uid), count), nil
	}
	other, otherFound, err := tx.RelationTx(r.Context, target, uid)
	if err != nil {
		return giveFailure("server_error", a.currencySnapshot(uid), count), nil
	}
	ownAbilities := intList(own.Abilities)
	already := abilityContains(ownAbilities, abilityID)
	charged := false
	result := any(true)
	if permanent && already && own.Given > 0 {
		result = "already_unlocked"
	} else if !isWax {
		if err = tx.EnsureCurrency(r.Context, uid); err != nil {
			return giveFailure("server_error", a.currencySnapshot(uid), count), nil
		}
		balance, err := tx.CurrencyBalance(r.Context, uid, curType)
		if err != nil {
			return giveFailure("server_error", a.currencySnapshot(uid), count), nil
		}
		if balance < cost {
			// Legacy logs the shortfall but still clamps the balance at zero.
		}
		if err = tx.ClampSpend(r.Context, uid, curType, cost); err != nil {
			return giveFailure("server_error", a.currencySnapshot(uid), count), nil
		}
		charged = true
	}
	newAbilities := unionAbilities(ownAbilities, ids)
	ownHints := intList(own.Hints)
	for _, id := range ids {
		if !abilityContains(ownAbilities, id) {
			ownHints = append(ownHints, id)
		}
	}
	if strings.TrimSpace(own.Nickname) != "" {
		ownHints = removeHint(ownHints, 1)
	}
	given := own.Given
	if charged {
		given++
	}
	if err = tx.SaveRelationTx(r.Context, uid, target, intsJSON(newAbilities), intsJSON(sortedUnique(ownHints)), given, own.Received, stamp); err != nil {
		return giveFailure("server_error", a.currencySnapshot(uid), count), nil
	}
	if otherFound {
		otherAbilities := unionAbilities(intList(other.Abilities), ids)
		otherHints := intList(other.Hints)
		for _, id := range ids {
			if !abilityContains(intList(other.Abilities), id) {
				otherHints = append(otherHints, id)
			}
		}
		if strings.TrimSpace(other.Nickname) != "" {
			otherHints = removeHint(otherHints, 1)
		}
		received := other.Received
		if charged {
			received++
		}
		if err = tx.SaveRelationTx(r.Context, target, uid, intsJSON(otherAbilities), intsJSON(sortedUnique(otherHints)), other.Given, received, stamp); err != nil {
			return giveFailure("server_error", a.currencySnapshot(uid), count), nil
		}
	}
	if charged && def.RewardAmount > 0 && currencyField(def.RewardType) {
		if err = tx.EnsureCurrency(r.Context, target); err != nil {
			return giveFailure("server_error", a.currencySnapshot(uid), count), nil
		}
		if err = tx.AddCurrency(r.Context, target, def.RewardType, def.RewardAmount); err != nil {
			return giveFailure("server_error", a.currencySnapshot(uid), count), nil
		}
	}
	if err = tx.Commit(); err != nil {
		return giveFailure("server_error", a.currencySnapshot(uid), count), nil
	}
	own, found, err = a.store.Relation(r.Context, uid, target)
	if err != nil || !found {
		return giveFailure("server_error", a.currencySnapshot(uid), count), nil
	}
	sentGifts, err := a.store.SentGifts(r.Context, uid, stamp)
	if err != nil {
		return giveFailure("server_error", a.currencySnapshot(uid), count), nil
	}
	sent := []Object{}
	for _, gift := range sentGifts {
		sent = append(sent, sentGiftEntry(gift))
	}
	count, _ = a.store.CountAddedFriends(r.Context, uid)
	source := def.TrustActivity
	if source == "" {
		source = "fake_full_trust"
	}
	return Object{"res": "ok", "result": result, "currency": a.currencySnapshot(uid), "update_friends": []Object{friendUpdateEntry(own)}, "set_sent_messages": sent, "set_friend_count": count, "update_online_friends": []Object{Object{"friend_id": target, "level_id": 0}}, "trust_update": Object{"result": "ok", "applied": true, "target_player": target, "xp_delta": 0, "xp": 999999, "tier": 999, "source": source}}, nil
}

func (a *Service) giftIDs(m Object) []string {
	out := []string{}
	add := func(v any) {
		if s := scalarString(v); s != "" {
			out = append(out, s)
		}
	}
	add(m["gift_id"])
	for _, key := range []string{"gift_ids", "messages", "gifts"} {
		list, _ := m[key].([]any)
		for _, item := range list {
			if obj, ok := item.(map[string]any); ok {
				add(obj["gift_id"])
				add(obj["id"])
			} else {
				add(item)
			}
		}
	}
	return out
}

func (a *Service) ClaimMessageGift(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	if uid == "" || text(m, "session") == "" {
		return Object{"result": "invalid"}, nil
	}
	u, err := a.findUser(r.Context, "id", uid)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != text(m, "session")) {
		return Object{"result": "invalid"}, nil
	}
	if err != nil {
		return Object{"result": "invalid"}, nil
	}
	fromID := repository.CleanCreator(text(m, "from_id"))
	ids := a.giftIDs(m)
	if len(ids) == 0 && fromID == "" {
		return Object{"result": "invalid"}, nil
	}
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return Object{"result": "invalid"}, nil
	}
	defer tx.Rollback()
	stamp := now()
	claimed := []Object{}
	claim := func(g repository.FreeGift, ok bool) error {
		if !ok || g.Received {
			return nil
		}
		spec := giftSpecs[g.GiftType]
		if spec.recvType != "" && spec.recv > 0 {
			if err := tx.EnsureCurrency(r.Context, uid); err != nil {
				return err
			}
			if err := tx.AddCurrency(r.Context, uid, spec.recvType, spec.recv); err != nil {
				return err
			}
		}
		if err := tx.MarkGiftReceived(r.Context, g.ID, stamp); err != nil {
			return err
		}
		g.Received = true
		claimed = append(claimed, recvdMessageEntry(g))
		return nil
	}
	for _, id := range ids {
		gift, found, err := tx.GiftByID(r.Context, uid, id)
		if err != nil {
			return Object{"result": "invalid"}, nil
		}
		if !found && fromID != "" {
			gift, found, err = tx.LatestUnreceivedGiftFrom(r.Context, uid, fromID)
			if err != nil {
				return Object{"result": "invalid"}, nil
			}
		}
		if err = claim(gift, found); err != nil {
			return Object{"result": "invalid"}, nil
		}
	}
	if len(ids) == 0 && fromID != "" {
		gifts, err := tx.UnreceivedGiftsFrom(r.Context, uid, fromID)
		if err != nil {
			return Object{"result": "invalid"}, nil
		}
		for _, gift := range gifts {
			if err = claim(gift, true); err != nil {
				return Object{"result": "invalid"}, nil
			}
		}
	}
	if err = tx.Commit(); err != nil {
		return Object{"result": "invalid"}, nil
	}
	badge, _ := a.store.PendingGiftCount(r.Context, uid, stamp)
	return Object{"result": "ok", "currency": a.currencySnapshot(uid), "set_recvd_messages": claimed, "set_app_badge_number": badge, "get_buffs": []Object{}, "get_buffs_sign": emptyBuffSign}, nil
}

func (a *Service) MessageGetAllV2(r Request, m Object) (Object, error) {
	return a.mailboxList(r, m, true), nil
}

func (a *Service) MessageGetAllV1(r Request, m Object) (Object, error) {
	return a.mailboxList(r, m, false), nil
}

func (a *Service) mailboxList(r Request, m Object, byVersion bool) Object {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	limit := number(m, "max_count", 32)
	cfg := a.mailbox()
	var version [3]int
	known := false
	if byVersion {
		version, known = mailboxVersion(r.UserAgent)
	}
	messages := []Object{}
	for i, mail := range cfg.Mails {
		if limit >= 0 && int64(i) >= limit {
			break
		}
		if limit < 0 && int64(i) >= int64(len(cfg.Mails))+limit {
			break
		}
		if !mailVisible(mail, version, known) {
			continue
		}
		messages = append(messages, Object{"payload": mailPayload(mail, cfg.FromID, uid, now(), false)})
	}
	return Object{"status": "OK", "messages": messages}
}

func mailboxVersion(ua string) ([3]int, bool) {
	major, minor, patch, known := clientVersion(ua)
	return [3]int{major, minor, patch}, known
}

func (a *Service) MessageGet(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	requested, _ := m["messages"].([]any)
	cfg := a.mailbox()
	version, known := mailboxVersion(r.UserAgent)
	byID := map[string]mailMessage{}
	for _, mail := range cfg.Mails {
		byID[mail.ID] = mail
	}
	out := []Object{}
	for _, item := range requested {
		id := ""
		if obj, ok := item.(map[string]any); ok {
			id = scalarString(obj["id"])
		} else {
			id = scalarString(item)
		}
		mail, ok := byID[id]
		if !ok || !mailVisible(mail, version, known) {
			continue
		}
		out = append(out, Object{"payload": mailPayload(mail, cfg.FromID, uid, now(), true)})
	}
	return Object{"status": "OK", "update_messages": out}, nil
}

func (a *Service) MarkMessagesReceived(r Request, m Object) (Object, error) {
	if text(m, "user") == "" || text(m, "session") == "" {
		return nil, fail(400, "Invalid request body")
	}
	if _, ok := m["messages"]; !ok {
		return nil, fail(400, "Invalid request body")
	}
	uid := text(m, "user")
	u, err := a.findUser(r.Context, "id", uid)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != text(m, "session")) {
		return nil, fail(401, "Invalid session or user not found")
	}
	if err != nil {
		return nil, err
	}
	stamp := now()
	out := []Object{}
	items, _ := m["messages"].([]any)
	for _, item := range items {
		id := ""
		if obj, ok := item.(map[string]any); ok {
			id = scalarString(obj["id"])
		}
		if id == "" {
			continue
		}
		if err = a.store.MarkMessageReceived(r.Context, uid, id, stamp); err != nil {
			return nil, err
		}
		out = append(out, Object{"payload": Object{"headers": Object{"id": id, "received_at": stamp}}})
	}
	return Object{"status": "OK", "messages": out}, nil
}

func (a *Service) MarkMessagesSeen(_ Request, m Object) (Object, error) {
	stamp := now()
	out := []Object{}
	items, _ := m["messages"].([]any)
	for _, item := range items {
		id := ""
		if obj, ok := item.(map[string]any); ok {
			id = scalarString(obj["id"])
		} else {
			id = scalarString(item)
		}
		out = append(out, Object{"payload": Object{"headers": Object{"id": id, "seen_at": stamp}}})
	}
	return Object{"status": "OK", "messages": out}, nil
}

func (a *Service) FreeGiftsGetPending(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" || text(m, "session") == "" {
		return nil, fail(400, "Invalid request body")
	}
	u, err := a.findUser(r.Context, "id", uid)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != text(m, "session")) {
		return nil, fail(401, "Invalid session or user not found")
	}
	if err != nil {
		return nil, err
	}
	stamp := now()
	sentGifts, err := a.store.SentGifts(r.Context, uid, stamp)
	if err != nil {
		sentGifts = []repository.FreeGift{}
	}
	recvGifts, err := a.store.ReceivedGifts(r.Context, uid, stamp)
	if err != nil {
		recvGifts = []repository.FreeGift{}
	}
	sent := []Object{}
	for _, gift := range sentGifts {
		sent = append(sent, sentGiftEntry(gift))
	}
	recv := []Object{}
	for _, gift := range recvGifts {
		recv = append(recv, recvdGiftEntry(gift))
	}
	return Object{"set_sent_free_gifts": sent, "set_recvd_free_gifts": recv}, nil
}

func (a *Service) FreeGiftsSend(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	if uid == "" || text(m, "session") == "" {
		return Object{"result": "invalid"}, nil
	}
	u, err := a.findUser(r.Context, "id", uid)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != text(m, "session")) {
		return Object{"result": "invalid"}, nil
	}
	if err != nil {
		return Object{"result": "invalid"}, nil
	}
	target := text(m, "target")
	if target == "" {
		target = text(m, "recv_user")
	}
	if target == "" {
		target = text(m, "to")
	}
	target = repository.CleanCreator(target)
	if target == "" || target == uid {
		return Object{"result": "invalid"}, nil
	}
	giftType := normalizeGiftType(m["gift_type"])
	if giftType == "" {
		return Object{"result": "invalid"}, nil
	}
	exists, err := a.store.HasUser(r.Context, target)
	if err != nil || !exists {
		return Object{"result": "invalid"}, nil
	}
	spec := giftSpecs[giftType]
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return Object{"result": "invalid"}, nil
	}
	defer tx.Rollback()
	if spec.pay > 0 {
		if err = tx.EnsureCurrency(r.Context, uid); err != nil {
			return Object{"result": "invalid"}, nil
		}
		ok, err := tx.SpendCurrency(r.Context, uid, spec.payType, spec.pay)
		if err != nil || !ok {
			return Object{"result": "not_enough_currency"}, nil
		}
	}
	stamp := now()
	gift := repository.FreeGift{SenderID: uid, ReceiverID: target, GiftType: giftType, GiftID: strconv.FormatInt(newGiftID(), 10), SentAt: stamp}
	gift.ExpiresAt.Int64 = stamp + giftTTLSeconds
	gift.ExpiresAt.Valid = true
	if err = tx.InsertGift(r.Context, gift); err != nil {
		return Object{"result": "invalid"}, nil
	}
	if err = tx.Commit(); err != nil {
		return Object{"result": "invalid"}, nil
	}
	return Object{"result": "ok", "sent_free_gift": sentGiftEntry(gift)}, nil
}

func (a *Service) FreeGiftsClaim(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	if uid == "" || text(m, "session") == "" {
		return Object{"result": "invalid"}, nil
	}
	u, err := a.findUser(r.Context, "id", uid)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && stringValue(u.Session) != text(m, "session")) {
		return Object{"result": "invalid"}, nil
	}
	if err != nil {
		return Object{"result": "invalid"}, nil
	}
	fromID := repository.CleanCreator(text(m, "from_id"))
	giftID := scalarString(m["gift_id"])
	if giftID == "" && fromID == "" {
		return Object{"result": "invalid"}, nil
	}
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return Object{"result": "invalid"}, nil
	}
	defer tx.Rollback()
	stamp := now()
	gift, found, err := tx.GiftByID(r.Context, uid, giftID)
	if err != nil {
		return Object{"result": "invalid"}, nil
	}
	if !found && fromID != "" {
		gift, found, err = tx.LatestUnreceivedGiftFrom(r.Context, uid, fromID)
		if err != nil {
			return Object{"result": "invalid"}, nil
		}
	}
	if !found {
		return Object{"result": "invalid"}, nil
	}
	if !gift.Received {
		spec := giftSpecs[gift.GiftType]
		if spec.recvType != "" && spec.recv > 0 {
			if err = tx.EnsureCurrency(r.Context, uid); err != nil {
				return Object{"result": "invalid"}, nil
			}
			if err = tx.AddCurrency(r.Context, uid, spec.recvType, spec.recv); err != nil {
				return Object{"result": "invalid"}, nil
			}
		}
		if err = tx.MarkGiftReceived(r.Context, gift.ID, stamp); err != nil {
			return Object{"result": "invalid"}, nil
		}
	}
	if err = tx.Commit(); err != nil {
		return Object{"result": "invalid"}, nil
	}
	badge, _ := a.store.PendingGiftCount(r.Context, uid, stamp)
	return Object{"result": "ok", "update_currency": a.currencySnapshot(uid), "delete_claimed_free_gift": giftIDInt(gift), "get_buffs": []Object{}, "get_buffs_sign": emptyBuffSign, "set_app_badge_number": badge}, nil
}
