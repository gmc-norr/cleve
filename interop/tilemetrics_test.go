package interop

import (
	"errors"
	"math"
	"os"
	"testing"
)

func TestReadTileMetrics(t *testing.T) {
	type lanestat struct {
		reads       float64 // millions
		readsPf     float64 // millions
		percPf      float64 // percent
		density     float64 // K/mm^2
		percAligned float64 // percent
	}
	testcases := []struct {
		name      string
		path      string
		version   int
		lanes     int
		percPf    float64
		density   float64
		laneStats map[int]lanestat
	}{
		{
			name:    "novaseq",
			path:    "./testdata/20250123_LH00352_0033_A225H35LT1/InterOp/TileMetricsOut.bin",
			version: 3,
			lanes:   2,
			percPf:  77.99,
			density: 7235.21,
			laneStats: map[int]lanestat{
				1: {
					reads:       1393.25,
					readsPf:     1083.77,
					percPf:      77.79,
					density:     7235.21,
					percAligned: 0.0,
				},
				2: {
					reads:       1393.25,
					readsPf:     1089.44,
					percPf:      78.19,
					density:     7235.21,
					percAligned: 0.0,
				},
			},
		},
		{
			name:    "novaseq 2",
			path:    "./testdata/20250115_LH00352_0031_A225HMVLT1/InterOp/TileMetricsOut.bin",
			version: 3,
			lanes:   2,
			percPf:  76.09,
			density: 7235.21,
			laneStats: map[int]lanestat{
				1: {
					reads:       1393.25,
					readsPf:     1067.27,
					percPf:      76.60,
					density:     7235.21,
					percAligned: 0.145,
				},
				2: {
					reads:       1393.25,
					readsPf:     1052.88,
					percPf:      75.57,
					density:     7235.21,
					percAligned: 0.14,
				},
			},
		},
		{
			name:    "nextseq",
			path:    "./testdata/250210_NB551119_0457_AHL3Y2AFX7/InterOp/TileMetricsOut.bin",
			version: 2,
			lanes:   4,
			percPf:  92.24,
			density: (172.13 + 167.66 + 171.33 + 163.95) / 4,
			laneStats: map[int]lanestat{
				1: {
					reads:       37.22,
					readsPf:     34.32,
					percPf:      92.21,
					density:     172.12,
					percAligned: 34.51,
				},
				2: {
					reads:       36.25,
					readsPf:     33.47,
					percPf:      92.30,
					density:     167.66,
					percAligned: 35.33,
				},
				3: {
					reads:       37.05,
					readsPf:     34.11,
					percPf:      92.07,
					density:     171.33,
					percAligned: 34.83,
				},
				4: {
					reads:       35.45,
					readsPf:     32.74,
					percPf:      92.36,
					density:     163.95,
					percAligned: 35.00,
				},
			},
		},
		{
			name:    "miseq",
			path:    "./testdata/250207_M00568_0665_000000000-LMWPP/InterOp/TileMetricsOut.bin",
			version: 2,
			lanes:   1,
			percPf:  79.31,
			laneStats: map[int]lanestat{
				1: {
					reads:       22.86,
					readsPf:     18.13,
					percPf:      79.19,
					density:     1031.06,
					percAligned: 22.41,
				},
			},
		},
		{
			name:    "miseq old",
			path:    "./testdata/160122_M00568_0146_000000000-ALYCY/InterOp/TileMetricsOut.bin",
			version: 2,
			lanes:   1,
			percPf:  79.73,
			laneStats: map[int]lanestat{
				1: {
					reads:       26.98,
					readsPf:     21.51,
					percPf:      79.64,
					density:     1210.09,
					percAligned: 0.98,
				},
			},
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := os.Stat(c.path); errors.Is(err, os.ErrNotExist) {
				t.Skip("test data not found, skipping")
			}
			tm, err := ReadTileMetrics(c.path)
			if err != nil {
				t.Errorf("failed to read tile metrics: %s", err)
			}

			if tm.Version != uint8(c.version) {
				t.Errorf("expected version %d, got %d", c.version, tm.Version)
			}

			if tm.LaneCount != c.lanes {
				t.Errorf("expected %d lanes, got %d", c.lanes, tm.LaneCount)
			}

			// Illumina takes the average of the tile percentages, but I would argue
			// that it would be better to calculate the percentage for the actual cluster
			// counts across all tiles for a lane. For testing purposes I will keep this here
			// though.
			lanePercPfSum := make(map[int]float64)
			laneTileCount := make(map[int]int)
			for _, r := range tm.Records {
				lanePercPfSum[r.Lane] += 100 * float64(r.PfClusterCount) / float64(r.ClusterCount)
				laneTileCount[r.Lane]++
			}
			for lane, sum := range lanePercPfSum {
				laneAverage := math.Round(100*sum/float64(laneTileCount[lane])) / 100
				if laneAverage != c.laneStats[lane].percPf {
					t.Errorf("expected %.2f%% passing filter reads for lane %d, got %.2f%%", c.laneStats[lane].percPf, lane, laneAverage)
				}
			}

			laneCount := tm.LaneClusters()
			laneCountPf := tm.LanePfClusters()
			for lane := range laneCount {
				mReads := math.Round(float64(laneCount[lane])/1e4) / 100
				mReadsPf := math.Round(float64(laneCountPf[lane])/1e4) / 100
				if mReads != c.laneStats[lane].reads {
					t.Errorf("expected %.2fM reads for lane %d, got %.2fM", c.laneStats[lane].reads, lane, mReads)
				}
				if mReadsPf != c.laneStats[lane].readsPf {
					t.Errorf("expected %.2fM passing filter reads for lane %d, got %.2fM", c.laneStats[lane].readsPf, lane, mReadsPf)
				}
			}

			lanePercentAligned := tm.LanePercentAligned()
			for lane := range c.laneStats {
				expAligned := c.laneStats[lane].percAligned
				obsAligned := lanePercentAligned[lane]
				if math.IsNaN(obsAligned) != math.IsNaN(expAligned) || math.Abs(obsAligned-expAligned) > 0.0099 {
					t.Errorf("expected %.2f%% aligned in lane %d, got %.2f%%", expAligned, lane, obsAligned)
				}
			}

			obsMReads := math.Round(float64(tm.Clusters())/1e4) / 100
			obsMReadsPf := math.Round(float64(tm.PfClusters())/1e4) / 100
			obsPercPf := math.Round(float64(tm.PfClusters())/float64(tm.Clusters())*1e4) / 100

			expMReads := 0.0
			expMReadsPf := 0.0
			for _, s := range c.laneStats {
				expMReads += s.reads
				expMReadsPf += s.readsPf
			}

			if obsMReads != expMReads {
				t.Errorf("expected %.2f M reads, found %.2f M", expMReads, obsMReads)
			}

			if obsMReadsPf != expMReadsPf {
				t.Errorf("expected %.2f M reads passing filters, found %.2f M", expMReadsPf, obsMReadsPf)
			}

			if obsPercPf != c.percPf {
				t.Errorf("expected %.2f%% passing filter reads, found %.2f%%", c.percPf, obsPercPf)
			}

			laneDensity := tm.LaneDensity()
			if len(laneDensity) != len(c.laneStats) {
				t.Errorf("expected %d lanes for density stats, found %d", len(c.laneStats), len(laneDensity))
			}
			for lane, obsDensity := range laneDensity {
				obsDensityK := obsDensity / 1000
				// Add some error margin. This can be considered a rounding error.
				if math.Abs(obsDensityK-c.laneStats[lane].density) > 0.02 {
					t.Errorf("expected density %.2fK/mm^2 for lane %d, found %.2fK/mm^2", c.laneStats[lane].density, lane, obsDensityK)
				}
			}
		})
	}
}
