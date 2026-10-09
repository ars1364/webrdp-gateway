package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ars1364/webrdp-gateway/backend/internal/auth"
	"github.com/ars1364/webrdp-gateway/backend/internal/config"
	"github.com/ars1364/webrdp-gateway/backend/internal/core"
	"github.com/ars1364/webrdp-gateway/backend/internal/guac"
	"github.com/ars1364/webrdp-gateway/backend/internal/metrics"
	"github.com/ars1364/webrdp-gateway/backend/internal/netguard"
	"github.com/ars1364/webrdp-gateway/backend/internal/recording"
	"github.com/ars1364/webrdp-gateway/backend/internal/seal"
)

const (
	testOrigin = "https://rdp.test"
	testHost   = "rdp.test"
)

type harness struct {
	t      *testing.T
	repo   *fakeRepo
	sealer *seal.Sealer
	recDir string
	h      http.Handler
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	sealer, err := seal.New(1, map[byte][]byte{1: bytes.Repeat([]byte{7}, 32)})
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		AllowedOrigin: testOrigin, AllowedHosts: []string{testHost}, CookieSecure: true,
		SessionTTL: time.Hour, MaxTunnels: 2, MaxTunnelsPerUser: 1, MaxTunnelDuration: time.Minute,
		IdempotencyTTL: time.Hour, GuacdAddr: "guacd.invalid:4822", ClipboardUpload: true,
	}
	repo := newFakeRepo()
	failDial := func(context.Context, string, guac.Target, guac.Features) (net.Conn, *guac.Reader, string, error) {
		return nil, nil, "", errors.New("no guacd in tests")
	}
	recDir := t.TempDir()
	srv := New(cfg, Deps{Repo: repo, Sealer: sealer, Dial: failDial, Resolve: netguard.Resolve,
		Recs: recording.New(recDir, 1<<20), Metrics: metrics.New(), Log: slog.New(slog.NewTextHandler(io.Discard, nil))})
	return &harness{t: t, repo: repo, sealer: sealer, recDir: recDir, h: srv.Routes()}
}

// addUser stores a user and returns its TOTP secret.
func (h *harness) addUser(name, pw, role string) (id, secret string) {
	h.t.Helper()
	id = newUUID()
	secret, _ = auth.NewTOTPSecret()
	hash, _ := auth.HashPassword(pw)
	enc, _ := h.sealer.Seal([]byte(secret), []byte("totp:"+id))
	h.repo.users[strings.ToLower(name)] = &core.User{ID: id, Username: name, PasswordHash: hash,
		TOTPSecretEnc: enc, Role: role}
	return id, secret
}

// session returns a valid session cookie value for user id.
func (h *harness) session(id string) string {
	token, hash, _ := auth.NewToken()
	_ = h.repo.CreateSession(context.Background(), hash, id, "", "", time.Hour)
	return token
}

type reqOpt func(*http.Request)

func withCookie(tok string) reqOpt {
	return func(r *http.Request) { r.AddCookie(&http.Cookie{Name: cookieName, Value: tok}) }
}
func withHeader(k, v string) reqOpt { return func(r *http.Request) { r.Header.Set(k, v) } }
func withHost(host string) reqOpt   { return func(r *http.Request) { r.Host = host } }

// do sends a request with the right Host and, for mutations, the right
// Origin and a fresh Idempotency-Key; opts may override any of them.
func (h *harness) do(method, path string, body any, opts ...reqOpt) *httptest.ResponseRecorder {
	h.t.Helper()
	var rdr io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		rdr = strings.NewReader(b)
	default:
		j, _ := json.Marshal(b)
		rdr = bytes.NewReader(j)
	}
	r := httptest.NewRequest(method, path, rdr)
	r.Host = testHost
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	if method != http.MethodGet {
		r.Header.Set("Origin", testOrigin)
		r.Header.Set(idemHeader, newUUID())
	}
	for _, o := range opts {
		o(r)
	}
	w := httptest.NewRecorder()
	h.h.ServeHTTP(w, r)
	return w
}

// errCode extracts error.code from the envelope ("" if none).
func errCode(w *httptest.ResponseRecorder) string {
	var env struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	return env.Error.Code
}
