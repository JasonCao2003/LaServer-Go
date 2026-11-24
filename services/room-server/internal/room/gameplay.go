// 房间游戏消息与快照转发

package room

import (
	"encoding/binary"

	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/protocol"
	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/protocol/snapshot"
)

type snapshotStream struct {
	reader snapshot.Reader
	writer snapshot.Writer
}

// The UDP version pair, not the APK version, controls wire compatibility.
type levelGroup struct {
	levelID uint32
	version [2]uint16
}

// 返回连接所在关卡分组
func (conn *connection) group() levelGroup {
	return levelGroup{levelID: conn.player.LevelID, version: conn.version}
}

// 处理客户端游戏消息
func (s *Server) receiveGameMessage(source *connection, frame []byte) {
	msg, err := protocol.DecodeGameMessage(frame, protocol.ClientIDSequence)
	if err != nil || msg.LevelChangeCount != source.levelChangeCount {
		return
	}
	switch msg.Type {
	case protocol.GameMsgPlayerState:
		s.receiveSnapshot(source, msg, &source.playerSync.reader)
	case protocol.GameMsgLevelData:
		if level := s.levels[source.group()]; level != nil && level.authority == source {
			s.receiveSnapshot(source, msg, &source.levelSync.reader)
		}
	case protocol.GameMsgSnapshotAck:
		source.playerSync.writer.Ack(msg.Payload[0])
		source.levelSync.writer.Ack(msg.Payload[1])
	case protocol.GameMsgNetRPC, protocol.GameMsgCritters, protocol.GameMsgAffinity, protocol.GameMsgMusicSync:
		msg.SourcePlayer = source.player.NetID
		for _, target := range s.connections {
			if target != source && target.player != nil && target.group() == source.group() {
				s.sendGameMessage(target, msg)
			}
		}
	}
}

// 处理客户端快照消息
func (s *Server) receiveSnapshot(source *connection, msg protocol.GameMessage, reader *snapshot.Reader) {
	frame, err := snapshot.Parse(msg.Payload)
	if err != nil {
		return
	}
	data, err := reader.Read(frame)
	if err != nil {
		return
	}
	if frame.Sequence != 0 {
		s.sendGameMessage(source, protocol.GameMessage{
			Type:    protocol.GameMsgSnapshotAck,
			Payload: []byte{source.playerSync.reader.LatestSequence(), source.levelSync.reader.LatestSequence()},
		})
	}
	if msg.Type == protocol.GameMsgLevelData {
		level := s.levels[source.group()]
		level.cached = data
		level.cacheSource = source.player.NetID
		level.cacheHash = source.levelHash
	} else {
		data = encodePlayerState(source.player.NetID, data)
	}
	for _, target := range s.connections {
		if target != source && target.player != nil && target.group() == source.group() {
			s.sendSnapshot(target, msg.Type, source.player.NetID, data)
		}
	}
}

// 编码玩家状态快照数据
func encodePlayerState(source uint8, data []byte) []byte {
	const headerSize = 5
	out := make([]byte, headerSize+len(data))
	out[0] = source
	binary.LittleEndian.PutUint32(out[1:headerSize], uint32(len(data)))
	copy(out[headerSize:], data)
	return out
}

// 向目标发送快照
func (s *Server) sendSnapshot(target *connection, msgType, source uint8, data []byte) bool {
	if len(data)+protocol.GameSnapshotOverhead > s.maxFrame {
		return false
	}
	writer := &target.levelSync.writer
	if msgType == protocol.GameMsgPlayerState {
		writer = &target.playerSync.writer
	}
	frame, err := writer.Write(data)
	if err != nil {
		return false
	}
	return s.sendGameMessage(target, protocol.GameMessage{Type: msgType, SourcePlayer: source, Payload: frame.Bytes()})
}

// 向目标发送游戏消息
func (s *Server) sendGameMessage(target *connection, msg protocol.GameMessage) bool {
	msg.LevelChangeCount = target.levelChangeCount
	frame, err := protocol.EncodeGameMessage(msg)
	if err != nil || len(frame) > s.maxFrame {
		return false
	}
	return s.host.Send(target.ticket.Peer, 0, false, frame) == nil
}
