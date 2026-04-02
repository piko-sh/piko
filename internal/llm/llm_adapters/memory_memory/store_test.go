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

package memory_memory

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
)

func TestMatchPattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pattern  string
		input    string
		expected bool
	}{
		{
			name:     "empty pattern matches anything",
			pattern:  "",
			input:    "hello-world",
			expected: true,
		},
		{
			name:     "star alone matches anything",
			pattern:  "*",
			input:    "hello-world",
			expected: true,
		},
		{
			name:     "exact match",
			pattern:  "hello-world",
			input:    "hello-world",
			expected: true,
		},
		{
			name:     "prefix wildcard matches",
			pattern:  "hello*",
			input:    "hello-world",
			expected: true,
		},
		{
			name:     "suffix wildcard matches",
			pattern:  "*world",
			input:    "hello-world",
			expected: true,
		},
		{
			name:     "middle wildcard matches",
			pattern:  "hel*rld",
			input:    "hello-world",
			expected: true,
		},
		{
			name:     "multiple wildcards match",
			pattern:  "*lo*rld",
			input:    "hello-world",
			expected: true,
		},
		{
			name:     "no match returns false",
			pattern:  "goodbye",
			input:    "hello-world",
			expected: false,
		},
		{
			name:     "empty input with non-empty pattern returns false",
			pattern:  "hello",
			input:    "",
			expected: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := matchPattern(testCase.pattern, testCase.input)
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestCopyState(t *testing.T) {
	t.Parallel()

	t.Run("nil input returns nil output", func(t *testing.T) {
		t.Parallel()

		result := copyState(nil)
		assert.Nil(t, result)
	})

	t.Run("nil summary preserved as nil in copy", func(t *testing.T) {
		t.Parallel()

		original := &llm_dto.ConversationState{
			ID:      "test-id",
			Summary: nil,
		}

		result := copyState(original)
		require.NotNil(t, result)
		assert.Nil(t, result.Summary)
	})

	t.Run("non-nil summary is deep copied", func(t *testing.T) {
		t.Parallel()

		original := &llm_dto.ConversationState{
			ID:      "test-id",
			Summary: new("original summary"),
		}

		result := copyState(original)
		require.NotNil(t, result)
		require.NotNil(t, result.Summary)
		assert.Equal(t, "original summary", *result.Summary)

		*result.Summary = "modified summary"
		assert.Equal(t, "original summary", *original.Summary)
	})

	t.Run("messages are deep copied", func(t *testing.T) {
		t.Parallel()

		original := &llm_dto.ConversationState{
			ID:         "test-id",
			TokenCount: 42,
			CreatedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:  time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
			Messages: []llm_dto.Message{
				{
					Role:    llm_dto.RoleUser,
					Content: "hello",
				},
				{
					Role:    llm_dto.RoleAssistant,
					Content: "hi there",
				},
			},
		}

		result := copyState(original)
		require.NotNil(t, result)
		assert.Equal(t, original.ID, result.ID)
		assert.Equal(t, original.TokenCount, result.TokenCount)
		assert.Equal(t, original.CreatedAt, result.CreatedAt)
		assert.Equal(t, original.UpdatedAt, result.UpdatedAt)
		require.Len(t, result.Messages, 2)
		assert.Equal(t, llm_dto.RoleUser, result.Messages[0].Role)
		assert.Equal(t, "hello", result.Messages[0].Content)

		result.Messages[0].Content = "modified"
		assert.Equal(t, "hello", original.Messages[0].Content)
	})
}

func TestCopyMessage(t *testing.T) {
	t.Parallel()

	t.Run("nil Name pointer preserved as nil", func(t *testing.T) {
		t.Parallel()

		original := llm_dto.Message{
			Role:    llm_dto.RoleUser,
			Content: "hello",
			Name:    nil,
		}

		result := copyMessage(original)
		assert.Nil(t, result.Name)
		assert.Equal(t, llm_dto.RoleUser, result.Role)
		assert.Equal(t, "hello", result.Content)
	})

	t.Run("non-nil Name pointer is deep copied", func(t *testing.T) {
		t.Parallel()

		original := llm_dto.Message{
			Role:    llm_dto.RoleUser,
			Content: "hello",
			Name:    new("alice"),
		}

		result := copyMessage(original)
		require.NotNil(t, result.Name)
		assert.Equal(t, "alice", *result.Name)

		*result.Name = "bob"
		assert.Equal(t, "alice", *original.Name)
	})

	t.Run("ContentParts are copied", func(t *testing.T) {
		t.Parallel()

		original := llm_dto.Message{
			Role:    llm_dto.RoleUser,
			Content: "hello",
			ContentParts: []llm_dto.ContentPart{
				{
					Type: llm_dto.ContentPartTypeText,
					Text: new("some text"),
				},
			},
		}

		result := copyMessage(original)
		require.Len(t, result.ContentParts, 1)
		assert.Equal(t, llm_dto.ContentPartTypeText, result.ContentParts[0].Type)
		require.NotNil(t, result.ContentParts[0].Text)
		assert.Equal(t, "some text", *result.ContentParts[0].Text)

		*result.ContentParts[0].Text = "changed"
		assert.Equal(t, "some text", *original.ContentParts[0].Text)
	})

	t.Run("ToolCalls are copied", func(t *testing.T) {
		t.Parallel()

		original := llm_dto.Message{
			Role:    llm_dto.RoleAssistant,
			Content: "",
			ToolCalls: []llm_dto.ToolCall{
				{
					ID:   "call-1",
					Type: "function",
					Function: llm_dto.FunctionCall{
						Name:      "get_weather",
						Arguments: `{"city":"london"}`,
					},
				},
			},
		}

		result := copyMessage(original)
		require.Len(t, result.ToolCalls, 1)
		assert.Equal(t, "call-1", result.ToolCalls[0].ID)
		assert.Equal(t, "get_weather", result.ToolCalls[0].Function.Name)

		result.ToolCalls[0].Function.Name = "changed"
		assert.Equal(t, "get_weather", original.ToolCalls[0].Function.Name)
	})
}

func TestCopyContentPart(t *testing.T) {
	t.Parallel()

	t.Run("text part with non-nil Text pointer", func(t *testing.T) {
		t.Parallel()

		original := llm_dto.ContentPart{
			Type: llm_dto.ContentPartTypeText,
			Text: new("hello world"),
		}

		result := copyContentPart(original)
		assert.Equal(t, llm_dto.ContentPartTypeText, result.Type)
		require.NotNil(t, result.Text)
		assert.Equal(t, "hello world", *result.Text)

		*result.Text = "modified"
		assert.Equal(t, "hello world", *original.Text)
	})

	t.Run("image URL part with Detail pointer", func(t *testing.T) {
		t.Parallel()

		original := llm_dto.ContentPart{
			Type: llm_dto.ContentPartTypeImageURL,
			ImageURL: &llm_dto.ImageURL{
				URL:    "https://example.com/image.png",
				Detail: new("high"),
			},
		}

		result := copyContentPart(original)
		assert.Equal(t, llm_dto.ContentPartTypeImageURL, result.Type)
		require.NotNil(t, result.ImageURL)
		assert.Equal(t, "https://example.com/image.png", result.ImageURL.URL)
		require.NotNil(t, result.ImageURL.Detail)
		assert.Equal(t, "high", *result.ImageURL.Detail)

		*result.ImageURL.Detail = "low"
		assert.Equal(t, "high", *original.ImageURL.Detail)
	})

	t.Run("image data part", func(t *testing.T) {
		t.Parallel()

		original := llm_dto.ContentPart{
			Type: llm_dto.ContentPartTypeImageData,
			ImageData: &llm_dto.ImageData{
				MIMEType: "image/png",
				Data:     "iVBORw0KGgo=",
			},
		}

		result := copyContentPart(original)
		assert.Equal(t, llm_dto.ContentPartTypeImageData, result.Type)
		require.NotNil(t, result.ImageData)
		assert.Equal(t, "image/png", result.ImageData.MIMEType)
		assert.Equal(t, "iVBORw0KGgo=", result.ImageData.Data)
	})
}

func TestCopyToolCall(t *testing.T) {
	t.Parallel()

	original := llm_dto.ToolCall{
		ID:   "call-abc",
		Type: "function",
		Function: llm_dto.FunctionCall{
			Name:      "search",
			Arguments: `{"query":"piko"}`,
		},
	}

	result := copyToolCall(original)
	assert.Equal(t, "call-abc", result.ID)
	assert.Equal(t, "function", result.Type)
	assert.Equal(t, "search", result.Function.Name)
	assert.Equal(t, `{"query":"piko"}`, result.Function.Arguments)

	result.Function.Name = "changed"
	assert.Equal(t, "search", original.Function.Name)
}

func TestStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store := New()
	ctx := context.Background()

	state := &llm_dto.ConversationState{
		ID:         "conv-1",
		TokenCount: 100,
		CreatedAt:  time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 3, 1, 11, 0, 0, 0, time.UTC),
		Summary:    new("a conversation about testing"),
		Messages: []llm_dto.Message{
			{
				Role:    llm_dto.RoleUser,
				Content: "hello",
			},
			{
				Role:    llm_dto.RoleAssistant,
				Content: "hi there",
			},
		},
	}

	err := store.Save(ctx, state)
	require.NoError(t, err)

	loaded, err := store.Load(ctx, "conv-1")
	require.NoError(t, err)
	assert.Equal(t, state.ID, loaded.ID)
	assert.Equal(t, state.TokenCount, loaded.TokenCount)
	assert.Equal(t, state.CreatedAt, loaded.CreatedAt)
	assert.Equal(t, state.UpdatedAt, loaded.UpdatedAt)
	require.NotNil(t, loaded.Summary)
	assert.Equal(t, *state.Summary, *loaded.Summary)
	require.Len(t, loaded.Messages, 2)
	assert.Equal(t, "hello", loaded.Messages[0].Content)
	assert.Equal(t, "hi there", loaded.Messages[1].Content)

	_, err = store.Load(ctx, "non-existent")
	assert.ErrorIs(t, err, llm_domain.ErrConversationNotFound)
}

func TestStore_Load_DeepCopyIsolation(t *testing.T) {
	t.Parallel()

	store := New()
	ctx := context.Background()

	state := &llm_dto.ConversationState{
		ID: "conv-isolation",
		Messages: []llm_dto.Message{
			{
				Role:    llm_dto.RoleUser,
				Content: "original",
			},
		},
	}

	err := store.Save(ctx, state)
	require.NoError(t, err)

	firstLoad, err := store.Load(ctx, "conv-isolation")
	require.NoError(t, err)

	firstLoad.Messages[0].Content = "tampered"
	firstLoad.Messages = append(firstLoad.Messages, llm_dto.Message{
		Role:    llm_dto.RoleAssistant,
		Content: "injected",
	})

	secondLoad, err := store.Load(ctx, "conv-isolation")
	require.NoError(t, err)
	require.Len(t, secondLoad.Messages, 1)
	assert.Equal(t, "original", secondLoad.Messages[0].Content)
}

func TestStore_Save_DeepCopyIsolation(t *testing.T) {
	t.Parallel()

	store := New()
	ctx := context.Background()

	state := &llm_dto.ConversationState{
		ID: "conv-save-isolation",
		Messages: []llm_dto.Message{
			{
				Role:    llm_dto.RoleUser,
				Content: "before mutation",
			},
		},
	}

	err := store.Save(ctx, state)
	require.NoError(t, err)

	state.Messages[0].Content = "after mutation"
	state.Messages = append(state.Messages, llm_dto.Message{
		Role:    llm_dto.RoleAssistant,
		Content: "appended after save",
	})

	loaded, err := store.Load(ctx, "conv-save-isolation")
	require.NoError(t, err)
	require.Len(t, loaded.Messages, 1)
	assert.Equal(t, "before mutation", loaded.Messages[0].Content)
}

func TestStore_Delete(t *testing.T) {
	t.Parallel()

	store := New()
	ctx := context.Background()

	state := &llm_dto.ConversationState{
		ID: "conv-delete",
		Messages: []llm_dto.Message{
			{
				Role:    llm_dto.RoleUser,
				Content: "to be deleted",
			},
		},
	}

	err := store.Save(ctx, state)
	require.NoError(t, err)

	err = store.Delete(ctx, "conv-delete")
	require.NoError(t, err)

	_, err = store.Load(ctx, "conv-delete")
	assert.ErrorIs(t, err, llm_domain.ErrConversationNotFound)

	err = store.Delete(ctx, "non-existent")
	assert.NoError(t, err)
}

func TestStore_List(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("empty store returns empty slice", func(t *testing.T) {
		t.Parallel()

		emptyStore := New()
		ids, err := emptyStore.List(ctx, "*")
		require.NoError(t, err)
		assert.Empty(t, ids)
	})

	t.Run("star returns all conversations", func(t *testing.T) {
		t.Parallel()

		listStore := New()

		require.NoError(t, listStore.Save(ctx, &llm_dto.ConversationState{ID: "chat-alpha"}))
		require.NoError(t, listStore.Save(ctx, &llm_dto.ConversationState{ID: "chat-beta"}))
		require.NoError(t, listStore.Save(ctx, &llm_dto.ConversationState{ID: "session-gamma"}))

		ids, err := listStore.List(ctx, "*")
		require.NoError(t, err)
		assert.Len(t, ids, 3)
		assert.ElementsMatch(t, []string{"chat-alpha", "chat-beta", "session-gamma"}, ids)
	})

	t.Run("prefix pattern matches correctly", func(t *testing.T) {
		t.Parallel()

		prefixStore := New()

		require.NoError(t, prefixStore.Save(ctx, &llm_dto.ConversationState{ID: "chat-alpha"}))
		require.NoError(t, prefixStore.Save(ctx, &llm_dto.ConversationState{ID: "chat-beta"}))
		require.NoError(t, prefixStore.Save(ctx, &llm_dto.ConversationState{ID: "session-gamma"}))

		ids, err := prefixStore.List(ctx, "chat*")
		require.NoError(t, err)
		assert.Len(t, ids, 2)
		assert.ElementsMatch(t, []string{"chat-alpha", "chat-beta"}, ids)
	})
}

func TestStore_Size(t *testing.T) {
	t.Parallel()

	store := New()
	ctx := context.Background()

	assert.Equal(t, 0, store.Size())

	require.NoError(t, store.Save(ctx, &llm_dto.ConversationState{ID: "conv-a"}))
	require.NoError(t, store.Save(ctx, &llm_dto.ConversationState{ID: "conv-b"}))
	assert.Equal(t, 2, store.Size())

	require.NoError(t, store.Delete(ctx, "conv-a"))
	assert.Equal(t, 1, store.Size())
}

func TestStore_Clear(t *testing.T) {
	t.Parallel()

	store := New()
	ctx := context.Background()

	require.NoError(t, store.Save(ctx, &llm_dto.ConversationState{ID: "conv-1"}))
	require.NoError(t, store.Save(ctx, &llm_dto.ConversationState{ID: "conv-2"}))
	require.NoError(t, store.Save(ctx, &llm_dto.ConversationState{ID: "conv-3"}))
	assert.Equal(t, 3, store.Size())

	store.Clear()
	assert.Equal(t, 0, store.Size())

	_, err := store.Load(ctx, "conv-1")
	assert.ErrorIs(t, err, llm_domain.ErrConversationNotFound)
}

func TestStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	store := New()
	ctx := context.Background()

	const goroutineCount = 10

	var waitGroup sync.WaitGroup
	waitGroup.Add(goroutineCount)

	for index := range goroutineCount {
		go func(workerIndex int) {
			defer waitGroup.Done()

			conversationID := fmt.Sprintf("conv-%d", workerIndex)
			sharedID := "conv-shared"

			state := &llm_dto.ConversationState{
				ID: conversationID,
				Messages: []llm_dto.Message{
					{
						Role:    llm_dto.RoleUser,
						Content: fmt.Sprintf("message from worker %d", workerIndex),
					},
				},
			}

			_ = store.Save(ctx, state)

			sharedState := &llm_dto.ConversationState{
				ID: sharedID,
				Messages: []llm_dto.Message{
					{
						Role:    llm_dto.RoleUser,
						Content: fmt.Sprintf("shared from worker %d", workerIndex),
					},
				},
			}
			_ = store.Save(ctx, sharedState)

			_, _ = store.Load(ctx, conversationID)
			_, _ = store.Load(ctx, sharedID)
			_, _ = store.List(ctx, "conv*")
			_ = store.Size()

			if workerIndex%2 == 0 {
				_ = store.Delete(ctx, conversationID)
			}
		}(index)
	}

	waitGroup.Wait()

	_, err := store.Load(ctx, "conv-shared")
	assert.NoError(t, err)
}
