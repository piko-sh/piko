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

package notification_domain

import (
	"context"
	"fmt"
	"maps"
	"time"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/notification/notification_dto"
)

// NotificationBuilder provides a fluent interface for building notifications.
type NotificationBuilder struct {
	// service is the notification service used to send messages.
	service *service

	// params holds the notification settings being built.
	params *notification_dto.SendParams
}

// Title sets the notification title.
//
// Takes title (string) which is the notification title or subject.
//
// Returns *NotificationBuilder for method chaining.
func (b *NotificationBuilder) Title(title string) *NotificationBuilder {
	b.params.Content.Title = title
	return b
}

// Message sets the notification message body.
//
// Takes message (string) which is the main text shown in the notification.
//
// Returns *NotificationBuilder for method chaining.
func (b *NotificationBuilder) Message(message string) *NotificationBuilder {
	b.params.Content.Message = message
	return b
}

// Field adds a key-value pair to the notification's structured fields.
//
// Takes key (string) which is the field name.
// Takes value (string) which is the field value.
//
// Returns *NotificationBuilder for method chaining.
func (b *NotificationBuilder) Field(key, value string) *NotificationBuilder {
	if b.params.Content.Fields == nil {
		b.params.Content.Fields = make(map[string]string)
	}
	b.params.Content.Fields[key] = value
	return b
}

// Fields sets multiple structured fields at once.
//
// Takes fields (map[string]string) which contains the key-value pairs.
//
// Returns *NotificationBuilder for method chaining.
func (b *NotificationBuilder) Fields(fields map[string]string) *NotificationBuilder {
	if b.params.Content.Fields == nil {
		b.params.Content.Fields = make(map[string]string)
	}
	maps.Copy(b.params.Content.Fields, fields)
	return b
}

// Image sets an inline image URL for the notification.
//
// Takes imageURL (string) which is the URL of the image to display.
//
// Returns *NotificationBuilder for method chaining.
func (b *NotificationBuilder) Image(imageURL string) *NotificationBuilder {
	b.params.Content.ImageURL = imageURL
	return b
}

// Priority sets the notification priority level.
//
// Takes priority (notification_dto.NotificationPriority) which is the urgency
// level.
//
// Returns *NotificationBuilder for method chaining.
func (b *NotificationBuilder) Priority(priority notification_dto.NotificationPriority) *NotificationBuilder {
	b.params.Context.Priority = priority
	return b
}

// Source sets the source identifier for the notification.
//
// Takes source (string) which identifies where the notification came from.
//
// Returns *NotificationBuilder for method chaining.
func (b *NotificationBuilder) Source(source string) *NotificationBuilder {
	b.params.Context.Source = source
	return b
}

// Environment sets the environment identifier for the notification.
//
// Takes environment (string) which identifies the environment (e.g. "dev",
// "prod").
//
// Returns *NotificationBuilder for method chaining.
func (b *NotificationBuilder) Environment(environment string) *NotificationBuilder {
	b.params.Context.Environment = environment
	return b
}

// Service sets the service identifier for the notification.
//
// Takes service (string) which identifies the service that generated the
// notification.
//
// Returns *NotificationBuilder for method chaining.
func (b *NotificationBuilder) Service(service string) *NotificationBuilder {
	b.params.Context.Service = service
	return b
}

// TraceID sets the OpenTelemetry trace ID for correlation.
//
// Takes traceID (string) which is the trace ID for correlation.
//
// Returns *NotificationBuilder for method chaining.
func (b *NotificationBuilder) TraceID(traceID string) *NotificationBuilder {
	b.params.Context.TraceID = traceID
	return b
}

// Type sets the notification content type.
//
// Takes notificationType (notification_dto.NotificationType) which specifies
// the formatting.
//
// Returns *NotificationBuilder for method chaining.
func (b *NotificationBuilder) Type(notificationType notification_dto.NotificationType) *NotificationBuilder {
	b.params.Content.Type = notificationType
	return b
}

// Provider sets a specific provider to use instead of the default.
//
// Takes provider (string) which is the name of the provider to use.
//
// Returns *NotificationBuilder for method chaining.
func (b *NotificationBuilder) Provider(provider string) *NotificationBuilder {
	b.params.Providers = []string{provider}
	return b
}

// ToProviders sets multiple providers for multi-cast sending.
//
// Takes providers (...string) which are the names of the providers to send to.
//
// Returns *NotificationBuilder for method chaining.
func (b *NotificationBuilder) ToProviders(providers ...string) *NotificationBuilder {
	b.params.Providers = providers
	return b
}

// ProviderOption sets a provider-specific option.
//
// Takes key (string) which is the option name.
// Takes value (any) which is the option value.
//
// Returns *NotificationBuilder for method chaining.
func (b *NotificationBuilder) ProviderOption(key string, value any) *NotificationBuilder {
	if b.params.ProviderOptions == nil {
		b.params.ProviderOptions = make(map[string]any)
	}
	b.params.ProviderOptions[key] = value
	return b
}

// ProviderOptions sets multiple provider-specific options at once.
//
// Takes options (map[string]any) which contains the key-value pairs.
//
// Returns *NotificationBuilder for method chaining.
func (b *NotificationBuilder) ProviderOptions(options map[string]any) *NotificationBuilder {
	if b.params.ProviderOptions == nil {
		b.params.ProviderOptions = make(map[string]any)
	}
	maps.Copy(b.params.ProviderOptions, options)
	return b
}

// Do sends the notification using the configured providers.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Returns error when validation fails or the notification cannot be sent.
func (b *NotificationBuilder) Do(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("sending notification: %w", err)
	}

	startTime := time.Now()
	builderSendCount.Add(ctx, 1)

	if b.params.Context.Timestamp.IsZero() {
		b.params.Context.Timestamp = time.Now()
	}

	if err := b.validate(); err != nil {
		builderSendErrorCount.Add(ctx, 1)
		return fmt.Errorf("validating notification: %w", err)
	}

	var err error
	if len(b.params.Providers) > 1 {
		err = b.service.SendToProviders(ctx, b.params, b.params.Providers)
	} else {
		providerName := ""
		if len(b.params.Providers) == 1 {
			providerName = b.params.Providers[0]
		}

		provider, getErr := b.service.getProvider(providerName)
		if getErr != nil {
			builderSendErrorCount.Add(ctx, 1)
			return fmt.Errorf("resolving notification provider: %w", getErr)
		}

		err = goroutine.SafeCall(ctx, "notification.Send", func() error { return provider.Send(ctx, b.params) })
	}

	duration := time.Since(startTime).Milliseconds()
	builderSendDuration.Record(ctx, float64(duration))

	if err != nil {
		builderSendErrorCount.Add(ctx, 1)
		return fmt.Errorf("sending notification: %w", err)
	}

	return nil
}

// validate checks that the notification has all required fields.
//
// Returns error when both title and message are empty.
func (b *NotificationBuilder) validate() error {
	if b.params.Content.Title == "" && b.params.Content.Message == "" {
		return ErrNotificationEmpty
	}

	return nil
}
