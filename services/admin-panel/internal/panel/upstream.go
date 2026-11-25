package panel

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
)

type probeStatus struct{ Name, State string }

func (p *panelServer) overview(w http.ResponseWriter, r *http.Request) {
	_, s, ok := p.activeSession(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	probes := []struct {
		name    string
		origin  *url.URL
		path    string
		success int
	}{
		{"API 服务", p.cfg.APIURL, "/healthz", http.StatusOK},
		{"房间服务", p.cfg.RoomURL, "/healthz", http.StatusNoContent},
		{"房间就绪", p.cfg.RoomURL, "/readyz", http.StatusNoContent},
	}
	results := make([]probeStatus, len(probes))
	done := make(chan struct{}, len(probes))
	for i, probe := range probes {
		go func() {
			results[i] = probeStatus{probe.name, p.probe(r.Context(), probe.origin, probe.path, probe.success)}
			done <- struct{}{}
		}()
	}
	for range probes {
		<-done
	}
	p.render(w, "overview.html", http.StatusOK, struct {
		CSRF          string
		PasswordLogin bool
		Statuses      []probeStatus
	}{s.csrf, p.cfg.password != "", results})
}

func (p *panelServer) probe(ctx context.Context, origin *url.URL, path string, success int) string {
	if origin == nil {
		return "未配置"
	}
	target := *origin
	target.Path = path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return "地址无效"
	}
	resp, err := p.client.Do(req)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return "连接超时"
		}
		return "不可达"
	}
	// Health contracts use status only. Closing immediately avoids reading upstream bodies.
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == success:
		if path == "/readyz" {
			return "已就绪"
		}
		return "在线"
	case path == "/readyz" && resp.StatusCode == http.StatusServiceUnavailable:
		return "尚未就绪"
	default:
		return fmt.Sprintf("响应异常（HTTP %d）", resp.StatusCode)
	}
}
