// Package httpapi exposes the JSON API and the guacamole WebSocket tunnel.
package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"github.com/ars1364/webrdp-gateway/backend/internal/config"
	"github.com/ars1364/webrdp-gateway/backend/internal/guac"
	"github.com/ars1364/webrdp-gateway/backend/internal/seal"
	"github.com/ars1364/webrdp-gateway/backend/internal/store"
)

type Server struct {
	cfg      *config.Config
	store    *store.Store
	sealer   *seal.Sealer
	tickets  *ticketStore
	loginRL  *rateLimiter
	ticketRL *rateLimiter
	upgrader websocket.Upgrader
	features guac.Features
	log      *slog.Logger
}

func New(cfg *config.Config, st *store.Store, s *seal.Sealer, log *slog.Logger) *Server {
	srv := &Server{
		cfg:      cfg,
		store:    st,
		sealer:   s,
		tickets:  newTicketStore(30 * time.Second),
		loginRL:  newRateLimiter(10, time.Minute),
		ticketRL: newRateLimiter(30, time.Minute),
		features: guac.Features{Clipboard: false, FileTransfer: false},
		log:      log,
	}
	srv.upgrader = websocket.Upgrader{
		ReadBufferSize:  16 << 10,
		WriteBufferSize: 64 << 10,
		Subprotocols:    []string{"guacamole"},
		CheckOrigin:     func(r *http.Request) bool { return r.Header.Get("Origin") == cfg.AllowedOrigin },
	}
	return srv
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /api/v1/auth/login", s.login)
	mux.Handle("POST /api/v1/auth/logout", s.requireAuth(s.logout))
	mux.Handle("GET /api/v1/auth/me", s.requireAuth(s.me))

	mux.Handle("GET /api/v1/connections", s.requireAuth(s.listConnections))
	mux.Handle("POST /api/v1/connections", s.requireAuth(s.createConnection))
	mux.Handle("PUT /api/v1/connections/{id}", s.requireAuth(s.updateConnection))
	mux.Handle("DELETE /api/v1/connections/{id}", s.requireAuth(s.deleteConnection))

	mux.Handle("POST /api/v1/tunnel/ticket", s.requireAuth(s.createTicket))
	mux.HandleFunc("GET /api/v1/tunnel", s.tunnel) // auth = single-use ticket + session cookie
	mux.Handle("GET /api/v1/audit", s.requireAuth(s.listAudit))

	var h http.Handler = mux
	h = s.checkOrigin(h)
	h = securityHeaders(h)
	h = s.recoverer(h)
	h = s.logRequests(h)
	return h
}

func (s *Server) listAudit(w http.ResponseWriter, r *http.Request, sess *store.Session) {
	items, err := s.store.ListAudit(r.Context(), sess.UserID, 200)
	if err != nil {
		s.internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items, "meta": map[string]int{"total": len(items)}})
}
