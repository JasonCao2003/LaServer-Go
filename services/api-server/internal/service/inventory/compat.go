// 本业务包使用的兼容类型和转换。
package inventory

import "github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/core"

type Request = core.Request
type Object = core.Object
type object = Object
type Error = core.Error

func text(m object, k string) string { return core.Text(m, k) }

func number(m object, k string, def int64) int64 { return core.Number(m, k, def) }

func integer(p *int64) int64 { return core.Integer(p) }

func stringValue(p *string) string { return core.StringValue(p) }

func require(m object, keys ...string) error { return core.Require(m, keys...) }

func fail(status int, message string) error { return core.Fail(status, message) }

func now() int64 { return core.Now() }

func defInt(v any) int64 { return core.DefInt(v) }

func legacyResponse(name string, target any) error { return core.LegacyResponse(name, target) }

func clientVersion(ua string) (int, int, int, bool) { return core.ClientVersion(ua) }

func defSeconds(v any) int64 { return core.DefSeconds(v) }

func isUGCMagic(d Object) bool { return core.IsUGCMagic(d) }

func legacyRelationshipDefs(ua string, defs []map[string]any) []map[string]any {
	return core.LegacyRelationshipDefs(ua, defs)
}
