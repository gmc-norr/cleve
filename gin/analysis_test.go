package gin

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

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
			data:   []byte(`{"analysis_id": "analysis1", "path": "/path/to/analysis1", "run_id": "run1", "state": "ready", "software": "software1", "software_version": "1.0.0", "output_files": [{"path": "fastq/sample1_1.fastq.gz", "type": "fastq"}, {"path": "fastq/sample1_2.fastq.gz", "type": "fastq"}]}`),
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
				t.Errorf("expected status code %d, got %d", c.code, w.Code)
			}
		})
	}
}
