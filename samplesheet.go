package cleve

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type SectionType = int

const (
	UnknownSection SectionType = iota
	SettingsSection
	DataSection
)

type SampleSheet struct {
	Sections []Section
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

type Section struct {
	Name string
	Type int
	Rows [][]string
}

func (s Section) Get(name string, index ...int) (string, bool) {
	switch s.Type {
	case SettingsSection:
		for _, row := range s.Rows {
			if row[0] == name {
				return row[1], true
			}
		}
	case DataSection:
		colIndex := -1
		for i, s := range s.Rows[0] {
			if s == name {
				colIndex = i
				break
			}
		}

		rowIndex := -1
		if len(index) > 0 {
			rowIndex = index[0]
		} else {
			return "", false
		}

		if colIndex >= 0 && rowIndex >= 0 {
			return s.Rows[rowIndex][colIndex], true
		}

	}

	return "", false
}

func (s Section) GetInt(name string, index ...int) (int, error) {
	v, ok := s.Get(name, index...)
	if !ok {
		if index != nil {
			return 0, fmt.Errorf("index %d for key %s not found", index[0], name)
		}
		return 0, fmt.Errorf("key %s not found", name)
	}
	return strconv.Atoi(v)
}

func (s Section) GetFloat(name string, index ...int) (float64, error) {
	v, ok := s.Get(name, index...)
	if !ok {
		if index != nil {
			return 0, fmt.Errorf("index %d for key %s not found", index[0], name)
		}
		return 0, fmt.Errorf("key %s not found", name)
	}
	return strconv.ParseFloat(v, 64)
}

type sampleSheetParser struct {
	reader *bufio.Reader
}

func (p *sampleSheetParser) Peek() (rune, error) {
	r, _, err := p.reader.ReadRune()
	p.reader.UnreadRune()
	return r, err
}

func (p *sampleSheetParser) ParseHeader() (string, int, error) {
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

	var sectionType int
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
		return name, UnknownSection, fmt.Errorf("unknown section type: %s", name)
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

	return sheet, nil
}

func ReadSampleSheet(filename string) (SampleSheet, error) {
	f, err := os.Open(filename)
	if err != nil {
		return SampleSheet{}, err
	}
	r := bufio.NewReader(f)
	return ParseSampleSheet(r)
}
