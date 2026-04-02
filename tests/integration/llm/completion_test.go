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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_dto"
)

func TestComplete_BasicPrompt(t *testing.T) {

	handle, ctx := createOllamaProvider(t)

	response, err := handle.llm.Complete(ctx, &llm_dto.CompletionRequest{
		Model: globalEnv.completionModel,
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Reply with only the word: hello"},
		},
		MaxTokens: new(500),
	})
	require.NoError(t, err, "completion request")

	require.NotNil(t, response)
	require.NotEmpty(t, response.Choices, "expected at least one choice")
	assert.NotEmpty(t, response.Choices[0].Message.Content, "expected non-empty content")
	assert.Equal(t, llm_dto.RoleAssistant, response.Choices[0].Message.Role)
	assert.Equal(t, globalEnv.completionModel, response.Model)
	assert.NotNil(t, response.Usage, "expected usage stats")
}

func TestComplete_SystemAndUserMessages(t *testing.T) {

	handle, ctx := createOllamaProvider(t)

	response, err := handle.llm.Complete(ctx, &llm_dto.CompletionRequest{
		Model: globalEnv.completionModel,
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleSystem, Content: "You are a calculator. Reply with only the numeric result."},
			{Role: llm_dto.RoleUser, Content: "What is 2+2?"},
		},
		MaxTokens: new(500),
	})
	require.NoError(t, err, "completion with system message")

	require.NotNil(t, response)
	require.NotEmpty(t, response.Choices)
	assert.NotEmpty(t, response.Choices[0].Message.Content)
}

func TestComplete_WithTemperature(t *testing.T) {

	handle, ctx := createOllamaProvider(t)

	response, err := handle.llm.Complete(ctx, &llm_dto.CompletionRequest{
		Model: globalEnv.completionModel,
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Say the word 'test'"},
		},
		MaxTokens:   new(50),
		Temperature: new(0.1),
	})
	require.NoError(t, err, "completion with temperature")

	require.NotNil(t, response)
	require.NotEmpty(t, response.Choices)
}

func TestComplete_UsageTokens(t *testing.T) {

	handle, ctx := createOllamaProvider(t)

	response, err := handle.llm.Complete(ctx, &llm_dto.CompletionRequest{
		Model: globalEnv.completionModel,
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hi"},
		},
		MaxTokens: new(50),
	})
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Usage)

	assert.Greater(t, response.Usage.PromptTokens, 0, "expected prompt tokens > 0")
	assert.Greater(t, response.Usage.CompletionTokens, 0, "expected completion tokens > 0")
	assert.Equal(t, response.Usage.PromptTokens+response.Usage.CompletionTokens, response.Usage.TotalTokens)
}
