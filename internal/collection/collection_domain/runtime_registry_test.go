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

package collection_domain

import (
	"context"
	"errors"
	"sync"
	"testing"

	"piko.sh/piko/internal/collection/collection_dto"
)

func TestRegisterProvider(t *testing.T) {
	ResetRuntimeProviderRegistry()

	provider := &MockRuntimeProvider{
		NameFunc: func() string { return "test-provider" },
	}

	err := RegisterProvider(provider)
	if err != nil {
		t.Fatalf("RegisterProvider() failed: %v", err)
	}

	retrieved, err := GetProvider("test-provider")
	if err != nil {
		t.Fatalf("GetProvider() failed: %v", err)
	}

	if retrieved.Name() != "test-provider" {
		t.Errorf("Expected provider name 'test-provider', got %q", retrieved.Name())
	}
}

func TestRegisterProvider_DuplicateFails(t *testing.T) {
	ResetRuntimeProviderRegistry()

	provider1 := &MockRuntimeProvider{
		NameFunc: func() string { return "duplicate" },
	}
	provider2 := &MockRuntimeProvider{
		NameFunc: func() string { return "duplicate" },
	}

	err := RegisterProvider(provider1)
	if err != nil {
		t.Fatalf("First RegisterProvider() failed: %v", err)
	}

	err = RegisterProvider(provider2)
	if err == nil {
		t.Fatal("Expected error for duplicate registration, got nil")
	}

	expectedMessage := "runtime provider 'duplicate' already registered"
	if err.Error() != expectedMessage {
		t.Errorf("Expected error message %q, got %q", expectedMessage, err.Error())
	}
}

func TestRegisterProvider_MultipleProviders(t *testing.T) {
	ResetRuntimeProviderRegistry()

	providers := []string{"provider-a", "provider-b", "provider-c"}

	for _, name := range providers {
		n := name
		err := RegisterProvider(&MockRuntimeProvider{
			NameFunc: func() string { return n },
		})
		if err != nil {
			t.Fatalf("RegisterProvider(%q) failed: %v", name, err)
		}
	}

	for _, name := range providers {
		_, err := GetProvider(name)
		if err != nil {
			t.Errorf("GetProvider(%q) failed: %v", name, err)
		}
	}
}

func TestGetProvider_NotFound(t *testing.T) {
	ResetRuntimeProviderRegistry()

	_, err := GetProvider("nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent provider, got nil")
	}

	if !containsString(err.Error(), "nonexistent") {
		t.Errorf("Error message should contain provider name, got: %v", err)
	}
}

func TestGetProvider_Success(t *testing.T) {
	ResetRuntimeProviderRegistry()

	provider := &MockRuntimeProvider{
		NameFunc: func() string { return "my-provider" },
	}
	if err := RegisterProvider(provider); err != nil {
		t.Fatalf("RegisterProvider() failed: %v", err)
	}

	retrieved, err := GetProvider("my-provider")
	if err != nil {
		t.Fatalf("GetProvider() failed: %v", err)
	}

	if retrieved != provider {
		t.Error("GetProvider() returned different instance than registered")
	}
}

func TestFetchCollection_Success(t *testing.T) {
	ResetRuntimeProviderRegistry()

	fetchCalled := false
	provider := &MockRuntimeProvider{
		NameFunc: func() string { return "fetch-provider" },
		FetchFunc: func(_ context.Context, collectionName string, _ *collection_dto.FetchOptions, _ any) error {
			fetchCalled = true
			if collectionName != "blog" {
				t.Errorf("Expected collection 'blog', got %q", collectionName)
			}
			return nil
		},
	}

	if err := RegisterProvider(provider); err != nil {
		t.Fatalf("RegisterProvider() failed: %v", err)
	}

	var target []any
	err := FetchCollection(context.Background(), "fetch-provider", "blog", nil, &target)
	if err != nil {
		t.Fatalf("FetchCollection() failed: %v", err)
	}

	if !fetchCalled {
		t.Error("Provider Fetch() was not called")
	}
}

func TestFetchCollection_ProviderNotFound(t *testing.T) {
	ResetRuntimeProviderRegistry()

	var target []any
	err := FetchCollection(context.Background(), "nonexistent", "blog", nil, &target)
	if err == nil {
		t.Error("Expected error for nonexistent provider, got nil")
	}
}

func TestFetchCollection_ProviderFetchError(t *testing.T) {
	ResetRuntimeProviderRegistry()

	expectedErr := errors.New("fetch failed")
	provider := &MockRuntimeProvider{
		NameFunc: func() string { return "error-provider" },
		FetchFunc: func(_ context.Context, _ string, _ *collection_dto.FetchOptions, _ any) error {
			return expectedErr
		},
	}

	if err := RegisterProvider(provider); err != nil {
		t.Fatalf("RegisterProvider() failed: %v", err)
	}

	var target []any
	err := FetchCollection(context.Background(), "error-provider", "blog", nil, &target)
	if err == nil {
		t.Error("Expected error from provider fetch, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected wrapped error containing %v, got %v", expectedErr, err)
	}
}

func TestFetchCollection_WithOptions(t *testing.T) {
	ResetRuntimeProviderRegistry()

	var receivedOptions *collection_dto.FetchOptions
	provider := &MockRuntimeProvider{
		NameFunc: func() string { return "options-provider" },
		FetchFunc: func(_ context.Context, _ string, options *collection_dto.FetchOptions, _ any) error {
			receivedOptions = options
			return nil
		},
	}

	if err := RegisterProvider(provider); err != nil {
		t.Fatalf("RegisterProvider() failed: %v", err)
	}

	options := &collection_dto.FetchOptions{
		Locale: "en-GB",
	}

	var target []any
	err := FetchCollection(context.Background(), "options-provider", "blog", options, &target)
	if err != nil {
		t.Fatalf("FetchCollection() failed: %v", err)
	}

	if receivedOptions == nil {
		t.Fatal("Provider did not receive options")
	}

	if receivedOptions.Locale != "en-GB" {
		t.Errorf("Expected locale 'en-GB', got %q", receivedOptions.Locale)
	}
}

func TestDefaultRuntimeProviderRegistry_Register(t *testing.T) {
	ResetRuntimeProviderRegistry()

	registry := NewDefaultRuntimeProviderRegistry()

	provider := &MockRuntimeProvider{
		NameFunc: func() string { return "default-test" },
	}
	err := registry.Register(provider)
	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	_, err = GetProvider("default-test")
	if err != nil {
		t.Errorf("Provider not accessible via GetProvider: %v", err)
	}
}

func TestDefaultRuntimeProviderRegistry_Get(t *testing.T) {
	ResetRuntimeProviderRegistry()

	provider := &MockRuntimeProvider{
		NameFunc: func() string { return "get-test" },
	}
	if err := RegisterProvider(provider); err != nil {
		t.Fatalf("RegisterProvider() failed: %v", err)
	}

	registry := NewDefaultRuntimeProviderRegistry()
	retrieved, err := registry.Get("get-test")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if retrieved.Name() != "get-test" {
		t.Errorf("Expected name 'get-test', got %q", retrieved.Name())
	}
}

func TestDefaultRuntimeProviderRegistry_List(t *testing.T) {
	ResetRuntimeProviderRegistry()

	providers := []string{"list-a", "list-b", "list-c"}
	for _, name := range providers {
		n := name
		if err := RegisterProvider(&MockRuntimeProvider{
			NameFunc: func() string { return n },
		}); err != nil {
			t.Fatalf("RegisterProvider() failed: %v", err)
		}
	}

	registry := NewDefaultRuntimeProviderRegistry()
	names := registry.List()

	if len(names) != len(providers) {
		t.Errorf("Expected %d providers, got %d", len(providers), len(names))
	}

	nameMap := make(map[string]bool)
	for _, n := range names {
		nameMap[n] = true
	}

	for _, expected := range providers {
		if !nameMap[expected] {
			t.Errorf("Expected provider %q in list", expected)
		}
	}
}

func TestDefaultRuntimeProviderRegistry_Has(t *testing.T) {
	ResetRuntimeProviderRegistry()

	registry := NewDefaultRuntimeProviderRegistry()

	if registry.Has("nonexistent") {
		t.Error("Has() returned true for nonexistent provider")
	}

	if err := RegisterProvider(&MockRuntimeProvider{
		NameFunc: func() string { return "has-test" },
	}); err != nil {
		t.Fatalf("RegisterProvider() failed: %v", err)
	}

	if !registry.Has("has-test") {
		t.Error("Has() returned false for existing provider")
	}
}

func TestDefaultRuntimeProviderRegistry_Fetch(t *testing.T) {
	ResetRuntimeProviderRegistry()

	fetchCalled := false
	provider := &MockRuntimeProvider{
		NameFunc: func() string { return "registry-fetch" },
		FetchFunc: func(_ context.Context, _ string, _ *collection_dto.FetchOptions, _ any) error {
			fetchCalled = true
			return nil
		},
	}

	if err := RegisterProvider(provider); err != nil {
		t.Fatalf("RegisterProvider() failed: %v", err)
	}

	registry := NewDefaultRuntimeProviderRegistry()
	var target []any
	err := registry.Fetch(context.Background(), "registry-fetch", "collection", nil, &target)
	if err != nil {
		t.Fatalf("Fetch() failed: %v", err)
	}

	if !fetchCalled {
		t.Error("Provider Fetch() was not called")
	}
}

func TestRegisterProvider_Concurrent(t *testing.T) {
	ResetRuntimeProviderRegistry()

	var wg sync.WaitGroup
	numGoroutines := 50

	for i := range numGoroutines {
		index := i
		wg.Go(func() {
			n := "concurrent-" + string(rune('A'+index%26))
			provider := &MockRuntimeProvider{
				NameFunc: func() string { return n },
			}

			_ = RegisterProvider(provider)
		})
	}

	wg.Wait()

	registry := NewDefaultRuntimeProviderRegistry()
	names := registry.List()
	if len(names) == 0 {
		t.Error("Expected at least one provider to be registered")
	}
}

func TestGetProvider_Concurrent(t *testing.T) {
	ResetRuntimeProviderRegistry()

	if err := RegisterProvider(&MockRuntimeProvider{
		NameFunc: func() string { return "concurrent-get" },
	}); err != nil {
		t.Fatalf("RegisterProvider() failed: %v", err)
	}

	var wg sync.WaitGroup
	numGoroutines := 100

	for range numGoroutines {
		wg.Go(func() {
			provider, err := GetProvider("concurrent-get")
			if err != nil {
				t.Errorf("GetProvider() failed: %v", err)
				return
			}
			if provider.Name() != "concurrent-get" {
				t.Errorf("Got wrong provider name: %q", provider.Name())
			}
		})
	}

	wg.Wait()
}

func TestFetchCollection_Concurrent(t *testing.T) {
	ResetRuntimeProviderRegistry()

	var fetchCount int
	var mu sync.Mutex

	provider := &MockRuntimeProvider{
		NameFunc: func() string { return "concurrent-fetch" },
		FetchFunc: func(_ context.Context, _ string, _ *collection_dto.FetchOptions, _ any) error {
			mu.Lock()
			fetchCount++
			mu.Unlock()
			return nil
		},
	}

	if err := RegisterProvider(provider); err != nil {
		t.Fatalf("RegisterProvider() failed: %v", err)
	}

	var wg sync.WaitGroup
	numGoroutines := 100

	for range numGoroutines {
		wg.Go(func() {
			var target []any
			if err := FetchCollection(context.Background(), "concurrent-fetch", "blog", nil, &target); err != nil {
				t.Errorf("FetchCollection() failed: %v", err)
			}
		})
	}

	wg.Wait()

	mu.Lock()
	if fetchCount != numGoroutines {
		t.Errorf("Expected %d fetch calls, got %d", numGoroutines, fetchCount)
	}
	mu.Unlock()
}

func TestResetRuntimeProviderRegistry(t *testing.T) {
	ResetRuntimeProviderRegistry()

	for _, name := range []string{"reset-a", "reset-b"} {
		n := name
		if err := RegisterProvider(&MockRuntimeProvider{
			NameFunc: func() string { return n },
		}); err != nil {
			t.Fatalf("RegisterProvider() failed: %v", err)
		}
	}

	registry := NewDefaultRuntimeProviderRegistry()
	if len(registry.List()) != 2 {
		t.Fatal("Expected 2 providers before reset")
	}

	ResetRuntimeProviderRegistry()

	if len(registry.List()) != 0 {
		t.Error("Expected 0 providers after reset")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
