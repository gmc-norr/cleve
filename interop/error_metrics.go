package interop

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type ErrorMetrics struct {
	Header
	Records []ErrorMetricRecord
}

type ErrorMetricRecord struct {
	LTC
	ErrorRate float64
}

type errorMetricRecordV3 struct {
	ltc1
	ErrorRate    float32
	PerfectReads uint32
	Reads1Error  uint32
	Reads2Error  uint32
	Reads3Error  uint32
	Reads4Error  uint32
}

func parseErrorMetricRecordsV3(r io.Reader, em *ErrorMetrics) error {
	for {
		rec := errorMetricRecordV3{}
		err := binary.Read(r, binary.LittleEndian, &rec)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		em.Records = append(em.Records, ErrorMetricRecord{
			LTC:       rec.ltc1.normalize(),
			ErrorRate: float64(rec.ErrorRate),
		})
	}
	return nil
}

func parseErrorMetricsV3(r io.Reader, em *ErrorMetrics) error {
	return parseErrorMetricRecordsV3(r, em)
}

type errorMetricsV6 struct {
	AdapterCount     uint16
	AdapterBaseCount uint16
	AdapterBases     []uint8
}

type errorMetricRecordV6 struct {
	ltc2
	ErrorRate               float32
	FracReadsAdapterTrimmed []float32
}

func parseErrorMetricRecordsV6(r io.Reader, adapterCount int, em *ErrorMetrics) error {
	for {
		rec := errorMetricRecordV6{}
		rec.FracReadsAdapterTrimmed = make([]float32, adapterCount)
		if err := binary.Read(r, binary.LittleEndian, &rec.ltc2); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if err := binary.Read(r, binary.LittleEndian, &rec.ErrorRate); err != nil {
			return err
		}
		if err := binary.Read(r, binary.LittleEndian, &rec.FracReadsAdapterTrimmed); err != nil {
			return err
		}
		em.Records = append(em.Records, ErrorMetricRecord{
			LTC:       rec.ltc2.normalize(),
			ErrorRate: float64(rec.ErrorRate),
		})
	}
	return nil
}

func parseErrorMetricsV6(r io.Reader, em *ErrorMetrics) error {
	rawMetrics := errorMetricsV6{}
	err := binary.Read(r, binary.LittleEndian, &rawMetrics.AdapterCount)
	if err != nil {
		return err
	}
	err = binary.Read(r, binary.LittleEndian, &rawMetrics.AdapterBaseCount)
	if err != nil {
		return err
	}
	rawMetrics.AdapterBases = make([]uint8, rawMetrics.AdapterCount*rawMetrics.AdapterBaseCount)
	err = binary.Read(r, binary.LittleEndian, &rawMetrics.AdapterBases)
	if err != nil {
		return err
	}
	return parseErrorMetricRecordsV6(r, int(rawMetrics.AdapterCount), em)
}

func ReadErrorMetrics(path string) (ErrorMetrics, error) {
	em := ErrorMetrics{}
	f, err := os.Open(path)
	if err != nil {
		return em, err
	}
	defer f.Close()
	r := bufio.NewReader(f)

	err = binary.Read(r, binary.LittleEndian, &em.Header)
	if err != nil {
		return em, err
	}

	switch em.Version {
	case 3:
		err = parseErrorMetricsV3(r, &em)
	case 6:
		err = parseErrorMetricsV6(r, &em)
	default:
		err = fmt.Errorf("invalid error metrics version: %d", em.Version)
	}

	return em, err
}
