// 共享工具适配

package content

import (
	"context"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/core"
)

type Request = core.Request
type Object = core.Object
type object = Object
type Error = core.Error
type currencyReader = core.CurrencyReader

var currencyFields = core.CurrencyFields
var createResources = core.CreateResources
var loginResources = core.LoginResources
var authCreateResources = core.AuthCreateResources
var authLoginResources = core.AuthLoginResources
var defaultUserdata = core.DefaultUserdata
var checkpointOrders = core.CheckpointOrders
var variablesJSON = core.VariablesJSON

const admissionTTLSeconds = core.AdmissionTTLSeconds

func text(m object, k string) string                { return core.Text(m, k) }
func number(m object, k string, def int64) int64    { return core.Number(m, k, def) }
func integer(p *int64) int64                        { return core.Integer(p) }
func stringValue(p *string) string                  { return core.StringValue(p) }
func require(m object, keys ...string) error        { return core.Require(m, keys...) }
func fail(status int, message string) error         { return core.Fail(status, message) }
func now() int64                                    { return core.Now() }
func remoteIP(r Request) string                     { return core.RemoteIP(r) }
func token(n int) (string, error)                   { return core.Token(n) }
func uuid() (string, error)                         { return core.UUID() }
func legacyVersion(ua string) bool                  { return core.LegacyVersion(ua) }
func defInt(v any) int64                            { return core.DefInt(v) }
func legacyResponse(name string, target any) error  { return core.LegacyResponse(name, target) }
func clientVersion(ua string) (int, int, int, bool) { return core.ClientVersion(ua) }
func currencyField(k string) bool                   { return core.CurrencyField(k) }
func roomAdmissionToken(key []byte, account, session, room string, expires int64) [32]byte {
	return core.RoomAdmissionToken(key, account, session, room, expires)
}
func currencySnapshot(ctx context.Context, reader currencyReader, uid string) (Object, error) {
	return core.CurrencySnapshotReader(ctx, reader, uid)
}
func boolValue(v any) bool                      { return core.BoolValue(v) }
func floatValue(v any) (float64, bool)          { return core.FloatValue(v) }
func intList(raw string) []int64                { return core.IntList(raw) }
func relationTime(epoch int64) string           { return core.RelationTime(epoch) }
func defSeconds(v any) int64                    { return core.DefSeconds(v) }
func asObject(value any) Object                 { return core.AsObject(value) }
func boolSetting(value any, fallback bool) bool { return core.BoolSetting(value, fallback) }
func validConfiguredArray(value any, requiredField string) bool {
	return core.ValidConfiguredArray(value, requiredField)
}
func nowMillis() int64                          { return core.NowMillis() }
func scalarString(v any) string                 { return core.ScalarString(v) }
func firstText(m Object, keys ...string) string { return core.FirstText(m, keys...) }
func isUGCMagic(d Object) bool                  { return core.IsUGCMagic(d) }
func relationshipEventAllowed(event string, version [3]int, known bool) bool {
	return core.RelationshipEventAllowed(event, version, known)
}
func legacyRelationshipDefs(ua string, defs []map[string]any) []map[string]any {
	return core.LegacyRelationshipDefs(ua, defs)
}
func (a *Service) genericShopDef(m Object) Object { return a.Base.GenericShopDef(m) }
func (a *Service) spiritNode(name string) Object  { return a.Base.SpiritNode(name) }
