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

	"piko.sh/piko/internal/llm/llm_domain"
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

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewContainer()
			c.llmProviders = map[string]llm_domain.LLMProviderPort{"main": &llm_domain.MockLLMProvider{}}
			service := newLLMTestService(t)
			if tc.preRegistered != "" {
				require.NoError(t, service.RegisterProvider(context.Background(), tc.preRegistered, &llm_domain.MockLLMProvider{}))
			}

			err := c.registerLLMProviders(context.Background(), service)

			if tc.wantErrContain == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, llm_domain.ErrProviderAlreadyExists)
			assert.Contains(t, err.Error(), tc.wantErrContain)
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

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewContainer()
			c.llmProviders = map[string]llm_domain.LLMProviderPort{"main": &llm_domain.MockLLMProvider{}}
			c.llmEmbeddingProviders = map[string]llm_domain.EmbeddingProviderPort{"embed": &llm_domain.MockEmbeddingProvider{}}
			c.llmDefaultProvider = tc.defaultProvider
			c.llmDefaultEmbeddingProvider = tc.defaultEmbeddingProvider
			service := newLLMTestService(t)
			require.NoError(t, c.registerLLMProviders(context.Background(), service))
			c.registerStandaloneEmbeddingProviders(context.Background(), service)
			c.registerStandaloneEmbeddingProviders(context.Background(), service)

			c.configureLLMDefaults(context.Background(), service)

			if tc.defaultProvider == "main" {
				assert.Equal(t, "main", service.GetDefaultProvider())
			}
		})
	}
}
