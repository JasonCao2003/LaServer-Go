package gameapi

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"
)

var currencyFields = []string{"candles", "heart", "heart_wax", "prestige", "prestige_vip", "prestige_wax", "season_candle", "season_wax", "season_wax_14", "season_heart", "season_pass_token", "vip", "wax"}

func currencyField(k string) bool {
	for _, f := range currencyFields {
		if f == k {
			return true
		}
	}
	return false
}

func (a *API) currency(r *http.Request, m object) (object, error) {
	u, err := a.authenticated(r, m)
	if err != nil {
		return nil, err
	}
	if err = a.dailyBonus(r, u.ID); err != nil {
		return nil, err
	}
	rows, err := a.db.QueryContext(r.Context(), `SELECT candles,heart,heart_wax,prestige,prestige_vip,prestige_wax,season_candle,season_wax,season_wax_14,season_heart,season_pass_token,vip,wax FROM user_currency WHERE user_id=?`, u.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err = rows.Err(); err != nil {
			return nil, err
		}
		return nil, errors.New("missing currency")
	}
	vals := make([]sql.NullInt64, len(currencyFields))
	dest := make([]any, len(vals))
	for i := range vals {
		dest[i] = &vals[i]
	}
	if err = rows.Scan(dest...); err != nil {
		return nil, err
	}
	out := object{}
	for i, k := range currencyFields {
		out[k] = vals[i].Int64
	}
	if a.cfg.CurrentSeason > 0 {
		suffix := strconv.Itoa(a.cfg.CurrentSeason)
		out["season_candle_"+suffix] = out["season_candle"]
		out["season_heart_"+suffix] = out["season_heart"]
	}
	return object{"currency": out}, rows.Err()
}

// dailyBonus commits the grant and its marker together, including concurrent requests.
func (a *API) dailyBonus(r *http.Request, userID string) error {
	zone, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		return err
	}
	day := time.Now().In(zone).Format("2006-01-02")
	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(r.Context(), `INSERT INTO season_pass_daily(user_id,day) SELECT user_id,? FROM user_currency WHERE user_id=? AND vip>0 ON CONFLICT(user_id,day) DO NOTHING`, day, userID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count > 0 {
		if _, err = tx.ExecContext(r.Context(), "UPDATE user_currency SET season_candle=COALESCE(season_candle,0)+1,updated_at=CURRENT_TIMESTAMP WHERE user_id=?", userID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (a *API) userdata(r *http.Request, m object) (object, error) {
	u, err := a.authenticated(r, m)
	if err != nil {
		return nil, err
	}
	if val, ok := m["data"]; ok && val != nil {
		if _, ok = val.(string); !ok {
			return nil, fail(400, "Invalid userdata")
		}
	}
	if data := text(m, "data"); data != "" {
		_, err = a.db.ExecContext(r.Context(), `INSERT INTO user_data(user_id,data,data_time,data_version,updated_at) VALUES(?,?,?,?,?) ON CONFLICT(user_id) DO UPDATE SET data=excluded.data,data_time=excluded.data_time,data_version=excluded.data_version,updated_at=excluded.updated_at`, u.ID, data, number(m, "time", 0), number(m, "version", 3), now())
		if err != nil {
			return nil, err
		}
	}
	var data string
	var stamp, version sql.NullInt64
	err = a.db.QueryRowContext(r.Context(), "SELECT data,data_time,data_version FROM user_data WHERE user_id=?", u.ID).Scan(&data, &stamp, &version)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if data == "" {
		data = defaultUserdata
	}
	t, v := stamp.Int64, version.Int64
	if t == 0 {
		t = 1738138669
	}
	if v == 0 {
		v = 3
	}
	return object{"userdata": object{"time": t, "version": v, "data": data}}, nil
}

func (a *API) rank(r *http.Request, m object) (object, error) {
	u, err := a.authenticated(r, m)
	if err != nil {
		return nil, err
	}
	if _, err = a.db.ExecContext(r.Context(), "INSERT INTO user_rank(user_id) VALUES(?) ON CONFLICT(user_id) DO NOTHING", u.ID); err != nil {
		return nil, err
	}
	var social float64
	var shepherd, beginner bool
	err = a.db.QueryRowContext(r.Context(), "SELECT COALESCE(social_rank,0),COALESCE(is_shepherd,0),COALESCE(is_beginner,0) FROM user_rank WHERE user_id=?", u.ID).Scan(&social, &shepherd, &beginner)
	return object{"set_rank": object{"social": social, "is_shepherd": shepherd, "is_beginner": beginner}}, err
}

func (a *API) discover(r *http.Request, m object) (object, error) {
	if _, err := a.authenticated(r, m); err != nil {
		return nil, err
	}
	uri := fmt.Sprintf("%s:%d", a.cfg.RoomHost, a.cfg.RoomPort)
	sig := func(salt string) string {
		sum := sha256.Sum256([]byte(uri + "|" + salt))
		return hex.EncodeToString(sum[:])
	}
	return object{"uri": uri, "move_ts": now(), "signature": sig("signature"), "sig": sig("sig"), "ps_only": false, "platform_filter": 0}, nil
}

func (a *API) checkpoints(r *http.Request, m object) (object, error) {
	u, err := a.authenticated(r, m)
	if err != nil {
		return nil, err
	}
	write := r.URL.Path == "/account/set_checkpoint"
	if write {
		if err = require(m, "level_id", "level_hash", "bst_id", "states"); err != nil {
			return nil, err
		}
		if _, ok := levelKey(m["level_id"]); !ok {
			return nil, fail(400, "Invalid level_id")
		}
	}
	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var raw string
	err = tx.QueryRowContext(r.Context(), "SELECT checkpoints FROM user_checkpoint WHERE user_id=?", u.ID).Scan(&raw)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	points := []object{}
	if raw != "" {
		decoder := json.NewDecoder(strings.NewReader(raw))
		decoder.UseNumber()
		if err = decoder.Decode(&points); err != nil {
			return nil, errors.New("invalid stored checkpoints")
		}
	}
	if points == nil {
		points = []object{}
	}
	if write {
		key, _ := levelKey(m["level_id"])
		order, known := checkpointOrders[key]
		if !known {
			order = -1
		}
		furthest := int64(-1)
		for _, p := range points {
			if n := number(p, "arc_order", -1); n > furthest {
				furthest = n
			}
		}
		next := object{"user": u.ID, "level_id": m["level_id"], "level_hash": m["level_hash"], "bst_id": m["bst_id"], "states": m["states"], "arc_order": order, "is_furthest": order > furthest}
		rest := []object{}
		existing := false
		for _, p := range points {
			oldKey, _ := levelKey(p["level_id"])
			if oldKey == key {
				existing = true
				next["arc_order"] = p["arc_order"]
				next["is_furthest"] = p["is_furthest"]
			} else {
				rest = append(rest, p)
			}
		}
		if !existing && order > furthest {
			for _, p := range rest {
				p["is_furthest"] = false
			}
		}
		points = append([]object{next}, rest...)
	}
	encoded, err := json.Marshal(points)
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(r.Context(), "INSERT INTO user_checkpoint(user_id,checkpoints) VALUES(?,?) ON CONFLICT(user_id) DO UPDATE SET checkpoints=excluded.checkpoints", u.ID, string(encoded)); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return object{"checkpoints": points}, nil
}

func levelKey(v any) (string, bool) {
	switch x := v.(type) {
	case string:
		return "s:" + x, x != ""
	case json.Number:
		n, err := x.Int64()
		return strconv.FormatInt(n, 10), err == nil
	default:
		return "", false
	}
}
