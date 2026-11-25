// 管理面板的玩家道具、装扮和内容查询。
package repository

import (
	"context"
	"fmt"
)

type PanelRow struct{ ID, Name, Value, Detail string }

func (s *Store) PanelRows(ctx context.Context, kind, keyword string) ([]PanelRow, error) {
	var query Statement
	switch kind {
	case "magic":
		query = `SELECT CAST(consumable_id AS TEXT),'魔法',CAST(quantity AS TEXT),CAST(cooldown_until AS TEXT) FROM user_consumable WHERE user_id=? ORDER BY consumable_id`
	case "ugc":
		query = `SELECT id,kind,message,pool FROM (SELECT geonote_id id,'纸船' kind,message,pool_name pool,created_at FROM geonote UNION ALL SELECT social_feed_id,pool_type,message,pool_name,created_at FROM social_feed_post) WHERE ?1='' OR instr(id,?1)>0 OR instr(message,?1)>0 OR instr(pool,?1)>0 ORDER BY created_at DESC LIMIT 100`
	case "other":
		query = `SELECT user_id,'斗篷',CAST(COALESCE(wing_id,0) AS TEXT),'wing' FROM user_outfit WHERE user_id=?1 UNION ALL SELECT user_id,'发型',CAST(COALESCE(hair_id,0) AS TEXT),'hair' FROM user_outfit WHERE user_id=?1 UNION ALL SELECT user_id,'帽子',CAST(COALESCE(hat_id,0) AS TEXT),'hat' FROM user_outfit WHERE user_id=?1 UNION ALL SELECT user_id,'背饰',CAST(COALESCE(prop_id,0) AS TEXT),'prop' FROM user_outfit WHERE user_id=?1 UNION ALL SELECT user_id,'颈饰',CAST(COALESCE(neck_id,0) AS TEXT),'neck' FROM user_outfit WHERE user_id=?1 UNION ALL SELECT user_id,'头饰',CAST(COALESCE(horn_id,0) AS TEXT),'horn' FROM user_outfit WHERE user_id=?1 UNION ALL SELECT user_id,'鞋子',CAST(COALESCE(feet_id,0) AS TEXT),'feet' FROM user_outfit WHERE user_id=?1 UNION ALL SELECT user_id,'面部',CAST(COALESCE(face_id,0) AS TEXT),'face' FROM user_outfit WHERE user_id=?1 UNION ALL SELECT user_id,'服装',CAST(COALESCE(body_id,0) AS TEXT),'body' FROM user_outfit WHERE user_id=?1 UNION ALL SELECT user_id,'面具',CAST(COALESCE(mask_id,0) AS TEXT),'mask' FROM user_outfit WHERE user_id=?1 UNION ALL SELECT user_id,'手臂',CAST(COALESCE(arms_id,0) AS TEXT),'arms' FROM user_outfit WHERE user_id=?1`
	default:
		return nil, fmt.Errorf("unknown section")
	}
	rows, err := s.QueryContext(ctx, query, keyword)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []PanelRow{}
	for rows.Next() {
		var row PanelRow
		if err := rows.Scan(&row.ID, &row.Name, &row.Value, &row.Detail); err != nil {
			return nil, err
		}
		items = append(items, row)
	}
	return items, rows.Err()
}

func (s *Store) PanelEdit(ctx context.Context, kind, user, id, value string, num int64) (bool, error) {
	var query Statement
	var args []any
	switch kind {
	case "magic":
		query = `INSERT INTO user_consumable(user_id,consumable_id,quantity) SELECT id,?,? FROM user WHERE id=? ON CONFLICT(user_id,consumable_id) DO UPDATE SET quantity=excluded.quantity,updated_at=CURRENT_TIMESTAMP`
		args = []any{id, num, user}
	case "magic_delete":
		query = `DELETE FROM user_consumable WHERE user_id=? AND consumable_id=?`
		args = []any{user, id}
	case "geonote":
		query = `UPDATE geonote SET message=?,edited_at=unixepoch() WHERE geonote_id=?`
		args = []any{value, id}
	case "feed":
		query = `UPDATE social_feed_post SET message=? WHERE social_feed_id=?`
		args = []any{value, id}
	case "wing", "hair", "hat", "prop", "neck", "horn", "feet", "face", "body", "mask", "arms":
		query = Statement("UPDATE user_outfit SET " + kind + "_id=?,updated_at=CURRENT_TIMESTAMP WHERE user_id=?")
		args = []any{num, user}
	default:
		return false, fmt.Errorf("unknown operation")
	}
	result, err := s.ExecContext(ctx, query, args...)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n > 0, err
}
