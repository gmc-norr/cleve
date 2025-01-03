package cleve

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/mongo"
)

type (
	UpdateResult = mongo.UpdateResult
	SectionType  int
)

const (
	UnknownSection SectionType = iota
	SettingsSection
	DataSection
)

func (s SectionType) String() string {
	switch s {
	case SettingsSection:
		return "settings"
	case DataSection:
		return "data"
	case UnknownSection:
		return "unknown"
	default:
		return "unknown"
	}
}

func (s SectionType) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bson.MarshalValue(s.String())
}

func (s *SectionType) UnmarshalBSONValue(t bsontype.Type, b []byte) error {
	var typeString string
	raw := bson.RawValue{
		Type:  t,
		Value: b,
	}
	err := raw.Unmarshal(&typeString)
	if err != nil {
		return err
	}

	switch typeString {
	case "settings":
		*s = SettingsSection
	case "data":
		*s = DataSection
	case "unknown":
		*s = DataSection
	default:
		*s = UnknownSection
	}
	return nil
}

func (s SectionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

type SampleSheetInfo struct {
	Path             string    `bson:"path" json:"path"`
	ModificationTime time.Time `bson:"modification_time" json:"modification_time"`
}

type SampleSheet struct {
	RunID    *string           `bson:"run_id" json:"run_id"`
	UUID     *uuid.UUID        `bson:"uuid" json:"uuid"`
	Files    []SampleSheetInfo `bson:"files" json:"files"`
	Sections []Section         `bson:"sections" json:"sections"`
}

func (s SampleSheet) Section(name string) *Section {
	for _, section := range s.Sections {
		if section.Name == name {
			return &section
		}
	}
	return nil
}

func (s SampleSheet) Version() int {
	v, _ := s.Section("Header").GetInt("FileFormatVersion")
	return v
}

func (s SampleSheet) IsValid() bool {
	return s.Section("Header") != nil && s.Section("Reads") != nil
}

func (s SampleSheet) LastModified() (time.Time, error) {
	var mostRecent time.Time
	if len(s.Files) == 0 {
		return mostRecent, fmt.Errorf("no modification times registered")
	}
	for i, f := range s.Files {
		if i == 0 || mostRecent.Compare(f.ModificationTime) == -1 {
			mostRecent = f.ModificationTime
		}
	}
	return mostRecent, nil
}

// Merge two sample sheets. Merging is only allowed if the UUIDs of the sample
// sheets are the same, and the run IDs are the same. An exception to this is if
// the run ID of the current sample sheet is nil. If the run ID in the current
// sample sheet is non-nil and different from the other sample sheet, or if the
// UUIDs are different, an error is returned.
func (s SampleSheet) Merge(other *SampleSheet) (*SampleSheet, error) {
	// Can only merge if run IDs are the same, or if the run ID of this
	// sample sheet is nil.
	if s.RunID != nil && other.RunID != nil && *s.RunID != *other.RunID {
		return nil, fmt.Errorf("cannot merge sample sheets with different run IDs")
	}
	if s.UUID != nil && other.UUID != nil && *s.UUID != *other.UUID {
		return nil, fmt.Errorf("cannot merge sample sheets with different UUIDs")
	}

	// Ignore errors, since the default will always be the oldest
	modtime, _ := s.LastModified()
	otherModtime, _ := other.LastModified()

	otherNewer := otherModtime.Compare(modtime) == 1
	mergedSampleSheet := SampleSheet{
		UUID:     s.UUID,
		Sections: make([]Section, 0),
		Files:    make([]SampleSheetInfo, 0),
	}

	if s.RunID == nil {
		mergedSampleSheet.RunID = other.RunID
	} else {
		mergedSampleSheet.RunID = s.RunID
	}

	sampleSheetFiles := make(map[string]time.Time)

	for _, f := range append(s.Files, other.Files...) {
		if mt, ok := sampleSheetFiles[f.Path]; !ok {
			// We haven't seen this file before, add it
			sampleSheetFiles[f.Path] = f.ModificationTime
		} else {
			// We haven't seen it, update the modification time, but only if it's newer.
			if f.ModificationTime.Compare(mt) == 1 {
				sampleSheetFiles[f.Path] = f.ModificationTime
			}
		}
	}

	for p, mt := range sampleSheetFiles {
		mergedSampleSheet.Files = append(mergedSampleSheet.Files, SampleSheetInfo{
			Path:             p,
			ModificationTime: mt,
		})
	}

	slices.SortStableFunc(mergedSampleSheet.Files, func(a, b SampleSheetInfo) int {
		return a.ModificationTime.Compare(b.ModificationTime)
	})

	for _, section := range s.Sections {
		if otherSection := other.Section(section.Name); otherSection != nil {
			// Other sample sheet has this section too.
			// Are they different? If so, use the newer one.
			if !reflect.DeepEqual(section, *otherSection) {
				if otherNewer {
					mergedSampleSheet.Sections = append(mergedSampleSheet.Sections, *otherSection)
				} else {
					mergedSampleSheet.Sections = append(mergedSampleSheet.Sections, section)
				}
			} else {
				mergedSampleSheet.Sections = append(mergedSampleSheet.Sections, section)
			}
		} else {
			// Only found in this, add it.
			mergedSampleSheet.Sections = append(mergedSampleSheet.Sections, section)
		}
	}
	// Go through the sections of other and add any that are not in the current one
	for _, section := range other.Sections {
		if thisSection := s.Section(section.Name); thisSection == nil {
			mergedSampleSheet.Sections = append(mergedSampleSheet.Sections, section)
		}
	}

	return &mergedSampleSheet, nil
}

type Section struct {
	Name string      `bson:"name" json:"name"`
	Type SectionType `bson:"type" json:"type"`
	Rows [][]string  `bson:"rows" json:"rows"`
}

func (s Section) Get(name string, index ...int) (string, error) {
	switch s.Type {
	case SettingsSection:
		for _, row := range s.Rows {
			if row[0] == name {
				if len(row) == 1 {
					return row[0], nil
				}
				return row[1], nil
			}
		}
		return "", fmt.Errorf("key %s not found in section %s", name, s.Name)
	case DataSection:
		colIndex := -1
		for i, s := range s.Rows[0] {
			if s == name {
				colIndex = i
				break
			}
		}

		if colIndex == -1 {
			return "", fmt.Errorf("column %s not found in section %s", name, s.Name)
		}

		var rowIndex int
		if len(index) > 0 {
			rowIndex = index[0]
		} else {
			return "", fmt.Errorf("no index given")
		}

		if colIndex >= 0 && rowIndex >= 0 {
			return s.Rows[rowIndex][colIndex], nil
		} else {
			return "", fmt.Errorf("index %d out of bounds for column %q", rowIndex, name)
		}
	default:
		return "", fmt.Errorf("unknown section type: %v", s.Type)
	}
}

func (s Section) GetInt(name string, index ...int) (int, error) {
	v, err := s.Get(name, index...)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(v)
}

func (s Section) GetFloat(name string, index ...int) (float64, error) {
	v, err := s.Get(name, index...)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(v, 64)
}

func (s Section) GetColumn(name string) ([]string, error) {
	if s.Type != DataSection {
		return nil, fmt.Errorf("section %q is not a data section", s.Name)
	}
	header := s.Rows[0]
	colIndex := -1
	for i, colName := range header {
		if colName == name {
			colIndex = i
		}
	}
	if colIndex == -1 {
		return nil, fmt.Errorf("column %q not found in section %q", name, s.Name)
	}
	colValues := make([]string, len(s.Rows)-1)
	for i := 1; i < len(s.Rows); i++ {
		colValues[i-1] = s.Rows[i][colIndex]
	}
	return colValues, nil
}

type sampleSheetParser struct {
	reader *bufio.Reader
}

func (p *sampleSheetParser) Peek() (rune, error) {
	r, _, err := p.reader.ReadRune()
	if err != nil {
		return r, err
	}
	return r, p.reader.UnreadRune()
}

func (p *sampleSheetParser) ParseHeader() (string, SectionType, error) {
	r, _, _ := p.reader.ReadRune()
	if r == '\x00' {
		return "", UnknownSection, io.EOF
	}
	if r != '[' {
		return "", UnknownSection, fmt.Errorf("parsing error: expected section header")
	}
	name := ""
	for {
		r, _, err := p.reader.ReadRune()
		if err != nil {
			return "", UnknownSection, err
		}
		if r == ']' {
			if len(name) == 0 {
				return "", UnknownSection, fmt.Errorf("parsing error: empty section header")
			}
			// check that the rest of the line is empty
			rest, err := p.reader.ReadBytes('\n')
			if err != nil && err.Error() != "EOF" {
				return "", UnknownSection, err
			}
			restTrim := strings.TrimSpace(string(rest))
			if len(restTrim) > 0 {
				// Allow trailing commas
				for _, rune := range restTrim {
					if rune != ',' {
						return "", UnknownSection, fmt.Errorf(
							"parsing error: trailing characters after header %q",
							string(name),
						)
					}
				}
			}
			break
		}
		name += string(r)
	}

	name = strings.TrimSpace(name)

	var sectionType SectionType
	switch {
	case name == "Header", name == "Reads":
		sectionType = SettingsSection
	case name == "Data":
		sectionType = DataSection
	case strings.HasSuffix(strings.ToLower(name), "data"):
		sectionType = DataSection
	case strings.HasSuffix(strings.ToLower(name), "settings"):
		sectionType = SettingsSection
	default:
		return name, SettingsSection, nil
	}
	return name, sectionType, nil
}

// Parse a row of a section in the sample sheet.
// If the row is empty, or end of file is reached and no values
// have been read, then nil is returned for both the value and the error.
func (p *sampleSheetParser) ParseSectionRow() ([]string, error) {
	values := []string{}
	currentValue := ""
	if c, _ := p.Peek(); c == '[' {
		return nil, fmt.Errorf("parsing error: empty section")
	}
	for {
		r, _, err := p.reader.ReadRune()
		if err != nil {
			if err.Error() == "EOF" {
				if currentValue != "" {
					currentValue = strings.TrimSpace(currentValue)
					values = append(values, currentValue)
				}
				if len(values) > 0 {
					// eof on the last line of a section, that's ok
					return values, nil
				}
				// empty line at the end of the file, that's ok
				return nil, nil
			}
			return values, err
		}
		switch r {
		case '\n':
			if currentValue == "" && len(values) == 0 {
				// Empty line
				return nil, nil
			}
			currentValue = strings.TrimSpace(currentValue)
			values = append(values, currentValue)
			allEmpty := true
			for _, v := range values {
				if v != "" {
					allEmpty = false
					break
				}
			}
			if allEmpty {
				// only commas, ignore
				return nil, nil
			}
			return values, nil
		case ',':
			currentValue = strings.TrimSpace(currentValue)
			values = append(values, currentValue)
			currentValue = ""
			continue
		}
		currentValue += string(r)
	}
}

func ParseSampleSheet(r *bufio.Reader) (SampleSheet, error) {
	p := sampleSheetParser{r}
	sheet := SampleSheet{}
	for {
		name, typ, err := p.ParseHeader()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return sheet, err
		}

		var s Section
		s.Type = typ
		s.Name = name

		for {
			values, err := p.ParseSectionRow()
			if err != nil {
				if err.Error() == "EOF" {
					break
				}
				return sheet, err
			}

			if values == nil {
				break
			}

			s.Rows = append(s.Rows, values)

			next, err := p.Peek()
			if err != nil {
				if err.Error() == "EOF" {
					break
				}
				return sheet, err
			}
			if next == '[' {
				// New section
				break
			}
		}

		if len(s.Rows) == 0 {
			return sheet, fmt.Errorf("parsing error: empty section")
		}

		rowItemCount := len(s.Rows[0])
		if s.Type == SettingsSection && rowItemCount > 2 {
			return sheet, fmt.Errorf("parsing error: expected at most 2 items per row in section %q", s.Name)
		}
		for _, row := range s.Rows {
			if len(row) != rowItemCount {
				return sheet, fmt.Errorf("parsing error: expected %d items per row in section %q", rowItemCount, s.Name)
			}
		}

		sheet.Sections = append(sheet.Sections, s)
	}

	if !sheet.IsValid() {
		return sheet, fmt.Errorf("invalid sample sheet")
	}

	// Set the UUID of the sample sheet if one has been defined
	rd, err := sheet.Section("Header").Get("RunDescription")
	if err == nil {
		uuid, err := uuid.Parse(rd)
		if err == nil {
			sheet.UUID = &uuid
		}
	}

	return sheet, nil
}

func ReadSampleSheet(filename string) (SampleSheet, error) {
	f, err := os.Open(filename)
	if err != nil {
		return SampleSheet{}, err
	}
	defer f.Close()
	finfo, err := os.Stat(filename)
	if err != nil {
		return SampleSheet{}, err
	}

	r := bufio.NewReader(f)
	sampleSheet, err := ParseSampleSheet(r)
	if err != nil {
		return sampleSheet, err
	}

	sampleSheet.Files = make([]SampleSheetInfo, 1)
	sampleSheet.Files[0].Path, err = filepath.Abs(filename)
	if err != nil {
		return sampleSheet, err
	}
	sampleSheet.Files[0].ModificationTime = finfo.ModTime()
	return sampleSheet, nil
}

// Find the SampleSheet with the most recent modification time
// in a directory. The file name must be on the format `SampleSheet*.csv`.
func MostRecentSamplesheet(path string) (string, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}

	var samplesheet string
	modtime := time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC)

	for _, f := range files {
		fname := f.Name()
		if strings.HasPrefix(fname, "SampleSheet") && strings.HasSuffix(fname, ".csv") {
			ss := filepath.Join(path, fname)
			s, err := os.Stat(ss)
			if err != nil {
				return "", err
			}
			if s.ModTime().Compare(modtime) > 0 {
				modtime = s.ModTime()
				samplesheet = ss
			}
		}
	}

	if samplesheet == "" {
		return "", fmt.Errorf("no samplesheet found")
	}

	return samplesheet, nil
}
