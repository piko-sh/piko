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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cache_provider_mock "piko.sh/piko/internal/cache/cache_adapters/provider_mock"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
)

func newLLMTestService(t *testing.T) llm_domain.Service {
	t.Helper()

	service := llm_domain.NewService("")
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	return service
}

func TestContainer_RegisterLLMProviders(t *testing.T) {
	testCases := []struct {
		name           string
		preRegistered  string
		wantErrContain string
	}{
		{name: "registers every provider"},
		{name: "reports the provider that failed", preRegistered: "main", wantErrContain: `registering LLM provider "main"`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			container := NewContainer()
			container.llmProviders = map[string]llm_domain.LLMProviderPort{"main": &llm_domain.MockLLMProvider{}}
			service := newLLMTestService(t)
			if testCase.preRegistered != "" {
				require.NoError(t, service.RegisterProvider(t.Context(), testCase.preRegistered, &llm_domain.MockLLMProvider{}))
			}

			err := container.registerLLMProviders(t.Context(), service)

			if testCase.wantErrContain == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, llm_domain.ErrProviderAlreadyExists)
			assert.Contains(t, err.Error(), testCase.wantErrContain)
		})
	}
}

func TestContainer_ConfigureLLMDefaults(t *testing.T) {
	testCases := []struct {
		name                     string
		defaultProvider          string
		defaultEmbeddingProvider string
	}{
		{name: "no defaults configured"},
		{name: "a missing default provider is reported", defaultProvider: "missing"},
		{name: "an existing default provider is selected", defaultProvider: "main"},
		{name: "a missing default embedding provider is reported", defaultEmbeddingProvider: "missing"},
		{name: "an existing default embedding provider is selected", defaultEmbeddingProvider: "embed"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			container := NewContainer()
			container.llmProviders = map[string]llm_domain.LLMProviderPort{"main": &llm_domain.MockLLMProvider{}}
			container.llmEmbeddingProviders = map[string]llm_domain.EmbeddingProviderPort{"embed": &llm_domain.MockEmbeddingProvider{}}
			container.llmDefaultProvider = testCase.defaultProvider
			container.llmDefaultEmbeddingProvider = testCase.defaultEmbeddingProvider
			service := newLLMTestService(t)
			require.NoError(t, container.registerLLMProviders(t.Context(), service))
			container.registerStandaloneEmbeddingProviders(t.Context(), service)
			container.registerStandaloneEmbeddingProviders(t.Context(), service)

			container.configureLLMDefaults(t.Context(), service)

			if testCase.defaultProvider == "main" {
				assert.Equal(t, "main", service.GetDefaultProvider())
			}
		})
	}
}

func TestContainer_ConfigureLLMCache(t *testing.T) {
	testCases := []struct {
		cacheErr     error
		name         string
		provider     string
		wantErr      string
		wantManagers bool
	}{
		{
			name:         "stores entries on the named provider",
			provider:     sharedCacheProviderName,
			wantManagers: true,
		},
		{
			name:     "reports an unknown provider",
			provider: "missing",
			wantErr:  "creating LLM",
		},
		{
			name:     "reports a missing cache service",
			provider: sharedCacheProviderName,
			cacheErr: errors.New("cache unavailable"),
			wantErr:  "getting cache service",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			container := NewContainer(WithLLMCacheMaxBytes(64 << 20))
			container.AddCacheProvider(sharedCacheProviderName, cache_provider_mock.NewMockProvider())
			container.SetCacheDefaultProvider(sharedCacheProviderName)
			container.SetLLMCacheProvider(testCase.provider)
			if testCase.cacheErr != nil {
				container.cacheOnce.Do(func() {})
				container.cacheErr = testCase.cacheErr
			}
			service := newLLMTestService(t)

			cacheErr := container.configureLLMCache(service)
			budgetErr := container.configureLLMBudget(service)

			if testCase.wantErr != "" {
				require.ErrorContains(t, cacheErr, testCase.wantErr)
				require.ErrorContains(t, budgetErr, testCase.wantErr)
				assert.Nil(t, service.GetCacheManager())
				assert.Nil(t, service.GetBudgetManager())
				return
			}
			require.NoError(t, cacheErr)
			require.NoError(t, budgetErr)
			assert.Equal(t, uint64(64<<20), container.llmCacheMaxBytesOverride)
			require.NotNil(t, service.GetCacheManager())
			require.NotNil(t, service.GetBudgetManager())

			key := "cache-key"
			entry := &llm_dto.CacheEntry{Response: &llm_dto.CompletionResponse{ID: "cached"}}
			require.NoError(t, service.GetCacheManager().Set(t.Context(), key, entry))
		})
	}
}

func TestContainer_GetLLMService_WithSharedCacheProvider(t *testing.T) {
	container := NewContainer()
	container.AddCacheProvider(sharedCacheProviderName, cache_provider_mock.NewMockProvider())
	container.SetCacheDefaultProvider(sharedCacheProviderName)
	container.SetLLMCacheProvider(sharedCacheProviderName)
	container.AddLLMProvider("main", &llm_domain.MockLLMProvider{})
	container.SetLLMDefaultProvider("main")

	service, err := container.GetLLMService()

	require.NoError(t, err)
	require.NotNil(t, service)
	assert.NotNil(t, service.GetCacheManager())
	assert.NotNil(t, service.GetBudgetManager())
}

func TestContainer_LLMSetters(t *testing.T) {
	container := NewContainer()
	embedding := &llm_domain.MockEmbeddingProvider{}
	override := newLLMTestService(t)

	container.AddEmbeddingProvider("embed", embedding)
	container.SetDefaultEmbeddingProvider("embed")
	container.SetLLMService(override)

	assert.Same(t, embedding, container.llmEmbeddingProviders["embed"])
	assert.Equal(t, "embed", container.llmDefaultEmbeddingProvider)

	service, err := container.GetLLMService()
	require.NoError(t, err)
	assert.Same(t, override, service, "the override replaces the default service")
}
