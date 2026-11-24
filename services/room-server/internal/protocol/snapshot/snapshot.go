// 快照帧解析、读写与历史管理

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

// 解析快照帧
func Parse(data []byte) (Frame, error) {
	if len(data) < HeaderSize {
		return Frame{}, errors.New("truncated snapshot header")
	}
	return Frame{Sequence: data[0], Base: data[1], Checksum: data[2], Payload: data[HeaderSize:]}, nil
}

// 返回快照帧字节
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

// 返回已读到的最新序号
func (r *Reader) LatestSequence() uint8 { return r.latest }

// 读取并应用快照帧
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

// 写入快照数据并返回帧
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

// 记录对端确认序号
func (w *Writer) Ack(sequence uint8) {
	if index := findKeyframe(w.history, sequence); index >= 0 {
		w.discardBefore(index)
		w.synced = true
	}
}

// 丢弃指定序号前的历史
func (w *Writer) discardBefore(index int) {
	remaining := copy(w.history, w.history[index:])
	clear(w.history[remaining:])
	w.history = w.history[:remaining]
}

// 分配下一个快照序号
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

// 查找可用的关键帧
func findKeyframe(history []keyframe, sequence uint8) int {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].sequence == sequence {
			return i
		}
	}
	return -1
}

// 记录快照关键帧历史
func remember(history []keyframe, snapshot keyframe) []keyframe {
	if len(history) < historySize {
		return append(history, snapshot)
	}
	copy(history, history[1:])
	history[len(history)-1] = snapshot
	return history
}

// 计算快照校验和
func checksum(data []byte) uint8 {
	hash := uint8(197)
	for _, value := range data {
		hash = (hash ^ value) * 147
	}
	return hash
}
