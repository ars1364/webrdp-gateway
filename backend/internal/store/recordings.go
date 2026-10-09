package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/ars1364/webrdp-gateway/backend/internal/core"
)

const recCols = `id, label, target, size_bytes, started_at, ended_at`

func scanRec(row pgx.Row) (*core.Recording, error) {
	r := &core.Recording{}
	err := row.Scan(&r.ID, &r.Label, &r.Target, &r.SizeBytes, &r.StartedAt, &r.EndedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, core.ErrNotFound
	}
	return r, err
}

func (s *Store) CreateRecording(ctx context.Context, userID string, r *core.Recording) error {
	return s.DB.QueryRow(ctx, `INSERT INTO recordings (id, user_id, label, target, created_by)
		VALUES ($2, $1, $3, $4, $1) RETURNING started_at`, userID, r.ID, r.Label, r.Target).Scan(&r.StartedAt)
}

func (s *Store) FinishRecording(ctx context.Context, userID, id string, size int64) error {
	_, err := s.DB.Exec(ctx, `UPDATE recordings SET ended_at = now(), size_bytes = $3
		WHERE user_id = $1 AND id = $2`, userID, id, size)
	return err
}

func (s *Store) ListRecordings(ctx context.Context, userID string, p core.Page) ([]core.Recording, int, error) {
	var total int
	if err := s.DB.QueryRow(ctx, `SELECT count(*) FROM recordings WHERE user_id = $1 AND NOT is_deleted`,
		userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.DB.Query(ctx, `SELECT `+recCols+` FROM recordings WHERE user_id = $1 AND NOT is_deleted
		ORDER BY started_at DESC, id LIMIT $2 OFFSET $3`, userID, p.PerPage, p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []core.Recording{}
	for rows.Next() {
		r, err := scanRec(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *r)
	}
	return out, total, rows.Err()
}

func (s *Store) GetRecording(ctx context.Context, userID, id string) (*core.Recording, error) {
	return scanRec(s.DB.QueryRow(ctx, `SELECT `+recCols+` FROM recordings
		WHERE user_id = $1 AND id = $2 AND NOT is_deleted`, userID, id))
}

func (s *Store) DeleteRecording(ctx context.Context, userID, id string) error {
	tag, err := s.DB.Exec(ctx, `UPDATE recordings SET is_deleted = TRUE, deleted_at = now()
		WHERE user_id = $1 AND id = $2 AND NOT is_deleted`, userID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return core.ErrNotFound
	}
	return nil
}

// ExpireRecordings soft-deletes recordings older than days and returns their
// ids so the caller can delete the files.
// unscoped: retention job, spans all users by design.
func (s *Store) ExpireRecordings(ctx context.Context, days int) ([]string, error) {
	rows, err := s.DB.Query(ctx, `UPDATE recordings SET is_deleted = TRUE, deleted_at = now()
		WHERE NOT is_deleted AND started_at < now() - make_interval(days => $1) RETURNING id::text`, days)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}
