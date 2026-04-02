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

package llm_provider_ollama

import (
	"context"
	"fmt"
)

// OllamaProvider wraps the internal Ollama provider and exposes
// Ollama-specific methods such as [EnsureModels]. It implements
// llm.ProviderPort and llm.EmbeddingProviderPort via embedding.
type OllamaProvider struct {
	*ollamaProvider
}

// NewOllamaProvider creates a new Ollama LLM provider with the given
// configuration.
//
// The returned provider implements both llm.ProviderPort (completions) and
// llm.EmbeddingProviderPort (embeddings). When registered via
// piko.WithLLMProvider, the framework automatically detects and registers the
// embedding capability.
//
// Takes config (Config) which contains the provider configuration.
//
// Returns *OllamaProvider which can be registered with the LLM service.
// Returns error when the configuration is invalid or Ollama cannot be reached.
func NewOllamaProvider(config Config) (*OllamaProvider, error) {
	p, err := newProvider(config)
	if err != nil {
		return nil, err
	}
	return &OllamaProvider{ollamaProvider: p}, nil
}

// EnsureModels downloads the configured default completion and embedding
// models if they are not already available locally. Call this at application
// startup to avoid first-request latency from on-demand model pulls.
//
// After ensuring the embedding model, this also queries its vector dimension
// so that [EmbeddingDimensions] returns the correct value immediately.
//
// Returns error when a model cannot be pulled or verified.
func (p *OllamaProvider) EnsureModels(ctx context.Context) error {
	if err := p.ensureModel(ctx, p.defaultModel.Name, p.defaultModel); err != nil {
		return fmt.Errorf("ensuring completion model %q: %w", p.defaultModel.Name, err)
	}
	if err := p.ensureModel(ctx, p.defaultEmbeddingModel.Name, p.defaultEmbeddingModel); err != nil {
		return fmt.Errorf("ensuring embedding model %q: %w", p.defaultEmbeddingModel.Name, err)
	}

	p.tryPopulateEmbeddingDim(ctx, p.defaultEmbeddingModel.Name)

	return nil
}
