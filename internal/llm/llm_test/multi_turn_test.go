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
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
)

func TestMultiTurn_BufferMemory_TwoTurns(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()

	h.mockProvider.SetResponse(makeResponse("Hello!", 10, 5))
	resp1, err := h.service.NewCompletion().
		Model("test-model").
		System("You are helpful.").
		User("Hi").
		BufferMemory(h.memoryStore, "conv-two-turn", 20).
		Do(ctx)

	require.NoError(t, err)
	assert.Equal(t, "Hello!", resp1.Choices[0].Message.Content)

	calls := h.mockProvider.GetCompleteCalls()
	require.Len(t, calls, 1)
	assert.Len(t, calls[0].Messages, 2, "turn 1: system + user")

	state, err := h.memoryStore.Load(ctx, "conv-two-turn")
	require.NoError(t, err)
	require.Equal(t, 3, state.MessageCount())

	h.mockProvider.SetResponse(makeResponse("I'm fine!", 15, 5))
	resp2, err := h.service.NewCompletion().
		Model("test-model").
		User("How are you?").
		BufferMemory(h.memoryStore, "conv-two-turn", 20).
		Do(ctx)

	require.NoError(t, err)
	assert.Equal(t, "I'm fine!", resp2.Choices[0].Message.Content)

	calls = h.mockProvider.GetCompleteCalls()
	require.Len(t, calls, 2)
	require.Len(t, calls[1].Messages, 4, "turn 2: 3 history + 1 new user")
	assert.Equal(t, llm_dto.RoleSystem, calls[1].Messages[0].Role)
	assert.Equal(t, llm_dto.RoleUser, calls[1].Messages[1].Role)
	assert.Equal(t, llm_dto.RoleAssistant, calls[1].Messages[2].Role)
	assert.Equal(t, llm_dto.RoleUser, calls[1].Messages[3].Role)
	assert.Equal(t, "How are you?", calls[1].Messages[3].Content)

	state, err = h.memoryStore.Load(ctx, "conv-two-turn")
	require.NoError(t, err)
	assert.Equal(t, 5, state.MessageCount())
}

func TestMultiTurn_BufferMemory_HistoryTrimming(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()
	const bufferSize = 4

	for i := 1; i <= 4; i++ {
		h.mockProvider.SetResponse(makeResponse(fmt.Sprintf("Reply %d", i), 10, 5))
		_, err := h.service.NewCompletion().
			Model("test-model").
			User(fmt.Sprintf("Turn %d", i)).
			BufferMemory(h.memoryStore, "conv-trim", bufferSize).
			Do(ctx)
		require.NoError(t, err, "turn %d failed", i)
	}

	state, err := h.memoryStore.Load(ctx, "conv-trim")
	require.NoError(t, err)
	require.Equal(t, bufferSize, state.MessageCount())

	assert.Equal(t, "Turn 3", state.Messages[0].Content)
	assert.Equal(t, "Reply 3", state.Messages[1].Content)
	assert.Equal(t, "Turn 4", state.Messages[2].Content)
	assert.Equal(t, "Reply 4", state.Messages[3].Content)

	calls := h.mockProvider.GetCompleteCalls()
	lastCall := calls[len(calls)-1]
	require.Len(t, lastCall.Messages, 5)
	assert.Equal(t, "Turn 2", lastCall.Messages[0].Content, "oldest history message")
	assert.Equal(t, "Turn 4", lastCall.Messages[4].Content, "new user message")
}

func TestMultiTurn_WindowMemory_TokenTrimming(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()

	const tokenLimit = 50
	windowMem := llm_domain.NewWindowMemory(h.memoryStore, llm_domain.WithTokenLimit(tokenLimit))

	for i := 1; i <= 4; i++ {
		h.mockProvider.SetResponse(makeResponse(fmt.Sprintf("R%d", i), 10, 5))
		_, err := h.service.NewCompletion().
			Model("test-model").
			User(strings.Repeat("x", 32)).
			Memory(windowMem, "conv-window").
			Do(ctx)
		require.NoError(t, err, "turn %d failed", i)
	}

	state, err := h.memoryStore.Load(ctx, "conv-window")
	require.NoError(t, err)
	assert.LessOrEqual(t, state.TokenCount, tokenLimit,
		"token count %d should be within limit %d", state.TokenCount, tokenLimit)

	assert.Less(t, state.MessageCount(), 8, "expected some messages to be trimmed from 4 turns")
	assert.Greater(t, state.MessageCount(), 0, "should still have some messages")
}

func TestMultiTurn_SummaryMemory_Summarisation(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()

	summariser := newTestSummariser("The user greeted the assistant multiple times.")
	summaryMem := llm_domain.NewSummaryMemory(h.memoryStore, summariser, llm_dto.MemoryConfig{
		Type:       llm_dto.MemoryTypeSummary,
		BufferSize: 5,
	})

	for i := 1; i <= 6; i++ {
		h.mockProvider.SetResponse(makeResponse(fmt.Sprintf("Reply %d", i), 10, 5))
		_, err := h.service.NewCompletion().
			Model("test-model").
			User(fmt.Sprintf("Turn %d", i)).
			Memory(summaryMem, "conv-summary").
			Do(ctx)
		require.NoError(t, err, "turn %d failed", i)
	}

	assert.GreaterOrEqual(t, summariser.callCount(), 1, "summariser should have been invoked")

	state, err := h.memoryStore.Load(ctx, "conv-summary")
	require.NoError(t, err)
	require.True(t, state.HasSummary(), "conversation should have a summary after 6 turns")

	messages, err := summaryMem.GetMessages(ctx, "conv-summary")
	require.NoError(t, err)
	require.NotEmpty(t, messages)
	assert.Equal(t, llm_dto.RoleSystem, messages[0].Role)
	assert.Contains(t, messages[0].Content, "Previous conversation summary:")
	assert.Contains(t, messages[0].Content, "The user greeted the assistant multiple times.")

	h.mockProvider.SetResponse(makeResponse("Reply 7", 10, 5))
	_, err = h.service.NewCompletion().
		Model("test-model").
		User("Turn 7").
		Memory(summaryMem, "conv-summary").
		Do(ctx)
	require.NoError(t, err)

	calls := h.mockProvider.GetCompleteCalls()
	lastCall := calls[len(calls)-1]

	assert.Equal(t, llm_dto.RoleSystem, lastCall.Messages[0].Role)
	assert.Contains(t, lastCall.Messages[0].Content, "Previous conversation summary:")
}

func TestMultiTurn_IsolatedConversations(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()

	h.mockProvider.SetResponse(makeResponse("Alpha answer", 10, 5))
	_, err := h.service.NewCompletion().
		Model("test-model").
		User("Alpha question").
		BufferMemory(h.memoryStore, "conv-alpha", 20).
		Do(ctx)
	require.NoError(t, err)

	h.mockProvider.SetResponse(makeResponse("Beta answer", 10, 5))
	_, err = h.service.NewCompletion().
		Model("test-model").
		User("Beta question").
		BufferMemory(h.memoryStore, "conv-beta", 20).
		Do(ctx)
	require.NoError(t, err)

	h.mockProvider.SetResponse(makeResponse("Alpha follow-up answer", 10, 5))
	_, err = h.service.NewCompletion().
		Model("test-model").
		User("Alpha follow-up").
		BufferMemory(h.memoryStore, "conv-alpha", 20).
		Do(ctx)
	require.NoError(t, err)

	calls := h.mockProvider.GetCompleteCalls()

	alphaCall := calls[2]

	require.Len(t, alphaCall.Messages, 3)
	assert.Equal(t, "Alpha question", alphaCall.Messages[0].Content)
	assert.Equal(t, "Alpha answer", alphaCall.Messages[1].Content)
	assert.Equal(t, "Alpha follow-up", alphaCall.Messages[2].Content)

	for _, message := range alphaCall.Messages {
		assert.NotContains(t, message.Content, "Beta")
	}

	h.mockProvider.SetResponse(makeResponse("Beta follow-up answer", 10, 5))
	_, err = h.service.NewCompletion().
		Model("test-model").
		User("Beta follow-up").
		BufferMemory(h.memoryStore, "conv-beta", 20).
		Do(ctx)
	require.NoError(t, err)

	calls = h.mockProvider.GetCompleteCalls()
	betaCall := calls[3]
	require.Len(t, betaCall.Messages, 3)
	assert.Equal(t, "Beta question", betaCall.Messages[0].Content)
	assert.Equal(t, "Beta answer", betaCall.Messages[1].Content)
	assert.Equal(t, "Beta follow-up", betaCall.Messages[2].Content)

	for _, message := range betaCall.Messages {
		assert.NotContains(t, message.Content, "Alpha")
	}
}

func TestMultiTurn_AssistantToolCalls_Preserved(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()

	respWithTools := &llm_dto.CompletionResponse{
		ID:    "response-tools",
		Model: "test-model",
		Choices: []llm_dto.Choice{
			{
				Index: 0,
				Message: llm_dto.Message{
					Role:    llm_dto.RoleAssistant,
					Content: "Let me check that.",
					ToolCalls: []llm_dto.ToolCall{
						{
							ID:   "call-1",
							Type: "function",
							Function: llm_dto.FunctionCall{
								Name:      "get_weather",
								Arguments: `{"city":"London"}`,
							},
						},
					},
				},
				FinishReason: llm_dto.FinishReasonToolCalls,
			},
		},
		Usage: &llm_dto.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}
	h.mockProvider.SetResponse(respWithTools)

	_, err := h.service.NewCompletion().
		Model("test-model").
		User("What is the weather?").
		BufferMemory(h.memoryStore, "conv-tools", 20).
		Do(ctx)
	require.NoError(t, err)

	state, err := h.memoryStore.Load(ctx, "conv-tools")
	require.NoError(t, err)
	require.Equal(t, 2, state.MessageCount())

	assistantMessage := state.Messages[1]
	assert.Equal(t, llm_dto.RoleAssistant, assistantMessage.Role)
	require.Len(t, assistantMessage.ToolCalls, 1)
	assert.Equal(t, "get_weather", assistantMessage.ToolCalls[0].Function.Name)
	assert.Equal(t, `{"city":"London"}`, assistantMessage.ToolCalls[0].Function.Arguments)

	h.mockProvider.SetResponse(makeResponse("It is sunny.", 15, 5))
	_, err = h.service.NewCompletion().
		Model("test-model").
		User("Thanks!").
		BufferMemory(h.memoryStore, "conv-tools", 20).
		Do(ctx)
	require.NoError(t, err)

	calls := h.mockProvider.GetCompleteCalls()
	require.Len(t, calls, 2)
	turn2Messages := calls[1].Messages

	require.Len(t, turn2Messages, 3)
	assert.Equal(t, llm_dto.RoleAssistant, turn2Messages[1].Role)
	require.Len(t, turn2Messages[1].ToolCalls, 1, "tool calls should be preserved in history")
	assert.Equal(t, "get_weather", turn2Messages[1].ToolCalls[0].Function.Name)
}
