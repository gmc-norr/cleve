package mock

import (
	"github.com/gmc-norr/cleve"
)

type RunService struct {
	AllFn                     func(bool, string, string) ([]*cleve.Run, error)
	AllInvoked                bool
	CreateFn                  func(*cleve.Run) error
	CreateInvoked             bool
	DeleteFn                  func(string) error
	DeleteInvoked             bool
	GetFn                     func(string, bool) (*cleve.Run, error)
	GetInvoked                bool
	GetAnalysesFn             func(string) ([]*cleve.Analysis, error)
	GetAnalysesInvoked        bool
	GetAnalysisFn             func(string, string) (*cleve.Analysis, error)
	GetAnalysisInvoked        bool
	CreateAnalysisFn          func(string, *cleve.Analysis) error
	CreateAnalysisInvoked     bool
	SetAnalysisStateFn        func(string, string, cleve.RunState) error
	SetAnalysisStateInvoked   bool
	SetAnalysisSummaryFn      func(string, string, *cleve.AnalysisSummary) error
	SetAnalysisSummaryInvoked bool
	GetStateHistoryFn         func(string) ([]cleve.TimedRunState, error)
	GetStateHistoryInvoked    bool
	SetStateFn                func(string, cleve.RunState) error
	SetStateInvoked           bool
	GetIndexFn                func() ([]map[string]string, error)
	GetIndexInvoked           bool
	SetIndexFn                func() (string, error)
	SetIndexInvoked           bool
}

func (s *RunService) All(brief bool, platform string, state string) ([]*cleve.Run, error) {
	s.AllInvoked = true
	return s.AllFn(brief, platform, state)
}

func (s *RunService) Create(r *cleve.Run) error {
	s.CreateInvoked = true
	return s.CreateFn(r)
}

func (s *RunService) Delete(id string) error {
	s.DeleteInvoked = true
	return s.DeleteFn(id)
}

func (s *RunService) Get(id string, brief bool) (*cleve.Run, error) {
	s.GetInvoked = true
	return s.GetFn(id, brief)
}

func (s *RunService) GetAnalyses(id string) ([]*cleve.Analysis, error) {
	s.GetAnalysesInvoked = true
	return s.GetAnalysesFn(id)
}

func (s *RunService) GetAnalysis(run_id string, analysis_id string) (*cleve.Analysis, error) {
	s.GetAnalysisInvoked = true
	return s.GetAnalysisFn(run_id, analysis_id)
}

func (s *RunService) CreateAnalysis(run_id string, a *cleve.Analysis) error {
	s.CreateAnalysisInvoked = true
	return s.CreateAnalysisFn(run_id, a)
}

func (s *RunService) SetAnalysisState(run_id string, analysis_id string, state cleve.RunState) error {
	s.SetAnalysisStateInvoked = true
	return s.SetAnalysisStateFn(run_id, analysis_id, state)
}

func (s *RunService) SetAnalysisSummary(run_id string, analysis_id string, summary *cleve.AnalysisSummary) error {
	s.SetAnalysisSummaryInvoked = true
	return s.SetAnalysisSummaryFn(run_id, analysis_id, summary)
}

func (s *RunService) GetStateHistory(run_id string) ([]cleve.TimedRunState, error) {
	s.GetStateHistoryInvoked = true
	return s.GetStateHistoryFn(run_id)
}

func (s *RunService) SetState(run_id string, state cleve.RunState) error {
	s.SetStateInvoked = true
	return s.SetStateFn(run_id, state)
}

func (s *RunService) GetIndex() ([]map[string]string, error) {
	s.GetIndexInvoked = true
	return s.GetIndexFn()
}

func (s *RunService) SetIndex() (string, error) {
	s.SetIndexInvoked = true
	return s.SetIndexFn()
}
