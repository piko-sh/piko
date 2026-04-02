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

package ratelimiter_adapters

import (
	"fmt"

	cache_adapters_otter "piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
)

// createTokenBucketCache creates an Otter-backed cache for token
// bucket rate limiter state.
//
// Takes options (any) which must be
// cache_dto.Options[string, *ratelimiter_domain.TokenBucketState].
//
// Returns any which is the constructed cache provider.
// Returns error when the options type is wrong or cache creation fails.
func createTokenBucketCache(
	_ cache_domain.Service,
	_ string,
	options any,
) (any, error) {
	opts, ok := options.(cache_dto.Options[string, *ratelimiter_domain.TokenBucketState])
	if !ok {
		return nil, fmt.Errorf(
			"invalid options type for token bucket cache: expected cache_dto.Options[string, *ratelimiter_domain.TokenBucketState], got %T",
			options,
		)
	}

	cache, err := cache_adapters_otter.OtterProviderFactory[string, *ratelimiter_domain.TokenBucketState](opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create token bucket cache: %w", err)
	}

	return cache, nil
}

// createCounterCache creates an Otter-backed cache for counter-based
// rate limiter entries.
//
// Takes options (any) which must be
// cache_dto.Options[string, *counterEntry].
//
// Returns any which is the constructed cache provider.
// Returns error when the options type is wrong or cache creation fails.
func createCounterCache(
	_ cache_domain.Service,
	_ string,
	options any,
) (any, error) {
	opts, ok := options.(cache_dto.Options[string, *counterEntry])
	if !ok {
		return nil, fmt.Errorf(
			"invalid options type for counter cache: expected cache_dto.Options[string, *counterEntry], got %T",
			options,
		)
	}

	cache, err := cache_adapters_otter.OtterProviderFactory[string, *counterEntry](opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create counter cache: %w", err)
	}

	return cache, nil
}

func init() {
	cache_domain.RegisterProviderFactory(
		"ratelimiter-token-bucket",
		createTokenBucketCache,
	)
	cache_domain.RegisterProviderFactory(
		"ratelimiter-counter",
		createCounterCache,
	)
}
