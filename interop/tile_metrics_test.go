package interop

import (
	"math"
	"testing"
)

var m TileMetrics = TileMetrics{
	InteropFile{Version: 3},
	15,
	1.0,
	[]TileRecord{
		{LTC{1, 1, uint8('t')}, TileMetric{1000, 800}, ReadMetric{0, 0}},
		{LTC{1, 1, uint8('r')}, TileMetric{0, 0}, ReadMetric{1, float32(math.NaN())}},
		{LTC{1, 1, uint8('r')}, TileMetric{0, 0}, ReadMetric{4, float32(math.NaN())}},
		{LTC{2, 1, uint8('t')}, TileMetric{1500, 1200}, ReadMetric{0, 0}},
		{LTC{2, 1, uint8('r')}, TileMetric{0, 0}, ReadMetric{1, 0.80}},
		{LTC{2, 1, uint8('r')}, TileMetric{0, 0}, ReadMetric{4, 0.85}},
		{LTC{3, 1, uint8('t')}, TileMetric{1200, 1000}, ReadMetric{0, 0}},
		{LTC{3, 1, uint8('r')}, TileMetric{0, 0}, ReadMetric{1, 0.83}},
		{LTC{3, 1, uint8('r')}, TileMetric{0, 0}, ReadMetric{4, 0.81}},
	},
}

func almostEqual(a, b float64) bool {
	return (a-b) < math.Pow(10, -6)
}

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val * ratio) / ratio
}

func TestParseTileMetrics(t *testing.T) {
	table := []struct {
		Filename string
		Version  uint8
	} {
		{"../test_data/novaseq/TileMetricsOut.bin", 3},
	}

	for _, v := range table {
		m, err := ParseTileMetrics(v.Filename)
		if err != nil {
			t.Fatal(err)
		}

		if m.Version != v.Version {
			t.Fatalf("expected version %d, got %d", v.Version, m.Version)
		}

		if m.RecordSize != 15 {
			t.Fatalf("expected record size 15, got %d", m.RecordSize)
		}

		lanePF, lanePFSD := m.PercentPFLane()
		if !almostEqual(79.05, roundFloat(lanePF[1], 2)) {
			t.Fatalf("expected %%PF = %f for lane 1, got %f", 79.05, lanePF[1])
		}
		if !almostEqual(76.65, roundFloat(lanePF[2], 2)) {
			t.Fatalf("expected %%PF = %f for lane 1, got %f", 76.65, lanePF[2])
		}
		if !almostEqual(2.69, roundFloat(lanePFSD[1], 2)) {
			t.Fatalf("expected %%PF SD = %f for lane 1, got %f", 2.69, lanePFSD[1])
		}
		if !almostEqual(2.41, roundFloat(lanePFSD[2], 2)) {
			t.Fatalf("expected %%PF SD = %f for lane 1, got %f", 2.41, lanePFSD[2])
		}
	}
}

func TestPercentPF(t *testing.T) {
	p, sd := m.PercentPF()
	expectedPercentage := 81.0810810811
	expectedSD := 1.924500897
	if !almostEqual(p, expectedPercentage) {
		t.Fatalf("expected %f%% passing filter, got %f%%", expectedPercentage, p)
	}
	if !almostEqual(sd, expectedSD) {
		t.Fatalf("expected standard deviation of %f, got %f", expectedSD, sd)
	}
}

func TestPercentAligned(t *testing.T) {
	p, sd := m.PercentAligned()
	expectedPercent := 0.8225
	expectedSD := 0.2217356
	if !almostEqual(p, expectedPercent) {
		t.Fatalf("expected %f percent aligned, got %f percent", expectedPercent, p)
	}
	if !almostEqual(sd, expectedSD) {
		t.Fatalf("expected sd %f, got %f", expectedSD, sd)
	}
}
