package gin

import (
	"bytes"
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
	"github.com/gmc-norr/cleve/mock"
	"github.com/gmc-norr/cleve/mongo"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var novaseq1 *cleve.Run = &cleve.Run{
	ID:             primitive.NewObjectID(),
	RunID:          "run1",
	ExperimentName: "experiment 1",
	Path:           "/path/to/run1",
	Platform:       "NovaSeq",
	Created:        time.Now(),
	StateHistory:   nil,
	RunParameters:  &cleve.NovaSeqParameters{},
}

var novaseq2 *cleve.Run = &cleve.Run{
	ID:             primitive.NewObjectID(),
	RunID:          "run3",
	ExperimentName: "experiment 3",
	Path:           "/path/to/run3",
	Platform:       "NovaSeq",
	Created:        time.Now(),
	StateHistory:   nil,
	RunParameters:  &cleve.NovaSeqParameters{},
}

var nextseq1 *cleve.Run = &cleve.Run{
	ID:             primitive.NewObjectID(),
	RunID:          "run2",
	ExperimentName: "experiment 2",
	Path:           "/path/to/run2",
	Platform:       "NextSeq",
	Created:        time.Now(),
	StateHistory:   nil,
	RunParameters:  &cleve.NextSeqParameters{},
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
				Runs: nil,
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
			"brief",
			cleve.RunResult{
				Runs: []*cleve.Run{novaseq1, novaseq2, nextseq1},
			},
			nil,
			"/api/runs?brief=true",
			cleve.RunFilter{
				Brief: true,
				PaginationFilter: cleve.PaginationFilter{
					Page:     1,
					PageSize: 10,
				},
			},
		},
		{
			"brief and platform filter",
			cleve.RunResult{
				Runs: []*cleve.Run{novaseq1, novaseq2},
			},
			nil,
			"/api/runs?brief=true&platform=NovaSeq",
			cleve.RunFilter{
				Brief:    true,
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
			"/api/runs?brief=true&platform=NovaSeq&page=3&page_size=5",
			cleve.RunFilter{
				Brief:    true,
				Platform: "NovaSeq",
				PaginationFilter: cleve.PaginationFilter{
					Page:     3,
					PageSize: 5,
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
			count := strings.Count(string(b), "experiment_name")

			if count != len(v.Runs.Runs) {
				t.Fatalf("found %d runs, expected %d", count, len(v.Runs.Runs))
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
				gin.Param{Key: "brief", Value: "false"},
			},
			http.StatusOK,
			`"experiment_name":"experiment 3"`,
		},
		{
			"run3",
			novaseq2,
			gin.Params{
				gin.Param{Key: "runId", Value: "run2"},
				gin.Param{Key: "brief", Value: "true"},
			},
			http.StatusOK,
			`"experiment_name":"experiment 3"`,
		},
	}

	for _, v := range table {
		rg.RunInvoked = false

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		rg.RunFn = func(run_id string, brief bool) (*cleve.Run, error) {
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
			"path missing",
			"",
			[]byte(`{"path": "/path/to/run", "state": "ready"}`),
			http.StatusInternalServerError,
			false,
			false,
		},
		{
			"valid run",
			"/home/nima18/git/cleve/test_data/novaseq_full",
			[]byte(`{"path": "/home/nima18/git/cleve/test_data/novaseq_full", "state": "ready"}`),
			http.StatusOK,
			true,
			true,
		},
		{
			"missing state",
			"/home/nima18/git/cleve/test_data/novaseq_full",
			[]byte(`{"path": "/home/nima18/git/cleve/test_data/novaseq_full"}`),
			http.StatusBadRequest,
			false,
			false,
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
			rs.CreateSampleSheetFn = func(runId string, samplesheet cleve.SampleSheet) (*cleve.UpdateResult, error) {
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
			rs.CreateSampleSheetFn = func(runId string, samplesheet cleve.SampleSheet) (*cleve.UpdateResult, error) {
				return nil, nil
			}

			runDir := "/nonexistent/path/to/run"
			if v.destinationExists {
				runDir = t.TempDir()
				if v.hasSampleSheet {
					os.OpenFile(filepath.Join(runDir, "SampleSheet.csv"), os.O_RDONLY|os.O_CREATE, 0644)
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
