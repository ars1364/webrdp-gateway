package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ars1364/webrdp-gateway/backend/internal/core"
	"github.com/ars1364/webrdp-gateway/backend/internal/guac"
)

// ticketReq: either a saved connection_id, or an ad-hoc target typed in the UI.
type ticketReq struct {
	ConnectionID string   `json:"connection_id"`
	AdHoc        *connReq `json:"ad_hoc"`
}

func (s *Server) createTicket(w http.ResponseWriter, r *http.Request, sess *core.Session) {
	if ok, retry := s.ticketRL.Allow(sess.UserID); !ok {
		w.Header().Set("Retry-After", strconv.Itoa(retry))
		writeErr(w, http.StatusTooManyRequests, "RATE_LIMITED", "Too many connection attempts.")
		return
	}
	var req ticketReq
	if !decode(w, r, &req) {
		return
	}
	var t guac.Target
	var label string
	switch {
	case req.ConnectionID != "" && isUUID(req.ConnectionID):
		c, err := s.repo.GetConnection(r.Context(), sess.UserID, req.ConnectionID)
		if errors.Is(err, core.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "NOT_FOUND", "Connection not found.")
			return
		} else if err != nil {
			s.internal(w, r, err)
			return
		}
		t = guac.Target{Port: c.Port, Username: c.Username, Domain: c.Domain,
			Security: c.Security, IgnoreCert: c.IgnoreCert}
		if c.PasswordEnc != nil {
			pw, err := s.sealer.Open(c.PasswordEnc, []byte("conn:"+c.ID))
			if err != nil {
				s.internal(w, r, err)
				return
			}
			t.Password = string(pw)
		}
		t.IP, label = c.Host, c.Name
	case req.AdHoc != nil:
		if errs := req.AdHoc.validate(false); len(errs) > 0 {
			writeErr(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "Check the highlighted fields.", errs...)
			return
		}
		a := req.AdHoc
		t = guac.Target{IP: a.Host, Port: a.Port, Username: a.Username, Domain: a.Domain,
			Security: a.Security, IgnoreCert: a.IgnoreCert}
		if a.Password != nil {
			t.Password = *a.Password
		}
		label = fmt.Sprintf("%s:%d", a.Host, a.Port)
	default:
		writeErr(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "Give connection_id or ad_hoc.")
		return
	}
	ip, err := s.resolve(r.Context(), t.IP, s.cfg.AllowPrivateTargets)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, "TARGET_REJECTED", err.Error(),
			fieldErr{"host", err.Error()})
		return
	}
	t.IP = ip
	id, err := s.tickets.Put(ticket{userID: sess.UserID, target: t, label: label})
	if err != nil {
		s.internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"ticket": id}})
}

// tunnel upgrades to WebSocket. It needs BOTH the session cookie and a
// single-use ticket owned by the same user.
func (s *Server) tunnel(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.sessionFrom(r)
	if sess == nil {
		writeErr(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Please sign in.")
		return
	}
	q := r.URL.Query()
	tk, ok := s.tickets.Take(q.Get("ticket"))
	if !ok || tk.userID != sess.UserID {
		writeErr(w, http.StatusForbidden, "BAD_TICKET", "Connection ticket invalid or expired.")
		return
	}
	t := tk.target
	t.Width = clamp(atoi(q.Get("width")), 320, 4096, 1280)
	t.Height = clamp(atoi(q.Get("height")), 240, 2160, 720)
	t.DPI = clamp(atoi(q.Get("dpi")), 72, 300, 96)

	if !s.tunnels.Acquire(sess.UserID) {
		writeErr(w, http.StatusTooManyRequests, "TOO_MANY_SESSIONS", "Too many open remote desktops. Close one first.")
		return
	}
	defer s.tunnels.Release(sess.UserID)
	conn, rd, uuid, err := s.dial(r.Context(), s.cfg.GuacdAddr, t, s.features)
	ip := clientIP(r)
	target := fmt.Sprintf("%s:%d", t.IP, t.Port)
	if err != nil {
		s.log.Warn("rdp connect failed", "user", sess.Username, "target", target, "err", err)
		s.repo.Audit(r.Context(), sess.UserID, "rdp.failed", target, ip, map[string]any{"label": tk.label})
		writeErr(w, http.StatusBadGateway, "RDP_CONNECT_FAILED", "Could not reach the remote desktop.")
		return
	}
	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		conn.Close()
		return
	}
	s.repo.Audit(r.Context(), sess.UserID, "rdp.open", target, ip, map[string]any{"label": tk.label})
	s.metrics.Gauge("rdp_sessions_active", 1)
	guac.Bridge(ws, conn, rd, uuid, s.cfg.MaxTunnelDuration)
	s.metrics.Gauge("rdp_sessions_active", -1)
	s.repo.Audit(context.WithoutCancel(r.Context()), sess.UserID, "rdp.close", target, ip, map[string]any{"label": tk.label})
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func clamp(v, lo, hi, def int) int {
	if v == 0 {
		return def
	}
	return min(max(v, lo), hi)
}
