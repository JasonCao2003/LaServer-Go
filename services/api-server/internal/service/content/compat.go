// 本业务包使用的兼容类型和转换。
package content

import "github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/core"

type Request = core.Request
type Object = core.Object
type object = Object
type Error = core.Error

func text(m object, k string) string { return core.Text(m, k) }

func number(m object, k string, def int64) int64 { return core.Number(m, k, def) }

func integer(p *int64) int64 { return core.Integer(p) }

func stringValue(p *string) string { return core.StringValue(p) }

func fail(status int, message string) error { return core.Fail(status, message) }

func now() int64 { return core.Now() }

func token(n int) (string, error) { return core.Token(n) }

func uuid() (string, error) { return core.UUID() }

func legacyResponse(name string, target any) error { return core.LegacyResponse(name, target) }

func clientVersion(ua string) (int, int, int, bool) { return core.ClientVersion(ua) }

func asObject(value any) Object { return core.AsObject(value) }

func boolSetting(value any, fallback bool) bool { return core.BoolSetting(value, fallback) }

func validConfiguredArray(value any, requiredField string) bool {
	return core.ValidConfiguredArray(value, requiredField)
}
