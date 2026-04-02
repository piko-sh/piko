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

package notification

import (
	"errors"
	"fmt"

	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/notification/notification_domain"
	"piko.sh/piko/internal/notification/notification_dto"
)

// Service manages notification providers and operations.
type Service = notification_domain.Service

// ProviderPort represents the interface that all notification providers
// must implement. Implement it to create custom or mock providers.
type ProviderPort = notification_domain.NotificationProviderPort

// DispatcherPort represents the interface for asynchronous notification
// dispatchers.
type DispatcherPort = notification_domain.NotificationDispatcherPort

// NotificationBuilder provides a fluent API for building and sending
// notifications.
type NotificationBuilder = notification_domain.NotificationBuilder

// SendParams holds all parameters needed to send a notification.
type SendParams = notification_dto.SendParams

// NotificationContent contains the core content of a notification.
type NotificationContent = notification_dto.NotificationContent

// NotificationPriority represents the urgency level of a notification.
type NotificationPriority = notification_dto.NotificationPriority

const (
	// PriorityLow is the lowest priority level for notifications.
	PriorityLow = notification_dto.PriorityLow

	// PriorityNormal is the default priority level for notifications.
	PriorityNormal = notification_dto.PriorityNormal

	// PriorityHigh is the highest priority level for notifications.
	PriorityHigh = notification_dto.PriorityHigh

	// PriorityCritical is the highest priority level for notifications.
	PriorityCritical = notification_dto.PriorityCritical

	// TypePlain is a plain text notification without special formatting.
	TypePlain = notification_dto.NotificationTypePlain

	// TypeRich is the notification type for rich content notifications.
	TypeRich = notification_dto.NotificationTypeRich

	// TypeTemplated is a templated notification type.
	TypeTemplated = notification_dto.NotificationTypeTemplated

	// ProviderSlack is the provider name for Slack notifications.
	ProviderSlack = notification_dto.NotificationNameSlack

	// ProviderDiscord is the provider name for Discord notifications.
	ProviderDiscord = notification_dto.NotificationNameDiscord

	// ProviderPagerDuty is the provider name for PagerDuty notifications.
	ProviderPagerDuty = notification_dto.NotificationNamePagerDuty

	// ProviderTeams is the provider name for Microsoft Teams notifications.
	ProviderTeams = notification_dto.NotificationNameTeams

	// ProviderGoogleChat is the provider name for Google Chat notifications.
	ProviderGoogleChat = notification_dto.NotificationNameGoogleChat

	// ProviderNtfy is the provider name for ntfy.sh notifications.
	ProviderNtfy = notification_dto.NotificationNameNtfy

	// ProviderWebhook is the provider name for generic webhook notifications.
	ProviderWebhook = notification_dto.NotificationNameWebhook

	// ProviderStdout is the provider name for stdout (development) notifications.
	ProviderStdout = notification_dto.NotificationNameStdout
)

// NotificationType represents the formatting and structure of notification
// content.
type NotificationType = notification_dto.NotificationType

// DispatcherConfig holds configuration for the notification dispatcher.
type DispatcherConfig = notification_dto.DispatcherConfig

// DeadLetterEntry represents a failed notification in the dead letter queue.
type DeadLetterEntry = notification_dto.DeadLetterEntry

// RetryConfig defines the retry behaviour for failed operations.
type RetryConfig = notification_domain.RetryConfig

// DispatcherStats holds counts and timing data about the notification
// dispatcher.
type DispatcherStats = notification_domain.DispatcherStats

// ProviderCapabilities describes the features and limits of a notification
// provider.
type ProviderCapabilities = notification_domain.ProviderCapabilities

// NewService creates a new notification service instance.
//
// Returns Service which is the configured notification service ready for use.
//
// Example:
//
//	service := notification.NewService()
//	provider, _ := notification_provider_slack.NewProvider(config)
//	service.RegisterProvider("slack", provider)
func NewService() Service {
	return notification_domain.NewService()
}

// GetDefaultService returns the notification service initialised by the
// framework.
//
// Returns Service which is the initialised notification service instance.
// Returns error when the framework has not been bootstrapped.
//
// Example:
//
//	service, err := notification.GetDefaultService()
//	if err != nil {
//	    return err
//	}
func GetDefaultService() (Service, error) {
	service, err := bootstrap.GetNotificationService()
	if err != nil {
		return nil, fmt.Errorf("notification: get default service: %w", err)
	}
	return service, nil
}

// NewNotificationBuilder creates a new notification builder for composing and
// sending notifications.
//
// Takes service (Service) which is the notification service to use for sending.
//
// Returns *NotificationBuilder which provides a fluent interface for building
// notifications.
// Returns error when service is nil.
//
// Example:
//
//	service := notification.NewService()
//	builder, err := notification.NewNotificationBuilder(service)
//	if err != nil {
//	    return err
//	}
//	err = builder.
//	    Title("Alert").
//	    Message("System event occurred").
//	    Do(ctx)
func NewNotificationBuilder(service Service) (*NotificationBuilder, error) {
	if service == nil {
		return nil, errors.New("notification: service must not be nil")
	}
	return service.NewNotification(), nil
}

// NewNotificationBuilderFromDefault creates a new notification builder using
// the framework's bootstrapped service.
//
// Returns *NotificationBuilder which is the builder ready for use.
// Returns error when the framework has not been bootstrapped.
//
// Example:
//
//	builder, err := notification.NewNotificationBuilderFromDefault()
//	if err != nil {
//	    return err
//	}
//	err = builder.
//	    Title("Alert").
//	    Message("System event occurred").
//	    Do(ctx)
func NewNotificationBuilderFromDefault() (*NotificationBuilder, error) {
	service, err := GetDefaultService()
	if err != nil {
		return nil, fmt.Errorf("notification: get default service: %w", err)
	}
	return NewNotificationBuilder(service)
}
