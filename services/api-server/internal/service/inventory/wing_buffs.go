// 光翼增益收集/存入/转换

package inventory

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

// 解析光翼接口用户
func (a *Service) wingUser(r Request, m Object) (string, error) { return a.MagicUser(r, m, true) }

// 提取光翼名称
func wingNames(m Object) []string {
	result := []string{}
	if raw, ok := m["names"].([]any); ok {
		for _, v := range raw {
			if s, ok := v.(string); ok && s != "" {
				result = append(result, s)
			}
		}
	}
	if len(result) == 0 && text(m, "name") != "" {
		result = append(result, text(m, "name"))
	}
	return result
}

// 解析存入掩码
func depositMask(v any) int64 {
	n, err := strconv.ParseUint(stringifyNumber(v), 10, 64)
	if err != nil {
		return defInt(v) & 0x7fffffffffffffff
	}
	return int64(n & 0x7fffffffffffffff)
}

// 把数字转为字符串
func stringifyNumber(v any) string {
	switch n := v.(type) {
	case json.Number:
		return n.String()
	case string:
		return n
	}
	return ""
}

// 读取光翼数据
func (a *Service) wingRows(r Request, uid string) ([]Object, error) {
	rows, err := a.Store.QueryContext(r.Context, repository.ReadWingBuffs, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Object{}
	for rows.Next() {
		var name string
		var collected, deposited bool
		var converted, depositID int64
		if err = rows.Scan(&name, &collected, &deposited, &converted, &depositID); err != nil {
			return nil, err
		}
		out = append(out, Object{"name": name, "collected": collected, "deposited": deposited, "last_conversion": converted, "deposit_id": depositID})
	}
	return out, rows.Err()
}

// 读取完整光翼数据
func (a *Service) fullWingRows() []Object {
	if a.Cfg.GameConfigDir == "" {
		return nil
	}
	raw, err := os.ReadFile(filepath.Join(a.Cfg.GameConfigDir, "all_wing_buffs.json"))
	if err != nil {
		return nil
	}
	var envelope struct {
		WingBuffs []Object `json:"wing_buffs"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&envelope) != nil {
		return nil
	}
	return envelope.WingBuffs
}

// 获取光翼增益
func (a *Service) GetWingBuffs(r Request, m Object) (Object, error) {
	uid, err := a.wingUser(r, m)
	if err != nil {
		return nil, err
	}
	stored, err := a.wingRows(r, uid)
	if err != nil {
		return nil, err
	}
	u, err := a.FindUser(r.Context, "id", uid)
	if err != nil {
		return nil, err
	}
	full, err := a.Policy("auto_full_unlock.json", integer(u.CreatedAt), false)
	if err != nil {
		return nil, err
	}
	out := []Object{}
	positions := map[string]int{}
	if full {
		for _, v := range a.fullWingRows() {
			name := text(v, "name")
			if name == "" {
				continue
			}
			positions[name] = len(out)
			out = append(out, v)
		}
	}
	for _, v := range stored {
		name := text(v, "name")
		if i, ok := positions[name]; ok {
			out[i] = v
		} else {
			positions[name] = len(out)
			out = append(out, v)
		}
	}
	return Object{"result": "ok", "wing_buffs": out}, nil
}

// 收集光翼增益
func (a *Service) CollectWingBuffs(r Request, m Object) (Object, error) {
	return a.wingChange(r, m, true)
}

// 丢弃光翼增益
func (a *Service) DropWingBuffs(r Request, m Object) (Object, error) {
	return a.wingChange(r, m, false)
}

// 变更光翼数据
func (a *Service) wingChange(r Request, m Object, collect bool) (Object, error) {
	uid, err := a.wingUser(r, m)
	if err != nil {
		return nil, err
	}
	names := wingNames(m)
	if len(names) == 0 {
		return nil, fail(400, "no names")
	}
	tx, err := a.Store.BeginTx(r.Context, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	updates := []Object{}
	query := repository.DropWingBuff
	if collect {
		query = repository.CollectWingBuff
	}
	for _, name := range names {
		if strings.TrimSpace(name) == "" {
			continue
		}
		if _, err = tx.ExecContext(r.Context, query, uid, name); err != nil {
			return nil, err
		}
		updates = append(updates, Object{"name": name, "collected": collect})
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return Object{"result": "ok", "update_wing_buffs": updates}, nil
}

// 存入光翼增益
func (a *Service) DepositWingBuffs(r Request, m Object) (Object, error) {
	uid, err := a.wingUser(r, m)
	if err != nil {
		return nil, err
	}
	pairs, ok := m["name_deposit_id_pairs"].([]any)
	if !ok || len(pairs) == 0 {
		return nil, fail(400, "no pairs")
	}
	tx, err := a.Store.BeginTx(r.Context, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	updates := []Object{}
	for _, v := range pairs {
		pair, ok := v.([]any)
		if !ok || len(pair) < 2 {
			continue
		}
		name, ok := pair[0].(string)
		if !ok || name == "" {
			continue
		}
		id := number(Object{"id": pair[1]}, "id", 0) & 0x7fffffffffffffff
		if _, err = tx.ExecContext(r.Context, repository.DepositWingBuff, uid, name, id); err != nil {
			return nil, err
		}
		updates = append(updates, Object{"name": name, "collected": false, "deposited": true, "deposit_id": id})
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return Object{"result": "ok", "update_wing_buffs": updates}, nil
}

// 转换光翼增益
func (a *Service) ConvertWingBuffs(r Request, m Object) (Object, error) {
	uid, err := a.wingUser(r, m)
	if err != nil {
		return nil, err
	}
	tx, err := a.Store.BeginTx(r.Context, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(r.Context, repository.EnsureWingCurrency, uid); err != nil {
		return nil, err
	}
	var existing int64
	if err = tx.QueryRowContext(r.Context, repository.CountWingBuffs, uid).Scan(&existing); err != nil {
		return nil, err
	}
	if existing == 0 {
		for _, v := range a.fullWingRows() {
			name := text(v, "name")
			if name == "" {
				continue
			}
			id := depositMask(v["deposit_id"])
			if _, err = tx.ExecContext(r.Context, repository.SeedWingBuff, uid, name, id); err != nil {
				return nil, err
			}
		}
	}
	res, err := tx.ExecContext(r.Context, repository.ConvertWingBuffs, now(), uid)
	if err != nil {
		return nil, err
	}
	converted, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if converted > 0 {
		if _, err = tx.ExecContext(r.Context, repository.GrantPrestige, converted, uid); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	rows, err := a.wingRows(r, uid)
	if err != nil {
		return nil, err
	}
	values := make([]sql.NullInt64, len(currencyFields))
	dest := make([]any, len(values))
	for i := range values {
		dest[i] = &values[i]
	}
	if err = a.Store.QueryRowContext(r.Context, repository.ReadCurrency, uid).Scan(dest...); err != nil {
		return nil, err
	}
	currency := Object{}
	for i, k := range currencyFields {
		currency[k] = values[i].Int64
	}
	return Object{"result": "ok", "wing_buffs": rows, "converted": converted, "currency": currency}, nil
}
