package interop

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type TileCycle interface {
	NBases() int
}

type CorrectedIntensity struct {
	Header
	Records []TileCycle
}

type TileCycleV2 struct {
	ltc1
	AveIntensity      uint16
	AChannelIntensity uint16
	CChannelIntensity uint16
	GChannelIntensity uint16
	TChannelIntensity uint16
	AClusterIntensity uint16
	CClusterIntensity uint16
	GClusterIntensity uint16
	TClusterIntensity uint16
	NCount            uint32
	ACount            uint32
	CCount            uint32
	GCount            uint32
	TCount            uint32
	SNRatio           float32
}

func (r TileCycleV2) NBases() int {
	return int(r.NCount)
}

type TileCycleV3 struct {
	ltc1
	AIntensity uint16
	CIntensity uint16
	GIntensity uint16
	TIntensity uint16
	NCount     uint32
	ACount     uint32
	CCount     uint32
	GCount     uint32
	TCount     uint32
}

func (r TileCycleV3) NBases() int {
	return int(r.NCount)
}

type TileCycleV4 struct {
	ltc2
	NCount uint32
	ACount uint32
	CCount uint32
	GCount uint32
	TCount uint32
}

func (r TileCycleV4) NBases() int {
	return int(r.NCount)
}

func parseCorrectedIntensityRecordsV2(r io.Reader) ([]TileCycle, error) {
	var records []TileCycle
	for {
		record := TileCycleV2{}
		err := binary.Read(r, binary.LittleEndian, &record)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return records, err
		}
		records = append(records, record)
	}
	return records, nil
}

func parseCorrectedIntensityRecordsV3(r io.Reader) ([]TileCycle, error) {
	var records []TileCycle
	for {
		record := TileCycleV3{}
		err := binary.Read(r, binary.LittleEndian, &record)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return records, err
		}
		records = append(records, record)
	}
	return records, nil
}

func parseCorrectedIntensityRecordsV4(r io.Reader) ([]TileCycle, error) {
	var records []TileCycle
	for {
		record := TileCycleV4{}
		err := binary.Read(r, binary.LittleEndian, &record)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return records, err
		}
		records = append(records, record)
	}
	return records, nil
}

func ParseCorrectedIntensity(r io.Reader) (CorrectedIntensity, error) {
	ci := CorrectedIntensity{}
	err := binary.Read(r, binary.LittleEndian, &ci.Header)
	if err != nil {
		return ci, err
	}

	switch ci.Version {
	case 2:
		ci.Records, err = parseCorrectedIntensityRecordsV2(r)
	case 3:
		ci.Records, err = parseCorrectedIntensityRecordsV3(r)
	case 4:
		ci.Records, err = parseCorrectedIntensityRecordsV4(r)
	default:
		err = fmt.Errorf("unsupported corrected intensity version: %d", ci.Version)
	}

	return ci, err
}

func ReadCorrectedIntensity(filename string) (CorrectedIntensity, error) {
	ci := CorrectedIntensity{}
	f, err := os.Open(filename)
	if err != nil {
		return ci, err
	}
	defer func() { _ = f.Close() }()
	r := bufio.NewReader(f)
	return ParseCorrectedIntensity(r)
}
