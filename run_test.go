package cleve

import (
	"bytes"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
)

func generateBson(data any) ([]byte, error) {
	buf := make([]byte, 0)
	b := bytes.NewBuffer(buf)
	rw, err := bsonrw.NewBSONValueWriter(b)
	if err != nil {
		return buf, err
	}
	encoder, err := bson.NewEncoder(rw)
	if err != nil {
		return nil, err
	}
	if err := encoder.Encode(data); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func TestUnmarshalV1(t *testing.T) {
	testcases := []struct {
		name           string
		runId          string
		runNumber      int
		experimentName string
		side           string
		platform       string
		flowcell       string
		reagentSerial  string
		bufferSerial   string
		doc            bson.M
		schemaVersion  int
		shouldError    bool
	}{
		{
			name:           "novaseq version 1",
			runId:          "run1",
			runNumber:      3,
			experimentName: "run1",
			side:           "A",
			platform:       "NovaSeq X Plus",
			flowcell:       "1.5B",
			doc: bson.M{
				"schema_version":  1,
				"run_id":          "run1",
				"path":            "/path/to/run1",
				"experiment_name": "run1",
				"platform":        "NovaSeq X Plus",
				"run_info": bson.M{
					"run": bson.M{
						"number":     3,
						"instrument": "LH00000",
						"flowcell":   "225H35LT1",
					},
				},
				"run_parameters": bson.M{
					"side":           "A",
					"experimentname": "run1",
				},
			},
		},
		{
			name:           "novaseq missing version",
			runId:          "run1",
			runNumber:      5,
			experimentName: "run1",
			side:           "B",
			platform:       "NovaSeq X Plus",
			flowcell:       "1.5B",
			doc: bson.M{
				"run_id":          "run1",
				"path":            "/path/to/run1",
				"experiment_name": "run1",
				"platform":        "NovaSeq X Plus",
				"run_info": bson.M{
					"run": bson.M{
						"number":     5,
						"instrument": "LH00000",
						"flowcell":   "225H35LT1",
					},
				},
				"run_parameters": bson.M{
					"side":           "B",
					"experimentname": "run1",
				},
			},
		},
		{
			name:           "nextseq missing version",
			runId:          "run1",
			runNumber:      5,
			experimentName: "run1",
			platform:       "NextSeq 5x0",
			flowcell:       "Mid",
			bufferSerial:   "pr2serial",
			reagentSerial:  "reagentserial",
			doc: bson.M{
				"run_id":          "run1",
				"path":            "/path/to/run1",
				"experiment_name": "run1",
				"platform":        "NextSeq",
				"run_info": bson.M{
					"run": bson.M{
						"number":     5,
						"instrument": "NB000000",
						"flowcell":   "HL3Y2AFX7",
					},
				},
				"run_parameters": bson.M{
					"experimentname": "run1",
					"reagentkitrfidtag": bson.M{
						"serialnumber": "reagentserial",
					},
					"pr2bottlerfidtag": bson.M{
						"serialnumber": "pr2serial",
					},
				},
			},
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			var run Run
			doc, err := generateBson(c.doc)
			if err != nil {
				t.Fatal(err)
			}
			err = bson.Unmarshal(doc, &run)
			if err != nil {
				t.Fatal(err)
			}

			if run.SchemaVersion != 1 {
				t.Errorf("expected schema version 1, got %d", run.SchemaVersion)
			}
			if run.Platform != c.platform {
				t.Errorf("expected platform %q, got %q", c.platform, run.Platform)
			}
			if run.RunID != c.runId {
				t.Errorf("expected run id %q, got %q", c.runId, run.RunID)
			}
			if run.RunInfo.FlowcellName != c.flowcell {
				t.Errorf("expected flowcell name %q, got %q", c.flowcell, run.RunInfo.FlowcellName)
			}
			if run.RunInfo.RunNumber != c.runNumber {
				t.Errorf("expected run number %d, got %d", c.runNumber, run.RunInfo.RunNumber)
			}
			if run.RunParameters.ExperimentName != c.experimentName {
				t.Errorf("expected experiment name %q, got %q", c.experimentName, run.RunParameters.ExperimentName)
			}
			if run.RunParameters.Side != c.side {
				t.Errorf("expected side %q, got %q", c.side, run.RunParameters.Side)
			}
			for _, consumable := range run.RunParameters.Consumables {
				switch consumable.Type {
				case "Buffer":
					if consumable.SerialNumber != c.bufferSerial {
						t.Errorf("expected buffer serial %q, got %q", c.bufferSerial, consumable.SerialNumber)
					}
				case "Reagent":
					if consumable.SerialNumber != c.reagentSerial {
						t.Errorf("expected reagent kit serial %q, got %q", c.reagentSerial, consumable.SerialNumber)
					}
				}
			}
		})
	}
}

func TestUnmarshalV2(t *testing.T) {
	testcases := []struct {
		name          string
		runId         string
		platform      string
		flowcell      string
		doc           bson.M
		schemaVersion int
		shouldError   bool
	}{
		{
			name:     "NovaSeq run",
			runId:    "run1",
			platform: "NovaSeq X Plus",
			flowcell: "1.5B",
			doc: bson.M{
				"schema_version":  2,
				"run_id":          "run1",
				"path":            "/path/to/run1",
				"experiment_name": "run1",
				"platform":        "NovaSeq X Plus",
				"run_info": bson.M{
					"instrument":    "LH00000",
					"flowcell_id":   "225H35LT1",
					"flowcell_name": "1.5B",
				},
				"run_parameters": bson.M{
					"experiment_name": "run1",
				},
			},
		},
		{
			name:     "NextSeq run",
			runId:    "run1",
			platform: "NextSeq 5x0",
			flowcell: "Mid",
			doc: bson.M{
				"schema_version":  2,
				"run_id":          "run1",
				"path":            "/path/to/run1",
				"experiment_name": "run1",
				"platform":        "NextSeq 5x0",
				"run_info": bson.M{
					"instrument_id": "NB000000",
					"flowcell_id":   "HL3Y2AFX7",
					"flowcell_name": "Mid",
				},
				"run_parameters": bson.M{
					"experiment_name": "run1",
				},
			},
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			var run Run
			doc, err := generateBson(c.doc)
			if err != nil {
				t.Fatal(err)
			}
			err = bson.Unmarshal(doc, &run)
			if err != nil {
				t.Fatal(err)
			}

			if run.Platform != c.platform {
				t.Errorf("expected platform %q, got %q", c.platform, run.Platform)
			}
			if run.RunID != c.runId {
				t.Errorf("expected run id %q, got %q", c.runId, run.RunID)
			}
			if run.RunInfo.FlowcellName != c.flowcell {
				t.Errorf("expected flowcell name %q, got %q", c.flowcell, run.RunInfo.FlowcellName)
			}
		})
	}
}
