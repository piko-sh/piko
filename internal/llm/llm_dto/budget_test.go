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

package llm_dto

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/wdk/maths"
)

func TestBudgetConfig_HasTotalSpendLimit(t *testing.T) {
	t.Parallel()

	t.Run("zero value", func(t *testing.T) {
		t.Parallel()

		config := &BudgetConfig{}
		assert.False(t, config.HasTotalSpendLimit())
	})

	t.Run("non-zero", func(t *testing.T) {
		t.Parallel()

		config := &BudgetConfig{
			MaxTotalSpend: maths.NewMoneyFromString("100.00", "USD"),
		}
		assert.True(t, config.HasTotalSpendLimit())
	})
}

func TestBudgetConfig_HasDailySpendLimit(t *testing.T) {
	t.Parallel()

	t.Run("zero value", func(t *testing.T) {
		t.Parallel()

		config := &BudgetConfig{}
		assert.False(t, config.HasDailySpendLimit())
	})

	t.Run("non-zero", func(t *testing.T) {
		t.Parallel()

		config := &BudgetConfig{
			MaxDailySpend: maths.NewMoneyFromString("50.00", "USD"),
		}
		assert.True(t, config.HasDailySpendLimit())
	})
}

func TestBudgetConfig_HasHourlySpendLimit(t *testing.T) {
	t.Parallel()

	t.Run("zero value", func(t *testing.T) {
		t.Parallel()

		config := &BudgetConfig{}
		assert.False(t, config.HasHourlySpendLimit())
	})

	t.Run("non-zero", func(t *testing.T) {
		t.Parallel()

		config := &BudgetConfig{
			MaxHourlySpend: maths.NewMoneyFromString("10.00", "USD"),
		}
		assert.True(t, config.HasHourlySpendLimit())
	})
}

func TestBudgetConfig_HasPerRequestLimit(t *testing.T) {
	t.Parallel()

	t.Run("zero value", func(t *testing.T) {
		t.Parallel()

		config := &BudgetConfig{}
		assert.False(t, config.HasPerRequestLimit())
	})

	t.Run("non-zero", func(t *testing.T) {
		t.Parallel()

		config := &BudgetConfig{
			MaxCostPerRequest: maths.NewMoneyFromString("1.00", "USD"),
		}
		assert.True(t, config.HasPerRequestLimit())
	})
}

func TestBudgetConfig_HasAlertThreshold(t *testing.T) {
	t.Parallel()

	t.Run("zero threshold", func(t *testing.T) {
		t.Parallel()

		config := &BudgetConfig{}
		assert.False(t, config.HasAlertThreshold())
	})

	t.Run("positive threshold but nil handler", func(t *testing.T) {
		t.Parallel()

		config := &BudgetConfig{
			AlertThreshold: 0.8,
		}
		assert.False(t, config.HasAlertThreshold())
	})

	t.Run("positive threshold with handler", func(t *testing.T) {
		t.Parallel()

		config := &BudgetConfig{
			AlertThreshold: 0.8,
			OnAlert:        func(status BudgetStatus) {},
		}
		assert.True(t, config.HasAlertThreshold())
	})
}
