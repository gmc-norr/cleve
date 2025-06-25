package interop

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type IndexMetrics struct {
	Version uint8
	Records []IndexMetricRecord
}

type IndexMetricRecord struct {
	LT
	Read         int
	IndexName    string
	SampleName   string
	ProjectName  string
	ClusterCount int
}

func (m IndexMetrics) TotalYield() int {
	sum := 0
	for _, r := range m.Records {
		sum += r.ClusterCount
	}
	return sum
}

func (m IndexMetrics) SampleYield() map[string]int {
	yield := make(map[string]int)
	for _, r := range m.Records {
		yield[r.SampleName] += r.ClusterCount
	}
	return yield
}

type indexMetricRecordV1 struct {
	lt1
	Read              uint16
	IndexNameLength   uint16
	IndexName         []byte
	ClusterCount      uint32
	SampleNameLength  uint16
	SampleName        []byte
	ProjectNameLength uint16
	ProjectName       []byte
}

type indexMetricRecordV2 struct {
	lt2
	Read              uint16
	IndexNameLength   uint16
	IndexName         []byte
	ClusterCount      uint64
	SampleNameLength  uint16
	SampleName        []byte
	ProjectNameLength uint16
	ProjectName       []byte
}

func parseIndexMetricsV1(r io.Reader, im *IndexMetrics) error {
	for {
		rec := indexMetricRecordV1{}
		err := binary.Read(r, binary.LittleEndian, &rec.lt1)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if err := binary.Read(r, binary.LittleEndian, &rec.Read); err != nil {
			return err
		}
		if err := binary.Read(r, binary.LittleEndian, &rec.IndexNameLength); err != nil {
			return err
		}
		rec.IndexName = make([]byte, rec.IndexNameLength)
		if err := binary.Read(r, binary.LittleEndian, &rec.IndexName); err != nil {
			return err
		}
		if err := binary.Read(r, binary.LittleEndian, &rec.ClusterCount); err != nil {
			return err
		}
		if err := binary.Read(r, binary.LittleEndian, &rec.SampleNameLength); err != nil {
			return err
		}
		rec.SampleName = make([]byte, rec.SampleNameLength)
		if err := binary.Read(r, binary.LittleEndian, &rec.SampleName); err != nil {
			return err
		}
		if err := binary.Read(r, binary.LittleEndian, &rec.ProjectNameLength); err != nil {
			return err
		}
		rec.ProjectName = make([]byte, rec.ProjectNameLength)
		if err := binary.Read(r, binary.LittleEndian, &rec.ProjectName); err != nil {
			return err
		}

		im.Records = append(im.Records, IndexMetricRecord{
			LT:           rec.normalize(),
			Read:         int(rec.Read),
			IndexName:    string(rec.IndexName),
			SampleName:   string(rec.SampleName),
			ProjectName:  string(rec.ProjectName),
			ClusterCount: int(rec.ClusterCount),
		})
	}
	return nil
}

func parseIndexMetricsV2(r io.Reader, im *IndexMetrics) error {
	for {
		rec := indexMetricRecordV2{}
		err := binary.Read(r, binary.LittleEndian, &rec.lt2)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if err := binary.Read(r, binary.LittleEndian, &rec.Read); err != nil {
			return err
		}
		if err := binary.Read(r, binary.LittleEndian, &rec.IndexNameLength); err != nil {
			return err
		}
		rec.IndexName = make([]byte, rec.IndexNameLength)
		if err := binary.Read(r, binary.LittleEndian, &rec.IndexName); err != nil {
			return err
		}
		if err := binary.Read(r, binary.LittleEndian, &rec.ClusterCount); err != nil {
			return err
		}
		if err := binary.Read(r, binary.LittleEndian, &rec.SampleNameLength); err != nil {
			return err
		}
		rec.SampleName = make([]byte, rec.SampleNameLength)
		if err := binary.Read(r, binary.LittleEndian, &rec.SampleName); err != nil {
			return err
		}
		if err := binary.Read(r, binary.LittleEndian, &rec.ProjectNameLength); err != nil {
			return err
		}
		rec.ProjectName = make([]byte, rec.ProjectNameLength)
		if err := binary.Read(r, binary.LittleEndian, &rec.ProjectName); err != nil {
			return err
		}

		im.Records = append(im.Records, IndexMetricRecord{
			LT:           rec.normalize(),
			Read:         int(rec.Read),
			IndexName:    string(rec.IndexName),
			SampleName:   string(rec.SampleName),
			ProjectName:  string(rec.ProjectName),
			ClusterCount: int(rec.ClusterCount),
		})
	}
	return nil
}

func ReadIndexMetrics(path string) (IndexMetrics, error) {
	im := IndexMetrics{}
	f, err := os.Open(path)
	if err != nil {
		return im, err
	}
	defer func() { _ = f.Close() }()

	r := bufio.NewReader(f)

	err = binary.Read(r, binary.LittleEndian, &im.Version)
	if err != nil {
		return im, err
	}

	switch im.Version {
	case 1:
		err = parseIndexMetricsV1(r, &im)
	case 2:
		err = parseIndexMetricsV2(r, &im)
	default:
		err = fmt.Errorf("unsupported index metrics version: %d", im.Version)
	}

	return im, err
}
