package mock

import (
	"github.com/gmc-norr/cleve"
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
	AnalysisFn              func(string, ...string) (*cleve.Analysis, error)
	AnalysisInvoked         bool
	CreateAnalysisFn        func(*cleve.Analysis) error
	CreateAnalysisInvoked   bool
	SetAnalysisStateFn      func(string, cleve.State) error
	SetAnalysisStateInvoked bool
	SetAnalysisPathFn       func(string, string) error
	SetAnalysisPathInvoked  bool
	SetAnalysisFilesFn      func(string, []cleve.AnalysisFile) error
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

func (gs *AnalysisGetterSetter) Analysis(analysisId string, runId ...string) (*cleve.Analysis, error) {
	gs.AnalysisInvoked = true
	return gs.AnalysisFn(analysisId, runId...)
}

func (gs *AnalysisGetterSetter) CreateAnalysis(analysis *cleve.Analysis) error {
	gs.CreateAnalysisInvoked = true
	return gs.CreateAnalysisFn(analysis)
}

func (gs *AnalysisGetterSetter) SetAnalysisState(analysisId string, state cleve.State) error {
	gs.SetAnalysisStateInvoked = true
	return gs.SetAnalysisStateFn(analysisId, state)
}

func (gs *AnalysisGetterSetter) SetAnalysisPath(analysisId string, path string) error {
	gs.SetAnalysisPathInvoked = true
	return gs.SetAnalysisPathFn(analysisId, path)
}

func (gs *AnalysisGetterSetter) SetAnalysisFiles(analysisId string, files []cleve.AnalysisFile) error {
	gs.SetAnalysisFilesInvoked = true
	return gs.SetAnalysisFilesFn(analysisId, files)
}
