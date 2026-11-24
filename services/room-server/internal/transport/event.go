// 传输层事件结构定义

package transport

type EventKind uint8

const (
	Connected EventKind = iota + 1
	Disconnected
	Received
)

type Event struct {
	Kind     EventKind
	Peer     uint64
	Channel  uint8
	Reliable bool
	Data     []byte
}
