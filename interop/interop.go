package interop

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"time"
)

// Interop is the representation of Illumina Interop data.
type Interop struct {
	dir string

	runparamsFile string
	RunParameters RunParameters

	runinfoFile string
	RunInfo     RunInfo

	qmetricsFile string
	QMetrics     QMetrics

	tilemetricsFile string
	TileMetrics     TileMetrics

	extendedTileMetricsFile string
	ExtendedTileMetrics     ExtTileMetrics

	errorMetricsFile string
	ErrorMetrics     ErrorMetrics

	indexMetricsFile string
	IndexMetrics     IndexMetrics
}

// Returns the first file that exists, have read permission set,
// and is not a directory. If none is found, returns the last
// error seen.
func alternativeFile(dir string, filenames ...string) (string, error) {
	var err error
	for _, fn := range filenames {
		var info os.FileInfo
		path := filepath.Join(dir, fn)
		info, err = os.Stat(path)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}
		return path, nil
	}
	if err != nil {
		return "", err
	}
	return "", fmt.Errorf("no matching files found")
}

// InteropFromDir creates an Interop object from an Illumina
// run directory. This makes some assumptions when it comes
// to the paths of individual files.
func InteropFromDir(rundir string) (Interop, error) {
	var err error
	i := Interop{}
	i.dir, _ = filepath.Abs(rundir)

	// Mandatory files
	i.runinfoFile, err = alternativeFile(i.dir, "RunInfo.xml")
	if err != nil {
		return i, err
	}
	i.runparamsFile, err = alternativeFile(i.dir, "RunParameters.xml", "runParameters.xml")
	if err != nil {
		return i, err
	}

	// Optional files
	interopdir := filepath.Join(i.dir, "InterOp")
	i.qmetricsFile, _ = alternativeFile(interopdir, "QMetricsOut.bin", "QMetrics.bin")
	i.tilemetricsFile, _ = alternativeFile(interopdir, "TileMetricsOut.bin", "TileMetrics.bin")
	i.extendedTileMetricsFile, _ = alternativeFile(interopdir, "ExtendedTileMetricsOut.bin", "ExtendedTileMetrics.bin")
	i.errorMetricsFile, _ = alternativeFile(interopdir, "ErrorMetricsOut.bin", "ErrorMetrics.bin")

	i.RunInfo, err = ReadRunInfo(i.runinfoFile)
	if err != nil {
		return i, fmt.Errorf("error reading run info: %w", err)
	}
	i.RunParameters, err = ReadRunParameters(i.runparamsFile)
	if err != nil {
		return i, fmt.Errorf("error reading run parameters: %w", err)
	}

	if i.qmetricsFile != "" {
		i.QMetrics, err = ReadQMetrics(i.qmetricsFile)
		if err != nil {
			return i, fmt.Errorf("error reading QMetrics: %w", err)
		}
	}

	if i.tilemetricsFile != "" {
		i.TileMetrics, err = ReadTileMetrics(i.tilemetricsFile)
		if err != nil {
			return i, fmt.Errorf("error reading TileMetrics: %w", err)
		}
	}

	if i.extendedTileMetricsFile != "" {
		i.ExtendedTileMetrics, err = ReadExtendedTileMetrics(i.extendedTileMetricsFile)
		if err != nil {
			return i, fmt.Errorf("error reading ExtendedTileMetrics: %w", err)
		}
	}

	if i.errorMetricsFile != "" {
		i.ErrorMetrics, err = ReadErrorMetrics(i.errorMetricsFile)
		if err != nil {
			return i, fmt.Errorf("error reading ExtendedTileMetrics: %w", err)
		}
	}

	if i.indexMetricsFile != "" {
		i.IndexMetrics, err = ReadIndexMetrics(i.indexMetricsFile)
		if err != nil {
			return i, fmt.Errorf("error reading IndexMetrics: %w", err)
		}
	}

	return i, nil
}

func (i Interop) excludedCycles() []int {
	exclude := make([]int, 0, 4)
	sum := 0
	for _, read := range i.RunInfo.Reads {
		sum += read.Cycles
		exclude = append(exclude, sum)
	}
	return exclude
}

type Header struct {
	Version    uint8
	RecordSize uint8
}

type LT struct {
	Lane int
	Tile int
}

func (lt LT) TileName() string {
	return fmt.Sprintf("%d_%d", lt.Lane, lt.Tile)
}

type lt1 struct {
	Lane uint16
	Tile uint16
}

func (lt lt1) normalize() LT {
	return LT{
		Lane: int(lt.Lane),
		Tile: int(lt.Tile),
	}
}

type lt2 struct {
	Lane uint16
	Tile uint32
}

func (lt lt2) normalize() LT {
	return LT{
		Lane: int(lt.Lane),
		Tile: int(lt.Tile),
	}
}

type LTC struct {
	LT
	Cycle int
}

type ltc1 struct {
	lt1
	Cycle uint16
}

func (ltc ltc1) normalize() LTC {
	return LTC{
		LT:    ltc.lt1.normalize(),
		Cycle: int(ltc.Cycle),
	}
}

type ltc2 struct {
	lt2
	Cycle uint16
}

func (ltc ltc2) normalize() LTC {
	return LTC{
		LT:    ltc.lt2.normalize(),
		Cycle: int(ltc.Cycle),
	}
}

func parseHeader(r io.Reader) (Header, error) {
	h := Header{}
	err := binary.Read(r, binary.LittleEndian, &h)
	return h, err
}

// TotalYield returns the total yield for the sequencing run in bases.
func (i Interop) TotalYield() int {
	bases := 0
	excluded := i.excludedCycles()
	for _, record := range i.QMetrics.Records {
		if slices.Contains(excluded, record.Cycle) {
			continue
		}
		bases += record.BaseCount()
	}
	return bases
}

// LaneYield returns the yield per lane in bases for the sequencing run.
func (i Interop) LaneYield() map[int]int {
	laneYield := make(map[int]int, i.RunInfo.Flowcell.Lanes)
	excluded := i.excludedCycles()
	for _, record := range i.QMetrics.Records {
		if slices.Contains(excluded, record.Cycle) {
			continue
		}
		laneYield[record.Lane] += record.BaseCount()
	}
	return laneYield
}

// cycleToRead takes a sequencing cycle number and returns the read number that corresponds to that cycle.
func (i Interop) cycleToRead(cycle int) int {
	cumulativeCycles := 0
	for _, read := range i.RunInfo.Reads {
		cumulativeCycles += read.Cycles
		if cycle <= cumulativeCycles {
			return read.Number
		}
	}
	return 0
}

type RunSummary struct {
	Yield           int     `bson:"yield" json:"yield"`
	PercentQ30      float64 `bson:"percent_q30,omitempty" json:"percent_q30,omitempty"`
	PercentAligned  float64 `bson:"percent_aligned,omitempty" json:"percent_aligned"`
	ErrorRate       float64 `bson:"error_rate,omitempty" json:"error_rate,omitempty"`
	PercentOccupied float64 `bson:"percent_occupied,omitempty" json:"percent_occupied,omitempty"`
}

func (i Interop) RunSummary() (rs RunSummary) {
	return RunSummary{
		Yield:           i.TotalYield(),
		PercentQ30:      i.RunPercentQ30(),
		PercentAligned:  i.TileMetrics.PercentAligned(),
		ErrorRate:       i.RunErrorRate(),
		PercentOccupied: i.RunPercentOccupied(),
	}
}

type LaneSummary struct {
	Yield     int
	ErrorRate float64
}

func (i Interop) LaneSummary() map[int]LaneSummary {
	ls := make(map[int]LaneSummary)
	laneErrors := i.LaneErrorRate()
	for lane, e := range laneErrors {
		if math.IsNaN(e) {
			continue
		}
		lse := ls[lane]
		lse.ErrorRate = e
		ls[lane] = lse
	}
	for lane, yield := range i.LaneYield() {
		lsy := ls[lane]
		lsy.Yield = yield
		ls[lane] = lsy
	}
	return ls
}

type TileSummaryRecord struct {
	LT
	Name            string
	ClusterCount    int     `bson:"cluster_count" json:"cluster_count"`
	PFClusterCount  int     `bson:"pf_cluster_count" json:"pf_cluster_count"`
	PercentOccupied float64 `bson:"percent_occupied" json:"percent_occupied"`
	PercentPF       float64 `bson:"percent_pf" json:"percent_pf"`
	PercentQ30      float64 `bson:"percent_q30" json:"percent_q30"`
	PercentAligned  float64 `bson:"percent_aligned" json:"percent_aligned"`
	ErrorRate       float64 `bson:"error_rate" json:"error_rate"`
}

func (i Interop) TileSummary() []TileSummaryRecord {
	tiles := make(map[string]TileSummaryRecord)
	for _, record := range i.TileMetrics.Records {
		name := record.LT.TileName()
		ts, ok := tiles[name]
		if !ok {
			ts.LT = record.LT
			ts.Name = record.TileName()
		}
		ts.PFClusterCount = record.PfClusterCount
		ts.ClusterCount = record.ClusterCount
		ts.PercentPF = 100 * float64(record.PfClusterCount) / float64(record.ClusterCount)
		percAligned := 0.0
		for _, v := range record.PercentAligned {
			percAligned += v
		}
		percAligned /= float64(len(record.PercentAligned))
		ts.PercentAligned = percAligned
		tiles[name] = ts
	}

	for name, errorRate := range i.TileErrorRate() {
		ts := tiles[name]
		ts.ErrorRate = errorRate
		tiles[name] = ts
	}

	for name, q30 := range i.TilePercentQ30() {
		ts := tiles[name]
		ts.PercentQ30 = q30
		tiles[name] = ts
	}

	for _, record := range i.ExtendedTileMetrics.Records {
		name := record.LT.TileName()
		ts := tiles[name]
		ts.PercentOccupied = 100 * float64(record.OccupiedClusters) / float64(tiles[name].ClusterCount)
		tiles[name] = ts
	}

	tileSummaries := make([]TileSummaryRecord, 0, i.RunInfo.Flowcell.Tiles)
	for _, ts := range tiles {
		tileSummaries = append(tileSummaries, ts)
	}
	return tileSummaries
}

type InteropSummary struct {
	RunId       string              `bson:"run_id" json:"run_id"`
	Platform    string              `bson:"platform" json:"platform"`
	Flowcell    string              `bson:"flowcell" json:"flowcell"`
	Date        time.Time           `bson:"date" json:"date"`
	RunSummary  RunSummary          `bson:"run_summary" json:"run_summary"`
	TileSummary []TileSummaryRecord `bson:"tile_summary" json:"tile_summary"`
	LaneSummary map[int]LaneSummary `bson:"lane_summary" json:"lane_summary"`
}

func (i Interop) Summarise() InteropSummary {
	return InteropSummary{
		RunId:       i.RunInfo.RunId,
		Platform:    i.RunInfo.Platform,
		Flowcell:    i.RunInfo.FlowcellName,
		Date:        i.RunInfo.Date,
		RunSummary:  i.RunSummary(),
		LaneSummary: i.LaneSummary(),
		TileSummary: i.TileSummary(),
	}
}

// TotalFracOccupied returns the fraction of occupied clusters across the whole flow cell.
func (i Interop) TotalFracOccupied() float64 {
	nClusters := i.TileMetrics.Clusters()
	nOccupiedClusters := i.ExtendedTileMetrics.OccupiedClusters()
	return float64(nOccupiedClusters) / float64(nClusters)
}

// TotalFracOccupied returns the fraction of occupied clusters per lane.
func (i Interop) LaneFracOccupied() map[int]float64 {
	laneCount := i.TileMetrics.LaneClusters()
	laneOccupiedCount := i.ExtendedTileMetrics.LaneOccupiedClusters()
	laneFracOccupied := make(map[int]float64)
	for lane := range laneCount {
		laneFracOccupied[lane] = float64(laneOccupiedCount[lane]) / float64(laneCount[lane])
	}
	return laneFracOccupied
}

// TilePercentQ30 calculates the Q30 across all usable cycles for each tile on the flow cell.
// The return value is a map with tile names as keys and the percent Q30 as values. If Q30
// is not represented in the bin definitions, nil will be returned.
func (i Interop) TilePercentQ30() map[string]float64 {
	q30bin := -1
	for i, b := range i.QMetrics.BinDefs {
		if b.Value >= 30 {
			q30bin = i
			break
		}
	}

	if q30bin == -1 {
		return nil
	}

	tileQ30 := make(map[string]float64)
	counts := make(map[string]int)
	excluded := i.excludedCycles()
	for _, record := range i.QMetrics.Records {
		if slices.Contains(excluded, record.Cycle) {
			continue
		}
		name := record.TileName()
		tileQCounts := 0
		for bi := q30bin; bi < int(i.QMetrics.Bins); bi++ {
			tileQCounts += record.Histogram[bi]
		}
		tileQ30[name] += 100 * float64(tileQCounts) / float64(record.BaseCount())
		counts[name]++
	}

	for name := range tileQ30 {
		tileQ30[name] /= float64(counts[name])
	}
	return tileQ30
}

// ReadPercentQ30 calculates the fraction of passing filter clusters with a Q score >= 30
// for each lane on the flowcell. It is calculated by first getting the Q30 fraction
// for each read in each lane and then averaging these for each lane.
func (i Interop) ReadPercentQ30() map[int]map[int]float64 {
	pfCount := make(map[int]map[int]int)
	totalCount := make(map[int]map[int]int)
	q30bin := -1
	for i, b := range i.QMetrics.BinDefs {
		if b.Value >= 30 {
			q30bin = i
			break
		}
	}

	if q30bin == -1 {
		return nil
	}

	excluded := i.excludedCycles()
	for _, record := range i.QMetrics.Records {
		if slices.Contains(excluded, record.Cycle) {
			continue
		}
		read := i.cycleToRead(record.Cycle)
		if _, ok := pfCount[read]; !ok {
			pfCount[read] = make(map[int]int)
			totalCount[read] = make(map[int]int)
		}
		for bi := q30bin; bi < int(i.QMetrics.Bins); bi++ {
			pfCount[read][record.Lane] += record.Histogram[bi]
		}
		totalCount[read][record.Lane] += record.BaseCount()
	}

	readQ30 := make(map[int]map[int]float64)
	for read := range pfCount {
		for lane := range pfCount[read] {
			if _, ok := readQ30[read]; !ok {
				readQ30[read] = make(map[int]float64)
			}
			readQ30[read][lane] = 100 * float64(pfCount[read][lane]) / float64(totalCount[read][lane])
		}
	}
	return readQ30
}

// LanePercentQ30 calculates the fraction of passing filter clusters with a Q score >= 30
// for each lane on the flowcell. It is calculated by first getting the Q30 fraction
// for each read in each lane and then averaging these for each lane.
func (i Interop) LanePercentQ30() map[int]float64 {
	laneQ30 := make(map[int]float64)
	readQ30 := i.ReadPercentQ30()
	for read := range readQ30 {
		for lane, q30 := range readQ30[read] {
			laneQ30[lane] += q30
		}
	}
	for lane := range laneQ30 {
		laneQ30[lane] = laneQ30[lane] / float64(len(readQ30))
	}
	return laneQ30
}

// RunPercentQ30 returns the fraction of clusters with a Q score >= 30 for all
// passing filter clusters for a flow cell. It is calculated by summing
// up the number of clusters with a Q score >= 30 across all tiles.
func (i Interop) RunPercentQ30() float64 {
	pfCount := 0
	totalCount := 0
	q30bin := -1
	for i, b := range i.QMetrics.BinDefs {
		if b.Value >= 30 {
			q30bin = i
			break
		}
	}

	if q30bin == -1 {
		return 0.0
	}

	excluded := i.excludedCycles()
	for _, record := range i.QMetrics.Records {
		if slices.Contains(excluded, record.Cycle) {
			continue
		}
		for bi := q30bin; bi < int(i.QMetrics.Bins); bi++ {
			pfCount += record.Histogram[bi]
		}
		totalCount += record.BaseCount()
	}

	return 100 * float64(pfCount) / float64(totalCount)
}

// TileErrorRate calculates the average error for all tiles over usable cycles for the whole flow cell.
func (i Interop) TileErrorRate() map[string]float64 {
	tileErrors := make(map[string]float64)
	counts := make(map[string]int)
	excluded := i.excludedCycles()
	for _, record := range i.ErrorMetrics.Records {
		if slices.Contains(excluded, record.Cycle) {
			continue
		}
		name := record.TileName()
		tileErrors[name] += record.ErrorRate
		counts[name]++
	}
	for name := range tileErrors {
		tileErrors[name] /= float64(counts[name])
	}
	return tileErrors
}

// ReadErrorRate calculates the average error rate for reads, on a per lane basis. It works by first
// calculating the average error rate across all usable cycles for each tile. These are then averaged
// for each tile for a given lane. The return value is a nested map where the first key is the read number
// and the second key is the lane number.
func (i Interop) ReadErrorRate() map[int]map[int]float64 {
	readErrors := make(map[int]map[int]float64)
	counts := make(map[int]map[int]int)
	excluded := i.excludedCycles()
	for _, record := range i.ErrorMetrics.Records {
		if slices.Contains(excluded, record.Cycle) {
			continue
		}
		read := i.cycleToRead(record.Cycle)
		if _, ok := readErrors[read]; !ok {
			readErrors[read] = make(map[int]float64)
			counts[read] = make(map[int]int)
		}
		readErrors[read][record.Lane] += record.ErrorRate
		counts[read][record.Lane]++
	}
	for read := range readErrors {
		for lane := range readErrors[read] {
			readErrors[read][lane] /= float64(counts[read][lane])
		}
	}
	return readErrors
}

// LaneErrorRate calculates the average error rate for each lane of the flow cell. It is calculated
// from the lane averages produced by `Interop.ReadErrorRate`.
func (i Interop) LaneErrorRate() map[int]float64 {
	laneErrors := make(map[int]float64)
	counts := make(map[int]int)
	readErrors := i.ReadErrorRate()
	for read := range readErrors {
		for lane, e := range readErrors[read] {
			laneErrors[lane] += e
			counts[lane]++
		}
	}
	for lane := range laneErrors {
		laneErrors[lane] /= float64(counts[lane])
	}
	return laneErrors
}

// RunErrorRate calculates the average error rate for each lane of the flow cell. It is the average
// of the lane error rates from `Interop.LaneErrorRate`.
func (i Interop) RunErrorRate() float64 {
	errorRate := 0.0
	laneError := i.LaneErrorRate()
	for _, e := range laneError {
		errorRate += e
	}
	return errorRate / float64(len(laneError))
}

func (i Interop) RunPercentOccupied() float64 {
	return 100 * float64(i.ExtendedTileMetrics.OccupiedClusters()) / float64(i.TileMetrics.Clusters())
}
