// Package seal encrypts small secrets at rest with AES-256-GCM.
// Ciphertext layout: version(1) | nonce(12) | sealed. The AAD binds a
// ciphertext to its owning row so it can't be swapped between rows.
package seal

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
)

const version byte = 1

type Sealer struct{ aead cipher.AEAD }

func New(key []byte) (*Sealer, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Sealer{aead: aead}, nil
}

func (s *Sealer) Seal(plain, aad []byte) ([]byte, error) {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	out := append([]byte{version}, nonce...)
	return s.aead.Seal(out, nonce, plain, aad), nil
}

func (s *Sealer) Open(ct, aad []byte) ([]byte, error) {
	n := s.aead.NonceSize()
	if len(ct) < 1+n || ct[0] != version {
		return nil, errors.New("seal: bad ciphertext")
	}
	return s.aead.Open(nil, ct[1:1+n], ct[1+n:], aad)
}
