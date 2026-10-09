package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"

	"github.com/ars1364/webrdp-gateway/backend/internal/guac"
)

// rateLimiter is a fixed-window counter per key (IP or user id).
type rateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string]*window
}

type window struct {
	start time.Time
	n     int
}

func newRateLimiter(limit int, per time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: per, hits: map[string]*window{}}
}

// Allow records a hit and reports whether the key is under its limit, plus
// seconds until the window resets (for Retry-After).
func (rl *rateLimiter) Allow(key string) (bool, int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	if len(rl.hits) > 10_000 { // bound memory under a flood
		for k, w := range rl.hits {
			if now.Sub(w.start) > rl.window {
				delete(rl.hits, k)
			}
		}
	}
	w, ok := rl.hits[key]
	if !ok || now.Sub(w.start) > rl.window {
		w = &window{start: now}
		rl.hits[key] = w
	}
	w.n++
	reset := int(rl.window.Seconds() - now.Sub(w.start).Seconds())
	return w.n <= rl.limit, max(reset, 1)
}

// ticket is a single-use, short-lived grant to open one tunnel. It keeps
// credentials out of the WebSocket URL and out of logs.
type ticket struct {
	userID  string
	target  guac.Target
	label   string
	expires time.Time
}

type ticketStore struct {
	mu  sync.Mutex
	ttl time.Duration
	m   map[string]ticket
}

func newTicketStore(ttl time.Duration) *ticketStore {
	return &ticketStore{ttl: ttl, m: map[string]ticket{}}
}

func (ts *ticketStore) Put(t ticket) (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	id := base64.RawURLEncoding.EncodeToString(b)
	ts.mu.Lock()
	defer ts.mu.Unlock()
	now := time.Now()
	for k, v := range ts.m {
		if now.After(v.expires) {
			delete(ts.m, k)
		}
	}
	t.expires = now.Add(ts.ttl)
	ts.m[id] = t
	return id, nil
}

// Take removes and returns the ticket; a second Take of the same id fails.
func (ts *ticketStore) Take(id string) (ticket, bool) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	t, ok := ts.m[id]
	delete(ts.m, id)
	if !ok || time.Now().After(t.expires) {
		return ticket{}, false
	}
	return t, true
}

// tunnelLimiter caps concurrent RDP tunnels globally and per user, so one
// account can't exhaust guacd or the host's memory.
type tunnelLimiter struct {
	mu      sync.Mutex
	max     int
	perUser int
	total   int
	byUser  map[string]int
}

func newTunnelLimiter(max, perUser int) *tunnelLimiter {
	return &tunnelLimiter{max: max, perUser: perUser, byUser: map[string]int{}}
}

func (t *tunnelLimiter) Acquire(userID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.total >= t.max || t.byUser[userID] >= t.perUser {
		return false
	}
	t.total++
	t.byUser[userID]++
	return true
}

func (t *tunnelLimiter) Release(userID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.total--
	if t.byUser[userID]--; t.byUser[userID] <= 0 {
		delete(t.byUser, userID)
	}
}
