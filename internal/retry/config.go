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

package retry

import (
	mathrand "math/rand/v2"
	"time"
)

// Config holds settings for retrying failed operations with exponential
// backoff.
type Config struct {
	// JitterFunc adds randomness to retry delays to spread out requests.
	// If nil, defaults to [DefaultJitter] which adds 10% of the delay.
	JitterFunc func(delay time.Duration) time.Duration `json:"-"`

	// InitialDelay is the wait time before the first retry attempt.
	InitialDelay time.Duration `json:"initial_delay"`

	// MaxDelay caps the delay between retry attempts.
	MaxDelay time.Duration `json:"max_delay"`

	// BackoffFactor is the multiplier applied to the delay after each retry.
	BackoffFactor float64 `json:"backoff_factor"`

	// MaxRetries is the maximum number of retry attempts; 0 means no retries.
	MaxRetries int `json:"max_retries"`
}

// CalculateNextRetry calculates the next retry time using exponential backoff
// with jitter.
//
// Takes attempt (int) which is the current retry attempt number.
// Takes baseTime (time.Time) which is the reference time to calculate from.
//
// Returns time.Time which is the calculated next retry time.
func (c Config) CalculateNextRetry(attempt int, baseTime time.Time) time.Time {
	if attempt <= 0 {
		return baseTime.Add(c.InitialDelay)
	}

	delay := c.InitialDelay
	for i := 1; i < attempt; i++ {
		delay = time.Duration(float64(delay) * c.BackoffFactor)
		if delay > c.MaxDelay {
			delay = c.MaxDelay
			break
		}
	}

	jitter := c.getJitter(delay)
	return baseTime.Add(delay + jitter)
}

// ShouldRetry determines if an operation should be retried based on attempt
// count.
//
// Takes attempt (int) which is the current attempt number.
//
// Returns bool which is true if the attempt is within the retry limit.
func (c Config) ShouldRetry(attempt int) bool {
	return attempt <= c.MaxRetries
}

// getJitter returns the jitter duration using the configured function or the
// default.
//
// Takes delay (time.Duration) which is the base delay to calculate jitter for.
//
// Returns time.Duration which is the calculated jitter value.
func (c Config) getJitter(delay time.Duration) time.Duration {
	if c.JitterFunc != nil {
		return c.JitterFunc(delay)
	}
	return DefaultJitter(delay)
}

// DefaultJitter returns a random duration between 0 and 10% of the given
// delay. This is the default jitter strategy used when Config.JitterFunc is
// nil.
//
// Takes delay (time.Duration) which is the base delay to calculate jitter
// from.
//
// Returns time.Duration which is a random value in [0, delay/10).
func DefaultJitter(delay time.Duration) time.Duration {
	if delay <= 0 {
		return 0
	}
	return time.Duration(mathrand.Int64N(int64(delay) / 10)) //nolint:gosec // jitter, not security
}
