// 本业务包使用的兼容类型和转换。
package messaging

import "github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/core"

type Request = core.Request
type Object = core.Object
type object = Object

func text(m object, k string) string { return core.Text(m, k) }

func number(m object, k string, def int64) int64 { return core.Number(m, k, def) }

func stringValue(p *string) string { return core.StringValue(p) }

func fail(status int, message string) error { return core.Fail(status, message) }

func now() int64 { return core.Now() }

func uuid() (string, error) { return core.UUID() }

func legacyResponse(name string, target any) error { return core.LegacyResponse(name, target) }

func clientVersion(ua string) (int, int, int, bool) { return core.ClientVersion(ua) }

func currencyField(k string) bool { return core.CurrencyField(k) }

func floatValue(v any) (float64, bool) { return core.FloatValue(v) }

func intList(raw string) []int64 { return core.IntList(raw) }

func relationTime(epoch int64) string { return core.RelationTime(epoch) }

func nowMillis() int64 { return core.NowMillis() }

func scalarString(v any) string { return core.ScalarString(v) }
