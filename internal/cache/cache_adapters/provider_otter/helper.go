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

package provider_otter

import (
	"context"
	"fmt"
	"reflect"

	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// createOtterCacheGeneric uses type assertions to call the Otter cache factory
// with the correct types, making it possible to work with type-erased
// Options[K, V] and still create properly typed caches.
//
// Takes namespace (string) which identifies the cache namespace.
// Takes optionsAny (any) which provides type-erased cache options.
//
// Returns any which is the created cache instance.
// Returns error when the factory function fails to create the cache.
func createOtterCacheGeneric(ctx context.Context, namespace string, optionsAny any) (any, error) {
	cache, err := callOtterFactory(optionsAny)
	if err != nil {
		return nil, fmt.Errorf("creating otter cache for namespace %q: %w", namespace, err)
	}

	_, l := logger_domain.From(ctx, log)
	l.Internal("Created new Otter namespace",
		logger_domain.String("namespace", namespace))

	return cache, nil
}

// callOtterFactory calls OtterProviderFactory using type assertion on the
// options parameter. It falls back to reflection for unsupported type pairs.
//
// Takes optionsAny (any) which contains cache options with generic type
// parameters.
//
// Returns any which is the created cache provider.
// Returns error when the factory fails to create the provider.
func callOtterFactory(optionsAny any) (any, error) {
	switch opts := optionsAny.(type) {
	case cache_dto.Options[string, []byte]:
		return OtterProviderFactory[string, []byte](opts)
	case cache_dto.Options[string, string]:
		return OtterProviderFactory[string, string](opts)
	case cache_dto.Options[string, int64]:
		return OtterProviderFactory[string, int64](opts)
	case cache_dto.Options[int, string]:
		return OtterProviderFactory[int, string](opts)
	case cache_dto.Options[string, any]:
		return OtterProviderFactory[string, any](opts)
	case cache_dto.Options[int, int]:
		return OtterProviderFactory[int, int](opts)
	case cache_dto.Options[int, any]:
		return OtterProviderFactory[int, any](opts)
	default:
		return callFactoryViaReflection(optionsAny)
	}
}

// callFactoryViaReflection returns an error for unsupported cache types.
//
// Go's type system makes it hard to create generic functions using reflection.
// For custom domain types, there are two suggested approaches:
//
// 1. Use Cache[string, any] with type assertions in the adapter layer.
// The current approach for ArtefactMeta. It allows resource sharing,
// avoids circular dependencies, and is simple. The trade-off is that type
// safety happens at runtime rather than compile time.
//
// 2. Register a ProviderFactoryBlueprint using
// cache_domain.RegisterProviderFactory. This gives full compile-time type
// safety and works with CacheBuilder, but needs init() registration.
//
// For most cases, approach one is best. The type assertions stay in a single
// adapter layer, and users of that adapter still get full compile-time type
// safety.
//
// Takes optionsAny (any) which is the cache options to check.
//
// Returns any which is always nil; only the error result is meaningful.
// Returns error when the type is not in the predefined type switch.
func callFactoryViaReflection(optionsAny any) (any, error) {
	optionsType := reflect.TypeOf(optionsAny)
	return nil, fmt.Errorf(
		"unsupported cache type %v - this type is not in the predefined type switch. "+
			"\n\nRecommended solution: Use Cache[string, any] with type assertions in your adapter. "+
			"See internal/registry/registry_adapters/driven_metadata_cache.go for an example. "+
			"\n\nAlternative: Register a ProviderFactoryBlueprint via cache_domain.RegisterProviderFactory()",
		optionsType,
	)
}
