package interop

import (
	"math"
	"testing"
)

func TestInteropYield(t *testing.T) {
	testcases := []struct {
		name       string
		path       string
		totalYield float64         // yield in GB
		laneYield  map[int]float64 // yield in GB
	}{
		{
			name:       "novaseq",
			path:       "./test/20250123_LH00352_0033_A225H35LT1",
			totalYield: 675.83,
			laneYield: map[int]float64{
				1: 337.09,
				2: 338.74,
			},
		},
		{
			name:       "nextseq",
			path:       "./test/250210_NB551119_0457_AHL3Y2AFX7",
			totalYield: 42.30,
			laneYield: map[int]float64{
				1: 10.79,
				2: 10.51,
				3: 10.72,
				4: 10.28,
			},
		},
		{
			name:       "miseq",
			path:       "./test/250207_M00568_0665_000000000-LMWPP",
			totalYield: 10.97,
			laneYield: map[int]float64{
				1: 10.97,
			},
		},
		{
			name:       "miseq_old",
			path:       "./test/160122_M00568_0146_000000000-ALYCY",
			totalYield: 6.60,
			laneYield: map[int]float64{
				1: 6.60,
			},
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			i, err := InteropFromDir(c.path)
			if err != nil {
				t.Fatal(err)
			}

			yield := math.Round(float64(i.TotalYield())/1e7) / 100
			if yield != c.totalYield {
				t.Errorf("expected yield of %.2f GB, got %.2f GB", c.totalYield, yield)
			}

			laneYield := i.LaneYield()

			for lane, yield := range c.laneYield {
				obsYield, ok := laneYield[lane]
				if !ok {
					t.Errorf("lane %d not found for lane yield data", lane)
					continue
				}
				obsYieldGB := math.Round(float64(obsYield)/1e7) / 100
				if obsYieldGB != yield {
					t.Errorf("expected yield of lane %d to be %.2f GB, got %.2f GB", lane, yield, obsYieldGB)
				}
			}
		})
	}
}

// Check that the tile count from RunInfo.xml matches with the number of tiles that were
// parsed from the tile metrics.
func TestTileCount(t *testing.T) {
	testcases := []struct {
		name string
		path string
	}{
		{
			name: "novaseq",
			path: "./test/20250123_LH00352_0033_A225H35LT1",
		},
		{
			name: "nextseq",
			path: "./test/250210_NB551119_0457_AHL3Y2AFX7",
		},
		{
			name: "miseq",
			path: "./test/250207_M00568_0665_000000000-LMWPP",
		},
		{
			name: "miseq_old",
			path: "./test/160122_M00568_0146_000000000-ALYCY",
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			i, err := InteropFromDir(c.path)
			if err != nil {
				t.Fatal(err)
			}

			tileMap := make(map[string]bool)
			laneMap := make(map[int]bool)
			for _, tile := range i.TileMetrics.Records {
				if _, ok := tileMap[tile.TileName()]; !ok {
					tileMap[tile.TileName()] = true
				}
				if _, ok := laneMap[tile.Lane]; !ok {
					laneMap[tile.Lane] = true
				}
			}

			riCount := i.RunInfo.TileCount()
			tmCount := len(tileMap)

			if tmCount != riCount {
				t.Errorf("mismatch between tile count in run info and tile metrics: %d vs %d", riCount, tmCount)
			}
		})
	}
}

func TestOccupancy(t *testing.T) {
	testcases := []struct {
		name          string
		path          string
		expectedTotal float64
		expectedLane  map[int]float64
	}{
		{
			name:          "novaseq",
			path:          "./test/20250123_LH00352_0033_A225H35LT1",
			expectedTotal: 97.13,
			expectedLane: map[int]float64{
				1: 97.03,
				2: 97.22,
			},
		},
		{
			name:          "nextseq",
			path:          "./test/250210_NB551119_0457_AHL3Y2AFX7",
			expectedTotal: 99.73,
			expectedLane: map[int]float64{
				1: 99.70,
				2: 99.77,
				3: 99.78,
				4: 99.68,
			},
		},
		{
			name: "miseq",
			path: "./test/250207_M00568_0665_000000000-LMWPP",
		},
		{
			name: "miseq_old",
			path: "./test/160122_M00568_0146_000000000-ALYCY",
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			i, err := InteropFromDir(c.path)
			if err != nil {
				t.Fatal(err)
			}

			obsPercOccupied := math.Round(1e4*i.TotalFracOccupied()) / 100

			if obsPercOccupied != c.expectedTotal {
				t.Errorf("expected %.2f%% occupied clusters, got %.2f%%", c.expectedTotal, obsPercOccupied)
			}

			laneFracOccupied := i.LaneFracOccupied()
			for lane, expPercOccupied := range c.expectedLane {
				obsPercOccupied = math.Round(1e4*laneFracOccupied[lane]) / 100
				if obsPercOccupied != expPercOccupied {
					t.Errorf("expected %.2f%% occupied for lane %d, got %.2f%%", expPercOccupied, lane, obsPercOccupied)
				}
			}
		})
	}
}

func TestLaneErrorRate(t *testing.T) {
	testcases := []struct {
		name        string
		path        string
		shouldError bool
		version     int
		laneErrors  map[int]map[int]float64
	}{
		{
			name: "novaseq missing",
			path: "./test/20250123_LH00352_0033_A225H35LT1",
			laneErrors: map[int]map[int]float64{
				1: {
					1: 0,
					2: 0,
				},
				2: {
					1: 0,
					2: 0,
				},
				3: {
					1: 0,
					2: 0,
				},
				4: {
					1: 0,
					2: 0,
				},
			},
		},
		{
			name: "novaseq",
			path: "./test/20250115_LH00352_0031_A225HMVLT1",
			laneErrors: map[int]map[int]float64{
				1: {
					1: 0.16,
					2: 0.15,
				},
				2: {
					1: 0,
					2: 0,
				},
				3: {
					1: 0,
					2: 0,
				},
				4: {
					1: 0.19,
					2: 0.17,
				},
			},
		},
		{
			name: "nextseq",
			path: "./test/250210_NB551119_0457_AHL3Y2AFX7",
			laneErrors: map[int]map[int]float64{
				1: {
					1: 0.62,
					2: 0.61,
					3: 0.60,
					4: 0.63,
				},
				2: {
					1: math.NaN(),
					2: math.NaN(),
					3: math.NaN(),
					4: math.NaN(),
				},
				3: {
					1: math.NaN(),
					2: math.NaN(),
					3: math.NaN(),
					4: math.NaN(),
				},
				4: {
					1: 0.68,
					2: 0.66,
					3: 0.68,
					4: 0.69,
				},
			},
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			i, err := InteropFromDir(c.path)
			if err != nil {
				t.Fatal(err)
			}

			for read, laneErrors := range i.LaneErrorRate() {
				for lane, obsError := range laneErrors {
					if math.IsNaN(c.laneErrors[read][lane]) && !math.IsNaN(obsError) {
						t.Errorf("expected error to be missing for read %d in lane %d, got %.2f%%", read, lane, obsError)
					} else if obsError-c.laneErrors[read][lane] > 0.005 {
						t.Errorf("expected error rate of %.2f%% for read %d in lane %d, got %.2f%%", c.laneErrors[read][lane], read, lane, obsError)
					}
				}
			}
		})
	}
}
