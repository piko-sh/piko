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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_adapters/budget_store/memory"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/maths"
)

func TestBudget_WithinLimit(t *testing.T) {
	pricingTable := &llm_dto.PricingTable{
		LastUpdated: time.Now(),
		Models: []llm_dto.ModelPricing{
			{
				ModelID:         "zoltai-1",
				Provider:        "zoltai",
				InputCostPer1M:  maths.NewDecimalFromInt(1),
				OutputCostPer1M: maths.NewDecimalFromInt(1),
			},
		},
	}

	budgetMgr := llm_domain.NewBudgetManager(memory.New(), llm_domain.NewCostCalculatorWithPricing(pricingTable))
	service, ctx := createZoltaiService(t, llm_domain.WithPricingTable(pricingTable), llm_domain.WithBudgetManager(budgetMgr))
	service.SetBudget("test", &llm_dto.BudgetConfig{
		MaxTotalSpend: maths.NewMoneyFromDecimal(maths.NewDecimalFromInt(100), "USD"),
	})

	response, err := service.NewCompletion().
		User("Tell me a fortune").
		BudgetScope("test").
		Do(ctx)

	require.NoError(t, err)
	assert.NotEmpty(t, response.Content())
}

func TestBudget_ExceedsLimit(t *testing.T) {
	pricingTable := &llm_dto.PricingTable{
		LastUpdated: time.Now(),
		Models: []llm_dto.ModelPricing{
			{
				ModelID:         "zoltai-1",
				Provider:        "zoltai",
				InputCostPer1M:  maths.NewDecimalFromInt(1000000),
				OutputCostPer1M: maths.NewDecimalFromInt(1000000),
			},
		},
	}

	budgetMgr := llm_domain.NewBudgetManager(memory.New(), llm_domain.NewCostCalculatorWithPricing(pricingTable))
	service, ctx := createZoltaiService(t, llm_domain.WithPricingTable(pricingTable), llm_domain.WithBudgetManager(budgetMgr))
	service.SetBudget("test", &llm_dto.BudgetConfig{
		MaxTotalSpend: maths.NewMoneyFromDecimal(maths.NewDecimalFromInt(10), "USD"),
	})

	response, err := service.NewCompletion().
		User("First request to blow budget").
		BudgetScope("test").
		Do(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, response.Content())

	_, err = service.NewCompletion().
		User("Second request should exceed budget").
		BudgetScope("test").
		Do(ctx)

	assert.ErrorIs(t, err, llm_domain.ErrBudgetExceeded)
}

func TestBudget_PerRequestMaxCost(t *testing.T) {
	pricingTable := &llm_dto.PricingTable{
		LastUpdated: time.Now(),
		Models: []llm_dto.ModelPricing{
			{
				ModelID:         "zoltai-1",
				Provider:        "zoltai",
				InputCostPer1M:  maths.NewDecimalFromInt(1000000),
				OutputCostPer1M: maths.NewDecimalFromInt(1000000),
			},
		},
	}

	budgetMgr := llm_domain.NewBudgetManager(memory.New(), llm_domain.NewCostCalculatorWithPricing(pricingTable))
	service, ctx := createZoltaiService(t, llm_domain.WithPricingTable(pricingTable), llm_domain.WithBudgetManager(budgetMgr))

	_, err := service.NewCompletion().
		User("This should be too expensive").
		MaxCost(maths.NewMoneyFromDecimal(maths.NewDecimalFromString("0.0001"), "USD")).
		Do(ctx)

	assert.ErrorIs(t, err, llm_domain.ErrMaxCostExceeded)
}
