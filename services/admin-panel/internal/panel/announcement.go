package panel

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

func (p *panelServer) announcement(w http.ResponseWriter, r *http.Request) {
	_, session, ok := p.activeSession(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	data := struct {
		CSRF, Title, Body, Buttons, Message string
		Version                             int64
		Ready                               bool
	}{CSRF: session.csrf}
	payload := map[string]string{"action": "read"}
	if r.Method == http.MethodPost {
		if !p.parsePost(w, r) {
			return
		}
		if subtle.ConstantTimeCompare([]byte(r.PostForm.Get("csrf")), []byte(session.csrf)) != 1 {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		payload = map[string]string{"action": "save", "title": r.PostForm.Get("title"), "body": r.PostForm.Get("body"), "version": r.PostForm.Get("version"), "buttons": r.PostForm.Get("buttons")}
	}
	raw, err := p.callAdmin(r.Context(), "/admin/api/motd", payload)
	var result struct {
		OK           bool
		Announcement struct {
			Title   string `json:"motd_title"`
			Body    string `json:"motd"`
			Version int64  `json:"motd_version"`
			Buttons []struct {
				URL string
				Img string
			} `json:"motd_buttons"`
		}
	}
	if err != nil || json.Unmarshal(raw, &result) != nil || !result.OK {
		if r.Method == http.MethodPost {
			http.Error(w, "无法确认发布结果。请返回核对版本、配置和服务状态。", http.StatusBadGateway)
			return
		}
		data.Message = "无法读取公告，请检查 API 服务与管理令牌。"
		p.render(w, "announcement.html", http.StatusBadGateway, data)
		return
	}
	if r.Method == http.MethodPost {
		http.Redirect(w, r, "/announcement", http.StatusSeeOther)
		return
	}
	data.Ready = true
	data.Title, data.Body, data.Version = result.Announcement.Title, result.Announcement.Body, result.Announcement.Version+1
	var lines []string
	for _, b := range result.Announcement.Buttons {
		lines = append(lines, b.URL+"|"+b.Img)
	}
	data.Buttons = strings.Join(lines, "\n")
	p.render(w, "announcement.html", http.StatusOK, data)
}
