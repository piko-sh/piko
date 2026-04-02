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

package retry_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/retry"
)

func TestCalculateNextRetry(t *testing.T) {
	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	config := retry.Config{
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
	}

	t.Run("first attempt uses initial delay", func(t *testing.T) {
		nextRetry := config.CalculateNextRetry(1, baseTime)

		minExpected := baseTime.Add(100 * time.Millisecond)
		maxExpected := baseTime.Add(110 * time.Millisecond)
		assert.True(t, !nextRetry.Before(minExpected),
			"nextRetry %v should be >= minExpected %v", nextRetry, minExpected)
		assert.True(t, !nextRetry.After(maxExpected),
			"nextRetry %v should be <= maxExpected %v", nextRetry, maxExpected)
	})

	t.Run("second attempt doubles delay", func(t *testing.T) {
		nextRetry := config.CalculateNextRetry(2, baseTime)

		minExpected := baseTime.Add(200 * time.Millisecond)
		maxExpected := baseTime.Add(220 * time.Millisecond)
		assert.True(t, !nextRetry.Before(minExpected),
			"nextRetry %v should be >= minExpected %v", nextRetry, minExpected)
		assert.True(t, !nextRetry.After(maxExpected),
			"nextRetry %v should be <= maxExpected %v", nextRetry, maxExpected)
	})

	t.Run("third attempt quadruples initial delay", func(t *testing.T) {
		nextRetry := config.CalculateNextRetry(3, baseTime)

		minExpected := baseTime.Add(400 * time.Millisecond)
		maxExpected := baseTime.Add(440 * time.Millisecond)
		assert.True(t, !nextRetry.Before(minExpected),
			"nextRetry %v should be >= minExpected %v", nextRetry, minExpected)
		assert.True(t, !nextRetry.After(maxExpected),
			"nextRetry %v should be <= maxExpected %v", nextRetry, maxExpected)
	})

	t.Run("delay is capped at max delay", func(t *testing.T) {
		nextRetry := config.CalculateNextRetry(10, baseTime)

		maxExpected := baseTime.Add(1*time.Second + 100*time.Millisecond)
		assert.True(t, !nextRetry.After(maxExpected),
			"delay should be capped at max delay + jitter, got %v, max %v", nextRetry, maxExpected)
	})

	t.Run("zero attempt falls back to initial delay", func(t *testing.T) {
		nextRetry := config.CalculateNextRetry(0, baseTime)

		expected := baseTime.Add(100 * time.Millisecond)
		assert.Equal(t, expected, nextRetry,
			"zero attempt should return baseTime + InitialDelay without jitter")
	})

	t.Run("negative attempt falls back to initial delay", func(t *testing.T) {
		nextRetry := config.CalculateNextRetry(-1, baseTime)

		expected := baseTime.Add(100 * time.Millisecond)
		assert.Equal(t, expected, nextRetry,
			"negative attempt should return baseTime + InitialDelay without jitter")
	})

	t.Run("returns time in the future", func(t *testing.T) {
		for attempt := 0; attempt <= 5; attempt++ {
			result := config.CalculateNextRetry(attempt, baseTime)
			assert.True(t, !result.Before(baseTime),
				"attempt %d: expected result at or after base time", attempt)
		}
	})
}

func TestCalculateNextRetry_CustomJitter(t *testing.T) {
	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	config := retry.Config{
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
		JitterFunc:    func(_ time.Duration) time.Duration { return 0 },
	}

	t.Run("zero jitter produces exact delay", func(t *testing.T) {
		nextRetry := config.CalculateNextRetry(1, baseTime)
		expected := baseTime.Add(100 * time.Millisecond)
		assert.Equal(t, expected, nextRetry)
	})

	t.Run("custom jitter is applied", func(t *testing.T) {
		fixedJitter := 42 * time.Millisecond
		config.JitterFunc = func(_ time.Duration) time.Duration { return fixedJitter }

		nextRetry := config.CalculateNextRetry(1, baseTime)
		expected := baseTime.Add(100*time.Millisecond + fixedJitter)
		assert.Equal(t, expected, nextRetry)
	})

	t.Run("jitter receives calculated delay", func(t *testing.T) {
		var receivedDelay time.Duration
		config.JitterFunc = func(delay time.Duration) time.Duration {
			receivedDelay = delay
			return 0
		}

		config.CalculateNextRetry(2, baseTime)
		assert.Equal(t, 200*time.Millisecond, receivedDelay,
			"jitter function should receive the calculated delay")
	})
}

func TestShouldRetry(t *testing.T) {
	config := retry.Config{
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
	}

	t.Run("should retry within max retries", func(t *testing.T) {
		assert.True(t, config.ShouldRetry(1))
		assert.True(t, config.ShouldRetry(2))
		assert.True(t, config.ShouldRetry(3))
	})

	t.Run("should not retry after max retries", func(t *testing.T) {
		assert.False(t, config.ShouldRetry(4))
		assert.False(t, config.ShouldRetry(5))
		assert.False(t, config.ShouldRetry(100))
	})

	t.Run("zero max retries disables retry", func(t *testing.T) {
		noRetryConfig := retry.Config{MaxRetries: 0}
		assert.False(t, noRetryConfig.ShouldRetry(1))
	})

	t.Run("attempt zero is within limit", func(t *testing.T) {
		assert.True(t, config.ShouldRetry(0))
	})
}

func TestDefaultJitter(t *testing.T) {
	t.Run("zero delay returns zero jitter", func(t *testing.T) {
		assert.Equal(t, time.Duration(0), retry.DefaultJitter(0))
	})

	t.Run("negative delay returns zero jitter", func(t *testing.T) {
		assert.Equal(t, time.Duration(0), retry.DefaultJitter(-1*time.Second))
	})

	t.Run("jitter is within expected range", func(t *testing.T) {
		delay := 1 * time.Second
		maxJitter := delay / 10

		for range 100 {
			jitter := retry.DefaultJitter(delay)
			assert.True(t, jitter >= 0 && jitter < maxJitter,
				"jitter %v should be in [0, %v)", jitter, maxJitter)
		}
	})
}
