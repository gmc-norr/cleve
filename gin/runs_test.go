package gin

import (
	"io"
	"net/http"
	"net/http/httptest"
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
