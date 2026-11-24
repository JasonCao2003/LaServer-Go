package room

import (
	"context"
	"time"

	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/protocol"
)

// 开始处理进房请求
func (s *Server) begin(ctx context.Context, conn *connection, frame []byte, now time.Time) {
	if err := s.sessions.StartJoin(conn.ticket, now); err != nil {
		return
	}
	join, proof, err := protocol.DecodeJoinAdmission(frame)
	if err != nil || !join.HasLevelID {
		s.drop(conn.ticket.Peer)
		return
	}
	select {
	case s.verifiers <- struct{}{}:
	default:
		s.drop(conn.ticket.Peer)
		return
	}

	job, cancel := context.WithTimeout(ctx, s.joinTimeout)
	conn.cancel = cancel
	ticket := conn.ticket
	go func() {
		defer func() { <-s.verifiers }()
		defer cancel()
		identity, err := s.verifier.Verify(job, s.roomURI, proof)
		select {
		case s.results <- admissionResult{ticket: ticket, join: join, identity: identity, err: err}:
		case <-ctx.Done():
		}
	}()
}

// 完成进房校验并加入房间
func (s *Server) complete(result admissionResult, now time.Time) {
	peer := result.ticket.Peer
	conn := s.connections[peer]
	if conn == nil || conn.ticket != result.ticket {
		return
	}
	if result.err != nil || !now.Before(result.identity.ExpiresAt) ||
		s.sessions.Admit(result.ticket, result.identity.AccountID, now) != nil {
		s.drop(peer)
		return
	}

	players, id := s.rosterAndNextID()
	if id == 0 {
		s.drop(peer)
		return
	}
	player := protocol.RosterPlayer{NetID: id, UUID: result.join.UUID, LevelID: result.join.LevelID}
	frame, err := protocol.EncodeEnterGame(result.join.EnterGame, append(players, player))
	if err != nil || s.host.Send(peer, 0, false, frame) != nil {
		s.drop(peer)
		return
	}
	conn.player = &player
	conn.version = [2]uint16{result.join.ServerVersion, result.join.ClientVersion}
	conn.levelChangeCount = 1
	s.levelFor(conn)
	// Active sky0134 clients also receive the complete roster when a player joins.
	s.broadcast(peer, frame)
}
