// 光翼增益收集/存入/转换

package inventory

import (
	"bytes"
	"encoding/json"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/core"
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
	rows, err := a.Store.WingBuffs(r.Context, uid)
	if err != nil {
		return nil, err
	}
	out := make([]Object, 0, len(rows))
	for _, row := range rows {
		out = append(out, Object{"name": row.Name, "collected": row.Collected, "deposited": row.Deposited, "last_conversion": row.LastConversion, "deposit_id": row.DepositID})
	}
	return out, nil
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
	existing, err := tx.PrepareWingConversion(r.Context, uid)
	if err != nil {
		return nil, err
	}
	if existing == 0 {
		for _, v := range a.fullWingRows() {
			name := text(v, "name")
			if name == "" {
				continue
			}
			id := depositMask(v["deposit_id"])
			if err = tx.SeedWingForConversion(r.Context, uid, name, id); err != nil {
				return nil, err
			}
		}
	}
	converted, err := tx.MarkWingsConverted(r.Context, uid, now())
	if err != nil {
		return nil, err
	}
	if converted > 0 {
		if err = tx.CreditWingPrestige(r.Context, uid, converted); err != nil {
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
	currency, err := core.CurrencySnapshotReader(r.Context, a.Store, uid)
	if err != nil {
		return nil, err
	}
	return Object{"result": "ok", "wing_buffs": rows, "converted": converted, "currency": currency}, nil
}
