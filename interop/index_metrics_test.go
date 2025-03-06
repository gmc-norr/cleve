package interop

import (
	"math"
	"testing"
)

func TestReadIndexMetrics(t *testing.T) {
	testcases := []struct {
		name        string
		path        string
		shouldError bool
		version     int
		pfClusters  int
		totalYield  int
		samplePerc  map[string]float64
	}{
		{
			name:       "novaseq 1",
			path:       "./test/20250123_LH00352_0033_A225H35LT1/InterOp/IndexMetricsOut.bin",
			version:    2,
			pfClusters: 4_346_424_164,
			samplePerc: map[string]float64{
				"cfDNA-PoN-1":  5.536,
				"cfDNA-PoN-2":  5.274,
				"cfDNA-PoN-3":  6.201,
				"cfDNA-PoN-4":  5.949,
				"cfDNA-PoN-5":  5.54,
				"cfDNA-PoN-6":  5.159,
				"cfDNA-PoN-7":  6.841,
				"cfDNA-PoN-8":  4.695,
				"cfDNA-PoN-9":  6.368,
				"cfDNA-PoN-10": 6.064,
				"cfDNA-PoN-11": 6.491,
				"cfDNA-PoN-12": 6.594,
				"cfDNA-PoN-13": 6.022,
				"cfDNA-PoN-14": 6.668,
				"cfDNA-PoN-15": 5.975,
				"cfDNA-PoN-16": 6.219,
			},
		},
		{
			name:       "novaseq 2",
			path:       "./test/20250115_LH00352_0031_A225HMVLT1/InterOp/IndexMetricsOut.bin",
			version:    2,
			pfClusters: 4_240_295_482,
			samplePerc: map[string]float64{
				"PoN_25": 2.746,
				"PoN_26": 1.375,
				"PoN_27": 1.199,
				"PoN_28": 2.238,
				"PoN_29": 1.16,
				"PoN_30": 1.673,
				"PoN_31": 1.387,
				"PoN_32": 1.314,
				"PoN_33": 1.508,
				"PoN_34": 1.214,
				"PoN_35": 1.384,
				"PoN_36": 1.834,
				"PoN_37": 1.832,
				"PoN_38": 1.499,
				"PoN_39": 1.5,
				"PoN_40": 1.239,
				"7":      1.185,
				"218":    1.277,
				"236":    1.487,
				"322":    1.551,
				"342":    1.743,
				"807":    2.233,
				"921":    1.284,
				"1008":   1.429,
				"1044":   2.105,
				"1438":   1.571,
				"1495":   1.741,
				"1524":   1.465,
				"1696":   1.501,
				"1708":   1.681,
				"2446":   1.423,
				"2458":   1.554,
				"2465":   1.392,
				"2491":   1.132,
				"2655":   1.619,
				"3094":   1.436,
				"3235":   1.453,
				"3236":   1.368,
				"3327":   1.32,
				"3400":   1.338,
				"3534":   1.726,
				"3540":   1.357,
				"3623":   1.775,
				"3957":   1.372,
				"4462":   1.459,
				"4728":   1.15,
				"5046":   1.879,
				"5051":   1.287,
			},
		},
		{
			name:       "nextseq",
			path:       "./test/250210_NB551119_0457_AHL3Y2AFX7/InterOp/IndexMetricsOut.bin",
			version:    2,
			pfClusters: 269_272_238,
			samplePerc: map[string]float64{
				"R25-609":  3.935,
				"R25-610":  3.537,
				"R25-924":  3.069,
				"R25-923":  2.77,
				"R25-936":  3.228,
				"R25-974":  3.421,
				"R25-978":  3.986,
				"R25-982":  4.121,
				"D25-1453": 0,
				"R25-609n": 4.661,
				"R25-610n": 4.117,
				"R25-924n": 1.71,
				"R25-923n": 1.398,
				"R25-936n": 2.793,
				"R25-974n": 1.776,
				"R25-978n": 2.043,
				"R25-982n": 4.739,
			},
		},
		{
			name:       "miseq",
			path:       "./test/250207_M00568_0665_000000000-LMWPP/InterOp/IndexMetricsOut.bin",
			version:    1,
			pfClusters: 18125806,
			samplePerc: map[string]float64{
				"1": 9.8044,
				"2": 10.1302,
				"3": 22.3386,
				"4": 8.6187,
				"5": 9.3575,
				"6": 5.8574,
				"7": 6.3033,
				"8": 8.6855,
			},
		},
		{
			name:       "miseq old",
			path:       "./test/160122_M00568_0146_000000000-ALYCY/InterOp/IndexMetricsOut.bin",
			version:    1,
			pfClusters: 21511012,
			samplePerc: map[string]float64{
				"15-8851": 15.9134,
				"15-8852": 15.5889,
				"15-8923": 16.5010,
				"15-8932": 16.5410,
				"15-8951": 16.2346,
				"15-8962": 16.5886,
			},
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			im, err := ReadIndexMetrics(c.path)
			if err != nil && !c.shouldError {
				t.Fatal(err)
			} else if err == nil && c.shouldError {
				t.Fatal("expected error, got nil")
			} else if err != nil && c.shouldError {
				return
			}

			if im.Version != uint8(c.version) {
				t.Errorf("expected version %d, got %d", c.version, im.Version)
			}

			for sample, yield := range im.SampleYield() {
				percPf := 100 * float64(yield) / float64(c.pfClusters)
				if math.Abs(percPf-c.samplePerc[sample]) > 0.0006 {
					t.Errorf("expected %.2f%% of PF reads for sample %s, got %.2f%%", c.samplePerc[sample], sample, percPf)
				}
			}
		})
	}
}
