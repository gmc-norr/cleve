package interop

import (
	"testing"
	"time"
)

func TestReadRunInfo(t *testing.T) {
	testcases := []struct {
		name       string
		path       string
		version    int
		date       time.Time
		instrument string
		reads      []read
	}{
		{
			name:    "miseq",
			path:    "./test/250207_M00568_0665_000000000-LMWPP/RunInfo.xml",
			version: 2,
			// 250207
			date:       time.Date(2025, 2, 7, 0, 0, 0, 0, time.UTC),
			instrument: "M00568",
			reads: []read{
				{},
				{},
				{},
			},
		},
		{
			name:    "miseq old",
			path:    "./test/160122_M00568_0146_000000000-ALYCY/RunInfo.xml",
			version: 2,
			// 160122
			date:       time.Date(2016, 1, 22, 0, 0, 0, 0, time.UTC),
			instrument: "M00568",
			reads: []read{
				{},
				{},
				{},
			},
		},
		{
			name:    "nextseq",
			path:    "./test/250210_NB551119_0457_AHL3Y2AFX7/RunInfo.xml",
			version: 4,
			// 250210
			date:       time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC),
			instrument: "NB551119",
			reads: []read{
				{},
				{},
				{},
				{},
			},
		},
		{
			name:    "novaseq",
			path:    "./test/20250123_LH00352_0033_A225H35LT1/RunInfo.xml",
			version: 6,
			// 2025-01-23T19:07:33Z
			date:       time.Date(2025, 1, 23, 19, 7, 33, 0, time.UTC),
			instrument: "LH00352",
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

			if ri.Instrument != c.instrument {
				t.Errorf("expected instrument %q, found %q", c.instrument, ri.Instrument)
			}

			if len(ri.Reads) != len(c.reads) {
				t.Errorf("expected %d reads, found %d", len(c.reads), len(ri.Reads))
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
