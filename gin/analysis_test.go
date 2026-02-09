package gin

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mock"
	"github.com/gmc-norr/cleve/mongo"
)

func TestAddAnalysis(t *testing.T) {
	gin.SetMode("test")
	testcases := []struct {
		name   string
		exists bool
		data   []byte
		code   int
		error  bool
	}{
		{
			name:   "analysis without files",
			exists: false,
			data:   []byte(`{"analysis_id": "analysis1", "path": "/path/to/analysis1", "run_id": "run1", "state": "ready", "software": "software1", "software_version": "1.0.0"}`),
			code:   http.StatusOK,
			error:  false,
		},
		{
			name:   "analysis with output files",
			exists: false,
			data:   []byte(`{"analysis_id": "analysis1", "path": "/path/to/analysis1", "run_id": "run1", "state": "ready", "software": "software1", "software_version": "1.0.0", "output_files": [{"path": "fastq/sample1_1.fastq.gz", "type": "fastq", "level": "sample", "parent_id": "sample1"}, {"path": "fastq/sample1_2.fastq.gz", "type": "fastq", "level": "sample", "parent_id": "sample1"}]}`),
			code:   http.StatusOK,
			error:  false,
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			gs := mock.AnalysisGetterSetter{}
			gs.CreateAnalysisFn = func(a *cleve.Analysis) error {
				if c.error {
					return fmt.Errorf("an error occurred")
				}
				return nil
			}
			gs.AnalysisFn = func(s1 string, s2 ...string) (*cleve.Analysis, error) {
				if c.exists {
					return &cleve.Analysis{}, nil
				}
				return nil, mongo.ErrNoDocuments
			}
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)

			ctx.Request = httptest.NewRequest(http.MethodPost, "/analyses", bytes.NewBuffer(c.data))
			ctx.Request.Header.Set("Content-Type", "application/json")

			AddAnalysisHandler(&gs)(ctx)

			if w.Code != c.code {
				t.Errorf("expected status code %d, got %d: %s", c.code, w.Code, w.Body)
			}
		})
	}
}

func TestUpdateAnalysis(t *testing.T) {
	gin.SetMode("test")
	testcases := []struct {
		name             string
		existingAnalysis *cleve.Analysis
		exists           bool
		pathUpdated      bool
		stateUpdated     bool
		filesUpdated     bool
		data             []byte
		code             int
	}{
		{
			name: "update analysis state",
			existingAnalysis: &cleve.Analysis{
				AnalysisId:   "analysis1",
				Path:         "/path/to/analysis1",
				StateHistory: cleve.StateHistory{cleve.TimedRunState{State: cleve.StatePending, Time: time.Now()}},
			},
			exists:       true,
			stateUpdated: true,
			data:         []byte(`{"state": "ready"}`),
			code:         http.StatusOK,
		},
		{
			name: "update analysis path",
			existingAnalysis: &cleve.Analysis{
				AnalysisId:   "analysis1",
				Path:         "/path/to/analysis1",
				StateHistory: cleve.StateHistory{cleve.TimedRunState{State: cleve.StatePending, Time: time.Now()}},
			},
			exists:      true,
			pathUpdated: true,
			data:        []byte(`{"path": "/new/path/to/analysis1"}`),
			code:        http.StatusOK,
		},
		{
			name: "update analysis files (relative paths)",
			existingAnalysis: &cleve.Analysis{
				AnalysisId:   "analysis1",
				Path:         "/path/to/analysis1",
				StateHistory: cleve.StateHistory{cleve.TimedRunState{State: cleve.StatePending, Time: time.Now()}},
			},
			exists:       true,
			filesUpdated: true,
			data:         []byte(`{"files": [{"path": "output/sample1.data", "level": "sample", "parent_id": "analysis1"}]}`),
			code:         http.StatusOK,
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			gs := mock.AnalysisGetterSetter{}
			gs.AnalysisFn = func(s1 string, s2 ...string) (*cleve.Analysis, error) {
				if c.exists {
					return nil, mongo.ErrNoDocuments
				}
				return &cleve.Analysis{}, nil
			}
			gs.SetAnalysisPathFn = func(string, string) error {
				return nil
			}
			gs.SetAnalysisStateFn = func(string, cleve.State) error {
				return nil
			}
			gs.SetAnalysisFilesFn = func(string, []cleve.AnalysisFile) error {
				return nil
			}
			gs.CreateAnalysisFn = func(a *cleve.Analysis) error {
				return nil
			}
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)

			ctx.Request = httptest.NewRequest(http.MethodPatch, "/analyses/analysis_1", bytes.NewBuffer(c.data))
			ctx.Request.Header.Set("Content-Type", "application/json")

			UpdateAnalysisHandler(&gs)(ctx)

			if w.Code != c.code {
				t.Errorf("expected status code %d, got %d: %s", c.code, w.Code, w.Body)
			}

			if c.stateUpdated != gs.SetAnalysisStateInvoked {
				t.Errorf("state updated: %t, expected it: %t", gs.SetAnalysisStateInvoked, c.stateUpdated)
			}

			if c.pathUpdated != gs.SetAnalysisPathInvoked {
				t.Errorf("state updated: %t, expected it: %t", gs.SetAnalysisPathInvoked, c.pathUpdated)
			}

			if c.filesUpdated != gs.SetAnalysisFilesInvoked {
				t.Errorf("state updated: %t, expected it: %t", gs.SetAnalysisFilesInvoked, c.filesUpdated)
			}
		})
	}
}
