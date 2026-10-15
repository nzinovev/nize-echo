package http

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestAdminDevProxy(t *testing.T) {
	vite := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, "vite "+r.URL.Path)
	}))
	defer vite.Close()
	target, err := url.Parse(vite.URL)
	if err != nil {
		t.Fatal(err)
	}

	h := newHandler(t, Options{Logger: slog.New(&logRecorder{}), AdminDevURL: target})

	res := serve(t, h, http.MethodGet, "/admin/src/main.tsx", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	if got := res.Body.String(); got != "vite /admin/src/main.tsx" {
		t.Errorf("body = %q, want the upstream response for the same path", got)
	}
	if res.Header().Get("Content-Security-Policy") != "" {
		t.Error("dev proxy should drop the CSP so Vite's inline preamble runs")
	}
	if res.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("dev proxy should keep the other security headers")
	}

	res = serve(t, h, http.MethodGet, "/admin", nil)
	if res.Code != http.StatusMovedPermanently || res.Header().Get("Location") != "/admin/" {
		t.Errorf("GET /admin = %d to %q, want 301 to /admin/", res.Code, res.Header().Get("Location"))
	}
}

func TestAdminDevProxyUnreachable(t *testing.T) {
	target, _ := url.Parse("http://127.0.0.1:1")
	h := newHandler(t, Options{Logger: slog.New(&logRecorder{}), AdminDevURL: target})
	if res := serve(t, h, http.MethodGet, "/admin/", nil); res.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", res.Code)
	}
}

func TestAdminNotMountedWithoutDevURL(t *testing.T) {
	h := newHandler(t, Options{Logger: slog.New(&logRecorder{})})
	if res := serve(t, h, http.MethodGet, "/admin/", nil); res.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", res.Code)
	}
}
