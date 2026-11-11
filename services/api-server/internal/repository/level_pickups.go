package repository

import (
	"context"
	"encoding/json"
)

type PickupEmitter struct {
	Name   string           `json:"emitter_name"`
	Events []map[string]any `json:"events"`
}

func (s *Store) LevelPickupEmitters(ctx context.Context, userID string, level int64, defaults []PickupEmitter) ([]PickupEmitter, error) {
	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var count int
	if err = tx.QueryRowContext(ctx, Statement(`SELECT count(*) FROM level_pickup WHERE user_id=? AND level=?`), userID, level).Scan(&count); err != nil {
		return nil, err
	}
	if count == 0 {
		for _, emitter := range defaults {
			for _, event := range emitter.Events {
				likelihood, _ := json.Marshal(event["chunk_size_likelihood"])
				amount, _ := json.Marshal(event["chunk_size_amount"])
				_, err = tx.ExecContext(ctx, Statement(`INSERT INTO level_pickup(user_id,level,emitter_name,event_name,preactive_duration,pool_name,pool_amount,last_chunk_seq,emit_interval,currency_type,collected,chunks_per_emit,chunk_size_likelihood,chunk_size_amount,active_duration) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`), userID, level, emitter.Name, event["event_name"], event["preactive_duration"], event["pool_name"], event["pool_amount"], event["last_chunk_seq"], event["emit_interval"], event["currency_type"], event["collected"], event["chunks_per_emit"], string(likelihood), string(amount), event["active_duration"])
				if err != nil {
					return nil, err
				}
			}
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	rows, err := s.QueryContext(ctx, Statement(`SELECT emitter_name,event_name,COALESCE(preactive_duration,0),pool_name,COALESCE(pool_amount,0),COALESCE(last_chunk_seq,-1),COALESCE(emit_interval,0),currency_type,COALESCE(collected,0),COALESCE(chunks_per_emit,0),COALESCE(chunk_size_likelihood,'[1.0,0.0,0.0]'),COALESCE(chunk_size_amount,'[1,0,0]'),COALESCE(active_duration,0) FROM level_pickup WHERE user_id=? AND level=? ORDER BY id`), userID, level)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PickupEmitter{}
	positions := map[string]int{}
	for rows.Next() {
		var emitter, eventName, poolName, likelihoodRaw, amountRaw string
		var currency *string
		var pre, pool, last, interval, collected, chunks, active int64
		if err = rows.Scan(&emitter, &eventName, &pre, &poolName, &pool, &last, &interval, &currency, &collected, &chunks, &likelihoodRaw, &amountRaw, &active); err != nil {
			return nil, err
		}
		var likelihood, amount any
		_ = json.Unmarshal([]byte(likelihoodRaw), &likelihood)
		_ = json.Unmarshal([]byte(amountRaw), &amount)
		position, ok := positions[emitter]
		if !ok {
			position = len(out)
			positions[emitter] = position
			out = append(out, PickupEmitter{Name: emitter, Events: []map[string]any{}})
		}
		out[position].Events = append(out[position].Events, map[string]any{"preactive_duration": pre, "pool_name": poolName, "pool_amount": pool, "last_chunk_seq": last, "event_name": eventName, "emit_interval": interval, "currency_type": currency, "collected": collected, "chunks_per_emit": chunks, "chunk_size_likelihood": likelihood, "chunk_size_amount": amount, "active_duration": active})
	}
	return out, rows.Err()
}
