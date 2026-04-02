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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_dto"
)

func TestStream_BasicPrompt(t *testing.T) {

	handle, ctx := createOllamaProvider(t)

	events, err := handle.llm.Stream(ctx, &llm_dto.CompletionRequest{
		Model: globalEnv.completionModel,
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Count from 1 to 5"},
		},
		MaxTokens: new(500),
	})
	require.NoError(t, err, "starting stream")
	require.NotNil(t, events)

	var chunks []string
	var gotDone bool

	for event := range events {
		switch event.Type {
		case llm_dto.StreamEventChunk:
			require.NotNil(t, event.Chunk, "chunk event should have chunk")
			require.NotNil(t, event.Chunk.Delta, "chunk should have delta")
			if event.Chunk.Delta.Content != nil {
				chunks = append(chunks, *event.Chunk.Delta.Content)
			}
		case llm_dto.StreamEventDone:
			gotDone = true
			assert.NotNil(t, event.FinalResponse, "done event should have final response")
		case llm_dto.StreamEventError:
			t.Fatalf("unexpected stream error: %v", event.Error)
		}
	}

	assert.True(t, gotDone, "expected done event")
	assert.NotEmpty(t, chunks, "expected at least one chunk")

	full := strings.Join(chunks, "")
	assert.NotEmpty(t, full, "concatenated stream content should not be empty")
}

func TestStream_ChannelCloses(t *testing.T) {

	handle, ctx := createOllamaProvider(t)

	events, err := handle.llm.Stream(ctx, &llm_dto.CompletionRequest{
		Model: globalEnv.completionModel,
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Say hi"},
		},
		MaxTokens: new(50),
	})
	require.NoError(t, err)

	count := 0
	for range events {
		count++
	}
	assert.Greater(t, count, 0, "expected at least one event before channel close")
}

func TestStream_DoneEventHasUsage(t *testing.T) {

	handle, ctx := createOllamaProvider(t)

	events, err := handle.llm.Stream(ctx, &llm_dto.CompletionRequest{
		Model: globalEnv.completionModel,
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hello"},
		},
		MaxTokens: new(50),
	})
	require.NoError(t, err)

	for event := range events {
		if event.Type == llm_dto.StreamEventDone {
			require.NotNil(t, event.FinalResponse, "done should have final response")
			require.NotNil(t, event.FinalResponse.Usage, "final response should have usage")
			assert.Greater(t, event.FinalResponse.Usage.TotalTokens, 0)
			return
		}
	}
	t.Fatal("stream ended without done event")
}
