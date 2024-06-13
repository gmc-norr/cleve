package cleve

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
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
			false,
			[]string{"Header"},
			[]int{4},
			[]SectionType{SettingsSection},
			[]byte(`[Header]
FileFormatVersion,2
RunName,TestRun
InstrumentPlatform,NovaSeq
InstrumentType,NovaSeq X Plus`),
			map[string]map[string]string{
				"Header": {
					"FileFormatVersion":  "2",
					"RunName":            "TestRun",
					"InstrumentPlatform": "NovaSeq",
					"InstrumentType":     "NovaSeq X Plus",
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
			if s.IsValid() != c.Valid {
				t.Errorf("expected valid %t, found %t", c.Valid, s.IsValid())
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
`)))
	s, err := ParseSampleSheet(r)
	if err != nil {
		t.Fatalf("%s", err)
	}

	v, ok := s.Section("Header").Get("key3")
	if ok {
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
			"test_data/novaseq_full/SampleSheet.csv",
		},
		{
			"nextseq",
			"test_data/nextseq1_full/SampleSheet.csv",
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
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
