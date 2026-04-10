package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (w *statusRecorder) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusRecorder) Write(b []byte) (int, error) {
	if w.code == 0 {
		w.code = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// RequestLogger emits structured JSON logs per request.
func RequestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, code: http.StatusOK}
			uidHolder := new(string)
			ctx := context.WithValue(r.Context(), logUserIDKey{}, uidHolder)
			next.ServeHTTP(rec, r.WithContext(ctx))
			log.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.code,
				"latency_ms", time.Since(start).Milliseconds(),
				"remote_ip", clientIP(r),
				"user_id", *uidHolder,
			)
		})
	}
}

type logUserIDKey struct{}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	return r.RemoteAddr
}
