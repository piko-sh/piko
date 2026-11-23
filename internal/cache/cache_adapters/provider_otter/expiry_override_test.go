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

package provider_otter

import (
	"math"
	"testing"
	"time"

	"github.com/maypok86/otter/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_dto"
)

const (
	innerCreate          = 7 * time.Second
	innerUpdate          = 8 * time.Second
	innerRead            = 9 * time.Second
	overrideSnapshotNano = int64(1_768_471_200_000_000_000)
)

type stubExpiryCalculator struct{}

func (stubExpiryCalculator) ExpireAfterCreate(otter.Entry[string, string]) time.Duration {
	return innerCreate
}

func (stubExpiryCalculator) ExpireAfterUpdate(otter.Entry[string, string], string) time.Duration {
	return innerUpdate
}

func (stubExpiryCalculator) ExpireAfterRead(otter.Entry[string, string]) time.Duration {
	return innerRead
}

func TestOverridingExpiryCalculator(t *testing.T) {
	t.Parallel()

	entry := otter.Entry[string, string]{Key: ttlTestKey, SnapshotAtNano: overrideSnapshotNano}

	testCases := []struct {
		calculate func(calculator otter.ExpiryCalculator[string, string]) time.Duration
		name      string
		override  time.Duration
		want      time.Duration
		pending   bool
	}{
		{
			name: "create uses the pending TTL",
			calculate: func(calculator otter.ExpiryCalculator[string, string]) time.Duration {
				return calculator.ExpireAfterCreate(entry)
			},
			override: time.Minute,
			pending:  true,
			want:     time.Minute,
		},
		{
			name: "update uses the pending TTL",
			calculate: func(calculator otter.ExpiryCalculator[string, string]) time.Duration {
				return calculator.ExpireAfterUpdate(entry, "old")
			},
			override: time.Minute,
			pending:  true,
			want:     time.Minute,
		},
		{
			name: "read ignores the pending TTL",
			calculate: func(calculator otter.ExpiryCalculator[string, string]) time.Duration {
				return calculator.ExpireAfterRead(entry)
			},
			override: time.Minute,
			pending:  true,
			want:     innerRead,
		},
		{
			name: "create without a pending TTL defers to the cache's calculator",
			calculate: func(calculator otter.ExpiryCalculator[string, string]) time.Duration {
				return calculator.ExpireAfterCreate(entry)
			},
			want: innerCreate,
		},
		{
			name: "update without a pending TTL defers to the cache's calculator",
			calculate: func(calculator otter.ExpiryCalculator[string, string]) time.Duration {
				return calculator.ExpireAfterUpdate(entry, "old")
			},
			want: innerUpdate,
		},
		{
			name: "NoExpiry reaches the latest representable instant",
			calculate: func(calculator otter.ExpiryCalculator[string, string]) time.Duration {
				return calculator.ExpireAfterCreate(entry)
			},
			override: cache_dto.NoExpiry,
			pending:  true,
			want:     time.Duration(math.MaxInt64 - overrideSnapshotNano),
		},
		{
			name: "NoExpiry without a clock reading gives the longest duration",
			calculate: func(calculator otter.ExpiryCalculator[string, string]) time.Duration {
				return calculator.ExpireAfterCreate(otter.Entry[string, string]{Key: ttlTestKey})
			},
			override: cache_dto.NoExpiry,
			pending:  true,
			want:     time.Duration(math.MaxInt64),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			overrides := &expiryOverrides[string]{}
			calculator := newOverridingExpiryCalculator[string, string](stubExpiryCalculator{}, overrides)
			if testCase.pending {
				overrides.store(ttlTestKey, testCase.override)
			}

			assert.Equal(t, testCase.want, testCase.calculate(calculator))
		})
	}
}

func TestExpiryOverrides_ConsumedOnce(t *testing.T) {
	t.Parallel()

	overrides := &expiryOverrides[string]{}
	overrides.store(ttlTestKey, time.Minute)

	duration, ok := overrides.take(ttlTestKey)
	require.True(t, ok)
	assert.Equal(t, time.Minute, duration)

	_, ok = overrides.take(ttlTestKey)
	assert.False(t, ok, "a pending TTL belongs to one write only")
	assert.Zero(t, overrides.count.Load())
}

func TestExpiryOverrides_DiscardRemovesOnlyItsOwnOverride(t *testing.T) {
	t.Parallel()

	overrides := &expiryOverrides[string]{}
	stale := overrides.store(ttlTestKey, time.Minute)
	overrides.store(ttlTestKey, time.Hour)
	assert.Equal(t, int64(1), overrides.count.Load(), "a replaced override is no longer pending")

	overrides.discard(ttlTestKey, stale)

	duration, ok := overrides.take(ttlTestKey)
	require.True(t, ok, "discarding a stale override must not remove a newer write's TTL")
	assert.Equal(t, time.Hour, duration)

	current := overrides.store(ttlTestKey, time.Second)
	overrides.discard(ttlTestKey, current)
	_, ok = overrides.take(ttlTestKey)
	assert.False(t, ok)
	assert.Zero(t, overrides.count.Load())
}

func TestOverrideDuration_RaisesNonPositiveTTLsToOneNanosecond(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		duration time.Duration
		want     time.Duration
	}{
		{name: "zero", duration: 0, want: time.Nanosecond},
		{name: "negative", duration: -time.Second, want: time.Nanosecond},
		{name: "positive", duration: time.Millisecond, want: time.Millisecond},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, overrideDuration(testCase.duration, overrideSnapshotNano),
				"otter keeps the old expiry for a duration of zero or less, so it must never see one")
		})
	}
}

func TestOverrideDuration_CapsATTLSoTheExpiryCannotOverflow(t *testing.T) {
	t.Parallel()

	snapshotAtNano := time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano()

	longestTTL := cache_dto.NoExpiry - 1

	duration := overrideDuration(longestTTL, snapshotAtNano)

	expiresAt := snapshotAtNano + int64(duration)
	assert.Greater(t, expiresAt, snapshotAtNano, "otter adds the duration to the time of the write, so the sum must not wrap")
	assert.Less(t, expiresAt, int64(math.MaxInt64), "a capped TTL is still an expiry, not the never-expires marker")
}
