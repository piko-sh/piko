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
	"context"
	"fmt"
	"net/http"
	"time"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/notification/notification_domain"
	"piko.sh/piko/internal/notification/notification_dto"
	"piko.sh/piko/internal/safeerror"
)

var _ notification_domain.NotificationProviderPort = (*WebhookProvider)(nil)

const (
	// webhookMaxMessageLength is the maximum message length in bytes for webhook
	// notifications.
	webhookMaxMessageLength = 100000

	// httpStatusOKMin is the lowest HTTP status code indicating success.
	httpStatusOKMin = 200

	// httpStatusOKMax is the exclusive upper bound for successful HTTP status
	// codes.
	httpStatusOKMax = 300
)

// webhookPayload holds the JSON data sent to generic webhook endpoints.
type webhookPayload struct {
	// Timestamp is when the webhook event occurred.
	Timestamp time.Time `json:"timestamp"`

	// Priority is the urgency level of the webhook event.
	Priority string `json:"priority"`

	// Source is the origin identifier for the webhook event.
	Source string `json:"source"`

	// Environment is the deployment environment name.
	Environment string `json:"environment"`

	// Service is the name of the service that triggered the webhook.
	Service string `json:"service,omitempty"`

	// TraceID is a unique identifier for tracing requests; empty if not traced.
	TraceID string `json:"trace_id,omitempty"`

	// Title is the title text for the webhook notification.
	Title string `json:"title"`

	// Message is the text content of the webhook notification.
	Message string `json:"message"`

	// Fields maps custom field names to their values.
	Fields map[string]string `json:"fields,omitempty"`

	// ImageURL is the URL of an image to include in the webhook message.
	ImageURL string `json:"image_url,omitempty"`
}

// WebhookProvider sends notifications to webhook endpoints. It implements the
// NotificationProviderPort interface.
type WebhookProvider struct {
	// headers contains custom HTTP headers to include with each webhook request.
	headers map[string]string

	// httpClient sends HTTP requests to the webhook endpoint.
	httpClient *http.Client

	// webhookURL is the HTTP endpoint for sending webhook notifications.
	webhookURL string
}

// Send delivers a notification to the webhook endpoint.
//
// Takes params (*notification_dto.SendParams) which contains the notification
// data to send.
//
// Returns error when the payload cannot be formatted, the request fails, or
// the webhook returns a non-success status code.
func (w *WebhookProvider) Send(ctx context.Context, params *notification_dto.SendParams) error {
	payload, err := w.formatWebhookPayload(params)
	if err != nil {
		return fmt.Errorf("formatting webhook payload: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, w.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("creating webhook request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	for key, value := range w.headers {
		request.Header.Set(key, value)
	}

	response, err := w.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("sending to webhook: %w", err)
	}
	defer drainAndCloseResponse(response)

	if response.StatusCode < httpStatusOKMin || response.StatusCode >= httpStatusOKMax {
		statusErr := buildProviderStatusError("webhook", response)
		if isClientError(response.StatusCode) {
			return safeerror.NewError(safeNotificationDeliveryFailed, statusErr)
		}
		return statusErr
	}

	return nil
}

// SendBulk sends multiple notifications.
//
// Takes notifications ([]*notification_dto.SendParams) which contains the
// notifications to send.
//
// Returns error when any notification fails to send.
func (w *WebhookProvider) SendBulk(ctx context.Context, notifications []*notification_dto.SendParams) error {
	for _, params := range notifications {
		if err := w.Send(ctx, params); err != nil {
			return fmt.Errorf("sending webhook notification in bulk: %w", err)
		}
	}
	return nil
}

// SupportsBulkSending reports whether webhook supports bulk sending.
//
// Returns bool which is always false for webhooks.
func (*WebhookProvider) SupportsBulkSending() bool {
	return false
}

// GetCapabilities returns the capabilities of the webhook provider.
//
// Returns notification_domain.ProviderCapabilities which describes what
// features this provider supports.
func (*WebhookProvider) GetCapabilities() notification_domain.ProviderCapabilities {
	return notification_domain.ProviderCapabilities{
		SupportsRichFormatting: false,
		SupportsImages:         true,
		SupportsAttachments:    false,
		MaxMessageLength:       webhookMaxMessageLength,
		SupportsBulkSending:    false,
		RequiresAuthentication: false,
	}
}

// Close releases any resources held by the provider.
//
// Returns error when releasing resources fails.
func (*WebhookProvider) Close(_ context.Context) error {
	return nil
}

// formatWebhookPayload converts notification params to generic webhook JSON
// format.
//
// Takes params (*notification_dto.SendParams) which provides the notification
// content and context to format.
//
// Returns []byte which contains the JSON-encoded webhook payload.
// Returns error when JSON marshalling fails.
func (*WebhookProvider) formatWebhookPayload(params *notification_dto.SendParams) ([]byte, error) {
	payload := webhookPayload{
		Timestamp:   params.Context.Timestamp,
		Priority:    params.Context.Priority.String(),
		Source:      params.Context.Source,
		Environment: params.Context.Environment,
		Service:     params.Context.Service,
		TraceID:     params.Context.TraceID,
		Title:       params.Content.Title,
		Message:     params.Content.Message,
		Fields:      params.Content.Fields,
		ImageURL:    params.Content.ImageURL,
	}

	return json.Marshal(payload)
}

// NewWebhookProvider creates a new generic webhook notification provider.
//
// Takes webhookURL (string) which specifies the webhook endpoint.
// Takes headers (map[string]string) which provides optional HTTP headers.
// Takes client (*http.Client) which provides the HTTP client. If nil, a default
// client with a 30-second timeout is used.
//
// Returns notification_domain.NotificationProviderPort which is ready to send
// notifications.
func NewWebhookProvider(webhookURL string, headers map[string]string, client *http.Client) notification_domain.NotificationProviderPort {
	if client == nil {
		client = defaultHTTPClient
	}
	return &WebhookProvider{
		webhookURL: webhookURL,
		headers:    headers,
		httpClient: client,
	}
}
