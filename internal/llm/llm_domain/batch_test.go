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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/llm/llm_dto"
)

func TestNewBatchBuilder(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	builder := NewBatchBuilder(service)

	require.NotNil(t, builder)
	assert.Equal(t, service, builder.service)
	assert.Empty(t, builder.requests)
	assert.Empty(t, builder.providerName)
	assert.Empty(t, builder.budgetScope)
	assert.Nil(t, builder.metadata)
	assert.Equal(t, time.Duration(0), builder.window)
}

func TestBatchBuilder_Add(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	builder := NewBatchBuilder(service)

	request := llm_dto.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
	}

	result := builder.Add(request)

	assert.Equal(t, builder, result)
	require.Len(t, builder.requests, 1)
	assert.Equal(t, "gpt-4o", builder.requests[0].Model)
}

func TestBatchBuilder_AddMultiple(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	builder := NewBatchBuilder(service)

	req1 := llm_dto.CompletionRequest{Model: "gpt-4o"}
	req2 := llm_dto.CompletionRequest{Model: "gpt-5"}

	builder.Add(req1).Add(req2)

	assert.Len(t, builder.requests, 2)
	assert.Equal(t, "gpt-4o", builder.requests[0].Model)
	assert.Equal(t, "gpt-5", builder.requests[1].Model)
}

func TestBatchBuilder_AddBuilder(t *testing.T) {
	llmService := NewService("")
	concrete, ok := llmService.(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	batchBuilder := NewBatchBuilder(concrete)

	completionBuilder := llmService.NewCompletion().
		Model("gpt-4o").
		User("Hello").
		Temperature(0.5)

	result := batchBuilder.AddBuilder(completionBuilder)

	assert.Equal(t, batchBuilder, result)
	require.Len(t, batchBuilder.requests, 1)
	assert.Equal(t, "gpt-4o", batchBuilder.requests[0].Model)
	require.NotNil(t, batchBuilder.requests[0].Temperature)
	assert.Equal(t, 0.5, *batchBuilder.requests[0].Temperature)
}

func TestBatchBuilder_Window(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	builder := NewBatchBuilder(service)

	result := builder.Window(24 * time.Hour)

	assert.Equal(t, builder, result)
	assert.Equal(t, 24*time.Hour, builder.window)
}

func TestBatchBuilder_WithProvider(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	builder := NewBatchBuilder(service)

	result := builder.WithProvider("openai")

	assert.Equal(t, builder, result)
	assert.Equal(t, "openai", builder.providerName)
}

func TestBatchBuilder_WithBudgetScope(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	builder := NewBatchBuilder(service)

	result := builder.WithBudgetScope("user:123")

	assert.Equal(t, builder, result)
	assert.Equal(t, "user:123", builder.budgetScope)
}

func TestBatchBuilder_WithMetadata(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	builder := NewBatchBuilder(service)

	t.Run("initialises map when nil", func(t *testing.T) {
		result := builder.WithMetadata("env", "production")

		assert.Equal(t, builder, result)
		require.NotNil(t, builder.metadata)
		assert.Equal(t, "production", builder.metadata["env"])
	})

	t.Run("adds to existing map", func(t *testing.T) {
		builder.WithMetadata("team", "ml")

		assert.Equal(t, "production", builder.metadata["env"])
		assert.Equal(t, "ml", builder.metadata["team"])
	})
}

func TestBatchBuilder_Build(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	builder := NewBatchBuilder(service)

	builder.
		Add(llm_dto.CompletionRequest{Model: "gpt-4o"}).
		Add(llm_dto.CompletionRequest{Model: "gpt-5"}).
		Window(24*time.Hour).
		WithMetadata("env", "test")

	request := builder.Build()

	assert.Len(t, request.Requests, 2)
	assert.Equal(t, 24*time.Hour, request.CompletionWindow)
	assert.Equal(t, "test", request.Metadata["env"])
}

func TestBatchBuilder_BuildEmpty(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	builder := NewBatchBuilder(service)

	request := builder.Build()

	assert.Empty(t, request.Requests)
	assert.Equal(t, time.Duration(0), request.CompletionWindow)
	assert.Nil(t, request.Metadata)
}

func TestBatchBuilder_FullChain(t *testing.T) {
	llmService := NewService("")
	concrete, ok := llmService.(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	batchBuilder := NewBatchBuilder(concrete)

	completionBuilder := llmService.NewCompletion().
		Model("gpt-4o").
		System("You are helpful").
		User("Hello")

	request := batchBuilder.
		AddBuilder(completionBuilder).
		Add(llm_dto.CompletionRequest{
			Model:    "gpt-5",
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hi"}},
		}).
		Window(24*time.Hour).
		WithProvider("openai").
		WithBudgetScope("team:alpha").
		WithMetadata("batch_id", "batch-001").
		Build()

	assert.Len(t, request.Requests, 2)
	assert.Equal(t, 24*time.Hour, request.CompletionWindow)
	assert.Equal(t, "batch-001", request.Metadata["batch_id"])
}

func TestNewBatchPoller(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	t.Run("creates with provided interval", func(t *testing.T) {
		poller := NewBatchPoller(service, "batch-123", "openai", 10*time.Second)

		require.NotNil(t, poller)
		assert.Equal(t, service, poller.service)
		assert.Equal(t, "batch-123", poller.batchID)
		assert.Equal(t, "openai", poller.providerName)
		assert.Equal(t, 10*time.Second, poller.interval)
	})

	t.Run("uses default interval when zero", func(t *testing.T) {
		poller := NewBatchPoller(service, "batch-456", "anthropic", 0)

		assert.Equal(t, DefaultPollingInterval, poller.interval)
	})
}

func TestBatchPoller_PollChannel(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	poller := NewBatchPoller(service, "batch-123", "openai", 100*time.Millisecond)

	ctx, cancel := context.WithCancelCause(context.Background())
	responseChannel := poller.PollChannel(ctx)
	require.NotNil(t, responseChannel)

	cancel(fmt.Errorf("test: simulating cancelled context"))

	for range responseChannel {
	}
}

func TestDefaultPollingInterval(t *testing.T) {
	assert.Equal(t, 30*time.Second, DefaultPollingInterval)
}
