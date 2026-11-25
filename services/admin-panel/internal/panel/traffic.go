package panel

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

type trafficRow struct {
	Time              time.Time
	Request, Response string
}

func (p *panelServer) traffic(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := p.activeSession(r); !ok {
		http.Redirect(w, r, "/login", 303)
		return
	}
	origin := p.cfg.APIURL
	title := "HTTP 请求响应"
	if r.URL.Path == "/traffic/udp" {
		origin = p.cfg.RoomURL
		title = "UDP 收发记录"
	}
	if origin == nil {
		http.Error(w, "服务未配置", 503)
		return
	}
	target := *origin
	target.Path = "/admin/traffic"
	req, err := http.NewRequestWithContext(r.Context(), "GET", target.String(), nil)
	if err != nil {
		http.Error(w, "地址错误", 500)
		return
	}
	req.Header.Set("X-Admin-Token", p.cfg.apiToken)
	resp, err := p.client.Do(req)
	if err != nil {
		http.Error(w, "上游服务不可达", 502)
		return
	}
	defer resp.Body.Close()
	var rows []trafficRow
	if resp.StatusCode != 200 || json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&rows) != nil {
		http.Error(w, "上游记录不可用，请核对管理令牌", 502)
		return
	}
	q := r.URL.Query().Get("q")
	selected := rows[:0]
	for _, row := range rows {
		if strings.Contains(row.Request+" "+row.Response+" "+row.Time.Format(time.RFC3339), q) {
			selected = append(selected, row)
		}
	}
	p.render(w, "traffic.html", 200, struct {
		Title, Q string
		Rows     []trafficRow
	}{title, q, selected})
}
