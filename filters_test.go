package cleve

import (
	"regexp"
	"testing"
)

func TestAnalysisFileFilter(t *testing.T) {
	testcases := []struct {
		name       string
		filter     AnalysisFileFilter
		isComplete bool
	}{
		{
			name: "complete filter filetype",
			filter: AnalysisFileFilter{
				AnalysisId: "run1_1_bclconvert",
				Level:      LevelSample,
				FileType:   FileFastq,
			},
			isComplete: true,
		},
		{
			name: "complete filter name",
			filter: AnalysisFileFilter{
				AnalysisId: "run1_1_bclconvert",
				Level:      LevelSample,
				Name:       "sample1.fastq.gz",
			},
			isComplete: true,
		},
		{
			name: "complete filter pattern",
			filter: AnalysisFileFilter{
				AnalysisId: "run1_1_bclconvert",
				Level:      LevelSample,
				Pattern:    regexp.MustCompile(`\.fastq\.gz$`),
			},
			isComplete: true,
		},
		{
			name: "complete filter parent id",
			filter: AnalysisFileFilter{
				AnalysisId: "run1_1_bclconvert",
				Level:      LevelSample,
				ParentId:   "sample1",
			},
			isComplete: true,
		},
		{
			name: "complete filter parent id filetype",
			filter: AnalysisFileFilter{
				AnalysisId: "run1_1_bclconvert",
				Level:      LevelSample,
				ParentId:   "sample1",
				FileType:   FileFastq,
			},
			isComplete: true,
		},
		{
			name: "incomplete filter analysis id",
			filter: AnalysisFileFilter{
				Level:    LevelSample,
				ParentId: "sample1",
				FileType: FileFastq,
			},
			isComplete: false,
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
			isComplete: false,
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			err := c.filter.IsValid()
			if c.isComplete != (err == nil) {
				if c.isComplete {
					t.Errorf("expected filter to be complete, but got error=%v", err)
				} else {
					t.Errorf("expected filter to be incomplete, but got error=%v", err)
				}
			}
		})
	}
}
