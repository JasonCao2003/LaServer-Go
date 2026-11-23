package snapshot

import (
	"errors"
	"slices"
)

// MaxDataBytes is the legacy snapshot reader capacity after reconstruction.
const MaxDataBytes = 7500

// HeaderSize counts the sequence, baseline, and checksum bytes.
const HeaderSize = 3

const historySize = 16

var zeroBaseline [MaxDataBytes]byte

type Frame struct {
	Sequence uint8
	Base     uint8
	Checksum uint8
	Payload  []byte
}

func Parse(data []byte) (Frame, error) {
	if len(data) < HeaderSize {
		return Frame{}, errors.New("truncated snapshot header")
	}
	return Frame{Sequence: data[0], Base: data[1], Checksum: data[2], Payload: data[HeaderSize:]}, nil
}

func (f Frame) Bytes() []byte {
	data := make([]byte, HeaderSize+len(f.Payload))
	data[0], data[1], data[2] = f.Sequence, f.Base, f.Checksum
	copy(data[HeaderSize:], f.Payload)
	return data
}

type keyframe struct {
	sequence uint8
	data     []byte
}

// Reader reconstructs one incoming snapshot stream. Its zero value is ready to use.
type Reader struct {
	history   []keyframe
	latest    uint8
	resyncing bool
}

func (r *Reader) LatestSequence() uint8 { return r.latest }

// Read returns a read-only snapshot that may also serve as a later baseline.
func (r *Reader) Read(frame Frame) ([]byte, error) {
	if len(frame.Payload) > MaxDataBytes || checksum(frame.Payload) != frame.Checksum {
		return nil, errors.New("invalid snapshot size or checksum")
	}
	if r.history == nil || (frame.Base == 0 && frame.Sequence != 0 && !r.resyncing) {
		clear(r.history)
		r.history = append(r.history[:0], keyframe{sequence: 1, data: zeroBaseline[:]})
	}

	var data []byte
	if frame.Base == 0 {
		data = slices.Clone(frame.Payload)
		if frame.Sequence != 0 {
			r.resyncing = true
		}
	} else {
		index := findKeyframe(r.history, frame.Base)
		if index < 0 {
			return nil, errors.New("snapshot baseline is unavailable")
		}
		var err error
		data, err = applyDelta(r.history[index].data, frame.Payload)
		if err != nil {
			return nil, err
		}
		r.resyncing = false
	}
	if frame.Sequence != 0 {
		r.latest = frame.Sequence
		r.history = remember(r.history, keyframe{sequence: frame.Sequence, data: data})
	}
	return data, nil
}

// Writer owns the outgoing baseline for one recipient and one snapshot type.
// Its zero value starts a new stream with a full keyframe.
type Writer struct {
	history  []keyframe
	sequence uint8
	frames   uint32
	synced   bool
}

func (w *Writer) Write(data []byte) (Frame, error) {
	if len(data) > MaxDataBytes {
		return Frame{}, errors.New("snapshot exceeds the legacy size limit")
	}
	if len(w.history) == historySize {
		if w.synced {
			w.synced = false
			w.frames = 0
		} else {
			w.discardBefore(historySize / 2)
		}
	}
	interval := uint32(5)
	if w.synced {
		interval = 10
	}
	frame := Frame{Payload: slices.Clone(data)}
	if w.frames%interval == 0 {
		frame.Sequence = w.allocateSequence()
	}
	if w.synced {
		base := w.history[0]
		if len(base.data) == len(data) {
			delta := createDelta(base.data, data)
			if len(delta) < len(data) {
				frame.Base, frame.Payload = base.sequence, delta
			}
		}
		if frame.Base == 0 && frame.Sequence == 0 {
			frame.Sequence = w.allocateSequence()
		}
	}
	if frame.Sequence != 0 {
		w.history = remember(w.history, keyframe{sequence: frame.Sequence, data: slices.Clone(data)})
	}
	w.frames++
	frame.Checksum = checksum(frame.Payload)
	return frame, nil
}

func (w *Writer) Ack(sequence uint8) {
	if index := findKeyframe(w.history, sequence); index >= 0 {
		w.discardBefore(index)
		w.synced = true
	}
}

func (w *Writer) discardBefore(index int) {
	remaining := copy(w.history, w.history[index:])
	clear(w.history[remaining:])
	w.history = w.history[:remaining]
}

func (w *Writer) allocateSequence() uint8 {
	if w.sequence == 0 {
		w.sequence = 1
	}
	w.sequence = (w.sequence + 1) & 0x7f
	if w.sequence == 0 {
		w.sequence = 1
	}
	return w.sequence
}

func findKeyframe(history []keyframe, sequence uint8) int {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].sequence == sequence {
			return i
		}
	}
	return -1
}

func remember(history []keyframe, snapshot keyframe) []keyframe {
	if len(history) < historySize {
		return append(history, snapshot)
	}
	copy(history, history[1:])
	history[len(history)-1] = snapshot
	return history
}

func checksum(data []byte) uint8 {
	hash := uint8(197)
	for _, value := range data {
		hash = (hash ^ value) * 147
	}
	return hash
}
