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

// FallbackTrigger defines the conditions that trigger fallback to the next
// provider. Multiple triggers can be combined using bitwise OR.
type FallbackTrigger int

const (
	// FallbackOnError triggers fallback when any provider returns an error.
	FallbackOnError FallbackTrigger = 1 << iota

	// FallbackOnRateLimit triggers fallback when the provider rate limits the
	// request.
	FallbackOnRateLimit

	// FallbackOnTimeout triggers fallback when the provider request times out.
	FallbackOnTimeout

	// FallbackOnBudgetExceeded triggers fallback when the provider's budget is
	// exceeded.
	FallbackOnBudgetExceeded

	// FallbackOnAll triggers fallback on any failure condition.
	FallbackOnAll = FallbackOnError | FallbackOnRateLimit | FallbackOnTimeout | FallbackOnBudgetExceeded
)

// HasTrigger checks if a specific trigger is set.
//
// Takes trigger (FallbackTrigger) which is the trigger to check.
//
// Returns bool which is true if the trigger is set.
func (t FallbackTrigger) HasTrigger(trigger FallbackTrigger) bool {
	return t&trigger != 0
}

// FallbackConfig configures fallback behaviour for LLM completion requests.
type FallbackConfig struct {
	// ModelMapping maps provider names to model overrides. For example,
	// {"anthropic": "claude-3-5-sonnet-20241022"} uses that model for the
	// Anthropic provider instead of the default.
	ModelMapping map[string]string

	// Providers lists the provider names to try in order. The first provider is
	// tried first; if it fails, the next provider is tried, and so on.
	Providers []string

	// Triggers sets which conditions cause a fallback to the next provider.
	// If not set (zero), defaults to FallbackOnAll.
	Triggers FallbackTrigger
}

// NewFallbackConfig creates a new FallbackConfig with the given providers.
// The default triggers are set to FallbackOnAll.
//
// Takes providers (...string) which are the provider names in priority order.
//
// Returns *FallbackConfig configured with the providers.
func NewFallbackConfig(providers ...string) *FallbackConfig {
	return &FallbackConfig{
		Providers: providers,
		Triggers:  FallbackOnAll,
	}
}

// WithTriggers sets the fallback triggers.
//
// Takes triggers (FallbackTrigger) which defines when to fallback.
//
// Returns *FallbackConfig for method chaining.
func (c *FallbackConfig) WithTriggers(triggers FallbackTrigger) *FallbackConfig {
	c.Triggers = triggers
	return c
}

// WithModelMapping sets model overrides for specific providers.
//
// Takes mapping (map[string]string) which maps provider names to models.
//
// Returns *FallbackConfig for method chaining.
func (c *FallbackConfig) WithModelMapping(mapping map[string]string) *FallbackConfig {
	c.ModelMapping = mapping
	return c
}

// GetModel returns the model to use for a provider, applying any mapping.
//
// Takes provider (string) which is the provider name.
// Takes defaultModel (string) which is the model to use if no mapping exists.
//
// Returns string which is the model to use.
func (c *FallbackConfig) GetModel(provider, defaultModel string) string {
	if c.ModelMapping != nil {
		if model, ok := c.ModelMapping[provider]; ok {
			return model
		}
	}
	return defaultModel
}

// FallbackResult contains information about fallback execution.
type FallbackResult struct {
	// Errors maps provider names to the errors they returned. Only providers
	// that failed are included; successful providers are not present in this map.
	Errors map[string]error

	// UsedProvider is the name of the provider that handled the request.
	UsedProvider string

	// AttemptedProviders lists the providers that were tried, in order.
	AttemptedProviders []string
}

// WasFallbackUsed reports whether a fallback provider was used.
//
// Returns bool which is true if the request required fallback.
func (r *FallbackResult) WasFallbackUsed() bool {
	return len(r.AttemptedProviders) > 1
}

// GetError returns the error for a specific provider, or nil if none.
//
// Takes provider (string) which is the provider name.
//
// Returns error which is the error for that provider, or nil.
func (r *FallbackResult) GetError(provider string) error {
	if r.Errors == nil {
		return nil
	}
	return r.Errors[provider]
}
