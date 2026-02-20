package cleve

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

type Webhook struct {
	http.Client
	URL          string
	APIKey       WebhookApiKey
	Method       string
	CleveVersion string
}

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

func NewAuthWebhook(url string, apiKey WebhookApiKey) *Webhook {
	return &Webhook{
		Client: http.Client{},
		URL:    url,
		APIKey: apiKey,
		Method: "POST",
	}
}

func NewWebhook(url string) *Webhook {
	return NewAuthWebhook(url, WebhookApiKey{})
}

func (h *Webhook) DisableTLSVerification() {
	h.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
}

func (h *Webhook) Check() error {
	r, err := http.NewRequest("GET", h.URL, nil)
	if err != nil {
		return err
	}
	if h.APIKey.Key != "" {
		r.Header.Add(h.APIKey.Key, h.APIKey.Value)
	}
	r.Header.Add("Content-Type", "application/json")
	res, err := h.Do(r)
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("%s", res.Status)
	}
	return nil
}

func (h *Webhook) SetCertificates(certs string) error {
	certFile, err := os.Open(certs)
	if err != nil {
		return fmt.Errorf("failed to open certificate file: %w", err)
	}
	caCert, err := io.ReadAll(certFile)
	if err != nil {
		return fmt.Errorf("failed to read certificates: %w", err)
	}
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		return fmt.Errorf("failed to parse certificates: %w", err)
	}
	h.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: caCertPool,
		},
	}
	return nil
}

func (h *Webhook) webhookRequest(payload any) (*http.Request, error) {
	switch pt := payload.(type) {
	case WebhookMessage:
		pt.Time = time.Now()
		payload = pt
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	slog.Debug("webhook request", "url", h.URL, "payload", jsonPayload)
	bodyReader := bytes.NewReader(jsonPayload)
	r, err := http.NewRequest(h.Method, h.URL, bodyReader)
	if err != nil {
		return r, err
	}
	if h.APIKey.Key != "" && h.APIKey.Value != "" {
		r.Header.Add(h.APIKey.Key, h.APIKey.Value)
	}
	r.Header.Add("X-Cleve-Version", h.CleveVersion)
	r.Header.Add("Content-Type", "application/json")
	return r, nil
}

func (h *Webhook) Send(payload any) error {
	r, err := h.webhookRequest(payload)
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}
	res, err := h.Do(r)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	slog.Debug("webhook response", "url", h.URL, "status", res.Status)
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusAccepted {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("failed to read webhook response body: %w", err)
		}
		return fmt.Errorf("webhook denied: status=%s, body=%s", res.Status, body)
	}
	return nil
}
