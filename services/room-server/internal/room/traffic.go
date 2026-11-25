package room

import (
	"fmt"
	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/traffic"
	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/transport"
	"time"
)

type observedHost struct{ Host }

func frameInfo(peer uint64, ch uint8, data []byte) string {
	kind := -1
	if len(data) > 0 {
		kind = int(data[0])
	}
	return fmt.Sprintf("peer=%d channel=%d type=%d bytes=%d", peer, ch, kind, len(data))
}
func (h observedHost) Poll(t time.Duration) (transport.Event, bool, error) {
	e, ok, err := h.Host.Poll(t)
	if ok && e.Kind == transport.Received {
		traffic.Add("接收 "+frameInfo(e.Peer, e.Channel, e.Data), "")
	}
	return e, ok, err
}
func (h observedHost) Send(peer uint64, ch uint8, reliable bool, data []byte) error {
	err := h.Host.Send(peer, ch, reliable, data)
	result := "已提交"
	if err != nil {
		result = "失败"
	}
	traffic.Add("", fmt.Sprintf("发送 %s reliable=%t %s", frameInfo(peer, ch, data), reliable, result))
	return err
}
