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
	"context"
	"fmt"
	"sync"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/clock"
)

// EmbeddingProviderPort defines the interface that embedding provider adapters
// must implement. It is a driven port in the hexagonal architecture pattern.
type EmbeddingProviderPort interface {
	// Embed generates embeddings for the given input texts.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	// Takes request (*llm_dto.EmbeddingRequest) which contains the embedding
	// parameters.
	//
	// Returns *llm_dto.EmbeddingResponse containing the generated embeddings.
	// Returns error when the request fails.
	Embed(ctx context.Context, request *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error)

	// ListEmbeddingModels returns information about available embedding models.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	//
	// Returns []llm_dto.ModelInfo which contains model metadata.
	// Returns error when the model list cannot be retrieved.
	ListEmbeddingModels(ctx context.Context) ([]llm_dto.ModelInfo, error)

	// EmbeddingDimensions returns the default vector dimension produced by this
	// provider's embedding model.
	//
	// Returns int which is the vector dimension, or 0 if the dimension is not
	// known statically (e.g. server-determined models).
	EmbeddingDimensions() int

	// Close releases any resources held by the provider.
	//
	// Returns error when the provider cannot be closed cleanly.
	Close(ctx context.Context) error
}

// embeddingService provides text embedding through configurable providers.
type embeddingService struct {
	// clock provides time operations for measuring embedding duration.
	clock clock.Clock

	// providers maps provider names to their implementations.
	providers map[string]EmbeddingProviderPort

	// defaultProvider is the name of the embedding provider to use when none is
	// specified; empty means no default is set.
	defaultProvider string

	// mu guards access to the providers map and defaultProvider field.
	mu sync.RWMutex
}

// embeddingServiceOption is a functional option for configuring an
// embeddingService.
type embeddingServiceOption func(*embeddingService)

// RegisterEmbeddingProvider adds an embedding provider to the registry.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes name (string) which identifies the provider.
// Takes provider (EmbeddingProviderPort) which handles embedding requests.
//
// Returns error when registration fails.
//
// Safe for concurrent use; protects the provider map with a mutex.
func (s *embeddingService) RegisterEmbeddingProvider(ctx context.Context, name string, provider EmbeddingProviderPort) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.providers[name]; exists {
		return fmt.Errorf("%w: %s", ErrProviderAlreadyExists, name)
	}

	s.providers[name] = provider
	_, l := logger_domain.From(ctx, log)
	l.Internal("Registered embedding provider",
		logger_domain.String(AttrKeyProvider, name),
	)
	return nil
}

// SetDefaultEmbeddingProvider sets the default embedding provider by name.
//
// Takes name (string) which identifies the provider.
//
// Returns error when the named provider does not exist.
//
// Safe for concurrent use.
func (s *embeddingService) SetDefaultEmbeddingProvider(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.providers[name]; !exists {
		return fmt.Errorf("%w: %s", ErrProviderNotFound, name)
	}

	s.defaultProvider = name
	return nil
}

// Embed generates embeddings using the specified provider.
//
// Takes providerName (string) which identifies the embedding provider to use.
// Takes request (*llm_dto.EmbeddingRequest) which contains the
// input data to embed.
//
// Returns *llm_dto.EmbeddingResponse which contains the generated embeddings.
// Returns error when the provider is not found or the embedding request fails.
func (s *embeddingService) Embed(ctx context.Context, providerName string, request *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
	ctx, l := logger_domain.From(ctx, log)
	var response *llm_dto.EmbeddingResponse

	err := l.RunInSpan(ctx, "EmbeddingService.Embed", func(spanCtx context.Context, spanLog logger_domain.Logger) error {
		start := s.clock.Now()
		defer func() {
			embeddingDuration.Record(spanCtx, float64(s.clock.Now().Sub(start).Milliseconds()))
		}()
		embeddingCount.Add(spanCtx, 1)

		provider, err := s.getEmbeddingProvider(providerName)
		if err != nil {
			embeddingErrorCount.Add(spanCtx, 1)
			return fmt.Errorf("getting embedding provider: %w", err)
		}

		spanLog.Debug("Sending embedding request",
			logger_domain.String(AttrKeyProvider, providerName),
			logger_domain.String("model", request.Model),
			logger_domain.Int("input_count", len(request.Input)),
		)

		response, err = goroutine.SafeCall1(spanCtx, "llm.Embed", func() (*llm_dto.EmbeddingResponse, error) { return provider.Embed(spanCtx, request) })
		if err != nil {
			embeddingErrorCount.Add(spanCtx, 1)
			return fmt.Errorf("provider embedding failed: %w", err)
		}

		if response.Usage != nil {
			embeddingTokenCount.Add(spanCtx, int64(response.Usage.TotalTokens))
		}

		return nil
	},
		logger_domain.String(AttrKeyProvider, providerName),
		logger_domain.String("model", request.Model),
	)

	return response, err
}

// getEmbeddingProvider retrieves an embedding provider by name.
//
// Takes name (string) which specifies the provider to retrieve. If empty, the
// default provider is used.
//
// Returns EmbeddingProviderPort which is the requested provider.
// Returns error when no default provider is configured or the named provider
// does not exist.
//
// Safe for concurrent use; protected by a read lock.
func (s *embeddingService) getEmbeddingProvider(name string) (EmbeddingProviderPort, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if name == "" {
		name = s.defaultProvider
	}
	if name == "" {
		return nil, ErrNoDefaultProvider
	}

	provider, exists := s.providers[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, name)
	}
	return provider, nil
}

// EmbeddingBuilder provides a fluent API for building and executing embedding
// requests.
type EmbeddingBuilder struct {
	// embeddingService is the service used to execute embedding requests.
	embeddingService *embeddingService

	// request holds the embedding request being built.
	request *llm_dto.EmbeddingRequest

	// providerName specifies which embedding provider to use.
	providerName string
}

// NewEmbeddingBuilder creates a new embedding builder.
//
// Takes service (*embeddingService) which provides the embedding service for
// generating embeddings.
//
// Returns *EmbeddingBuilder which is ready to configure and execute embedding
// requests.
func NewEmbeddingBuilder(service *embeddingService) *EmbeddingBuilder {
	return &EmbeddingBuilder{
		embeddingService: service,
		request:          &llm_dto.EmbeddingRequest{},
		providerName:     "",
	}
}

// Model sets the embedding model to use.
//
// Takes model (string) which identifies the model (e.g.,
// "text-embedding-3-small").
//
// Returns *EmbeddingBuilder for method chaining.
func (b *EmbeddingBuilder) Model(model string) *EmbeddingBuilder {
	b.request.Model = model
	return b
}

// Input sets the texts to embed.
//
// Takes texts (...string) which are the input texts to embed.
//
// Returns *EmbeddingBuilder for method chaining.
func (b *EmbeddingBuilder) Input(texts ...string) *EmbeddingBuilder {
	b.request.Input = append(b.request.Input, texts...)
	return b
}

// Dimensions sets the output dimension for the embedding.
// Only supported by certain models.
//
// Takes d (int) which is the output dimension.
//
// Returns *EmbeddingBuilder for method chaining.
func (b *EmbeddingBuilder) Dimensions(d int) *EmbeddingBuilder {
	b.request.Dimensions = &d
	return b
}

// EncodingFormat sets the output encoding format.
//
// Takes format (string) which is "float" or "base64".
//
// Returns *EmbeddingBuilder for method chaining.
func (b *EmbeddingBuilder) EncodingFormat(format string) *EmbeddingBuilder {
	b.request.EncodingFormat = &format
	return b
}

// Provider specifies which registered provider to use.
//
// Takes name (string) which identifies the provider.
//
// Returns *EmbeddingBuilder for method chaining.
func (b *EmbeddingBuilder) Provider(name string) *EmbeddingBuilder {
	b.providerName = name
	return b
}

// User sets an optional user identifier.
//
// Takes user (string) which identifies the end-user.
//
// Returns *EmbeddingBuilder for method chaining.
func (b *EmbeddingBuilder) User(user string) *EmbeddingBuilder {
	b.request.User = &user
	return b
}

// Metadata adds metadata for tracking and logging.
//
// Takes key (string) which identifies the metadata.
// Takes value (string) which is the metadata value.
//
// Returns *EmbeddingBuilder for method chaining.
func (b *EmbeddingBuilder) Metadata(key, value string) *EmbeddingBuilder {
	if b.request.Metadata == nil {
		b.request.Metadata = make(map[string]string)
	}
	b.request.Metadata[key] = value
	return b
}

// Build returns the configured request without running it.
//
// Returns llm_dto.EmbeddingRequest which contains the configured settings.
func (b *EmbeddingBuilder) Build() llm_dto.EmbeddingRequest {
	return *b.request
}

// Embed executes the embedding request and returns the response.
//
// Returns *llm_dto.EmbeddingResponse which contains the generated embeddings.
// Returns error when the request fails.
func (b *EmbeddingBuilder) Embed(ctx context.Context) (*llm_dto.EmbeddingResponse, error) {
	return b.embeddingService.Embed(ctx, b.providerName, b.request)
}

// withEmbeddingServiceClock sets the clock used for time operations.
//
// Takes c (clock.Clock) which provides the clock for time-related operations.
//
// Returns embeddingServiceOption which configures the embedding service.
func withEmbeddingServiceClock(c clock.Clock) embeddingServiceOption {
	return func(s *embeddingService) {
		s.clock = c
	}
}

// newEmbeddingService creates a new embedding service.
//
// Takes opts (...embeddingServiceOption) which configures the service.
//
// Returns *embeddingService which is ready for use with default settings.
func newEmbeddingService(opts ...embeddingServiceOption) *embeddingService {
	s := &embeddingService{
		clock:           clock.RealClock(),
		providers:       make(map[string]EmbeddingProviderPort),
		defaultProvider: "",
		mu:              sync.RWMutex{},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}
