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
package cache_provider_redis

import (
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"piko.sh/piko/wdk/cache"
)

func TestRedisAdapter_WriteExpiry(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		ttl      time.Duration
		fallback time.Duration
		want     time.Duration
		perEntry bool
	}{
		{name: "a positive TTL is used", ttl: time.Minute, fallback: time.Hour, want: time.Minute},
		{name: "a sub-millisecond TTL rounds up", ttl: time.Microsecond, want: time.Millisecond},
		{name: "NoExpiry sets none", ttl: cache.NoExpiry, fallback: time.Hour, want: 0},
		{name: "no TTL under per-entry expiry keeps the TTL", ttl: 0, fallback: time.Hour, perEntry: true, want: redis.KeepTTL},
		{name: "no TTL otherwise applies the default", ttl: 0, fallback: time.Hour, want: time.Hour},
		{name: "a sub-millisecond default rounds up", ttl: 0, fallback: time.Microsecond, want: time.Millisecond},
		{name: "a default of minus one sets none rather than keeping the TTL", ttl: 0, fallback: -1, want: 0},
		{name: "a negative default sets none", ttl: 0, fallback: -time.Second, want: 0},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			adapter := &RedisAdapter[string, string]{perEntry: testCase.perEntry}

			assert.Equal(t, testCase.want, adapter.writeExpiry(testCase.ttl, testCase.fallback))
		})
	}
}

func TestExpiryFromPTTL(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	testCases := []struct {
		name string
		ttl  time.Duration
		want int64
	}{
		{name: "no TTL", ttl: -1, want: 0},
		{name: "missing key", ttl: -2, want: 0},
		{name: "under one millisecond left", ttl: 0, want: now.Add(time.Millisecond).UnixNano()},
		{name: "a TTL", ttl: 1500 * time.Millisecond, want: now.Add(1500 * time.Millisecond).UnixNano()},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, expiryFromPTTL(now, testCase.ttl))
		})
	}
}
