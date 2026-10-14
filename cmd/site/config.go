package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"time"
)

// config is the server configuration, read from environment variables.
type config struct {
	Addr            string        // SITE_ADDR: listen address
	LogLevel        slog.Level    // LOG_LEVEL: debug, info, warn or error
	ShutdownTimeout time.Duration // SHUTDOWN_TIMEOUT: grace period for in-flight requests
	AdminDevURL     *url.URL      // ADMIN_DEV_URL: Vite dev server to proxy /admin/ to; dev only
}

// loadConfig reads the configuration through getenv, applying defaults for
// unset variables.
func loadConfig(getenv func(string) string) (config, error) {
	cfg := config{
		Addr:            ":8080",
		LogLevel:        slog.LevelInfo,
		ShutdownTimeout: 10 * time.Second,
	}
	var errs []error

	if v := getenv("SITE_ADDR"); v != "" {
		cfg.Addr = v
	}
	if v := getenv("LOG_LEVEL"); v != "" {
		if err := cfg.LogLevel.UnmarshalText([]byte(v)); err != nil {
			errs = append(errs, fmt.Errorf("LOG_LEVEL: %w", err))
		}
	}
	if v := getenv("SHUTDOWN_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		switch {
		case err != nil:
			errs = append(errs, fmt.Errorf("SHUTDOWN_TIMEOUT: %w", err))
		case d <= 0:
			errs = append(errs, fmt.Errorf("SHUTDOWN_TIMEOUT: must be positive, got %s", d))
		default:
			cfg.ShutdownTimeout = d
		}
	}

	if v := getenv("ADMIN_DEV_URL"); v != "" {
		u, err := url.Parse(v)
		switch {
		case err != nil:
			errs = append(errs, fmt.Errorf("ADMIN_DEV_URL: %w", err))
		case (u.Scheme != "http" && u.Scheme != "https") || u.Host == "":
			errs = append(errs, fmt.Errorf("ADMIN_DEV_URL: want an absolute http(s) URL, got %q", v))
		default:
			cfg.AdminDevURL = u
		}
	}

	return cfg, errors.Join(errs...)
}
