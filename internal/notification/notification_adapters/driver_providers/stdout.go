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
	"os"

	"piko.sh/piko/internal/notification/notification_domain"
	"piko.sh/piko/internal/notification/notification_dto"
)

var _ notification_domain.NotificationProviderPort = (*StdoutProvider)(nil)

// StdoutProvider writes notifications to standard output for development and
// testing. It implements the NotificationProviderPort interface.
type StdoutProvider struct{}

// Send delivers a notification to stdout.
//
// Takes params (*notification_dto.SendParams) which contains the notification
// content and context to display.
//
// Returns error when the notification cannot be written.
func (*StdoutProvider) Send(_ context.Context, params *notification_dto.SendParams) error {
	_, _ = fmt.Fprint(os.Stdout, "=== NOTIFICATION ===\n")
	_, _ = fmt.Fprintf(os.Stdout, "Priority: %s\n", params.Context.Priority.String())
	_, _ = fmt.Fprintf(os.Stdout, "Source: %s\n", params.Context.Source)
	_, _ = fmt.Fprintf(os.Stdout, "Environment: %s\n", params.Context.Environment)
	_, _ = fmt.Fprintf(os.Stdout, "Title: %s\n", params.Content.Title)
	_, _ = fmt.Fprintf(os.Stdout, "Message: %s\n", params.Content.Message)

	if len(params.Content.Fields) > 0 {
		_, _ = fmt.Fprint(os.Stdout, "Fields:\n")
		for key, value := range params.Content.Fields {
			_, _ = fmt.Fprintf(os.Stdout, "  %s: %s\n", key, value)
		}
	}

	_, _ = fmt.Fprint(os.Stdout, "===================\n")
	return nil
}

// SendBulk sends multiple notifications to stdout.
//
// Takes notifications ([]*notification_dto.SendParams) which contains the
// notifications to send.
//
// Returns error when any notification fails to send.
func (s *StdoutProvider) SendBulk(ctx context.Context, notifications []*notification_dto.SendParams) error {
	for _, params := range notifications {
		if err := s.Send(ctx, params); err != nil {
			return fmt.Errorf("sending stdout notification in bulk: %w", err)
		}
	}
	return nil
}

// SupportsBulkSending reports whether stdout supports bulk sending.
//
// Returns bool which is always true for this provider.
func (*StdoutProvider) SupportsBulkSending() bool {
	return true
}

// GetCapabilities returns the capabilities of the stdout provider.
//
// Returns notification_domain.ProviderCapabilities which describes the
// supported features including bulk sending but no rich formatting, images,
// or attachments.
func (*StdoutProvider) GetCapabilities() notification_domain.ProviderCapabilities {
	return notification_domain.ProviderCapabilities{
		SupportsRichFormatting: false,
		SupportsImages:         false,
		SupportsAttachments:    false,
		MaxMessageLength:       0,
		SupportsBulkSending:    true,
		RequiresAuthentication: false,
	}
}

// Close releases any resources held by the provider.
//
// Returns error when the provider cannot be closed; always returns nil.
func (*StdoutProvider) Close(_ context.Context) error {
	return nil
}

// NewStdoutProvider creates a new stdout notification provider.
//
// Returns notification_domain.NotificationProviderPort which writes to stdout.
func NewStdoutProvider() notification_domain.NotificationProviderPort {
	return &StdoutProvider{}
}
