package http

import (
	"log/slog"
	"net/http"
)

// Options configures the site handler.
type Options struct {
	Logger *slog.Logger
}

// New returns the site's root handler: the router wrapped in the middleware
// chain.
func New(opts Options) http.Handler {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("GET /{$}", handleHome)

	return chain(mux,
		requestLog(logger),
		recoverer(logger),
		securityHeaders,
		compress,
	)
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte("ok\n"))
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
