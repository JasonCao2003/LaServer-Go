package panel

import (
	"crypto/subtle"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

func (p *panelServer) catalog(w http.ResponseWriter, r *http.Request) {
	_, session, ok := p.activeSession(r)
	if !ok {
		http.Redirect(w, r, "/login", 303)
		return
	}
	kind := strings.TrimPrefix(r.URL.Path, "/")
	titles := map[string]string{"magic": "魔法控制", "ugc": "纸船 / 留影 / 空间", "other": "其他"}
	data := struct {
		Title, Kind, Keyword, CSRF string
		Rows                       []struct{ ID, Name, Value, Detail string }
	}{Title: titles[kind], Kind: kind, CSRF: session.csrf, Keyword: r.URL.Query().Get("keyword")}
	payload := map[string]string{"kind": kind, "keyword": data.Keyword, "action": "list"}
	if r.Method == http.MethodPost {
		if !p.parseLimitedPost(w, r, 32768) {
			return
		}
		if subtle.ConstantTimeCompare([]byte(session.csrf), []byte(r.PostForm.Get("csrf"))) != 1 {
			http.Error(w, "Forbidden", 403)
			return
		}
		op := r.PostForm.Get("kind")
		if !(kind == "magic" && (op == "magic" || op == "magic_delete") || kind == "ugc" && (op == "feed" || op == "geonote") || kind == "other" && strings.Contains("|wing|hair|hat|prop|neck|horn|feet|face|body|mask|arms|", "|"+op+"|")) {
			http.Error(w, "无效操作", 400)
			return
		}
		payload["action"], payload["kind"], payload["id"], payload["value"] = "save", op, r.PostForm.Get("id"), r.PostForm.Get("value")
	}
	raw, err := p.callAdmin(r.Context(), "/admin/api/catalog", payload)
	var result struct {
		OK   bool
		Rows []struct{ ID, Name, Value, Detail string }
	}
	if err != nil || json.Unmarshal(raw, &result) != nil || !result.OK {
		http.Error(w, "操作未确认，请检查目标 ID、输入值及 API 状态。", 502)
		return
	}
	if r.Method == http.MethodPost {
		http.Redirect(w, r, "/"+kind+"?keyword="+url.QueryEscape(data.Keyword), 303)
		return
	}
	if kind == "ugc" {
		switch r.URL.Query().Get("format") {
		case "json":
			w.Header().Set("Content-Disposition", "attachment; filename=ugc.json")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result.Rows)
			return
		case "csv":
			w.Header().Set("Content-Disposition", "attachment; filename=ugc.csv")
			w.Header().Set("Content-Type", "text/csv; charset=utf-8")
			writer := csv.NewWriter(w)
			writer.Write([]string{"ID", "类型", "内容", "地图"})
			for _, row := range result.Rows {
				cells := []string{row.ID, row.Name, row.Value, row.Detail}
				for i, c := range cells {
					if strings.HasPrefix(strings.TrimLeft(c, " \t\r\n"), "=") || strings.HasPrefix(c, "+") || strings.HasPrefix(c, "-") || strings.HasPrefix(c, "@") {
						cells[i] = "'" + c
					}
				}
				writer.Write(cells)
			}
			writer.Flush()
			return
		}
	}
	data.Rows = result.Rows
	p.render(w, "catalog.html", 200, data)
}
