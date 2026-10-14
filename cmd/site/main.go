// Command site is the nize-echo server: one binary serving the public pages,
// the admin API and the embedded admin app.
package main

import (
	"log/slog"
	"os"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	logger.Info("nize-echo starting", "version", version)
}
