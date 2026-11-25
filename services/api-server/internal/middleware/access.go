// 访问控制与来源校验中间件

package middleware

import (
	"fmt"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/traffic"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

var requestSequence atomic.Uint64

type statusWriter struct {
	http.ResponseWriter
	status  int
	bytes   int
	preview preview
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
	w.preview.Write(b)
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// 返回底层响应对象
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// 记录访问日志的中间件
func Access(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		id := requestSequence.Add(1)
		var incoming preview
		if r.Body != nil {
			r.Body = previewReader{io.TeeReader(r.Body, &incoming), r.Body}
		}
		tracked := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(tracked, r)
		if tracked.status == 0 {
			tracked.status = 200
		}
		path := r.URL.Path
		if len(path) > 256 {
			path = path[:256]
		}
		if !strings.HasPrefix(path, "/admin/") {
			traffic.Add(fmt.Sprintf("#%d %s %s (%d bytes)", id, r.Method, path, r.ContentLength)+" "+incoming.String(), fmt.Sprintf("HTTP %d · %d bytes · %s", tracked.status, tracked.bytes, time.Since(start).Round(time.Microsecond))+" "+tracked.preview.String())
		}
		slog.Info("http_request", "method", r.Method, "path", r.URL.Path, "status", tracked.status)
	})
}
