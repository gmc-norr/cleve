package cleve

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSampleSheet(t *testing.T) {
	cases := []struct {
		Name                  string
		Version               int
		Valid                 bool
		Sections              []string
		SectionRows           []int
		SectionTypes          []SectionType
		Data                  []byte
		ExpectedSettingValues map[string]map[string]string
		ExpectedDataValues    map[string]map[string]map[int]string
	}{
		{
			"case 1",
			2,
			true,
			[]string{"Header", "Reads"},
			[]int{4, 1},
			[]SectionType{SettingsSection, SettingsSection},
			[]byte(`[Header]
FileFormatVersion,2
RunName,TestRun
InstrumentPlatform,NovaSeq
InstrumentType,NovaSeq X Plus
[Reads]
151`),
			map[string]map[string]string{
				"Header": {
					"FileFormatVersion":  "2",
					"RunName":            "TestRun",
					"InstrumentPlatform": "NovaSeq",
					"InstrumentType":     "NovaSeq X Plus",
				},
				"Reads": {
					"151": "151",
				},
			},
			nil,
		},
		{
			"case 2",
			2,
			true,
			[]string{"Header", "Reads"},
			[]int{4, 4},
			[]SectionType{SettingsSection, SettingsSection},
			[]byte(`[Header]
FileFormatVersion,2
RunName,TestRun
InstrumentPlatform,NovaSeq
InstrumentType,NovaSeq X Plus
[Reads]
Read1Cycles,151
Index1Cycles,8
Index2Cycles,8
Read2Cycles,151`),
			map[string]map[string]string{
				"Reads": {
					"Read1Cycles":  "151",
					"Index1Cycles": "8",
					"Index2Cycles": "8",
					"Read2Cycles":  "151",
				},
			},
			nil,
		},
		{
			"case 3",
			2,
			true,
			[]string{"Header", "Reads", "Data", "Sequencing_Settings"},
			[]int{4, 4, 5, 1},
			[]SectionType{SettingsSection, SettingsSection, DataSection, SettingsSection},
			[]byte(`[Header]
FileFormatVersion,2
RunName,TestRun
InstrumentPlatform,NovaSeq
InstrumentType,NovaSeq X Plus

[Reads]
Read1Cycles,151
Read2Cycles,151
Index1Cycles,8
Index2Cycles,8

[Data]
col1,col2,col3
val1.1,val2.1,val3.1
val1.2,val2.2,val3.2
val1.3,val2.3,val3.3
val1.4,val2.4,val3.4

[Sequencing_Settings]
LibraryPrepkits,prepkit1;prepkit2`),
			map[string]map[string]string{
				"Header": {
					"FileFormatVersion":  "2",
					"RunName":            "TestRun",
					"InstrumentPlatform": "NovaSeq",
					"InstrumentType":     "NovaSeq X Plus",
				},
				"Reads": {
					"Read1Cycles":  "151",
					"Read2Cycles":  "151",
					"Index1Cycles": "8",
					"Index2Cycles": "8",
				},
				"Sequencing_Settings": {
					"LibraryPrepkits": "prepkit1;prepkit2",
				},
			},
			map[string]map[string]map[int]string{
				"Data": {
					"col1": {
						1: "val1.1",
						2: "val1.2",
						3: "val1.3",
						4: "val1.4",
					},
					"col2": {
						1: "val2.1",
						2: "val2.2",
						3: "val2.3",
						4: "val2.4",
					},
					"col3": {
						1: "val3.1",
						2: "val3.2",
						3: "val3.3",
						4: "val3.4",
					},
				},
			},
		},
		{
			"trailing commas",
			2,
			true,
			[]string{"Header", "Reads", "Data", "Sequencing_Settings"},
			[]int{4, 4, 5, 1},
			[]SectionType{SettingsSection, SettingsSection, DataSection, SettingsSection},
			[]byte(`[Header]
FileFormatVersion,2
RunName,TestRun
InstrumentPlatform,NovaSeq
InstrumentType,NovaSeq X Plus

[Reads],
Read1Cycles,151
Read2Cycles,151
Index1Cycles,8
Index2Cycles,8

[Data],,
col1,col2,col3
val1.1,val2.1,val3.1
val1.2,val2.2,val3.2
val1.3,val2.3,val3.3
val1.4,val2.4,val3.4

[Sequencing_Settings],
LibraryPrepkits,prepkit1;prepkit2`),
			map[string]map[string]string{
				"Reads": {
					"Read1Cycles":  "151",
					"Read2Cycles":  "151",
					"Index1Cycles": "8",
					"Index2Cycles": "8",
				},
			},
			map[string]map[string]map[int]string{
				"Data": {
					"col1": {
						1: "val1.1",
						2: "val1.2",
						3: "val1.3",
						4: "val1.4",
					},
					"col2": {
						1: "val2.1",
						2: "val2.2",
						3: "val2.3",
						4: "val2.4",
					},
					"col3": {
						1: "val3.1",
						2: "val3.2",
						3: "val3.3",
						4: "val3.4",
					},
				},
			},
		},
		{
			"empty values",
			2,
			true,
			[]string{"Header", "Reads", "Data", "Sequencing_Settings"},
			[]int{4, 4, 5, 1},
			[]SectionType{SettingsSection, SettingsSection, DataSection, SettingsSection},
			[]byte(`[Header]
FileFormatVersion,2
RunName,TestRun
InstrumentPlatform,NovaSeq
InstrumentType,NovaSeq X Plus

[Reads],
Read1Cycles,151
Read2Cycles,151
Index1Cycles,8
Index2Cycles,8
,
[Data],,
col1,col2,col3
val1.1,val2.1,val3.1
val1.2,,val3.2
val1.3,val2.3,
val1.4,val2.4,val3.4

[Sequencing_Settings],
LibraryPrepkits,prepkit1;prepkit2`),
			map[string]map[string]string{
				"Reads": {
					"Read1Cycles":  "151",
					"Read2Cycles":  "151",
					"Index1Cycles": "8",
					"Index2Cycles": "8",
				},
			},
			map[string]map[string]map[int]string{
				"Data": {
					"col1": {
						1: "val1.1",
						2: "val1.2",
						3: "val1.3",
						4: "val1.4",
					},
					"col2": {
						1: "val2.1",
						2: "",
						3: "val2.3",
						4: "val2.4",
					},
					"col3": {
						1: "val3.1",
						2: "val3.2",
						3: "",
						4: "val3.4",
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			r := bufio.NewReader(bytes.NewReader(c.Data))
			s, err := ParseSampleSheet(r)
			if err != nil {
				t.Fatalf("%s", err)
			}
			if s.Version() != c.Version {
				t.Errorf("expected version %d, found %d", c.Version, s.Version())
			}
			if len(s.Sections) != len(c.Sections) {
				t.Errorf("expected %d sections, found %d", len(c.Sections), len(s.Sections))
			}
			if err := s.Validate(); err != nil && c.Valid {
				t.Error("expected valid sample sheet, but is invalid")
			} else if err == nil && !c.Valid {
				t.Error("expected invalid sample sheet, but is valid")
			}
			for i, section := range c.Sections {
				ssSection := s.Section(section)
				if ssSection.Name != c.Sections[i] {
					t.Errorf("expected section %s, found %s", c.Sections[i], ssSection.Name)
				}
				if ssSection.Type != c.SectionTypes[i] {
					t.Errorf("expected section type %d, found %d", c.SectionTypes[i], ssSection.Type)
				}
				if len(ssSection.Rows) != c.SectionRows[i] {
					t.Errorf("expected %d rows, found %d", c.SectionRows[i], len(ssSection.Rows))
				}
			}
			for sName, sData := range c.ExpectedSettingValues {
				for k, v := range sData {
					observedValue, _ := s.Section(sName).Get(k)
					if observedValue != v {
						t.Errorf("expected %s %s to be %s, found %s", sName, k, v, observedValue)
					}
				}
			}
			for sName, sData := range c.ExpectedDataValues {
				for k, indexes := range sData {
					for i, v := range indexes {
						observedValue, _ := s.Section(sName).Get(k, i)
						if observedValue != v {
							t.Errorf("expected %s %s[%d] to be %s, found %s", sName, k, i, v, observedValue)
						}
					}
				}
			}
		})
	}
}

func TestMalformedSampleSheets(t *testing.T) {
	cases := []struct {
		Name  string
		Data  []byte
		Error string
	}{
		{
			"trailing characters after header",
			[]byte(`[Header] no text should be here`),
			`parsing error: trailing characters after header "Header"`,
		},
		{
			"no header at start of file",
			[]byte(`key1,val1
key2,val2`),
			`parsing error: expected section header`,
		},
		{
			"empty section",
			[]byte(`[Header]
[Reads]
key1,val1
key2,val2
`),
			`parsing error: empty section`,
		},
		{
			"empty header",
			[]byte(`[]`),
			"parsing error: empty section header",
		},
		{
			"differing number of items per row",
			[]byte(`[Header]
key1,val1
key2
`),
			`parsing error: expected 2 items per row in section "Header"`,
		},
		{
			"too many items for settings",
			[]byte(`[Header]
key1,val1,val2
`),
			`parsing error: expected at most 2 items per row in section "Header"`,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			r := bufio.NewReader(bytes.NewReader(c.Data))
			_, err := ParseSampleSheet(r)
			if err == nil {
				t.Error("expected error, got nothing")
			} else if err.Error() != c.Error {
				t.Errorf("expected error: %q, got: %q", c.Error, err.Error())
			}
		})
	}
}

func TestGettingNonexistentValues(t *testing.T) {
	r := bufio.NewReader(bytes.NewReader([]byte(`[Header]
key1,val1
key2,val2
[Reads]
151
`)))
	s, err := ParseSampleSheet(r)
	if err != nil {
		t.Fatalf("%s", err)
	}

	v, err := s.Section("Header").Get("key3")
	if err == nil {
		t.Error("expected not ok, got ok")
	}
	if v != "" {
		t.Errorf("expected empty string, got %s", v)
	}
}

func TestReadSampleSheet(t *testing.T) {
	cases := []struct {
		Name     string
		Filename string
	}{
		{
			"novaseq",
			"/home/nima18/git/cleve/test_data/novaseq_full/SampleSheet.csv",
		},
		{
			"nextseq",
			"/home/nima18/git/cleve/test_data/nextseq1_full/SampleSheet.csv",
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			if _, err := os.Stat(c.Filename); errors.Is(err, os.ErrNotExist) {
				t.Skip("test data not found, skipping")
			}
			s, err := ReadSampleSheet(c.Filename)
			if err != nil {
				t.Errorf("unexpected error: %s", err)
			}

			foundHeader := false
			for _, section := range s.Sections {
				if section.Name == "Header" {
					foundHeader = true
					if section.Type != SettingsSection {
						t.Errorf("expected settings section, got %s", section.Type)
					}
				}
			}

			if !foundHeader {
				t.Error(`did not find a "Header" section`)
			}
		})
	}
}

func TestSectionType(t *testing.T) {
	cases := []struct {
		String string
		Type   SectionType
	}{
		{
			"settings",
			SettingsSection,
		},
		{
			"data",
			DataSection,
		},
		{
			"unknown",
			UnknownSection,
		},
	}

	for _, c := range cases {
		t.Run(c.String, func(t *testing.T) {
			if c.Type.String() != c.String {
				t.Errorf(`expected %q, got %q`, c.String, c.Type)
			}
			b, err := c.Type.MarshalJSON()
			if err != nil {
				t.Fatalf("failed to marshal into JSON: %s", err.Error())
			}
			if string(b) != fmt.Sprintf(`%q`, c.String) {
				t.Errorf(`expected %q, got %q`, c.String, string(b))
			}
		})
	}
}

func TestGetDataColumn(t *testing.T) {
	cases := []struct {
		name    string
		nrows   int
		section Section
		tests   map[string][]string
	}{
		{
			"data section",
			3,
			Section{
				Name: "Data",
				Type: DataSection,
				Rows: [][]string{
					{"col1", "col2", "col3"},
					{"val1.1", "val2.1", "val3.1"},
					{"val1.2", "val2.2", "val3.2"},
					{"val1.3", "val2.3", "val3.3"},
				},
			},
			map[string][]string{
				"col1": {"val1.1", "val1.2", "val1.3"},
				"col4": nil,
			},
		}, {
			"settings section",
			0,
			Section{
				Name: "Header",
				Type: SettingsSection,
				Rows: [][]string{
					{"FileFormatVersion", "2"},
					{"RunName", "TestRun"},
					{"InstrumentPlatform", "NovaSeq"},
					{"InstrumentType", "NovaSeq X Plus"},
				},
			},
			map[string][]string{
				"FileFormatVersion": nil,
				"RunName":           nil,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for colName, v := range c.tests {
				col, err := c.section.GetColumn(colName)
				if err != nil {
					t.Logf("got error: %s", err)
					if v != nil {
						t.Fatal(err.Error())
					}
				}
				if v != nil && len(col) != c.nrows {
					t.Errorf("expected %d rows, got %d", c.nrows, len(col))
				}
				if !reflect.DeepEqual(col, v) {
					t.Errorf("expected column %v, got %v", v, col)
				}
			}
		})
	}
}

func TestMostRecentSamplesheet(t *testing.T) {
	cases := []struct {
		name      string
		filenames []string
		modtimes  []time.Time
	}{
		// The first samplesheet should be the most recent for the test to work properly
		{
			"no samplesheet",
			[]string{},
			[]time.Time{},
		},
		{
			"single samplesheet",
			[]string{"SampleSheet.csv"},
			[]time.Time{time.Now()},
		},
		{
			"two samplesheets",
			[]string{
				"SampleSheet_updated.csv",
				"SampleSheet.csv",
			},
			[]time.Time{
				time.Now(),
				time.Now().Add(-2 * time.Hour),
			},
		},
		{
			"three samplesheets",
			[]string{
				"SampleSheet_updated.final.csv",
				"SampleSheet.csv",
				"SampleSheet_updated.csv",
			},
			[]time.Time{
				time.Now(),
				time.Now().Add(-2 * time.Hour),
				time.Now().Add(-2 * time.Second),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			for i, fname := range c.filenames {
				ssPath := filepath.Join(dir, fname)
				_, err := os.Create(ssPath)
				if err != nil {
					t.Fatal(err.Error())
				}
				if os.Chtimes(ssPath, c.modtimes[i], c.modtimes[i]) != nil {
					t.Fatal(err.Error())
				}
			}
			ss, err := MostRecentSamplesheet(dir)
			if err != nil && err.Error() != "no samplesheet found" {
				t.Fatal(err.Error())
			}
			if len(c.filenames) > 0 && ss != filepath.Join(dir, c.filenames[0]) {
				t.Errorf(`expected to get "%s", got "%s"`, c.filenames[0], ss)
			}
		})
	}
}

func TestSampleSheetUUID(t *testing.T) {
	cases := []struct {
		name    string
		data    []byte
		hasUuid bool
	}{
		{
			name: "SampleSheet with UUID",
			data: []byte(`[Header]
FileFormatVersion,2
RunName,TestRun
RunDescription,91f48115-71a2-41ba-843e-a4803542ec5c
InstrumentPlatform,NovaSeq
InstrumentType,NovaSeq X Plus
[Reads]
151
151`),
			hasUuid: true,
		},
		{
			name: "SampleSheet without UUID",
			data: []byte(`[Header]
FileFormatVersion,2
RunName,TestRun
RunDescription,WGS on a number of samples
InstrumentPlatform,NovaSeq
InstrumentType,NovaSeq X Plus
[Reads]
151
151`),
			hasUuid: false,
		},
		{
			name: "SampleSheet without RunDescription",
			data: []byte(`[Header]
FileFormatVersion,2
RunName,TestRun
InstrumentPlatform,NovaSeq
InstrumentType,NovaSeq X Plus
[Reads]
151
151`),
			hasUuid: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := bufio.NewReader(bytes.NewReader(c.data))
			ss, err := ParseSampleSheet(r)
			if err != nil {
				t.Fatal(err)
			}
			if ss.UUID == nil && c.hasUuid {
				t.Error("expected UUID but found nil")
			}
			if ss.UUID != nil && !c.hasUuid {
				t.Error("found UUID, but expected nil")
			}
		})
	}
}

func TestLastModified(t *testing.T) {
	testcases := []struct {
		name        string
		times       []time.Time
		expected    time.Time
		shouldError bool
	}{
		{
			name: "one file",
			times: []time.Time{
				time.Date(2024, 9, 18, 12, 8, 0, 0, time.Local),
			},
			expected: time.Date(2024, 9, 18, 12, 8, 0, 0, time.Local),
		},
		{
			name: "two files",
			times: []time.Time{
				time.Date(2024, 9, 18, 12, 8, 0, 0, time.Local),
				time.Date(2024, 9, 18, 13, 8, 0, 0, time.Local),
			},
			expected: time.Date(2024, 9, 18, 13, 8, 0, 0, time.Local),
		},
		{
			name: "two files reversed",
			times: []time.Time{
				time.Date(2024, 9, 18, 13, 8, 0, 0, time.Local),
				time.Date(2024, 9, 18, 12, 8, 0, 0, time.Local),
			},
			expected: time.Date(2024, 9, 18, 13, 8, 0, 0, time.Local),
		},
		{
			name:        "no files",
			times:       []time.Time{},
			expected:    time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC),
			shouldError: true,
		},
		{
			name:        "nil files",
			times:       nil,
			expected:    time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC),
			shouldError: true,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			ss := SampleSheet{}
			if testcase.times != nil {
				ss.Files = make([]SampleSheetInfo, 0)
				for _, mt := range testcase.times {
					ss.Files = append(ss.Files, SampleSheetInfo{
						ModificationTime: mt,
					})
				}
			}

			observed, err := ss.LastModified()
			if err != nil && !testcase.shouldError {
				t.Fatal("got error, expected nil")
			}
			if err == nil && testcase.shouldError {
				t.Fatal("expected error, got nil")
			}

			if observed != testcase.expected {
				t.Errorf("expected %v, got %v", testcase.expected, observed)
			}
		})
	}
}

func TestMergeSampleSheets(t *testing.T) {
	run1_1 := "run1"
	run1_2 := "run1"
	// run2_1 := "run2"
	run2_2 := "run2"

	uuid1_1, _ := uuid.NewUUID()
	uuid1_2, _ := uuid.Parse(uuid1_1.String())
	uuid2_1, _ := uuid.NewUUID()

	path1 := "/path/to/samplesheets/SampleSheet.csv"
	path2 := "/path/to/run1/SampleSheet.csv"
	// path3 := "/path/to/run2/SampleSheet.csv"

	older := time.Now()
	newer := time.Now()

	testCases := []struct {
		name              string
		sampleSheet       SampleSheet
		otherSampleSheet  SampleSheet
		mergedSampleSheet SampleSheet
		shouldError       bool
	}{
		{
			name: "identical sections, other is newer",
			sampleSheet: SampleSheet{
				RunID: &run1_1,
				UUID:  &uuid1_1,
				Files: []SampleSheetInfo{
					{
						Path:             path1,
						ModificationTime: older,
					},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run1"},
							{"RunDescription", "this is run 1"},
						},
					},
				},
			},
			otherSampleSheet: SampleSheet{
				RunID: &run1_2,
				UUID:  &uuid1_2,
				Files: []SampleSheetInfo{
					{
						Path:             path2,
						ModificationTime: newer,
					},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run1"},
							{"RunDescription", "this is run 1"},
						},
					},
				},
			},
			mergedSampleSheet: SampleSheet{
				RunID: &run1_1,
				UUID:  &uuid1_1,
				Files: []SampleSheetInfo{
					{
						Path:             path1,
						ModificationTime: older,
					},
					{
						Path:             path2,
						ModificationTime: newer,
					},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run1"},
							{"RunDescription", "this is run 1"},
						},
					},
				},
			},
		},
		{
			name: "identical sections and file names, other is newer",
			sampleSheet: SampleSheet{
				RunID: &run1_1,
				UUID:  &uuid1_1,
				Files: []SampleSheetInfo{
					{
						Path:             path1,
						ModificationTime: older,
					},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run1"},
							{"RunDescription", "this is run 1"},
						},
					},
				},
			},
			otherSampleSheet: SampleSheet{
				RunID: &run1_2,
				UUID:  &uuid1_2,
				Files: []SampleSheetInfo{
					{
						Path:             path1,
						ModificationTime: newer,
					},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run1"},
							{"RunDescription", "this is run 1"},
						},
					},
				},
			},
			mergedSampleSheet: SampleSheet{
				RunID: &run1_1,
				UUID:  &uuid1_1,
				Files: []SampleSheetInfo{
					{
						Path:             path1,
						ModificationTime: newer,
					},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run1"},
							{"RunDescription", "this is run 1"},
						},
					},
				},
			},
		},
		{
			name: "other has more sections",
			sampleSheet: SampleSheet{
				RunID: &run1_1,
				Files: []SampleSheetInfo{
					{Path: path1, ModificationTime: older},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run1"},
							{"RunDescription", "this is run 1"},
						},
					},
				},
			},
			otherSampleSheet: SampleSheet{
				RunID: &run1_2,
				Files: []SampleSheetInfo{
					{Path: path2, ModificationTime: newer},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run1"},
							{"RunDescription", "this is run 1"},
						},
					},
					{
						Name: "Reads",
						Rows: [][]string{
							{"151"},
							{"151"},
						},
					},
				},
			},
			mergedSampleSheet: SampleSheet{
				RunID: &run1_1,
				Files: []SampleSheetInfo{
					{Path: path1, ModificationTime: older},
					{Path: path2, ModificationTime: newer},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run1"},
							{"RunDescription", "this is run 1"},
						},
					},
					{
						Name: "Reads",
						Rows: [][]string{
							{"151"},
							{"151"},
						},
					},
				},
			},
		},
		{
			name: "this has more sections",
			sampleSheet: SampleSheet{
				RunID: &run1_1,
				Files: []SampleSheetInfo{
					{Path: path1, ModificationTime: older},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run1"},
							{"RunDescription", "this is run 1"},
						},
					},
					{
						Name: "Reads",
						Rows: [][]string{
							{"151"},
							{"151"},
						},
					},
				},
			},
			otherSampleSheet: SampleSheet{
				RunID: &run1_2,
				Files: []SampleSheetInfo{
					{Path: path2, ModificationTime: newer},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run1"},
							{"RunDescription", "this is run 1"},
						},
					},
				},
			},
			mergedSampleSheet: SampleSheet{
				RunID: &run1_1,
				Files: []SampleSheetInfo{
					{Path: path1, ModificationTime: older},
					{Path: path2, ModificationTime: newer},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run1"},
							{"RunDescription", "this is run 1"},
						},
					},
					{
						Name: "Reads",
						Rows: [][]string{
							{"151"},
							{"151"},
						},
					},
				},
			},
		},
		{
			name: "section has been updated",
			sampleSheet: SampleSheet{
				RunID: &run1_1,
				Files: []SampleSheetInfo{
					{Path: path1, ModificationTime: older},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run1"},
							{"RunDescription", "this is run 1"},
						},
					},
				},
			},
			otherSampleSheet: SampleSheet{
				RunID: &run1_2,
				Files: []SampleSheetInfo{
					{Path: path2, ModificationTime: newer},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run1"},
							{"RunDescription", "this is a better run 1"},
						},
					},
				},
			},
			mergedSampleSheet: SampleSheet{
				RunID: &run1_1,
				Files: []SampleSheetInfo{
					{Path: path1, ModificationTime: older},
					{Path: path2, ModificationTime: newer},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run1"},
							{"RunDescription", "this is a better run 1"},
						},
					},
				},
			},
		},
		{
			name: "section has been updated this is newer",
			sampleSheet: SampleSheet{
				RunID: &run1_1,
				Files: []SampleSheetInfo{
					{Path: path1, ModificationTime: newer},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run1"},
							{"RunDescription", "this is run 1"},
						},
					},
				},
			},
			otherSampleSheet: SampleSheet{
				RunID: &run1_2,
				Files: []SampleSheetInfo{
					{Path: path2, ModificationTime: older},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run1"},
							{"RunDescription", "this is a better run 1"},
						},
					},
					{
						Name: "Reads",
						Rows: [][]string{
							{"151"},
							{"151"},
						},
					},
				},
			},
			mergedSampleSheet: SampleSheet{
				RunID: &run1_1,
				Files: []SampleSheetInfo{
					{Path: path2, ModificationTime: older},
					{Path: path1, ModificationTime: newer},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run1"},
							{"RunDescription", "this is run 1"},
						},
					},
					{
						Name: "Reads",
						Rows: [][]string{
							{"151"},
							{"151"},
						},
					},
				},
			},
		},
		{
			name: "run ids are different",
			sampleSheet: SampleSheet{
				RunID: &run1_1,
				Files: []SampleSheetInfo{
					{Path: path1, ModificationTime: newer},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run1"},
							{"RunDescription", "this is run 1"},
						},
					},
				},
			},
			otherSampleSheet: SampleSheet{
				RunID: &run2_2,
				Files: []SampleSheetInfo{
					{Path: path2, ModificationTime: older},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run2"},
							{"RunDescription", "this is a better run 1"},
						},
					},
					{
						Name: "Reads",
						Rows: [][]string{
							{"151"},
							{"151"},
						},
					},
				},
			},
			shouldError: true,
		},
		{
			name: "uuids are different",
			sampleSheet: SampleSheet{
				RunID: &run1_1,
				UUID:  &uuid1_1,
				Files: []SampleSheetInfo{
					{Path: path1, ModificationTime: newer},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run1"},
							{"RunDescription", "this is run 1"},
						},
					},
				},
			},
			otherSampleSheet: SampleSheet{
				RunID: &run1_2,
				UUID:  &uuid2_1,
				Files: []SampleSheetInfo{
					{Path: path2, ModificationTime: older},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run2"},
							{"RunDescription", "this is a better run 1"},
						},
					},
					{
						Name: "Reads",
						Rows: [][]string{
							{"151"},
							{"151"},
						},
					},
				},
			},
			shouldError: true,
		},
		{
			name: "should allow merging if this run id is nil",
			sampleSheet: SampleSheet{
				RunID: nil,
				Files: []SampleSheetInfo{
					{Path: path1, ModificationTime: newer},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run1"},
							{"RunDescription", "this is run 1"},
						},
					},
				},
			},
			otherSampleSheet: SampleSheet{
				RunID: &run2_2,
				Files: []SampleSheetInfo{
					{Path: path2, ModificationTime: older},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run2"},
							{"RunDescription", "this is a better run 1"},
						},
					},
					{
						Name: "Reads",
						Rows: [][]string{
							{"151"},
							{"151"},
						},
					},
				},
			},
			mergedSampleSheet: SampleSheet{
				RunID: &run2_2,
				Files: []SampleSheetInfo{
					{Path: path2, ModificationTime: older},
					{Path: path1, ModificationTime: newer},
				},
				Sections: []Section{
					{
						Name: "Header",
						Rows: [][]string{
							{"RunName", "run1"},
							{"RunDescription", "this is run 1"},
						},
					},
					{
						Name: "Reads",
						Rows: [][]string{
							{"151"},
							{"151"},
						},
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			m, err := testCase.sampleSheet.Merge(&testCase.otherSampleSheet)
			if err != nil && !testCase.shouldError {
				t.Fatalf("merging should not have errored: %s", err.Error())
			}
			if err == nil && testCase.shouldError {
				t.Fatal("merging should have errored")
			}

			if !testCase.shouldError && !reflect.DeepEqual(m, &testCase.mergedSampleSheet) {
				t.Errorf("merging did not result in the expected samplesheet.\n\nExpected: %+v\n\nGot: %+v", &testCase.mergedSampleSheet, m)
			}
		})
	}
}

func TestSampleMetadata(t *testing.T) {
	testcases := []struct {
		name    string
		content string
		valid   bool
	}{
		{
			name:    "valid metadata",
			content: "[Header]\nRunName,run1\n[Reads]\n151\n151\n[cleve_data]\nsample_id,sample_name,seq_type,pipeline,pipeline_version,destination",
			valid:   true,
		},
		{
			name:    "valid unordered metadata with extra columns",
			content: "[Header]\nRunName,run1\n[Reads]\n151\n151\n[cleve_data]\nsample_id,reference,owner,seq_type,pipeline,pipeline_version,destination,sample_name",
			valid:   true,
		},
		{
			name:    "invalid metadata",
			content: "[Header]\nRunName,run1\n[Reads]\n151\n151\n[cleve_data]\nsample_id,sample_name,reference",
			valid:   false,
		},
	}

	dir := t.TempDir()

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			sspath := filepath.Join(dir, "SampleSheet.csv")
			err := os.WriteFile(sspath, []byte(testcase.content), 0666)
			if err != nil {
				t.Fatal(err)
			}
			_, err = ReadSampleSheet(sspath)
			if err != nil && testcase.valid {
				t.Fatalf("got error, expected nil: %s", err.Error())
			}
			if err == nil && !testcase.valid {
				t.Fatal("got nil, expected error")
			}
		})
	}
}
