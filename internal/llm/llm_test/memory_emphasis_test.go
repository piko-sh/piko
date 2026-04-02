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
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_adapters/memory_memory"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
)

func TestMemory_Buffer_ExactBoundary(t *testing.T) {
	skipIfShort(t)

	store := memory_memory.New()
	mem := llm_domain.NewBufferMemory(store, llm_domain.WithBufferSize(5))
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		err := mem.AddMessage(ctx, "conv-boundary", llm_dto.NewUserMessage(fmt.Sprintf("message-%d", i)))
		require.NoError(t, err)
	}

	messages, err := mem.GetMessages(ctx, "conv-boundary")
	require.NoError(t, err)
	require.Len(t, messages, 5)
	assert.Equal(t, "message-1", messages[0].Content)
	assert.Equal(t, "message-5", messages[4].Content)

	err = mem.AddMessage(ctx, "conv-boundary", llm_dto.NewUserMessage("message-6"))
	require.NoError(t, err)

	messages, err = mem.GetMessages(ctx, "conv-boundary")
	require.NoError(t, err)
	require.Len(t, messages, 5)
	assert.Equal(t, "message-2", messages[0].Content, "message-1 should have been trimmed")
	assert.Equal(t, "message-6", messages[4].Content)
}

func TestMemory_Buffer_LargeConversation(t *testing.T) {
	skipIfShort(t)

	store := memory_memory.New()
	const bufSize = 10
	mem := llm_domain.NewBufferMemory(store, llm_domain.WithBufferSize(bufSize))
	ctx := context.Background()

	for i := 1; i <= 100; i++ {
		err := mem.AddMessage(ctx, "conv-large", llm_dto.NewUserMessage(fmt.Sprintf("message-%d", i)))
		require.NoError(t, err)
	}

	messages, err := mem.GetMessages(ctx, "conv-large")
	require.NoError(t, err)
	require.Len(t, messages, bufSize, "only last %d messages should remain", bufSize)
	assert.Equal(t, "message-91", messages[0].Content, "oldest retained should be message-91")
	assert.Equal(t, "message-100", messages[bufSize-1].Content, "newest should be message-100")
}

func TestMemory_Buffer_ZeroSize_UsesDefault(t *testing.T) {
	skipIfShort(t)

	store := memory_memory.New()
	mem := llm_domain.NewBufferMemory(store)
	ctx := context.Background()

	for i := 1; i <= 25; i++ {
		err := mem.AddMessage(ctx, "conv-default", llm_dto.NewUserMessage(fmt.Sprintf("message-%d", i)))
		require.NoError(t, err)
	}

	messages, err := mem.GetMessages(ctx, "conv-default")
	require.NoError(t, err)
	assert.Len(t, messages, llm_domain.DefaultBufferSize,
		"zero size should default to %d", llm_domain.DefaultBufferSize)
	assert.Equal(t, "message-6", messages[0].Content)
}

func TestMemory_Window_TokenEstimation(t *testing.T) {
	skipIfShort(t)

	store := memory_memory.New()

	mem := llm_domain.NewWindowMemory(store, llm_domain.WithTokenLimit(10000))
	ctx := context.Background()

	err := mem.AddMessage(ctx, "conv-tokens", llm_dto.NewUserMessage(strings.Repeat("a", 40)))
	require.NoError(t, err)

	state, err := store.Load(ctx, "conv-tokens")
	require.NoError(t, err)

	assert.Equal(t, 14, state.TokenCount, "token estimate should match heuristic (overhead + len/4)")
}

func TestMemory_Window_LargeMessageTrimming(t *testing.T) {
	skipIfShort(t)

	store := memory_memory.New()
	const tokenLimit = 100
	mem := llm_domain.NewWindowMemory(store, llm_domain.WithTokenLimit(tokenLimit))
	ctx := context.Background()

	for range 3 {
		err := mem.AddMessage(ctx, "conv-large-message", llm_dto.NewUserMessage("short"))
		require.NoError(t, err)
	}

	largeContent := strings.Repeat("x", 360)
	err := mem.AddMessage(ctx, "conv-large-message", llm_dto.NewUserMessage(largeContent))
	require.NoError(t, err)

	state, err := store.Load(ctx, "conv-large-message")
	require.NoError(t, err)

	assert.LessOrEqual(t, state.TokenCount, tokenLimit)

	messages, err := mem.GetMessages(ctx, "conv-large-message")
	require.NoError(t, err)
	require.NotEmpty(t, messages)
	assert.Equal(t, largeContent, messages[len(messages)-1].Content)
}

func TestMemory_Window_ImageTokenEstimate(t *testing.T) {
	skipIfShort(t)

	store := memory_memory.New()
	mem := llm_domain.NewWindowMemory(store, llm_domain.WithTokenLimit(10000))
	ctx := context.Background()

	message := llm_dto.NewUserMessageWithImageURL("Describe this", "https://example.com/img.png")
	err := mem.AddMessage(ctx, "conv-image", message)
	require.NoError(t, err)

	state, err := store.Load(ctx, "conv-image")
	require.NoError(t, err)

	assert.GreaterOrEqual(t, state.TokenCount, llm_domain.ImageTokenEstimateLowDetail,
		"image message should include at least %d tokens for image", llm_domain.ImageTokenEstimateLowDetail)
}

func TestMemory_Window_ToolCallTokenEstimate(t *testing.T) {
	skipIfShort(t)

	store := memory_memory.New()
	mem := llm_domain.NewWindowMemory(store, llm_domain.WithTokenLimit(10000))
	ctx := context.Background()

	message := llm_dto.Message{
		Role:    llm_dto.RoleAssistant,
		Content: "Let me check.",
		ToolCalls: []llm_dto.ToolCall{
			{
				ID:   "tc-1",
				Type: "function",
				Function: llm_dto.FunctionCall{
					Name:      "get_weather",
					Arguments: `{"city":"London","units":"celsius"}`,
				},
			},
		},
	}
	err := mem.AddMessage(ctx, "conv-toolcall", message)
	require.NoError(t, err)

	state, err := store.Load(ctx, "conv-toolcall")
	require.NoError(t, err)

	baseTokens := llm_domain.TokenOverheadPerMessage + len("Let me check.")/llm_domain.CharactersPerToken
	assert.Greater(t, state.TokenCount, baseTokens,
		"token count should exceed base content tokens due to tool call")
}

func TestMemory_Summary_TriggerThreshold(t *testing.T) {
	skipIfShort(t)

	store := memory_memory.New()
	summariser := newTestSummariser("Conversation summary goes here.")
	mem := llm_domain.NewSummaryMemory(store, summariser, llm_dto.MemoryConfig{
		BufferSize: 5,
	})
	ctx := context.Background()

	for i := 1; i <= 10; i++ {
		err := mem.AddMessage(ctx, "conv-threshold", llm_dto.NewUserMessage(fmt.Sprintf("message-%d", i)))
		require.NoError(t, err)
	}
	assert.Equal(t, 0, summariser.callCount(), "10 messages should not trigger summarisation")

	err := mem.AddMessage(ctx, "conv-threshold", llm_dto.NewUserMessage("message-11"))
	require.NoError(t, err)
	assert.Equal(t, 1, summariser.callCount(), "11th message should trigger summarisation")
}

func TestMemory_Summary_PrependsSummaryMessage(t *testing.T) {
	skipIfShort(t)

	store := memory_memory.New()
	summariser := newTestSummariser("Users discussed weather topics.")
	mem := llm_domain.NewSummaryMemory(store, summariser, llm_dto.MemoryConfig{
		BufferSize: 3,
	})
	ctx := context.Background()

	for i := 1; i <= 7; i++ {
		err := mem.AddMessage(ctx, "conv-prepend", llm_dto.NewUserMessage(fmt.Sprintf("message-%d", i)))
		require.NoError(t, err)
	}

	require.GreaterOrEqual(t, summariser.callCount(), 1)

	messages, err := mem.GetMessages(ctx, "conv-prepend")
	require.NoError(t, err)

	require.NotEmpty(t, messages)
	assert.Equal(t, llm_dto.RoleSystem, messages[0].Role)
	assert.Contains(t, messages[0].Content, "Previous conversation summary:")
	assert.Contains(t, messages[0].Content, "Users discussed weather topics.")
}

func TestMemory_Summary_SummariserError_NonFatal(t *testing.T) {
	skipIfShort(t)

	store := memory_memory.New()
	summariser := newTestSummariser("")
	summariser.err = errors.New("summariser unavailable")

	mem := llm_domain.NewSummaryMemory(store, summariser, llm_dto.MemoryConfig{
		BufferSize: 3,
	})
	ctx := context.Background()

	for i := 1; i <= 7; i++ {
		err := mem.AddMessage(ctx, "conv-err", llm_dto.NewUserMessage(fmt.Sprintf("message-%d", i)))
		require.NoError(t, err, "AddMessage should succeed even when summariser fails")
	}

	assert.GreaterOrEqual(t, summariser.callCount(), 1)

	messages, err := mem.GetMessages(ctx, "conv-err")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(messages), 7, "all messages should be retained when summariser fails")
}

func TestMemory_Summary_MultipleSummarisations(t *testing.T) {
	skipIfShort(t)

	store := memory_memory.New()
	summariser := newTestSummariser("Cumulative summary.")
	mem := llm_domain.NewSummaryMemory(store, summariser, llm_dto.MemoryConfig{
		BufferSize: 3,
	})
	ctx := context.Background()

	for i := 1; i <= 7; i++ {
		err := mem.AddMessage(ctx, "conv-multi-sum", llm_dto.NewUserMessage(fmt.Sprintf("batch1-message-%d", i)))
		require.NoError(t, err)
	}
	firstCallCount := summariser.callCount()
	require.GreaterOrEqual(t, firstCallCount, 1, "first summarisation should occur")

	for i := 1; i <= 10; i++ {
		err := mem.AddMessage(ctx, "conv-multi-sum", llm_dto.NewUserMessage(fmt.Sprintf("batch2-message-%d", i)))
		require.NoError(t, err)
	}
	assert.Greater(t, summariser.callCount(), firstCallCount, "additional summarisation should have occurred")

	state, err := store.Load(ctx, "conv-multi-sum")
	require.NoError(t, err)
	require.True(t, state.HasSummary())
}

func TestMemory_StoreIsolation(t *testing.T) {
	skipIfShort(t)

	store := memory_memory.New()
	mem := llm_domain.NewBufferMemory(store, llm_domain.WithBufferSize(20))
	ctx := context.Background()

	err := mem.AddMessage(ctx, "conv-isolation", llm_dto.NewUserMessage("original"))
	require.NoError(t, err)

	messages, err := mem.GetMessages(ctx, "conv-isolation")
	require.NoError(t, err)
	require.Len(t, messages, 1)
	messages[0].Content = "MUTATED"

	messages2, err := mem.GetMessages(ctx, "conv-isolation")
	require.NoError(t, err)
	require.Len(t, messages2, 1)
	assert.Equal(t, "original", messages2[0].Content, "store should not be affected by external mutation")
}

func TestMemory_ConcurrentAccess(t *testing.T) {
	skipIfShort(t)

	store := memory_memory.New()
	mem := llm_domain.NewBufferMemory(store, llm_domain.WithBufferSize(50))
	ctx := context.Background()

	const goroutines = 10
	const msgsPerGoroutine = 20
	var wg sync.WaitGroup

	for g := range goroutines {
		wg.Go(func() {
			for i := range msgsPerGoroutine {
				content := fmt.Sprintf("g%d-message-%d", g, i)
				_ = mem.AddMessage(ctx, "conv-concurrent", llm_dto.NewUserMessage(content))
			}
		})
	}

	for range goroutines {
		wg.Go(func() {
			for range msgsPerGoroutine {
				_, _ = mem.GetMessages(ctx, "conv-concurrent")
			}
		})
	}

	wg.Wait()

	messages, err := mem.GetMessages(ctx, "conv-concurrent")
	require.NoError(t, err)
	assert.NotEmpty(t, messages, "should have some messages after concurrent writes")

	assert.LessOrEqual(t, len(messages), 50)
}

func TestMemory_ConversationState_Lifecycle(t *testing.T) {
	skipIfShort(t)

	store := memory_memory.New()
	mem := llm_domain.NewBufferMemory(store, llm_domain.WithBufferSize(20))
	ctx := context.Background()

	err := mem.AddMessage(ctx, "conv-lifecycle", llm_dto.NewUserMessage("hello"))
	require.NoError(t, err)
	err = mem.AddMessage(ctx, "conv-lifecycle", llm_dto.NewAssistantMessage("hi there"))
	require.NoError(t, err)

	messages, err := mem.GetMessages(ctx, "conv-lifecycle")
	require.NoError(t, err)
	require.Len(t, messages, 2)

	err = mem.Clear(ctx, "conv-lifecycle")
	require.NoError(t, err)

	messages, err = mem.GetMessages(ctx, "conv-lifecycle")
	require.NoError(t, err)
	assert.Empty(t, messages, "should be empty after clear")

	err = mem.AddMessage(ctx, "conv-lifecycle", llm_dto.NewUserMessage("fresh start"))
	require.NoError(t, err)

	messages, err = mem.GetMessages(ctx, "conv-lifecycle")
	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, "fresh start", messages[0].Content)
}
