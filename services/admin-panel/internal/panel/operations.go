package panel

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"net/url"
)

var operationSections = []struct{ Key, Title string }{
	{"events", "活动控制"}, {"tcandles", "烛火控制"}, {"season", "季节设置"}, {"daily_quests", "每日任务池"}, {"season_quests", "任务奖励"}, {"spirit_shops", "兑换树配置"},
}

func (p *panelServer) operations(w http.ResponseWriter, r *http.Request) {
	_, session, ok := p.activeSession(r)
	if !ok {
		http.Redirect(w, r, "/login", 303)
		return
	}
	section := r.URL.Query().Get("section")
	if section == "" {
		section = "events"
	}
	title := ""
	for _, item := range operationSections {
		if item.Key == section {
			title = item.Title
			break
		}
	}
	if title == "" {
		http.NotFound(w, r)
		return
	}
	data := struct {
		CSRF, Key, Title, Message, Value, Revision string
		Sections                                   []struct{ Key, Title string }
		OK                                         bool
	}{CSRF: session.csrf, Key: section, Title: title, Sections: operationSections}
	payload := map[string]string{"action": "read", "section": section}
	if r.Method == http.MethodPost {
		if !p.parseLimitedPost(w, r, 1<<20) {
			return
		}
		if subtle.ConstantTimeCompare([]byte(session.csrf), []byte(r.PostForm.Get("csrf"))) != 1 {
			http.Error(w, "Forbidden", 403)
			return
		}
		payload["action"], payload["value"], payload["revision"] = "save", r.PostForm.Get("value"), r.PostForm.Get("revision")
		data.Value, data.Revision = payload["value"], payload["revision"]
	}
	raw, err := p.callAdmin(r.Context(), "/admin/api/operations", payload)
	var result struct {
		OK              bool
		Value, Revision string
	}
	if err != nil || json.Unmarshal(raw, &result) != nil || !result.OK {
		data.OK = r.Method == http.MethodPost
		data.Message = "操作未确认，请检查 JSON 配置、版本及 API 服务状态。刷新可读取服务器当前版本。"
		p.render(w, "operations.html", 502, data)
		return
	}
	if r.Method == http.MethodPost {
		http.Redirect(w, r, "/operations?section="+url.QueryEscape(section), 303)
		return
	}
	data.OK, data.Value, data.Revision = true, result.Value, result.Revision
	p.render(w, "operations.html", 200, data)
}
