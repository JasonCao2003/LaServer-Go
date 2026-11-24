// 玩家进出与关卡变更通知编解码

package protocol

import (
	"encoding/binary"
	"fmt"
)

const (
	playerJoinedID       = 6
	playerLeftID         = 7
	playerChangedLevelID = 8
	moveResultID         = 10
	cancelMoveID         = 11
)

// 编码玩家加入通知
func EncodePlayerJoined(player RosterPlayer) []byte {
	frame := newServerFrame(playerJoinedID, rosterRecordSize)
	writeRosterPlayer(frame[serverHeader:], player)
	return frame
}

// 编码玩家离开通知
func EncodePlayerLeft(playerID, reason uint8) []byte {
	frame := newServerFrame(playerLeftID, 2)
	frame[serverHeader] = playerID
	frame[serverHeader+1] = reason
	return frame
}

// 编码玩家切换关卡通知
func EncodePlayerChangedLevel(playerID uint8, levelID uint32) []byte {
	frame := newServerFrame(playerChangedLevelID, 5)
	frame[serverHeader] = playerID
	binary.LittleEndian.PutUint32(frame[serverHeader+1:], levelID)
	return frame
}

// MoveControl preserves the result or cancellation code and any version-specific tail.
type MoveControl struct {
	Token    uint32
	Sequence uint8
	Code     uint8
	Extra    []byte
}

// 解析移动结果消息
func DecodeMoveResult(frame []byte, layout ClientLayout) (MoveControl, error) {
	return decodeMoveControl(frame, layout, moveResultID)
}

// 解析取消移动消息
func DecodeCancelMove(frame []byte, layout ClientLayout) (MoveControl, error) {
	return decodeMoveControl(frame, layout, cancelMoveID)
}

// 解析移动控制消息
func decodeMoveControl(frame []byte, layout ClientLayout, id byte) (MoveControl, error) {
	packet, err := decodeClientFrame(frame, layout, id)
	if err != nil {
		return MoveControl{}, err
	}
	if len(packet.payload) == 0 {
		return MoveControl{}, fmt.Errorf("move packet %d is missing its code", id)
	}
	return MoveControl{
		Token:    packet.token,
		Sequence: packet.sequence,
		Code:     packet.payload[0],
		Extra:    append([]byte(nil), packet.payload[1:]...),
	}, nil
}
