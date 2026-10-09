// Package drive manages the per-session "Transfer" folders that guacd
// exposes to Windows as a redirected drive. The folders live on a volume
// shared by guacd and the API; the API owns their life cycle and size cap.
package drive

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type Manager struct {
	root     string
	maxBytes int64
	interval time.Duration
}

// New returns nil when root is missing or not writable: file transfer is
// then disabled instead of failing sessions.
func New(root string, maxBytes int64) *Manager {
	probe := filepath.Join(root, ".probe")
	if err := os.WriteFile(probe, nil, 0o600); err != nil {
		return nil
	}
	_ = os.Remove(probe)
	return &Manager{root: root, maxBytes: maxBytes, interval: 5 * time.Second}
}

// PurgeAll removes every session folder (call at startup: none are live).
func (m *Manager) PurgeAll() error {
	entries, err := os.ReadDir(m.root)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(m.root, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// Prepare creates the folder for one session. id must be server-generated.
func (m *Manager) Prepare(id string) (string, error) {
	p := filepath.Join(m.root, id)
	if filepath.Dir(p) != filepath.Clean(m.root) {
		return "", fmt.Errorf("drive: bad session id")
	}
	return p, os.Mkdir(p, 0o700)
}

func (m *Manager) Remove(path string) error { return os.RemoveAll(path) }

// Watch cancels the session when its folder grows past the cap.
func (m *Manager) Watch(ctx context.Context, path string, cancel context.CancelCauseFunc) {
	t := time.NewTicker(m.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if Size(path) > m.maxBytes {
				cancel(fmt.Errorf("Transfer drive is over the %d MB limit; session closed.", m.maxBytes>>20))
				return
			}
		}
	}
}

// Size is the total size of regular files under path.
func Size(path string) int64 {
	var total int64
	_ = filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			if info, err := d.Info(); err == nil {
				total += info.Size()
			}
		}
		return nil
	})
	return total
}
