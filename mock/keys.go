package mock

import (
	"github.com/gmc-norr/cleve"
)

type APIKeyService struct {
	CreateFn       func(*cleve.APIKey) error
	CreateInvoked  bool
	DeleteFn       func(string) error
	DeleteInvoked  bool
	GetFn          func(string) (*cleve.APIKey, error)
	GetInvoked     bool
	AllFn          func() ([]*cleve.APIKey, error)
	AllInvoked     bool
	UserKeyFn      func(string) (*cleve.APIKey, error)
	UserKeyInvoked bool
}

func (s *APIKeyService) Create(k *cleve.APIKey) error {
	s.CreateInvoked = true
	return s.CreateFn(k)
}

func (s *APIKeyService) Delete(k string) error {
	s.DeleteInvoked = true
	return s.DeleteFn(k)
}

func (s *APIKeyService) Get(k string) (*cleve.APIKey, error) {
	s.GetInvoked = true
	return s.GetFn(k)
}

func (s *APIKeyService) All() ([]*cleve.APIKey, error) {
	s.AllInvoked = true
	return s.AllFn()
}

func (s *APIKeyService) UserKey(u string) (*cleve.APIKey, error) {
	s.UserKeyInvoked = true
	return s.UserKeyFn(u)
}
