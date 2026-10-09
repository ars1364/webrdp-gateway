package httpapi

import (
	"bufio"
	"net"
	"net/http"
	"time"

	"github.com/ars1364/webrdp-gateway/backend/internal/auth"
	"github.com/ars1364/webrdp-gateway/backend/internal/store"
)

const cookieName = "__Host-rdpgw_session"

type authedHandler func(http.ResponseWriter, *http.Request, *store.Session)

func (s *Server) sessionFrom(r *http.Request) (*store.Session, []byte) {
	c, err := r.Cookie(cookieName)
	if err != nil || c.Value == "" || len(c.Value) > 128 {
		return nil, nil
	}
	h := auth.HashToken(c.Value)
	sess, err := s.store.SessionByHash(r.Context(), h)
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

// checkOrigin blocks cross-site state changes. The session cookie is
// SameSite=Strict already; this is the second, independent layer.
func (s *Server) checkOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
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

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				s.log.Error("panic", "path", r.URL.Path, "panic", v)
				writeErr(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

// Unwrap lets http.ResponseController / the WebSocket upgrader reach Hijack.
func (sw *statusWriter) Unwrap() http.ResponseWriter { return sw.ResponseWriter }

func (sw *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(sw.ResponseWriter).Hijack()
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		s.log.Info("http", "method", r.Method, "path", r.URL.Path, "status", sw.status,
			"ms", time.Since(start).Milliseconds(), "ip", clientIP(r))
	})
}

// clientIP trusts X-Real-IP because the API only listens on the internal
// docker network behind nginx, which sets it from CF-Connecting-IP.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" && net.ParseIP(ip) != nil {
		return ip
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}
