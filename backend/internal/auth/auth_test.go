package auth

import (
	"encoding/base32"
	"testing"
	"time"
)

// RFC 6238 appendix B, SHA-1 seed "12345678901234567890" (8-digit vectors
// truncated to 6 digits).
func TestTOTPVectors(t *testing.T) {
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte("12345678901234567890"))
	cases := map[int64]string{59: "287082", 1111111109: "081804", 1234567890: "005924", 2000000000: "279037"}
	for ts, code := range cases {
		if _, ok := CheckTOTP(secret, code, time.Unix(ts, 0)); !ok {
			t.Errorf("t=%d code %s rejected", ts, code)
		}
	}
	if _, ok := CheckTOTP(secret, "000000", time.Unix(59, 0)); ok {
		t.Error("wrong code accepted")
	}
}

func TestPasswordHash(t *testing.T) {
	h, err := HashPassword("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := VerifyPassword("correct horse", h); !ok {
		t.Error("valid password rejected")
	}
	if ok, _ := VerifyPassword("wrong", h); ok {
		t.Error("wrong password accepted")
	}
}
