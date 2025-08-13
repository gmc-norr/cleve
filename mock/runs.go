package mock

import (
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
)

// Mock implementing the gin.RunGetter interface.
//
// The *Fn fields are the functions that will get called in the end, and
// the corresponding *Invoked fields register whether the function has been
// called. The interface implementation then just wraps the *Fn functions.
type RunGetter struct {
	RunFn       func(string, bool) (*cleve.Run, error)
	RunInvoked  bool
	RunsFn      func(cleve.RunFilter) (cleve.RunResult, error)
	RunsInvoked bool
}

func (g *RunGetter) Run(id string, brief bool) (*cleve.Run, error) {
	g.RunInvoked = true
	return g.RunFn(id, brief)
}

func (g *RunGetter) Runs(filter cleve.RunFilter) (cleve.RunResult, error) {
	g.RunsInvoked = true
	return g.RunsFn(filter)
}

// Mock implementing the gin.RunSetter interface.
//
// See [mock.RunGetter] for more information.
type RunSetter struct {
	CreateRunFn              func(*cleve.Run) error
	CreateRunInvoked         bool
	CreateSampleSheetFn      func(cleve.SampleSheet, ...mongo.SampleSheetOption) (*cleve.UpdateResult, error)
	CreateSampleSheetInvoked bool
	SetRunStateFn            func(string, cleve.State) error
	SetRunStateInvoked       bool
	SetRunPathFn             func(string, string) error
	SetRunPathInvoked        bool
}

func (s *RunSetter) CreateRun(run *cleve.Run) error {
	s.CreateRunInvoked = true
	return s.CreateRunFn(run)
}

func (s *RunSetter) CreateSampleSheet(samplesheet cleve.SampleSheet, opts ...mongo.SampleSheetOption) (*cleve.UpdateResult, error) {
	s.CreateSampleSheetInvoked = true
	return s.CreateSampleSheetFn(samplesheet, opts...)
}

func (s *RunSetter) SetRunState(runId string, state cleve.State) error {
	s.SetRunStateInvoked = true
	return s.SetRunStateFn(runId, state)
}

func (s *RunSetter) SetRunPath(runId string, path string) error {
	s.SetRunPathInvoked = true
	return s.SetRunPathFn(runId, path)
}

// Mock implementing the runHandler for RunWatcher
type RunHandler struct {
	RunsFn             func(cleve.RunFilter) (cleve.RunResult, error)
	RunsInvoked        bool
	SetRunStateFn      func(string, cleve.State) error
	SetRunStateInvoked bool
}

func (h *RunHandler) Runs(filter cleve.RunFilter) (cleve.RunResult, error) {
	h.RunsInvoked = true
	return h.RunsFn(filter)
}

func (h *RunHandler) SetRunState(runId string, state cleve.State) error {
	h.SetRunStateInvoked = true
	return h.SetRunStateFn(runId, state)
}
