// 玩家通知编解码测试

package protocol_test

import (
	"encoding/hex"
	"testing"

	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/protocol"
)

func TestPlayerNotificationFrames(t *testing.T) {
	tests := []struct {
		name string
		got  []byte
		want string
	}{
		{"joined", protocol.EncodePlayerJoined(protocol.RosterPlayer{NetID: 7, Name: "Sky", LevelID: 0x12345678}), "0625000700000000000000000000000000000000536b790000000000000000000000000078563412"},
		{"left", protocol.EncodePlayerLeft(7, 2), "0702000702"},
		{"changed level", protocol.EncodePlayerChangedLevel(7, 0x12345678), "0805000778563412"},
	}
	for _, test := range tests {
		if got := hex.EncodeToString(test.got); got != test.want {
			t.Errorf("%s = %s, want %s", test.name, got, test.want)
		}
	}
}

func TestDecodeMoveControl(t *testing.T) {
	for _, test := range []struct {
		id     byte
		decode func([]byte, protocol.ClientLayout) (protocol.MoveControl, error)
	}{
		{10, protocol.DecodeMoveResult},
		{11, protocol.DecodeCancelMove},
	} {
		got, err := test.decode([]byte{test.id, 9, 3, 4, 5}, protocol.ClientIDSequence)
		if err != nil || got.Sequence != 9 || got.Code != 3 || hex.EncodeToString(got.Extra) != "0405" {
			t.Fatalf("packet %d = %+v, %v", test.id, got, err)
		}
		if _, err := test.decode([]byte{test.id, 9}, protocol.ClientIDSequence); err == nil {
			t.Errorf("packet %d accepted missing code", test.id)
		}
	}
}
