package room

import (
	"context"
	"sort"
	"time"

	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/config"
	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/protocol"
	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/session"
	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/transport"
)

const (
	pollInterval = 20 * time.Millisecond
	maxVerifiers = 8
)

type Host interface {
	Poll(time.Duration) (transport.Event, bool, error)
	Send(peer uint64, channel uint8, reliable bool, data []byte) error
	Disconnect(peer uint64, code uint32)
	Close() error
}

type Verifier interface {
	Verify(context.Context, string, protocol.AdmissionProof) (session.VerifiedAdmission, error)
}

type connection struct {
	ticket           session.Ticket
	cancel           context.CancelFunc
	player           *protocol.RosterPlayer
	version          [2]uint16
	levelChangeCount uint8
	levelHash        uint16
	hasLevelUpdate   bool
	playerSync       snapshotStream
	levelSync        snapshotStream
}

type admissionResult struct {
	ticket   session.Ticket
	join     protocol.JoinFingerprint
	identity session.VerifiedAdmission
	err      error
}

// Server is a single room for the measured 284/542 client profile.
type Server struct {
	host        Host
	verifier    Verifier
	roomURI     string
	joinTimeout time.Duration
	maxFrame    int
	sessions    *session.Registry
	connections map[uint64]*connection
	levels      map[levelGroup]*levelState
	results     chan admissionResult
	verifiers   chan struct{}
	start       time.Time
}

func NewServer(host Host, cfg config.Config, roomURI string, verifier Verifier) *Server {
	timeout := time.Duration(cfg.JoinTimeoutMS) * time.Millisecond
	return &Server{
		host:        host,
		verifier:    verifier,
		roomURI:     roomURI,
		joinTimeout: timeout,
		maxFrame:    cfg.MaxFrameBytes,
		sessions:    session.NewRegistry(cfg.MaxPeers, cfg.MaxPlayers, timeout),
		connections: make(map[uint64]*connection),
		levels:      make(map[levelGroup]*levelState),
		results:     make(chan admissionResult, maxVerifiers),
		verifiers:   make(chan struct{}, maxVerifiers),
		start:       time.Now(),
	}
}

func (s *Server) Serve(ctx context.Context, ready func()) error {
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		for _, conn := range s.connections {
			s.remove(conn.ticket.Peer)
			s.host.Disconnect(conn.ticket.Peer, 0)
		}
	}()
	ready()
	for ctx.Err() == nil {
		now := time.Now()
		for _, ticket := range s.sessions.Expire(now) {
			s.drop(ticket.Peer)
		}
		select {
		case result := <-s.results:
			s.complete(result, now)
		default:
		}
		event, ok, err := s.host.Poll(pollInterval)
		if err != nil {
			return err
		}
		if ok {
			s.handleEvent(ctx, event, time.Now())
		}
	}
	return ctx.Err()
}

func (s *Server) Close() error { return s.host.Close() }

func (s *Server) handleEvent(ctx context.Context, event transport.Event, now time.Time) {
	switch event.Kind {
	case transport.Connected:
		ticket, err := s.sessions.Connect(event.Peer, now)
		if err != nil {
			s.host.Disconnect(event.Peer, 0)
			return
		}
		s.connections[event.Peer] = &connection{ticket: ticket}
	case transport.Disconnected:
		s.remove(event.Peer)
	case transport.Received:
		conn := s.connections[event.Peer]
		if conn == nil || len(event.Data) == 0 {
			return
		}
		if conn.player == nil {
			if event.Channel == 0 && event.Data[0] == 3 {
				s.begin(ctx, conn, event.Data, now)
			}
			return
		}
		s.receiveActive(conn, event.Data, now)
	}
}

func (s *Server) begin(ctx context.Context, conn *connection, frame []byte, now time.Time) {
	if err := s.sessions.StartJoin(conn.ticket, now); err != nil {
		return
	}
	join, err := protocol.DecodeJoinGameFingerprint(frame)
	if err != nil || !join.HasLevelID {
		s.drop(conn.ticket.Peer)
		return
	}
	proof, err := protocol.DecodeAdmissionProof(frame)
	if err != nil {
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

func (s *Server) receiveActive(conn *connection, frame []byte, now time.Time) {
	peer := conn.ticket.Peer
	switch frame[0] {
	case 1:
		s.drop(peer)
	case 4:
		update, err := protocol.DecodeLevelUpdate(frame, protocol.ClientIDSequence)
		if err == nil {
			s.changeLevel(conn, update)
		}
	case 12:
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

func (s *Server) broadcast(except uint64, frame []byte) {
	for peer, conn := range s.connections {
		if peer != except && conn.player != nil {
			_ = s.host.Send(peer, 0, false, frame)
		}
	}
}

func (s *Server) rosterAndNextID() ([]protocol.RosterPlayer, uint8) {
	players := make([]protocol.RosterPlayer, 0, len(s.connections))
	var occupied [256]bool
	for _, conn := range s.connections {
		if conn.player != nil {
			player := *conn.player
			occupied[player.NetID] = true
			players = append(players, player)
		}
	}
	sort.Slice(players, func(i, j int) bool { return players[i].NetID < players[j].NetID })
	for id := 1; id < len(occupied); id++ {
		if !occupied[id] {
			return players, uint8(id)
		}
	}
	return players, 0
}

func (s *Server) drop(peer uint64) {
	s.remove(peer)
	s.host.Disconnect(peer, 0)
}

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
