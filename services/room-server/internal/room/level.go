package room

import (
	"sort"

	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/protocol"
	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/protocol/snapshot"
)

type levelState struct {
	authority   *connection
	cached      []byte
	cacheSource uint8
	cacheHash   uint16
}

// levelFor retains the first admitted authority until it leaves this level group.
func (s *Server) levelFor(conn *connection) *levelState {
	group := conn.group()
	level := s.levels[group]
	if level == nil {
		level = &levelState{authority: conn}
		s.levels[group] = level
	}
	return level
}

func (s *Server) changeLevel(conn *connection, update protocol.LevelUpdate) {
	previous := conn.group()
	// Legacy level generations are byte counters. Accept wraparound, but an
	// older update must not rewind either the level or its snapshot windows.
	advance := update.LevelChangeCount - conn.levelChangeCount
	if conn.hasLevelUpdate && (advance >= 128 || (advance == 0 && previous.levelID != update.LevelID)) {
		return
	}
	conn.hasLevelUpdate = true
	if previous.levelID != update.LevelID || conn.levelChangeCount != update.LevelChangeCount {
		// The client resets its incoming chain on reload, but its outgoing
		// snapshots may keep referencing frames sent before the reload.
		conn.playerSync.writer = snapshot.Writer{}
		conn.levelSync.writer = snapshot.Writer{}
	}
	conn.levelChangeCount = update.LevelChangeCount
	conn.levelHash = update.NetVersion[2]
	conn.player.LevelID = update.LevelID
	if previous.levelID != update.LevelID {
		s.leaveLevel(previous, conn)
	}
	if !s.sendElection(conn, s.levelFor(conn)) {
		s.drop(conn.ticket.Peer)
		return
	}
	if previous.levelID != update.LevelID {
		s.broadcast(conn.ticket.Peer, protocol.EncodePlayerChangedLevel(conn.player.NetID, update.LevelID))
		s.refreshLevelRoster(conn.group())
	}
}

func (s *Server) leaveLevel(group levelGroup, conn *connection) {
	level := s.levels[group]
	if level == nil || level.authority != conn {
		return
	}
	level.authority = nil
	for _, candidate := range s.connections {
		if candidate.player != nil && candidate.group() == group &&
			(level.authority == nil || candidate.player.NetID < level.authority.player.NetID) {
			level.authority = candidate
		}
	}
	if level.authority == nil {
		delete(s.levels, group)
		return
	}
	if authority := level.authority; !s.sendElection(authority, level) {
		s.drop(authority.ticket.Peer)
		return
	}
	for _, target := range s.connections {
		if target != level.authority && target.player != nil && target.group() == group && !s.sendElection(target, level) {
			s.drop(target.ticket.Peer)
		}
	}
}

func (s *Server) sendElection(target *connection, level *levelState) bool {
	payload := protocol.EncodeLevelElection(level.authority.player.NetID, target.player.LevelID, level.cacheHash)
	if !s.sendSnapshot(target, protocol.GameMsgLevelElection, 0, payload) {
		return false
	}
	if len(level.cached) != 0 && target.player.NetID != level.cacheSource {
		s.sendSnapshot(target, protocol.GameMsgLevelData, level.cacheSource, level.cached)
	}
	return true
}

// The 284/542 client rebuilds its visible roster when it enters a different level.
func (s *Server) refreshLevelRoster(group levelGroup) {
	var members []*connection
	var roster []protocol.RosterPlayer
	for _, conn := range s.connections {
		if conn.player != nil && conn.group() == group {
			members = append(members, conn)
			roster = append(roster, *conn.player)
		}
	}
	sort.Slice(roster, func(i, j int) bool { return roster[i].NetID < roster[j].NetID })
	frame, err := protocol.EncodeEnterGame(protocol.EnterGameModern, roster)
	if err != nil {
		return
	}
	for _, member := range members {
		_ = s.host.Send(member.ticket.Peer, 0, false, frame)
	}
}
