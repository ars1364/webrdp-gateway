package seal

import (
	"bytes"
	"testing"
)

func key(b byte) []byte { return bytes.Repeat([]byte{b}, 32) }

func TestSeal(t *testing.T) {
	s, _ := New(1, map[byte][]byte{1: key(1)})
	ct, err := s.Seal([]byte("secret"), []byte("conn:a"))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		ct      []byte
		aad     string
		wantErr bool
	}{
		{"same row", ct, "conn:a", false},
		{"other row's AAD", ct, "conn:b", true},
		{"tampered", append(append([]byte{}, ct[:len(ct)-1]...), ct[len(ct)-1]^1), "conn:a", true},
		{"unknown key id", append([]byte{9}, ct[1:]...), "conn:a", true},
		{"empty", nil, "conn:a", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pt, err := s.Open(tc.ct, []byte(tc.aad))
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && string(pt) != "secret" {
				t.Fatalf("plaintext = %q", pt)
			}
		})
	}
}

func TestRotation(t *testing.T) {
	old, _ := New(1, map[byte][]byte{1: key(1)})
	ct, _ := old.Seal([]byte("v"), []byte("a"))
	both, err := New(2, map[byte][]byte{1: key(1), 2: key(2)})
	if err != nil {
		t.Fatal(err)
	}
	if both.IsCurrent(ct) {
		t.Fatal("old ciphertext reported current")
	}
	if pt, err := both.Open(ct, []byte("a")); err != nil || string(pt) != "v" {
		t.Fatalf("old ciphertext unreadable after rotation: %v", err)
	}
	ct2, _ := both.Seal([]byte("v"), []byte("a"))
	if !both.IsCurrent(ct2) || ct2[0] != 2 {
		t.Fatal("new ciphertext not sealed with key 2")
	}
	if _, err := New(3, map[byte][]byte{1: key(1)}); err == nil {
		t.Fatal("missing current key accepted")
	}
}
