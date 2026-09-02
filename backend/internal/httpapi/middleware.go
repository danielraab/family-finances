package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

type ctxKey int

const (
	requestIDKey ctxKey = iota
	loggerKey
)

// RequestID returns the request id attached to ctx by the middleware, or "" if
// there is none.
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// Logger returns the request-scoped logger (tagged with request_id), or the
// default logger if none is set. Handlers should log through this so their
// lines correlate with the access log.
func Logger(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// withMiddleware wraps h with the standard chain, outermost first:
// requestContext (assigns the request id, request-scoped logger, and
// X-Request-Id header), recoverPanic (turns a panic in any inner layer into a
// 500), then logRequests (one structured access-log line per request).
func withMiddleware(h http.Handler) http.Handler {
	return requestContext(recoverPanic(logRequests(h)))
}

func requestContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := newRequestID()
		w.Header().Set("X-Request-Id", id)

		ctx := context.WithValue(r.Context(), requestIDKey, id)
		ctx = context.WithValue(ctx, loggerKey, slog.With("request_id", id))

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()

		next.ServeHTTP(rec, r)

		Logger(r.Context()).LogAttrs(r.Context(), slog.LevelInfo, "request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rec.status),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

func recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				Logger(r.Context()).ErrorContext(r.Context(), "panic recovered", "value", v)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// statusRecorder captures the response status code for the access log.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(status int) {
	if !r.wroteHeader {
		r.status = status
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	r.wroteHeader = true
	return r.ResponseWriter.Write(b)
}

func newRequestID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b[:])
}
