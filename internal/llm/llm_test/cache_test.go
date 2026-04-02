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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_dto"
)

func TestCache_Hit_SameRequest(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	h.mockProvider.SetResponse(makeResponse("cached", 10, 5))
	ctx := context.Background()

	resp1, err := h.service.NewCompletion().
		Model("test-model").
		User("Hello").
		Cache(time.Hour).
		Do(ctx)
	require.NoError(t, err)
	assert.Equal(t, "cached", resp1.Choices[0].Message.Content)

	h.mockProvider.SetResponse(makeResponse("should not see this", 10, 5))
	resp2, err := h.service.NewCompletion().
		Model("test-model").
		User("Hello").
		Cache(time.Hour).
		Do(ctx)
	require.NoError(t, err)
	assert.Equal(t, "cached", resp2.Choices[0].Message.Content, "should return cached response")

	assert.Len(t, h.mockProvider.GetCompleteCalls(), 1)
}

func TestCache_Miss_DifferentRequest(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()

	h.mockProvider.SetResponse(makeResponse("reply-a", 10, 5))
	_, err := h.service.NewCompletion().
		Model("test-model").
		User("Question A").
		Cache(time.Hour).
		Do(ctx)
	require.NoError(t, err)

	h.mockProvider.SetResponse(makeResponse("reply-b", 10, 5))
	resp2, err := h.service.NewCompletion().
		Model("test-model").
		User("Question B").
		Cache(time.Hour).
		Do(ctx)
	require.NoError(t, err)
	assert.Equal(t, "reply-b", resp2.Choices[0].Message.Content)

	assert.Len(t, h.mockProvider.GetCompleteCalls(), 2)
}

func TestCache_TTLExpiry(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	h.mockProvider.SetResponse(makeResponse("fresh", 10, 5))
	ctx := context.Background()

	_, err := h.service.NewCompletion().
		Model("test-model").
		User("Cached query").
		Cache(time.Hour).
		Do(ctx)
	require.NoError(t, err)
	assert.Len(t, h.mockProvider.GetCompleteCalls(), 1)

	h.clock.Advance(2 * time.Hour)

	h.mockProvider.SetResponse(makeResponse("refreshed", 10, 5))
	response, err := h.service.NewCompletion().
		Model("test-model").
		User("Cached query").
		Cache(time.Hour).
		Do(ctx)
	require.NoError(t, err)
	assert.Equal(t, "refreshed", response.Choices[0].Message.Content)
	assert.Len(t, h.mockProvider.GetCompleteCalls(), 2)
}

func TestCache_SkipRead(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()

	h.mockProvider.SetResponse(makeResponse("original", 10, 5))
	_, err := h.service.NewCompletion().
		Model("test-model").
		User("Query").
		Cache(time.Hour).
		Do(ctx)
	require.NoError(t, err)

	h.mockProvider.SetResponse(makeResponse("updated", 10, 5))
	response, err := h.service.NewCompletion().
		Model("test-model").
		User("Query").
		CacheConfig(&llm_dto.CacheConfig{
			Enabled:  true,
			TTL:      time.Hour,
			SkipRead: true,
		}).
		Do(ctx)
	require.NoError(t, err)
	assert.Equal(t, "updated", response.Choices[0].Message.Content)

	assert.Len(t, h.mockProvider.GetCompleteCalls(), 2)
}

func TestCache_SkipWrite(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()

	h.mockProvider.SetResponse(makeResponse("uncached", 10, 5))
	_, err := h.service.NewCompletion().
		Model("test-model").
		User("Ephemeral query").
		CacheConfig(&llm_dto.CacheConfig{
			Enabled:   true,
			TTL:       time.Hour,
			SkipWrite: true,
		}).
		Do(ctx)
	require.NoError(t, err)

	h.mockProvider.SetResponse(makeResponse("also uncached", 10, 5))
	response, err := h.service.NewCompletion().
		Model("test-model").
		User("Ephemeral query").
		Cache(time.Hour).
		Do(ctx)
	require.NoError(t, err)
	assert.Equal(t, "also uncached", response.Choices[0].Message.Content)

	assert.Len(t, h.mockProvider.GetCompleteCalls(), 2)
}

func TestCache_CustomKey(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()

	h.mockProvider.SetResponse(makeResponse("keyed-response", 10, 5))
	_, err := h.service.NewCompletion().
		Model("test-model").
		User("First question").
		CacheConfig(&llm_dto.CacheConfig{
			Enabled: true,
			TTL:     time.Hour,
			Key:     "my-custom-key-for-testing",
		}).
		Do(ctx)
	require.NoError(t, err)

	h.mockProvider.SetResponse(makeResponse("different-response", 10, 5))
	response, err := h.service.NewCompletion().
		Model("test-model").
		User("Completely different question").
		CacheConfig(&llm_dto.CacheConfig{
			Enabled: true,
			TTL:     time.Hour,
			Key:     "my-custom-key-for-testing",
		}).
		Do(ctx)
	require.NoError(t, err)
	assert.Equal(t, "keyed-response", response.Choices[0].Message.Content, "same custom key should hit cache")
	assert.Len(t, h.mockProvider.GetCompleteCalls(), 1)
}
