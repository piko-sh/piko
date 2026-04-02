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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/maths"
)

func TestBudget_Exhaustion_TotalSpend(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()

	h.service.SetBudget("total-scope", &llm_dto.BudgetConfig{
		MaxTotalSpend: maths.NewMoneyFromString("0.005", "USD"),
	})
	h.mockProvider.SetResponse(makeResponse("spend", 1000, 500))

	var successCount int
	for range 10 {
		_, err := h.service.NewCompletion().
			Model("test-model").
			User("Spend money").
			BudgetScope("total-scope").
			Do(ctx)
		if err != nil {
			assert.ErrorIs(t, err, llm_domain.ErrBudgetExceeded,
				"should fail with budget exceeded, got: %v", err)
			break
		}
		successCount++
	}

	assert.Greater(t, successCount, 0, "at least one request should succeed before budget exhaustion")
	assert.Less(t, successCount, 10, "budget should be exhausted before 10 requests")
}

func TestBudget_PerRequestLimit(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()

	h.service.SetBudget("per-request-scope", &llm_dto.BudgetConfig{
		MaxCostPerRequest: maths.NewMoneyFromString("0.0001", "USD"),
	})
	h.mockProvider.SetResponse(makeResponse("expensive", 1000, 500))

	_, err := h.service.NewCompletion().
		Model("test-model").
		User(strings.Repeat("x", 2000)).
		BudgetScope("per-request-scope").
		Do(ctx)

	require.Error(t, err)
	assert.ErrorIs(t, err, llm_domain.ErrMaxCostExceeded)
}

func TestBudget_AlertThreshold(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()

	alertCh := make(chan llm_dto.BudgetStatus, 10)

	h.service.SetBudget("alert-scope", &llm_dto.BudgetConfig{
		MaxTotalSpend:  maths.NewMoneyFromString("0.01", "USD"),
		AlertThreshold: 0.5,
		OnAlert: func(status llm_dto.BudgetStatus) {
			alertCh <- status
		},
	})
	h.mockProvider.SetResponse(makeResponse("alert", 1000, 500))

	for range 10 {
		_, err := h.service.NewCompletion().
			Model("test-model").
			User("Alert test").
			BudgetScope("alert-scope").
			Do(ctx)
		if err != nil {
			break
		}
	}

	select {
	case <-alertCh:

	case <-time.After(time.Second):
		t.Fatal("alert should have fired within timeout")
	}

	select {
	case <-alertCh:
		t.Fatal("alert should fire exactly once, but fired again")
	case <-time.After(50 * time.Millisecond):

	}
}

func TestBudget_MultiScope(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()
	h.mockProvider.SetResponse(makeResponse("scoped", 500, 250))

	h.service.SetBudget("scope-a", &llm_dto.BudgetConfig{
		MaxTotalSpend: maths.NewMoneyFromString("0.01", "USD"),
	})
	h.service.SetBudget("scope-b", &llm_dto.BudgetConfig{
		MaxTotalSpend: maths.NewMoneyFromString("0.01", "USD"),
	})

	_, err := h.service.NewCompletion().
		Model("test-model").
		User("Scope A request").
		BudgetScope("scope-a").
		Do(ctx)
	require.NoError(t, err)

	statusB, err := h.service.GetBudgetStatus(ctx, "scope-b")
	require.NoError(t, err)
	assert.Equal(t, int64(0), statusB.RequestCount, "scope-b should have zero requests")

	statusA, err := h.service.GetBudgetStatus(ctx, "scope-a")
	require.NoError(t, err)
	assert.Equal(t, int64(1), statusA.RequestCount, "scope-a should have one request")
}
