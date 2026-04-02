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

package collection_dto

import (
	"go/ast"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRuntimeFetcherCode_HasRetries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		r    *RuntimeFetcherCode
		name string
		want bool
	}{
		{name: "with retries", r: &RuntimeFetcherCode{RetryConfig: &RetryConfig{MaxAttempts: 3}}, want: true},
		{name: "zero attempts", r: &RuntimeFetcherCode{RetryConfig: &RetryConfig{MaxAttempts: 0}}, want: false},
		{name: "nil config", r: &RuntimeFetcherCode{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.r.HasRetries())
		})
	}
}

func TestRuntimeFetcherCode_HasFallback(t *testing.T) {
	t.Parallel()

	t.Run("with fallback", func(t *testing.T) {
		t.Parallel()

		r := &RuntimeFetcherCode{FallbackFunc: &ast.FuncDecl{}}
		assert.True(t, r.HasFallback())
	})

	t.Run("without fallback", func(t *testing.T) {
		t.Parallel()

		r := &RuntimeFetcherCode{}
		assert.False(t, r.HasFallback())
	})
}

func TestRuntimeFetcherCode_ShouldCache(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		strategy string
		want     bool
	}{
		{name: "cache-first", strategy: "cache-first", want: true},
		{name: "network-first", strategy: "network-first", want: true},
		{name: "stale-while-revalidate", strategy: "stale-while-revalidate", want: true},
		{name: "none", strategy: "none", want: false},
		{name: "empty", strategy: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &RuntimeFetcherCode{CacheStrategy: tt.strategy}
			assert.Equal(t, tt.want, r.ShouldCache())
		})
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	t.Parallel()

	config := DefaultRetryConfig()

	assert.Equal(t, 3, config.MaxAttempts)
	assert.Equal(t, 100*time.Millisecond, config.InitialDelay)
	assert.Equal(t, 5*time.Second, config.MaxDelay)
	assert.InDelta(t, 2.0, config.BackoffMultiplier, 0.001)
	assert.Empty(t, config.RetryableErrors)
}
