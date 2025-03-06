package interop

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type TileCode int

const (
	TileClusterCountOccupied          = 0
	TileReadNumber                    = 1 // not part of the original spec
	TileClusterDensity                = 100
	TileClusterDensityPf              = 101
	TileClusterCount                  = 102
	TileClusterCountPf                = 103
	TilePhasing                       = 200
	TilePrephasing                    = 201
	TilePercentAligned                = 300
	TileControlLane          TileCode = 400
)

type TileRecord struct {
	LT
	ClusterCount   int
	PfClusterCount int
	Density        float64
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
	tiles := make(map[[2]uint16]TileRecord)
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
		t, ok := tiles[key]
		if !ok {
			t = TileRecord{
				LT: rt.lt1.normalize(),
			}
			tiles[key] = t
		}
		switch rt.Code {
		case TileClusterCount:
			if t.ClusterCount != 0 {
				return fmt.Errorf("cluster count already set for tile %s", t.TileName())
			}
			t.ClusterCount = int(rt.Value)
			tiles[key] = t
		case TileClusterCountPf:
			if t.PfClusterCount != 0 {
				return fmt.Errorf("pf cluster count already set for tile %s", t.TileName())
			}
			t.PfClusterCount = int(rt.Value)
			tiles[key] = t
		case TileClusterDensity:
			if t.Density != 0 {
				return fmt.Errorf("cluster density already set for tile %s", t.TileName())
			}
			t.Density = float64(rt.Value)
			tiles[key] = t
		}
	}
	lanes := make(map[int]bool)
	tm.Records = make([]TileRecord, 0, len(tiles))
	for key, t := range tiles {
		tm.Records = append(tm.Records, t)
		lanes[int(key[0])] = true
	}
	tm.LaneCount = len(lanes)
	return nil
}

func parseTileMetricRecordsV3(r io.Reader, tm *TileMetrics) error {
	lanes := make(map[int]bool)
	for {
		t := rawTileV3{}
		err := binary.Read(r, binary.LittleEndian, &t.lt2)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		_ = binary.Read(r, binary.LittleEndian, &t.Code)
		switch t.Code {
		case 't':
			_ = binary.Read(r, binary.LittleEndian, &t.ClusterCount)
			_ = binary.Read(r, binary.LittleEndian, &t.PfClusterCount)
			tm.Records = append(tm.Records, TileRecord{
				LT:             t.lt2.normalize(),
				ClusterCount:   int(t.ClusterCount),
				PfClusterCount: int(t.PfClusterCount),
			})
			lanes[int(t.Lane)] = true
		case 'r':
			// Discard these for now
			_ = binary.Read(r, binary.LittleEndian, &t.ReadNumber)
			_ = binary.Read(r, binary.LittleEndian, &t.PercentAligned)
		default:
			return fmt.Errorf("invalid tile code: %d", t.Code)
		}
	}
	tm.LaneCount = len(lanes)
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
	defer f.Close()
	r := bufio.NewReader(f)
	tm.Header, err = parseHeader(r)
	if err != nil {
		return tm, nil
	}

	switch tm.Header.Version {
	case 2:
		err = parseTileMetricsV2(r, &tm)
	case 3:
		err = parseTileMetricsV3(r, &tm)
	default:
		return tm, fmt.Errorf("unsupported tile metrics version: %d", tm.Header.Version)
	}

	return tm, err
}
