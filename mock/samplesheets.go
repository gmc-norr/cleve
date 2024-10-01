package mock

import (
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
)

type SampleSheetGetter struct {
	SampleSheetFn      func(...mongo.SampleSheetOption) (*cleve.SampleSheet, error)
	SampleSheetInvoked bool
}

func (g *SampleSheetGetter) SampleSheet(opts ...mongo.SampleSheetOption) (*cleve.SampleSheet, error) {
	g.SampleSheetInvoked = true
	return g.SampleSheetFn(opts...)
}

type SampleSheetSetter struct {
	CreateSampleSheetFn      func(cleve.SampleSheet, ...mongo.SampleSheetOption) (*cleve.UpdateResult, error)
	CreateSampleSheetInvoked bool
}

func (s *SampleSheetSetter) CreateSampleSheet(samplesheet cleve.SampleSheet, opts ...mongo.SampleSheetOption) (*cleve.UpdateResult, error) {
	s.CreateSampleSheetInvoked = true
	return s.CreateSampleSheetFn(samplesheet, opts...)
}
