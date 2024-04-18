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

		t.Logf("%s bins: %v", k, metrics.Bins)

		// if v.Incomplete {
		// 	t.Logf("%v", metrics)
		// }

		// t.Logf("%s %% >= Q30: %.2f", k, metrics.PercentOverQ30())
		// t.Logf("%s %% >= Q30 per lane: %v", k, metrics.PercentOverQ30PerLane())
	}
}
