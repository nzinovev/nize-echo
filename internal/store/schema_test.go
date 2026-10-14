package store_test

import (
	"context"
	"errors"
	"io/fs"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/nzinovev/nize-echo/db"
	"github.com/nzinovev/nize-echo/internal/store"
	"github.com/nzinovev/nize-echo/internal/store/storetest"
)

// v1Tables lists every table schema v1 creates; db/SCHEMA.md documents them.
var v1Tables = []string{
	"context_posts", "context_slug_history", "context_translations", "contexts",
	"media", "media_alt_texts", "media_variants",
	"page_files", "page_translations", "pages",
	"post_slug_history", "post_tags", "post_translations", "posts",
	"sessions", "settings", "tag_labels", "tags", "users",
}

// migrated returns a fresh database with the full schema applied.
func migrated(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := storetest.New(t)
	if _, err := store.Migrate(context.Background(), pool, discard); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return pool
}

func exec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

func insertID(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&id); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return id
}

// wantPgError fails unless err is a Postgres error with the given SQLSTATE.
func wantPgError(t *testing.T, err error, code string) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("got error %v, want SQLSTATE %s", err, code)
	}
	if pgErr.Code != code {
		t.Fatalf("got SQLSTATE %s (%s), want %s", pgErr.Code, pgErr.Message, code)
	}
}

const (
	uniqueViolation     = "23505"
	foreignKeyViolation = "23503"
	checkViolation      = "23514"
)

func publicTables(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		  AND table_name <> 'goose_db_version'
		ORDER BY table_name`)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	return names
}

func TestSchemaTables(t *testing.T) {
	pool := migrated(t)
	if got := publicTables(t, pool); !slices.Equal(got, v1Tables) {
		t.Errorf("tables = %v, want %v", got, v1Tables)
	}
}

func TestPostSlugUniquePerLanguage(t *testing.T) {
	pool := migrated(t)
	ctx := context.Background()
	a := insertID(t, pool, "INSERT INTO posts DEFAULT VALUES RETURNING id")
	b := insertID(t, pool, "INSERT INTO posts DEFAULT VALUES RETURNING id")
	const insert = "INSERT INTO post_translations (post_id, lang, slug, title) VALUES ($1, $2, $3, 't')"

	exec(t, pool, insert, a, "ru", "hello")
	_, err := pool.Exec(ctx, insert, b, "ru", "hello")
	wantPgError(t, err, uniqueViolation)
	exec(t, pool, insert, b, "en", "hello")

	_, err = pool.Exec(ctx, insert, b, "ru", "Not-A-Slug")
	wantPgError(t, err, checkViolation)
}

func TestPublishedPostNeedsPublishAt(t *testing.T) {
	pool := migrated(t)
	_, err := pool.Exec(context.Background(), "INSERT INTO posts (status) VALUES ('published')")
	wantPgError(t, err, checkViolation)
	exec(t, pool, "INSERT INTO posts (status, publish_at) VALUES ('published', now())")
}

func TestPostInAtMostOneContext(t *testing.T) {
	pool := migrated(t)
	post := insertID(t, pool, "INSERT INTO posts DEFAULT VALUES RETURNING id")
	c1 := insertID(t, pool, "INSERT INTO contexts (slug, position) VALUES ('one', 1) RETURNING id")
	c2 := insertID(t, pool, "INSERT INTO contexts (slug, position) VALUES ('two', 2) RETURNING id")

	exec(t, pool, "INSERT INTO context_posts (post_id, context_id, position) VALUES ($1, $2, 1)", post, c1)
	_, err := pool.Exec(context.Background(),
		"INSERT INTO context_posts (post_id, context_id, position) VALUES ($1, $2, 1)", post, c2)
	wantPgError(t, err, uniqueViolation)
}

func TestPositionsSwapInOneTransaction(t *testing.T) {
	pool := migrated(t)
	ctx := context.Background()
	c := insertID(t, pool, "INSERT INTO contexts (slug, position) VALUES ('one', 1) RETURNING id")
	insertID(t, pool, "INSERT INTO contexts (slug, position) VALUES ('two', 2) RETURNING id")
	p1 := insertID(t, pool, "INSERT INTO posts DEFAULT VALUES RETURNING id")
	p2 := insertID(t, pool, "INSERT INTO posts DEFAULT VALUES RETURNING id")
	exec(t, pool, "INSERT INTO context_posts (post_id, context_id, position) VALUES ($1, $3, 1), ($2, $3, 2)", p1, p2, c)

	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		for _, sql := range []string{
			"UPDATE contexts SET position = 2 WHERE slug = 'one'",
			"UPDATE contexts SET position = 1 WHERE slug = 'two'",
			"UPDATE context_posts SET position = 3 - position",
		} {
			if _, err := tx.Exec(ctx, sql); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("swap positions: %v", err)
	}

	var pos int
	if err := pool.QueryRow(ctx, "SELECT position FROM context_posts WHERE post_id = $1", p1).Scan(&pos); err != nil {
		t.Fatal(err)
	}
	if pos != 2 {
		t.Errorf("post %d position = %d, want 2", p1, pos)
	}

	// The constraint still holds at commit.
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE contexts SET position = 1")
		return err
	})
	wantPgError(t, err, uniqueViolation)
}

func TestDeletePostCascades(t *testing.T) {
	pool := migrated(t)
	ctx := context.Background()
	post := insertID(t, pool, "INSERT INTO posts DEFAULT VALUES RETURNING id")
	c := insertID(t, pool, "INSERT INTO contexts (slug, position) VALUES ('one', 1) RETURNING id")
	tag := insertID(t, pool, "INSERT INTO tags (slug) VALUES ('go') RETURNING id")
	exec(t, pool, "INSERT INTO post_translations (post_id, lang, slug, title) VALUES ($1, 'ru', 'new', 't')", post)
	exec(t, pool, "INSERT INTO post_slug_history (post_id, lang, slug) VALUES ($1, 'ru', 'old')", post)
	exec(t, pool, "INSERT INTO post_tags (post_id, tag_id) VALUES ($1, $2)", post, tag)
	exec(t, pool, "INSERT INTO context_posts (post_id, context_id, position) VALUES ($1, $2, 1)", post, c)

	exec(t, pool, "DELETE FROM posts WHERE id = $1", post)

	for _, table := range []string{"post_translations", "post_slug_history", "post_tags", "context_posts"} {
		var n int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("%s has %d rows after deleting the post, want 0", table, n)
		}
	}
	var tags int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM tags").Scan(&tags); err != nil {
		t.Fatal(err)
	}
	if tags != 1 {
		t.Errorf("tags = %d after deleting the post, want 1", tags)
	}
}

func TestUsedMediaCannotBeDeleted(t *testing.T) {
	pool := migrated(t)
	ctx := context.Background()
	const insertMedia = "INSERT INTO media (storage_key, filename, mime_type, size_bytes) VALUES ($1, 'f', 'image/png', 1) RETURNING id"
	cover := insertID(t, pool, insertMedia, "cover")
	pdf := insertID(t, pool, insertMedia, "pdf")
	exec(t, pool, "INSERT INTO posts (cover_media_id) VALUES ($1)", cover)
	page := insertID(t, pool, "INSERT INTO pages (key) VALUES ('cv') RETURNING id")
	exec(t, pool, "INSERT INTO page_files (page_id, lang, media_id) VALUES ($1, 'ru', $2)", page, pdf)

	for _, id := range []int64{cover, pdf} {
		_, err := pool.Exec(ctx, "DELETE FROM media WHERE id = $1", id)
		wantPgError(t, err, foreignKeyViolation)
	}
}

func TestUsernameCaseInsensitive(t *testing.T) {
	pool := migrated(t)
	exec(t, pool, "INSERT INTO users (username, password_hash) VALUES ('Nikita', 'x')")
	_, err := pool.Exec(context.Background(), "INSERT INTO users (username, password_hash) VALUES ('nikita', 'x')")
	wantPgError(t, err, uniqueViolation)
}

// TestSchemaDownUp keeps the Down sections honest: rolling back to the
// baseline leaves no tables and applying again restores them.
func TestSchemaDownUp(t *testing.T) {
	pool := migrated(t)
	ctx := context.Background()

	migrations, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := provider.DownTo(ctx, 1); err != nil {
		t.Fatalf("DownTo(1): %v", err)
	}
	if got := publicTables(t, pool); len(got) != 0 {
		t.Errorf("tables after down = %v, want none", got)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("Up after down: %v", err)
	}
	if got := publicTables(t, pool); !slices.Equal(got, v1Tables) {
		t.Errorf("tables after up = %v, want %v", got, v1Tables)
	}
}
