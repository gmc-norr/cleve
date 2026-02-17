package gin

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mock"
	"github.com/gmc-norr/cleve/mongo"
)

func TestAddAnalysis(t *testing.T) {
	gin.SetMode("test")
	tmpdir := t.TempDir()
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
			data:   fmt.Appendf([]byte{}, `{"analysis_id": "analysis1", "path": "%s/analysis1", "run_id": "run1", "state": "ready", "software": "software1", "software_version": "1.0.0"}`, tmpdir),
			code:   http.StatusOK,
			error:  false,
		},
		{
			name:   "analysis with output files",
			exists: false,
			data:   fmt.Appendf([]byte{}, `{"analysis_id": "analysis1", "path": "%s/analysis1", "run_id": "run1", "state": "ready", "software": "software1", "software_version": "1.0.0", "output_files": [{"path": "fastq/sample1_1.fastq.gz", "type": "fastq", "level": "sample", "parent_id": "sample1"}, {"path": "fastq/sample1_2.fastq.gz", "type": "fastq", "level": "sample", "parent_id": "sample1"}]}`, tmpdir),
			code:   http.StatusOK,
			error:  false,
		},
	}

	// Create mock files
	fq1 := filepath.Join(tmpdir, "analysis1/fastq/sample1_1.fastq.gz")
	fq2 := filepath.Join(tmpdir, "analysis1/fastq/sample1_2.fastq.gz")
	if err := os.MkdirAll(filepath.Dir(fq1), 0o777); err != nil {
		t.Fatal(err)
	}
	f1, err := os.Create(fq1)
	if err != nil {
		t.Fatal(err)
	}
	_ = f1.Close()
	f2, err := os.Create(fq2)
	if err != nil {
		t.Fatal(err)
	}
	_ = f2.Close()

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
	tmpdir := t.TempDir()
	gin.SetMode("test")
	testcases := []struct {
		name             string
		existingAnalysis *cleve.Analysis
		exists           bool
		pathUpdated      bool
		stateUpdated     bool
		filesUpdated     bool
		outputFiles      []string
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
				Path:         filepath.Join(tmpdir, "analysis1"),
				StateHistory: cleve.StateHistory{cleve.TimedRunState{State: cleve.StatePending, Time: time.Now()}},
			},
			exists:       true,
			filesUpdated: true,
			outputFiles: []string{
				filepath.Join(tmpdir, "analysis1/output/sample1.tsv"),
			},
			data: []byte(`{"files": [{"path": "output/sample1.tsv", "level": "sample", "parent_id": "analysis1", "type": "text"}]}`),
			code: http.StatusOK,
		},
		{
			name: "update analysis files (invalid path)",
			existingAnalysis: &cleve.Analysis{
				AnalysisId:   "analysis1",
				Path:         filepath.Join(tmpdir, "analysis1"),
				StateHistory: cleve.StateHistory{cleve.TimedRunState{State: cleve.StatePending, Time: time.Now()}},
			},
			exists: true,
			data:   []byte(`{"files": [{"path": "../output/sample1.tsv", "level": "sample", "parent_id": "sample1", "type": "text"}]}`),
			code:   http.StatusBadRequest,
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			for _, path := range c.outputFiles {
				if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
					t.Fatal(err)
				}
				f, err := os.Create(path)
				if err != nil {
					t.Fatal(err)
				}
				_ = f.Close()
				defer func() {
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
				}()
			}
			gs := mock.AnalysisGetterSetter{}
			gs.AnalysisFn = func(s1 string, s2 ...string) (*cleve.Analysis, error) {
				if !c.exists {
					return nil, mongo.ErrNoDocuments
				}
				return c.existingAnalysis, nil
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

			ctx.Request = httptest.NewRequest(http.MethodPatch, "/analyses/analysis1", bytes.NewBuffer(c.data))
			ctx.Params = append(ctx.Params, gin.Param{Key: "analysisId", Value: c.existingAnalysis.AnalysisId})
			ctx.Request.Header.Set("Content-Type", "application/json")

			UpdateAnalysisHandler(&gs)(ctx)

			if w.Code != c.code {
				t.Errorf("expected status code %d, got %d: %s", c.code, w.Code, w.Body)
			}

			if c.stateUpdated != gs.SetAnalysisStateInvoked {
				t.Errorf("state updated: %t, expected it: %t", gs.SetAnalysisStateInvoked, c.stateUpdated)
			}

			if c.pathUpdated != gs.SetAnalysisPathInvoked {
				t.Errorf("path updated: %t, expected it: %t", gs.SetAnalysisPathInvoked, c.pathUpdated)
			}

			if c.filesUpdated != gs.SetAnalysisFilesInvoked {
				t.Errorf("files updated: %t, expected it: %t", gs.SetAnalysisFilesInvoked, c.filesUpdated)
			}
		})
	}
}
