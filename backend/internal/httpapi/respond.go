package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type fieldErr struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeErr emits the standard envelope:
// {"error":{"code":"SCREAMING_SNAKE","message":"human","details":[...]}}
func writeErr(w http.ResponseWriter, status int, code, msg string, details ...fieldErr) {
	if details == nil {
		details = []fieldErr{}
	}
	writeJSON(w, status, map[string]any{"error": map[string]any{
		"code": code, "message": msg, "details": details,
	}})
}

func (s *Server) internal(w http.ResponseWriter, r *http.Request, err error) {
	s.log.Error("internal error", "path", r.URL.Path, "err", err)
	writeErr(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong.")
}

// decode reads a JSON body (max 64 KiB) and rejects unknown fields, so a
// client can never set fields the DTO doesn't declare (mass assignment).
func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		writeErr(w, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "Send application/json.")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			writeErr(w, http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE", "Request body too large.")
			return false
		}
		writeErr(w, http.StatusBadRequest, "BAD_JSON", "Malformed JSON body.")
		return false
	}
	return true
}
