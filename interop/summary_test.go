package interop

import (
	"testing"
)

func TestGenerateInteropSummary(t *testing.T) {
	rundirectory := "../test_data/novaseq_full"
	runId := "20240403_LH00352_0006_A222H73LT1"
	summary, err := GenerateSummary(runId, rundirectory)
	if err != nil {
		t.Fatalf("%s when generating summary", err.Error())
	}

	t.Logf("%#v\n", summary)

	if summary.Version == "" {
		t.Fail()
	}

	if summary.RunID != runId {
		t.Fail()
	}

	if summary.RunDirectory == "" {
		t.Fail()
	}

	if summary.ReadSummaries == nil || len(summary.ReadSummaries) != 4 {
		t.Fail()
	}
}
