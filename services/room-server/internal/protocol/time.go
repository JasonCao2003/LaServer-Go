package protocol

import (
	"encoding/binary"
	"fmt"
	"math"
)

const (
	timePingID   = 12
	timePongID   = 13
	timePingSize = 2
	timePongSize = 18
)

// TimePing carries the client sequence and the request ID to echo in TimePong.
// Token is the value in the frame header when present.
type TimePing struct {
	Token     uint32
	Sequence  uint8
	RequestID uint16
}

// DecodeTimePing decodes a ping using the layout selected by the caller.
func DecodeTimePing(frame []byte, layout ClientLayout) (TimePing, error) {
	packet, err := decodeClientFrame(frame, layout, timePingID)
	if err != nil {
		return TimePing{}, err
	}
	if len(packet.payload) != timePingSize {
		return TimePing{}, fmt.Errorf("invalid time ping payload length %d", len(packet.payload))
	}
	return TimePing{
		Token:     packet.token,
		Sequence:  packet.sequence,
		RequestID: binary.LittleEndian.Uint16(packet.payload),
	}, nil
}

// EncodeTimePong encodes the server frame with the original client request ID.
func EncodeTimePong(requestID uint16, receivedAt, sentAt float64) []byte {
	frame := newServerFrame(timePongID, timePongSize)

	payload := frame[serverHeader:]
	binary.LittleEndian.PutUint16(payload[0:2], requestID)
	binary.LittleEndian.PutUint64(payload[2:10], math.Float64bits(receivedAt))
	binary.LittleEndian.PutUint64(payload[10:18], math.Float64bits(sentAt))
	return frame
}
