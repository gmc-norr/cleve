package cleve

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/google/uuid"
)

func TestAnalysisFileFilter(t *testing.T) {
	testcases := []struct {
		name    string
		filter  AnalysisFileFilter
		isValid bool
	}{
		{
			name: "complete filter filetype",
			filter: AnalysisFileFilter{
				AnalysisId: uuid.New(),
				Level:      LevelSample,
				FileType:   FileFastq,
			},
			isValid: true,
		},
		{
			name: "complete filter name",
			filter: AnalysisFileFilter{
				AnalysisId: uuid.New(),
				Level:      LevelSample,
				Name:       "sample1.fastq.gz",
			},
			isValid: true,
		},
		{
			name: "complete filter pattern",
			filter: AnalysisFileFilter{
				AnalysisId: uuid.New(),
				Level:      LevelSample,
				Pattern:    regexp.MustCompile(`\.fastq\.gz$`),
			},
			isValid: true,
		},
		{
			name: "complete filter parent id",
			filter: AnalysisFileFilter{
				AnalysisId: uuid.New(),
				Level:      LevelSample,
				ParentId:   "sample1",
			},
			isValid: true,
		},
		{
			name: "complete filter parent id filetype",
			filter: AnalysisFileFilter{
				AnalysisId: uuid.New(),
				Level:      LevelSample,
				ParentId:   "sample1",
				FileType:   FileFastq,
			},
			isValid: true,
		},
		{
			name: "valid filter without analysis id",
			filter: AnalysisFileFilter{
				Level:    LevelSample,
				ParentId: "sample1",
				FileType: FileFastq,
			},
			isValid: true,
		},
		{
			name: "conflicting filter",
			filter: AnalysisFileFilter{
				AnalysisId: uuid.New(),
				FileType:   FileFastq,
				Level:      LevelSample,
				ParentId:   "sample1",
				Name:       "sample1.fastq.gz",
				Pattern:    regexp.MustCompile(`\.fastq\.gz$`),
			},
			isValid: false,
		},
		{
			name: "invalid analysis file type",
			filter: AnalysisFileFilter{
				Level:    LevelSample,
				FileType: AnalysisFileTypeFromString("blabla"),
			},
			isValid: false,
		},
		{
			name: "valid filter with zero analysis file type",
			filter: AnalysisFileFilter{
				Level:    LevelSample,
				ParentId: "sample1",
			},
			isValid: true,
		},
		{
			name: "invalid file type",
			filter: AnalysisFileFilter{
				Level:    LevelSample,
				FileType: AnalysisFileTypeFromString("blabla"),
				ParentId: "sample1",
			},
			isValid: false,
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			err := c.filter.Validate()
			if c.isValid != (err == nil) {
				if c.isValid {
					t.Errorf("expected filter to be valid, but got error=%v", err)
				} else {
					t.Errorf("expected filter to be invalid, but got error=%v", err)
				}
			}
		})
	}
}

func TestAnalysisFileFilterJSON(t *testing.T) {
	// Focus on the UUIDs in this test.
	testcases := []struct {
		name        string
		json        string
		filter      AnalysisFileFilter
		valid       bool
		shouldError bool
	}{
		{
			name: "valid analysis id",
			json: `{"analysis_id": "3b259e93-632d-47f2-b71f-453f01462a0d", "type": "fastq"}`,
			filter: AnalysisFileFilter{
				AnalysisId: uuid.MustParse("3b259e93-632d-47f2-b71f-453f01462a0d"),
				FileType:   FileFastq,
			},
			shouldError: false,
			valid:       true,
		},
		{
			name:        "invalid analysis id",
			json:        `{"analysis_id": "b259e93-632d-47f2-b71f-453f01462a0d", "type": "fastq"}`,
			shouldError: true,
			valid:       false,
		},
		{
			name:        "empty analysis id",
			json:        `{"analysis_id": ""}`,
			shouldError: true,
			valid:       false,
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			var filter AnalysisFileFilter
			err := json.Unmarshal([]byte(c.json), &filter)
			if err != nil {
				if !c.shouldError {
					t.Fatalf("failed to unmarshal json: %v", err)
				}
				t.Log(err)
			}
			if filter != c.filter {
				t.Error("filters are mismatching")
				t.Log(filter)
				t.Log(c.filter)
			}
			err = filter.Validate()
			if err != nil {
				if c.valid {
					t.Errorf("expected a valid filter, got an invalid filter")
				}
				t.Log(err)
			} else if !c.valid {
				t.Errorf("expected an invalid filter, got a valid filter")
			}
		})
	}
}
