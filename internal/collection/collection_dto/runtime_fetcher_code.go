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
	"time"
)

const (
	// defaultRetryMaxAttempts is the default number of times to retry a request.
	defaultRetryMaxAttempts = 3

	// defaultRetryInitialDelayMs is the starting delay between retries in
	// milliseconds.
	defaultRetryInitialDelayMs = 100

	// defaultRetryMaxDelaySeconds is the longest wait in seconds between retries.
	defaultRetryMaxDelaySeconds = 5

	// defaultRetryBackoffMultiplier is the multiplier applied to the delay after
	// each retry.
	defaultRetryBackoffMultiplier = 2.0
)

// RuntimeFetcherCode represents generated Go code for runtime data fetching.
//
// Acts as the output of a dynamic provider's GenerateRuntimeFetcher method.
// Contains the complete specification for code that will be injected into the
// compiled component to fetch data at runtime.
//
// Design Philosophy:
//   - AST-based: Uses Go's ast package for type-safe code generation
//   - Self-contained: Includes all dependencies (imports, cache config,
//     error handling)
//   - Flexible: Provider controls exact runtime behaviour
type RuntimeFetcherCode struct {
	// FetcherFunc is the Go AST for the function that fetches data at runtime.
	//
	// The function signature should be:
	//   func(ctx context.Context, opts FetchOptions) ([]TargetType, error)
	//
	// The generator injects the AST into the component's code and calls it
	// from the Render function.
	FetcherFunc *ast.FuncDecl

	// RequiredImports specifies the Go packages needed by the fetcher function.
	//
	// Key: import path (e.g., "piko.sh/piko/wdk/runtime") Value: import alias
	// (empty string means no alias)
	RequiredImports map[string]string

	// RetryConfig holds the retry settings for failed fetches; nil means no
	// retries.
	RetryConfig *RetryConfig

	// FallbackFunc is an optional function to call if the fetch fails.
	//
	// Providers use it to specify graceful degradation logic.
	// For example, serving a static snapshot if the CMS is unavailable.
	FallbackFunc *ast.FuncDecl

	// CacheStrategy sets how the fetcher stores data for later use.
	//
	// Strategies:
	//   - "none": Do not cache.
	//   - "cache-first": Use cached data if present, fetch if not.
	//   - "network-first": Fetch first, use cache if fetch fails.
	//   - "stale-while-revalidate": Return cached data now, update in background.
	CacheStrategy string

	// CacheTTL is how long cached data stays valid.
	//
	// After this time, the cache entry is seen as stale and will be
	// refreshed on the next request.
	CacheTTL time.Duration
}

// RetryConfig specifies retry behaviour for failed fetches.
type RetryConfig struct {
	// RetryableErrors is a list of error types that should trigger a retry.
	//
	// If empty, all errors trigger retries.
	// Use this to avoid retrying on permanent failures (e.g., 404, auth errors).
	RetryableErrors []string

	// InitialDelay is the wait time before the first retry attempt.
	InitialDelay time.Duration

	// MaxDelay is the upper limit for delay between retries.
	// Used with exponential backoff to prevent the delay from growing too large.
	MaxDelay time.Duration

	// BackoffMultiplier is the factor by which the delay grows after each retry.
	//
	// For example, with BackoffMultiplier=2:
	// Retry 1: InitialDelay
	// Retry 2: InitialDelay * 2
	// Retry 3: InitialDelay * 4
	// and so on.
	BackoffMultiplier float64

	// MaxAttempts is the maximum number of fetch attempts.
	// Set to 0 to disable retries and fail on the first error.
	MaxAttempts int
}

// HasRetries returns true if retry behaviour is configured.
//
// Returns bool which is true when retry config exists with max attempts > 0.
func (r *RuntimeFetcherCode) HasRetries() bool {
	return r.RetryConfig != nil && r.RetryConfig.MaxAttempts > 0
}

// HasFallback reports whether a fallback function is configured.
//
// Returns bool which is true if a fallback function exists.
func (r *RuntimeFetcherCode) HasFallback() bool {
	return r.FallbackFunc != nil
}

// ShouldCache returns true if caching is enabled.
//
// Returns bool which is true when a cache strategy is set and is not "none".
func (r *RuntimeFetcherCode) ShouldCache() bool {
	return r.CacheStrategy != "" && r.CacheStrategy != "none"
}

// DefaultRetryConfig returns a sensible default retry configuration.
//
// Default settings:
//   - 3 retry attempts
//   - 100ms initial delay
//   - 5 second max delay
//   - 2x exponential backoff
//
// Returns *RetryConfig which contains the default retry settings.
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:       defaultRetryMaxAttempts,
		InitialDelay:      defaultRetryInitialDelayMs * time.Millisecond,
		MaxDelay:          defaultRetryMaxDelaySeconds * time.Second,
		BackoffMultiplier: defaultRetryBackoffMultiplier,
	}
}
