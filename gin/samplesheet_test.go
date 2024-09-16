package gin

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mock"
	"github.com/gmc-norr/cleve/mongo"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestAddRunSampleSheet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name     string
		url      string
		params   gin.Params
		filename string
		content  string
		error    bool
		code     int
	}{
		{
			name: "minimal samplesheet",
			url:  "/api/runs/run1/samplesheet",
			params: gin.Params{
				gin.Param{Key: "runId", Value: "run1"},
			},
			filename: "SampleSheet.csv",
			content:  "[Header]\nRunName,run1\nRunDescription,this is a description\n[Reads]\n151",
			error:    false,
			code:     http.StatusOK,
		},
		{
			name: "samplesheet with uuid",
			url:  "/api/runs/run1/samplesheet",
			params: gin.Params{
				gin.Param{Key: "runId", Value: "run1"},
			},
			filename: "SampleSheet.csv",
			content:  "[Header]\nRunName,run1\nRunDescription,d351b2b5-84c5-42b8-8b5c-e2520b0b9ace\n[Reads]\n151",
			error:    false,
			code:     http.StatusOK,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ssPath, err := mock.WriteTempFile(t, c.filename, c.content)
			if err != nil {
				t.Fatal(err)
			}

			ss := mock.SampleSheetSetter{}

			ss.CreateSampleSheetFn = func(ss cleve.SampleSheet, opts ...mongo.SampleSheetOption) (*cleve.UpdateResult, error) {
				if c.error {
					return nil, fmt.Errorf("error creating samplesheet")
				}
				ur := cleve.UpdateResult{
					MatchedCount:  0,
					ModifiedCount: 0,
					UpsertedCount: 1,
					UpsertedID:    primitive.NewObjectID(),
				}
				return &ur, nil
			}

			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.Request = httptest.NewRequest("POST", c.url, nil)
			ctx.Params = c.params
			mock.MockJSONBody(ctx, gin.H{"samplesheet": ssPath})
			AddRunSampleSheetHandler(&ss)(ctx)

			if ss.CreateSampleSheetInvoked && c.error {
				t.Error("CreateSampleSheet invoked but should not have been")
			}

			if !ss.CreateSampleSheetInvoked && !c.error {
				t.Error("CreateSampleSheet not invoked but should have been")
			}

			if c.code != w.Code {
				t.Errorf("expected status %d, got %d", c.code, w.Code)
			}
		})
	}
}
