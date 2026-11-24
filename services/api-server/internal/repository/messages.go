// 消息数据访问

package repository

import (
	"context"
	"database/sql"
	"encoding/json"
)

type FreeGift struct {
	ID                   int64
	SenderID, ReceiverID string
	GiftType, GiftID     string
	SentAt               int64
	Received             bool
	ReceivedAt           sql.NullInt64
	ExpiresAt            sql.NullInt64
}

// 读取用户 ID 列表
func (s *Store) UserIDs(ctx context.Context) ([]string, error) {
	rows, err := s.QueryContext(ctx, `SELECT id FROM user`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

const giftColumns = `id,sender_id,receiver_id,gift_type,gift_id,sent_at,COALESCE(received,0),received_at,expires_at`

// 扫描礼物
func scanGift(scanner rowScanner) (FreeGift, error) {
	var g FreeGift
	err := scanner.Scan(&g.ID, &g.SenderID, &g.ReceiverID, &g.GiftType, &g.GiftID, &g.SentAt, &g.Received, &g.ReceivedAt, &g.ExpiresAt)
	return g, err
}

// 读取已发送礼物
func (s *Store) SentGifts(ctx context.Context, userID string, stamp int64) ([]FreeGift, error) {
	rows, err := s.QueryContext(ctx, `SELECT `+giftColumns+` FROM free_gift WHERE sender_id=? AND (expires_at IS NULL OR expires_at>?) ORDER BY sent_at DESC,id DESC`, userID, stamp)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FreeGift{}
	for rows.Next() {
		g, err := scanGift(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// 读取已接收礼物
func (s *Store) ReceivedGifts(ctx context.Context, userID string, stamp int64) ([]FreeGift, error) {
	rows, err := s.QueryContext(ctx, `SELECT `+giftColumns+` FROM free_gift WHERE receiver_id=? AND COALESCE(received,0)=0 AND (expires_at IS NULL OR expires_at>?) ORDER BY sent_at ASC,id ASC`, userID, stamp)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FreeGift{}
	for rows.Next() {
		g, err := scanGift(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// 统计待领取礼物数量
func (s *Store) PendingGiftCount(ctx context.Context, userID string, stamp int64) (int64, error) {
	var count int64
	err := s.QueryRowContext(ctx, `SELECT count(*) FROM free_gift WHERE receiver_id=? AND COALESCE(received,0)=0 AND (expires_at IS NULL OR expires_at>?)`, userID, stamp).Scan(&count)
	return count, err
}

// 确保货币
func (t *Tx) EnsureCurrency(ctx context.Context, userID string) error {
	_, err := t.ExecContext(ctx, `INSERT INTO user_currency(user_id,wax) VALUES(?,3) ON CONFLICT(user_id) DO NOTHING`, userID)
	return err
}

// 处理货币余额
func (t *Tx) CurrencyBalance(ctx context.Context, userID, field string) (int64, error) {
	if !spendableCurrency[field] {
		return 0, sql.ErrNoRows
	}
	var balance int64
	err := t.QueryRowContext(ctx, Statement(`SELECT COALESCE(`+field+`,0) FROM user_currency WHERE user_id=?`), userID).Scan(&balance)
	return balance, err
}

// 限制扣减
func (t *Tx) ClampSpend(ctx context.Context, userID, field string, cost int64) error {
	if !spendableCurrency[field] || cost < 0 {
		return sql.ErrNoRows
	}
	_, err := t.ExecContext(ctx, Statement(`UPDATE user_currency SET `+field+`=MAX(0,COALESCE(`+field+`,0)-?) WHERE user_id=?`), cost, userID)
	return err
}

// 新增货币
func (t *Tx) AddCurrency(ctx context.Context, userID, field string, amount int64) error {
	if !spendableCurrency[field] || amount < 0 {
		return sql.ErrNoRows
	}
	_, err := t.ExecContext(ctx, Statement(`UPDATE user_currency SET `+field+`=COALESCE(`+field+`,0)+? WHERE user_id=?`), amount, userID)
	return err
}

// 设置货币
func (t *Tx) SetCurrency(ctx context.Context, userID, field string, value int64) error {
	if !spendableCurrency[field] || value < 0 {
		return sql.ErrNoRows
	}
	_, err := t.ExecContext(ctx, Statement(`UPDATE user_currency SET `+field+`=? WHERE user_id=?`), value, userID)
	return err
}

// 插入礼物
func (t *Tx) InsertGift(ctx context.Context, g FreeGift) error {
	_, err := t.ExecContext(ctx, `INSERT INTO free_gift(sender_id,receiver_id,gift_type,gift_id,sent_at,received,expires_at) VALUES(?,?,?,?,?,0,?)`, g.SenderID, g.ReceiverID, g.GiftType, g.GiftID, g.SentAt, g.ExpiresAt.Int64)
	return err
}

// 按 ID 获取礼物
func (t *Tx) GiftByID(ctx context.Context, receiverID, giftID string) (FreeGift, bool, error) {
	g, err := scanGift(t.QueryRowContext(ctx, `SELECT `+giftColumns+` FROM free_gift WHERE gift_id=? AND receiver_id=? AND (expires_at IS NULL OR expires_at>?) ORDER BY id LIMIT 1`, giftID, receiverID, nowEpoch()))
	if err == sql.ErrNoRows {
		return g, false, nil
	}
	return g, err == nil, err
}

// 获取来自指定好友的最新未接收礼物
func (t *Tx) LatestUnreceivedGiftFrom(ctx context.Context, receiverID, senderID string) (FreeGift, bool, error) {
	g, err := scanGift(t.QueryRowContext(ctx, `SELECT `+giftColumns+` FROM free_gift WHERE receiver_id=? AND sender_id=? AND COALESCE(received,0)=0 AND (expires_at IS NULL OR expires_at>?) ORDER BY sent_at DESC,id DESC LIMIT 1`, receiverID, senderID, nowEpoch()))
	if err == sql.ErrNoRows {
		return g, false, nil
	}
	return g, err == nil, err
}

// 获取来自指定好友的未接收礼物
func (t *Tx) UnreceivedGiftsFrom(ctx context.Context, receiverID, senderID string) ([]FreeGift, error) {
	rows, err := t.QueryContext(ctx, `SELECT `+giftColumns+` FROM free_gift WHERE receiver_id=? AND sender_id=? AND COALESCE(received,0)=0 AND (expires_at IS NULL OR expires_at>?) ORDER BY sent_at ASC,id ASC`, receiverID, senderID, nowEpoch())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FreeGift{}
	for rows.Next() {
		g, err := scanGift(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// 标记礼物已接收
func (t *Tx) MarkGiftReceived(ctx context.Context, id, stamp int64) error {
	_, err := t.ExecContext(ctx, `UPDATE free_gift SET received=1,received_at=? WHERE id=?`, stamp, id)
	return err
}

// 标记消息已接收
func (s *Store) MarkMessageReceived(ctx context.Context, userID, messageID string, stamp int64) error {
	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO message(id,user_id,received_at,created_at,updated_at) VALUES(?,?,?,?,?) ON CONFLICT(id,user_id) DO NOTHING`, messageID, userID, stamp, stamp, stamp); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE message SET received_at=?,updated_at=? WHERE id=? AND user_id=? AND received_at IS NULL`, stamp, stamp, messageID, userID); err != nil {
		return err
	}
	return tx.Commit()
}

// 读取关系事务数据
func (t *Tx) RelationTx(ctx context.Context, userID, friendID string) (RelationEntry, bool, error) {
	e, err := scanRelation(t.QueryRowContext(ctx, Statement(relationSelect+` WHERE fs.user_id=? AND fs.friend_id=?`), userID, friendID))
	if err == sql.ErrNoRows {
		return RelationEntry{}, false, nil
	}
	if err != nil {
		return RelationEntry{}, false, err
	}
	return e, true, nil
}

// 确保关系配对事务
func (t *Tx) EnsureRelationPairTx(ctx context.Context, userID, friendID, abilities string, stamp int64) error {
	if abilities == "" {
		abilities = defaultFriendAbilities
	}
	for _, pair := range [][2]string{{userID, friendID}, {friendID, userID}} {
		if _, err := t.ExecContext(ctx, `INSERT INTO friendship(user_id,friend_id,status) VALUES(?,?,'active') ON CONFLICT(user_id,friend_id) DO UPDATE SET status='active',updated_at=CURRENT_TIMESTAMP WHERE friendship.status='pending'`, pair[0], pair[1]); err != nil {
			return err
		}
		if _, err := t.ExecContext(ctx, `INSERT INTO friend_status(user_id,friend_id,given,recvd,nickname,when_created,level,abilities,hints,outfit,player_badge_type,local_soft_deleted,attitude) VALUES(?,?,0,0,'',?,1,?,'[]','{}',0,0,'0') ON CONFLICT(user_id,friend_id) DO NOTHING`, pair[0], pair[1], stamp, abilities); err != nil {
			return err
		}
	}
	return nil
}

// 保存关系事务
func (t *Tx) SaveRelationTx(ctx context.Context, userID, friendID, abilities, hints string, given, received int64, stamp int64) error {
	_, err := t.ExecContext(ctx, `UPDATE friend_status SET abilities=?,hints=?,given=?,recvd=?,local_soft_deleted=0,when_created=CASE WHEN COALESCE(when_created,0)=0 THEN ? ELSE when_created END,updated_at=CURRENT_TIMESTAMP WHERE user_id=? AND friend_id=?`, abilities, hints, given, received, stamp, userID, friendID)
	return err
}

// 统计已添加好友
func (s *Store) CountAddedFriends(ctx context.Context, userID string) (int64, error) {
	records, err := s.Statues(ctx, userID)
	if err != nil {
		return 0, err
	}
	var count int64
	for _, f := range records {
		added := f.Given > 0 || f.Received > 0
		if !added {
			var abilities []json.RawMessage
			if json.Unmarshal([]byte(f.Abilities), &abilities) == nil {
				for _, raw := range abilities {
					var n int64
					if json.Unmarshal(raw, &n) == nil && n == 1 {
						added = true
					}
				}
			}
		}
		if added || f.Blocked || f.BlockedByFriend {
			count++
		}
	}
	return count, nil
}
