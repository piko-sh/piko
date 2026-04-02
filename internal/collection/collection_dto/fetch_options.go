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
	"maps"
	"time"
)

// FetchOptions contains options for fetching collection data.
//
// These options are parsed from the user's GetCollection() call arguments
// and passed to providers during data fetching.
//
// Design Philosophy:
//   - Declarative: User specifies what they want, provider decides how
//   - Composable: Options can be combined
//   - Type-safe: Validated at build time
type FetchOptions struct {
	// Cache specifies caching settings for this fetch. If nil, the provider's
	// default caching behaviour is used.
	Cache *CacheConfig

	// Filters holds provider-specific filter values as key-value pairs.
	//
	// Providers may use these values for server-side filtering, sorting,
	// or pagination. The supported keys depend on the provider.
	Filters map[string]any

	// FilterGroup holds structured filter conditions. It takes priority over
	// the Filters map when both are set.
	FilterGroup *FilterGroup

	// Pagination specifies where to start and how many results to return.
	Pagination *PaginationOptions

	// Projection specifies which fields to include or exclude from responses.
	// If nil, all fields are included.
	Projection *FieldProjection

	// ProviderName specifies which provider to use; if empty, the default
	// provider from config is used.
	ProviderName string

	// Locale specifies a single locale to fetch; mutually exclusive with
	// ExplicitLocales and AllLocales.
	Locale string

	// ExplicitLocales specifies a list of locales to fetch.
	// Cannot be used with Locale or AllLocales.
	ExplicitLocales []string

	// Sort specifies sorting options.
	//
	// Multiple sort options are applied in order (first by
	// field1, then by field2, etc).
	Sort []SortOption

	// AllLocales indicates whether to fetch all configured locales.
	// Cannot be used with Locale or ExplicitLocales.
	AllLocales bool
}

// FieldProjection specifies which fields to include/exclude from responses.
// This reduces payload size by omitting fields the client doesn't need.
type FieldProjection struct {
	// IncludeFields lists the metadata fields to include. Empty means all fields.
	IncludeFields []string

	// ExcludeFields lists metadata fields to exclude from projection results.
	// Applied after IncludeFields.
	ExcludeFields []string

	// MaxArrayItems limits items in array fields (e.g., images). 0 means no limit.
	MaxArrayItems int32
}

// CacheConfig specifies caching behaviour for data fetching.
type CacheConfig struct {
	// Strategy specifies the caching approach used when fetching data.
	// Supported values are "cache-first" (check cache before fetching),
	// "network-first" (fetch first, cache as fallback), "stale-while-revalidate"
	// (serve cached data, update in background), and "no-cache" (always fetch).
	Strategy string

	// Key is an optional custom cache key. If empty, a key is generated
	// from the collection name, locale, and filters.
	Key string

	// Tags lists cache invalidation tags for clearing related cache entries.
	// For example, tag entries with "blog" to clear all blog caches at once.
	Tags []string

	// TTL is the cache time-to-live in seconds.
	//
	// After this duration, cached data is considered stale.
	TTL int
}

// ShouldCache returns true if caching is enabled.
//
// Returns bool which is true when the config is non-nil, the strategy is not
// "no-cache", and the TTL is positive.
func (c *CacheConfig) ShouldCache() bool {
	return c != nil && c.Strategy != "no-cache" && c.TTL > 0
}

// GetTTLDuration returns TTL as a time.Duration.
//
// Returns time.Duration which is the cache TTL in seconds, or zero if the
// receiver is nil.
func (c *CacheConfig) GetTTLDuration() time.Duration {
	if c == nil {
		return 0
	}
	return time.Duration(c.TTL) * time.Second
}

// GetTargetLocales returns the effective list of locales to fetch.
//
// This resolves the mutually exclusive locale options into a simple slice
// by checking AllLocales, ExplicitLocales, Locale, and defaultLocale in
// priority order.
//
// Takes configuredLocales ([]string) which lists all locales in the project.
// Takes defaultLocale (string) which specifies the fallback locale.
//
// Returns []string which contains the locale codes to fetch.
func (f *FetchOptions) GetTargetLocales(configuredLocales []string, defaultLocale string) []string {
	if f.AllLocales {
		return configuredLocales
	}

	if len(f.ExplicitLocales) > 0 {
		return f.ExplicitLocales
	}

	if f.Locale != "" {
		return []string{f.Locale}
	}

	return []string{defaultLocale}
}

// HasFilters reports whether any filters are specified.
//
// Returns bool which is true if there are one or more filters.
func (f *FetchOptions) HasFilters() bool {
	return len(f.Filters) > 0
}

// GetFilterString retrieves a string filter value with a fallback.
//
// Takes key (string) which identifies the filter to retrieve.
// Takes defaultValue (string) which is returned if the key is not found or
// the value is not a string.
//
// Returns string which is the filter value or the default if not found.
func (f *FetchOptions) GetFilterString(key, defaultValue string) string {
	if f.Filters == nil {
		return defaultValue
	}
	if value, ok := f.Filters[key]; ok {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}
	return defaultValue
}

// GetFilterInt retrieves an integer filter value with a fallback.
//
// Takes key (string) which specifies the filter name to look up.
// Takes defaultValue (int) which is returned when the key is missing or not a
// number.
//
// Returns int which is the filter value or the default if not found.
func (f *FetchOptions) GetFilterInt(key string, defaultValue int) int {
	if f.Filters == nil {
		return defaultValue
	}
	if value, ok := f.Filters[key]; ok {
		switch v := value.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return defaultValue
}

// GetFilterBool retrieves a boolean filter value with a fallback.
//
// Takes key (string) which identifies the filter to retrieve.
// Takes defaultValue (bool) which is returned when the key is missing or not a
// boolean.
//
// Returns bool which is the filter value or the default if not found.
func (f *FetchOptions) GetFilterBool(key string, defaultValue bool) bool {
	if f.Filters == nil {
		return defaultValue
	}
	if value, ok := f.Filters[key]; ok {
		if boolValue, ok := value.(bool); ok {
			return boolValue
		}
	}
	return defaultValue
}

// Clone creates a shallow copy of FetchOptions.
//
// Use it when a provider needs to modify options without affecting the
// original.
//
// Returns *FetchOptions which is the cloned options instance.
func (f *FetchOptions) Clone() *FetchOptions {
	clone := &FetchOptions{
		ProviderName:    f.ProviderName,
		Locale:          f.Locale,
		ExplicitLocales: make([]string, len(f.ExplicitLocales)),
		AllLocales:      f.AllLocales,
		FilterGroup:     f.FilterGroup,
		Sort:            make([]SortOption, len(f.Sort)),
		Pagination:      f.Pagination,
	}

	copy(clone.ExplicitLocales, f.ExplicitLocales)
	copy(clone.Sort, f.Sort)

	if f.Cache != nil {
		clone.Cache = &CacheConfig{
			Strategy: f.Cache.Strategy,
			TTL:      f.Cache.TTL,
			Key:      f.Cache.Key,
			Tags:     make([]string, len(f.Cache.Tags)),
		}
		copy(clone.Cache.Tags, f.Cache.Tags)
	}

	if f.Filters != nil {
		clone.Filters = make(map[string]any, len(f.Filters))
		maps.Copy(clone.Filters, f.Filters)
	}

	if f.Projection != nil {
		clone.Projection = &FieldProjection{
			IncludeFields: make([]string, len(f.Projection.IncludeFields)),
			ExcludeFields: make([]string, len(f.Projection.ExcludeFields)),
			MaxArrayItems: f.Projection.MaxArrayItems,
		}
		copy(clone.Projection.IncludeFields, f.Projection.IncludeFields)
		copy(clone.Projection.ExcludeFields, f.Projection.ExcludeFields)
	}

	return clone
}
