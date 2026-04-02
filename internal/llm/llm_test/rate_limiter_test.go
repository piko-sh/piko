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

	"piko.sh/piko/internal/llm/llm_domain"
)

func TestRateLimit_RequestsPerMinute(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()
	h.mockProvider.SetResponse(makeResponse("ok", 10, 5))

	h.rateLimiter.SetLimits("rl-scope", 3, 0)

	var successCount int
	for range 5 {
		_, err := h.service.NewCompletion().
			Model("test-model").
			User("Rate limited request").
			BudgetScope("rl-scope").
			Do(ctx)
		if err != nil {
			assert.ErrorIs(t, err, llm_domain.ErrRateLimited,
				"should fail with rate limited, got: %v", err)
			break
		}
		successCount++
	}

	assert.Equal(t, 3, successCount, "exactly 3 requests should succeed with 3 RPM limit")
}

func TestRateLimit_TokenRefill(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()
	h.mockProvider.SetResponse(makeResponse("ok", 10, 5))

	h.rateLimiter.SetLimits("refill-scope", 2, 0)

	for range 2 {
		_, err := h.service.NewCompletion().
			Model("test-model").
			User("Exhaust limit").
			BudgetScope("refill-scope").
			Do(ctx)
		require.NoError(t, err)
	}

	_, err := h.service.NewCompletion().
		Model("test-model").
		User("Should fail").
		BudgetScope("refill-scope").
		Do(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, llm_domain.ErrRateLimited)

	h.clock.Advance(time.Minute)

	_, err = h.service.NewCompletion().
		Model("test-model").
		User("Should succeed after refill").
		BudgetScope("refill-scope").
		Do(ctx)
	require.NoError(t, err, "request should succeed after token refill")
}

func TestRateLimit_NoScope_Unlimited(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()
	h.mockProvider.SetResponse(makeResponse("unlimited", 10, 5))

	h.rateLimiter.SetLimits("tight-scope", 1, 0)

	for i := range 5 {
		_, err := h.service.NewCompletion().
			Model("test-model").
			User("No scope request").
			Do(ctx)
		require.NoError(t, err, "request %d without scope should not be rate limited", i)
	}
}

func TestRateLimit_MultipleScopes(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()
	h.mockProvider.SetResponse(makeResponse("scoped", 10, 5))

	h.rateLimiter.SetLimits("scope-x", 2, 0)
	h.rateLimiter.SetLimits("scope-y", 3, 0)

	for range 2 {
		_, err := h.service.NewCompletion().
			Model("test-model").
			User("Scope X request").
			BudgetScope("scope-x").
			Do(ctx)
		require.NoError(t, err)
	}

	_, err := h.service.NewCompletion().
		Model("test-model").
		User("Scope X overflow").
		BudgetScope("scope-x").
		Do(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, llm_domain.ErrRateLimited)

	_, err = h.service.NewCompletion().
		Model("test-model").
		User("Scope Y request").
		BudgetScope("scope-y").
		Do(ctx)
	require.NoError(t, err, "scope-y should be unaffected by scope-x exhaustion")
}
