package panel

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

type friendRow struct {
	ID            string `json:"friend_id"`
	Nickname      string `json:"nickname"`
	Remark        string `json:"remark"`
	Status        string `json:"status"`
	ReverseStatus string `json:"reverse_status"`
	TwoWay        bool   `json:"two_way"`
}

type friendResult struct {
	OK           *bool       `json:"ok"`
	Error        string      `json:"error"`
	Detail       string      `json:"detail"`
	Friends      []friendRow `json:"friends"`
	RemovedCount *int        `json:"removed_count"`
	Failed       []string    `json:"failed"`
}

type friendsPage struct {
	CSRF, User, Keyword, Message, RefreshURL string
	Result                                   friendResult
}

const maxFriendField = 256
const maxFriendResponse = 2 << 20

func (p *panelServer) friendsList(w http.ResponseWriter, r *http.Request) {
	_, s, ok := p.activeSession(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	query := r.URL.Query()
	data := friendsPage{CSRF: s.csrf, User: strings.TrimSpace(query.Get("user")), Keyword: strings.TrimSpace(query.Get("keyword"))}
	if len(data.User) > maxFriendField || len(data.Keyword) > maxFriendField {
		http.Error(w, "查询字段太长", http.StatusBadRequest)
		return
	}
	if data.User != "" {
		p.loadFriendResult(r.Context(), &data, "list", map[string]string{"user": data.User, "keyword": data.Keyword})
	}
	p.render(w, "friends.html", http.StatusOK, data)
}

func (p *panelServer) friendsSet(w http.ResponseWriter, r *http.Request) {
	p.changeFriends(w, r, "set")
}
func (p *panelServer) friendsClear(w http.ResponseWriter, r *http.Request) {
	p.changeFriends(w, r, "clear")
}

func (p *panelServer) changeFriends(w http.ResponseWriter, r *http.Request, operation string) {
	if !p.parsePost(w, r) {
		return
	}
	_, s, ok := p.activeSession(r)
	if !ok || subtle.ConstantTimeCompare([]byte(r.PostForm.Get("csrf")), []byte(s.csrf)) != 1 {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	user := strings.TrimSpace(r.PostForm.Get("user"))
	if !validFriendID(user) || len(r.PostForm["user"]) != 1 {
		http.Error(w, "请输入有效的用户 ID", http.StatusBadRequest)
		return
	}
	payload := map[string]string{"user": user}
	if operation == "set" {
		friend := strings.TrimSpace(r.PostForm.Get("friend"))
		action := r.PostForm.Get("action")
		if !validFriendID(friend) || friend == user || len(r.PostForm["friend"]) != 1 || len(r.PostForm["action"]) != 1 ||
			(action != "add" && action != "remove" && action != "restore") {
			http.Error(w, "好友 ID 或操作无效", http.StatusBadRequest)
			return
		}
		payload["friend"], payload["action"] = friend, action
	} else if len(r.PostForm["confirm"]) != 1 || strings.TrimSpace(r.PostForm.Get("confirm")) != user {
		http.Error(w, "确认 ID 必须与目标用户一致", http.StatusBadRequest)
		return
	}
	requestID, err := randomToken()
	if err != nil {
		http.Error(w, "无法处理操作", http.StatusInternalServerError)
		return
	}
	w.Header().Set("X-Request-ID", requestID)
	data := friendsPage{CSRF: s.csrf, User: user, RefreshURL: "/friends?" + url.Values{"user": {user}}.Encode()}
	outcome := p.loadFriendResult(r.Context(), &data, operation, payload)
	slog.Info("friend management", "request_id", requestID, "action", operation, "result", outcome)
	p.render(w, "friends.html", http.StatusOK, data)
}

func validFriendID(value string) bool { return value != "" && len(value) <= maxFriendField }

func (p *panelServer) loadFriendResult(ctx context.Context, data *friendsPage, op string, payload map[string]string) string {
	result, err := p.callFriends(ctx, op, payload)
	if err != nil {
		data.Message = "无法读取 API 结果，请检查服务状态。"
		if op != "list" {
			data.Message = "操作结果待确认，请重新查询好友列表。"
		}
		return "unconfirmed"
	}
	data.Result = result
	if !*result.OK {
		data.Message = "操作失败或部分成功。"
		return "rejected"
	}
	data.Message = "操作成功。"
	return "ok"
}

func (p *panelServer) callFriends(ctx context.Context, op string, payload map[string]string) (friendResult, error) {
	if p.cfg.APIURL == nil || p.cfg.apiToken == "" {
		return friendResult{}, fmt.Errorf("missing API configuration")
	}
	target := *p.cfg.APIURL
	target.Path = "/admin/api/friends_" + op
	body, err := json.Marshal(payload)
	if err != nil {
		return friendResult{}, err
	}
	// A non-rewindable body prevents transport retries after a connection failure.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), io.NopCloser(bytes.NewReader(body)))
	if err != nil {
		return friendResult{}, err
	}
	req.ContentLength = int64(len(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Token", p.cfg.apiToken)
	resp, err := p.client.Do(req)
	if err != nil {
		return friendResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return friendResult{}, fmt.Errorf("API status %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxFriendResponse+1))
	if err != nil {
		return friendResult{}, err
	}
	if len(raw) > maxFriendResponse {
		return friendResult{}, fmt.Errorf("API response too large")
	}
	var result friendResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return friendResult{}, err
	}
	if result.OK == nil {
		return friendResult{}, fmt.Errorf("API response missing result")
	}
	return result, nil
}
