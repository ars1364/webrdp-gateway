package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/ars1364/webrdp-gateway/backend/internal/core"
)

const connCols = `id, name, host, port, username, domain, security, ignore_cert, quality,
	password_enc IS NOT NULL, created_at, updated_at, password_enc`

func scanConn(row pgx.Row) (*core.Connection, error) {
	c := &core.Connection{}
	err := row.Scan(&c.ID, &c.Name, &c.Host, &c.Port, &c.Username, &c.Domain, &c.Security,
		&c.IgnoreCert, &c.Quality, &c.HasPassword, &c.CreatedAt, &c.UpdatedAt, &c.PasswordEnc)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, core.ErrNotFound
	}
	return c, err
}

func (s *Store) ListConnections(ctx context.Context, userID string, p core.Page) ([]*core.Connection, int, error) {
	var total int
	if err := s.DB.QueryRow(ctx, `SELECT count(*) FROM connections WHERE user_id = $1 AND NOT is_deleted`,
		userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.DB.Query(ctx, `SELECT `+connCols+` FROM connections
		WHERE user_id = $1 AND NOT is_deleted ORDER BY lower(name), id LIMIT $2 OFFSET $3`,
		userID, p.PerPage, p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []*core.Connection{}
	for rows.Next() {
		c, err := scanConn(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	return out, total, rows.Err()
}

func (s *Store) GetConnection(ctx context.Context, userID, id string) (*core.Connection, error) {
	return scanConn(s.DB.QueryRow(ctx, `SELECT `+connCols+` FROM connections
		WHERE user_id = $1 AND id = $2 AND NOT is_deleted`, userID, id))
}

func (s *Store) CreateConnection(ctx context.Context, userID string, c *core.Connection) error {
	return s.DB.QueryRow(ctx, `INSERT INTO connections
		(id, user_id, name, host, port, username, domain, password_enc, security, ignore_cert, quality, created_by)
		VALUES ($2, $1, $3, $4, $5, $6, $7, $8, $9, $10, $11, $1) RETURNING created_at, updated_at`,
		userID, c.ID, c.Name, c.Host, c.Port, c.Username, c.Domain, c.PasswordEnc, c.Security, c.IgnoreCert, c.Quality).
		Scan(&c.CreatedAt, &c.UpdatedAt)
}

// UpdateConnection replaces the editable fields. keepPassword leaves the
// stored ciphertext untouched (the client didn't send a new password).
func (s *Store) UpdateConnection(ctx context.Context, userID string, c *core.Connection, keepPassword bool) error {
	tag, err := s.DB.Exec(ctx, `UPDATE connections SET name = $3, host = $4, port = $5, username = $6,
		domain = $7, security = $8, ignore_cert = $9, quality = $12,
		password_enc = CASE WHEN $10 THEN password_enc ELSE $11 END, updated_at = now()
		WHERE user_id = $1 AND id = $2 AND NOT is_deleted`,
		userID, c.ID, c.Name, c.Host, c.Port, c.Username, c.Domain, c.Security, c.IgnoreCert,
		keepPassword, c.PasswordEnc, c.Quality)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return core.ErrNotFound
	}
	return nil
}

func (s *Store) DeleteConnection(ctx context.Context, userID, id string) error {
	tag, err := s.DB.Exec(ctx, `UPDATE connections SET is_deleted = TRUE, deleted_at = now(),
		password_enc = NULL WHERE user_id = $1 AND id = $2 AND NOT is_deleted`, userID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return core.ErrNotFound
	}
	return nil
}
