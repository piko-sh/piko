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

package llm_provider_grok

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_domain"
)

func TestRewrapError(t *testing.T) {
	t.Run("rewraps ProviderError from openai to grok", func(t *testing.T) {
		original := &llm_domain.ProviderError{
			Provider:   "openai",
			StatusCode: 429,
			Message:    "rate limit exceeded",
			Err:        errors.New("underlying error"),
		}

		result := rewrapError(original)
		require.NotNil(t, result)

		pe, ok := errors.AsType[*llm_domain.ProviderError](result)
		require.True(t, ok)
		assert.Equal(t, "grok", pe.Provider)
		assert.Equal(t, 429, pe.StatusCode)
		assert.Equal(t, "rate limit exceeded", pe.Message)
		assert.EqualError(t, pe.Err, "underlying error")
	})

	t.Run("rewraps wrapped ProviderError", func(t *testing.T) {
		inner := &llm_domain.ProviderError{
			Provider:   "openai",
			StatusCode: 500,
			Message:    "internal server error",
			Err:        errors.New("api failure"),
		}
		wrapped := fmt.Errorf("openai completion failed: %w", inner)

		result := rewrapError(wrapped)
		require.NotNil(t, result)

		pe, ok := errors.AsType[*llm_domain.ProviderError](result)
		require.True(t, ok)
		assert.Equal(t, "grok", pe.Provider)
		assert.Equal(t, 500, pe.StatusCode)
	})

	t.Run("passes through plain error unchanged", func(t *testing.T) {
		original := errors.New("something went wrong")
		result := rewrapError(original)
		assert.Equal(t, original, result)
	})

	t.Run("returns nil for nil error", func(t *testing.T) {
		assert.Nil(t, rewrapError(nil))
	})
}
