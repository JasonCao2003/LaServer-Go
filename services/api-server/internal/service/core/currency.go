// 货币字段与快照

package core

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

var CurrencyFields = []string{"candles", "heart", "heart_wax", "prestige", "prestige_vip", "prestige_wax", "season_candle", "season_wax", "season_wax_14", "season_heart", "season_pass_token", "vip", "wax"}

type CurrencyReader interface {
	QueryRowContext(context.Context, repository.Statement, ...any) *sql.Row
}

// 判断是否为货币字段
func CurrencyField(k string) bool {
	for _, f := range CurrencyFields {
		if f == k {
			return true
		}
	}
	return false
}

// 读取货币快照
func (a *Base) CurrencySnapshot(uid string) Object {
	out := Object{}
	row := a.Store.QueryRowContext(context.Background(), repository.ReadCurrency, uid)
	var values [13]sql.NullInt64
	dest := make([]any, len(values))
	for i := range values {
		dest[i] = &values[i]
	}
	if err := row.Scan(dest...); err != nil {
		for _, field := range CurrencyFields {
			out[field] = int64(0)
		}
	} else {
		for i, field := range CurrencyFields {
			out[field] = values[i].Int64
		}
	}
	if a.Cfg.CurrentSeason > 0 {
		suffix := strconv.Itoa(a.Cfg.CurrentSeason)
		out["season_candle_"+suffix] = out["season_candle"]
		out["season_heart_"+suffix] = out["season_heart"]
	}
	return out
}

// 读取货币快照（可指定读取器）
func CurrencySnapshotReader(ctx context.Context, reader CurrencyReader, uid string) (Object, error) {
	values := make([]sql.NullInt64, len(CurrencyFields))
	dest := make([]any, len(values))
	for i := range values {
		dest[i] = &values[i]
	}
	if err := reader.QueryRowContext(ctx, repository.ReadCurrency, uid).Scan(dest...); err != nil {
		return nil, err
	}
	out := Object{}
	for i, key := range CurrencyFields {
		out[key] = values[i].Int64
	}
	return out, nil
}
