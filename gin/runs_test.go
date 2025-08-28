package gin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/interop"
	"github.com/gmc-norr/cleve/mock"
	"github.com/gmc-norr/cleve/mongo"
)

var novaseq1 *cleve.Run = &cleve.Run{
	RunID:          "run1",
	ExperimentName: "experiment 1",
	Path:           "/path/to/run1",
	Platform:       "NovaSeq",
	Created:        time.Now(),
	StateHistory:   nil,
	RunParameters:  interop.RunParameters{},
}

var novaseq2 *cleve.Run = &cleve.Run{
	RunID:          "run3",
	ExperimentName: "experiment 3",
	Path:           "/path/to/run3",
	Platform:       "NovaSeq",
	Created:        time.Now(),
	StateHistory:   nil,
	RunParameters:  interop.RunParameters{},
}

var nextseq1 *cleve.Run = &cleve.Run{
	RunID:          "run2",
	ExperimentName: "experiment 2",
	Path:           "/path/to/run2",
	Platform:       "NextSeq",
	Created:        time.Now(),
	StateHistory:   nil,
	RunParameters:  interop.RunParameters{},
}

func TestRunsHandler(t *testing.T) {
	gin.SetMode("test")
	rg := mock.RunGetter{}

	table := []struct {
		Name           string
		Runs           cleve.RunResult
		Error          error
		URL            string
		ExpectedFilter cleve.RunFilter
	}{
		{
			"no params, no results",
			cleve.RunResult{
				Runs: []*cleve.Run{},
			},
			nil,
			"/api/runs",
			cleve.RunFilter{
				PaginationFilter: cleve.PaginationFilter{
					Page:     1,
					PageSize: 10,
				},
			},
		},
		{
			"no params, with results",
			cleve.RunResult{
				Runs: []*cleve.Run{novaseq1, novaseq2, nextseq1},
			},
			nil,
			"/api/runs",
			cleve.RunFilter{
				PaginationFilter: cleve.PaginationFilter{
					Page:     1,
					PageSize: 10,
				},
			},
		},
		{
			"platform filter",
			cleve.RunResult{
				Runs: []*cleve.Run{novaseq1, novaseq2},
			},
			nil,
			"/api/runs?platform=NovaSeq",
			cleve.RunFilter{
				Platform: "NovaSeq",
				PaginationFilter: cleve.PaginationFilter{
					Page:     1,
					PageSize: 10,
				},
			},
		},
		{
			"third page, 5 results per page",
			cleve.RunResult{
				Runs: []*cleve.Run{novaseq1, novaseq2},
			},
			nil,
			"/api/runs?platform=NovaSeq&page=3&page_size=5",
			cleve.RunFilter{
				Platform: "NovaSeq",
				PaginationFilter: cleve.PaginationFilter{
					Page:     3,
					PageSize: 5,
				},
			},
		},
		{
			"filter that returns no results",
			cleve.RunResult{
				Runs: []*cleve.Run{},
			},
			nil,
			"/api/runs?platform=NovaSeq",
			cleve.RunFilter{
				Platform: "NovaSeq",
				PaginationFilter: cleve.PaginationFilter{
					Page:     1,
					PageSize: 10,
				},
			},
		},
	}

	for _, v := range table {
		t.Run(v.Name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", v.URL, nil)

			rg.RunsFn = func(filter cleve.RunFilter) (cleve.RunResult, error) {
				return v.Runs, v.Error
			}

			filter, err := getRunFilter(c)
			if err != nil {
				t.Errorf("error parsing filter: %s", err)
			}
			if filter != v.ExpectedFilter {
				t.Errorf("expected filter %v, got %v", v.ExpectedFilter, filter)
			}

			RunsHandler(&rg)(c)

			if !rg.RunsInvoked {
				t.Fatal("Runs not invoked")
			}

			if w.Code != 200 {
				t.Fatalf("HTTP status %d != 200", w.Code)
			}

			b, _ := io.ReadAll(w.Body)
			runResult := map[string]any{}
			if err := json.Unmarshal(b, &runResult); err != nil {
				t.Error(err)
			}

			if runResult["runs"] == nil {
				t.Error("runs is nil")
			}

			nRuns := len(runResult["runs"].([]any))
			if nRuns != len(v.Runs.Runs) {
				t.Fatalf("found %d runs, expected %d", nRuns, len(v.Runs.Runs))
			}
		})
	}
}

func TestRunHandler(t *testing.T) {
	gin.SetMode("test")
	rg := mock.RunGetter{}

	table := []struct {
		RunID  string
		Run    *cleve.Run
		Params gin.Params
		Code   int
		Body   string
	}{
		{
			"nosuchrun",
			nil,
			gin.Params{
				gin.Param{Key: "runId", Value: "nosuchrun"},
			},
			http.StatusNotFound,
			`error`,
		},
		{
			"run1",
			novaseq1,
			gin.Params{
				gin.Param{Key: "runId", Value: "run1"},
			},
			http.StatusOK,
			`"experiment_name":"experiment 1"`,
		},
		{
			"run3",
			novaseq2,
			gin.Params{
				gin.Param{Key: "runId", Value: "run2"},
			},
			http.StatusOK,
			`"experiment_name":"experiment 3"`,
		},
		{
			"run3",
			novaseq2,
			gin.Params{
				gin.Param{Key: "runId", Value: "run2"},
			},
			http.StatusOK,
			`"experiment_name":"experiment 3"`,
		},
	}

	for _, v := range table {
		rg.RunInvoked = false

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		rg.RunFn = func(run_id string) (*cleve.Run, error) {
			switch run_id {
			case "run1":
				return novaseq1, nil
			case "run2":
				return novaseq2, nil
			case "run3":
				return nextseq1, nil
			default:
				return nil, mongo.ErrNoDocuments
			}
		}

		c.Params = v.Params
		RunHandler(&rg)(c)

		if !rg.RunInvoked {
			t.Fatal("Run not invoked")
		}

		if w.Code != v.Code {
			t.Fatalf(`Got HTTP %d, expected %d for "%s"`, w.Code, v.Code, v.RunID)
		}

		b, _ := io.ReadAll(w.Body)
		if strings.Count(string(b), v.Body) == 0 {
			t.Fatalf(`Expected %v in body: %v`, v.Body, string(b))
		}
	}
}

func TestAddRunHandler(t *testing.T) {
	gin.SetMode("test")

	cases := []struct {
		name           string
		runPath        string
		data           []byte
		code           int
		createInvoked  bool
		hasSamplesheet bool
	}{
		{
			name:           "path missing",
			runPath:        "",
			data:           []byte(`{"path": "/path/to/run", "state": "ready"}`),
			code:           http.StatusInternalServerError,
			createInvoked:  false,
			hasSamplesheet: false,
		},
		{
			name:           "valid run",
			runPath:        "/home/nima18/git/cleve/testdata/20250305_LH00352_0035_A222VYLLT1",
			data:           []byte(`{"path": "/home/nima18/git/cleve/testdata/20250305_LH00352_0035_A222VYLLT1", "state": "ready"}`),
			code:           http.StatusOK,
			createInvoked:  true,
			hasSamplesheet: true,
		},
	}

	for _, v := range cases {
		t.Run(v.name, func(t *testing.T) {
			rs := mock.RunSetter{}

			if _, err := os.Stat(v.runPath); errors.Is(err, os.ErrNotExist) && v.name != "path missing" {
				t.Skip("test data not found, skipping")
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			rs.CreateRunFn = func(run *cleve.Run) error {
				return nil
			}
			rs.CreateSampleSheetFn = func(samplesheet cleve.SampleSheet, opts ...mongo.SampleSheetOption) (*cleve.UpdateResult, error) {
				return nil, nil
			}

			c.Request = httptest.NewRequest(http.MethodPost, "/runs", bytes.NewBuffer(v.data))
			AddRunHandler(&rs)(c)

			if rs.CreateRunInvoked && !v.createInvoked {
				t.Error(`CreateRun was invoked, but it shouldn't have been`)
			} else if !rs.CreateRunInvoked && v.createInvoked {
				t.Error(`CreateRun was not invoked, but it should have been`)
			}

			if rs.CreateSampleSheetInvoked && !v.hasSamplesheet {
				t.Error(`CreateSampleSheet was invoked, but it shouldn't have been`)
			} else if !rs.CreateSampleSheetInvoked && v.hasSamplesheet {
				t.Error(`CreateSampleSheet was not invoked, but it should have been`)
			}

			if w.Code != v.code {
				t.Errorf(`Got HTTP %d, expected %d`, w.Code, v.code)
				t.Errorf(`Message: %s`, w.Body.String())
			}
		})
	}
}

func TestUpdateRunPathHandler(t *testing.T) {
	gin.SetMode("test")

	cases := []struct {
		name              string
		runId             string
		code              int
		destinationExists bool
		hasSampleSheet    bool
	}{
		{
			"moved run with samplesheet",
			"run1",
			http.StatusOK,
			true,
			true,
		},
		{
			"moved run without samplesheet",
			"run1",
			http.StatusOK,
			true,
			false,
		},
		{
			"moved run with missing path",
			"run1",
			http.StatusInternalServerError,
			false,
			false,
		},
	}

	for _, v := range cases {
		t.Run(v.name, func(t *testing.T) {
			rs := mock.RunSetter{}
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			rs.SetRunPathFn = func(runId string, path string) error {
				s, err := os.Stat(path)
				if err != nil {
					return err
				}
				if !s.IsDir() {
					return fmt.Errorf(`%s is not a directory`, path)
				}
				return nil
			}
			rs.CreateSampleSheetFn = func(samplesheet cleve.SampleSheet, opts ...mongo.SampleSheetOption) (*cleve.UpdateResult, error) {
				return nil, nil
			}

			runDir := "/nonexistent/path/to/run"
			if v.destinationExists {
				runDir = t.TempDir()
				if v.hasSampleSheet {
					f, err := os.Create(filepath.Join(runDir, "SampleSheet.csv"))
					if err != nil {
						t.Fatal(err)
					}
					if _, err := io.WriteString(f, "[Header]\nRunName,run1\n[Reads]\n151"); err != nil {
						t.Fatal(err)
					}
				}
			}

			c.Request = httptest.NewRequest(
				http.MethodPost,
				"/runs/run1/path",
				bytes.NewBuffer([]byte(fmt.Sprintf(`{"run_id": "%s", "path": "%s"}`, v.runId, runDir))),
			)
			UpdateRunPathHandler(&rs)(c)

			if rs.CreateSampleSheetInvoked && !v.hasSampleSheet {
				t.Error(`CreateSampleSheet was invoked, but it should not have been`)
			}
			if !rs.CreateSampleSheetInvoked && v.hasSampleSheet {
				t.Error(`CreateSampleSheet was not invoked, but it should have been`)
			}

			if w.Code != v.code {
				t.Errorf(`Got HTTP %d, expected %d`, w.Code, v.code)
			}
		})
	}
}
