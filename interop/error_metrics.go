package interop

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
)

type ErrorMetrics interface {
	InteropFile
	InteropRecordHolder
	LaneErrorRate() StatsMap1
	TileErrorRate() StatsMap2
	CycleErrorRate() StatsMap3
	ReadErrorRate(ReadConfig) StatsMap2
}

type ErrorMetricRecord6 struct {
	Lane            uint16
	Tile            uint32
	Cycle           uint16
	ErrorRate       float32
	FractionTrimmed []float32
}

func NewErrorMetricRecord6(nAdapters int) ErrorMetricRecord6 {
	return ErrorMetricRecord6{FractionTrimmed: make([]float32, nAdapters)}
}

func (record *ErrorMetricRecord6) Parse(r io.Reader, nAdapters uint16) error {
	if err := binary.Read(r, binary.LittleEndian, &record.Lane); err != nil {
		if err == io.EOF {
			return err
		}
		return fmt.Errorf("%s when parsing lane for error metric v6 record", err.Error())
	}

	if err := binary.Read(r, binary.LittleEndian, &record.Tile); err != nil {
		return fmt.Errorf("%s when parsing tile for error metric v6 record", err.Error())
	}

	if err := binary.Read(r, binary.LittleEndian, &record.Cycle); err != nil {
		return fmt.Errorf("%s when parsing cycle for error metric v6 record", err.Error())
	}

	if err := binary.Read(r, binary.LittleEndian, &record.ErrorRate); err != nil {
		return fmt.Errorf("%s when parsing ErrorRate for error metric v6 record", err.Error())
	}

	record.FractionTrimmed = make([]float32, nAdapters)
	if err := binary.Read(r, binary.LittleEndian, &record.FractionTrimmed); err != nil {
		return fmt.Errorf("%s when parsing FractionTrimmed for error metric v6 record", err.Error())
	}

	return nil
}

func (record ErrorMetricRecord6) Type() string {
	return "error_metric"
}

type ErrorMetrics6 struct {
	InteropHeader
	NumAdapter         uint16
	AdapterBaseCount   uint16
	AdapterBases       []uint8
	ErrorMetricRecords []InteropRecord
}

func (m ErrorMetrics6) Records() []InteropRecord {
	return m.ErrorMetricRecords
}

func (m ErrorMetrics6) CycleErrorRate() StatsMap3 {
	summaries := make(StatsMap3)
	for _, r := range m.ErrorMetricRecords {
		record := r.(ErrorMetricRecord6)
		lane := int(record.Lane)
		tile := int(record.Tile)
		cycle := int(record.Cycle)
		if _, ok := summaries[lane]; !ok {
			summaries[lane] = make(StatsMap2)
		}
		if _, ok := summaries[lane][tile]; !ok {
			summaries[lane][tile] = make(StatsMap1)
		}
		if _, ok := summaries[lane][tile][cycle]; !ok {
			summaries[lane][tile][cycle] = &RunningSummary[float64]{}
		}
		summaries[lane][tile][cycle].Push(float64(record.ErrorRate))
	}

	return summaries
}

func (m ErrorMetrics6) TileErrorRate() StatsMap2 {
	summaries := make(StatsMap2)
	for _, r := range m.ErrorMetricRecords {
		record := r.(ErrorMetricRecord6)
		lane := int(record.Lane)
		tile := int(record.Tile)
		if _, ok := summaries[lane]; !ok {
			summaries[lane] = make(StatsMap1)
		}
		if _, ok := summaries[lane][tile]; !ok {
			summaries[lane][tile] = &RunningSummary[float64]{}
		}
		summaries[lane][tile].Push(float64(record.ErrorRate))
	}

	return summaries
}

func (m ErrorMetrics6) LaneErrorRate() StatsMap1 {
	summaries := make(StatsMap1)
	return summaries
}

func (m ErrorMetrics6) ReadErrorRate(readConfig ReadConfig) StatsMap2 {
	// Summarise the error rate for each lane/tile/read combination
	tileSummaries := make(StatsMap3)
	for _, r := range m.ErrorMetricRecords {
		record := r.(ErrorMetricRecord6)
		if math.IsNaN(float64(record.ErrorRate)) {
			continue
		}
		lane := int(record.Lane)
		cycle := int(record.Cycle)
		tile := int(record.Tile)
		read := readConfig.CycleToRead(cycle)
		if _, ok := tileSummaries[lane]; !ok {
			tileSummaries[lane] = make(StatsMap2)
		}
		if _, ok := tileSummaries[lane][tile]; !ok {
			tileSummaries[lane][tile] = make(StatsMap1)
		}
		if _, ok := tileSummaries[lane][tile][read]; !ok {
			tileSummaries[lane][tile][read] = &RunningSummary[float64]{}
		}
		tileSummaries[lane][tile][read].Push(float64(record.ErrorRate))
	}

	// Summarise the stats for each lane/read
	readSummaries := make(StatsMap2)
	for lane, lanestats := range tileSummaries {
		if _, ok := readSummaries[lane]; !ok {
			readSummaries[lane] = make(StatsMap1)
		}
		for _, tilestats := range lanestats {
			for read, readstats := range tilestats {
				if _, ok := readSummaries[lane][read]; !ok {
					readSummaries[lane][read] = &RunningSummary[float64]{}
				}
				readSummaries[lane][read].Push(readstats.Mean)
			}
		}
	}

	return readSummaries
}

func (m *ErrorMetrics6) Parse(r io.Reader) error {
	if err := m.InteropHeader.Parse(r); err != nil {
		return err
	}

	if m.Version != 6 {
		return fmt.Errorf("expected version 6, got version %d", m.Version)
	}

	if err := binary.Read(r, binary.LittleEndian, &m.NumAdapter); err != nil {
		return fmt.Errorf("%s when parsing NumAdapter", err.Error())
	}

	if err := binary.Read(r, binary.LittleEndian, &m.AdapterBaseCount); err != nil {
		return fmt.Errorf("%s when parsing AdapterBaseCount", err.Error())
	}

	m.AdapterBases = make([]uint8, m.NumAdapter*m.AdapterBaseCount)
	if err := binary.Read(r, binary.LittleEndian, &m.AdapterBases); err != nil {
		return fmt.Errorf("%s when parsing AdapterBases", err.Error())
	}

	for {
		record := ErrorMetricRecord6{}
		if err := record.Parse(r, m.NumAdapter); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		m.ErrorMetricRecords = append(m.ErrorMetricRecords, record)
	}
}

func ParseErrorMetrics(filename string) (ErrorMetrics, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	r := bufio.NewReader(f)
	
	m := ErrorMetrics6{}
	if err := m.Parse(r); err != nil {
		return m, err
	}

	return m, nil
}
