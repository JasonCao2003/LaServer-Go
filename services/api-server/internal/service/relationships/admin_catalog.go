package relationships

import (
	"crypto/subtle"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (a *Service) AdminCatalog(r Request, m Object) (Object, error) {
	if a.Cfg.AdminToken == "" || subtle.ConstantTimeCompare([]byte(a.Cfg.AdminToken), []byte(r.Header.Get("X-Admin-Token"))) != 1 {
		return nil, adminForbidden()
	}
	kind, user, id, value := text(m, "kind"), text(m, "keyword"), text(m, "id"), text(m, "value")
	bad := &Error{Status: 400, Body: Object{"error": "invalid parameters"}}
	if len(user) > 256 || len(id) > 256 || len(value) > 4096 {
		return nil, bad
	}
	if text(m, "action") == "list" {
		if kind != "magic" && kind != "ugc" && kind != "other" {
			return nil, bad
		}
		rows, err := a.Store.PanelRows(r.Context, kind, user)
		if err == nil && kind == "ugc" {
			entries, e := os.ReadDir(filepath.Join(a.Cfg.InstanceDir, "recordings"))
			if e != nil && !os.IsNotExist(e) {
				return nil, e
			}
			seen := map[string]bool{}
			for _, entry := range entries {
				ext := filepath.Ext(entry.Name())
				id := strings.TrimSuffix(entry.Name(), ext)
				if entry.Type().IsRegular() && (ext == ".ndjson" || ext == ".json") && !seen[id] && strings.Contains(id, user) {
					rows = append(rows, repository.PanelRow{ID: id, Name: "留影文件", Value: "仅供查看"})
					seen[id] = true
					if len(seen) >= 100 {
						break
					}
				}
			}
		}
		return Object{"ok": err == nil, "rows": rows}, err
	}
	if text(m, "action") != "save" {
		return nil, bad
	}
	var num int64
	switch kind {
	case "magic_delete":
		if user == "" || id == "" {
			return nil, bad
		}
	case "magic", "wing", "hair", "hat", "prop", "neck", "horn", "feet", "face", "body", "mask", "arms":
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil || n < 0 || n > 4294967295 || user == "" {
			return nil, bad
		}
		num = n
		if kind == "magic" {
			v, err := strconv.ParseInt(id, 10, 64)
			if err != nil || v == 0 || v < -2147483648 || v > 4294967295 || n > 2147483647 {
				return nil, bad
			}
		}
	case "feed", "geonote":
		if id == "" {
			return nil, bad
		}
	default:
		return nil, bad
	}
	changed, err := a.Store.PanelEdit(r.Context, kind, user, id, value, num)
	if err != nil {
		return nil, err
	}
	if !changed {
		return nil, &Error{Status: 404, Body: Object{"error": "not found"}}
	}
	return Object{"ok": true}, nil
}
