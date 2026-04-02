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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/llm/llm_dto"
)

func TestResponseValidator_BuilderSetsField(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	validatorFunction := ResponseValidatorFunc(func(_ context.Context, _ *llm_dto.CompletionResponse) error {
		return nil
	})

	b := service.NewCompletion().ResponseValidator(validatorFunction)

	assert.NotNil(t, b.responseValidator, "responseValidator should be set after calling ResponseValidator()")
}

func TestResponseValidator_NilByDefault(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	b := service.NewCompletion()

	assert.Nil(t, b.responseValidator, "responseValidator should be nil by default")
}

func TestResponseValidator_AcceptingValidatorAllowsDoToSucceed(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	var receivedResp *llm_dto.CompletionResponse
	validator := ResponseValidatorFunc(func(_ context.Context, response *llm_dto.CompletionResponse) error {
		receivedResp = response
		return nil
	})

	response, err := service.NewCompletion().
		Model("mock-model").
		User("Hello").
		ResponseValidator(validator).
		Do(context.Background())

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.NotNil(t, receivedResp, "validator should have been called")
	assert.Equal(t, response, receivedResp, "validator should receive the same response")
}

func TestResponseValidator_RejectingValidatorCausesDoToFail(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	validationErr := errors.New("response contains harmful content")
	validator := ResponseValidatorFunc(func(_ context.Context, _ *llm_dto.CompletionResponse) error {
		return validationErr
	})

	_, err := service.NewCompletion().
		Model("mock-model").
		User("Hello").
		ResponseValidator(validator).
		Do(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "response validation failed")
	assert.ErrorIs(t, err, validationErr)
}

func TestResponseValidator_NoValidatorDoSucceedsNormally(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	response, err := service.NewCompletion().
		Model("mock-model").
		User("Hello").
		Do(context.Background())

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, "Mock response", response.Content())
}

func TestResponseValidator_ReceivesActualResponse(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	provider.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		return &llm_dto.CompletionResponse{
			ID:    "test-id-123",
			Model: "test-model",
			Choices: []llm_dto.Choice{
				{
					Index: 0,
					Message: llm_dto.Message{
						Role:    llm_dto.RoleAssistant,
						Content: "Specific test content",
					},
					FinishReason: llm_dto.FinishReasonStop,
				},
			},
		}, nil
	}
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	var validatedResp *llm_dto.CompletionResponse
	validator := ResponseValidatorFunc(func(_ context.Context, response *llm_dto.CompletionResponse) error {
		validatedResp = response
		return nil
	})

	_, err := service.NewCompletion().
		Model("test-model").
		User("Hello").
		ResponseValidator(validator).
		Do(context.Background())

	require.NoError(t, err)
	require.NotNil(t, validatedResp)
	assert.Equal(t, "test-id-123", validatedResp.ID)
	assert.Equal(t, "Specific test content", validatedResp.Content())
}

func TestResponseValidator_FailurePreventsMemoryRecording(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	memStore := NewMockMemoryStore()
	mem := NewBufferMemory(memStore, WithBufferSize(10))

	validator := ResponseValidatorFunc(func(_ context.Context, _ *llm_dto.CompletionResponse) error {
		return errors.New("validation failed")
	})

	_, err := service.NewCompletion().
		Model("mock-model").
		User("Hello").
		Memory(mem, "conv-1").
		ResponseValidator(validator).
		Do(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "response validation failed")

	_, loadErr := memStore.Load(context.Background(), "conv-1")
	assert.ErrorIs(t, loadErr, ErrConversationNotFound,
		"memory should not have been written when validator rejects")
}

func TestResponseValidator_ChainsWithBuilder(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	validator := ResponseValidatorFunc(func(_ context.Context, _ *llm_dto.CompletionResponse) error {
		return nil
	})

	b := service.NewCompletion().
		Model("mock-model").
		User("Hello").
		ResponseValidator(validator).
		MaxTokens(100)

	assert.NotNil(t, b.responseValidator)
	request := b.Build()
	require.NotNil(t, request.MaxTokens)
	assert.Equal(t, 100, *request.MaxTokens)
}
