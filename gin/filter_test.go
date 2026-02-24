package gin

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetAnalysisFileFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testcases := []struct {
		// test name
		name string
		// parameters
		analysisId string
		runId      string
		// query parameters
		qAnalysisId   string
		qRunId        string
		parentId      string
		fileType      string
		analysisLevel string
		fileName      string
		pattern       string
		// for validation
		isValid bool
	}{
		// IDs from both parameters and query parameters,
		// parameters should be prioritised
		{
			name:        "prioritise ids from params",
			analysisId:  "8ff19e60-bab1-464f-9d03-de76bbd215aa",
			runId:       "run1",
			qAnalysisId: "892224d9-4865-47c5-83a7-0ef20fba77b4",
			qRunId:      "run2",
			fileName:    "test.txt",
			isValid:     true,
		},
		// IDs only from query parameters
		{
			name:        "ids only from query params",
			qAnalysisId: "8ff19e60-bab1-464f-9d03-de76bbd215aa",
			qRunId:      "run2",
			fileType:    "text",
			isValid:     true,
		},
		// Invalid filter, missing properties
		{
			name:        "invalid filter (missing parameters)",
			qAnalysisId: "8ff19e60-bab1-464f-9d03-de76bbd215aa",
			qRunId:      "run2",
			isValid:     false,
		},
		// Invalid filter, invalid file type
		{
			name:        "invalid filter (invalid file type)",
			qAnalysisId: "8ff19e60-bab1-464f-9d03-de76bbd215aa",
			qRunId:      "run2",
			fileType:    "no_such_filetype",
			isValid:     false,
		},
		// Invalid filter, invalid level
		{
			name:          "invalid filter (invalid level)",
			qAnalysisId:   "8ff19e60-bab1-464f-9d03-de76bbd215aa",
			qRunId:        "run2",
			analysisLevel: "no_such_level",
			isValid:       false,
		},
		// Invalid filter, both name and pattern supplied
		{
			name:        "invalid filter (both name and pattern)",
			qAnalysisId: "8ff19e60-bab1-464f-9d03-de76bbd215aa",
			fileName:    "sample1.txt",
			pattern:     ".+.txt",
			isValid:     false,
		},
		// Invalid filter, invalid regex
		{
			name:        "invalid filter (invalid regex)",
			qAnalysisId: "8ff19e60-bab1-464f-9d03-de76bbd215aa",
			pattern:     "(.+)).txt",
			isValid:     false,
		},
		{
			name:        "url-encoded pattern",
			qAnalysisId: "8ff19e60-bab1-464f-9d03-de76bbd215aa",
			pattern:     ".%2B%5C.bin%24", // ".+\.bin$"
			isValid:     true,
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			if c.analysisId != "" {
				ctx.Params = append(ctx.Params, gin.Param{Key: "analysisId", Value: c.analysisId})
			}
			if c.analysisId != "" {
				ctx.Params = append(ctx.Params, gin.Param{Key: "runId", Value: c.runId})
			}
			buf := io.NopCloser(bytes.NewBuffer([]byte{}))
			ctx.Request = httptest.NewRequest("GET", fmt.Sprintf("/?parent_id=%s&level=%s&type=%s&name=%s&analysis_id=%s&run_id=%s&pattern=%s", c.parentId, c.analysisLevel, c.fileType, c.fileName, c.qAnalysisId, c.qRunId, c.pattern), buf)

			filter, validationErr := getAnalysisFileFilter(ctx)
			t.Logf("validation error: %v, should be valid: %t", validationErr, c.isValid)
			if (validationErr != nil) == c.isValid {
				if c.isValid {
					t.Fatalf("expected validation to succeed, got error: %v", validationErr)
				} else {
					t.Fatal("expected validation to fail, got no error")
				}
			}
			if validationErr != nil {
				return
			}
			if c.analysisId != "" && filter.AnalysisId.String() != c.analysisId {
				t.Error("analysis id in parameters not priorised over query parameter")
			} else if c.analysisId == "" && filter.AnalysisId.String() != c.qAnalysisId {
				t.Errorf("analysis id mismatch, expected %q, found %q", c.qAnalysisId, filter.AnalysisId)
			}
			if c.runId != "" && filter.RunId != c.runId {
				t.Error("run id in parameters not priorised over query parameter")
			} else if c.runId == "" && filter.RunId != c.qRunId {
				t.Errorf("run id mismatch, expected %q, found %q", c.qRunId, filter.RunId)
			}
			if filter.FileType.String() != c.fileType {
				t.Errorf("file type mismatch, expected %q found %q", c.fileType, filter.FileType)
			}
			if filter.Name != c.fileName {
				t.Errorf("name mismatch, expected %q found %q", c.fileName, filter.Name)
			}
			if filter.Level.String() != c.analysisLevel {
				t.Errorf("level mismatch, expected %q found %q", c.analysisLevel, filter.Level.String())
			}
			if filter.Pattern == nil && c.pattern != "" {
				t.Error("pattern is nil, but shouldn't be")
			}
			if filter.Pattern != nil {
				t.Log(filter.Pattern)
			}
		})
	}
}
