package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
)

const admissionTTLSeconds = 120

func (a *Service) roomAdmissionURI() string {
	return fmt.Sprintf("%s:%d", a.cfg.RoomHost, a.cfg.RoomPort)
}

func (a *Service) issueRoomAdmission(account, session, room string, stamp int64) (string, error) {
	key, err := hex.DecodeString(a.cfg.RoomAdmissionKey)
	if err != nil || len(key) != 32 {
		return "", errors.New("invalid room admission key")
	}
	token := roomAdmissionToken(key, account, session, room, stamp+admissionTTLSeconds)
	return hex.EncodeToString(token[:]), nil
}

func roomAdmissionToken(key []byte, account, session, room string, expires int64) [32]byte {
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

func (a *Service) VerifyRoomAdmission(r Request, body Object) (Object, error) {
	if a.cfg.RoomAdmissionKey == "" || a.cfg.RoomAdmissionBearer == "" {
		return nil, fail(404, "Room admission is disabled")
	}
	remote := net.ParseIP(r.RemoteIP)
	if !r.Secure && (remote == nil || !remote.IsLoopback()) {
		return nil, fail(403, "Room admission requires TLS or loopback")
	}
	supplied, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || subtle.ConstantTimeCompare([]byte(supplied), []byte(a.cfg.RoomAdmissionBearer)) != 1 {
		return nil, fail(401, "Invalid service credential")
	}

	account, room, signature := text(body, "account_id"), text(body, "room_uri"), text(body, "signature")
	token, err := hex.DecodeString(signature)
	if account == "" || room == "" || err != nil || len(token) != 32 {
		return nil, fail(400, "Invalid room admission request")
	}
	stamp := now()
	expires := int64(binary.BigEndian.Uint64(token[:8]))
	if room != a.roomAdmissionURI() || expires <= stamp || expires-stamp > admissionTTLSeconds {
		return nil, fail(401, "Invalid or expired room admission")
	}
	u, err := a.findUser(r.Context, "id", account)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fail(401, "Invalid or expired room admission")
	}
	if err != nil {
		return nil, err
	}
	key, err := hex.DecodeString(a.cfg.RoomAdmissionKey)
	if err != nil || len(key) != 32 {
		return nil, errors.New("invalid room admission key")
	}
	session := stringValue(u.Session)
	expected := roomAdmissionToken(key, u.ID, session, room, expires)
	if session == "" || !hmac.Equal(token, expected[:]) {
		return nil, fail(401, "Invalid or expired room admission")
	}
	if err := a.checkAllowed(r.Context, &u, 403); err != nil {
		return nil, err
	}
	return Object{"account_id": u.ID, "expires_at": expires}, nil
}
