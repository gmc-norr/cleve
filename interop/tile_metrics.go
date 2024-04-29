package interop

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
)

type TileMetrics interface {
	InteropFile
	InteropRecordHolder
	Parse(io.Reader) error
	ClusterCount() float64
	PFClusterCount() float64
	LaneReadPercentAligned() map[int]map[int][2]float64
	PercentAligned() float64
}

type TileMetricsRecord interface {
	InteropRecord
}

type TileMetricRecord2 struct {
	Lane  uint16
	Tile  uint16
	Code  uint16
	Value float32
}

func (m *TileMetricRecord2) Parse(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, m); err != nil {
		// Expecting EOF here
		if err != io.EOF {
			return fmt.Errorf("%s when parsing tile metric v2 record", err.Error())
		}
		return err
	}
	return nil
}

func (m *TileMetricRecord2) Type() string {
	// Tile metric codes come from https://github.com/Illumina/interop/blob/520e8ab8a5a3f3d5fa44ab4d32643fb7c6da0b30/src/interop/model/metrics/tile_metric.cpp#L76-L86
	c := m.Code
	switch {
	case c == 100:
		return "cluster_density"
	case c == 101:
		return "pf_cluster_density"
	case c == 102:
		return "cluster_count"
	case c == 103:
		return "pf_cluster_count"
	case c >= 200 && c < 300 && c%2 == 0:
		return "phasing"
	case c >= 201 && c < 300 && c%2 == 1:
		return "prephasing"
	case c >= 300 && c < 400:
		return "percent_aligned"
	case c == 400:
		return "control_lane"
	default:
		return "unknown"
	}
}

func (m *TileMetricRecord2) GetRead() int {
	switch m.Type() {
	case "phasing":
		return (int(m.Code) - 200 + 2) / 2
	case "prephasing":
		return (int(m.Code) - 201 + 2) / 2
	case "percent_aligned":
		return int(m.Code) - 300 + 1
	default:
		return -1
	}
}

type TileMetrics2 struct {
	InteropHeader
	TileMetricRecords []InteropRecord
}

func (m TileMetrics2) Records() []InteropRecord {
	return m.TileMetricRecords
}

func (m *TileMetrics2) Parse(r io.Reader) error {
	if err := m.InteropHeader.Parse(r); err != nil {
		return err
	}

	if m.Version != 2 {
		return fmt.Errorf("expected v3, got v%d", m.Version)
	}

	if m.RecordSize != 10 {
		return fmt.Errorf("expected a record size of 10 bytes, got %d bytes", m.RecordSize)
	}

	for {
		record := &TileMetricRecord2{}
		if err := record.Parse(r); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		m.TileMetricRecords = append(m.TileMetricRecords, record)
	}
}

func (m TileMetrics2) LaneClusterCount() map[int][2]float64 {
	laneSummaries := make(map[int]*RunningSummary[float64])

	for _, r := range m.TileMetricRecords {
		if r.Type() != "cluster_count" {
			continue
		}
		record := r.(*TileMetricRecord2)
		lane := int(record.Lane)
		if _, ok := laneSummaries[lane]; !ok {
			laneSummaries[lane] = &RunningSummary[float64]{}
		}
		laneSummaries[lane].Push(float64(record.Value))
	}

	laneCounts := make(map[int][2]float64)
	for lane, v := range laneSummaries {
		laneCounts[lane] = [2]float64{v.Sum, v.SD()}
	}
	return laneCounts
}

func (m TileMetrics2) ClusterCount() float64 {
	var clusterCount float64 = 0
	laneCounts := m.LaneClusterCount()
	for _, v := range laneCounts {
		clusterCount += v[0]
	}
	return clusterCount
}

func (m TileMetrics2) LanePFClusterCount() map[int]float64 {
	laneCounts := make(map[int]float64)
	for _, r := range m.TileMetricRecords {
		if r.Type() != "pf_cluster_count" {
			continue
		}
		laneCounts[int(r.(*TileMetricRecord2).Lane)] += float64(r.(*TileMetricRecord2).Value)
	}
	return laneCounts
}

func (m TileMetrics2) PFClusterCount() float64 {
	var clusterCount float64 = 0
	laneCounts := m.LanePFClusterCount()
	for _, v := range laneCounts {
		clusterCount += v
	}
	return clusterCount
}

func (m TileMetrics2) LaneReadPercentAligned() map[int]map[int][2]float64 {
	laneSummaries := make(map[int]map[int]*RunningSummary[float64])
	for _, r := range m.TileMetricRecords {
		if r.Type() != "percent_aligned" {
			continue
		}
		record := r.(*TileMetricRecord2)
		lane := int(record.Lane)
		read := record.GetRead()
		if _, ok := laneSummaries[lane]; !ok {
			laneSummaries[lane] = make(map[int]*RunningSummary[float64])
		}
		if _, ok := laneSummaries[lane][read]; !ok {
			laneSummaries[lane][read] = &RunningSummary[float64]{}
		}
		laneSummaries[lane][read].Push(float64(record.Value))
	}

	lanePercentages := make(map[int]map[int][2]float64)
	for lane, v := range laneSummaries {
		if _, ok := lanePercentages[lane]; !ok {
			lanePercentages[lane] = make(map[int][2]float64)
		}
		for read, v2 := range v {
			lanePercentages[lane][read] = [2]float64{v2.Mean, v2.SD()}
		}
	}

	return lanePercentages
}

func (m TileMetrics2) PercentAligned() float64 {
	lanePercentages := m.LaneReadPercentAligned()
	var percentAligned float64 = 0
	var n float64 = 0
	for _, v := range lanePercentages {
		for _, v2 := range v {
			n++
			percentAligned += v2[0]
		}
	}

	return percentAligned / n
}

type TileRecord3 struct {
	ClusterCount   float32
	PFClusterCount float32
}

type ReadRecord3 struct {
	ReadNumber     uint32
	PercentAligned float32
}

type TileMetricRecord3 struct {
	Lane uint16
	Tile uint32
	Code byte
	TileRecord3
	ReadRecord3
}

func (m *TileMetricRecord3) Parse(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, &m.Lane); err != nil {
		// Expecting EOF here
		if err != io.EOF {
			return fmt.Errorf("%s when parsing tile metric v3 lane", err.Error())
		}
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &m.Tile); err != nil {
		return fmt.Errorf("%s when parsing tile metric v3 tile", err.Error())
	}
	if err := binary.Read(r, binary.LittleEndian, &m.Code); err != nil {
		return fmt.Errorf("%s when parsing tile metric v3 code", err.Error())
	}

	switch rune(m.Code) {
	case 'r':
		if err := binary.Read(r, binary.LittleEndian, &m.ReadRecord3); err != nil {
			return fmt.Errorf("%s when parsing tile metric v3 read record", err.Error())
		}
	case 't':
		if err := binary.Read(r, binary.LittleEndian, &m.TileRecord3); err != nil {
			return fmt.Errorf("%s when parsing tile metric v3 tile record", err.Error())
		}
	default:
		return fmt.Errorf("invalid record code for version 3: %c", m.Code)
	}

	return nil
}

func (m TileMetricRecord3) Type() string {
	return string(m.Code)
}

type TileMetrics3 struct {
	InteropHeader
	Density           float32
	TileMetricRecords []InteropRecord
}

func (m *TileMetrics3) Parse(r io.Reader) error {
	if err := m.InteropHeader.Parse(r); err != nil {
		return err
	}

	if m.Version != 3 {
		return fmt.Errorf("expected v3, got v%d", m.Version)
	}

	if m.RecordSize != 15 {
		return fmt.Errorf("expected a record size of 15 bytes, got %d bytes", m.RecordSize)
	}

	if err := binary.Read(r, binary.LittleEndian, &m.Density); err != nil {
		return fmt.Errorf("%s when parsing v3 tile metric density", err.Error())
	}

	for {
		record := &TileMetricRecord3{}
		if err := record.Parse(r); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("%s when parsing v3 tile metric record", err.Error())
		}
		m.TileMetricRecords = append(m.TileMetricRecords, record)
	}
}

func (m TileMetrics3) LaneClusterCount() map[int]float64 {
	laneCounts := make(map[int]float64)
	for _, r := range m.TileMetricRecords {
		if r.Type() != "t" {
			continue
		}
		laneCounts[int(r.(*TileMetricRecord3).Lane)] += float64(r.(*TileMetricRecord3).ClusterCount)
	}
	return laneCounts
}

func (m TileMetrics3) ClusterCount() float64 {
	var count float64 = 0
	for _, v := range m.LaneClusterCount() {
		count += v
	}
	return count
}

func (m TileMetrics3) LanePFClusterCount() map[int]float64 {
	laneCounts := make(map[int]float64)
	for _, r := range m.TileMetricRecords {
		if r.Type() != "t" {
			continue
		}
		laneCounts[int(r.(*TileMetricRecord3).Lane)] += float64(r.(*TileMetricRecord3).PFClusterCount)
	}
	return laneCounts
}

func (m TileMetrics3) PFClusterCount() float64 {
	var count float64 = 0
	for _, v := range m.LanePFClusterCount() {
		count += v
	}
	return count
}

func (m TileMetrics3) LanePercentPF() map[int]float64 {
	laneCounts := m.LaneClusterCount()
	lanePFCounts := m.LanePFClusterCount()
	lanePercentages := make(map[int]float64)

	for k, v := range laneCounts {
		lanePercentages[k] = 100 * v / lanePFCounts[k]
	}

	return lanePercentages
}

func (m TileMetrics3) PercentPF() float64 {
	var sum float64 = 0
	lanePercentages := m.LanePercentPF()
	for _, v := range m.LanePercentPF() {
		sum += v
	}
	return sum / float64(len(lanePercentages))
}

func (m TileMetrics3) LaneReadPercentAligned() map[int]map[int][2]float64 {
	var laneSummaries = make(map[int]map[int]*RunningSummary[float64])
	for _, v := range m.TileMetricRecords {
		if v.Type() != "r" {
			continue
		}
		record := v.(*TileMetricRecord3)
		if math.IsNaN(float64(record.PercentAligned)) {
			continue
		}
		if _, ok := laneSummaries[int(record.Lane)]; !ok {
			laneSummaries[int(record.Lane)] = make(map[int]*RunningSummary[float64])
		}
		if _, ok := laneSummaries[int(record.Lane)][int(record.ReadNumber)]; !ok {
			laneSummaries[int(record.Lane)][int(record.ReadNumber)] = &RunningSummary[float64]{}
		}
		laneSummaries[int(record.Lane)][int(record.ReadNumber)].Push(float64(record.PercentAligned))
	}

	lanePercent := make(map[int]map[int][2]float64)
	for lane, v := range laneSummaries {
		for read, v2 := range v {
			if _, ok := lanePercent[lane]; !ok {
				lanePercent[lane] = make(map[int][2]float64)
			}
			lanePercent[lane][read] = [2]float64{v2.Mean, v2.SD()}
		}
	}

	return lanePercent
}

func (m TileMetrics3) PercentAligned() float64 {
	var sum float64 = 0
	lanePercent := m.LaneReadPercentAligned()
	n := 0
	for _, v := range lanePercent {
		for _, v2 := range v {
			n++
			sum += v2[0]
		}
	}
	return sum / float64(n)
}

func (m TileMetrics3) Records() []InteropRecord {
	return m.TileMetricRecords
}

func ParseTileMetrics(filename string) (TileMetrics, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := bufio.NewReader(f)

	// Try version 3 first
	m3 := &TileMetrics3{}
	err = m3.Parse(r)

	if err == nil {
		return m3, nil
	}

	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to rewind tile metrics file: %s, %s", filename, err.Error())
	}
	r = bufio.NewReader(f)

	// Then try version 2
	m2 := &TileMetrics2{}
	err = m2.Parse(r)

	if err == nil {
		return m2, nil
	}

	return nil, fmt.Errorf("file is not tile metrics v2 or v3: %s", filename)
}
