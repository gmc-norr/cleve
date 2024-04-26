package interop

import (
	"bytes"
	"math"
	"testing"
)

func almostEqual[T float32 | float64](a, b T) bool {
	return math.Abs(float64(a-b)) < math.Pow(10, -6)
}

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func TestParseTileMetricRecord2(t *testing.T) {
	b := []byte{0x1, 0x0, 0x7e, 0x4, 0xc8, 0x0, 0x9a, 0x99, 0xcd, 0x41}
	r := bytes.NewReader(b)
	record := &TileMetricRecord2{}
	if err := record.Parse(r); err != nil {
		t.Error(err.Error())
	}

	if record.Code != 200 {
		t.Errorf("expected code 200, got %d", record.Code)
	}
	if record.Type() != "phasing" {
		t.Errorf(`expected type "phasing", got %s`, record.Type())
	}
	if record.GetRead() != 1 {
		t.Errorf(`expected record to be associated with read 1, got read %d`, record.GetRead())
	}
	if record.Value != 25.7 {
		t.Errorf("expected value 25.7, got %.2f", record.Value)
	}
}

func TestReadMetrics2(t *testing.T) {
	cases := map[uint16]struct {
		Type string
		Read int
	}{
		100: {"cluster_density", -1},
		101: {"pf_cluster_density", -1},
		102: {"cluster_count", -1},
		103: {"pf_cluster_count", -1},
		200: {"phasing", 1},
		201: {"prephasing", 1},
		202: {"phasing", 2},
		203: {"prephasing", 2},
		204: {"phasing", 3},
		205: {"prephasing", 3},
		206: {"phasing", 4},
		207: {"prephasing", 4},
		300: {"percent_aligned", 1},
		301: {"percent_aligned", 2},
		302: {"percent_aligned", 3},
		303: {"percent_aligned", 4},
	}

	for code, v := range cases {
		m := TileMetricRecord2{}
		m.Code = code
		if m.Type() != v.Type {
			t.Errorf(`expected type "%s" for code %d, got "%s"`, v.Type, code, m.Type())
		}
		if m.GetRead() != v.Read {
			t.Errorf(`expected read %d for code %d, got %d`, v.Read, code, m.GetRead())
		}
	}
}

func TestParseTileMetrics2(t *testing.T) {
	b := []byte{0x2, 0xa, 0x1, 0x0, 0x7e, 0x4, 0xc8, 0x0, 0x9a, 0x99, 0xcd, 0x41}
	r := bytes.NewReader(b)
	m := &TileMetrics2{}
	if err := m.Parse(r); err != nil {
		t.Fatal(err.Error())
	}

	if m.Version != 2 {
		t.Errorf("expected version 2, found %d", m.Version)
	}

	if m.RecordSize != 10 {
		t.Errorf("expected record size 10, found %d", m.RecordSize)
	}

	records := m.Records()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, found %d", len(records))
	}
}

func TestParseTileMetricRecord3(t *testing.T) {
	cases := map[string]struct {
		Bytes          []byte
		Lane           uint16
		Tile           uint32
		Code           rune
		ClusterCount   float32
		PFClusterCount float32
		ReadNumber     uint32
		PercentAligned float32
	}{
		"tile record": {
			[]byte{0x2, 0x0, 0x7e, 0x4, 0x0, 0x0, 0x74, 0x0, 0x20, 0xf1, 0x47, 0x0, 0x4, 0xf1, 0x47},
			2,
			1150,
			't',
			123456,
			123400,
			0,
			0,
		},
		"read record": {
			[]byte{0x2, 0x0, 0x7e, 0x4, 0x0, 0x0, 0x72, 0x4, 0x0, 0x0, 0x0, 0x33, 0xb3, 0xa8, 0x42},
			2,
			1150,
			'r',
			0,
			0,
			4,
			84.35,
		},
	}

	for k, v := range cases {
		r := bytes.NewReader(v.Bytes)
		m := &TileMetricRecord3{}
		if err := m.Parse(r); err != nil {
			t.Fatal(err.Error())
		}

		if m.Lane != v.Lane {
			t.Errorf(`case "%s": expected lane %d, got %d`, k, v.Lane, m.Lane)
		}

		if m.Tile != v.Tile {
			t.Errorf(`case "%s": expected tile %d, got %d`, k, v.Tile, m.Tile)
		}

		if rune(m.Code) != v.Code {
			t.Errorf(`case "%s": expected record code '%c', got '%c'`, k, v.Code, rune(m.Code))
		}

		if m.ClusterCount != v.ClusterCount {
			t.Errorf(`case "%s": expected cluster count %.0f, got %.0f`, k, v.ClusterCount, m.ClusterCount)
		}

		if m.PFClusterCount != v.PFClusterCount {
			t.Errorf(`case "%s": expected PF cluster count %.0f, got %.0f`, k, v.PFClusterCount, m.PFClusterCount)
		}

		if m.ReadNumber != v.ReadNumber {
			t.Errorf(`case "%s": expected read number %d, got %d`, k, v.ReadNumber, m.ReadNumber)
		}

		if m.PercentAligned != v.PercentAligned {
			t.Errorf(`case "%s": expected %.02f%% aligned, got %.02f%%`, k, v.PercentAligned, m.PercentAligned)
		}
	}
}

func TestParseTileMetrics3(t *testing.T) {
	cases := map[string]struct {
		Bytes          []byte
		RecordCount    int
		Version        uint8
		RecordSize     uint8
		Lane           uint16
		Tile           uint32
		Code           rune
		ClusterCount   float32
		PFClusterCount float32
		ReadNumber     uint32
		PercentAligned float32
	}{
		"first": {
			[]byte{0x3, 0xf, 0x14, 0xae, 0x47, 0xf3,
				0x2, 0x0,
				0x7e, 0x4, 0x0, 0x0,
				0x74,
				0x0, 0x20, 0xf1, 0x47,
				0x0, 0x4, 0xf1, 0x47,
			},
			1,
			3,
			15,
			2,
			1150,
			't',
			123456,
			123400,
			0,
			0,
		},
		"second": {
			[]byte{0x3, 0xf, 0x14, 0xae, 0x47, 0xf3, // version, record size, density
				// record 1
				0x2, 0x0,
				0x7e, 0x4, 0x0, 0x0,
				0x74,
				0x0, 0x20, 0xf1, 0x47,
				0x0, 0x4, 0xf1, 0x47,
				// record 2
				0x2, 0x0,
				0x7e, 0x4, 0x0, 0x0,
				0x72,
				0x4, 0x00, 0x00, 0x00,
				0x9a, 0x99, 0xb2, 0x42,
			},
			2,
			3,
			15,
			2,
			1150,
			't',
			123456,
			123400,
			0,
			0,
		},
	}

	for k, v := range cases {
		r := bytes.NewReader(v.Bytes)
		m := &TileMetrics3{}
		if err := m.Parse(r); err != nil {
			t.Fatalf(`case "%s": %s`, k, err.Error())
		}

		if len(m.Records()) != v.RecordCount {
			t.Errorf(`case "%s": expected %d records, found %d`, k, v.RecordCount, len(m.Records()))
		}
	}
}

func TestParseTileMetrics(t *testing.T) {
	cases := map[string]struct {
		Filename               string
		ShouldFail             bool
		RecordCount            int
		Version                uint8
		MegaClusterCount       float64
		MegaPFClusterCount     float64
		PercentAligned         float64
		LaneReadPercentAligned map[int]map[int][2]float64
	}{
		"novaseq": {
			"../test_data/novaseq/TileMetricsOut.bin",
			false,
			1680,
			3,
			2 * 1_393_250_000 * 1e-6,
			(1_101_380_000 + 1_067_940_000) * 1e-6,
			0.73,
			map[int]map[int][2]float64{
				1: {1: {0.85, 0.03}, 4: {0.80, 0.05}},
				2: {1: {0.64, 0.01}, 4: {0.63, 0.01}},
			},
		},
		"nextseq": {
			"../test_data/nextseq1/TileMetricsOut.bin",
			false,
			2880,
			2,
			(39_720_000 + 38_320_000 + 36_870_000 + 36_150_000) * 1e-6,
			(35_110_000 + 33_940_000 + 34_330_000 + 33_760_000) * 1e-6,
			45.15,
			map[int]map[int][2]float64{
				1: {1: {45.07, 0.75}, 4: {44.31, 0.76}},
				2: {1: {45.64, 0.97}, 4: {44.75, 0.97}},
				3: {1: {45.59, 0.36}, 4: {44.86, 0.33}},
				4: {1: {45.90, 0.60}, 4: {45.08, 0.61}},
			},
		},
		"invalid": {
			Filename:   "../test_data/novaseq/QMetricsOut.bin",
			ShouldFail: true,
		},
	}

	for k, v := range cases {
		m, err := ParseTileMetrics(v.Filename)
		if err != nil && !v.ShouldFail {
			t.Fatalf("%s: %s", k, err.Error())
		} else if err != nil {
			t.Logf(`case "%s" failed, as expected: %s`, k, err.Error())
			continue
		}

		if m.GetVersion() != v.Version {
			t.Errorf(`case "%s": expected version %d, got %d`, k, v.Version, m.GetVersion())
		}

		if len(m.Records()) != v.RecordCount {
			t.Errorf(`case "%s": expected %d records, got %d`, k, v.RecordCount, len(m.Records()))
		}

		megaClusters := m.ClusterCount() * math.Pow(10, -6)
		if !almostEqual(roundFloat(megaClusters, 2), v.MegaClusterCount) {
			t.Errorf(`case "%s": expected %.02f million clusters, got %.02f`, k, v.MegaClusterCount, megaClusters)
		}

		megaPFClusters := m.PFClusterCount() * math.Pow(10, -6)
		if !almostEqual(roundFloat(megaPFClusters, 2), v.MegaPFClusterCount) {
			t.Errorf(`case "%s": expected %.02f million clusters, got %.02f`, k, v.MegaPFClusterCount, megaPFClusters)
		}

		if !almostEqual(roundFloat(m.PercentAligned(), 2), v.PercentAligned) {
			t.Errorf(`case "%s": expected %.02f%% percent aligned, got %.02f%%`, k, v.PercentAligned, m.PercentAligned())
		}

		a := m.LaneReadPercentAligned()
		for lane, readmap := range v.LaneReadPercentAligned {
			for read, stats := range readmap {
				if !almostEqual(stats[0], roundFloat(a[lane][read][0], 2)) {
					t.Errorf(`case "%s": expected %.02f%% aligned for lane %d, read %d, got %.02f%%`,
						k, stats[0], lane, read, a[lane][read][0])
				}
				if !almostEqual(stats[1], roundFloat(a[lane][read][1], 2)) {
					t.Errorf(`case "%s": expected ± %.02f%% aligned for lane %d, read %d, got ± %.02f%%`,
						k, stats[1], lane, read, a[lane][read][1])
				}
			}
		}
	}
}
