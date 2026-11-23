package room

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/config"
	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/protocol"
	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/session"
	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/transport"
)

type sentFrame struct {
	peer uint64
	data []byte
}

type fakeHost struct {
	sent         []sentFrame
	disconnected []uint64
}

func (*fakeHost) Poll(time.Duration) (transport.Event, bool, error) {
	return transport.Event{}, false, nil
}

func (h *fakeHost) Send(peer uint64, _ uint8, _ bool, data []byte) error {
	h.sent = append(h.sent, sentFrame{peer, data})
	return nil
}

func (h *fakeHost) Disconnect(peer uint64, _ uint32) {
	h.disconnected = append(h.disconnected, peer)
}

func (*fakeHost) Close() error { return nil }

type fakeVerifier struct{}

func (fakeVerifier) Verify(context.Context, string, protocol.AdmissionProof) (session.VerifiedAdmission, error) {
	return session.VerifiedAdmission{}, nil
}

func TestAdmissionBroadcastsAndDisconnectCleansUp(t *testing.T) {
	host := &fakeHost{}
	server := NewServer(host, config.Config{MaxPeers: 4, MaxPlayers: 4, JoinTimeoutMS: 6000}, "127.0.0.1:25565", fakeVerifier{})
	now := time.Unix(1, 0)
	for _, peer := range []uint64{1, 2} {
		server.handleEvent(context.Background(), transport.Event{Kind: transport.Connected, Peer: peer}, now)
		conn := server.connections[peer]
		if err := server.sessions.StartJoin(conn.ticket, now); err != nil {
			t.Fatal(err)
		}
		server.complete(admissionResult{
			ticket: conn.ticket,
			join:   protocol.JoinFingerprint{LevelID: 42, HasLevelID: true, EnterGame: protocol.EnterGameModern},
			identity: session.VerifiedAdmission{
				AccountID: string(rune('a' + peer)),
				ExpiresAt: now.Add(time.Minute),
			},
		}, now)
	}

	if !server.sessions.Online("b") || !server.sessions.Online("c") {
		t.Fatal("verified players are not online")
	}
	if len(host.sent) != 3 || host.sent[0].peer != 1 || host.sent[1].peer != 2 || host.sent[2].peer != 1 {
		t.Fatalf("unexpected roster delivery: %+v", host.sent)
	}
	if host.sent[0].data[0] != 17 || host.sent[0].data[3] != 1 || host.sent[1].data[3] != 2 {
		t.Fatal("incorrect EnterGame rosters")
	}
	if !bytes.Equal(host.sent[2].data, host.sent[1].data) {
		t.Fatal("existing players must receive the complete EnterGame roster")
	}

	server.handleEvent(context.Background(), transport.Event{Kind: transport.Disconnected, Peer: 2}, now)
	if server.sessions.Online("c") || len(host.sent) != 4 || host.sent[3].data[0] != 7 {
		t.Fatal("disconnect did not remove and announce the member")
	}
}
