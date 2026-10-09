package httpapi

import (
	"net/http"
	"strings"
	"testing"
)

func TestEdgeMiddleware(t *testing.T) {
	h := newHarness(t)
	id, _ := h.addUser("a", "pw", "admin")
	uid, _ := h.addUser("u", "pw", "user")
	admin, user := h.session(id), h.session(uid)

	tests := []struct {
		name     string
		method   string
		path     string
		opts     []reqOpt
		want     int
		wantCode string
	}{
		{"healthz", http.MethodGet, "/healthz", nil, 200, ""},
		{"unknown Host", http.MethodGet, "/api/v1/auth/me", []reqOpt{withHost("evil.test")}, 421, "BAD_HOST"},
		{"Host with port ok", http.MethodGet, "/api/v1/auth/me", []reqOpt{withHost("rdp.test:443"), withCookie(admin)}, 200, ""},
		{"cross-origin POST", http.MethodPost, "/api/v1/auth/logout", []reqOpt{withCookie(admin), withHeader("Origin", "https://evil.test")}, 403, "BAD_ORIGIN"},
		{"missing Origin POST", http.MethodPost, "/api/v1/auth/logout", []reqOpt{withCookie(admin), withHeader("Origin", "")}, 403, "BAD_ORIGIN"},
		{"CORS preflight refused", http.MethodOptions, "/api/v1/connections", nil, 403, "CORS_DISABLED"},
		{"metrics needs auth", http.MethodGet, "/metrics", nil, 401, "UNAUTHENTICATED"},
		{"metrics needs admin (RBAC)", http.MethodGet, "/metrics", []reqOpt{withCookie(user)}, 403, "FORBIDDEN"},
		{"metrics for admin", http.MethodGet, "/metrics", []reqOpt{withCookie(admin)}, 200, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := h.do(tc.method, tc.path, nil, tc.opts...)
			if w.Code != tc.want || errCode(w) != tc.wantCode {
				t.Fatalf("got %d %q want %d %q", w.Code, errCode(w), tc.want, tc.wantCode)
			}
		})
	}
}

func TestCorrelationAndHeaders(t *testing.T) {
	h := newHarness(t)
	given := "3f2b8c1e-9d4a-4f6b-8e2d-1a2b3c4d5e6f"
	tests := []struct {
		name string
		in   string
		echo bool
	}{
		{"valid v4 echoed", given, true},
		{"v1 replaced (leaks MAC/time)", "6ba7b810-9dad-11d1-80b4-00c04fd430c8", false},
		{"garbage replaced", "hello", false},
		{"absent minted", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := h.do(http.MethodGet, "/healthz", nil, withHeader(correlationHeader, tc.in))
			got := w.Header().Get(correlationHeader)
			if !isUUIDv4(got) || (tc.echo && got != tc.in) || (!tc.echo && got == tc.in) {
				t.Fatalf("correlation id %q for input %q", got, tc.in)
			}
			for _, hd := range []string{"X-Content-Type-Options", "X-Frame-Options", "Content-Security-Policy"} {
				if w.Header().Get(hd) == "" {
					t.Fatalf("missing %s", hd)
				}
			}
			if !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
				t.Fatalf("content-type %q", w.Header().Get("Content-Type"))
			}
		})
	}
}
