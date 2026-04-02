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

func TestModelPricing_HasCachedPricing(t *testing.T) {
	t.Parallel()

	t.Run("with cached pricing", func(t *testing.T) {
		t.Parallel()

		p := ModelPricing{
			CachedInputPer1M: maths.NewDecimalFromString("1.50"),
		}
		assert.True(t, p.HasCachedPricing())
	})

	t.Run("without cached pricing", func(t *testing.T) {
		t.Parallel()

		p := ModelPricing{}
		assert.False(t, p.HasCachedPricing())
	})
}

func TestPricingTable_GetPricing(t *testing.T) {
	t.Parallel()

	table := &PricingTable{
		Models: []ModelPricing{
			{ModelID: "gpt-5", Provider: "openai", InputCostPer1M: maths.NewDecimalFromString("3.00")},
			{ModelID: "claude-sonnet", Provider: "anthropic", InputCostPer1M: maths.NewDecimalFromString("2.00")},
		},
	}

	t.Run("found", func(t *testing.T) {
		t.Parallel()

		result := table.GetPricing("gpt-5")
		assert.NotNil(t, result)
		assert.Equal(t, "gpt-5", result.ModelID)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		result := table.GetPricing("nonexistent")
		assert.Nil(t, result)
	})

	t.Run("nil table", func(t *testing.T) {
		t.Parallel()

		var nilTable *PricingTable
		assert.Nil(t, nilTable.GetPricing("gpt-5"))
	})
}

func TestPricingTable_GetPricingByProvider(t *testing.T) {
	t.Parallel()

	table := &PricingTable{
		Models: []ModelPricing{
			{ModelID: "gpt-5", Provider: "openai"},
			{ModelID: "gpt-5-mini", Provider: "openai"},
			{ModelID: "claude-sonnet", Provider: "anthropic"},
		},
	}

	t.Run("found multiple", func(t *testing.T) {
		t.Parallel()

		result := table.GetPricingByProvider("openai")
		assert.Len(t, result, 2)
	})

	t.Run("found single", func(t *testing.T) {
		t.Parallel()

		result := table.GetPricingByProvider("anthropic")
		assert.Len(t, result, 1)
		assert.Equal(t, "claude-sonnet", result[0].ModelID)
	})

	t.Run("none found", func(t *testing.T) {
		t.Parallel()

		result := table.GetPricingByProvider("unknown")
		assert.Nil(t, result)
	})

	t.Run("nil table", func(t *testing.T) {
		t.Parallel()

		var nilTable *PricingTable
		assert.Nil(t, nilTable.GetPricingByProvider("openai"))
	})
}
