// Package store is the Postgres data layer. Every connection query is scoped
// by the owning user_id as its first argument.
package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type Store struct{ DB *pgxpool.Pool }

type User struct {
	ID            string
	Username      string
	PasswordHash  string
	TOTPSecretEnc []byte
	Role          string
	FailedLogins  int
	LockedUntil   *time.Time
	LastTOTPStep  int64
}

const userCols = `id, username, password_hash, totp_secret_enc, role, failed_logins, locked_until, last_totp_step`

func scanUser(row pgx.Row) (*User, error) {
	u := &User{}
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.TOTPSecretEnc, &u.Role,
		&u.FailedLogins, &u.LockedUntil, &u.LastTOTPStep)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (s *Store) UserByName(ctx context.Context, username string) (*User, error) {
	return scanUser(s.DB.QueryRow(ctx,
		`SELECT `+userCols+` FROM users WHERE lower(username) = lower($1) AND NOT is_deleted`, username))
}

func (s *Store) UserByID(ctx context.Context, id string) (*User, error) {
	return scanUser(s.DB.QueryRow(ctx,
		`SELECT `+userCols+` FROM users WHERE id = $1 AND NOT is_deleted`, id))
}

// CreateUser inserts a user; the caller seals the TOTP secret with the user id
// as AAD, so the id is generated here and passed back to seal before insert.
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

// RecordLogin resets failures and stores the TOTP step to block code replay.
// It fails (ErrNotFound) if the step was already used, closing the race
// between two concurrent logins with the same code.
func (s *Store) RecordLogin(ctx context.Context, id string, step int64) error {
	tag, err := s.DB.Exec(ctx, `UPDATE users SET failed_logins = 0, locked_until = NULL,
		last_totp_step = $2, updated_at = now() WHERE id = $1 AND last_totp_step < $2`, id, step)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
