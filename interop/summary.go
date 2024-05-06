package interop

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var InteropSummaryColumns = map[string]FieldValue {
	"Level": &StringValue{},
	"Yield": &FloatValue{},
	"Projected Yield": &FloatValue{},
	"Aligned": &FloatValue{},
	"Error Rate": &FloatValue{},
	"Intensity C1": &FloatValue{},
	"%>=Q30": &FloatValue{},
	"% Occupied": &FloatValue{},
}

var InteropReadColumns = map[string]FieldValue {
	"Lane": &IntValue{},
	"Surface": &IntValue{},
	"Tiles": &IntValue{},
	"Density": &MeanSdValue{},
	"Cluster PF": &MeanSdValue{},
	"Legacy Phasing/Prephasing Rate": &DoubleValue{},
	"Phasing  slope/offset": &DoubleValue{},
	"Prephasing slope/offset": &DoubleValue{},
	"Reads": &FloatValue{},
	"Reads PF": &FloatValue{},
	"%>=Q30": &FloatValue{},
	"Yield": &FloatValue{},
	"Cycles Error": &RangeValue{},
	"Aligned": &MeanSdValue{},
	"Error": &MeanSdValue{},
	"Error (35)": &MeanSdValue{},
	"Error (75)": &MeanSdValue{},
	"Error (100)": &MeanSdValue{},
	"% Occupied": &MeanSdValue{},
	"Intensity C1": &MeanSdValue{},
}

type InteropSummary struct {
	Version      string
	RunDirectory string
	RunSummary
	ReadSummaries []ReadSummary
}

type RunSummary struct {
	Header []string
	Fields map[string]FieldValue
}

type ReadSummary struct {
	ReadName   string
	Header     []string
	EntryCount int
	Fields     map[string][]FieldValue
}

type FieldValue interface {
	Parse(string) error
	String() string
}

type IntValue struct {
	Value int
}

func (v IntValue) String() string {
	return fmt.Sprintf("%d", v.Value)
}

func (v *IntValue) Parse(s string) error {
	if s == "-" {
		v.Value = -1
		return nil
	}
	x, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	v.Value = x
	return nil
}

func NewIntValue(s string) *IntValue {
	i := IntValue{}
	if err := i.Parse(s); err != nil {
		panic(err)
	}
	return &i
}

type FloatValue struct {
	Value float64
}

func (v FloatValue) String() string {
	return fmt.Sprintf("%.02f", v.Value)
}

func (v *FloatValue) Parse(s string) error {
	x, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return(err)
	}
	v.Value = x
	return nil
}

func NewFloatValue(s string) *FloatValue {
	f := FloatValue{}
	err := f.Parse(s)
	if err != nil {
		panic(err)
	}
	return &f
}

type StringValue struct {
	Value string
}

func (v StringValue) String() string {
	return v.Value
}

func (v *StringValue) Parse(s string) error {
	v.Value = s
	return nil
}

func NewStringValue(s string) *StringValue {
	return &StringValue{s}
}

type DoubleValue struct {
	First  float64
	Second float64
}

func (v DoubleValue) String() string {
	return fmt.Sprintf("%.02f / %.02f", v.First, v.Second)
}

func (v *DoubleValue) Parse(s string) error {
	splitString := strings.Split(s, "/")
	if len(splitString) != 2 {
		panic("illegal DoubleValue")
	}
	first, err := strconv.ParseFloat(strings.TrimSpace(splitString[0]), 64)
	if err != nil {
		panic(err)
	}
	second, err := strconv.ParseFloat(strings.TrimSpace(splitString[0]), 64)
	if err != nil {
		panic(err)
	}

	v.First = first
	v.Second = second
	return nil
}

func NewDoubleValue(s string) *DoubleValue {
	d := DoubleValue{}
	if err := d.Parse(s); err != nil {
		panic(err)
	}
	return &d
}

type RangeValue struct {
	Start float64
	End   float64
}

func (v RangeValue) String() string {
	return fmt.Sprintf("%.02f - %.02f", v.Start, v.End)
}

func (v *RangeValue) Parse(s string) error {
	splitString := strings.Split(s, "-")
	if len(splitString) == 1 {
		start, err := strconv.ParseFloat(strings.TrimSpace(splitString[0]), 64)
		if err != nil {
			return err
		}
		v.Start = start
		v.End = start
		return nil
	}
	if len(splitString) != 2 {
		return fmt.Errorf("illegal RangeValue: %s", s)
	}
	start, err := strconv.ParseFloat(strings.TrimSpace(splitString[0]), 64)
	if err != nil {
		return err
	}
	end, err := strconv.ParseFloat(strings.TrimSpace(splitString[0]), 64)
	if err != nil {
		return err
	}
	v.Start = start
	v.End = end
	return nil
}

func NewRangeValue(s string) *RangeValue {
	r := RangeValue{}
	if err := r.Parse(s); err != nil {
		panic(err)
	}
	return &r
}

type MissingValue struct{}

func (v MissingValue) String() string {
	return "-"
}

func (v MissingValue) Parse(s string) error {
	return nil
}

func NewMissingValue() *MissingValue {
	return &MissingValue{}
}

type MeanSdValue struct {
	Mean float64
	SD   float64
}

func (v MeanSdValue) String() string {
	return fmt.Sprintf("%.02f +/- %.02f", v.Mean, v.SD)
}

func (v *MeanSdValue) Parse(s string) error {
	splitString := strings.Split(s, "+/-")
	if len(splitString) != 2 {
		return fmt.Errorf("illegal MeanSdValue: %s", s)
	}
	mean, err := strconv.ParseFloat(strings.TrimSpace(splitString[0]), 64)
	if err != nil {
		return err
	}

	sd, err := strconv.ParseFloat(strings.TrimSpace(splitString[0]), 64)
	if err != nil {
		return err
	}

	v.Mean = mean
	v.SD = sd
	return nil
}

func NewMeanSdValue(s string) *MeanSdValue {
	msd := MeanSdValue{}
	if err := msd.Parse(s); err != nil {
		panic(err)
	}
	return &msd
}

func GenerateSummary(runDirectory string) (*InteropSummary, error) {
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

	return ParseInteropSummary(r)
}

func ParseReadSummary(r *bufio.Reader) (*ReadSummary, error) {
	csvReader := csv.NewReader(r)
	csvReader.FieldsPerRecord = 20

	peek, err := r.Peek(6)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(strings.TrimSpace(string(peek)), "Read") {
		return nil, fmt.Errorf("this is not a read section")
	}

	summary := &ReadSummary{}

	sectionHeader, _ := csvReader.Read()

	summary.ReadName = sectionHeader[0]

	header, err := csvReader.Read()
	if err != nil {
		return nil, err
	}

	summary.Header = header
	summary.Fields = make(map[string][]FieldValue)

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
			return nil, err
		}

		for i, val := range rec {
			var f FieldValue
			switch InteropReadColumns[summary.Header[i]].(type) {
			case *IntValue:
				f = NewIntValue(val)
			case *FloatValue:
				f = NewFloatValue(val)
			case *DoubleValue:
				f = NewDoubleValue(val)
			case *MeanSdValue:
				f = NewMeanSdValue(val)
			case *RangeValue:
				f = NewRangeValue(val)
			case *MissingValue:
				f = NewMissingValue()
			case *StringValue:
				f = NewStringValue(val)
			default:
				return nil, fmt.Errorf("invalid type for read summary column %s: %s", summary.Header[i], val)
			}
			summary.Fields[summary.Header[i]] = append(summary.Fields[summary.Header[i]], f)
		}
		summary.EntryCount++
	}

	return summary, nil
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

	runSummary := &RunSummary{}
	runSummary.Header = header
	runSummary.Fields = make(map[string]FieldValue)

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

		for i, val := range rec {
			var f FieldValue
			switch InteropSummaryColumns[runSummary.Header[i]].(type) {
			case *StringValue:
				f = NewStringValue(val)
			case *IntValue:
				f = NewIntValue(val)
			case *FloatValue:
				f = NewFloatValue(val)
			default:
				return nil, fmt.Errorf("invalid type in summary for column %s: %s", runSummary.Header[i], val)
			}
			runSummary.Fields[runSummary.Header[i]] = f
		}
	}

	summary.RunSummary = *runSummary

	for {
		readSummary, err := ParseReadSummary(r)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		summary.ReadSummaries = append(summary.ReadSummaries, *readSummary)
	}

	return summary, nil
}
