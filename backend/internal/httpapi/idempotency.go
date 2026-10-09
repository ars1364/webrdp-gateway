package httpapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"

	"github.com/ars1364/webrdp-gateway/backend/internal/core"
)

const idemHeader = "Idempotency-Key"

// bufferedWriter captures a handler's response so it can be stored.
type bufferedWriter struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (b *bufferedWriter) Header() http.Header         { return b.header }
func (b *bufferedWriter) Write(p []byte) (int, error) { return b.body.Write(p) }
func (b *bufferedWriter) WriteHeader(code int)        { b.status = code }

// idempotent dedups mutations on Idempotency-Key (UUID v4, required). A
// replay with the same body returns the stored status + body and the header
// Idempotent-Replayed: true; the same key with a different body is 422; a
// replay while the first request still runs is 409.
func (s *Server) idempotent(next authedHandler) authedHandler {
	return func(w http.ResponseWriter, r *http.Request, sess *core.Session) {
		key := r.Header.Get(idemHeader)
		if !isUUIDv4(key) {
			writeErr(w, http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED",
				"Send a UUID v4 Idempotency-Key header on every mutation.")
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
		if err != nil {
			writeErr(w, http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE", "Request body too large.")
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		sum := sha256.Sum256(append([]byte(r.Method+" "+r.URL.Path+"\n"), body...))
		hash := hex.EncodeToString(sum[:])

		ctx := r.Context()
		rec, owned, err := s.repo.IdemBegin(ctx, sess.UserID, key, hash, s.cfg.IdempotencyTTL)
		if err != nil {
			s.internal(w, r, err)
			return
		}
		if !owned {
			switch {
			case rec.RequestHash != hash:
				writeErr(w, http.StatusUnprocessableEntity, "IDEMPOTENCY_KEY_REUSED",
					"This Idempotency-Key was used for a different request.")
			case rec.Status == 0:
				writeErr(w, http.StatusConflict, "IDEMPOTENCY_IN_PROGRESS", "The original request is still running.")
			default:
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.Header().Set("Idempotent-Replayed", "true")
				w.WriteHeader(rec.Status)
				_, _ = w.Write(rec.Body)
			}
			return
		}

		buf := &bufferedWriter{header: w.Header(), status: http.StatusOK}
		next(buf, r, sess)
		if buf.status >= 500 {
			_ = s.repo.IdemAbort(ctx, sess.UserID, key) // let the client retry
		} else if err := s.repo.IdemFinish(ctx, sess.UserID, key, buf.status, buf.body.Bytes()); err != nil {
			s.log.Error("idempotency finish", "correlation_id", correlationFrom(ctx), "err", err)
		}
		w.WriteHeader(buf.status)
		_, _ = w.Write(buf.body.Bytes())
	}
}
