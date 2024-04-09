package cleve

import (
	"crypto/sha1"
	"fmt"
	"time"
)

type APIKeyService interface {
	Create(*APIKey) error
	Delete(string) error
	Get(string) (*APIKey, error)
	All() ([]*APIKey, error)
	UserKey(string) (*APIKey, error)
}

type APIKey struct {
	Key     string
	User    string
	Created time.Time
}

func NewAPIKey(user string) *APIKey {
	created := time.Now()
	hash := sha1.Sum([]byte(fmt.Sprintf("%s %s", user, created)))
	return &APIKey{
		Key:     fmt.Sprintf("%x", hash),
		Created: created,
		User:    user,
	}
}
