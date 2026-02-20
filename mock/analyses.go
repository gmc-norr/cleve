package mock

import (
	"github.com/gmc-norr/cleve"
	"github.com/google/uuid"
)

// Mock implementing the gin.AnalysisGetterSetter interface.
//
// The *Fn fields are the functions that will get called in the end, and
// the corresponding *Invoked fields register whether the function has been
// called. The interface implementation then just wraps the *Fn functions.
type AnalysisGetterSetter struct {
	AnalysesFn              func(cleve.AnalysisFilter) (cleve.AnalysisResult, error)
	AnalysesInvoked         bool
	AnalysesFilesFn         func(cleve.AnalysisFileFilter) ([]cleve.AnalysisFile, error)
	AnalysesFilesInvoked    bool
	AnalysisFn              func(uuid.UUID, ...string) (*cleve.Analysis, error)
	AnalysisInvoked         bool
	CreateAnalysisFn        func(*cleve.Analysis) error
	CreateAnalysisInvoked   bool
	SetAnalysisStateFn      func(uuid.UUID, cleve.State) error
	SetAnalysisStateInvoked bool
	SetAnalysisPathFn       func(uuid.UUID, string) error
	SetAnalysisPathInvoked  bool
	SetAnalysisFilesFn      func(uuid.UUID, []cleve.AnalysisFile) error
	SetAnalysisFilesInvoked bool
}

func (gs *AnalysisGetterSetter) Analyses(filter cleve.AnalysisFilter) (cleve.AnalysisResult, error) {
	gs.AnalysesInvoked = true
	return gs.AnalysesFn(filter)
}

func (gs *AnalysisGetterSetter) AnalysesFiles(filter cleve.AnalysisFileFilter) ([]cleve.AnalysisFile, error) {
	gs.AnalysesFilesInvoked = true
	return gs.AnalysesFilesFn(filter)
}

func (gs *AnalysisGetterSetter) Analysis(analysisId uuid.UUID, runId ...string) (*cleve.Analysis, error) {
	gs.AnalysisInvoked = true
	return gs.AnalysisFn(analysisId, runId...)
}

func (gs *AnalysisGetterSetter) CreateAnalysis(analysis *cleve.Analysis) error {
	gs.CreateAnalysisInvoked = true
	return gs.CreateAnalysisFn(analysis)
}

func (gs *AnalysisGetterSetter) SetAnalysisState(analysisId uuid.UUID, state cleve.State) error {
	gs.SetAnalysisStateInvoked = true
	return gs.SetAnalysisStateFn(analysisId, state)
}

func (gs *AnalysisGetterSetter) SetAnalysisPath(analysisId uuid.UUID, path string) error {
	gs.SetAnalysisPathInvoked = true
	return gs.SetAnalysisPathFn(analysisId, path)
}

func (gs *AnalysisGetterSetter) SetAnalysisFiles(analysisId uuid.UUID, files []cleve.AnalysisFile) error {
	gs.SetAnalysisFilesInvoked = true
	return gs.SetAnalysisFilesFn(analysisId, files)
}
