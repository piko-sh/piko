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

package llm_domain

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// MaxRetryAfterDuration caps the Retry-After value honoured by the retry
	// executor so a hostile or misbehaving server cannot push the client into
	// an excessively long sleep.
	MaxRetryAfterDuration = 5 * time.Minute
)

// RetryableError is an optional interface that errors can implement to
// indicate whether they should be retried. When present, this takes priority
// over string-based error classification in the retry executor.
type RetryableError interface {
	error

	// IsRetryable reports whether this error represents a transient
	// failure that should be retried.
	//
	// Returns bool which is true if the error is retryable.
	IsRetryable() bool
}

var (
	// ErrProviderNotFound is returned when the requested LLM provider is not
	// registered.
	ErrProviderNotFound = errors.New("llm provider not found")

	// ErrNoDefaultProvider is returned when no default provider has been set.
	ErrNoDefaultProvider = errors.New("no default llm provider configured")

	// ErrProviderAlreadyExists indicates a provider with that name is already
	// registered.
	ErrProviderAlreadyExists = errors.New("llm provider already exists")

	// ErrStreamingNotSupported is returned when the LLM provider does not support
	// streaming responses.
	ErrStreamingNotSupported = errors.New("llm provider does not support streaming")

	// ErrToolsNotSupported indicates the provider does not support tool calling.
	ErrToolsNotSupported = errors.New("llm provider does not support tools")

	// ErrStructuredOutputNotSupported indicates the provider does not support
	// structured output.
	ErrStructuredOutputNotSupported = errors.New("llm provider does not support structured output")

	// ErrPenaltiesNotSupported indicates the provider does not support
	// frequency/presence penalties.
	ErrPenaltiesNotSupported = errors.New("llm provider does not support frequency/presence penalties")

	// ErrSeedNotSupported indicates the provider does not support the seed
	// parameter.
	ErrSeedNotSupported = errors.New("llm provider does not support seed")

	// ErrParallelToolCallsNotSupported indicates the provider does not support
	// parallel tool calls.
	ErrParallelToolCallsNotSupported = errors.New("llm provider does not support parallel tool calls")

	// ErrMessageNameNotSupported indicates the provider does not support the
	// Name field on messages.
	ErrMessageNameNotSupported = errors.New("llm provider does not support message names")

	// ErrEmptyMessages is returned when a completion request has no messages.
	ErrEmptyMessages = errors.New("completion request must contain at least one message")

	// ErrEmptyModel indicates no model was specified in the request.
	ErrEmptyModel = errors.New("completion request must specify a model")

	// ErrInvalidTemperature is returned when the temperature value is not between
	// 0 and 2.
	ErrInvalidTemperature = errors.New("temperature must be between 0 and 2")

	// ErrInvalidTopP indicates the top_p value is outside the valid range.
	ErrInvalidTopP = errors.New("top_p must be between 0 and 1")

	// ErrInvalidMaxTokens indicates the max_tokens value is invalid.
	ErrInvalidMaxTokens = errors.New("max_tokens must be positive")

	// ErrBudgetExceeded is returned when the budget limit
	// has been exceeded.
	ErrBudgetExceeded = errors.New("budget limit exceeded")

	// ErrRateLimited is returned when too many requests have been made.
	ErrRateLimited = errors.New("rate limit exceeded")

	// ErrMaxCostExceeded is returned when the estimated cost is higher than the
	// limit allowed for a single request.
	ErrMaxCostExceeded = errors.New("estimated cost exceeds per-request limit")

	// ErrUnknownModelPrice indicates no pricing information is available for the
	// model.
	ErrUnknownModelPrice = errors.New("no pricing information for model")

	// ErrProviderOverloaded indicates the provider is
	// temporarily overloaded.
	ErrProviderOverloaded = errors.New("provider overloaded")

	// ErrProviderTimeout is returned when a provider request takes too long.
	ErrProviderTimeout = errors.New("provider timeout")

	// ErrVectorStoreNotConfigured is returned when vector operations are
	// attempted without a configured vector store.
	ErrVectorStoreNotConfigured = errors.New("vector store is not configured")

	// retryableStatusCodes contains HTTP status codes that indicate transient
	// failures worth retrying.
	retryableStatusCodes = map[int]struct{}{
		408: {},
		409: {},
		425: {},
		429: {},
		500: {},
		502: {},
		503: {},
		504: {},
	}
)

// ProviderError represents an error returned by an LLM provider, carrying
// the provider name, HTTP status code, and a descriptive message. It
// implements both error and RetryableError.
type ProviderError struct {
	// Err is the underlying error, if any.
	Err error

	// Provider is the name of the LLM provider that returned the error.
	Provider string

	// Message is a human-readable description of the error.
	Message string

	// StatusCode is the HTTP status code from the provider response.
	StatusCode int

	// RetryAfter is the duration the server has hinted the client should wait
	// before retrying, parsed from the Retry-After HTTP response header.
	// A zero value indicates no hint was provided.
	RetryAfter time.Duration
}

// Error returns a formatted string describing the provider error.
//
// Returns string in the form "provider [Provider]: [StatusCode] [Message]".
func (e *ProviderError) Error() string {
	return fmt.Sprintf("provider %s: %d %s", e.Provider, e.StatusCode, e.Message)
}

// Unwrap returns the underlying error.
//
// Returns error which is the wrapped cause, or nil.
func (e *ProviderError) Unwrap() error {
	return e.Err
}

// IsRetryable reports whether this provider error represents a transient
// failure that should be retried.
//
// Returns true for status codes 408, 409, 425, 429, 500, 502, 503, 504.
func (e *ProviderError) IsRetryable() bool {
	_, ok := retryableStatusCodes[e.StatusCode]
	return ok
}

// ParseRetryAfter converts a Retry-After header value into a duration.
//
// The header may carry either an integer number of seconds or an HTTP-date
// timestamp. Returns the parsed duration capped at MaxRetryAfterDuration,
// or zero when the header is absent or unparseable.
//
// Takes header (string) which is the raw Retry-After header value.
// Takes now (time.Time) which is the reference time used to convert HTTP-date
// values into a relative duration.
//
// Returns time.Duration which is the parsed value, capped at the maximum, or
// zero when the header carries no usable hint.
func ParseRetryAfter(header string, now time.Time) time.Duration {
	header = strings.TrimSpace(header)
	if header == "" {
		return 0
	}

	if seconds, err := strconv.Atoi(header); err == nil {
		if seconds <= 0 {
			return 0
		}
		duration := time.Duration(seconds) * time.Second
		return capRetryAfter(duration)
	}

	if when, err := http.ParseTime(header); err == nil {
		duration := when.Sub(now)
		if duration <= 0 {
			return 0
		}
		return capRetryAfter(duration)
	}

	return 0
}

// capRetryAfter clamps duration to MaxRetryAfterDuration so a server hint
// cannot push the retry into an unreasonably long sleep.
//
// Takes duration (time.Duration) which is the parsed Retry-After value.
//
// Returns time.Duration which is the value clamped at the cap.
func capRetryAfter(duration time.Duration) time.Duration {
	if duration > MaxRetryAfterDuration {
		return MaxRetryAfterDuration
	}
	return duration
}
