package protocol

import (
	"encoding/binary"
	"fmt"

	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/protocol/snapshot"
)

// EnterGameLayout is the roster format selected by the receiver's engine profile.
type EnterGameLayout uint8

const (
	EnterGameLegacy EnterGameLayout = iota + 1
	EnterGameModern
)

const (
	enterGameLegacyID = 5
	enterGameModernID = 17
	rosterRecordSize  = 37
	MaxRosterPlayers  = 255
)

// MinimumRoomFrameBytes covers the largest supported snapshot and a full room roster.
func MinimumRoomFrameBytes(maxPlayers int) int {
	return max(GameSnapshotOverhead+snapshot.MaxDataBytes, serverHeader+1+maxPlayers*rosterRecordSize)
}

// RosterPlayer is one player in the EnterGame roster.
type RosterPlayer struct {
	NetID   uint8
	UUID    [16]byte
	Name    string
	LevelID uint32
}

// EncodeEnterGame encodes a roster using the format selected for the receiver.
func EncodeEnterGame(layout EnterGameLayout, players []RosterPlayer) ([]byte, error) {
	var id byte
	countOffset := serverHeader
	switch layout {
	case EnterGameLegacy:
		id = enterGameLegacyID
		countOffset++ // The legacy roster starts with a reserved zero byte.
	case EnterGameModern:
		id = enterGameModernID
	default:
		return nil, fmt.Errorf("unsupported EnterGame layout %d", layout)
	}

	if len(players) > MaxRosterPlayers {
		return nil, fmt.Errorf("EnterGame roster has %d players, maximum is %d", len(players), MaxRosterPlayers)
	}

	frame := newServerFrame(id, uint16(countOffset+1-serverHeader+len(players)*rosterRecordSize))
	frame[countOffset] = uint8(len(players))

	records := frame[countOffset+1:]
	for i, player := range players {
		offset := i * rosterRecordSize
		writeRosterPlayer(records[offset:offset+rosterRecordSize], player)
	}

	return frame, nil
}

func writeRosterPlayer(record []byte, player RosterPlayer) {
	record[0] = player.NetID
	copy(record[1:17], player.UUID[:])
	copy(record[17:33], player.Name)
	binary.LittleEndian.PutUint32(record[33:37], player.LevelID)
}
