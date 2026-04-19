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

package llm_provider_zoltai

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_dto"
)

func TestZoltaiProvider_Close_Idempotent(t *testing.T) {
	provider, err := newProvider(Config{})
	require.NoError(t, err)

	require.NoError(t, provider.Close(context.Background()))
	require.NoError(t, provider.Close(context.Background()))
	require.NoError(t, provider.Close(context.Background()))
}

func TestZoltaiProvider_Close_HasCloseContext(t *testing.T) {
	provider, err := newProvider(Config{})
	require.NoError(t, err)

	require.NotNil(t, provider.closeContext, "close context should be initialised")
	require.NotNil(t, provider.closeCancel, "close cancel should be initialised")

	select {
	case <-provider.closeContext.Done():
		t.Fatal("close context should not be cancelled before Close")
	default:
	}

	require.NoError(t, provider.Close(context.Background()))

	select {
	case <-provider.closeContext.Done():
	default:
		t.Fatal("close context should be cancelled after Close")
	}
}

func TestZoltaiProvider_Close_DrainsActiveStream(t *testing.T) {
	provider, err := newProvider(Config{Seed: 42})
	require.NoError(t, err)

	streamChannel, err := provider.Stream(context.Background(), &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
	})
	require.NoError(t, err)

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- provider.Close(context.Background())
	}()

	for range streamChannel {
	}

	select {
	case closeErr := <-closeDone:
		assert.NoError(t, closeErr)
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not return after stream completed")
	}
}

func TestZoltaiProvider_Close_DrainsActiveToolStream(t *testing.T) {
	provider, err := newProvider(Config{Seed: 42})
	require.NoError(t, err)

	streamChannel, err := provider.Stream(context.Background(), &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
		Tools: []llm_dto.ToolDefinition{
			llm_dto.NewFunctionTool("noop", "Do nothing", nil),
		},
	})
	require.NoError(t, err)

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- provider.Close(context.Background())
	}()

	for range streamChannel {
	}

	select {
	case closeErr := <-closeDone:
		assert.NoError(t, closeErr)
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not return after tool stream completed")
	}
}
