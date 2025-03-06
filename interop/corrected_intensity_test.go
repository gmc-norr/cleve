package interop

import (
	"errors"
	"os"
	"testing"
)

func TestReadCorrectedIntensity(t *testing.T) {
	testcases := []struct {
		name string
		path string
	}{
		{
			name: "novaseq",
			path: "./testdata/20250123_LH00352_0033_A225H35LT1/InterOp/CorrectedIntMetricsOut.bin",
		},
		{
			name: "nextseq",
			path: "./testdata/250210_NB551119_0457_AHL3Y2AFX7/InterOp/CorrectedIntMetricsOut.bin",
		},
		{
			name: "miseq",
			path: "./testdata/250207_M00568_0665_000000000-LMWPP/InterOp/CorrectedIntMetricsOut.bin",
		},
		{
			name: "miseq old",
			path: "./testdata/160122_M00568_0146_000000000-ALYCY/InterOp/CorrectedIntMetricsOut.bin",
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := os.Stat(c.path); errors.Is(err, os.ErrNotExist) {
				t.Skip("test data not found, skipping")
			}
			_, err := ReadCorrectedIntensity(c.path)
			if err != nil {
				t.Errorf("failed to parse corrected intensity metrics: %s", err)
			}
		})
	}
}
