package httpapi

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ars1364/webrdp-gateway/backend/internal/core"
)

// fakeRepo is an in-memory core.Repository with the same scoping rules as
// the Postgres adapter (every connection lookup filters by user id).
type fakeRepo struct {
	mu       sync.Mutex
	users    map[string]*core.User // by lower(username)
	sessions map[string]*core.Session
	conns    map[string]map[string]*core.Connection // userID -> id -> conn
	audit    []core.AuditEntry
	idem     map[string]*core.IdemRecord
	recs     map[string]map[string]*core.Recording
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{users: map[string]*core.User{}, sessions: map[string]*core.Session{},
		conns: map[string]map[string]*core.Connection{}, idem: map[string]*core.IdemRecord{}}
}

var _ core.Repository = (*fakeRepo)(nil)

func (f *fakeRepo) UserByName(_ context.Context, n string) (*core.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if u, ok := f.users[strings.ToLower(n)]; ok {
		cp := *u
		return &cp, nil
	}
	return nil, core.ErrNotFound
}

func (f *fakeRepo) user(id string) *core.User {
	for _, u := range f.users {
		if u.ID == id {
			return u
		}
	}
	return nil
}

func (f *fakeRepo) RecordFailedLogin(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	u := f.user(id)
	u.FailedLogins++
	if u.FailedLogins >= 5 {
		t := time.Now().Add(15 * time.Minute)
		u.LockedUntil = &t
	}
	return nil
}

func (f *fakeRepo) RecordLogin(_ context.Context, id string, step int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	u := f.user(id)
	if u.LastTOTPStep >= step {
		return core.ErrConflict
	}
	u.FailedLogins, u.LockedUntil, u.LastTOTPStep = 0, nil, step
	return nil
}

func (f *fakeRepo) CreateSession(_ context.Context, h []byte, uid, _, _ string, ttl time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	u := f.user(uid)
	f.sessions[string(h)] = &core.Session{UserID: uid, Username: u.Username, Role: u.Role, ExpiresAt: time.Now().Add(ttl)}
	return nil
}

func (f *fakeRepo) SessionByHash(_ context.Context, h []byte) (*core.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.sessions[string(h)]; ok && s.ExpiresAt.After(time.Now()) {
		return s, nil
	}
	return nil, core.ErrNotFound
}

func (f *fakeRepo) DeleteSession(_ context.Context, h []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.sessions, string(h))
	return nil
}

func (f *fakeRepo) ListConnections(_ context.Context, uid string, p core.Page) ([]*core.Connection, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	all := []*core.Connection{}
	for _, c := range f.conns[uid] {
		all = append(all, c)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Name < all[j].Name })
	lo, hi := min(p.Offset(), len(all)), min(p.Offset()+p.PerPage, len(all))
	return all[lo:hi], len(all), nil
}

func (f *fakeRepo) GetConnection(_ context.Context, uid, id string) (*core.Connection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if c, ok := f.conns[uid][id]; ok {
		return c, nil
	}
	return nil, core.ErrNotFound
}

func (f *fakeRepo) CreateConnection(_ context.Context, uid string, c *core.Connection) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.conns[uid] == nil {
		f.conns[uid] = map[string]*core.Connection{}
	}
	c.HasPassword = c.PasswordEnc != nil
	f.conns[uid][c.ID] = c
	return nil
}

func (f *fakeRepo) UpdateConnection(_ context.Context, uid string, c *core.Connection, keep bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	old, ok := f.conns[uid][c.ID]
	if !ok {
		return core.ErrNotFound
	}
	if keep {
		c.PasswordEnc = old.PasswordEnc
	}
	c.HasPassword = c.PasswordEnc != nil
	f.conns[uid][c.ID] = c
	return nil
}

func (f *fakeRepo) DeleteConnection(_ context.Context, uid, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.conns[uid][id]; !ok {
		return core.ErrNotFound
	}
	delete(f.conns[uid], id)
	return nil
}

func (f *fakeRepo) Audit(_ context.Context, _, action, target, ip string, _ map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.audit = append(f.audit, core.AuditEntry{Action: action, Target: target, IP: ip})
}

func (f *fakeRepo) ListAudit(_ context.Context, _ string, _ core.Page) ([]core.AuditEntry, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.audit, len(f.audit), nil
}

func (f *fakeRepo) IdemBegin(_ context.Context, uid, key, hash string, _ time.Duration) (*core.IdemRecord, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.idem[uid+key]; ok {
		cp := *r
		return &cp, false, nil
	}
	f.idem[uid+key] = &core.IdemRecord{RequestHash: hash}
	return nil, true, nil
}

func (f *fakeRepo) IdemFinish(_ context.Context, uid, key string, status int, body []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	r := f.idem[uid+key]
	r.Status, r.Body = status, append([]byte{}, body...)
	return nil
}

func (f *fakeRepo) IdemAbort(_ context.Context, uid, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.idem, uid+key)
	return nil
}

func (f *fakeRepo) CreateRecording(_ context.Context, uid string, r *core.Recording) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.recs == nil {
		f.recs = map[string]map[string]*core.Recording{}
	}
	if f.recs[uid] == nil {
		f.recs[uid] = map[string]*core.Recording{}
	}
	r.StartedAt = time.Now()
	f.recs[uid][r.ID] = r
	return nil
}

func (f *fakeRepo) FinishRecording(_ context.Context, uid, id string, size int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.recs[uid][id]; ok {
		now := time.Now()
		r.SizeBytes, r.EndedAt = size, &now
	}
	return nil
}

func (f *fakeRepo) ListRecordings(_ context.Context, uid string, _ core.Page) ([]core.Recording, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []core.Recording{}
	for _, r := range f.recs[uid] {
		out = append(out, *r)
	}
	return out, len(out), nil
}

func (f *fakeRepo) GetRecording(_ context.Context, uid, id string) (*core.Recording, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.recs[uid][id]; ok {
		return r, nil
	}
	return nil, core.ErrNotFound
}

func (f *fakeRepo) DeleteRecording(_ context.Context, uid, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.recs[uid][id]; !ok {
		return core.ErrNotFound
	}
	delete(f.recs[uid], id)
	return nil
}
