package cleve

import (
	"testing"
)

func TestParseRunCompletionStatus(t *testing.T) {
	cases := []struct {
		name    string
		xml     []byte
		error   bool
		success bool
		message string
	}{
		{
			name:    "novaseq success",
			xml:     []byte(`<?xml version="1.0" encoding="utf-8"?><RunCompletionStatus><RunStatus>RunCompleted</RunStatus></RunCompletionStatus>`),
			success: true,
			message: "RunCompleted",
		},
		{
			name: "novaseq error",
			xml: []byte(`<?xml version="1.0" encoding="utf-8"?>
				<RunCompletionStatus>
					<RunStatus>RunErrored</RunStatus>
					<RunError>
						<Type>Illumina.Instrument.Recipe.Runtime.Exceptions.RecipeExecutionFailedException</Type>
						<Message>I/O data integrity check failed</Message>
					</RunError>
				</RunCompletionStatus>`),
			success: false,
			message: "RunErrored: Illumina.Instrument.Recipe.Runtime.Exceptions.RecipeExecutionFailedException: I/O data integrity check failed",
		},
		{
			name: "nextseq success",
			xml: []byte(`<?xml version="1.0"?>
				<RunCompletionStatus>
					<CompletionStatus>CompletedAsPlanned</CompletionStatus>
					<ErrorDescription>None</ErrorDescription>
				</RunCompletionStatus>`),
			success: true,
			message: "CompletedAsPlanned",
		},
		{
			name: "nextseq error",
			xml: []byte(`<?xml version="1.0"?>
				<RunCompletionStatus>
					<CompletionStatus>UserEndedEarly</CompletionStatus>
					<ErrorDescription>Thread was aborted</ErrorDescription>
				</RunCompletionStatus>`),
			success: false,
			message: "UserEndedEarly: Thread was aborted",
		},
		{
			name:    "miseq success",
			xml:     []byte(`<?xml version="1.0"?><AnalysisJobInfo></AnalysisJobInfo>`),
			success: true,
			message: "Success",
		},
		{
			name:    "miseq error",
			xml:     []byte(`<?xml version="1.0"?><AnalysisJobInfo><Error>Sequencing failed</Error></AnalysisJobInfo>`),
			success: false,
			message: "Error: Sequencing failed",
		},
		{
			name:    "miseq warning",
			xml:     []byte(`<?xml version="1.0"?><AnalysisJobInfo><Warning>Stuff went wrong</Warning></AnalysisJobInfo>`),
			success: false,
			message: "Warning: Stuff went wrong",
		},
		{
			name:    "miseq error and warning",
			xml:     []byte(`<?xml version="1.0"?><AnalysisJobInfo><Error>Sequencing failed</Error><Warning>Stuff went wrong</Warning></AnalysisJobInfo>`),
			success: false,
			message: "Error: Sequencing failed, Warning: Stuff went wrong",
		},
		{
			name:  "invalid document",
			xml:   []byte(`<?xml version="1.0"?><OtherCompletionStatus><Error>Sequencing failed</Error><Warning>Stuff went wrong</Warning></OtherCompletionStatus>`),
			error: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rct, err := ParseRunCompletionStatus(c.xml)
			if c.error != (err != nil) {
				t.Fatal(err.Error())
			}
			if c.success != rct.Success {
				t.Errorf(`expected success to be %t, got %t`, c.success, rct.Success)
			}
			if c.message != rct.Message {
				t.Errorf(`expected message "%s", got "%s"`, c.message, rct.Message)
			}
		})
	}
}
