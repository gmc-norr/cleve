package cleve

import (
	"encoding/xml"
	"io"
	"os"
)

type RunCompletionStatus struct {
	Status  string
	Message string
}

func ParseRunCompletionStatus(data []byte) (RunCompletionStatus, error) {
	type xmlData struct {
		CompletionStatus string `xml:"CompletionStatus"`
		RunStatus        string `xml:"RunStatus"`
		ErrorDescription string `xml:"ErrorDescription"`
	}
	var status RunCompletionStatus
	var d xmlData

	if err := xml.Unmarshal(data, &d); err != nil {
		return status, err
	}

	if d.CompletionStatus != "" {
		switch d.CompletionStatus {
		case "CompletedAsPlanned":
			status.Status = "success"
			status.Message = d.CompletionStatus
		default:
			status.Status = "error"
		}
	}

	if d.RunStatus != "" {
		switch d.RunStatus {
		case "RunCompleted":
			status.Status = "success"
			status.Message = d.RunStatus
		default:
			status.Status = "error"
			status.Message = d.RunStatus
		}
	}

	if d.ErrorDescription != "" && d.ErrorDescription != "None" {
		status.Message = d.ErrorDescription
	}

	return status, nil
}

func ReadRunCompletionStatus(filename string) (RunCompletionStatus, error) {
	var status RunCompletionStatus

	f, err := os.Open(filename)
	if err != nil {
		return status, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return status, err
	}

	return ParseRunCompletionStatus(data)
}
