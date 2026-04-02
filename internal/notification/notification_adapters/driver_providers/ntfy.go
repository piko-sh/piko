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

	"piko.sh/piko/internal/notification/notification_domain"
	"piko.sh/piko/internal/notification/notification_dto"
)

var _ notification_domain.NotificationProviderPort = (*NtfyProvider)(nil)

const (
	// ntfyMaxMessageLength is the maximum message length in bytes for ntfy.
	ntfyMaxMessageLength = 4096
)

// NtfyProvider sends notifications to an Ntfy server using HTTP requests.
// It implements NotificationProviderPort and io.Closer.
type NtfyProvider struct {
	// httpClient sends HTTP requests to the ntfy server.
	httpClient *http.Client

	// serverURL is the base URL of the ntfy server.
	serverURL string

	// topic is the ntfy topic name to publish messages to.
	topic string
}

// Send delivers a notification to Ntfy.
//
// Takes params (*notification_dto.SendParams) which contains the notification
// content and context.
//
// Returns error when the request cannot be created or sent, or when Ntfy
// returns a non-success status code.
func (n *NtfyProvider) Send(ctx context.Context, params *notification_dto.SendParams) error {
	message := n.formatNtfyMessage(params)

	url := fmt.Sprintf("%s/%s", n.serverURL, n.topic)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer([]byte(message)))
	if err != nil {
		return fmt.Errorf("creating ntfy request: %w", err)
	}

	request.Header.Set("Title", params.Content.Title)
	request.Header.Set("Priority", n.priorityToNtfyPriority(params.Context.Priority))
	request.Header.Set("Tags", n.priorityToNtfyTags(params.Context.Priority))

	response, err := n.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("sending to ntfy: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < httpStatusOK || response.StatusCode >= httpStatusMultiStatus {
		return fmt.Errorf("ntfy returned status %d: %s", response.StatusCode, response.Status)
	}

	return nil
}

// SendBulk sends multiple notifications.
//
// Takes notifications ([]*notification_dto.SendParams) which specifies the
// notifications to send.
//
// Returns error when any notification fails to send.
func (n *NtfyProvider) SendBulk(ctx context.Context, notifications []*notification_dto.SendParams) error {
	for _, params := range notifications {
		if err := n.Send(ctx, params); err != nil {
			return fmt.Errorf("sending ntfy notification in bulk: %w", err)
		}
	}
	return nil
}

// SupportsBulkSending reports whether Ntfy supports bulk sending.
//
// Returns bool which is always false as Ntfy does not support bulk sending.
func (*NtfyProvider) SupportsBulkSending() bool {
	return false
}

// GetCapabilities returns the capabilities of the Ntfy provider.
//
// Returns notification_domain.ProviderCapabilities which describes the
// supported features and limits of this provider.
func (*NtfyProvider) GetCapabilities() notification_domain.ProviderCapabilities {
	return notification_domain.ProviderCapabilities{
		SupportsRichFormatting: false,
		SupportsImages:         false,
		SupportsAttachments:    false,
		MaxMessageLength:       ntfyMaxMessageLength,
		SupportsBulkSending:    false,
		RequiresAuthentication: false,
	}
}

// Close releases any resources held by the provider.
//
// Returns error when the provider cannot be closed cleanly.
func (*NtfyProvider) Close(_ context.Context) error {
	return nil
}

// formatNtfyMessage formats the notification as plain text for Ntfy.
//
// Takes params (*notification_dto.SendParams) which contains the message
// content, fields, and context to format.
//
// Returns string which is the formatted plain text message.
func (*NtfyProvider) formatNtfyMessage(params *notification_dto.SendParams) string {
	var message strings.Builder

	message.WriteString(params.Content.Message)

	if len(params.Content.Fields) > 0 {
		message.WriteString("\n\n")
		for key, value := range params.Content.Fields {
			_, _ = fmt.Fprintf(&message, "%s: %s\n", key, value)
		}
	}

	if params.Context.Source != "" || params.Context.Environment != "" {
		message.WriteString("\n---\n")
		if params.Context.Source != "" {
			_, _ = fmt.Fprintf(&message, "Source: %s\n", params.Context.Source)
		}
		if params.Context.Environment != "" {
			_, _ = fmt.Fprintf(&message, "Environment: %s\n", params.Context.Environment)
		}
	}

	return message.String()
}

// priorityToNtfyPriority maps notification priority to Ntfy priority level.
//
// Takes priority (NotificationPriority) which is the notification priority to
// convert.
//
// Returns string which is the Ntfy priority level.
func (*NtfyProvider) priorityToNtfyPriority(priority notification_dto.NotificationPriority) string {
	switch priority {
	case notification_dto.PriorityCritical:
		return "urgent"
	case notification_dto.PriorityHigh:
		return "high"
	case notification_dto.PriorityNormal:
		return "default"
	default:
		return "low"
	}
}

// priorityToNtfyTags returns appropriate tags for the notification priority.
//
// Takes priority (NotificationPriority) which specifies the urgency level.
//
// Returns string which contains comma-separated ntfy tags for the priority.
func (*NtfyProvider) priorityToNtfyTags(priority notification_dto.NotificationPriority) string {
	switch priority {
	case notification_dto.PriorityCritical:
		return "rotating_light,fire,sos"
	case notification_dto.PriorityHigh:
		return "warning,exclamation"
	default:
		return "bell"
	}
}

// NewNtfyProvider creates a new Ntfy notification provider.
//
// Takes serverURL (string) which specifies the Ntfy server URL.
// Takes topic (string) which specifies the topic to publish to.
// Takes client (*http.Client) which provides the HTTP client. If nil, a default
// client with a 30-second timeout is used.
//
// Returns notification_domain.NotificationProviderPort which is ready to send
// notifications.
func NewNtfyProvider(serverURL, topic string, client *http.Client) notification_domain.NotificationProviderPort {
	if client == nil {
		client = defaultHTTPClient
	}
	return &NtfyProvider{
		serverURL:  strings.TrimSuffix(serverURL, "/"),
		topic:      topic,
		httpClient: client,
	}
}
