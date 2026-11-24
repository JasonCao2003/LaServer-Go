// 进房消息编码测试

package protocol

import (
	"encoding/hex"
	"testing"
)

func TestEncodeEnterGame(t *testing.T) {
	player := RosterPlayer{
		NetID:   7,
		UUID:    [16]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		Name:    "abcdef012345678夜",
		LevelID: 0x12345678,
	}
	for _, test := range []struct {
		name   string
		layout EnterGameLayout
		want   string
	}{
		{"legacy", EnterGameLegacy, "05270000010700112233445566778899aabbccddeeff616263646566303132333435363738e578563412"},
		{"sky0134", EnterGameModern, "112600010700112233445566778899aabbccddeeff616263646566303132333435363738e578563412"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := EncodeEnterGame(test.layout, []RosterPlayer{player})
			if err != nil {
				t.Fatal(err)
			}
			if encoded := hex.EncodeToString(got); encoded != test.want {
				t.Fatalf("frame = %s, want %s", encoded, test.want)
			}
		})
	}
}

func TestEncodeEnterGameRosterBounds(t *testing.T) {
	frame, err := EncodeEnterGame(EnterGameModern, nil)
	if err != nil || hex.EncodeToString(frame) != "11010000" {
		t.Fatalf("empty roster = %x, %v", frame, err)
	}

	frame, err = EncodeEnterGame(EnterGameModern, make([]RosterPlayer, 255))
	if err != nil || len(frame) != 3+1+255*37 || frame[3] != 255 {
		t.Fatalf("maximum roster length = %d, error = %v", len(frame), err)
	}
	if _, err := EncodeEnterGame(EnterGameModern, make([]RosterPlayer, 256)); err == nil {
		t.Error("accepted overflowing roster")
	}
	if _, err := EncodeEnterGame(0, nil); err == nil {
		t.Error("accepted unknown EnterGame layout")
	}
}
