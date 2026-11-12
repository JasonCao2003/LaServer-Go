package service

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
	"strings"
)

var commonBoxMagic = []string{"energy", "glow", "glow_rainbow", "grow", "shrink", "huge", "tiny", "gravity", "fireworks", "dnd", "bonfire", "beachball", "rainbow", "pinwheel", "table", "seesaw", "swing", "candle_trick", "crab_rock_trick", "currency_trick", "ground_trail_rainbow", "butterfly_pastel", "bloom_sakura", "resize_potion"}
var rareBoxMagic = []string{"glow_tgc", "hairtint_rainbow", "trail_rainbow", "message_boat", "message_crane", "message_gondola", "message_lantern", "message_lotus", "message_rose", "dark_dragon_firework", "firework_crab", "firework_jelly", "firework_manta", "sharedspace_candle", "outfit_prop_twinribbonposter", "height_small_3x_handhold", "potion_growth_huge", "potion_growth_tiny", "potion_growth_keychain", "teleport_forest_elder", "teleport_prairie_elder", "teleport_sanctuary_island"}

func (a *Service) boxMagicID(name, ua string) int64 {
	pool := commonBoxMagic
	if strings.Contains(strings.ToLower(name), "rare") {
		pool = rareBoxMagic
	}
	allowed := map[string]int64{}
	for _, d := range a.versionMagicDefinitions("get_consumable_defs", "account__consumable__get_consumable_defs.json", ua) {
		allowed[text(d, "name")] = defInt(d["id"])
	}
	ids := []int64{}
	for _, item := range pool {
		if id := allowed[item]; id != 0 {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return allowed["energy"]
	}
	return ids[int(spiritUnlockID(name))%len(ids)]
}
func redeemBoxError(message string) Object {
	return Object{"result": "error", "error": message, "currency": Object{}, "get_consumables": []Object{}, "get_lootboxes": []Object{}}
}
func (a *Service) RedeemBox(r Request, m Object) (Object, error) {
	uid := text(m, "user")
	if uid == "" {
		uid = text(m, "user_id")
	}
	box := text(m, "box_id")
	if box == "" {
		box = text(m, "lootbox_id")
	}
	if uid == "" || text(m, "session") == "" || box == "" {
		return redeemBoxError("Invalid request body"), nil
	}
	if _, err := a.magicUser(r, m, true); err != nil {
		return redeemBoxError("Invalid session or user not found"), nil
	}
	node := a.spiritNode(box)
	if node == nil || text(node, "typ") != "lootbox" {
		return redeemBoxError("Unknown lootbox"), nil
	}
	cid := a.boxMagicID(box, r.UserAgent)
	if cid == 0 {
		return redeemBoxError("No supported consumable"), nil
	}
	field := text(node, "ctyp")
	if field == "season_heart" {
		field = "heart"
	}
	if field == "" {
		field = "candles"
	}
	cost := defInt(node["cst"])
	if cost <= 0 {
		cost = 1
	}
	if !currencyField(field) {
		return redeemBoxError("Unsupported currency"), nil
	}
	tx, err := a.store.BeginTx(r.Context, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var quantity int64
	if err = tx.QueryRowContext(r.Context, repository.ReadOpenBoxQuantity, uid, box).Scan(&quantity); err != nil || quantity <= 0 {
		return redeemBoxError("Lootbox unavailable"), nil
	}
	if _, err = tx.ExecContext(r.Context, repository.EnsureWingCurrency, uid); err != nil {
		return nil, err
	}
	ok, err := tx.SpendCurrency(r.Context, uid, field, cost)
	if err != nil {
		return nil, err
	}
	if !ok {
		currency, e := currencySnapshot(r.Context, tx, uid)
		if e != nil {
			return nil, e
		}
		_ = tx.Rollback()
		items, e := a.consumables(r, uid, 0)
		if e != nil {
			return nil, e
		}
		boxes, e := a.openLootboxes(r, uid)
		if e != nil {
			return nil, e
		}
		return Object{"result": "insufficient_funds", "currency": currency, "get_consumables": items, "get_lootboxes": boxes}, nil
	}
	result, err := tx.ExecContext(r.Context, repository.UseLootbox, uid, box)
	if err != nil {
		return nil, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if changed != 1 {
		return redeemBoxError("Lootbox unavailable"), nil
	}
	if _, err = tx.ExecContext(r.Context, repository.GiveConsumable, uid, cid, 1); err != nil {
		return nil, err
	}
	currency, err := currencySnapshot(r.Context, tx, uid)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	items, err := a.consumables(r, uid, 0)
	if err != nil {
		return nil, err
	}
	boxes, err := a.openLootboxes(r, uid)
	if err != nil {
		return nil, err
	}
	return Object{"result": "ok", "currency": currency, "get_consumables": items, "get_lootboxes": boxes}, nil
}
