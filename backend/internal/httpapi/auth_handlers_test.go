package httpapi

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ars1364/webrdp-gateway/backend/internal/auth"
)

func totpNow(t *testing.T, secret string, offset time.Duration) string {
	t.Helper()
	c, err := auth.TOTPCode(secret, time.Now().Add(offset))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestLogin(t *testing.T) {
	tests := []struct {
		name     string
		user     string
		password string
		code     func(secret string) string
		want     int
		wantCode string
	}{
		{"valid", "admin", "pw-correct", func(s string) string { return totpNow(t, s, 0) }, 200, ""},
		{"case-insensitive username", "ADMIN", "pw-correct", func(s string) string { return totpNow(t, s, 0) }, 200, ""},
		{"wrong password", "admin", "nope", func(s string) string { return totpNow(t, s, 0) }, 401, "INVALID_CREDENTIALS"},
		{"wrong code", "admin", "pw-correct", func(string) string { return "000000" }, 401, "INVALID_CREDENTIALS"},
		{"stale code (-2 steps)", "admin", "pw-correct", func(s string) string { return totpNow(t, s, -60*time.Second) }, 401, "INVALID_CREDENTIALS"},
		{"unknown user", "ghost", "pw-correct", func(string) string { return "123456" }, 401, "INVALID_CREDENTIALS"},
		{"oversized username", strings.Repeat("a", 65), "x", func(string) string { return "123456" }, 401, "INVALID_CREDENTIALS"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			_, secret := h.addUser("admin", "pw-correct", "admin")
			w := h.do(http.MethodPost, "/api/v1/auth/login",
				map[string]string{"username": tc.user, "password": tc.password, "totp": tc.code(secret)})
			if w.Code != tc.want || errCode(w) != tc.wantCode {
				t.Fatalf("got %d %q, want %d %q: %s", w.Code, errCode(w), tc.want, tc.wantCode, w.Body)
			}
			if tc.want == 200 {
				c := w.Result().Cookies()
				if len(c) != 1 || c[0].Name != cookieName || !c[0].HttpOnly || !c[0].Secure || c[0].SameSite != http.SameSiteStrictMode {
					t.Fatalf("bad session cookie: %+v", c)
				}
			}
		})
	}
}

func TestLoginReplayAndLockout(t *testing.T) {
	h := newHarness(t)
	_, secret := h.addUser("admin", "pw", "admin")
	body := map[string]string{"username": "admin", "password": "pw", "totp": totpNow(t, secret, 0)}
	if w := h.do(http.MethodPost, "/api/v1/auth/login", body); w.Code != 200 {
		t.Fatalf("first login: %d %s", w.Code, w.Body)
	}
	if w := h.do(http.MethodPost, "/api/v1/auth/login", body); w.Code != 401 {
		t.Fatalf("replayed TOTP accepted: %d", w.Code)
	}
	for i := 0; i < 5; i++ {
		h.do(http.MethodPost, "/api/v1/auth/login", map[string]string{"username": "admin", "password": "x", "totp": "000000"})
	}
	w := h.do(http.MethodPost, "/api/v1/auth/login", map[string]string{"username": "admin", "password": "pw", "totp": "000000"})
	if w.Code != http.StatusLocked {
		t.Fatalf("expected lockout 423, got %d", w.Code)
	}
}

func TestMeAndLogout(t *testing.T) {
	h := newHarness(t)
	id, _ := h.addUser("admin", "pw", "admin")
	tok := h.session(id)
	tests := []struct {
		name   string
		method string
		path   string
		opts   []reqOpt
		want   int
	}{
		{"me without cookie", http.MethodGet, "/api/v1/auth/me", nil, 401},
		{"me with cookie", http.MethodGet, "/api/v1/auth/me", []reqOpt{withCookie(tok)}, 200},
		{"logout", http.MethodPost, "/api/v1/auth/logout", []reqOpt{withCookie(tok)}, 200},
		{"me after logout", http.MethodGet, "/api/v1/auth/me", []reqOpt{withCookie(tok)}, 401},
	}
	for _, tc := range tests { // sequential on purpose: steps depend on each other
		if w := h.do(tc.method, tc.path, nil, tc.opts...); w.Code != tc.want {
			t.Fatalf("%s: got %d want %d", tc.name, w.Code, tc.want)
		}
	}
}

func TestMeReportsFeatureFlags(t *testing.T) {
	h := newHarness(t)
	id, _ := h.addUser("a", "pw", "admin")
	w := h.do(http.MethodGet, "/api/v1/auth/me", nil, withCookie(h.session(id)))
	tests := []struct{ want string }{
		{`"clipboard_upload":true`}, {`"clipboard_download":false`}, {`"file_upload":false`}, {`"file_download":false`},
	}
	for _, tc := range tests {
		if !strings.Contains(w.Body.String(), tc.want) {
			t.Fatalf("missing %s in %s", tc.want, w.Body)
		}
	}
}
