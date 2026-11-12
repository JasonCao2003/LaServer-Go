package repository

const (
	ReadConsumables      Statement = `SELECT consumable_id,cooldown_until,quantity FROM user_consumable WHERE user_id=? ORDER BY id`
	ConsumeItem          Statement = `UPDATE user_consumable SET quantity=quantity-?,cooldown_until=? WHERE user_id=? AND consumable_id=? AND quantity>=?`
	ConsumeUGCItem       Statement = `UPDATE user_consumable SET quantity=MAX(MAX(quantity,99)-?,0),cooldown_until=? WHERE user_id=? AND consumable_id=?`
	ReadBuffs            Statement = `SELECT buff_id,active_until,cooldown_until,giver_user_id FROM user_buff WHERE user_id=? ORDER BY id`
	UpsertBuff           Statement = `INSERT INTO user_buff(user_id,buff_id,active_until,cooldown_until,giver_user_id) VALUES(?,?,?,?,?) ON CONFLICT(user_id,buff_id) DO UPDATE SET active_until=excluded.active_until,cooldown_until=excluded.cooldown_until,giver_user_id=excluded.giver_user_id`
	DeleteBuff           Statement = `DELETE FROM user_buff WHERE user_id=? AND buff_id=?`
	ReadWingBuffs        Statement = `SELECT name,collected,deposited,last_conversion,deposit_id FROM user_wing_buff WHERE user_id=? ORDER BY id`
	CountWingBuffs       Statement = `SELECT count(*) FROM user_wing_buff WHERE user_id=?`
	CollectWingBuff      Statement = `INSERT INTO user_wing_buff(user_id,name,collected) VALUES(?,?,1) ON CONFLICT(user_id,name) DO UPDATE SET collected=1`
	DropWingBuff         Statement = `UPDATE user_wing_buff SET collected=0 WHERE user_id=? AND name=?`
	DepositWingBuff      Statement = `INSERT INTO user_wing_buff(user_id,name,collected,deposited,deposit_id) VALUES(?,?,0,1,?) ON CONFLICT(user_id,name) DO UPDATE SET collected=0,deposited=1,deposit_id=excluded.deposit_id`
	SeedWingBuff         Statement = `INSERT INTO user_wing_buff(user_id,name,collected,deposited,deposit_id) VALUES(?,?,1,0,?) ON CONFLICT(user_id,name) DO NOTHING`
	ConvertWingBuffs     Statement = `UPDATE user_wing_buff SET deposited=1,last_conversion=? WHERE user_id=? AND deposited=0`
	GrantPrestige        Statement = `UPDATE user_currency SET prestige=COALESCE(prestige,0)+? WHERE user_id=?`
	EnsureWingCurrency   Statement = `INSERT INTO user_currency(user_id) VALUES(?) ON CONFLICT(user_id) DO NOTHING`
	ReadPrestige         Statement = `SELECT COALESCE(prestige,0) FROM user_currency WHERE user_id=?`
	ReadStatusUnlocks    Statement = `SELECT name,unlock_type,ack,unlocked_at FROM status_unlock WHERE user_id=? ORDER BY id`
	ReadStatusUnlock     Statement = `SELECT unlock_type,ack,unlocked_at FROM status_unlock WHERE user_id=? AND name=?`
	AddStatusUnlock      Statement = `INSERT INTO status_unlock(user_id,name,unlock_type,ack,unlocked_at) VALUES(?,?,'level',?,?) ON CONFLICT(user_id,name) DO NOTHING`
	DeleteStatusUnlock   Statement = `DELETE FROM status_unlock WHERE user_id=? AND name=?`
	ReadStatusTombstones Statement = `SELECT name FROM status_unlock_deleted WHERE user_id=?`
	AddStatusTombstone   Statement = `INSERT INTO status_unlock_deleted(user_id,name,deleted_at) VALUES(?,?,?) ON CONFLICT(user_id,name) DO UPDATE SET deleted_at=excluded.deleted_at`
	ClearStatusTombstone Statement = `DELETE FROM status_unlock_deleted WHERE user_id=? AND name=?`
	DeleteItemDye        Statement = `DELETE FROM item_dye WHERE user_id=? AND unlock_name=?`
	SeedCollectible      Statement = `INSERT INTO collectible(user_id,id,used,candle_space,carrying,level,permanent_level) VALUES(?,?,?,?,?,?,?) ON CONFLICT(user_id,id) DO NOTHING`
	CollectCollectible   Statement = `INSERT INTO collectible(user_id,id,carrying,level,permanent_level) VALUES(?,?,?,1,1) ON CONFLICT(user_id,id) DO UPDATE SET carrying=excluded.carrying,level=COALESCE(collectible.level,0)+1,permanent_level=MAX(COALESCE(collectible.permanent_level,0),1)`
	CountCollectibles    Statement = `SELECT count(*) FROM collectible WHERE user_id=?`
	ReadOutfitHeight     Statement = `SELECT COALESCE(height,1.0) FROM user_outfit WHERE user_id=?`
	SetOutfitHeight      Statement = `UPDATE user_outfit SET height=? WHERE user_id=?`
)
