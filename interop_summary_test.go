package cleve

import (
	"testing"
)

func TestGenerateInteropSummary(t *testing.T) {
	rundirectory := "test_data/novaseq_full"
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
