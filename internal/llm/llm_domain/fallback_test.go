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
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/maths"
)

func TestNewFallbackRouter(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	router := NewFallbackRouter(service)

	require.NotNil(t, router)
	assert.Equal(t, service, router.service)
}

func TestFallbackRouter_Execute_NilConfig(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	router := NewFallbackRouter(service)

	response, result, err := router.Execute(
		context.Background(),
		nil,
		&llm_dto.CompletionRequest{Model: "gpt-4o"},
		"",
		maths.ZeroMoney(llm_dto.CostCurrency),
		nil,
	)

	assert.Nil(t, response)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one provider")
}

func TestFallbackRouter_Execute_EmptyProviders(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	router := NewFallbackRouter(service)

	config := &llm_dto.FallbackConfig{
		Providers: []string{},
	}

	response, result, err := router.Execute(
		context.Background(),
		config,
		&llm_dto.CompletionRequest{Model: "gpt-4o"},
		"",
		maths.ZeroMoney(llm_dto.CostCurrency),
		nil,
	)

	assert.Nil(t, response)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one provider")
}

func TestFallbackRouter_Execute_FirstProviderSucceeds(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	mock := NewMockLLMProvider()
	_ = service.RegisterProvider(context.Background(), "openai", mock)

	router := NewFallbackRouter(service)
	config := llm_dto.NewFallbackConfig("openai")

	request := &llm_dto.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
	}

	response, result, err := router.Execute(
		context.Background(),
		config,
		request,
		"",
		maths.ZeroMoney(llm_dto.CostCurrency),
		nil,
	)

	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, result)
	assert.Equal(t, "openai", result.UsedProvider)
	assert.Equal(t, []string{"openai"}, result.AttemptedProviders)
	assert.Empty(t, result.Errors)
}

func TestFallbackRouter_Execute_FallbackToSecondProvider(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	failingProvider := NewMockLLMProvider()
	failingProvider.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		return nil, ErrProviderOverloaded
	}

	successProvider := NewMockLLMProvider()

	_ = service.RegisterProvider(context.Background(), "primary", failingProvider)
	_ = service.RegisterProvider(context.Background(), "secondary", successProvider)

	router := NewFallbackRouter(service)
	config := llm_dto.NewFallbackConfig("primary", "secondary")

	request := &llm_dto.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
	}

	response, result, err := router.Execute(
		context.Background(),
		config,
		request,
		"",
		maths.ZeroMoney(llm_dto.CostCurrency),
		nil,
	)

	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, result)
	assert.Equal(t, "secondary", result.UsedProvider)
	assert.Equal(t, []string{"primary", "secondary"}, result.AttemptedProviders)
	assert.NotNil(t, result.Errors["primary"])
}

func TestFallbackRouter_Execute_AllProvidersFail(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	failingProvider1 := NewMockLLMProvider()
	failingProvider1.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		return nil, ErrProviderOverloaded
	}

	failingProvider2 := NewMockLLMProvider()
	failingProvider2.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		return nil, ErrProviderTimeout
	}

	_ = service.RegisterProvider(context.Background(), "p1", failingProvider1)
	_ = service.RegisterProvider(context.Background(), "p2", failingProvider2)

	router := NewFallbackRouter(service)
	config := llm_dto.NewFallbackConfig("p1", "p2")

	request := &llm_dto.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
	}

	response, result, err := router.Execute(
		context.Background(),
		config,
		request,
		"",
		maths.ZeroMoney(llm_dto.CostCurrency),
		nil,
	)

	assert.Nil(t, response)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "all 2 fallback providers failed")
	require.NotNil(t, result)
	assert.Len(t, result.AttemptedProviders, 2)
	assert.Len(t, result.Errors, 2)
}

func TestFallbackRouter_Execute_CancelledContext(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	mock := NewMockLLMProvider()
	_ = service.RegisterProvider(context.Background(), "openai", mock)

	router := NewFallbackRouter(service)
	config := llm_dto.NewFallbackConfig("openai")

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	request := &llm_dto.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
	}

	response, result, err := router.Execute(
		ctx,
		config,
		request,
		"",
		maths.ZeroMoney(llm_dto.CostCurrency),
		nil,
	)

	assert.Nil(t, response)
	assert.Error(t, err)
	require.NotNil(t, result)
}

func TestFallbackRouter_Execute_WithModelMapping(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	mock := NewMockLLMProvider()

	var capturedModel string
	mock.CompleteFunc = func(_ context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		capturedModel = request.Model
		return &llm_dto.CompletionResponse{
			ID:    "mock-response-id",
			Model: request.Model,
			Choices: []llm_dto.Choice{{
				Index:        0,
				Message:      llm_dto.Message{Role: llm_dto.RoleAssistant, Content: "Mock response"},
				FinishReason: "stop",
			}},
			Usage: &llm_dto.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		}, nil
	}
	_ = service.RegisterProvider(context.Background(), "anthropic", mock)

	router := NewFallbackRouter(service)
	config := llm_dto.NewFallbackConfig("anthropic").
		WithModelMapping(map[string]string{
			"anthropic": "claude-3-5-sonnet-20241022",
		})

	request := &llm_dto.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
	}

	response, result, err := router.Execute(
		context.Background(),
		config,
		request,
		"",
		maths.ZeroMoney(llm_dto.CostCurrency),
		nil,
	)

	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, result)
	assert.Equal(t, "anthropic", result.UsedProvider)

	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.CompleteCallCount))
	assert.Equal(t, "claude-3-5-sonnet-20241022", capturedModel)
}

func TestFallbackRouter_Execute_DefaultTriggersToAll(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	failingProvider := NewMockLLMProvider()
	failingProvider.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		return nil, errors.New("some arbitrary error")
	}

	successProvider := NewMockLLMProvider()

	_ = service.RegisterProvider(context.Background(), "p1", failingProvider)
	_ = service.RegisterProvider(context.Background(), "p2", successProvider)

	router := NewFallbackRouter(service)

	config := &llm_dto.FallbackConfig{
		Providers: []string{"p1", "p2"},
		Triggers:  0,
	}

	request := &llm_dto.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
	}

	response, result, err := router.Execute(
		context.Background(),
		config,
		request,
		"",
		maths.ZeroMoney(llm_dto.CostCurrency),
		nil,
	)

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, "p2", result.UsedProvider)
}

func TestFallbackRouter_ShouldFallback_OnError(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	router := NewFallbackRouter(service)

	assert.True(t, router.shouldFallback(errors.New("any error"), llm_dto.FallbackOnError))
	assert.True(t, router.shouldFallback(ErrProviderTimeout, llm_dto.FallbackOnError))
}

func TestFallbackRouter_ShouldFallback_OnRateLimit(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	router := NewFallbackRouter(service)

	assert.True(t, router.shouldFallback(ErrRateLimited, llm_dto.FallbackOnRateLimit))
	assert.False(t, router.shouldFallback(ErrProviderTimeout, llm_dto.FallbackOnRateLimit))
	assert.False(t, router.shouldFallback(errors.New("random error"), llm_dto.FallbackOnRateLimit))
}

func TestFallbackRouter_ShouldFallback_OnTimeout(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	router := NewFallbackRouter(service)

	assert.True(t, router.shouldFallback(ErrProviderTimeout, llm_dto.FallbackOnTimeout))
	assert.False(t, router.shouldFallback(ErrRateLimited, llm_dto.FallbackOnTimeout))
	assert.False(t, router.shouldFallback(errors.New("random error"), llm_dto.FallbackOnTimeout))
}

func TestFallbackRouter_ShouldFallback_OnBudgetExceeded(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	router := NewFallbackRouter(service)

	assert.True(t, router.shouldFallback(ErrBudgetExceeded, llm_dto.FallbackOnBudgetExceeded))
	assert.True(t, router.shouldFallback(ErrMaxCostExceeded, llm_dto.FallbackOnBudgetExceeded))
	assert.False(t, router.shouldFallback(ErrRateLimited, llm_dto.FallbackOnBudgetExceeded))
}

func TestFallbackRouter_ShouldFallback_OnAll(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	router := NewFallbackRouter(service)

	assert.True(t, router.shouldFallback(ErrRateLimited, llm_dto.FallbackOnAll))
	assert.True(t, router.shouldFallback(ErrProviderTimeout, llm_dto.FallbackOnAll))
	assert.True(t, router.shouldFallback(ErrBudgetExceeded, llm_dto.FallbackOnAll))
	assert.True(t, router.shouldFallback(errors.New("any error"), llm_dto.FallbackOnAll))
}

func TestFallbackRouter_ShouldFallback_NoMatch(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	router := NewFallbackRouter(service)

	assert.False(t, router.shouldFallback(ErrProviderTimeout, llm_dto.FallbackOnRateLimit))

	assert.False(t, router.shouldFallback(ErrRateLimited, llm_dto.FallbackOnTimeout))
}

func TestFallbackRouter_Execute_NonRetryableErrorStopsFallback(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	failingProvider := NewMockLLMProvider()
	failingProvider.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		return nil, ErrProviderTimeout
	}

	neverCalledProvider := NewMockLLMProvider()

	_ = service.RegisterProvider(context.Background(), "p1", failingProvider)
	_ = service.RegisterProvider(context.Background(), "p2", neverCalledProvider)

	router := NewFallbackRouter(service)

	config := &llm_dto.FallbackConfig{
		Providers: []string{"p1", "p2"},
		Triggers:  llm_dto.FallbackOnRateLimit,
	}

	request := &llm_dto.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
	}

	response, result, err := router.Execute(
		context.Background(),
		config,
		request,
		"",
		maths.ZeroMoney(llm_dto.CostCurrency),
		nil,
	)

	assert.Nil(t, response)
	assert.Error(t, err)
	require.NotNil(t, result)

	assert.Len(t, result.AttemptedProviders, 1)
	assert.Equal(t, int64(0), atomic.LoadInt64(&neverCalledProvider.CompleteCallCount))
}

func TestFallbackRouter_Execute_WithRetryPolicy(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	callCount := 0
	failingThenSuccessProvider := NewMockLLMProvider()
	failingThenSuccessProvider.CompleteFunc = func(_ context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
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

	_ = service.RegisterProvider(context.Background(), "retry-provider", failingThenSuccessProvider)

	router := NewFallbackRouter(service)
	config := llm_dto.NewFallbackConfig("retry-provider")

	retryPolicy := &llm_dto.RetryPolicy{
		MaxRetries:        5,
		InitialBackoff:    1,
		MaxBackoff:        10,
		BackoffMultiplier: 1.0,
		JitterFraction:    0,
	}

	request := &llm_dto.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
	}

	response, result, err := router.Execute(
		context.Background(),
		config,
		request,
		"",
		maths.ZeroMoney(llm_dto.CostCurrency),
		retryPolicy,
	)

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, "success", response.ID)
	assert.Equal(t, "retry-provider", result.UsedProvider)
}

func TestFallback_DeepCopyMessages(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	primary := NewMockLLMProvider()
	primary.CompleteFunc = func(_ context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		request.Messages = append(request.Messages, llm_dto.Message{Role: llm_dto.RoleAssistant, Content: "injected"})
		return nil, ErrProviderOverloaded
	}

	secondary := NewMockLLMProvider()
	var secondaryMessageCount int
	secondary.CompleteFunc = func(_ context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		secondaryMessageCount = len(request.Messages)
		return &llm_dto.CompletionResponse{
			ID: "ok", Model: "m", Choices: []llm_dto.Choice{{
				Message:      llm_dto.Message{Role: llm_dto.RoleAssistant, Content: "ok"},
				FinishReason: llm_dto.FinishReasonStop,
			}},
		}, nil
	}

	require.NoError(t, service.RegisterProvider(context.Background(), "primary", primary))
	require.NoError(t, service.RegisterProvider(context.Background(), "secondary", secondary))

	router := NewFallbackRouter(service)
	config := llm_dto.NewFallbackConfig("primary", "secondary")

	request := &llm_dto.CompletionRequest{
		Model:    "m",
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "hello"}},
	}

	_, _, err := router.Execute(context.Background(), config, request, "", maths.ZeroMoney(llm_dto.CostCurrency), nil)
	require.NoError(t, err)

	assert.Equal(t, 1, secondaryMessageCount)
	assert.Len(t, request.Messages, 1)
}

func TestFallback_DeepCopyProviderOptions(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	primary := NewMockLLMProvider()
	primary.CompleteFunc = func(_ context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		request.ProviderOptions["injected"] = true
		return nil, ErrProviderOverloaded
	}

	secondary := NewMockLLMProvider()
	var secondaryOptCount int
	secondary.CompleteFunc = func(_ context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		secondaryOptCount = len(request.ProviderOptions)
		return &llm_dto.CompletionResponse{
			ID: "ok", Model: "m", Choices: []llm_dto.Choice{{
				Message:      llm_dto.Message{Role: llm_dto.RoleAssistant, Content: "ok"},
				FinishReason: llm_dto.FinishReasonStop,
			}},
		}, nil
	}

	require.NoError(t, service.RegisterProvider(context.Background(), "primary", primary))
	require.NoError(t, service.RegisterProvider(context.Background(), "secondary", secondary))

	router := NewFallbackRouter(service)
	config := llm_dto.NewFallbackConfig("primary", "secondary")

	request := &llm_dto.CompletionRequest{
		Model:           "m",
		Messages:        []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "hello"}},
		ProviderOptions: map[string]any{"original": "value"},
	}

	_, _, err := router.Execute(context.Background(), config, request, "", maths.ZeroMoney(llm_dto.CostCurrency), nil)
	require.NoError(t, err)

	assert.Equal(t, 1, secondaryOptCount)
	assert.Len(t, request.ProviderOptions, 1)
}

func TestFallback_DeepCopyToolParameters(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	primary := NewMockLLMProvider()
	primary.CompleteFunc = func(_ context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		if len(request.Tools) > 0 && request.Tools[0].Function.Parameters != nil {
			request.Tools[0].Function.Parameters.Description = new("mutated")
		}
		return nil, ErrProviderOverloaded
	}

	secondary := NewMockLLMProvider()
	var secondaryToolDesc *string
	secondary.CompleteFunc = func(_ context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		if len(request.Tools) > 0 && request.Tools[0].Function.Parameters != nil {
			secondaryToolDesc = request.Tools[0].Function.Parameters.Description
		}
		return &llm_dto.CompletionResponse{
			ID: "ok", Model: "m", Choices: []llm_dto.Choice{{
				Message:      llm_dto.Message{Role: llm_dto.RoleAssistant, Content: "ok"},
				FinishReason: llm_dto.FinishReasonStop,
			}},
		}, nil
	}

	require.NoError(t, service.RegisterProvider(context.Background(), "primary", primary))
	require.NoError(t, service.RegisterProvider(context.Background(), "secondary", secondary))

	router := NewFallbackRouter(service)
	config := llm_dto.NewFallbackConfig("primary", "secondary")

	request := &llm_dto.CompletionRequest{
		Model:    "m",
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "hello"}},
		Tools: []llm_dto.ToolDefinition{
			llm_dto.NewFunctionTool("test", "test tool", &llm_dto.JSONSchema{
				Type:        "object",
				Description: new("original description"),
			}),
		},
	}

	_, _, err := router.Execute(context.Background(), config, request, "", maths.ZeroMoney(llm_dto.CostCurrency), nil)
	require.NoError(t, err)

	require.NotNil(t, secondaryToolDesc)
	assert.Equal(t, "original description", *secondaryToolDesc)
	assert.Equal(t, "original description", *request.Tools[0].Function.Parameters.Description)
}
