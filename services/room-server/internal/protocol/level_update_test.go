// 关卡更新消息解析测试

package protocol

import (
	"encoding/binary"
	"testing"
)

func TestDecodeLevelUpdate(t *testing.T) {
	for _, test := range []struct {
		name    string
		layout  ClientLayout
		header  []byte
		trailer []byte
	}{
		{"0.1.8", ClientIDSequence, []byte{4, 9}, nil},
		{"extended", ClientTokenIDSequence, []byte{1, 2, 3, 4, 4, 9}, []byte{42}},
	} {
		t.Run(test.name, func(t *testing.T) {
			payload := make([]byte, 82)
			payload[0], payload[13], payload[18] = 5, 8, 0xaa
			binary.LittleEndian.PutUint16(payload[1:], 12274)
			binary.LittleEndian.PutUint16(payload[11:], 11920)
			binary.LittleEndian.PutUint32(payload[14:], 0x12345678)
			frame := append(append(append([]byte(nil), test.header...), payload...), test.trailer...)
			got, err := DecodeLevelUpdate(frame, test.layout)
			if err != nil {
				t.Fatal(err)
			}
			if got.Sequence != 9 || got.LevelChangeCount != 5 || got.LevelStatus != 8 ||
				got.LevelID != 0x12345678 || got.NetVersion[0] != 12274 ||
				got.NetVersion[5] != 11920 || got.LevelSummary[0] != 0xaa ||
				got.HasTrailingState != (len(test.trailer) != 0) ||
				(got.HasTrailingState && got.TrailingState != 42) {
				t.Fatalf("decoded LevelUpdate = %+v", got)
			}
			if _, err := DecodeLevelUpdate(frame[:len(test.header)+81], test.layout); err == nil {
				t.Error("accepted truncated LevelUpdate")
			}
		})
	}
}
