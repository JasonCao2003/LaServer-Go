// 本业务包使用的兼容类型和转换。
package social

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

func uuid() (string, error) { return core.UUID() }

func boolValue(v any) bool { return core.BoolValue(v) }

func floatValue(v any) (float64, bool) { return core.FloatValue(v) }

func nowMillis() int64 { return core.NowMillis() }

func scalarString(v any) string { return core.ScalarString(v) }

func firstText(m Object, keys ...string) string { return core.FirstText(m, keys...) }
