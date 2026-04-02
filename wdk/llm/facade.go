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

package llm

import (
	"io/fs"
	"time"

	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/llm/llm_adapters/vector_cache"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
)

// Type re-exports from llm_domain for public API.
type (
	// Service is the primary interface for LLM operations.
	Service = llm_domain.Service

	// ProviderPort is the interface that provider adapters implement.
	ProviderPort = llm_domain.LLMProviderPort

	// CompletionBuilder provides a fluent API for building completions.
	CompletionBuilder = llm_domain.CompletionBuilder

	// CostCalculator computes estimated costs from token usage.
	CostCalculator = llm_domain.CostCalculator

	// BudgetManager enforces spending limits.
	BudgetManager = llm_domain.BudgetManager

	// BudgetStorePort is the driven port for budget persistence.
	BudgetStorePort = llm_domain.BudgetStorePort

	// RateLimiter enforces request rate limits.
	RateLimiter = llm_domain.RateLimiter

	// RateLimiterOption is a functional option for configuring the RateLimiter.
	RateLimiterOption = llm_domain.RateLimiterOption

	// RateLimiterStorePort is the driven port for rate limiter state storage.
	RateLimiterStorePort = ratelimiter_domain.TokenBucketStorePort

	// CacheStorePort is the driven port for cache persistence.
	CacheStorePort = llm_domain.CacheStorePort

	// CacheManager manages LLM response caching.
	CacheManager = llm_domain.CacheManager

	// EmbeddingProviderPort is the interface that embedding provider adapters
	// implement.
	EmbeddingProviderPort = llm_domain.EmbeddingProviderPort

	// EmbeddingBuilder provides a fluent API for building embedding requests.
	EmbeddingBuilder = llm_domain.EmbeddingBuilder

	// MemoryStorePort is the driven port for conversation memory persistence.
	MemoryStorePort = llm_domain.MemoryStorePort

	// Memory is the interface for conversation memory management.
	Memory = llm_domain.Memory

	// BufferMemory implements Memory using a fixed-size message buffer.
	BufferMemory = llm_domain.BufferMemory

	// BufferMemoryOption is a functional option for configuring BufferMemory.
	BufferMemoryOption = llm_domain.BufferMemoryOption

	// WindowMemory implements Memory using a token-based window.
	WindowMemory = llm_domain.WindowMemory

	// WindowMemoryOption is a functional option for configuring WindowMemory.
	WindowMemoryOption = llm_domain.WindowMemoryOption

	// SummaryMemory implements Memory using LLM summarisation.
	SummaryMemory = llm_domain.SummaryMemory

	// VectorStorePort is the driven port for vector storage
	// and similarity search.
	VectorStorePort = llm_domain.VectorStorePort

	// VectorNamespaceConfig configures a vector store namespace including
	// the similarity metric and vector dimensionality.
	VectorNamespaceConfig = llm_domain.VectorNamespaceConfig

	// Document represents raw content and metadata before
	// it is vectorised.
	Document = llm_domain.Document

	// IngestBuilder provides a fluent API for loading, splitting,
	// transforming, and vectorising documents into a namespace.
	IngestBuilder = llm_domain.IngestBuilder

	// TransformFunc is a function that transforms a document before splitting.
	TransformFunc = llm_domain.TransformFunc

	// FrontmatterOption configures the behaviour of [ExtractFrontmatter].
	FrontmatterOption = llm_domain.FrontmatterOption

	// SplitterPort is the driven port for splitting documents into chunks.
	SplitterPort = llm_domain.SplitterPort

	// MarkdownSplitter splits markdown documents on heading boundaries.
	MarkdownSplitter = llm_domain.MarkdownSplitter

	// MarkdownSplitterOption configures optional behaviour of
	// [NewMarkdownSplitter].
	MarkdownSplitterOption = llm_domain.MarkdownSplitterOption

	// LoaderPort is the driven port for loading documents from sources.
	LoaderPort = llm_domain.LoaderPort

	// RequestDump captures the assembled state of a completion request for
	// debugging. Obtained via [CompletionBuilder.DryRun].
	RequestDump = llm_domain.RequestDump

	// ToolHandlerFunc handles tool calls from the model.
	ToolHandlerFunc = llm_domain.ToolHandlerFunc

	// RAGOption is a functional option for configuring automatic RAG
	// behaviour on the CompletionBuilder.
	RAGOption = llm_domain.RAGOption

	// QueryRewriterFunc rewrites a user query into one or more search
	// queries for multi-query RAG. See [WithRAGQueryRewriter].
	QueryRewriterFunc = llm_domain.QueryRewriterFunc

	// QueryRewriterOption is a functional option for configuring the
	// built-in LLM-based query rewriter created by [LLMQueryRewriter].
	QueryRewriterOption = llm_domain.QueryRewriterOption

	// ResponseValidatorFunc validates the final LLM response after tool
	// dispatch. See [CompletionBuilder.ResponseValidator].
	ResponseValidatorFunc = llm_domain.ResponseValidatorFunc

	// ServiceOption is a functional option for configuring a Service.
	ServiceOption = llm_domain.ServiceOption
)

// Type re-exports from llm_dto for public API.
type (
	// CompletionRequest contains parameters for a completion request.
	CompletionRequest = llm_dto.CompletionRequest

	// CompletionResponse contains the response from a completion request.
	CompletionResponse = llm_dto.CompletionResponse

	// Message represents a single message in a conversation.
	Message = llm_dto.Message

	// Choice represents a single completion choice.
	Choice = llm_dto.Choice

	// Usage contains token usage statistics.
	Usage = llm_dto.Usage

	// StreamEvent represents an event during streaming.
	StreamEvent = llm_dto.StreamEvent

	// StreamChunk contains a chunk of streaming data.
	StreamChunk = llm_dto.StreamChunk

	// MessageDelta contains incremental message changes during streaming.
	MessageDelta = llm_dto.MessageDelta

	// ToolDefinition defines a tool the model can call.
	ToolDefinition = llm_dto.ToolDefinition

	// FunctionDefinition describes a callable function.
	FunctionDefinition = llm_dto.FunctionDefinition

	// ToolCall represents a tool call from the model.
	ToolCall = llm_dto.ToolCall

	// FunctionCall contains function call details.
	FunctionCall = llm_dto.FunctionCall

	// ToolChoice controls how the model uses tools.
	ToolChoice = llm_dto.ToolChoice

	// ResponseFormat specifies output format.
	ResponseFormat = llm_dto.ResponseFormat

	// JSONSchema defines a JSON schema for structured output.
	JSONSchema = llm_dto.JSONSchema

	// JSONSchemaDefinition wraps a schema with metadata.
	JSONSchemaDefinition = llm_dto.JSONSchemaDefinition

	// ModelInfo contains metadata about a model.
	ModelInfo = llm_dto.ModelInfo

	// StreamOptions configures streaming behaviour.
	StreamOptions = llm_dto.StreamOptions

	// ModelPricing defines cost per token for a model.
	ModelPricing = llm_dto.ModelPricing

	// CostEstimate represents calculated cost for a request.
	CostEstimate = llm_dto.CostEstimate

	// PricingTable is a collection of model pricing.
	PricingTable = llm_dto.PricingTable

	// BudgetConfig defines spending limits.
	BudgetConfig = llm_dto.BudgetConfig

	// BudgetStatus represents current budget state.
	BudgetStatus = llm_dto.BudgetStatus

	// Limit represents a single limit configuration.
	Limit = llm_dto.Limit

	// RetryPolicy configures retry behaviour for LLM
	// requests.
	RetryPolicy = llm_dto.RetryPolicy

	// FallbackConfig configures fallback behaviour for LLM
	// requests.
	FallbackConfig = llm_dto.FallbackConfig

	// FallbackResult contains information about fallback execution.
	FallbackResult = llm_dto.FallbackResult

	// CacheConfig configures caching behaviour for LLM
	// requests.
	CacheConfig = llm_dto.CacheConfig

	// CacheEntry represents a cached LLM response.
	CacheEntry = llm_dto.CacheEntry

	// CacheStats contains statistics about cache usage.
	CacheStats = llm_dto.CacheStats

	// EmbeddingRequest contains parameters for an embedding
	// request.
	EmbeddingRequest = llm_dto.EmbeddingRequest

	// EmbeddingResponse contains the response from an embedding request.
	EmbeddingResponse = llm_dto.EmbeddingResponse

	// Embedding represents a single embedding vector.
	Embedding = llm_dto.Embedding

	// EmbeddingUsage contains token usage statistics for an embedding request.
	EmbeddingUsage = llm_dto.EmbeddingUsage

	// ContentPart represents a single part of multi-modal
	// message content.
	ContentPart = llm_dto.ContentPart

	// ImageURL contains image URL information for vision requests.
	ImageURL = llm_dto.ImageURL

	// ImageData contains inline base64-encoded image data.
	ImageData = llm_dto.ImageData

	// MemoryConfig configures conversation memory behaviour.
	MemoryConfig = llm_dto.MemoryConfig

	// ConversationState represents the current state of a conversation's memory.
	ConversationState = llm_dto.ConversationState

	// VectorDocument represents a document stored in the
	// vector store with its embedding and metadata.
	VectorDocument = llm_dto.VectorDocument

	// VectorSearchRequest holds the settings for a vector similarity search
	// including query vector, top-K limit, and optional filters.
	VectorSearchRequest = llm_dto.VectorSearchRequest

	// VectorSearchResponse contains the results of a vector similarity search.
	VectorSearchResponse = llm_dto.VectorSearchResponse

	// VectorSearchResult represents a single result from a vector similarity
	// search with its similarity score.
	VectorSearchResult = llm_dto.VectorSearchResult
)

// Role is a type alias re-exported from the llm_dto package.
type Role = llm_dto.Role

// FinishReason is an alias for the LLM DTO finish reason type.
type FinishReason = llm_dto.FinishReason

// StreamEventType is re-exported from llm_dto and defines the kind of event
// in a stream.
type StreamEventType = llm_dto.StreamEventType

// ResponseFormatType is a type alias re-exported from the llm_dto package.
type ResponseFormatType = llm_dto.ResponseFormatType

// LimitType is a type alias re-exported from llm_dto.
type LimitType = llm_dto.LimitType

// FallbackTrigger is a type alias re-exported from llm_dto.
type FallbackTrigger = llm_dto.FallbackTrigger

// ContentPartType defines the kind of content in a message part.
// Re-exported from llm_dto.
type ContentPartType = llm_dto.ContentPartType

// SimilarityMetric specifies how to measure the distance between vectors in a
// similarity search. Re-exported from llm_dto.
type SimilarityMetric = llm_dto.SimilarityMetric

// MemoryType is a type alias re-exported from the llm_dto package.
type MemoryType = llm_dto.MemoryType

const (
	// RoleSystem is the role identifier for system messages in LLM conversations.
	RoleSystem = llm_dto.RoleSystem

	// RoleUser is the role identifier for user messages in LLM conversations.
	RoleUser = llm_dto.RoleUser

	// RoleAssistant is the role identifier for assistant messages in
	// conversations.
	RoleAssistant = llm_dto.RoleAssistant

	// RoleTool is the role identifier for tool messages in LLM conversations.
	RoleTool = llm_dto.RoleTool

	// FinishReasonStop indicates the model stopped generating naturally.
	FinishReasonStop = llm_dto.FinishReasonStop

	// FinishReasonLength indicates the output was truncated due to length limits.
	FinishReasonLength = llm_dto.FinishReasonLength

	// FinishReasonToolCalls indicates the model stopped to invoke tool calls.
	FinishReasonToolCalls = llm_dto.FinishReasonToolCalls

	// FinishReasonContentFilter indicates the response was stopped due to content
	// filtering policies.
	FinishReasonContentFilter = llm_dto.FinishReasonContentFilter

	// StreamEventChunk is an alias for llm_dto.StreamEventChunk.
	StreamEventChunk = llm_dto.StreamEventChunk

	// StreamEventDone indicates the stream has completed.
	StreamEventDone = llm_dto.StreamEventDone

	// StreamEventError is the event type for stream errors.
	StreamEventError = llm_dto.StreamEventError

	// ResponseFormatText is the response format for plain text output.
	ResponseFormatText = llm_dto.ResponseFormatText

	// ResponseFormatJSONObject is the response format for JSON object output.
	ResponseFormatJSONObject = llm_dto.ResponseFormatJSONObject

	// ResponseFormatJSONSchema is the response format type for JSON schema output.
	ResponseFormatJSONSchema = llm_dto.ResponseFormatJSONSchema

	// LimitTypeSpend is the limit type for spend limits.
	LimitTypeSpend = llm_dto.LimitTypeSpend

	// LimitTypeRequests is the limit type for request-based rate limiting.
	LimitTypeRequests = llm_dto.LimitTypeRequests

	// LimitTypeTokens is a limit type that counts tokens.
	LimitTypeTokens = llm_dto.LimitTypeTokens

	// FallbackOnError is a constant that indicates fallback behaviour when an
	// error occurs.
	FallbackOnError = llm_dto.FallbackOnError

	// FallbackOnRateLimit is a fallback strategy that switches providers when rate
	// limited.
	FallbackOnRateLimit = llm_dto.FallbackOnRateLimit

	// FallbackOnTimeout indicates that the operation should use a fallback value
	// when a timeout occurs.
	FallbackOnTimeout = llm_dto.FallbackOnTimeout

	// FallbackOnBudgetExceeded is a budget strategy that uses the fallback model
	// when the token budget is exceeded.
	FallbackOnBudgetExceeded = llm_dto.FallbackOnBudgetExceeded

	// FallbackOnAll is the fallback strategy that applies to all operations.
	FallbackOnAll = llm_dto.FallbackOnAll

	// ContentPartTypeText is the content part type for plain text content.
	ContentPartTypeText = llm_dto.ContentPartTypeText

	// ContentPartTypeImageURL is the content part type for image URLs.
	ContentPartTypeImageURL = llm_dto.ContentPartTypeImageURL

	// ContentPartTypeImageData represents image data content in a message part.
	ContentPartTypeImageData = llm_dto.ContentPartTypeImageData

	// SimilarityCosine uses cosine similarity for vector comparison. Values
	// range from -1 to 1, where 1 means the vectors point in the same direction.
	SimilarityCosine = llm_dto.SimilarityCosine

	// SimilarityEuclidean uses Euclidean distance to compare vectors. The range
	// is [0, +inf) where 0 means the vectors are identical.
	SimilarityEuclidean = llm_dto.SimilarityEuclidean

	// SimilarityDotProduct uses dot product for vector comparison. Higher values
	// indicate greater similarity.
	SimilarityDotProduct = llm_dto.SimilarityDotProduct

	// MemoryTypeBuffer is a memory type constant for buffer-based memory storage.
	MemoryTypeBuffer = llm_dto.MemoryTypeBuffer

	// MemoryTypeSummary is a memory type for summary memories.
	MemoryTypeSummary = llm_dto.MemoryTypeSummary

	// MemoryTypeWindow is the memory type that uses a sliding window of messages.
	MemoryTypeWindow = llm_dto.MemoryTypeWindow

	// DefaultMaxToolRounds is the maximum number of tool dispatch rounds
	// before the loop terminates. Each round consists of dispatching all
	// tool calls in a response and re-calling the LLM.
	DefaultMaxToolRounds = llm_domain.DefaultMaxToolRounds
)

var (
	// ErrProviderNotFound is returned when a requested LLM provider does not
	// exist.
	ErrProviderNotFound = llm_domain.ErrProviderNotFound

	// ErrNoDefaultProvider is returned when no default LLM provider is configured.
	ErrNoDefaultProvider = llm_domain.ErrNoDefaultProvider

	// ErrProviderAlreadyExists is returned when attempting to register a provider
	// that has already been registered.
	ErrProviderAlreadyExists = llm_domain.ErrProviderAlreadyExists

	// ErrStreamingNotSupported is returned when a model does not support
	// streaming.
	ErrStreamingNotSupported = llm_domain.ErrStreamingNotSupported

	// ErrToolsNotSupported is returned when the LLM does not support tool calls.
	ErrToolsNotSupported = llm_domain.ErrToolsNotSupported

	// ErrStructuredOutputNotSupported is returned when the LLM provider does not
	// support structured output format.
	ErrStructuredOutputNotSupported = llm_domain.ErrStructuredOutputNotSupported

	// ErrEmptyMessages is returned when no messages are provided to the LLM.
	ErrEmptyMessages = llm_domain.ErrEmptyMessages

	// ErrEmptyModel is returned when a model name is not provided.
	ErrEmptyModel = llm_domain.ErrEmptyModel

	// ErrInvalidTemperature is returned when a temperature value is outside the
	// valid range.
	ErrInvalidTemperature = llm_domain.ErrInvalidTemperature

	// ErrInvalidTopP is returned when the TopP value is outside the valid range.
	ErrInvalidTopP = llm_domain.ErrInvalidTopP

	// ErrInvalidMaxTokens is returned when the max tokens value is invalid.
	ErrInvalidMaxTokens = llm_domain.ErrInvalidMaxTokens

	// ErrBudgetExceeded is a budget or rate limiting error returned when the
	// allowed budget has been exceeded.
	ErrBudgetExceeded = llm_domain.ErrBudgetExceeded

	// ErrRateLimited is returned when the API rate limit has been exceeded.
	ErrRateLimited = llm_domain.ErrRateLimited

	// ErrMaxCostExceeded is returned when an LLM operation exceeds the maximum
	// allowed cost.
	ErrMaxCostExceeded = llm_domain.ErrMaxCostExceeded

	// ErrUnknownModelPrice is returned when the price for a model is not known.
	ErrUnknownModelPrice = llm_domain.ErrUnknownModelPrice

	// ErrProviderOverloaded is a transient provider error that may be retried.
	ErrProviderOverloaded = llm_domain.ErrProviderOverloaded

	// ErrProviderTimeout is returned when a provider fails to respond within the
	// allowed time limit.
	ErrProviderTimeout = llm_domain.ErrProviderTimeout

	// ErrConversationNotFound is returned when a conversation cannot be found.
	ErrConversationNotFound = llm_domain.ErrConversationNotFound

	// NewSystemMessage creates a system message.
	NewSystemMessage = llm_dto.NewSystemMessage

	// NewUserMessage creates a new user message for the language model.
	NewUserMessage = llm_dto.NewUserMessage

	// NewAssistantMessage creates an assistant message.
	NewAssistantMessage = llm_dto.NewAssistantMessage

	// NewToolResultMessage creates a message containing a tool result.
	NewToolResultMessage = llm_dto.NewToolResultMessage

	// NewUserMessageWithImages creates a user message with text and images.
	NewUserMessageWithImages = llm_dto.NewUserMessageWithImages

	// NewUserMessageWithImageURL creates a user message with text and an image
	// URL.
	NewUserMessageWithImageURL = llm_dto.NewUserMessageWithImageURL

	// NewUserMessageWithImageData creates a user message with text and inline
	// image data.
	NewUserMessageWithImageData = llm_dto.NewUserMessageWithImageData

	// NewFunctionTool is a function that creates a new function tool definition.
	NewFunctionTool = llm_dto.NewFunctionTool

	// NewStrictFunctionTool creates a strict function tool definition.
	NewStrictFunctionTool = llm_dto.NewStrictFunctionTool

	// ToolChoiceAuto returns a ToolChoice for automatic selection.
	ToolChoiceAuto = llm_dto.ToolChoiceAuto

	// ToolChoiceNone returns a ToolChoice that disables tools.
	ToolChoiceNone = llm_dto.ToolChoiceNone

	// ToolChoiceRequired returns a ToolChoice requiring tool use.
	ToolChoiceRequired = llm_dto.ToolChoiceRequired

	// ToolChoiceSpecific returns a ToolChoice for a specific function.
	ToolChoiceSpecific = llm_dto.ToolChoiceSpecific

	// NewChunkEvent creates a chunk stream event.
	NewChunkEvent = llm_dto.NewChunkEvent

	// NewDoneEvent creates a done stream event.
	NewDoneEvent = llm_dto.NewDoneEvent

	// NewErrorEvent creates an error stream event.
	NewErrorEvent = llm_dto.NewErrorEvent

	// EncodingFormatFloat returns a pointer to "float" for use in
	// EmbeddingRequest.
	EncodingFormatFloat = llm_dto.EncodingFormatFloat

	// EncodingFormatBase64 returns a pointer to "base64" for use in
	// EmbeddingRequest.
	EncodingFormatBase64 = llm_dto.EncodingFormatBase64

	// TextPart creates a text content part for multi-modal messages.
	TextPart = llm_dto.TextPart

	// ImageURLPart creates an image URL content part for vision requests.
	ImageURLPart = llm_dto.ImageURLPart

	// ImageDataPart creates an inline image content part for vision requests.
	ImageDataPart = llm_dto.ImageDataPart

	// ImageDetailAuto is the value "auto" for use in ImageURL.Detail.
	ImageDetailAuto = llm_dto.ImageDetailAuto

	// ImageDetailLow returns "low" for use in ImageURL.Detail.
	ImageDetailLow = llm_dto.ImageDetailLow

	// ImageDetailHigh returns "high" for use in ImageURL.Detail.
	ImageDetailHigh = llm_dto.ImageDetailHigh

	// DefaultPricingTable contains built-in pricing for common models.
	DefaultPricingTable = llm_domain.DefaultPricingTable

	// WithCostCalculator sets the cost calculator for the
	// service.
	WithCostCalculator = llm_domain.WithCostCalculator

	// WithBudgetManager sets the budget manager for the service.
	WithBudgetManager = llm_domain.WithBudgetManager

	// WithRateLimiter sets the rate limiter for the service.
	WithRateLimiter = llm_domain.WithRateLimiter

	// WithClock sets the clock used by the service.
	WithClock = llm_domain.WithClock

	// WithVectorStore sets the vector store for the service.
	WithVectorStore = llm_domain.WithVectorStore

	// WithRateLimiterClock sets the clock used for time operations on the
	// RateLimiter.
	//
	// Takes c (clock.Clock) which provides time operations.
	//
	// Returns RateLimiterOption to apply to the rate limiter.
	WithRateLimiterClock = llm_domain.WithRateLimiterClock

	// WithRAGQuery sets an explicit query string for the
	// embedding lookup. If not set, the content of the last
	// user message is used.
	WithRAGQuery = llm_domain.WithRAGQuery

	// WithRAGMinScore sets a minimum similarity score threshold.
	WithRAGMinScore = llm_domain.WithRAGMinScore

	// WithRAGFilter sets metadata filter criteria for the vector search.
	WithRAGFilter = llm_domain.WithRAGFilter

	// WithRAGEmbeddingProvider sets the embedding provider to use for the
	// RAG query.
	WithRAGEmbeddingProvider = llm_domain.WithRAGEmbeddingProvider

	// WithRAGEmbeddingModel sets the embedding model to use for the RAG
	// query.
	WithRAGEmbeddingModel = llm_domain.WithRAGEmbeddingModel

	// WithRAGQueryRewriter sets a query rewriter function for the RAG pipeline
	// that transforms the original query before embedding. See
	// [QueryRewriterFunc].
	WithRAGQueryRewriter = llm_domain.WithRAGQueryRewriter

	// WithRAGHybridSearch enables combined vector and text search for RAG
	// retrieval. When enabled, the query text is passed to the vector store
	// alongside the embedding vector, allowing the store to combine semantic
	// similarity with lexical text matching using Reciprocal Rank Fusion.
	WithRAGHybridSearch = llm_domain.WithRAGHybridSearch

	// LLMQueryRewriter creates a [QueryRewriterFunc] that uses the LLM
	// service to rewrite or expand queries for improved vector search
	// retrieval.
	LLMQueryRewriter = llm_domain.LLMQueryRewriter

	// WithRewriterModel sets the LLM model for the built-in
	// query rewriter.
	WithRewriterModel = llm_domain.WithRewriterModel

	// WithRewriterProvider sets the LLM provider for the built-in query
	// rewriter.
	WithRewriterProvider = llm_domain.WithRewriterProvider

	// WithRewriterPrompt sets a custom system prompt for the built-in query
	// rewriter.
	WithRewriterPrompt = llm_domain.WithRewriterPrompt

	// WithRewriterMaxQueries sets the maximum number of expanded queries.
	// A value of 1 (default) produces a single rewritten query, while values
	// greater than 1 enable multi-query expansion.
	WithRewriterMaxQueries = llm_domain.WithRewriterMaxQueries

	// WithRewriterMaxTokens sets the max tokens for the rewriter completion.
	WithRewriterMaxTokens = llm_domain.WithRewriterMaxTokens

	// WithBufferSize sets the maximum number of messages to keep in a
	// [BufferMemory]. When omitted, the default size (20) is used.
	//
	// Takes size (int) which is the maximum number of messages.
	//
	// Returns BufferMemoryOption to apply to NewBufferMemory.
	WithBufferSize = llm_domain.WithBufferSize

	// WithTokenLimit sets the maximum number of tokens to keep in a
	// [WindowMemory]. When omitted, the default limit (4000) is used.
	//
	// Takes limit (int) which is the maximum token count.
	//
	// Returns WindowMemoryOption to apply to NewWindowMemory.
	WithTokenLimit = llm_domain.WithTokenLimit

	// WithMaxSplitLevel sets the maximum heading level that acts as a
	// split boundary for [NewMarkdownSplitter], where
	// WithMaxSplitLevel(3) splits on h1-h3 and groups h4-h6 content
	// with their parent section (default 2: h1 and h2 only).
	WithMaxSplitLevel = llm_domain.WithMaxSplitLevel

	// WithMinChunkSize sets the minimum chunk size in bytes for
	// [NewMarkdownSplitter], merging smaller chunks with an adjacent
	// chunk to eliminate orphaned fragments (0 disables merging).
	WithMinChunkSize = llm_domain.WithMinChunkSize
)

// CacheFactory creates a cache instance for a given namespace configuration.
// The factory is called once per namespace when
// [VectorStorePort.CreateNamespace] is invoked, and must return a cache
// configured with an appropriate search schema (vector fields, text fields) for
// the namespace.
//
// Use [cache_provider_otter.OtterProviderFactory] for in-memory storage or a
// Redis/Valkey provider factory for distributed storage.
type CacheFactory = vector_cache.CacheFactory

// NewService creates a new LLM service.
//
// Takes defaultProviderName (string) which sets the default provider. Pass an
// empty string to require explicit provider selection.
// Takes opts (...ServiceOption) which are optional settings such as
// [WithCostCalculator], [WithBudgetManager], [WithRateLimiter], [WithClock],
// and [WithVectorStore].
//
// Returns Service which is ready for provider registration.
func NewService(defaultProviderName string, opts ...ServiceOption) Service {
	return llm_domain.NewService(defaultProviderName, opts...)
}

// GetDefaultService returns the LLM service initialised by the framework.
//
// Returns Service which is the service instance ready for use.
// Returns error when the framework has not been bootstrapped.
//
// Example:
//
//	service, err := llm.GetDefaultService()
//	if err != nil {
//	    return err
//	}
//	response, err := service.NewCompletion().Model("gpt-4o").User("Hello!").Do(ctx)
func GetDefaultService() (Service, error) {
	return bootstrap.GetLLMService()
}

// NewCompletionBuilder creates a new completion builder.
//
// Takes service (Service) which is the LLM service to use.
//
// Returns *CompletionBuilder which provides a fluent interface for building
// completions.
func NewCompletionBuilder(service Service) *CompletionBuilder {
	return service.NewCompletion()
}

// NewCompletionBuilderFromDefault creates a new completion builder using the
// framework's bootstrapped service.
//
// Returns *CompletionBuilder which is the builder ready for use.
// Returns error when the framework has not been bootstrapped.
//
// Example:
//
//	builder, err := llm.NewCompletionBuilderFromDefault()
//	if err != nil {
//	    return err
//	}
//	response, err := builder.Model("gpt-4o").User("Hello").Do(ctx)
func NewCompletionBuilderFromDefault() (*CompletionBuilder, error) {
	service, err := GetDefaultService()
	if err != nil {
		return nil, err
	}
	return NewCompletionBuilder(service), nil
}

// NewEmbeddingBuilder creates a new embedding builder.
//
// Takes service (Service) which is the LLM service to use.
//
// Returns *EmbeddingBuilder which provides a fluent interface for building
// embedding requests.
func NewEmbeddingBuilder(service Service) *EmbeddingBuilder {
	return service.NewEmbedding()
}

// NewEmbeddingBuilderFromDefault creates a new embedding builder using the
// framework's bootstrapped service.
//
// Returns *EmbeddingBuilder which is the configured builder ready for use.
// Returns error when the framework has not been bootstrapped.
func NewEmbeddingBuilderFromDefault() (*EmbeddingBuilder, error) {
	service, err := GetDefaultService()
	if err != nil {
		return nil, err
	}
	return NewEmbeddingBuilder(service), nil
}

// NewCostCalculator creates a new CostCalculator with the default pricing
// table.
//
// Returns *CostCalculator which is set up with default pricing values.
func NewCostCalculator() *CostCalculator {
	return llm_domain.NewCostCalculator()
}

// NewCostCalculatorWithPricing creates a new CostCalculator with custom
// pricing.
//
// Takes table (*PricingTable) which is the custom pricing table to use.
//
// Returns *CostCalculator which is set up with the provided pricing.
func NewCostCalculatorWithPricing(table *PricingTable) *CostCalculator {
	return llm_domain.NewCostCalculatorWithPricing(table)
}

// NewBudgetManager creates a new BudgetManager with the given store and
// calculator.
//
// Takes store (BudgetStorePort) which saves and loads budget data.
// Takes calculator (*CostCalculator) which works out costs.
//
// Returns *BudgetManager which is ready for use.
func NewBudgetManager(store BudgetStorePort, calculator *CostCalculator) *BudgetManager {
	return llm_domain.NewBudgetManager(store, calculator)
}

// NewRateLimiter creates a new rate limiter with the given store and options.
//
// Takes store (RateLimiterStorePort) which provides storage for token bucket
// state.
// Takes opts (...RateLimiterOption) which are optional settings to configure
// the limiter.
//
// Returns *RateLimiter which is ready for use.
func NewRateLimiter(store RateLimiterStorePort, opts ...RateLimiterOption) *RateLimiter {
	return llm_domain.NewRateLimiter(store, opts...)
}

// NewCacheManager creates a new cache manager with the given storage backend.
//
// Takes store (CacheStorePort) which provides the cache storage backend.
// Takes defaultTTL (time.Duration) which sets the default time-to-live for
// cache entries.
//
// Returns *CacheManager which is ready for use.
func NewCacheManager(store CacheStorePort, defaultTTL time.Duration) *CacheManager {
	return llm_domain.NewCacheManager(store, defaultTTL)
}

// NewBufferMemory creates a new buffer memory with the given store.
//
// Takes store (MemoryStorePort) which handles persistence.
// Takes opts (...BufferMemoryOption) which configure the memory. Use
// [WithBufferSize] to override the default buffer size.
//
// Returns *BufferMemory ready for use.
func NewBufferMemory(store MemoryStorePort, opts ...BufferMemoryOption) *BufferMemory {
	return llm_domain.NewBufferMemory(store, opts...)
}

// NewWindowMemory creates a new WindowMemory instance.
//
// Takes store (MemoryStorePort) which handles storage of memory data.
// Takes opts (...WindowMemoryOption) which configure the memory. Use
// [WithTokenLimit] to override the default token limit.
//
// Returns *WindowMemory which is ready for use.
func NewWindowMemory(store MemoryStorePort, opts ...WindowMemoryOption) *WindowMemory {
	return llm_domain.NewWindowMemory(store, opts...)
}

// NewSummaryMemory creates a new SummaryMemory instance.
//
// Takes store (MemoryStorePort) which handles data storage.
// Takes service (Service) which provides the LLM for making summaries.
// Takes config (MemoryConfig) which sets up the memory behaviour.
//
// Returns *SummaryMemory which is ready for use.
func NewSummaryMemory(store MemoryStorePort, service Service, config MemoryConfig) *SummaryMemory {
	return llm_domain.NewSummaryMemory(store, service, config)
}

// ValidateRequest checks that a completion request is valid.
//
// Takes request (*CompletionRequest) which is the request to check.
//
// Returns error when the request is not valid.
func ValidateRequest(request *CompletionRequest) error {
	return llm_domain.ValidateRequest(request)
}

// NewVectorStore creates a cache-backed vector store, calling
// the factory once per namespace to create the backing cache
// instance (Otter for in-memory, Redis/Valkey for distributed).
//
// Takes factory (CacheFactory) which creates cache instances per namespace.
//
// Returns VectorStorePort which is ready for use.
func NewVectorStore(factory CacheFactory) VectorStorePort {
	return vector_cache.New(factory)
}

// StripFrontmatter returns a [TransformFunc] that removes YAML frontmatter
// delimited by --- from the beginning of a document's content.
//
// Returns TransformFunc which strips YAML frontmatter delimited by ---.
func StripFrontmatter() TransformFunc {
	return llm_domain.StripFrontmatter()
}

// ExtractFrontmatter returns a TransformFunc that parses YAML frontmatter,
// merges the extracted keys into the document's metadata, and strips the
// frontmatter from the content.
//
// Takes opts (...FrontmatterOption) which configures extraction behaviour.
//
// Returns TransformFunc which performs the frontmatter extraction.
//
// Use [WithFrontmatterKeys] to restrict which keys are extracted and
// [WithFrontmatterPrefix] to namespace metadata keys.
func ExtractFrontmatter(opts ...FrontmatterOption) TransformFunc {
	return llm_domain.ExtractFrontmatter(opts...)
}

// WithFrontmatterKeys restricts [ExtractFrontmatter] to only extract the
// named keys from the frontmatter.
//
// Takes keys (...string) which specifies the frontmatter keys to extract.
//
// Returns FrontmatterOption which configures the extraction behaviour.
func WithFrontmatterKeys(keys ...string) FrontmatterOption {
	return llm_domain.WithFrontmatterKeys(keys...)
}

// WithFrontmatterPrefix prepends a string to each extracted metadata key.
// For example, WithFrontmatterPrefix("doc_") turns "title" into "doc_title".
//
// Takes prefix (string) which is the string to prepend to each metadata key.
//
// Returns FrontmatterOption which configures the prefix behaviour.
func WithFrontmatterPrefix(prefix string) FrontmatterOption {
	return llm_domain.WithFrontmatterPrefix(prefix)
}

// NewRecursiveCharacterSplitter creates a [SplitterPort] that recursively
// splits documents using a hierarchy of separators ("\n\n", "\n", " ", "").
// Chunks target chunkSize bytes with overlap bytes repeated between chunks.
//
// Takes chunkSize (int) which is the target chunk size in bytes.
// Takes overlap (int) which is the number of bytes to repeat between chunks.
//
// Returns SplitterPort which splits documents into smaller chunks.
// Returns error when overlap is greater than or equal to chunkSize.
func NewRecursiveCharacterSplitter(chunkSize, overlap int) (SplitterPort, error) {
	return llm_domain.NewRecursiveCharacterSplitter(chunkSize, overlap)
}

// NewMarkdownSplitter creates a SplitterPort that splits markdown documents
// on heading boundaries using the goldmark AST parser.
//
// Headings inside fenced code blocks are correctly ignored. Sections exceeding
// chunkSize are sub-split using [NewRecursiveCharacterSplitter]. Each chunk
// receives a "heading" metadata key containing the nearest heading text.
//
// By default only h1 and h2 headings act as split boundaries. Use
// [WithMaxSplitLevel] to include deeper headings.
//
// Takes chunkSize (int) which is the maximum chunk size in bytes.
// Takes overlap (int) which is the character overlap for sub-split chunks.
// Takes opts (...MarkdownSplitterOption) which provides optional settings.
//
// Returns SplitterPort which splits markdown documents into chunks.
// Returns error when overlap is greater than or equal to chunkSize.
func NewMarkdownSplitter(chunkSize, overlap int, opts ...MarkdownSplitterOption) (SplitterPort, error) {
	return llm_domain.NewMarkdownSplitter(chunkSize, overlap, opts...)
}

// PrependChunkContext returns a [TransformFunc] intended for use as a
// post-split transform. It reads the "doc_title" and "heading" metadata keys
// and prepends them to the chunk content so that the embedding model can use
// the surrounding context for better semantic matching.
//
// Returns TransformFunc which prepends document title and heading metadata
// to chunk content for improved embedding context.
func PrependChunkContext() TransformFunc {
	return llm_domain.PrependChunkContext()
}

// NewRecursiveFSLoader creates a [LoaderPort] that recursively walks an
// [fs.FS] and loads files matching the given glob patterns.
//
// Takes fsys (fs.FS) which is the filesystem to walk.
// Takes patterns (...string) which are glob patterns to match file names.
// Patterns prefixed with "**/" walk the directory tree recursively.
//
// Returns LoaderPort which loads documents from the filesystem.
func NewRecursiveFSLoader(fsys fs.FS, patterns ...string) LoaderPort {
	return llm_domain.NewRecursiveFSLoader(fsys, patterns...)
}
