// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package driver_providers

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"piko.sh/piko/internal/json"
	"github.com/cespare/xxhash/v2"
	"piko.sh/piko/internal/notification/notification_domain"
	"piko.sh/piko/internal/notification/notification_dto"
)

var _ notification_domain.NotificationProviderPort = (*PagerDutyProvider)(nil)

const (
	// pagerDutyEventsURL is the PagerDuty Events API v2 endpoint for sending
	// alerts.
	pagerDutyEventsURL = "https://events.pagerduty.com/v2/enqueue"

	// pagerDutySummaryMaxLength is the maximum length of a PagerDuty summary.
	pagerDutySummaryMaxLength = 1024

	// pagerDutyDedupKeyMaxLength is the maximum length for a PagerDuty dedup key.
	pagerDutyDedupKeyMaxLength = 255

	// pagerDutyTruncationSuffix is the number of characters reserved for "...".
	pagerDutyTruncationSuffix = 3

	// httpStatusSuccessMin is the minimum HTTP status code for success responses.
	httpStatusSuccessMin = 200

	// httpStatusSuccessMax is the upper bound (exclusive) for successful HTTP
	// status codes.
	httpStatusSuccessMax = 300
)

// pagerDutyPayload holds the details sent to PagerDuty in an event.
type pagerDutyPayload struct {
	// CustomDetails holds additional context as key-value pairs.
	CustomDetails map[string]any `json:"custom_details,omitempty"`

	// Summary is a brief description of the incident for display.
	Summary string `json:"summary"`

	// Source identifies the origin of this PagerDuty alert.
	Source string `json:"source"`

	// Severity indicates the alert level (critical, error, warning, or info).
	Severity string `json:"severity"`

	// Timestamp is when the alert was triggered in ISO 8601 format.
	Timestamp string `json:"timestamp"`

	// Component identifies the part of the system that triggered the alert.
	Component string `json:"component,omitempty"`

	// Group is the logical grouping for the alert.
	Group string `json:"group,omitempty"`

	// Class specifies the severity class of the event (e.g. "error", "warning").
	Class string `json:"class,omitempty"`
}

// pagerDutyEvent represents a complete PagerDuty event including payload and
// routing.
type pagerDutyEvent struct {
	// Payload contains the event details sent to PagerDuty.
	Payload pagerDutyPayload `json:"payload"`

	// RoutingKey is the integration key for routing the event to the correct
	// service.
	RoutingKey string `json:"routing_key"`

	// EventAction specifies the type of event (trigger, acknowledge, or resolve).
	EventAction string `json:"event_action"`

	// DedupKey is the deduplication key for grouping related events.
	DedupKey string `json:"dedup_key"`
}

// PagerDutyProvider sends alerts to PagerDuty using the Events API v2.
// It implements NotificationProviderPort.
type PagerDutyProvider struct {
	// httpClient sends HTTP requests to the PagerDuty API.
	httpClient *http.Client

	// routingKey is the PagerDuty routing key for event delivery.
	routingKey string
}

// Send delivers a notification to PagerDuty.
//
// Takes params (*notification_dto.SendParams) which contains the notification
// details to send.
//
// Returns error when payload formatting fails, request creation fails, the HTTP
// request fails, or PagerDuty returns a non-success status code.
func (p *PagerDutyProvider) Send(ctx context.Context, params *notification_dto.SendParams) error {
	payload, err := p.formatPagerDutyPayload(params)
	if err != nil {
		return fmt.Errorf("formatting pagerduty payload: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, pagerDutyEventsURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("creating pagerduty request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := p.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("sending to pagerduty: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < httpStatusSuccessMin || response.StatusCode >= httpStatusSuccessMax {
		return fmt.Errorf("pagerduty returned status %d: %s", response.StatusCode, response.Status)
	}

	return nil
}

// SendBulk sends multiple notifications.
//
// Takes notifications ([]*notification_dto.SendParams) which contains the
// notification parameters to send.
//
// Returns error when any notification fails to send.
func (p *PagerDutyProvider) SendBulk(ctx context.Context, notifications []*notification_dto.SendParams) error {
	for _, params := range notifications {
		if err := p.Send(ctx, params); err != nil {
			return fmt.Errorf("sending PagerDuty notification in bulk: %w", err)
		}
	}
	return nil
}

// SupportsBulkSending reports whether PagerDuty supports bulk sending.
//
// Returns bool which is always false as PagerDuty does not support bulk
// sending.
func (*PagerDutyProvider) SupportsBulkSending() bool {
	return false
}

// GetCapabilities returns the capabilities of the PagerDuty provider.
//
// Returns notification_domain.ProviderCapabilities which describes the
// supported features and limits of this provider.
func (*PagerDutyProvider) GetCapabilities() notification_domain.ProviderCapabilities {
	return notification_domain.ProviderCapabilities{
		SupportsRichFormatting: false,
		SupportsImages:         false,
		SupportsAttachments:    false,
		MaxMessageLength:       pagerDutySummaryMaxLength,
		SupportsBulkSending:    false,
		RequiresAuthentication: true,
	}
}

// Close releases any resources held by the provider.
//
// Returns error when resources cannot be released.
func (*PagerDutyProvider) Close(_ context.Context) error {
	return nil
}

// formatPagerDutyPayload converts notification params to PagerDuty Events API
// format.
//
// Takes params (*notification_dto.SendParams) which contains the notification
// content and context to format.
//
// Returns []byte which is the JSON-encoded PagerDuty event payload.
// Returns error when JSON marshalling fails.
func (p *PagerDutyProvider) formatPagerDutyPayload(params *notification_dto.SendParams) ([]byte, error) {
	event := pagerDutyEvent{
		RoutingKey:  p.routingKey,
		EventAction: "trigger",
		DedupKey:    buildDedupKey(params.Content.Title, params.Context.Source),
		Payload: pagerDutyPayload{
			Summary:       buildSummary(params.Content.Title, params.Content.Message),
			Source:        getSource(params.Context.Source),
			Severity:      priorityToPagerDutySeverity(params.Context.Priority),
			Timestamp:     params.Context.Timestamp.Format(time.RFC3339),
			Component:     params.Context.Source,
			CustomDetails: buildCustomDetails(params),
		},
	}
	return json.Marshal(event)
}

// NewPagerDutyProvider creates a new PagerDuty notification provider.
//
// Takes routingKey (string) which is the PagerDuty integration key.
// Takes client (*http.Client) which provides the HTTP client. If nil, a default
// client with a 30-second timeout is used.
//
// Returns notification_domain.NotificationProviderPort which is ready to send
// notifications.
func NewPagerDutyProvider(routingKey string, client *http.Client) notification_domain.NotificationProviderPort {
	if client == nil {
		client = defaultHTTPClient
	}
	return &PagerDutyProvider{
		routingKey: routingKey,
		httpClient: client,
	}
}

// buildSummary returns a truncated summary from title or message.
//
// Takes title (string) which is the preferred summary text if not empty.
// Takes message (string) which is used as the summary when title is empty.
//
// Returns string which is the summary, truncated if it exceeds the maximum
// length.
func buildSummary(title, message string) string {
	summary := message
	if title != "" {
		summary = title
	}
	if len(summary) > pagerDutySummaryMaxLength {
		summary = summary[:pagerDutySummaryMaxLength-pagerDutyTruncationSuffix] + "..."
	}
	return summary
}

// getSource returns the source hostname or a fallback.
//
// Takes contextSource (string) which provides a fallback value if the hostname
// cannot be determined.
//
// Returns string which is the hostname, the context source, or the default
// value "piko-application".
func getSource(contextSource string) string {
	source, _ := os.Hostname()
	return cmp.Or(source, contextSource, "piko-application")
}

// buildCustomDetails builds the custom details map for PagerDuty.
//
// Takes params (*notification_dto.SendParams) which provides the notification
// content and context to include in the details.
//
// Returns map[string]any which contains the formatted details for the alert.
func buildCustomDetails(params *notification_dto.SendParams) map[string]any {
	details := make(map[string]any)
	if params.Content.Message != "" {
		details["message"] = params.Content.Message
	}
	if params.Content.Title != "" {
		details["title"] = params.Content.Title
	}
	details["priority"] = params.Context.Priority.String()
	details["environment"] = params.Context.Environment
	if params.Context.Service != "" {
		details["service"] = params.Context.Service
	}
	for key, value := range params.Content.Fields {
		details[key] = value
	}
	return details
}

// buildDedupKey generates a deduplication key from title and source.
//
// Takes title (string) which provides the primary text for the hash.
// Takes source (string) which provides optional additional text for the hash.
//
// Returns string which is the hexadecimal hash, truncated to the maximum
// PagerDuty dedup key length.
func buildDedupKey(title, source string) string {
	hasher := xxhash.New()
	_, _ = hasher.WriteString(title)
	if source != "" {
		_, _ = hasher.WriteString(source)
	}
	dedupKey := fmt.Sprintf("%x", hasher.Sum(nil))
	if len(dedupKey) > pagerDutyDedupKeyMaxLength {
		dedupKey = dedupKey[:pagerDutyDedupKeyMaxLength]
	}
	return dedupKey
}

// priorityToPagerDutySeverity maps notification priority to PagerDuty severity.
//
// Takes priority (NotificationPriority) which is the notification priority to
// convert.
//
// Returns string which is the PagerDuty severity level (critical, error,
// warning, or info).
func priorityToPagerDutySeverity(priority notification_dto.NotificationPriority) string {
	switch priority {
	case notification_dto.PriorityCritical:
		return "critical"
	case notification_dto.PriorityHigh:
		return "error"
	case notification_dto.PriorityNormal:
		return "warning"
	default:
		return "info"
	}
}
