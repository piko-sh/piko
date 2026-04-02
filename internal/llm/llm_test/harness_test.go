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

package llm_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"piko.sh/piko/internal/llm/llm_adapters/budget_store/memory"
	"piko.sh/piko/internal/llm/llm_adapters/memory_memory"
	"piko.sh/piko/internal/llm/llm_adapters/provider_mock"
	"piko.sh/piko/internal/llm/llm_adapters/vector_cache"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/maths"
)

type testHarness struct {
	service        llm_domain.Service
	mockProvider   *provider_mock.MockProvider
	mockProvider2  *provider_mock.MockProvider
	memoryStore    *memory_memory.Store
	vectorStore    *vector_cache.Store
	budgetStore    *memory.Store
	cacheStore     *testCacheStore
	rateLimitStore *testRateLimiterStore
	costCalc       *llm_domain.CostCalculator
	budgetMgr      *llm_domain.BudgetManager
	rateLimiter    *llm_domain.RateLimiter
	cacheMgr       *llm_domain.CacheManager
	clock          *clock.MockClock
}

func testPricingTable() *llm_dto.PricingTable {
	return &llm_dto.PricingTable{
		Models: []llm_dto.ModelPricing{
			{
				ModelID:         "test-model",
				Provider:        "mock",
				InputCostPer1M:  maths.NewDecimalFromString("1.00"),
				OutputCostPer1M: maths.NewDecimalFromString("2.00"),
			},
		},
	}
}

func newTestHarness(t *testing.T) *testHarness {
	t.Helper()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))

	mp := provider_mock.New(provider_mock.WithMockClock(mockClock))
	mp2 := provider_mock.New(provider_mock.WithMockClock(mockClock))

	memStore := memory_memory.New()
	vecStore := createOtterVectorStore(t, 3, "cosine")
	budStore := memory.New(memory.WithClock(mockClock))
	cacheString := newTestCacheStore()
	rlStore := newTestRateLimiterStore(mockClock)

	costCalc := llm_domain.NewCostCalculatorWithPricing(
		testPricingTable(),
		llm_domain.WithCostCalculatorClock(mockClock),
	)
	budgetMgr := llm_domain.NewBudgetManager(budStore, costCalc)
	rateLimiter := llm_domain.NewRateLimiter(rlStore, llm_domain.WithRateLimiterClock(mockClock))
	cacheMgr := llm_domain.NewCacheManager(cacheString, time.Hour, llm_domain.WithCacheManagerClock(mockClock))

	service := llm_domain.NewService(
		"mock",
		llm_domain.WithCostCalculator(costCalc),
		llm_domain.WithBudgetManager(budgetMgr),
		llm_domain.WithRateLimiter(rateLimiter),
		llm_domain.WithClock(mockClock),
	)
	if err := service.RegisterProvider(context.Background(), "mock", mp); err != nil {
		t.Fatalf("failed to register mock provider: %v", err)
	}
	if err := service.RegisterProvider(context.Background(), "fallback", mp2); err != nil {
		t.Fatalf("failed to register fallback provider: %v", err)
	}
	service.SetCacheManager(cacheMgr)
	service.SetVectorStore(vecStore)

	return &testHarness{
		service:        service,
		mockProvider:   mp,
		mockProvider2:  mp2,
		memoryStore:    memStore,
		vectorStore:    vecStore,
		budgetStore:    budStore,
		cacheStore:     cacheString,
		rateLimitStore: rlStore,
		costCalc:       costCalc,
		budgetMgr:      budgetMgr,
		rateLimiter:    rateLimiter,
		cacheMgr:       cacheMgr,
		clock:          mockClock,
	}
}

func (h *testHarness) RegisterEmbeddingProvider(name string, p llm_domain.EmbeddingProviderPort) {
	_ = h.service.RegisterEmbeddingProvider(context.Background(), name, p)
}

type testCacheStore struct {
	entries map[string]*llm_dto.CacheEntry
	hits    int64
	misses  int64
	mu      sync.RWMutex
}

func newTestCacheStore() *testCacheStore {
	return &testCacheStore{
		entries: make(map[string]*llm_dto.CacheEntry),
	}
}

func (s *testCacheStore) Get(_ context.Context, key string) (*llm_dto.CacheEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.entries[key]
	if !ok {
		s.mu.RUnlock()
		s.mu.Lock()
		s.misses++
		s.mu.Unlock()
		s.mu.RLock()
		return nil, nil
	}
	s.mu.RUnlock()
	s.mu.Lock()
	s.hits++
	s.mu.Unlock()
	s.mu.RLock()
	return entry, nil
}

func (s *testCacheStore) Set(_ context.Context, key string, entry *llm_dto.CacheEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[key] = entry
	return nil
}

func (s *testCacheStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, key)
	return nil
}

func (s *testCacheStore) Clear(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = make(map[string]*llm_dto.CacheEntry)
	return nil
}

func (s *testCacheStore) GetStats(_ context.Context) (*llm_dto.CacheStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return &llm_dto.CacheStats{
		Hits:   s.hits,
		Misses: s.misses,
		Size:   int64(len(s.entries)),
	}, nil
}

func (s *testCacheStore) entryCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

type testRateLimiterStore struct {
	clk     clock.Clock
	buckets map[string]*ratelimiter_domain.TokenBucketState
	mu      sync.Mutex
}

func newTestRateLimiterStore(clk clock.Clock) *testRateLimiterStore {
	return &testRateLimiterStore{
		buckets: make(map[string]*ratelimiter_domain.TokenBucketState),
		clk:     clk,
	}
}

func (s *testRateLimiterStore) TryTake(_ context.Context, key string, n float64, config *ratelimiter_dto.TokenBucketConfig) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.clk.Now().UnixNano()
	state := s.getOrCreate(key, config, now)
	state = ratelimiter_domain.RefillBucket(state, now)

	if state.Tokens >= n {
		s.buckets[key] = &ratelimiter_domain.TokenBucketState{
			Tokens:         state.Tokens - n,
			MaxTokens:      state.MaxTokens,
			RefillRate:     state.RefillRate,
			LastRefillNano: state.LastRefillNano,
		}
		return true, nil
	}

	s.buckets[key] = state
	return false, nil
}

func (s *testRateLimiterStore) WaitDuration(_ context.Context, key string, n float64, config *ratelimiter_dto.TokenBucketConfig) (time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.clk.Now().UnixNano()
	state := s.getOrCreate(key, config, now)
	state = ratelimiter_domain.RefillBucket(state, now)

	deficit := n - state.Tokens
	if deficit <= 0 {
		return 0, nil
	}
	if state.RefillRate <= 0 {
		return time.Hour, nil
	}
	waitNanos := deficit / state.RefillRate
	return time.Duration(waitNanos), nil
}

func (s *testRateLimiterStore) DeleteBucket(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.buckets, key)
	return nil
}

func (s *testRateLimiterStore) getOrCreate(key string, config *ratelimiter_dto.TokenBucketConfig, nowNano int64) *ratelimiter_domain.TokenBucketState {
	state, ok := s.buckets[key]
	if !ok {
		state = ratelimiter_domain.NewBucketState(config, nowNano)
		s.buckets[key] = state
	}
	return state
}

type testSummariser struct {
	err      error
	response string
	calls    int
	mu       sync.Mutex
}

func newTestSummariser(response string) *testSummariser {
	return &testSummariser{response: response}
}

func (s *testSummariser) Complete(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return makeResponse(s.response, 10, 10), nil
}

func (s *testSummariser) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func makeResponse(content string, promptTokens, completionTokens int) *llm_dto.CompletionResponse {
	return &llm_dto.CompletionResponse{
		ID:    "response-" + content[:min(8, len(content))],
		Model: "test-model",
		Choices: []llm_dto.Choice{
			{
				Index: 0,
				Message: llm_dto.Message{
					Role:    llm_dto.RoleAssistant,
					Content: content,
				},
				FinishReason: llm_dto.FinishReasonStop,
			},
		},
		Usage: &llm_dto.Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}
}
