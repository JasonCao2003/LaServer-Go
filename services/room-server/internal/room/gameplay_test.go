// 房间游戏逻辑测试

package room

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"
	"time"

	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/config"
	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/protocol"
	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/session"
	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/transport"
)

type gameplayFixture struct {
	server *Server
	host   *fakeHost
	now    time.Time
}

func newGameplayFixture(t *testing.T) gameplayFixture {
	t.Helper()
	host := &fakeHost{}
	server := NewServer(host, config.Config{MaxPeers: 4, MaxPlayers: 4, JoinTimeoutMS: 6000, MaxFrameBytes: 16384}, "127.0.0.1:25565", fakeVerifier{})
	now := time.Unix(1, 0)
	for peer, levelID := range []uint32{42, 42, 84} {
		id := uint64(peer + 1)
		server.handleEvent(context.Background(), transport.Event{Kind: transport.Connected, Peer: id}, now)
		conn := server.connections[id]
		if err := server.sessions.StartJoin(conn.ticket, now); err != nil {
			t.Fatal(err)
		}
		server.complete(admissionResult{
			ticket: conn.ticket,
			join:   protocol.JoinFingerprint{ServerVersion: 284, ClientVersion: 542, LevelID: levelID, HasLevelID: true, EnterGame: protocol.EnterGameModern},
			identity: session.VerifiedAdmission{
				AccountID: string(rune('a' + peer)),
				ExpiresAt: now.Add(time.Minute),
			},
		}, now)
	}
	host.sent = nil
	return gameplayFixture{server: server, host: host, now: now}
}

func (f gameplayFixture) receive(peer uint64, frame []byte) {
	f.server.handleEvent(context.Background(), transport.Event{Kind: transport.Received, Peer: peer, Data: frame}, f.now)
}

func (f gameplayFixture) updateLevel(peer uint64, count uint8, level uint32) {
	frame := make([]byte, 84)
	frame[0], frame[2] = 4, count
	binary.LittleEndian.PutUint16(frame[7:9], 0x2345)
	binary.LittleEndian.PutUint32(frame[16:20], level)
	f.receive(peer, frame)
}

func (f gameplayFixture) wantFrames(t *testing.T, want ...sentFrame) {
	t.Helper()
	if len(f.host.sent) != len(want) {
		t.Fatalf("sent %d packets, want %d: %+v", len(f.host.sent), len(want), f.host.sent)
	}
	for i, sent := range f.host.sent {
		if sent.peer != want[i].peer || !bytes.Equal(sent.data, want[i].data) {
			t.Fatalf("packet %d to %d = %x, want to %d = %x", i, sent.peer, sent.data, want[i].peer, want[i].data)
		}
	}
	f.host.sent = nil
}

func TestGameMessageRelayUsesDestinationGenerationAndCurrentLevel(t *testing.T) {
	f := newGameplayFixture(t)
	f.updateLevel(2, 3, 42)
	f.host.sent = nil
	f.receive(1, []byte{14, 10, 2, 1, 0xff, 0x34, 0x44, 1, 2})
	f.wantFrames(t, sentFrame{2, []byte{14, 7, 0, 2, 3, 1, 0x34, 0x44, 1, 2}})
	for _, frame := range [][]byte{
		{14, 10, 2, 0, 0, 1, 2}, // Previous level generation.
		{14, 10, 16, 1, 0, 1},   // Client metrics.
		{14, 10, 20, 1, 0, 1},   // Audience event newer than this profile.
	} {
		f.receive(1, frame)
	}
	f.wantFrames(t)
	f.updateLevel(2, 4, 84)
	f.host.sent = nil
	f.receive(1, []byte{14, 11, 4, 1, 0xff, 0x33})
	f.wantFrames(t)
}

func TestPlayerSnapshotIsAckedAndReencodedForOtherPeers(t *testing.T) {
	f := newGameplayFixture(t)
	f.receive(1, []byte{14, 1, 3, 1, 0xff, 2, 0, 46, 0x43, 0, 0x2c})
	f.wantFrames(t,
		sentFrame{1, []byte{14, 5, 0, 14, 1, 0, 2, 0}},
		sentFrame{2, []byte{14, 14, 0, 3, 1, 1, 2, 0, 16, 1, 3, 0, 0, 0, 0x43, 0, 0x2c}},
	)
	f.receive(1, []byte{14, 2, 3, 1, 0xff, 3, 0, 0, 0x43, 0, 0x2c}) // Bad checksum.
	f.wantFrames(t)
	f.receive(2, []byte{14, 3, 14, 1, 0, 2, 0}) // Acknowledge the recipient's chain.
	f.updateLevel(2, 2, 42)                     // Reload the same map with a new generation.
	f.host.sent = nil
	f.receive(1, []byte{14, 4, 3, 1, 0, 2, 0, 46, 0x43, 0, 0x2c})
	f.wantFrames(t,
		sentFrame{1, []byte{14, 5, 0, 14, 1, 0, 2, 0}},
		sentFrame{2, []byte{14, 14, 0, 3, 2, 1, 2, 0, 16, 1, 3, 0, 0, 0, 0x43, 0, 0x2c}},
	)
}

func TestLevelAuthorityElectsForSameLevelAndTransfersOnDisconnect(t *testing.T) {
	f := newGameplayFixture(t)
	f.updateLevel(1, 1, 42)
	elect := append([]byte{14, 44, 0, 8, 1, 0, 2, 0, 114, 1, 42}, make([]byte, 36)...)
	f.wantFrames(t, sentFrame{1, elect})
	world := []byte{14, 1, 11, 1, 0, 2, 0, 171, 1, 2, 3}
	f.receive(2, world)
	f.wantFrames(t)
	f.receive(1, world)
	f.wantFrames(t,
		sentFrame{1, []byte{14, 5, 0, 14, 1, 0, 0, 2}},
		sentFrame{2, []byte{14, 9, 0, 11, 1, 1, 2, 0, 171, 1, 2, 3}},
	)
	f.updateLevel(1, 1, 42)
	if len(f.host.sent) != 1 || f.host.sent[0].peer != 1 || f.host.sent[0].data[3] != 8 {
		t.Fatalf("publisher should receive an election without its own cached state: %+v", f.host.sent)
	}
	f.host.sent = nil
	f.server.handleEvent(context.Background(), transport.Event{Kind: transport.Disconnected, Peer: 1}, f.now)
	var newAuthority, catchUp bool
	for _, sent := range f.host.sent {
		if len(sent.data) < 10 || sent.data[0] != 14 {
			continue
		}
		if sent.peer != 2 {
			t.Fatalf("cross-level state sent to peer %d", sent.peer)
		}
		if len(sent.data) == 47 && sent.data[3] == 8 && sent.data[9] == 2 &&
			binary.LittleEndian.Uint32(sent.data[20:24]) == 0x2345 {
			newAuthority = true
		}
		if sent.data[3] == 11 && bytes.Equal(sent.data[9:], []byte{1, 2, 3}) {
			catchUp = true
		}
	}
	if !newAuthority || !catchUp {
		t.Fatalf("authority handoff did not send election and cached data: %+v", f.host.sent)
	}
}

func TestLevelChangeRefreshesDestinationRoster(t *testing.T) {
	f := newGameplayFixture(t)
	f.updateLevel(1, 2, 84)
	want := map[uint64]bool{1: true, 3: true}
	for _, sent := range f.host.sent {
		if len(sent.data) == 0 || sent.data[0] != 17 {
			continue
		}
		if !want[sent.peer] {
			t.Fatalf("unexpected roster recipient %d", sent.peer)
		}
		if len(sent.data) != 78 || sent.data[3] != 2 || sent.data[4] != 1 || sent.data[41] != 3 ||
			binary.LittleEndian.Uint32(sent.data[37:41]) != 84 || binary.LittleEndian.Uint32(sent.data[74:78]) != 84 {
			t.Fatalf("destination roster to %d = %x", sent.peer, sent.data)
		}
		delete(want, sent.peer)
	}
	if len(want) != 0 {
		t.Fatalf("destination roster missing for %v", want)
	}
}

func TestPreviousLevelGenerationCannotAdvanceSnapshotAck(t *testing.T) {
	f := newGameplayFixture(t)
	f.receive(1, []byte{14, 1, 3, 1, 0, 2, 0, 46, 0x43, 0, 0x2c})
	f.updateLevel(1, 2, 42)
	f.host.sent = nil
	f.receive(1, []byte{14, 2, 3, 1, 0, 3, 0, 46, 0x43, 0, 0x2c})
	f.wantFrames(t)

	// The fresh generation's delta can still reference the last ACKed old frame.
	f.receive(1, []byte{14, 3, 3, 2, 0, 4, 2, 20, 0, 3})
	if len(f.host.sent) != 2 || f.host.sent[0].peer != 1 ||
		!bytes.Equal(f.host.sent[0].data, []byte{14, 5, 0, 14, 2, 0, 4, 0}) || f.host.sent[1].peer != 2 {
		t.Fatalf("current-generation snapshot did not apply to the acknowledged baseline: %+v", f.host.sent)
	}
}

func TestLevelUpdateIgnoresOldGenerationAfterWrap(t *testing.T) {
	f := newGameplayFixture(t)
	f.updateLevel(1, 255, 42)
	f.updateLevel(1, 0, 84)
	f.host.sent = nil
	f.updateLevel(1, 255, 42)
	f.wantFrames(t)
	if conn := f.server.connections[1]; conn.player.LevelID != 84 || conn.levelChangeCount != 0 {
		t.Fatalf("stale LevelUpdate moved current player: %+v", conn.player)
	}
}

func TestFailedElectionRecoversAuthorityForRemainingPlayers(t *testing.T) {
	f := newGameplayFixture(t)
	f.host.failSend = func(peer uint64, data []byte) bool {
		return peer == 1 && len(data) > 3 && data[0] == 14 && data[3] == 8
	}
	f.updateLevel(1, 1, 42)
	if f.server.sessions.Online("a") || len(f.host.disconnected) != 1 || f.host.disconnected[0] != 1 {
		t.Fatalf("failed election left its recipient online: %v", f.host.disconnected)
	}
	for _, sent := range f.host.sent {
		if sent.peer == 2 && len(sent.data) > 9 && sent.data[0] == 14 && sent.data[3] == 8 && sent.data[9] == 2 {
			return
		}
	}
	t.Fatalf("remaining player was not elected: %+v", f.host.sent)
}

func TestFailedSnapshotDeliveryKeepsPeersAndSourceAck(t *testing.T) {
	f := newGameplayFixture(t)
	failures := 0
	f.host.failSend = func(peer uint64, _ []byte) bool {
		if peer == 2 {
			failures++
			return true
		}
		return false
	}
	f.receive(1, []byte{14, 1, 3, 1, 0xff, 2, 0, 46, 0x43, 0, 0x2c})
	f.wantFrames(t, sentFrame{1, []byte{14, 5, 0, 14, 1, 0, 2, 0}})
	if failures != 1 || len(f.host.disconnected) != 0 {
		t.Fatalf("failed send count=%d disconnects=%v", failures, f.host.disconnected)
	}
	f.host.failSend = nil
	f.receive(2, []byte{14, 3, 14, 1, 0, 2, 0})
	f.wantFrames(t)
	f.receive(1, []byte{14, 10, 2, 1, 0xff, 0x34, 0x44, 1, 2})
	f.wantFrames(t, sentFrame{2, []byte{14, 7, 0, 2, 1, 1, 0x34, 0x44, 1, 2}})
}
