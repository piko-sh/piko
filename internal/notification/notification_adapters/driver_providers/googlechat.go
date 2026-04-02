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

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/notification/notification_domain"
	"piko.sh/piko/internal/notification/notification_dto"
)

var _ notification_domain.NotificationProviderPort = (*GoogleChatProvider)(nil)

const (
	// googleChatMaxMessageLength is the maximum number of characters allowed in a
	// single Google Chat message.
	googleChatMaxMessageLength = 4096
)

// googleChatPayload holds the data for a Google Chat Card v2 API message.
type googleChatPayload struct {
	// CardsV2 contains the list of cards to display in the message.
	CardsV2 []googleChatCard `json:"cardsV2"`
}

// googleChatCard represents a single card in a Google Chat message payload.
type googleChatCard struct {
	// CardID is the unique identifier for this card.
	CardID string `json:"cardId"`

	// Card contains the structured body content for the Google Chat card.
	Card googleChatCardBody `json:"card"`
}

// googleChatCardBody holds the main content of a Google Chat card.
type googleChatCardBody struct {
	// Header specifies the card header with title and optional subtitle.
	Header *googleChatHeader `json:"header,omitempty"`

	// Sections contains the card sections to display.
	Sections []*googleChatSection `json:"sections"`
}

// googleChatHeader holds the title and subtitle for a card.
type googleChatHeader struct {
	// Title is the header title text displayed in Google Chat cards.
	Title string `json:"title"`

	// Subtitle is the secondary text shown below the title.
	Subtitle string `json:"subtitle"`

	// ImageURL is the URL for the header image; empty means no image.
	ImageURL string `json:"imageUrl,omitempty"`
}

// googleChatSection represents a section within a Google Chat card.
type googleChatSection struct {
	// Widgets contains the interactive components displayed in this section.
	Widgets []googleChatWidget `json:"widgets"`
}

// googleChatWidget represents a single widget within a Google Chat section.
type googleChatWidget struct {
	// TextParagraph contains formatted text to display in the widget.
	TextParagraph *googleChatTextParagraph `json:"textParagraph,omitempty"`

	// KeyValue specifies a key-value pair widget for displaying labelled content.
	KeyValue *googleChatKeyValue `json:"keyValue,omitempty"`
}

// googleChatTextParagraph holds the text content for a paragraph widget.
type googleChatTextParagraph struct {
	// Text is the paragraph content to display.
	Text string `json:"text"`
}

// googleChatKeyValue represents a key-value pair widget for Google Chat
// messages.
type googleChatKeyValue struct {
	// TopLabel is the label shown above the value.
	TopLabel string `json:"topLabel,omitempty"`

	// Content is the text content of the key-value pair.
	Content string `json:"content"`

	// ContentMultiline indicates whether the content spans multiple lines.
	ContentMultiline bool `json:"contentMultiline,omitempty"`
}

// GoogleChatProvider sends notifications to Google Chat using webhooks.
type GoogleChatProvider struct {
	// httpClient sends HTTP requests to the Google Chat API.
	httpClient *http.Client

	// webhookURL is the Google Chat webhook URL for sending messages.
	webhookURL string
}

// Send delivers a notification to Google Chat.
//
// Takes params (*notification_dto.SendParams) which contains the notification
// content and recipient details.
//
// Returns error when the payload cannot be formatted, the request fails, or
// Google Chat returns a non-success status code.
func (g *GoogleChatProvider) Send(ctx context.Context, params *notification_dto.SendParams) error {
	payload, err := g.formatGoogleChatPayload(params)
	if err != nil {
		return fmt.Errorf("formatting googlechat payload: %w", err)
	}

	return sendHTTPJSONPayload(ctx, g.httpClient, g.webhookURL, payload, "googlechat")
}

// SendBulk sends multiple notifications.
//
// Takes notifications ([]*notification_dto.SendParams) which contains the
// notifications to send.
//
// Returns error when any notification fails to send.
func (g *GoogleChatProvider) SendBulk(ctx context.Context, notifications []*notification_dto.SendParams) error {
	for _, params := range notifications {
		if err := g.Send(ctx, params); err != nil {
			return fmt.Errorf("sending Google Chat notification in bulk: %w", err)
		}
	}
	return nil
}

// SupportsBulkSending reports whether Google Chat supports bulk sending.
//
// Returns bool which is always false as Google Chat does not support bulk
// sending.
func (*GoogleChatProvider) SupportsBulkSending() bool {
	return false
}

// GetCapabilities returns the capabilities of the Google Chat provider.
//
// Returns notification_domain.ProviderCapabilities which describes the
// supported features and limits for this provider.
func (*GoogleChatProvider) GetCapabilities() notification_domain.ProviderCapabilities {
	return notification_domain.ProviderCapabilities{
		SupportsRichFormatting: true,
		SupportsImages:         true,
		SupportsAttachments:    false,
		MaxMessageLength:       googleChatMaxMessageLength,
		SupportsBulkSending:    false,
		RequiresAuthentication: false,
	}
}

// Close releases any resources held by the provider.
//
// Returns error when the provider cannot be closed cleanly.
func (*GoogleChatProvider) Close(_ context.Context) error {
	return nil
}

// formatGoogleChatPayload converts notification params to Google Chat Card v2
// format.
//
// Takes params (*notification_dto.SendParams) which contains the notification
// content and context to format.
//
// Returns []byte which is the JSON-encoded Google Chat card payload.
// Returns error when JSON marshalling fails.
func (*GoogleChatProvider) formatGoogleChatPayload(params *notification_dto.SendParams) ([]byte, error) {
	widgets := []googleChatWidget{}

	if params.Content.Message != "" {
		widgets = append(widgets, googleChatWidget{
			TextParagraph: &googleChatTextParagraph{
				Text: params.Content.Message,
			},
		})
	}

	for key, value := range params.Content.Fields {
		widgets = append(widgets, googleChatWidget{
			KeyValue: &googleChatKeyValue{
				TopLabel: key,
				Content:  value,
			},
		})
	}

	if params.Context.Source != "" {
		widgets = append(widgets, googleChatWidget{
			KeyValue: &googleChatKeyValue{
				TopLabel: "Source",
				Content:  params.Context.Source,
			},
		})
	}
	if params.Context.Environment != "" {
		widgets = append(widgets, googleChatWidget{
			KeyValue: &googleChatKeyValue{
				TopLabel: "Environment",
				Content:  params.Context.Environment,
			},
		})
	}

	header := &googleChatHeader{
		Title:    params.Content.Title,
		Subtitle: params.Context.Priority.String() + " Priority",
	}
	if params.Content.ImageURL != "" {
		header.ImageURL = params.Content.ImageURL
	}

	card := googleChatCard{
		CardID: "notification-card",
		Card: googleChatCardBody{
			Header: header,
			Sections: []*googleChatSection{
				{Widgets: widgets},
			},
		},
	}

	payload := googleChatPayload{
		CardsV2: []googleChatCard{card},
	}

	return json.Marshal(payload)
}

// NewGoogleChatProvider creates a new Google Chat notification provider.
//
// Takes webhookURL (string) which specifies the Google Chat webhook endpoint.
// Takes client (*http.Client) which provides the HTTP client. If nil, a default
// client with a 30-second timeout is used.
//
// Returns notification_domain.NotificationProviderPort which is ready to send
// notifications.
func NewGoogleChatProvider(webhookURL string, client *http.Client) notification_domain.NotificationProviderPort {
	if client == nil {
		client = defaultHTTPClient
	}
	return &GoogleChatProvider{
		webhookURL: webhookURL,
		httpClient: client,
	}
}
