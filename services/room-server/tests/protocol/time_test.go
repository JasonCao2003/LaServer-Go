// 时间同步消息测试

package protocol_test

import (
	"encoding/hex"
	"testing"

	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/protocol"
)

func TestDecodeTimePing(t *testing.T) {
	for _, test := range []struct {
		name   string
		layout protocol.ClientLayout
		frame  string
		token  uint32
	}{
		{"legacy", protocol.ClientIDSequence, "0c073412", 0},
		{"token", protocol.ClientTokenIDSequence, "785634120c073412", 0x12345678},
	} {
		t.Run(test.name, func(t *testing.T) {
			frame, err := hex.DecodeString(test.frame)
			if err != nil {
				t.Fatal(err)
			}
			got, err := protocol.DecodeTimePing(frame, test.layout)
			if err != nil {
				t.Fatal(err)
			}
			want := protocol.TimePing{Token: test.token, Sequence: 7, RequestID: 0x1234}
			if got != want {
				t.Fatalf("ping = %+v, want %+v", got, want)
			}
		})
	}
}

func TestDecodeTimePingRejectsInvalidFrame(t *testing.T) {
	for _, test := range []struct {
		layout protocol.ClientLayout
		frame  []byte
	}{
		{protocol.ClientIDSequence, []byte{12, 7, 0x34}},
		{protocol.ClientIDSequence, []byte{12, 7, 0x34, 0x12, 0}},
		{protocol.ClientIDSequence, []byte{14, 7, 0x34, 0x12}},
		{protocol.ClientTokenIDSequence, []byte{12, 7, 0x34, 0x12}},
		{protocol.ClientLayout(0), []byte{12, 7, 0x34, 0x12}},
	} {
		if _, err := protocol.DecodeTimePing(test.frame, test.layout); err == nil {
			t.Errorf("accepted layout %d frame %x", test.layout, test.frame)
		}
	}
}

func TestEncodeTimePong(t *testing.T) {
	got := hex.EncodeToString(protocol.EncodeTimePong(0x1234, 1, 2))
	const want = "0d12003412000000000000f03f0000000000000040"
	if got != want {
		t.Fatalf("pong = %s, want %s", got, want)
	}
}
