// Package httpapi is the driving adapter: JSON API + guacamole WebSocket.
// It depends on core ports only, so handlers are tested with fakes.
package httpapi

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"github.com/ars1364/webrdp-gateway/backend/internal/config"
	"github.com/ars1364/webrdp-gateway/backend/internal/core"
	"github.com/ars1364/webrdp-gateway/backend/internal/drive"
	"github.com/ars1364/webrdp-gateway/backend/internal/guac"
	"github.com/ars1364/webrdp-gateway/backend/internal/metrics"
	"github.com/ars1364/webrdp-gateway/backend/internal/seal"
)

// DialFunc opens an RDP session through guacd (port for the guac adapter).
type DialFunc func(ctx context.Context, addr string, t guac.Target, f guac.Features) (net.Conn, *guac.Reader, string, error)

// ResolveFunc validates and resolves a user-supplied target host.
type ResolveFunc func(ctx context.Context, host string, allowPrivate bool) (string, error)

type Server struct {
	cfg      *config.Config
	repo     core.Repository
	sealer   *seal.Sealer
	dial     DialFunc
	resolve  ResolveFunc
	tickets  *ticketStore
	tunnels  *tunnelLimiter
	loginRL  *rateLimiter
	ticketRL *rateLimiter
	upgrader websocket.Upgrader
	features guac.Features
	drives   *drive.Manager // nil = file transfer unavailable
	metrics  *metrics.Registry
	log      *slog.Logger
}

type Deps struct {
	Repo    core.Repository
	Sealer  *seal.Sealer
	Dial    DialFunc
	Resolve ResolveFunc
	Drives  *drive.Manager
	Metrics *metrics.Registry
	Log     *slog.Logger
}

func New(cfg *config.Config, d Deps) *Server {
	s := &Server{
		cfg: cfg, repo: d.Repo, sealer: d.Sealer, dial: d.Dial, resolve: d.Resolve, drives: d.Drives,
		metrics:  d.Metrics,
		log:      d.Log,
		tickets:  newTicketStore(30 * time.Second),
		tunnels:  newTunnelLimiter(cfg.MaxTunnels, cfg.MaxTunnelsPerUser),
		loginRL:  newRateLimiter(10, time.Minute),
		ticketRL: newRateLimiter(30, time.Minute),
		features: guac.Features{ClipboardUpload: cfg.ClipboardUpload, ClipboardDownload: cfg.ClipboardDownload,
			FileUpload: cfg.FileUpload, FileDownload: cfg.FileDownload},
	}
	s.upgrader = websocket.Upgrader{
		ReadBufferSize:  16 << 10,
		WriteBufferSize: 64 << 10,
		Subprotocols:    []string{"guacamole"},
		CheckOrigin:     func(r *http.Request) bool { return r.Header.Get("Origin") == cfg.AllowedOrigin },
	}
	s.metrics.Help("http_requests_total", "HTTP requests by method and status class.")
	s.metrics.Help("rdp_sessions_active", "Open RDP tunnels.")
	s.metrics.Help("login_failures_total", "Rejected login attempts.")
	s.metrics.Help("file_transfers_total", "Files moved through the Transfer drive, by direction.")
	if s.drives == nil {
		s.features.FileUpload, s.features.FileDownload = false, false
	}
	return s
}

// Routes wires the middleware stack in the policy order:
// CorrelationID → Logger → Recovery → HostValidator → Origin/CORS → (per
// route) Auth → RateLimit → Idempotency.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.Handle("GET /metrics", s.requireAuth(s.adminOnly(s.serveMetrics)))

	mux.HandleFunc("POST /api/v1/auth/login", s.login)
	mux.Handle("POST /api/v1/auth/logout", s.requireAuth(s.logout))
	mux.Handle("GET /api/v1/auth/me", s.requireAuth(s.me))

	mux.Handle("GET /api/v1/connections", s.requireAuth(s.listConnections))
	mux.Handle("POST /api/v1/connections", s.requireAuth(s.idempotent(s.createConnection)))
	mux.Handle("PUT /api/v1/connections/{id}", s.requireAuth(s.idempotent(s.updateConnection)))
	mux.Handle("DELETE /api/v1/connections/{id}", s.requireAuth(s.idempotent(s.deleteConnection)))

	mux.Handle("POST /api/v1/tunnel/ticket", s.requireAuth(s.createTicket))
	mux.HandleFunc("GET /api/v1/tunnel", s.tunnel) // session cookie + single-use ticket
	mux.Handle("GET /api/v1/audit", s.requireAuth(s.listAudit))

	var h http.Handler = mux
	h = securityHeaders(h)
	h = s.checkOrigin(h)
	h = s.hostValidator(h)
	h = s.recoverer(h)
	h = s.logRequests(h)
	h = correlationID(h)
	return h
}

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) serveMetrics(w http.ResponseWriter, _ *http.Request, _ *core.Session) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	s.metrics.Write(w)
}

func (s *Server) listAudit(w http.ResponseWriter, r *http.Request, sess *core.Session) {
	p, ok := parsePage(w, r)
	if !ok {
		return
	}
	items, total, err := s.repo.ListAudit(r.Context(), sess.UserID, p)
	if err != nil {
		s.internal(w, r, err)
		return
	}
	writeList(w, items, total, p)
}
