package storetest_test

import (
	"context"
	"strings"
	"testing"

	"github.com/nzinovev/nize-echo/internal/store/storetest"
)

func TestNewGivesFreshDatabase(t *testing.T) {
	a, b := storetest.New(t), storetest.New(t)
	ctx := context.Background()

	var nameA, nameB string
	if err := a.QueryRow(ctx, "SELECT current_database()").Scan(&nameA); err != nil {
		t.Fatal(err)
	}
	if err := b.QueryRow(ctx, "SELECT current_database()").Scan(&nameB); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(nameA, "nize_test_") || nameA == nameB {
		t.Errorf("databases %q and %q: want two distinct nize_test_* databases", nameA, nameB)
	}

	var tables int
	if err := a.QueryRow(ctx,
		"SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public'").Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables != 0 {
		t.Errorf("fresh database has %d tables, want 0", tables)
	}
}
