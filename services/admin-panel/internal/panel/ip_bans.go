package panel

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"net/netip"
	"strings"
)

func (p *panelServer) ipBans(w http.ResponseWriter, r *http.Request) {
	_, session, ok := p.activeSession(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := struct {
		CSRF, Message string
		Entries       []string
		Ready         bool
	}{CSRF: session.csrf}
	payload := map[string]string{"action": "list"}
	if r.Method == http.MethodPost {
		if !p.parsePost(w, r) {
			return
		}
		if subtle.ConstantTimeCompare([]byte(session.csrf), []byte(r.PostForm.Get("csrf"))) != 1 {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		action := r.PostForm.Get("action")
		entry := strings.TrimSpace(r.PostForm.Get("entry"))
		if (action != "add" && action != "remove") || !validIPBanEntry(entry) {
			http.Error(w, "请输入有效的 IPv4、IPv6 或 CIDR 网段。", http.StatusBadRequest)
			return
		}
		payload = map[string]string{"action": action, "entry": entry}
	}
	raw, err := p.callAdmin(r.Context(), "/admin/api/ip_bans", payload)
	var result struct {
		OK      bool
		Entries []string
	}
	if err != nil || json.Unmarshal(raw, &result) != nil || !result.OK {
		data.Message = "无法确认 IP 封禁操作结果，请检查 API 服务和管理令牌，再刷新核对。"
		p.render(w, "ip_bans.html", http.StatusBadGateway, data)
		return
	}
	if r.Method == http.MethodPost {
		http.Redirect(w, r, "/ip-bans", http.StatusSeeOther)
		return
	}
	data.Entries, data.Ready = result.Entries, true
	p.render(w, "ip_bans.html", http.StatusOK, data)
}

func validIPBanEntry(entry string) bool {
	if addr, err := netip.ParseAddr(entry); err == nil {
		return addr.Zone() == ""
	}
	prefix, err := netip.ParsePrefix(entry)
	return err == nil && !prefix.Addr().Is4In6()
}
