package cleve

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/gmc-norr/cleve/interop"
	"go.mongodb.org/mongo-driver/bson"
)

type analysisDirOption func(*analysisDir)

type analysisDir struct {
	copyComplete     bool
	analysisComplete bool
	error            string
	samples          int
	lanes            int
	dragenVersion    string
}

func createMockAnalysisDir(t *testing.T, options ...analysisDirOption) string {
	d := analysisDir{
		copyComplete:     false,
		analysisComplete: false,
		error:            "",
		samples:          3,
		lanes:            8,
		dragenVersion:    "4.3.16",
	}

	for _, o := range options {
		o(&d)
	}

	path, err := mockAnalysisDirectory(t, d)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func withCopyComplete() analysisDirOption {
	return func(d *analysisDir) {
		d.copyComplete = true
	}
}

func withAnalysisComplete() analysisDirOption {
	return func(d *analysisDir) {
		d.analysisComplete = true
	}
}

func withError(e string) analysisDirOption {
	return func(d *analysisDir) {
		d.error = e
	}
}

func withSamples(s int) analysisDirOption {
	return func(d *analysisDir) {
		d.samples = s
	}
}

func withLanes(l int) analysisDirOption {
	return func(d *analysisDir) {
		d.lanes = l
	}
}

func withDragenVersion(v string) analysisDirOption {
	return func(d *analysisDir) {
		d.dragenVersion = v
	}
}

func mockFile(path string, content string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	_, err = io.WriteString(f, content)
	return err
}

func mockSummaryJson(samples int) string {
	summary := `{"result": "success", "workflows": [{"workflow_name": "bcl_convert", "samples": [`
	for i := range samples {
		summary += fmt.Sprintf(`{"sample_id": "sample%d"}`, i+1)
		if i < samples-1 {
			summary += ", "
		}
	}
	summary += `]}]}`
	return summary
}

func mockManifest(dir string) error {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	var content string
	for i, f := range files {
		content = fmt.Sprintf("%s%s\thash%d\n", content, f, i+1)
	}
	return mockFile(filepath.Join(dir, "Manifest.tsv"), content)
}

func mockAnalysisDirectory(t *testing.T, config analysisDir) (string, error) {
	runDir := t.TempDir()
	analysisDir := filepath.Join(runDir, "Analysis", "1")
	if err := os.MkdirAll(analysisDir, 0o755); err != nil {
		return analysisDir, err
	}
	if err := os.MkdirAll(filepath.Join(analysisDir, "Data", "summary", config.dragenVersion), 0o755); err != nil {
		return analysisDir, err
	}
	if config.copyComplete {
		if err := mockFile(filepath.Join(analysisDir, "CopyComplete.txt"), ""); err != nil {
			return analysisDir, err
		}
	}
	if config.analysisComplete {
		if err := mockFile(filepath.Join(analysisDir, "Data", "Secondary_Analysis_Complete.txt"), ""); err != nil {
			return analysisDir, err
		}
	}
	if config.error != "" {
		// Analysis is "ready" with errors
		s := DragenErrorSummary{
			Result: config.error,
		}
		b, err := json.Marshal(s)
		if err != nil {
			return analysisDir, err
		}
		if err := mockFile(filepath.Join(analysisDir, "Data", "Error_Summary.json"), string(b)); err != nil {
			return analysisDir, err
		}
	}
	if config.copyComplete && config.analysisComplete && config.error == "" {
		// Analysis is ready without errors
		if err := mockFile(filepath.Join(analysisDir, "Data", "summary", config.dragenVersion, "detailed_summary.json"), mockSummaryJson(config.samples)); err != nil {
			return analysisDir, err
		}
		demuxDir := filepath.Join(analysisDir, "Data", "Demux")
		if err := os.MkdirAll(demuxDir, 0o755); err != nil {
			return analysisDir, err
		}
		for _, name := range []string{"Index_Hopping_Counts.csv", "Demultiplex_Stats.csv", "Top_Unknown_Barcodes.csv"} {
			if err := mockFile(filepath.Join(demuxDir, name), ""); err != nil {
				return analysisDir, err
			}
		}
		fastqDir := filepath.Join(analysisDir, "Data", "BCLConvert", "fastq")
		if err := os.MkdirAll(fastqDir, 0o755); err != nil {
			return analysisDir, err
		}
		for s := range config.samples {
			for l := range config.lanes {
				err1 := mockFile(filepath.Join(fastqDir, fmt.Sprintf("sample%d_L%d_1.fastq.gz", s+1, l+1)), "")
				err2 := mockFile(filepath.Join(fastqDir, fmt.Sprintf("sample%d_L%d_2.fastq.gz", s+1, l+1)), "")
				err := errors.Join(err1, err2)
				if err != nil {
					return analysisDir, err
				}
			}
		}
	}
	return analysisDir, mockManifest(analysisDir)
}

func TestDragenAnalysis(t *testing.T) {
	testcases := []struct {
		name          string
		run           Run
		state         State
		expectedFiles int
		analysisDir   string
	}{
		{
			name: "analysis ready",
			run: Run{
				RunID: "run1",
				RunParameters: interop.RunParameters{
					Software: []interop.Software{
						{Name: "Dragen", Version: "4.3.16"},
					},
				},
			},
			analysisDir: createMockAnalysisDir(
				t,
				withCopyComplete(),
				withAnalysisComplete(),
				withDragenVersion("4.3.16"),
				withSamples(3),
				withLanes(8),
			),
			expectedFiles: 3*8*2 + 3, // 2 fastq per sample per lane + 3 stats files
			state:         StateReady,
		},
		{
			name: "analysis pending",
			run: Run{
				RunID: "run1",
				RunParameters: interop.RunParameters{
					Software: []interop.Software{
						{Name: "Dragen", Version: "4.3.16"},
					},
				},
			},
			analysisDir:   createMockAnalysisDir(t),
			expectedFiles: 0,
			state:         StatePending,
		},
		{
			name: "error in analysis",
			run: Run{
				RunID: "run1",
				RunParameters: interop.RunParameters{
					Software: []interop.Software{
						{Name: "Dragen", Version: "4.3.16"},
					},
				},
			},
			analysisDir: createMockAnalysisDir(
				t,
				withCopyComplete(),
				withAnalysisComplete(),
				withDragenVersion("4.3.16"),
				withError("error"),
			),
			expectedFiles: 0,
			state:         StateError,
		},
	}
	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
			slog.SetDefault(logger)
			analysis, err := NewDragenAnalysis(c.analysisDir, &c.run)
			if err != nil {
				t.Fatal(err)
			}
			state := analysis.StateHistory.LastState()
			if state != c.state {
				t.Errorf("expected state %s, got %s", c.state, state)
			}
			if len(analysis.OutputFiles) != c.expectedFiles {
				t.Errorf("expected %d files, got %d", c.expectedFiles, len(analysis.OutputFiles))
			}
		})
	}
}

func TestDragenManifest(t *testing.T) {
	testcases := []struct {
		name  string
		data  []byte
		files []string
		error bool
	}{
		{
			name: "valid manifest one file",
			data: []byte("Data/file1.txt\thash1\n"),
			files: []string{
				"Data/file1.txt",
			},
		},
		{
			name:  "invalid manifest",
			data:  []byte("Data/file1.txt\n"),
			error: true,
		},
		{
			name: "empty manifest",
			data: []byte(""),
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			r := bytes.NewReader(c.data)
			m, err := ReadDragenManifest(r)
			if c.error != (err != nil) {
				t.Fatal(err)
			}
			if len(m.Files) != len(c.files) {
				t.Fatalf("expected %d files, got %d files", len(c.files), len(m.Files))
			}
			for i := range m.Files {
				if m.Files[i] != c.files[i] {
					t.Errorf("expected file %d to be %s, got %s", i+1, c.files[i], m.Files[i])
				}
			}
		})
	}
}

func TestDragenManifestFindFiles(t *testing.T) {
	testcases := []struct {
		name     string
		manifest DragenManifest
		regex    *regexp.Regexp
		matches  []string
	}{
		{
			name: "contains matches",
			manifest: DragenManifest{
				Files: []string{
					"data/subdir1/file1.txt",
					"data/subdir1/file1.fastq.gz",
					"data/subdir2/file2.txt",
					"data/subdir2/file2.fastq.gz",
					"data/subdir2-1/file2.txt",
					"data/subdir2-1/file2.fastq.gz",
				},
			},
			regex: regexp.MustCompile(`^file2.fastq.gz$`),
			matches: []string{
				"data/subdir2/file2.fastq.gz",
				"data/subdir2-1/file2.fastq.gz",
			},
		},
		{
			name: "nil regex",
			manifest: DragenManifest{
				Files: []string{
					"data/subdir1/file1.txt",
					"data/subdir1/file1.fastq.gz",
					"data/subdir2/file2.txt",
					"data/subdir2/file2.fastq.gz",
					"data/subdir2-1/file2.txt",
					"data/subdir2-1/file2.fastq.gz",
				},
			},
		},
		{
			name: "no matcher",
			manifest: DragenManifest{
				Files: []string{
					"data/subdir1/file1.txt",
					"data/subdir1/file1.fastq.gz",
					"data/subdir2/file2.txt",
					"data/subdir2/file2.fastq.gz",
					"data/subdir2-1/file2.txt",
					"data/subdir2-1/file2.fastq.gz",
				},
			},
			regex: regexp.MustCompile(`^file3.fastq.gz$`),
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			matches := c.manifest.FindFiles(c.regex)
			if len(matches) != len(c.matches) {
				t.Fatalf("expected %d matches, got %d", len(c.matches), len(matches))
			}
			for i := range matches {
				if matches[i] != c.matches[i] {
					t.Errorf("expected match %d to be %q, got %q", i+1, c.matches[i], matches[i])
				}
			}
		})
	}
}

func TestDragenManifestFindFile(t *testing.T) {
	testcases := []struct {
		name      string
		manifest  DragenManifest
		matchWith string
		match     string
		error     bool
	}{
		{
			name: "multiple matches",
			manifest: DragenManifest{
				Files: []string{
					"data/subdir1/file1.txt",
					"data/subdir1/file1.fastq.gz",
					"data/subdir2/file2.txt",
					"data/subdir2/file2.fastq.gz",
					"data/subdir2-1/file2.txt",
					"data/subdir2-1/file2.fastq.gz",
				},
			},
			matchWith: "file2.txt",
			error:     true,
		},
		{
			name: "no matches",
			manifest: DragenManifest{
				Files: []string{
					"data/subdir1/file1.txt",
					"data/subdir1/file1.fastq.gz",
					"data/subdir2/file2.txt",
					"data/subdir2/file2.fastq.gz",
					"data/subdir2-1/file2.txt",
					"data/subdir2-1/file2.fastq.gz",
				},
			},
			matchWith: "file3.txt",
			error:     true,
		},
		{
			name: "single matches",
			manifest: DragenManifest{
				Files: []string{
					"data/subdir1/file1.txt",
					"data/subdir1/file1.fastq.gz",
					"data/subdir2/file2.txt",
					"data/subdir2/file2.fastq.gz",
					"data/subdir2-1/file2.txt",
					"data/subdir2-1/file2.fastq.gz",
				},
			},
			matchWith: "file1.fastq.gz",
			match:     "data/subdir1/file1.fastq.gz",
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			match, err := c.manifest.FindFile(c.matchWith)
			if c.error != (err != nil) {
				t.Fatalf("expected error to be %t, got %q", c.error, err)
			}
			if match != c.match {
				t.Fatalf("expected %s, got %s", c.match, match)
			}
		})
	}
}

func TestRealDragenManifest(t *testing.T) {
	manifestFile := "testdata/dragen_manifest.tsv"
	f, err := os.Open(manifestFile)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := ReadDragenManifest(f)
	if err != nil {
		t.Fatal(err)
	}

	testcases := []struct {
		name   string
		regex  *regexp.Regexp
		expect []string
	}{
		{
			name:  "D25-445",
			regex: regexp.MustCompile(`^` + regexp.QuoteMeta("D25-445") + `.*\.fastq\.gz$`),
			expect: []string{
				"Data/DragenSomatic/fastq/D25-445_S16_L001_R1_001.fastq.gz",
				"Data/DragenSomatic/fastq/D25-445_S16_L001_R2_001.fastq.gz",
				"Data/DragenSomatic/fastq/D25-445_S16_L002_R1_001.fastq.gz",
				"Data/DragenSomatic/fastq/D25-445_S16_L002_R2_001.fastq.gz",
				"Data/DragenSomatic/fastq/D25-445_S16_L003_R1_001.fastq.gz",
				"Data/DragenSomatic/fastq/D25-445_S16_L003_R2_001.fastq.gz",
			},
		},
		{
			name:  "Seq25-9259",
			regex: regexp.MustCompile(`^` + regexp.QuoteMeta("Seq25-9259") + `.*\.fastq\.gz$`),
			expect: []string{
				"Data/DragenGermline/fastq/Seq25-9259_S11_L007_R1_001.fastq.gz",
				"Data/DragenGermline/fastq/Seq25-9259_S11_L007_R2_001.fastq.gz",
				"Data/DragenGermline/fastq/Seq25-9259_S11_L008_R1_001.fastq.gz",
				"Data/DragenGermline/fastq/Seq25-9259_S11_L008_R2_001.fastq.gz",
			},
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			matches := manifest.FindFiles(c.regex)
			if len(matches) != len(c.expect) {
				t.Errorf("expected %d fastq files, found %d", len(c.expect), len(matches))
			}
			for i := range c.expect {
				if c.expect[i] != matches[i] {
					t.Errorf("expected match %d to be %q, got %q", i, c.expect[i], matches[i])
				}
			}
		})
	}
}

func TestDragenAnalysisState(t *testing.T) {
	testcases := []struct {
		name             string
		copycomplete     bool
		analysiscomplete bool
		state            State
	}{
		{
			name:             "pending",
			copycomplete:     false,
			analysiscomplete: false,
			state:            StatePending,
		},
		{
			name:             "pending",
			copycomplete:     true,
			analysiscomplete: false,
			state:            StatePending,
		},
		{
			name:             "pending",
			copycomplete:     false,
			analysiscomplete: true,
			state:            StatePending,
		},
		{
			name:             "ready",
			copycomplete:     true,
			analysiscomplete: true,
			state:            StateReady,
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			if c.copycomplete {
				f, err := os.Create(filepath.Join(dir, "CopyComplete.txt"))
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = f.Close() }()
			}
			if c.analysiscomplete {
				if err := os.Mkdir(filepath.Join(dir, "Data"), 0o755); err != nil {
					t.Fatal(err)
				}
				f, err := os.Create(filepath.Join(dir, "Data", "Secondary_Analysis_Complete.txt"))
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = f.Close() }()
			}
			state := dragenAnalysisState(dir)
			if state != c.state {
				t.Errorf("expected state %s, got %s", c.state, state)
			}
		})
	}
}

func TestGetFiles(t *testing.T) {
	testcases := []struct {
		name     string
		analysis Analysis
		filetype AnalysisFileType
		level    AnalysisLevel
		parentId string
		files    []string
	}{
		{
			name: "no fastq files",
			analysis: Analysis{
				Path: "/path/to/analysis/1",
				OutputFiles: []AnalysisFile{
					{
						partOfAnalysis: true,
						Path:           "data/sample1.vcf.gz",
						FileType:       FileSnvVcf,
						Level:          LevelSample,
						ParentId:       "sample1",
					},
					{
						partOfAnalysis: true,
						Path:           "data/sample2.vcf.gz",
						FileType:       FileSnvVcf,
						Level:          LevelSample,
						ParentId:       "sample1",
					},
				},
			},
			filetype: FileFastq,
			files:    []string{},
		},
		{
			name: "2 fastq files",
			analysis: Analysis{
				Path: "/path/to/analysis/1",
				OutputFiles: []AnalysisFile{
					{
						partOfAnalysis: true,
						Path:           "data/sample1.vcf.gz",
						FileType:       FileSnvVcf,
						Level:          LevelSample,
						ParentId:       "sample1",
					},
					{
						partOfAnalysis: true,
						Path:           "data/sample1.fastq.gz",
						FileType:       FileFastq,
						Level:          LevelSample,
						ParentId:       "sample1",
					},
					{
						partOfAnalysis: true,
						Path:           "data/sample2.fastq.gz",
						FileType:       FileFastq,
						Level:          LevelSample,
						ParentId:       "sample2",
					},
				},
			},
			filetype: FileFastq,
			level:    LevelSample,
			files: []string{
				"/path/to/analysis/1/data/sample1.fastq.gz",
				"/path/to/analysis/1/data/sample2.fastq.gz",
			},
		},
		{
			name: "1 fastq files for specific sample",
			analysis: Analysis{
				Path: "/path/to/analysis/1",
				OutputFiles: []AnalysisFile{
					{
						partOfAnalysis: true,
						Path:           "data/sample1.vcf.gz",
						FileType:       FileSnvVcf,
						Level:          LevelSample,
						ParentId:       "sample1",
					},
					{
						partOfAnalysis: true,
						Path:           "data/sample1.fastq.gz",
						FileType:       FileFastq,
						Level:          LevelSample,
						ParentId:       "sample1",
					},
					{
						partOfAnalysis: true,
						Path:           "data/sample2.fastq.gz",
						FileType:       FileFastq,
						Level:          LevelSample,
						ParentId:       "sample2",
					},
				},
			},
			filetype: FileFastq,
			level:    LevelSample,
			parentId: "sample1",
			files: []string{
				"/path/to/analysis/1/data/sample1.fastq.gz",
			},
		},
		{
			name: "3 run level text files",
			analysis: Analysis{
				Path: "/path/to/analysis/1",
				OutputFiles: []AnalysisFile{
					{
						partOfAnalysis: true,
						Path:           "data/stats1.tsv",
						FileType:       FileText,
						Level:          LevelRun,
						ParentId:       "run1",
					},
					{
						partOfAnalysis: true,
						Path:           "data/sample1.fastq.gz",
						FileType:       FileFastq,
						Level:          LevelSample,
						ParentId:       "sample1",
					},
					{
						partOfAnalysis: true,
						Path:           "data/stats2.csv",
						FileType:       FileText,
						Level:          LevelRun,
						ParentId:       "run1",
					},
					{
						partOfAnalysis: true,
						Path:           "data/sample2.fastq.gz",
						FileType:       FileFastq,
						Level:          LevelSample,
						ParentId:       "sample2",
					},
					{
						partOfAnalysis: true,
						Path:           "data/stats3.txt",
						FileType:       FileText,
						Level:          LevelRun,
						ParentId:       "run1",
					},
				},
			},
			filetype: FileText,
			level:    LevelRun,
			files: []string{
				"/path/to/analysis/1/data/stats1.tsv",
				"/path/to/analysis/1/data/stats2.csv",
				"/path/to/analysis/1/data/stats3.txt",
			},
		},
		{
			name: "2 sample level html files",
			analysis: Analysis{
				Path: "/path/to/analysis/1",
				OutputFiles: []AnalysisFile{
					{
						partOfAnalysis: true,
						Path:           "data/reports/DragenGermline/report_files/samples/sample1/sample1.html",
						FileType:       FileHtml,
						Level:          LevelSample,
						ParentId:       "sample1",
					},
					{
						partOfAnalysis: true,
						Path:           "data/reports/DragenGermline/report_files/samples/sample2/sample2.html",
						FileType:       FileHtml,
						Level:          LevelSample,
						ParentId:       "sample2",
					},
					{
						partOfAnalysis: true,
						Path:           "data/stats1.csv",
						FileType:       FileText,
						Level:          LevelRun,
						ParentId:       "run1",
					},
					{
						partOfAnalysis: true,
						Path:           "data/stats2.txt",
						FileType:       FileText,
						Level:          LevelRun,
						ParentId:       "run1",
					},
				},
			},
			filetype: FileHtml,
			level:    LevelSample,
			files: []string{
				"/path/to/analysis/1/data/reports/DragenGermline/report_files/samples/sample1/sample1.html",
				"/path/to/analysis/1/data/reports/DragenGermline/report_files/samples/sample2/sample2.html",
			},
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			files := c.analysis.GetFiles(AnalysisFileFilter{
				FileType: c.filetype,
				Level:    c.level,
				ParentId: c.parentId,
			})
			if len(files) != len(c.files) {
				t.Fatalf("expected %d files, got %d", len(c.files), len(files))
			}
			for i, f := range files {
				if err := f.Validate(); err != nil {
					t.Error(err)
				}
				if c.files[i] != f.Path {
					t.Errorf("expected file %v, got %v", c.files[i], f)
				}
			}
		})
	}
}

func TestUnmarshalJSONLevel(t *testing.T) {
	testcases := []struct {
		name    string
		json    []byte
		expect  AnalysisLevel
		isError bool
		isValid bool
	}{
		{
			name:    "level run",
			json:    []byte(`"run"`),
			expect:  LevelRun,
			isValid: true,
		},
		{
			name:    "level case",
			json:    []byte(`"case"`),
			expect:  LevelCase,
			isValid: true,
		},
		{
			name:    "level sample",
			json:    []byte(`"sample"`),
			expect:  LevelSample,
			isValid: true,
		},
		{
			name:    "empty string",
			json:    []byte(`""`),
			expect:  0,
			isValid: false,
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			var l AnalysisLevel
			err := json.Unmarshal(c.json, &l)
			if c.isError != (err != nil) {
				t.Fatalf("isError is %t, but got %s", c.isError, err)
			}
			if err != nil && l != c.expect {
				t.Errorf("expected level %s, got %s", c.expect, l)
			}
			if c.isValid != l.IsValid() {
				t.Errorf("expected IsValid to be %t, got %t", c.isValid, l.IsValid())
			}
		})
	}
}

func TestBSONLevel(t *testing.T) {
	testcases := []struct {
		name    string
		level   AnalysisLevel
		isError bool
		isValid bool
	}{
		{
			name:    "level run",
			level:   LevelRun,
			isValid: true,
		},
		{
			name:    "level case",
			level:   LevelCase,
			isValid: true,
		},
		{
			name:    "level sample",
			level:   LevelSample,
			isValid: true,
		},
		{
			name:    "empty string",
			level:   0,
			isValid: false,
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			doc, err := bson.Marshal(bson.M{"level": c.level})
			if err != nil {
				t.Fatal(err)
			}
			var tmp struct {
				Level AnalysisLevel `bson:"level"`
			}
			err = bson.Unmarshal(doc, &tmp)
			if c.isError != (err != nil) {
				t.Fatalf("isError is %t, but got %s", c.isError, err)
			}
			if c.isValid != tmp.Level.IsValid() {
				t.Errorf("expected IsValid to be %t, got %t", c.isValid, tmp.Level.IsValid())
			}
			if err == nil && tmp.Level != c.level {
				t.Errorf("expected level %s, got %s", c.level, tmp.Level)
			}
		})
	}
}

func TestAnalysisFileType(t *testing.T) {
	testcases := []struct {
		name       string
		typeString string
		isValid    bool
		isZero     bool
	}{
		{
			name:       "fastq",
			typeString: "fastq",
			isValid:    true,
			isZero:     false,
		},
		{
			name:       "interop",
			typeString: "interop",
			isValid:    true,
			isZero:     false,
		},
		{
			name:       "empty",
			typeString: "",
			isValid:    false,
			isZero:     true,
		},
		{
			name:       "exe",
			typeString: "exe",
			isValid:    false,
			isZero:     false,
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			ft := AnalysisFileTypeFromString(c.typeString)
			if ft.IsValid() != c.isValid {
				t.Errorf("expected IsValid = %t, got %t", c.isValid, ft.IsValid())
			}
			if ft.IsZero() != c.isZero {
				t.Errorf("expected IsZero = %t, got %t", c.isZero, ft.IsZero())
			}
		})
	}
}

func TestAnalysisOutputFilesGlobbing(t *testing.T) {
	tmpdir := t.TempDir()
	testcases := []struct {
		name          string
		expectedPaths []string
		extraPaths    []string
		files         AnalysisFiles
		shouldError   bool
	}{
		{
			name: "two files in the same directory",
			expectedPaths: []string{
				filepath.Join(tmpdir, "path/to/file1.png"),
				filepath.Join(tmpdir, "path/to/file2.png"),
			},
			extraPaths: []string{
				filepath.Join(tmpdir, "path/file1.png"),
				filepath.Join(tmpdir, "path/file2.png"),
			},
			files: []AnalysisFile{
				{
					Path:     filepath.Join(tmpdir, "path/to/file*"),
					Level:    LevelRun,
					FileType: FilePng,
				},
			},
		},
		{
			name: "directories shouldn't be matched",
			expectedPaths: []string{
				filepath.Join(tmpdir, "path/file1.png"),
				filepath.Join(tmpdir, "path/file2.png"),
			},
			extraPaths: []string{
				filepath.Join(tmpdir, "path/to/file1.png"),
				filepath.Join(tmpdir, "path/to/file2.png"),
			},
			files: []AnalysisFile{
				{
					Path:     filepath.Join(tmpdir, "path/*"),
					Level:    LevelRun,
					FileType: FilePng,
				},
			},
		},
		{
			name: "no matches",
			extraPaths: []string{
				filepath.Join(tmpdir, "path/to/file1.png"),
				filepath.Join(tmpdir, "path/to/file2.png"),
			},
			files: []AnalysisFile{
				{
					Path:     filepath.Join(tmpdir, "path/*"),
					Level:    LevelRun,
					FileType: FilePng,
				},
			},
			shouldError: true,
		},
		{
			name: "multidir match",
			expectedPaths: []string{
				filepath.Join(tmpdir, "path/to/file1.png"),
				filepath.Join(tmpdir, "path/to/file2.png"),
				filepath.Join(tmpdir, "path/of/file1.png"),
				filepath.Join(tmpdir, "path/of/file2.png"),
			},
			files: []AnalysisFile{
				{
					Path:     filepath.Join(tmpdir, "path/*/file*.png"),
					Level:    LevelRun,
					FileType: FilePng,
				},
			},
		},
		{
			name: "match only on dir",
			expectedPaths: []string{
				filepath.Join(tmpdir, "path/to/file1.png"),
			},
			files: []AnalysisFile{
				{
					Path:     filepath.Join(tmpdir, "path/*/file1.png"),
					Level:    LevelRun,
					FileType: FilePng,
				},
			},
		},
		{
			name: "no wildcards with match",
			expectedPaths: []string{
				filepath.Join(tmpdir, "path/to/file1.png"),
			},
			files: []AnalysisFile{
				{
					Path:     filepath.Join(tmpdir, "path/to/file1.png"),
					Level:    LevelRun,
					FileType: FilePng,
				},
			},
		},
		{
			name: "no wildcards without match",
			files: []AnalysisFile{
				{
					Path:     filepath.Join(tmpdir, "path/to/file1.png"),
					Level:    LevelRun,
					FileType: FilePng,
				},
			},
			shouldError: true,
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			for _, path := range append(c.expectedPaths, c.extraPaths...) {
				if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
					t.Fatal(err)
				}
				f, err := os.Create(path)
				if err != nil {
					t.Fatal(err)
				}
				_ = f.Close()
				defer func() {
					_ = os.Remove(path)
				}()
			}
			err := c.files.ResolvePaths()
			if (err == nil) && c.shouldError {
				t.Fatal("expected error to be non-nil, got nil")
			}
			if err != nil {
				if !c.shouldError {
					t.Fatalf("expected error to be nil, got %q", err)
				}
				// We got an error as expected, nothing more to check
				return
			}
			if len(c.files) != len(c.expectedPaths) {
				t.Errorf("expected %d files, got %d files", len(c.expectedPaths), len(c.files))
			}
		})
	}
}

func TestAnalysisFileValidation(t *testing.T) {
	testcases := []struct {
		name  string
		file  AnalysisFile
		valid bool
	}{
		{
			name: "ascending above the analysis directory",
			file: AnalysisFile{
				partOfAnalysis: true,
				Path:           "../test.txt",
				FileType:       FileText,
				Level:          LevelRun,
			},
			valid: false,
		},
		{
			name: "ascending above the analysis directory",
			file: AnalysisFile{
				partOfAnalysis: true,
				Path:           "./../test.txt",
				FileType:       FileText,
				Level:          LevelRun,
			},
			valid: false,
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			err := c.file.Validate()
			if err != nil {
				t.Log(err)
				if c.valid {
					t.Errorf("expected file to be valid, got error: %q", err)
				}
			} else {
				if !c.valid {
					t.Error("expected file to be invalid, got no error")
				}
			}
		})
	}
}
