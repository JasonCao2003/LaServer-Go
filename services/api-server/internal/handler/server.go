// API 服务构造：创建 Gin 引擎并装配路由

package handler

import (
	"net/http"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/config"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service"
	"github.com/gin-gonic/gin"
)

type Config = config.Config
type object = service.Object
type endpoint func(service.Request, object) (object, error)

type API struct {
	cfg    *Config
	router *gin.Engine
}

// 创建 API 实例并配置 Gin 引擎
func NewAPI(s *service.Service, cfg *Config) *API {
	r := gin.New()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.HandleMethodNotAllowed = true
	r.ForwardedByClientIP = false
	r.Use(func(c *gin.Context) {
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Header("Cache-Control", "no-store")
	})
	registerIPBans(r, cfg)
	registerRoutes(r, s)

	legacy := make(map[string]bool, len(service.LegacyPaths))
	for _, path := range service.LegacyPaths {
		legacy[path] = true
	}
	r.NoRoute(func(c *gin.Context) {
		status := http.StatusNotFound
		if legacy[c.Request.URL.Path] {
			status = http.StatusNotImplemented
		}
		write(c.Writer, status, object{"error": http.StatusText(status)})
	})
	r.NoMethod(func(c *gin.Context) {
		// The legacy health probe omits Allow. Other routes use Gin's registered methods.
		if c.Request.URL.Path == "/healthz" {
			c.Writer.Header().Del("Allow")
		}
		write(c.Writer, http.StatusMethodNotAllowed, object{"error": "Method not allowed"})
	})
	return &API{cfg: cfg, router: r}
}

// 提供 HTTP 处理入口
func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

// 返回运行时配置
func (a *API) RuntimeConfig() *Config {
	return a.cfg
}

// 返回底层路由引擎
func (a *API) Engine() *gin.Engine {
	return a.router
}
