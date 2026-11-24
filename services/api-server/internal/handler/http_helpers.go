// 健康检查、UGC 静态文件与 CORS 预检处理

package handler

import (
	"net/http"
	"strings"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service"
	"github.com/gin-gonic/gin"
)

// 返回健康检查响应
func health(c *gin.Context) {
	write(c.Writer, http.StatusOK, object{"status": "ok"})
}

// 提供 UGC 文件下载
func serveUGC(s *service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := s.ServeUGC(strings.TrimPrefix(c.Param("subpath"), "/"), c.GetHeader("Range"))
		for key, value := range resp.Headers {
			c.Header(key, value)
		}
		c.Status(resp.Status)
		_, _ = c.Writer.Write(resp.Body)
	}
}

// 生成 CORS 预检响应
func preflight(methods, headers, maxAge string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", methods)
		c.Header("Access-Control-Allow-Headers", headers)
		if maxAge != "" {
			c.Header("Access-Control-Max-Age", maxAge)
		}
		c.Status(http.StatusOK)
	}
}
