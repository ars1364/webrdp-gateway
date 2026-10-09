//go:build integration

// Integration tests against a real Postgres. CI runs them with a service
// container: TEST_DATABASE_URL=postgres://... go test -tags=integration ./...
package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ars1364/webrdp-gateway/backend/internal/core"
	"github.com/ars1364/webrdp-gateway/backend/internal/db"
)

func open(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := db.Open(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func tables(t *testing.T, pool *pgxpool.Pool) int {
	var n int
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name <> 'schema_migrations'`).Scan(&n)
	return n
}

// Every migration must be reversible and re-appliable (up → down → up).
func TestMigrationsReversible(t *testing.T) {
	ctx := context.Background()
	pool := open(t)
	names, _ := db.Names()
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("up: %v", err)
	}
	full := tables(t, pool)
	if err := db.Migrate(ctx, pool); err != nil { // run-once ledger: no-op
		t.Fatalf("second up: %v", err)
	}
	if err := db.MigrateDown(ctx, pool, len(names)); err != nil {
		t.Fatalf("down: %v", err)
	}
	if n := tables(t, pool); n != 0 {
		t.Fatalf("%d tables left after full down", n)
	}
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("re-up: %v", err)
	}
	if n := tables(t, pool); n != full {
		t.Fatalf("re-up produced %d tables, want %d", n, full)
	}
}

func seedUser(t *testing.T, s *Store, id, name string) {
	t.Helper()
	if err := s.CreateUser(context.Background(), id, name, "x", []byte{1}, "user"); err != nil {
		t.Fatal(err)
	}
}

func TestRepoScopingAndConstraints(t *testing.T) {
	ctx := context.Background()
	pool := open(t)
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	s := &Store{DB: pool}
	a, b := "11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222"
	_, _ = pool.Exec(ctx, `DELETE FROM idempotency_keys; DELETE FROM connections; DELETE FROM sessions; DELETE FROM users`)
	seedUser(t, s, a, "alice")
	seedUser(t, s, b, "bob")

	c := &core.Connection{ID: "33333333-3333-4333-8333-333333333333", Name: "x", Host: "203.0.113.1",
		Port: 3389, Security: "any"}
	if err := s.CreateConnection(ctx, a, c); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetConnection(ctx, b, c.ID); err != core.ErrNotFound {
		t.Fatalf("bob read alice's row: %v", err)
	}
	if err := s.UpdateConnection(ctx, b, c, true); err != core.ErrNotFound {
		t.Fatalf("bob updated alice's row: %v", err)
	}
	if err := s.DeleteConnection(ctx, b, c.ID); err != core.ErrNotFound {
		t.Fatalf("bob deleted alice's row: %v", err)
	}
	if _, total, _ := s.ListConnections(ctx, b, core.Page{Page: 1, PerPage: 50}); total != 0 {
		t.Fatalf("bob lists %d rows", total)
	}
	bad := *c
	bad.ID, bad.Port = "44444444-4444-4444-8444-444444444444", 70000
	if err := s.CreateConnection(ctx, a, &bad); err == nil {
		t.Fatal("DB accepted port 70000 (CHECK constraint missing)")
	}
	if err := s.CreateUser(ctx, "55555555-5555-4555-8555-555555555555", "ALICE", "x", []byte{1}, "user"); err == nil {
		t.Fatal("duplicate username (case-insensitive) accepted")
	}

	s.Audit(ctx, a, "test", "t", "1.2.3.4", nil)
	if _, err := pool.Exec(ctx, `UPDATE audit_log SET action = 'tampered'`); err == nil {
		t.Fatal("audit_log UPDATE allowed; must be append-only")
	}
}

func TestIdempotencyStore(t *testing.T) {
	ctx := context.Background()
	pool := open(t)
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	s := &Store{DB: pool}
	u := "66666666-6666-4666-8666-666666666666"
	_, _ = pool.Exec(ctx, `DELETE FROM idempotency_keys WHERE user_id = $1`, u)
	_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, u)
	seedUser(t, s, u, "idem")
	key := "77777777-7777-4777-8777-777777777777"
	if _, owned, err := s.IdemBegin(ctx, u, key, "h1", time.Hour); err != nil || !owned {
		t.Fatalf("first begin: owned=%v err=%v", owned, err)
	}
	if rec, owned, _ := s.IdemBegin(ctx, u, key, "h1", time.Hour); owned || rec.Status != 0 {
		t.Fatalf("concurrent begin should see in-progress, got owned=%v %+v", owned, rec)
	}
	_ = s.IdemFinish(ctx, u, key, 201, []byte(`{"ok":true}`))
	if rec, _, _ := s.IdemBegin(ctx, u, key, "h1", time.Hour); rec.Status != 201 || string(rec.Body) != `{"ok":true}` {
		t.Fatalf("replay record %+v", rec)
	}
	if err := s.Reap(ctx, 180); err != nil {
		t.Fatalf("reap: %v", err)
	}
}
