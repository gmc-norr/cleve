package interop

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
)

type TileCode int

const (
	TileClusterCountOccupied = 0
	TileClusterDensity       = 100
	TileClusterDensityPf     = 101
	TileClusterCount         = 102
	TileClusterCountPf       = 103
	TilePhasing              = 200
	TilePrephasing           = 201
	TilePercentAligned       = 300
	TileControlLane          = 400
)

type TileRecord struct {
	LT
	ClusterCount   int
	PfClusterCount int
	Density        float64
	PercentAligned map[int]float64 // Percent aligned to PhiX for each read
}

type TileMetrics struct {
	Header
	LaneCount int
	Density   float64
	Records   []TileRecord
}

func (m TileMetrics) LaneClusters() map[int]int {
	sums := make(map[int]int)
	for _, t := range m.Records {
		sums[t.Lane] += t.ClusterCount
	}
	return sums
}

// Clusters returns the total number of clusters.
func (m TileMetrics) Clusters() int {
	sum := 0
	for _, n := range m.LaneClusters() {
		sum += n
	}
	return sum
}

func (m TileMetrics) LanePfClusters() map[int]int {
	sums := make(map[int]int)
	for _, t := range m.Records {
		sums[t.Lane] += t.PfClusterCount
	}
	return sums
}

// PfClusters returns the number of passing filter clusters.
func (m TileMetrics) PfClusters() int {
	sum := 0
	for _, n := range m.LanePfClusters() {
		sum += n
	}
	return sum
}

func (m TileMetrics) ReadPercentAligned() map[int]map[int]float64 {
	readPercentAligned := make(map[int]map[int]float64)
	counts := make(map[int]map[int]int)
	for _, r := range m.Records {
		for read, v := range r.PercentAligned {
			if math.IsNaN(v) {
				continue
			}
			if _, ok := readPercentAligned[read]; !ok {
				readPercentAligned[read] = make(map[int]float64)
				counts[read] = make(map[int]int)
			}
			readPercentAligned[read][r.Lane] += v
			counts[read][r.Lane]++
		}
	}
	for read := range readPercentAligned {
		for lane := range readPercentAligned[read] {
			readPercentAligned[read][lane] /= float64(counts[read][lane])
		}
	}
	return readPercentAligned
}

func (m TileMetrics) LanePercentAligned() map[int]float64 {
	lanePercentAligned := make(map[int]float64)
	counts := make(map[int]int)
	readPercentAligned := m.ReadPercentAligned()
	for read := range readPercentAligned {
		for lane, v := range readPercentAligned[read] {
			lanePercentAligned[lane] += v
			counts[lane]++
		}
	}
	for lane := range lanePercentAligned {
		lanePercentAligned[lane] /= float64(counts[lane])
	}
	return lanePercentAligned
}

func (m TileMetrics) PercentAligned() float64 {
	sum := 0.0
	laneAligned := m.LanePercentAligned()
	for _, v := range laneAligned {
		sum += v
	}
	return sum / float64(len(laneAligned))
}

func (m TileMetrics) LaneDensity() map[int]float64 {
	laneDensities := make(map[int]float64)
	laneCounts := make(map[int]int, m.LaneCount)
	for _, r := range m.Records {
		if m.Density != 0 {
			// Patterened flow cell, all tiles have the same density
			// Not obvious from the docs how this was calculated: https://github.com/Illumina/interop/blob/cda2299f286965eb2768f0491607bf340bbe0f38/src/interop/model/metrics/tile_metric.cpp#L379-L390
			laneCounts[r.Lane]++
			laneDensities[r.Lane] += float64(r.ClusterCount) / float64(m.Density)
		} else {
			laneCounts[r.Lane]++
			laneDensities[r.Lane] += r.Density
		}
	}
	for lane := range laneDensities {
		laneDensities[lane] /= float64(laneCounts[lane])
	}
	return laneDensities
}

func (m TileMetrics) RunDensity() float64 {
	runDensity := 0.0
	n := 0
	for _, d := range m.LaneDensity() {
		runDensity += d
		n++
	}
	return runDensity / float64(n)
}

// FractionPassingFilter returns the fraction of clusters passing filters.
func (m TileMetrics) FractionPassingFilter() float64 {
	return float64(m.PfClusters()) / float64(m.Clusters())
}

type rawTileV2 struct {
	lt1
	Code  uint16
	Value float32
}

type rawTileV3 struct {
	lt2
	Code           uint8
	ClusterCount   float32
	PfClusterCount float32
	ReadNumber     uint32
	PercentAligned float32
}

func parseTileMetricRecordsV2(r io.Reader, tm *TileMetrics) error {
	tiles := make(map[[2]uint16]*TileRecord)
	for {
		rt := rawTileV2{}
		err := binary.Read(r, binary.LittleEndian, &rt)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		key := [2]uint16{rt.Lane, rt.Tile}
		if _, ok := tiles[key]; !ok {
			tiles[key] = &TileRecord{
				LT:             rt.normalize(),
				PercentAligned: make(map[int]float64),
			}
		}
		t := tiles[key]

		switch rt.Code {
		case TileControlLane:
			continue
		case TileClusterCount:
			if t.ClusterCount != 0 {
				return fmt.Errorf("cluster count already set for tile %s", t.TileName())
			}
			t.ClusterCount = int(rt.Value)
		case TileClusterCountPf:
			if t.PfClusterCount != 0 {
				return fmt.Errorf("pf cluster count already set for tile %s", t.TileName())
			}
			t.PfClusterCount = int(rt.Value)
		case TileClusterDensity:
			if t.Density != 0 {
				return fmt.Errorf("cluster density already set for tile %s", t.TileName())
			}
			t.Density = float64(rt.Value)
		case TileClusterDensityPf:
			continue
		default:
			if rt.Code%TilePercentAligned < 100 {
				readIndex := int(rt.Code % TilePercentAligned)
				t.PercentAligned[readIndex+1] = float64(rt.Value)
			} else if rt.Code%TilePhasing < 100 {
			} else {
				return fmt.Errorf("unknown tile code: %d", rt.Code)
			}
		}
	}
	lanes := make(map[int]bool)
	tm.Records = make([]TileRecord, 0, len(tiles))
	for key, t := range tiles {
		tm.Records = append(tm.Records, *t)
		lanes[int(key[0])] = true
	}
	tm.LaneCount = len(lanes)
	return nil
}

func parseTileMetricRecordsV3(r io.Reader, tm *TileMetrics) error {
	records := make(map[int]map[int]*TileRecord)
	for {
		t := rawTileV3{}
		err := binary.Read(r, binary.LittleEndian, &t.lt2)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		lane := int(t.Lane)
		tile := int(t.Tile)

		if _, ok := records[lane]; !ok {
			records[lane] = make(map[int]*TileRecord)
		}
		if _, ok := records[lane][tile]; !ok {
			records[lane][tile] = &TileRecord{
				LT:             t.normalize(),
				PercentAligned: make(map[int]float64),
			}
		}

		_ = binary.Read(r, binary.LittleEndian, &t.Code)
		switch t.Code {
		case 't':
			_ = binary.Read(r, binary.LittleEndian, &t.ClusterCount)
			_ = binary.Read(r, binary.LittleEndian, &t.PfClusterCount)
			records[lane][tile].ClusterCount = int(t.ClusterCount)
			records[lane][tile].PfClusterCount = int(t.PfClusterCount)
		case 'r':
			_ = binary.Read(r, binary.LittleEndian, &t.ReadNumber)
			_ = binary.Read(r, binary.LittleEndian, &t.PercentAligned)
			records[lane][tile].PercentAligned[int(t.ReadNumber)] = float64(t.PercentAligned)
		default:
			return fmt.Errorf("invalid tile code: %d", t.Code)
		}
	}
	tm.LaneCount = len(records)
	for lane := range records {
		for _, record := range records[lane] {
			tm.Records = append(tm.Records, *record)
		}
	}
	return nil
}

func parseTileMetricsV2(r io.Reader, tm *TileMetrics) error {
	return parseTileMetricRecordsV2(r, tm)
}

func parseTileMetricsV3(r io.Reader, tm *TileMetrics) error {
	var density float32
	err := binary.Read(r, binary.LittleEndian, &density)
	if err != nil {
		return err
	}
	tm.Density = float64(density)
	return parseTileMetricRecordsV3(r, tm)
}

func ReadTileMetrics(filename string) (tm TileMetrics, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return tm, err
	}
	defer func() { _ = f.Close() }()
	r := bufio.NewReader(f)
	tm.Header, err = parseHeader(r)
	if err != nil {
		return tm, nil
	}

	switch tm.Version {
	case 2:
		err = parseTileMetricsV2(r, &tm)
	case 3:
		err = parseTileMetricsV3(r, &tm)
	default:
		return tm, fmt.Errorf("unsupported tile metrics version: %d", tm.Version)
	}

	return tm, err
}
