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
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/clock"
)

func TestNewRetryExecutor(t *testing.T) {
	t.Run("creates with default policy when nil", func(t *testing.T) {
		executor := NewRetryExecutor(nil)

		require.NotNil(t, executor)
		assert.NotNil(t, executor.policy)
		assert.Equal(t, 3, executor.policy.MaxRetries)
	})

	t.Run("creates with custom policy", func(t *testing.T) {
		policy := &llm_dto.RetryPolicy{
			MaxRetries:     5,
			InitialBackoff: 100 * time.Millisecond,
		}

		executor := NewRetryExecutor(policy)

		require.NotNil(t, executor)
		assert.Equal(t, 5, executor.policy.MaxRetries)
	})

	t.Run("creates with custom clock", func(t *testing.T) {
		mockClock := clock.NewMockClock(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
		executor := NewRetryExecutor(nil, WithRetryExecutorClock(mockClock))

		require.NotNil(t, executor)
		assert.Equal(t, mockClock, executor.clock)
	})
}

func TestRetryExecutor_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("succeeds on first attempt", func(t *testing.T) {
		executor := NewRetryExecutor(&llm_dto.RetryPolicy{MaxRetries: 3})

		attempts := 0
		err := executor.Execute(ctx, func() error {
			attempts++
			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("succeeds after retries", func(t *testing.T) {
		mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
		executor := NewRetryExecutor(&llm_dto.RetryPolicy{
			MaxRetries:        3,
			InitialBackoff:    10 * time.Millisecond,
			MaxBackoff:        100 * time.Millisecond,
			BackoffMultiplier: 2.0,
		}, WithRetryExecutorClock(mockClock))

		attempts := 0
		errChan := make(chan error, 1)
		baseline := mockClock.TimerCount()

		go func() {
			err := executor.Execute(ctx, func() error {
				attempts++
				if attempts < 3 {
					return errors.New("rate limit exceeded")
				}
				return nil
			})
			errChan <- err
		}()

		for range 2 {
			require.True(t, mockClock.AwaitTimerSetup(baseline, time.Second))
			baseline = mockClock.TimerCount()
			mockClock.Advance(50 * time.Millisecond)
		}

		err := <-errChan
		require.NoError(t, err)
		assert.Equal(t, 3, attempts)
	})

	t.Run("fails after max retries", func(t *testing.T) {
		mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
		executor := NewRetryExecutor(&llm_dto.RetryPolicy{
			MaxRetries:        2,
			InitialBackoff:    10 * time.Millisecond,
			MaxBackoff:        100 * time.Millisecond,
			BackoffMultiplier: 2.0,
		}, WithRetryExecutorClock(mockClock))

		attempts := 0
		errChan := make(chan error, 1)
		baseline := mockClock.TimerCount()

		go func() {
			err := executor.Execute(ctx, func() error {
				attempts++
				return errors.New("rate limit")
			})
			errChan <- err
		}()

		for range 2 {
			require.True(t, mockClock.AwaitTimerSetup(baseline, time.Second))
			baseline = mockClock.TimerCount()
			mockClock.Advance(50 * time.Millisecond)
		}

		err := <-errChan
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rate limit")
		assert.Equal(t, 3, attempts)
	})

	t.Run("returns immediately for non-retryable error", func(t *testing.T) {
		executor := NewRetryExecutor(&llm_dto.RetryPolicy{MaxRetries: 3})

		attempts := 0
		err := executor.Execute(ctx, func() error {
			attempts++
			return errors.New("authentication failed")
		})

		require.Error(t, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		executor := NewRetryExecutor(&llm_dto.RetryPolicy{MaxRetries: 3})

		ctx, cancel := context.WithCancelCause(ctx)
		cancel(fmt.Errorf("test: simulating cancelled context"))

		err := executor.Execute(ctx, func() error {
			return errors.New("should not execute")
		})

		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("calls OnRetry callback", func(t *testing.T) {
		mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
		callbackCalled := false

		policy := &llm_dto.RetryPolicy{
			MaxRetries:        3,
			InitialBackoff:    10 * time.Millisecond,
			MaxBackoff:        100 * time.Millisecond,
			BackoffMultiplier: 2.0,
			OnRetry: func(attempt int, err error, nextBackoff time.Duration) {
				callbackCalled = true
				assert.Equal(t, 1, attempt)
				assert.NotNil(t, err)
			},
		}

		executor := NewRetryExecutor(policy, WithRetryExecutorClock(mockClock))
		errChan := make(chan error, 1)
		baseline := mockClock.TimerCount()

		go func() {
			attempts := 0
			err := executor.Execute(ctx, func() error {
				attempts++
				if attempts < 2 {
					return errors.New("rate limit")
				}
				return nil
			})
			errChan <- err
		}()

		require.True(t, mockClock.AwaitTimerSetup(baseline, time.Second))
		mockClock.Advance(50 * time.Millisecond)

		<-errChan
		assert.True(t, callbackCalled)
	})
}

func TestRetryExecutor_ExecuteWithResult(t *testing.T) {
	ctx := context.Background()

	t.Run("returns result on success", func(t *testing.T) {
		executor := NewRetryExecutor(&llm_dto.RetryPolicy{MaxRetries: 3})

		result, err := ExecuteWithResult(ctx, executor, func() (string, error) {
			return "success", nil
		})

		require.NoError(t, err)
		assert.Equal(t, "success", result)
	})

	t.Run("returns result after retries", func(t *testing.T) {
		mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
		executor := NewRetryExecutor(&llm_dto.RetryPolicy{
			MaxRetries:        3,
			InitialBackoff:    10 * time.Millisecond,
			MaxBackoff:        100 * time.Millisecond,
			BackoffMultiplier: 2.0,
		}, WithRetryExecutorClock(mockClock))

		attempts := 0
		resultChan := make(chan struct {
			err    error
			result string
		}, 1)
		baseline := mockClock.TimerCount()

		go func() {
			result, err := ExecuteWithResult(ctx, executor, func() (string, error) {
				attempts++
				if attempts < 2 {
					return "", errors.New("rate limit")
				}
				return "success after retry", nil
			})
			resultChan <- struct {
				err    error
				result string
			}{err: err, result: result}
		}()

		require.True(t, mockClock.AwaitTimerSetup(baseline, time.Second))
		mockClock.Advance(50 * time.Millisecond)

		retryResult := <-resultChan
		require.NoError(t, retryResult.err)
		assert.Equal(t, "success after retry", retryResult.result)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		executor := NewRetryExecutor(&llm_dto.RetryPolicy{MaxRetries: 3})

		ctx, cancel := context.WithCancelCause(ctx)
		cancel(fmt.Errorf("test: simulating cancelled context"))

		_, err := ExecuteWithResult(ctx, executor, func() (string, error) {
			return "", errors.New("should not execute")
		})

		assert.ErrorIs(t, err, context.Canceled)
	})
}

func TestRetryExecutor_IsRetryable(t *testing.T) {
	executor := NewRetryExecutor(nil)

	testCases := []struct {
		err         error
		name        string
		isRetryable bool
	}{
		{
			name:        "nil error",
			err:         nil,
			isRetryable: false,
		},
		{
			name:        "ErrProviderOverloaded",
			err:         ErrProviderOverloaded,
			isRetryable: true,
		},
		{
			name:        "ErrProviderTimeout",
			err:         ErrProviderTimeout,
			isRetryable: true,
		},
		{
			name:        "ErrRateLimited",
			err:         ErrRateLimited,
			isRetryable: true,
		},
		{
			name:        "rate limit in message",
			err:         errors.New("rate limit exceeded"),
			isRetryable: true,
		},
		{
			name:        "rate_limit in message",
			err:         errors.New("rate_limit error"),
			isRetryable: true,
		},
		{
			name:        "too many requests",
			err:         errors.New("too many requests"),
			isRetryable: true,
		},
		{
			name:        "HTTP 429",
			err:         errors.New("got 429 response"),
			isRetryable: true,
		},
		{
			name:        "overloaded",
			err:         errors.New("server overloaded"),
			isRetryable: true,
		},
		{
			name:        "HTTP 503",
			err:         errors.New("503 service unavailable"),
			isRetryable: true,
		},
		{
			name:        "HTTP 502",
			err:         errors.New("502 bad gateway"),
			isRetryable: true,
		},
		{
			name:        "HTTP 504",
			err:         errors.New("504 gateway timeout"),
			isRetryable: true,
		},
		{
			name:        "timeout",
			err:         errors.New("request timeout"),
			isRetryable: true,
		},
		{
			name:        "connection reset",
			err:         errors.New("connection reset by peer"),
			isRetryable: true,
		},
		{
			name:        "connection refused",
			err:         errors.New("connection refused"),
			isRetryable: true,
		},
		{
			name:        "eof",
			err:         errors.New("unexpected eof"),
			isRetryable: true,
		},
		{
			name:        "broken pipe",
			err:         errors.New("broken pipe"),
			isRetryable: true,
		},
		{
			name:        "temporary error",
			err:         errors.New("temporary error"),
			isRetryable: true,
		},
		{
			name:        "authentication error (not retryable)",
			err:         errors.New("authentication failed"),
			isRetryable: false,
		},
		{
			name:        "invalid request (not retryable)",
			err:         errors.New("invalid request parameters"),
			isRetryable: false,
		},
		{
			name:        "not found (not retryable)",
			err:         errors.New("model not found"),
			isRetryable: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := executor.IsRetryable(tc.err)
			assert.Equal(t, tc.isRetryable, result, "IsRetryable(%v)", tc.err)
		})
	}
}

func TestRetryExecutor_GetPolicy(t *testing.T) {
	policy := &llm_dto.RetryPolicy{
		MaxRetries:        5,
		InitialBackoff:    200 * time.Millisecond,
		MaxBackoff:        10 * time.Second,
		BackoffMultiplier: 1.5,
		JitterFraction:    0.2,
	}

	executor := NewRetryExecutor(policy)

	result := executor.GetPolicy()

	assert.Equal(t, policy, result)
}

func TestRetryExecutor_CalculateBackoff(t *testing.T) {
	t.Run("calculates exponential backoff", func(t *testing.T) {
		policy := &llm_dto.RetryPolicy{
			MaxRetries:        3,
			InitialBackoff:    100 * time.Millisecond,
			MaxBackoff:        10 * time.Second,
			BackoffMultiplier: 2.0,
			JitterFraction:    0,
		}

		executor := NewRetryExecutor(policy)

		backoff0 := executor.calculateBackoff(0)
		assert.Equal(t, 100*time.Millisecond, backoff0)

		backoff1 := executor.calculateBackoff(1)
		assert.Equal(t, 200*time.Millisecond, backoff1)

		backoff2 := executor.calculateBackoff(2)
		assert.Equal(t, 400*time.Millisecond, backoff2)
	})

	t.Run("caps at max backoff", func(t *testing.T) {
		policy := &llm_dto.RetryPolicy{
			MaxRetries:        10,
			InitialBackoff:    100 * time.Millisecond,
			MaxBackoff:        500 * time.Millisecond,
			BackoffMultiplier: 10.0,
			JitterFraction:    0,
		}

		executor := NewRetryExecutor(policy)

		backoff := executor.calculateBackoff(5)
		assert.Equal(t, 500*time.Millisecond, backoff)
	})

	t.Run("adds jitter when configured", func(t *testing.T) {
		policy := &llm_dto.RetryPolicy{
			MaxRetries:        3,
			InitialBackoff:    100 * time.Millisecond,
			MaxBackoff:        10 * time.Second,
			BackoffMultiplier: 2.0,
			JitterFraction:    0.1,
		}

		executor := NewRetryExecutor(policy)

		backoffs := make([]time.Duration, 10)
		for i := range backoffs {
			backoffs[i] = executor.calculateBackoff(0)
		}

		for _, b := range backoffs {
			assert.GreaterOrEqual(t, b, 100*time.Millisecond)
			assert.LessOrEqual(t, b, 110*time.Millisecond)
		}
	})
}

type retryableTestError struct {
	message   string
	retryable bool
}

func (e *retryableTestError) Error() string     { return e.message }
func (e *retryableTestError) IsRetryable() bool { return e.retryable }

func TestIsRetryable_RetryableErrorInterface(t *testing.T) {
	executor := NewRetryExecutor(nil)

	t.Run("retryable true via interface", func(t *testing.T) {
		err := &retryableTestError{message: "custom error", retryable: true}
		assert.True(t, executor.IsRetryable(err))
	})

	t.Run("retryable false via interface", func(t *testing.T) {
		err := &retryableTestError{message: "rate limit exceeded", retryable: false}
		assert.False(t, executor.IsRetryable(err))
	})

	t.Run("wrapped retryable error", func(t *testing.T) {
		inner := &retryableTestError{message: "transient", retryable: true}
		err := fmt.Errorf("provider call failed: %w", inner)
		assert.True(t, executor.IsRetryable(err))
	})
}

func TestIsRetryable_FallbackStringMatching(t *testing.T) {
	executor := NewRetryExecutor(nil)

	assert.True(t, executor.IsRetryable(errors.New("rate limit exceeded")))
	assert.True(t, executor.IsRetryable(errors.New("429 Too Many Requests")))
	assert.True(t, executor.IsRetryable(errors.New("service unavailable")))
	assert.False(t, executor.IsRetryable(errors.New("invalid api key")))
}

func TestIsRetryable_ProviderError(t *testing.T) {
	executor := NewRetryExecutor(nil)

	t.Run("retryable status code", func(t *testing.T) {
		err := &ProviderError{Provider: "openai", StatusCode: 429, Message: "rate limited"}
		assert.True(t, executor.IsRetryable(err))
	})

	t.Run("non-retryable status code", func(t *testing.T) {
		err := &ProviderError{Provider: "openai", StatusCode: 401, Message: "unauthorised"}
		assert.False(t, executor.IsRetryable(err))
	})

	t.Run("server error retryable", func(t *testing.T) {
		err := &ProviderError{Provider: "anthropic", StatusCode: 500, Message: "internal server error"}
		assert.True(t, executor.IsRetryable(err))
	})

	t.Run("wrapped provider error", func(t *testing.T) {
		inner := &ProviderError{Provider: "ollama", StatusCode: 503, Message: "service unavailable"}
		err := fmt.Errorf("completion failed: %w", inner)
		assert.True(t, executor.IsRetryable(err))
	})
}

func TestIsRetryable_408_409_425(t *testing.T) {
	executor := NewRetryExecutor(nil)

	assert.True(t, executor.IsRetryable(&ProviderError{StatusCode: 408, Message: "request timeout"}))
	assert.True(t, executor.IsRetryable(&ProviderError{StatusCode: 409, Message: "conflict"}))
	assert.True(t, executor.IsRetryable(&ProviderError{StatusCode: 425, Message: "too early"}))
}

func TestRetry_HonoursRetryAfterHeader(t *testing.T) {
	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	var observedBackoff time.Duration
	policy := &llm_dto.RetryPolicy{
		MaxRetries:        2,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        500 * time.Millisecond,
		BackoffMultiplier: 2.0,
		OnRetry: func(_ int, _ error, nextBackoff time.Duration) {
			observedBackoff = nextBackoff
		},
	}

	executor := NewRetryExecutor(policy, WithRetryExecutorClock(mockClock))
	errChan := make(chan error, 1)
	baseline := mockClock.TimerCount()

	rateLimitErr := &ProviderError{
		Provider:   "openai",
		StatusCode: http.StatusTooManyRequests,
		Message:    "rate limited",
		RetryAfter: 2 * time.Second,
	}

	go func() {
		attempts := 0
		err := executor.Execute(context.Background(), func() error {
			attempts++
			if attempts == 1 {
				return rateLimitErr
			}
			return nil
		})
		errChan <- err
	}()

	require.True(t, mockClock.AwaitTimerSetup(baseline, time.Second))
	mockClock.Advance(3 * time.Second)

	require.NoError(t, <-errChan)
	assert.GreaterOrEqual(t, observedBackoff, 2*time.Second,
		"expected backoff to honour Retry-After hint of 2 seconds")
}

func TestRetry_RetryAfterCappedAtMaximum(t *testing.T) {
	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	var observedBackoff time.Duration
	policy := &llm_dto.RetryPolicy{
		MaxRetries:        2,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        500 * time.Millisecond,
		BackoffMultiplier: 2.0,
		OnRetry: func(_ int, _ error, nextBackoff time.Duration) {
			observedBackoff = nextBackoff
		},
	}

	executor := NewRetryExecutor(policy, WithRetryExecutorClock(mockClock))
	errChan := make(chan error, 1)
	baseline := mockClock.TimerCount()

	hostileErr := &ProviderError{
		Provider:   "openai",
		StatusCode: http.StatusServiceUnavailable,
		Message:    "service unavailable",
		RetryAfter: time.Hour,
	}

	go func() {
		attempts := 0
		err := executor.Execute(context.Background(), func() error {
			attempts++
			if attempts == 1 {
				return hostileErr
			}
			return nil
		})
		errChan <- err
	}()

	require.True(t, mockClock.AwaitTimerSetup(baseline, time.Second))
	mockClock.Advance(MaxRetryAfterDuration + time.Second)

	require.NoError(t, <-errChan)
	assert.LessOrEqual(t, observedBackoff, MaxRetryAfterDuration,
		"expected Retry-After to be clamped at MaxRetryAfterDuration")
}

func TestParseRetryAfter_Seconds(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("integer seconds", func(t *testing.T) {
		assert.Equal(t, 30*time.Second, ParseRetryAfter("30", now))
	})

	t.Run("zero seconds returns zero", func(t *testing.T) {
		assert.Equal(t, time.Duration(0), ParseRetryAfter("0", now))
	})

	t.Run("negative seconds returns zero", func(t *testing.T) {
		assert.Equal(t, time.Duration(0), ParseRetryAfter("-5", now))
	})

	t.Run("empty header returns zero", func(t *testing.T) {
		assert.Equal(t, time.Duration(0), ParseRetryAfter("", now))
	})

	t.Run("whitespace header returns zero", func(t *testing.T) {
		assert.Equal(t, time.Duration(0), ParseRetryAfter("   ", now))
	})

	t.Run("seconds capped at maximum", func(t *testing.T) {
		assert.Equal(t, MaxRetryAfterDuration, ParseRetryAfter("3600", now))
	})

	t.Run("invalid format returns zero", func(t *testing.T) {
		assert.Equal(t, time.Duration(0), ParseRetryAfter("not-a-number", now))
	})
}

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	future := now.Add(45 * time.Second)
	header := future.Format(http.TimeFormat)

	got := ParseRetryAfter(header, now)
	assert.InDelta(t, (45 * time.Second).Seconds(), got.Seconds(), 1.0)
}

func TestParseRetryAfter_HTTPDateInPastReturnsZero(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	past := now.Add(-30 * time.Second).Format(http.TimeFormat)

	assert.Equal(t, time.Duration(0), ParseRetryAfter(past, now))
}
