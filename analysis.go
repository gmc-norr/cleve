package cleve

import (
	"encoding/json"
	"path/filepath"
)

type Analysis struct {
	AnalysisId string           `bson:"analysis_id" json:"analysis_id"`
	Path       string           `bson:"path" json:"path"`
	State      RunState         `bson:"state" json:"state"`
	Summary    *AnalysisSummary `bson:"summary" json:"summary"`
}

type AnalysisSummary struct {
	RunID           string `bson:"run_id" json:"run_id"`
	Result          string `bson:"result" json:"result"`
	SoftwareVersion string `bson:"software_version" json:"software_version"`
	Workflows       []struct {
		WorkflowName      string `bson:"workflow_name" json:"workflow_name"`
		ReportAggregation string `bson:"report_aggregation" json:"report_aggregation"`
		Samples           []struct {
			SampleID          string `bson:"sample_id" json:"sample_id"`
			BclToFastq        string `bson:"bcl_to_fastq" json:"bcl_to_fastq"`
			OraCompression    string `bson:"ora_compression" json:"ora_compression"`
			SecondaryAnalysis string `bson:"secondary_analysis" json:"secondary_analysis"`
			ReportGeneration  string `bson:"report_generation" json:"report_generation"`
		} `bson:"samples" json:"samples"`
	} `bson:"workflows" json:"workflows"`
}

func ParseAnalysisSummary(data []byte) (AnalysisSummary, error) {
	var summary AnalysisSummary
	err := json.Unmarshal(data, &summary)
	return summary, err
}

func NewAnalysis(path string, state RunState, data []byte) (Analysis, error) {
	var analysis Analysis
	summary, err := ParseAnalysisSummary(data)
	if err != nil {
		return analysis, err
	}
	analysis.AnalysisId = filepath.Base(path)
	analysis.Path = path
	analysis.State = state
	analysis.Summary = &summary
	return analysis, err
}
