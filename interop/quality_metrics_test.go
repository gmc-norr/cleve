package interop

import (
	"testing"
)

func TestParseQMetrics(t *testing.T) {
	table := map[string]struct {
		Filename   string
		Incomplete bool
		Version    uint8
		RecordSize uint8
		BinCount   uint8
	}{
		"novaseq": {
			"../test_data/novaseq/QMetricsOut.bin",
			true,
			7,
			8 + 4*3,
			3,
		},
		"nextseq_v6": {
			"../test_data/nextseq1/QMetricsOut.bin",
			true,
			6,
			6 + 4*7,
			7,
		},
		"nextseq_v5": {
			"../test_data/nextseq2/QMetricsOut.bin",
			true,
			5,
			206,
			7,
		},
		// "novaseq_full": {
		// 	"../test_data/novaseq/QMetricsOut_full.bin",
		// 	false,
		// 	7,
		// 	8 + 4*3,
		// 	3,
		// },
		// "nextseq_v6_full": {
		// 	"../test_data/nextseq1/QMetricsOut_full.bin",
		// 	false,
		// 	6,
		// 	6 + 4*7,
		// 	7,
		// },
		// "nextseq_v5_full": {
		// 	"../test_data/nextseq2/QMetricsOut_full.bin",
		// 	false,
		// 	5,
		// 	206,
		// 	7,
		// },
	}

	for k, v := range table {
		metrics, err := ParseQMetrics(v.Filename)
		if err != nil {
			t.Fatalf("failed to parse qmetrics for %s: %s", k, err.Error())
		}

		if metrics.Version != v.Version {
			t.Fatalf("expected version %d, got %d", v.Version, metrics.Version)
		}

		if !metrics.HasBins {
			t.Fatalf("%s does not have bins", k)
		}

		if metrics.RecordSize != v.RecordSize {
			t.Fatalf("expected record size %d, got %d", v.RecordSize, metrics.RecordSize)
		}

		if metrics.BinCount != v.BinCount {
			t.Fatalf("expected bin count %d, got %d", v.BinCount, metrics.BinCount)
		}

		if len(metrics.Records[0].Histogram) != int(v.BinCount) {
			t.Fatalf("expected histogram length %d, got %d", v.BinCount, len(metrics.Records[0].Histogram))
		}

		// t.Logf("%s bins: %v", k, metrics.Bins)

		// if v.Incomplete {
		// 	t.Logf("%v", metrics)
		// }

		// t.Logf("%s %% >= Q30: %.2f", k, metrics.PercentOverQ30())
		// t.Logf("%s %% >= Q30 per lane: %v", k, metrics.PercentOverQ30PerLane())
	}
}

func TestMaxCyclesPerLane(t *testing.T) {
	var tile = Tile32(1)
	m := QMetrics{
		InteropHeader{Version: 7, RecordSize: 20},
		true,
		QBinConfig{
			3,
			QBins{
				QBin{0, 18, 12},
				QBin{28, 30, 27},
				QBin{30, 50, 40},
			},
		},
		QRecords{
			QRecord{Lane: 1, Tile: &tile, Cycle: 1, Histogram: []uint32{0, 150, 300}},
			QRecord{Lane: 1, Tile: &tile, Cycle: 2, Histogram: []uint32{0, 150, 300}},
			QRecord{Lane: 1, Tile: &tile, Cycle: 3, Histogram: []uint32{0, 150, 300}},
			QRecord{Lane: 1, Tile: &tile, Cycle: 4, Histogram: []uint32{0, 150, 300}},
			QRecord{Lane: 1, Tile: &tile, Cycle: 5, Histogram: []uint32{0, 150, 300}},
			QRecord{Lane: 2, Tile: &tile, Cycle: 1, Histogram: []uint32{0, 150, 300}},
			QRecord{Lane: 2, Tile: &tile, Cycle: 2, Histogram: []uint32{0, 150, 300}},
			QRecord{Lane: 2, Tile: &tile, Cycle: 3, Histogram: []uint32{0, 150, 300}},
			QRecord{Lane: 2, Tile: &tile, Cycle: 4, Histogram: []uint32{0, 150, 300}},
			QRecord{Lane: 2, Tile: &tile, Cycle: 5, Histogram: []uint32{0, 150, 300}},
			QRecord{Lane: 2, Tile: &tile, Cycle: 6, Histogram: []uint32{0, 150, 300}},
			QRecord{Lane: 2, Tile: &tile, Cycle: 7, Histogram: []uint32{0, 150, 300}},
			QRecord{Lane: 2, Tile: &tile, Cycle: 8, Histogram: []uint32{0, 150, 300}},
		},
	}

	cycles := m.MaxCyclePerLane()

	// t.Log(cycles)

	if cycles[1] != 5 {
		t.Fatalf("expected 5 cycles for lane 1, got %d", cycles[1])
	}
	if cycles[2] != 8 {
		t.Fatalf("expected 8 cycles for lane 1, got %d", cycles[1])
	}
}

func TestPercentOverQ(t *testing.T) {
	var tile = Tile32(1)
	m := QMetrics{
		InteropHeader{Version: 7, RecordSize: 20},
		true,
		QBinConfig{
			3,
			QBins{
				QBin{0, 18, 12},
				QBin{18, 30, 27},
				QBin{30, 50, 40},
			},
		},
		QRecords{
			QRecord{Lane: 1, Tile: &tile, Cycle: 1, Histogram: []uint32{0, 150, 300}},
			QRecord{Lane: 1, Tile: &tile, Cycle: 2, Histogram: []uint32{0, 100, 200}},
			QRecord{Lane: 1, Tile: &tile, Cycle: 3, Histogram: []uint32{0, 150, 300}},
			QRecord{Lane: 1, Tile: &tile, Cycle: 4, Histogram: []uint32{0, 150, 300}},
			QRecord{Lane: 1, Tile: &tile, Cycle: 5, Histogram: []uint32{0, 200, 500}},
			QRecord{Lane: 2, Tile: &tile, Cycle: 1, Histogram: []uint32{0, 150, 300}},
			QRecord{Lane: 2, Tile: &tile, Cycle: 2, Histogram: []uint32{0, 200, 300}},
			QRecord{Lane: 2, Tile: &tile, Cycle: 3, Histogram: []uint32{0, 150, 300}},
			QRecord{Lane: 2, Tile: &tile, Cycle: 4, Histogram: []uint32{0, 150, 300}},
			QRecord{Lane: 2, Tile: &tile, Cycle: 5, Histogram: []uint32{0, 150, 300}},
			QRecord{Lane: 2, Tile: &tile, Cycle: 6, Histogram: []uint32{10, 200, 200}},
			QRecord{Lane: 2, Tile: &tile, Cycle: 7, Histogram: []uint32{0, 150, 300}},
			QRecord{Lane: 2, Tile: &tile, Cycle: 8, Histogram: []uint32{0, 150, 300}},
		},
	}

	table := []struct {
		Threshold uint8
		Expected  float32
	}{
		{
			Threshold: 30,
			// 3900 is the sum of the last bin
			// 800 is the last cycle in the last bin
			// 10 is the sum of the first bin
			// 2050 is the sum of the second bin
			// 1150 is the sum of the last cycle in each lane for all bins
			Expected: 100. * ((3900. - 800.) / (10. + 2050. + 3900. - 1150.)),
		},
		{
			Threshold: 40,
			Expected:  0,
		},
		{
			Threshold: 25,
			Expected:  100. * ((3900. - 800.) / (10. + 2050. + 3900. - 1150.)),
		},
		{
			Threshold: 15,
			// Add the sum of the second bin and subtract the last cycle within the second bin
			Expected: 100. * ((2050 + 3900. - 800. - 350.) / (10. + 2050. + 3900. - 1150.)),
		},
	}

	for _, v := range table {
		p := m.TotalPercentOverQ(v.Threshold)
		if p != v.Expected {
			t.Fatalf("expected %f, got %f for threshold %d", v.Expected, p, v.Threshold)
		}
	}
}
