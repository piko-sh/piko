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

package llm_provider_grok

import (
	"context"
	"fmt"
	"strings"
	"time"

	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/llm/llm_provider_openai"
	"piko.sh/piko/wdk/logger"
)

// grokProvider implements llm_domain.LLMProviderPort for xAI Grok.
// It delegates OpenAI-compatible wire protocol handling to an internal
// OpenAI provider instance and overrides provider-specific concerns
// (model filtering, error wrapping, observability).
type grokProvider struct {
	// inner is the OpenAI provider configured with the Grok base URL.
	inner llm_domain.LLMProviderPort

	// config holds the Grok provider configuration.
	config Config
}

var _ llm_domain.LLMProviderPort = (*grokProvider)(nil)

// Complete sends a completion request to Grok.
//
// Takes request (*llm_dto.CompletionRequest) which specifies the prompt and model
// settings.
//
// Returns *llm_dto.CompletionResponse which contains the generated completion.
// Returns error when the API request fails.
func (p *grokProvider) Complete(ctx context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
	_, l := logger.From(ctx, log)
	completeCount.Add(ctx, 1)
	start := time.Now()

	defer func() {
		completeDuration.Record(ctx, float64(time.Since(start).Milliseconds()))
	}()

	l.Debug("Sending Grok completion request",
		logger.String("model", request.Model),
		logger.Int("message_count", len(request.Messages)),
	)

	response, err := p.inner.Complete(ctx, request)
	if err != nil {
		completeErrorCount.Add(ctx, 1)
		return nil, rewrapError(err)
	}

	return response, nil
}

// Stream starts a streaming completion request to Grok.
//
// Takes request (*llm_dto.CompletionRequest) which specifies the prompt and model
// settings.
//
// Returns <-chan llm_dto.StreamEvent which delivers incremental response
// chunks.
// Returns error when the streaming connection cannot be established.
func (p *grokProvider) Stream(ctx context.Context, request *llm_dto.CompletionRequest) (<-chan llm_dto.StreamEvent, error) {
	_, l := logger.From(ctx, log)
	streamCount.Add(ctx, 1)

	l.Debug("Starting Grok streaming completion",
		logger.String("model", request.Model),
		logger.Int("message_count", len(request.Messages)),
	)

	eventChannel, err := p.inner.Stream(ctx, request)
	if err != nil {
		streamErrorCount.Add(ctx, 1)
		return nil, rewrapError(err)
	}

	return eventChannel, nil
}

// SupportsStreaming reports whether the provider supports streaming.
//
// Returns bool which is true if streaming is supported.
func (*grokProvider) SupportsStreaming() bool {
	return true
}

// SupportsStructuredOutput reports whether the provider supports structured
// output.
//
// Returns bool which is true if structured output is supported.
func (*grokProvider) SupportsStructuredOutput() bool {
	return true
}

// SupportsTools reports whether the provider supports tool calling.
//
// Returns bool which is true if tool calling is supported.
func (*grokProvider) SupportsTools() bool {
	return true
}

// SupportsPenalties reports whether the provider supports frequency and
// presence penalties.
//
// Returns bool which is true if penalties are supported.
func (*grokProvider) SupportsPenalties() bool { return true }

// SupportsSeed reports whether the provider supports deterministic seed.
//
// Returns bool which is true if seed is supported.
func (*grokProvider) SupportsSeed() bool { return true }

// SupportsParallelToolCalls reports whether the provider supports parallel
// tool calls.
//
// Returns bool which is true if parallel tool calls are supported.
func (*grokProvider) SupportsParallelToolCalls() bool { return true }

// SupportsMessageName reports whether the provider supports the name field
// on messages.
//
// Returns bool which is true if message names are supported.
func (*grokProvider) SupportsMessageName() bool { return true }

// DefaultModel returns the provider's default model identifier.
//
// Returns string which is the default model name.
func (p *grokProvider) DefaultModel() string {
	return p.config.DefaultModel
}

// Close releases any resources held by the provider.
//
// Returns error if the underlying client fails to close.
func (p *grokProvider) Close(ctx context.Context) error {
	return p.inner.Close(ctx)
}

// ListModels returns available Grok models.
//
// Returns []llm_dto.ModelInfo which contains the known Grok chat models.
// Returns error which is always nil for this static implementation.
func (*grokProvider) ListModels(_ context.Context) ([]llm_dto.ModelInfo, error) {
	return grokModels(), nil
}

// New creates a new Grok provider with the given settings.
//
// Takes config (Config) which contains the provider settings.
//
// Returns llm_domain.LLMProviderPort which is the configured provider.
// Returns error when the configuration is not valid.
func New(config Config) (llm_domain.LLMProviderPort, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	config = config.WithDefaults()

	inner, err := llm_provider_openai.New(llm_provider_openai.Config{
		APIKey:       config.APIKey,
		BaseURL:      config.BaseURL,
		DefaultModel: config.DefaultModel,
	})
	if err != nil {
		return nil, fmt.Errorf("grok: failed to initialise OpenAI-compatible client: %w", err)
	}

	return &grokProvider{
		inner:  inner,
		config: config,
	}, nil
}

// grokModels returns the known Grok chat models.
//
// Returns []llm_dto.ModelInfo which contains the statically
// defined Grok model list.
func grokModels() []llm_dto.ModelInfo {
	ids := []string{
		"grok-3",
		"grok-3-mini",
		"grok-4-0709",
		"grok-4-fast-reasoning",
		"grok-4-fast-non-reasoning",
		"grok-4-1-fast-reasoning",
		"grok-4-1-fast-non-reasoning",
		"grok-code-fast-1",
	}

	result := make([]llm_dto.ModelInfo, len(ids))
	for i, id := range ids {
		result[i] = llm_dto.ModelInfo{
			ID:                       id,
			Name:                     id,
			Provider:                 "grok",
			SupportsStreaming:        true,
			SupportsTools:            true,
			SupportsStructuredOutput: true,
		}
	}
	return result
}

// isGrokChatModel checks whether a model ID belongs to the Grok family.
//
// Takes id (string) which is the model identifier to check.
//
// Returns bool which is true when the model ID starts with "grok-".
func isGrokChatModel(id string) bool {
	return strings.HasPrefix(id, "grok-")
}
