package cleve

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	APIKeyLength   = 40
	APIKeyIdLength = 8
)

type APIKey struct {
	Id      []byte
	Key     []byte
	User    string
	Created time.Time
}

type PlainKey []byte

// PlainKey returns a 40-bit API key. The first APIKeyIdLength bytes
// represents the ID for the API key.
func NewPlainKey() PlainKey {
	key := make([]byte, APIKeyLength)
	_, _ = rand.Read(key) // According to docs, this never returns an error
	return key
}

// Id returns the ID part of the plain API key.
func (k PlainKey) Id() []byte {
	return k[:APIKeyIdLength]
}

// Key returns the key part of the plain API key.
func (k PlainKey) Key() []byte {
	return k[APIKeyIdLength:]
}

// String represents the key as a URL-encoded string
func (k PlainKey) String() string {
	return base64.URLEncoding.EncodeToString(k)
}

// NewAPIKey creates a new API key based on a plain text key and a user name.
// If the hashing of the key fails
func NewAPIKey(plainKey PlainKey, user string) (*APIKey, error) {
	hashedKey, err := bcrypt.GenerateFromPassword(plainKey.Key(), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	created := time.Now().Local()
	return &APIKey{
		Id:      plainKey[:APIKeyIdLength],
		Key:     hashedKey,
		Created: created,
		User:    user,
	}, nil
}

// Compare the API key with a plaintext key. An error is returned if they
// do not match, and nil is returned if they do match.
func (k APIKey) Compare(plainKey PlainKey) error {
	return bcrypt.CompareHashAndPassword(k.Key, plainKey.Key())
}
