package cleve

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
)

type RunCompletionStatus struct {
	Success bool
	Message string
}

type completionStatusNovaSeq struct {
	RunStatus string `xml:"RunStatus"`
	RunError  struct {
		Type    string `xml:"Type"`
		Message string `xml:"Message"`
	} `xml:"RunError"`
}

func (s completionStatusNovaSeq) toStatus() RunCompletionStatus {
	msg := s.RunStatus
	if s.RunError.Type != "" {
		msg += ": " + s.RunError.Type
	}
	if s.RunError.Message != "" {
		msg += ": " + s.RunError.Message
	}
	return RunCompletionStatus{
		Success: s.RunStatus == "RunCompleted",
		Message: msg,
	}
}

func (s completionStatusNovaSeq) valid() bool {
	return s.RunStatus != ""
}

type completionStatusNextSeq struct {
	CompletionStatus string `xml:"CompletionStatus"`
	ErrorDescription string `xml:"ErrorDescription"`
}

func (s completionStatusNextSeq) toStatus() RunCompletionStatus {
	msg := s.CompletionStatus
	if s.ErrorDescription != "" && s.ErrorDescription != "None" {
		msg += ": " + s.ErrorDescription
	}
	return RunCompletionStatus{
		Success: s.CompletionStatus == "CompletedAsPlanned",
		Message: msg,
	}
}

func (s completionStatusNextSeq) valid() bool {
	return s.CompletionStatus != ""
}

type completionStatusMiSeq struct {
	XMLName xml.Name
	Error   string `xml:"Error"`
	Warning string `xml:"Warning"`
}

func (s completionStatusMiSeq) toStatus() RunCompletionStatus {
	success := s.Error == "" && s.Warning == ""
	var msg string
	if success {
		msg = "Success"
	} else {
		if s.Error != "" {
			msg = "Error: " + s.Error
			if s.Warning != "" {
				msg += ", "
			}
		}
		if s.Warning != "" {
			msg += "Warning: " + s.Warning
		}
	}
	return RunCompletionStatus{
		Success: success,
		Message: msg,
	}
}

func (s completionStatusMiSeq) valid() bool {
	return s.XMLName.Local == "AnalysisJobInfo"
}

func ParseRunCompletionStatus(data []byte) (RunCompletionStatus, error) {
	var err error
	var novaseqStatus completionStatusNovaSeq
	if err = xml.Unmarshal(data, &novaseqStatus); novaseqStatus.valid() && err == nil {
		return novaseqStatus.toStatus(), nil
	}

	var nextseqStatus completionStatusNextSeq
	if err = xml.Unmarshal(data, &nextseqStatus); nextseqStatus.valid() && err == nil {
		return nextseqStatus.toStatus(), nil
	}

	var miseqStatus completionStatusMiSeq
	if err = xml.Unmarshal(data, &miseqStatus); miseqStatus.valid() && err == nil {
		return miseqStatus.toStatus(), nil
	}

	return RunCompletionStatus{}, fmt.Errorf("failed to parse completion status")
}

func ReadRunCompletionStatus(filename string) (RunCompletionStatus, error) {
	var status RunCompletionStatus

	f, err := os.Open(filename)
	if err != nil {
		return status, err
	}
	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(f)
	if err != nil {
		return status, err
	}

	return ParseRunCompletionStatus(data)
}
