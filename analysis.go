package cleve

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type AnalysisFileType int

const (
	FileVcf AnalysisFileType = iota
	FileBam
	FileSnvVcf
	FileSvVcf
	FileFastq
)

type AnalysisFile struct {
	// Path is a relative path to the file within the analysis directory.
	Path     string
	FileType AnalysisFileType
}

type AnalysisLevel int

const (
	LevelRun AnalysisLevel = iota
	LevelCase
	LevelSample
)

type Analysis struct {
	AnalysisId      string         `bson:"analysis_id" json:"analysis_id"`
	Level           AnalysisLevel  `bson:"level" json:"level"`
	Path            string         `bson:"path" json:"path"`
	Software        string         `bson:"software" json:"software"`
	SoftwareVersion string         `bson:"software_version" json:"software_version"`
	StateHistory    StateHistory   `bson:"state_history" json:"state_history"`
	Files           []AnalysisFile `bson:"files" json:"files"`
}

// GetFiles returns all paths to files of a particular type associated with an analysis.
// If there are no such files, and empty slice is returned.
func (a *Analysis) GetFiles(t AnalysisFileType) []string {
	var files []string
	for _, f := range a.Files {
		if f.FileType == t {
			files = append(files, filepath.Join(a.Path, f.Path))
		}
	}
	return files
}

type DragenAnalysisSummary struct {
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

func (s *DragenAnalysisSummary) State() State {
	switch s.Result {
	case "success":
		return StateComplete
	default:
		return StateUnknown
	}
}

func ParseDragenAnalysisSummary(r io.Reader) (DragenAnalysisSummary, error) {
	var summary DragenAnalysisSummary
	data, err := io.ReadAll(r)
	if err != nil {
		return summary, err
	}
	err = json.Unmarshal(data, &summary)
	return summary, err
}

// NewDragenAnalysis creates a new Analysis representing a Dragen analysis,
// specifically the results from BCLConvert.
func NewDragenAnalysis(path string, run *Run) (Analysis, error) {
	state := dragenAnalysisState(path)
	id := run.RunID + "_" + filepath.Base(path) + "_bclconvert"
	analysis := Analysis{
		AnalysisId: id,
		Level:      LevelRun,
		Software:   "Dragen BCLConvert",
		Path:       path,
	}
	var dragenVersion string
	for _, sw := range run.RunParameters.Software {
		if strings.ToLower(sw.Name) != "Dragen" {
			continue
		}
		dragenVersion = sw.Version
		break
	}
	if dragenVersion == "" {
		return analysis, fmt.Errorf("failed to identify dragen version")
	}
	analysis.SoftwareVersion = dragenVersion
	var summary DragenAnalysisSummary
	if state == StateReady {
		f, err := os.Open("Data/" + dragenVersion + "/detailed_summary.json")
		if err != nil {
			return analysis, err
		}
		summary, err = ParseDragenAnalysisSummary(f)
		if err != nil {
			return analysis, err
		}
	}
	switch summary.Result {
	case "success":
		state = StateReady
	case "error":
		state = StateError
	default:
		state = StateUnknown
	}

	if state == StateReady {
		// Add the fastq files to the analysis
		f, err := os.Open(filepath.Join(analysis.Path, "Manifest.tsv"))
		if err != nil {
			return analysis, err
		}
		defer func() { _ = f.Close() }()
		manifest, err := readDragenManifest(f)
		if err != nil {
			return analysis, fmt.Errorf("failed to read dragen manifest: %w", err)
		}
		fqRegex := regexp.MustCompile(`\.f(ast)?q(\.gz)?$`)
		for _, f := range manifest {
			if fqRegex.MatchString(f) {
				analysis.Files = append(analysis.Files, AnalysisFile{
					Path:     f,
					FileType: FileFastq,
				})
			}
		}
	}

	analysis.StateHistory.Add(state)
	return analysis, nil
}

// dragenAnalysisState identifies the state of a Dragen analysis. This is just
// a temporary state indicating whether the data is avaliable. The analysis could
// still be in a bad/incomplete state, and this has to be checked downstream.
func dragenAnalysisState(path string) State {
	copyComplete := filepath.Join(path, "CopyComplete.txt")
	analysisComplete := filepath.Join(path, "Data", "Secondary_Analysis_Complete.txt")
	if _, err := os.Stat(copyComplete); os.IsNotExist(err) {
		return StatePending
	}
	if _, err := os.Stat(analysisComplete); os.IsNotExist(err) {
		return StatePending
	}
	return StateReady
}

// readDragenManifest reads a Dragen analysis manifest file and returns a slice of
// strings with all paths listed in the manifest.
func readDragenManifest(r io.Reader) ([]string, error) {
	var files []string
	csvReader := csv.NewReader(r)
	csvReader.Comma = '\t'
	csvReader.FieldsPerRecord = 2
	lines, err := csvReader.ReadAll()
	if err != nil {
		return files, err
	}
	for _, line := range lines {
		files = append(files, line[0])
	}
	return files, nil
}
