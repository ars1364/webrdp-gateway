package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
)

func adHoc(host string, port int) map[string]any {
	return map[string]any{"ad_hoc": map[string]any{"name": "", "host": host, "port": port,
		"username": "u", "domain": "", "password": "p", "security": "any", "ignore_cert": true}}
}

func TestCreateTicket(t *testing.T) {
	tests := []struct {
		name     string
		body     any
		want     int
		wantCode string
	}{
		{"public IP + custom port", adHoc("146.70.41.142", 6579), 200, ""},
		{"loopback (SSRF)", adHoc("127.0.0.1", 3389), 422, "TARGET_REJECTED"},
		{"private range (SSRF)", adHoc("10.0.0.5", 3389), 422, "TARGET_REJECTED"},
		{"cloud metadata (SSRF)", adHoc("169.254.169.254", 80), 422, "TARGET_REJECTED"},
		{"localhost name (SSRF)", adHoc("localhost", 3389), 422, "TARGET_REJECTED"},
		{"bad port", adHoc("146.70.41.142", 0), 422, "VALIDATION_FAILED"},
		{"neither id nor ad_hoc", map[string]any{}, 422, "VALIDATION_FAILED"},
		{"unknown connection", map[string]any{"connection_id": newUUID()}, 404, "NOT_FOUND"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			id, _ := h.addUser("a", "pw", "admin")
			w := h.do(http.MethodPost, "/api/v1/tunnel/ticket", tc.body, withCookie(h.session(id)))
			if w.Code != tc.want || errCode(w) != tc.wantCode {
				t.Fatalf("got %d %q want %d %q: %s", w.Code, errCode(w), tc.want, tc.wantCode, w.Body)
			}
		})
	}
}

func TestTunnelRequiresSessionAndOwnTicket(t *testing.T) {
	h := newHarness(t)
	a, _ := h.addUser("a", "pw", "admin")
	b, _ := h.addUser("b", "pw", "user")
	ta, tb := h.session(a), h.session(b)
	w := h.do(http.MethodPost, "/api/v1/tunnel/ticket", adHoc("146.70.41.142", 6579), withCookie(ta))
	var res struct{ Data struct{ Ticket string } }
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	tk := res.Data.Ticket

	tests := []struct {
		name string
		opts []reqOpt
		q    string
		want int
	}{
		{"no session", nil, tk, 401},
		{"other user's ticket", []reqOpt{withCookie(tb)}, tk, 403},
		{"ticket is single-use (consumed above)", []reqOpt{withCookie(ta)}, tk, 403},
		{"garbage ticket", []reqOpt{withCookie(ta)}, "nope", 403},
	}
	for _, tc := range tests { // sequential: the 403 for B consumes the ticket
		if w := h.do(http.MethodGet, "/api/v1/tunnel?ticket="+tc.q, nil, tc.opts...); w.Code != tc.want {
			t.Fatalf("%s: got %d want %d", tc.name, w.Code, tc.want)
		}
	}
}

func TestTunnelGuacdDownIsBadGateway(t *testing.T) {
	h := newHarness(t)
	a, _ := h.addUser("a", "pw", "admin")
	tok := h.session(a)
	w := h.do(http.MethodPost, "/api/v1/tunnel/ticket", adHoc("146.70.41.142", 6579), withCookie(tok))
	var res struct{ Data struct{ Ticket string } }
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	w = h.do(http.MethodGet, "/api/v1/tunnel?ticket="+res.Data.Ticket, nil, withCookie(tok))
	if w.Code != http.StatusBadGateway || errCode(w) != "RDP_CONNECT_FAILED" {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
}
