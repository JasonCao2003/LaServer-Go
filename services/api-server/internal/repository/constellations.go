package repository

import "context"

type StatueRecord struct {
	ID, Nickname, Abilities, Hints         string
	Received, Given, Level, WhenCreated    int64
	Blocked, BlockedByFriend, LocalDeleted bool
}
type ConstellationSlot struct {
	Page, Node int64
	FriendID   string
}

func (s *Store) Statues(ctx context.Context, userID string) ([]StatueRecord, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT fs.friend_id,COALESCE(fs.nickname,''),COALESCE(fs.abilities,'[]'),COALESCE(fs.hints,'[]'),COALESCE(fs.recvd,0),COALESCE(fs.given,0),COALESCE(fs.level,1),COALESCE(fs.when_created,0),COALESCE(f.status='blocked',0),COALESCE(r.status='blocked',0),COALESCE(fs.local_soft_deleted,0) FROM friend_status fs JOIN user u ON u.id=fs.friend_id LEFT JOIN friendship f ON f.user_id=fs.user_id AND f.friend_id=fs.friend_id LEFT JOIN friendship r ON r.user_id=fs.friend_id AND r.friend_id=fs.user_id WHERE fs.user_id=? AND COALESCE(f.status,'active')!='removed' AND COALESCE(r.status,'active')!='removed' ORDER BY fs.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StatueRecord{}
	for rows.Next() {
		var f StatueRecord
		if err = rows.Scan(&f.ID, &f.Nickname, &f.Abilities, &f.Hints, &f.Received, &f.Given, &f.Level, &f.WhenCreated, &f.Blocked, &f.BlockedByFriend, &f.LocalDeleted); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
func (s *Store) Constellations(ctx context.Context, userID string) ([]ConstellationSlot, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT page_index,constellation_node_index,friend_id FROM constellation_friend_page WHERE user_id=? AND page_index BETWEEN 0 AND 9 ORDER BY page_index,constellation_node_index,id", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ConstellationSlot{}
	for rows.Next() {
		var slot ConstellationSlot
		if err = rows.Scan(&slot.Page, &slot.Node, &slot.FriendID); err != nil {
			return nil, err
		}
		out = append(out, slot)
	}
	return out, rows.Err()
}
