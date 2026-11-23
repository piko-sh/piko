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
package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cache_provider_mock "piko.sh/piko/internal/cache/cache_adapters/provider_mock"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
)

const (
	sharedCacheProviderName = "shared"
)

func TestContainer_GetRateLimiter(t *testing.T) {
	testCases := []struct {
		name          string
		provider      string
		cacheErr      error
		wantStoreName string
	}{
		{
			name:          "uses cache-backed stores on the named provider",
			provider:      sharedCacheProviderName,
			wantStoreName: "cache",
		},
		{
			name:          "falls back to no-op stores without a cache service or named provider",
			provider:      "",
			cacheErr:      errors.New("cache unavailable"),
			wantStoreName: providerNoop,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			container := newRateLimiterTestContainer(t, testCase.provider, testCase.cacheErr)

			limiter, err := container.GetRateLimiter()
			require.NoError(t, err)
			require.NotNil(t, limiter)

			again, err := container.GetRateLimiter()
			require.NoError(t, err)
			assert.Same(t, limiter, again, "the limiter is created once")

			status, err := limiter.GetStatus(t.Context())
			require.NoError(t, err)
			assert.Equal(t, testCase.wantStoreName, status.CounterStore)
			assert.Equal(t, testCase.wantStoreName, status.TokenBucketStore)

			result, err := limiter.CheckFixedWindow(t.Context(), "key",
				ratelimiter_dto.FixedWindowConfig{Limit: 1, Window: time.Minute})
			require.NoError(t, err)
			assert.True(t, result.Allowed)
		})
	}
}

func TestContainer_GetRateLimiter_NamedProviderFailureIsAnError(t *testing.T) {
	cacheErr := errors.New("cache unavailable")
	testCases := []struct {
		cacheErr error
		wantIs   error
		name     string
		provider string
	}{
		{
			name:     "named provider without a cache service",
			provider: sharedCacheProviderName,
			cacheErr: cacheErr,
			wantIs:   cacheErr,
		},
		{
			name:     "named provider whose stores cannot be built",
			provider: "missing",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			container := newRateLimiterTestContainer(t, testCase.provider, testCase.cacheErr)

			limiter, err := container.GetRateLimiter()

			require.Error(t, err, "a named provider must never fall back to allowing every request")
			assert.ErrorContains(t, err, testCase.provider)
			if testCase.wantIs != nil {
				assert.ErrorIs(t, err, testCase.wantIs)
			}
			assert.Nil(t, limiter)

			service, err := container.GetRateLimitService()
			require.Error(t, err, "the rate limit service fails so startup fails")
			assert.Nil(t, service)
		})
	}
}

func newRateLimiterTestContainer(t *testing.T, provider string, cacheErr error) *Container {
	t.Helper()

	container := NewContainer()
	container.AddCacheProvider(sharedCacheProviderName, cache_provider_mock.NewMockProvider())
	container.SetCacheDefaultProvider(sharedCacheProviderName)
	container.SetRateLimiterCacheProvider(provider)
	if cacheErr != nil {
		container.cacheOnce.Do(func() {})
		container.cacheErr = cacheErr
	}
	return container
}

func TestNewCacheRateLimiter(t *testing.T) {
	testCases := []struct {
		name     string
		provider string
		wantErr  bool
	}{
		{name: "in-memory stores", provider: ""},
		{name: "stores on a named provider", provider: sharedCacheProviderName},
		{name: "unknown provider", provider: "missing", wantErr: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			service := cache_domain.NewService(sharedCacheProviderName)
			require.NoError(t, service.RegisterProvider(t.Context(), sharedCacheProviderName, cache_provider_mock.NewMockProvider()))
			t.Cleanup(func() { _ = service.Close(context.Background()) })

			limiter, closeStores, err := newCacheRateLimiter(t.Context(), service, testCase.provider)
			if testCase.wantErr {
				require.ErrorContains(t, err, "creating token bucket store")
				assert.Nil(t, limiter)
				assert.Nil(t, closeStores)
				return
			}
			require.NoError(t, err)

			require.NoError(t, limiter.AllowTokenBucket(t.Context(), "key", 1,
				ratelimiter_dto.TokenBucketConfig{Rate: 1, Burst: 1}))
			require.NoError(t, closeStores(t.Context()))
		})
	}
}

func TestCloseAll(t *testing.T) {
	t.Parallel()

	firstErr := errors.New("first close failed")
	secondErr := errors.New("second close failed")
	healthy := &fakeContextCloser{err: nil}
	first := &fakeContextCloser{err: firstErr}
	second := &fakeContextCloser{err: secondErr}

	err := closeAll(t.Context(), first, healthy, second)

	require.ErrorIs(t, err, firstErr)
	require.ErrorIs(t, err, secondErr)
	assert.True(t, first.closed && healthy.closed && second.closed, "every closer is closed even after a failure")
	require.NoError(t, closeAll(t.Context(), healthy))
}

type fakeContextCloser struct {
	err    error
	closed bool
}

func (f *fakeContextCloser) Close(context.Context) error {
	f.closed = true
	return f.err
}
