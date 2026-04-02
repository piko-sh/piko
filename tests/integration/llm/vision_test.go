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

//go:build integration

package llm_integration_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
)

func TestVision_ContentPartsPassedThroughService(t *testing.T) {
	var captured atomic.Pointer[llm_dto.CompletionRequest]

	mock := llm_domain.NewMockLLMProvider()
	mock.DefaultModelValue = "mock-vision"
	mock.CompleteFunc = func(_ context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		captured.Store(request)
		return &llm_dto.CompletionResponse{
			Model: "mock-vision",
			Choices: []llm_dto.Choice{
				{
					Message:      llm_dto.Message{Role: llm_dto.RoleAssistant, Content: "I see an image"},
					FinishReason: llm_dto.FinishReasonStop,
				},
			},
			Usage: &llm_dto.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		}, nil
	}

	ctx, cancel := context.WithTimeoutCause(t.Context(), perTestTimeout,
		fmt.Errorf("test: vision integration test exceeded %s timeout", perTestTimeout))
	defer cancel()

	service := llm_domain.NewService("mock-vision")
	require.NoError(t, service.RegisterProvider(ctx, "mock-vision", mock))
	require.NoError(t, service.SetDefaultProvider(ctx, "mock-vision"))

	response, err := service.Complete(ctx, &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			llm_dto.NewUserMessageWithImageURL("Describe this image", "https://example.com/photo.jpg"),
		},
	})
	require.NoError(t, err)
	require.NotNil(t, response)

	got := captured.Load()
	require.NotNil(t, got, "expected provider to receive request")
	require.Len(t, got.Messages, 1)

	message := got.Messages[0]
	assert.Equal(t, llm_dto.RoleUser, message.Role)
	require.Len(t, message.ContentParts, 2)
	assert.Equal(t, llm_dto.ContentPartTypeText, message.ContentParts[0].Type)
	require.NotNil(t, message.ContentParts[0].Text)
	assert.Equal(t, "Describe this image", *message.ContentParts[0].Text)
	assert.Equal(t, llm_dto.ContentPartTypeImageURL, message.ContentParts[1].Type)
	require.NotNil(t, message.ContentParts[1].ImageURL)
	assert.Equal(t, "https://example.com/photo.jpg", message.ContentParts[1].ImageURL.URL)
}

func TestVision_ImageDataPassedThroughService(t *testing.T) {
	var captured atomic.Pointer[llm_dto.CompletionRequest]

	mock := llm_domain.NewMockLLMProvider()
	mock.DefaultModelValue = "mock-vision"
	mock.CompleteFunc = func(_ context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		captured.Store(request)
		return &llm_dto.CompletionResponse{
			Model: "mock-vision",
			Choices: []llm_dto.Choice{
				{
					Message:      llm_dto.Message{Role: llm_dto.RoleAssistant, Content: "I see pixels"},
					FinishReason: llm_dto.FinishReasonStop,
				},
			},
			Usage: &llm_dto.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		}, nil
	}

	ctx, cancel := context.WithTimeoutCause(t.Context(), perTestTimeout,
		fmt.Errorf("test: vision integration test exceeded %s timeout", perTestTimeout))
	defer cancel()

	service := llm_domain.NewService("mock-vision")
	require.NoError(t, service.RegisterProvider(ctx, "mock-vision", mock))
	require.NoError(t, service.SetDefaultProvider(ctx, "mock-vision"))

	response, err := service.Complete(ctx, &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			llm_dto.NewUserMessageWithImageData("What is this?", "image/png", "iVBORw0KGgo="),
		},
	})
	require.NoError(t, err)
	require.NotNil(t, response)

	got := captured.Load()
	require.NotNil(t, got)
	require.Len(t, got.Messages, 1)

	message := got.Messages[0]
	require.Len(t, message.ContentParts, 2)
	assert.Equal(t, llm_dto.ContentPartTypeText, message.ContentParts[0].Type)
	assert.Equal(t, llm_dto.ContentPartTypeImageData, message.ContentParts[1].Type)
	require.NotNil(t, message.ContentParts[1].ImageData)
	assert.Equal(t, "image/png", message.ContentParts[1].ImageData.MIMEType)
	assert.Equal(t, "iVBORw0KGgo=", message.ContentParts[1].ImageData.Data)
}

func TestVision_MultipleImagesPassedThroughService(t *testing.T) {
	var captured atomic.Pointer[llm_dto.CompletionRequest]

	mock := llm_domain.NewMockLLMProvider()
	mock.DefaultModelValue = "mock-vision"
	mock.CompleteFunc = func(_ context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		captured.Store(request)
		return &llm_dto.CompletionResponse{
			Model: "mock-vision",
			Choices: []llm_dto.Choice{
				{
					Message:      llm_dto.Message{Role: llm_dto.RoleAssistant, Content: "I see two images"},
					FinishReason: llm_dto.FinishReasonStop,
				},
			},
			Usage: &llm_dto.Usage{PromptTokens: 20, CompletionTokens: 5, TotalTokens: 25},
		}, nil
	}

	ctx, cancel := context.WithTimeoutCause(t.Context(), perTestTimeout,
		fmt.Errorf("test: vision integration test exceeded %s timeout", perTestTimeout))
	defer cancel()

	service := llm_domain.NewService("mock-vision")
	require.NoError(t, service.RegisterProvider(ctx, "mock-vision", mock))
	require.NoError(t, service.SetDefaultProvider(ctx, "mock-vision"))

	response, err := service.Complete(ctx, &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			llm_dto.NewUserMessageWithImages("Compare these images",
				llm_dto.ImageURLPart("https://example.com/a.jpg"),
				llm_dto.ImageDataPart("image/png", "dGVzdA=="),
			),
		},
	})
	require.NoError(t, err)
	require.NotNil(t, response)

	got := captured.Load()
	require.NotNil(t, got)
	require.Len(t, got.Messages, 1)

	message := got.Messages[0]
	require.Len(t, message.ContentParts, 3)
	assert.Equal(t, llm_dto.ContentPartTypeText, message.ContentParts[0].Type)
	assert.Equal(t, llm_dto.ContentPartTypeImageURL, message.ContentParts[1].Type)
	assert.Equal(t, llm_dto.ContentPartTypeImageData, message.ContentParts[2].Type)
}

func TestVision_StreamWithContentParts(t *testing.T) {
	var captured atomic.Pointer[llm_dto.CompletionRequest]

	mock := llm_domain.NewMockLLMProvider()
	mock.DefaultModelValue = "mock-vision"
	mock.StreamFunc = func(_ context.Context, request *llm_dto.CompletionRequest) (<-chan llm_dto.StreamEvent, error) {
		captured.Store(request)
		eventChannel := make(chan llm_dto.StreamEvent, 2)
		eventChannel <- llm_dto.StreamEvent{
			Type: llm_dto.StreamEventChunk,
			Chunk: &llm_dto.StreamChunk{
				Delta: &llm_dto.MessageDelta{Content: new("I see the image")},
			},
		}
		eventChannel <- llm_dto.StreamEvent{
			Type: llm_dto.StreamEventDone,
			Done: true,
			FinalResponse: &llm_dto.CompletionResponse{
				Model: "mock-vision",
				Usage: &llm_dto.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
			},
		}
		close(eventChannel)
		return eventChannel, nil
	}

	ctx, cancel := context.WithTimeoutCause(t.Context(), perTestTimeout,
		fmt.Errorf("test: vision streaming integration test exceeded %s timeout", perTestTimeout))
	defer cancel()

	service := llm_domain.NewService("mock-vision")
	require.NoError(t, service.RegisterProvider(ctx, "mock-vision", mock))
	require.NoError(t, service.SetDefaultProvider(ctx, "mock-vision"))

	events, err := service.Stream(ctx, &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			llm_dto.NewUserMessageWithImageURL("Describe this", "https://example.com/photo.jpg"),
		},
	})
	require.NoError(t, err)

	for range events {
	}

	got := captured.Load()
	require.NotNil(t, got, "expected provider to receive request")
	require.Len(t, got.Messages, 1)
	require.Len(t, got.Messages[0].ContentParts, 2)
	assert.Equal(t, llm_dto.ContentPartTypeText, got.Messages[0].ContentParts[0].Type)
	assert.Equal(t, llm_dto.ContentPartTypeImageURL, got.Messages[0].ContentParts[1].Type)
}

func TestVision_BuilderWithImage(t *testing.T) {
	var captured atomic.Pointer[llm_dto.CompletionRequest]

	mock := llm_domain.NewMockLLMProvider()
	mock.DefaultModelValue = "mock-vision"
	mock.CompleteFunc = func(_ context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		captured.Store(request)
		return &llm_dto.CompletionResponse{
			Model: "mock-vision",
			Choices: []llm_dto.Choice{
				{
					Message:      llm_dto.Message{Role: llm_dto.RoleAssistant, Content: "I see it"},
					FinishReason: llm_dto.FinishReasonStop,
				},
			},
			Usage: &llm_dto.Usage{PromptTokens: 10, CompletionTokens: 3, TotalTokens: 13},
		}, nil
	}

	ctx, cancel := context.WithTimeoutCause(t.Context(), perTestTimeout,
		fmt.Errorf("test: vision builder integration test exceeded %s timeout", perTestTimeout))
	defer cancel()

	service := llm_domain.NewService("mock-vision")
	require.NoError(t, service.RegisterProvider(ctx, "mock-vision", mock))
	require.NoError(t, service.SetDefaultProvider(ctx, "mock-vision"))

	response, err := service.NewCompletion().
		UserWithImage("Describe this image", "https://example.com/photo.jpg").
		Do(ctx)
	require.NoError(t, err)
	require.NotNil(t, response)

	got := captured.Load()
	require.NotNil(t, got)
	require.Len(t, got.Messages, 1)

	message := got.Messages[0]
	require.Len(t, message.ContentParts, 2)
	assert.Equal(t, llm_dto.ContentPartTypeText, message.ContentParts[0].Type)
	require.NotNil(t, message.ContentParts[0].Text)
	assert.Equal(t, "Describe this image", *message.ContentParts[0].Text)
	assert.Equal(t, llm_dto.ContentPartTypeImageURL, message.ContentParts[1].Type)
	require.NotNil(t, message.ContentParts[1].ImageURL)
	assert.Equal(t, "https://example.com/photo.jpg", message.ContentParts[1].ImageURL.URL)
}

func TestVision_BuilderWithImageData(t *testing.T) {
	var captured atomic.Pointer[llm_dto.CompletionRequest]

	mock := llm_domain.NewMockLLMProvider()
	mock.DefaultModelValue = "mock-vision"
	mock.CompleteFunc = func(_ context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		captured.Store(request)
		return &llm_dto.CompletionResponse{
			Model: "mock-vision",
			Choices: []llm_dto.Choice{
				{
					Message:      llm_dto.Message{Role: llm_dto.RoleAssistant, Content: "I see it"},
					FinishReason: llm_dto.FinishReasonStop,
				},
			},
			Usage: &llm_dto.Usage{PromptTokens: 10, CompletionTokens: 3, TotalTokens: 13},
		}, nil
	}

	ctx, cancel := context.WithTimeoutCause(t.Context(), perTestTimeout,
		fmt.Errorf("test: vision builder integration test exceeded %s timeout", perTestTimeout))
	defer cancel()

	service := llm_domain.NewService("mock-vision")
	require.NoError(t, service.RegisterProvider(ctx, "mock-vision", mock))
	require.NoError(t, service.SetDefaultProvider(ctx, "mock-vision"))

	response, err := service.NewCompletion().
		UserWithImageData("Analyse this", "image/png", "iVBORw0KGgo=").
		Do(ctx)
	require.NoError(t, err)
	require.NotNil(t, response)

	got := captured.Load()
	require.NotNil(t, got)
	require.Len(t, got.Messages, 1)

	message := got.Messages[0]
	require.Len(t, message.ContentParts, 2)
	assert.Equal(t, llm_dto.ContentPartTypeText, message.ContentParts[0].Type)
	assert.Equal(t, llm_dto.ContentPartTypeImageData, message.ContentParts[1].Type)
	require.NotNil(t, message.ContentParts[1].ImageData)
	assert.Equal(t, "image/png", message.ContentParts[1].ImageData.MIMEType)
	assert.Equal(t, "iVBORw0KGgo=", message.ContentParts[1].ImageData.Data)
}

func TestVision_MemoryPreservesContentParts(t *testing.T) {
	var callCount atomic.Int64
	var lastReq atomic.Pointer[llm_dto.CompletionRequest]

	mock := llm_domain.NewMockLLMProvider()
	mock.DefaultModelValue = "mock-vision"
	mock.CompleteFunc = func(_ context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		callCount.Add(1)
		lastReq.Store(request)
		return &llm_dto.CompletionResponse{
			Model: "mock-vision",
			Choices: []llm_dto.Choice{
				{
					Message:      llm_dto.Message{Role: llm_dto.RoleAssistant, Content: "Noted"},
					FinishReason: llm_dto.FinishReasonStop,
				},
			},
			Usage: &llm_dto.Usage{PromptTokens: 10, CompletionTokens: 1, TotalTokens: 11},
		}, nil
	}

	ctx, cancel := context.WithTimeoutCause(t.Context(), perTestTimeout,
		fmt.Errorf("test: vision memory integration test exceeded %s timeout", perTestTimeout))
	defer cancel()

	service := llm_domain.NewService("mock-vision")
	require.NoError(t, service.RegisterProvider(ctx, "mock-vision", mock))
	require.NoError(t, service.SetDefaultProvider(ctx, "mock-vision"))

	memStore := llm_domain.NewMockMemoryStore()

	_, err := service.NewCompletion().
		UserWithImage("Describe this", "https://example.com/photo.jpg").
		BufferMemory(memStore, "vision-conv", 100).
		Do(ctx)
	require.NoError(t, err)

	_, err = service.NewCompletion().
		User("What did I just show you?").
		BufferMemory(memStore, "vision-conv", 100).
		Do(ctx)
	require.NoError(t, err)

	require.Equal(t, int64(2), callCount.Load())

	got := lastReq.Load()
	require.NotNil(t, got)

	require.GreaterOrEqual(t, len(got.Messages), 3,
		"expected at least 3 messages (first user + assistant + second user)")

	firstUser := got.Messages[0]
	assert.Equal(t, llm_dto.RoleUser, firstUser.Role)
	require.Len(t, firstUser.ContentParts, 2,
		"memory should preserve ContentParts from first user message")
	assert.Equal(t, llm_dto.ContentPartTypeText, firstUser.ContentParts[0].Type)
	assert.Equal(t, llm_dto.ContentPartTypeImageURL, firstUser.ContentParts[1].Type)
}
