package interop

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
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
		if slices.Contains(excluded, record.Cycle()) {
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
		if slices.Contains(excluded, record.Cycle()) {
			continue
		}
		laneYield[record.Lane()] += record.BaseCount()
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

// LaneErrorRate calculates the average error rate for lanes, on a per read basis. It works by first
// calculating the average error rate across all usable cycles for each tile. These are then averaged
// for each tile for a given read. The return value is a nested map where the first key is the read number
// and the second key is the lane number.
func (i Interop) LaneErrorRate() map[int]map[int]float64 {
	errorRates := make(map[int]map[int]float64, len(i.RunInfo.Reads))
	for _, read := range i.RunInfo.Reads {
		errorRates[read.Number] = make(map[int]float64)
	}

	excluded := i.excludedCycles()

	tileMeans := make(map[int]map[[2]int]*RunningAverage)
	for _, read := range i.RunInfo.Reads {
		tileMeans[read.Number] = make(map[[2]int]*RunningAverage)
	}

	// First calculate the mean error for each tile per read per lane
	for _, record := range i.ErrorMetrics.Records {
		if slices.Contains(excluded, record.Cycle) {
			continue
		}
		read := i.cycleToRead(record.Cycle)
		key := [2]int{record.Lane, record.Tile}
		avg, ok := tileMeans[read][key]
		if !ok {
			avg = &RunningAverage{}
		}
		avg.Add(record.ErrorRate)
		tileMeans[read][key] = avg
	}

	// Calculate the mean error across all tiles per read per lane
	for read := range tileMeans {
		for lane := 1; lane <= i.RunInfo.Flowcell.Lanes; lane++ {
			sum := 0.0
			n := 0
			for key, x := range tileMeans[read] {
				if key[0] != lane {
					continue
				}
				sum += x.Average
				n++
			}
			if n == 0 {
				// Set NaN explicitly
				errorRates[read][lane] = math.NaN()
			} else {
				errorRates[read][lane] = sum / float64(n)
			}
		}
	}

	return errorRates
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
