// 房间准入令牌

package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
)

const AdmissionTTLSeconds = 120

// 返回房间准入服务地址
func (a *Base) RoomAdmissionURI() string {
	return fmt.Sprintf("%s:%d", a.Cfg.RoomHost, a.Cfg.RoomPort)
}

// 签发房间准入令牌
func (a *Base) IssueRoomAdmission(account, session, room string, stamp int64) (string, error) {
	key, err := hex.DecodeString(a.Cfg.RoomAdmissionKey)
	if err != nil || len(key) != 32 {
		return "", errors.New("invalid room admission key")
	}
	token := RoomAdmissionToken(key, account, session, room, stamp+AdmissionTTLSeconds)
	return hex.EncodeToString(token[:]), nil
}

// 计算房间准入令牌
func RoomAdmissionToken(key []byte, account, session, room string, expires int64) [32]byte {
	var token [32]byte
	binary.BigEndian.PutUint64(token[:8], uint64(expires))

	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte("skyla-room-admission-v1"))
	_, _ = mac.Write(token[:8])
	for _, field := range []string{account, session, room} {
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(len(field)))
		_, _ = mac.Write(length[:])
		_, _ = mac.Write([]byte(field))
	}
	copy(token[8:], mac.Sum(nil))
	return token
}
