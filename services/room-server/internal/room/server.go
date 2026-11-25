// 房间服务事件循环与玩家管理

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

// 创建房间服务实例
func NewServer(host Host, cfg config.Config, roomURI string, verifier Verifier) *Server {
	timeout := time.Duration(cfg.JoinTimeoutMS) * time.Millisecond
	return &Server{
		host:        observedHost{host},
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

// 运行房间事件循环
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

// 关闭传输资源
func (s *Server) Close() error { return s.host.Close() }

// 分发底层网络事件
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
			if event.Channel == 0 && event.Data[0] == clientJoinGame {
				s.begin(ctx, conn, event.Data, now)
			}
			return
		}
		s.receiveActive(conn, event.Data, now)
	}
}

// 向其他玩家广播数据
func (s *Server) broadcast(except uint64, frame []byte) {
	for peer, conn := range s.connections {
		if peer != except && conn.player != nil {
			_ = s.host.Send(peer, 0, false, frame)
		}
	}
}

// 生成玩家列表与下一个玩家 ID
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
