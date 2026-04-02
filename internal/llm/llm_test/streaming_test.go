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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
)

func TestStream_FullFlow(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()

	h.mockProvider.SetStreamChunks([]llm_dto.StreamChunk{
		{
			ID:    "chunk-1",
			Model: "test-model",
			Delta: &llm_dto.MessageDelta{Content: new("Hello")},
		},
		{
			ID:    "chunk-2",
			Model: "test-model",
			Delta: &llm_dto.MessageDelta{Content: new(" world")},
		},
		{
			ID:    "chunk-3",
			Model: "test-model",
			Delta: &llm_dto.MessageDelta{Content: new("!")},
		},
	})

	events, err := h.service.NewCompletion().
		Model("test-model").
		User("Say hello world").
		Stream(ctx)

	require.NoError(t, err)
	require.NotNil(t, events)

	var chunks []string
	var gotDone bool
	for event := range events {
		switch event.Type {
		case llm_dto.StreamEventChunk:
			require.NotNil(t, event.Chunk, "chunk event should have a Chunk field")
			if event.Chunk.Delta != nil && event.Chunk.Delta.Content != nil {
				chunks = append(chunks, *event.Chunk.Delta.Content)
			}
		case llm_dto.StreamEventDone:
			gotDone = true
		case llm_dto.StreamEventError:
			t.Fatalf("unexpected error event: %v", event.Error)
		}
	}

	assert.True(t, gotDone, "should receive a done event")
	assert.Equal(t, []string{"Hello", " world", "!"}, chunks)

	assert.Len(t, h.mockProvider.GetStreamCalls(), 1)
}

func TestStream_WithBudget(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()
	h.service.SetBudget("stream-budget", &llm_dto.BudgetConfig{})

	h.mockProvider.SetStreamChunks([]llm_dto.StreamChunk{
		{
			ID:    "chunk-1",
			Model: "test-model",
			Delta: &llm_dto.MessageDelta{Content: new("Streamed")},
		},
	})

	events, err := h.service.NewCompletion().
		Model("test-model").
		User("Stream with budget").
		BudgetScope("stream-budget").
		Stream(ctx)

	require.NoError(t, err)

	for range events {
	}

	assert.Len(t, h.mockProvider.GetStreamCalls(), 1)
}

func TestStream_ProviderError(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()

	h.mockProvider.SetStreamError(errors.New("stream failed"))

	events, err := h.service.NewCompletion().
		Model("test-model").
		User("Fail stream").
		Stream(ctx)

	require.NoError(t, err, "Stream() itself should not error; errors arrive on the channel")
	require.NotNil(t, events)

	var gotError bool
	for event := range events {
		if event.Type == llm_dto.StreamEventError {
			gotError = true
			assert.Error(t, event.Error)
		}
	}

	assert.True(t, gotError, "should receive an error event on the stream channel")
}

func TestStream_UnsupportedProvider(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()

	h.mockProvider.SetSupportsStreaming(false)

	_, err := h.service.NewCompletion().
		Model("test-model").
		User("No streaming").
		Stream(ctx)

	require.Error(t, err)
	assert.ErrorIs(t, err, llm_domain.ErrStreamingNotSupported)
}
