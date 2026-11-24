// 本业务包使用的兼容类型和转换。
package quests

import (
	"context"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/core"
)

type Request = core.Request
type Object = core.Object
type object = Object
type currencyReader = core.CurrencyReader

func text(m object, k string) string { return core.Text(m, k) }

func number(m object, k string, def int64) int64 { return core.Number(m, k, def) }

func fail(status int, message string) error { return core.Fail(status, message) }

func now() int64 { return core.Now() }

func uuid() (string, error) { return core.UUID() }

func defInt(v any) int64 { return core.DefInt(v) }

func currencySnapshot(ctx context.Context, reader currencyReader, uid string) (Object, error) {
	return core.CurrencySnapshotReader(ctx, reader, uid)
}
