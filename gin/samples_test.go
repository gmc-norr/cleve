package gin

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mock"
	"github.com/gmc-norr/cleve/mongo"
)

func TestSample(t *testing.T) {
	t.Run("non-existent sample", func(t *testing.T) {
		sg := mock.SampleGetter{}
		sg.SampleFn = func(sampleId string) (*cleve.Sample, error) {
			return nil, mongo.ErrNoDocuments
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{gin.Param{Key: "sampleId", Value: "1234"}}

		SampleHandler(&sg)(c)

		if !sg.SampleInvoked {
			t.Errorf("Sample was not invoked")
		}

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("simple sample", func(t *testing.T) {
		sg := mock.SampleGetter{}
		sg.SampleFn = func(sampleId string) (*cleve.Sample, error) {
			return &cleve.Sample{
				Id:       "1234",
				Name:     "1234",
				Fastq:    []string{"/path/to/fastq/1234_1.fq.gz", "/path/to/fastq/1234_2.fq.gz"},
				Analyses: make([]*cleve.SampleAnalysis, 0),
			}, nil
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		SampleHandler(&sg)(c)

		if !sg.SampleInvoked {
			t.Errorf("Sample was not invoked")
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		body, err := io.ReadAll(w.Body)
		if err != nil {
			t.Errorf(err.Error())
		}

		var sample cleve.Sample
		if err := json.Unmarshal(body, &sample); err != nil {
			t.Errorf("error unmarshaling sample: %s", err.Error())
		}

		if sample.Name != "1234" {
			t.Errorf("expected name 1234, got %s", sample.Name)
		}

		if sample.Id != "1234" {
			t.Errorf("expected id 1234, got %s", sample.Id)
		}

		if len(sample.Fastq) != 2 {
			t.Errorf("expected 2 fastqs, got %d", len(sample.Fastq))
		}

		if len(sample.Analyses) != 0 {
			t.Errorf("expected 0 analyses, got %d", len(sample.Analyses))
		}
	})
}

func TestSamples(t *testing.T) {
	t.Run("no samples", func(t *testing.T) {
		sg := mock.SampleGetter{}
		sg.SamplesFn = func() ([]*cleve.Sample, error) {
			return make([]*cleve.Sample, 0), mongo.ErrNoDocuments
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		SamplesHandler(&sg)(c)

		if !sg.SamplesInvoked {
			t.Errorf("Samples was not invoked")
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected %d, got %d", http.StatusOK, w.Code)
		}
	})
}

func TestCreateSample(t *testing.T) {
	t.Run("add sample", func(t *testing.T) {
		sampleCollection := make([]*cleve.Sample, 0)
		ss := mock.SampleSetter{}
		ss.CreateSampleFn = func(sample *cleve.Sample) error {
			sampleCollection = append(sampleCollection, sample)
			return nil
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		b := bytes.NewBufferString(`{"name": "1234", "id": "D24-1234"}`)
		c.Request, _ = http.NewRequest("POST", "/samples", b)

		AddSampleHandler(&ss)(c)

		if !ss.CreateSampleInvoked {
			t.Errorf("CreateSample was not invoked")
		}

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}

		if len(sampleCollection) != 1 {
			t.Errorf("expected 1 sample, got %d", len(sampleCollection))
		}

		if sampleCollection[0].Name != "1234" {
			t.Errorf("expected name 1234, got %s", sampleCollection[0].Name)
		}

		if sampleCollection[0].Id != "D24-1234" {
			t.Errorf("expected id D24-1234, got %s", sampleCollection[0].Id)
		}

		if len(sampleCollection[0].Fastq) != 0 {
			t.Errorf("expected 0 fastqs, got %d", len(sampleCollection[0].Fastq))
		}

		if len(sampleCollection[0].Analyses) != 0 {
			t.Errorf("expected 0 analyses, got %d", len(sampleCollection[0].Analyses))
		}

		body, err := io.ReadAll(w.Body)
		if err != nil {
			t.Error(err.Error())
		}

		if string(body) != `{"message":"sample added","sample_id":"D24-1234"}` {
			t.Errorf("unexpected message, got %s", string(body))
		}
	})

	t.Run("missing name", func(t *testing.T) {
		ss := mock.SampleSetter{}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		b := bytes.NewBufferString(`{"name": "1234"}`)
		c.Request, _ = http.NewRequest("/samples", "POST", b)

		AddSampleHandler(&ss)(c)

		if ss.CreateSampleInvoked {
			t.Errorf("CreateSample was invoked")
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}

		body, err := io.ReadAll(w.Body)
		if err != nil {
			t.Error(err.Error())
		}

		if string(body) != `{"error":"invalid request"}` {
			t.Errorf("unexpected message, got %s", string(body))
		}
	})
}
