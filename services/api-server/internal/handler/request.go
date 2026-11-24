// HTTP 请求解析与统一 JSON 响应/错误输出

package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/middleware"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service"
	"github.com/gin-gonic/gin"
)

// 解析 POST 请求并交给服务处理
func post(fn endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := middleware.ReadObject(c.Writer, c.Request)
		if err != nil {
			write(c.Writer, err.Status, object{"error": err.Message})
			return
		}
		respond(c, fn, data)
	}
}

// 把查询参数交给管理接口处理
func adminQuery(fn endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		data := object{}
		for key, values := range c.Request.URL.Query() {
			if len(values) == 1 {
				data[key] = values[0]
			}
		}
		respond(c, fn, data)
	}
}

// 调用服务并写出 JSON 响应
func respond(c *gin.Context, fn endpoint, data object) {
	r := c.Request
	result, err := fn(service.Request{
		Context: r.Context(), Path: r.URL.Path, UserAgent: r.UserAgent(),
		RemoteIP: remoteIP(r), Host: r.Host, Secure: r.TLS != nil, Header: r.Header,
	}, data)
	if err == nil {
		write(c.Writer, http.StatusOK, result)
		return
	}
	var serviceErr *service.Error
	if errors.As(err, &serviceErr) {
		write(c.Writer, serviceErr.Status, serviceErr.Body)
		return
	}
	slog.Error("request failed", "path", r.URL.Path, "error_type", "internal")
	write(c.Writer, http.StatusInternalServerError, object{"error": "Internal server error"})
}

// 写出 JSON 响应
func write(w http.ResponseWriter, status int, body object) {
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(body)
}

// 返回请求来源 IP
func remoteIP(r *http.Request) string {
	h, _, e := net.SplitHostPort(r.RemoteAddr)
	if e != nil {
		return r.RemoteAddr
	}
	return h
}
