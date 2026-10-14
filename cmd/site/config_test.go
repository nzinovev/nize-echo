package main

import (
	"log/slog"
	"testing"
	"time"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoadConfigDefaults(t *testing.T) {
	cfg, err := loadConfig(env(nil))
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	want := config{Addr: ":8080", LogLevel: slog.LevelInfo, ShutdownTimeout: 10 * time.Second}
	if cfg != want {
		t.Errorf("got %+v, want %+v", cfg, want)
	}
}

func TestLoadConfigOverrides(t *testing.T) {
	cfg, err := loadConfig(env(map[string]string{
		"SITE_ADDR":        "127.0.0.1:9000",
		"LOG_LEVEL":        "debug",
		"SHUTDOWN_TIMEOUT": "3s",
	}))
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	want := config{Addr: "127.0.0.1:9000", LogLevel: slog.LevelDebug, ShutdownTimeout: 3 * time.Second}
	if cfg != want {
		t.Errorf("got %+v, want %+v", cfg, want)
	}
}

func TestLoadConfigInvalid(t *testing.T) {
	tests := map[string]map[string]string{
		"bad level":        {"LOG_LEVEL": "loud"},
		"bad timeout":      {"SHUTDOWN_TIMEOUT": "soon"},
		"negative timeout": {"SHUTDOWN_TIMEOUT": "-1s"},
		"zero timeout":     {"SHUTDOWN_TIMEOUT": "0s"},
		"several bad vars": {"LOG_LEVEL": "loud", "SHUTDOWN_TIMEOUT": "soon"},
	}
	for name, vars := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := loadConfig(env(vars)); err == nil {
				t.Error("want an error, got nil")
			}
		})
	}
}
