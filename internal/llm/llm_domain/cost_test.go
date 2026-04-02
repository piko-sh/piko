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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/maths"
)

func TestNewCostCalculator(t *testing.T) {
	t.Run("creates calculator with default pricing table", func(t *testing.T) {
		calc := NewCostCalculator()
		require.NotNil(t, calc)

		pricing := calc.GetPricing("gpt-5")
		require.NotNil(t, pricing)
		assert.Equal(t, "gpt-5", pricing.ModelID)
		assert.Equal(t, "openai", pricing.Provider)
	})

	t.Run("accepts functional options", func(t *testing.T) {
		mockClock := clock.NewMockClock(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC))
		calc := NewCostCalculator(WithCostCalculatorClock(mockClock))
		require.NotNil(t, calc)

		clockFromCalc := GetCostCalculatorClock(calc)
		assert.Equal(t, mockClock, clockFromCalc)
	})
}

func TestNewCostCalculatorWithPricing(t *testing.T) {
	t.Run("uses custom pricing table", func(t *testing.T) {
		customTable := &llm_dto.PricingTable{
			Models: []llm_dto.ModelPricing{
				{
					ModelID:          "custom-model",
					Provider:         "custom-provider",
					InputCostPer1M:   maths.NewDecimalFromString("1.00").Must(),
					OutputCostPer1M:  maths.NewDecimalFromString("2.00").Must(),
					CachedInputPer1M: maths.ZeroDecimal(),
				},
			},
			LastUpdated: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		}

		calc := NewCostCalculatorWithPricing(customTable)
		require.NotNil(t, calc)

		pricing := calc.GetPricing("custom-model")
		require.NotNil(t, pricing)
		assert.Equal(t, "custom-model", pricing.ModelID)

		defaultPricing := calc.GetPricing("gpt-5")
		assert.Nil(t, defaultPricing)
	})

	t.Run("accepts functional options", func(t *testing.T) {
		mockClock := clock.NewMockClock(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC))
		customTable := &llm_dto.PricingTable{
			Models: []llm_dto.ModelPricing{},
		}

		calc := NewCostCalculatorWithPricing(customTable, WithCostCalculatorClock(mockClock))
		require.NotNil(t, calc)

		clockFromCalc := GetCostCalculatorClock(calc)
		assert.Equal(t, mockClock, clockFromCalc)
	})
}

func TestCostCalculator_Calculate(t *testing.T) {
	testCases := []struct {
		usage          *llm_dto.Usage
		pricingTable   *llm_dto.PricingTable
		name           string
		model          string
		provider       string
		wantInputCost  string
		wantOutputCost string
		wantTotalCost  string
		wantNil        bool
	}{
		{
			name:     "nil usage returns nil",
			model:    "gpt-5",
			provider: "openai",
			usage:    nil,
			wantNil:  true,
		},
		{
			name:     "unknown model returns zero costs",
			model:    "unknown-model",
			provider: "unknown-provider",
			usage: &llm_dto.Usage{
				PromptTokens:     1000,
				CompletionTokens: 500,
				TotalTokens:      1500,
			},
			wantInputCost:  "0",
			wantOutputCost: "0",
			wantTotalCost:  "0",
		},
		{
			name:     "calculates cost for known model",
			model:    "gpt-5",
			provider: "openai",
			usage: &llm_dto.Usage{
				PromptTokens:     1_000_000,
				CompletionTokens: 500_000,
				TotalTokens:      1_500_000,
			},

			wantInputCost:  "1.25",
			wantOutputCost: "5",
			wantTotalCost:  "6.25",
		},
		{
			name:     "calculates cost for small token count",
			model:    "gpt-5",
			provider: "openai",
			usage: &llm_dto.Usage{
				PromptTokens:     100,
				CompletionTokens: 50,
				TotalTokens:      150,
			},

			wantInputCost:  "0.000125",
			wantOutputCost: "0.0005",
			wantTotalCost:  "0.000625",
		},
		{
			name:     "calculates cost for anthropic model",
			model:    "claude-sonnet-4-5-20250929",
			provider: "anthropic",
			usage: &llm_dto.Usage{
				PromptTokens:     1_000_000,
				CompletionTokens: 1_000_000,
				TotalTokens:      2_000_000,
			},

			wantInputCost:  "3",
			wantOutputCost: "15",
			wantTotalCost:  "18",
		},
		{
			name:     "calculates cost for gemini model",
			model:    "gemini-2.5-pro",
			provider: "gemini",
			usage: &llm_dto.Usage{
				PromptTokens:     1_000_000,
				CompletionTokens: 1_000_000,
				TotalTokens:      2_000_000,
			},

			wantInputCost:  "1.25",
			wantOutputCost: "10",
			wantTotalCost:  "11.25",
		},
		{
			name:     "handles zero tokens",
			model:    "gpt-5",
			provider: "openai",
			usage: &llm_dto.Usage{
				PromptTokens:     0,
				CompletionTokens: 0,
				TotalTokens:      0,
			},
			wantInputCost:  "0",
			wantOutputCost: "0",
			wantTotalCost:  "0",
		},
		{
			name:     "uses custom pricing table",
			model:    "test-model",
			provider: "test-provider",
			usage: &llm_dto.Usage{
				PromptTokens:     1_000_000,
				CompletionTokens: 1_000_000,
				TotalTokens:      2_000_000,
			},
			pricingTable: &llm_dto.PricingTable{
				Models: []llm_dto.ModelPricing{
					{
						ModelID:          "test-model",
						Provider:         "test-provider",
						InputCostPer1M:   maths.NewDecimalFromString("5.00").Must(),
						OutputCostPer1M:  maths.NewDecimalFromString("10.00").Must(),
						CachedInputPer1M: maths.ZeroDecimal(),
					},
				},
			},
			wantInputCost:  "5",
			wantOutputCost: "10",
			wantTotalCost:  "15",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var calc *CostCalculator
			if tc.pricingTable != nil {
				calc = NewCostCalculatorWithPricing(tc.pricingTable)
			} else {
				calc = NewCostCalculator()
			}

			result := calc.Calculate(tc.model, tc.provider, tc.usage)

			if tc.wantNil {
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			assert.Equal(t, tc.model, result.Model)
			assert.Equal(t, tc.provider, result.Provider)

			if tc.usage != nil {
				assert.Equal(t, tc.usage.PromptTokens, result.InputTokens)
				assert.Equal(t, tc.usage.CompletionTokens, result.OutputTokens)
				assert.Equal(t, tc.usage.TotalTokens, result.TotalTokens)
			}

			expectedInputCost := maths.NewMoneyFromString(tc.wantInputCost, llm_dto.CostCurrency)
			expectedOutputCost := maths.NewMoneyFromString(tc.wantOutputCost, llm_dto.CostCurrency)
			expectedTotalCost := maths.NewMoneyFromString(tc.wantTotalCost, llm_dto.CostCurrency)

			assert.True(t, result.InputCost.MustEquals(expectedInputCost),
				"InputCost: got %s, want %s", result.InputCost.MustNumber(), tc.wantInputCost)
			assert.True(t, result.OutputCost.MustEquals(expectedOutputCost),
				"OutputCost: got %s, want %s", result.OutputCost.MustNumber(), tc.wantOutputCost)
			assert.True(t, result.TotalCost.MustEquals(expectedTotalCost),
				"TotalCost: got %s, want %s", result.TotalCost.MustNumber(), tc.wantTotalCost)
		})
	}
}

func TestCostCalculator_Calculate_UsesClockForTimestamp(t *testing.T) {
	fixedTime := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(fixedTime)

	calc := NewCostCalculator(WithCostCalculatorClock(mockClock))
	usage := &llm_dto.Usage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	result := calc.Calculate("gpt-5", "openai", usage)
	require.NotNil(t, result)
	assert.Equal(t, fixedTime, result.Timestamp)
}

func TestCostCalculator_Calculate_WithCachedTokens(t *testing.T) {
	table := &llm_dto.PricingTable{
		Models: []llm_dto.ModelPricing{
			{
				ModelID:          "test-model",
				Provider:         "test",
				InputCostPer1M:   maths.NewDecimalFromString("10.00").Must(),
				OutputCostPer1M:  maths.NewDecimalFromString("20.00").Must(),
				CachedInputPer1M: maths.NewDecimalFromString("1.00").Must(),
			},
		},
	}

	calc := NewCostCalculatorWithPricing(table)

	t.Run("split pricing with cached tokens", func(t *testing.T) {
		usage := &llm_dto.Usage{
			PromptTokens:     1_000_000,
			CompletionTokens: 500_000,
			TotalTokens:      1_500_000,
			CachedTokens:     600_000,
		}

		result := calc.Calculate("test-model", "test", usage)
		require.NotNil(t, result)

		assert.Equal(t, 400_000, result.InputTokens)
		assert.Equal(t, 600_000, result.CachedTokens)
		assert.Equal(t, 500_000, result.OutputTokens)

		expectedInput := maths.NewMoneyFromString("4.00", llm_dto.CostCurrency)
		expectedCached := maths.NewMoneyFromString("0.60", llm_dto.CostCurrency)
		expectedOutput := maths.NewMoneyFromString("10.00", llm_dto.CostCurrency)
		expectedTotal := maths.NewMoneyFromString("14.60", llm_dto.CostCurrency)

		assert.True(t, result.InputCost.MustEquals(expectedInput),
			"InputCost: got %s, want 4.00", result.InputCost.MustNumber())
		assert.True(t, result.CachedInputCost.MustEquals(expectedCached),
			"CachedInputCost: got %s, want 0.60", result.CachedInputCost.MustNumber())
		assert.True(t, result.OutputCost.MustEquals(expectedOutput),
			"OutputCost: got %s, want 10.00", result.OutputCost.MustNumber())
		assert.True(t, result.TotalCost.MustEquals(expectedTotal),
			"TotalCost: got %s, want 14.60", result.TotalCost.MustNumber())
	})

	t.Run("no cached pricing on model", func(t *testing.T) {
		noCacheTable := &llm_dto.PricingTable{
			Models: []llm_dto.ModelPricing{
				{
					ModelID:          "no-cache",
					Provider:         "test",
					InputCostPer1M:   maths.NewDecimalFromString("10.00").Must(),
					OutputCostPer1M:  maths.NewDecimalFromString("20.00").Must(),
					CachedInputPer1M: maths.ZeroDecimal(),
				},
			},
		}
		noCacheCalc := NewCostCalculatorWithPricing(noCacheTable)

		usage := &llm_dto.Usage{
			PromptTokens:     1_000_000,
			CompletionTokens: 500_000,
			TotalTokens:      1_500_000,
			CachedTokens:     600_000,
		}

		result := noCacheCalc.Calculate("no-cache", "test", usage)
		require.NotNil(t, result)

		assert.Equal(t, 400_000, result.InputTokens)
		assert.Equal(t, 600_000, result.CachedTokens)

		expectedInput := maths.NewMoneyFromString("4.00", llm_dto.CostCurrency)
		expectedCached := maths.NewMoneyFromString("0", llm_dto.CostCurrency)

		assert.True(t, result.InputCost.MustEquals(expectedInput),
			"InputCost: got %s, want 4.00", result.InputCost.MustNumber())
		assert.True(t, result.CachedInputCost.MustEquals(expectedCached),
			"CachedInputCost: got %s, want 0", result.CachedInputCost.MustNumber())
	})

	t.Run("cached tokens exceed prompt tokens clamped", func(t *testing.T) {
		usage := &llm_dto.Usage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			CachedTokens:     200,
		}

		result := calc.Calculate("test-model", "test", usage)
		require.NotNil(t, result)

		assert.Equal(t, 0, result.InputTokens)
		assert.Equal(t, 100, result.CachedTokens)
	})
}

func TestCostCalculator_GetPricing(t *testing.T) {
	testCases := []struct {
		name         string
		modelID      string
		wantProvider string
		wantFound    bool
	}{
		{
			name:         "finds gpt-5",
			modelID:      "gpt-5",
			wantFound:    true,
			wantProvider: "openai",
		},
		{
			name:         "finds claude-sonnet-4-5-20250929",
			modelID:      "claude-sonnet-4-5-20250929",
			wantFound:    true,
			wantProvider: "anthropic",
		},
		{
			name:         "finds gemini-2.5-pro",
			modelID:      "gemini-2.5-pro",
			wantFound:    true,
			wantProvider: "gemini",
		},
		{
			name:      "returns nil for unknown model",
			modelID:   "unknown-model-xyz",
			wantFound: false,
		},
		{
			name:      "returns nil for empty model ID",
			modelID:   "",
			wantFound: false,
		},
	}

	calc := NewCostCalculator()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pricing := calc.GetPricing(tc.modelID)

			if tc.wantFound {
				require.NotNil(t, pricing)
				assert.Equal(t, tc.modelID, pricing.ModelID)
				assert.Equal(t, tc.wantProvider, pricing.Provider)
			} else {
				assert.Nil(t, pricing)
			}
		})
	}
}

func TestCostCalculator_SetPricing(t *testing.T) {
	t.Run("adds new pricing", func(t *testing.T) {

		customTable := &llm_dto.PricingTable{
			Models: []llm_dto.ModelPricing{
				{
					ModelID:          "existing-model",
					Provider:         "test",
					InputCostPer1M:   maths.NewDecimalFromString("1.00").Must(),
					OutputCostPer1M:  maths.NewDecimalFromString("2.00").Must(),
					CachedInputPer1M: maths.ZeroDecimal(),
				},
			},
		}
		calc := NewCostCalculatorWithPricing(customTable)

		pricing := calc.GetPricing("new-test-model")
		assert.Nil(t, pricing)

		calc.SetPricing(llm_dto.ModelPricing{
			ModelID:          "new-test-model",
			Provider:         "test-provider",
			InputCostPer1M:   maths.NewDecimalFromString("1.50").Must(),
			OutputCostPer1M:  maths.NewDecimalFromString("3.00").Must(),
			CachedInputPer1M: maths.ZeroDecimal(),
		})

		pricing = calc.GetPricing("new-test-model")
		require.NotNil(t, pricing)
		assert.Equal(t, "new-test-model", pricing.ModelID)
		assert.Equal(t, "test-provider", pricing.Provider)
		assert.True(t, pricing.InputCostPer1M.MustEquals(maths.NewDecimalFromString("1.50").Must()))
	})

	t.Run("updates existing pricing", func(t *testing.T) {

		customTable := &llm_dto.PricingTable{
			Models: []llm_dto.ModelPricing{
				{
					ModelID:          "test-model",
					Provider:         "test",
					InputCostPer1M:   maths.NewDecimalFromString("1.00").Must(),
					OutputCostPer1M:  maths.NewDecimalFromString("2.00").Must(),
					CachedInputPer1M: maths.ZeroDecimal(),
				},
			},
		}
		calc := NewCostCalculatorWithPricing(customTable)

		originalPricing := calc.GetPricing("test-model")
		require.NotNil(t, originalPricing)
		originalInputCost := originalPricing.InputCostPer1M

		calc.SetPricing(llm_dto.ModelPricing{
			ModelID:          "test-model",
			Provider:         "test",
			InputCostPer1M:   maths.NewDecimalFromString("99.99").Must(),
			OutputCostPer1M:  maths.NewDecimalFromString("199.99").Must(),
			CachedInputPer1M: maths.ZeroDecimal(),
		})

		updatedPricing := calc.GetPricing("test-model")
		require.NotNil(t, updatedPricing)
		assert.False(t, updatedPricing.InputCostPer1M.MustEquals(originalInputCost))
		assert.True(t, updatedPricing.InputCostPer1M.MustEquals(maths.NewDecimalFromString("99.99").Must()))
	})
}

func TestCostCalculator_SetPricingTable(t *testing.T) {
	t.Run("replaces entire pricing table", func(t *testing.T) {

		initialTable := &llm_dto.PricingTable{
			Models: []llm_dto.ModelPricing{
				{
					ModelID:          "initial-model",
					Provider:         "test",
					InputCostPer1M:   maths.NewDecimalFromString("1.00").Must(),
					OutputCostPer1M:  maths.NewDecimalFromString("2.00").Must(),
					CachedInputPer1M: maths.ZeroDecimal(),
				},
			},
		}
		calc := NewCostCalculatorWithPricing(initialTable)

		assert.NotNil(t, calc.GetPricing("initial-model"))

		newTable := &llm_dto.PricingTable{
			Models: []llm_dto.ModelPricing{
				{
					ModelID:          "replacement-model",
					Provider:         "replacement-provider",
					InputCostPer1M:   maths.NewDecimalFromString("7.77").Must(),
					OutputCostPer1M:  maths.NewDecimalFromString("8.88").Must(),
					CachedInputPer1M: maths.ZeroDecimal(),
				},
			},
			LastUpdated: time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
		}
		calc.SetPricingTable(newTable)

		assert.Nil(t, calc.GetPricing("initial-model"))

		pricing := calc.GetPricing("replacement-model")
		require.NotNil(t, pricing)
		assert.Equal(t, "replacement-model", pricing.ModelID)
	})
}

func TestCostCalculator_GetPricingTable(t *testing.T) {
	t.Run("returns copy of pricing table", func(t *testing.T) {
		customTable := &llm_dto.PricingTable{
			Models: []llm_dto.ModelPricing{
				{
					ModelID:          "test-model",
					Provider:         "test-provider",
					InputCostPer1M:   maths.NewDecimalFromString("1.00").Must(),
					OutputCostPer1M:  maths.NewDecimalFromString("2.00").Must(),
					CachedInputPer1M: maths.ZeroDecimal(),
				},
			},
			LastUpdated: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		}

		calc := NewCostCalculatorWithPricing(customTable)
		tableCopy := calc.GetPricingTable()

		require.NotNil(t, tableCopy)
		assert.Equal(t, 1, len(tableCopy.Models))
		assert.Equal(t, "test-model", tableCopy.Models[0].ModelID)
		assert.Equal(t, customTable.LastUpdated, tableCopy.LastUpdated)

		tableCopy.Models[0].ModelID = "modified"
		originalPricing := calc.GetPricing("test-model")
		require.NotNil(t, originalPricing)
		assert.Equal(t, "test-model", originalPricing.ModelID)
	})
}

func TestCostCalculator_EstimateRequestCost(t *testing.T) {

	pricingTable := &llm_dto.PricingTable{
		Models: []llm_dto.ModelPricing{
			{
				ModelID:          "test-gpt-4o",
				Provider:         "openai",
				InputCostPer1M:   maths.NewDecimalFromString("2.50").Must(),
				OutputCostPer1M:  maths.NewDecimalFromString("10.00").Must(),
				CachedInputPer1M: maths.ZeroDecimal(),
			},
			{
				ModelID:          "test-claude-opus",
				Provider:         "anthropic",
				InputCostPer1M:   maths.NewDecimalFromString("15.00").Must(),
				OutputCostPer1M:  maths.NewDecimalFromString("75.00").Must(),
				CachedInputPer1M: maths.ZeroDecimal(),
			},
		},
		LastUpdated: time.Date(2024, 10, 1, 0, 0, 0, 0, time.UTC),
	}

	testCases := []struct {
		name                 string
		model                string
		wantCost             string
		estimatedInputTokens int
	}{
		{
			name:                 "estimates cost for known model",
			model:                "test-gpt-4o",
			estimatedInputTokens: 1_000_000,

			wantCost: "2.5",
		},
		{
			name:                 "estimates cost for small token count",
			model:                "test-gpt-4o",
			estimatedInputTokens: 1000,

			wantCost: "0.0025",
		},
		{
			name:                 "returns zero for unknown model",
			model:                "unknown-model",
			estimatedInputTokens: 1_000_000,
			wantCost:             "0",
		},
		{
			name:                 "handles zero tokens",
			model:                "test-gpt-4o",
			estimatedInputTokens: 0,
			wantCost:             "0",
		},
		{
			name:                 "estimates for anthropic model",
			model:                "test-claude-opus",
			estimatedInputTokens: 1_000_000,

			wantCost: "15",
		},
	}

	calc := NewCostCalculatorWithPricing(pricingTable)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cost := calc.EstimateRequestCost(tc.model, tc.estimatedInputTokens)

			expectedCost := maths.NewMoneyFromString(tc.wantCost, llm_dto.CostCurrency)
			assert.True(t, cost.MustEquals(expectedCost),
				"got %s, want %s", cost.MustNumber(), tc.wantCost)
			currencyCode, err := cost.CurrencyCode()
			require.NoError(t, err)
			assert.Equal(t, llm_dto.CostCurrency, currencyCode)
		})
	}
}

func TestDefaultPricingTable(t *testing.T) {
	t.Run("contains OpenAI models", func(t *testing.T) {
		openaiModels := []string{"gpt-5", "gpt-5-mini", "gpt-5-nano", "gpt-5.1", "gpt-5.2", "gpt-4.1", "gpt-4.1-mini", "gpt-4.1-nano", "o3", "o3-mini", "o4-mini"}
		for _, modelID := range openaiModels {
			pricing := DefaultPricingTable.GetPricing(modelID)
			require.NotNil(t, pricing, "missing pricing for %s", modelID)
			assert.Equal(t, "openai", pricing.Provider)
		}
	})

	t.Run("contains Anthropic models", func(t *testing.T) {
		anthropicModels := []string{
			"claude-opus-4-6",
			"claude-opus-4-5-20251101",
			"claude-sonnet-4-5-20250929",
			"claude-sonnet-4-20250514",
			"claude-haiku-4-5-20251001",
		}
		for _, modelID := range anthropicModels {
			pricing := DefaultPricingTable.GetPricing(modelID)
			require.NotNil(t, pricing, "missing pricing for %s", modelID)
			assert.Equal(t, "anthropic", pricing.Provider)
		}
	})

	t.Run("contains Gemini models", func(t *testing.T) {
		geminiModels := []string{"gemini-2.5-pro", "gemini-2.5-flash", "gemini-2.5-flash-lite"}
		for _, modelID := range geminiModels {
			pricing := DefaultPricingTable.GetPricing(modelID)
			require.NotNil(t, pricing, "missing pricing for %s", modelID)
			assert.Equal(t, "gemini", pricing.Provider)
		}
	})

	t.Run("contains DeepSeek models", func(t *testing.T) {
		deepseekModels := []string{"deepseek-chat", "deepseek-reasoner"}
		for _, modelID := range deepseekModels {
			pricing := DefaultPricingTable.GetPricing(modelID)
			require.NotNil(t, pricing, "missing pricing for %s", modelID)
			assert.Equal(t, "deepseek", pricing.Provider)
		}
	})

	t.Run("contains Mistral models", func(t *testing.T) {
		mistralModels := []string{"mistral-large-latest", "mistral-medium-latest", "mistral-small-latest"}
		for _, modelID := range mistralModels {
			pricing := DefaultPricingTable.GetPricing(modelID)
			require.NotNil(t, pricing, "missing pricing for %s", modelID)
			assert.Equal(t, "mistral", pricing.Provider)
		}
	})

	t.Run("contains xAI models", func(t *testing.T) {
		xaiModels := []string{"grok-4", "grok-3-mini"}
		for _, modelID := range xaiModels {
			pricing := DefaultPricingTable.GetPricing(modelID)
			require.NotNil(t, pricing, "missing pricing for %s", modelID)
			assert.Equal(t, "xai", pricing.Provider)
		}
	})

	t.Run("contains Cohere models", func(t *testing.T) {
		cohereModels := []string{"command-a", "command-r-08-2024", "command-r7b-12-2024"}
		for _, modelID := range cohereModels {
			pricing := DefaultPricingTable.GetPricing(modelID)
			require.NotNil(t, pricing, "missing pricing for %s", modelID)
			assert.Equal(t, "cohere", pricing.Provider)
		}
	})

	t.Run("contains Amazon Nova models", func(t *testing.T) {
		amazonModels := []string{"amazon.nova-micro-v1:0", "amazon.nova-pro-v1:0", "amazon.nova-premier-v1:0"}
		for _, modelID := range amazonModels {
			pricing := DefaultPricingTable.GetPricing(modelID)
			require.NotNil(t, pricing, "missing pricing for %s", modelID)
			assert.Equal(t, "amazon", pricing.Provider)
		}
	})

	t.Run("has valid pricing values", func(t *testing.T) {
		for _, model := range DefaultPricingTable.Models {
			assert.NotEmpty(t, model.ModelID, "model ID should not be empty")
			assert.NotEmpty(t, model.Provider, "provider should not be empty")
			assert.False(t, model.InputCostPer1M.MustIsNegative(),
				"input cost should not be negative for %s", model.ModelID)
			assert.False(t, model.OutputCostPer1M.MustIsNegative(),
				"output cost should not be negative for %s", model.ModelID)
		}
	})

	t.Run("populates cached input pricing where available", func(t *testing.T) {
		modelsWithCaching := []string{"gpt-5", "gpt-4.1", "o3", "claude-opus-4-6", "claude-sonnet-4-5-20250929", "gemini-2.5-pro", "deepseek-chat"}
		for _, modelID := range modelsWithCaching {
			pricing := DefaultPricingTable.GetPricing(modelID)
			require.NotNil(t, pricing, "missing pricing for %s", modelID)
			assert.True(t, pricing.HasCachedPricing(), "expected cached pricing for %s", modelID)
		}
	})
}

func TestCostCalculator_ConcurrentAccess(t *testing.T) {

	customTable := &llm_dto.PricingTable{
		Models: []llm_dto.ModelPricing{
			{
				ModelID:          "concurrent-model",
				Provider:         "test",
				InputCostPer1M:   maths.NewDecimalFromString("1.00").Must(),
				OutputCostPer1M:  maths.NewDecimalFromString("2.00").Must(),
				CachedInputPer1M: maths.ZeroDecimal(),
			},
		},
	}
	calc := NewCostCalculatorWithPricing(customTable)
	done := make(chan struct{})

	go func() {
		for range 100 {
			_ = calc.GetPricing("concurrent-model")
		}
		done <- struct{}{}
	}()

	go func() {
		for range 100 {
			calc.SetPricing(llm_dto.ModelPricing{
				ModelID:          "concurrent-test",
				Provider:         "test",
				InputCostPer1M:   maths.NewDecimalFromString("1.00").Must(),
				OutputCostPer1M:  maths.NewDecimalFromString("2.00").Must(),
				CachedInputPer1M: maths.ZeroDecimal(),
			})
		}
		done <- struct{}{}
	}()

	go func() {
		for range 100 {
			_ = calc.Calculate("concurrent-model", "test", &llm_dto.Usage{
				PromptTokens:     1000,
				CompletionTokens: 500,
				TotalTokens:      1500,
			})
		}
		done <- struct{}{}
	}()

	for range 3 {
		<-done
	}
}
