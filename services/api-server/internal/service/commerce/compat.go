// 本业务包使用的兼容类型和转换。
package commerce

import (
	"context"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/core"
)

type Request = core.Request
type Object = core.Object
type object = Object
type Error = core.Error
type currencyReader = core.CurrencyReader

func text(m object, k string) string { return core.Text(m, k) }

func number(m object, k string, def int64) int64 { return core.Number(m, k, def) }

func stringValue(p *string) string { return core.StringValue(p) }

func fail(status int, message string) error { return core.Fail(status, message) }

func now() int64 { return core.Now() }

func token(n int) (string, error) { return core.Token(n) }

func defInt(v any) int64 { return core.DefInt(v) }

func legacyResponse(name string, target any) error { return core.LegacyResponse(name, target) }

func clientVersion(ua string) (int, int, int, bool) { return core.ClientVersion(ua) }

func currencyField(k string) bool { return core.CurrencyField(k) }

func currencySnapshot(ctx context.Context, reader currencyReader, uid string) (Object, error) {
	return core.CurrencySnapshotReader(ctx, reader, uid)
}

func defSeconds(v any) int64 { return core.DefSeconds(v) }

func asObject(value any) Object { return core.AsObject(value) }

func boolSetting(value any, fallback bool) bool { return core.BoolSetting(value, fallback) }

func validConfiguredArray(value any, requiredField string) bool {
	return core.ValidConfiguredArray(value, requiredField)
}

func firstText(m Object, keys ...string) string { return core.FirstText(m, keys...) }

func (a *Service) genericShopDef(m Object) Object { return a.Base.GenericShopDef(m) }

func (a *Service) spiritNode(name string) Object { return a.Base.SpiritNode(name) }
