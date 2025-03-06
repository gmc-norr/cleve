package interop

import (
	"testing"
)

func TestReadQMetrics(t *testing.T) {
	testcases := []struct {
		name string
		path string
	}{
		{
			name: "novaseq",
			path: "./test/20250123_LH00352_0033_A225H35LT1/InterOp/QMetricsOut.bin",
		},
		{
			name: "nextseq",
			path: "./test/250210_NB551119_0457_AHL3Y2AFX7/InterOp/QMetricsOut.bin",
		},
		{
			name: "miseq",
			path: "./test/250207_M00568_0665_000000000-LMWPP/InterOp/QMetricsOut.bin",
		},
		{
			name: "miseq old",
			path: "./test/160122_M00568_0146_000000000-ALYCY/InterOp/QMetricsOut.bin",
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ReadQMetrics(c.path)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
