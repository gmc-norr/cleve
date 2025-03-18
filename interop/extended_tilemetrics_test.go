package interop

import (
	"errors"
	"math"
	"os"
	"testing"
)

func TestExtendedReadTileMetrics(t *testing.T) {
	testcases := []struct {
		name     string
		path     string
		version  int
		nTiles   int
		expected map[int]map[int]float64
	}{
		{
			name:    "novaseq",
			path:    "./testdata/20250123_LH00352_0033_A225H35LT1/InterOp/ExtendedTileMetricsOut.bin",
			version: 3,
			nTiles:  560,
			expected: map[int]map[int]float64{
				1: {
					1101: 4890.899902,
				},
				2: {
					1112: 4883.899902,
				},
			},
		},
		{
			name:    "nextseq",
			path:    "./testdata/250210_NB551119_0457_AHL3Y2AFX7/InterOp/ExtendedTileMetricsOut.bin",
			version: 1,
			nTiles:  288,
			expected: map[int]map[int]float64{
				1: {
					11101: 512.7999878,
				},
				2: {
					11112: 500.2999878,
				},
			},
		},
		// MiSeq does not output these metrics
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := os.Stat(c.path); errors.Is(err, os.ErrNotExist) {
				t.Skip("test data not found, skipping")
			}
			tm, err := ReadExtendedTileMetrics(c.path)
			if err != nil {
				t.Errorf("failed to read extended tile metrics: %s", err)
			}

			if tm.Version != uint8(c.version) {
				t.Errorf("expected version %d, got %d", c.version, tm.Version)
			}

			if len(tm.Records) != c.nTiles {
				t.Errorf("expected results for %d tiles, got %d", c.nTiles, len(tm.Records))
			}

			for lane := range c.expected {
				for tile, count := range c.expected[lane] {
					foundIt := false
					for _, r := range tm.Records {
						if r.Lane == lane && r.Tile == tile {
							foundIt = true
							// Damn rounding errors! What's the point of storing this as a float that in the ASCII
							// representation STILL takes up more space than the corresponding int!?
							if math.Abs(float64(r.OccupiedClusters)/1000-count) > 0.1 {
								t.Errorf("expected %.3fK occupied clusters, got %d", count, r.OccupiedClusters)
							}
						}
					}
					if !foundIt {
						t.Errorf("couldn't find an entry for lane %d tile %d", lane, tile)
					}
				}
			}
		})
	}
}
