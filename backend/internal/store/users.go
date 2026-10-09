// Package store is the Postgres adapter for core.Repository. Every
// connection/audit query is scoped by user_id in SQL, not just in handlers.
package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ars1364/webrdp-gateway/backend/internal/core"
)

type Store struct{ DB *pgxpool.Pool }

var _ core.Repository = (*Store)(nil)

const userCols = `id, username, password_hash, totp_secret_enc, role, failed_logins, locked_until, last_totp_step`

func scanUser(row pgx.Row) (*core.User, error) {
	u := &core.User{}
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.TOTPSecretEnc, &u.Role,
		&u.FailedLogins, &u.LockedUntil, &u.LastTOTPStep)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, core.ErrNotFound
	}
	return u, err
}

func (s *Store) UserByName(ctx context.Context, username string) (*core.User, error) {
	return scanUser(s.DB.QueryRow(ctx,
		`SELECT `+userCols+` FROM users WHERE lower(username) = lower($1) AND NOT is_deleted`, username))
}

// CreateUser inserts a user. The caller generated id and sealed the TOTP
// secret with it as AAD.
func (s *Store) CreateUser(ctx context.Context, id, username, pwHash string, totpEnc []byte, role string) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO users (id, username, password_hash, totp_secret_enc, role)
		VALUES ($1, $2, $3, $4, $5)`, id, username, pwHash, totpEnc, role)
	return err
}

// RecordFailedLogin increments the counter and locks the account for 15 min
// after 5 consecutive failures.
func (s *Store) RecordFailedLogin(ctx context.Context, id string) error {
	_, err := s.DB.Exec(ctx, `UPDATE users SET failed_logins = failed_logins + 1,
		locked_until = CASE WHEN failed_logins + 1 >= 5 THEN now() + interval '15 minutes' ELSE locked_until END,
		updated_at = now() WHERE id = $1`, id)
	return err
}

// RecordLogin resets failures and stores the TOTP step. The step predicate
// makes it an optimistic lock: two concurrent logins with the same code
// can't both succeed.
func (s *Store) RecordLogin(ctx context.Context, id string, step int64) error {
	tag, err := s.DB.Exec(ctx, `UPDATE users SET failed_logins = 0, locked_until = NULL,
		last_totp_step = $2, updated_at = now() WHERE id = $1 AND last_totp_step < $2`, id, step)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return core.ErrConflict
	}
	return nil
}

// SealedRow is one encrypted value plus the AAD it is bound to (for rekey).
type SealedRow struct {
	Table, ID string
	Value     []byte
}

// SealedValues lists every encrypted column value for KEK rotation.
// unscoped: operator CLI (`server rekey`), spans all users by design.
func (s *Store) SealedValues(ctx context.Context) ([]SealedRow, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT 'users', id::text, totp_secret_enc FROM users
		UNION ALL
		SELECT 'connections', id::text, password_enc FROM connections WHERE password_enc IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SealedRow
	for rows.Next() {
		var r SealedRow
		if err := rows.Scan(&r.Table, &r.ID, &r.Value); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ResealAll writes every re-encrypted value in ONE transaction: a crash
// mid-rotation leaves the old ciphertexts intact.
// unscoped: operator CLI (`server rekey`), spans all users by design.
func (s *Store) ResealAll(ctx context.Context, rows []SealedRow) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, r := range rows {
		q := `UPDATE users SET totp_secret_enc = $2 WHERE id = $1`
		if r.Table == "connections" {
			q = `UPDATE connections SET password_enc = $2 WHERE id = $1`
		}
		if _, err := tx.Exec(ctx, q, r.ID, r.Value); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
