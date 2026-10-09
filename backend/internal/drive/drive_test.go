package drive

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestManager(t *testing.T) {
	root := t.TempDir()
	m := New(root, 10)
	if m == nil {
		t.Fatal("writable root rejected")
	}
	m.interval = 10 * time.Millisecond
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"uuid", "0b2a0616-e470-444a-9670-d7f391386d2e", false},
		{"traversal", "../escape", true},
		{"nested", "a/b", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := m.Prepare(tc.id)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v", err)
			}
		})
	}

	p := filepath.Join(root, "0b2a0616-e470-444a-9670-d7f391386d2e")
	ctx, cancel := context.WithCancelCause(context.Background())
	go m.Watch(ctx, p, cancel)
	_ = os.WriteFile(filepath.Join(p, "big.bin"), make([]byte, 64), 0o600)
	select {
	case <-ctx.Done():
		if !strings.Contains(context.Cause(ctx).Error(), "limit") {
			t.Fatalf("cause %v", context.Cause(ctx))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("size cap not enforced")
	}
	if err := m.PurgeAll(); err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(root); len(entries) != 0 {
		t.Fatalf("%d entries left after purge", len(entries))
	}
	if New(filepath.Join(root, "missing"), 1) != nil {
		t.Fatal("missing root should disable file transfer")
	}
}
