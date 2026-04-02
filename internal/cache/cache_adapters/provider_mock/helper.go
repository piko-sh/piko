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

package provider_mock

import (
	"errors"
	"fmt"
	"reflect"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
)

// createMockCache creates a mock cache for the given namespace using type
// assertions.
//
// Takes p (*MockProvider) which provides the mock cache storage.
// Takes namespace (string) which identifies the cache namespace.
// Takes optionsAny (any) which specifies the cache options with type info.
//
// Returns any which is the created cache instance.
// Returns error when the options type is not supported.
func createMockCache(p *MockProvider, namespace string, optionsAny any) (any, error) {
	switch opts := optionsAny.(type) {
	case cache_dto.Options[string, []byte]:
		return createNamespaceGeneric[string, []byte](p, namespace, opts)
	case cache_dto.Options[string, string]:
		return createNamespaceGeneric[string, string](p, namespace, opts)
	case cache_dto.Options[int, string]:
		return createNamespaceGeneric[int, string](p, namespace, opts)
	case cache_dto.Options[string, any]:
		return createNamespaceGeneric[string, any](p, namespace, opts)
	case cache_dto.Options[int, int]:
		return createNamespaceGeneric[int, int](p, namespace, opts)
	case cache_dto.Options[int, any]:
		return createNamespaceGeneric[int, any](p, namespace, opts)
	default:
		optionsType := reflect.TypeOf(optionsAny)
		return nil, fmt.Errorf(
			"unsupported cache type %v - for custom domain types, register a ProviderFactoryBlueprint via cache_domain.RegisterProviderFactory",
			optionsType,
		)
	}
}

// createNamespaceGeneric is a helper that handles the type-specific mock cache
// creation.
//
// Takes p (*MockProvider) which provides the mock cache storage.
// Takes namespace (string) which identifies the cache namespace.
//
// Returns cache_domain.Cache[K, V] which is the mock cache
// instance.
// Returns error when the provider is closed or a namespace
// exists with different types.
//
// Safe for concurrent use. Access is serialised by an internal
// mutex.
func createNamespaceGeneric[K comparable, V any](p *MockProvider, namespace string, _ cache_dto.Options[K, V]) (cache_domain.Cache[K, V], error) {
	if namespace == "" {
		namespace = "default"
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, errors.New("provider is closed")
	}

	if existing, exists := p.namespaces[namespace]; exists {
		if cache, ok := existing.(cache_domain.Cache[K, V]); ok {
			return cache, nil
		}
		return nil, fmt.Errorf("namespace '%s' already exists with different key/value types", namespace)
	}

	cache := NewMockAdapter[K, V]()
	p.namespaces[namespace] = cache

	return cache, nil
}
