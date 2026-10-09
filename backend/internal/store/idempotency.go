package store

import (
	"context"
	"time"

	"github.com/ars1364/webrdp-gateway/backend/internal/core"
)

// IdemBegin claims (user, key) with an in-progress row. If the key exists it
// returns the stored record instead; the caller compares request hashes.
func (s *Store) IdemBegin(ctx context.Context, userID, key, reqHash string, ttl time.Duration) (*core.IdemRecord, bool, error) {
	tag, err := s.DB.Exec(ctx, `INSERT INTO idempotency_keys (user_id, key, request_hash, expires_at)
		VALUES ($1, $2, $3, $4) ON CONFLICT (user_id, key) DO NOTHING`,
		userID, key, reqHash, time.Now().Add(ttl))
	if err != nil {
		return nil, false, err
	}
	if tag.RowsAffected() == 1 {
		return nil, true, nil
	}
	rec := &core.IdemRecord{}
	err = s.DB.QueryRow(ctx, `SELECT request_hash, status, body FROM idempotency_keys
		WHERE user_id = $1 AND key = $2`, userID, key).Scan(&rec.RequestHash, &rec.Status, &rec.Body)
	if err != nil {
		return nil, false, err
	}
	return rec, false, nil
}

func (s *Store) IdemFinish(ctx context.Context, userID, key string, status int, body []byte) error {
	_, err := s.DB.Exec(ctx, `UPDATE idempotency_keys SET status = $3, body = $4
		WHERE user_id = $1 AND key = $2`, userID, key, status, body)
	return err
}

// IdemAbort releases a key whose request failed server-side, so the client
// may retry with the same key.
func (s *Store) IdemAbort(ctx context.Context, userID, key string) error {
	_, err := s.DB.Exec(ctx, `DELETE FROM idempotency_keys WHERE user_id = $1 AND key = $2 AND status = 0`,
		userID, key)
	return err
}
