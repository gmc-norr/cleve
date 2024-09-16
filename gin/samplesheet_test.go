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

func TestAddSampleSheet(t *testing.T) {
	cases := []struct {
		name         string
		filename     string
		content      string
		runId        string
		code         int
		shouldCreate bool
		matchCount   int
		modifyCount  int
		upsertCount  int
	}{
		{
			name:         "request without path",
			filename:     "",
			content:      "",
			code:         http.StatusBadRequest,
			shouldCreate: false,
		},
		{
			name:         "samplesheet with no uuid and no run id",
			filename:     "SampleSheet.csv",
			content:      "[Header]\nRunDescription,description\n[Reads]\n151",
			code:         http.StatusBadRequest,
			shouldCreate: false,
		},
		{
			name:         "samplesheet with run id and no uuid",
			filename:     "SampleSheet.csv",
			content:      "[Header]\nRunDescription,description\n[Reads]\n151",
			runId:        "run1",
			code:         http.StatusOK,
			shouldCreate: true,
			matchCount:   0,
			upsertCount:  1,
		},
		{
			name:         "samplesheet with uuid id and no run id",
			filename:     "SampleSheet.csv",
			content:      "[Header]\nRunDescription,1311f4cf-8361-494d-9f82-9b231c93c809\n[Reads]\n151",
			code:         http.StatusOK,
			shouldCreate: true,
			matchCount:   0,
			upsertCount:  1,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var sampleSheetPath *string
			if c.filename != "" {
				ssPath, err := mock.WriteTempFile(t, c.filename, c.content)
				if err != nil {
					t.Fatal(err)
				}
				sampleSheetPath = &ssPath
			}

			ss := mock.SampleSheetSetter{}
			ss.CreateSampleSheetFn = func(ss cleve.SampleSheet, opts ...mongo.SampleSheetOption) (*cleve.UpdateResult, error) {
				ur := cleve.UpdateResult{
					MatchedCount:  int64(c.matchCount),
					ModifiedCount: int64(c.modifyCount),
					UpsertedCount: int64(c.upsertCount),
				}
				return &ur, nil
			}

			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.Request = httptest.NewRequest("POST", "/api/samplesheets", nil)
			mock.MockJSONBody(ctx, gin.H{"samplesheet": sampleSheetPath, "run_id": c.runId})

			AddSampleSheetHandler(&ss)(ctx)

			t.Log(w.Body)

			if ss.CreateSampleSheetInvoked && !c.shouldCreate {
				t.Error("CreateSampleSheet invoked but should not have been")
			}

			if !ss.CreateSampleSheetInvoked && c.shouldCreate {
				t.Error("CreateSampleSheet not invoked but should have been")
			}

			if c.code != w.Code {
				t.Errorf("expected status %d, got %d", c.code, w.Code)
			}
		})
	}
}
