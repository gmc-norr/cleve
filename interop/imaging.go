package interop

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type ImagingRecord struct {
	Lane  int
	Tile  int
	Cycle int
	Read  int

	PercentOccupied float64
	PercentPF       float64
}

type ImagingTable struct {
	Records []ImagingRecord
}

type TileSummary struct {
	PercentOccupied float64
	PercentPF       float64
}

func (t ImagingTable) LaneTileSummary() map[int]map[int]TileSummary {
	tileSummary := make(map[int]map[int]TileSummary)
	laneTileCount := make(map[int]map[int]int)
	for _, r := range t.Records {
		if _, ok := tileSummary[r.Lane]; !ok {
			tileSummary[r.Lane] = make(map[int]TileSummary)
		}
		summary, ok := tileSummary[r.Lane][r.Tile]
		if !ok {
			tileSummary[r.Lane][r.Tile] = TileSummary{}
		}
		summary.PercentOccupied += r.PercentOccupied
		summary.PercentPF += r.PercentPF

		_, ok = laneTileCount[r.Lane]
		if !ok {
			laneTileCount[r.Lane] = make(map[int]int)
		}

		laneTileCount[r.Lane][r.Tile] += 1

		tileSummary[r.Lane][r.Tile] = summary
	}

	for lane := range tileSummary {
		for tile := range tileSummary[lane] {
			summary := tileSummary[lane][tile]
			summary.PercentOccupied /= float64(laneTileCount[lane][tile])
			summary.PercentPF /= float64(laneTileCount[lane][tile])
			tileSummary[lane][tile] = summary
		}
	}

	return tileSummary
}

type ImagingTableParser struct {
	reader      *csv.Reader
	headerIndex map[string]int
}

func (p ImagingTableParser) header() ([]string, error) {
	var header []string
	var line []string
	var err error

	p.reader.FieldsPerRecord = -1

	for {
		line, err = p.reader.Read()
		if err != nil {
			return header, err
		}
		if !strings.HasPrefix(strings.TrimSpace(line[0]), "#") {
			break
		}
	}

	componentRegexp := regexp.MustCompile(`<(.+)>$`)

	for _, v := range line {
		s := strings.TrimSpace(v)
		componentString := componentRegexp.FindStringSubmatch(s)
		columnName := componentRegexp.ReplaceAllString(s, "")
		if componentString == nil {
			header = append(header, s)
			continue
		}
		components := strings.Split(componentString[1], ";")
		for _, c := range components {
			header = append(header, fmt.Sprintf("%s %s", columnName, strings.TrimSpace(c)))
		}
	}

	return header, nil
}

func (p ImagingTableParser) record() (*ImagingRecord, error) {
	rec, err := p.reader.Read()
	if err != nil {
		return nil, err
	}

	record := &ImagingRecord{}

	record.Lane, err = strconv.Atoi(rec[p.headerIndex["Lane"]])
	if err != nil {
		return record, err
	}

	record.Tile, err = strconv.Atoi(rec[p.headerIndex["Tile"]])
	if err != nil {
		return record, err
	}

	record.Cycle, err = strconv.Atoi(rec[p.headerIndex["Cycle"]])
	if err != nil {
		return record, err
	}

	record.Read, err = strconv.Atoi(rec[p.headerIndex["Read"]])
	if err != nil {
		return record, err
	}

	record.PercentOccupied, err = strconv.ParseFloat(rec[p.headerIndex["% Occupied"]], 64)
	if err != nil {
		return record, err
	}

	record.PercentPF, err = strconv.ParseFloat(rec[p.headerIndex["% Pass Filter"]], 64)
	if err != nil {
		return record, err
	}

	return record, nil
}

func GenerateImagingTable(runId string, runDirectory string) (*ImagingTable, error) {
	interopBin, ok := os.LookupEnv("INTEROP_BIN")
	if !ok {
		return nil, fmt.Errorf("INTEROP_BIN env var not found")
	}
	interopSummary := fmt.Sprintf("%s/imaging_table", interopBin)
	res, err := exec.Command(interopSummary, runDirectory).Output()
	if err != nil {
		return nil, err
	}

	buf := bytes.NewReader(res)
	r := bufio.NewReader(buf)

	imaging, err := ParseImagingTable(r)
	if err != nil {
		return nil, err
	}
	return imaging, nil
}

func ParseImagingTable(r *bufio.Reader) (*ImagingTable, error) {
	parser := ImagingTableParser{reader: csv.NewReader(r)}
	header, err := parser.header()
	if err != nil {
		return nil, fmt.Errorf("error parsing header: %s", err.Error())
	}

	parser.reader.FieldsPerRecord = len(header)
	parser.headerIndex = make(map[string]int)

	for i, v := range header {
		parser.headerIndex[v] = i
	}

	imagingTable := &ImagingTable{}

	for {
		rec, err := parser.record()
		if err != nil {
			if err == io.EOF {
				break
			}
			return imagingTable, err
		}
		imagingTable.Records = append(imagingTable.Records, *rec)
	}

	return imagingTable, nil
}
