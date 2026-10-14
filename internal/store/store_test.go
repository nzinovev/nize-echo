package store_test

import (
	"context"
	"io/fs"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nzinovev/nize-echo/db"
	"github.com/nzinovev/nize-echo/internal/store"
	"github.com/nzinovev/nize-echo/internal/store/storetest"
)

var discard = slog.New(slog.DiscardHandler)

// embeddedVersions lists the versions of the embedded migration files.
func embeddedVersions(t *testing.T) []int64 {
	t.Helper()
	names, err := fs.Glob(db.Migrations, "migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	var versions []int64
	for _, name := range names {
		prefix, _, _ := strings.Cut(strings.TrimPrefix(name, "migrations/"), "_")
		v, err := strconv.ParseInt(prefix, 10, 64)
		if err != nil {
			t.Fatalf("migration %s has no numeric version: %v", name, err)
		}
		versions = append(versions, v)
	}
	if len(versions) == 0 {
		t.Fatal("no embedded migrations")
	}
	return versions
}

func dbVersion(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()
	var v int64
	err := pool.QueryRow(context.Background(),
		"SELECT max(version_id) FROM goose_db_version WHERE is_applied").Scan(&v)
	if err != nil {
		t.Fatalf("read goose version: %v", err)
	}
	return v
}

func TestMigrateEmptyThenNoop(t *testing.T) {
	ctx := context.Background()
	pool, err := store.Open(ctx, storetest.URL(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer pool.Close()
	want := embeddedVersions(t)

	applied, err := store.Migrate(ctx, pool, discard)
	if err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	if !slices.Equal(applied, want) {
		t.Errorf("first Migrate applied %v, want %v", applied, want)
	}
	if got := dbVersion(t, pool); got != slices.Max(want) {
		t.Errorf("schema version = %d, want %d", got, slices.Max(want))
	}

	applied, err = store.Migrate(ctx, pool, discard)
	if err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	if len(applied) != 0 {
		t.Errorf("second Migrate applied %v, want nothing", applied)
	}
}

func TestMigrateConcurrentStarts(t *testing.T) {
	ctx := context.Background()
	url := storetest.URL(t)
	want := embeddedVersions(t)

	const starts = 3
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		applied []int64
	)
	for range starts {
		wg.Go(func() {
			pool, err := store.Open(ctx, url)
			if err != nil {
				t.Errorf("Open: %v", err)
				return
			}
			defer pool.Close()
			versions, err := store.Migrate(ctx, pool, discard)
			if err != nil {
				t.Errorf("Migrate: %v", err)
				return
			}
			mu.Lock()
			applied = append(applied, versions...)
			mu.Unlock()
		})
	}
	wg.Wait()

	slices.Sort(applied)
	if !slices.Equal(applied, want) {
		t.Errorf("concurrent starts applied %v in total, want each of %v exactly once", applied, want)
	}
}

func TestOpenRejectsBadURL(t *testing.T) {
	if _, err := store.Open(context.Background(), "not a url ::"); err == nil {
		t.Error("Open with a malformed URL: want an error")
	}
}
