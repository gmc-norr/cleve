package interop

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type QBins = []QBin
type QBin struct {
	Low   uint8
	High  uint8
	Value uint8
}

type QBinConfig struct {
	BinCount uint8
	Bins     QBins
}

type QRecords = []QRecord
type QRecord struct {
	Lane      uint16
	Tile      Tile
	Cycle     uint16
	Histogram []uint32
}

type QMetrics struct {
	InteropHeader
	HasBins bool
	QBinConfig
	Records QRecords
}

func (m *QMetrics) IsSupported() bool {
	return m.Version == 5 || m.Version == 6 || m.Version == 7
}

func (m *QMetrics) ParseBinnedRecord(r io.Reader) error {
	record := QRecord{}
	if err := binary.Read(r, binary.LittleEndian, &record.Lane); err != nil {
		return err
	}
	switch m.InteropHeader.Version {
	case 5, 6:
		record.Tile = new(Tile16)
	case 7:
		record.Tile = new(Tile32)
	}
	if err := record.Tile.Parse(r); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &record.Cycle); err != nil {
		return err
	}

	record.Histogram = make([]uint32, m.BinCount)
	if m.Version == 5 {
		sparseHistogram := make([]uint32, 50)
		if err := binary.Read(r, binary.LittleEndian, &sparseHistogram); err != nil {
			return err
		}
		for i, bin := range m.Bins {
			record.Histogram[i] = sparseHistogram[bin.Value-1]
		}
	} else {
		if err := binary.Read(r, binary.LittleEndian, &record.Histogram); err != nil {
			return err
		}
	}
	m.Records = append(m.Records, record)
	return nil
}

func (m *QMetrics) ParseBinnedRecords(r io.Reader) error {
	for {
		if err := m.ParseBinnedRecord(r); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	return nil
}

func (m *QMetrics) ParseRecord(r io.Reader) error {
	record := QRecord{}
	if err := binary.Read(r, binary.LittleEndian, &record.Lane); err != nil {
		return err
	}
	switch m.Version {
	case 5, 6:
		record.Tile = new(Tile16)
	case 7:
		record.Tile = new(Tile32)
	}
	if err := record.Tile.Parse(r); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &record.Cycle); err != nil {
		return err
	}
	record.Histogram = make([]uint32, 50)
	if err := binary.Read(r, binary.LittleEndian, &record.Histogram); err != nil {
		return err
	}
	m.Records = append(m.Records, record)
	return nil
}

func (m *QMetrics) ParseRecords(r io.Reader) error {
	for {
		if err := m.ParseRecord(r); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	return nil
}

func (m *QMetrics) ParseBins(r io.Reader) error {
	switch m.Version {
	case 7:
		for range m.BinCount {
			var bin QBin
			if err := binary.Read(r, binary.LittleEndian, &bin); err != nil {
				return err
			}
			m.Bins = append(m.Bins, bin)
		}
	case 5, 6:
		v6bins := make([]uint8, m.BinCount*3)
		if err := binary.Read(r, binary.LittleEndian, &v6bins); err != nil {
			return err
		}
		for i := range m.BinCount {
			m.Bins = append(m.Bins, QBin{
				Low:   v6bins[i],
				High:  v6bins[m.BinCount+i],
				Value: v6bins[2*m.BinCount+i],
			})
		}
	}
	return nil
}

func (m QMetrics) TotalPercentOverQ(threshold uint8) float32 {
	maxCycles := m.MaxCyclePerLane()
	all := 0
	passed := 0
	for _, r := range m.Records {
		if r.Cycle >= maxCycles[r.Lane] {
			continue
		}
		for i, b := range m.Bins {
			if b.Low >= threshold {
				passed += int(r.Histogram[i])
			}
			all += int(r.Histogram[i])
		}
	}
	return 100 * float32(passed) / float32(all)
}

func (m QMetrics) MaxCyclePerLane() map[uint16]uint16 {
	maxCycles := make(map[uint16]uint16)
	for _, r := range m.Records {
		if r.Cycle > maxCycles[r.Lane] {
			maxCycles[r.Lane] = r.Cycle
		}
	}
	return maxCycles
}

func ParseQMetrics(filename string) (*QMetrics, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := bufio.NewReader(f)

	metrics := QMetrics{}

	if err := binary.Read(r, binary.LittleEndian, &metrics.Version); err != nil {
		return nil, err
	}

	if !metrics.IsSupported() {
		return nil, fmt.Errorf("unsupported version %d", metrics.Version)
	}

	if err := binary.Read(r, binary.LittleEndian, &metrics.RecordSize); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &metrics.HasBins); err != nil {
		return nil, err
	}

	if metrics.HasBins {
		if err := binary.Read(r, binary.LittleEndian, &metrics.BinCount); err != nil {
			return nil, err
		}
		if err := metrics.ParseBins(r); err != nil {
			return nil, err
		}
		if err := metrics.ParseBinnedRecords(r); err != nil {
			return nil, err
		}
	} else {
		if err := metrics.ParseRecords(r); err != nil {
			return nil, err
		}
	}

	return &metrics, nil
}
