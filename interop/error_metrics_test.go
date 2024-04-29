package interop

import (
	"bytes"
	"testing"
)

func TestParseErrorMetrics6(t *testing.T) {
	cases := map[string]struct {
		Bytes []byte
	}{
		"case 1": {
			[]byte{6, 12 + 4*1, 0x1, 0x0, 0x3, 0x0, 0x1, 0x2, 0x3},
		},
	}

	for k, v := range cases {
		r := bytes.NewReader(v.Bytes)
		m := ErrorMetrics6{}
		m.Parse(r)

		t.Logf("%s: %#v", k, m)
	}
}

func TestParseErrorMetrics(t *testing.T) {
	cases := map[string]struct {
		Filename    string
		ReadConfig
		CycleErrors map[int]map[int]map[int]float64
		ReadErrors  map[int]map[int][2]float64
	}{
		"novaseq": {
			"../test_data/novaseq_full/ErrorMetricsOut.bin",
			ReadConfig{
				ReadLengths: map[int]int{
					1: 151,
					2: 8,
					3: 8,
					4: 151,
				},
			},
			map[int]map[int]map[int]float64{
				1: {1123: {105: 0.118, 313: 0.403}},
				2: {1103: {273: 0.416, 283: 0.250}},
			},
			map[int]map[int][2]float64{
				1: {1: {0.13, 0.03}, 4: {0.29, 0.10}},
				2: {1: {0.15, 0.04}, 4: {0.21, 0.06}},
			},
		},
	}

	for k, v := range cases {
		m, err := ParseErrorMetrics(v.Filename)
		if err != nil {
			t.Fatalf(`case "%s": %s when parsing error metrics`, k, err.Error())
		}

		tiles := make(map[int]map[int]bool)
		for _, r := range m.Records() {
			record := r.(ErrorMetricRecord6)
			if _, ok := tiles[int(record.Lane)]; !ok {
				tiles[int(record.Lane)] = make(map[int]bool)
			}
			tiles[int(record.Lane)][int(record.Tile)] = true
		}

		s := m.CycleErrorRate()
		for lane, lanestats := range v.CycleErrors {
			for tile, tilestats := range lanestats {
				for cycle, errorRate := range tilestats {
					if !almostEqual(errorRate, roundFloat(s[lane][tile][cycle].Mean, 3)) {
						t.Errorf(`case "%s": expected error rate %.03f for lane %d, tile %d, cycle %d, got %f`,
							k, errorRate, lane, tile, cycle, s[lane][tile][cycle].Mean)
					}
				}
			}
		}

		r := m.ReadErrorRate(v.ReadConfig)
		for lane, lanestats := range v.ReadErrors {
			for read, errorRate := range lanestats {
				if !almostEqual(errorRate[0], roundFloat(r[lane][read].Mean, 2)) {
					t.Errorf(`case "%s": expected error rate %.02f for lane %d, read %d, got %f`,
						k, errorRate[0], lane, read, r[lane][read].Mean)
				}
				if !almostEqual(errorRate[1], roundFloat(r[lane][read].SD(), 2)) {
					t.Errorf(`case "%s": expected error rate SD %.02f for lane %d, read %d, got %f`,
						k, errorRate[1], lane, read, r[lane][read].SD())
				}
			}
		}
	}
}
