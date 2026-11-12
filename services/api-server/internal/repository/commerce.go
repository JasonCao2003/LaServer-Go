package repository

import (
	"context"
	"database/sql"
)

var spendableCurrency = map[string]bool{"candles": true, "heart": true, "heart_wax": true, "prestige": true, "prestige_vip": true, "prestige_wax": true, "season_candle": true, "season_wax": true, "season_wax_14": true, "season_heart": true, "season_pass_token": true, "vip": true, "wax": true}

func CurrencyRead(field string) Statement {
	if !spendableCurrency[field] {
		panic("invalid currency field")
	}
	return Statement(`SELECT COALESCE(` + field + `,0) FROM user_currency WHERE user_id=?`)
}
func (t *Tx) SpendCurrency(ctx context.Context, uid, field string, amount int64) (bool, error) {
	if !spendableCurrency[field] || amount < 0 {
		return false, sql.ErrNoRows
	}
	if amount == 0 {
		return true, nil
	}
	q := Statement(`UPDATE user_currency SET ` + field + `=COALESCE(` + field + `,0)-? WHERE user_id=? AND COALESCE(` + field + `,0)>=?`)
	result, err := t.ExecContext(ctx, q, amount, uid, amount)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}
func (t *Tx) GrantCurrency(ctx context.Context, uid, field string, amount int64) error {
	if !spendableCurrency[field] || amount < 0 {
		return sql.ErrNoRows
	}
	_, err := t.ExecContext(ctx, Statement(`UPDATE user_currency SET `+field+`=COALESCE(`+field+`,0)+? WHERE user_id=?`), amount, uid)
	return err
}

const (
	CountShopItems            Statement = `SELECT count(*) FROM shop_item`
	InsertShopItem            Statement = `INSERT INTO shop_item(name,unlockid,unlock_cost,currency_type,listed,unlock_dep,visual_dep,x,y) VALUES(?,?,?,?,?,?,?,?,?) ON CONFLICT(unlockid) DO NOTHING`
	ReadShopItems             Statement = `SELECT name,unlockid,unlock_cost,currency_type,listed,unlock_dep,visual_dep,x,y FROM shop_item WHERE listed=1 ORDER BY id`
	ReadOpenLootboxes         Statement = `SELECT lootbox_id,lootbox_type,quantity,acquired_at FROM lootbox WHERE user_id=? AND opened=0 AND (expires_at IS NULL OR expires_at>=?) ORDER BY id`
	ReadInviteByFormattedCode Statement = `SELECT inviter_user_id,token_id,formatted_code,nickname,created FROM invite WHERE formatted_code=?`
	InsertDyeUnlock           Statement = `INSERT INTO unlock_item(user_id,name,unlock_type,ack,created_at) VALUES(?,?,'spiritshop',1,?) ON CONFLICT(user_id,name) DO NOTHING`
	ReadDyeUnlock             Statement = `SELECT unlock_type,ack,created_at FROM unlock_item WHERE user_id=? AND name=?`
	SetItemDye                Statement = `INSERT INTO item_dye(user_id,unlock_name,upgrade_dye,created_at,updated_at) VALUES(?,?,?,?,?) ON CONFLICT(user_id,unlock_name,upgrade_dye) DO UPDATE SET updated_at=excluded.updated_at`
	EnsureColorDye            Statement = `INSERT INTO color_dye_currency(user_id,color_name,value) VALUES(?,?,0) ON CONFLICT(user_id,color_name) DO NOTHING`
	ReadColorDyes             Statement = `SELECT color_name,value FROM color_dye_currency WHERE user_id=?`
	GiveConsumable            Statement = `INSERT INTO user_consumable(user_id,consumable_id,quantity,cooldown_until) VALUES(?,?,?,0) ON CONFLICT(user_id,consumable_id) DO UPDATE SET quantity=COALESCE(user_consumable.quantity,0)+excluded.quantity,updated_at=CURRENT_TIMESTAMP`
	ReadShopPrice             Statement = `SELECT currency_type,unlock_cost FROM shop_item WHERE unlockid=?`
	ReadShopPurchaseItem      Statement = `SELECT name,currency_type,unlock_cost FROM shop_item WHERE unlockid=?`
	ReadOwnedUnlock           Statement = `SELECT ack FROM unlock_item WHERE user_id=? AND name=?`
	InsertOwnedUnlock         Statement = `INSERT INTO unlock_item(user_id,name,unlock_type,ack,created_at) VALUES(?,?,?,?,?) ON CONFLICT(user_id,name) DO NOTHING`
	UpsertLootbox             Statement = `INSERT INTO lootbox(user_id,lootbox_id,lootbox_type,quantity,acquired_at,opened) VALUES(?,?,?,1,?,0)`
	IncrementOpenLootbox      Statement = `UPDATE lootbox SET quantity=quantity+1 WHERE user_id=? AND lootbox_id=? AND opened=0`
	ReadOpenBoxQuantity       Statement = `SELECT quantity FROM lootbox WHERE user_id=? AND lootbox_id=? AND opened=0 ORDER BY id LIMIT 1`
	UseLootbox                Statement = `UPDATE lootbox SET quantity=quantity-1,opened=CASE WHEN quantity<=1 THEN 1 ELSE 0 END WHERE id=(SELECT id FROM lootbox WHERE user_id=? AND lootbox_id=? AND opened=0 AND quantity>0 ORDER BY id LIMIT 1)`
	ReadCommerceOrder         Statement = `SELECT user_id,product_id,status FROM commerce_order WHERE transaction_id=?`
	InsertCommerceOrder       Statement = `INSERT INTO commerce_order(user_id,product_id,transaction_id,status,created_at) VALUES(?,?,?,'simulated',?)`
	FulfillCommerceOrder      Statement = `UPDATE commerce_order SET status='fulfilled' WHERE user_id=? AND transaction_id=?`
	EnsureWaxForgeRate        Statement = `INSERT INTO forge_rate(user_id,source_type,current_tier,current_level,tier_completion,level_completion,level_tier_index,level_tier_count,reset_acknowledged,enabled) VALUES(?,'wax',1,1,0,0,0,5,1,1) ON CONFLICT(user_id,source_type) DO NOTHING`
	ReadWaxForgeRate          Statement = `SELECT current_tier,current_level,tier_completion,level_completion,level_tier_index,level_tier_count,reset_acknowledged,enabled FROM forge_rate WHERE user_id=? AND source_type='wax'`
)
