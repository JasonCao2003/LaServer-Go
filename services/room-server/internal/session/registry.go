// 连接会话与准入状态注册表

package session

import (
	"errors"
	"time"
)

var (
	ErrUnknown          = errors.New("connection is no longer current")
	ErrState            = errors.New("operation is invalid for this session state")
	ErrExpired          = errors.New("join deadline expired")
	ErrFull             = errors.New("session capacity reached")
	ErrIdentity         = errors.New("missing verified account identity")
	ErrDuplicateAccount = errors.New("account is already online")
)

// Ticket identifies a specific use of a transport peer slot.
type Ticket struct {
	Peer       uint64
	generation uint64
}

type entry struct {
	ticket    Ticket
	deadline  time.Time
	verifying bool
	account   string
}

// Registry owns the connection and online-account indexes. Call its methods
// from the same event loop; it does not synchronize concurrent calls.
type Registry struct {
	maxPeers    int
	maxPlayers  int
	joinTimeout time.Duration
	generation  uint64
	peers       map[uint64]*entry
	accounts    map[string]struct{}
}

// 创建会话注册表
func NewRegistry(maxPeers, maxPlayers int, joinTimeout time.Duration) *Registry {
	return &Registry{
		maxPeers:    maxPeers,
		maxPlayers:  maxPlayers,
		joinTimeout: joinTimeout,
		peers:       make(map[uint64]*entry),
		accounts:    make(map[string]struct{}),
	}
}

// 登记新连接票据
func (r *Registry) Connect(peer uint64, now time.Time) (Ticket, error) {
	if _, exists := r.peers[peer]; exists {
		return Ticket{}, ErrState
	}
	if len(r.peers) >= r.maxPeers {
		return Ticket{}, ErrFull
	}
	r.generation++
	ticket := Ticket{Peer: peer, generation: r.generation}
	r.peers[peer] = &entry{ticket: ticket, deadline: now.Add(r.joinTimeout)}
	return ticket, nil
}

// 标记票据进入加入流程
func (r *Registry) StartJoin(ticket Ticket, now time.Time) error {
	session := r.current(ticket)
	if session == nil {
		return ErrUnknown
	}
	if session.verifying || session.account != "" {
		return ErrState
	}
	if !now.Before(session.deadline) {
		return ErrExpired
	}
	session.verifying = true
	return nil
}

// 标记票据已准入
func (r *Registry) Admit(ticket Ticket, verifiedAccount string, now time.Time) error {
	session := r.current(ticket)
	if session == nil {
		return ErrUnknown
	}
	if !session.verifying || session.account != "" {
		return ErrState
	}
	if !now.Before(session.deadline) {
		return ErrExpired
	}
	if verifiedAccount == "" {
		return ErrIdentity
	}
	if r.Online(verifiedAccount) {
		return ErrDuplicateAccount
	}
	if len(r.accounts) >= r.maxPlayers {
		return ErrFull
	}
	session.verifying = false
	session.account = verifiedAccount
	r.accounts[verifiedAccount] = struct{}{}
	return nil
}

// 断开并移除连接票据
func (r *Registry) Disconnect(ticket Ticket) {
	session := r.current(ticket)
	if session == nil {
		return
	}
	if session.account != "" {
		delete(r.accounts, session.account)
	}
	delete(r.peers, ticket.Peer)
}

// 清理超时票据
func (r *Registry) Expire(now time.Time) []Ticket {
	var expired []Ticket
	for peer, session := range r.peers {
		if session.account == "" && !now.Before(session.deadline) {
			expired = append(expired, session.ticket)
			delete(r.peers, peer)
		}
	}
	return expired
}

// 判断账号是否在线
func (r *Registry) Online(account string) bool {
	_, ok := r.accounts[account]
	return ok
}

// 读取票据当前记录
func (r *Registry) current(ticket Ticket) *entry {
	session := r.peers[ticket.Peer]
	if session == nil || session.ticket != ticket {
		return nil
	}
	return session
}
