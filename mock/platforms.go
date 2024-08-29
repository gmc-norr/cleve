package mock

import (
	"github.com/gmc-norr/cleve"
)

// Mock implementing the gin.PlatformGetter interface.
//
// See [mock.RunGetter] for more information.
type PlatformGetter struct {
	PlatformFn       func(string) (*cleve.Platform, error)
	PlatformInvoked  bool
	PlatformsFn      func() ([]*cleve.Platform, error)
	PlatformsInvoked bool
}

func (s *PlatformGetter) Platforms() ([]*cleve.Platform, error) {
	s.PlatformsInvoked = true
	return s.PlatformsFn()
}

func (s *PlatformGetter) Platform(name string) (*cleve.Platform, error) {
	s.PlatformInvoked = true
	return s.PlatformFn(name)
}

// Mock implementing the gin.PlatformSetter interface.
//
// See [mock.RunGetter] for more information.
type PlatformSetter struct {
	CreatePlatformFn      func(*cleve.Platform) error
	CreatePlatformInvoked bool
	DeletePlatformFn      func(string) error
	DeletePlatformInvoked bool
}

func (s *PlatformSetter) CreatePlatform(p *cleve.Platform) error {
	s.CreatePlatformInvoked = true
	return s.CreatePlatformFn(p)
}

func (s *PlatformSetter) DeletePlatform(name string) error {
	s.DeletePlatformInvoked = true
	return s.DeletePlatformFn(name)
}
