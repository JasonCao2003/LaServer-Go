package snapshot

import (
	"bytes"
	"testing"
)

func TestWriterMatchesLegacyFramesAndReaderReconstructs(t *testing.T) {
	var reader Reader
	var writer Writer
	tests := []struct {
		data []byte
		want Frame
	}{
		{[]byte{1, 2, 3, 4, 5, 6, 7, 8}, Frame{Sequence: 2, Checksum: 141, Payload: []byte{1, 2, 3, 4, 5, 6, 7, 8}}},
		{[]byte{1, 2, 3, 4, 0, 6, 7, 8}, Frame{Base: 2, Checksum: 101, Payload: []byte{0, 4, 5, 0, 3}}},
		{[]byte{1, 2, 3, 4, 5, 6, 7, 8, 0}, Frame{Sequence: 3, Checksum: 247, Payload: []byte{1, 2, 3, 4, 5, 6, 7, 8, 0}}},
	}
	for i, test := range tests {
		frame, err := writer.Write(test.data)
		if err != nil || frame.Sequence != test.want.Sequence || frame.Base != test.want.Base ||
			frame.Checksum != test.want.Checksum || !bytes.Equal(frame.Payload, test.want.Payload) {
			t.Fatalf("frame %d = %+v, %v, want %+v", i, frame, err, test.want)
		}
		decoded, err := reader.Read(frame)
		if err != nil || !bytes.Equal(decoded, test.data) {
			t.Fatalf("read frame %d = %x, %v", i, decoded, err)
		}
		if i == 0 {
			writer.Ack(frame.Sequence)
		}
	}
	if reader.LatestSequence() != 3 {
		t.Fatalf("latest accepted sequence = %d", reader.LatestSequence())
	}
}

func TestReaderRejectsMalformedDeltaWithoutAdvancingAck(t *testing.T) {
	var reader Reader
	if _, err := reader.Read(Frame{Sequence: 2, Checksum: 171, Payload: []byte{1, 2, 3}}); err != nil {
		t.Fatal(err)
	}
	tooLarge := bytes.Repeat([]byte{0, 255}, 30)
	for _, frame := range []Frame{
		{Sequence: 3, Checksum: 0, Payload: []byte{1}},
		{Sequence: 3, Base: 99, Checksum: checksum([]byte{1}), Payload: []byte{1}},
		{Sequence: 3, Base: 2, Checksum: checksum([]byte{0}), Payload: []byte{0}},
		{Sequence: 3, Base: 2, Checksum: checksum(tooLarge), Payload: tooLarge},
	} {
		if _, err := reader.Read(frame); err == nil {
			t.Errorf("accepted malformed snapshot %+v", frame)
		}
	}
	if reader.LatestSequence() != 2 {
		t.Fatalf("invalid frame advanced ACK to %d", reader.LatestSequence())
	}
}
