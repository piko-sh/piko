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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cache_provider_mock "piko.sh/piko/internal/cache/cache_adapters/provider_mock"
	"piko.sh/piko/internal/cache/cache_domain"
	email_mock "piko.sh/piko/internal/email/email_adapters/provider_mock"
	storage_mock "piko.sh/piko/internal/storage/storage_adapters/provider_mock"
)

func TestValidateProviderConfiguration_NoProvidersNoDefaults(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	err := c.ValidateProviderConfiguration()
	assert.NoError(t, err, "expected no error with empty configuration")
}

func TestValidateProviderConfiguration_EmailDefaultWithoutProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer()
	c.SetEmailDefaultProvider("missing-provider")

	err := c.ValidateProviderConfiguration()
	require.Error(t, err, "expected error when default provider is not registered")
	assert.Contains(t, err.Error(), "email default provider", "error should mention email provider")
	assert.Contains(t, err.Error(), "missing-provider", "error should mention the missing provider name")
}

func TestValidateProviderConfiguration_EmailDefaultWithProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer()
	c.AddEmailProvider("my-email", email_mock.NewMockEmailProvider())
	c.SetEmailDefaultProvider("my-email")

	err := c.ValidateProviderConfiguration()
	assert.NoError(t, err, "expected no error when default matches registered provider")
}

func TestValidateProviderConfiguration_EmailProvidersWithoutDefault(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		providers []string
		name      string
		wantError bool
	}{
		{name: "single provider needs no default", providers: []string{"outbox"}, wantError: false},
		{name: "provider named default wins", providers: []string{"default", "archive"}, wantError: false},
		{name: "several providers need a default", providers: []string{"outbox", "archive"}, wantError: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			c := NewContainer()
			for _, name := range testCase.providers {
				c.AddEmailProvider(name, email_mock.NewMockEmailProvider())
			}

			err := c.ValidateProviderConfiguration()
			if !testCase.wantError {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), errEmailDefaultProviderRequired.Error())
			assert.Contains(t, err.Error(), "archive, outbox")
		})
	}
}

func TestValidateProviderConfiguration_EmailProvidersWithServiceOverride(t *testing.T) {
	t.Parallel()
	c := NewContainer()
	c.AddEmailProvider("outbox", email_mock.NewMockEmailProvider())
	c.AddEmailProvider("archive", email_mock.NewMockEmailProvider())
	c.emailServiceOverride = &stubEmailService{}

	require.NoError(t, c.ValidateProviderConfiguration())
}

func TestValidateProviderConfiguration_StorageDefaultWithoutProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer()
	c.SetStorageDefaultProvider("missing-storage")

	err := c.ValidateProviderConfiguration()
	require.Error(t, err, "expected error when default provider is not registered")
	assert.Contains(t, err.Error(), "storage default provider", "error should mention storage provider")
}

func TestValidateProviderConfiguration_StorageDefaultWithProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer()
	c.AddStorageProvider("my-storage", storage_mock.NewMockStorageProvider())
	c.SetStorageDefaultProvider("my-storage")

	err := c.ValidateProviderConfiguration()
	assert.NoError(t, err, "expected no error when default matches registered provider")
}

func TestValidateProviderConfiguration_CacheDefaultWithoutProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer()
	c.SetCacheDefaultProvider("missing-cache")

	err := c.ValidateProviderConfiguration()
	require.Error(t, err, "expected error when default provider is not registered")
	assert.Contains(t, err.Error(), "cache default provider", "error should mention cache provider")
}

func TestValidateProviderConfiguration_CryptoDefaultWithoutProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer()
	c.SetCryptoDefaultProvider("missing-crypto")

	err := c.ValidateProviderConfiguration()
	require.Error(t, err, "expected error when default provider is not registered")
	assert.Contains(t, err.Error(), "crypto default provider", "error should mention crypto provider")
}

func TestValidateProviderConfiguration_NotificationDefaultWithoutProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer()
	c.SetNotificationDefaultProvider("missing-notification")

	err := c.ValidateProviderConfiguration()
	require.Error(t, err, "expected error when default provider is not registered")
	assert.Contains(t, err.Error(), "notification default provider", "error should mention notification provider")
}

func TestValidateProviderConfiguration_MultipleErrors(t *testing.T) {
	t.Parallel()
	c := NewContainer()
	c.SetEmailDefaultProvider("missing-email")
	c.SetStorageDefaultProvider("missing-storage")

	err := c.ValidateProviderConfiguration()
	require.Error(t, err, "expected error when multiple defaults are missing")
	assert.Contains(t, err.Error(), "email default provider", "error should mention email provider")
	assert.Contains(t, err.Error(), "storage default provider", "error should mention storage provider")
}

func TestValidateProviderConfiguration_DefaultPointsToWrongProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer()
	c.AddEmailProvider("provider-a", email_mock.NewMockEmailProvider())
	c.SetEmailDefaultProvider("provider-b")

	err := c.ValidateProviderConfiguration()
	require.Error(t, err, "expected error when default points to unregistered provider")
	assert.Contains(t, err.Error(), "provider-b", "error should mention the missing provider name")
	assert.Contains(t, err.Error(), "provider-a", "error should list available providers")
}

func TestValidateProviderConfiguration_NamedCacheProviders(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		registered      string
		rateLimiterName string
		llmName         string
		wantErr         string
	}{
		{name: "no names set"},
		{name: "built-in otter without registered providers", rateLimiterName: "otter", llmName: "otter"},
		{name: "registered provider", registered: "redis", rateLimiterName: "redis", llmName: "redis"},
		{name: "rate limiter provider without registered providers", rateLimiterName: "redis", wantErr: `rate limiter cache provider "redis" specified but no cache providers registered`},
		{name: "unregistered LLM provider", registered: "redis", llmName: "valkey", wantErr: `LLM cache provider "valkey" not found in registered cache providers: redis`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			container := NewContainer()
			if testCase.registered != "" {
				container.cacheProviders = map[string]cache_domain.Provider{testCase.registered: cache_provider_mock.NewMockProvider()}
			}
			container.SetRateLimiterCacheProvider(testCase.rateLimiterName)
			container.SetLLMCacheProvider(testCase.llmName)

			err := container.ValidateProviderConfiguration()

			if testCase.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, testCase.wantErr)
		})
	}
}

func TestValidateProviderConfiguration_NamedCacheProvidersOnSuppliedService(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		rateLimiterName string
		llmName         string
		wantErr         string
	}{
		{name: "provider registered on the supplied service", rateLimiterName: "redis", llmName: "redis"},
		{name: "provider missing from the supplied service", llmName: "valkey", wantErr: `LLM cache provider "valkey" not found in registered cache providers: redis`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := cache_domain.NewService("redis")
			require.NoError(t, service.RegisterProvider(t.Context(), "redis", cache_provider_mock.NewMockProvider()))
			t.Cleanup(func() { _ = service.Close(context.Background()) })

			container := NewContainer()
			container.SetCacheService(service)
			container.SetRateLimiterCacheProvider(testCase.rateLimiterName)
			container.SetLLMCacheProvider(testCase.llmName)

			err := container.ValidateProviderConfiguration()

			if testCase.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, testCase.wantErr)
		})
	}
}

func TestValidateProviderConfiguration_MediaAndLLMDefaultsWithoutProviders(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		configure func(c *Container)
		name      string
		wantText  string
	}{
		{
			name:      "image transformer",
			configure: func(c *Container) { c.defaultImageTransformer = "imgproxy" },
			wantText:  "image default transformer",
		},
		{
			name:      "video transcoder",
			configure: func(c *Container) { c.defaultVideoTranscoder = "ffmpeg" },
			wantText:  "video default transcoder",
		},
		{
			name:      "llm provider",
			configure: func(c *Container) { c.llmDefaultProvider = "openai" },
			wantText:  "llm default provider",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			c := NewContainer()
			testCase.configure(c)

			err := c.ValidateProviderConfiguration()

			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.wantText)
			assert.Contains(t, err.Error(), "specified but no")
		})
	}
}
