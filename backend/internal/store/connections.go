package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Connection is a saved RDP target. PasswordEnc is sealed with the row id as
// AAD and is never serialized to clients.
type Connection struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	Username    string    `json:"username"`
	Domain      string    `json:"domain"`
	Security    string    `json:"security"`
	IgnoreCert  bool      `json:"ignore_cert"`
	HasPassword bool      `json:"has_password"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	PasswordEnc []byte    `json:"-"`
}

const connCols = `id, name, host, port, username, domain, security, ignore_cert,
	password_enc IS NOT NULL, created_at, updated_at, password_enc`

func scanConn(row pgx.Row) (*Connection, error) {
	c := &Connection{}
	err := row.Scan(&c.ID, &c.Name, &c.Host, &c.Port, &c.Username, &c.Domain, &c.Security,
		&c.IgnoreCert, &c.HasPassword, &c.CreatedAt, &c.UpdatedAt, &c.PasswordEnc)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}

func (s *Store) ListConnections(ctx context.Context, userID string) ([]*Connection, error) {
	rows, err := s.DB.Query(ctx, `SELECT `+connCols+` FROM connections
		WHERE user_id = $1 AND NOT is_deleted ORDER BY lower(name)`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Connection{}
	for rows.Next() {
		c, err := scanConn(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) GetConnection(ctx context.Context, userID, id string) (*Connection, error) {
	return scanConn(s.DB.QueryRow(ctx, `SELECT `+connCols+` FROM connections
		WHERE user_id = $1 AND id = $2 AND NOT is_deleted`, userID, id))
}

func (s *Store) CreateConnection(ctx context.Context, userID string, c *Connection) error {
	return s.DB.QueryRow(ctx, `INSERT INTO connections
		(id, user_id, name, host, port, username, domain, password_enc, security, ignore_cert, created_by)
		VALUES ($2, $1, $3, $4, $5, $6, $7, $8, $9, $10, $1) RETURNING created_at, updated_at`,
		userID, c.ID, c.Name, c.Host, c.Port, c.Username, c.Domain, c.PasswordEnc, c.Security, c.IgnoreCert).
		Scan(&c.CreatedAt, &c.UpdatedAt)
}

// UpdateConnection replaces the editable fields. keepPassword leaves the
// stored ciphertext untouched (the client didn't send a new password).
func (s *Store) UpdateConnection(ctx context.Context, userID string, c *Connection, keepPassword bool) error {
	tag, err := s.DB.Exec(ctx, `UPDATE connections SET name = $3, host = $4, port = $5, username = $6,
		domain = $7, security = $8, ignore_cert = $9,
		password_enc = CASE WHEN $10 THEN password_enc ELSE $11 END, updated_at = now()
		WHERE user_id = $1 AND id = $2 AND NOT is_deleted`,
		userID, c.ID, c.Name, c.Host, c.Port, c.Username, c.Domain, c.Security, c.IgnoreCert,
		keepPassword, c.PasswordEnc)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
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
		return ErrNotFound
	}
	return nil
}
