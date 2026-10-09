// Package core holds the domain types and the ports (interfaces) the HTTP
// adapter depends on. Adapters (store, guac) implement the ports; nothing in
// core imports an adapter.
package core

import (
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

type User struct {
	ID            string
	Username      string
	PasswordHash  string
	TOTPSecretEnc []byte
	Role          string
	FailedLogins  int
	LockedUntil   *time.Time
	LastTOTPStep  int64
}

type Session struct {
	UserID    string
	Username  string
	Role      string
	ExpiresAt time.Time
}

// Connection is a saved RDP target. PasswordEnc is sealed with the row id as
// AAD and is never serialized: clients only ever see HasPassword.
type Connection struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	Username    string    `json:"username"`
	Domain      string    `json:"domain"`
	Security    string    `json:"security"`
	IgnoreCert  bool      `json:"ignore_cert"`
	Quality     string    `json:"quality"` // high | balanced | low (bandwidth profile)
	HasPassword bool      `json:"has_password"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	PasswordEnc []byte    `json:"-"`
}

type AuditEntry struct {
	Action    string         `json:"action"`
	Target    string         `json:"target"`
	IP        string         `json:"ip"`
	Detail    map[string]any `json:"detail"`
	CreatedAt time.Time      `json:"created_at"`
}

// Recording is a server-side session recording (guacd protocol dump).
type Recording struct {
	ID        string     `json:"id"`
	Label     string     `json:"label"`
	Target    string     `json:"target"`
	SizeBytes int64      `json:"size_bytes"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at"`
}

// Page is a validated, bounded page request (per_page ≤ MaxPerPage).
type Page struct {
	Page    int
	PerPage int
}

const (
	DefaultPerPage = 50
	MaxPerPage     = 500
)

func (p Page) Offset() int { return (p.Page - 1) * p.PerPage }

// IdemRecord is a stored idempotent response. Status 0 means "in progress".
type IdemRecord struct {
	RequestHash string
	Status      int
	Body        []byte
}
