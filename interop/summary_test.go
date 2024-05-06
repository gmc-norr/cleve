package interop

import (
	"testing"
)

func TestGenerateInteropSummary(t *testing.T) {
	rundirectory := "../test_data/novaseq_full"
	summary, err := GenerateSummary(rundirectory)
	if err != nil {
		t.Fatalf("%s when generating summary", err.Error())
	}

	t.Log(summary)

	if summary.Version == "" {
		t.Fail()
	}

	if summary.RunDirectory == "" {
		t.Fail()
	}

	if summary.RunSummary.Header == nil || len(summary.RunSummary.Header) == 0 {
		t.Fail()
	}

	if summary.ReadSummaries == nil || len(summary.ReadSummaries) != 4 {
		t.Fail()
	}
}
