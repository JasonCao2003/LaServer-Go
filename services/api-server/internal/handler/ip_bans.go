// 接入 IP 封禁管理和游戏请求检查。
package handler

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"net/netip"
	"slices"
	"strings"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/core"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service/ipban"
	"github.com/gin-gonic/gin"
)

func registerIPBans(r *gin.Engine, cfg *Config) {
	rules := ipban.Open(cfg.InstanceDir)
	proxies := make([]netip.Prefix, 0, len(cfg.IPBanTrustedProxies))
	for _, raw := range cfg.IPBanTrustedProxies {
		if prefix, err := netip.ParsePrefix(raw); err == nil && prefix.Bits() > 0 && !prefix.Addr().Is4In6() {
			proxies = append(proxies, prefix.Masked())
		}
	}
	r.Use(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/admin/") || strings.HasPrefix(path, "/internal/") || path == "/healthz" {
			return
		}
		blocked, err := rules.Match(ipBanSource(c.Request, proxies))
		switch {
		case err != nil:
			c.Abort()
			write(c.Writer, http.StatusServiceUnavailable, object{"error": "IP ban rules unavailable"})
		case blocked:
			c.Abort()
			write(c.Writer, http.StatusMethodNotAllowed, object{"error": "forbidden", "message": "该IP已被封禁"})
		}
	})
	r.POST("/admin/api/ip_bans", post(func(req service.Request, data object) (object, error) {
		if cfg.AdminToken == "" || subtle.ConstantTimeCompare([]byte(req.Header.Get("X-Admin-Token")), []byte(cfg.AdminToken)) != 1 {
			return nil, &service.Error{Status: http.StatusForbidden, Body: object{"ok": false, "error": "forbidden"}}
		}
		action := core.Text(data, "action")
		var entries []string
		var err error
		switch action {
		case "list":
			entries, err = rules.List()
		case "add", "remove":
			entries, err = rules.Change(core.Text(data, "entry"), action == "add")
		default:
			return nil, &service.Error{Status: http.StatusBadRequest, Body: object{"ok": false, "error": "invalid action"}}
		}
		if errors.Is(err, ipban.ErrInvalid) || errors.Is(err, ipban.ErrFull) {
			return nil, &service.Error{Status: http.StatusBadRequest, Body: object{"ok": false, "error": err.Error()}}
		}
		if err != nil {
			return nil, err
		}
		return object{"ok": true, "entries": entries}, nil
	}))
}

func ipBanSource(r *http.Request, proxies []netip.Prefix) netip.Addr {
	peer, err := netip.ParseAddr(remoteIP(r))
	if err != nil {
		return netip.Addr{}
	}
	peer = peer.WithZone("").Unmap()
	trusted := func(ip netip.Addr) bool {
		return slices.ContainsFunc(proxies, func(prefix netip.Prefix) bool { return prefix.Contains(ip) })
	}
	if !trusted(peer) {
		return peer
	}
	forwarded := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
	if len(forwarded) > 32 {
		return peer
	}
	for i := len(forwarded) - 1; i >= 0; i-- {
		address, err := netip.ParseAddr(strings.TrimSpace(forwarded[i]))
		if err != nil || address.Zone() != "" {
			return peer
		}
		address = address.Unmap()
		if !trusted(address) || i == 0 {
			return address
		}
	}
	return peer
}
