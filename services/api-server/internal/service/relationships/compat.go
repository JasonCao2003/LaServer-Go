// 本业务包使用的兼容类型和转换。
package relationships

import "github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/core"

type Request = core.Request
type Object = core.Object
type object = Object
type Error = core.Error

func text(m object, k string) string { return core.Text(m, k) }

func number(m object, k string, def int64) int64 { return core.Number(m, k, def) }

func stringValue(p *string) string { return core.StringValue(p) }

func fail(status int, message string) error { return core.Fail(status, message) }

func now() int64 { return core.Now() }

func token(n int) (string, error) { return core.Token(n) }

func legacyResponse(name string, target any) error { return core.LegacyResponse(name, target) }

func clientVersion(ua string) (int, int, int, bool) { return core.ClientVersion(ua) }

func boolValue(v any) bool { return core.BoolValue(v) }

func floatValue(v any) (float64, bool) { return core.FloatValue(v) }

func intList(raw string) []int64 { return core.IntList(raw) }

func relationTime(epoch int64) string { return core.RelationTime(epoch) }
