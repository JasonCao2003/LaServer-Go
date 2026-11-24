// 房间准入令牌签发与校验

package account

import (
	"crypto/hmac"
	"crypto/subtle"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"net"
	"strings"
)

// 校验房间准入
func (a *Service) VerifyRoomAdmission(r Request, body Object) (Object, error) {
	if a.Cfg.RoomAdmissionKey == "" || a.Cfg.RoomAdmissionBearer == "" {
		return nil, fail(404, "Room admission is disabled")
	}
	remote := net.ParseIP(r.RemoteIP)
	if !r.Secure && (remote == nil || !remote.IsLoopback()) {
		return nil, fail(403, "Room admission requires TLS or loopback")
	}
	supplied, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || subtle.ConstantTimeCompare([]byte(supplied), []byte(a.Cfg.RoomAdmissionBearer)) != 1 {
		return nil, fail(401, "Invalid service credential")
	}

	account, room, signature := text(body, "account_id"), text(body, "room_uri"), text(body, "signature")
	token, err := hex.DecodeString(signature)
	if account == "" || room == "" || err != nil || len(token) != 32 {
		return nil, fail(400, "Invalid room admission request")
	}
	stamp := now()
	expires := int64(binary.BigEndian.Uint64(token[:8]))
	if room != a.RoomAdmissionURI() || expires <= stamp || expires-stamp > admissionTTLSeconds {
		return nil, fail(401, "Invalid or expired room admission")
	}
	u, err := a.FindUser(r.Context, "id", account)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fail(401, "Invalid or expired room admission")
	}
	if err != nil {
		return nil, err
	}
	key, err := hex.DecodeString(a.Cfg.RoomAdmissionKey)
	if err != nil || len(key) != 32 {
		return nil, errors.New("invalid room admission key")
	}
	session := stringValue(u.Session)
	expected := roomAdmissionToken(key, u.ID, session, room, expires)
	if session == "" || !hmac.Equal(token, expected[:]) {
		return nil, fail(401, "Invalid or expired room admission")
	}
	if err := a.CheckAllowed(r.Context, &u, 403); err != nil {
		return nil, err
	}
	return Object{"account_id": u.ID, "expires_at": expires}, nil
}
