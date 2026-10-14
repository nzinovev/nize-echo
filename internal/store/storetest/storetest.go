// Package storetest gives tests a fresh, empty Postgres database.
//
// It connects to TEST_DATABASE_URL (a postgres:// URL), creates a uniquely
// named database for each test and drops it when the test ends. When the
// variable is unset or the server is unreachable the test is skipped, unless
// TEST_DATABASE_REQUIRED=1 (set in CI), in which case it fails.
package storetest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// URL returns the connection string for a fresh, empty database that is
// dropped when t ends.
func URL(t testing.TB) string {
	t.Helper()

	adminURL := os.Getenv("TEST_DATABASE_URL")
	if adminURL == "" {
		unavailable(t, "TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		unavailable(t, "cannot connect to TEST_DATABASE_URL: "+err.Error())
	}
	defer admin.Close(context.Background())

	name := "nize_test_" + randomSuffix(t)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		t.Fatalf("create test database: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		conn, err := pgx.Connect(ctx, adminURL)
		if err != nil {
			t.Errorf("drop test database %s: %v", name, err)
			return
		}
		defer conn.Close(ctx)
		if _, err := conn.Exec(ctx, "DROP DATABASE "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)"); err != nil {
			t.Errorf("drop test database %s: %v", name, err)
		}
	})

	u, err := url.Parse(adminURL)
	if err != nil || u.Scheme == "" {
		t.Fatalf("TEST_DATABASE_URL must be a postgres:// URL: %v", err)
	}
	u.Path = "/" + name
	return u.String()
}

// New returns a pool connected to a fresh, empty database that is dropped
// when t ends.
func New(t testing.TB) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), URL(t))
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	// Registered after URL's cleanup, so it runs first: close, then drop.
	t.Cleanup(pool.Close)
	return pool
}

func unavailable(t testing.TB, reason string) {
	t.Helper()
	if os.Getenv("TEST_DATABASE_REQUIRED") == "1" {
		t.Fatalf("test database required: %s", reason)
	}
	t.Skipf("skipping database test: %s", reason)
}

func randomSuffix(t testing.TB) string {
	t.Helper()
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("random database name: %v", err)
	}
	return hex.EncodeToString(b)
}
