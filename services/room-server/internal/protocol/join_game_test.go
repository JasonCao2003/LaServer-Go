// 加入游戏指纹解析测试

package protocol

import (
	"encoding/binary"
	"testing"
)

func TestDecodeJoinGameFingerprint(t *testing.T) {
	for _, test := range []struct {
		name           string
		server, client uint16
		build          uint32
		splitPlayers   uint32
		levelOffset    int
		engine         string
		layout         EnterGameLayout
	}{
		{"0.1.8 engine", 284, 542, 354648, 0, 21, "sky0134", EnterGameModern},
		{"split move", 284, 541, 169938, 2, 21, "sky0134", EnterGameModern},
		{"repack", 284, 542, 999999, 0, 21, "sky0134", EnterGameModern},
		{"0.14.0", 286, 544, 172143, 0, 21, "sky0134", EnterGameModern},
		{"0.15.5", 294, 4852, 179644, 0, 21, "sky0155", EnterGameModern},
		{"0.12.2", 275, 516, 160278, 0, 21, "sky0122", EnterGameModern},
		{"0.7.1", 252, 477, 0x23211213, 0, 17, "sky071", EnterGameLegacy},
		{"0.12.x compatible build", 275, 516, 160083, 0, 21, "sky0122", EnterGameModern},
	} {
		t.Run(test.name, func(t *testing.T) {
			frame := joinFixture(test.server, test.client, test.build, test.splitPlayers, test.levelOffset)
			got, err := DecodeJoinGameFingerprint(frame)
			if err != nil {
				t.Fatal(err)
			}
			if got.ServerVersion != test.server || got.ClientVersion != test.client ||
				got.Engine != test.engine || got.EnterGame != test.layout ||
				got.Sequence != 7 || got.UUID[0] != 0xa5 {
				t.Fatalf("decoded join = %+v", got)
			}
			wantBuild := test.engine != "sky071"
			if got.HasBuild != wantBuild || (wantBuild && got.Build != test.build) || (!wantBuild && got.Build != 0) {
				t.Fatalf("build = %d, present = %v", got.Build, got.HasBuild)
			}
			wantLevel := test.engine != "sky0155"
			if got.HasLevelID != wantLevel || (wantLevel && got.LevelID != 0x12345678) {
				t.Fatalf("level ID = %x, present = %v", got.LevelID, got.HasLevelID)
			}
		})
	}
}

func TestDecodeJoinGameFingerprintRejectsMalformedFrames(t *testing.T) {
	valid := joinFixture(284, 542, 354648, 0, 21)
	wrongID := append([]byte(nil), valid...)
	wrongID[0] = 4
	for _, frame := range [][]byte{
		nil, valid[:70], wrongID,
		joinFixture(400, 542, 354648, 0, 21),
		joinFixture(284, 542, 0, 0, 21),
	} {
		if _, err := DecodeJoinGameFingerprint(frame); err == nil {
			t.Errorf("accepted unsupported JoinGame of length %d", len(frame))
		}
	}
	oversizedCount := append([]byte(nil), valid...)
	binary.LittleEndian.PutUint32(oversizedCount[64:], 1<<31)
	if _, err := DecodeJoinGameFingerprint(oversizedCount); err == nil {
		t.Error("accepted oversized split player count")
	}
}

// Synthetic JoinGame follows the version offsets in the old engine profiles.
func joinFixture(server, client uint16, build, split uint32, levelOffset int) []byte {
	const header = 2
	frame := make([]byte, 1931)
	frame[0], frame[1] = 3, 7
	payload := frame[header:]
	payload[0] = 0xa5
	binary.LittleEndian.PutUint32(payload[62:], split)
	start := 82 + int(split)*16
	binary.LittleEndian.PutUint16(payload[start:], server)
	binary.LittleEndian.PutUint16(payload[start+2:], client)
	binary.LittleEndian.PutUint32(payload[start+4:], build)
	binary.LittleEndian.PutUint32(payload[start+levelOffset:], 0x12345678)
	return frame
}
