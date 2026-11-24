// 光翼兑换的事务内持久化操作。
package repository

import "context"

// 确保货币行存在，并返回决定是否初始化光翼所需的当前行数。
func (tx *Tx) PrepareWingConversion(ctx context.Context, uid string) (int64, error) {
	if _, err := tx.ExecContext(ctx, EnsureWingCurrency, uid); err != nil {
		return 0, err
	}
	var count int64
	err := tx.QueryRowContext(ctx, CountWingBuffs, uid).Scan(&count)
	return count, err
}

// 按业务层给定的名称和存入 ID 初始化光翼。
func (tx *Tx) SeedWingForConversion(ctx context.Context, uid, name string, id int64) error {
	_, err := tx.ExecContext(ctx, SeedWingBuff, uid, name, id)
	return err
}

// 返回本次实际发生状态变化的行数，用于决定声望奖励。
func (tx *Tx) MarkWingsConverted(ctx context.Context, uid string, stamp int64) (int64, error) {
	res, err := tx.ExecContext(ctx, ConvertWingBuffs, stamp, uid)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// 在当前事务内增加兑换所得声望。
func (tx *Tx) CreditWingPrestige(ctx context.Context, uid string, amount int64) error {
	_, err := tx.ExecContext(ctx, GrantPrestige, amount, uid)
	return err
}

// WingBuffRow 保存 SQLite 中一条光翼记录的读取结果。
type WingBuffRow struct {
	Name                      string
	Collected, Deposited      bool
	LastConversion, DepositID int64
}

// 按原有记录顺序读取玩家的光翼。
func (s *Store) WingBuffs(ctx context.Context, uid string) ([]WingBuffRow, error) {
	rows, err := s.QueryContext(ctx, ReadWingBuffs, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []WingBuffRow
	for rows.Next() {
		var row WingBuffRow
		if err := rows.Scan(&row.Name, &row.Collected, &row.Deposited, &row.LastConversion, &row.DepositID); err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	return results, rows.Err()
}
