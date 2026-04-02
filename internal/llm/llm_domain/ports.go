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

	"piko.sh/piko/internal/llm/llm_dto"
)

// Service provides the main interface for LLM completions, embeddings, and
// provider management. It implements io.Closer and is the hexagon's
// public API in the ports and adapters pattern.
type Service interface {
	// NewCompletion creates a new completion builder for composing and executing
	// LLM requests.
	//
	// Returns *CompletionBuilder which provides a fluent interface for building
	// completion requests.
	NewCompletion() *CompletionBuilder

	// Complete sends a completion request to the default or specified provider.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	// Takes request (*llm_dto.CompletionRequest) which contains
	// the completion parameters.
	//
	// Returns *llm_dto.CompletionResponse containing the model's response.
	// Returns error when the request fails or the provider is not found.
	Complete(ctx context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error)

	// CompleteWithProvider sends a completion request to a specific provider.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	// Takes providerName (string) which identifies the provider to use.
	// Takes request (*llm_dto.CompletionRequest) which contains
	// the completion parameters.
	//
	// Returns *llm_dto.CompletionResponse containing the model's response.
	// Returns error when the request fails or the provider is not found.
	CompleteWithProvider(ctx context.Context, providerName string, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error)

	// Stream sends a streaming completion request to the default or specified
	// provider.
	//
	// Takes request (*llm_dto.CompletionRequest) which contains the completion
	// parameters.
	//
	// Returns <-chan llm_dto.StreamEvent which emits streaming events.
	// Returns error when the stream cannot be started or the provider is not found.
	Stream(ctx context.Context, request *llm_dto.CompletionRequest) (<-chan llm_dto.StreamEvent, error)

	// StreamWithProvider sends a streaming completion request to a specific provider.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	// Takes providerName (string) which identifies the provider to use.
	// Takes request (*llm_dto.CompletionRequest) which contains
	// the completion parameters.
	//
	// Returns <-chan llm_dto.StreamEvent which emits streaming events.
	// Returns error when the stream cannot be started or the provider is not found.
	StreamWithProvider(ctx context.Context, providerName string, request *llm_dto.CompletionRequest) (<-chan llm_dto.StreamEvent, error)

	// RegisterProvider adds an LLM provider to the registry.
	//
	// Takes ctx (context.Context) which carries logging context for trace/request
	// ID propagation.
	// Takes name (string) which identifies the provider.
	// Takes provider (LLMProviderPort) which handles LLM requests.
	//
	// Returns error when registration fails.
	RegisterProvider(ctx context.Context, name string, provider LLMProviderPort) error

	// SetDefaultProvider sets the default provider to use by name.
	//
	// Takes ctx (context.Context) which carries logging context for trace/request
	// ID propagation.
	// Takes name (string) which identifies the provider to set as default.
	//
	// Returns error when the named provider does not exist.
	SetDefaultProvider(ctx context.Context, name string) error

	// GetDefaultProvider returns the name of the default provider.
	//
	// Returns string which is the default provider name, or empty if none is set.
	GetDefaultProvider() string

	// GetProviders returns the names of all registered providers.
	//
	// Returns []string which contains the provider names.
	GetProviders() []string

	// HasProvider reports whether a provider with the given name exists.
	//
	// Takes name (string) which identifies the provider to check.
	//
	// Returns bool which is true if the provider exists, false otherwise.
	HasProvider(name string) bool

	// Close releases resources held by all registered providers.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	//
	// Returns error when any provider fails to close cleanly.
	Close(ctx context.Context) error

	// SetBudget configures a budget for a scope.
	//
	// Takes scope (string) which identifies the budget scope.
	// Takes config (*llm_dto.BudgetConfig) which contains the budget limits.
	SetBudget(scope string, config *llm_dto.BudgetConfig)

	// GetBudgetStatus returns the current budget status for a scope.
	//
	// Takes ctx (context.Context) which controls cancellation.
	// Takes scope (string) which identifies the budget scope.
	//
	// Returns *llm_dto.BudgetStatus containing the current state.
	// Returns error if the status cannot be retrieved.
	GetBudgetStatus(ctx context.Context, scope string) (*llm_dto.BudgetStatus, error)

	// SetPricingTable replaces the pricing table used for cost calculation.
	//
	// Takes table (*llm_dto.PricingTable) which is the new pricing table.
	SetPricingTable(table *llm_dto.PricingTable)

	// SetRateLimits configures rate limits for a scope.
	//
	// Takes scope (string) which identifies the rate limit scope.
	// Takes requestsPerMinute (int) which limits requests per minute (0 = no limit).
	// Takes tokensPerMinute (int) which limits tokens per minute (0 = no limit).
	SetRateLimits(scope string, requestsPerMinute, tokensPerMinute int)

	// GetCostCalculator returns the cost calculator for this instance.
	GetCostCalculator() *CostCalculator

	// GetBudgetManager returns the budget manager.
	//
	// Returns *BudgetManager which handles budget tracking and control.
	GetBudgetManager() *BudgetManager

	// GetRateLimiter returns the rate limiter for this client.
	//
	// Returns *RateLimiter which controls the rate of requests.
	GetRateLimiter() *RateLimiter

	// GetCacheManager returns the cache manager.
	//
	// Returns *CacheManager which may be nil if caching is not configured.
	GetCacheManager() *CacheManager

	// SetCacheManager sets the cache manager.
	//
	// Takes cacheManager (*CacheManager) which is the cache manager to use.
	SetCacheManager(cacheManager *CacheManager)

	// SetBudgetManager sets the budget manager.
	//
	// Takes manager (*BudgetManager) which handles budget enforcement.
	SetBudgetManager(manager *BudgetManager)

	// GetVectorStore returns the vector store for similarity search.
	//
	// Returns VectorStorePort which may be nil if vector search is not
	// configured.
	GetVectorStore() VectorStorePort

	// SetVectorStore sets the vector store for the service.
	//
	// Takes store (VectorStorePort) which provides vector storage and search.
	SetVectorStore(store VectorStorePort)

	// NewEmbedding creates a new embedding builder for composing and executing
	// embedding requests.
	//
	// Returns *EmbeddingBuilder which provides a fluent interface for building
	// embedding requests.
	NewEmbedding() *EmbeddingBuilder

	// Embed generates embeddings using the default provider.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	// Takes request (*llm_dto.EmbeddingRequest) which contains
	// the embedding parameters.
	//
	// Returns *llm_dto.EmbeddingResponse containing the generated embeddings.
	// Returns error when the request fails.
	Embed(ctx context.Context, request *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error)

	// EmbedWithProvider generates embeddings using a specific provider.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	// Takes providerName (string) which identifies the provider to use.
	// Takes request (*llm_dto.EmbeddingRequest) which contains
	// the embedding parameters.
	//
	// Returns *llm_dto.EmbeddingResponse containing the generated embeddings.
	// Returns error when the request fails.
	EmbedWithProvider(ctx context.Context, providerName string, request *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error)

	// RegisterEmbeddingProvider adds an embedding provider to the registry.
	//
	// Takes ctx (context.Context) which carries logging context for trace/request
	// ID propagation.
	// Takes name (string) which identifies the provider.
	// Takes provider (EmbeddingProviderPort) which handles embedding requests.
	//
	// Returns error when registration fails.
	RegisterEmbeddingProvider(ctx context.Context, name string, provider EmbeddingProviderPort) error

	// SetDefaultEmbeddingProvider sets the default embedding provider by name.
	//
	// Takes name (string) which identifies the provider.
	//
	// Returns error when the named provider does not exist.
	SetDefaultEmbeddingProvider(name string) error

	// EmbeddingDimensions returns the default vector dimension from the currently
	// configured default embedding provider.
	//
	// Returns int which is the vector dimension, or 0 if no provider is configured
	// or the dimension is not known statically (e.g. server-determined models).
	EmbeddingDimensions() int

	// NewIngest creates a new ingestion builder for loading, splitting, and
	// vectorising documents into a namespace.
	//
	// Takes namespace (string) which is the target vector store namespace.
	//
	// Returns *IngestBuilder which provides a fluent interface for ingestion.
	NewIngest(namespace string) *IngestBuilder

	// AddText is a convenience method that embeds and stores a single piece of
	// text in the vector store.
	//
	// Takes ctx (context.Context) which controls cancellation.
	// Takes namespace (string) which identifies the target collection.
	// Takes id (string) which is the document ID.
	// Takes content (string) which is the raw text to embed and store.
	//
	// Returns error when embedding or storage fails.
	AddText(ctx context.Context, namespace, id, content string) error

	// AddDocuments embeds and stores multiple documents in the vector store.
	// It handles batching of embedding requests for efficiency.
	//
	// Takes ctx (context.Context) which controls cancellation.
	// Takes namespace (string) which identifies the target collection.
	// Takes docs ([]Document) which contains the documents to process.
	//
	// Returns error when embedding or storage fails.
	AddDocuments(ctx context.Context, namespace string, docs []Document) error
}

// Document represents raw content and metadata before it is vectorised.
type Document struct {
	// Metadata holds optional key-value pairs associated with the document.
	Metadata map[string]any

	// ID is the unique identifier for the document.
	ID string

	// Content is the raw text content.
	Content string
}

// LoaderPort is the driven port for loading documents from various sources
// (e.g., local files, S3, databases).
type LoaderPort interface {
	// Load retrieves documents from the source.
	//
	// Takes ctx (context.Context) which controls cancellation.
	//
	// Returns []Document containing the loaded content.
	// Returns error when loading fails.
	Load(ctx context.Context) ([]Document, error)
}

// SplitterPort is the driven port for breaking large documents into smaller
// chunks for vector search indexing.
type SplitterPort interface {
	// Split divides a document into one or more smaller documents.
	//
	// Takes document (Document) which is the document to split.
	//
	// Returns []Document containing the resulting chunks.
	Split(document Document) []Document
}

// LLMProviderPort defines the interface that LLM provider adapters must implement.
// It is a driven port in the hexagonal architecture pattern, allowing the domain
// to send requests to different LLM providers.
type LLMProviderPort interface {
	// Complete sends a completion request to the provider.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	// Takes request (*llm_dto.CompletionRequest) which contains
	// the completion parameters.
	//
	// Returns *llm_dto.CompletionResponse containing the model's response.
	// Returns error when the request fails.
	Complete(ctx context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error)

	// Stream sends a streaming completion request to the provider.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	// Takes request (*llm_dto.CompletionRequest) which contains
	// the completion parameters.
	//
	// Returns <-chan llm_dto.StreamEvent which emits streaming events.
	// Returns error when the stream cannot be started.
	Stream(ctx context.Context, request *llm_dto.CompletionRequest) (<-chan llm_dto.StreamEvent, error)

	// SupportsStreaming reports whether the provider supports streaming responses.
	//
	// Returns bool which is true if streaming is supported.
	SupportsStreaming() bool

	// SupportsStructuredOutput reports whether the provider supports JSON schema
	// structured output natively or via translation.
	//
	// Returns bool which is true if structured output is supported.
	SupportsStructuredOutput() bool

	// SupportsTools reports whether the provider supports tool/function calling.
	//
	// Returns bool which is true if tools are supported.
	SupportsTools() bool

	// SupportsPenalties reports whether the provider supports frequency and
	// presence penalty parameters.
	//
	// Returns bool which is true if penalties are supported.
	SupportsPenalties() bool

	// SupportsSeed reports whether the provider supports the seed parameter
	// for deterministic sampling.
	//
	// Returns bool which is true if seed is supported.
	SupportsSeed() bool

	// SupportsParallelToolCalls reports whether the provider supports making
	// multiple tool calls in a single response.
	//
	// Returns bool which is true if parallel tool calls are supported.
	SupportsParallelToolCalls() bool

	// SupportsMessageName reports whether the provider supports the optional
	// Name field on messages for multi-participant conversations.
	//
	// Returns bool which is true if message names are supported.
	SupportsMessageName() bool

	// ListModels returns information about available models.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	//
	// Returns []llm_dto.ModelInfo which contains model metadata.
	// Returns error when the model list cannot be retrieved.
	ListModels(ctx context.Context) ([]llm_dto.ModelInfo, error)

	// Close releases any resources held by the provider.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	//
	// Returns error when the provider cannot be closed cleanly.
	Close(ctx context.Context) error

	// DefaultModel returns the provider's default completion model name. The
	// service calls this method when a request omits the model, before validation.
	//
	// Returns string which is the default model identifier, or empty if the
	// caller must specify a model explicitly.
	DefaultModel() string
}
