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

package email_domain

import (
	"context"
	"time"

	"piko.sh/piko/internal/deadletter/deadletter_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/provider/provider_domain"
)

// Service is the driving port: the hexagon's public API for sending emails
// and managing providers. It implements email_domain.Service and
// wdk/email.Service interfaces.
type Service interface {
	// NewEmail creates a new email builder for writing and sending emails.
	//
	// Returns *EmailBuilder which provides a fluent interface for building emails.
	NewEmail() *EmailBuilder

	// SendBulk sends multiple emails in a single batch operation.
	//
	// Takes emails ([]*email_dto.SendParams) which contains the email details to send.
	//
	// Returns error when any email fails to send.
	SendBulk(ctx context.Context, emails []*email_dto.SendParams) error

	// SendBulkWithProvider sends multiple emails using the specified provider.
	//
	// Takes providerName (string) which identifies which email provider to use.
	// Takes emails ([]*email_dto.SendParams) which contains the emails to send.
	//
	// Returns error when sending fails or the provider is not found.
	SendBulkWithProvider(ctx context.Context, providerName string, emails []*email_dto.SendParams) error

	// RegisterProvider adds an email provider to the registry.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	// Takes name (string) which identifies the provider.
	// Takes provider (EmailProviderPort) which handles email delivery.
	//
	// Returns error when registration fails.
	RegisterProvider(ctx context.Context, name string, provider EmailProviderPort) error

	// SetDefaultProvider sets the default provider to use by name.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	// Takes name (string) which identifies the provider to set as default.
	//
	// Returns error when the named provider does not exist.
	SetDefaultProvider(ctx context.Context, name string) error

	// GetProviders returns the names of all registered providers.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	//
	// Returns []string which contains the provider names.
	GetProviders(ctx context.Context) []string

	// HasProvider reports whether a provider with the given name exists.
	//
	// Takes name (string) which identifies the provider to check.
	//
	// Returns bool which is true if the provider exists, false otherwise.
	HasProvider(name string) bool

	// ListProviders returns detailed information about all registered providers.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	//
	// Returns []provider_domain.ProviderInfo which contains provider metadata,
	// health status, and capabilities.
	ListProviders(ctx context.Context) []provider_domain.ProviderInfo

	// RegisterDispatcher sets the email dispatcher for sending notifications.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	// Takes dispatcher (EmailDispatcherPort) which handles email delivery.
	//
	// Returns error when the dispatcher cannot be registered.
	RegisterDispatcher(ctx context.Context, dispatcher EmailDispatcherPort) error

	// FlushDispatcher flushes any pending dispatches to their destinations.
	//
	// Returns error when the flush operation fails.
	FlushDispatcher(ctx context.Context) error
}

// EmailProviderPort defines the interface that email provider adapters must
// implement. It is a driven port in the hexagonal architecture pattern,
// allowing the domain to send emails through different providers.
type EmailProviderPort interface {
	// Send delivers an email using the provided parameters.
	//
	// Takes params (*email_dto.SendParams) which specifies the email details.
	//
	// Returns error when sending fails.
	Send(ctx context.Context, params *email_dto.SendParams) error

	// SendBulk sends multiple emails in a single operation.
	//
	// Takes emails ([]*email_dto.SendParams) which contains the email parameters
	// for each message to send.
	//
	// Returns error when the bulk send operation fails.
	SendBulk(ctx context.Context, emails []*email_dto.SendParams) error

	// SupportsBulkSending reports whether the sender supports bulk sending.
	//
	// Returns bool which is true if bulk sending is supported.
	SupportsBulkSending() bool

	// Close releases any resources held by the source.
	//
	// Returns error when the source cannot be closed cleanly.
	Close(ctx context.Context) error
}

// EmailDispatcherPort defines the interface for email dispatching services.
// It provides batched sending, retry handling, dead letter queue management,
// and lifecycle control for email delivery.
type EmailDispatcherPort interface {
	// Queue adds an email to the batch queue for later sending.
	//
	// Takes params (*email_dto.SendParams) which specifies the email details.
	//
	// Returns error when the email cannot be queued.
	Queue(ctx context.Context, params *email_dto.SendParams) error

	// Flush sends all queued emails at once.
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

// DeadLetterPort is the dead letter queue port for email entries.
type DeadLetterPort = deadletter_domain.DeadLetterPort[*email_dto.DeadLetterEntry]

// DispatcherStats holds counts and timing data for the email dispatcher.
type DispatcherStats struct {
	// QueuedEmails is the number of emails waiting to be sent.
	QueuedEmails int

	// RetryQueueSize is the number of emails waiting to be retried.
	RetryQueueSize int

	// DeadLetterCount is the number of emails in the dead letter queue.
	DeadLetterCount int

	// TotalProcessed is the number of emails processed since start.
	TotalProcessed int64

	// TotalSuccessful is the total number of emails sent successfully.
	TotalSuccessful int64

	// TotalFailed is the count of emails that failed permanently.
	TotalFailed int64

	// TotalRetries is the total number of times tasks were retried.
	TotalRetries int64

	// Uptime is how long the dispatcher has been running.
	Uptime time.Duration
}

// AssetResolverPort is the driven port for resolving email assets from the
// registry. It implements email_domain.AssetResolverPort and bridges the email
// domain with the asset processing system, enabling automatic CID embedding of
// local images in email templates.
//
// The resolver fetches assets from the registry using the source path, applies
// the requested transformation profile (e.g. "email-default", "email-outlook"),
// and returns ready-to-embed attachments with ContentID set. It handles missing
// assets and transformation errors gracefully.
type AssetResolverPort interface {
	// ResolveAsset fetches and transforms a single asset based on the request.
	//
	// Takes request (*email_dto.EmailAssetRequest) which specifies the asset to
	// resolve.
	//
	// Returns *email_dto.Attachment which is the ready-to-embed attachment with
	// ContentID, MIME type, and content populated.
	// Returns error when the asset cannot be found or transformation fails.
	ResolveAsset(ctx context.Context, request *email_dto.EmailAssetRequest) (*email_dto.Attachment, error)

	// ResolveAssets resolves multiple assets in batch for efficiency.
	//
	// Takes requests ([]*email_dto.EmailAssetRequest) which contains the assets
	// to resolve.
	//
	// Returns []*email_dto.Attachment which contains the successfully resolved
	// attachments.
	// Returns []error which contains one error per request, where non-nil errors
	// indicate which assets failed to resolve, supporting partial success.
	ResolveAssets(ctx context.Context, requests []*email_dto.EmailAssetRequest) ([]*email_dto.Attachment, []error)
}
