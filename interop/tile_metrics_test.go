package interop

import (
	"bytes"
	"math"
	"testing"
)

func almostEqual[T float32 | float64](a, b T) bool {
	return (a - b) < T(math.Pow(10, -6))
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
	if record.Value != 25.7 {
		t.Errorf("expected value 25.7, got %.2f", record.Value)
	}
}

func TestParseTileMetrics2(t *testing.T) {
	b := []byte{0x2, 0xf, 0x1, 0x0, 0x7e, 0x4, 0xc8, 0x0, 0x9a, 0x99, 0xcd, 0x41}
	r := bytes.NewReader(b)
	m := &TileMetrics2{}
	if err := m.Parse(r); err != nil {
		t.Fatal(err.Error())
	}

	if m.Version != 2 {
		t.Errorf("expected version 2, found %d", m.Version)
	}

	if m.RecordSize != 15 {
		t.Errorf("expected record size 15, found %d", m.RecordSize)
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
		RecordCount int
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
			[]byte{0x3, 0x14, 0x14, 0xae, 0x47, 0xf3,
				0x2, 0x0,
				0x7e, 0x4, 0x0, 0x0,
				0x74,
				0x0, 0x20, 0xf1, 0x47,
				0x0, 0x4, 0xf1, 0x47,
			},
			1,
			3,
			20,
			2,
			1150,
			't',
			123456,
			123400,
			0,
			0,
		},
		"second": {
			[]byte{0x3, 0x14, 0x14, 0xae, 0x47, 0xf3, // version, record size, density
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
			20,
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
	cases := map[string]struct{
		Filename string
		ShouldFail bool
		RecordCount int
		Version uint8
	} {
		"novaseq": {
			"../test_data/novaseq/TileMetricsOut.bin",
			false,
			1680,
			3,
		},
		"nextseq": {
			"../test_data/nextseq1/TileMetricsOut.bin",
			false,
			2880,
			2,
		},
		"invalid": {
			"../test_data/novaseq/QMetricsOut.bin",
			true,
			0,
			0,
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
	}
}
