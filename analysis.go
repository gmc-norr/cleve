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

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

type AnalysisFileType int

const (
	_ AnalysisFileType = iota
	FileVcf
	FileBam
	FileSnvVcf
	FileSvVcf
	FileFastq
	FileText
)

var validAnalysisFileTypes = map[string]AnalysisFileType{
	"vcf":     FileVcf,
	"bam":     FileBam,
	"vcf_snv": FileSnvVcf,
	"vcf_sv":  FileSvVcf,
	"fastq":   FileFastq,
	"text":    FileText,
}

func (t AnalysisFileType) String() string {
	switch t {
	case FileVcf:
		return "vcf"
	case FileBam:
		return "bam"
	case FileSnvVcf:
		return "vcf_snv"
	case FileSvVcf:
		return "vcf_sv"
	case FileFastq:
		return "fastq"
	case FileText:
		return "text"
	default:
		return ""
	}
}

func (t AnalysisFileType) IsValid() bool {
	return t > 0 && t <= FileText
}

func AnalysisFileTypeFromString(stringType string) AnalysisFileType {
	t, ok := validAnalysisFileTypes[stringType]
	if !ok {
		return 0
	}
	return t
}

func (ft AnalysisFileType) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bson.MarshalValue(ft.String())
}

func (ft *AnalysisFileType) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	var typeString string
	if err := bson.UnmarshalValue(t, data, &typeString); err != nil {
		return err
	}
	fileType := AnalysisFileTypeFromString(typeString)
	if !t.IsValid() {
		return fmt.Errorf("invalid analysis file type: %q", typeString)
	}
	*ft = fileType
	return nil
}

func (ft AnalysisFileType) MarshalJSON() ([]byte, error) {
	return json.Marshal(ft.String())
}

func (ft *AnalysisFileType) UnmarshalJSON(data []byte) error {
	var typeString string
	if err := json.Unmarshal(data, &typeString); err != nil {
		return err
	}
	fileType := AnalysisFileTypeFromString(typeString)
	if !fileType.IsValid() {
		return fmt.Errorf("invalid analysis file type: %q", typeString)
	}
	*ft = fileType
	return nil
}

type TextFileOptions struct {
	Format    string
	Delimiter string
	Columns   string
}

type AnalysisFile struct {
	// Path is a relative path to the file within the analysis directory.
	Path     string           `bson:"path" json:"path"`
	FileType AnalysisFileType `bson:"type" json:"type"`
	Level    AnalysisLevel    `bson:"level" json:"level"`
	ParentId string           `bson:"parent_id" json:"parent_id"`
}

type AnalysisFileFilter struct {
	AnalysisId string
	FileType   AnalysisFileType
	Level      AnalysisLevel
	ParentId   string
	Pattern    *regexp.Regexp
}

func (f *AnalysisFileFilter) Apply(file AnalysisFile) bool {
	pass := true
	if f.FileType.IsValid() && f.FileType != file.FileType {
		return false
	}
	if f.Level.IsValid() && f.Level != file.Level {
		return false
	}
	if f.ParentId != "" && f.ParentId != file.ParentId {
		return false
	}
	if f.Pattern != nil {
		pass = f.Pattern.Match([]byte(file.Path))
	}
	return pass
}

type AnalysisLevel int

const (
	_ AnalysisLevel = iota
	LevelRun
	LevelCase
	LevelSample
)

// IsValid returns true if the AnalysisLevel l represents a valid level.
func (l AnalysisLevel) IsValid() bool {
	return l > 0 && l <= LevelSample
}

func (l AnalysisLevel) String() string {
	switch l {
	case LevelRun:
		return "run"
	case LevelCase:
		return "case"
	case LevelSample:
		return "sample"
	default:
		return ""
	}
}

func AnalysisLevelFromString(level string) (AnalysisLevel, error) {
	switch level {
	case "run":
		return LevelRun, nil
	case "case":
		return LevelCase, nil
	case "sample":
		return LevelSample, nil
	default:
		return 0, fmt.Errorf("invalid analysis level %q", level)
	}
}

func (l *AnalysisLevel) UnmarshalParam(param string) error {
	level, err := AnalysisLevelFromString(param)
	if err != nil {
		return err
	}
	*l = level
	return nil
}

func (l AnalysisLevel) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bson.MarshalValue(l.String())
}

func (l *AnalysisLevel) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	var ls string
	err := bson.UnmarshalValue(bson.TypeString, data, &ls)
	if err != nil {
		return fmt.Errorf("unmarshal failed: %w", err)
	}
	level, err := AnalysisLevelFromString(ls)
	if err != nil {
		return err
	}
	*l = level
	return nil
}

func (l *AnalysisLevel) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.String())
}

func (l *AnalysisLevel) UnmarshalJSON(data []byte) error {
	var ls string
	if err := json.Unmarshal(data, &ls); err != nil {
		return err
	}
	level, err := AnalysisLevelFromString(ls)
	if err != nil {
		return err
	}
	*l = level
	return nil
}

type AnalysisResult struct {
	PaginationMetadata `bson:"metadata" json:"metadata"`
	Analyses           []*Analysis `bson:"analyses" json:"analyses"`
}

type Analysis struct {
	AnalysisId      string               `bson:"analysis_id" json:"analysis_id"`
	Runs            []string             `bson:"runs" json:"runs"`
	Path            string               `bson:"path" json:"path"`
	Software        string               `bson:"software" json:"software"`
	SoftwareVersion string               `bson:"software_version" json:"software_version"`
	StateHistory    StateHistory         `bson:"state_history" json:"state_history"`
	InputFiles      []AnalysisFileFilter `bson:"input_files" json:"input_files"`
	OutputFiles     []AnalysisFile       `bson:"files" json:"files"`
}

// GetFiles returns all paths to analysis output files of a particular type that are
// associated with the analysis. If there are no such files, and empty slice is returned.
func (a *Analysis) GetFiles(filter AnalysisFileFilter) []string {
	var files []string
	for _, f := range a.OutputFiles {
		if filter.Apply(f) {
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

// NewDragenAnalysis creates a slice of Analysis representing a Dragen analysis,
// specifically the results from BCLConvert. The first element in the slice will
// represent all samples in the sequencing run, and the rest of the analyses represent
// each individual sample in run.
func NewDragenAnalysis(path string, run *Run) (Analysis, error) {
	state := dragenAnalysisState(path)
	id := run.RunID + "_" + filepath.Base(path) + "_bclconvert"
	analysis := Analysis{
		AnalysisId: id,
		Runs:       []string{run.RunID},
		Software:   "Dragen BCLConvert",
		Path:       path,
	}
	var dragenVersion string
	for _, sw := range run.RunParameters.Software {
		if strings.ToLower(sw.Name) != "dragen" {
			continue
		}
		dragenVersion = sw.Version
		break
	}
	if dragenVersion == "" {
		return analysis, fmt.Errorf("failed to identify dragen version")
	}
	analysis.SoftwareVersion = dragenVersion

	if state != StateReady {
		analysis.StateHistory.Add(state)
		return analysis, nil
	}

	var summary DragenAnalysisSummary
	f, err := os.Open(filepath.Join(analysis.Path, "Data", "summary", dragenVersion, "detailed_summary.json"))
	if err != nil {
		return analysis, err
	}
	summary, err = ParseDragenAnalysisSummary(f)
	if err != nil {
		return analysis, err
	}

	switch summary.Result {
	case "success":
		state = StateReady
	case "error":
		// TODO: I actually don't know what values this can take.
		state = StateError
	default:
		state = StatePending
	}

	analysis.StateHistory.Add(state)

	if state == StateReady {
		// Add the stats files to the analysis
		analysis.OutputFiles = []AnalysisFile{
			{
				Path:     "Data/Demux/Demultiplex_Stats.csv",
				FileType: FileText,
				Level:    LevelRun,
			},
			{
				Path:     "Data/Demux/Index_Hopping_Counts.csv",
				FileType: FileText,
				Level:    LevelRun,
			},
			{
				Path:     "Data/Demux/Top_Unknown_Barcodes.csv",
				FileType: FileText,
				Level:    LevelRun,
			},
		}

		f, err := os.Open(filepath.Join(analysis.Path, "Manifest.tsv"))
		if err != nil {
			return analysis, err
		}
		defer func() { _ = f.Close() }()
		manifest, err := ReadDragenManifest(f)
		if err != nil {
			return analysis, fmt.Errorf("failed to read dragen manifest: %w", err)
		}

		for _, wf := range summary.Workflows {
			for _, sample := range wf.Samples {
				// Add the fastq files to the analysis
				fqRegex, err := regexp.Compile(`^` + regexp.QuoteMeta(sample.SampleID) + `.*\.f(ast)?q(\.gz)?$`)
				if err != nil {
					return analysis, fmt.Errorf("failed to compile regex for sample fastq files: %w", err)
				}
				for _, f := range manifest.FindFiles(fqRegex) {
					analysis.OutputFiles = append(analysis.OutputFiles, AnalysisFile{
						Path:     f,
						FileType: FileFastq,
						Level:    LevelSample,
						ParentId: sample.SampleID,
					})
				}
			}
		}
	}

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

type DragenManifest struct {
	Files []string
}

// ReadDragenManifest reads a Dragen analysis manifest file and returns a slice of
// strings with all paths listed in the manifest.
func ReadDragenManifest(r io.Reader) (DragenManifest, error) {
	var files []string
	csvReader := csv.NewReader(r)
	csvReader.Comma = '\t'
	csvReader.FieldsPerRecord = 2
	lines, err := csvReader.ReadAll()
	if err != nil {
		return DragenManifest{}, err
	}
	for _, line := range lines {
		files = append(files, line[0])
	}
	return DragenManifest{Files: files}, nil
}

// FindFiles returns a list of paths where the file name (not the full path) matches the
// supplied regular expression. If the regular expression is nil, or no files are found,
// an empty slice is returned.
func (m *DragenManifest) FindFiles(r *regexp.Regexp) []string {
	var matches []string
	if r == nil {
		return matches
	}
	for _, f := range m.Files {
		if r.MatchString(filepath.Base(f)) {
			matches = append(matches, f)
		}
	}
	return matches
}
