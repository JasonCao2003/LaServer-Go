// 关卡更新消息解析

package protocol

import (
	"encoding/binary"
	"fmt"
)

const (
	levelUpdateID   = 4
	levelUpdateSize = 82
)

// LevelUpdate carries the fixed fields shared by legacy and extended layouts.
type LevelUpdate struct {
	Token            uint32
	Sequence         uint8
	LevelChangeCount uint8
	NetVersion       [6]uint16
	LevelStatus      uint8
	LevelID          uint32
	LevelSummary     [64]byte
	HasTrailingState bool
	TrailingState    uint8
}

// 解析关卡更新消息
func DecodeLevelUpdate(frame []byte, layout ClientLayout) (LevelUpdate, error) {
	packet, err := decodeClientFrame(frame, layout, levelUpdateID)
	if err != nil {
		return LevelUpdate{}, err
	}
	payload := packet.payload
	if len(payload) < levelUpdateSize {
		return LevelUpdate{}, fmt.Errorf("truncated LevelUpdate payload of length %d", len(payload))
	}

	update := LevelUpdate{
		Token:            packet.token,
		Sequence:         packet.sequence,
		LevelChangeCount: payload[0],
		LevelStatus:      payload[13],
		LevelID:          binary.LittleEndian.Uint32(payload[14:18]),
	}
	for i := range update.NetVersion {
		offset := 1 + i*2
		update.NetVersion[i] = binary.LittleEndian.Uint16(payload[offset : offset+2])
	}
	copy(update.LevelSummary[:], payload[18:levelUpdateSize])
	if len(payload) > levelUpdateSize {
		update.HasTrailingState = true
		update.TrailingState = payload[levelUpdateSize]
	}
	return update, nil
}
