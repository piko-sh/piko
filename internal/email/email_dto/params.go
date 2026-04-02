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

package email_dto

import (
	"time"

	"piko.sh/piko/wdk/clock"
)

// Attachment represents an email attachment with its content and metadata.
type Attachment struct {
	// Filename is the name of the attached file.
	Filename string

	// MIMEType is the content type of the attachment (e.g. "image/png").
	MIMEType string

	// ContentID is an optional identifier for inline or embedded images.
	// For example, "logo" allows HTML to reference the image via cid:logo.
	ContentID string

	// Content holds the raw binary data of the attachment.
	Content []byte
}

// SendParams holds the data needed to send an email, including recipients,
// subject, body content, and attachments.
type SendParams struct {
	// From is the sender email address; nil uses the default sender.
	From *string

	// ProviderOptions holds provider-specific settings passed through to the adapter.
	ProviderOptions map[string]any

	// Subject is the email subject line.
	Subject string

	// BodyHTML is the HTML content of the email body.
	BodyHTML string

	// BodyPlain is the plain text version of the email body.
	BodyPlain string

	// To contains the recipient email addresses.
	To []string

	// Cc holds email addresses that receive a copy of the message.
	Cc []string

	// Bcc holds the blind carbon copy recipient email addresses.
	Bcc []string

	// Attachments is the list of files to attach to the email.
	Attachments []Attachment
}

// DispatcherConfig holds configuration for the email dispatcher, controlling
// batching, retries, and circuit breaker behaviour.
type DispatcherConfig struct {
	// Clock provides time operations for testing determinism; nil uses RealClock().
	Clock clock.Clock

	// JitterFunc adds randomness to retry delays by receiving the calculated
	// delay and returning additional jitter. If nil, uses default flat jitter
	// (0-1000ms); set to return 0 in tests for deterministic timing.
	JitterFunc func(time.Duration) time.Duration

	// BatchSize specifies how many emails to group together for batch processing.
	// Defaults to a preset value if zero or negative.
	BatchSize int

	// FlushInterval specifies how often to flush pending emails. Zero or negative
	// values use the default interval.
	FlushInterval time.Duration

	// QueueSize is the maximum number of pending emails in the queue.
	QueueSize int

	// RetryQueueSize is the buffer size for failed emails waiting to be retried.
	RetryQueueSize int

	// MaxRetries is the maximum number of retry attempts per email.
	MaxRetries int

	// RetryWorkerCount is the number of workers that process retries at the same
	// time. If zero or negative, defaults to the number of CPU cores.
	RetryWorkerCount int

	// InitialDelay is the time to wait before the first retry attempt.
	InitialDelay time.Duration

	// MaxDelay is the longest delay allowed between retries. Zero uses the default.
	MaxDelay time.Duration

	// BackoffFactor is the multiplier for retry delays; values <= 0 use the default.
	BackoffFactor float64

	// DeadLetterQueue enables storing emails that fail permanently for later review.
	DeadLetterQueue bool

	// MaxRetryHeapSize is the maximum number of emails in the retry heap.
	// Set to 0 for unlimited; a positive value protects against memory exhaustion.
	MaxRetryHeapSize int

	// MaxConsecutiveFailures is the number of consecutive failures before the
	// circuit breaker opens. Set to 0 to disable the circuit breaker.
	MaxConsecutiveFailures int

	// CircuitBreakerTimeout is how long the circuit breaker stays open before
	// trying to recover. Zero or negative uses the default timeout.
	CircuitBreakerTimeout time.Duration

	// CircuitBreakerInterval is the time period after which the failure count
	// resets when the circuit is closed. A value of 0 or less uses the default.
	CircuitBreakerInterval time.Duration
}

// DeadLetterEntry represents an email that has failed all retry attempts.
type DeadLetterEntry struct {
	// FirstAttempt is when processing was first tried.
	FirstAttempt time.Time

	// LastAttempt is when this entry was last processed.
	LastAttempt time.Time

	// AddedToDeadLetter is when this entry was added to the dead letter queue.
	AddedToDeadLetter time.Time

	// ID is the unique identifier for this dead letter entry.
	ID string

	// OriginalError is the error message from the failed delivery attempt.
	OriginalError string

	// Email holds the original email parameters that failed to send.
	Email SendParams

	// TotalAttempts is the number of delivery attempts made for this email.
	TotalAttempts int
}

const (
	// defaultBatchSize is the number of items handled in each batch.
	defaultBatchSize = 10

	// defaultFlushIntervalSeconds is the default number of seconds between flushes.
	defaultFlushIntervalSeconds = 30

	// defaultQueueSize is the default capacity of the queue in items.
	defaultQueueSize = 1000

	// defaultRetryQueueSize is the default size of the retry queue.
	defaultRetryQueueSize = 500

	// defaultMaxRetries is the number of times to retry a failed operation.
	defaultMaxRetries = 3

	// defaultInitialDelaySeconds is the starting delay before the first retry.
	defaultInitialDelaySeconds = 5

	// defaultMaxDelayMinutes is the upper limit for the delay between retry attempts.
	defaultMaxDelayMinutes = 5

	// defaultBackoffFactor is the multiplier for delays between retries.
	defaultBackoffFactor = 2.0

	// defaultMaxRetryHeapSize is the maximum number of emails in the retry heap.
	defaultMaxRetryHeapSize = 50000

	// defaultMaxConsecutiveFailures is the failure count before the circuit opens.
	defaultMaxConsecutiveFailures = 5

	// defaultCircuitBreakerTimeout is the seconds to wait before recovery attempts.
	defaultCircuitBreakerTimeout = 60

	// defaultCircuitBreakerInterval is the seconds between failure counter resets.
	defaultCircuitBreakerInterval = 10
)

// DefaultDispatcherConfig returns a sensible default dispatcher configuration.
//
// Returns DispatcherConfig which contains sensible defaults for batch
// processing, retry behaviour, and circuit breaker settings.
func DefaultDispatcherConfig() DispatcherConfig {
	return DispatcherConfig{
		BatchSize:              defaultBatchSize,
		FlushInterval:          defaultFlushIntervalSeconds * time.Second,
		QueueSize:              defaultQueueSize,
		RetryQueueSize:         defaultRetryQueueSize,
		MaxRetries:             defaultMaxRetries,
		InitialDelay:           defaultInitialDelaySeconds * time.Second,
		MaxDelay:               defaultMaxDelayMinutes * time.Minute,
		BackoffFactor:          defaultBackoffFactor,
		DeadLetterQueue:        true,
		MaxRetryHeapSize:       defaultMaxRetryHeapSize,
		MaxConsecutiveFailures: defaultMaxConsecutiveFailures,
		CircuitBreakerTimeout:  defaultCircuitBreakerTimeout * time.Second,
		CircuitBreakerInterval: defaultCircuitBreakerInterval * time.Second,
	}
}
