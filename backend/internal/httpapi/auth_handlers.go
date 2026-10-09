package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/ars1364/webrdp-gateway/backend/internal/auth"
	"github.com/ars1364/webrdp-gateway/backend/internal/core"
)

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	TOTP     string `json:"totp"`
}

// login is one step: username + password + TOTP. Every failure returns the
// same message so an attacker can't tell which factor was wrong.
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if ok, retry := s.loginRL.Allow(ip); !ok {
		w.Header().Set("Retry-After", strconv.Itoa(retry))
		writeErr(w, http.StatusTooManyRequests, "RATE_LIMITED", "Too many attempts. Try again later.")
		return
	}
	var req loginReq
	if !decode(w, r, &req) {
		return
	}
	if len(req.Username) > 64 || len(req.Password) > 256 {
		s.denyLogin(w, r, "", req.Username)
		return
	}
	ctx := r.Context()
	u, err := s.repo.UserByName(ctx, req.Username)
	if errors.Is(err, core.ErrNotFound) {
		auth.DummyVerify(req.Password)
		s.denyLogin(w, r, "", req.Username)
		return
	} else if err != nil {
		s.internal(w, r, err)
		return
	}
	if u.LockedUntil != nil && u.LockedUntil.After(time.Now()) {
		s.repo.Audit(ctx, u.ID, "login.locked", u.Username, ip, nil)
		writeErr(w, http.StatusLocked, "ACCOUNT_LOCKED", "Account temporarily locked after failed attempts.")
		return
	}
	pwOK, err := auth.VerifyPassword(req.Password, u.PasswordHash)
	if err != nil {
		s.internal(w, r, err)
		return
	}
	secret, err := s.sealer.Open(u.TOTPSecretEnc, []byte("totp:"+u.ID))
	if err != nil {
		s.internal(w, r, err)
		return
	}
	step, totpOK := auth.CheckTOTP(string(secret), req.TOTP, time.Now())
	if !pwOK || !totpOK || step <= u.LastTOTPStep {
		_ = s.repo.RecordFailedLogin(ctx, u.ID)
		s.denyLogin(w, r, u.ID, u.Username)
		return
	}
	if err := s.repo.RecordLogin(ctx, u.ID, step); err != nil {
		s.denyLogin(w, r, u.ID, u.Username) // concurrent replay of the same code
		return
	}
	token, hash, err := auth.NewToken()
	if err != nil {
		s.internal(w, r, err)
		return
	}
	if err := s.repo.CreateSession(ctx, hash, u.ID, ip, truncate(r.UserAgent(), 256), s.cfg.SessionTTL); err != nil {
		s.internal(w, r, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: token, Path: "/", HttpOnly: true, Secure: s.cfg.CookieSecure,
		SameSite: http.SameSiteStrictMode, MaxAge: int(s.cfg.SessionTTL.Seconds()),
	})
	s.repo.Audit(ctx, u.ID, "login.success", u.Username, ip, nil)
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"username": u.Username, "role": u.Role}})
}

func (s *Server) denyLogin(w http.ResponseWriter, r *http.Request, userID, username string) {
	s.metrics.Inc("login_failures_total")
	s.repo.Audit(r.Context(), userID, "login.failed", truncate(username, 64), clientIP(r), nil)
	writeErr(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid username, password or code.")
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request, sess *core.Session) {
	if _, h := s.sessionFrom(r); h != nil {
		_ = s.repo.DeleteSession(r.Context(), h)
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteStrictMode})
	s.repo.Audit(r.Context(), sess.UserID, "logout", sess.Username, clientIP(r), nil)
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]bool{"ok": true}})
}

func (s *Server) me(w http.ResponseWriter, _ *http.Request, sess *core.Session) {
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{
		"username": sess.Username, "role": sess.Role, "expires_at": sess.ExpiresAt,
		"features": map[string]bool{
			"clipboard_upload":   s.features.ClipboardUpload,
			"clipboard_download": s.features.ClipboardDownload,
			"file_transfer":      s.features.FileTransfer,
		},
	}})
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
