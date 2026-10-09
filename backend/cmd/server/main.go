// Command server runs the webrdp-gateway API.
//
//	server                       run the HTTP API (default)
//	server create-admin -u NAME  create a user, print password + TOTP seed as JSON
//	server healthcheck           exit 0 if the local API answers /healthz
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ars1364/webrdp-gateway/backend/internal/config"
	"github.com/ars1364/webrdp-gateway/backend/internal/db"
	"github.com/ars1364/webrdp-gateway/backend/internal/httpapi"
	"github.com/ars1364/webrdp-gateway/backend/internal/seal"
	"github.com/ars1364/webrdp-gateway/backend/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	if cmd == "healthcheck" {
		os.Exit(healthcheck())
	}
	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		log.Error("migrate", "err", err)
		os.Exit(1)
	}
	sealer, err := seal.New(cfg.KEK)
	if err != nil {
		log.Error("kek", "err", err)
		os.Exit(1)
	}
	st := &store.Store{DB: pool}

	switch cmd {
	case "create-admin":
		os.Exit(createAdmin(ctx, st, sealer, os.Args[2:]))
	case "serve":
	default:
		log.Error("unknown command", "cmd", cmd)
		os.Exit(2)
	}

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           httpapi.New(cfg, st, sealer, log).Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    32 << 10,
	}
	go purgeSessions(ctx, st)
	go func() {
		log.Info("listening", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("listen", "err", err)
			stop()
		}
	}()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
}

func purgeSessions(ctx context.Context, st *store.Store) {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_ = st.PurgeExpiredSessions(ctx)
		}
	}
}

func healthcheck() int {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" || addr[0] == ':' {
		addr = "127.0.0.1" + addr
		if addr == "127.0.0.1" {
			addr += ":8080"
		}
	}
	c := http.Client{Timeout: 3 * time.Second}
	resp, err := c.Get("http://" + addr + "/healthz")
	if err != nil || resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
