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
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_dto"
)

func TestToolLoop_BasicCompletion(t *testing.T) {
	skipIfNoToolSupport(t)

	service, ctx := createLLMService(t)

	var callCount atomic.Int64

	response, err := service.NewCompletion().
		Model(globalEnv.toolModel).
		System("You are a helpful assistant with access to tools. Use the add tool to answer arithmetic questions.").
		User("What is 2 + 3?").
		ToolFunc("add", "Add two numbers together and return the sum", &llm_dto.JSONSchema{
			Type: "object",
			Properties: map[string]*llm_dto.JSONSchema{
				"a": new(llm_dto.IntegerSchema()),
				"b": new(llm_dto.IntegerSchema()),
			},
			Required: []string{"a", "b"},
		}, func(ctx context.Context, arguments string) (string, error) {
			callCount.Add(1)

			var params struct {
				A int `json:"a"`
				B int `json:"b"`
			}
			if err := json.Unmarshal([]byte(arguments), &params); err != nil {
				return "", fmt.Errorf("parsing arguments: %w", err)
			}

			return fmt.Sprintf("%d", params.A+params.B), nil
		}).
		MaxTokens(500).
		Do(ctx)
	require.NoError(t, err, "tool loop completion")
	require.NotNil(t, response)

	content := response.Content()
	t.Logf("Response: %s", content)
	t.Logf("Add tool called %d time(s)", callCount.Load())

	assert.NotEmpty(t, content, "expected non-empty response")
	assert.GreaterOrEqual(t, callCount.Load(), int64(1), "expected add tool to be called at least once")
}

func TestToolLoop_MultipleTools(t *testing.T) {
	skipIfNoToolSupport(t)

	service, ctx := createLLMService(t)

	var addCount atomic.Int64
	var mulCount atomic.Int64

	response, err := service.NewCompletion().
		Model(globalEnv.toolModel).
		System("You are a helpful assistant with access to tools. You MUST use the add tool and the multiply tool to answer the question. First add 2 and 3, then multiply that result by 4.").
		User("First add 2 and 3, then multiply the result by 4. What do you get?").
		ToolFunc("add", "Add two numbers together and return the sum", &llm_dto.JSONSchema{
			Type: "object",
			Properties: map[string]*llm_dto.JSONSchema{
				"a": new(llm_dto.IntegerSchema()),
				"b": new(llm_dto.IntegerSchema()),
			},
			Required: []string{"a", "b"},
		}, func(ctx context.Context, arguments string) (string, error) {
			addCount.Add(1)

			var params struct {
				A int `json:"a"`
				B int `json:"b"`
			}
			if err := json.Unmarshal([]byte(arguments), &params); err != nil {
				return "", fmt.Errorf("parsing arguments: %w", err)
			}

			return fmt.Sprintf("%d", params.A+params.B), nil
		}).
		ToolFunc("multiply", "Multiply two numbers together and return the product", &llm_dto.JSONSchema{
			Type: "object",
			Properties: map[string]*llm_dto.JSONSchema{
				"a": new(llm_dto.IntegerSchema()),
				"b": new(llm_dto.IntegerSchema()),
			},
			Required: []string{"a", "b"},
		}, func(ctx context.Context, arguments string) (string, error) {
			mulCount.Add(1)

			var params struct {
				A int `json:"a"`
				B int `json:"b"`
			}
			if err := json.Unmarshal([]byte(arguments), &params); err != nil {
				return "", fmt.Errorf("parsing arguments: %w", err)
			}

			return fmt.Sprintf("%d", params.A*params.B), nil
		}).
		MaxTokens(500).
		Do(ctx)
	require.NoError(t, err, "multi-tool completion")
	require.NotNil(t, response)

	content := response.Content()
	t.Logf("Response: %s", content)
	t.Logf("Add tool called %d time(s), multiply tool called %d time(s)", addCount.Load(), mulCount.Load())

	assert.NotEmpty(t, content, "expected non-empty response")
	assert.GreaterOrEqual(t, addCount.Load(), int64(1), "expected add tool to be called at least once")
	assert.GreaterOrEqual(t, mulCount.Load(), int64(1), "expected multiply tool to be called at least once")
}

func TestToolLoop_WithAutoRAG(t *testing.T) {

	service, ctx := createLLMService(t)
	vectorStore := createOtterVectorStore(t, 3, "cosine")
	service.SetVectorStore(vectorStore)

	err := service.AddText(ctx, "tool-loop-rag", "fact-1",
		"The Piko framework was first released in 2025 and is written in Go.")
	require.NoError(t, err, "ingesting fact-1")

	err = service.AddText(ctx, "tool-loop-rag", "fact-2",
		"Piko uses hexagonal architecture with ports and adapters for clean separation of concerns.")
	require.NoError(t, err, "ingesting fact-2")

	err = service.AddText(ctx, "tool-loop-rag", "fact-3",
		"Piko supports multiple cache providers including in-memory, Redis, and Valkey.")
	require.NoError(t, err, "ingesting fact-3")

	response, err := service.NewCompletion().
		Model(globalEnv.toolModel).
		System("You are a helpful assistant. Answer questions using ONLY the provided context documents. If the context doesn't contain the answer, say so.").
		User("What architecture does Piko use?").
		RAG("tool-loop-rag", 3).
		MaxTokens(500).
		Do(ctx)
	require.NoError(t, err, "RAG completion")
	require.NotNil(t, response)

	content := response.Content()
	t.Logf("RAG response: %s", content)
	assert.NotEmpty(t, content, "expected non-empty RAG response")
}
