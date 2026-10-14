package http

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/andybalholm/brotli"
)

// logRecorder is a slog.Handler that keeps records for assertions.
type logRecorder struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *logRecorder) Enabled(context.Context, slog.Level) bool { return true }

func (h *logRecorder) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r.Clone())
	return nil
}

func (h *logRecorder) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *logRecorder) WithGroup(string) slog.Handler      { return h }

// find returns the first record with the given message.
func (h *logRecorder) find(msg string) (slog.Record, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, r := range h.records {
		if r.Message == msg {
			return r, true
		}
	}
	return slog.Record{}, false
}

func attr(r slog.Record, key string) slog.Value {
	var v slog.Value
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == key {
			v = a.Value
			return false
		}
		return true
	})
	return v
}

func serve(t *testing.T, h http.Handler, method, target string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, http.NoBody)
	for k, vs := range header {
		req.Header[k] = vs
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHealthz(t *testing.T) {
	h := New(Options{Logger: slog.New(&logRecorder{})})
	res := serve(t, h, http.MethodGet, "/healthz", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	body, _ := io.ReadAll(res.Body)
	if string(body) != "ok\n" {
		t.Errorf("body = %q, want %q", body, "ok\n")
	}
}

func TestHomeAndNotFound(t *testing.T) {
	h := New(Options{Logger: slog.New(&logRecorder{})})
	if res := serve(t, h, http.MethodGet, "/", nil); res.Code != http.StatusOK {
		t.Errorf("GET / status = %d, want 200", res.Code)
	}
	if res := serve(t, h, http.MethodGet, "/nope", nil); res.Code != http.StatusNotFound {
		t.Errorf("GET /nope status = %d, want 404", res.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	h := New(Options{Logger: slog.New(&logRecorder{})})
	for _, path := range []string{"/", "/healthz", "/nope"} {
		res := serve(t, h, http.MethodGet, path, nil)
		for _, name := range []string{"Content-Security-Policy", "X-Content-Type-Options", "Referrer-Policy"} {
			if res.Header().Get(name) == "" {
				t.Errorf("%s: missing %s", path, name)
			}
		}
		if got := res.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("%s: X-Content-Type-Options = %q", path, got)
		}
	}
}

func TestPanicReturns500AndIsLogged(t *testing.T) {
	logs := &logRecorder{}
	logger := slog.New(logs)
	h := chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}), requestLog(logger), recoverer(logger), securityHeaders, compress)

	res := serve(t, h, http.MethodGet, "/", http.Header{"Accept-Encoding": {"gzip"}})
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", res.Code)
	}
	if res.Header().Get("Content-Security-Policy") == "" {
		t.Error("500 response lacks security headers")
	}

	r, ok := logs.find("handler panic")
	if !ok {
		t.Fatal("panic was not logged")
	}
	if r.Level != slog.LevelError {
		t.Errorf("panic logged at %s, want ERROR", r.Level)
	}
	if got := attr(r, "panic").String(); got != "boom" {
		t.Errorf("panic attr = %q, want %q", got, "boom")
	}
	if !strings.Contains(attr(r, "stack").String(), "goroutine") {
		t.Error("panic log has no stack trace")
	}

	reqLog, ok := logs.find("request")
	if !ok {
		t.Fatal("request was not logged")
	}
	if got := attr(reqLog, "status").Int64(); got != http.StatusInternalServerError {
		t.Errorf("request log status = %d, want 500", got)
	}
}

func TestAbortHandlerPanicPropagates(t *testing.T) {
	h := recoverer(slog.New(&logRecorder{}))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))
	defer func() {
		if v := recover(); v != http.ErrAbortHandler { //nolint:errorlint // identity check on the panic value
			t.Errorf("recovered %v, want http.ErrAbortHandler", v)
		}
	}()
	serve(t, h, http.MethodGet, "/", nil)
}

func TestRequestLog(t *testing.T) {
	logs := &logRecorder{}
	h := New(Options{Logger: slog.New(logs)})
	serve(t, h, http.MethodGet, "/nope", nil)

	r, ok := logs.find("request")
	if !ok {
		t.Fatal("request was not logged")
	}
	if got := attr(r, "status").Int64(); got != http.StatusNotFound {
		t.Errorf("status = %d, want 404", got)
	}
	if got := attr(r, "path").String(); got != "/nope" {
		t.Errorf("path = %q, want /nope", got)
	}
}

var largeText = strings.Repeat("nize-echo compresses text responses. ", 100)

func textHandler(contentType, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		_, _ = io.WriteString(w, body)
	})
}

func TestCompress(t *testing.T) {
	decoders := map[string]func(io.Reader) (io.Reader, error){
		"gzip": func(r io.Reader) (io.Reader, error) { return gzip.NewReader(r) },
		"br":   func(r io.Reader) (io.Reader, error) { return brotli.NewReader(r), nil },
	}
	tests := []struct {
		accept, want string
	}{
		{"gzip", "gzip"},
		{"br", "br"},
		{"gzip, deflate, br", "br"},
		{"br;q=0, gzip", "gzip"},
		{"", ""},
		{"identity", ""},
	}
	h := compress(textHandler("text/html; charset=utf-8", largeText))
	for _, tt := range tests {
		t.Run(tt.accept, func(t *testing.T) {
			res := serve(t, h, http.MethodGet, "/", http.Header{"Accept-Encoding": {tt.accept}})
			if got := res.Header().Get("Content-Encoding"); got != tt.want {
				t.Fatalf("Content-Encoding = %q, want %q", got, tt.want)
			}
			if !strings.Contains(res.Header().Get("Vary"), "Accept-Encoding") {
				t.Error("missing Vary: Accept-Encoding")
			}
			var body io.Reader = res.Body
			if tt.want != "" {
				var err error
				if body, err = decoders[tt.want](res.Body); err != nil {
					t.Fatalf("decoder: %v", err)
				}
			}
			got, err := io.ReadAll(body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			if string(got) != largeText {
				t.Error("decoded body differs from the original")
			}
		})
	}
}

func TestCompressSkips(t *testing.T) {
	tests := map[string]http.Handler{
		"binary type": textHandler("image/png", largeText),
		"small body": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Content-Length", "2")
			_, _ = io.WriteString(w, "ok")
		}),
		"already encoded": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Content-Encoding", "identity")
			_, _ = io.WriteString(w, largeText)
		}),
	}
	for name, inner := range tests {
		t.Run(name, func(t *testing.T) {
			res := serve(t, compress(inner), http.MethodGet, "/", http.Header{"Accept-Encoding": {"gzip"}})
			if got := res.Header().Get("Content-Encoding"); got == "gzip" {
				t.Errorf("Content-Encoding = %q, want no compression", got)
			}
			body, _ := io.ReadAll(res.Body)
			if !bytes.Contains([]byte(largeText), body) && string(body) != "ok" {
				t.Error("body was altered")
			}
		})
	}
}

func TestCompressKeepsStatus(t *testing.T) {
	h := compress(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusTeapot)
		_, _ = io.WriteString(w, largeText)
	}))
	res := serve(t, h, http.MethodGet, "/", http.Header{"Accept-Encoding": {"gzip"}})
	if res.Code != http.StatusTeapot {
		t.Errorf("status = %d, want 418", res.Code)
	}
}
