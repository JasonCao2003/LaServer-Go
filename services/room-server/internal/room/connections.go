package room

import "github.com/JasonCao2003/LaServer-Go/services/room-server/internal/protocol"

// 丢弃指定连接
func (s *Server) drop(peer uint64) {
	s.remove(peer)
	s.host.Disconnect(peer, 0)
}

// 移除指定玩家
func (s *Server) remove(peer uint64) {
	conn := s.connections[peer]
	if conn == nil {
		return
	}
	if conn.cancel != nil {
		conn.cancel()
	}
	s.sessions.Disconnect(conn.ticket)
	delete(s.connections, peer)
	if conn.player != nil {
		s.leaveLevel(conn.group(), conn)
		s.broadcast(peer, protocol.EncodePlayerLeft(conn.player.NetID, 0))
	}
}
