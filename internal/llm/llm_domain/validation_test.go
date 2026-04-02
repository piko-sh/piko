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

package llm_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/llm/llm_dto"
)

func TestValidateRequest(t *testing.T) {
	testCases := []struct {
		wantErr error
		request *llm_dto.CompletionRequest
		name    string
	}{
		{
			name: "valid request",
			request: &llm_dto.CompletionRequest{
				Model: "gpt-4o",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleUser, Content: "Hello"},
				},
			},
			wantErr: nil,
		},
		{
			name: "empty model",
			request: &llm_dto.CompletionRequest{
				Model: "",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleUser, Content: "Hello"},
				},
			},
			wantErr: ErrEmptyModel,
		},
		{
			name: "empty messages",
			request: &llm_dto.CompletionRequest{
				Model:    "gpt-4o",
				Messages: []llm_dto.Message{},
			},
			wantErr: ErrEmptyMessages,
		},
		{
			name: "nil messages",
			request: &llm_dto.CompletionRequest{
				Model:    "gpt-4o",
				Messages: nil,
			},
			wantErr: ErrEmptyMessages,
		},
		{
			name: "temperature below zero",
			request: &llm_dto.CompletionRequest{
				Model: "gpt-4o",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleUser, Content: "Hello"},
				},
				Temperature: new(-0.1),
			},
			wantErr: ErrInvalidTemperature,
		},
		{
			name: "temperature above two",
			request: &llm_dto.CompletionRequest{
				Model: "gpt-4o",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleUser, Content: "Hello"},
				},
				Temperature: new(2.1),
			},
			wantErr: ErrInvalidTemperature,
		},
		{
			name: "temperature at zero is valid",
			request: &llm_dto.CompletionRequest{
				Model: "gpt-4o",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleUser, Content: "Hello"},
				},
				Temperature: new(float64(0)),
			},
			wantErr: nil,
		},
		{
			name: "temperature at two is valid",
			request: &llm_dto.CompletionRequest{
				Model: "gpt-4o",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleUser, Content: "Hello"},
				},
				Temperature: new(float64(2)),
			},
			wantErr: nil,
		},
		{
			name: "top_p below zero",
			request: &llm_dto.CompletionRequest{
				Model: "gpt-4o",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleUser, Content: "Hello"},
				},
				TopP: new(-0.1),
			},
			wantErr: ErrInvalidTopP,
		},
		{
			name: "top_p above one",
			request: &llm_dto.CompletionRequest{
				Model: "gpt-4o",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleUser, Content: "Hello"},
				},
				TopP: new(1.1),
			},
			wantErr: ErrInvalidTopP,
		},
		{
			name: "top_p at zero is valid",
			request: &llm_dto.CompletionRequest{
				Model: "gpt-4o",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleUser, Content: "Hello"},
				},
				TopP: new(float64(0)),
			},
			wantErr: nil,
		},
		{
			name: "top_p at one is valid",
			request: &llm_dto.CompletionRequest{
				Model: "gpt-4o",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleUser, Content: "Hello"},
				},
				TopP: new(float64(1)),
			},
			wantErr: nil,
		},
		{
			name: "max_tokens zero",
			request: &llm_dto.CompletionRequest{
				Model: "gpt-4o",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleUser, Content: "Hello"},
				},
				MaxTokens: new(0),
			},
			wantErr: ErrInvalidMaxTokens,
		},
		{
			name: "max_tokens negative",
			request: &llm_dto.CompletionRequest{
				Model: "gpt-4o",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleUser, Content: "Hello"},
				},
				MaxTokens: new(-1),
			},
			wantErr: ErrInvalidMaxTokens,
		},
		{
			name: "max_tokens positive is valid",
			request: &llm_dto.CompletionRequest{
				Model: "gpt-4o",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleUser, Content: "Hello"},
				},
				MaxTokens: new(100),
			},
			wantErr: nil,
		},
		{
			name: "valid request with all parameters",
			request: &llm_dto.CompletionRequest{
				Model: "gpt-4o",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleSystem, Content: "You are helpful"},
					{Role: llm_dto.RoleUser, Content: "Hello"},
				},
				Temperature: new(0.7),
				TopP:        new(0.9),
				MaxTokens:   new(1000),
			},
			wantErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateRequest(tc.request)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRequestForProvider(t *testing.T) {
	testCases := []struct {
		wantErr                 error
		request                 *llm_dto.CompletionRequest
		name                    string
		supportsStreaming       bool
		supportsStructuredValue bool
		supportsToolsValue      bool
	}{
		{
			name: "valid basic request",
			request: &llm_dto.CompletionRequest{
				Model: "gpt-4o",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleUser, Content: "Hello"},
				},
			},
			supportsStreaming:       true,
			supportsStructuredValue: true,
			supportsToolsValue:      true,
			wantErr:                 nil,
		},
		{
			name: "validates basic request first",
			request: &llm_dto.CompletionRequest{
				Model:    "",
				Messages: []llm_dto.Message{},
			},
			supportsStreaming:       true,
			supportsStructuredValue: true,
			supportsToolsValue:      true,
			wantErr:                 ErrEmptyModel,
		},
		{
			name: "streaming requested but not supported",
			request: &llm_dto.CompletionRequest{
				Model: "gpt-4o",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleUser, Content: "Hello"},
				},
				Stream: true,
			},
			supportsStreaming:       false,
			supportsStructuredValue: true,
			supportsToolsValue:      true,
			wantErr:                 ErrStreamingNotSupported,
		},
		{
			name: "streaming requested and supported",
			request: &llm_dto.CompletionRequest{
				Model: "gpt-4o",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleUser, Content: "Hello"},
				},
				Stream: true,
			},
			supportsStreaming:       true,
			supportsStructuredValue: true,
			supportsToolsValue:      true,
			wantErr:                 nil,
		},
		{
			name: "tools requested but not supported",
			request: &llm_dto.CompletionRequest{
				Model: "gpt-4o",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleUser, Content: "Hello"},
				},
				Tools: []llm_dto.ToolDefinition{
					{Type: "function", Function: llm_dto.FunctionDefinition{Name: "test"}},
				},
			},
			supportsStreaming:       true,
			supportsStructuredValue: true,
			supportsToolsValue:      false,
			wantErr:                 ErrToolsNotSupported,
		},
		{
			name: "tools requested and supported",
			request: &llm_dto.CompletionRequest{
				Model: "gpt-4o",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleUser, Content: "Hello"},
				},
				Tools: []llm_dto.ToolDefinition{
					{Type: "function", Function: llm_dto.FunctionDefinition{Name: "test"}},
				},
			},
			supportsStreaming:       true,
			supportsStructuredValue: true,
			supportsToolsValue:      true,
			wantErr:                 nil,
		},
		{
			name: "structured output requested but not supported",
			request: &llm_dto.CompletionRequest{
				Model: "gpt-4o",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleUser, Content: "Hello"},
				},
				ResponseFormat: &llm_dto.ResponseFormat{
					Type: llm_dto.ResponseFormatJSONSchema,
				},
			},
			supportsStreaming:       true,
			supportsStructuredValue: false,
			supportsToolsValue:      true,
			wantErr:                 ErrStructuredOutputNotSupported,
		},
		{
			name: "structured output requested and supported",
			request: &llm_dto.CompletionRequest{
				Model: "gpt-4o",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleUser, Content: "Hello"},
				},
				ResponseFormat: &llm_dto.ResponseFormat{
					Type: llm_dto.ResponseFormatJSONSchema,
				},
			},
			supportsStreaming:       true,
			supportsStructuredValue: true,
			supportsToolsValue:      true,
			wantErr:                 nil,
		},
		{
			name: "json object response format does not require structured output support",
			request: &llm_dto.CompletionRequest{
				Model: "gpt-4o",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleUser, Content: "Hello"},
				},
				ResponseFormat: &llm_dto.ResponseFormat{
					Type: llm_dto.ResponseFormatJSONObject,
				},
			},
			supportsStreaming:       true,
			supportsStructuredValue: false,
			supportsToolsValue:      true,
			wantErr:                 nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provider := NewMockLLMProvider()
			provider.SupportsStreamingValue = tc.supportsStreaming
			provider.SupportsStructuredValue = tc.supportsStructuredValue
			provider.SupportsToolsValue = tc.supportsToolsValue

			err := ValidateRequestForProvider(tc.request, provider)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRequestForProvider_Penalties(t *testing.T) {
	t.Run("frequency penalty not supported", func(t *testing.T) {
		provider := NewMockLLMProvider()
		provider.SupportsPenaltiesValue = false

		request := &llm_dto.CompletionRequest{
			Model: "test-model",
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
			FrequencyPenalty: new(0.5),
		}

		err := ValidateRequestForProvider(request, provider)

		assert.ErrorIs(t, err, ErrPenaltiesNotSupported)
	})

	t.Run("frequency penalty supported", func(t *testing.T) {
		provider := NewMockLLMProvider()
		provider.SupportsPenaltiesValue = true

		request := &llm_dto.CompletionRequest{
			Model: "test-model",
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
			FrequencyPenalty: new(0.5),
		}

		err := ValidateRequestForProvider(request, provider)

		assert.NoError(t, err)
	})

	t.Run("presence penalty not supported", func(t *testing.T) {
		provider := NewMockLLMProvider()
		provider.SupportsPenaltiesValue = false

		request := &llm_dto.CompletionRequest{
			Model: "test-model",
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
			PresencePenalty: new(0.5),
		}

		err := ValidateRequestForProvider(request, provider)

		assert.ErrorIs(t, err, ErrPenaltiesNotSupported)
	})
}

func TestValidateRequestForProvider_Seed(t *testing.T) {
	t.Run("seed not supported", func(t *testing.T) {
		provider := NewMockLLMProvider()
		provider.SupportsSeedValue = false

		request := &llm_dto.CompletionRequest{
			Model: "test-model",
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
			Seed: new(int64(42)),
		}

		err := ValidateRequestForProvider(request, provider)

		assert.ErrorIs(t, err, ErrSeedNotSupported)
	})

	t.Run("seed supported", func(t *testing.T) {
		provider := NewMockLLMProvider()
		provider.SupportsSeedValue = true

		request := &llm_dto.CompletionRequest{
			Model: "test-model",
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
			Seed: new(int64(42)),
		}

		err := ValidateRequestForProvider(request, provider)

		assert.NoError(t, err)
	})
}

func TestValidateRequestForProvider_ParallelToolCalls(t *testing.T) {
	t.Run("parallel tool calls not supported", func(t *testing.T) {
		provider := NewMockLLMProvider()
		provider.SupportsParallelToolCallsValue = false

		request := &llm_dto.CompletionRequest{
			Model: "test-model",
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
			ParallelToolCalls: new(true),
		}

		err := ValidateRequestForProvider(request, provider)

		assert.ErrorIs(t, err, ErrParallelToolCallsNotSupported)
	})

	t.Run("parallel tool calls supported", func(t *testing.T) {
		provider := NewMockLLMProvider()
		provider.SupportsParallelToolCallsValue = true

		request := &llm_dto.CompletionRequest{
			Model: "test-model",
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
			ParallelToolCalls: new(true),
		}

		err := ValidateRequestForProvider(request, provider)

		assert.NoError(t, err)
	})
}

func TestValidateRequestForProvider_MessageName(t *testing.T) {
	t.Run("message name not supported", func(t *testing.T) {
		provider := NewMockLLMProvider()
		provider.SupportsMessageNameValue = false

		request := &llm_dto.CompletionRequest{
			Model: "test-model",
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello", Name: new("Alice")},
			},
		}

		err := ValidateRequestForProvider(request, provider)

		assert.ErrorIs(t, err, ErrMessageNameNotSupported)
	})

	t.Run("message name supported", func(t *testing.T) {
		provider := NewMockLLMProvider()
		provider.SupportsMessageNameValue = true

		request := &llm_dto.CompletionRequest{
			Model: "test-model",
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello", Name: new("Alice")},
			},
		}

		err := ValidateRequestForProvider(request, provider)

		assert.NoError(t, err)
	})

	t.Run("message name not supported but no names in messages", func(t *testing.T) {
		provider := NewMockLLMProvider()
		provider.SupportsMessageNameValue = false

		request := &llm_dto.CompletionRequest{
			Model: "test-model",
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
		}

		err := ValidateRequestForProvider(request, provider)

		assert.NoError(t, err)
	})
}
