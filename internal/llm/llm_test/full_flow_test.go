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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/maths"
)

func TestLLM_FullFlow_RAG_Budget_Cache(t *testing.T) {
	h := newTestHarness(t)
	ctx := context.Background()

	h.service.SetBudget("user:123", &llm_dto.BudgetConfig{
		MaxTotalSpend: maths.NewMoneyFromFloat(0.10, llm_dto.CostCurrency),
		MaxDailySpend: maths.NewMoneyFromFloat(0.05, llm_dto.CostCurrency),
	})

	results := []llm_dto.VectorSearchResult{
		{ID: "doc-1", Content: "Piko is a Go framework.", Score: 0.95},
	}

	h.mockProvider.SetResponse(makeResponse("Piko is indeed a Go framework.", 50, 20))

	resp1, err := h.service.NewCompletion().
		Model("test-model").
		User("What is Piko?").
		WithVectorContext(results).
		BudgetScope("user:123").
		Cache(time.Hour).
		Do(ctx)

	require.NoError(t, err)
	assert.Contains(t, resp1.Choices[0].Message.Content, "Go framework")

	calls := h.mockProvider.GetCompleteCalls()
	require.Len(t, calls, 1)
	assert.Contains(t, calls[0].Messages[0].Content, "Piko is a Go framework.")

	status, err := h.service.GetBudgetStatus(ctx, "user:123")
	require.NoError(t, err)
	assert.True(t, status.TotalSpent.CheckGreaterThan(maths.ZeroMoney(llm_dto.CostCurrency)))
	assert.Equal(t, int64(1), status.RequestCount)

	h.mockProvider.SetResponse(makeResponse("This should be ignored due to cache", 10, 10))

	resp2, err := h.service.NewCompletion().
		Model("test-model").
		User("What is Piko?").
		WithVectorContext(results).
		BudgetScope("user:123").
		Cache(time.Hour).
		Do(ctx)

	require.NoError(t, err)
	assert.Equal(t, resp1.Choices[0].Message.Content, resp2.Choices[0].Message.Content, "Should match cached response")

	assert.Len(t, h.mockProvider.GetCompleteCalls(), 1, "Provider should only be called once")

	status2, _ := h.service.GetBudgetStatus(ctx, "user:123")
	assert.Equal(t, int64(1), status2.RequestCount, "Cache hit should not increment request count in budget")
}

func TestLLM_FullFlow_BudgetExceeded(t *testing.T) {
	h := newTestHarness(t)
	ctx := context.Background()

	h.service.SetBudget("broke-user", &llm_dto.BudgetConfig{
		MaxTotalSpend: maths.NewMoneyFromFloat(0.000001, llm_dto.CostCurrency),
	})

	_, err := h.service.NewCompletion().
		Model("test-model").
		User("Tell me a very long story about why I have no money.").
		BudgetScope("broke-user").
		Do(ctx)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, llm_domain.ErrBudgetExceeded), "error should be ErrBudgetExceeded")
	assert.Len(t, h.mockProvider.GetCompleteCalls(), 0, "Provider should never be reached")
}
