package protocol

import (
	"encoding/binary"
	"fmt"
	"math"
)

// ClientLayout is the frame layout selected for an established connection.
type ClientLayout uint8

const (
	ClientIDSequence ClientLayout = iota + 1
	ClientTokenIDSequence
)

const (
	timePingID    = 12
	timePongID    = 13
	timePingSize  = 2
	timePongSize  = 18
	serverHeader  = 3
	clientHeader  = 2
	wireTokenSize = 4
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
	var idOffset int
	switch layout {
	case ClientIDSequence:
	case ClientTokenIDSequence:
		idOffset = wireTokenSize
	default:
		return TimePing{}, fmt.Errorf("unsupported client layout %d", layout)
	}

	payloadOffset := idOffset + clientHeader
	if len(frame) != payloadOffset+timePingSize {
		return TimePing{}, fmt.Errorf("invalid time ping frame length %d for layout %d", len(frame), layout)
	}
	if id := frame[idOffset]; id != timePingID {
		return TimePing{}, fmt.Errorf("invalid time ping packet ID %d", id)
	}

	ping := TimePing{
		Sequence:  frame[idOffset+1],
		RequestID: binary.LittleEndian.Uint16(frame[payloadOffset:]),
	}
	if layout == ClientTokenIDSequence {
		ping.Token = binary.LittleEndian.Uint32(frame)
	}
	return ping, nil
}

// EncodeTimePong encodes the server frame with the original client request ID.
func EncodeTimePong(requestID uint16, receivedAt, sentAt float64) []byte {
	frame := make([]byte, serverHeader+timePongSize)
	frame[0] = timePongID
	binary.LittleEndian.PutUint16(frame[1:serverHeader], timePongSize)

	payload := frame[serverHeader:]
	binary.LittleEndian.PutUint16(payload[0:2], requestID)
	binary.LittleEndian.PutUint64(payload[2:10], math.Float64bits(receivedAt))
	binary.LittleEndian.PutUint64(payload[10:18], math.Float64bits(sentAt))
	return frame
}
