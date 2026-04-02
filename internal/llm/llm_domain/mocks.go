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
	"sync"
	"sync/atomic"
	"time"

	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/wdk/maths"
)

const (
	// mockPromptTokens is the default prompt token count in mock responses.
	mockPromptTokens = 10

	// mockCompletionTokens is the default completion token count in mock responses.
	mockCompletionTokens = 5

	// mockTotalTokens is the default total token count in mock responses.
	mockTotalTokens = 15

	// mockEmbedDim1 is the first dimension of mock embedding vectors.
	mockEmbedDim1 = 0.1

	// mockEmbedDim2 is the second dimension of mock embedding vectors.
	mockEmbedDim2 = 0.2

	// mockEmbedDim3 is the third dimension of mock embedding vectors.
	mockEmbedDim3 = 0.3

	// mockTokensPerInput is the token multiplier per input for mock embedding usage.
	mockTokensPerInput = 5
)

// MockLLMProvider is a test double for LLMProviderPort that returns
// zero values from nil function fields and tracks call counts
// atomically.
type MockLLMProvider struct {
	// CompleteFunc is the function called by Complete.
	CompleteFunc func(ctx context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error)

	// StreamFunc is the function called by Stream.
	StreamFunc func(ctx context.Context, request *llm_dto.CompletionRequest) (<-chan llm_dto.StreamEvent, error)

	// ListModelsFunc is the function called by
	// ListModels.
	ListModelsFunc func(ctx context.Context) ([]llm_dto.ModelInfo, error)

	// CloseFunc is the function called by Close.
	CloseFunc func(ctx context.Context) error

	// DefaultModelValue is the model name returned by
	// DefaultModel.
	DefaultModelValue string

	// CompleteCallCount tracks how many times Complete
	// was called.
	CompleteCallCount int64

	// StreamCallCount tracks how many times Stream was
	// called.
	StreamCallCount int64

	// SupportsStreamingValue is the value returned by
	// SupportsStreaming.
	SupportsStreamingValue bool

	// SupportsStructuredValue is the value returned by
	// SupportsStructuredOutput.
	SupportsStructuredValue bool

	// SupportsToolsValue is the value returned by
	// SupportsTools.
	SupportsToolsValue bool

	// SupportsPenaltiesValue is the value returned by SupportsPenalties.
	SupportsPenaltiesValue bool

	// SupportsSeedValue is the value returned by SupportsSeed.
	SupportsSeedValue bool

	// SupportsParallelToolCallsValue is the value returned by
	// SupportsParallelToolCalls.
	SupportsParallelToolCallsValue bool

	// SupportsMessageNameValue is the value returned by SupportsMessageName.
	SupportsMessageNameValue bool
}

// NewMockLLMProvider creates a new MockLLMProvider with default settings.
//
// Returns *MockLLMProvider which is configured with all capabilities enabled.
func NewMockLLMProvider() *MockLLMProvider {
	return &MockLLMProvider{
		SupportsStreamingValue:  true,
		SupportsStructuredValue: true,
		SupportsToolsValue:      true,
	}
}

// Complete delegates to CompleteFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes request (*llm_dto.CompletionRequest) which contains the completion request
// parameters.
//
// Returns (zero, nil) if CompleteFunc is nil.
func (m *MockLLMProvider) Complete(ctx context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
	atomic.AddInt64(&m.CompleteCallCount, 1)

	if m.CompleteFunc != nil {
		return m.CompleteFunc(ctx, request)
	}
	return &llm_dto.CompletionResponse{
		ID:      "mock-response-id",
		Model:   request.Model,
		Created: time.Now().Unix(),
		Choices: []llm_dto.Choice{
			{
				Index: 0,
				Message: llm_dto.Message{
					Role:    llm_dto.RoleAssistant,
					Content: "Mock response",
				},
				FinishReason: "stop",
			},
		},
		Usage: &llm_dto.Usage{
			PromptTokens:     mockPromptTokens,
			CompletionTokens: mockCompletionTokens,
			TotalTokens:      mockTotalTokens,
		},
	}, nil
}

// Stream delegates to StreamFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes request (*llm_dto.CompletionRequest) which contains the completion request
// parameters.
//
// Returns (zero, nil) if StreamFunc is nil.
//
// Concurrent; spawns a goroutine that sends a done event and closes the
// channel when StreamFunc is nil.
func (m *MockLLMProvider) Stream(ctx context.Context, request *llm_dto.CompletionRequest) (<-chan llm_dto.StreamEvent, error) {
	atomic.AddInt64(&m.StreamCallCount, 1)

	if m.StreamFunc != nil {
		return m.StreamFunc(ctx, request)
	}

	eventChannel := make(chan llm_dto.StreamEvent, 1)
	go func() {
		defer close(eventChannel)
		eventChannel <- llm_dto.NewDoneEvent(nil)
	}()
	return eventChannel, nil
}

// SupportsStreaming implements LLMProviderPort.SupportsStreaming.
//
// Returns bool which indicates whether this provider supports streaming.
func (m *MockLLMProvider) SupportsStreaming() bool {
	return m.SupportsStreamingValue
}

// SupportsStructuredOutput implements LLMProviderPort.SupportsStructuredOutput.
//
// Returns bool which indicates whether this provider supports structured
// output.
func (m *MockLLMProvider) SupportsStructuredOutput() bool {
	return m.SupportsStructuredValue
}

// SupportsTools implements LLMProviderPort.SupportsTools.
//
// Returns bool which indicates whether the provider supports tool calling.
func (m *MockLLMProvider) SupportsTools() bool {
	return m.SupportsToolsValue
}

// SupportsPenalties implements LLMProviderPort.SupportsPenalties.
//
// Returns bool which indicates whether the provider supports penalties.
func (m *MockLLMProvider) SupportsPenalties() bool {
	return m.SupportsPenaltiesValue
}

// SupportsSeed implements LLMProviderPort.SupportsSeed.
//
// Returns bool which indicates whether the provider supports seed.
func (m *MockLLMProvider) SupportsSeed() bool {
	return m.SupportsSeedValue
}

// SupportsParallelToolCalls implements LLMProviderPort.SupportsParallelToolCalls.
//
// Returns bool which indicates whether parallel tool calls are supported.
func (m *MockLLMProvider) SupportsParallelToolCalls() bool {
	return m.SupportsParallelToolCallsValue
}

// SupportsMessageName implements LLMProviderPort.SupportsMessageName.
//
// Returns bool which indicates whether message names are supported.
func (m *MockLLMProvider) SupportsMessageName() bool {
	return m.SupportsMessageNameValue
}

// ListModels delegates to ListModelsFunc if set.
//
// Returns (zero, nil) if ListModelsFunc is nil.
func (m *MockLLMProvider) ListModels(ctx context.Context) ([]llm_dto.ModelInfo, error) {
	if m.ListModelsFunc != nil {
		return m.ListModelsFunc(ctx)
	}
	return []llm_dto.ModelInfo{
		{ID: "mock-model", Name: "Mock Model"},
	}, nil
}

// Close delegates to CloseFunc if set.
//
// Returns nil if CloseFunc is nil.
func (m *MockLLMProvider) Close(ctx context.Context) error {
	if m.CloseFunc != nil {
		return m.CloseFunc(ctx)
	}
	return nil
}

// DefaultModel implements LLMProviderPort.DefaultModel.
//
// Returns string which is the configured default model name.
func (m *MockLLMProvider) DefaultModel() string {
	return m.DefaultModelValue
}

// SetResponse sets a fixed response for the Complete method.
//
// Takes response (*llm_dto.CompletionResponse) which is the response to return.
func (m *MockLLMProvider) SetResponse(response *llm_dto.CompletionResponse) {
	m.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		return response, nil
	}
}

// MockEmbeddingProvider is a test double for EmbeddingProviderPort that
// returns zero values from nil function fields and tracks call counts
// atomically.
type MockEmbeddingProvider struct {
	// EmbedFunc is called when Embed is invoked; if nil, a default response is
	// used.
	EmbedFunc func(ctx context.Context, request *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error)

	// ListModelsFunc is called by ListEmbeddingModels when set; nil uses the
	// default behaviour.
	ListModelsFunc func(ctx context.Context) ([]llm_dto.ModelInfo, error)

	// CloseFunc is called by Close to release resources; if nil, Close returns
	// nil.
	CloseFunc func(ctx context.Context) error

	// EmbeddingDimensionsFunc is called by EmbeddingDimensions; if nil, returns 0.
	EmbeddingDimensionsFunc func() int

	// EmbedCallCount tracks the number of calls to Embed.
	EmbedCallCount int64
}

// NewMockEmbeddingProvider creates a new MockEmbeddingProvider.
//
// Returns *MockEmbeddingProvider which is ready for use in tests.
func NewMockEmbeddingProvider() *MockEmbeddingProvider {
	return &MockEmbeddingProvider{}
}

// SetEmbedFunc sets the implementation for the Embed method.
//
// Takes f (func(...)) which provides the function to call when Embed is
// invoked.
func (m *MockEmbeddingProvider) SetEmbedFunc(f func(ctx context.Context, request *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error)) {
	m.EmbedFunc = f
}

// Embed delegates to EmbedFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes request (*llm_dto.EmbeddingRequest) which contains the embedding request
// parameters.
//
// Returns (zero, nil) if EmbedFunc is nil.
func (m *MockEmbeddingProvider) Embed(ctx context.Context, request *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
	atomic.AddInt64(&m.EmbedCallCount, 1)

	if m.EmbedFunc != nil {
		return m.EmbedFunc(ctx, request)
	}
	embeddings := make([]llm_dto.Embedding, len(request.Input))
	for i := range request.Input {
		embeddings[i] = llm_dto.Embedding{
			Index:  i,
			Vector: []float32{mockEmbedDim1, mockEmbedDim2, mockEmbedDim3},
		}
	}
	return &llm_dto.EmbeddingResponse{
		Model:      request.Model,
		Embeddings: embeddings,
		Usage: &llm_dto.EmbeddingUsage{
			PromptTokens: len(request.Input) * mockTokensPerInput,
			TotalTokens:  len(request.Input) * mockTokensPerInput,
		},
	}, nil
}

// ListEmbeddingModels delegates to ListModelsFunc if set.
//
// Returns (zero, nil) if ListModelsFunc is nil.
func (m *MockEmbeddingProvider) ListEmbeddingModels(ctx context.Context) ([]llm_dto.ModelInfo, error) {
	if m.ListModelsFunc != nil {
		return m.ListModelsFunc(ctx)
	}
	return []llm_dto.ModelInfo{
		{ID: "mock-embedding-model", Name: "Mock Embedding Model"},
	}, nil
}

// EmbeddingDimensions delegates to EmbeddingDimensionsFunc if set.
//
// Returns 0 if EmbeddingDimensionsFunc is nil.
func (m *MockEmbeddingProvider) EmbeddingDimensions() int {
	if m.EmbeddingDimensionsFunc != nil {
		return m.EmbeddingDimensionsFunc()
	}
	return 0
}

// Close delegates to CloseFunc if set.
//
// Returns nil if CloseFunc is nil.
func (m *MockEmbeddingProvider) Close(ctx context.Context) error {
	if m.CloseFunc != nil {
		return m.CloseFunc(ctx)
	}
	return nil
}

// MockCacheStore is a mock implementation of the cache store for testing.
type MockCacheStore struct {
	// entries stores cached items by their string key.
	entries map[string]*llm_dto.CacheEntry

	// GetFunc overrides the default Get behaviour when set.
	GetFunc func(ctx context.Context, key string) (*llm_dto.CacheEntry, error)

	// SetFunc is the mock implementation for the Set method.
	SetFunc func(ctx context.Context, key string, entry *llm_dto.CacheEntry) error

	// mu guards concurrent access to mock function fields.
	mu sync.Mutex

	// hits counts the number of cache lookups that found an entry.
	hits int64

	// misses counts the number of cache lookups that found no entry.
	misses int64
}

// NewMockCacheStore creates a new MockCacheStore.
//
// Returns *MockCacheStore which is ready for use with an empty cache.
func NewMockCacheStore() *MockCacheStore {
	return &MockCacheStore{
		entries: make(map[string]*llm_dto.CacheEntry),
	}
}

// Get retrieves a cache entry by key, implementing the cache store interface.
//
// Takes key (string) which identifies the cache entry to retrieve.
//
// Returns *llm_dto.CacheEntry which is the cached entry, or nil if not found.
// Returns error when the underlying GetFunc fails.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockCacheStore) Get(ctx context.Context, key string) (*llm_dto.CacheEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.GetFunc != nil {
		return m.GetFunc(ctx, key)
	}

	entry, exists := m.entries[key]
	if !exists {
		m.misses++
		return nil, nil
	}
	m.hits++
	return entry, nil
}

// Set implements the cache store Set method.
//
// Takes key (string) which identifies the cache entry.
// Takes entry (*llm_dto.CacheEntry) which contains the data to store.
//
// Returns error when the custom SetFunc fails.
//
// Safe for concurrent use; access is protected by a mutex.
func (m *MockCacheStore) Set(ctx context.Context, key string, entry *llm_dto.CacheEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.SetFunc != nil {
		return m.SetFunc(ctx, key, entry)
	}

	m.entries[key] = entry
	return nil
}

// Delete removes an entry from the mock cache store.
//
// Takes key (string) which specifies the key to remove.
//
// Returns error when the deletion fails.
//
// Safe for concurrent use; protects access with a mutex.
func (m *MockCacheStore) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.entries, key)
	return nil
}

// Clear implements the cache store Clear method.
//
// Returns error when the cache cannot be cleared.
//
// Safe for concurrent use.
func (m *MockCacheStore) Clear(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries = make(map[string]*llm_dto.CacheEntry)
	return nil
}

// GetStats implements the cache store GetStats method.
//
// Returns *llm_dto.CacheStats which contains the current cache statistics.
// Returns error which is always nil for this mock implementation.
//
// Safe for concurrent use. Uses a mutex to protect access to internal state.
func (m *MockCacheStore) GetStats(_ context.Context) (*llm_dto.CacheStats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return &llm_dto.CacheStats{
		Hits:   m.hits,
		Misses: m.misses,
		Size:   int64(len(m.entries)),
	}, nil
}

// MockBudgetStore is a mock implementation of the budget store for testing.
type MockBudgetStore struct {
	// statuses maps scope names to their budget status.
	statuses map[string]*llm_dto.BudgetStatus

	// RecordFunc is the mock implementation for the Record method.
	RecordFunc func(ctx context.Context, scope string, cost *llm_dto.CostEstimate) error

	// GetStatusFunc is the mock implementation for the GetStatus method.
	GetStatusFunc func(ctx context.Context, scope string) (*llm_dto.BudgetStatus, error)

	// CheckAndReserveFunc is the mock implementation for the CheckAndReserve
	// method.
	CheckAndReserveFunc func(ctx context.Context, scope string, estimatedCost maths.Money, limits llm_dto.BudgetLimits) error

	// UnreserveFunc is the mock implementation for the Unreserve method.
	UnreserveFunc func(ctx context.Context, scope string, cost maths.Money) error

	// IncrementRequestsFunc is the mock implementation for IncrementRequests.
	IncrementRequestsFunc func(ctx context.Context, scope string, count int64) error

	// IncrementTokensFunc is the mock implementation for IncrementTokens.
	IncrementTokensFunc func(ctx context.Context, scope string, count int64) error

	// mu guards concurrent access to the mock store fields.
	mu sync.Mutex
}

// NewMockBudgetStore creates a new MockBudgetStore.
//
// Returns *MockBudgetStore which is ready for use in tests.
func NewMockBudgetStore() *MockBudgetStore {
	return &MockBudgetStore{
		statuses: make(map[string]*llm_dto.BudgetStatus),
	}
}

// Record implements the budget store Record method.
//
// Takes scope (string) which identifies the budget scope to record against.
// Takes cost (*llm_dto.CostEstimate) which contains the cost to add.
//
// Returns error when the custom RecordFunc returns an error.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockBudgetStore) Record(ctx context.Context, scope string, cost *llm_dto.CostEstimate) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.RecordFunc != nil {
		return m.RecordFunc(ctx, scope, cost)
	}

	status, exists := m.statuses[scope]
	if !exists {
		status = &llm_dto.BudgetStatus{
			Scope:       scope,
			TotalSpent:  maths.ZeroMoney(llm_dto.CostCurrency),
			DailySpent:  maths.ZeroMoney(llm_dto.CostCurrency),
			HourlySpent: maths.ZeroMoney(llm_dto.CostCurrency),
			LastUpdated: time.Now(),
		}
	}
	status.TotalSpent = status.TotalSpent.Add(cost.TotalCost)
	status.DailySpent = status.DailySpent.Add(cost.TotalCost)
	status.HourlySpent = status.HourlySpent.Add(cost.TotalCost)

	status.LastUpdated = time.Now()
	m.statuses[scope] = status
	return nil
}

// GetStatus implements the budget store GetStatus method.
//
// Takes scope (string) which identifies the budget scope to retrieve.
//
// Returns *llm_dto.BudgetStatus which contains the current budget status for
// the scope.
// Returns error when the status cannot be retrieved.
//
// Safe for concurrent use; protected by mutex.
func (m *MockBudgetStore) GetStatus(ctx context.Context, scope string) (*llm_dto.BudgetStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.GetStatusFunc != nil {
		return m.GetStatusFunc(ctx, scope)
	}

	status, exists := m.statuses[scope]
	if !exists {
		return &llm_dto.BudgetStatus{
			Scope:           scope,
			TotalSpent:      maths.ZeroMoney(llm_dto.CostCurrency),
			DailySpent:      maths.ZeroMoney(llm_dto.CostCurrency),
			HourlySpent:     maths.ZeroMoney(llm_dto.CostCurrency),
			RemainingBudget: maths.ZeroMoney(llm_dto.CostCurrency),
			LastUpdated:     time.Now(),
		}, nil
	}

	return new(*status), nil
}

// CheckAndReserve delegates to CheckAndReserveFunc if set.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes scope (string) which identifies the budget scope.
// Takes estimatedCost (maths.Money) which is the cost to reserve.
// Takes limits (llm_dto.BudgetLimits) which carries the spend limits.
//
// Returns nil if CheckAndReserveFunc is nil.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockBudgetStore) CheckAndReserve(ctx context.Context, scope string, estimatedCost maths.Money, limits llm_dto.BudgetLimits) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.CheckAndReserveFunc != nil {
		return m.CheckAndReserveFunc(ctx, scope, estimatedCost, limits)
	}

	status := m.statuses[scope]
	if status == nil {
		return nil
	}

	if limits.MaxTotalSpend.MustIsPositive() &&
		status.TotalSpent.Add(estimatedCost).CheckGreaterThan(limits.MaxTotalSpend) {
		return ErrBudgetExceeded
	}
	if limits.MaxDailySpend.MustIsPositive() &&
		status.DailySpent.Add(estimatedCost).CheckGreaterThan(limits.MaxDailySpend) {
		return ErrBudgetExceeded
	}
	if limits.MaxHourlySpend.MustIsPositive() &&
		status.HourlySpent.Add(estimatedCost).CheckGreaterThan(limits.MaxHourlySpend) {
		return ErrBudgetExceeded
	}
	return nil
}

// Unreserve delegates to UnreserveFunc if set.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes scope (string) which identifies the budget scope.
// Takes cost (maths.Money) which is the cost to release.
//
// Returns nil if UnreserveFunc is nil.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockBudgetStore) Unreserve(ctx context.Context, scope string, cost maths.Money) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.UnreserveFunc != nil {
		return m.UnreserveFunc(ctx, scope, cost)
	}
	return nil
}

// IncrementRequests implements the budget store IncrementRequests method.
//
// Takes scope (string) which identifies the budget scope to update.
// Takes count (int64) which specifies the number of requests to add.
//
// Returns error when the increment fails.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockBudgetStore) IncrementRequests(ctx context.Context, scope string, count int64) error {
	m.mu.Lock()
	f := m.IncrementRequestsFunc
	m.mu.Unlock()

	if f != nil {
		return f(ctx, scope, count)
	}

	m.increment(scope, count, true)
	return nil
}

// IncrementTokens implements the budget store IncrementTokens method.
//
// Takes scope (string) which identifies the budget scope to update.
// Takes count (int64) which specifies the number of tokens to add.
//
// Returns error when the operation fails.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockBudgetStore) IncrementTokens(ctx context.Context, scope string, count int64) error {
	m.mu.Lock()
	f := m.IncrementTokensFunc
	m.mu.Unlock()

	if f != nil {
		return f(ctx, scope, count)
	}

	m.increment(scope, count, false)
	return nil
}

// Reset implements the budget store Reset method.
//
// Takes scope (string) which identifies the budget scope to clear.
//
// Returns error when the reset operation fails.
//
// Safe for concurrent use; protected by mutex.
func (m *MockBudgetStore) Reset(_ context.Context, scope string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.statuses, scope)
	return nil
}

// increment updates the request or token count for a budget scope.
//
// Takes scope (string) which identifies the budget to update.
// Takes count (int64) which is the amount to add to the counter.
// Takes isRequest (bool) which selects between request and token counters.
//
// Safe for concurrent use; protected by mutex.
func (m *MockBudgetStore) increment(scope string, count int64, isRequest bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	status, exists := m.statuses[scope]
	if !exists {
		status = &llm_dto.BudgetStatus{
			Scope:       scope,
			TotalSpent:  maths.ZeroMoney(llm_dto.CostCurrency),
			DailySpent:  maths.ZeroMoney(llm_dto.CostCurrency),
			HourlySpent: maths.ZeroMoney(llm_dto.CostCurrency),
		}
	}
	if isRequest {
		status.RequestCount += count
	} else {
		status.TokenCount += count
	}
	m.statuses[scope] = status
}

// MockMemoryStore is a mock implementation of the memory store for testing.
type MockMemoryStore struct {
	// states maps conversation IDs to their state data.
	states map[string]*llm_dto.ConversationState

	// mu guards concurrent access to the states map.
	mu sync.Mutex
}

// NewMockMemoryStore creates a new MockMemoryStore.
//
// Returns *MockMemoryStore which is an empty store ready for use.
func NewMockMemoryStore() *MockMemoryStore {
	return &MockMemoryStore{
		states: make(map[string]*llm_dto.ConversationState),
	}
}

// Load implements the memory store Load method.
//
// Takes conversationID (string) which identifies the conversation to load.
//
// Returns *llm_dto.ConversationState which is the stored conversation state.
// Returns error when the conversation is not found.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockMemoryStore) Load(_ context.Context, conversationID string) (*llm_dto.ConversationState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.states[conversationID]
	if !exists {
		return nil, ErrConversationNotFound
	}
	return state, nil
}

// Save implements the memory store Save method.
//
// Takes state (*llm_dto.ConversationState) which is the conversation to store.
//
// Returns error when the save operation fails.
//
// Safe for concurrent use.
func (m *MockMemoryStore) Save(_ context.Context, state *llm_dto.ConversationState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.states[state.ID] = state
	return nil
}

// Delete removes the conversation state for the given identifier.
// Implements the memory store Delete method.
//
// Takes conversationID (string) which identifies the conversation to remove.
//
// Returns error when the deletion fails.
//
// Safe for concurrent use.
func (m *MockMemoryStore) Delete(_ context.Context, conversationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.states, conversationID)
	return nil
}

// List implements the memory store List method.
//
// Returns []string which contains all stored state identifiers.
// Returns error when the operation fails.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockMemoryStore) List(_ context.Context, _ string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ids := make([]string, 0, len(m.states))
	for id := range m.states {
		ids = append(ids, id)
	}
	return ids, nil
}

// MockSummariser is a test double for the summariser that returns zero
// values from nil function fields and tracks call counts atomically.
type MockSummariser struct {
	// CompleteFunc is called when Complete is invoked; nil uses the default stub.
	CompleteFunc func(ctx context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error)

	// CompleteCallCount tracks the number of calls to Complete.
	CompleteCallCount int64
}

// NewMockSummariser creates a new MockSummariser.
//
// Returns *MockSummariser which is ready for use in tests.
func NewMockSummariser() *MockSummariser {
	return &MockSummariser{}
}

// Complete delegates to CompleteFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes request (*llm_dto.CompletionRequest) which contains the completion request
// parameters.
//
// Returns (zero, nil) if CompleteFunc is nil.
func (m *MockSummariser) Complete(ctx context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
	atomic.AddInt64(&m.CompleteCallCount, 1)

	if m.CompleteFunc != nil {
		return m.CompleteFunc(ctx, request)
	}
	return &llm_dto.CompletionResponse{
		ID:      "mock-summary-id",
		Model:   request.Model,
		Created: time.Now().Unix(),
		Choices: []llm_dto.Choice{
			{
				Index: 0,
				Message: llm_dto.Message{
					Role:    llm_dto.RoleAssistant,
					Content: "This is a mock summary of the conversation.",
				},
				FinishReason: "stop",
			},
		},
	}, nil
}

// MockRateLimiterStore is a mock implementation of
// ratelimiter_domain.TokenBucketStorePort for testing.
type MockRateLimiterStore struct {
	// buckets maps bucket keys to their current token bucket state.
	buckets map[string]*ratelimiter_domain.TokenBucketState

	// clock returns the current time for token bucket calculations.
	clock func() time.Time

	// TryTakeFunc is called when TryTake is invoked; nil uses default behaviour.
	TryTakeFunc func(ctx context.Context, key string, n float64, config *ratelimiter_dto.TokenBucketConfig) (bool, error)

	// WaitDurationFunc is called to calculate wait time for token availability.
	WaitDurationFunc func(ctx context.Context, key string, n float64, config *ratelimiter_dto.TokenBucketConfig) (time.Duration, error)

	// mu guards concurrent access to the bucket state map.
	mu sync.Mutex
}

// NewMockRateLimiterStore creates a new MockRateLimiterStore.
//
// Returns *MockRateLimiterStore which is ready for use in tests.
func NewMockRateLimiterStore() *MockRateLimiterStore {
	return &MockRateLimiterStore{
		buckets: make(map[string]*ratelimiter_domain.TokenBucketState),
		clock:   time.Now,
	}
}

// NewMockRateLimiterStoreWithClock creates a new MockRateLimiterStore with a
// custom clock.
//
// Takes clock (func() time.Time) which provides the current time for rate
// limiting calculations.
//
// Returns *MockRateLimiterStore which is the configured mock store ready for
// use in tests.
func NewMockRateLimiterStoreWithClock(clock func() time.Time) *MockRateLimiterStore {
	return &MockRateLimiterStore{
		buckets: make(map[string]*ratelimiter_domain.TokenBucketState),
		clock:   clock,
	}
}

// TryTake implements ratelimiter_domain.TokenBucketStorePort.
//
// Takes key (string) which identifies the rate limit bucket.
// Takes n (float64) which is the number of tokens to take.
// Takes config (*ratelimiter_dto.TokenBucketConfig) which provides bucket
// settings.
//
// Returns bool which is true if the tokens were taken successfully.
// Returns error when the operation fails.
//
// Safe for concurrent use; protects bucket state with a mutex.
func (m *MockRateLimiterStore) TryTake(ctx context.Context, key string, n float64, config *ratelimiter_dto.TokenBucketConfig) (bool, error) {
	if m.TryTakeFunc != nil {
		return m.TryTakeFunc(ctx, key, n, config)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	nowNs := m.clock().UnixNano()

	state, exists := m.buckets[key]
	if !exists {
		state = ratelimiter_domain.NewBucketState(config, nowNs)
		m.buckets[key] = state
	}

	state = ratelimiter_domain.RefillBucket(state, nowNs)

	if state.Tokens >= n {
		m.buckets[key] = &ratelimiter_domain.TokenBucketState{
			Tokens:         state.Tokens - n,
			MaxTokens:      state.MaxTokens,
			RefillRate:     state.RefillRate,
			LastRefillNano: state.LastRefillNano,
		}
		return true, nil
	}

	m.buckets[key] = state
	return false, nil
}

// WaitDuration implements ratelimiter_domain.TokenBucketStorePort.
//
// Takes key (string) which identifies the rate limit bucket.
// Takes n (float64) which is the number of tokens to check availability for.
// Takes config (*ratelimiter_dto.TokenBucketConfig) which provides the bucket
// configuration.
//
// Returns time.Duration which is the time to wait before n tokens are
// available.
// Returns error when the wait duration cannot be calculated.
//
// Safe for concurrent use. Uses a mutex to protect bucket state access.
func (m *MockRateLimiterStore) WaitDuration(ctx context.Context, key string, n float64, config *ratelimiter_dto.TokenBucketConfig) (time.Duration, error) {
	if m.WaitDurationFunc != nil {
		return m.WaitDurationFunc(ctx, key, n, config)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	nowNs := m.clock().UnixNano()

	state, exists := m.buckets[key]
	if !exists {
		return 0, nil
	}

	state = ratelimiter_domain.RefillBucket(state, nowNs)

	if state.Tokens >= n {
		return 0, nil
	}

	tokensNeeded := n - state.Tokens
	if state.RefillRate <= 0 {
		return time.Hour, nil
	}
	waitNanos := tokensNeeded / state.RefillRate
	return time.Duration(waitNanos), nil
}

// DeleteBucket implements ratelimiter_domain.TokenBucketStorePort.
//
// Takes key (string) which identifies the rate limit bucket to delete.
//
// Returns error when the deletion fails.
//
// Safe for concurrent use.
func (m *MockRateLimiterStore) DeleteBucket(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.buckets, key)
	return nil
}
