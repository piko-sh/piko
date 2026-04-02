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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/maths"
)

func TestService_AddDocuments_NoVectorStore(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	err := service.AddDocuments(context.Background(), "ns", []Document{
		{ID: "doc1", Content: "Hello"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "vector store is not configured")
}

func TestService_AddDocuments_EmptyDocuments(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	mockVS := &mockVectorStore{}
	service.SetVectorStore(mockVS)

	err := service.AddDocuments(context.Background(), "ns", []Document{})

	assert.NoError(t, err)
	assert.Empty(t, mockVS.bulkStoreCalls)
}

func TestService_AddDocuments_Success(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	mockEmbedding := NewMockEmbeddingProvider()
	require.NoError(t, service.RegisterEmbeddingProvider(context.Background(), "default", mockEmbedding))
	require.NoError(t, service.SetDefaultEmbeddingProvider("default"))

	mockVS := &mockVectorStore{}
	service.SetVectorStore(mockVS)

	docs := []Document{
		{ID: "doc1", Content: "Hello"},
		{ID: "doc2", Content: "World"},
	}

	err := service.AddDocuments(context.Background(), "ns", docs)

	require.NoError(t, err)
	require.Len(t, mockVS.bulkStoreCalls, 1)
	assert.Len(t, mockVS.bulkStoreCalls[0], 2)
	assert.Equal(t, "doc1", mockVS.bulkStoreCalls[0][0].ID)
	assert.Equal(t, "doc2", mockVS.bulkStoreCalls[0][1].ID)
}

func TestService_AddDocuments_EmbeddingError(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	mockEmbedding := NewMockEmbeddingProvider()
	mockEmbedding.SetEmbedFunc(func(_ context.Context, _ *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
		return nil, errors.New("embedding failed")
	})
	require.NoError(t, service.RegisterEmbeddingProvider(context.Background(), "default", mockEmbedding))
	require.NoError(t, service.SetDefaultEmbeddingProvider("default"))

	mockVS := &mockVectorStore{}
	service.SetVectorStore(mockVS)

	err := service.AddDocuments(context.Background(), "ns", []Document{
		{ID: "doc1", Content: "Hello"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "batch embedding failed")
}

func TestService_AddDocuments_BulkStoreError(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	mockEmbedding := NewMockEmbeddingProvider()
	require.NoError(t, service.RegisterEmbeddingProvider(context.Background(), "default", mockEmbedding))
	require.NoError(t, service.SetDefaultEmbeddingProvider("default"))

	mockVS := &mockVectorStore{bulkStoreErr: errors.New("storage failed")}
	service.SetVectorStore(mockVS)

	err := service.AddDocuments(context.Background(), "ns", []Document{
		{ID: "doc1", Content: "Hello"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "bulk storage failed")
}

func TestService_AddDocuments_CancelledContext(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	mockEmbedding := NewMockEmbeddingProvider()
	require.NoError(t, service.RegisterEmbeddingProvider(context.Background(), "default", mockEmbedding))
	require.NoError(t, service.SetDefaultEmbeddingProvider("default"))

	mockVS := &mockVectorStore{}
	service.SetVectorStore(mockVS)

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	docs := make([]Document, 25)
	for i := range docs {
		docs[i] = Document{ID: "doc", Content: "c"}
	}

	err := service.AddDocuments(ctx, "ns", docs)

	assert.Error(t, err)
}

func TestService_AddText_Success(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	mockEmbedding := NewMockEmbeddingProvider()
	require.NoError(t, service.RegisterEmbeddingProvider(context.Background(), "default", mockEmbedding))
	require.NoError(t, service.SetDefaultEmbeddingProvider("default"))

	mockVS := &mockVectorStore{}
	service.SetVectorStore(mockVS)

	err := service.AddText(context.Background(), "ns", "my-id", "Hello World")

	require.NoError(t, err)
	require.Len(t, mockVS.bulkStoreCalls, 1)
	assert.Equal(t, "my-id", mockVS.bulkStoreCalls[0][0].ID)
	assert.Equal(t, "Hello World", mockVS.bulkStoreCalls[0][0].Content)
}

func TestService_AddText_NoVectorStore(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	err := service.AddText(context.Background(), "ns", "id", "content")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "vector store is not configured")
}

func TestService_NewIngest(t *testing.T) {
	service := NewService("")

	builder := service.NewIngest("test-ns")

	require.NotNil(t, builder)
	assert.Equal(t, "test-ns", builder.namespace)
}

func TestService_Close_WithVectorStore(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	mockVS := &mockVectorStore{}
	service.SetVectorStore(mockVS)

	err := service.Close(context.Background())

	assert.NoError(t, err)
	assert.True(t, mockVS.closeCalled)
	assert.Nil(t, service.vectorStore)
}

func TestService_Close_VectorStoreError(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	mockVS := &mockVectorStore{closeErr: errors.New("close failed")}
	service.SetVectorStore(mockVS)

	err := service.Close(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "close failed")
}

func TestService_Close_ProviderAndVectorStoreErrors(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	failingProvider := NewMockLLMProvider()
	failingProvider.CloseFunc = func(_ context.Context) error {
		return errors.New("provider close failed")
	}
	require.NoError(t, service.RegisterProvider(context.Background(), "p1", failingProvider))

	mockVS := &mockVectorStore{closeErr: errors.New("vs close failed")}
	service.SetVectorStore(mockVS)

	err := service.Close(context.Background())

	assert.Error(t, err)
}

func TestServiceOption_WithVectorStore(t *testing.T) {
	mockVS := &mockVectorStore{}
	service := NewService("", WithVectorStore(mockVS))

	assert.Equal(t, mockVS, service.GetVectorStore())
}

func TestService_GetSetVectorStore(t *testing.T) {
	service := NewService("")

	assert.Nil(t, service.GetVectorStore())

	mockVS := &mockVectorStore{}
	service.SetVectorStore(mockVS)
	assert.Equal(t, mockVS, service.GetVectorStore())
}

func TestCompletionBuilder_WithVectorContext_Empty(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	result := builder.WithVectorContext(nil)

	assert.Equal(t, builder, result)
	assert.Nil(t, builder.vectorContext)
}

func TestCompletionBuilder_WithVectorContext_EmptySlice(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	result := builder.WithVectorContext([]llm_dto.VectorSearchResult{})

	assert.Equal(t, builder, result)
	assert.Nil(t, builder.vectorContext)
}

func TestCompletionBuilder_WithVectorContext_WithResults(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	results := []llm_dto.VectorSearchResult{
		{ID: "doc1", Content: "Hello", Score: 0.95},
		{ID: "doc2", Content: "World", Score: 0.85},
	}

	result := builder.WithVectorContext(results)

	assert.Equal(t, builder, result)
	require.Len(t, builder.vectorContext, 2)
	assert.Equal(t, "doc1", builder.vectorContext[0].ID)
}

func TestCompletionBuilder_InjectVectorContext_NoContext(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion().
		User("Hello")

	originalLen := len(builder.request.Messages)
	builder.injectVectorContext()

	assert.Equal(t, originalLen, len(builder.request.Messages))
}

func TestCompletionBuilder_InjectVectorContext_WithContext(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion().
		User("Hello")

	builder.vectorContext = []llm_dto.VectorSearchResult{
		{ID: "doc1", Content: "Relevant document", Score: 0.95},
	}

	builder.injectVectorContext()

	require.Len(t, builder.request.Messages, 2)
	assert.Equal(t, llm_dto.RoleSystem, builder.request.Messages[0].Role)
	assert.Contains(t, builder.request.Messages[0].Content, "Relevant document")
	assert.Contains(t, builder.request.Messages[0].Content, "knowledge base")
	assert.Contains(t, builder.request.Messages[0].Content, "0.95")
	assert.Equal(t, llm_dto.RoleUser, builder.request.Messages[1].Role)
}

func TestCompletionBuilder_InjectVectorContext_MultipleDocuments(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion().
		User("Hello")

	builder.vectorContext = []llm_dto.VectorSearchResult{
		{ID: "doc1", Content: "First doc", Score: 0.95},
		{ID: "doc2", Content: "Second doc", Score: 0.80},
	}

	builder.injectVectorContext()

	require.Len(t, builder.request.Messages, 2)
	contextMessage := builder.request.Messages[0].Content
	assert.Contains(t, contextMessage, "First doc")
	assert.Contains(t, contextMessage, "Second doc")
	assert.Contains(t, contextMessage, "---")
}

func TestCompletionBuilder_ResolveProviderName_ExplicitProvider(t *testing.T) {
	service := NewService("default-provider")
	builder := service.NewCompletion().Provider("explicit-provider")

	name := builder.resolveProviderName()

	assert.Equal(t, "explicit-provider", name)
}

func TestCompletionBuilder_ResolveProviderName_DefaultProvider(t *testing.T) {
	service := NewService("default-provider")
	builder := service.NewCompletion()

	name := builder.resolveProviderName()

	assert.Equal(t, "default-provider", name)
}

func TestBuildBucketKey(t *testing.T) {
	key := buildBucketKey("user:123", BucketTypeRequest)
	assert.Equal(t, "user:123:request", key)

	key = buildBucketKey("global", BucketTypeToken)
	assert.Equal(t, "global:token", key)
}

func TestRequestBucketConfig(t *testing.T) {
	config := &rateLimitConfig{
		requestsPerMinute: 120,
		tokensPerMinute:   10000,
	}

	bucketConfig := requestBucketConfig(config)

	assert.Equal(t, 120.0/SecondsPerMinute, bucketConfig.Rate)
	assert.Equal(t, 120, bucketConfig.Burst)
}

func TestTokenBucketConfig(t *testing.T) {
	config := &rateLimitConfig{
		requestsPerMinute: 120,
		tokensPerMinute:   60000,
	}

	bucketConfig := tokenBucketConfig(config)

	assert.Equal(t, 60000.0/SecondsPerMinute, bucketConfig.Rate)
	assert.Equal(t, 60000, bucketConfig.Burst)
}

func TestRequestBucketConfig_ZeroValues(t *testing.T) {
	config := &rateLimitConfig{
		requestsPerMinute: 0,
		tokensPerMinute:   0,
	}

	bucketConfig := requestBucketConfig(config)

	assert.Equal(t, 0.0, bucketConfig.Rate)
	assert.Equal(t, 0, bucketConfig.Burst)
}

func TestTokenBucketConfig_ZeroValues(t *testing.T) {
	config := &rateLimitConfig{
		requestsPerMinute: 0,
		tokensPerMinute:   0,
	}

	bucketConfig := tokenBucketConfig(config)

	assert.Equal(t, 0.0, bucketConfig.Rate)
	assert.Equal(t, 0, bucketConfig.Burst)
}

func TestErrorSentinels(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		message string
	}{
		{name: "ErrProviderNotFound", err: ErrProviderNotFound, message: "llm provider not found"},
		{name: "ErrNoDefaultProvider", err: ErrNoDefaultProvider, message: "no default llm provider configured"},
		{name: "ErrProviderAlreadyExists", err: ErrProviderAlreadyExists, message: "llm provider already exists"},
		{name: "ErrStreamingNotSupported", err: ErrStreamingNotSupported, message: "llm provider does not support streaming"},
		{name: "ErrToolsNotSupported", err: ErrToolsNotSupported, message: "llm provider does not support tools"},
		{name: "ErrStructuredOutputNotSupported", err: ErrStructuredOutputNotSupported, message: "llm provider does not support structured output"},
		{name: "ErrEmptyMessages", err: ErrEmptyMessages, message: "completion request must contain at least one message"},
		{name: "ErrEmptyModel", err: ErrEmptyModel, message: "completion request must specify a model"},
		{name: "ErrInvalidTemperature", err: ErrInvalidTemperature, message: "temperature must be between 0 and 2"},
		{name: "ErrInvalidTopP", err: ErrInvalidTopP, message: "top_p must be between 0 and 1"},
		{name: "ErrInvalidMaxTokens", err: ErrInvalidMaxTokens, message: "max_tokens must be positive"},
		{name: "ErrBudgetExceeded", err: ErrBudgetExceeded, message: "budget limit exceeded"},
		{name: "ErrRateLimited", err: ErrRateLimited, message: "rate limit exceeded"},
		{name: "ErrMaxCostExceeded", err: ErrMaxCostExceeded, message: "estimated cost exceeds per-request limit"},
		{name: "ErrUnknownModelPrice", err: ErrUnknownModelPrice, message: "no pricing information for model"},
		{name: "ErrProviderOverloaded", err: ErrProviderOverloaded, message: "provider overloaded"},
		{name: "ErrProviderTimeout", err: ErrProviderTimeout, message: "provider timeout"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
			assert.Equal(t, tt.message, tt.err.Error())
		})
	}
}

func TestErrorSentinels_AreDistinct(t *testing.T) {
	errs := []error{
		ErrProviderNotFound,
		ErrNoDefaultProvider,
		ErrProviderAlreadyExists,
		ErrStreamingNotSupported,
		ErrToolsNotSupported,
		ErrStructuredOutputNotSupported,
		ErrEmptyMessages,
		ErrEmptyModel,
		ErrInvalidTemperature,
		ErrInvalidTopP,
		ErrInvalidMaxTokens,
		ErrBudgetExceeded,
		ErrRateLimited,
		ErrMaxCostExceeded,
		ErrUnknownModelPrice,
		ErrProviderOverloaded,
		ErrProviderTimeout,
	}

	for i := range errs {
		for j := range errs {
			if i != j {
				assert.NotEqual(t, errs[i], errs[j],
					"errors at index %d and %d should be distinct", i, j)
			}
		}
	}
}

func TestErrorSentinels_ErrorsIs(t *testing.T) {
	wrapped := errors.Join(errors.New("context"), ErrProviderNotFound)
	assert.True(t, errors.Is(wrapped, ErrProviderNotFound))
	assert.False(t, errors.Is(wrapped, ErrNoDefaultProvider))
}

func TestService_RecordTokenUsage_NilResponse(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	service.recordTokenUsage(context.Background(), nil)
}

func TestService_RecordTokenUsage_NilUsage(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	response := &llm_dto.CompletionResponse{}

	service.recordTokenUsage(context.Background(), response)
}

func TestService_RecordTokenUsage_WithUsage(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	response := &llm_dto.CompletionResponse{
		Usage: &llm_dto.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	service.recordTokenUsage(context.Background(), response)
}

func TestService_RecordCostAndBudget_NilResponse(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	service.recordCostAndBudget(context.Background(), "openai", "gpt-4o", nil, "scope")
}

func TestService_RecordCostAndBudget_NilUsage(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	response := &llm_dto.CompletionResponse{}

	service.recordCostAndBudget(context.Background(), "openai", "gpt-4o", response, "scope")
}

func TestRateLimiter_AllowN_StoreError(t *testing.T) {
	store := NewMockRateLimiterStore()
	store.TryTakeFunc = func(_ context.Context, key string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (bool, error) {
		return false, errors.New("store error")
	}

	limiter := NewRateLimiter(store)
	limiter.SetLimits("scope", 60, 0)

	err := limiter.AllowN(context.Background(), "scope", 1, 0)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "checking request rate limit")
}

func TestRateLimiter_AllowN_TokenStoreError(t *testing.T) {
	store := NewMockRateLimiterStore()
	callCount := 0
	store.TryTakeFunc = func(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (bool, error) {
		callCount++
		if callCount == 1 {
			return true, nil
		}
		return false, errors.New("token store error")
	}

	limiter := NewRateLimiter(store)
	limiter.SetLimits("scope", 60, 1000)

	err := limiter.AllowN(context.Background(), "scope", 1, 100)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "checking token rate limit")
}

func TestRateLimiter_AllowN_CancelledContext(t *testing.T) {
	store := NewMockRateLimiterStore()
	limiter := NewRateLimiter(store)
	limiter.SetLimits("scope", 60, 0)

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	err := limiter.AllowN(ctx, "scope", 1, 0)

	assert.Error(t, err)
}

func TestRateLimiter_HasLimits(t *testing.T) {
	store := NewMockRateLimiterStore()
	limiter := NewRateLimiter(store)

	assert.False(t, limiter.HasLimits("scope"))

	limiter.SetLimits("scope", 60, 0)
	assert.True(t, limiter.HasLimits("scope"))

	limiter.RemoveLimits("scope")
	assert.False(t, limiter.HasLimits("scope"))
}

func TestService_GetProvider_EmptyName_NoDefault(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	_, err := service.getProvider("")

	assert.ErrorIs(t, err, ErrNoDefaultProvider)
}

func TestService_GetProvider_NotFound(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	_, err := service.getProvider("nonexistent")

	assert.ErrorIs(t, err, ErrProviderNotFound)
}

func TestService_GetProvider_Found(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	mock := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "openai", mock))

	provider, err := service.getProvider("openai")

	require.NoError(t, err)
	assert.Equal(t, mock, provider)
}

func TestService_GetProvider_UsesDefault(t *testing.T) {
	service, ok := NewService("openai").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	mock := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "openai", mock))

	provider, err := service.getProvider("")

	require.NoError(t, err)
	assert.Equal(t, mock, provider)
}

func TestService_CheckRateLimit_NoLimiter(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	err := service.checkRateLimit(context.Background(), "scope")

	assert.NoError(t, err)
}

func TestService_CheckRateLimit_EmptyScope(t *testing.T) {
	store := NewMockRateLimiterStore()
	limiter := NewRateLimiter(store)
	service, ok := NewService("", WithRateLimiter(limiter)).(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	err := service.checkRateLimit(context.Background(), "")

	assert.NoError(t, err)
}

func TestService_CheckBudget_NoBudgetManager(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	request := &llm_dto.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
	}

	err := service.checkBudget(context.Background(), request, "scope", service.costCalculator.EstimateRequestCost("gpt-4o", 10))

	assert.NoError(t, err)
}

func TestService_CheckBudget_EmptyScope(t *testing.T) {
	budgetStore := NewMockBudgetStore()
	budgetMgr := NewBudgetManager(budgetStore, NewCostCalculator())
	service, ok := NewService("", WithBudgetManager(budgetMgr)).(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	request := &llm_dto.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
	}

	err := service.checkBudget(context.Background(), request, "", service.costCalculator.EstimateRequestCost("gpt-4o", 10))

	assert.NoError(t, err)
}

func TestService_InitDefaults(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	assert.NotNil(t, service.costCalculator)
	assert.NotNil(t, service.embeddingService)
}

func TestDefaultJitterFallback(t *testing.T) {
	assert.Equal(t, 0.5, DefaultJitterFallback)
}

func TestSecondsPerMinute(t *testing.T) {
	assert.Equal(t, 60.0, SecondsPerMinute)
}

func TestService_GetProviders_Empty(t *testing.T) {
	service := NewService("")
	providers := service.GetProviders()
	assert.Empty(t, providers)
}

func TestService_GetProviders_WithProviders(t *testing.T) {
	service := NewService("")
	require.NoError(t, service.RegisterProvider(context.Background(), "p1", NewMockLLMProvider()))
	require.NoError(t, service.RegisterProvider(context.Background(), "p2", NewMockLLMProvider()))

	providers := service.GetProviders()
	assert.Len(t, providers, 2)
	assert.Contains(t, providers, "p1")
	assert.Contains(t, providers, "p2")
}

func TestService_GetDefaultProvider_Empty(t *testing.T) {
	service := NewService("")
	assert.Equal(t, "", service.GetDefaultProvider())
}

func TestService_GetDefaultProvider_Set(t *testing.T) {
	service := NewService("openai")
	assert.Equal(t, "openai", service.GetDefaultProvider())
}

func TestCompletionBuilder_ExecuteWithFallback(t *testing.T) {
	service, ok := NewService("primary").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	mock1 := NewMockLLMProvider()
	mock2 := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "primary", mock1))
	require.NoError(t, service.RegisterProvider(context.Background(), "secondary", mock2))

	builder := service.NewCompletion().
		Model("gpt-4o").
		User("Hello").
		FallbackProviders("primary", "secondary")

	response, err := builder.Do(context.Background())

	require.NoError(t, err)
	require.NotNil(t, response)
}

func TestCompletionBuilder_ExecuteWithFallback_FirstFails(t *testing.T) {
	service, ok := NewService("primary").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	failingProvider := NewMockLLMProvider()
	failingProvider.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		return nil, ErrProviderOverloaded
	}

	successProvider := NewMockLLMProvider()

	require.NoError(t, service.RegisterProvider(context.Background(), "primary", failingProvider))
	require.NoError(t, service.RegisterProvider(context.Background(), "secondary", successProvider))

	builder := service.NewCompletion().
		Model("gpt-4o").
		User("Hello").
		FallbackProviders("primary", "secondary")

	response, err := builder.Do(context.Background())

	require.NoError(t, err)
	require.NotNil(t, response)

	require.NotNil(t, response.FallbackInfo)
	assert.True(t, response.FallbackInfo.WasFallbackUsed())
}

func TestCompletionBuilder_ExecuteWithRetry(t *testing.T) {
	service, ok := NewService("openai").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	mock := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "openai", mock))

	builder := service.NewCompletion().
		Model("gpt-4o").
		User("Hello").
		Retry(&llm_dto.RetryPolicy{
			MaxRetries:        3,
			InitialBackoff:    1,
			MaxBackoff:        10,
			BackoffMultiplier: 1.0,
			JitterFraction:    0,
		})

	response, err := builder.Do(context.Background())

	require.NoError(t, err)
	require.NotNil(t, response)
}

func TestCompletionBuilder_ExecuteWithRetry_SucceedsAfterRetries(t *testing.T) {
	service, ok := NewService("openai").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	callCount := 0
	mock := NewMockLLMProvider()
	mock.CompleteFunc = func(_ context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		callCount++
		if callCount < 3 {
			return nil, ErrProviderOverloaded
		}
		return &llm_dto.CompletionResponse{
			ID:    "success",
			Model: request.Model,
			Choices: []llm_dto.Choice{
				{Message: llm_dto.Message{Role: llm_dto.RoleAssistant, Content: "OK"}},
			},
			Usage: &llm_dto.Usage{PromptTokens: 5, CompletionTokens: 2, TotalTokens: 7},
		}, nil
	}
	require.NoError(t, service.RegisterProvider(context.Background(), "openai", mock))

	builder := service.NewCompletion().
		Model("gpt-4o").
		User("Hello").
		Retry(&llm_dto.RetryPolicy{
			MaxRetries:        5,
			InitialBackoff:    1,
			MaxBackoff:        10,
			BackoffMultiplier: 1.0,
			JitterFraction:    0,
		})

	response, err := builder.Do(context.Background())

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, "success", response.ID)
	assert.Equal(t, 3, callCount)
}

func TestService_CheckRateLimit_WithLimiter_Allowed(t *testing.T) {
	store := NewMockRateLimiterStore()
	limiter := NewRateLimiter(store)
	limiter.SetLimits("scope", 60, 0)
	service, ok := NewService("", WithRateLimiter(limiter)).(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	err := service.checkRateLimit(context.Background(), "scope")

	assert.NoError(t, err)
}

func TestService_CheckRateLimit_WithLimiter_Denied(t *testing.T) {
	store := NewMockRateLimiterStore()
	store.TryTakeFunc = func(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (bool, error) {
		return false, nil
	}

	limiter := NewRateLimiter(store)
	limiter.SetLimits("scope", 60, 0)
	service, ok := NewService("", WithRateLimiter(limiter)).(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	err := service.checkRateLimit(context.Background(), "scope")

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrRateLimited)
}

func TestService_CheckBudget_WithManager_Allowed(t *testing.T) {
	budgetStore := NewMockBudgetStore()
	calculator := NewCostCalculator()
	budgetMgr := NewBudgetManager(budgetStore, calculator)
	service, ok := NewService("", WithBudgetManager(budgetMgr), WithCostCalculator(calculator)).(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	request := &llm_dto.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
	}

	err := service.checkBudget(context.Background(), request, "scope", maths.ZeroMoney(llm_dto.CostCurrency))

	assert.NoError(t, err)
}

func TestService_CheckBudget_MaxCostExceeded(t *testing.T) {
	budgetStore := NewMockBudgetStore()
	calculator := NewCostCalculator()
	budgetMgr := NewBudgetManager(budgetStore, calculator)
	service, ok := NewService("", WithBudgetManager(budgetMgr), WithCostCalculator(calculator)).(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	request := &llm_dto.CompletionRequest{
		Model:    "gpt-5",
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello world this is a test message with enough tokens"}},
	}

	maxCost := maths.NewMoneyFromFloat(0.0000000001, llm_dto.CostCurrency)

	err := service.checkBudget(context.Background(), request, "scope", maxCost)

	assert.ErrorIs(t, err, ErrMaxCostExceeded)
}

func TestWithRateLimiterClock(t *testing.T) {
	store := NewMockRateLimiterStore()
	mc := clock.RealClock()
	limiter := NewRateLimiter(store, WithRateLimiterClock(mc))

	assert.Equal(t, mc, limiter.clock)
}

func TestService_CheckPreRequestLimits_NoLimiterNoBudget(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	request := &llm_dto.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
	}

	err := service.checkPreRequestLimits(context.Background(), request, "scope", maths.ZeroMoney(llm_dto.CostCurrency))

	assert.NoError(t, err)
}

func TestService_CheckPreRequestLimits_RateLimitFails(t *testing.T) {
	store := NewMockRateLimiterStore()
	store.TryTakeFunc = func(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (bool, error) {
		return false, nil
	}

	limiter := NewRateLimiter(store)
	limiter.SetLimits("scope", 60, 0)
	service, ok := NewService("", WithRateLimiter(limiter)).(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	request := &llm_dto.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
	}

	err := service.checkPreRequestLimits(context.Background(), request, "scope", maths.ZeroMoney(llm_dto.CostCurrency))

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrRateLimited)
}

func TestCompletionBuilder_CreateExecuteFunc_DirectPath(t *testing.T) {
	service, ok := NewService("openai").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	mock := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "openai", mock))

	builder := service.NewCompletion().
		Model("gpt-4o").
		User("Hello")

	executeFunction := builder.createExecuteFunc(context.Background(), "openai")
	response, err := executeFunction()

	require.NoError(t, err)
	require.NotNil(t, response)
}

func TestCompletionBuilder_CreateExecuteFunc_FallbackPath(t *testing.T) {
	service, ok := NewService("openai").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	mock := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "openai", mock))

	builder := service.NewCompletion().
		Model("gpt-4o").
		User("Hello").
		FallbackProviders("openai")

	executeFunction := builder.createExecuteFunc(context.Background(), "openai")
	response, err := executeFunction()

	require.NoError(t, err)
	require.NotNil(t, response)
}

func TestCompletionBuilder_CreateExecuteFunc_RetryPath(t *testing.T) {
	service, ok := NewService("openai").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	mock := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "openai", mock))

	builder := service.NewCompletion().
		Model("gpt-4o").
		User("Hello").
		Retry(&llm_dto.RetryPolicy{
			MaxRetries:        3,
			InitialBackoff:    1,
			MaxBackoff:        10,
			BackoffMultiplier: 1.0,
		})

	executeFunction := builder.createExecuteFunc(context.Background(), "openai")
	response, err := executeFunction()

	require.NoError(t, err)
	require.NotNil(t, response)
}

func TestCompletionBuilder_LoadConversationHistory_NoMemory(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion().User("Hello")

	originalLen := len(builder.request.Messages)
	builder.loadConversationHistory(context.Background())

	assert.Equal(t, originalLen, len(builder.request.Messages))
}

func TestCompletionBuilder_LoadConversationHistory_EmptyConversationID(t *testing.T) {
	service := NewService("")
	store := NewMockMemoryStore()
	mem := NewBufferMemory(store, WithBufferSize(100))
	builder := service.NewCompletion().User("Hello")
	builder.memory = mem
	builder.conversationID = ""

	originalLen := len(builder.request.Messages)
	builder.loadConversationHistory(context.Background())

	assert.Equal(t, originalLen, len(builder.request.Messages))
}

func TestCompletionBuilder_LoadConversationHistory_WithHistory(t *testing.T) {
	service := NewService("")
	store := NewMockMemoryStore()
	mem := NewBufferMemory(store, WithBufferSize(100))

	ctx := context.Background()
	require.NoError(t, mem.AddMessage(ctx, "conv-1", llm_dto.NewUserMessage("Previous question")))
	require.NoError(t, mem.AddMessage(ctx, "conv-1", llm_dto.NewAssistantMessage("Previous answer")))

	builder := service.NewCompletion().User("New question")
	builder.memory = mem
	builder.conversationID = "conv-1"

	builder.loadConversationHistory(ctx)

	require.Len(t, builder.request.Messages, 3)
	assert.Equal(t, "Previous question", builder.request.Messages[0].Content)
	assert.Equal(t, "Previous answer", builder.request.Messages[1].Content)
	assert.Equal(t, "New question", builder.request.Messages[2].Content)
}

func TestCompletionBuilder_LoadConversationHistory_EmptyHistory(t *testing.T) {
	service := NewService("")
	store := NewMockMemoryStore()
	mem := NewBufferMemory(store, WithBufferSize(100))

	builder := service.NewCompletion().User("Hello")
	builder.memory = mem
	builder.conversationID = "nonexistent-conv"

	originalLen := len(builder.request.Messages)
	builder.loadConversationHistory(context.Background())

	assert.Equal(t, originalLen, len(builder.request.Messages))
}

func TestCompletionBuilder_RecordToMemory_NoMemory(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion().User("Hello")

	response := &llm_dto.CompletionResponse{
		Choices: []llm_dto.Choice{{Message: llm_dto.Message{Role: llm_dto.RoleAssistant, Content: "Hi"}}},
	}

	builder.recordToMemory(context.Background(), builder.request.Messages, response)
}

func TestCompletionBuilder_RecordToMemory_NilResponse(t *testing.T) {
	service := NewService("")
	store := NewMockMemoryStore()
	mem := NewBufferMemory(store, WithBufferSize(100))
	builder := service.NewCompletion().User("Hello")
	builder.memory = mem
	builder.conversationID = "conv-1"

	builder.recordToMemory(context.Background(), builder.request.Messages, nil)
}

func TestCompletionBuilder_RecordToMemory_WithResponse(t *testing.T) {
	service := NewService("")
	store := NewMockMemoryStore()
	mem := NewBufferMemory(store, WithBufferSize(100))
	builder := service.NewCompletion().User("Hello")
	builder.memory = mem
	builder.conversationID = "conv-1"

	response := &llm_dto.CompletionResponse{
		Choices: []llm_dto.Choice{{Message: llm_dto.Message{Role: llm_dto.RoleAssistant, Content: "Hi there"}}},
	}

	builder.recordToMemory(context.Background(), builder.request.Messages, response)

	messages, err := mem.GetMessages(context.Background(), "conv-1")
	require.NoError(t, err)
	require.Len(t, messages, 2)
	assert.Equal(t, "Hello", messages[0].Content)
	assert.Equal(t, "Hi there", messages[1].Content)
}

func TestCompletionBuilder_RecordAssistantResponse_EmptyContent(t *testing.T) {
	service := NewService("")
	store := NewMockMemoryStore()
	mem := NewBufferMemory(store, WithBufferSize(100))
	builder := service.NewCompletion()
	builder.memory = mem
	builder.conversationID = "conv-1"

	response := &llm_dto.CompletionResponse{
		Choices: []llm_dto.Choice{{Message: llm_dto.Message{Role: llm_dto.RoleAssistant, Content: ""}}},
	}

	builder.recordAssistantResponse(context.Background(), response)

	messages, err := mem.GetMessages(context.Background(), "conv-1")
	require.NoError(t, err)
	assert.Empty(t, messages)
}

func TestCompletionBuilder_RecordAssistantResponse_WithContent(t *testing.T) {
	service := NewService("")
	store := NewMockMemoryStore()
	mem := NewBufferMemory(store, WithBufferSize(100))

	require.NoError(t, mem.AddMessage(context.Background(), "conv-1", llm_dto.NewUserMessage("Hi")))

	builder := service.NewCompletion()
	builder.memory = mem
	builder.conversationID = "conv-1"

	response := &llm_dto.CompletionResponse{
		Choices: []llm_dto.Choice{{Message: llm_dto.Message{Role: llm_dto.RoleAssistant, Content: "Hello!"}}},
	}

	builder.recordAssistantResponse(context.Background(), response)

	messages, err := mem.GetMessages(context.Background(), "conv-1")
	require.NoError(t, err)
	require.Len(t, messages, 2)
	assert.Equal(t, "Hello!", messages[1].Content)
}

func TestRecursiveCharacterSplitter_HardSplit_Direct(t *testing.T) {
	s, err := NewRecursiveCharacterSplitter(5, 0)
	require.NoError(t, err)

	chunks := s.hardSplit("abcdefghijklmno")

	require.Len(t, chunks, 3)
	assert.Equal(t, "abcde", chunks[0])
	assert.Equal(t, "fghij", chunks[1])
	assert.Equal(t, "klmno", chunks[2])
}

func TestRecursiveCharacterSplitter_HardSplit_WithOverlap(t *testing.T) {
	s, err := NewRecursiveCharacterSplitter(5, 2)
	require.NoError(t, err)

	chunks := s.hardSplit("abcdefghij")

	require.True(t, len(chunks) >= 2)
	assert.Equal(t, "abcde", chunks[0])

	assert.Equal(t, "defgh", chunks[1])
}

func TestRecursiveCharacterSplitter_HardSplit_ShortText(t *testing.T) {
	s, err := NewRecursiveCharacterSplitter(10, 0)
	require.NoError(t, err)

	chunks := s.hardSplit("short")

	require.Len(t, chunks, 1)
	assert.Equal(t, "short", chunks[0])
}

func TestRecursiveCharacterSplitter_HardSplit_ExactChunkSize(t *testing.T) {
	s, err := NewRecursiveCharacterSplitter(5, 0)
	require.NoError(t, err)

	chunks := s.hardSplit("abcde")

	require.Len(t, chunks, 1)
	assert.Equal(t, "abcde", chunks[0])
}
