package interop

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type TileMetric struct {
	ClusterCount   float32
	PFClusterCount float32
}

type ReadMetric struct {
	ReadNumber     uint32
	PercentAligned float32
}

type LTC struct {
	Lane uint16
	Tile uint32
	Code uint8
}

type TileRecord struct {
	LTC
	TileMetric
	ReadMetric
}

type TileMetrics struct {
	InteropFile
	RecordSize uint8
	Density    float32
	TileRecords []TileRecord
}

func (m *TileMetrics) ParseRecords(r io.Reader) error {
	for {
		record := TileRecord{}
		if err := binary.Read(r, binary.LittleEndian, &record.LTC); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		switch rune(record.Code) {
		case 't':
			binary.Read(r, binary.LittleEndian, &record.TileMetric)
		case 'r':
			binary.Read(r, binary.LittleEndian, &record.ReadMetric)
		default:
			return fmt.Errorf("unknown code %q", record.Code)
		}
		m.TileRecords = append(m.TileRecords, record)
	}
	return nil
}

// Parse tile metrics from `InterOp/TileMetricsOut.bin` or
// `InterOp/TileMetrics.bin` files.
func ParseTileMetrics(filename string) (*TileMetrics, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := bufio.NewReader(f)

	m := &TileMetrics{}
	binary.Read(r, binary.LittleEndian, &m.Version)
	binary.Read(r, binary.LittleEndian, &m.RecordSize)
	binary.Read(r, binary.LittleEndian, &m.Density)
	
	m.ParseRecords(r)

	return m, nil
}

// Calculate the percentage of all clusters passing filters on the
// flowcell, returning the mean percentage passing filters and the
// standard deviation across all tiles.
func (m TileMetrics) PercentPF() (float64, float64) {
	v := NewRunningVariance[float64](true)
	for _, r := range m.TileRecords {
		if rune(r.Code) != 't' {
			continue
		}
		weight := float64(r.TileMetric.ClusterCount)
		v.Push(100 * float64(r.TileMetric.PFClusterCount) / float64(r.TileMetric.ClusterCount), weight)
	}
	return v.Mean, v.SD()
}

// Calculate the percentage of all clusters passing filters per lane.
// Two maps are returned with the keys representing the lane and the
// values represent the percentage passing filters and the standard
// deviation, respectively.
func (m TileMetrics) PercentPFLane() (map[uint16]float64, map[uint16]float64) {
	vars := make(map[uint16]*RunningVariance[float64])

	for _, r := range m.TileRecords {
		if rune(r.Code) != 't' {
			continue
		}
		if _, ok := vars[r.Lane]; !ok {
			vars[r.Lane] = NewRunningVariance[float64](true)
		}
		weight := float64(r.TileMetric.ClusterCount)
		vars[r.Lane].Push(100 * float64(r.TileMetric.PFClusterCount) / float64(r.TileMetric.ClusterCount), weight)
	}
	
	lanePercent := make(map[uint16]float64)
	laneSD := make(map[uint16]float64)
	for k, v := range vars {
		lanePercent[k] = v.Mean
		laneSD[k] = v.SD()
	}
	return lanePercent, laneSD
}

// Calculate percentage aligned against PhiX across all tiles on the
// flowcell, returning the mean percentage aligned and the standard
// deviation.
func (m *TileMetrics) PercentAligned() (float64, float64) {
	v := NewRunningVariance[float32](false)
	for _, r := range m.TileRecords {
		if rune(r.Code) != 'r' {
			continue
		}

		v.Push(r.ReadMetric.PercentAligned)
	}

	return v.Mean, v.SD()
}
