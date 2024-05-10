package interop

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type InteropSummary struct {
	Version       string                   `bson:"version" json:"version"`
	RunID         string                   `bson:"run_id" json:"run_id"`
	RunDirectory  string                   `bson:"run_directory" json:"run_directory"`
	RunSummary    map[string]RunSummary    `bson:"run_summmary" json:"run_summary"`
	ReadSummaries map[string][]ReadSummary `bson:"read_summary" json:"read_summary"`
}

type RunSummary struct {
	Level           string    `bson:"level" json:"level"`
	Yield           int       `bson:"yield" json:"yield"`
	ProjectedYield  int       `bson:"projected_yield" json:"projected_yield"`
	PercentAligned  JsonFloat `bson:"percent_aligned" json:"percent_aligned"`
	ErrorRate       JsonFloat `bson:"error_rate" json:"error_rate"`
	IntensityC1     JsonFloat `bson:"intensity_c1" json:"intensity_c1"`
	PercentQ30      JsonFloat `bson:"percent_q30" json:"percent_q30"`
	PercentOccupied JsonFloat `bson:"percent_occupied" json:"percent_occupied"`
}

type JsonFloat float64

func (x JsonFloat) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if math.IsNaN(float64(x)) {
		buf.WriteString(`null`)
		return buf.Bytes(), nil
	}
	return json.Marshal(float64(x))
}

type MeanSd struct {
	Mean JsonFloat `bson:"mean" json:"mean"`
	SD   JsonFloat `bson:"sd" json:"sd"`
}

type Range struct {
	Start int `bson:"start" json:"start"`
	End   int `bson:"end" json:"end"`
}

type ReadSummary struct {
	Lane             int       `bson:"lane" json:"lane"`
	Tiles            int       `bson:"tiles" json:"tiles"`
	Density          MeanSd    `bson:"density" json:"density"`
	ClusterPF        MeanSd    `bson:"cluster_pf" json:"cluster_pf"`
	PhasingRate      JsonFloat `bson:"phasing_rate" json:"phasing_rate"`
	PrephasingRate   JsonFloat `bson:"prephasing_rate" json:"prephasing_rate"`
	PhasingSlope     JsonFloat `bson:"phasing_slope" json:"phasing_slope"`
	PhasingOffset    JsonFloat `bson:"phasing_offset" json:"phasing_offset"`
	PrephasingSlope  JsonFloat `bson:"prephasing_slope" json:"prephasing_slope"`
	PrephasingOffset JsonFloat `bson:"prephasing_offset" json:"prephasing_offset"`
	Reads            int       `bson:"reads" json:"reads"`
	ReadsPF          int       `bson:"reads_pf" json:"reads_pf"`
	PercentQ30       JsonFloat `bson:"percent_q30" json:"percent_q30"`
	Yield            int       `bson:"yield" json:"yield"`
	CyclesError      Range     `bson:"cycles_error" json:"cycles_error"`
	PercentAligned   MeanSd    `bson:"percent_aligned" json:"percent_aligned"`
	Error            MeanSd    `bson:"error" json:"error"`
	Error35          MeanSd    `bson:"error35" json:"error35"`
	Error75          MeanSd    `bson:"error75" json:"error75"`
	Error100         MeanSd    `bson:"error100" json:"error100"`
	PercentOccupied  MeanSd    `bson:"percent_occupied" json:"percent_occupied"`
	IntensityC1      MeanSd    `bson:"intensity_c1" json:"intensity_c1"`
}

func parseMeanSd(s string) (MeanSd, error) {
	res := MeanSd{}
	splitString := strings.Split(s, "+/-")
	if len(splitString) != 2 {
		return res, fmt.Errorf("illegal MeanSdValue: %s", s)
	}
	mean, err := strconv.ParseFloat(strings.TrimSpace(splitString[0]), 64)
	if err != nil {
		return res, err
	}

	sd, err := strconv.ParseFloat(strings.TrimSpace(splitString[1]), 64)
	if err != nil {
		return res, err
	}

	res.Mean = JsonFloat(mean)
	res.SD = JsonFloat(sd)
	return res, nil
}

func parseInt(s string) (int, error) {
	if s == "-" {
		return -1, nil
	}
	x, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return x, nil
}

func parseFloat(s string) (JsonFloat, error) {
	x, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return JsonFloat(x), nil
}

func parsePair(s string) (JsonFloat, JsonFloat, error) {
	splitString := strings.Split(s, "/")
	if len(splitString) != 2 {
		return 0, 0, fmt.Errorf("illegal DoubleValue")
	}
	first, err := strconv.ParseFloat(strings.TrimSpace(splitString[0]), 64)
	if err != nil {
		return 0, 0, err
	}
	second, err := strconv.ParseFloat(strings.TrimSpace(splitString[1]), 64)
	if err != nil {
		return 0, 0, err
	}

	return JsonFloat(first), JsonFloat(second), nil
}

func parseRange(s string) (Range, error) {
	splitString := strings.Split(s, "-")
	if len(splitString) == 1 {
		start, err := strconv.Atoi(strings.TrimSpace(splitString[0]))
		if err != nil {
			return Range{0, 0}, err
		}
		return Range{start, start}, nil
	}
	if len(splitString) != 2 {
		return Range{}, fmt.Errorf("illegal RangeValue: %s", s)
	}
	start, err := strconv.Atoi(strings.TrimSpace(splitString[0]))
	if err != nil {
		return Range{}, err
	}
	end, err := strconv.Atoi(strings.TrimSpace(splitString[1]))
	if err != nil {
		return Range{}, err
	}
	return Range{start, end}, nil
}

func GenerateSummary(runId string, runDirectory string) (*InteropSummary, error) {
	interopBin, ok := os.LookupEnv("INTEROP_BIN")
	if !ok {
		return nil, fmt.Errorf("INTEROP_BIN env var not found")
	}
	interopSummary := fmt.Sprintf("%s/summary", interopBin)
	res, err := exec.Command(interopSummary, runDirectory, "--level=3", "--csv=1").Output()
	if err != nil {
		return nil, err
	}

	buf := bytes.NewReader(res)
	r := bufio.NewReader(buf)

	summary, err := ParseInteropSummary(r)
	if err != nil {
		return nil, err
	}
	summary.RunID = runId
	return summary, nil
}

func ParseReadSummary(r *bufio.Reader) (string, []ReadSummary, error) {
	csvReader := csv.NewReader(r)
	csvReader.FieldsPerRecord = 20

	peek, err := r.Peek(6)
	if err != nil {
		return "", nil, err
	}

	if !strings.HasPrefix(strings.TrimSpace(string(peek)), "Read") {
		return "", nil, fmt.Errorf("this is not a read section")
	}

	reads := make([]ReadSummary, 0)
	sectionHeader, _ := csvReader.Read()

	header, err := csvReader.Read()
	if err != nil {
		return "", nil, err
	}

	for {
		peek, _ = r.Peek(4)
		if string(peek) == "Read" {
			// next section
			break
		}

		rec, err := csvReader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", nil, err
		}

		readSummary := ReadSummary{}

		for i, val := range rec {
			switch header[i] {
			case "Lane":
				readSummary.Lane, _ = parseInt(val)
			case "Tiles":
				readSummary.Tiles, _ = parseInt(val)
			case "Density":
				readSummary.Density, _ = parseMeanSd(val)
			case "Cluster PF":
				readSummary.ClusterPF, _ = parseMeanSd(val)
			case "Legacy Phasing/Prephasing Rate":
				first, second, _ := parsePair(val)
				readSummary.PhasingRate = first
				readSummary.PrephasingRate = second
			case "Phasing slope/offset":
				first, second, _ := parsePair(val)
				readSummary.PhasingSlope = first
				readSummary.PhasingOffset = second
			case "Prephasing slope/offset":
				first, second, _ := parsePair(val)
				readSummary.PrephasingSlope = first
				readSummary.PrephasingOffset = second
			case "Reads":
				readCount, _ := parseFloat(val)
				readSummary.Reads = int(readCount * 1e6)
			case "Reads PF":
				readCount, _ := parseFloat(val)
				readSummary.ReadsPF = int(readCount * 1e6)
			case "%>=Q30":
				readSummary.PercentQ30, _ = parseFloat(val)
			case "Yield":
				gigaBases, _ := parseFloat(val)
				readSummary.Yield = int(gigaBases * 1e9)
			case "Cycles Error":
				readSummary.CyclesError, _ = parseRange(val)
			case "Aligned":
				readSummary.PercentAligned, _ = parseMeanSd(val)
			case "Error":
				readSummary.Error, _ = parseMeanSd(val)
			case "Error (35)":
				readSummary.Error35, _ = parseMeanSd(val)
			case "Error (75)":
				readSummary.Error75, _ = parseMeanSd(val)
			case "Error (100)":
				readSummary.Error100, _ = parseMeanSd(val)
			case "% Occupied":
				readSummary.PercentOccupied, _ = parseMeanSd(val)
			case "Intensity C1":
				readSummary.IntensityC1, _ = parseMeanSd(val)
			}
		}
		reads = append(reads, readSummary)
	}

	return sectionHeader[0], reads, nil
}

func ParseInteropSummary(r *bufio.Reader) (*InteropSummary, error) {
	versionString, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	versionString = strings.TrimSpace(strings.Split(versionString, " ")[2])

	runDirectory, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	runDirectory = strings.TrimSpace(runDirectory)

	csvReader := csv.NewReader(r)
	csvReader.FieldsPerRecord = 8
	header, err := csvReader.Read()
	if err != nil {
		return nil, err
	}

	summary := &InteropSummary{}
	summary.Version = versionString
	summary.RunDirectory = runDirectory

	summary.RunSummary = make(map[string]RunSummary)

	for {
		peek, err := r.Peek(2)
		if err != nil {
			return nil, err
		}
		if string(peek) == "\n\n" {
			// read sections are starting
			break
		}
		rec, err := csvReader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("%s when reading total summary", err.Error())
		}

		runSummary := RunSummary{}

		for i, val := range rec {
			switch header[i] {
			case "Level":
				runSummary.Level = val
			case "Yield":
				gigaBases, _ := parseFloat(val)
				runSummary.Yield = int(gigaBases * 1e9)
			case "Projected Yield":
				gigaBases, _ := parseFloat(val)
				runSummary.ProjectedYield = int(gigaBases * 1e9)
			case "Aligned":
				runSummary.PercentAligned, _ = parseFloat(val)
			case "Error Rate":
				runSummary.ErrorRate, _ = parseFloat(val)
			case "Intensity C1":
				runSummary.IntensityC1, _ = parseFloat(val)
			case "%>=Q30":
				runSummary.PercentQ30, _ = parseFloat(val)
			case "% Occupied":
				runSummary.PercentOccupied, _ = parseFloat(val)
			}
		}
		summary.RunSummary[runSummary.Level] = runSummary
	}

	summary.ReadSummaries = make(map[string][]ReadSummary)

	for {
		readName, readSummary, err := ParseReadSummary(r)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		summary.ReadSummaries[readName] = readSummary
	}

	return summary, nil
}
