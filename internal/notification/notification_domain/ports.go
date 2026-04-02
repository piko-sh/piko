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
	"time"

	"piko.sh/piko/internal/deadletter/deadletter_domain"
	"piko.sh/piko/internal/notification/notification_dto"
	"piko.sh/piko/internal/retry"
)

// Service is the driving port: the hexagon's public API for sending
// notifications and managing providers. It implements io.Closer.
type Service interface {
	// NewNotification creates a new notification builder for composing and sending
	// notifications.
	//
	// Returns *NotificationBuilder which provides a fluent interface for building
	// notifications.
	NewNotification() *NotificationBuilder

	// SendBulk sends multiple notifications in a single batch operation.
	//
	// Takes notifications ([]*notification_dto.SendParams) which contains the
	// notification details to send.
	//
	// Returns error when any notification fails to send.
	SendBulk(ctx context.Context, notifications []*notification_dto.SendParams) error

	// SendBulkWithProvider sends multiple notifications using the specified
	// provider.
	//
	// Takes providerName (string) which identifies the notification provider.
	// Takes notifications ([]*notification_dto.SendParams) which contains the
	// notifications to send.
	//
	// Returns error when sending fails or the provider is not found.
	SendBulkWithProvider(ctx context.Context, providerName string, notifications []*notification_dto.SendParams) error

	// SendToProviders sends a single notification to multiple providers at
	// once (multi-cast).
	//
	// Takes params (*notification_dto.SendParams) which contains the
	// notification details.
	// Takes providers ([]string) which specifies the provider names to send to.
	//
	// Returns error when sending fails. Partial failures are reported
	// via MultiError.
	SendToProviders(ctx context.Context, params *notification_dto.SendParams, providers []string) error

	// RegisterProvider adds a notification provider to the registry.
	//
	// Takes name (string) which identifies the provider.
	// Takes provider (NotificationProviderPort) which handles notification
	// delivery.
	//
	// Returns error when registration fails.
	RegisterProvider(name string, provider NotificationProviderPort) error

	// SetDefaultProvider sets the default provider to use by name.
	//
	// Takes name (string) which identifies the provider to set as default.
	//
	// Returns error when the named provider does not exist.
	SetDefaultProvider(name string) error

	// GetProviders returns the names of all registered providers.
	//
	// Returns []string which contains the provider names.
	GetProviders() []string

	// HasProvider reports whether a provider with the given name exists.
	//
	// Takes name (string) which identifies the provider to check.
	//
	// Returns bool which is true if the provider exists, false otherwise.
	HasProvider(name string) bool

	// RegisterDispatcher sets the notification dispatcher for sending
	// notifications.
	//
	// Takes dispatcher (NotificationDispatcherPort) which handles notification
	// delivery.
	//
	// Returns error when the dispatcher cannot be registered.
	RegisterDispatcher(dispatcher NotificationDispatcherPort) error

	// FlushDispatcher flushes any pending dispatches to their destinations.
	//
	// Returns error when the flush operation fails.
	FlushDispatcher(ctx context.Context) error

	// Close releases any resources held by the service.
	//
	// Returns error when the service cannot be closed cleanly.
	Close(ctx context.Context) error
}

// NotificationProviderPort defines the interface that notification provider
// adapters must implement. It is a driven port in the hexagonal architecture
// pattern, allowing the domain to send notifications through different
// providers such as Discord, Slack, Teams, and others.
type NotificationProviderPort interface {
	// Send delivers a notification using the provided parameters.
	//
	// Takes params (*notification_dto.SendParams) which specifies the notification
	// details.
	//
	// Returns error when sending fails.
	Send(ctx context.Context, params *notification_dto.SendParams) error

	// SendBulk sends multiple notifications in a single operation.
	//
	// Takes notifications ([]*notification_dto.SendParams) which contains the
	// notification parameters
	// for each message to send.
	//
	// Returns error when the bulk send operation fails.
	SendBulk(ctx context.Context, notifications []*notification_dto.SendParams) error

	// SupportsBulkSending reports whether the provider supports bulk sending.
	//
	// Returns bool which is true if bulk sending is supported.
	SupportsBulkSending() bool

	// GetCapabilities returns the capabilities of this provider.
	//
	// Returns ProviderCapabilities which describes what features this provider
	// supports.
	GetCapabilities() ProviderCapabilities

	// Close releases any resources held by the provider.
	//
	// Returns error when the provider cannot be closed cleanly.
	Close(ctx context.Context) error
}

// ProviderCapabilities describes the features and limits of a notification
// provider.
type ProviderCapabilities struct {
	// MaxMessageLength is the maximum message length allowed; 0 means unlimited.
	MaxMessageLength int

	// SupportsRichFormatting indicates whether the provider supports markdown,
	// HTML, or rich blocks/embeds.
	SupportsRichFormatting bool

	// SupportsImages indicates whether the provider can display inline images.
	SupportsImages bool

	// SupportsAttachments indicates whether the provider can handle file
	// attachments.
	SupportsAttachments bool

	// SupportsBulkSending indicates whether the provider has a bulk send API.
	SupportsBulkSending bool

	// RequiresAuthentication indicates whether the provider needs authentication.
	RequiresAuthentication bool
}

// NotificationDispatcherPort defines the interface for notification
// dispatching services. It provides batched sending, retry handling,
// dead letter queue management, and lifecycle control.
type NotificationDispatcherPort interface {
	// Queue adds a notification to the batch queue for later sending.
	//
	// Takes params (*notification_dto.SendParams) which specifies the notification
	// details.
	//
	// Returns error when the notification cannot be queued.
	Queue(ctx context.Context, params *notification_dto.SendParams) error

	// Flush sends all queued notifications at once.
	//
	// Returns error when the flush operation fails.
	Flush(ctx context.Context) error

	// SetBatchSize configures the batch size for processing operations.
	//
	// Takes size (int) which specifies the number of items per batch.
	SetBatchSize(size int)

	// SetFlushInterval sets the interval between automatic buffer flushes.
	//
	// Takes interval (time.Duration) which specifies how often to flush.
	SetFlushInterval(interval time.Duration)

	// SetRetryConfig sets the retry configuration for operations.
	//
	// Takes config (RetryConfig) which specifies the retry behaviour.
	SetRetryConfig(config RetryConfig)

	// GetRetryConfig returns the retry configuration for this operation.
	//
	// Returns RetryConfig which specifies the retry behaviour.
	GetRetryConfig() RetryConfig

	// GetDeadLetterQueue returns the dead letter queue for storing failed events.
	//
	// Returns DeadLetterPort which provides access to the dead letter queue.
	GetDeadLetterQueue() DeadLetterPort

	// GetDeadLetterCount returns the number of messages in the dead letter queue.
	//
	// Returns int which is the count of dead letter messages.
	// Returns error when the count cannot be retrieved.
	GetDeadLetterCount(ctx context.Context) (int, error)

	// ClearDeadLetterQueue removes all messages from the dead letter queue.
	//
	// Returns error when the queue cannot be cleared.
	ClearDeadLetterQueue(ctx context.Context) error

	// GetRetryQueueSize returns the number of items waiting in the retry queue.
	//
	// Returns int which is the current retry queue depth.
	// Returns error when the queue size cannot be determined.
	GetRetryQueueSize(ctx context.Context) (int, error)

	// GetProcessingStats retrieves the current processing statistics.
	//
	// Returns DispatcherStats which contains the current processing metrics.
	// Returns error when the statistics cannot be retrieved.
	GetProcessingStats(ctx context.Context) (DispatcherStats, error)

	// Start begins the dispatcher and processes queued items.
	//
	// Returns error when the dispatcher fails to start.
	Start(ctx context.Context) error

	// Stop halts the service gracefully.
	//
	// Returns error when the shutdown fails or the context is cancelled.
	Stop(ctx context.Context) error
}

// DeadLetterPort is a queue port for storing failed notification entries.
type DeadLetterPort = deadletter_domain.DeadLetterPort[*notification_dto.DeadLetterEntry]

// DispatcherStats holds counts and timing data for the notification dispatcher.
type DispatcherStats struct {
	// QueuedNotifications is the number of notifications waiting to be sent.
	QueuedNotifications int `json:"queued_notifications"`

	// RetryQueueSize is the number of notifications waiting to be retried.
	RetryQueueSize int `json:"retry_queue_size"`

	// DeadLetterCount is the number of notifications in the dead letter queue.
	DeadLetterCount int `json:"dead_letter_count"`

	// TotalProcessed is the number of notifications handled since start.
	TotalProcessed int64 `json:"total_processed"`

	// TotalSuccessful is the total number of notifications sent without error.
	TotalSuccessful int64 `json:"total_successful"`

	// TotalFailed is the number of notifications that failed and will not be
	// retried.
	TotalFailed int64 `json:"total_failed"`

	// TotalRetries is the total number of retry attempts made.
	TotalRetries int64 `json:"total_retries"`

	// Uptime is how long the dispatcher has been running since it started.
	Uptime time.Duration `json:"uptime"`
}

// RetryConfig is an alias for the shared retry configuration.
type RetryConfig = retry.Config
