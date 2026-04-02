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
	"time"

	"piko.sh/piko/wdk/maths"
)

// CostCurrency is the currency code used for LLM cost calculations.
const CostCurrency = "USD"

// ModelPricing defines the cost per token for a model.
type ModelPricing struct {
	// ModelID is the unique model name, e.g. "gpt-5" or "claude-sonnet-4-5-20250929".
	ModelID string

	// Provider is the name of the LLM provider, e.g. "openai", "anthropic", "gemini".
	Provider string

	// InputCostPer1M is the cost per 1 million input tokens in USD.
	InputCostPer1M maths.Decimal

	// OutputCostPer1M is the cost per 1 million output tokens in USD.
	OutputCostPer1M maths.Decimal

	// CachedInputPer1M is the cost for cached input tokens per 1 million.
	// A zero value means cached pricing is not available.
	CachedInputPer1M maths.Decimal
}

// HasCachedPricing reports whether this model has cached input pricing.
//
// Returns bool which is true if cached input pricing is available.
func (p ModelPricing) HasCachedPricing() bool {
	return !p.CachedInputPer1M.CheckIsZero()
}

// CostEstimate represents the calculated cost for a request.
type CostEstimate struct {
	// Timestamp is when the cost was calculated.
	Timestamp time.Time

	// InputCost is the cost for non-cached input tokens.
	InputCost maths.Money

	// CachedInputCost is the cost for cached input tokens. Zero when the
	// model does not support cached pricing or no tokens were cached.
	CachedInputCost maths.Money

	// OutputCost is the cost for output tokens.
	OutputCost maths.Money

	// TotalCost is the combined cost of input, cached input, and output tokens.
	TotalCost maths.Money

	// Model is the name of the model used for the cost estimate.
	Model string

	// Provider is the LLM provider used for this request.
	Provider string

	// InputTokens is the number of non-cached tokens in the input prompt.
	InputTokens int

	// CachedTokens is the number of prompt tokens served from cache.
	CachedTokens int

	// OutputTokens is the number of tokens in the generated response.
	OutputTokens int

	// TotalTokens is the combined count of input and output tokens.
	TotalTokens int
}

// PricingTable holds pricing details for language models.
type PricingTable struct {
	// LastUpdated is when the pricing data was last refreshed.
	LastUpdated time.Time

	// Models holds the pricing details for each supported model.
	Models []ModelPricing
}

// GetPricing returns the pricing for a model by ID.
// Returns nil if the model is not found.
//
// Takes modelID (string) which is the model identifier to look up.
//
// Returns *ModelPricing containing the pricing, or nil if not found.
func (t *PricingTable) GetPricing(modelID string) *ModelPricing {
	if t == nil {
		return nil
	}
	for i := range t.Models {
		if t.Models[i].ModelID == modelID {
			return &t.Models[i]
		}
	}
	return nil
}

// GetPricingByProvider returns all pricing entries for a provider.
//
// Takes provider (string) which is the provider name to filter by.
//
// Returns []ModelPricing containing all matching pricing entries.
func (t *PricingTable) GetPricingByProvider(provider string) []ModelPricing {
	if t == nil {
		return nil
	}
	var result []ModelPricing
	for i := range t.Models {
		if t.Models[i].Provider == provider {
			result = append(result, t.Models[i])
		}
	}
	return result
}
