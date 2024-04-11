package gin

import (
	"io"
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
		Runs   []*cleve.Run
		Error  error
		Params gin.Params
	}{
		{
			nil,
			nil,
			gin.Params{},
		},
		{
			[]*cleve.Run{novaseq1, novaseq2, nextseq1},
			nil,
			gin.Params{},
		},
		{
			[]*cleve.Run{novaseq1, novaseq2, nextseq1},
			nil,
			gin.Params{gin.Param{Key: "brief", Value: "true"}},
		},
		{
			[]*cleve.Run{novaseq1, novaseq2},
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

		rs.AllFn = func(bool, string, string) ([]*cleve.Run, error) {
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
		count := strings.Count(string(b), "run_id")

		if count != len(v.Runs) {
			t.Fatalf("found %d runs, expected %d", count, len(v.Runs))
		}
	}
}
