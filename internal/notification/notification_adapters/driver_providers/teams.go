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
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/notification/notification_domain"
	"piko.sh/piko/internal/notification/notification_dto"
	"piko.sh/piko/internal/safeerror"
)

const (
	// teamsMaxMessageLength is the maximum message length in characters for Teams.
	teamsMaxMessageLength = 28000
)

// ErrTeamsResponseTooLarge indicates the Teams webhook reply exceeded the
// maximum size cap and was truncated before parsing.
var ErrTeamsResponseTooLarge = errors.New("teams response body exceeded maximum allowed size")

var _ notification_domain.NotificationProviderPort = (*TeamsProvider)(nil)

// teamsMessageCard holds the structure for a Microsoft Teams message card.
type teamsMessageCard struct {
	// Type specifies the MessageCard schema type for Teams.
	Type string `json:"@type"`

	// Context is the JSON-LD context URL for the message card schema.
	Context string `json:"@context"`

	// ThemeColor is the hex colour code for the card accent; empty uses the
	// default.
	ThemeColor string `json:"themeColor,omitempty"`

	// Summary is a short text summary of the card content.
	Summary string `json:"summary,omitempty"`

	// Title is the heading text for the message card.
	Title string `json:"title,omitempty"`

	// Sections contains the message card sections with detailed content.
	Sections []teamsSection `json:"sections"`
}

// teamsSection represents a section within a Teams message card.
type teamsSection struct {
	// ActivityTitle is the title text for the activity notification.
	ActivityTitle string `json:"activityTitle,omitempty"`

	// ActivitySubtitle is the secondary text shown below the activity title.
	ActivitySubtitle string `json:"activitySubtitle,omitempty"`

	// Facts holds the key-value pairs displayed in the Teams message section.
	Facts []teamsFact `json:"facts,omitempty"`

	// Markdown indicates whether the description uses Markdown format.
	Markdown bool `json:"markdown"`
}

// teamsFact represents a name-value pair shown in a Teams message section.
type teamsFact struct {
	// Name is the label for this fact in the Teams message card.
	Name string `json:"name"`

	// Value is the fact content displayed to the user.
	Value string `json:"value"`
}

// TeamsProvider sends notifications to Microsoft Teams using webhooks.
// It implements the NotificationProviderPort interface.
type TeamsProvider struct {
	// httpClient sends HTTP requests to the Teams webhook endpoint.
	httpClient *http.Client

	// webhookURL is the Microsoft Teams webhook endpoint for sending
	// notifications.
	webhookURL string
}

// Send delivers a notification to Microsoft Teams.
//
// Takes params (*notification_dto.SendParams) which contains the notification
// content and recipients.
//
// Returns error when the payload cannot be formatted, the request fails, or
// Teams returns an unsuccessful response.
func (t *TeamsProvider) Send(ctx context.Context, params *notification_dto.SendParams) error {
	payload, err := t.formatTeamsPayload(params)
	if err != nil {
		return fmt.Errorf("formatting teams payload: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, t.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("creating teams request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := t.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("sending to teams: %w", err)
	}
	defer drainAndCloseResponse(response)

	if response.StatusCode != http.StatusOK {
		statusErr := buildProviderStatusError("teams", response)
		if isClientError(response.StatusCode) {
			return safeerror.NewError(safeNotificationDeliveryFailed, statusErr)
		}
		return statusErr
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxNotificationResponseBytes+1))
	if err != nil {
		return fmt.Errorf("reading teams response: %w", err)
	}
	if len(body) > maxNotificationResponseBytes {
		return fmt.Errorf("teams response truncated at %d bytes: %w", maxNotificationResponseBytes, ErrTeamsResponseTooLarge)
	}

	if strings.TrimSpace(string(body)) != "1" {
		return fmt.Errorf("teams returned unexpected body: %s", string(body))
	}

	return nil
}

// SendBulk sends multiple notifications.
//
// Takes notifications ([]*notification_dto.SendParams) which contains the
// notifications to send.
//
// Returns error when any notification fails to send.
func (t *TeamsProvider) SendBulk(ctx context.Context, notifications []*notification_dto.SendParams) error {
	for _, params := range notifications {
		if err := t.Send(ctx, params); err != nil {
			return fmt.Errorf("sending Teams notification in bulk: %w", err)
		}
	}
	return nil
}

// SupportsBulkSending reports whether Teams supports bulk sending.
//
// Returns bool which is always false as Teams does not support bulk sending.
func (*TeamsProvider) SupportsBulkSending() bool {
	return false
}

// GetCapabilities returns the capabilities of the Teams provider.
//
// Returns notification_domain.ProviderCapabilities which describes the
// supported features and limits for the Teams notification provider.
func (*TeamsProvider) GetCapabilities() notification_domain.ProviderCapabilities {
	return notification_domain.ProviderCapabilities{
		SupportsRichFormatting: true,
		SupportsImages:         false,
		SupportsAttachments:    false,
		MaxMessageLength:       teamsMaxMessageLength,
		SupportsBulkSending:    false,
		RequiresAuthentication: false,
	}
}

// Close releases any resources held by the provider.
//
// Returns error when releasing resources fails.
func (*TeamsProvider) Close(_ context.Context) error {
	return nil
}

// formatTeamsPayload converts notification params to Teams MessageCard format.
//
// Takes params (*notification_dto.SendParams) which contains the notification
// content and context to format.
//
// Returns []byte which is the JSON-encoded MessageCard payload.
// Returns error when JSON marshalling fails.
func (t *TeamsProvider) formatTeamsPayload(params *notification_dto.SendParams) ([]byte, error) {
	themeColor := t.priorityToThemeColor(params.Context.Priority)
	title := t.determineTitle(params)

	section := teamsSection{
		ActivityTitle:    params.Content.Title,
		ActivitySubtitle: params.Content.Message,
		Facts:            []teamsFact{},
		Markdown:         true,
	}

	section.Facts = append(section.Facts, teamsFact{
		Name:  "Priority",
		Value: params.Context.Priority.String(),
	})

	if params.Context.Source != "" {
		section.Facts = append(section.Facts, teamsFact{
			Name:  "Source",
			Value: params.Context.Source,
		})
	}
	if params.Context.Environment != "" {
		section.Facts = append(section.Facts, teamsFact{
			Name:  "Environment",
			Value: params.Context.Environment,
		})
	}

	for key, value := range params.Content.Fields {
		section.Facts = append(section.Facts, teamsFact{
			Name:  key,
			Value: value,
		})
	}

	card := teamsMessageCard{
		Type:       "MessageCard",
		Context:    "http://schema.org/extensions",
		ThemeColor: themeColor,
		Summary:    params.Content.Title,
		Title:      title,
		Sections:   []teamsSection{section},
	}

	return json.Marshal(card)
}

// priorityToThemeColor maps notification priority to Teams theme colour.
//
// Takes priority (NotificationPriority) which specifies the notification level.
//
// Returns string which is the hex colour code for the Teams message theme.
func (*TeamsProvider) priorityToThemeColor(priority notification_dto.NotificationPriority) string {
	switch priority {
	case notification_dto.PriorityCritical:
		return "FF0000"
	case notification_dto.PriorityHigh:
		return "FFA500"
	case notification_dto.PriorityNormal:
		return "0078D4"
	default:
		return "808080"
	}
}

// determineTitle returns a title for the Teams card based on priority.
//
// Takes params (*notification_dto.SendParams) which provides the notification
// context including priority level.
//
// Returns string which is the formatted title with an appropriate emoji.
func (*TeamsProvider) determineTitle(params *notification_dto.SendParams) string {
	switch params.Context.Priority {
	case notification_dto.PriorityCritical:
		return "🚨 Critical Notification"
	case notification_dto.PriorityHigh:
		return "⚠️ High Priority Notification"
	default:
		return "📢 Notification"
	}
}

// NewTeamsProvider creates a new Microsoft Teams notification provider.
//
// Takes webhookURL (string) which specifies the Teams webhook endpoint.
// Takes client (*http.Client) which provides the HTTP client. If nil, a default
// client with a 30-second timeout is used.
//
// Returns notification_domain.NotificationProviderPort which is ready to send
// notifications.
func NewTeamsProvider(webhookURL string, client *http.Client) notification_domain.NotificationProviderPort {
	if client == nil {
		client = defaultHTTPClient
	}
	return &TeamsProvider{
		webhookURL: webhookURL,
		httpClient: client,
	}
}
