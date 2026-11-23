package protocol

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/protocol/snapshot"
)

const GameMessageID = 14

const (
	GameMsgNetRPC        = 2
	GameMsgPlayerState   = 3
	GameMsgCritters      = 4
	GameMsgAffinity      = 7
	GameMsgLevelElection = 8
	GameMsgLevelData     = 11
	GameMsgSnapshotAck   = 14
	GameMsgMusicSync     = 15
	gameMessageHeader    = 3
	// GameSnapshotOverhead includes the server, GameMsg, and snapshot headers.
	GameSnapshotOverhead = serverHeader + gameMessageHeader + snapshot.HeaderSize
)

// GameMessage carries the nested header and the type-specific bytes that follow it.
type GameMessage struct {
	Type             uint8
	LevelChangeCount uint8
	SourcePlayer     uint8
	Payload          []byte
}

func DecodeGameMessage(frame []byte, layout ClientLayout) (GameMessage, error) {
	packet, err := decodeClientFrame(frame, layout, GameMessageID)
	if err != nil {
		return GameMessage{}, err
	}
	if len(packet.payload) < gameMessageHeader {
		return GameMessage{}, fmt.Errorf("truncated GameMsg header")
	}
	msg := GameMessage{
		Type:             packet.payload[0],
		LevelChangeCount: packet.payload[1],
		SourcePlayer:     packet.payload[2],
		Payload:          packet.payload[gameMessageHeader:],
	}
	var minimum int
	switch msg.Type {
	case GameMsgNetRPC:
		minimum = 1 // The target player precedes the opaque RPC bitstream.
	case GameMsgPlayerState, GameMsgLevelElection, GameMsgLevelData:
		minimum = snapshot.HeaderSize
	case GameMsgSnapshotAck:
		minimum = 2
	}
	if len(msg.Payload) < minimum {
		return GameMessage{}, fmt.Errorf("truncated GameMsg type %d", msg.Type)
	}
	return msg, nil
}

func EncodeGameMessage(msg GameMessage) ([]byte, error) {
	if len(msg.Payload) > math.MaxUint16-gameMessageHeader {
		return nil, fmt.Errorf("GameMsg payload is too large")
	}
	frame := newServerFrame(GameMessageID, uint16(gameMessageHeader+len(msg.Payload)))
	header := frame[serverHeader:]
	header[0], header[1], header[2] = msg.Type, msg.LevelChangeCount, msg.SourcePlayer
	copy(header[gameMessageHeader:], msg.Payload)
	return frame, nil
}

// EncodeLevelElection writes the empty NetLevelData header and default legacy padding.
// The authority's opaque world state is sent in a separate GameMsg.
func EncodeLevelElection(authority uint8, levelID uint32, levelHash uint16) []byte {
	const electHeaderSize, electPadding = 22, 16
	data := make([]byte, electHeaderSize+electPadding)
	data[0] = authority
	binary.LittleEndian.PutUint32(data[1:5], levelID)
	binary.LittleEndian.PutUint32(data[11:15], uint32(levelHash))
	return data
}
