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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheConfig_ShouldCache(t *testing.T) {
	t.Parallel()

	tests := []struct {
		config *CacheConfig
		name   string
		want   bool
	}{
		{name: "nil config", config: nil, want: false},
		{name: "no-cache strategy", config: &CacheConfig{Strategy: "no-cache", TTL: 60}, want: false},
		{name: "zero TTL", config: &CacheConfig{Strategy: "cache-first", TTL: 0}, want: false},
		{name: "valid config", config: &CacheConfig{Strategy: "cache-first", TTL: 60}, want: true},
		{name: "stale-while-revalidate", config: &CacheConfig{Strategy: "stale-while-revalidate", TTL: 30}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.config.ShouldCache())
		})
	}
}

func TestCacheConfig_GetTTLDuration(t *testing.T) {
	t.Parallel()

	t.Run("nil config", func(t *testing.T) {
		t.Parallel()

		var c *CacheConfig
		assert.Equal(t, time.Duration(0), c.GetTTLDuration())
	})

	t.Run("with TTL", func(t *testing.T) {
		t.Parallel()

		c := &CacheConfig{TTL: 120}
		assert.Equal(t, 120*time.Second, c.GetTTLDuration())
	})
}

func TestFetchOptions_GetTargetLocales(t *testing.T) {
	t.Parallel()

	configured := []string{"en", "fr", "de"}

	t.Run("all locales", func(t *testing.T) {
		t.Parallel()

		opts := &FetchOptions{AllLocales: true}
		assert.Equal(t, configured, opts.GetTargetLocales(configured, "en"))
	})

	t.Run("explicit locales", func(t *testing.T) {
		t.Parallel()

		opts := &FetchOptions{ExplicitLocales: []string{"fr", "de"}}
		assert.Equal(t, []string{"fr", "de"}, opts.GetTargetLocales(configured, "en"))
	})

	t.Run("single locale", func(t *testing.T) {
		t.Parallel()

		opts := &FetchOptions{Locale: "fr"}
		assert.Equal(t, []string{"fr"}, opts.GetTargetLocales(configured, "en"))
	})

	t.Run("default locale", func(t *testing.T) {
		t.Parallel()

		opts := &FetchOptions{}
		assert.Equal(t, []string{"en"}, opts.GetTargetLocales(configured, "en"))
	})
}

func TestFetchOptions_HasFilters(t *testing.T) {
	t.Parallel()

	t.Run("with filters", func(t *testing.T) {
		t.Parallel()

		opts := &FetchOptions{Filters: map[string]any{"status": "published"}}
		assert.True(t, opts.HasFilters())
	})

	t.Run("empty filters", func(t *testing.T) {
		t.Parallel()

		opts := &FetchOptions{Filters: map[string]any{}}
		assert.False(t, opts.HasFilters())
	})

	t.Run("nil filters", func(t *testing.T) {
		t.Parallel()

		opts := &FetchOptions{}
		assert.False(t, opts.HasFilters())
	})
}

func TestFetchOptions_GetFilterString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		filters      map[string]any
		key          string
		defaultValue string
		want         string
	}{
		{name: "string value", filters: map[string]any{"status": "active"}, key: "status", defaultValue: "all", want: "active"},
		{name: "missing key", filters: map[string]any{}, key: "status", defaultValue: "all", want: "all"},
		{name: "non-string value", filters: map[string]any{"status": 42}, key: "status", defaultValue: "all", want: "all"},
		{name: "nil filters", filters: nil, key: "status", defaultValue: "all", want: "all"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := &FetchOptions{Filters: tt.filters}
			assert.Equal(t, tt.want, opts.GetFilterString(tt.key, tt.defaultValue))
		})
	}
}

func TestFetchOptions_GetFilterInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		filters      map[string]any
		key          string
		defaultValue int
		want         int
	}{
		{name: "int value", filters: map[string]any{"limit": 10}, key: "limit", defaultValue: 0, want: 10},
		{name: "int64 value", filters: map[string]any{"limit": int64(20)}, key: "limit", defaultValue: 0, want: 20},
		{name: "float64 value", filters: map[string]any{"limit": float64(30)}, key: "limit", defaultValue: 0, want: 30},
		{name: "missing key", filters: map[string]any{}, key: "limit", defaultValue: 50, want: 50},
		{name: "non-numeric value", filters: map[string]any{"limit": "many"}, key: "limit", defaultValue: 0, want: 0},
		{name: "nil filters", filters: nil, key: "limit", defaultValue: 25, want: 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := &FetchOptions{Filters: tt.filters}
			assert.Equal(t, tt.want, opts.GetFilterInt(tt.key, tt.defaultValue))
		})
	}
}

func TestFetchOptions_GetFilterBool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		filters      map[string]any
		key          string
		defaultValue bool
		want         bool
	}{
		{name: "true value", filters: map[string]any{"published": true}, key: "published", defaultValue: false, want: true},
		{name: "false value", filters: map[string]any{"published": false}, key: "published", defaultValue: true, want: false},
		{name: "missing key", filters: map[string]any{}, key: "published", defaultValue: true, want: true},
		{name: "non-bool value", filters: map[string]any{"published": "yes"}, key: "published", defaultValue: false, want: false},
		{name: "nil filters", filters: nil, key: "published", defaultValue: true, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := &FetchOptions{Filters: tt.filters}
			assert.Equal(t, tt.want, opts.GetFilterBool(tt.key, tt.defaultValue))
		})
	}
}

func TestFetchOptions_Clone(t *testing.T) {
	t.Parallel()

	t.Run("full clone", func(t *testing.T) {
		t.Parallel()

		original := &FetchOptions{
			ProviderName:    "markdown",
			Locale:          "en",
			ExplicitLocales: []string{"en", "fr"},
			AllLocales:      false,
			Filters:         map[string]any{"status": "published"},
			Sort:            []SortOption{{Field: "date"}},
			Cache: &CacheConfig{
				Strategy: "cache-first",
				TTL:      60,
				Key:      "custom-key",
				Tags:     []string{"blog", "posts"},
			},
			Projection: &FieldProjection{
				IncludeFields: []string{"title", "slug"},
				ExcludeFields: []string{"content"},
				MaxArrayItems: 5,
			},
		}

		clone := original.Clone()

		require.NotNil(t, clone)
		assert.Equal(t, original.ProviderName, clone.ProviderName)
		assert.Equal(t, original.Locale, clone.Locale)
		assert.Equal(t, original.ExplicitLocales, clone.ExplicitLocales)
		assert.Equal(t, original.AllLocales, clone.AllLocales)
		assert.Equal(t, original.Filters, clone.Filters)

		clone.Filters["status"] = "draft"
		assert.Equal(t, "published", original.Filters["status"])

		require.NotNil(t, clone.Cache)
		assert.Equal(t, original.Cache.Strategy, clone.Cache.Strategy)
		assert.Equal(t, original.Cache.TTL, clone.Cache.TTL)

		require.NotNil(t, clone.Projection)
		assert.Equal(t, original.Projection.IncludeFields, clone.Projection.IncludeFields)
		assert.Equal(t, original.Projection.MaxArrayItems, clone.Projection.MaxArrayItems)
	})

	t.Run("nil optional fields", func(t *testing.T) {
		t.Parallel()

		original := &FetchOptions{ProviderName: "markdown"}
		clone := original.Clone()

		assert.Equal(t, "markdown", clone.ProviderName)
		assert.Nil(t, clone.Cache)
		assert.Nil(t, clone.Filters)
		assert.Nil(t, clone.Projection)
	})
}
