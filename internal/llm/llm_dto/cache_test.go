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

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultCacheConfig(t *testing.T) {
	t.Parallel()

	config := DefaultCacheConfig()

	assert.True(t, config.Enabled)
	assert.Equal(t, time.Hour, config.TTL)
	assert.False(t, config.SkipWrite)
	assert.False(t, config.SkipRead)
	assert.False(t, config.UseProviderCache)
	assert.Empty(t, config.Key)
}

func TestCacheEntry_IsExpiredAt(t *testing.T) {
	t.Parallel()

	expiry := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	entry := &CacheEntry{
		ExpiresAt: expiry,
	}

	tests := []struct {
		now      time.Time
		name     string
		expected bool
	}{
		{name: "before expiry", now: expiry.Add(-time.Minute), expected: false},
		{name: "at expiry", now: expiry, expected: false},
		{name: "after expiry", now: expiry.Add(time.Minute), expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, entry.IsExpiredAt(tt.now))
		})
	}
}

func TestCacheStats_HitRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		hits     int64
		misses   int64
		expected float64
	}{
		{name: "zero total", hits: 0, misses: 0, expected: 0.0},
		{name: "all hits", hits: 10, misses: 0, expected: 1.0},
		{name: "all misses", hits: 0, misses: 10, expected: 0.0},
		{name: "mixed", hits: 3, misses: 7, expected: 0.3},
		{name: "half and half", hits: 5, misses: 5, expected: 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stats := &CacheStats{
				Hits:   tt.hits,
				Misses: tt.misses,
			}
			assert.InDelta(t, tt.expected, stats.HitRate(), 0.001)
		})
	}
}
