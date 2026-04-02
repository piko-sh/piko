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

package cache_provider_mock

import (
	"piko.sh/piko/internal/cache/cache_adapters/provider_mock"
	"piko.sh/piko/wdk/cache"
)

// NewMockProvider creates a new mock provider using the namespace pattern.
// This is the recommended way to use mock caches in tests.
//
// The provider maintains a single in-memory map shared across all namespaces.
// Each namespace becomes a key prefix (e.g., "users:", "products:").
//
// Returns cache.Provider which is the configured mock provider ready for use.
//
// Example:
//
//	service := cache.NewService("mock")
//	provider := cache_provider_mock.NewMockProvider()
//	service.RegisterProvider("mock", provider)
//
//	// Create multiple namespaced caches (all sharing ONE map)
//	userCache, _ := cache.NewCacheBuilder[string, User](service).
//	    WithProvider("mock").
//	    WithNamespace("users").
//	    Build(ctx)
//
//	productCache, _ := cache.NewCacheBuilder[int, Product](service).
//	    WithProvider("mock").
//	    WithNamespace("products").
//	    Build(ctx)
func NewMockProvider() cache.Provider {
	return provider_mock.NewMockProvider()
}

// NewMockAdapter creates a standalone mock adapter without a cache service,
// suitable for unit testing individual components that depend on a cache.
//
// Returns *MockAdapter[K, V] which is an in-memory cache ready for use in
// tests.
//
// Example:
//
//	// In unit test
//	mockCache := cache_provider_mock.NewMockAdapter[string, User]()
//	mockCache.Set("alice", User{Name: "Alice"})
//
//	// Wrap with Cache facade for full API
//	cache := cache.NewCacheFromAdapter(mockCache, "test")
func NewMockAdapter[K comparable, V any]() *provider_mock.MockAdapter[K, V] {
	return provider_mock.NewMockAdapter[K, V]()
}
