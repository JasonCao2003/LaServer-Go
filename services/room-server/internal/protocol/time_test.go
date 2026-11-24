// 时间同步消息测试

package protocol

import (
	"encoding/hex"
	"testing"
)

func TestDecodeTimePing(t *testing.T) {
	for _, test := range []struct {
		name   string
		layout ClientLayout
		frame  string
		token  uint32
	}{
		{"legacy", ClientIDSequence, "0c073412", 0},
		{"token", ClientTokenIDSequence, "785634120c073412", 0x12345678},
	} {
		t.Run(test.name, func(t *testing.T) {
			frame, err := hex.DecodeString(test.frame)
			if err != nil {
				t.Fatal(err)
			}
			got, err := DecodeTimePing(frame, test.layout)
			if err != nil {
				t.Fatal(err)
			}
			want := TimePing{Token: test.token, Sequence: 7, RequestID: 0x1234}
			if got != want {
				t.Fatalf("ping = %+v, want %+v", got, want)
			}
		})
	}
}

func TestDecodeTimePingRejectsInvalidFrame(t *testing.T) {
	for _, test := range []struct {
		layout ClientLayout
		frame  []byte
	}{
		{ClientIDSequence, []byte{12, 7, 0x34}},
		{ClientIDSequence, []byte{12, 7, 0x34, 0x12, 0}},
		{ClientIDSequence, []byte{14, 7, 0x34, 0x12}},
		{ClientTokenIDSequence, []byte{12, 7, 0x34, 0x12}},
		{ClientLayout(0), []byte{12, 7, 0x34, 0x12}},
	} {
		if _, err := DecodeTimePing(test.frame, test.layout); err == nil {
			t.Errorf("accepted layout %d frame %x", test.layout, test.frame)
		}
	}
}

func TestEncodeTimePong(t *testing.T) {
	got := hex.EncodeToString(EncodeTimePong(0x1234, 1, 2))
	const want = "0d12003412000000000000f03f0000000000000040"
	if got != want {
		t.Fatalf("pong = %s, want %s", got, want)
	}
}
