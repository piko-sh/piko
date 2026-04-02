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

package llm_dto

import "time"

const (
	// DefaultRetryMaxRetries is the default number of retry attempts.
	DefaultRetryMaxRetries = 3

	// DefaultRetryInitialBackoff is the default delay before the first retry
	// attempt.
	DefaultRetryInitialBackoff = 500 * time.Millisecond

	// DefaultRetryMaxBackoff is the default maximum delay between retry attempts.
	DefaultRetryMaxBackoff = 8 * time.Second

	// DefaultRetryBackoffMultiplier is the default factor by which backoff
	// increases after each retry.
	DefaultRetryBackoffMultiplier = 2.0

	// DefaultRetryJitterFraction is the default maximum fraction of backoff to add
	// as random jitter.
	DefaultRetryJitterFraction = 0.1
)

// RetryPolicy configures the retry behaviour for LLM completion requests. It
// supports exponential backoff with jitter to avoid thundering herd problems.
type RetryPolicy struct {
	// OnRetry is an optional callback invoked before each retry attempt. It
	// receives the attempt number (1-based), the error that triggered the retry,
	// and the backoff duration before the next attempt.
	OnRetry func(attempt int, err error, nextBackoff time.Duration)

	// MaxRetries is the maximum number of retry attempts, not including the
	// initial attempt.
	MaxRetries int

	// InitialBackoff is the delay before the first retry attempt.
	InitialBackoff time.Duration

	// MaxBackoff is the maximum delay between retry attempts.
	MaxBackoff time.Duration

	// BackoffMultiplier is the value used to increase the delay after each retry.
	BackoffMultiplier float64

	// JitterFraction is the maximum fraction of the backoff to add as random
	// jitter. For example, 0.1 means up to 10% of the backoff is added as jitter.
	JitterFraction float64
}

// DefaultRetryPolicy returns a retry policy with sensible defaults for LLM
// requests. It uses exponential backoff starting at 500ms, doubling each
// attempt up to 8s, with 10% jitter and a maximum of 3 retries.
//
// Returns *RetryPolicy configured with sensible defaults.
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:        DefaultRetryMaxRetries,
		InitialBackoff:    DefaultRetryInitialBackoff,
		MaxBackoff:        DefaultRetryMaxBackoff,
		BackoffMultiplier: DefaultRetryBackoffMultiplier,
		JitterFraction:    DefaultRetryJitterFraction,
	}
}

// NoRetryPolicy returns a retry policy that disables retries.
//
// Returns *RetryPolicy configured with zero retries.
func NoRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries: 0,
	}
}
