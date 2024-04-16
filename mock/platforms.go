package mock

import (
	"github.com/gmc-norr/cleve"
)

type PlatformService struct {
	AllFn           func() ([]*cleve.Platform, error)
	AllInvoked      bool
	GetFn           func(string) (*cleve.Platform, error)
	GetInvoked      bool
	CreateFn        func(*cleve.Platform) error
	CreateInvoked   bool
	DeleteFn        func(string) error
	DeleteInvoked   bool
	SetIndexFn      func() (string, error)
	SetIndexInvoked bool
}

func (s *PlatformService) All() ([]*cleve.Platform, error) {
	s.AllInvoked = true
	return s.AllFn()
}

func (s *PlatformService) Get(name string) (*cleve.Platform, error) {
	s.GetInvoked = true
	return s.GetFn(name)
}

func (s *PlatformService) Create(p *cleve.Platform) error {
	s.CreateInvoked = true
	return s.CreateFn(p)
}

func (s *PlatformService) Delete(name string) error {
	s.DeleteInvoked = true
	return s.DeleteFn(name)
}

func (s *PlatformService) SetIndex() (string, error) {
	s.SetIndexInvoked = true
	return s.SetIndexFn()
}
