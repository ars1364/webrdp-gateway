// Package recording owns the guacd session-recording files: a shared volume
// where guacd writes one file per session (named by the recording id) and
// the API sizes, streams, caps and deletes them.
package recording

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

var idRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

var ErrBadID = errors.New("recording: bad id")

type Store struct {
	root     string
	maxBytes int64
	interval time.Duration
}

// New returns nil when root is not writable (recording then unavailable).
func New(root string, maxBytes int64) *Store {
	probe := filepath.Join(root, ".probe")
	if err := os.WriteFile(probe, nil, 0o600); err != nil {
		return nil
	}
	_ = os.Remove(probe)
	return &Store{root: root, maxBytes: maxBytes, interval: 10 * time.Second}
}

func (s *Store) Root() string { return s.root }

func (s *Store) path(id string) (string, error) {
	if !idRe.MatchString(id) {
		return "", ErrBadID
	}
	return filepath.Join(s.root, id), nil
}

func (s *Store) Size(id string) int64 {
	p, err := s.path(id)
	if err != nil {
		return 0
	}
	if fi, err := os.Stat(p); err == nil {
		return fi.Size()
	}
	return 0
}

func (s *Store) Open(id string) (*os.File, error) {
	p, err := s.path(id)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}

func (s *Store) Remove(id string) error {
	p, err := s.path(id)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// Watch ends the session when its recording outgrows the cap.
func (s *Store) Watch(ctx context.Context, id string, cancel context.CancelCauseFunc) {
	t := time.NewTicker(s.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if s.Size(id) > s.maxBytes {
				cancel(fmt.Errorf("Session recording reached its %d MB limit; session closed.", s.maxBytes>>20))
				return
			}
		}
	}
}
