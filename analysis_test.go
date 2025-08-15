package cleve

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

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
			files, err := readDragenManifest(r)
			if c.error != (err != nil) {
				t.Fatal(err)
			}
			if len(files) != len(c.files) {
				t.Fatalf("expected %d files, got %d files", len(c.files), len(files))
			}
			for i := range files {
				if files[i] != c.files[i] {
					t.Errorf("expected file %d to be %s, got %s", i+1, c.files[i], files[i])
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
