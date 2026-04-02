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

package cache_provider_multilevel

import (
	"context"

	"piko.sh/piko/internal/cache/cache_adapters/provider_multilevel"
	"piko.sh/piko/wdk/cache"
)

// Config holds the settings for the multilevel cache provider.
// This is re-exported from the internal adapter package.
type Config = provider_multilevel.Config

// NewMultiLevelAdapter creates a new multi-level cache provider that
// coordinates between an L1 (local/fast) and L2 (remote/distributed) cache.
//
// The adapter automatically:
//   - Checks L1 first for the fastest possible lookups
//   - Falls back to L2 on L1 misses
//   - Back-populates L1 when data is found in L2
//   - Protects against L2 failures using a circuit breaker
//
// Takes ctx (context.Context) which carries logging context for trace and
// request ID propagation.
// Takes name (string) which is a descriptive name for this multilevel
// cache instance, used in metrics.
// Takes l1 (cache.ProviderPort[K, V]) which is the L1 local cache
// provider (e.g. Otter).
// Takes l2 (cache.ProviderPort[K, V]) which is the L2 remote cache
// provider (e.g. Redis).
// Takes cbConfig (Config) which configures the circuit breaker for L2
// resilience.
//
// Returns the configured multilevel cache provider.
//
// Example:
//
//	multilevelCache := cache_provider_multilevel.NewMultiLevelAdapter(
//	    ctx,
//	    "user-cache",
//	    l1Cache,
//	    l2Cache,
//	    cache_provider_multilevel.Config{
//	        MaxConsecutiveFailures: 5,
//	        OpenStateTimeout:       30 * time.Second,
//	    },
//	)
func NewMultiLevelAdapter[K comparable, V any](
	ctx context.Context,
	name string,
	l1 cache.ProviderPort[K, V],
	l2 cache.ProviderPort[K, V],
	cbConfig Config,
) cache.ProviderPort[K, V] {
	return provider_multilevel.NewMultiLevelAdapter[K, V](ctx, name, l1, l2, cbConfig)
}
