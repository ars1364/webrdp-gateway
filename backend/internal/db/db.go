// Package db opens the Postgres pool and runs embedded migrations.
//
// Migrations are NNN_name.up.sql / NNN_name.down.sql pairs. The
// schema_migrations ledger makes each one run exactly once, each inside its
// own transaction; an advisory lock stops two replicas migrating at once.
package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrations embed.FS

const lockID = 726_193_004 // arbitrary, constant advisory-lock key

func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 10
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}

// Names returns migration base names ("001_init") in order.
func Names() ([]string, error) {
	files, err := fs.Glob(migrations, "migrations/*.up.sql")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(files))
	for _, f := range files {
		names = append(names, strings.TrimSuffix(strings.TrimPrefix(f, "migrations/"), ".up.sql"))
	}
	sort.Strings(names)
	return names, nil
}

// Migrate applies every pending up migration.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	return withLock(ctx, pool, func(conn *pgxpool.Conn) error {
		names, err := Names()
		if err != nil {
			return err
		}
		for _, n := range names {
			var done bool
			if err := conn.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name = $1)`, n).Scan(&done); err != nil {
				return err
			}
			if done {
				continue
			}
			if err := apply(ctx, conn, n, ".up.sql",
				`INSERT INTO schema_migrations (name) VALUES ($1)`); err != nil {
				return err
			}
		}
		return nil
	})
}

// MigrateDown reverts the last `steps` applied migrations (newest first).
func MigrateDown(ctx context.Context, pool *pgxpool.Pool, steps int) error {
	return withLock(ctx, pool, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `SELECT name FROM schema_migrations ORDER BY name DESC LIMIT $1`, steps)
		if err != nil {
			return err
		}
		applied, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return err
		}
		for _, n := range applied {
			if err := apply(ctx, conn, n, ".down.sql",
				`DELETE FROM schema_migrations WHERE name = $1`); err != nil {
				return err
			}
		}
		return nil
	})
}

func apply(ctx context.Context, conn *pgxpool.Conn, name, suffix, ledgerSQL string) error {
	sql, err := migrations.ReadFile("migrations/" + name + suffix)
	if err != nil {
		return err
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, string(sql)); err != nil {
		return fmt.Errorf("%s%s: %w", name, suffix, err)
	}
	if _, err := tx.Exec(ctx, ledgerSQL, name); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func withLock(ctx context.Context, pool *pgxpool.Pool, fn func(*pgxpool.Conn) error) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, lockID); err != nil {
		return err
	}
	defer func() { _, _ = conn.Exec(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1)`, lockID) }()
	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		name TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	return fn(conn)
}
