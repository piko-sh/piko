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
	"strings"
	"time"

	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/maths"
)

const (
	// maxRedactedContentLength is the maximum number of characters to keep in
	// redacted message content.
	maxRedactedContentLength = 50

	// fieldConversationID is the structured log field for conversation identifiers.
	fieldConversationID = "conversation_id"
)

// CompletionBuilder provides a fluent API for building and running LLM completion
// requests. Builder functions like Model, System, and User return the builder to
// allow method chaining.
type CompletionBuilder struct {
	// memory stores conversation history for multi-turn interactions.
	memory Memory

	// toolHandlers maps tool names to their handler functions for auto-dispatch.
	toolHandlers map[string]ToolHandlerFunc

	// request holds the completion request being built.
	request *llm_dto.CompletionRequest

	// ragConfig holds the configuration for automatic RAG context injection.
	// When set, Do() will embed the query, search the vector store, and inject
	// results before calling the LLM.
	ragConfig *ragConfig

	// service provides access to the LLM completion service for executing
	// requests.
	service *service

	// retryPolicy configures retry behaviour for failed requests; nil disables
	// retries.
	retryPolicy *llm_dto.RetryPolicy

	// fallbackConfig specifies providers to try if the primary provider fails.
	fallbackConfig *llm_dto.FallbackConfig

	// cacheConfig holds the cache settings for completions; nil disables caching.
	cacheConfig *llm_dto.CacheConfig

	// responseValidator is an optional post-processing function that validates
	// the final LLM response after tool dispatch. When set and it returns an
	// error, Do() fails with that error and the response is not recorded to
	// memory.
	responseValidator ResponseValidatorFunc

	// maxCost is the maximum cost allowed for this completion request.
	maxCost maths.Money

	// conversationID identifies the conversation for memory storage and retrieval.
	conversationID string

	// providerName specifies the LLM provider to use; empty uses the service
	// default.
	providerName string

	// originalQuery is the base query text before rewriting. Captured during
	// resolveRAGContext for RequestDump visibility.
	originalQuery string

	// budgetScope identifies the budget to track costs against.
	budgetScope string

	// vectorContext holds retrieved vector search results for RAG injection.
	vectorContext []llm_dto.VectorSearchResult

	// toolLoopMessages collects intermediate assistant and tool_result messages
	// generated during the tool dispatch loop, for recording to memory.
	toolLoopMessages []llm_dto.Message

	// rewrittenQueries holds queries produced by the rewriter. Captured during
	// resolveRAGContext for RequestDump visibility.
	rewrittenQueries []string

	// maxToolRounds is the maximum number of tool dispatch rounds. A value of
	// 0 uses DefaultMaxToolRounds; negative allows unlimited rounds.
	maxToolRounds int

	// ragResolved is true after resolveRAGContext has run, preventing duplicate
	// resolution when DryRun is called before Do.
	ragResolved bool

	// vectorInjected is true after injectVectorContext has run, preventing
	// duplicate message injection when DryRun is called before Do.
	vectorInjected bool
}

// Model sets the model to use for the completion.
//
// Takes model (string) which identifies the model to use
// (e.g., "gpt-4o", "claude-3-5-sonnet").
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) Model(model string) *CompletionBuilder {
	b.request.Model = model
	return b
}

// System adds a system message to the conversation.
//
// Takes content (string) which sets the system prompt or context.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) System(content string) *CompletionBuilder {
	b.request.Messages = append(b.request.Messages, llm_dto.NewSystemMessage(content))
	return b
}

// User adds a user message to the conversation.
//
// Takes content (string) which contains the user's input.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) User(content string) *CompletionBuilder {
	b.request.Messages = append(b.request.Messages, llm_dto.NewUserMessage(content))
	return b
}

// UserWithImage adds a user message with text and an image URL. This enables
// vision/multi-modal requests where an image is sent alongside text.
//
// Takes text (string) which is the text content of the message.
// Takes imageURL (string) which is the URL of the image to include.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) UserWithImage(text, imageURL string) *CompletionBuilder {
	b.request.Messages = append(b.request.Messages, llm_dto.NewUserMessageWithImageURL(text, imageURL))
	return b
}

// UserWithImageData adds a user message with text and inline image data.
// This enables vision and multi-modal requests where an inline image is sent
// alongside text.
//
// Takes text (string) which is the text content of the message.
// Takes mimeType (string) which is the MIME type of the image (e.g.,
// "image/png").
// Takes base64Data (string) which is the base64-encoded image data.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) UserWithImageData(text, mimeType, base64Data string) *CompletionBuilder {
	b.request.Messages = append(b.request.Messages, llm_dto.NewUserMessageWithImageData(text, mimeType, base64Data))
	return b
}

// UserWithImages adds a user message with text and multiple images.
// This enables vision and multi-modal requests where multiple images are sent
// alongside text.
//
// Takes text (string) which is the text content of the message.
// Takes images (...llm_dto.ContentPart) which are the image content parts to
// include.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) UserWithImages(text string, images ...llm_dto.ContentPart) *CompletionBuilder {
	b.request.Messages = append(b.request.Messages, llm_dto.NewUserMessageWithImages(text, images...))
	return b
}

// Assistant adds an assistant message to the conversation.
//
// Takes content (string) which contains the assistant's response.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) Assistant(content string) *CompletionBuilder {
	b.request.Messages = append(b.request.Messages, llm_dto.NewAssistantMessage(content))
	return b
}

// Messages adds multiple messages to the conversation.
//
// Takes messages (...llm_dto.Message) which are the messages to add.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) Messages(messages ...llm_dto.Message) *CompletionBuilder {
	b.request.Messages = append(b.request.Messages, messages...)
	return b
}

// ToolResult adds a tool result message to the conversation.
//
// Takes toolCallID (string) which identifies the tool call being responded to.
// Takes content (string) which contains the result from the tool execution.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) ToolResult(toolCallID, content string) *CompletionBuilder {
	b.request.Messages = append(b.request.Messages, llm_dto.NewToolResultMessage(toolCallID, content))
	return b
}

// Temperature sets the temperature for generation.
//
// Takes t (float64) which controls randomness. Range: 0 to 2.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) Temperature(t float64) *CompletionBuilder {
	b.request.Temperature = &t
	return b
}

// MaxTokens sets the maximum number of tokens to generate.
//
// Takes n (int) which limits the response length.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) MaxTokens(n int) *CompletionBuilder {
	b.request.MaxTokens = &n
	return b
}

// TopP sets the nucleus sampling parameter.
//
// Takes p (float64) which controls diversity. Range: 0 to 1.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) TopP(p float64) *CompletionBuilder {
	b.request.TopP = &p
	return b
}

// FrequencyPenalty sets the frequency penalty.
//
// Takes penalty (float64) which reduces repetition. Range: -2 to 2.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) FrequencyPenalty(penalty float64) *CompletionBuilder {
	b.request.FrequencyPenalty = &penalty
	return b
}

// PresencePenalty sets the presence penalty.
//
// Takes penalty (float64) which reduces repetition. Range: -2 to 2.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) PresencePenalty(penalty float64) *CompletionBuilder {
	b.request.PresencePenalty = &penalty
	return b
}

// Stop sets sequences where the model will stop generating.
//
// Takes sequences (...string) which are stop sequences.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) Stop(sequences ...string) *CompletionBuilder {
	b.request.Stop = sequences
	return b
}

// Seed sets the random seed for deterministic generation.
//
// Takes seed (int64) which enables reproducible outputs when supported.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) Seed(seed int64) *CompletionBuilder {
	b.request.Seed = &seed
	return b
}

// Tool adds a function tool definition.
//
// Takes name (string) which is the function name.
// Takes description (string) which explains what the function does.
// Takes params (*llm_dto.JSONSchema) which describes the function parameters.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) Tool(name, description string, params *llm_dto.JSONSchema) *CompletionBuilder {
	b.request.Tools = append(b.request.Tools, llm_dto.NewFunctionTool(name, description, params))
	return b
}

// StrictTool adds a function tool with strict schema enforcement.
//
// Takes name (string) which is the function name.
// Takes description (string) which explains what the function does.
// Takes params (*llm_dto.JSONSchema) which describes the function parameters.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) StrictTool(name, description string, params *llm_dto.JSONSchema) *CompletionBuilder {
	b.request.Tools = append(b.request.Tools, llm_dto.NewStrictFunctionTool(name, description, params))
	return b
}

// Tools adds multiple tool definitions.
//
// Takes tools (...llm_dto.ToolDefinition) which are the tools to add.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) Tools(tools ...llm_dto.ToolDefinition) *CompletionBuilder {
	b.request.Tools = append(b.request.Tools, tools...)
	return b
}

// ToolChoice sets how the model should use tools.
//
// Takes choice (*llm_dto.ToolChoice) which controls tool selection.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) ToolChoice(choice *llm_dto.ToolChoice) *CompletionBuilder {
	b.request.ToolChoice = choice
	return b
}

// ParallelToolCalls enables or disables parallel tool calls.
//
// Takes enabled (bool) which controls parallel execution.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) ParallelToolCalls(enabled bool) *CompletionBuilder {
	b.request.ParallelToolCalls = &enabled
	return b
}

// JSONResponse sets the response format to JSON without schema constraints.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) JSONResponse() *CompletionBuilder {
	b.request.ResponseFormat = llm_dto.ResponseFormatJSON()
	return b
}

// StructuredResponse requests JSON output conforming to a schema.
//
// Takes name (string) which identifies the schema.
// Takes schema (llm_dto.JSONSchema) which defines the required structure.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) StructuredResponse(name string, schema llm_dto.JSONSchema) *CompletionBuilder {
	b.request.ResponseFormat = llm_dto.ResponseFormatStructured(name, schema)
	return b
}

// ResponseFormat sets a custom response format.
//
// Takes format (*llm_dto.ResponseFormat) which specifies the output format.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) ResponseFormat(format *llm_dto.ResponseFormat) *CompletionBuilder {
	b.request.ResponseFormat = format
	return b
}

// Provider sets which registered provider to use.
//
// Takes name (string) which identifies the provider.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) Provider(name string) *CompletionBuilder {
	b.providerName = name
	return b
}

// ProviderOption sets a provider-specific option.
//
// Takes key (string) which identifies the option.
// Takes value (any) which is the option value.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) ProviderOption(key string, value any) *CompletionBuilder {
	if b.request.ProviderOptions == nil {
		b.request.ProviderOptions = make(map[string]any)
	}
	b.request.ProviderOptions[key] = value
	return b
}

// Metadata adds metadata for tracking and logging.
//
// Takes key (string) which identifies the metadata.
// Takes value (string) which is the metadata value.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) Metadata(key, value string) *CompletionBuilder {
	if b.request.Metadata == nil {
		b.request.Metadata = make(map[string]string)
	}
	b.request.Metadata[key] = value
	return b
}

// BudgetScope sets the budget scope for this request.
// Costs will be tracked and limits enforced against this scope.
//
// Takes scope (string) which identifies the budget scope, such as
// "global" or "user:123".
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) BudgetScope(scope string) *CompletionBuilder {
	b.budgetScope = scope
	return b
}

// MaxCost sets a per-request cost limit. The request will fail with
// ErrMaxCostExceeded if the estimated cost exceeds this limit.
//
// Takes maxCost (maths.Money) which is the maximum cost.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) MaxCost(maxCost maths.Money) *CompletionBuilder {
	b.maxCost = maxCost
	return b
}

// Retry configures retry behaviour for the completion request.
// Retries are triggered for transient errors such as rate limits, timeouts,
// and provider overload conditions.
//
// Takes policy (*llm_dto.RetryPolicy) which configures the retry behaviour.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) Retry(policy *llm_dto.RetryPolicy) *CompletionBuilder {
	b.retryPolicy = policy
	return b
}

// DefaultRetry enables retry with the default policy.
// The default policy uses exponential backoff starting at 500ms, doubling each
// attempt up to 8s maximum, with 10% jitter and a maximum of 3 retries.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) DefaultRetry() *CompletionBuilder {
	b.retryPolicy = llm_dto.DefaultRetryPolicy()
	return b
}

// Fallback configures fallback behaviour for the completion request. When the
// primary provider fails, the request will be retried with subsequent providers
// in the configured order.
//
// Takes config (*llm_dto.FallbackConfig) which configures the fallback
// behaviour.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) Fallback(config *llm_dto.FallbackConfig) *CompletionBuilder {
	b.fallbackConfig = config
	return b
}

// FallbackProviders is a convenience method to configure fallback with a list
// of provider names. The first provider is used as the primary, with subsequent
// providers used as fallbacks in order.
//
// Takes providers (...string) which are the provider names in priority order.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) FallbackProviders(providers ...string) *CompletionBuilder {
	b.fallbackConfig = llm_dto.NewFallbackConfig(providers...)
	return b
}

// Cache enables response caching with the specified TTL. Cached responses will
// be returned for identical requests within the TTL period.
//
// Takes ttl (time.Duration) which is the time-to-live for cached responses.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) Cache(ttl time.Duration) *CompletionBuilder {
	b.cacheConfig = &llm_dto.CacheConfig{
		Enabled:          true,
		TTL:              ttl,
		Key:              "",
		SkipWrite:        false,
		SkipRead:         false,
		UseProviderCache: false,
	}
	return b
}

// CacheConfig enables response caching with the specified configuration.
//
// Takes config (*llm_dto.CacheConfig) which configures caching behaviour.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) CacheConfig(config *llm_dto.CacheConfig) *CompletionBuilder {
	b.cacheConfig = config
	return b
}

// ProviderCache enables provider-specific caching.
//
// This uses the provider's native caching mechanism (e.g., Anthropic
// prompt caching) rather than the local cache.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) ProviderCache() *CompletionBuilder {
	if b.cacheConfig == nil {
		b.cacheConfig = &llm_dto.CacheConfig{
			Enabled:          true,
			TTL:              0,
			Key:              "",
			SkipWrite:        false,
			SkipRead:         false,
			UseProviderCache: false,
		}
	}
	b.cacheConfig.UseProviderCache = true
	return b
}

// Memory enables conversation memory for this request.
// Messages from previous interactions in the same conversation will be
// automatically prepended to the request, and the response will be recorded.
//
// Takes memory (Memory) which is the memory implementation to use.
// Takes conversationID (string) which identifies the conversation.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) Memory(memory Memory, conversationID string) *CompletionBuilder {
	b.memory = memory
	b.conversationID = conversationID
	return b
}

// BufferMemory enables buffer memory for this request. This is a convenience
// method that creates a BufferMemory with the given store.
//
// Takes store (MemoryStorePort) which handles persistence.
// Takes conversationID (string) which identifies the conversation.
// Takes size (int) which is the maximum number of messages to keep.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) BufferMemory(store MemoryStorePort, conversationID string, size int) *CompletionBuilder {
	b.memory = NewBufferMemory(store, WithBufferSize(size))
	b.conversationID = conversationID
	return b
}

// ResponseValidator sets a post-processing validation function that runs after
// the LLM produces its final response (after tool dispatch completes). If the
// validator returns a non-nil error, Do() fails with that error and the
// response is not recorded to memory.
//
// Takes validatorFunction (ResponseValidatorFunc) which validates the
// completion response.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) ResponseValidator(validatorFunction ResponseValidatorFunc) *CompletionBuilder {
	b.responseValidator = validatorFunction
	return b
}

// WithVectorContext injects retrieved vector search results as context into
// the completion request. The results are prepended as a system message
// summarising the retrieved documents, enabling retrieval-augmented generation
// (RAG).
//
// Takes results ([]llm_dto.VectorSearchResult) which are the documents to
// include as context.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) WithVectorContext(results []llm_dto.VectorSearchResult) *CompletionBuilder {
	if len(results) == 0 {
		return b
	}
	b.vectorContext = results
	return b
}

// Build returns the configured request without running it.
//
// Returns llm_dto.CompletionRequest which contains the set parameters.
func (b *CompletionBuilder) Build() llm_dto.CompletionRequest {
	return *b.request
}

// DryRun resolves RAG context and assembles the full request without calling
// the LLM provider. Use this to inspect exactly what would be sent to the
// model.
//
// Returns *RequestDump which contains the final messages, retrieved sources,
// and all request parameters.
func (b *CompletionBuilder) DryRun(ctx context.Context) *RequestDump {
	b.resolveRAGContext(ctx)
	b.injectVectorContext()

	return &RequestDump{
		Timestamp:        b.service.clock.Now(),
		Model:            b.request.Model,
		Provider:         b.resolveProviderName(),
		MaxTokens:        b.request.MaxTokens,
		Temperature:      b.request.Temperature,
		Messages:         b.request.Messages,
		Sources:          b.vectorContext,
		Tools:            b.request.Tools,
		OriginalQuery:    b.originalQuery,
		RewrittenQueries: b.rewrittenQueries,
	}
}

// DryRunRedacted returns the same data as DryRun but truncates message
// content to a maximum of 50 characters and replaces tool arguments with
// "[redacted]". Use this in production to avoid leaking sensitive data.
//
// Returns *RequestDump with redacted content.
func (b *CompletionBuilder) DryRunRedacted(ctx context.Context) *RequestDump {
	dump := b.DryRun(ctx)

	redacted := make([]llm_dto.Message, len(dump.Messages))
	for i, message := range dump.Messages {
		redacted[i] = message
		if len(message.Content) > maxRedactedContentLength {
			redacted[i].Content = message.Content[:maxRedactedContentLength] + "..."
		}
	}
	dump.Messages = redacted

	return dump
}

// Do executes the completion request and returns the response.
//
// Returns *llm_dto.CompletionResponse which contains the model's response.
// Returns error when the request fails.
func (b *CompletionBuilder) Do(ctx context.Context) (*llm_dto.CompletionResponse, error) {
	ctx, l := logger_domain.From(ctx, log)
	var response *llm_dto.CompletionResponse

	err := l.RunInSpan(ctx, "CompletionBuilder.Do", func(spanCtx context.Context, _ logger_domain.Logger) error {
		start := b.service.clock.Now()
		defer func() {
			builderCompleteDuration.Record(spanCtx, float64(b.service.clock.Now().Sub(start).Milliseconds()))
		}()
		builderCompleteCount.Add(spanCtx, 1)

		var err error
		response, err = b.executePipeline(spanCtx)
		return err
	})

	return response, err
}

// Stream executes the completion request with streaming enabled.
//
// Returns <-chan llm_dto.StreamEvent which emits streaming events.
// Returns error when the stream cannot be started.
func (b *CompletionBuilder) Stream(ctx context.Context) (<-chan llm_dto.StreamEvent, error) {
	b.request.Stream = true
	if b.providerName != "" {
		return b.service.StreamWithProvider(ctx, b.providerName, b.request)
	}
	return b.service.Stream(ctx, b.request)
}

// executePipeline runs the full completion pipeline: load history, resolve
// RAG context, execute with caching, run tool loop, validate, and record to
// memory.
//
// Returns *llm_dto.CompletionResponse which contains the model's response.
// Returns error when any pipeline step fails or the context is cancelled.
func (b *CompletionBuilder) executePipeline(ctx context.Context) (*llm_dto.CompletionResponse, error) {
	providerName := b.resolveProviderName()
	originalMessages := b.request.Messages

	if err := b.prepareContext(ctx); err != nil {
		return nil, err
	}

	response, err := b.runCompletionAndTools(ctx, providerName)
	if err != nil {
		return nil, err
	}

	if err := b.validateResponse(ctx, response); err != nil {
		return nil, err
	}

	b.recordToMemory(ctx, originalMessages, response)

	if len(b.vectorContext) > 0 {
		response.Sources = b.vectorContext
	}

	return response, nil
}

// prepareContext loads conversation history, resolves RAG context, and injects
// vector context into the request messages.
//
// Returns error when the context is cancelled during preparation.
func (b *CompletionBuilder) prepareContext(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled before loading conversation history: %w", err)
	}
	b.loadConversationHistory(ctx)

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled before resolving RAG context: %w", err)
	}
	b.resolveRAGContext(ctx)
	b.injectVectorContext()

	return nil
}

// runCompletionAndTools executes the completion with caching and runs the
// tool dispatch loop.
//
// Takes providerName (string) which identifies the LLM provider to use.
//
// Returns *llm_dto.CompletionResponse which contains the completion result.
// Returns error when the completion or tool loop fails.
func (b *CompletionBuilder) runCompletionAndTools(ctx context.Context, providerName string) (*llm_dto.CompletionResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled before executing completion: %w", err)
	}

	response, err := b.executeWithCaching(ctx, providerName)
	if err != nil {
		builderCompleteErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("executing completion with caching: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled before executing tool loop: %w", err)
	}

	response, err = b.executeToolLoop(ctx, providerName, response)
	if err != nil {
		builderCompleteErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("executing tool loop: %w", err)
	}

	return response, nil
}

// validateResponse runs the optional response validator and records an error
// metric on failure.
//
// Takes response (*llm_dto.CompletionResponse) which is the response to validate.
//
// Returns error when validation fails.
func (b *CompletionBuilder) validateResponse(ctx context.Context, response *llm_dto.CompletionResponse) error {
	if b.responseValidator == nil {
		return nil
	}
	if err := b.responseValidator(ctx, response); err != nil {
		builderCompleteErrorCount.Add(ctx, 1)
		return fmt.Errorf("response validation failed: %w", err)
	}
	return nil
}

// resolveProviderName returns the provider name to use, defaulting to the
// service default.
//
// Returns string which is the configured provider name or the service default.
func (b *CompletionBuilder) resolveProviderName() string {
	if b.providerName != "" {
		return b.providerName
	}
	return b.service.defaultProvider
}

// loadConversationHistory prepends conversation history to the request if
// memory is configured.
func (b *CompletionBuilder) loadConversationHistory(ctx context.Context) {
	if b.memory == nil || b.conversationID == "" {
		return
	}

	ctx, l := logger_domain.From(ctx, log)
	historyMessages, err := b.memory.GetMessages(ctx, b.conversationID)
	if err != nil {
		l.Debug("Failed to load conversation history",
			logger_domain.String(fieldConversationID, b.conversationID),
			logger_domain.Error(err),
		)
		return
	}

	if len(historyMessages) == 0 {
		return
	}

	allMessages := make([]llm_dto.Message, 0, len(historyMessages)+len(b.request.Messages))
	allMessages = append(allMessages, historyMessages...)
	allMessages = append(allMessages, b.request.Messages...)
	b.request.Messages = allMessages
}

// injectVectorContext prepends a system message containing the retrieved
// vector search results. This enables retrieval-augmented generation by
// providing relevant context from the vector store to the model.
func (b *CompletionBuilder) injectVectorContext() {
	if len(b.vectorContext) == 0 || b.vectorInjected {
		return
	}
	b.vectorInjected = true

	var builder strings.Builder
	for i, result := range b.vectorContext {
		if i > 0 {
			builder.WriteString("\n\n---\n\n")
		}
		_, _ = fmt.Fprintf(&builder, "[Document %d (score: %.2f)]\n%s", i+1, result.Score, result.Content)
	}

	contextMessage := llm_dto.NewSystemMessage(
		"The following documents were retrieved from the knowledge base and may be relevant to the user's query:\n\n" + builder.String(),
	)

	messages := make([]llm_dto.Message, 0, len(b.request.Messages)+1)
	messages = append(messages, contextMessage)
	messages = append(messages, b.request.Messages...)
	b.request.Messages = messages
}

// executeWithCaching executes the request, using cache if configured.
//
// Takes providerName (string) which identifies the LLM provider to use.
//
// Returns *llm_dto.CompletionResponse which contains the completion result.
// Returns error when the execution or cache retrieval fails.
func (b *CompletionBuilder) executeWithCaching(ctx context.Context, providerName string) (*llm_dto.CompletionResponse, error) {
	execute := b.createExecuteFunc(ctx, providerName)

	if b.cacheConfig != nil && b.cacheConfig.Enabled && b.service.cacheManager != nil {
		response, _, err := b.service.cacheManager.GetOrExecute(ctx, b.cacheConfig, b.request, providerName, execute)
		return response, err
	}

	return execute()
}

// createExecuteFunc creates the execution function with fallback/retry
// handling.
//
// Takes providerName (string) which identifies the LLM provider to use.
//
// Returns func() (*llm_dto.CompletionResponse, error) which executes the
// completion request with configured fallback and retry behaviour.
func (b *CompletionBuilder) createExecuteFunc(ctx context.Context, providerName string) func() (*llm_dto.CompletionResponse, error) {
	return func() (*llm_dto.CompletionResponse, error) {
		if b.fallbackConfig != nil && len(b.fallbackConfig.Providers) > 0 {
			return b.executeWithFallback(ctx)
		}

		if b.retryPolicy != nil && b.retryPolicy.MaxRetries > 0 {
			return b.executeWithRetry(ctx, providerName)
		}

		return b.service.completeWithScope(ctx, providerName, b.request, b.budgetScope, b.maxCost)
	}
}

// executeWithFallback executes the request with fallback routing.
//
// Returns *llm_dto.CompletionResponse which contains the completion result
// with fallback information attached if a fallback was used.
// Returns error when the request fails on all configured providers.
func (b *CompletionBuilder) executeWithFallback(ctx context.Context) (*llm_dto.CompletionResponse, error) {
	router := NewFallbackRouter(b.service)
	response, fallbackResult, err := router.Execute(ctx, b.fallbackConfig, b.request, b.budgetScope, b.maxCost, b.retryPolicy)
	if response != nil && fallbackResult != nil && fallbackResult.WasFallbackUsed() {
		response.FallbackInfo = fallbackResult
	}
	return response, err
}

// executeWithRetry executes the request with retry logic.
//
// Takes providerName (string) which specifies the LLM provider to use.
//
// Returns *llm_dto.CompletionResponse which contains the completion result.
// Returns error when the request fails after all retry attempts.
func (b *CompletionBuilder) executeWithRetry(ctx context.Context, providerName string) (*llm_dto.CompletionResponse, error) {
	executor := NewRetryExecutor(b.retryPolicy, WithRetryExecutorClock(b.service.clock))
	return ExecuteWithResult(ctx, executor, func() (*llm_dto.CompletionResponse, error) {
		return b.service.completeWithScope(ctx, providerName, b.request, b.budgetScope, b.maxCost)
	})
}

// recordToMemory records messages to memory if configured.
//
// Takes originalMessages ([]llm_dto.Message) which contains the user messages
// to record.
// Takes response (*llm_dto.CompletionResponse) which contains the assistant
// response to record.
func (b *CompletionBuilder) recordToMemory(ctx context.Context, originalMessages []llm_dto.Message, response *llm_dto.CompletionResponse) {
	if b.memory == nil || b.conversationID == "" || response == nil {
		return
	}

	ctx, l := logger_domain.From(ctx, log)

	for _, message := range originalMessages {
		if err := b.memory.AddMessage(ctx, b.conversationID, message); err != nil {
			l.Debug("Failed to record user message to memory",
				logger_domain.String(fieldConversationID, b.conversationID),
				logger_domain.Error(err),
			)
		}
	}

	for _, message := range b.toolLoopMessages {
		if err := b.memory.AddMessage(ctx, b.conversationID, message); err != nil {
			l.Debug("Failed to record tool loop message to memory",
				logger_domain.String(fieldConversationID, b.conversationID),
				logger_domain.Error(err),
			)
		}
	}

	b.recordAssistantResponse(ctx, response)
}

// recordAssistantResponse records the assistant's response to memory.
//
// Takes response (*llm_dto.CompletionResponse) which contains the assistant's
// response content and any tool calls to store.
func (b *CompletionBuilder) recordAssistantResponse(ctx context.Context, response *llm_dto.CompletionResponse) {
	content := response.Content()
	if content == "" {
		return
	}

	ctx, l := logger_domain.From(ctx, log)

	assistantMessage := llm_dto.NewAssistantMessage(content)
	if response.HasToolCalls() {
		assistantMessage.ToolCalls = response.ToolCalls()
	}

	if err := b.memory.AddMessage(ctx, b.conversationID, assistantMessage); err != nil {
		l.Debug("Failed to record assistant message to memory",
			logger_domain.String(fieldConversationID, b.conversationID),
			logger_domain.Error(err),
		)
	}
}
