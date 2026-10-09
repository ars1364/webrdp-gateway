package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ars1364/webrdp-gateway/backend/internal/core"
)

func (s *Store) CreateSession(ctx context.Context, hash []byte, userID, ip, ua string, ttl time.Duration) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO sessions (token_hash, user_id, ip, user_agent, expires_at)
		VALUES ($1, $2, $3, $4, $5)`, hash, userID, ip, ua, time.Now().Add(ttl))
	return err
}

func (s *Store) SessionByHash(ctx context.Context, hash []byte) (*core.Session, error) {
	se := &core.Session{}
	err := s.DB.QueryRow(ctx, `SELECT s.user_id, u.username, u.role, s.expires_at
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > now() AND NOT u.is_deleted`, hash).
		Scan(&se.UserID, &se.Username, &se.Role, &se.ExpiresAt)
	if err != nil {
		return nil, core.ErrNotFound
	}
	return se, nil
}

func (s *Store) DeleteSession(ctx context.Context, hash []byte) error {
	_, err := s.DB.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, hash)
	return err
}

// Reap enforces retention: expired sessions and idempotency keys go at
// once; audit rows after auditDays.
// unscoped: retention job, spans all users by design.
func (s *Store) Reap(ctx context.Context, auditDays int) error {
	if _, err := s.DB.Exec(ctx, `DELETE FROM sessions WHERE expires_at < now()`); err != nil {
		return err
	}
	if _, err := s.DB.Exec(ctx, `DELETE FROM idempotency_keys WHERE expires_at < now()`); err != nil {
		return err
	}
	_, err := s.DB.Exec(ctx, `DELETE FROM audit_log WHERE created_at < now() - make_interval(days => $1)`, auditDays)
	return err
}

// Audit appends one row. Audit is append-only: there is no update path.
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

func (s *Store) ListAudit(ctx context.Context, userID string, p core.Page) ([]core.AuditEntry, int, error) {
	var total int
	if err := s.DB.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.DB.Query(ctx, `SELECT action, target, ip, detail, created_at FROM audit_log
		WHERE user_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2 OFFSET $3`, userID, p.PerPage, p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []core.AuditEntry{}
	for rows.Next() {
		var e core.AuditEntry
		if err := rows.Scan(&e.Action, &e.Target, &e.IP, &e.Detail, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}
