package cleve

import (
	"errors"
	"io"
	"os"
	"testing"
)

func TestParseRunInfo(t *testing.T) {
	cases := map[string]struct {
		runinfo string
	}{
		"novaseq": {
			runinfo: "test_data/novaseq_full/RunInfo.xml",
		},
		"nextseq1": {
			runinfo: "test_data/nextseq1_full/RunInfo.xml",
		},
		"nextseq2": {
			runinfo: "test_data/nextseq2_full/RunInfo.xml",
		},
	}

	for k, v := range cases {
		if _, err := os.Stat(v.runinfo); errors.Is(err, os.ErrNotExist) {
			t.Skip("test data not found, skipping")
		}
		f, err := os.Open(v.runinfo)
		if err != nil {
			t.Fatalf(`case "%s": %s`, k, err.Error())
		}
		defer f.Close()
		b, err := io.ReadAll(f)
		if err != nil {
			t.Fatalf(`case "%s": %s`, k, err.Error())
		}
		_, err = ParseRunInfo(b)
		if err != nil {
			t.Fatalf(`case "%s": %s`, k, err.Error())
		}
	}
}
