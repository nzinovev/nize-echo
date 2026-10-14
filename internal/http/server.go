package http

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// Pinger checks that a dependency, such as the database pool, is reachable.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Options configures the site handler.
type Options struct {
	Logger *slog.Logger
	// DB, when set, is pinged by /healthz.
	DB Pinger
	// AdminDevURL, when set, proxies /admin/ to the Vite dev server.
	AdminDevURL *url.URL
}

// New returns the site's root handler: the router wrapped in the middleware
// chain.
func New(opts Options) http.Handler {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthz(opts.DB, logger))
	mux.HandleFunc("GET /{$}", handleHome)
	if opts.AdminDevURL != nil {
		mux.Handle("/admin/", adminDevProxy(opts.AdminDevURL, logger))
		mux.Handle("GET /admin", http.RedirectHandler("/admin/", http.StatusMovedPermanently))
	}

	return chain(mux,
		requestLog(logger),
		recoverer(logger),
		securityHeaders,
		compress,
	)
}

// healthTimeout bounds the dependency checks behind /healthz.
const healthTimeout = 2 * time.Second

// healthz reports 200 when the server and its database are reachable, 503
// otherwise.
func healthz(db Pinger, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if db != nil {
			ctx, cancel := context.WithTimeout(r.Context(), healthTimeout)
			defer cancel()
			if err := db.Ping(ctx); err != nil {
				logger.WarnContext(r.Context(), "health check: database unreachable", "err", err)
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte("database unreachable\n"))
				return
			}
		}
		_, _ = w.Write([]byte("ok\n"))
	})
}

// handleHome is a placeholder until the public layout lands in P-01.
func handleHome(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(homePage))
}

const homePage = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>nize-echo</title>
</head>
<body>
<h1>nize-echo</h1>
<p>Hello. The site is under construction.</p>
</body>
</html>
`
