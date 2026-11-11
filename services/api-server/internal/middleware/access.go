package middleware

import (
	"log/slog"
	"net/http"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
		w.ResponseWriter.WriteHeader(code)
	}
}
func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// Access records only request metadata. Credentials and request/response bodies stay private.
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
