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
	"strings"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/notification/notification_domain"
	"piko.sh/piko/internal/notification/notification_dto"
	"piko.sh/piko/internal/safeerror"
)

var _ notification_domain.NotificationProviderPort = (*SlackProvider)(nil)

const (
	// slackMaxMessageLength is the maximum character length for a Slack message.
	slackMaxMessageLength = 40000

	// slackBlockKeyType is the key name for the type field in Slack Block Kit.
	slackBlockKeyType = "type"

	// slackBlockKeyText is the key for text content in Slack Block Kit fields.
	slackBlockKeyText = "text"
)

// SlackProvider sends notifications to Slack using webhooks. It implements
// the NotificationProviderPort interface.
type SlackProvider struct {
	// httpClient sends HTTP requests to the Slack API.
	httpClient *http.Client

	// webhookURL is the Slack webhook URL for sending notifications.
	webhookURL string
}

// Send delivers a notification to Slack.
//
// Takes params (*notification_dto.SendParams) which contains the notification
// content and metadata.
//
// Returns error when the payload cannot be formatted, the request fails, or
// Slack returns a non-OK status.
func (s *SlackProvider) Send(ctx context.Context, params *notification_dto.SendParams) error {
	payload, err := s.formatSlackPayload(params)
	if err != nil {
		return fmt.Errorf("formatting slack payload: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("creating slack request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := s.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("sending to slack: %w", err)
	}
	defer drainAndCloseResponse(response)

	if response.StatusCode != http.StatusOK {
		statusErr := buildProviderStatusError("slack", response)
		if isClientError(response.StatusCode) {
			return safeerror.NewError(safeNotificationDeliveryFailed, statusErr)
		}
		return statusErr
	}

	return nil
}

// SendBulk sends multiple notifications by falling back to individual sends,
// as Slack webhooks do not support bulk operations.
//
// Takes notifications ([]*notification_dto.SendParams) which contains the
// notification payloads to send.
//
// Returns error when any individual send fails.
func (s *SlackProvider) SendBulk(ctx context.Context, notifications []*notification_dto.SendParams) error {
	for _, params := range notifications {
		if err := s.Send(ctx, params); err != nil {
			return fmt.Errorf("sending Slack notification in bulk: %w", err)
		}
	}
	return nil
}

// SupportsBulkSending reports whether Slack supports bulk sending (it does
// not).
//
// Returns bool which is always false for this provider.
func (*SlackProvider) SupportsBulkSending() bool {
	return false
}

// GetCapabilities returns the capabilities of the Slack provider.
//
// Returns notification_domain.ProviderCapabilities which describes the
// supported features such as rich formatting, images, and message limits.
func (*SlackProvider) GetCapabilities() notification_domain.ProviderCapabilities {
	return notification_domain.ProviderCapabilities{
		SupportsRichFormatting: true,
		SupportsImages:         true,
		SupportsAttachments:    false,
		MaxMessageLength:       slackMaxMessageLength,
		SupportsBulkSending:    false,
		RequiresAuthentication: false,
	}
}

// Close releases any resources held by the provider.
//
// Returns error when resources cannot be released.
func (*SlackProvider) Close(_ context.Context) error {
	return nil
}

// formatSlackPayload converts notification params to Slack Block Kit format.
//
// Takes params (*notification_dto.SendParams) which contains the notification
// content and context to format.
//
// Returns []byte which is the JSON-encoded Slack payload.
// Returns error when JSON marshalling fails.
func (s *SlackProvider) formatSlackPayload(params *notification_dto.SendParams) ([]byte, error) {
	blocks := make([]map[string]any, 0)

	if params.Content.Title != "" {
		blocks = append(blocks, s.buildHeaderBlock(params))
	}
	if params.Content.Message != "" {
		blocks = append(blocks, buildMarkdownSection(params.Content.Message))
	}
	if len(params.Content.Fields) > 0 {
		blocks = append(blocks, s.buildFieldsBlock(params.Content.Fields))
	}
	if params.Content.ImageURL != "" {
		blocks = append(blocks, buildImageBlock(params.Content.ImageURL))
	}
	if contextBlock := buildContextBlock(params.Context.Source, params.Context.Environment); contextBlock != nil {
		blocks = append(blocks, contextBlock)
	}

	slackPayload := map[string]any{
		"text":   params.Content.Title,
		"blocks": blocks,
	}

	return json.Marshal(slackPayload)
}

// buildHeaderBlock creates a Slack header block with priority emoji.
//
// Takes params (*notification_dto.SendParams) which provides the notification
// content and priority context.
//
// Returns map[string]any which contains the formatted Slack header block.
func (s *SlackProvider) buildHeaderBlock(params *notification_dto.SendParams) map[string]any {
	emoji := s.getPriorityEmoji(params.Context.Priority)
	return map[string]any{
		slackBlockKeyType: "header",
		slackBlockKeyText: map[string]any{
			slackBlockKeyType: "plain_text",
			slackBlockKeyText: emoji + " " + params.Content.Title,
			"emoji":           true,
		},
	}
}

// buildFieldsBlock creates a Slack section block with formatted fields.
//
// Takes fields (map[string]string) which contains key-value pairs to format.
//
// Returns map[string]any which is a Slack section block with markdown content.
func (*SlackProvider) buildFieldsBlock(fields map[string]string) map[string]any {
	var fieldText strings.Builder
	for key, value := range fields {
		_, _ = fmt.Fprintf(&fieldText, "*%s*: `%s`\n", key, value)
	}
	return buildMarkdownSection(fieldText.String())
}

// getPriorityEmoji returns an emoji for the notification priority.
//
// Takes priority (NotificationPriority) which specifies the notification level.
//
// Returns string which is the Slack emoji code for the given priority.
func (*SlackProvider) getPriorityEmoji(priority notification_dto.NotificationPriority) string {
	switch priority {
	case notification_dto.PriorityCritical:
		return ":rotating_light:"
	case notification_dto.PriorityHigh:
		return ":warning:"
	case notification_dto.PriorityNormal:
		return ":information_source:"
	case notification_dto.PriorityLow:
		return ":bulb:"
	default:
		return ":bell:"
	}
}

// NewSlackProvider creates a new Slack notification provider.
//
// Takes webhookURL (string) which specifies the Slack webhook endpoint.
// Takes client (*http.Client) which provides the HTTP client. If nil, a default
// client with a 30-second timeout is used.
//
// Returns notification_domain.NotificationProviderPort which is ready to send
// notifications.
func NewSlackProvider(webhookURL string, client *http.Client) notification_domain.NotificationProviderPort {
	if client == nil {
		client = defaultHTTPClient
	}
	return &SlackProvider{
		webhookURL: webhookURL,
		httpClient: client,
	}
}

// buildMarkdownSection creates a Slack section block with markdown text.
//
// Takes text (string) which contains the markdown content for the section.
//
// Returns map[string]any which is the formatted Slack block structure.
func buildMarkdownSection(text string) map[string]any {
	return map[string]any{
		slackBlockKeyType: "section",
		slackBlockKeyText: map[string]string{slackBlockKeyType: "mrkdwn", slackBlockKeyText: text},
	}
}

// buildImageBlock creates a Slack image block.
//
// Takes imageURL (string) which specifies the URL of the image to display.
//
// Returns map[string]any which contains the Slack block structure for an image.
func buildImageBlock(imageURL string) map[string]any {
	return map[string]any{
		slackBlockKeyType: "image",
		"image_url":       imageURL,
		"alt_text":        "notification image",
	}
}

// buildContextBlock creates a Slack context block with source and environment.
//
// Takes source (string) which specifies the source label for the context.
// Takes environment (string) which specifies the environment label.
//
// Returns map[string]any which contains the Slack context block, or nil if
// both source and environment are empty.
func buildContextBlock(source, environment string) map[string]any {
	elements := make([]map[string]any, 0)
	if source != "" {
		elements = append(elements, map[string]any{slackBlockKeyType: "mrkdwn", slackBlockKeyText: fmt.Sprintf("*Source:* %s", source)})
	}
	if environment != "" {
		elements = append(elements, map[string]any{slackBlockKeyType: "mrkdwn", slackBlockKeyText: fmt.Sprintf("*Environment:* %s", environment)})
	}
	if len(elements) == 0 {
		return nil
	}
	return map[string]any{slackBlockKeyType: "context", "elements": elements}
}
