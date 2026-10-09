package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ars1364/webrdp-gateway/backend/internal/auth"
	"github.com/ars1364/webrdp-gateway/backend/internal/db"
	"github.com/ars1364/webrdp-gateway/backend/internal/seal"
	"github.com/ars1364/webrdp-gateway/backend/internal/store"
)

// createAdmin creates a user with a generated password (unless ADMIN_PASSWORD
// is set) and a fresh TOTP seed, then prints both once as JSON on stdout.
func createAdmin(ctx context.Context, st *store.Store, s *seal.Sealer, args []string) int {
	fs := flag.NewFlagSet("create-admin", flag.ContinueOnError)
	username := fs.String("u", "", "username")
	role := fs.String("role", "admin", "admin | user")
	issuer := fs.String("issuer", "webrdp-gateway", "TOTP issuer label")
	if err := fs.Parse(args); err != nil || *username == "" {
		fmt.Fprintln(os.Stderr, "usage: server create-admin -u NAME [-role admin|user]")
		return 2
	}
	pw := os.Getenv("ADMIN_PASSWORD")
	if pw == "" {
		b := make([]byte, 18)
		_, _ = rand.Read(b)
		pw = base64.RawURLEncoding.EncodeToString(b)
	}
	hash, err := auth.HashPassword(pw)
	if err != nil {
		return fail(err)
	}
	secret, err := auth.NewTOTPSecret()
	if err != nil {
		return fail(err)
	}
	id := newID()
	enc, err := s.Seal([]byte(secret), []byte("totp:"+id))
	if err != nil {
		return fail(err)
	}
	if err := st.CreateUser(ctx, id, *username, hash, enc, *role); err != nil {
		return fail(err)
	}
	out, _ := json.MarshalIndent(map[string]string{
		"username":    *username,
		"password":    pw,
		"totp_secret": secret,
		"otpauth_uri": auth.TOTPURI(*issuer, *username, secret),
	}, "", "  ")
	fmt.Println(string(out))
	return 0
}

func migrateDown(ctx context.Context, pool *pgxpool.Pool, args []string) int {
	n := 1
	if len(args) > 0 {
		v, err := strconv.Atoi(args[0])
		if err != nil || v < 1 {
			fmt.Fprintln(os.Stderr, "usage: server migrate-down N")
			return 2
		}
		n = v
	}
	if err := db.MigrateDown(ctx, pool, n); err != nil {
		return fail(err)
	}
	fmt.Printf("reverted %d migration(s)\n", n)
	return 0
}

// rekey re-seals every value not yet under the current KEK, in one
// transaction. Run it after setting APP_KEK(+_ID) to the new key and the old
// one as APP_KEK_PREVIOUS(+_ID); then drop APP_KEK_PREVIOUS.
func rekey(ctx context.Context, st *store.Store, s *seal.Sealer) int {
	rows, err := st.SealedValues(ctx)
	if err != nil {
		return fail(err)
	}
	var changed []store.SealedRow
	for _, r := range rows {
		if s.IsCurrent(r.Value) {
			continue
		}
		aad := []byte("conn:" + r.ID)
		if r.Table == "users" {
			aad = []byte("totp:" + r.ID)
		}
		pt, err := s.Open(r.Value, aad)
		if err != nil {
			return fail(fmt.Errorf("%s %s: %w", r.Table, r.ID, err))
		}
		if r.Value, err = s.Seal(pt, aad); err != nil {
			return fail(err)
		}
		changed = append(changed, r)
	}
	if err := st.ResealAll(ctx, changed); err != nil {
		return fail(err)
	}
	fmt.Printf("re-sealed %d of %d value(s)\n", len(changed), len(rows))
	return 0
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "create-admin:", err)
	return 1
}
