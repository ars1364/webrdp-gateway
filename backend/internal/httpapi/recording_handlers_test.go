package httpapi

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ars1364/webrdp-gateway/backend/internal/core"
)

// seedRecording stores a row for uid and (optionally) its file.
func (h *harness) seedRecording(uid string, withFile bool) string {
	id := newUUID()
	_ = h.repo.CreateRecording(context.Background(), uid, &core.Recording{ID: id, Label: "x", Target: "203.0.113.1:3389"})
	if withFile {
		_ = os.WriteFile(filepath.Join(h.recDir, id), []byte("4.size,1.0,3.800,3.600;"), 0o600)
	}
	return id
}

func TestRecordingFileAndList(t *testing.T) {
	h := newHarness(t)
	a, _ := h.addUser("a", "pw", "admin")
	b, _ := h.addUser("b", "pw", "user")
	ta, tb := h.session(a), h.session(b)
	mine := h.seedRecording(a, true)
	noFile := h.seedRecording(a, false)

	tests := []struct {
		name string
		path string
		tok  string
		want int
		body string
	}{
		{"owner streams file", "/api/v1/recordings/" + mine + "/file", ta, 200, "4.size"},
		{"other user gets 404 (isolation)", "/api/v1/recordings/" + mine + "/file", tb, 404, ""},
		{"row without file", "/api/v1/recordings/" + noFile + "/file", ta, 404, ""},
		{"non-uuid id", "/api/v1/recordings/not-a-uuid/file", ta, 404, ""},
		{"traversal is normalised away by the mux (redirect, never served)", "/api/v1/recordings/../../etc/passwd/file", ta, 307, ""},
		{"list own", "/api/v1/recordings", ta, 200, `"total":2`},
		{"list other user sees none", "/api/v1/recordings", tb, 200, `"total":0`},
		{"no session", "/api/v1/recordings", "", 401, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var opts []reqOpt
			if tc.tok != "" {
				opts = append(opts, withCookie(tc.tok))
			}
			w := h.do(http.MethodGet, tc.path, nil, opts...)
			if w.Code != tc.want || !strings.Contains(w.Body.String(), tc.body) {
				t.Fatalf("got %d %q", w.Code, w.Body.String())
			}
		})
	}
}

func TestDeleteRecordingIsAdminOnly(t *testing.T) {
	h := newHarness(t)
	a, _ := h.addUser("a", "pw", "admin")
	u, _ := h.addUser("u", "pw", "user")
	recA, recU := h.seedRecording(a, true), h.seedRecording(u, true)
	tests := []struct {
		name string
		uid  string
		id   string
		want int
	}{
		{"user cannot delete own recording (audit evidence)", u, recU, 403},
		{"admin deletes own", a, recA, 200},
		{"admin delete again → 404", a, recA, 404},
		{"admin cannot delete another user's row", a, recU, 404},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := h.do(http.MethodDelete, "/api/v1/recordings/"+tc.id, nil, withCookie(h.session(tc.uid)))
			if w.Code != tc.want {
				t.Fatalf("got %d: %s", w.Code, w.Body)
			}
		})
	}
	if _, err := os.Stat(filepath.Join(h.recDir, recA)); !os.IsNotExist(err) {
		t.Fatal("recording file not removed")
	}
}
