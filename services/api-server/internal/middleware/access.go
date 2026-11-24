// 访问控制与来源校验中间件

package middleware

import (
	"log/slog"
	"net/http"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

// 记录状态码并写出响应头
func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
		w.ResponseWriter.WriteHeader(code)
	}
}

// 记录字节数并写出响应体
func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// 返回底层响应对象
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// 记录访问日志的中间件
func Access(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracked := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(tracked, r)
		if tracked.status == 0 {
			tracked.status = 200
		}
		slog.Info("http_request", "method", r.Method, "path", r.URL.Path, "status", tracked.status)
	})
}
