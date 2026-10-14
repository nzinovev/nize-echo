package http

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// adminDevProxy forwards /admin/ to the Vite dev server, including the HMR
// websocket. It is mounted only in local development.
func adminDevProxy(target *url.URL, logger *slog.Logger) http.Handler {
	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.Out.Host = r.In.Host
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.WarnContext(r.Context(), "admin dev server unreachable", "target", target.String(), "err", err)
			http.Error(w, "admin dev server is not running: start it with make dev", http.StatusBadGateway)
		},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Vite's React refresh preamble is an inline script, which the site
		// CSP forbids. The embedded production build keeps the full policy.
		w.Header().Del("Content-Security-Policy")
		proxy.ServeHTTP(w, r)
	})
}
