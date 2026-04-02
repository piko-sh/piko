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

package notification_test

import (
	"context"
	"testing"
	"time"

	"piko.sh/piko/internal/notification/notification_adapters/driver_providers"
	"piko.sh/piko/internal/notification/notification_dto"
)

func TestAllProvidersImplementInterface(t *testing.T) {
	tests := []struct {
		provider any
		name     string
	}{
		{name: "Slack", provider: driver_providers.NewSlackProvider("https://hooks.slack.com/test", nil)},
		{name: "Discord", provider: driver_providers.NewDiscordProvider("https://discord.com/api/webhooks/test", nil)},
		{name: "PagerDuty", provider: driver_providers.NewPagerDutyProvider("test-key", nil)},
		{name: "Teams", provider: driver_providers.NewTeamsProvider("https://teams.webhook.office.com/test", nil)},
		{name: "GoogleChat", provider: driver_providers.NewGoogleChatProvider("https://chat.googleapis.com/test", nil)},
		{name: "Ntfy", provider: driver_providers.NewNtfyProvider("https://ntfy.sh", "test-topic", nil)},
		{name: "Webhook", provider: driver_providers.NewWebhookProvider("https://example.com/webhook", nil, nil)},
		{name: "Stdout", provider: driver_providers.NewStdoutProvider()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.provider == nil {
				t.Fatal("Provider is nil")
			}
		})
	}
}

func TestProviderCapabilities(t *testing.T) {

	slackProvider := driver_providers.NewSlackProvider("test", nil)
	slackCaps := slackProvider.GetCapabilities()
	if !slackCaps.SupportsRichFormatting {
		t.Error("Slack should support rich formatting")
	}

	discordProvider := driver_providers.NewDiscordProvider("test", nil)
	discordCaps := discordProvider.GetCapabilities()
	if !discordCaps.SupportsRichFormatting {
		t.Error("Discord should support rich formatting")
	}

	stdoutProvider := driver_providers.NewStdoutProvider()
	stdoutCaps := stdoutProvider.GetCapabilities()
	if stdoutCaps.SupportsRichFormatting {
		t.Error("Stdout should not support rich formatting")
	}
	if !stdoutCaps.SupportsBulkSending {
		t.Error("Stdout should support bulk sending")
	}
}

func TestProviderClose(t *testing.T) {
	providers := []struct {
		provider interface{ Close(context.Context) error }
		name     string
	}{
		{name: "Slack", provider: driver_providers.NewSlackProvider("test", nil)},
		{name: "Discord", provider: driver_providers.NewDiscordProvider("test", nil)},
		{name: "PagerDuty", provider: driver_providers.NewPagerDutyProvider("test", nil)},
		{name: "Teams", provider: driver_providers.NewTeamsProvider("test", nil)},
		{name: "GoogleChat", provider: driver_providers.NewGoogleChatProvider("test", nil)},
		{name: "Ntfy", provider: driver_providers.NewNtfyProvider("https://ntfy.sh", "test", nil)},
		{name: "Webhook", provider: driver_providers.NewWebhookProvider("https://test.com", nil, nil)},
		{name: "Stdout", provider: driver_providers.NewStdoutProvider()},
	}

	ctx := context.Background()

	for _, tt := range providers {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.provider.Close(ctx); err != nil {
				t.Errorf("Close() returned error: %v", err)
			}
		})
	}
}

func TestStdoutProvider_Send(t *testing.T) {
	provider := driver_providers.NewStdoutProvider()
	ctx := context.Background()

	params := &notification_dto.SendParams{
		Context: notification_dto.NotificationContext{
			Source:      "test",
			Environment: "dev",
			Priority:    notification_dto.PriorityNormal,
			Timestamp:   time.Now(),
		},
		Content: notification_dto.NotificationContent{
			Type:    notification_dto.NotificationTypePlain,
			Title:   "Test Notification",
			Message: "This is a test",
			Fields: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}

	if err := provider.Send(ctx, params); err != nil {
		t.Fatalf("Send() failed: %v", err)
	}

	t.Log("Stdout provider sent notification successfully")
}
