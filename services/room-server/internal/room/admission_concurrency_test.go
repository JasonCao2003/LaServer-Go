package room

import (
	"context"
	"encoding/binary"
	"testing"
	"time"

	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/config"
	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/protocol"
	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/session"
	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/transport"
)

type heldVerification struct {
	ctx     context.Context
	release chan struct{}
}
type delayedVerifier chan heldVerification

func (v delayedVerifier) Verify(ctx context.Context, _ string, proof protocol.AdmissionProof) (session.VerifiedAdmission, error) {
	release := make(chan struct{})
	select {
	case v <- heldVerification{ctx, release}:
	case <-ctx.Done():
		return session.VerifiedAdmission{}, ctx.Err()
	}
	// Model a successful reply racing with cancellation.
	select {
	case <-release:
	case <-ctx.Done():
	}
	return session.VerifiedAdmission{AccountID: proof.AccountID, ExpiresAt: time.Now().Add(time.Minute)}, nil
}

func connectAndJoin(s *Server, ctx context.Context, peer uint64, identity byte) {
	frame := make([]byte, 1931)
	frame[0], frame[2] = 3, identity
	binary.LittleEndian.PutUint16(frame[84:], 284)
	binary.LittleEndian.PutUint16(frame[86:], 542)
	binary.LittleEndian.PutUint32(frame[88:], 354648)
	binary.LittleEndian.PutUint32(frame[105:], 42)
	s.handleEvent(ctx, transport.Event{Kind: transport.Connected, Peer: peer}, time.Now())
	s.handleEvent(ctx, transport.Event{Kind: transport.Received, Peer: peer, Data: frame}, time.Now())
}

func awaitValue[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for admission worker")
		var zero T
		return zero
	}
}
func TestLateAdmissionCannotAffectReusedPeer(t *testing.T) {
	host := &fakeHost{}
	v := make(delayedVerifier)
	s := NewServer(host, config.Config{MaxPeers: 2, MaxPlayers: 2, JoinTimeoutMS: 10000}, "127.0.0.1:25565", v)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connectAndJoin(s, ctx, 1, 1)
	old := awaitValue(t, v)
	s.handleEvent(ctx, transport.Event{Kind: transport.Disconnected, Peer: 1}, time.Now())
	awaitValue(t, old.ctx.Done())
	connectAndJoin(s, ctx, 1, 2)
	fresh := awaitValue(t, v)
	stale := awaitValue(t, s.results)
	s.complete(stale, time.Now())
	if len(host.sent) != 0 || len(host.disconnected) != 0 || s.sessions.Online(stale.identity.AccountID) {
		t.Fatal("stale verification affected reused peer")
	}
	close(fresh.release)
	current := awaitValue(t, s.results)
	s.complete(current, time.Now())
	if len(host.sent) != 1 || host.sent[0].data[0] != 17 || !s.sessions.Online("02000000-0000-0000-0000-000000000000") {
		t.Fatal("new connection did not receive its own admission")
	}
}

func TestVerificationBackpressureAndShutdown(t *testing.T) {
	host := &fakeHost{}
	v := make(delayedVerifier)
	s := NewServer(host, config.Config{MaxPeers: 12, MaxPlayers: 12, JoinTimeoutMS: 10000}, "127.0.0.1:25565", v)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for i := 0; i < 8; i++ {
		connectAndJoin(s, ctx, uint64(i+1), byte(i+1))
		awaitValue(t, v)
	}
	connectAndJoin(s, ctx, 9, 9)
	if len(host.disconnected) != 1 || host.disconnected[0] != 9 {
		t.Fatalf("saturated admission: disconnects=%v", host.disconnected)
	}
	// Old results can fill the queue while the owner is busy processing network events.
	for i := 0; i < cap(s.results); i++ {
		s.results <- admissionResult{}
	}
	cancel()
	// Each worker must release its slot even if nobody drains the results.
	deadline := time.After(3 * time.Second)
	for i := 0; i < cap(s.verifiers); i++ {
		select {
		case s.verifiers <- struct{}{}:
		case <-deadline:
			t.Fatal("worker retained verification slot after shutdown")
		}
	}
}
