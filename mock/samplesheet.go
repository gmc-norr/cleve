package mock

import (
	"github.com/gmc-norr/cleve"
)

type SampleSheetService struct {
	AllFn           func() ([]cleve.SampleSheet, error)
	AllInvoked      bool
	CreateFn        func(string, cleve.SampleSheet) (*cleve.UpdateResult, error)
	CreateInvoked   bool
	DeleteFn        func(string) error
	DeleteInvoked   bool
	GetFn           func(string) (cleve.SampleSheet, error)
	GetInvoked      bool
	GetIndexFn      func() ([]map[string]string, error)
	GetIndexInvoked bool
	SetIndexFn      func() (string, error)
	SetIndexInvoked bool
}

func (s *SampleSheetService) All() ([]cleve.SampleSheet, error) {
	s.AllInvoked = true
	return s.AllFn()
}

func (s *SampleSheetService) Create(runId string, samplesheet cleve.SampleSheet) (*cleve.UpdateResult, error) {
	s.CreateInvoked = true
	return s.CreateFn(runId, samplesheet)
}

func (s *SampleSheetService) Delete(runId string) error {
	s.DeleteInvoked = true
	return s.DeleteFn(runId)
}

func (s *SampleSheetService) Get(runId string) (cleve.SampleSheet, error) {
	s.GetInvoked = true
	return s.GetFn(runId)
}

func (s *SampleSheetService) GetIndex() ([]map[string]string, error) {
	s.GetIndexInvoked = true
	return s.GetIndexFn()
}

func (s *SampleSheetService) SetIndex() (string, error) {
	s.SetIndexInvoked = true
	return s.SetIndexFn()
}
