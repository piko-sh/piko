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
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"
	"time"

	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/retry"
	"piko.sh/piko/wdk/clock"
)

const (
	// DefaultJitterFallback is the fallback jitter value when crypto/rand fails.
	DefaultJitterFallback = 0.5
)

// llmErrorClassifier delegates string-based and network error classification
// to the shared retry package, extended with LLM-specific permanent errors
// and additional retryable HTTP status code patterns.
var llmErrorClassifier = retry.NewErrorClassifier(
	retry.WithPermanentErrors(
		ErrEmptyModel, ErrEmptyMessages, ErrInvalidTemperature,
		ErrInvalidTopP, ErrInvalidMaxTokens, ErrProviderNotFound,
		ErrNoDefaultProvider, ErrProviderAlreadyExists,
		ErrToolsNotSupported, ErrStructuredOutputNotSupported,
		ErrStreamingNotSupported,
	),
	retry.WithRetryablePatterns(
		"408", "409", "425", "429",
		"overloaded", "overload",
		"rate_limit", "timed out",
		"broken pipe", "eof", "temporary",
		"service unavailable",
	),
)

// RetryExecutor handles retry logic with exponential backoff for LLM requests.
type RetryExecutor struct {
	// clock provides time operations for backoff calculations during retries.
	clock clock.Clock

	// policy holds the retry configuration including max attempts and callbacks.
	policy *llm_dto.RetryPolicy
}

// RetryExecutorOption is a function type used to set options on a RetryExecutor.
type RetryExecutorOption func(*RetryExecutor)

// NewRetryExecutor creates a new RetryExecutor with the given policy.
//
// Takes policy (*llm_dto.RetryPolicy) which configures the retry behaviour.
// If policy is nil, a default policy is used.
// Takes opts (...RetryExecutorOption) which are optional configuration functions.
//
// Returns *RetryExecutor ready to execute operations with retries.
func NewRetryExecutor(policy *llm_dto.RetryPolicy, opts ...RetryExecutorOption) *RetryExecutor {
	if policy == nil {
		policy = llm_dto.DefaultRetryPolicy()
	}
	e := &RetryExecutor{
		clock:  clock.RealClock(),
		policy: policy,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Execute runs the given function with retries according to the configured
// policy.
//
// Takes operation (func() error) which is the operation to execute.
//
// Returns error from the final attempt, or nil if any attempt succeeds.
func (e *RetryExecutor) Execute(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt <= e.policy.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("retrying LLM request: %w", err)
		}

		lastErr = operation()
		if lastErr == nil {
			e.recordSuccessIfRetried(ctx, attempt)
			return nil
		}

		if done, err := e.handleFailedAttempt(ctx, attempt, lastErr); done {
			return fmt.Errorf("handling retry attempt %d: %w", attempt, err)
		}
	}

	return lastErr
}

// IsRetryable determines whether an error should trigger a retry.
// Errors are retryable if they indicate transient provider issues such as
// rate limiting, timeouts, overload, or network errors.
//
// The check order is:
//  1. RetryableError interface (typed errors self-classify)
//  2. LLM sentinel errors (ErrProviderOverloaded, etc.)
//  3. Shared ErrorClassifier (network, syscall, string patterns)
//
// Takes err (error) which is the error to check.
//
// Returns bool which is true if the error is retryable.
func (*RetryExecutor) IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	if re, ok := errors.AsType[RetryableError](err); ok {
		return re.IsRetryable()
	}

	if errors.Is(err, ErrProviderOverloaded) ||
		errors.Is(err, ErrProviderTimeout) ||
		errors.Is(err, ErrRateLimited) {
		return true
	}

	return llmErrorClassifier.IsRetryable(err)
}

// GetPolicy returns the retry policy used by this executor.
//
// Returns *llm_dto.RetryPolicy which is the current policy.
func (e *RetryExecutor) GetPolicy() *llm_dto.RetryPolicy {
	return e.policy
}

// recordSuccessIfRetried records a success metric if this was a retry.
//
// Takes attempt (int) which is the current attempt number, where 0 means the
// first attempt and values greater than 0 indicate retries.
func (*RetryExecutor) recordSuccessIfRetried(ctx context.Context, attempt int) {
	if attempt > 0 {
		retrySuccessCount.Add(ctx, 1)
	}
}

// handleFailedAttempt handles a failed attempt and decides whether to stop
// retrying.
//
// Takes attempt (int) which is the current attempt number.
// Takes err (error) which is the error from the failed attempt.
//
// Returns bool which is true when retrying should stop.
// Returns error when the error is not retryable, retries are exhausted, or
// the wait is interrupted.
func (e *RetryExecutor) handleFailedAttempt(ctx context.Context, attempt int, err error) (bool, error) {
	if !e.IsRetryable(err) {
		return true, err
	}

	if attempt >= e.policy.MaxRetries {
		retryExhaustedCount.Add(ctx, 1)
		return true, err
	}

	if waitErr := e.waitForRetry(ctx, attempt, err); waitErr != nil {
		return true, waitErr
	}
	return false, nil
}

// waitForRetry waits for the backoff duration before the next retry attempt.
// When the previous error carries a Retry-After hint via ProviderError.RetryAfter
// for status 429 or 503, that hint takes precedence over the calculated
// exponential backoff (capped at llm_domain.MaxRetryAfterDuration).
//
// Takes attempt (int) which is the current retry attempt number.
// Takes err (error) which is the error from the previous attempt.
//
// Returns error when the context is cancelled during the wait.
func (e *RetryExecutor) waitForRetry(ctx context.Context, attempt int, err error) error {
	ctx, l := logger_domain.From(ctx, log)
	retryAttemptCount.Add(ctx, 1)
	backoff := e.calculateBackoff(attempt)

	if hint := retryAfterHint(err); hint > 0 && hint > backoff {
		backoff = hint
	}

	if e.policy.OnRetry != nil {
		e.policy.OnRetry(attempt+1, err, backoff)
	}

	l.Debug("Retrying LLM request after error",
		logger_domain.Int("attempt", attempt+1),
		logger_domain.Int("max_retries", e.policy.MaxRetries),
		logger_domain.Duration("backoff", backoff),
		logger_domain.Error(err),
	)

	timer := e.clock.NewTimer(backoff)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C():
		return nil
	}
}

// retryAfterHint extracts the Retry-After hint from a ProviderError when the
// status code is 429 or 503. Returns zero when no usable hint is present.
//
// Takes err (error) which is the error to inspect.
//
// Returns time.Duration which is the parsed Retry-After value, or zero when
// not applicable.
func retryAfterHint(err error) time.Duration {
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) {
		return 0
	}
	if providerErr.RetryAfter <= 0 {
		return 0
	}
	if providerErr.StatusCode != http.StatusTooManyRequests &&
		providerErr.StatusCode != http.StatusServiceUnavailable {
		return 0
	}
	if providerErr.RetryAfter > MaxRetryAfterDuration {
		return MaxRetryAfterDuration
	}
	return providerErr.RetryAfter
}

// calculateBackoff computes the backoff duration for the given attempt number.
// It uses exponential backoff with optional jitter.
//
// Takes attempt (int) which specifies the retry attempt number.
//
// Returns time.Duration which is the computed backoff duration.
func (e *RetryExecutor) calculateBackoff(attempt int) time.Duration {
	backoff := float64(e.policy.InitialBackoff)

	for range attempt {
		backoff *= e.policy.BackoffMultiplier
	}

	maxBackoff := float64(e.policy.MaxBackoff)
	if backoff > maxBackoff {
		backoff = maxBackoff
	}

	if e.policy.JitterFraction > 0 {
		jitter := backoff * e.policy.JitterFraction * cryptoRandFloat64()
		backoff += jitter
	}

	return time.Duration(backoff)
}

// WithRetryExecutorClock sets the clock used for time operations.
// If not set, clock.RealClock() is used.
//
// Takes c (clock.Clock) which provides time operations.
//
// Returns RetryExecutorOption which applies this setting to the executor.
func WithRetryExecutorClock(c clock.Clock) RetryExecutorOption {
	return func(e *RetryExecutor) {
		e.clock = c
	}
}

// ExecuteWithResult runs the given function with retries, returning both
// a result and error.
//
// Takes e (*RetryExecutor) which provides the retry policy and behaviour.
// Takes operation (func() (T, error)) which is the operation to execute.
//
// Returns T which is the result from a successful attempt.
// Returns error when all retry attempts fail or the context is cancelled.
func ExecuteWithResult[T any](ctx context.Context, e *RetryExecutor, operation func() (T, error)) (T, error) {
	var result T
	var lastErr error

	for attempt := 0; attempt <= e.policy.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return result, err
		}

		result, lastErr = operation()
		if lastErr == nil {
			e.recordSuccessIfRetried(ctx, attempt)
			return result, nil
		}

		if done, err := e.handleFailedAttempt(ctx, attempt, lastErr); done {
			return result, err
		}
	}

	return result, lastErr
}

// cryptoRandFloat64 returns a cryptographically secure random float64 in [0, 1).
//
// Returns float64 which is a random value using crypto/rand for generation.
func cryptoRandFloat64() float64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return DefaultJitterFallback
	}
	u := binary.LittleEndian.Uint64(b[:])
	const maxUint53 = 1 << 53
	return float64(u>>11) / maxUint53
}
