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

package llm_provider_mistral

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_dto"
)

func TestReadSSELine_ValidData(t *testing.T) {
	p := newTestProvider(t)
	events := make(chan llm_dto.StreamEvent, 10)
	ctx := context.Background()

	input := "data: {\"id\":\"cmpl-1\",\"model\":\"mistral-large\"}\n"
	reader := bufio.NewReader(strings.NewReader(input))

	data, done := p.readSSELine(ctx, events, reader)

	assert.False(t, done)
	require.NotNil(t, data)
	assert.Equal(t, `{"id":"cmpl-1","model":"mistral-large"}`, string(data))
}

func TestReadSSELine_DoneSignal(t *testing.T) {
	p := newTestProvider(t)
	events := make(chan llm_dto.StreamEvent, 10)
	ctx := context.Background()

	input := "data: [DONE]\n"
	reader := bufio.NewReader(strings.NewReader(input))

	data, done := p.readSSELine(ctx, events, reader)

	assert.True(t, done)
	assert.Nil(t, data)
}

func TestReadSSELine_EmptyLines(t *testing.T) {
	p := newTestProvider(t)
	events := make(chan llm_dto.StreamEvent, 10)
	ctx := context.Background()

	t.Run("blank line returns nil and continues", func(t *testing.T) {
		input := "\n"
		reader := bufio.NewReader(strings.NewReader(input))

		data, done := p.readSSELine(ctx, events, reader)

		assert.False(t, done)
		assert.Nil(t, data)
	})

	t.Run("whitespace-only line returns nil and continues", func(t *testing.T) {
		input := "   \n"
		reader := bufio.NewReader(strings.NewReader(input))

		data, done := p.readSSELine(ctx, events, reader)

		assert.False(t, done)
		assert.Nil(t, data)
	})

	t.Run("non-data prefix line is skipped", func(t *testing.T) {
		input := "event: message\n"
		reader := bufio.NewReader(strings.NewReader(input))

		data, done := p.readSSELine(ctx, events, reader)

		assert.False(t, done)
		assert.Nil(t, data)
	})

	t.Run("EOF returns done", func(t *testing.T) {
		input := ""
		reader := bufio.NewReader(strings.NewReader(input))

		data, done := p.readSSELine(ctx, events, reader)

		assert.True(t, done)
		assert.Nil(t, data)
	})
}

func TestParseSSEData_ValidJSON(t *testing.T) {
	p := newTestProvider(t)

	data := []byte(`{"id":"cmpl-1","object":"chat.completion.chunk","created":1234,"model":"mistral-large","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}],"usage":null}`)

	chunk, ok := p.parseSSEData(data)

	require.True(t, ok)
	require.NotNil(t, chunk)
	assert.Equal(t, "cmpl-1", chunk.ID)
	assert.Equal(t, "chat.completion.chunk", chunk.Object)
	assert.Equal(t, int64(1234), chunk.Created)
	assert.Equal(t, "mistral-large", chunk.Model)
	require.Len(t, chunk.Choices, 1)
	assert.Equal(t, 0, chunk.Choices[0].Index)
	assert.Equal(t, "assistant", chunk.Choices[0].Delta.Role)
	assert.Equal(t, "Hello", chunk.Choices[0].Delta.Content)
	assert.Empty(t, chunk.Choices[0].FinishReason)
	assert.Nil(t, chunk.Usage)
}

func TestParseSSEData_InvalidJSON(t *testing.T) {
	p := newTestProvider(t)

	t.Run("malformed JSON returns false", func(t *testing.T) {
		data := []byte(`{invalid json}`)
		chunk, ok := p.parseSSEData(data)

		assert.False(t, ok)
		assert.Nil(t, chunk)
	})

	t.Run("empty data returns false", func(t *testing.T) {
		data := []byte(``)
		chunk, ok := p.parseSSEData(data)

		assert.False(t, ok)
		assert.Nil(t, chunk)
	})

	t.Run("non-JSON text returns false", func(t *testing.T) {
		data := []byte(`not json at all`)
		chunk, ok := p.parseSSEData(data)

		assert.False(t, ok)
		assert.Nil(t, chunk)
	})
}

func TestBuildDelta_TextContent(t *testing.T) {
	p := newTestProvider(t)
	state := &streamState{}

	choice := &mistralStreamChunkChoice{
		Index: 0,
		Delta: mistralDelta{
			Role:    "assistant",
			Content: "Hello world",
		},
	}

	delta := p.buildDelta(choice, state)

	require.NotNil(t, delta)
	require.NotNil(t, delta.Role)
	assert.Equal(t, llm_dto.Role("assistant"), *delta.Role)
	require.NotNil(t, delta.Content)
	assert.Equal(t, "Hello world", *delta.Content)
	assert.Nil(t, delta.ToolCalls)
}

func TestBuildDelta_EmptyContent(t *testing.T) {
	p := newTestProvider(t)
	state := &streamState{}

	t.Run("empty role and content", func(t *testing.T) {
		choice := &mistralStreamChunkChoice{
			Index: 0,
			Delta: mistralDelta{},
		}

		delta := p.buildDelta(choice, state)

		require.NotNil(t, delta)
		assert.Nil(t, delta.Role)
		assert.Nil(t, delta.Content)
		assert.Nil(t, delta.ToolCalls)
	})

	t.Run("role only without content", func(t *testing.T) {
		choice := &mistralStreamChunkChoice{
			Index: 0,
			Delta: mistralDelta{
				Role: "assistant",
			},
		}

		delta := p.buildDelta(choice, state)

		require.NotNil(t, delta)
		require.NotNil(t, delta.Role)
		assert.Equal(t, llm_dto.Role("assistant"), *delta.Role)
		assert.Nil(t, delta.Content)
	})
}

func TestBuildToolCallDeltas(t *testing.T) {
	p := newTestProvider(t)
	state := &streamState{}

	toolCalls := []mistralToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: mistralFunctionCall{
				Name:      "get_weather",
				Arguments: `{"location":`,
			},
		},
		{
			ID:   "call_2",
			Type: "function",
			Function: mistralFunctionCall{
				Name:      "get_time",
				Arguments: `{"zone":`,
			},
		},
	}

	deltas := p.buildToolCallDeltas(toolCalls, state)

	require.Len(t, deltas, 2)

	assert.Equal(t, 0, deltas[0].Index)
	require.NotNil(t, deltas[0].ID)
	assert.Equal(t, "call_1", *deltas[0].ID)
	require.NotNil(t, deltas[0].Type)
	assert.Equal(t, "function", *deltas[0].Type)
	require.NotNil(t, deltas[0].Function)
	require.NotNil(t, deltas[0].Function.Name)
	assert.Equal(t, "get_weather", *deltas[0].Function.Name)
	require.NotNil(t, deltas[0].Function.Arguments)
	assert.Equal(t, `{"location":`, *deltas[0].Function.Arguments)

	assert.Equal(t, 1, deltas[1].Index)
	require.NotNil(t, deltas[1].ID)
	assert.Equal(t, "call_2", *deltas[1].ID)
	require.NotNil(t, deltas[1].Function)
	require.NotNil(t, deltas[1].Function.Name)
	assert.Equal(t, "get_time", *deltas[1].Function.Name)

	require.Len(t, state.accumulatedToolCalls, 2)
	assert.Equal(t, "call_1", state.accumulatedToolCalls[0].ID)
	assert.Equal(t, "call_2", state.accumulatedToolCalls[1].ID)
}

func TestAccumulateToolCall_NewCall(t *testing.T) {
	p := newTestProvider(t)
	toolCalls := []llm_dto.ToolCall{}

	delta := llm_dto.ToolCallDelta{
		Index: 0,
		ID:    new("call_abc"),
		Type:  new("function"),
		Function: &llm_dto.FunctionCallDelta{
			Name:      new("get_weather"),
			Arguments: new(`{"location":`),
		},
	}

	p.accumulateToolCall(&toolCalls, delta)

	require.Len(t, toolCalls, 1)
	assert.Equal(t, "call_abc", toolCalls[0].ID)
	assert.Equal(t, "function", toolCalls[0].Type)
	assert.Equal(t, "get_weather", toolCalls[0].Function.Name)
	assert.Equal(t, `{"location":`, toolCalls[0].Function.Arguments)
}

func TestAccumulateToolCall_AppendArguments(t *testing.T) {
	p := newTestProvider(t)
	toolCalls := []llm_dto.ToolCall{
		{
			ID:   "call_abc",
			Type: "function",
			Function: llm_dto.FunctionCall{
				Name:      "get_weather",
				Arguments: `{"location":`,
			},
		},
	}

	delta := llm_dto.ToolCallDelta{
		Index: 0,
		Function: &llm_dto.FunctionCallDelta{
			Arguments: new(`"London"}`),
		},
	}

	p.accumulateToolCall(&toolCalls, delta)

	require.Len(t, toolCalls, 1)
	assert.Equal(t, "call_abc", toolCalls[0].ID)
	assert.Equal(t, "get_weather", toolCalls[0].Function.Name)
	assert.Equal(t, `{"location":"London"}`, toolCalls[0].Function.Arguments)
}

func TestAccumulateToolCall_MultipleIndices(t *testing.T) {
	p := newTestProvider(t)
	toolCalls := []llm_dto.ToolCall{}

	delta0 := llm_dto.ToolCallDelta{
		Index: 0,
		ID:    new("call_1"),
		Type:  new("function"),
		Function: &llm_dto.FunctionCallDelta{
			Name:      new("func_a"),
			Arguments: new(`{"a":1}`),
		},
	}
	p.accumulateToolCall(&toolCalls, delta0)

	delta1 := llm_dto.ToolCallDelta{
		Index: 1,
		ID:    new("call_2"),
		Type:  new("function"),
		Function: &llm_dto.FunctionCallDelta{
			Name:      new("func_b"),
			Arguments: new(`{"b":2}`),
		},
	}
	p.accumulateToolCall(&toolCalls, delta1)

	require.Len(t, toolCalls, 2)
	assert.Equal(t, "call_1", toolCalls[0].ID)
	assert.Equal(t, "func_a", toolCalls[0].Function.Name)
	assert.Equal(t, "call_2", toolCalls[1].ID)
	assert.Equal(t, "func_b", toolCalls[1].Function.Name)
}

func TestExtractFinishReason(t *testing.T) {
	p := newTestProvider(t)

	t.Run("stop reason", func(t *testing.T) {
		state := &streamState{}
		choice := &mistralStreamChunkChoice{
			FinishReason: "stop",
		}

		reason := p.extractFinishReason(choice, state)

		require.NotNil(t, reason)
		assert.Equal(t, llm_dto.FinishReasonStop, *reason)
		require.NotNil(t, state.lastFinishReason)
		assert.Equal(t, llm_dto.FinishReasonStop, *state.lastFinishReason)
	})

	t.Run("tool_calls reason", func(t *testing.T) {
		state := &streamState{}
		choice := &mistralStreamChunkChoice{
			FinishReason: "tool_calls",
		}

		reason := p.extractFinishReason(choice, state)

		require.NotNil(t, reason)
		assert.Equal(t, llm_dto.FinishReasonToolCalls, *reason)
	})

	t.Run("length reason", func(t *testing.T) {
		state := &streamState{}
		choice := &mistralStreamChunkChoice{
			FinishReason: "length",
		}

		reason := p.extractFinishReason(choice, state)

		require.NotNil(t, reason)
		assert.Equal(t, llm_dto.FinishReasonLength, *reason)
	})

	t.Run("empty finish reason returns nil", func(t *testing.T) {
		state := &streamState{}
		choice := &mistralStreamChunkChoice{
			FinishReason: "",
		}

		reason := p.extractFinishReason(choice, state)

		assert.Nil(t, reason)
		assert.Nil(t, state.lastFinishReason)
	})
}

func TestExtractUsage(t *testing.T) {
	p := newTestProvider(t)

	t.Run("extracts usage when present", func(t *testing.T) {
		state := &streamState{}
		chunk := &mistralStreamChunk{
			Usage: &mistralUsage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}
		streamChunk := &llm_dto.StreamChunk{}

		p.extractUsage(chunk, streamChunk, state)

		require.NotNil(t, streamChunk.Usage)
		assert.Equal(t, 10, streamChunk.Usage.PromptTokens)
		assert.Equal(t, 5, streamChunk.Usage.CompletionTokens)
		assert.Equal(t, 15, streamChunk.Usage.TotalTokens)
		require.NotNil(t, state.finalUsage)
		assert.Equal(t, 15, state.finalUsage.TotalTokens)
	})

	t.Run("skips nil usage", func(t *testing.T) {
		state := &streamState{}
		chunk := &mistralStreamChunk{
			Usage: nil,
		}
		streamChunk := &llm_dto.StreamChunk{}

		p.extractUsage(chunk, streamChunk, state)

		assert.Nil(t, streamChunk.Usage)
		assert.Nil(t, state.finalUsage)
	})

	t.Run("skips usage with zero total tokens", func(t *testing.T) {
		state := &streamState{}
		chunk := &mistralStreamChunk{
			Usage: &mistralUsage{
				PromptTokens:     0,
				CompletionTokens: 0,
				TotalTokens:      0,
			},
		}
		streamChunk := &llm_dto.StreamChunk{}

		p.extractUsage(chunk, streamChunk, state)

		assert.Nil(t, streamChunk.Usage)
		assert.Nil(t, state.finalUsage)
	})
}

func TestBuildFinalResponse(t *testing.T) {
	p := newTestProvider(t)

	t.Run("builds response with all fields", func(t *testing.T) {
		state := &streamState{
			lastID:           "cmpl-1",
			lastModel:        "mistral-large",
			lastFinishReason: new(llm_dto.FinishReasonStop),
			finalUsage: &llm_dto.Usage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
			accumulatedToolCalls: nil,
		}

		response := p.buildFinalResponse(state)

		assert.Equal(t, "cmpl-1", response.ID)
		assert.Equal(t, "mistral-large", response.Model)
		require.NotNil(t, response.Usage)
		assert.Equal(t, 15, response.Usage.TotalTokens)
		require.Len(t, response.Choices, 1)
		assert.Equal(t, llm_dto.RoleAssistant, response.Choices[0].Message.Role)
		assert.Equal(t, llm_dto.FinishReasonStop, response.Choices[0].FinishReason)
	})

	t.Run("builds response with tool calls", func(t *testing.T) {
		state := &streamState{
			lastID:           "cmpl-2",
			lastModel:        "mistral-large",
			lastFinishReason: new(llm_dto.FinishReasonToolCalls),
			accumulatedToolCalls: []llm_dto.ToolCall{
				{
					ID:   "call_abc",
					Type: "function",
					Function: llm_dto.FunctionCall{
						Name:      "get_weather",
						Arguments: `{"location":"Paris"}`,
					},
				},
			},
		}

		response := p.buildFinalResponse(state)

		require.Len(t, response.Choices, 1)
		assert.Equal(t, llm_dto.FinishReasonToolCalls, response.Choices[0].FinishReason)
		require.Len(t, response.Choices[0].Message.ToolCalls, 1)
		assert.Equal(t, "call_abc", response.Choices[0].Message.ToolCalls[0].ID)
		assert.Equal(t, "get_weather", response.Choices[0].Message.ToolCalls[0].Function.Name)
		assert.Equal(t, `{"location":"Paris"}`, response.Choices[0].Message.ToolCalls[0].Function.Arguments)
	})

	t.Run("builds response without finish reason yields no choices", func(t *testing.T) {
		state := &streamState{
			lastID:           "cmpl-3",
			lastModel:        "mistral-large",
			lastFinishReason: nil,
		}

		response := p.buildFinalResponse(state)

		assert.Equal(t, "cmpl-3", response.ID)
		assert.Empty(t, response.Choices)
	})

	t.Run("builds response without usage", func(t *testing.T) {
		state := &streamState{
			lastID:           "cmpl-4",
			lastModel:        "mistral-large",
			lastFinishReason: new(llm_dto.FinishReasonStop),
			finalUsage:       nil,
		}

		response := p.buildFinalResponse(state)

		assert.Nil(t, response.Usage)
		require.Len(t, response.Choices, 1)
	})
}

func newTestProviderWithServer(t *testing.T, server *httptest.Server) *mistralProvider {
	t.Helper()
	config := Config{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}
	provider, err := New(config)
	require.NoError(t, err)
	mp, ok := provider.(*mistralProvider)
	require.True(t, ok, "expected provider to be *mistralProvider")
	return mp
}

func collectEvents(t *testing.T, eventChannel <-chan llm_dto.StreamEvent) []llm_dto.StreamEvent {
	t.Helper()
	var events []llm_dto.StreamEvent
	timeout := time.After(5 * time.Second)
	for {
		select {
		case event, ok := <-eventChannel:
			if !ok {
				return events
			}
			events = append(events, event)
		case <-timeout:
			t.Fatal("timed out waiting for stream events")
			return events
		}
	}
}

func TestStream_BasicPrompt(t *testing.T) {
	sseData := `data: {"id":"cmpl-1","object":"chat.completion.chunk","created":1234,"model":"mistral-large","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":""}],"usage":null}

data: {"id":"cmpl-1","object":"chat.completion.chunk","created":1234,"model":"mistral-large","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":""}],"usage":null}

data: {"id":"cmpl-1","object":"chat.completion.chunk","created":1234,"model":"mistral-large","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}

data: [DONE]
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "text/event-stream", r.Header.Get("Accept"))

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, sseData)
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Say hello"},
		},
	}

	streamChannel, err := p.Stream(context.Background(), request)
	require.NoError(t, err)

	events := collectEvents(t, streamChannel)
	require.NotEmpty(t, events)

	var textContent strings.Builder
	chunkCount := 0
	for _, event := range events {
		if event.Type == llm_dto.StreamEventChunk && event.Chunk != nil {
			chunkCount++
			if event.Chunk.Delta != nil && event.Chunk.Delta.Content != nil {
				textContent.WriteString(*event.Chunk.Delta.Content)
			}
		}
	}

	assert.Equal(t, "Hello world", textContent.String())
	assert.GreaterOrEqual(t, chunkCount, 2, "expected at least 2 chunk events")

	lastEvent := events[len(events)-1]
	assert.Equal(t, llm_dto.StreamEventDone, lastEvent.Type)
	assert.True(t, lastEvent.Done)
	require.NotNil(t, lastEvent.FinalResponse)
	assert.Equal(t, "cmpl-1", lastEvent.FinalResponse.ID)
	assert.Equal(t, "mistral-large", lastEvent.FinalResponse.Model)
	require.NotNil(t, lastEvent.FinalResponse.Usage)
	assert.Equal(t, 10, lastEvent.FinalResponse.Usage.PromptTokens)
	assert.Equal(t, 5, lastEvent.FinalResponse.Usage.CompletionTokens)
	assert.Equal(t, 15, lastEvent.FinalResponse.Usage.TotalTokens)
	require.Len(t, lastEvent.FinalResponse.Choices, 1)
	assert.Equal(t, llm_dto.FinishReasonStop, lastEvent.FinalResponse.Choices[0].FinishReason)
}

func TestStream_ToolCallAccumulation(t *testing.T) {
	sseData := `data: {"id":"cmpl-2","object":"chat.completion.chunk","created":1234,"model":"mistral-large","choices":[{"index":0,"delta":{"role":"assistant","tool_calls":[{"id":"call_abc","type":"function","function":{"name":"get_weather","arguments":"{\"location\":"}}]},"finish_reason":""}],"usage":null}

data: {"id":"cmpl-2","object":"chat.completion.chunk","created":1234,"model":"mistral-large","choices":[{"index":0,"delta":{"tool_calls":[{"id":"","type":"","function":{"name":"","arguments":"\"London\"}"}}]},"finish_reason":""}],"usage":null}

data: {"id":"cmpl-2","object":"chat.completion.chunk","created":1234,"model":"mistral-large","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":20,"completion_tokens":10,"total_tokens":30}}

data: [DONE]
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, sseData)
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "What is the weather in London?"},
		},
	}

	streamChannel, err := p.Stream(context.Background(), request)
	require.NoError(t, err)

	events := collectEvents(t, streamChannel)
	require.NotEmpty(t, events)

	toolCallDeltaCount := 0
	for _, event := range events {
		if event.Type == llm_dto.StreamEventChunk && event.Chunk != nil &&
			event.Chunk.Delta != nil && len(event.Chunk.Delta.ToolCalls) > 0 {
			toolCallDeltaCount++
		}
	}
	assert.GreaterOrEqual(t, toolCallDeltaCount, 1, "expected at least 1 chunk with tool call deltas")

	lastEvent := events[len(events)-1]
	assert.Equal(t, llm_dto.StreamEventDone, lastEvent.Type)
	require.NotNil(t, lastEvent.FinalResponse)
	require.Len(t, lastEvent.FinalResponse.Choices, 1)
	assert.Equal(t, llm_dto.FinishReasonToolCalls, lastEvent.FinalResponse.Choices[0].FinishReason)
	require.Len(t, lastEvent.FinalResponse.Choices[0].Message.ToolCalls, 1)

	tc := lastEvent.FinalResponse.Choices[0].Message.ToolCalls[0]
	assert.Equal(t, "call_abc", tc.ID)
	assert.Equal(t, "function", tc.Type)
	assert.Equal(t, "get_weather", tc.Function.Name)
	assert.Equal(t, `{"location":"London"}`, tc.Function.Arguments)

	require.NotNil(t, lastEvent.FinalResponse.Usage)
	assert.Equal(t, 30, lastEvent.FinalResponse.Usage.TotalTokens)
}

func TestStream_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		require.True(t, ok, "expected ResponseWriter to be Flusher")

		_, _ = fmt.Fprint(w, "data: {\"id\":\"cmpl-3\",\"object\":\"chat.completion.chunk\",\"created\":1234,\"model\":\"mistral-large\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"Hi\"},\"finish_reason\":\"\"}],\"usage\":null}\n\n")
		flusher.Flush()

		<-r.Context().Done()
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)

	ctx, cancel := context.WithCancelCause(context.Background())

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hello"},
		},
	}

	streamChannel, err := p.Stream(ctx, request)
	require.NoError(t, err)

	select {
	case event := <-streamChannel:
		assert.Equal(t, llm_dto.StreamEventChunk, event.Type)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for first event")
	}

	cancel(fmt.Errorf("test: simulating cancelled context: %w", context.Canceled))

	var foundError bool
	timeout := time.After(5 * time.Second)
drainLoop:
	for {
		select {
		case event, ok := <-streamChannel:
			if !ok {
				break drainLoop
			}
			if event.Type == llm_dto.StreamEventError {
				foundError = true
				assert.ErrorIs(t, event.Error, context.Canceled)
			}
		case <-timeout:
			t.Fatal("timed out waiting for channel to close after cancellation")
		}
	}

	assert.True(t, foundError, "expected an error event after context cancellation")
}

func TestStream_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, `{"error":{"message":"Invalid API key"}}`)
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hello"},
		},
	}

	streamChannel, err := p.Stream(context.Background(), request)

	assert.Nil(t, streamChannel)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mistral API error")
	assert.Contains(t, err.Error(), "401")
}

func TestStream_DefaultModel(t *testing.T) {
	var receivedModel string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var reqBody struct {
			Model string `json:"model"`
		}
		err := func() error {
			defer func() { _ = r.Body.Close() }()
			return json.NewDecoder(r.Body).Decode(&reqBody)
		}()
		require.NoError(t, err)
		receivedModel = reqBody.Model

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "data: {\"id\":\"cmpl-5\",\"object\":\"chat.completion.chunk\",\"created\":1234,\"model\":\"mistral-large-latest\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":1,\"total_tokens\":2}}\n\ndata: [DONE]\n")
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hi"},
		},
	}

	streamChannel, err := p.Stream(context.Background(), request)
	require.NoError(t, err)

	_ = collectEvents(t, streamChannel)
	assert.Equal(t, "mistral-large-latest", receivedModel)
}

func TestStream_CustomModel(t *testing.T) {
	var receivedModel string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody struct {
			Model string `json:"model"`
		}
		err := func() error {
			defer func() { _ = r.Body.Close() }()
			return json.NewDecoder(r.Body).Decode(&reqBody)
		}()
		require.NoError(t, err)
		receivedModel = reqBody.Model

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "data: {\"id\":\"cmpl-6\",\"object\":\"chat.completion.chunk\",\"created\":1234,\"model\":\"mistral-small-latest\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":1,\"total_tokens\":2}}\n\ndata: [DONE]\n")
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)

	request := &llm_dto.CompletionRequest{
		Model: "mistral-small-latest",
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hi"},
		},
	}

	streamChannel, err := p.Stream(context.Background(), request)
	require.NoError(t, err)

	_ = collectEvents(t, streamChannel)
	assert.Equal(t, "mistral-small-latest", receivedModel)
}

func TestStream_MultipleChunksWithRoleOnFirstOnly(t *testing.T) {
	sseData := `data: {"id":"cmpl-7","object":"chat.completion.chunk","created":1234,"model":"mistral-large","choices":[{"index":0,"delta":{"role":"assistant","content":"A"},"finish_reason":""}],"usage":null}

data: {"id":"cmpl-7","object":"chat.completion.chunk","created":1234,"model":"mistral-large","choices":[{"index":0,"delta":{"content":"B"},"finish_reason":""}],"usage":null}

data: {"id":"cmpl-7","object":"chat.completion.chunk","created":1234,"model":"mistral-large","choices":[{"index":0,"delta":{"content":"C"},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":3,"total_tokens":8}}

data: [DONE]
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, sseData)
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)

	streamChannel, err := p.Stream(context.Background(), &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Count"},
		},
	})
	require.NoError(t, err)

	events := collectEvents(t, streamChannel)

	var roleCount int
	var fullContent strings.Builder
	for _, event := range events {
		if event.Type == llm_dto.StreamEventChunk && event.Chunk != nil && event.Chunk.Delta != nil {
			if event.Chunk.Delta.Role != nil {
				roleCount++
			}
			if event.Chunk.Delta.Content != nil {
				fullContent.WriteString(*event.Chunk.Delta.Content)
			}
		}
	}

	assert.Equal(t, 1, roleCount, "expected role to be set only on the first chunk")
	assert.Equal(t, "ABC", fullContent.String())
}

func TestStream_EmptySSEResponse(t *testing.T) {
	sseData := "data: [DONE]\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, sseData)
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)

	streamChannel, err := p.Stream(context.Background(), &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hi"},
		},
	})
	require.NoError(t, err)

	events := collectEvents(t, streamChannel)
	require.NotEmpty(t, events)

	lastEvent := events[len(events)-1]
	assert.Equal(t, llm_dto.StreamEventDone, lastEvent.Type)
	assert.True(t, lastEvent.Done)
}

func TestIsContextCancelled(t *testing.T) {
	p := newTestProvider(t)

	t.Run("returns false when context is active", func(t *testing.T) {
		events := make(chan llm_dto.StreamEvent, 10)
		ctx := context.Background()

		cancelled := p.isContextCancelled(ctx, events)

		assert.False(t, cancelled)
		assert.Empty(t, events)
	})

	t.Run("returns true and sends error when context is cancelled", func(t *testing.T) {
		events := make(chan llm_dto.StreamEvent, 10)
		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(fmt.Errorf("test: simulating cancelled context: %w", context.Canceled))

		cancelled := p.isContextCancelled(ctx, events)

		assert.True(t, cancelled)
		require.Len(t, events, 1)
		event := <-events
		assert.Equal(t, llm_dto.StreamEventError, event.Type)
		assert.ErrorIs(t, event.Error, context.Canceled)
	})
}

func TestSendEvent(t *testing.T) {
	p := newTestProvider(t)

	t.Run("sends event successfully", func(t *testing.T) {
		events := make(chan llm_dto.StreamEvent, 10)
		ctx := context.Background()
		event := llm_dto.NewChunkEvent(&llm_dto.StreamChunk{ID: "test"})

		ok := p.sendEvent(ctx, events, event)

		assert.True(t, ok)
		require.Len(t, events, 1)
		received := <-events
		assert.Equal(t, llm_dto.StreamEventChunk, received.Type)
		assert.Equal(t, "test", received.Chunk.ID)
	})

	t.Run("handles cancelled context", func(t *testing.T) {

		events := make(chan llm_dto.StreamEvent, 10)
		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(fmt.Errorf("test: simulating cancelled context: %w", context.Canceled))

		event := llm_dto.NewChunkEvent(&llm_dto.StreamChunk{ID: "test"})
		ok := p.sendEvent(ctx, events, event)

		if ok {

			received := <-events
			assert.Equal(t, llm_dto.StreamEventChunk, received.Type)
		} else {

			received := <-events
			assert.Equal(t, llm_dto.StreamEventError, received.Type)
			assert.ErrorIs(t, received.Error, context.Canceled)
		}
	})
}

func TestProcessChunkChoices(t *testing.T) {
	p := newTestProvider(t)

	t.Run("processes text chunk", func(t *testing.T) {
		events := make(chan llm_dto.StreamEvent, 10)
		ctx := context.Background()
		state := &streamState{}
		chunk := &mistralStreamChunk{
			ID:    "cmpl-test",
			Model: "mistral-large",
			Choices: []mistralStreamChunkChoice{
				{
					Index: 0,
					Delta: mistralDelta{
						Content: "Hello",
					},
				},
			},
		}

		ok := p.processChunkChoices(ctx, events, chunk, state)

		assert.True(t, ok)
		require.Len(t, events, 1)
		event := <-events
		assert.Equal(t, llm_dto.StreamEventChunk, event.Type)
		require.NotNil(t, event.Chunk)
		assert.Equal(t, "cmpl-test", event.Chunk.ID)
		assert.Equal(t, "mistral-large", event.Chunk.Model)
		require.NotNil(t, event.Chunk.Delta)
		require.NotNil(t, event.Chunk.Delta.Content)
		assert.Equal(t, "Hello", *event.Chunk.Delta.Content)
	})

	t.Run("processes multiple choices", func(t *testing.T) {
		events := make(chan llm_dto.StreamEvent, 10)
		ctx := context.Background()
		state := &streamState{}
		chunk := &mistralStreamChunk{
			ID:    "cmpl-test",
			Model: "mistral-large",
			Choices: []mistralStreamChunkChoice{
				{
					Index: 0,
					Delta: mistralDelta{Content: "A"},
				},
				{
					Index: 1,
					Delta: mistralDelta{Content: "B"},
				},
			},
		}

		ok := p.processChunkChoices(ctx, events, chunk, state)

		assert.True(t, ok)
		assert.Len(t, events, 2)
	})

	t.Run("processes chunk with usage", func(t *testing.T) {
		events := make(chan llm_dto.StreamEvent, 10)
		ctx := context.Background()
		state := &streamState{}
		chunk := &mistralStreamChunk{
			ID:    "cmpl-test",
			Model: "mistral-large",
			Choices: []mistralStreamChunkChoice{
				{
					Index:        0,
					Delta:        mistralDelta{},
					FinishReason: "stop",
				},
			},
			Usage: &mistralUsage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}

		ok := p.processChunkChoices(ctx, events, chunk, state)

		assert.True(t, ok)
		require.Len(t, events, 1)
		event := <-events
		require.NotNil(t, event.Chunk)
		require.NotNil(t, event.Chunk.Usage)
		assert.Equal(t, 15, event.Chunk.Usage.TotalTokens)
		require.NotNil(t, event.Chunk.FinishReason)
		assert.Equal(t, llm_dto.FinishReasonStop, *event.Chunk.FinishReason)
	})
}
