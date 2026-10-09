// Package seal encrypts small secrets at rest with AES-256-GCM.
//
// Ciphertext layout: keyID(1) | nonce(12) | sealed. The AAD binds a value to
// its owning row so ciphertexts can't be swapped between rows. Several keys
// can be loaded at once, so a KEK rotates without downtime: seal with the
// new key, keep the old one for Open until `server rekey` has run.
package seal

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
)

type Sealer struct {
	current byte
	keys    map[byte]cipher.AEAD
}

// New builds a sealer that seals with key `currentID` and opens with any
// key in `keys`.
func New(currentID byte, keys map[byte][]byte) (*Sealer, error) {
	if _, ok := keys[currentID]; !ok {
		return nil, fmt.Errorf("seal: current key %d not loaded", currentID)
	}
	s := &Sealer{current: currentID, keys: map[byte]cipher.AEAD{}}
	for id, k := range keys {
		block, err := aes.NewCipher(k)
		if err != nil {
			return nil, fmt.Errorf("seal: key %d: %w", id, err)
		}
		aead, err := cipher.NewGCM(block)
		if err != nil {
			return nil, err
		}
		s.keys[id] = aead
	}
	return s, nil
}

func (s *Sealer) Seal(plain, aad []byte) ([]byte, error) {
	aead := s.keys[s.current]
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	out := append([]byte{s.current}, nonce...)
	return aead.Seal(out, nonce, plain, aad), nil
}

func (s *Sealer) Open(ct, aad []byte) ([]byte, error) {
	if len(ct) < 1 {
		return nil, errors.New("seal: empty ciphertext")
	}
	aead, ok := s.keys[ct[0]]
	if !ok {
		return nil, fmt.Errorf("seal: key %d not loaded", ct[0])
	}
	n := aead.NonceSize()
	if len(ct) < 1+n {
		return nil, errors.New("seal: short ciphertext")
	}
	return aead.Open(nil, ct[1:1+n], ct[1+n:], aad)
}

// IsCurrent reports whether ct was sealed with the current key.
func (s *Sealer) IsCurrent(ct []byte) bool { return len(ct) > 0 && ct[0] == s.current }
