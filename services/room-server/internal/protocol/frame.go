package protocol

import (
	"encoding/binary"
	"fmt"
)

// ClientLayout is the frame layout selected for an established connection.
type ClientLayout uint8

const (
	ClientIDSequence ClientLayout = iota + 1
	ClientTokenIDSequence
)

const (
	serverHeader  = 3
	clientHeader  = 2
	wireTokenSize = 4
)

type clientFrame struct {
	token    uint32
	sequence uint8
	payload  []byte
}

func decodeClientFrame(frame []byte, layout ClientLayout, expectedID byte) (clientFrame, error) {
	var idOffset int
	switch layout {
	case ClientIDSequence:
	case ClientTokenIDSequence:
		idOffset = wireTokenSize
	default:
		return clientFrame{}, fmt.Errorf("unsupported client layout %d", layout)
	}

	payloadOffset := idOffset + clientHeader
	if len(frame) < payloadOffset {
		return clientFrame{}, fmt.Errorf("truncated client header for layout %d", layout)
	}
	if id := frame[idOffset]; id != expectedID {
		return clientFrame{}, fmt.Errorf("unexpected client packet ID %d, want %d", id, expectedID)
	}

	decoded := clientFrame{
		sequence: frame[idOffset+1],
		payload:  frame[payloadOffset:],
	}
	if layout == ClientTokenIDSequence {
		decoded.token = binary.LittleEndian.Uint32(frame)
	}
	return decoded, nil
}

func newServerFrame(id byte, payloadSize uint16) []byte {
	frame := make([]byte, serverHeader+int(payloadSize))
	frame[0] = id
	binary.LittleEndian.PutUint16(frame[1:serverHeader], payloadSize)
	return frame
}
