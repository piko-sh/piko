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
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/notification/notification_domain"
	"piko.sh/piko/internal/notification/notification_dto"
)

var _ notification_domain.NotificationProviderPort = (*DiscordProvider)(nil)

const (
	// discordColorError is the Discord embed colour for critical priority
	// messages.
	discordColorError = 15158332

	// discordColorWarn is the Discord embed colour for high priority
	// notifications.
	discordColorWarn = 15844367

	// discordColorInfo is the Discord embed colour for informational messages.
	discordColorInfo = 3447003

	// discordColorDefault is the fallback embed colour for Discord notifications.
	discordColorDefault = 9807270

	// discordMaxEmbedSize is the maximum character length for Discord embed
	// content.
	discordMaxEmbedSize = 6000
)

// discordEmbedField represents a single field within a Discord embed.
type discordEmbedField struct {
	// Name is the field title shown on the left side of the embed.
	Name string `json:"name"`

	// Value is the content text displayed for this embed field.
	Value string `json:"value"`

	// Inline indicates whether the field displays on the same line as others.
	Inline bool `json:"inline,omitempty"`
}

// discordEmbed holds the data for a Discord embed, used for rich messages.
type discordEmbed struct {
	// Title is the embed heading shown at the top.
	Title string `json:"title"`

	// Description is the main text content of the embed.
	Description string `json:"description"`

	// Timestamp is the ISO8601 time for this embed.
	Timestamp string `json:"timestamp"`

	// Fields contains additional field sections displayed in the embed.
	Fields []discordEmbedField `json:"fields,omitempty"`

	// Color is the embed sidebar colour as a decimal integer.
	Color int `json:"color"`
}

// discordPayload represents the complete payload sent to Discord webhooks.
type discordPayload struct {
	// Content is the text message to send.
	Content string `json:"content,omitempty"`

	// Embeds contains the rich embed objects for this message.
	Embeds []discordEmbed `json:"embeds"`
}

// DiscordProvider sends notifications to Discord using webhooks.
// It implements NotificationProviderPort.
type DiscordProvider struct {
	// httpClient is the HTTP client used to send webhook requests to Discord.
	httpClient *http.Client

	// webhookURL is the Discord webhook URL for sending notifications.
	webhookURL string
}

// Send delivers a notification to Discord.
//
// Takes params (*notification_dto.SendParams) which contains the notification
// content and metadata.
//
// Returns error when payload formatting fails, request creation fails, the
// HTTP request fails, or Discord returns a non-success status code.
func (d *DiscordProvider) Send(ctx context.Context, params *notification_dto.SendParams) error {
	payload, err := d.formatDiscordPayload(params)
	if err != nil {
		return fmt.Errorf("formatting discord payload: %w", err)
	}

	return sendHTTPJSONPayload(ctx, d.httpClient, d.webhookURL, payload, "discord")
}

// SendBulk sends multiple notifications by falling back to individual sends,
// as Discord webhooks do not support bulk operations.
//
// Takes notifications ([]*notification_dto.SendParams) which contains the
// notification data to send.
//
// Returns error when any individual send fails.
func (d *DiscordProvider) SendBulk(ctx context.Context, notifications []*notification_dto.SendParams) error {
	for _, params := range notifications {
		if err := d.Send(ctx, params); err != nil {
			return fmt.Errorf("sending Discord notification in bulk: %w", err)
		}
	}
	return nil
}

// SupportsBulkSending reports whether Discord supports bulk sending (it does
// not).
//
// Returns bool which is always false as Discord does not support bulk sending.
func (*DiscordProvider) SupportsBulkSending() bool {
	return false
}

// GetCapabilities returns the capabilities of the Discord provider.
//
// Returns notification_domain.ProviderCapabilities which describes what
// features the Discord provider supports.
func (*DiscordProvider) GetCapabilities() notification_domain.ProviderCapabilities {
	return notification_domain.ProviderCapabilities{
		SupportsRichFormatting: true,
		SupportsImages:         true,
		SupportsAttachments:    false,
		MaxMessageLength:       discordMaxEmbedSize,
		SupportsBulkSending:    false,
		RequiresAuthentication: false,
	}
}

// Close releases any resources held by the provider.
//
// Returns error when the provider fails to release resources.
func (*DiscordProvider) Close(_ context.Context) error {
	return nil
}

// formatDiscordPayload converts notification params to Discord embed format.
//
// Takes params (*notification_dto.SendParams) which contains the notification
// content and context to format.
//
// Returns []byte which is the JSON-encoded Discord webhook payload.
// Returns error when JSON marshalling fails.
func (d *DiscordProvider) formatDiscordPayload(params *notification_dto.SendParams) ([]byte, error) {
	color := d.priorityToDiscordColor(params.Context.Priority)

	var description strings.Builder
	if params.Content.Message != "" {
		description.WriteString(params.Content.Message)
	}

	if len(params.Content.Fields) > 0 {
		if description.Len() > 0 {
			description.WriteString("\n\n")
		}
		for key, value := range params.Content.Fields {
			_, _ = fmt.Fprintf(&description, "**%s**: `%s`\n", key, value)
		}
	}

	if params.Context.Source != "" || params.Context.Environment != "" {
		description.WriteString("\n---\n")
		if params.Context.Source != "" {
			_, _ = fmt.Fprintf(&description, "*Source: %s*", params.Context.Source)
		}
		if params.Context.Environment != "" {
			_, _ = fmt.Fprintf(&description, " | *Environment: %s*", params.Context.Environment)
		}
	}

	embed := discordEmbed{
		Title:       params.Content.Title,
		Description: description.String(),
		Timestamp:   params.Context.Timestamp.UTC().Format(time.RFC3339),
		Color:       color,
		Fields:      []discordEmbedField{},
	}

	payload := discordPayload{
		Embeds: []discordEmbed{embed},
	}

	return json.Marshal(payload)
}

// priorityToDiscordColor maps notification priority to Discord embed colour.
//
// Takes priority (NotificationPriority) which specifies the notification level.
//
// Returns int which is the Discord embed colour code for the given priority.
func (*DiscordProvider) priorityToDiscordColor(priority notification_dto.NotificationPriority) int {
	switch priority {
	case notification_dto.PriorityCritical:
		return discordColorError
	case notification_dto.PriorityHigh:
		return discordColorWarn
	case notification_dto.PriorityNormal:
		return discordColorInfo
	default:
		return discordColorDefault
	}
}

// NewDiscordProvider creates a new Discord notification provider.
//
// Takes webhookURL (string) which specifies the Discord webhook endpoint.
// Takes client (*http.Client) which provides the HTTP client. If nil, a default
// client with a 30-second timeout is used.
//
// Returns notification_domain.NotificationProviderPort which is ready to send
// notifications.
func NewDiscordProvider(webhookURL string, client *http.Client) notification_domain.NotificationProviderPort {
	if client == nil {
		client = defaultHTTPClient
	}
	return &DiscordProvider{
		webhookURL: webhookURL,
		httpClient: client,
	}
}
