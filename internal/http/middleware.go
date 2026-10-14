package http

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// middleware wraps a handler with extra behavior.
type middleware func(http.Handler) http.Handler

// chain wraps h so that the first middleware is the outermost.
func chain(h http.Handler, mws ...middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// statusRecorder remembers the status code and body size written through it.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.status == 0 {
		r.status = code
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += int64(n)
	return n, err
}

// Unwrap lets http.ResponseController reach Flush and Hijack underneath.
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// requestLog logs one line per request with its status, size and duration.
func requestLog(logger *slog.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}
			defer func() {
				status := rec.status
				if status == 0 {
					status = http.StatusOK
				}
				level := slog.LevelInfo
				if status >= http.StatusInternalServerError {
					level = slog.LevelError
				}
				logger.LogAttrs(r.Context(), level, "request",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", status),
					slog.Int64("bytes", rec.bytes),
					slog.Duration("duration", time.Since(start)),
				)
			}()
			next.ServeHTTP(rec, r)
		})
	}
}

// recoverer turns a panic in a handler into a logged 500 response.
func recoverer(logger *slog.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w}
			defer func() {
				v := recover()
				if v == nil {
					return
				}
				if err, ok := v.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(v)
				}
				logger.ErrorContext(r.Context(), "handler panic",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("panic", fmt.Sprint(v)),
					slog.String("stack", string(debug.Stack())),
				)
				if rec.status == 0 {
					http.Error(rec, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(rec, r)
		})
	}
}

// contentSecurityPolicy allows only same-origin resources; public pages need
// no inline scripts or styles.
const contentSecurityPolicy = "default-src 'self'; img-src 'self' data:; object-src 'none'; " +
	"base-uri 'self'; form-action 'self'; frame-ancestors 'none'"

// securityHeaders sets headers every response carries.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}
