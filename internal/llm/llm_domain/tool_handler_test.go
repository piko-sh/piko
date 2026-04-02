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
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/llm/llm_dto"
)

func toolCallResponse(calls ...llm_dto.ToolCall) *llm_dto.CompletionResponse {
	return &llm_dto.CompletionResponse{
		ID:      "response-tc",
		Model:   "mock",
		Created: time.Now().Unix(),
		Choices: []llm_dto.Choice{
			{
				Index: 0,
				Message: llm_dto.Message{
					Role:      llm_dto.RoleAssistant,
					ToolCalls: calls,
				},
				FinishReason: llm_dto.FinishReasonToolCalls,
			},
		},
		Usage: &llm_dto.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
	}
}

func textResponse(content string) *llm_dto.CompletionResponse {
	return &llm_dto.CompletionResponse{
		ID:      "response-text",
		Model:   "mock",
		Created: time.Now().Unix(),
		Choices: []llm_dto.Choice{
			{
				Index: 0,
				Message: llm_dto.Message{
					Role:    llm_dto.RoleAssistant,
					Content: content,
				},
				FinishReason: llm_dto.FinishReasonStop,
			},
		},
		Usage: &llm_dto.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
	}
}

func makeToolCall(id, name, arguments string) llm_dto.ToolCall {
	return llm_dto.ToolCall{
		ID:   id,
		Type: "function",
		Function: llm_dto.FunctionCall{
			Name:      name,
			Arguments: arguments,
		},
	}
}

func TestToolFunc(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	handler := func(_ context.Context, arguments string) (string, error) {
		return "ok", nil
	}
	params := &llm_dto.JSONSchema{Type: "object"}

	b := service.NewCompletion().
		ToolFunc("get_weather", "Get weather", params, handler)

	require.Len(t, b.request.Tools, 1)
	assert.Equal(t, "get_weather", b.request.Tools[0].Function.Name)
	assert.Equal(t, "function", b.request.Tools[0].Type)
	assert.Nil(t, b.request.Tools[0].Function.Strict)

	require.True(t, b.hasToolHandlers())
	_, exists := b.toolHandlers["get_weather"]
	assert.True(t, exists)
}

func TestToolFunc_Chaining(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	h := func(_ context.Context, _ string) (string, error) { return "", nil }

	b := service.NewCompletion().
		ToolFunc("a", "tool a", nil, h).
		ToolFunc("b", "tool b", nil, h)

	assert.Len(t, b.request.Tools, 2)
	assert.Len(t, b.toolHandlers, 2)
}

func TestStrictToolFunc(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	handler := func(_ context.Context, _ string) (string, error) { return "ok", nil }
	params := &llm_dto.JSONSchema{Type: "object"}

	b := service.NewCompletion().
		StrictToolFunc("strict_tool", "Strict tool", params, handler)

	require.Len(t, b.request.Tools, 1)
	assert.Equal(t, "strict_tool", b.request.Tools[0].Function.Name)
	require.NotNil(t, b.request.Tools[0].Function.Strict)
	assert.True(t, *b.request.Tools[0].Function.Strict)

	_, exists := b.toolHandlers["strict_tool"]
	assert.True(t, exists)
}

func TestMaxToolRounds(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	t.Run("explicit value", func(t *testing.T) {
		b := service.NewCompletion().MaxToolRounds(5)
		assert.Equal(t, 5, b.resolvedMaxToolRounds())
	})

	t.Run("default when zero", func(t *testing.T) {
		b := service.NewCompletion()
		assert.Equal(t, DefaultMaxToolRounds, b.resolvedMaxToolRounds())
	})

	t.Run("unlimited when negative", func(t *testing.T) {
		b := service.NewCompletion().MaxToolRounds(-1)
		assert.Equal(t, -1, b.resolvedMaxToolRounds())
	})
}

func TestHasToolHandlers(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	t.Run("false when no handlers", func(t *testing.T) {
		b := service.NewCompletion()
		assert.False(t, b.hasToolHandlers())
	})

	t.Run("true after ToolFunc", func(t *testing.T) {
		h := func(_ context.Context, _ string) (string, error) { return "", nil }
		b := service.NewCompletion().ToolFunc("x", "x", nil, h)
		assert.True(t, b.hasToolHandlers())
	})
}

func TestDispatchToolCall_Registered(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	b := service.NewCompletion().
		ToolFunc("echo", "echo back", nil, func(_ context.Context, arguments string) (string, error) {
			return "echoed: " + arguments, nil
		})

	tc := makeToolCall("call-1", "echo", `{"text":"hello"}`)
	result := b.dispatchToolCall(context.Background(), tc)

	assert.Equal(t, `echoed: {"text":"hello"}`, result)
}

func TestDispatchToolCall_NotRegistered(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	b := service.NewCompletion()
	b.toolHandlers = make(map[string]ToolHandlerFunc)

	tc := makeToolCall("call-1", "unknown_tool", `{}`)
	result := b.dispatchToolCall(context.Background(), tc)

	assert.Contains(t, result, "error:")
	assert.Contains(t, result, "unknown_tool")
}

func TestDispatchToolCall_HandlerError(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	b := service.NewCompletion().
		ToolFunc("fail", "always fails", nil, func(_ context.Context, _ string) (string, error) {
			return "", fmt.Errorf("database connection lost")
		})

	tc := makeToolCall("call-1", "fail", `{}`)
	result := b.dispatchToolCall(context.Background(), tc)

	assert.Equal(t, "error: the tool encountered an internal error", result)
}

func TestToolLoop_SingleRound(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	callNum := int32(0)
	provider.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		n := atomic.AddInt32(&callNum, 1)
		if n == 1 {
			return toolCallResponse(makeToolCall("tc-1", "greet", `{"name":"world"}`)), nil
		}
		return textResponse("Hello, world!"), nil
	}

	b := service.NewCompletion().
		Model("mock-model").
		ToolFunc("greet", "Greet someone", nil, func(_ context.Context, arguments string) (string, error) {
			return "greeted", nil
		}).
		User("say hi")

	response, err := b.Do(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "Hello, world!", response.Content())

	assert.Equal(t, int64(2), atomic.LoadInt64(&provider.CompleteCallCount))
}

func TestToolLoop_MultipleRounds(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	callNum := int32(0)
	provider.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		n := atomic.AddInt32(&callNum, 1)
		if n <= 3 {
			return toolCallResponse(makeToolCall(fmt.Sprintf("tc-%d", n), "step", `{}`)), nil
		}
		return textResponse("done after 3 rounds"), nil
	}

	b := service.NewCompletion().
		Model("mock-model").
		ToolFunc("step", "Do a step", nil, func(_ context.Context, _ string) (string, error) {
			return "stepped", nil
		}).
		User("do steps")

	response, err := b.Do(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "done after 3 rounds", response.Content())
	assert.Equal(t, int64(4), atomic.LoadInt64(&provider.CompleteCallCount))
}

func TestToolLoop_MaxRoundsGuard(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	callNum := int32(0)
	provider.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		n := atomic.AddInt32(&callNum, 1)
		return toolCallResponse(makeToolCall(fmt.Sprintf("tc-%d", n), "loop", `{}`)), nil
	}

	b := service.NewCompletion().
		Model("mock-model").
		ToolFunc("loop", "Infinite loop", nil, func(_ context.Context, _ string) (string, error) {
			return "ok", nil
		}).
		MaxToolRounds(3).
		User("loop forever")

	response, err := b.Do(context.Background())
	require.NoError(t, err)

	assert.Equal(t, int64(4), atomic.LoadInt64(&provider.CompleteCallCount))

	assert.True(t, response.HasToolCalls())
}

func TestToolLoop_ParallelToolCalls(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	callNum := int32(0)
	provider.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		n := atomic.AddInt32(&callNum, 1)
		if n == 1 {
			return toolCallResponse(
				makeToolCall("tc-1", "tool_a", `{"x":1}`),
				makeToolCall("tc-2", "tool_b", `{"y":2}`),
			), nil
		}
		return textResponse("both tools done"), nil
	}

	var aCalled, bCalled atomic.Int32
	b := service.NewCompletion().
		Model("mock-model").
		ToolFunc("tool_a", "Tool A", nil, func(_ context.Context, _ string) (string, error) {
			aCalled.Add(1)
			return "a result", nil
		}).
		ToolFunc("tool_b", "Tool B", nil, func(_ context.Context, _ string) (string, error) {
			bCalled.Add(1)
			return "b result", nil
		}).
		User("use both")

	response, err := b.Do(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "both tools done", response.Content())
	assert.Equal(t, int32(1), aCalled.Load())
	assert.Equal(t, int32(1), bCalled.Load())
}

func TestToolLoop_NoHandlers_NoLoop(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	provider.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		return toolCallResponse(makeToolCall("tc-1", "unhandled", `{}`)), nil
	}

	b := service.NewCompletion().
		Model("mock-model").
		Tool("unhandled", "not handled", nil).
		User("test")

	response, err := b.Do(context.Background())
	require.NoError(t, err)

	assert.Equal(t, int64(1), atomic.LoadInt64(&provider.CompleteCallCount))
	assert.True(t, response.HasToolCalls())
}

func TestToolLoop_WithMemory(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	memStore := NewMockMemoryStore()
	mem := NewBufferMemory(memStore, WithBufferSize(100))

	callNum := int32(0)
	provider.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		n := atomic.AddInt32(&callNum, 1)
		if n == 1 {
			return toolCallResponse(makeToolCall("tc-1", "lookup", `{"q":"test"}`)), nil
		}
		return textResponse("final answer"), nil
	}

	b := service.NewCompletion().
		Model("mock-model").
		Memory(mem, "conv-1").
		ToolFunc("lookup", "Look up info", nil, func(_ context.Context, _ string) (string, error) {
			return "found it", nil
		}).
		User("find info")

	_, err := b.Do(context.Background())
	require.NoError(t, err)

	state, err := memStore.Load(context.Background(), "conv-1")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(state.Messages), 4)

	var foundToolResult bool
	for _, message := range state.Messages {
		if message.Role == llm_dto.RoleTool && message.Content == "found it" {
			foundToolResult = true
			break
		}
	}
	assert.True(t, foundToolResult, "tool result message should be recorded in memory")
}

func TestToolLoop_LLMErrorDuringLoop(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	callNum := int32(0)
	provider.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		n := atomic.AddInt32(&callNum, 1)
		if n == 1 {
			return toolCallResponse(makeToolCall("tc-1", "tool", `{}`)), nil
		}
		return nil, fmt.Errorf("provider unavailable")
	}

	b := service.NewCompletion().
		Model("mock-model").
		ToolFunc("tool", "A tool", nil, func(_ context.Context, _ string) (string, error) {
			return "ok", nil
		}).
		User("test")

	_, err := b.Do(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider unavailable")
}

func TestToolLoop_ContextCancellation(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	ctx, cancel := context.WithCancelCause(context.Background())

	callNum := int32(0)
	provider.CompleteFunc = func(ctx context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		n := atomic.AddInt32(&callNum, 1)
		if n == 1 {
			return toolCallResponse(makeToolCall("tc-1", "tool", `{}`)), nil
		}

		cancel(fmt.Errorf("test: simulating cancelled context"))
		return nil, ctx.Err()
	}

	b := service.NewCompletion().
		Model("mock-model").
		ToolFunc("tool", "A tool", nil, func(_ context.Context, _ string) (string, error) {
			return "ok", nil
		}).
		User("test")

	_, err := b.Do(ctx)
	require.Error(t, err)
}

func TestExecuteToolLoop_ContextCancelled(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	ctx, cancel := context.WithCancelCause(context.Background())

	var callNum atomic.Int32
	provider.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		n := callNum.Add(1)
		if n == 1 {
			cancel(fmt.Errorf("test: simulating cancelled context"))
		}
		return toolCallResponse(makeToolCall(fmt.Sprintf("tc-%d", n), "step", `{}`)), nil
	}

	b := service.NewCompletion().
		Model("mock-model").
		ToolFunc("step", "Do a step", nil, func(_ context.Context, _ string) (string, error) {
			return "stepped", nil
		}).
		User("do steps")

	_, err := b.Do(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}
