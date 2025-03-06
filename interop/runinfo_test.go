package interop

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestReadRunInfo(t *testing.T) {
	testcases := []struct {
		name         string
		path         string
		version      int
		runId        string
		date         time.Time
		instrumentId string
		platform     string
		flowcell     string
		reads        []read
	}{
		{
			name:    "miseq",
			path:    "./testdata/250207_M00568_0665_000000000-LMWPP/RunInfo.xml",
			version: 2,
			runId:   "250207_M00568_0665_000000000-LMWPP",
			// 250207
			date:         time.Date(2025, 2, 7, 0, 0, 0, 0, time.UTC),
			instrumentId: "M00568",
			platform:     "MiSeq",
			flowcell:     "Standard",
			reads: []read{
				{},
				{},
				{},
			},
		},
		{
			name:    "miseq old",
			path:    "./testdata/160122_M00568_0146_000000000-ALYCY/RunInfo.xml",
			version: 2,
			runId:   "160122_M00568_0146_000000000-ALYCY",
			// 160122
			date:         time.Date(2016, 1, 22, 0, 0, 0, 0, time.UTC),
			instrumentId: "M00568",
			platform:     "MiSeq",
			flowcell:     "Standard",
			reads: []read{
				{},
				{},
				{},
			},
		},
		{
			name:    "nextseq",
			path:    "./testdata/250210_NB551119_0457_AHL3Y2AFX7/RunInfo.xml",
			version: 4,
			runId:   "250210_NB551119_0457_AHL3Y2AFX7",
			// 250210
			date:         time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC),
			instrumentId: "NB551119",
			platform:     "NextSeq 5x0",
			flowcell:     "Mid",
			reads: []read{
				{},
				{},
				{},
				{},
			},
		},
		{
			name:    "novaseq",
			path:    "./testdata/20250123_LH00352_0033_A225H35LT1/RunInfo.xml",
			version: 6,
			runId:   "20250123_LH00352_0033_A225H35LT1",
			// 2025-01-23T19:07:33Z
			date:         time.Date(2025, 1, 23, 19, 7, 33, 0, time.UTC),
			instrumentId: "LH00352",
			platform:     "NovaSeq X Plus",
			flowcell:     "1.5B",
			reads: []read{
				{},
				{},
				{},
				{},
			},
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := os.Stat(c.path); errors.Is(err, os.ErrNotExist) {
				t.Skip("test data not found, skipping")
			}
			ri, err := ReadRunInfo(c.path)
			if err != nil {
				t.Fatal(err)
			}

			if ri.Version != c.version {
				t.Errorf("expected version %d, found %d", c.version, ri.Version)
			}

			if !ri.Date.Equal(c.date) {
				t.Errorf("expected date %s, found %s", c.date, ri.Date)
			}

			if ri.InstrumentId != c.instrumentId {
				t.Errorf("expected instrument ID %q, found %q", c.instrumentId, ri.InstrumentId)
			}

			if ri.Platform != c.platform {
				t.Errorf("expected platform %q, found %q", c.platform, ri.Platform)
			}

			if ri.FlowcellName != c.flowcell {
				t.Errorf("expected flowcell %q, found %q", c.flowcell, ri.FlowcellName)
			}

			if len(ri.Reads) != len(c.reads) {
				t.Errorf("expected %d reads, found %d", len(c.reads), len(ri.Reads))
			}

			if ri.RunId != c.runId {
				t.Errorf("expected run ID %q, got %q", c.runId, ri.RunId)
			}
			//
			// if ri.Flowcell.Lanes != 2 {
			// 	t.Errorf("expected 2 lanes, found %d", ri.Flowcell.Lanes)
			// }
			// if ri.Flowcell.Tiles != 70 {
			// 	t.Errorf("expected 70 tiles, found %d", ri.Flowcell.Lanes)
			// }
			// if ri.Flowcell.Swaths != 2 {
			// 	t.Errorf("expected 2 swaths, found %d", ri.Flowcell.Lanes)
			// }
		})
	}
}
