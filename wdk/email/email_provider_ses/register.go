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

package email_provider_ses

import (
	"context"
	"fmt"

	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/wdk/logger"
)

// Register creates and registers the SES email provider with the email service.
//
// Takes emailService (email_domain.Service) which handles email operations.
// Takes arguments (SESProviderArgs) which specifies the provider configuration.
// Takes opts (...func(...)) which applies optional configuration changes.
//
// Returns error when the email provider fails to register.
func Register(
	ctx context.Context,
	emailService email_domain.Service,
	arguments SESProviderArgs,
	opts ...func(*SESProviderArgs),
) error {
	_, l := logger.From(ctx, log)
	for _, opt := range opts {
		opt(&arguments)
	}

	if err := registerEmailProvider(ctx, emailService, arguments); err != nil {
		return err
	}

	l.Internal("AWS SES provider registered successfully")
	return nil
}

// registerEmailProvider creates and registers the SES email provider.
//
// Takes emailService (email_domain.Service) which handles provider
// registration.
// Takes arguments (SESProviderArgs) which holds the SES settings.
//
// Returns error when the provider cannot be created or registered.
func registerEmailProvider(ctx context.Context, emailService email_domain.Service, arguments SESProviderArgs) error {
	sesProvider, err := NewSESProvider(ctx, arguments)
	if err != nil {
		return fmt.Errorf("failed to create SES email provider: %w", err)
	}
	if err := emailService.RegisterProvider(ctx, "ses", sesProvider); err != nil {
		return fmt.Errorf("failed to register SES email provider: %w", err)
	}
	return nil
}
