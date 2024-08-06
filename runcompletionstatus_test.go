package cleve

import (
	"errors"
	"os"
	"testing"
)

func TestReadRunCompletionStatus(t *testing.T) {
	cases := []struct {
		name     string
		filename string
		status   string
		message  string
	}{
		{
			"novaseq",
			"test_data/novaseq_full/RunCompletionStatus.xml",
			"success",
			"RunCompleted",
		},
		{
			"nextseq1",
			"test_data/nextseq1_full/RunCompletionStatus.xml",
			"success",
			"CompletedAsPlanned",
		},
		{
			"nextseq2",
			"test_data/nextseq2_full/RunCompletionStatus.xml",
			"success",
			"CompletedAsPlanned",
		},
		{
			"nextseq_failed",
			"test_data/230713_NB551119_0374_AHMVKTBGX/RunCompletionStatus.xml",
			"error",
			"Thread was being aborted. (System.Threading.ThreadAbortException)",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := os.Stat(c.filename); errors.Is(err, os.ErrNotExist) {
				t.Skip("test data not found, skipping")
			}
			rct, err := ReadRunCompletionStatus(c.filename)
			if err != nil {
				t.Fatal(err.Error())
			}
			if c.status != rct.Status {
				t.Errorf(`expected status "%s", got "%s"`, c.status, rct.Status)
			}
			if c.message != rct.Message {
				t.Errorf(`expected message "%s", got "%s"`, c.message, rct.Message)
			}
		})
	}
}
