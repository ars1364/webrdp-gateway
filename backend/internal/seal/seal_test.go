package seal

import "testing"

func TestSealBindsAAD(t *testing.T) {
	s, _ := New(make([]byte, 32))
	ct, err := s.Seal([]byte("secret"), []byte("conn:a"))
	if err != nil {
		t.Fatal(err)
	}
	if pt, err := s.Open(ct, []byte("conn:a")); err != nil || string(pt) != "secret" {
		t.Fatalf("open failed: %v", err)
	}
	if _, err := s.Open(ct, []byte("conn:b")); err == nil {
		t.Fatal("ciphertext opened under another row's AAD")
	}
}
