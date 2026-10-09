package core

import (
	"context"
	"time"
)

// Users is the authentication port.
type Users interface {
	UserByName(ctx context.Context, username string) (*User, error)
	RecordFailedLogin(ctx context.Context, userID string) error
	RecordLogin(ctx context.Context, userID string, totpStep int64) error
}

// Sessions is the session-token port. Only token hashes cross it.
type Sessions interface {
	CreateSession(ctx context.Context, hash []byte, userID, ip, ua string, ttl time.Duration) error
	SessionByHash(ctx context.Context, hash []byte) (*Session, error)
	DeleteSession(ctx context.Context, hash []byte) error
}

// Connections is the saved-target port. userID is ALWAYS the first scope
// argument; implementations must filter by it in SQL.
type Connections interface {
	ListConnections(ctx context.Context, userID string, p Page) ([]*Connection, int, error)
	GetConnection(ctx context.Context, userID, id string) (*Connection, error)
	CreateConnection(ctx context.Context, userID string, c *Connection) error
	UpdateConnection(ctx context.Context, userID string, c *Connection, keepPassword bool) error
	DeleteConnection(ctx context.Context, userID, id string) error
}

// Audit is the append-only audit-trail port.
type Audit interface {
	Audit(ctx context.Context, userID, action, target, ip string, detail map[string]any)
	ListAudit(ctx context.Context, userID string, p Page) ([]AuditEntry, int, error)
}

// Idempotency stores one response per (user, key). Begin returns
// (existing, false) for a replay, or (nil, true) when the caller owns the key.
type Idempotency interface {
	IdemBegin(ctx context.Context, userID, key, requestHash string, ttl time.Duration) (*IdemRecord, bool, error)
	IdemFinish(ctx context.Context, userID, key string, status int, body []byte) error
	IdemAbort(ctx context.Context, userID, key string) error
}

// Repository is everything the HTTP adapter needs from persistence.
type Repository interface {
	Users
	Sessions
	Connections
	Audit
	Idempotency
}
