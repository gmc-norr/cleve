package mock

import (
	"github.com/gmc-norr/cleve"
)

// Mock implementing the gin.PlatformGetter interface.
//
// See [mock.RunGetter] for more information.
type PlatformGetter struct {
	PlatformFn       func(string) (cleve.Platform, error)
	PlatformInvoked  bool
	PlatformsFn      func() (cleve.Platforms, error)
	PlatformsInvoked bool
}

func (s *PlatformGetter) Platforms() (cleve.Platforms, error) {
	s.PlatformsInvoked = true
	return s.PlatformsFn()
}

func (s *PlatformGetter) Platform(name string) (cleve.Platform, error) {
	s.PlatformInvoked = true
	return s.PlatformFn(name)
}
