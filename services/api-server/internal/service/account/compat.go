// 本业务包使用的兼容类型和转换。
package account

import "github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/core"

type Request = core.Request
type Object = core.Object
type object = Object
type Error = core.Error
var currencyFields = core.CurrencyFields
var createResources = core.CreateResources
var loginResources = core.LoginResources
var authCreateResources = core.AuthCreateResources
var authLoginResources = core.AuthLoginResources
var defaultUserdata = core.DefaultUserdata
var checkpointOrders = core.CheckpointOrders
var variablesJSON = core.VariablesJSON

const admissionTTLSeconds = core.AdmissionTTLSeconds

func text(m object, k string) string { return core.Text(m, k) }

func number(m object, k string, def int64) int64 { return core.Number(m, k, def) }

func integer(p *int64) int64 { return core.Integer(p) }

func stringValue(p *string) string { return core.StringValue(p) }

func require(m object, keys ...string) error { return core.Require(m, keys...) }

func fail(status int, message string) error { return core.Fail(status, message) }

func now() int64 { return core.Now() }

func remoteIP(r Request) string { return core.RemoteIP(r) }

func token(n int) (string, error) { return core.Token(n) }

func uuid() (string, error) { return core.UUID() }

func currencyField(k string) bool { return core.CurrencyField(k) }

func roomAdmissionToken(key []byte, account, session, room string, expires int64) [32]byte {
	return core.RoomAdmissionToken(key, account, session, room, expires)
}

func validConfiguredArray(value any, requiredField string) bool {
	return core.ValidConfiguredArray(value, requiredField)
}
