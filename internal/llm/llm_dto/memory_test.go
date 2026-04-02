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

package llm_dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultBufferMemoryConfig(t *testing.T) {
	t.Parallel()

	config := DefaultBufferMemoryConfig()
	assert.Equal(t, MemoryTypeBuffer, config.Type)
	assert.Equal(t, DefaultBufferSize, config.BufferSize)
	assert.Equal(t, 20, config.BufferSize)
}

func TestDefaultWindowMemoryConfig(t *testing.T) {
	t.Parallel()

	config := DefaultWindowMemoryConfig()
	assert.Equal(t, MemoryTypeWindow, config.Type)
	assert.Equal(t, DefaultWindowTokenLimit, config.TokenLimit)
	assert.Equal(t, 4000, config.TokenLimit)
}

func TestDefaultSummaryMemoryConfig(t *testing.T) {
	t.Parallel()

	config := DefaultSummaryMemoryConfig("gpt-4o")
	assert.Equal(t, MemoryTypeSummary, config.Type)
	assert.Equal(t, "gpt-4o", config.SummaryModel)
	assert.Equal(t, DefaultSummaryBufferSize, config.BufferSize)
	assert.Equal(t, 10, config.BufferSize)
	assert.NotEmpty(t, config.SummaryPrompt)
}

func TestNewConversationState(t *testing.T) {
	t.Parallel()

	state := NewConversationState("conv-123")

	assert.Equal(t, "conv-123", state.ID)
	assert.Empty(t, state.Messages)
	assert.NotNil(t, state.Messages)
	assert.False(t, state.CreatedAt.IsZero())
	assert.False(t, state.UpdatedAt.IsZero())
	assert.Nil(t, state.Summary)
	assert.Equal(t, 0, state.TokenCount)
}

func TestConversationState_AddMessage(t *testing.T) {
	t.Parallel()

	state := NewConversationState("conv-1")
	require.Equal(t, 0, state.MessageCount())

	state.AddMessage(NewUserMessage("hello"))
	assert.Equal(t, 1, state.MessageCount())
	assert.Equal(t, RoleUser, state.Messages[0].Role)
	assert.Equal(t, "hello", state.Messages[0].Content)

	state.AddMessage(NewAssistantMessage("hi there"))
	assert.Equal(t, 2, state.MessageCount())
}

func TestConversationState_HasSummary(t *testing.T) {
	t.Parallel()

	t.Run("no summary", func(t *testing.T) {
		t.Parallel()

		state := NewConversationState("conv-1")
		assert.False(t, state.HasSummary())
	})

	t.Run("empty summary", func(t *testing.T) {
		t.Parallel()

		state := NewConversationState("conv-1")
		state.Summary = new("")
		assert.False(t, state.HasSummary())
	})

	t.Run("with summary", func(t *testing.T) {
		t.Parallel()

		state := NewConversationState("conv-1")
		state.Summary = new("User discussed project setup")
		assert.True(t, state.HasSummary())
	})
}

func TestConversationState_Clear(t *testing.T) {
	t.Parallel()

	state := NewConversationState("conv-1")
	state.AddMessage(NewUserMessage("hello"))
	state.AddMessage(NewAssistantMessage("hi"))
	state.Summary = new("previous conversation")
	state.TokenCount = 100

	state.Clear()

	assert.Empty(t, state.Messages)
	assert.NotNil(t, state.Messages)
	assert.Nil(t, state.Summary)
	assert.Equal(t, 0, state.TokenCount)
	assert.Equal(t, 0, state.MessageCount())
}
