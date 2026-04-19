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
	"sync"
	"time"

	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/maths"
)

// tokensPerMillion is the divisor for converting per-million token pricing.
const tokensPerMillion = 1_000_000

// DefaultPricingTable contains built-in pricing for common LLM models
// in USD per 1 million tokens, last updated February 2026.
var DefaultPricingTable = &llm_dto.PricingTable{
	Models: []llm_dto.ModelPricing{
		{ModelID: "gpt-5", Provider: "openai", InputCostPer1M: decimal("1.25"), OutputCostPer1M: decimal("10.00"), CachedInputPer1M: decimal("0.125")},
		{ModelID: "gpt-5-mini", Provider: "openai", InputCostPer1M: decimal("0.25"), OutputCostPer1M: decimal("2.00"), CachedInputPer1M: decimal("0.025")},
		{ModelID: "gpt-5-nano", Provider: "openai", InputCostPer1M: decimal("0.05"), OutputCostPer1M: decimal("0.40"), CachedInputPer1M: decimal("0.005")},
		{ModelID: "gpt-5.1", Provider: "openai", InputCostPer1M: decimal("1.25"), OutputCostPer1M: decimal("10.00"), CachedInputPer1M: decimal("0.125")},
		{ModelID: "gpt-5.2", Provider: "openai", InputCostPer1M: decimal("1.75"), OutputCostPer1M: decimal("14.00"), CachedInputPer1M: decimal("0.175")},
		{ModelID: "gpt-4.1", Provider: "openai", InputCostPer1M: decimal("2.00"), OutputCostPer1M: decimal("8.00"), CachedInputPer1M: decimal("0.50")},
		{ModelID: "gpt-4.1-mini", Provider: "openai", InputCostPer1M: decimal("0.40"), OutputCostPer1M: decimal("1.60"), CachedInputPer1M: decimal("0.10")},
		{ModelID: "gpt-4.1-nano", Provider: "openai", InputCostPer1M: decimal("0.10"), OutputCostPer1M: decimal("0.40"), CachedInputPer1M: decimal("0.025")},
		{ModelID: "o3", Provider: "openai", InputCostPer1M: decimal("2.00"), OutputCostPer1M: decimal("8.00"), CachedInputPer1M: decimal("0.50")},
		{ModelID: "o3-mini", Provider: "openai", InputCostPer1M: decimal("1.10"), OutputCostPer1M: decimal("4.40"), CachedInputPer1M: maths.ZeroDecimal()},
		{ModelID: "o4-mini", Provider: "openai", InputCostPer1M: decimal("1.10"), OutputCostPer1M: decimal("4.40"), CachedInputPer1M: maths.ZeroDecimal()},

		{ModelID: "claude-opus-4-6", Provider: "anthropic", InputCostPer1M: decimal("5.00"), OutputCostPer1M: decimal("25.00"), CachedInputPer1M: decimal("0.50")},
		{ModelID: "claude-opus-4-5-20251101", Provider: "anthropic", InputCostPer1M: decimal("5.00"), OutputCostPer1M: decimal("25.00"), CachedInputPer1M: decimal("0.50")},
		{ModelID: "claude-sonnet-4-5-20250929", Provider: "anthropic", InputCostPer1M: decimal("3.00"), OutputCostPer1M: decimal("15.00"), CachedInputPer1M: decimal("0.30")},
		{ModelID: "claude-sonnet-4-20250514", Provider: "anthropic", InputCostPer1M: decimal("3.00"), OutputCostPer1M: decimal("15.00"), CachedInputPer1M: decimal("0.30")},
		{ModelID: "claude-haiku-4-5-20251001", Provider: "anthropic", InputCostPer1M: decimal("1.00"), OutputCostPer1M: decimal("5.00"), CachedInputPer1M: decimal("0.10")},
		{ModelID: "claude-opus-4-1-20250805", Provider: "anthropic", InputCostPer1M: decimal("15.00"), OutputCostPer1M: decimal("75.00"), CachedInputPer1M: decimal("1.50")},
		{ModelID: "claude-opus-4-20250514", Provider: "anthropic", InputCostPer1M: decimal("15.00"), OutputCostPer1M: decimal("75.00"), CachedInputPer1M: decimal("1.50")},

		{ModelID: "gemini-2.5-pro", Provider: "gemini", InputCostPer1M: decimal("1.25"), OutputCostPer1M: decimal("10.00"), CachedInputPer1M: decimal("0.125")},
		{ModelID: "gemini-2.5-flash", Provider: "gemini", InputCostPer1M: decimal("0.30"), OutputCostPer1M: decimal("2.50"), CachedInputPer1M: decimal("0.03")},
		{ModelID: "gemini-2.5-flash-lite", Provider: "gemini", InputCostPer1M: decimal("0.10"), OutputCostPer1M: decimal("0.40"), CachedInputPer1M: decimal("0.01")},

		{ModelID: "deepseek-chat", Provider: "deepseek", InputCostPer1M: decimal("0.28"), OutputCostPer1M: decimal("0.42"), CachedInputPer1M: decimal("0.028")},
		{ModelID: "deepseek-reasoner", Provider: "deepseek", InputCostPer1M: decimal("0.28"), OutputCostPer1M: decimal("0.42"), CachedInputPer1M: decimal("0.028")},

		{ModelID: "mistral-large-latest", Provider: "mistral", InputCostPer1M: decimal("0.50"), OutputCostPer1M: decimal("1.50"), CachedInputPer1M: maths.ZeroDecimal()},
		{ModelID: "mistral-medium-latest", Provider: "mistral", InputCostPer1M: decimal("0.40"), OutputCostPer1M: decimal("2.00"), CachedInputPer1M: maths.ZeroDecimal()},
		{ModelID: "mistral-small-latest", Provider: "mistral", InputCostPer1M: decimal("0.03"), OutputCostPer1M: decimal("0.11"), CachedInputPer1M: maths.ZeroDecimal()},

		{ModelID: "grok-4", Provider: "xai", InputCostPer1M: decimal("3.00"), OutputCostPer1M: decimal("15.00"), CachedInputPer1M: maths.ZeroDecimal()},
		{ModelID: "grok-3-mini", Provider: "xai", InputCostPer1M: decimal("0.30"), OutputCostPer1M: decimal("0.50"), CachedInputPer1M: maths.ZeroDecimal()},

		{ModelID: "command-a", Provider: "cohere", InputCostPer1M: decimal("2.50"), OutputCostPer1M: decimal("10.00"), CachedInputPer1M: maths.ZeroDecimal()},
		{ModelID: "command-r-08-2024", Provider: "cohere", InputCostPer1M: decimal("0.15"), OutputCostPer1M: decimal("0.60"), CachedInputPer1M: maths.ZeroDecimal()},
		{ModelID: "command-r7b-12-2024", Provider: "cohere", InputCostPer1M: decimal("0.04"), OutputCostPer1M: decimal("0.15"), CachedInputPer1M: maths.ZeroDecimal()},

		{ModelID: "amazon.nova-micro-v1:0", Provider: "amazon", InputCostPer1M: decimal("0.035"), OutputCostPer1M: decimal("0.14"), CachedInputPer1M: maths.ZeroDecimal()},
		{ModelID: "amazon.nova-pro-v1:0", Provider: "amazon", InputCostPer1M: decimal("0.80"), OutputCostPer1M: decimal("3.20"), CachedInputPer1M: maths.ZeroDecimal()},
		{ModelID: "amazon.nova-premier-v1:0", Provider: "amazon", InputCostPer1M: decimal("2.50"), OutputCostPer1M: decimal("12.50"), CachedInputPer1M: maths.ZeroDecimal()},
	},
	LastUpdated: time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
}

// CostCalculator calculates the cost of LLM API calls based on token usage.
type CostCalculator struct {
	// clock provides time functions for timestamp generation in cost estimates.
	clock clock.Clock

	// pricingTable stores the pricing data for all supported models.
	pricingTable *llm_dto.PricingTable

	// mu guards concurrent access to the pricing table.
	mu sync.RWMutex
}

// CostCalculatorOption is a function type that sets options on a CostCalculator.
type CostCalculatorOption func(*CostCalculator)

// NewCostCalculator creates a new CostCalculator with the default pricing table.
//
// Takes opts (...CostCalculatorOption) which are optional configuration functions.
//
// Returns *CostCalculator initialised with default pricing.
func NewCostCalculator(opts ...CostCalculatorOption) *CostCalculator {
	cc := &CostCalculator{
		clock:        clock.RealClock(),
		pricingTable: DefaultPricingTable,
		mu:           sync.RWMutex{},
	}
	for _, opt := range opts {
		opt(cc)
	}
	return cc
}

// NewCostCalculatorWithPricing creates a new CostCalculator with custom pricing.
//
// Takes table (*llm_dto.PricingTable) which is the custom pricing table to use.
// Takes opts (...CostCalculatorOption) which are optional configuration functions.
//
// Returns *CostCalculator initialised with the provided pricing.
func NewCostCalculatorWithPricing(table *llm_dto.PricingTable, opts ...CostCalculatorOption) *CostCalculator {
	cc := &CostCalculator{
		clock:        clock.RealClock(),
		pricingTable: table,
		mu:           sync.RWMutex{},
	}
	for _, opt := range opts {
		opt(cc)
	}
	return cc
}

// Calculate calculates the cost from usage for a given model and provider.
//
// Takes model (string) which is the model identifier.
// Takes provider (string) which is the provider name.
// Takes usage (*llm_dto.Usage) which contains the token counts.
//
// Returns *llm_dto.CostEstimate containing the calculated costs, or nil if
// the model pricing is not found or usage is nil.
func (c *CostCalculator) Calculate(model, provider string, usage *llm_dto.Usage) *llm_dto.CostEstimate {
	if usage == nil {
		return nil
	}

	pricing := c.GetPricing(model)
	if pricing == nil {
		return &llm_dto.CostEstimate{
			InputTokens:     usage.PromptTokens,
			CachedTokens:    usage.CachedTokens,
			OutputTokens:    usage.CompletionTokens,
			TotalTokens:     usage.TotalTokens,
			InputCost:       maths.ZeroMoney(llm_dto.CostCurrency),
			CachedInputCost: maths.ZeroMoney(llm_dto.CostCurrency),
			OutputCost:      maths.ZeroMoney(llm_dto.CostCurrency),
			TotalCost:       maths.ZeroMoney(llm_dto.CostCurrency),
			Model:           model,
			Provider:        provider,
			Timestamp:       c.clock.Now(),
		}
	}

	cachedTokens := min(usage.CachedTokens, usage.PromptTokens)
	uncachedInputTokens := usage.PromptTokens - cachedTokens

	inputCostDecimal := pricing.InputCostPer1M.MultiplyInt(int64(uncachedInputTokens)).DivideInt(tokensPerMillion)
	outputCostDecimal := pricing.OutputCostPer1M.MultiplyInt(int64(usage.CompletionTokens)).DivideInt(tokensPerMillion)

	var cachedCostDecimal maths.Decimal
	if cachedTokens > 0 && pricing.HasCachedPricing() {
		cachedCostDecimal = pricing.CachedInputPer1M.MultiplyInt(int64(cachedTokens)).DivideInt(tokensPerMillion)
	}

	inputCost := maths.NewMoneyFromDecimal(inputCostDecimal, llm_dto.CostCurrency)
	cachedInputCost := maths.NewMoneyFromDecimal(cachedCostDecimal, llm_dto.CostCurrency)
	outputCost := maths.NewMoneyFromDecimal(outputCostDecimal, llm_dto.CostCurrency)
	totalCost := inputCost.Add(cachedInputCost).Add(outputCost)

	return &llm_dto.CostEstimate{
		InputTokens:     uncachedInputTokens,
		CachedTokens:    cachedTokens,
		OutputTokens:    usage.CompletionTokens,
		TotalTokens:     usage.TotalTokens,
		InputCost:       inputCost,
		CachedInputCost: cachedInputCost,
		OutputCost:      outputCost,
		TotalCost:       totalCost,
		Model:           model,
		Provider:        provider,
		Timestamp:       c.clock.Now(),
	}
}

// SetPricing updates or adds pricing for a model.
//
// Takes pricing (llm_dto.ModelPricing) which is the pricing to set.
//
// Safe for concurrent use.
func (c *CostCalculator) SetPricing(pricing llm_dto.ModelPricing) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range c.pricingTable.Models {
		if c.pricingTable.Models[i].ModelID == pricing.ModelID {
			c.pricingTable.Models[i] = pricing
			return
		}
	}

	c.pricingTable.Models = append(c.pricingTable.Models, pricing)
}

// SetPricingTable replaces the entire pricing table.
//
// Takes table (*llm_dto.PricingTable) which is the new pricing table.
//
// Safe for concurrent use.
func (c *CostCalculator) SetPricingTable(table *llm_dto.PricingTable) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pricingTable = table
}

// GetPricing returns pricing for a model, or nil if unknown.
//
// Takes model (string) which is the model identifier.
//
// Returns *llm_dto.ModelPricing containing the pricing, or nil if not found.
//
// Safe for concurrent use.
func (c *CostCalculator) GetPricing(model string) *llm_dto.ModelPricing {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.pricingTable.GetPricing(model)
}

// GetPricingTable returns a copy of the current pricing table.
//
// Returns *llm_dto.PricingTable containing the current pricing.
//
// Safe for concurrent use. Uses a read lock to protect access.
func (c *CostCalculator) GetPricingTable() *llm_dto.PricingTable {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tableCopy := &llm_dto.PricingTable{
		LastUpdated: c.pricingTable.LastUpdated,
		Models:      make([]llm_dto.ModelPricing, len(c.pricingTable.Models)),
	}
	copy(tableCopy.Models, c.pricingTable.Models)
	return tableCopy
}

// EstimateRequestCost estimates the cost before making a request.
// This is a rough estimate based on message content length, assuming
// about 4 characters per token (a common approximation).
//
// Takes model (string) which is the model identifier.
// Takes estimatedInputTokens (int) which is the estimated number of
// input tokens.
//
// Returns maths.Money which is the estimated input cost in USD, or zero
// if pricing is unknown.
func (c *CostCalculator) EstimateRequestCost(model string, estimatedInputTokens int) maths.Money {
	pricing := c.GetPricing(model)
	if pricing == nil {
		return maths.ZeroMoney(llm_dto.CostCurrency)
	}
	costDecimal := pricing.InputCostPer1M.MultiplyInt(int64(estimatedInputTokens)).DivideInt(tokensPerMillion)
	return maths.NewMoneyFromDecimal(costDecimal, llm_dto.CostCurrency)
}

// WithCostCalculatorClock sets the clock used for time operations.
// If not set, clock.RealClock() is used.
//
// Takes c (clock.Clock) which provides time operations.
//
// Returns CostCalculatorOption to apply to the calculator.
func WithCostCalculatorClock(c clock.Clock) CostCalculatorOption {
	return func(cc *CostCalculator) {
		cc.clock = c
	}
}

// decimal parses a string into a Decimal for use in pricing tables.
//
// Takes s (string) which is the decimal value to parse.
//
// Returns maths.Decimal which is the parsed decimal value.
func decimal(s string) maths.Decimal {
	return maths.NewDecimalFromString(s).Must()
}
