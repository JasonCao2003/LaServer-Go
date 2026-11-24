// 加入游戏指纹解析

package protocol

import (
	"encoding/binary"
	"fmt"
)

const (
	joinGameID              = 3
	joinSplitCountOffset    = 62
	joinFingerprintOffset   = 82
	joinFingerprintSize     = 8
	joinSplitPlayerSize     = 16
	joinMaxSplitPlayers     = 8
	joinLegacyLevelOffset   = 17
	joinStandardLevelOffset = 21
)

// JoinFingerprint identifies a known two-byte-header profile.
// The UUID and version fields are client declarations, never authentication.
type JoinFingerprint struct {
	UUID          [16]byte
	Sequence      uint8
	ServerVersion uint16
	ClientVersion uint16
	Build         uint32
	HasBuild      bool
	LevelID       uint32
	HasLevelID    bool
	Engine        string
	EnterGame     EnterGameLayout
}

// 解析加入游戏指纹
func DecodeJoinGameFingerprint(frame []byte) (JoinFingerprint, error) {
	packet, err := decodeClientFrame(frame, ClientIDSequence, joinGameID)
	if err != nil {
		return JoinFingerprint{}, err
	}

	payload := packet.payload
	if len(payload) < joinFingerprintOffset+joinFingerprintSize {
		return JoinFingerprint{}, fmt.Errorf("truncated JoinGame fingerprint")
	}

	split := binary.LittleEndian.Uint32(payload[joinSplitCountOffset:])
	if split > joinMaxSplitPlayers {
		return JoinFingerprint{}, fmt.Errorf("JoinGame split count %d exceeds %d", split, joinMaxSplitPlayers)
	}
	offset := joinFingerprintOffset + int(split)*joinSplitPlayerSize
	if len(payload)-offset < joinFingerprintSize {
		return JoinFingerprint{}, fmt.Errorf("truncated JoinGame after split player list")
	}

	fingerprint := payload[offset:]
	join := JoinFingerprint{
		Sequence:      packet.sequence,
		ServerVersion: binary.LittleEndian.Uint16(fingerprint[0:2]),
		ClientVersion: binary.LittleEndian.Uint16(fingerprint[2:4]),
		Build:         binary.LittleEndian.Uint32(fingerprint[4:8]),
		HasBuild:      true,
		EnterGame:     EnterGameModern,
	}
	copy(join.UUID[:], payload[:16])

	levelOffset := joinStandardLevelOffset
	switch {
	case join.ServerVersion == 252 && join.ClientVersion == 477:
		join.Engine = "sky071"
		join.Build = 0
		join.HasBuild = false
		join.EnterGame = EnterGameLegacy
		levelOffset = joinLegacyLevelOffset
	case join.ServerVersion == 275 && join.ClientVersion == 516 && join.Build != 0 && join.Build != ^uint32(0):
		join.Engine = "sky0122"
	case join.ServerVersion == 294 && join.ClientVersion == 4852 && join.Build == 179644:
		// The old sky0155 profile confirms the engine but resolves level separately.
		join.Engine = "sky0155"
		return join, nil
	case join.ServerVersion >= 284 && join.ServerVersion <= 294 &&
		join.ClientVersion >= 500 && join.ClientVersion <= 9999 &&
		join.Build != 0 && join.Build != ^uint32(0):
		join.Engine = "sky0134"
	default:
		return JoinFingerprint{}, fmt.Errorf("unsupported JoinGame version %d/%d/%d", join.ServerVersion, join.ClientVersion, join.Build)
	}

	if len(fingerprint) < levelOffset+4 {
		return JoinFingerprint{}, fmt.Errorf("truncated JoinGame level ID")
	}
	join.LevelID = binary.LittleEndian.Uint32(fingerprint[levelOffset:])
	join.HasLevelID = true
	return join, nil
}
