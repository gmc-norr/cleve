package interop

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type ExtTileMetrics struct {
	Header
	Records []ExtTileMetricRecord
}

func (m ExtTileMetrics) OccupiedClusters() int {
	sum := 0
	for _, r := range m.Records {
		sum += r.OccupiedClusters
	}
	return sum
}

func (m ExtTileMetrics) LaneOccupiedClusters() map[int]int {
	sums := make(map[int]int)
	for _, r := range m.Records {
		sums[r.Lane] += r.OccupiedClusters
	}
	return sums
}

type ExtTileMetricRecord struct {
	LT
	OccupiedClusters int
}

type extTileMetricRecordV3 struct {
	lt2
	OccupiedCount float32
	LocX          float32
	LocY          float32
}

func parseExtendedTileMetricRecordsV3(r io.Reader, tm *ExtTileMetrics) error {
	tm.Records = make([]ExtTileMetricRecord, 0)
	for {
		rec := extTileMetricRecordV3{}
		err := binary.Read(r, binary.LittleEndian, &rec)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		tm.Records = append(tm.Records, ExtTileMetricRecord{
			LT:               rec.normalize(),
			OccupiedClusters: int(rec.OccupiedCount),
		})
	}
	return nil
}

func parseExtendedTileMetricsV3(r io.Reader, tm *ExtTileMetrics) error {
	return parseExtendedTileMetricRecordsV3(r, tm)
}

type extTileMetricRecordV1 struct {
	lt1
	Code  uint16
	Value float32
}

func parseExtendedTileMetricRecordsV1(r io.Reader, tm *ExtTileMetrics) error {
	tm.Records = make([]ExtTileMetricRecord, 0)
	for {
		rec := extTileMetricRecordV1{}
		err := binary.Read(r, binary.LittleEndian, &rec)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		switch rec.Code {
		case TileClusterCountOccupied:
			tm.Records = append(tm.Records, ExtTileMetricRecord{
				LT:               rec.normalize(),
				OccupiedClusters: int(rec.Value),
			})
		default:
			return fmt.Errorf("invalid tile code: %d", rec.Code)
		}
	}
	return nil
}

func parseExtendedTileMetricsV1(r io.Reader, tm *ExtTileMetrics) error {
	return parseExtendedTileMetricRecordsV1(r, tm)
}

func ReadExtendedTileMetrics(filename string) (ExtTileMetrics, error) {
	tm := ExtTileMetrics{}
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
	case 1:
		err = parseExtendedTileMetricsV1(r, &tm)
	case 3:
		err = parseExtendedTileMetricsV3(r, &tm)
	default:
		return tm, fmt.Errorf("unsupported extended tile metrics version: %d", tm.Version)
	}

	return tm, err
}
