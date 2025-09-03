package cleve

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/gmc-norr/cleve/interop"
	"go.mongodb.org/mongo-driver/bson"
)

func mockFile(path string, content string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	_, err = io.WriteString(f, content)
	return err
}

func mockSummaryJson(samples int, state State) string {
	var stringState string
	switch state {
	case StateReady:
		stringState = "success"
	case StateError:
		stringState = "error"
	}
	summary := fmt.Sprintf(`{"result": "%s", "workflows": [{"workflow_name": "bcl_convert", "samples": [`, stringState)
	for i := range samples {
		summary += fmt.Sprintf(`{"sample_id": "sample%d"}`, i+1)
		if i < samples-1 {
			summary += ", "
		}
	}
	summary += `]}]}`
	return summary
}

func mockManifest(samples int, lanes int) string {
	var manifest string
	var i int
	manifest += fmt.Sprintf("Data/Demux/Demultiplex_Stats.csv\thash%d\n", i+1)
	manifest += fmt.Sprintf("Data/Demux/Index_Hopping_Counts.csv\thash%d\n", i+2)
	manifest += fmt.Sprintf("Data/Demux/Top_Unknown_Barcodes.csv\thash%d\n", i+3)
	i += 3
	for s := range samples {
		for l := range lanes {
			manifest += fmt.Sprintf("Data/BCLConvert/fastq/sample%d_L%d_1.fastq.gz\thash%d\n", s+1, l+1, i+1)
			manifest += fmt.Sprintf("Data/BCLConvert/fastq/sample%d_L%d_2.fastq.gz\thash%d\n", s+1, l+1, i+2)
			i += 2
		}
	}
	return manifest
}

func mockAnalysisDirectory(t *testing.T, state State, dragenVersion string, samples int, lanes int) (string, error) {
	runDir := t.TempDir()
	analysisDir := filepath.Join(runDir, "Analysis", "1")
	if err := os.MkdirAll(analysisDir, 0o755); err != nil {
		return analysisDir, err
	}
	switch state {
	case StateReady:
		// create CopyComplete.txt, Data/Secondary_Analysis_Complete.txt, Manifest.tsv, Data/summary/{version}/detailed_summary.json
		if err := os.MkdirAll(filepath.Join(analysisDir, "Data", "summary", dragenVersion), 0o755); err != nil {
			return analysisDir, err
		}
		if err := mockFile(filepath.Join(analysisDir, "CopyComplete.txt"), ""); err != nil {
			return analysisDir, err
		}
		t.Logf("%+v", mockManifest(samples, lanes))
		if err := mockFile(filepath.Join(analysisDir, "Manifest.tsv"), mockManifest(samples, lanes)); err != nil {
			return analysisDir, err
		}
		if err := mockFile(filepath.Join(analysisDir, "Data", "Secondary_Analysis_Complete.txt"), ""); err != nil {
			return analysisDir, err
		}
		if err := mockFile(filepath.Join(analysisDir, "Data", "summary", dragenVersion, "detailed_summary.json"), mockSummaryJson(samples, state)); err != nil {
			return analysisDir, err
		}
	case StatePending:
		// all good
	case StateError:
		// create CopyComplete.txt, Data/Secondary_Analysis_Complete.txt
		if err := os.MkdirAll(filepath.Join(analysisDir, "Data", "summary", dragenVersion), 0o755); err != nil {
			return analysisDir, err
		}
		if err := mockFile(filepath.Join(analysisDir, "CopyComplete.txt"), ""); err != nil {
			return analysisDir, err
		}
		if err := mockFile(filepath.Join(analysisDir, "Data", "Secondary_Analysis_Complete.txt"), ""); err != nil {
			return analysisDir, err
		}
		if err := mockFile(filepath.Join(analysisDir, "Data", "summary", dragenVersion, "detailed_summary.json"), mockSummaryJson(samples, state)); err != nil {
			return analysisDir, err
		}
	}
	return analysisDir, nil
}

func TestDragenAnalysis(t *testing.T) {
	testcases := []struct {
		name          string
		run           Run
		state         State
		samples       int
		lanes         int
		expectedFiles int
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
			samples:       3,
			lanes:         8,
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
			expectedFiles: 0,
			state:         StateError,
		},
	}
	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			dir, err := mockAnalysisDirectory(t, c.state, c.run.RunParameters.Software[0].Version, c.samples, c.lanes)
			if err != nil {
				t.Fatal(err)
			}
			analysis, err := NewDragenAnalysis(dir, &c.run)
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
						Path:     "data/sample1.vcf.gz",
						FileType: FileSnvVcf,
						Level:    LevelSample,
						ParentId: "sample1",
					},
					{
						Path:     "data/sample2.vcf.gz",
						FileType: FileSnvVcf,
						Level:    LevelSample,
						ParentId: "sample1",
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
						Path:     "data/sample1.vcf.gz",
						FileType: FileSnvVcf,
						Level:    LevelSample,
						ParentId: "sample1",
					},
					{
						Path:     "data/sample1.fastq.gz",
						FileType: FileFastq,
						Level:    LevelSample,
						ParentId: "sample1",
					},
					{
						Path:     "data/sample2.fastq.gz",
						FileType: FileFastq,
						Level:    LevelSample,
						ParentId: "sample2",
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
						Path:     "data/sample1.vcf.gz",
						FileType: FileSnvVcf,
						Level:    LevelSample,
						ParentId: "sample1",
					},
					{
						Path:     "data/sample1.fastq.gz",
						FileType: FileFastq,
						Level:    LevelSample,
						ParentId: "sample1",
					},
					{
						Path:     "data/sample2.fastq.gz",
						FileType: FileFastq,
						Level:    LevelSample,
						ParentId: "sample2",
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
						Path:     "data/stats1.tsv",
						FileType: FileText,
						Level:    LevelRun,
						ParentId: "run1",
					},
					{
						Path:     "data/sample1.fastq.gz",
						FileType: FileFastq,
						Level:    LevelSample,
						ParentId: "sample1",
					},
					{
						Path:     "data/stats2.csv",
						FileType: FileText,
						Level:    LevelRun,
						ParentId: "run1",
					},
					{
						Path:     "data/sample2.fastq.gz",
						FileType: FileFastq,
						Level:    LevelSample,
						ParentId: "sample2",
					},
					{
						Path:     "data/stats3.txt",
						FileType: FileText,
						Level:    LevelRun,
						ParentId: "run1",
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
				if c.files[i] != f {
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
