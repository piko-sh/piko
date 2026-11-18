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

package cache_provider_valkey

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/cache/cache_encoder_json"
)

func TestValkeyAdapter_WriteExpiry(t *testing.T) {
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
		{name: "no TTL under per-entry expiry keeps the TTL", ttl: 0, fallback: time.Hour, perEntry: true, want: keepTTL},
		{name: "no TTL otherwise applies the default", ttl: 0, fallback: time.Hour, want: time.Hour},
		{name: "a negative default sets none", ttl: 0, fallback: -time.Second, want: 0},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			adapter := &ValkeyAdapter[string, string]{perEntry: testCase.perEntry}

			assert.Equal(t, testCase.want, adapter.writeExpiry(testCase.ttl, testCase.fallback))
		})
	}
}

func TestValkeyAdapter_ApplyJSONExpiryRoundsUpSubMillisecond(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	adapter, server := newMiniredisAdapter(t)
	require.NoError(t, server.Set("document", "{}"))

	require.NoError(t, adapter.applyJSONExpiry(ctx, "document", 500*time.Microsecond))

	assert.True(t, server.Exists("document"), "PEXPIRE 0 deleted the document")
	assert.Equal(t, time.Millisecond, server.TTL("document"))
}

func TestValkeyAdapter_ProbeEntryReportsSubMillisecondTTL(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	adapter, server := newMiniredisAdapter(t)
	require.NoError(t, adapter.SetWithTTL(ctx, "key", 1, time.Minute))
	keyString, err := adapter.encodeKey("key")
	require.NoError(t, err)
	server.SetTTL(keyString, 500*time.Microsecond)

	entry, found, err := adapter.ProbeEntry(ctx, "key")
	require.NoError(t, err)
	require.True(t, found)

	assert.NotZero(t, entry.ExpiresAtNano, "a TTL under one millisecond must not read as no expiry")
}

func TestExpiryFromPTTL(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	testCases := []struct {
		name            string
		ttlMilliseconds int64
		want            int64
	}{
		{name: "no TTL", ttlMilliseconds: -1, want: 0},
		{name: "missing key", ttlMilliseconds: -2, want: 0},
		{name: "under one millisecond left", ttlMilliseconds: 0, want: now.Add(time.Millisecond).UnixNano()},
		{name: "a TTL", ttlMilliseconds: 1500, want: now.Add(1500 * time.Millisecond).UnixNano()},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, expiryFromPTTL(now, testCase.ttlMilliseconds))
		})
	}
}

func TestValkeyAdapter_SetWithTTLKeepsSubSecondTTL(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	adapter, server := newMiniredisAdapter(t)

	require.NoError(t, adapter.SetWithTTL(ctx, "k", 1, 500*time.Millisecond))

	keyString, err := adapter.encodeKey("k")
	require.NoError(t, err)
	assert.Equal(t, 500*time.Millisecond, server.TTL(keyString), "a sub-second TTL must be sent in milliseconds")
}

func newMiniredisAdapter(t *testing.T) (*ValkeyAdapter[string, float64], *miniredis.Miniredis) {
	t.Helper()

	server := miniredis.RunT(t)
	encoder, ok := cache_encoder_json.New[float64]().(cache.AnyEncoder)
	require.True(t, ok)
	provider, err := NewValkeyProvider(Config{Address: server.Addr(), Registry: cache.NewEncodingRegistry(encoder)})
	require.NoError(t, err)
	t.Cleanup(func() { _ = provider.Close() })

	created, err := createNamespaceGeneric(provider, "test", cache.Options[string, float64]{})
	require.NoError(t, err)
	adapter, ok := created.(*ValkeyAdapter[string, float64])
	require.True(t, ok)
	return adapter, server
}
