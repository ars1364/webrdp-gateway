package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// RFC 6238 TOTP: SHA-1, 30 s step, 6 digits — what every authenticator app expects.
const (
	totpStep   = 30
	totpDigits = 1_000_000
)

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

func NewTOTPSecret() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return b32.EncodeToString(b), nil
}

func TOTPURI(issuer, account, secret string) string {
	label := url.PathEscape(issuer + ":" + account)
	q := url.Values{"secret": {secret}, "issuer": {issuer}, "algorithm": {"SHA1"},
		"digits": {"6"}, "period": {"30"}}
	return "otpauth://totp/" + label + "?" + q.Encode()
}

func totpAt(key []byte, step int64) string {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(step))
	mac := hmac.New(sha1.New, key)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	code := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", code%totpDigits)
}

// CheckTOTP accepts the current step ±1 and returns the matched step so the
// caller can reject replays (step must be > the last accepted step).
func CheckTOTP(secret, code string, now time.Time) (step int64, ok bool) {
	key, err := b32.DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil || len(code) != 6 {
		return 0, false
	}
	cur := now.Unix() / totpStep
	for _, s := range []int64{cur - 1, cur, cur + 1} {
		if hmac.Equal([]byte(totpAt(key, s)), []byte(code)) {
			return s, true
		}
	}
	return 0, false
}
