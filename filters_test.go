package cleve

import (
	"regexp"
	"testing"
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
				AnalysisId: "run1_1_bclconvert",
				Level:      LevelSample,
				FileType:   FileFastq,
			},
			isValid: true,
		},
		{
			name: "complete filter name",
			filter: AnalysisFileFilter{
				AnalysisId: "run1_1_bclconvert",
				Level:      LevelSample,
				Name:       "sample1.fastq.gz",
			},
			isValid: true,
		},
		{
			name: "complete filter pattern",
			filter: AnalysisFileFilter{
				AnalysisId: "run1_1_bclconvert",
				Level:      LevelSample,
				Pattern:    regexp.MustCompile(`\.fastq\.gz$`),
			},
			isValid: true,
		},
		{
			name: "complete filter parent id",
			filter: AnalysisFileFilter{
				AnalysisId: "run1_1_bclconvert",
				Level:      LevelSample,
				ParentId:   "sample1",
			},
			isValid: true,
		},
		{
			name: "complete filter parent id filetype",
			filter: AnalysisFileFilter{
				AnalysisId: "run1_1_bclconvert",
				Level:      LevelSample,
				ParentId:   "sample1",
				FileType:   FileFastq,
			},
			isValid: true,
		},
		{
			name: "incomplete filter analysis id",
			filter: AnalysisFileFilter{
				Level:    LevelSample,
				ParentId: "sample1",
				FileType: FileFastq,
			},
			isValid: false,
		},
		{
			name: "conflicting filter",
			filter: AnalysisFileFilter{
				AnalysisId: "run1_1_bclconvert",
				FileType:   FileFastq,
				Level:      LevelSample,
				ParentId:   "sample1",
				Name:       "sample1.fastq.gz",
				Pattern:    regexp.MustCompile(`\.fastq\.gz$`),
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
