package cleve

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
	Lane  int `bson:"lane" json:"lane"`
	Tile  int `bson:"tile" json:"tile"`
	Cycle int `bson:"cycle" json:"cycle"`
	Read  int `bson:"read" json:"read"`

	PercentOccupied float64 `bson:"percent_occupied" json:"percent_occupied"`
	PercentPF       float64 `bson:"percent_pf" json:"percent_pf"`
}

type ImagingTable struct {
	Records []ImagingRecord
}

type TileSummary struct {
	Lane            int     `bson:"lane" json:"lane"`
	Tile            int     `bson:"tile" json:"tile"`
	PercentOccupied float64 `bson:"percent_occupied" json:"percent_occupied"`
	PercentPF       float64 `bson:"percent_pf" json:"percent_pf"`
}

func (t ImagingTable) LaneTileSummary() []TileSummary {
	tileSummary := make([]TileSummary, 0)
	tileSummaryMap := make(map[int]map[int]TileSummary)
	laneTileCount := make(map[int]map[int]int)
	for _, r := range t.Records {
		if _, ok := tileSummaryMap[r.Lane]; !ok {
			tileSummaryMap[r.Lane] = make(map[int]TileSummary)
		}
		summary, ok := tileSummaryMap[r.Lane][r.Tile]
		if !ok {
			summary = TileSummary{
				Lane: r.Lane,
				Tile: r.Tile,
			}
			tileSummaryMap[r.Lane][r.Tile] = summary
		}
		summary.PercentOccupied += r.PercentOccupied
		summary.PercentPF += r.PercentPF

		_, ok = laneTileCount[r.Lane]
		if !ok {
			laneTileCount[r.Lane] = make(map[int]int)
		}

		laneTileCount[r.Lane][r.Tile] += 1

		tileSummaryMap[r.Lane][r.Tile] = summary
	}

	for lane := range tileSummaryMap {
		for tile := range tileSummaryMap[lane] {
			summary := tileSummaryMap[lane][tile]
			summary.PercentOccupied /= float64(laneTileCount[lane][tile])
			summary.PercentPF /= float64(laneTileCount[lane][tile])
			tileSummary = append(tileSummary, summary)
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

	// Lane, tile, cycle and read *should* be there, so I won't bother checking for errors
	// when fetching the index.
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

	pOccupiedIndex, ok := p.headerIndex["% Occupied"]
	if ok {
		record.PercentOccupied, err = strconv.ParseFloat(rec[pOccupiedIndex], 64)
		if err != nil {
			return record, err
		}
	}

	pPassFilterIndex, ok := p.headerIndex["% Pass Filter"]
	if ok {
		record.PercentPF, err = strconv.ParseFloat(rec[pPassFilterIndex], 64)
		if err != nil {
			return record, err
		}
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
