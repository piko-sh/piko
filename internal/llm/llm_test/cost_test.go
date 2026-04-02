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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_dto"
)

func TestCost_RecordedAfterCompletion(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)

	h.mockProvider.SetResponse(makeResponse("costed", 100, 50))
	ctx := context.Background()

	response, err := h.service.NewCompletion().
		Model("test-model").
		User("Calculate cost").
		BudgetScope("cost-scope").
		Do(ctx)

	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Usage)
	require.NotNil(t, response.Usage.EstimatedCost, "cost estimate should be attached to response")

	assert.Equal(t, "test-model", response.Usage.EstimatedCost.Model)
	assert.Equal(t, 100, response.Usage.EstimatedCost.InputTokens)
	assert.Equal(t, 50, response.Usage.EstimatedCost.OutputTokens)

	assert.False(t, response.Usage.EstimatedCost.TotalCost.CheckIsZero(),
		"total cost should be non-zero for a known model")
}

func TestCost_BudgetStatusReflectsSpend(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	h.mockProvider.SetResponse(makeResponse("tracked", 200, 100))
	ctx := context.Background()

	h.service.SetBudget("tracked-scope", &llm_dto.BudgetConfig{})

	_, err := h.service.NewCompletion().
		Model("test-model").
		User("Track me").
		BudgetScope("tracked-scope").
		Do(ctx)
	require.NoError(t, err)

	status, err := h.service.GetBudgetStatus(ctx, "tracked-scope")
	require.NoError(t, err)
	require.NotNil(t, status)

	assert.Greater(t, status.RequestCount, int64(0), "request count should be positive")
	assert.Greater(t, status.TokenCount, int64(0), "token count should be positive")
	assert.False(t, status.TotalSpent.CheckIsZero(), "total spent should be non-zero")
}

func TestCost_UnknownModel_ZeroCost(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	h.mockProvider.SetResponse(makeResponse("unknown", 10, 5))
	ctx := context.Background()

	response, err := h.service.NewCompletion().
		Model("unknown-model").
		User("Price me").
		Do(ctx)

	require.NoError(t, err)
	require.NotNil(t, response, "request should complete even with unknown model pricing")

	if response.Usage != nil && response.Usage.EstimatedCost != nil {
		assert.True(t, response.Usage.EstimatedCost.TotalCost.CheckIsZero(),
			"cost should be zero for unknown model")
	}
}

func TestCost_MultipleCompletions_Accumulate(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()
	h.service.SetBudget("accumulate-scope", &llm_dto.BudgetConfig{})

	for range 3 {
		h.mockProvider.SetResponse(makeResponse("reply", 100, 50))
		_, err := h.service.NewCompletion().
			Model("test-model").
			User("Request").
			BudgetScope("accumulate-scope").
			Do(ctx)
		require.NoError(t, err)
	}

	status, err := h.service.GetBudgetStatus(ctx, "accumulate-scope")
	require.NoError(t, err)
	require.NotNil(t, status)

	assert.Equal(t, int64(3), status.RequestCount, "should reflect 3 requests")

	assert.GreaterOrEqual(t, status.TokenCount, int64(450), "should accumulate tokens across requests")
}
