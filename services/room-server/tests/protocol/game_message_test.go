// 游戏消息编解码测试

package protocol_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/protocol"
)

func TestGameMessageRPCMatchesLegacyServer(t *testing.T) {
	message, err := protocol.DecodeGameMessage([]byte{14, 0x7b, 2, 7, 0xf0, 0x34, 0x44, 1, 2}, protocol.ClientIDSequence)
	if err != nil {
		t.Fatal(err)
	}
	if message.Type != protocol.GameMsgNetRPC || message.LevelChangeCount != 7 || message.SourcePlayer != 0xf0 {
		t.Fatalf("game message header = %+v", message)
	}
	message.LevelChangeCount, message.SourcePlayer = 3, 1
	frame, err := protocol.EncodeGameMessage(message)
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(frame); got != "0e070002030134440102" {
		t.Fatalf("relayed RPC = %s", got)
	}
}

func TestGameMessageRejectsTruncatedHeaders(t *testing.T) {
	for _, frame := range [][]byte{{14, 1, 2, 1}, {14, 1, 2, 1, 0}, {14, 1, 3, 1, 0, 2, 0}, {14, 1, 14, 1, 0, 2}} {
		if _, err := protocol.DecodeGameMessage(frame, protocol.ClientIDSequence); err == nil {
			t.Errorf("accepted short GameMsg %x", frame)
		}
	}
}

func TestLevelElectionMatchesLegacyPaddedHeader(t *testing.T) {
	want := append([]byte{1, 42}, make([]byte, 36)...)
	if got := protocol.EncodeLevelElection(1, 42, 0); !bytes.Equal(got, want) {
		t.Fatalf("election data = %x, want %x", got, want)
	}
}
