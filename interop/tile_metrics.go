package interop

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type TileMetrics interface {
	InteropFile
	InteropRecordHolder
	Parse(io.Reader) error
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
	return "record"
}

type TileMetrics2 struct {
	InteropHeader
	TileMetricRecords []InteropRecord
}

func (m TileMetrics2) Records() []InteropRecord {
	return m.TileMetricRecords
}

func (m TileMetrics2) GetRecordSize() uint8 {
	return m.RecordSize
}

func (m TileMetrics2) GetVersion() uint8 {
	return m.Version
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

func (m TileMetrics3) Records() []InteropRecord {
	return m.TileMetricRecords
}

func (m TileMetrics3) GetRecordSize() uint8 {
	return m.RecordSize
}

func (m TileMetrics3) GetVersion() uint8 {
	return m.Version
}

func ParseTileMetrics(filename string) (TileMetrics, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := bufio.NewReader(f)

	// Try version 3 first
	// var m TileMetrics
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

	m2 := &TileMetrics2{}
	err = m2.Parse(r)

	if err == nil {
		return m2, nil
	}

	return nil, fmt.Errorf("file is not tile metrics v2 or v3: %s", filename)
}
