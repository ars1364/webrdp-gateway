package httpapi

import (
	"net/http"
	"testing"
)

func TestIdempotency(t *testing.T) {
	h := newHarness(t)
	id, _ := h.addUser("a", "pw", "admin")
	tok := h.session(id)
	key := newUUID()
	post := func(body any, k string) (int, string, string) {
		w := h.do(http.MethodPost, "/api/v1/connections", body, withCookie(tok), withHeader(idemHeader, k))
		return w.Code, errCode(w), w.Header().Get("Idempotent-Replayed")
	}
	tests := []struct {
		name       string
		body       any
		key        string
		want       int
		wantCode   string
		wantReplay string
	}{
		{"first call", validConn(), key, 201, "", ""},
		{"same key + body replays", validConn(), key, 201, "", "true"},
		{"same key, other body", with(validConn(), "name", "Other"), key, 422, "IDEMPOTENCY_KEY_REUSED", ""},
		{"missing key", validConn(), "", 400, "IDEMPOTENCY_KEY_REQUIRED", ""},
		{"non-v4 key", validConn(), "6ba7b810-9dad-11d1-80b4-00c04fd430c8", 400, "IDEMPOTENCY_KEY_REQUIRED", ""},
	}
	for _, tc := range tests { // sequential: later rows depend on the first
		code, ecode, replay := post(tc.body, tc.key)
		if code != tc.want || ecode != tc.wantCode || replay != tc.wantReplay {
			t.Fatalf("%s: got %d %q replay=%q", tc.name, code, ecode, replay)
		}
	}
	l := h.do(http.MethodGet, "/api/v1/connections", nil, withCookie(tok))
	if w := l.Body.String(); !contains(w, `"total":1`) {
		t.Fatalf("replay created a duplicate: %s", w)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
