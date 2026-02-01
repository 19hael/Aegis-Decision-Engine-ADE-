package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SlackNotifier sends notifications to Slack
type SlackNotifier struct {
	webhookURL string
	httpClient *http.Client
}

// NewSlackNotifier creates a new Slack notifier
func NewSlackNotifier(webhookURL string) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SlackMessage represents a Slack message
type SlackMessage struct {
	Text        string       `json:"text,omitempty"`
	Blocks      []Block      `json:"blocks,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Block represents a Slack block
type Block struct {
	Type string      `json:"type"`
	Text *TextObject `json:"text,omitempty"`
}

// TextObject represents text in a block
type TextObject struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Emoji bool   `json:"emoji,omitempty"`
}

// Attachment represents a Slack attachment
type Attachment struct {
	Color  string  `json:"color"`
	Title  string  `json:"title"`
	Text   string  `json:"text"`
	Fields []Field `json:"fields"`
	Footer string  `json:"footer"`
	TS     int64   `json:"ts"`
}

// Field represents a field in an attachment
type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// NotifyDecision sends a decision notification
func (s *SlackNotifier) NotifyDecision(ctx context.Context, decision DecisionNotification) error {
	color := "good"
	if decision.Result == "deny" || decision.RiskScore > 0.7 {
		color = "danger"
	} else if decision.RiskScore > 0.4 {
		color = "warning"
	}

	msg := SlackMessage{
		Attachments: []Attachment{
			{
				Color: color,
				Title: fmt.Sprintf("ADE Decision: %s", decision.Result),
				Text:  decision.Description,
				Fields: []Field{
					{Title: "Service", Value: decision.ServiceID, Short: true},
					{Title: "Action", Value: decision.Action, Short: true},
					{Title: "Risk Score", Value: fmt.Sprintf("%.2f", decision.RiskScore), Short: true},
					{Title: "Confidence", Value: fmt.Sprintf("%.2f", decision.Confidence), Short: true},
				},
				Footer: "Aegis Decision Engine",
				TS:     time.Now().Unix(),
			},
		},
	}

	return s.send(ctx, msg)
}

// NotifyAlert sends an alert notification
func (s *SlackNotifier) NotifyAlert(ctx context.Context, alert AlertNotification) error {
	color := "danger"
	if alert.Severity == "warning" {
		color = "warning"
	} else if alert.Severity == "info" {
		color = "#439FE0"
	}

	msg := SlackMessage{
		Attachments: []Attachment{
			{
				Color: color,
				Title: fmt.Sprintf("🚨 ADE Alert: %s", alert.Title),
				Text:  alert.Message,
				Fields: []Field{
					{Title: "Service", Value: alert.ServiceID, Short: true},
					{Title: "Severity", Value: alert.Severity, Short: true},
				},
				Footer: "Aegis Decision Engine",
				TS:     time.Now().Unix(),
			},
		},
	}

	return s.send(ctx, msg)
}

func (s *SlackNotifier) send(ctx context.Context, msg SlackMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack notification failed with status: %d", resp.StatusCode)
	}

	return nil
}

// DecisionNotification represents a decision notification
type DecisionNotification struct {
	DecisionID  string
	ServiceID   string
	Result      string
	Action      string
	RiskScore   float64
	Confidence  float64
	Description string
}

// AlertNotification represents an alert notification
type AlertNotification struct {
	Title     string
	Message   string
	ServiceID string
	Severity  string
}
