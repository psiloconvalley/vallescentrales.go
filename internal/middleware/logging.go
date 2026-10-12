// internal/middleware/logging.go
// Custom slog-native request logging and structured panic recovery middleware.

package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// responseWriterDelegator intercepts and records write status and bytes written.
type responseWriterDelegator struct {
	http.ResponseWriter
	status      int
	written     int64
	wroteHeader bool
}

func newResponseWriterDelegator(w http.ResponseWriter) *responseWriterDelegator {
	return &responseWriterDelegator{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriterDelegator) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
	rw.wroteHeader = true
}

func (rw *responseWriterDelegator) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.written += int64(n)
	return n, err
}

// RequestLogger is a structured logging middleware using the global slog configuration.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := newResponseWriterDelegator(w)

		// Get request ID injected by chi's middleware
		reqID := chimiddleware.GetReqID(r.Context())

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		// Construct structured log fields
		args := []any{
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rw.status),
			slog.Int64("bytes", rw.written),
			slog.Duration("duration", duration),
			slog.String("remote_ip", r.RemoteAddr),
		}

		if reqID != "" {
			args = append(args, slog.String("request_id", reqID))
		}

		if r.URL.RawQuery != "" {
			args = append(args, slog.String("query", r.URL.RawQuery))
		}

		// Log warning/error levels automatically for client/server failures
		switch {
		case rw.status >= 500:
			slog.Error("request failed", args...)
		case rw.status >= 400:
			slog.Warn("request client error", args...)
		default:
			slog.Info("request completed", args...)
		}
	})
}

// PanicRecoverer catches unhandled panics, logs the stack trace with slog,
// and responds with a secure HTTP 500 status code.
func PanicRecoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				reqID := chimiddleware.GetReqID(r.Context())
				stack := debug.Stack()

				slog.Error("recovered from panic in HTTP handler",
					slog.Any("error", err),
					slog.String("request_id", reqID),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("stack", string(stack)),
				)

				w.Header().Set("Connection", "close")
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("Internal Server Error"))
			}
		}()

		next.ServeHTTP(w, r)
	})
}
