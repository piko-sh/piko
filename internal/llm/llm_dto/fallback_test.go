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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFallbackTrigger_HasTrigger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		trigger FallbackTrigger
		check   FallbackTrigger
		want    bool
	}{
		{name: "error has error", trigger: FallbackOnError, check: FallbackOnError, want: true},
		{name: "error does not have rate limit", trigger: FallbackOnError, check: FallbackOnRateLimit, want: false},
		{name: "all has error", trigger: FallbackOnAll, check: FallbackOnError, want: true},
		{name: "all has rate limit", trigger: FallbackOnAll, check: FallbackOnRateLimit, want: true},
		{name: "all has timeout", trigger: FallbackOnAll, check: FallbackOnTimeout, want: true},
		{name: "all has budget", trigger: FallbackOnAll, check: FallbackOnBudgetExceeded, want: true},
		{name: "combined has both", trigger: FallbackOnError | FallbackOnTimeout, check: FallbackOnTimeout, want: true},
		{name: "combined missing rate limit", trigger: FallbackOnError | FallbackOnTimeout, check: FallbackOnRateLimit, want: false},
		{name: "zero has nothing", trigger: FallbackTrigger(0), check: FallbackOnError, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.trigger.HasTrigger(tt.check))
		})
	}
}

func TestNewFallbackConfig(t *testing.T) {
	t.Parallel()

	config := NewFallbackConfig("openai", "anthropic", "cohere")

	assert.Equal(t, []string{"openai", "anthropic", "cohere"}, config.Providers)
	assert.Equal(t, FallbackOnAll, config.Triggers)
	assert.Nil(t, config.ModelMapping)
}

func TestFallbackConfig_WithTriggers(t *testing.T) {
	t.Parallel()

	config := NewFallbackConfig("openai").WithTriggers(FallbackOnError | FallbackOnRateLimit)

	assert.Equal(t, FallbackOnError|FallbackOnRateLimit, config.Triggers)
	assert.True(t, config.Triggers.HasTrigger(FallbackOnError))
	assert.True(t, config.Triggers.HasTrigger(FallbackOnRateLimit))
	assert.False(t, config.Triggers.HasTrigger(FallbackOnTimeout))
}

func TestFallbackConfig_WithModelMapping(t *testing.T) {
	t.Parallel()

	mapping := map[string]string{
		"anthropic": "claude-3-5-sonnet-20241022",
		"openai":    "gpt-4o",
	}
	config := NewFallbackConfig("openai", "anthropic").WithModelMapping(mapping)

	assert.Equal(t, mapping, config.ModelMapping)
}

func TestFallbackConfig_GetModel(t *testing.T) {
	t.Parallel()

	t.Run("with mapping", func(t *testing.T) {
		t.Parallel()

		config := NewFallbackConfig("openai", "anthropic").WithModelMapping(map[string]string{
			"anthropic": "claude-3-5-sonnet",
		})
		assert.Equal(t, "claude-3-5-sonnet", config.GetModel("anthropic", "default-model"))
	})

	t.Run("no mapping for provider", func(t *testing.T) {
		t.Parallel()

		config := NewFallbackConfig("openai", "anthropic").WithModelMapping(map[string]string{
			"anthropic": "claude-3-5-sonnet",
		})
		assert.Equal(t, "gpt-4o", config.GetModel("openai", "gpt-4o"))
	})

	t.Run("nil mapping", func(t *testing.T) {
		t.Parallel()

		config := NewFallbackConfig("openai")
		assert.Equal(t, "gpt-4o", config.GetModel("openai", "gpt-4o"))
	})
}

func TestFallbackResult_WasFallbackUsed(t *testing.T) {
	t.Parallel()

	t.Run("single provider", func(t *testing.T) {
		t.Parallel()

		r := &FallbackResult{
			UsedProvider:       "openai",
			AttemptedProviders: []string{"openai"},
		}
		assert.False(t, r.WasFallbackUsed())
	})

	t.Run("multiple providers", func(t *testing.T) {
		t.Parallel()

		r := &FallbackResult{
			UsedProvider:       "anthropic",
			AttemptedProviders: []string{"openai", "anthropic"},
		}
		assert.True(t, r.WasFallbackUsed())
	})
}

func TestFallbackResult_GetError(t *testing.T) {
	t.Parallel()

	t.Run("with error", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("rate limited")
		r := &FallbackResult{
			Errors: map[string]error{"openai": expectedErr},
		}
		assert.Equal(t, expectedErr, r.GetError("openai"))
	})

	t.Run("no error for provider", func(t *testing.T) {
		t.Parallel()

		r := &FallbackResult{
			Errors: map[string]error{"openai": errors.New("failed")},
		}
		assert.Nil(t, r.GetError("anthropic"))
	})

	t.Run("nil errors map", func(t *testing.T) {
		t.Parallel()

		r := &FallbackResult{}
		assert.Nil(t, r.GetError("openai"))
	})
}
