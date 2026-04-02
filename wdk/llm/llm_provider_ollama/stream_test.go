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

package llm_provider_ollama

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ollama/ollama/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_dto"
)

func newTestServerProvider(t *testing.T, responses []api.ChatResponse) *ollamaProvider {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		enc := json.NewEncoder(w)
		for _, response := range responses {
			if err := enc.Encode(response); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}))
	t.Cleanup(server.Close)

	u, err := url.Parse(server.URL)
	require.NoError(t, err)

	transport := http.DefaultTransport.(*http.Transport).Clone()
	t.Cleanup(transport.CloseIdleConnections)

	return &ollamaProvider{
		client:                api.NewClient(u, &http.Client{Transport: transport}),
		transport:             transport,
		defaultModel:          Model("llama3.2"),
		defaultEmbeddingModel: Model("all-minilm"),
		config:                Config{}.WithDefaults(),
	}
}

func TestProcessStream_BasicContent(t *testing.T) {
	responses := []api.ChatResponse{
		{
			Message: api.Message{Role: "assistant", Content: "Hello"},
		},
		{
			Message: api.Message{Role: "assistant", Content: " world"},
		},
		{
			Done: true,
		},
	}

	p := newTestServerProvider(t, responses)

	chatRequest := &api.ChatRequest{
		Model: "llama3.2",
		Messages: []api.Message{
			{Role: "user", Content: "Hi"},
		},
	}

	events := make(chan llm_dto.StreamEvent, 10)
	p.processStream(t.Context(), chatRequest, "llama3.2", events)

	var chunks []string
	var doneEvent *llm_dto.StreamEvent

	for event := range events {
		switch event.Type {
		case llm_dto.StreamEventChunk:
			require.NotNil(t, event.Chunk)
			require.NotNil(t, event.Chunk.Delta)
			require.NotNil(t, event.Chunk.Delta.Content)
			chunks = append(chunks, *event.Chunk.Delta.Content)
		case llm_dto.StreamEventDone:
			doneEvent = &event
		case llm_dto.StreamEventError:
			t.Fatalf("unexpected error event: %v", event.Error)
		}
	}

	assert.Equal(t, []string{"Hello", " world"}, chunks)
	require.NotNil(t, doneEvent, "expected a done event")
	assert.True(t, doneEvent.Done)
}

func TestProcessStream_TokenAccumulation(t *testing.T) {
	responses := []api.ChatResponse{
		{
			Message: api.Message{Role: "assistant", Content: "Hi"},
		},
		{
			Done: true,
			Metrics: api.Metrics{
				PromptEvalCount: 10,
				EvalCount:       5,
			},
		},
	}

	p := newTestServerProvider(t, responses)

	chatRequest := &api.ChatRequest{
		Model: "llama3.2",
		Messages: []api.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	events := make(chan llm_dto.StreamEvent, 10)
	p.processStream(t.Context(), chatRequest, "llama3.2", events)

	var doneEvent *llm_dto.StreamEvent
	for event := range events {
		if event.Type == llm_dto.StreamEventDone {
			doneEvent = &event
		}
	}

	require.NotNil(t, doneEvent, "expected a done event")
	require.NotNil(t, doneEvent.FinalResponse)
	require.NotNil(t, doneEvent.FinalResponse.Usage)
	assert.Equal(t, 10, doneEvent.FinalResponse.Usage.PromptTokens)
	assert.Equal(t, 5, doneEvent.FinalResponse.Usage.CompletionTokens)
	assert.Equal(t, 15, doneEvent.FinalResponse.Usage.TotalTokens)
}

func TestProcessStream_DoneEvent(t *testing.T) {
	responses := []api.ChatResponse{
		{
			Message: api.Message{Role: "assistant", Content: "Done test"},
		},
		{
			Done: true,
			Metrics: api.Metrics{
				PromptEvalCount: 3,
				EvalCount:       2,
			},
		},
	}

	p := newTestServerProvider(t, responses)

	chatRequest := &api.ChatRequest{
		Model: "llama3.2",
		Messages: []api.Message{
			{Role: "user", Content: "Test"},
		},
	}

	events := make(chan llm_dto.StreamEvent, 10)
	p.processStream(t.Context(), chatRequest, "llama3.2", events)

	var allEvents []llm_dto.StreamEvent
	for event := range events {
		allEvents = append(allEvents, event)
	}

	require.GreaterOrEqual(t, len(allEvents), 2)

	lastEvent := allEvents[len(allEvents)-1]
	assert.Equal(t, llm_dto.StreamEventDone, lastEvent.Type)
	assert.True(t, lastEvent.Done)
	require.NotNil(t, lastEvent.FinalResponse)
	assert.Equal(t, "llama3.2", lastEvent.FinalResponse.Model)
	require.Len(t, lastEvent.FinalResponse.Choices, 1)
	assert.Equal(t, llm_dto.RoleAssistant, lastEvent.FinalResponse.Choices[0].Message.Role)
	assert.Equal(t, llm_dto.FinishReasonStop, lastEvent.FinalResponse.Choices[0].FinishReason)
	require.NotNil(t, lastEvent.FinalResponse.Usage)
	assert.Equal(t, 3, lastEvent.FinalResponse.Usage.PromptTokens)
	assert.Equal(t, 2, lastEvent.FinalResponse.Usage.CompletionTokens)
	assert.Equal(t, 5, lastEvent.FinalResponse.Usage.TotalTokens)
}

func TestProcessStream_EmptyStream(t *testing.T) {

	responses := []api.ChatResponse{
		{
			Done: true,
		},
	}

	p := newTestServerProvider(t, responses)

	chatRequest := &api.ChatRequest{
		Model: "llama3.2",
		Messages: []api.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	events := make(chan llm_dto.StreamEvent, 10)
	p.processStream(t.Context(), chatRequest, "llama3.2", events)

	var allEvents []llm_dto.StreamEvent
	for event := range events {
		allEvents = append(allEvents, event)
	}

	require.Len(t, allEvents, 1)
	assert.Equal(t, llm_dto.StreamEventDone, allEvents[0].Type)
	assert.True(t, allEvents[0].Done)
}

func TestProcessStream_ModelPassedThrough(t *testing.T) {
	responses := []api.ChatResponse{
		{
			Message: api.Message{Role: "assistant", Content: "Hi"},
		},
		{
			Done: true,
		},
	}

	p := newTestServerProvider(t, responses)

	chatRequest := &api.ChatRequest{
		Model: "mistral:7b",
		Messages: []api.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	events := make(chan llm_dto.StreamEvent, 10)
	p.processStream(t.Context(), chatRequest, "mistral:7b", events)

	for event := range events {
		if event.Type == llm_dto.StreamEventChunk {
			assert.Equal(t, "mistral:7b", event.Chunk.Model)
		}
		if event.Type == llm_dto.StreamEventDone {
			require.NotNil(t, event.FinalResponse)
			assert.Equal(t, "mistral:7b", event.FinalResponse.Model)
		}
	}
}
