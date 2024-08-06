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
	var ks mock.APIKeyService
	var rs mock.RunService

	db := mongo.DB{
		Keys: &ks,
		Runs: &rs,
	}

	table := []struct {
		Runs   cleve.RunResult
		Error  error
		Params gin.Params
	}{
		{
			cleve.RunResult{
				Runs: nil,
			},
			nil,
			gin.Params{},
		},
		{
			cleve.RunResult{
				Runs: []*cleve.Run{novaseq1, novaseq2, nextseq1},
			},
			nil,
			gin.Params{},
		},
		{
			cleve.RunResult{
				Runs: []*cleve.Run{novaseq1, novaseq2, nextseq1},
			},
			nil,
			gin.Params{gin.Param{Key: "brief", Value: "true"}},
		},
		{
			cleve.RunResult{
				Runs: []*cleve.Run{novaseq1, novaseq2},
			},
			nil,
			gin.Params{
				gin.Param{Key: "brief", Value: "true"},
				gin.Param{Key: "platform", Value: "NovaSeq"},
			},
		},
	}

	for _, v := range table {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		rs.AllFn = func(filter cleve.RunFilter) (cleve.RunResult, error) {
			return v.Runs, v.Error
		}

		c.Params = v.Params
		RunsHandler(&db)(c)

		if !rs.AllInvoked {
			t.Fatal("RunService.All not invoked")
		}

		if w.Code != 200 {
			t.Fatalf("HTTP status %d != 200", w.Code)
		}

		b, _ := io.ReadAll(w.Body)
		count := strings.Count(string(b), "experiment_name")

		if count != len(v.Runs.Runs) {
			t.Fatalf("found %d runs, expected %d", count, len(v.Runs.Runs))
		}
	}
}

func TestRunHandler(t *testing.T) {
	gin.SetMode("test")
	var ks mock.APIKeyService
	var rs mock.RunService

	db := mongo.DB{
		Keys: &ks,
		Runs: &rs,
	}

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
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		rs.GetFn = func(run_id string, brief bool) (*cleve.Run, error) {
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
		RunHandler(&db)(c)

		if !rs.GetInvoked {
			t.Fatal("`RunService.Get` not invoked")
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
		runPath string
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
			var ks mock.APIKeyService
			var rs mock.RunService
			var ss mock.SampleSheetService

			if _, err := os.Stat(v.runPath); errors.Is(err, os.ErrNotExist) {
                t.Skip("test data not found, skipping")
            }

			db := mongo.DB{
				Keys:         &ks,
				Runs:         &rs,
				SampleSheets: &ss,
			}
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			rs.CreateFn = func(run *cleve.Run) error {
				return nil
			}
			ss.CreateFn = func(runId string, samplesheet cleve.SampleSheet) (*cleve.UpdateResult, error) {
				return nil, nil
			}

			c.Request = httptest.NewRequest(http.MethodPost, "/runs", bytes.NewBuffer(v.data))
			AddRunHandler(&db)(c)

			if rs.CreateInvoked && !v.createInvoked {
				t.Error(`RunService.Create was invoked, but it shouldn't have been`)
			} else if !rs.CreateInvoked && v.createInvoked {
				t.Error(`RunService.Create was not invoked, but it should have been`)
			}

			if ss.CreateInvoked && !v.hasSamplesheet {
				t.Error(`SampleSheetService.Create was invoked, but it shouldn't have been`)
			} else if !ss.CreateInvoked && v.hasSamplesheet {
				t.Error(`SampleSheetService.Create was not invoked, but it should have been`)
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
			var ks mock.APIKeyService
			var rs mock.RunService
			var ss mock.SampleSheetService

			db := mongo.DB{
				Keys:         &ks,
				Runs:         &rs,
				SampleSheets: &ss,
			}
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			rs.SetPathFn = func(runId string, path string) error {
				s, err := os.Stat(path)
				if err != nil {
					return err
				}
				if !s.IsDir() {
					return fmt.Errorf(`%s is not a directory`, path)
				}
				return nil
			}
			ss.CreateFn = func(runId string, samplesheet cleve.SampleSheet) (*cleve.UpdateResult, error) {
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
			UpdateRunPathHandler(&db)(c)

			if ss.CreateInvoked && !v.hasSampleSheet {
				t.Error(`SampleSheetService.Create was invoked, but it should not have been`)
			}
			if !ss.CreateInvoked && v.hasSampleSheet {
				t.Error(`SampleSheetService.Create was not invoked, but it should have been`)
			}

			if w.Code != v.code {
				t.Errorf(`Got HTTP %d, expected %d`, w.Code, v.code)
			}

			// Reset invoked fields
			ss.CreateInvoked = false
			rs.SetPathInvoked = false
		})
	}
}
