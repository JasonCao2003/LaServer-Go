package room

import (
	"time"

	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/protocol"
)

const (
	clientLeaveGame   byte = 1
	clientLevelUpdate byte = 4
	clientTimePing    byte = 12
	clientJoinGame    byte = 3
)

// 处理已进房连接的数据
func (s *Server) receiveActive(conn *connection, frame []byte, now time.Time) {
	peer := conn.ticket.Peer
	switch frame[0] {
	case clientLeaveGame:
		s.drop(peer)
	case clientLevelUpdate:
		update, err := protocol.DecodeLevelUpdate(frame, protocol.ClientIDSequence)
		if err == nil {
			s.changeLevel(conn, update)
		}
	case clientTimePing:
		ping, err := protocol.DecodeTimePing(frame, protocol.ClientIDSequence)
		if err != nil {
			return
		}
		pong := protocol.EncodeTimePong(ping.RequestID, now.Sub(s.start).Seconds(), time.Since(s.start).Seconds())
		if err := s.host.Send(peer, 0, false, pong); err != nil {
			s.drop(peer)
		}
	case protocol.GameMessageID:
		s.receiveGameMessage(conn, frame)
	}
}
