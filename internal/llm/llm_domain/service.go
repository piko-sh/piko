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
	"errors"
	"fmt"
	"sync"
	"time"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/maths"
)

// service provides the core LLM functionality, implementing the Service
// interface. It manages providers, caching, rate limiting, budgets, and cost
// tracking.
type service struct {
	// clock provides time operations for testing.
	clock clock.Clock

	// providers maps provider names to their LLM provider implementations.
	providers map[string]LLMProviderPort

	// costCalculator computes usage costs; nil uses a default with the service
	// clock.
	costCalculator *CostCalculator

	// budgetManager tracks and enforces usage limits by scope; nil disables
	// budget enforcement.
	budgetManager *BudgetManager

	// rateLimiter controls request and token rate limits per scope; nil
	// disables rate limiting.
	rateLimiter *RateLimiter

	// cacheManager stores cached conversation history for message retrieval.
	cacheManager *CacheManager

	// embeddingService handles embedding requests and delegates to providers.
	embeddingService *embeddingService

	// vectorStore provides vector storage and similarity search; nil disables
	// vector search.
	vectorStore VectorStorePort

	// circuitBreakerConfig holds the default circuit breaker settings. When
	// non-nil, all providers registered via RegisterProvider are wrapped.
	circuitBreakerConfig *circuitBreakerConfig

	// defaultProvider is the name of the provider used when none is specified.
	defaultProvider string

	// mu guards access to the providers map.
	mu sync.RWMutex
}

// circuitBreakerConfig holds circuit breaker settings for providers.
type circuitBreakerConfig struct {
	// maxFailures is the consecutive failure threshold before opening
	// the circuit.
	maxFailures int

	// timeout is how long the circuit stays open before attempting
	// recovery.
	timeout time.Duration
}

var _ Service = (*service)(nil)

// ServiceOption is a functional option for setting up the LLM service.
type ServiceOption func(*service)

// NewCompletion creates a new completion builder.
//
// Returns *CompletionBuilder which is ready for configuration.
func (s *service) NewCompletion() *CompletionBuilder {
	return &CompletionBuilder{
		service:        s,
		request:        &llm_dto.CompletionRequest{},
		providerName:   "",
		budgetScope:    "",
		maxCost:        maths.ZeroMoney(llm_dto.CostCurrency),
		retryPolicy:    nil,
		fallbackConfig: nil,
		cacheConfig:    nil,
		memory:         nil,
		conversationID: "",
	}
}

// Complete sends a completion request to the default provider.
//
// Takes request (*llm_dto.CompletionRequest) which specifies the prompt and
// completion parameters.
//
// Returns *llm_dto.CompletionResponse which contains the generated completion.
// Returns error when the completion request fails.
func (s *service) Complete(ctx context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
	return s.CompleteWithProvider(ctx, s.defaultProvider, request)
}

// CompleteWithProvider sends a completion request to a specific provider.
//
// Takes providerName (string) which identifies the provider to use.
// Takes request (*llm_dto.CompletionRequest) which contains the
// completion request.
//
// Returns *llm_dto.CompletionResponse which contains the
// provider's response.
// Returns error when the request fails or the provider is unavailable.
func (s *service) CompleteWithProvider(ctx context.Context, providerName string, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
	return s.completeWithScope(ctx, providerName, request, "", maths.ZeroMoney(llm_dto.CostCurrency))
}

// Stream sends a streaming completion request to the default provider.
//
// Takes request (*llm_dto.CompletionRequest) which specifies the completion
// parameters.
//
// Returns <-chan llm_dto.StreamEvent which yields streaming events as they
// arrive from the provider.
// Returns error when the request fails to start.
func (s *service) Stream(ctx context.Context, request *llm_dto.CompletionRequest) (<-chan llm_dto.StreamEvent, error) {
	return s.StreamWithProvider(ctx, s.defaultProvider, request)
}

// StreamWithProvider sends a streaming completion request to a specific
// provider.
//
// Takes providerName (string) which identifies the provider to use.
// Takes request (*llm_dto.CompletionRequest) which contains the
// completion request.
//
// Returns <-chan llm_dto.StreamEvent which yields stream events
// as they arrive.
// Returns error when the provider is not found, does not support streaming, or
// the request fails validation.
func (s *service) StreamWithProvider(ctx context.Context, providerName string, request *llm_dto.CompletionRequest) (<-chan llm_dto.StreamEvent, error) {
	ctx, l := logger_domain.From(ctx, log)
	streamCount.Add(ctx, 1)

	provider, err := s.getProvider(providerName)
	if err != nil {
		streamErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("getting stream provider %q: %w", providerName, err)
	}

	if !goroutine.SafeCallValue(ctx, "llm.SupportsStreaming", func() bool { return provider.SupportsStreaming() }) {
		streamErrorCount.Add(ctx, 1)
		return nil, ErrStreamingNotSupported
	}

	request.Stream = true

	if request.Model == "" {
		request.Model = goroutine.SafeCallValue(ctx, "llm.DefaultModel", func() string { return provider.DefaultModel() })
	}

	if err := ValidateRequestForProvider(request, provider); err != nil {
		streamErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	l.Debug("Starting streaming completion",
		logger_domain.String(AttrKeyProvider, providerName),
		logger_domain.String(attributeKeyModel, request.Model),
	)

	events, err := goroutine.SafeCall1(ctx, "llm.Stream", func() (<-chan llm_dto.StreamEvent, error) { return provider.Stream(ctx, request) })
	if err != nil {
		streamErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("provider stream failed: %w", err)
	}

	return s.wrapStreamWithMetrics(ctx, events), nil
}

// RegisterProvider adds a provider to the registry.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes name (string) which identifies the provider in the registry.
// Takes provider (LLMProviderPort) which is the provider implementation to add.
//
// Returns error when a provider with the given name already exists.
//
// Safe for concurrent use; protected by a mutex.
func (s *service) RegisterProvider(ctx context.Context, name string, provider LLMProviderPort) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.providers[name]; exists {
		return fmt.Errorf("%w: %s", ErrProviderAlreadyExists, name)
	}

	if s.circuitBreakerConfig != nil {
		provider = newCircuitBreakerProvider(ctx, name, provider, s.circuitBreakerConfig.maxFailures, s.circuitBreakerConfig.timeout)
	}

	s.providers[name] = provider
	_, l := logger_domain.From(ctx, log)
	l.Internal("Registered LLM provider",
		logger_domain.String(AttrKeyProvider, name),
	)
	return nil
}

// SetDefaultProvider sets the default provider by name.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes name (string) which specifies the provider to use as default.
//
// Returns error when the named provider does not exist.
//
// Safe for concurrent use.
func (s *service) SetDefaultProvider(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.providers[name]; !exists {
		return fmt.Errorf("%w: %s", ErrProviderNotFound, name)
	}

	s.defaultProvider = name
	_, l := logger_domain.From(ctx, log)
	l.Internal("Set default LLM provider",
		logger_domain.String(AttrKeyProvider, name),
	)
	return nil
}

// GetDefaultProvider returns the name of the default provider.
//
// Returns string which is the name of the currently configured default
// provider.
//
// Safe for concurrent use.
func (s *service) GetDefaultProvider() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.defaultProvider
}

// GetProviders returns all registered provider names.
//
// Returns []string which contains the names of all registered providers.
//
// Safe for concurrent use.
func (s *service) GetProviders() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.providers))
	for name := range s.providers {
		names = append(names, name)
	}
	return names
}

// HasProvider checks if a provider with the given name exists.
//
// Takes name (string) which is the provider name to look up.
//
// Returns bool which is true if the provider exists, false otherwise.
//
// Safe for concurrent use.
func (s *service) HasProvider(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.providers[name]
	return exists
}

// Close closes all registered providers.
//
// Returns error when any provider fails to close, returning the last error.
//
// Safe for concurrent use.
func (s *service) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx, l := logger_domain.From(ctx, log)

	var errs []error

	for name, provider := range s.providers {
		if err := ctx.Err(); err != nil {
			errs = append(errs, err)
			break
		}
		if err := provider.Close(ctx); err != nil {
			l.Error("Failed to close LLM provider",
				logger_domain.String(AttrKeyProvider, name),
				logger_domain.Error(err),
			)
			errs = append(errs, err)
		}
	}
	s.providers = make(map[string]LLMProviderPort)

	if s.vectorStore != nil {
		if err := ctx.Err(); err != nil {
			errs = append(errs, err)
		} else if err := s.vectorStore.Close(ctx); err != nil {
			l.Error("Failed to close vector store",
				logger_domain.Error(err),
			)
			errs = append(errs, err)
		}
		s.vectorStore = nil
	}

	return errors.Join(errs...)
}

// SetBudget configures a budget for a scope.
//
// Takes scope (string) which identifies the budget scope.
// Takes config (*llm_dto.BudgetConfig) which contains the budget limits.
func (s *service) SetBudget(scope string, config *llm_dto.BudgetConfig) {
	if s.budgetManager != nil {
		s.budgetManager.SetBudget(scope, config)
	}
}

// GetBudgetStatus returns the current budget status for a scope.
//
// Takes scope (string) which identifies the budget scope.
//
// Returns *llm_dto.BudgetStatus which contains the current state.
// Returns error when the status cannot be retrieved.
func (s *service) GetBudgetStatus(ctx context.Context, scope string) (*llm_dto.BudgetStatus, error) {
	if s.budgetManager == nil {
		return &llm_dto.BudgetStatus{
			Scope:            scope,
			TotalSpent:       maths.ZeroMoney(llm_dto.CostCurrency),
			DailySpent:       maths.ZeroMoney(llm_dto.CostCurrency),
			HourlySpent:      maths.ZeroMoney(llm_dto.CostCurrency),
			RequestCount:     0,
			TokenCount:       0,
			RemainingBudget:  maths.ZeroMoney(llm_dto.CostCurrency),
			ThresholdReached: false,
			LastUpdated:      time.Time{},
		}, nil
	}
	return s.budgetManager.GetStatus(ctx, scope)
}

// SetPricingTable replaces the pricing table used for cost calculation.
//
// Takes table (*llm_dto.PricingTable) which is the new pricing table to use.
func (s *service) SetPricingTable(table *llm_dto.PricingTable) {
	if s.costCalculator != nil {
		s.costCalculator.SetPricingTable(table)
	}
}

// SetRateLimits configures rate limits for a scope.
//
// Takes scope (string) which identifies the rate limit scope.
// Takes requestsPerMinute (int) which is the max requests per minute (0 =
// unlimited).
// Takes tokensPerMinute (int) which is the max tokens per minute (0 =
// unlimited).
func (s *service) SetRateLimits(scope string, requestsPerMinute, tokensPerMinute int) {
	if s.rateLimiter != nil {
		s.rateLimiter.SetLimits(scope, requestsPerMinute, tokensPerMinute)
	}
}

// GetCostCalculator returns the cost calculator.
//
// Returns *CostCalculator which provides cost calculation functionality.
func (s *service) GetCostCalculator() *CostCalculator {
	return s.costCalculator
}

// GetBudgetManager returns the budget manager.
//
// Returns *BudgetManager which manages budget tracking for the service.
func (s *service) GetBudgetManager() *BudgetManager {
	return s.budgetManager
}

// GetRateLimiter returns the rate limiter.
//
// Returns *RateLimiter which controls request rate limiting for the service.
func (s *service) GetRateLimiter() *RateLimiter {
	return s.rateLimiter
}

// GetCacheManager returns the cache manager.
//
// Returns *CacheManager which provides cache operations for the service.
func (s *service) GetCacheManager() *CacheManager {
	return s.cacheManager
}

// SetCacheManager sets the cache manager for the service.
//
// Takes cacheManager (*CacheManager) which provides caching for the service.
func (s *service) SetCacheManager(cacheManager *CacheManager) {
	s.cacheManager = cacheManager
}

// SetBudgetManager sets the budget manager for the service.
//
// Takes manager (*BudgetManager) which handles budget enforcement.
func (s *service) SetBudgetManager(manager *BudgetManager) {
	s.budgetManager = manager
}

// GetVectorStore returns the vector store.
//
// Returns VectorStorePort which may be nil if vector search is not configured.
func (s *service) GetVectorStore() VectorStorePort {
	return s.vectorStore
}

// SetVectorStore sets the vector store for the service.
//
// Takes store (VectorStorePort) which provides vector storage and search.
func (s *service) SetVectorStore(store VectorStorePort) {
	s.vectorStore = store
}

// NewEmbedding creates a new builder for embedding requests.
//
// Returns *EmbeddingBuilder which is used to build and configure embedding
// requests.
func (s *service) NewEmbedding() *EmbeddingBuilder {
	return NewEmbeddingBuilder(s.embeddingService)
}

// Embed generates embeddings using the default provider.
//
// Takes request (*llm_dto.EmbeddingRequest) which contains the embedding
// parameters.
//
// Returns *llm_dto.EmbeddingResponse which contains the generated embeddings.
// Returns error when the request fails.
func (s *service) Embed(ctx context.Context, request *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
	return s.embeddingService.Embed(ctx, "", request)
}

// EmbedWithProvider generates embeddings using a specific provider.
//
// Takes providerName (string) which identifies the provider to use.
// Takes request (*llm_dto.EmbeddingRequest) which contains the embedding
// parameters.
//
// Returns *llm_dto.EmbeddingResponse which contains the generated embeddings.
// Returns error when the request fails.
func (s *service) EmbedWithProvider(ctx context.Context, providerName string, request *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
	return s.embeddingService.Embed(ctx, providerName, request)
}

// RegisterEmbeddingProvider adds an embedding provider to the registry.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes name (string) which identifies the provider.
// Takes provider (EmbeddingProviderPort) which handles embedding requests.
//
// Returns error when registration fails.
func (s *service) RegisterEmbeddingProvider(ctx context.Context, name string, provider EmbeddingProviderPort) error {
	return s.embeddingService.RegisterEmbeddingProvider(ctx, name, provider)
}

// SetDefaultEmbeddingProvider sets the default embedding provider by name.
//
// Takes name (string) which identifies the provider.
//
// Returns error when the named provider does not exist.
func (s *service) SetDefaultEmbeddingProvider(name string) error {
	return s.embeddingService.SetDefaultEmbeddingProvider(name)
}

// NewIngest creates a new ingestion builder for loading and vectorising
// documents.
//
// Takes namespace (string) which specifies the target namespace for ingestion.
//
// Returns *IngestBuilder which is configured to ingest into the given
// namespace.
func (s *service) NewIngest(namespace string) *IngestBuilder {
	return NewIngestBuilder(s, namespace)
}

// AddText is a convenience method that embeds and stores a single piece of
// text.
//
// Takes namespace (string) which identifies the storage namespace.
// Takes id (string) which uniquely identifies the text within the namespace.
// Takes content (string) which is the text to embed and store.
//
// Returns error when embedding or storage fails.
func (s *service) AddText(ctx context.Context, namespace, id, content string) error {
	return s.AddDocuments(ctx, namespace, []Document{
		{
			ID:      id,
			Content: content,
		},
	})
}

// AddDocuments embeds and stores multiple documents in the vector store.
// It handles batching of embedding requests for efficiency.
//
// Takes namespace (string) which identifies the storage namespace.
// Takes docs ([]Document) which contains the documents to embed and store.
//
// Returns error when the vector store is not configured, embedding fails,
// or storage fails.
func (s *service) AddDocuments(ctx context.Context, namespace string, docs []Document) error {
	if s.vectorStore == nil {
		return ErrVectorStoreNotConfigured
	}

	if len(docs) == 0 {
		return nil
	}

	if err := s.ensureVectorNamespace(ctx, namespace); err != nil {
		return err
	}

	const batchSize = 20
	for i := 0; i < len(docs); i += batchSize {
		if selectDone(ctx) {
			return ctx.Err()
		}

		end := min(i+batchSize, len(docs))
		if err := s.embedAndStoreBatch(ctx, namespace, docs[i:end], i); err != nil {
			return err
		}
	}

	return nil
}

// EmbeddingDimensions returns the default embedding dimension from the
// currently configured default embedding provider. Returns 0 when no provider
// is configured or the dimension is not known statically.
//
// Returns int which is the vector dimension, or 0 if unknown.
func (s *service) EmbeddingDimensions() int {
	provider, err := s.embeddingService.getEmbeddingProvider("")
	if err != nil {
		return 0
	}
	return provider.EmbeddingDimensions()
}

// ensureVectorNamespace creates the vector namespace if the embedding
// dimension is known.
//
// Takes namespace (string) which identifies the namespace to create.
//
// Returns error when namespace creation fails.
func (s *service) ensureVectorNamespace(ctx context.Context, namespace string) error {
	dim := s.EmbeddingDimensions()
	if dim <= 0 {
		return nil
	}
	if err := s.vectorStore.CreateNamespace(ctx, namespace, &VectorNamespaceConfig{
		Dimension: dim,
		Metric:    llm_dto.SimilarityCosine,
	}); err != nil {
		return fmt.Errorf("creating vector namespace %q: %w", namespace, err)
	}
	return nil
}

// embedAndStoreBatch embeds a single batch of documents and stores them in the
// vector store.
//
// Takes namespace (string) which identifies the storage namespace.
// Takes batch ([]Document) which contains the documents to process.
// Takes offset (int) which is the starting index for error messages.
//
// Returns error when embedding or storage fails.
func (s *service) embedAndStoreBatch(ctx context.Context, namespace string, batch []Document, offset int) error {
	inputs := make([]string, len(batch))
	for j, document := range batch {
		inputs[j] = document.Content
	}

	response, err := s.embeddingService.Embed(ctx, "", &llm_dto.EmbeddingRequest{
		Input: inputs,
	})
	if err != nil {
		return fmt.Errorf("batch embedding failed at offset %d: %w", offset, err)
	}

	if len(response.Embeddings) != len(batch) {
		return fmt.Errorf("embedding count mismatch at offset %d: got %d embeddings for %d inputs",
			offset, len(response.Embeddings), len(batch))
	}

	vectorDocuments := make([]*llm_dto.VectorDocument, len(batch))
	for j, emb := range response.Embeddings {
		vectorDocuments[j] = &llm_dto.VectorDocument{
			ID:       batch[j].ID,       //nolint:gosec // len equality checked above
			Content:  batch[j].Content,  //nolint:gosec // len equality checked above
			Metadata: batch[j].Metadata, //nolint:gosec // len equality checked above
			Vector:   emb.Vector,
		}
	}

	if err := s.vectorStore.BulkStore(ctx, namespace, vectorDocuments); err != nil {
		return fmt.Errorf("bulk storage failed for batch at offset %d: %w", offset, err)
	}
	return nil
}

// initialiseDefaults sets up default components that were not provided.
// This is called after options are applied to ensure the service clock is used.
func (s *service) initialiseDefaults() {
	if s.costCalculator == nil {
		s.costCalculator = NewCostCalculator(WithCostCalculatorClock(s.clock))
	}
	if s.embeddingService == nil {
		s.embeddingService = newEmbeddingService(withEmbeddingServiceClock(s.clock))
	}
}

// completeWithScope sends a completion request with optional budget scope and
// max cost.
//
// Takes providerName (string) which identifies the LLM provider to use.
// Takes request (*llm_dto.CompletionRequest) which contains the
// completion request.
// Takes budgetScope (string) which specifies the budget scope for
// cost tracking.
// Takes maxCost (maths.Money) which sets the maximum allowed cost.
//
// Returns *llm_dto.CompletionResponse which contains the completion result.
// Returns error when pre-request limits fail or the completion fails.
func (s *service) completeWithScope(ctx context.Context, providerName string, request *llm_dto.CompletionRequest, budgetScope string, maxCost maths.Money) (*llm_dto.CompletionResponse, error) {
	ctx, l := logger_domain.From(ctx, log)
	var response *llm_dto.CompletionResponse

	err := l.RunInSpan(ctx, "LLMService.Complete", func(spanCtx context.Context, _ logger_domain.Logger) error {
		start := s.clock.Now()
		defer func() {
			completionDuration.Record(spanCtx, float64(s.clock.Now().Sub(start).Milliseconds()))
		}()
		completionCount.Add(spanCtx, 1)

		if request.Model == "" {
			provider, provErr := s.getProvider(providerName)
			if provErr != nil {
				return fmt.Errorf("getting provider for model resolution: %w", provErr)
			}
			request.Model = goroutine.SafeCallValue(spanCtx, "llm.DefaultModel", func() string { return provider.DefaultModel() })
		}

		if err := s.checkPreRequestLimits(spanCtx, request, budgetScope, maxCost); err != nil {
			return fmt.Errorf("checking pre-request limits: %w", err)
		}

		var err error
		response, err = s.executeCompletion(spanCtx, providerName, request, budgetScope)
		if err != nil {
			return fmt.Errorf("executing completion: %w", err)
		}
		return nil
	},
		logger_domain.String(AttrKeyProvider, providerName),
		logger_domain.String(attributeKeyModel, request.Model),
		logger_domain.String("budget_scope", budgetScope),
	)

	return response, err
}

// checkPreRequestLimits validates rate limits and budget before executing a
// request.
//
// Takes request (*llm_dto.CompletionRequest) which contains the request to
// validate.
// Takes budgetScope (string) which identifies the budget namespace to check.
// Takes maxCost (maths.Money) which specifies the maximum allowed cost.
//
// Returns error when rate limit is exceeded or budget is insufficient.
func (s *service) checkPreRequestLimits(ctx context.Context, request *llm_dto.CompletionRequest, budgetScope string, maxCost maths.Money) error {
	if err := s.checkRateLimit(ctx, budgetScope); err != nil {
		return fmt.Errorf("checking rate limit for scope %q: %w", budgetScope, err)
	}
	return s.checkBudget(ctx, request, budgetScope, maxCost)
}

// checkRateLimit checks if the request is allowed by rate limits.
//
// Takes budgetScope (string) which identifies the rate limit bucket to check.
//
// Returns error when the rate limit has been exceeded for the given scope.
func (s *service) checkRateLimit(ctx context.Context, budgetScope string) error {
	if s.rateLimiter == nil || budgetScope == "" {
		return nil
	}
	if err := s.rateLimiter.Allow(ctx, budgetScope); err != nil {
		rateLimitedCount.Add(ctx, 1)
		return fmt.Errorf("rate limiter rejected request for scope %q: %w", budgetScope, err)
	}
	return nil
}

// checkBudget checks if the request is allowed by budget constraints.
//
// Takes request (*llm_dto.CompletionRequest) which contains the request to check.
// Takes budgetScope (string) which identifies the budget to check against.
// Takes maxCost (maths.Money) which specifies the maximum allowed cost.
//
// Returns error when the estimated cost exceeds maxCost or the budget limit.
func (s *service) checkBudget(ctx context.Context, request *llm_dto.CompletionRequest, budgetScope string, maxCost maths.Money) error {
	hasMaxCost := !maxCost.CheckIsZero()
	hasBudgetScope := s.budgetManager != nil && budgetScope != ""

	if !hasMaxCost && !hasBudgetScope {
		return nil
	}

	estimatedInputTokens := EstimateInputTokens(request.Messages)
	estimatedCost := s.costCalculator.EstimateRequestCost(request.Model, estimatedInputTokens)

	if hasMaxCost && estimatedCost.CheckGreaterThan(maxCost) {
		budgetExceededCount.Add(ctx, 1)
		return ErrMaxCostExceeded
	}

	if hasBudgetScope {
		if err := s.budgetManager.CheckBudget(ctx, budgetScope, estimatedCost); err != nil {
			budgetExceededCount.Add(ctx, 1)
			return fmt.Errorf("checking budget for scope %q: %w", budgetScope, err)
		}
	}
	return nil
}

// executeCompletion executes the completion request with the provider.
//
// Takes providerName (string) which identifies the LLM provider to use.
// Takes request (*llm_dto.CompletionRequest) which contains the
// completion request.
// Takes budgetScope (string) which specifies the scope for cost
// tracking.
//
// Returns *llm_dto.CompletionResponse which contains the provider's response.
// Returns error when the provider is not found, request validation fails, or
// the provider completion fails.
func (s *service) executeCompletion(ctx context.Context, providerName string, request *llm_dto.CompletionRequest, budgetScope string) (*llm_dto.CompletionResponse, error) {
	ctx, l := logger_domain.From(ctx, log)

	provider, err := s.getProvider(providerName)
	if err != nil {
		completionErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("getting completion provider %q: %w", providerName, err)
	}

	if request.Model == "" {
		request.Model = goroutine.SafeCallValue(ctx, "llm.DefaultModel", func() string { return provider.DefaultModel() })
	}

	if err := ValidateRequestForProvider(request, provider); err != nil {
		completionErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	l.Debug("Sending completion request",
		logger_domain.String(AttrKeyProvider, providerName),
		logger_domain.String(attributeKeyModel, request.Model),
		logger_domain.Int("message_count", len(request.Messages)),
	)

	response, err := goroutine.SafeCall1(ctx, "llm.Complete", func() (*llm_dto.CompletionResponse, error) { return provider.Complete(ctx, request) })
	if err != nil {
		completionErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("provider completion failed: %w", err)
	}

	s.recordTokenUsage(ctx, response)
	s.recordRateLimiterTokens(ctx, budgetScope, response)
	s.recordCostAndBudget(ctx, providerName, request.Model, response, budgetScope)

	return response, nil
}

// recordRateLimiterTokens records actual token usage to the rate limiter
// after a request completes. This consumes tokens from the token bucket
// without consuming from the request bucket (requests=0).
//
// Takes budgetScope (string) which identifies the rate limit scope.
// Takes response (*llm_dto.CompletionResponse) which contains usage data.
func (s *service) recordRateLimiterTokens(ctx context.Context, budgetScope string, response *llm_dto.CompletionResponse) {
	if s.rateLimiter == nil || budgetScope == "" || response == nil || response.Usage == nil {
		return
	}
	if rateLimitError := s.rateLimiter.AllowN(ctx, budgetScope, 0, response.Usage.TotalTokens); rateLimitError != nil {
		_, warningLogger := logger_domain.From(ctx, nil)
		warningLogger.Warn("failed to record rate limiter tokens",
			logger_domain.String("budgetScope", budgetScope),
			logger_domain.Error(rateLimitError))
	}
}

// wrapStreamWithMetrics wraps a stream channel with metrics recording.
//
// Takes events (<-chan llm_dto.StreamEvent) which provides the stream events
// to wrap.
//
// Returns <-chan llm_dto.StreamEvent which yields the same events after
// recording metrics for errors, duration, and token usage.
//
// Spawns a goroutine that forwards events and records metrics. The goroutine
// runs until the input channel is closed.
func (s *service) wrapStreamWithMetrics(ctx context.Context, events <-chan llm_dto.StreamEvent) <-chan llm_dto.StreamEvent {
	wrapped := make(chan llm_dto.StreamEvent, streamEventBufferSize)
	start := s.clock.Now()

	go func() {
		defer close(wrapped)
		defer goroutine.RecoverPanic(ctx, "llm.wrapStreamWithMetrics")
		for event := range events {
			s.recordStreamEventMetrics(ctx, event, start)
			select {
			case wrapped <- event:
			case <-ctx.Done():
				return
			}
		}
	}()

	return wrapped
}

// recordStreamEventMetrics records metrics for a single stream event,
// including error counts, stream duration, and token usage on completion.
//
// Takes event (llm_dto.StreamEvent) which is the event to record metrics for.
// Takes start (time.Time) which is when the stream started.
func (s *service) recordStreamEventMetrics(ctx context.Context, event llm_dto.StreamEvent, start time.Time) {
	if event.Type == llm_dto.StreamEventError {
		streamErrorCount.Add(ctx, 1)
	}
	if event.Done || event.Type == llm_dto.StreamEventDone {
		streamDuration.Record(ctx, float64(s.clock.Now().Sub(start).Milliseconds()))
		if event.FinalResponse != nil {
			s.recordTokenUsage(ctx, event.FinalResponse)
		}
	}
}

// recordTokenUsage records token usage metrics from a response.
//
// Takes response (*llm_dto.CompletionResponse) which contains the usage data to
// record.
func (*service) recordTokenUsage(ctx context.Context, response *llm_dto.CompletionResponse) {
	if response == nil || response.Usage == nil {
		return
	}
	promptTokenCount.Add(ctx, int64(response.Usage.PromptTokens))
	completionTokenCount.Add(ctx, int64(response.Usage.CompletionTokens))
	totalTokenCount.Add(ctx, int64(response.Usage.TotalTokens))
}

// recordCostAndBudget calculates actual cost and records budget usage.
//
// Takes providerName (string) which identifies the LLM provider.
// Takes model (string) which specifies the model used for the request.
// Takes response (*llm_dto.CompletionResponse) which contains the usage data.
// Takes budgetScope (string) which defines the budget category to record
// against.
func (s *service) recordCostAndBudget(ctx context.Context, providerName, model string, response *llm_dto.CompletionResponse, budgetScope string) {
	if response == nil || response.Usage == nil || s.costCalculator == nil {
		return
	}

	ctx, l := logger_domain.From(ctx, log)

	cost := s.costCalculator.Calculate(model, providerName, response.Usage)
	if cost != nil {
		response.Usage.EstimatedCost = cost

		if amountDecimal, err := cost.TotalCost.Amount(); err == nil {
			if costFloat, err := amountDecimal.Float64(); err == nil {
				requestCostHistogram.Record(ctx, costFloat)
				totalSpendCounter.Add(ctx, costFloat)
			}
		}

		if s.budgetManager != nil && budgetScope != "" {
			if err := s.budgetManager.RecordUsage(ctx, budgetScope, cost); err != nil {
				l.Error("Failed to record budget usage",
					logger_domain.String(AttrKeyProvider, providerName),
					logger_domain.String(attributeKeyModel, model),
					logger_domain.String("scope", budgetScope),
					logger_domain.Error(err),
				)
			}
		}
	}
}

// getProvider retrieves a provider by name, using the default if name is empty.
//
// Takes name (string) which specifies the provider to retrieve; if empty, uses
// the default provider.
//
// Returns LLMProviderPort which is the requested provider instance.
// Returns error when no default provider is configured or the named provider
// does not exist.
//
// Safe for concurrent use; protects access with a read lock.
func (s *service) getProvider(name string) (LLMProviderPort, error) {
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

// WithClock sets the clock used for time operations. If not set,
// clock.RealClock() is used.
//
// Takes c (clock.Clock) which provides time operations.
//
// Returns ServiceOption which applies the clock setting to the service.
func WithClock(c clock.Clock) ServiceOption {
	return func(s *service) {
		s.clock = c
	}
}

// WithCostCalculator sets the cost calculator for the service. If not set, a
// default cost calculator is used.
//
// Takes calculator (*CostCalculator) which performs cost calculations.
//
// Returns ServiceOption which configures the service with the given calculator.
func WithCostCalculator(calculator *CostCalculator) ServiceOption {
	return func(s *service) {
		s.costCalculator = calculator
	}
}

// WithBudgetManager sets the budget manager for the service.
// If not set, budget management is disabled.
//
// Takes manager (*BudgetManager) which handles budget enforcement.
//
// Returns ServiceOption to apply to the service.
func WithBudgetManager(manager *BudgetManager) ServiceOption {
	return func(s *service) {
		s.budgetManager = manager
	}
}

// WithRateLimiter sets the rate limiter for the service. If not set, rate
// limiting is disabled.
//
// Takes limiter (*RateLimiter) which controls request rate limiting.
//
// Returns ServiceOption which configures the service with rate limiting.
func WithRateLimiter(limiter *RateLimiter) ServiceOption {
	return func(s *service) {
		s.rateLimiter = limiter
	}
}

// WithPricingTable sets a custom pricing table for cost calculations. This
// creates a new CostCalculator using the provided table, replacing any
// calculator set via WithCostCalculator.
//
// Takes table (*llm_dto.PricingTable) which contains the model pricing data.
//
// Returns ServiceOption which configures the service with custom pricing.
func WithPricingTable(table *llm_dto.PricingTable) ServiceOption {
	return func(s *service) {
		s.costCalculator = NewCostCalculatorWithPricing(table, WithCostCalculatorClock(s.clock))
	}
}

// WithVectorStore sets the vector store for similarity search. If not set,
// vector search is not available.
//
// Takes store (VectorStorePort) which provides vector storage and search.
//
// Returns ServiceOption which configures the service with the vector store.
func WithVectorStore(store VectorStorePort) ServiceOption {
	return func(s *service) {
		s.vectorStore = store
	}
}

// WithCircuitBreaker enables circuit breaking for all providers. When a
// provider fails maxFailures times consecutively, the circuit opens and
// requests fast-fail with ErrProviderOverloaded for the given timeout.
//
// Takes maxFailures (int) which sets the consecutive failure threshold.
// Takes timeout (time.Duration) which specifies how long the circuit stays
// open before attempting recovery.
//
// Returns ServiceOption which enables circuit breaking.
func WithCircuitBreaker(maxFailures int, timeout time.Duration) ServiceOption {
	return func(s *service) {
		s.circuitBreakerConfig = &circuitBreakerConfig{
			maxFailures: maxFailures,
			timeout:     timeout,
		}
	}
}

// NewService creates a new LLM service with an optional default provider.
//
// Takes defaultProviderName (string) which sets the initial default provider
// name. An empty string means no default is set.
// Takes opts (...ServiceOption) which are optional configuration functions.
//
// Returns Service ready for provider registration.
func NewService(defaultProviderName string, opts ...ServiceOption) Service {
	s := &service{
		clock:            clock.RealClock(),
		providers:        make(map[string]LLMProviderPort),
		costCalculator:   nil,
		budgetManager:    nil,
		rateLimiter:      nil,
		cacheManager:     nil,
		embeddingService: nil,
		defaultProvider:  defaultProviderName,
		mu:               sync.RWMutex{},
	}
	for _, opt := range opts {
		opt(s)
	}
	s.initialiseDefaults()
	return s
}
