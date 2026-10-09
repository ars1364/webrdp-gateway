package httpapi

import (
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/ars1364/webrdp-gateway/backend/internal/auth"
	"github.com/ars1364/webrdp-gateway/backend/internal/core"
)

const cookieName = "__Host-rdpgw_session"

var (
	uuidRe   = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	uuidV4Re = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

func isUUID(s string) bool   { return uuidRe.MatchString(s) }
func isUUIDv4(s string) bool { return uuidV4Re.MatchString(s) }

type authedHandler func(http.ResponseWriter, *http.Request, *core.Session)

func (s *Server) sessionFrom(r *http.Request) (*core.Session, []byte) {
	c, err := r.Cookie(cookieName)
	if err != nil || c.Value == "" || len(c.Value) > 128 {
		return nil, nil
	}
	h := auth.HashToken(c.Value)
	sess, err := s.repo.SessionByHash(r.Context(), h)
	if err != nil {
		return nil, nil
	}
	return sess, h
}

func (s *Server) requireAuth(next authedHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, _ := s.sessionFrom(r)
		if sess == nil {
			writeErr(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Please sign in.")
			return
		}
		next(w, r, sess)
	})
}

// adminOnly is the RBAC gate for operator surfaces (/metrics).
func (s *Server) adminOnly(next authedHandler) authedHandler {
	return func(w http.ResponseWriter, r *http.Request, sess *core.Session) {
		if sess.Role != "admin" {
			writeErr(w, http.StatusForbidden, "FORBIDDEN", "Admins only.")
			return
		}
		next(w, r, sess)
	}
}

// hostValidator rejects requests whose Host header isn't ours (DNS
// rebinding, cache poisoning). /healthz is exempt for the container probe.
func (s *Server) hostValidator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		host := strings.ToLower(r.Host)
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		for _, ok := range s.cfg.AllowedHosts {
			if host == ok {
				next.ServeHTTP(w, r)
				return
			}
		}
		writeErr(w, http.StatusMisdirectedRequest, "BAD_HOST", "Unknown host.")
	})
}

// checkOrigin is the CORS + CSRF policy: same-origin only. No
// Access-Control-* headers are ever sent, so browsers block cross-origin
// reads; state changes additionally need Origin == ALLOWED_ORIGIN.
func (s *Server) checkOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead:
		case http.MethodOptions:
			writeErr(w, http.StatusForbidden, "CORS_DISABLED", "Cross-origin requests are not allowed.")
			return
		default:
			if r.Header.Get("Origin") != s.cfg.AllowedOrigin {
				writeErr(w, http.StatusForbidden, "BAD_ORIGIN", "Cross-origin request refused.")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Cache-Control", "no-store")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}
