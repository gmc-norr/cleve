package mongo

import (
	"testing"
)

func TestSamplesSheetOptions(t *testing.T) {
	cases := []struct {
		name          string
		options       []SampleSheetOption
		error         bool
		expectedRunID string
		expectedUUID  string
	}{
		{
			name:          "samplesheet with run id",
			options:       []SampleSheetOption{SampleSheetWithRunId("run1")},
			error:         false,
			expectedRunID: "run1",
		},
		{
			name:    "empty run id",
			options: []SampleSheetOption{SampleSheetWithRunId("")},
			error:   true,
		},
		{
			name:         "valid uuid",
			options:      []SampleSheetOption{SampleSheetWithUuid("92f356e3-4d4b-4c8a-bbe2-a886bfd0f63c")},
			error:        false,
			expectedUUID: "92f356e3-4d4b-4c8a-bbe2-a886bfd0f63c",
		},
		{
			name:    "invalid uuid",
			options: []SampleSheetOption{SampleSheetWithUuid("92f356e3")},
			error:   true,
		},
		{
			name:    "empty uuid",
			options: []SampleSheetOption{SampleSheetWithUuid("")},
			error:   true,
		},
		{
			name: "uuid and run id",
			options: []SampleSheetOption{
				SampleSheetWithUuid("92f356e3-4d4b-4c8a-bbe2-a886bfd0f63c"),
				SampleSheetWithRunId("run1"),
			},
			expectedRunID: "run1",
			expectedUUID:  "92f356e3-4d4b-4c8a-bbe2-a886bfd0f63c",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			opts := sampleSheetOptions{}
			for _, opt := range c.options {
				err := opt(&opts)
				if err != nil {
					if !c.error {
						t.Error("got error, expected no error")
					}
				}
			}

			t.Logf("%+v", opts)

			if c.expectedRunID != "" && *opts.runId != c.expectedRunID {
				t.Errorf("expected run id %q, got %q", c.expectedRunID, *opts.runId)
			}

			if c.expectedUUID != "" && opts.uuid.String() != c.expectedUUID {
				t.Errorf("expected UUID %q, got %q", c.expectedUUID, opts.uuid.String())
			}
		})
	}
}
