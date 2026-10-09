package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func validConn() map[string]any {
	return map[string]any{"name": "Office", "host": "146.70.41.142", "port": 6579,
		"username": "u", "domain": "", "password": "s3cret", "security": "any", "ignore_cert": true}
}

func with(m map[string]any, k string, v any) map[string]any {
	out := map[string]any{}
	for kk, vv := range m {
		out[kk] = vv
	}
	if v == nil {
		delete(out, k)
	} else {
		out[k] = v
	}
	return out
}

func TestCreateConnectionValidation(t *testing.T) {
	tests := []struct {
		name      string
		body      any
		want      int
		wantField string
	}{
		{"valid", validConn(), 201, ""},
		{"port 0", with(validConn(), "port", 0), 422, "port"},
		{"port 65536", with(validConn(), "port", 65536), 422, "port"},
		{"empty host", with(validConn(), "host", ""), 422, "host"},
		{"host with path", with(validConn(), "host", "a.com/x"), 422, "host"},
		{"no name", with(validConn(), "name", ""), 422, "name"},
		{"bad security", with(validConn(), "security", "none"), 422, "security"},
		{"unknown field (mass assignment)", with(validConn(), "user_id", "x"), 400, ""},
		{"not json", "{", 400, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			id, _ := h.addUser("a", "pw", "admin")
			w := h.do(http.MethodPost, "/api/v1/connections", tc.body, withCookie(h.session(id)))
			if w.Code != tc.want {
				t.Fatalf("got %d want %d: %s", w.Code, tc.want, w.Body)
			}
			if tc.wantField != "" && !strings.Contains(w.Body.String(), `"field":"`+tc.wantField+`"`) {
				t.Fatalf("missing field error %q: %s", tc.wantField, w.Body)
			}
		})
	}
}

func TestConnectionNeverLeaksSecret(t *testing.T) {
	h := newHarness(t)
	id, _ := h.addUser("a", "pw", "admin")
	tok := h.session(id)
	w := h.do(http.MethodPost, "/api/v1/connections", validConn(), withCookie(tok))
	l := h.do(http.MethodGet, "/api/v1/connections", nil, withCookie(tok))
	for _, body := range []string{w.Body.String(), l.Body.String()} {
		if strings.Contains(body, "s3cret") || strings.Contains(body, "password_enc") || strings.Contains(body, "PasswordEnc") {
			t.Fatalf("secret leaked: %s", body)
		}
		if !strings.Contains(body, `"has_password":true`) {
			t.Fatalf("has_password missing: %s", body)
		}
	}
}

// Repo-layer isolation: B must not see, change or delete A's connection.
func TestCrossUserIsolation(t *testing.T) {
	h := newHarness(t)
	a, _ := h.addUser("a", "pw", "admin")
	b, _ := h.addUser("b", "pw", "user")
	ta, tb := h.session(a), h.session(b)
	w := h.do(http.MethodPost, "/api/v1/connections", validConn(), withCookie(ta))
	var created struct{ Data struct{ ID string } }
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	cid := created.Data.ID

	tests := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"update", http.MethodPut, "/api/v1/connections/" + cid, validConn()},
		{"delete", http.MethodDelete, "/api/v1/connections/" + cid, nil},
		{"ticket", http.MethodPost, "/api/v1/tunnel/ticket", map[string]string{"connection_id": cid}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if w := h.do(tc.method, tc.path, tc.body, withCookie(tb)); w.Code != http.StatusNotFound {
				t.Fatalf("user B got %d on A's connection: %s", w.Code, w.Body)
			}
		})
	}
	l := h.do(http.MethodGet, "/api/v1/connections", nil, withCookie(tb))
	if !strings.Contains(l.Body.String(), `"total":0`) {
		t.Fatalf("B sees A's rows: %s", l.Body)
	}
}

func TestPagination(t *testing.T) {
	h := newHarness(t)
	id, _ := h.addUser("a", "pw", "admin")
	tok := h.session(id)
	for i := 0; i < 3; i++ {
		h.do(http.MethodPost, "/api/v1/connections", with(validConn(), "name", fmt.Sprintf("c%d", i)), withCookie(tok))
	}
	tests := []struct {
		query string
		want  int
		count int
	}{
		{"", 200, 3},
		{"?page=2&per_page=2", 200, 1},
		{"?per_page=500", 200, 3},
		{"?per_page=501", 422, -1},
		{"?page=0", 422, -1},
		{"?page=abc", 422, -1},
	}
	for _, tc := range tests {
		t.Run(tc.query, func(t *testing.T) {
			w := h.do(http.MethodGet, "/api/v1/connections"+tc.query, nil, withCookie(tok))
			if w.Code != tc.want {
				t.Fatalf("got %d want %d", w.Code, tc.want)
			}
			if tc.count >= 0 {
				var res struct {
					Data []any
					Meta struct{ Total int }
				}
				_ = json.Unmarshal(w.Body.Bytes(), &res)
				if len(res.Data) != tc.count || res.Meta.Total != 3 {
					t.Fatalf("got %d items total %d", len(res.Data), res.Meta.Total)
				}
			}
		})
	}
}

func TestBadIDsAreNotFound(t *testing.T) {
	h := newHarness(t)
	id, _ := h.addUser("a", "pw", "admin")
	for _, bad := range []string{"1", "abc", "00000000-0000-0000-0000-00000000000g"} {
		w := h.do(http.MethodDelete, "/api/v1/connections/"+bad, nil, withCookie(h.session(id)))
		if w.Code != http.StatusNotFound {
			t.Fatalf("%q: got %d", bad, w.Code)
		}
	}
}
