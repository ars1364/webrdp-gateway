package store

import (
	"context"
	"encoding/json"
	"time"
)

type Session struct {
	UserID    string
	Username  string
	Role      string
	ExpiresAt time.Time
}

func (s *Store) CreateSession(ctx context.Context, hash []byte, userID, ip, ua string, ttl time.Duration) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO sessions (token_hash, user_id, ip, user_agent, expires_at)
		VALUES ($1, $2, $3, $4, $5)`, hash, userID, ip, ua, time.Now().Add(ttl))
	return err
}

func (s *Store) SessionByHash(ctx context.Context, hash []byte) (*Session, error) {
	se := &Session{}
	err := s.DB.QueryRow(ctx, `SELECT s.user_id, u.username, u.role, s.expires_at
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > now() AND NOT u.is_deleted`, hash).
		Scan(&se.UserID, &se.Username, &se.Role, &se.ExpiresAt)
	if err != nil {
		return nil, ErrNotFound
	}
	return se, nil
}

func (s *Store) DeleteSession(ctx context.Context, hash []byte) error {
	_, err := s.DB.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, hash)
	return err
}

func (s *Store) PurgeExpiredSessions(ctx context.Context) error {
	_, err := s.DB.Exec(ctx, `DELETE FROM sessions WHERE expires_at < now()`)
	return err
}

// Audit writes one audit row. userID may be empty for anonymous events.
func (s *Store) Audit(ctx context.Context, userID, action, target, ip string, detail map[string]any) {
	if detail == nil {
		detail = map[string]any{}
	}
	b, _ := json.Marshal(detail)
	var uid any
	if userID != "" {
		uid = userID
	}
	_, _ = s.DB.Exec(ctx, `INSERT INTO audit_log (user_id, action, target, ip, detail)
		VALUES ($1, $2, $3, $4, $5)`, uid, action, target, ip, b)
}

type AuditEntry struct {
	Action    string         `json:"action"`
	Target    string         `json:"target"`
	IP        string         `json:"ip"`
	Detail    map[string]any `json:"detail"`
	CreatedAt time.Time      `json:"created_at"`
}

func (s *Store) ListAudit(ctx context.Context, userID string, limit int) ([]AuditEntry, error) {
	rows, err := s.DB.Query(ctx, `SELECT action, target, ip, detail, created_at FROM audit_log
		WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AuditEntry{}
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.Action, &e.Target, &e.IP, &e.Detail, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
