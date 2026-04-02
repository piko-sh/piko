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

package provider_multilevel

import (
	"context"
	"errors"
	"fmt"
	"time"

	"piko.sh/piko/internal/cache/cache_domain"
)

// createMultiLevelAdapterGeneric is a helper that creates a multi-level adapter
// with type-erased parameters and returns it as an interface{}.
// The builder will type-assert it back to Cache[K, V].
//
// Takes name (string) which identifies the adapter.
// Takes l1 (any) which is the first level cache with baked-in types.
// Takes l2 (any) which is the second level cache with baked-in types.
// Takes config (Config) which specifies the adapter configuration.
//
// Returns any which is the constructed multi-level adapter.
// Returns error when the reflection-based construction fails.
func createMultiLevelAdapterGeneric(ctx context.Context, name string, l1, l2 any, config Config) (any, error) {
	return createMultiLevelAdapterReflection(ctx, name, l1, l2, config)
}

// createMultiLevelAdapterReflection uses reflection to create a correctly-typed
// multi-level adapter without knowing K and V at compile time.
//
// Takes name (string) which identifies the adapter.
// Takes l1 (any) which is the first level ProviderPort[K, V].
// Takes l2 (any) which is the second level ProviderPort[K, V].
// Takes config (Config) which specifies the adapter settings.
//
// Returns any which is the multi-level adapter for a supported type combination.
// Returns error when the provider type combination is not supported.
func createMultiLevelAdapterReflection(ctx context.Context, name string, l1, l2 any, config Config) (any, error) {
	if l1Provider, ok := l1.(cache_domain.ProviderPort[string, any]); ok {
		if l2Provider, ok := l2.(cache_domain.ProviderPort[string, any]); ok {
			adapter := NewMultiLevelAdapter[string, any](ctx, name, l1Provider, l2Provider, config)
			return adapter, nil
		}
	}

	if l1Provider, ok := l1.(cache_domain.ProviderPort[string, string]); ok {
		if l2Provider, ok := l2.(cache_domain.ProviderPort[string, string]); ok {
			adapter := NewMultiLevelAdapter[string, string](ctx, name, l1Provider, l2Provider, config)
			return adapter, nil
		}
	}

	if l1Provider, ok := l1.(cache_domain.ProviderPort[string, []byte]); ok {
		if l2Provider, ok := l2.(cache_domain.ProviderPort[string, []byte]); ok {
			adapter := NewMultiLevelAdapter[string, []byte](ctx, name, l1Provider, l2Provider, config)
			return adapter, nil
		}
	}

	if l1Provider, ok := l1.(cache_domain.ProviderPort[int, any]); ok {
		if l2Provider, ok := l2.(cache_domain.ProviderPort[int, any]); ok {
			adapter := NewMultiLevelAdapter[int, any](ctx, name, l1Provider, l2Provider, config)
			return adapter, nil
		}
	}

	return nil, errors.New("unsupported provider type combination for multi-level cache - manual construction required")
}

func init() {
	cache_domain.RegisterMultiLevelAdapterConstructor(func(ctx context.Context, name string, l1, l2, config any) (any, error) {
		configMap, ok := config.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid config type: expected map[string]any, got %T", config)
		}

		multilevelConfig := Config{
			L1ProviderName:         "",
			L2ProviderName:         "",
			MaxConsecutiveFailures: 5,
			OpenStateTimeout:       30,
		}

		if value, ok := configMap["l1_provider"].(string); ok {
			multilevelConfig.L1ProviderName = value
		}
		if value, ok := configMap["l2_provider"].(string); ok {
			multilevelConfig.L2ProviderName = value
		}
		if value, ok := configMap["l2_max_failures"].(int); ok {
			multilevelConfig.MaxConsecutiveFailures = value
		}
		if value, ok := configMap["l2_open_timeout"]; ok {
			switch v := value.(type) {
			case time.Duration:
				multilevelConfig.OpenStateTimeout = v
			case int64:
				multilevelConfig.OpenStateTimeout = time.Duration(v)
			default:
				return nil, fmt.Errorf("invalid l2_open_timeout type: expected time.Duration or int64, got %T", value)
			}
		}

		return createMultiLevelAdapterGeneric(ctx, name, l1, l2, multilevelConfig)
	})
}
