package recording

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStore(t *testing.T) {
	root := t.TempDir()
	s := New(root, 8)
	if s == nil {
		t.Fatal("writable root rejected")
	}
	s.interval = 10 * time.Millisecond
	id := "0b2a0616-e470-444a-9670-d7f391386d2e"
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"valid id", id, false},
		{"traversal", "../etc/passwd", true},
		{"not uuid", "abc", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := s.path(tc.id); (err != nil) != tc.wantErr {
				t.Fatalf("err = %v", err)
			}
		})
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	go s.Watch(ctx, id, cancel)
	_ = os.WriteFile(filepath.Join(root, id), make([]byte, 32), 0o600)
	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("cap not enforced")
	}
	if s.Size(id) != 32 {
		t.Fatalf("size %d", s.Size(id))
	}
	if err := s.Remove(id); err != nil || s.Size(id) != 0 {
		t.Fatalf("remove: %v", err)
	}
	if err := s.Remove(id); err != nil {
		t.Fatal("second remove should be a no-op")
	}
	if New(filepath.Join(root, "missing"), 1) != nil {
		t.Fatal("missing root should disable recording")
	}
}
