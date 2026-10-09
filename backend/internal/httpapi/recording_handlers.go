package httpapi

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/ars1364/webrdp-gateway/backend/internal/core"
)

func (s *Server) listRecordings(w http.ResponseWriter, r *http.Request, sess *core.Session) {
	p, ok := parsePage(w, r)
	if !ok {
		return
	}
	items, total, err := s.repo.ListRecordings(r.Context(), sess.UserID, p)
	if err != nil {
		s.internal(w, r, err)
		return
	}
	writeList(w, items, total, p)
}

// recordingFile streams the guacd recording for the in-browser player.
func (s *Server) recordingFile(w http.ResponseWriter, r *http.Request, sess *core.Session) {
	id := r.PathValue("id")
	if !isUUID(id) || s.recs == nil {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "Recording not found.")
		return
	}
	if _, err := s.repo.GetRecording(r.Context(), sess.UserID, id); errors.Is(err, core.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "Recording not found.")
		return
	} else if err != nil {
		s.internal(w, r, err)
		return
	}
	f, err := s.recs.Open(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "Recording file is missing.")
		return
	}
	defer f.Close()
	if fi, err := f.Stat(); err == nil {
		w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="webrdp-`+id+`.guac"`)
	s.repo.Audit(r.Context(), sess.UserID, "recording.view", id, clientIP(r), nil)
	_, _ = io.Copy(w, f)
}

// deleteRecording is admin-only: recordings are audit evidence.
func (s *Server) deleteRecording(w http.ResponseWriter, r *http.Request, sess *core.Session) {
	id := r.PathValue("id")
	if !isUUID(id) {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "Recording not found.")
		return
	}
	err := s.repo.DeleteRecording(r.Context(), sess.UserID, id)
	if errors.Is(err, core.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "Recording not found.")
		return
	} else if err != nil {
		s.internal(w, r, err)
		return
	}
	if s.recs != nil {
		_ = s.recs.Remove(id)
	}
	s.repo.Audit(r.Context(), sess.UserID, "recording.delete", id, clientIP(r), nil)
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]bool{"ok": true}})
}
