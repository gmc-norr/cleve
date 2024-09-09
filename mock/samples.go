package mock

import (
	"github.com/gmc-norr/cleve"
)

// Mock implementing the gin.SampleGetter interface.
//
// See [mock.RunGetter] for more information.
type SampleGetter struct {
	SamplesFn      func(*cleve.SampleFilter) (*cleve.SampleResult, error)
	SamplesInvoked bool
	SampleFn       func(string) (*cleve.Sample, error)
	SampleInvoked  bool
}

func (g *SampleGetter) Samples(filter *cleve.SampleFilter) (*cleve.SampleResult, error) {
	g.SamplesInvoked = true
	return g.SamplesFn(filter)
}

func (g *SampleGetter) Sample(sampleId string) (*cleve.Sample, error) {
	g.SampleInvoked = true
	return g.SampleFn(sampleId)
}

// Mock implementing the gin.SampleSetter interface.
//
// See [mock.RunGetter] for more information.
type SampleSetter struct {
	CreateSampleFn      func(*cleve.Sample) error
	CreateSampleInvoked bool
}

func (s *SampleSetter) CreateSample(sample *cleve.Sample) error {
	s.CreateSampleInvoked = true
	return s.CreateSampleFn(sample)
}
