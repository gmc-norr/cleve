package cleve

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type WebhookApiKey struct {
	Key   string
	Value string
}

func WebhookApiKeyFromString(s string) (WebhookApiKey, error) {
	key := WebhookApiKey{}
	if s == "" {
		return key, nil
	}
	parts := strings.SplitN(s, "=", 2)
	if len(parts) != 2 {
		return key, fmt.Errorf("failed to parse webhook api key")
	}
	key.Key = parts[0]
	key.Value = parts[1]
	return key, nil
}

type MarshableError struct {
	error
}

type MessageUnit int

const (
	UnitInvalid MessageUnit = iota
	UnitRun
	UnitAnalysis
)

func (u MessageUnit) String() string {
	switch u {
	case UnitRun:
		return "run"
	case UnitAnalysis:
		return "analysis"
	}
	return "undefined"
}

func (u MessageUnit) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.String())
}

type MessageType int

const (
	MessageStateUpdate MessageType = iota
)

func (t MessageType) String() string {
	switch t {
	case MessageStateUpdate:
		return "state_update"
	}
	return "undefined"
}

func (t MessageType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func NewMarshableError(err error) MarshableError {
	return MarshableError{
		error: err,
	}
}

func (err MarshableError) MarshalJSON() ([]byte, error) {
	if err.error == nil {
		return json.Marshal(nil)
	}
	return json.Marshal(err.Error())
}

type WebhookMessageRequest struct {
	Message     string
	Entity      any
	MessageType MessageType
}

type WebhookMessage struct {
	Unit        MessageUnit `json:"unit"`
	Id          string      `json:"id"`
	Platform    string      `json:"platform"`
	Message     any         `json:"message"`
	MessageType MessageType `json:"message_type"`
	State       State       `json:"state"`
	Path        string      `json:"path"`
	Time        time.Time   `json:"time"`
}

func NewRunMessage(run *Run, message string, messageType MessageType) WebhookMessage {
	return WebhookMessage{
		Unit:        UnitRun,
		Id:          run.RunID,
		Platform:    run.Platform,
		Message:     message,
		MessageType: messageType,
		State:       run.StateHistory.LastState(),
		Path:        run.Path,
		Time:        time.Now().Local(),
	}
}

func NewAnalysisMessage(analysis *Analysis, message string, messageType MessageType) WebhookMessage {
	return WebhookMessage{
		Unit:        UnitAnalysis,
		Id:          analysis.AnalysisId.String(),
		Platform:    analysis.Software,
		Message:     message,
		MessageType: messageType,
		State:       analysis.StateHistory.LastState(),
		Path:        analysis.Path,
		Time:        time.Now().Local(),
	}
}
