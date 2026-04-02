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
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/llm/llm_dto"
)

func TestNewBufferMemory(t *testing.T) {
	testCases := []struct {
		name            string
		maxSize         int
		expectedMaxSize int
	}{
		{
			name:            "uses provided max size",
			maxSize:         10,
			expectedMaxSize: 10,
		},
		{
			name:            "uses default for zero",
			maxSize:         0,
			expectedMaxSize: DefaultBufferSize,
		},
		{
			name:            "uses default for negative",
			maxSize:         -5,
			expectedMaxSize: DefaultBufferSize,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewMockMemoryStore()
			mem := NewBufferMemory(store, WithBufferSize(tc.maxSize))

			require.NotNil(t, mem)
			assert.Equal(t, tc.expectedMaxSize, mem.maxSize)
			assert.NotNil(t, mem.store)
		})
	}
}

func TestBufferMemory_Load(t *testing.T) {
	ctx := context.Background()

	t.Run("returns state when found", func(t *testing.T) {
		store := NewMockMemoryStore()
		state := llm_dto.NewConversationState("conv-1")
		state.AddMessage(llm_dto.NewUserMessage("Hello"))
		err := store.Save(ctx, state)
		require.NoError(t, err)

		mem := NewBufferMemory(store, WithBufferSize(10))
		loadedState, err := mem.Load(ctx, "conv-1")

		require.NoError(t, err)
		require.NotNil(t, loadedState)
		assert.Equal(t, "conv-1", loadedState.ID)
		assert.Len(t, loadedState.Messages, 1)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		store := NewMockMemoryStore()
		mem := NewBufferMemory(store, WithBufferSize(10))

		_, err := mem.Load(ctx, "non-existent")

		assert.ErrorIs(t, err, ErrConversationNotFound)
	})
}

func TestBufferMemory_AddMessage(t *testing.T) {
	ctx := context.Background()

	t.Run("creates new conversation when not found", func(t *testing.T) {
		store := NewMockMemoryStore()
		mem := NewBufferMemory(store, WithBufferSize(10))

		err := mem.AddMessage(ctx, "new-conv", llm_dto.NewUserMessage("Hello"))

		require.NoError(t, err)
		state, err := store.Load(ctx, "new-conv")
		require.NoError(t, err)
		assert.Len(t, state.Messages, 1)
		assert.Equal(t, "Hello", state.Messages[0].Content)
	})

	t.Run("adds to existing conversation", func(t *testing.T) {
		store := NewMockMemoryStore()
		state := llm_dto.NewConversationState("conv-1")
		state.AddMessage(llm_dto.NewUserMessage("First"))
		err := store.Save(ctx, state)
		require.NoError(t, err)

		mem := NewBufferMemory(store, WithBufferSize(10))
		err = mem.AddMessage(ctx, "conv-1", llm_dto.NewAssistantMessage("Second"))

		require.NoError(t, err)
		loadedState, err := store.Load(ctx, "conv-1")
		require.NoError(t, err)
		assert.Len(t, loadedState.Messages, 2)
	})

	t.Run("trims old messages when buffer full", func(t *testing.T) {
		store := NewMockMemoryStore()
		mem := NewBufferMemory(store, WithBufferSize(3))

		for i := range 5 {
			content := string(rune('A' + i))
			err := mem.AddMessage(ctx, "conv-1", llm_dto.NewUserMessage(content))
			require.NoError(t, err)
		}

		state, err := store.Load(ctx, "conv-1")
		require.NoError(t, err)
		assert.Len(t, state.Messages, 3)

		assert.Equal(t, "C", state.Messages[0].Content)
		assert.Equal(t, "D", state.Messages[1].Content)
		assert.Equal(t, "E", state.Messages[2].Content)
	})
}

func TestBufferMemory_GetMessages(t *testing.T) {
	ctx := context.Background()

	t.Run("returns empty slice for non-existent conversation", func(t *testing.T) {
		store := NewMockMemoryStore()
		mem := NewBufferMemory(store, WithBufferSize(10))

		messages, err := mem.GetMessages(ctx, "non-existent")

		require.NoError(t, err)
		assert.Empty(t, messages)
	})

	t.Run("returns copy of messages", func(t *testing.T) {
		store := NewMockMemoryStore()
		state := llm_dto.NewConversationState("conv-1")
		state.AddMessage(llm_dto.NewUserMessage("Hello"))
		err := store.Save(ctx, state)
		require.NoError(t, err)

		mem := NewBufferMemory(store, WithBufferSize(10))
		messages, err := mem.GetMessages(ctx, "conv-1")

		require.NoError(t, err)
		require.Len(t, messages, 1)
		assert.Equal(t, "Hello", messages[0].Content)

		messages[0].Content = "Modified"

		messages2, err := mem.GetMessages(ctx, "conv-1")
		require.NoError(t, err)
		assert.Equal(t, "Hello", messages2[0].Content)
	})
}

func TestBufferMemory_Clear(t *testing.T) {
	ctx := context.Background()

	store := NewMockMemoryStore()
	state := llm_dto.NewConversationState("conv-1")
	state.AddMessage(llm_dto.NewUserMessage("Hello"))
	err := store.Save(ctx, state)
	require.NoError(t, err)

	mem := NewBufferMemory(store, WithBufferSize(10))
	err = mem.Clear(ctx, "conv-1")

	require.NoError(t, err)
	_, err = store.Load(ctx, "conv-1")
	assert.ErrorIs(t, err, ErrConversationNotFound)
}

func TestNewWindowMemory(t *testing.T) {
	testCases := []struct {
		name               string
		tokenLimit         int
		expectedTokenLimit int
	}{
		{
			name:               "uses provided token limit",
			tokenLimit:         2000,
			expectedTokenLimit: 2000,
		},
		{
			name:               "uses default for zero",
			tokenLimit:         0,
			expectedTokenLimit: DefaultTokenLimit,
		},
		{
			name:               "uses default for negative",
			tokenLimit:         -100,
			expectedTokenLimit: DefaultTokenLimit,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewMockMemoryStore()
			mem := NewWindowMemory(store, WithTokenLimit(tc.tokenLimit))

			require.NotNil(t, mem)
			assert.Equal(t, tc.expectedTokenLimit, mem.tokenLimit)
		})
	}
}

func TestWindowMemory_Load(t *testing.T) {
	ctx := context.Background()

	t.Run("returns state when found", func(t *testing.T) {
		store := NewMockMemoryStore()
		state := llm_dto.NewConversationState("conv-1")
		err := store.Save(ctx, state)
		require.NoError(t, err)

		mem := NewWindowMemory(store, WithTokenLimit(4000))
		loadedState, err := mem.Load(ctx, "conv-1")

		require.NoError(t, err)
		require.NotNil(t, loadedState)
		assert.Equal(t, "conv-1", loadedState.ID)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		store := NewMockMemoryStore()
		mem := NewWindowMemory(store, WithTokenLimit(4000))

		_, err := mem.Load(ctx, "non-existent")

		assert.ErrorIs(t, err, ErrConversationNotFound)
	})
}

func TestWindowMemory_AddMessage(t *testing.T) {
	ctx := context.Background()

	t.Run("creates new conversation when not found", func(t *testing.T) {
		store := NewMockMemoryStore()
		mem := NewWindowMemory(store, WithTokenLimit(4000))

		err := mem.AddMessage(ctx, "new-conv", llm_dto.NewUserMessage("Hello"))

		require.NoError(t, err)
		state, err := store.Load(ctx, "new-conv")
		require.NoError(t, err)
		assert.Len(t, state.Messages, 1)
	})

	t.Run("trims old messages when token limit exceeded", func(t *testing.T) {
		store := NewMockMemoryStore()

		mem := NewWindowMemory(store, WithTokenLimit(50))

		for i := range 5 {
			content := strings.Repeat("A", 100) + string(rune('0'+i))
			err := mem.AddMessage(ctx, "conv-1", llm_dto.NewUserMessage(content))
			require.NoError(t, err)
		}

		state, err := store.Load(ctx, "conv-1")
		require.NoError(t, err)

		assert.Less(t, len(state.Messages), 5)
	})
}

func TestWindowMemory_GetMessages(t *testing.T) {
	ctx := context.Background()

	t.Run("returns empty slice for non-existent conversation", func(t *testing.T) {
		store := NewMockMemoryStore()
		mem := NewWindowMemory(store, WithTokenLimit(4000))

		messages, err := mem.GetMessages(ctx, "non-existent")

		require.NoError(t, err)
		assert.Empty(t, messages)
	})

	t.Run("returns copy of messages", func(t *testing.T) {
		store := NewMockMemoryStore()
		state := llm_dto.NewConversationState("conv-1")
		state.AddMessage(llm_dto.NewUserMessage("Hello"))
		err := store.Save(ctx, state)
		require.NoError(t, err)

		mem := NewWindowMemory(store, WithTokenLimit(4000))
		messages, err := mem.GetMessages(ctx, "conv-1")

		require.NoError(t, err)
		require.Len(t, messages, 1)
		assert.Equal(t, "Hello", messages[0].Content)
	})
}

func TestWindowMemory_Clear(t *testing.T) {
	ctx := context.Background()

	store := NewMockMemoryStore()
	state := llm_dto.NewConversationState("conv-1")
	err := store.Save(ctx, state)
	require.NoError(t, err)

	mem := NewWindowMemory(store, WithTokenLimit(4000))
	err = mem.Clear(ctx, "conv-1")

	require.NoError(t, err)
	_, err = store.Load(ctx, "conv-1")
	assert.ErrorIs(t, err, ErrConversationNotFound)
}

func TestNewSummaryMemory(t *testing.T) {
	t.Run("uses default buffer size when zero", func(t *testing.T) {
		store := NewMockMemoryStore()
		summariser := NewMockSummariser()
		config := llm_dto.MemoryConfig{BufferSize: 0}

		mem := NewSummaryMemory(store, summariser, config)

		require.NotNil(t, mem)
		assert.Equal(t, DefaultSummaryBufferSize, mem.config.BufferSize)
	})

	t.Run("uses default summary prompt when empty", func(t *testing.T) {
		store := NewMockMemoryStore()
		summariser := NewMockSummariser()
		config := llm_dto.MemoryConfig{BufferSize: 10, SummaryPrompt: ""}

		mem := NewSummaryMemory(store, summariser, config)

		require.NotNil(t, mem)
		assert.Contains(t, mem.config.SummaryPrompt, "Summarise")
	})

	t.Run("uses provided config", func(t *testing.T) {
		store := NewMockMemoryStore()
		summariser := NewMockSummariser()
		config := llm_dto.MemoryConfig{
			BufferSize:    15,
			SummaryPrompt: "Custom prompt: {{.Messages}}",
			SummaryModel:  "gpt-4o",
		}

		mem := NewSummaryMemory(store, summariser, config)

		require.NotNil(t, mem)
		assert.Equal(t, 15, mem.config.BufferSize)
		assert.Equal(t, "Custom prompt: {{.Messages}}", mem.config.SummaryPrompt)
		assert.Equal(t, "gpt-4o", mem.config.SummaryModel)
	})
}

func TestSummaryMemory_Load(t *testing.T) {
	ctx := context.Background()

	t.Run("returns state when found", func(t *testing.T) {
		store := NewMockMemoryStore()
		summariser := NewMockSummariser()
		state := llm_dto.NewConversationState("conv-1")
		err := store.Save(ctx, state)
		require.NoError(t, err)

		mem := NewSummaryMemory(store, summariser, llm_dto.MemoryConfig{BufferSize: 10})
		loadedState, err := mem.Load(ctx, "conv-1")

		require.NoError(t, err)
		require.NotNil(t, loadedState)
		assert.Equal(t, "conv-1", loadedState.ID)
	})
}

func TestSummaryMemory_AddMessage(t *testing.T) {
	ctx := context.Background()

	t.Run("creates new conversation when not found", func(t *testing.T) {
		store := NewMockMemoryStore()
		summariser := NewMockSummariser()
		mem := NewSummaryMemory(store, summariser, llm_dto.MemoryConfig{BufferSize: 10})

		err := mem.AddMessage(ctx, "new-conv", llm_dto.NewUserMessage("Hello"))

		require.NoError(t, err)
		state, err := store.Load(ctx, "new-conv")
		require.NoError(t, err)
		assert.Len(t, state.Messages, 1)
	})

	t.Run("triggers summarisation when buffer exceeds threshold", func(t *testing.T) {
		store := NewMockMemoryStore()
		summariser := NewMockSummariser()
		summariser.CompleteFunc = func(ctx context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
			return &llm_dto.CompletionResponse{
				Choices: []llm_dto.Choice{
					{
						Message: llm_dto.Message{
							Role:    llm_dto.RoleAssistant,
							Content: "Test summary",
						},
					},
				},
			}, nil
		}

		mem := NewSummaryMemory(store, summariser, llm_dto.MemoryConfig{BufferSize: 3})

		for i := range 7 {
			content := string(rune('A' + i))
			err := mem.AddMessage(ctx, "conv-1", llm_dto.NewUserMessage(content))
			require.NoError(t, err)
		}

		assert.Greater(t, atomic.LoadInt64(&summariser.CompleteCallCount), int64(0))

		state, err := store.Load(ctx, "conv-1")
		require.NoError(t, err)
		assert.True(t, state.HasSummary())
	})

	t.Run("continues without summary on summarisation error", func(t *testing.T) {
		store := NewMockMemoryStore()
		summariser := NewMockSummariser()
		summariser.CompleteFunc = func(ctx context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
			return nil, errors.New("summarisation failed")
		}

		mem := NewSummaryMemory(store, summariser, llm_dto.MemoryConfig{BufferSize: 3})

		for i := range 7 {
			content := string(rune('A' + i))
			err := mem.AddMessage(ctx, "conv-1", llm_dto.NewUserMessage(content))

			require.NoError(t, err)
		}

		state, err := store.Load(ctx, "conv-1")
		require.NoError(t, err)
		assert.NotNil(t, state)
	})
}

func TestSummaryMemory_GetMessages(t *testing.T) {
	ctx := context.Background()

	t.Run("returns empty slice for non-existent conversation", func(t *testing.T) {
		store := NewMockMemoryStore()
		summariser := NewMockSummariser()
		mem := NewSummaryMemory(store, summariser, llm_dto.MemoryConfig{BufferSize: 10})

		messages, err := mem.GetMessages(ctx, "non-existent")

		require.NoError(t, err)
		assert.Empty(t, messages)
	})

	t.Run("prepends summary as system message when exists", func(t *testing.T) {
		store := NewMockMemoryStore()
		summariser := NewMockSummariser()
		state := llm_dto.NewConversationState("conv-1")
		state.Summary = new("Previous discussion summary")
		state.AddMessage(llm_dto.NewUserMessage("Hello"))
		err := store.Save(ctx, state)
		require.NoError(t, err)

		mem := NewSummaryMemory(store, summariser, llm_dto.MemoryConfig{BufferSize: 10})
		messages, err := mem.GetMessages(ctx, "conv-1")

		require.NoError(t, err)
		require.Len(t, messages, 2)
		assert.Equal(t, llm_dto.RoleSystem, messages[0].Role)
		assert.Contains(t, messages[0].Content, "Previous discussion summary")
		assert.Equal(t, "Hello", messages[1].Content)
	})

	t.Run("returns messages without summary when no summary exists", func(t *testing.T) {
		store := NewMockMemoryStore()
		summariser := NewMockSummariser()
		state := llm_dto.NewConversationState("conv-1")
		state.AddMessage(llm_dto.NewUserMessage("Hello"))
		err := store.Save(ctx, state)
		require.NoError(t, err)

		mem := NewSummaryMemory(store, summariser, llm_dto.MemoryConfig{BufferSize: 10})
		messages, err := mem.GetMessages(ctx, "conv-1")

		require.NoError(t, err)
		require.Len(t, messages, 1)
		assert.Equal(t, "Hello", messages[0].Content)
	})
}

func TestSummaryMemory_Clear(t *testing.T) {
	ctx := context.Background()

	store := NewMockMemoryStore()
	summariser := NewMockSummariser()
	state := llm_dto.NewConversationState("conv-1")
	err := store.Save(ctx, state)
	require.NoError(t, err)

	mem := NewSummaryMemory(store, summariser, llm_dto.MemoryConfig{BufferSize: 10})
	err = mem.Clear(ctx, "conv-1")

	require.NoError(t, err)
	_, err = store.Load(ctx, "conv-1")
	assert.ErrorIs(t, err, ErrConversationNotFound)
}

func TestEstimateMessageTokens(t *testing.T) {
	testCases := []struct {
		name     string
		messages []llm_dto.Message
		wantMin  int
		wantMax  int
	}{
		{
			name:     "empty messages",
			messages: []llm_dto.Message{},
			wantMin:  0,
			wantMax:  0,
		},
		{
			name: "simple text message",
			messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello world"},
			},
			wantMin: 5,
			wantMax: 10,
		},
		{
			name: "message with tool calls",
			messages: []llm_dto.Message{
				{
					Role:    llm_dto.RoleAssistant,
					Content: "",
					ToolCalls: []llm_dto.ToolCall{
						{
							Function: llm_dto.FunctionCall{
								Name:      "get_weather",
								Arguments: `{"location": "London"}`,
							},
						},
					},
				},
			},
			wantMin: 10,
			wantMax: 20,
		},
		{
			name: "multiple messages",
			messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
				{Role: llm_dto.RoleAssistant, Content: "Hi there"},
				{Role: llm_dto.RoleUser, Content: "How are you?"},
			},
			wantMin: 15,
			wantMax: 25,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := estimateMessageTokens(tc.messages)

			assert.GreaterOrEqual(t, result, tc.wantMin, "token count should be >= %d", tc.wantMin)
			assert.LessOrEqual(t, result, tc.wantMax, "token count should be <= %d", tc.wantMax)
		})
	}
}

func TestEstimateSingleMessageTokens(t *testing.T) {
	testCases := []struct {
		name    string
		message llm_dto.Message
		wantMin int
		wantMax int
	}{
		{
			name:    "simple text",
			message: llm_dto.NewUserMessage("Hello"),
			wantMin: 4,
			wantMax: 10,
		},
		{
			name: "content parts with text",
			message: llm_dto.Message{
				Role: llm_dto.RoleUser,
				ContentParts: []llm_dto.ContentPart{
					{Text: new("Hello world")},
				},
			},
			wantMin: 5,
			wantMax: 15,
		},
		{
			name: "content parts with image",
			message: llm_dto.Message{
				Role: llm_dto.RoleUser,
				ContentParts: []llm_dto.ContentPart{
					{ImageURL: &llm_dto.ImageURL{URL: "https://example.com/image.jpg"}},
				},
			},
			wantMin: ImageTokenEstimateLowDetail,
			wantMax: ImageTokenEstimateLowDetail + 10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := estimateSingleMessageTokens(tc.message)

			assert.GreaterOrEqual(t, result, tc.wantMin)
			assert.LessOrEqual(t, result, tc.wantMax)
		})
	}
}

func TestErrConversationNotFound(t *testing.T) {
	assert.NotNil(t, ErrConversationNotFound)
	assert.Equal(t, "conversation not found", ErrConversationNotFound.Error())
}

func TestMemoryConstants(t *testing.T) {
	assert.Equal(t, 20, DefaultBufferSize)
	assert.Equal(t, 4000, DefaultTokenLimit)
	assert.Equal(t, 10, DefaultSummaryBufferSize)
	assert.Equal(t, 4, TokenOverheadPerMessage)
	assert.Equal(t, 4, CharactersPerToken)
	assert.Equal(t, 85, ImageTokenEstimateLowDetail)
}
