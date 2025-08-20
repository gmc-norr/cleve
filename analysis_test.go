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

func mockManifest(samples int, lanes int) string {
	var manifest string
	for s := range samples {
		for l := range lanes {
			manifest += fmt.Sprintf("Data/BCLConvert/fastq/sample%d_L%d.fastq.gz\thash%d\n", s+1, l+1, s*l+l+1)
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
		if err := mockFile(filepath.Join(analysisDir, "Manifest.tsv"), mockManifest(samples, lanes)); err != nil {
			return analysisDir, err
		}
		if err := mockFile(filepath.Join(analysisDir, "Data", "Secondary_Analysis_Complete.txt"), ""); err != nil {
			return analysisDir, err
		}
		if err := mockFile(filepath.Join(analysisDir, "Data", "summary", dragenVersion, "detailed_summary.json"), mockSummaryJson(samples)); err != nil {
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
		if err := mockFile(filepath.Join(analysisDir, "Data", "summary", dragenVersion, "detailed_summary.json"), `{"result": "error", "workflows": [{"workflow_name": "bcl_convert", "samples": [{"sample_id": "sample1"}]}]}`); err != nil {
			return analysisDir, err
		}
	}
	return analysisDir, nil
}

func TestDragenAnalysis(t *testing.T) {
	testcases := []struct {
		name    string
		run     Run
		state   State
		samples int
		lanes   int
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
			samples: 3,
			lanes:   8,
			state:   StateReady,
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
			state: StatePending,
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
			state: StateError,
		},
	}
	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			dir, err := mockAnalysisDirectory(t, c.state, c.run.RunParameters.Software[0].Version, c.samples, c.lanes)
			if err != nil {
				t.Fatal(err)
			}
			analyses, err := NewDragenAnalysis(dir, &c.run)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("analysis count: %d", len(analyses))
			if len(analyses) != c.samples+1 {
				t.Fatalf("expected %d analyses, got %d", c.samples+1, len(analyses))
			}
			state := analyses[0].StateHistory.LastState()
			if state != c.state {
				t.Errorf("expected state %s, got %s", c.state, state)
			}
			if analyses[0].Level != LevelRun {
				t.Error("first analysis should be on the run level")
			}
			for i, a := range analyses[1:] {
				if a.Level != LevelSample {
					t.Errorf("analysis %d is not on sample level (sample %s)", i+1, a.ParentId)
				}
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

func TestDragenManifestFind(t *testing.T) {
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
		files    []string
	}{
		{
			name: "no fastq files",
			analysis: Analysis{
				Path: "/path/to/analysis/1",
				Files: []AnalysisFile{
					{
						Path:     "data/sample1.vcf.gz",
						FileType: FileSnvVcf,
					},
					{
						Path:     "data/sample2.vcf.gz",
						FileType: FileSnvVcf,
					},
				},
			},
			filetype: FileFastq,
			files:    []string{},
		},
		{
			name: "1 fastq file",
			analysis: Analysis{
				Path: "/path/to/analysis/1",
				Files: []AnalysisFile{
					{
						Path:     "data/sample1.vcf.gz",
						FileType: FileSnvVcf,
					},
					{
						Path:     "data/sample1.fastq.gz",
						FileType: FileFastq,
					},
				},
			},
			filetype: FileFastq,
			files:    []string{"/path/to/analysis/1/data/sample1.fastq.gz"},
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			files := c.analysis.GetFiles(c.filetype)
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
	}{
		{
			name:   "level run",
			json:   []byte(`"run"`),
			expect: LevelRun,
		},
		{
			name:   "level case",
			json:   []byte(`"case"`),
			expect: LevelCase,
		},
		{
			name:   "level sample",
			json:   []byte(`"sample"`),
			expect: LevelSample,
		},
		{
			name:    "empty string",
			json:    []byte(`""`),
			isError: true,
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
		})
	}
}

func TestBSONLevel(t *testing.T) {
	testcases := []struct {
		name    string
		level   AnalysisLevel
		isError bool
	}{
		{
			name:  "level run",
			level: LevelRun,
		},
		{
			name:  "level case",
			level: LevelCase,
		},
		{
			name:  "level sample",
			level: LevelSample,
		},
		{
			name:    "empty string",
			isError: true,
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
			if err == nil && !tmp.Level.IsValid() {
				t.Errorf("level is invalid: %s", c.level)
			}
			if err == nil && tmp.Level != c.level {
				t.Errorf("expected level %s, got %s", c.level, tmp.Level)
			}
		})
	}
}
