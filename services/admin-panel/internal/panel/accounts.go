package panel

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type accountRow struct {
	ID, Device, Platform, Version string
	Authorized, Banned            bool
	BanEnd                        int64 `json:"ban_end"`
	Created                       int64
}

func displayAccountTime(stamp int64) string {
	if stamp <= 0 {
		return "—"
	}
	if stamp > 100000000000 {
		stamp /= 1000
	}
	return time.Unix(stamp, 0).Local().Format("2006-01-02 15:04")
}

func (a accountRow) Registration() string { return displayAccountTime(a.Created) }

func (a accountRow) Deadline() string {
	if !a.Banned {
		return "—"
	}
	if a.BanEnd == 0 {
		return "永久"
	}
	return displayAccountTime(a.BanEnd)
}

func (p *panelServer) accounts(w http.ResponseWriter, r *http.Request) {
	_, session, ok := p.activeSession(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	data := struct {
		CSRF, Keyword, Message                                     string
		Total, Banned, Unauthorized, Matched, Page, Previous, Next int
		Accounts                                                   []accountRow
		OK                                                         bool
	}{CSRF: session.csrf, Keyword: strings.TrimSpace(r.URL.Query().Get("keyword")), Page: 1}
	if raw := r.URL.Query().Get("page"); raw != "" {
		data.Page, _ = strconv.Atoi(raw)
	}
	if len(data.Keyword) > 256 || data.Page < 1 || data.Page > 100000 {
		http.Error(w, "查询参数无效", http.StatusBadRequest)
		return
	}
	raw, err := p.callAdmin(r.Context(), "/admin/api/accounts_list", map[string]string{"keyword": data.Keyword, "page": strconv.Itoa(data.Page)})
	var result struct {
		Total, Banned, Unauthorized, Matched int
		Accounts                             []accountRow
		OK                                   bool
	}
	status := http.StatusOK
	if err != nil || json.Unmarshal(raw, &result) != nil || !result.OK {
		data.Message = "暂时无法读取账号，请检查 API 服务及管理令牌。"
		status = http.StatusBadGateway
	} else {
		data.OK = true
		data.Total, data.Banned, data.Unauthorized, data.Matched, data.Accounts = result.Total, result.Banned, result.Unauthorized, result.Matched, result.Accounts
		if data.Page > 1 {
			data.Previous = data.Page - 1
		}
		if data.Page*50 < data.Matched && data.Page < 100000 {
			data.Next = data.Page + 1
		}
	}
	p.render(w, "accounts.html", status, data)
}

func (p *panelServer) changeAccount(w http.ResponseWriter, r *http.Request) {
	if !p.parsePost(w, r) {
		return
	}
	_, session, ok := p.activeSession(r)
	if !ok || subtle.ConstantTimeCompare([]byte(session.csrf), []byte(r.PostForm.Get("csrf"))) != 1 {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	id, action := r.PostForm.Get("user"), r.PostForm.Get("action")
	if !validFriendID(id) || len(r.PostForm["user"]) != 1 || len(r.PostForm["action"]) != 1 {
		http.Error(w, "账号或操作无效", http.StatusBadRequest)
		return
	}
	raw, err := p.callAdmin(r.Context(), "/admin/api/accounts_change", map[string]string{"user": id, "action": action, "seconds": r.PostForm.Get("seconds")})
	var result struct{ OK bool }
	if err != nil || json.Unmarshal(raw, &result) != nil || !result.OK {
		http.Error(w, "未确认操作成功，请返回账号列表刷新后核对状态。", http.StatusBadGateway)
		return
	}
	query := url.Values{"keyword": {id}}
	http.Redirect(w, r, "/accounts?"+query.Encode(), http.StatusSeeOther)
}

type accountBatchPage struct {
	CSRF, Action, Label, Seconds string
	Users                        []string
	Results                      []accountBatchResult
}

type accountBatchResult struct{ User, Status string }

func (p *panelServer) batchAccounts(w http.ResponseWriter, r *http.Request) {
	if !p.parsePost(w, r) {
		return
	}
	_, session, ok := p.activeSession(r)
	if !ok || subtle.ConstantTimeCompare([]byte(session.csrf), []byte(r.PostForm.Get("csrf"))) != 1 {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	data := accountBatchPage{CSRF: session.csrf, Action: r.PostForm.Get("action"), Seconds: r.PostForm.Get("seconds"), Users: r.PostForm["user"]}
	switch data.Action {
	case "authorize":
		data.Label = "批量授权"
	case "revoke":
		data.Label = "批量取消授权"
	case "ban":
		duration, err := strconv.ParseInt(data.Seconds, 10, 64)
		if err == nil && duration >= 0 && duration <= 315360000 {
			if duration == 0 {
				data.Label = "批量永久封号"
			} else {
				data.Label = "批量封号（" + data.Seconds + " 秒）"
			}
		}
	}
	mode := r.PostForm.Get("mode")
	if data.Label == "" || len(data.Users) < 2 || len(data.Users) > 50 || (mode != "preview" && mode != "apply") {
		http.Error(w, "请选择 2 至 50 个账号并指定有效操作。", http.StatusBadRequest)
		return
	}
	seen := make(map[string]bool, len(data.Users))
	for _, id := range data.Users {
		if !validFriendID(id) || id != strings.TrimSpace(id) || seen[id] {
			http.Error(w, "账号列表包含无效或重复的 ID。", http.StatusBadRequest)
			return
		}
		seen[id] = true
	}
	if mode == "apply" {
		for _, id := range data.Users {
			status := "结果待确认，请查询账号状态"
			if r.Context().Err() != nil {
				status = "未处理"
			} else {
				raw, err := p.callAdmin(r.Context(), "/admin/api/accounts_change", map[string]string{"user": id, "action": data.Action, "seconds": data.Seconds})
				var result struct{ OK bool }
				if err == nil && json.Unmarshal(raw, &result) == nil && result.OK {
					status = "已完成"
				}
			}
			data.Results = append(data.Results, accountBatchResult{User: id, Status: status})
		}
	}
	p.render(w, "account_batch.html", http.StatusOK, data)
}
