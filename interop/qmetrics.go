package interop

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

type BinDefinition struct {
	Low   uint8
	High  uint8
	Value uint8
}

type QMetrics struct {
	Header
	HasBins bool
	Bins    uint8
	BinDefs []BinDefinition
	Records []QMetricRecord
}

func (qm QMetrics) TotalYield() int {
	bases := 0
	for _, record := range qm.Records {
		bases += record.BaseCount()
	}
	return bases
}

type QMetricRecord interface {
	Tile() int
	Lane() int
	Cycle() int
	BaseCount() int
}

type QMetricRecordV4 = QMetricRecordV6

type QMetricRecordV6 struct {
	ltc1
	Histogram []uint32
}

func (qmr QMetricRecordV6) Tile() int {
	return int(qmr.ltc1.Tile)
}

func (qmr QMetricRecordV6) Lane() int {
	return int(qmr.ltc1.Lane)
}

func (qmr QMetricRecordV6) Cycle() int {
	return int(qmr.ltc1.Cycle)
}

func (qmr QMetricRecordV6) BaseCount() int {
	c := 0
	for _, binCount := range qmr.Histogram {
		c += int(binCount)
	}
	return c
}

type QMetricRecordV7 struct {
	ltc2
	Histogram []uint32
}

func (qmr QMetricRecordV7) Tile() int {
	return int(qmr.ltc2.Tile)
}

func (qmr QMetricRecordV7) Lane() int {
	return int(qmr.ltc2.Lane)
}

func (qmr QMetricRecordV7) Cycle() int {
	return int(qmr.ltc2.Cycle)
}

func (qmr QMetricRecordV7) BaseCount() int {
	c := 0
	for _, binCount := range qmr.Histogram {
		c += int(binCount)
	}
	return c
}

func parseBinDefinitionV4(qm *QMetrics) error {
	qm.HasBins = false
	qm.Bins = 50
	qm.BinDefs = make([]BinDefinition, qm.Bins)
	return nil
}

func parseBinDefinitionV6(r io.Reader, qm *QMetrics) error {
	err := binary.Read(r, binary.LittleEndian, &qm.HasBins)
	if err != nil {
		return err
	}
	err = binary.Read(r, binary.LittleEndian, &qm.Bins)
	if err != nil {
		return err
	}
	qm.BinDefs = make([]BinDefinition, qm.Bins)
	for i := uint8(0); i < qm.Bins; i++ {
		err := binary.Read(r, binary.LittleEndian, &qm.BinDefs[i].Low)
		if err != nil {
			return err
		}
	}
	for i := uint8(0); i < qm.Bins; i++ {
		err := binary.Read(r, binary.LittleEndian, &qm.BinDefs[i].High)
		if err != nil {
			return err
		}
	}
	for i := uint8(0); i < qm.Bins; i++ {
		err := binary.Read(r, binary.LittleEndian, &qm.BinDefs[i].Value)
		if err != nil {
			return err
		}
	}
	return nil
}

func parseBinDefinitionV7(r io.Reader, qm *QMetrics) error {
	if err := binary.Read(r, binary.LittleEndian, &qm.HasBins); err != nil {
		return err
	}

	if !qm.HasBins {
		qm.Bins = 50
		return nil
	}

	if err := binary.Read(r, binary.LittleEndian, &qm.Bins); err != nil {
		return err
	}

	qm.BinDefs = make([]BinDefinition, qm.Bins)
	for i := 0; i < int(qm.Bins); i++ {
		bd := BinDefinition{}
		if err := binary.Read(r, binary.LittleEndian, &bd); err != nil {
			return err
		}
		qm.BinDefs[i] = bd
	}
	return nil
}

func parseQMetricRecords4(r io.Reader, qm *QMetrics) error {
	for {
		record := QMetricRecordV4{}
		if err := binary.Read(r, binary.LittleEndian, &record.ltc1); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		record.Histogram = make([]uint32, qm.Bins)
		err := binary.Read(r, binary.LittleEndian, &record.Histogram)
		if err != nil {
			return err
		}
		qm.Records = append(qm.Records, record)
	}
	return nil
}

func parseQMetricRecords6(r io.Reader, qm *QMetrics) error {
	for {
		record := QMetricRecordV6{}
		if err := binary.Read(r, binary.LittleEndian, &record.ltc1); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return err
		}
		record.Histogram = make([]uint32, qm.Bins)
		err := binary.Read(r, binary.LittleEndian, &record.Histogram)
		if err != nil {
			return err
		}
		qm.Records = append(qm.Records, record)
	}
	return nil
}

func parseQMetricRecords7(r io.Reader, qm *QMetrics) error {
	for {
		record := QMetricRecordV7{}
		if err := binary.Read(r, binary.LittleEndian, &record.ltc2); err != nil {
			// A valid file should reach EOF here
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		record.Histogram = make([]uint32, qm.Bins)
		err := binary.Read(r, binary.LittleEndian, &record.Histogram)
		if err != nil {
			return err
		}
		qm.Records = append(qm.Records, record)
	}
	return nil
}

func parseQMetrics(r io.Reader) (QMetrics, error) {
	var err error
	qm := QMetrics{}
	qm.Header, err = parseHeader(r)
	if err != nil {
		return qm, err
	}

	switch qm.Version {
	case 4:
		err = parseBinDefinitionV4(&qm)
	case 6:
		err = parseBinDefinitionV6(r, &qm)
	case 7:
		err = parseBinDefinitionV7(r, &qm)
	default:
		err = fmt.Errorf("unsupported qmetrics version: %d", qm.Version)
	}

	if err != nil {
		return qm, err
	}

	switch qm.Version {
	case 4:
		err = parseQMetricRecords4(r, &qm)
	case 6:
		err = parseQMetricRecords6(r, &qm)
	case 7:
		err = parseQMetricRecords7(r, &qm)
	}

	return qm, err
}

func ReadQMetrics(filename string) (QMetrics, error) {
	f, err := os.Open(filename)
	if err != nil {
		return QMetrics{}, err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	return parseQMetrics(r)
}
