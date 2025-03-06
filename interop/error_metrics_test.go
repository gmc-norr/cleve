package interop

import (
	"errors"
	"os"
	"testing"
)

func TestReadErrorMetrics(t *testing.T) {
	testcases := []struct {
		name        string
		path        string
		shouldError bool
		version     int
		totalError  float64
		laneErrors  map[int]float64
		readErrors  map[int]map[int]float64
	}{
		{
			name:        "novaseq missing",
			path:        "./test/20250123_LH00352_0033_A225H35LT1/InterOp/ErrorMetricsOut.bin",
			shouldError: true,
		},
		{
			name:    "novaseq",
			path:    "./test/20250115_LH00352_0031_A225HMVLT1/InterOp/ErrorMetricsOut.bin",
			version: 6,
		},
		{
			name:    "nextseq",
			path:    "./test/250210_NB551119_0457_AHL3Y2AFX7/InterOp/ErrorMetricsOut.bin",
			version: 3,
		},
		{
			name:    "miseq",
			path:    "./test/250207_M00568_0665_000000000-LMWPP/InterOp/ErrorMetricsOut.bin",
			version: 3,
		},
		{
			name:    "miseq old",
			path:    "./test/160122_M00568_0146_000000000-ALYCY/InterOp/ErrorMetricsOut.bin",
			version: 3,
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := os.Stat(c.path); errors.Is(err, os.ErrNotExist) {
				t.Skip("test data not found, skipping")
			}
			em, err := ReadErrorMetrics(c.path)
			if err != nil && !c.shouldError {
				t.Fatalf("failed to read error metrics: %s", err)
			} else if err == nil && c.shouldError {
				t.Fatal("expected error when reading error metrics, got nil")
			} else if err != nil && c.shouldError {
				return
			}

			if em.Version != uint8(c.version) {
				t.Errorf("expected version %d, got %d", c.version, em.Version)
			}
		})
	}
}
