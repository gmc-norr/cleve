package cleve

import (
	"errors"
	"os"
	"testing"
)

func TestGenerateInteropSummary(t *testing.T) {
	rundirectory := "/home/nima18/git/cleve/test_data/novaseq_full"
	if _, err := os.Stat(rundirectory); errors.Is(err, os.ErrNotExist) {
		t.Skip("test data not found, skipping")
	}
	summary, err := GenerateSummary(rundirectory)
	if err != nil {
		t.Fatalf("%s when generating summary", err.Error())
	}

	if summary.Version == "" {
		t.Fail()
	}

	if summary.RunDirectory == "" {
		t.Fail()
	}

	if summary.ReadSummaries == nil || len(summary.ReadSummaries) != 4 {
		t.Fail()
	}
}
