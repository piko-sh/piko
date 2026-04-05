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

package driver_registry

import (
	"context"
	"go/ast"
	"sync"
	"testing"

	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/collection/collection_dto"
)

type mockProvider struct {
	name         string
	providerType collection_domain.ProviderType
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Type() collection_domain.ProviderType {
	return m.providerType
}

func (m *mockProvider) DiscoverCollections(
	ctx context.Context,
	config collection_dto.ProviderConfig,
) ([]collection_dto.CollectionInfo, error) {
	return nil, nil
}

func (m *mockProvider) ValidateTargetType(targetType ast.Expr) error {
	return nil
}

func (m *mockProvider) FetchStaticContent(
	ctx context.Context,
	collectionName string,
	source collection_dto.ContentSource,
) ([]collection_dto.ContentItem, error) {
	return nil, nil
}

func (m *mockProvider) GenerateRuntimeFetcher(
	ctx context.Context,
	collectionName string,
	targetType ast.Expr,
	options collection_dto.FetchOptions,
) (*collection_dto.RuntimeFetcherCode, error) {
	return nil, nil
}

func (m *mockProvider) ComputeETag(
	ctx context.Context,
	collectionName string,
	source collection_dto.ContentSource,
) (string, error) {
	return "mock-etag", nil
}

func (m *mockProvider) ValidateETag(
	ctx context.Context,
	collectionName string,
	expectedETag string,
	source collection_dto.ContentSource,
) (string, bool, error) {
	return expectedETag, false, nil
}

func (m *mockProvider) GenerateRevalidator(
	ctx context.Context,
	collectionName string,
	targetType ast.Expr,
	config collection_dto.HybridConfig,
) (*collection_dto.RuntimeFetcherCode, error) {
	return nil, nil
}

func TestNewMemoryRegistry(t *testing.T) {
	registry := NewMemoryRegistry()

	if registry == nil {
		t.Fatal("NewMemoryRegistry returned nil")
	}

	if registry.providers == nil {
		t.Error("providers map was not initialised")
	}

	if count := registry.count(); count != 0 {
		t.Errorf("new registry should be empty, got count=%d", count)
	}
}

func TestRegister_Success(t *testing.T) {
	registry := NewMemoryRegistry()
	provider := &mockProvider{name: "markdown", providerType: collection_domain.ProviderTypeStatic}

	err := registry.Register(provider)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if count := registry.count(); count != 1 {
		t.Errorf("expected count=1 after registration, got %d", count)
	}

	if !registry.Has("markdown") {
		t.Error("registry should contain 'markdown' provider")
	}
}

func TestRegister_NilProvider(t *testing.T) {
	registry := NewMemoryRegistry()

	err := registry.Register(nil)
	if err == nil {
		t.Fatal("expected error when registering nil provider")
	}

	expectedMessage := "cannot register nil provider"
	if err.Error() != expectedMessage {
		t.Errorf("expected error message %q, got %q", expectedMessage, err.Error())
	}

	if count := registry.count(); count != 0 {
		t.Errorf("registry should be empty after failed registration, got count=%d", count)
	}
}

func TestRegister_EmptyName(t *testing.T) {
	registry := NewMemoryRegistry()
	provider := &mockProvider{name: "", providerType: collection_domain.ProviderTypeStatic}

	err := registry.Register(provider)
	if err == nil {
		t.Fatal("expected error when registering provider with empty name")
	}

	expectedMessage := "provider name cannot be empty"
	if err.Error() != expectedMessage {
		t.Errorf("expected error message %q, got %q", expectedMessage, err.Error())
	}

	if count := registry.count(); count != 0 {
		t.Errorf("registry should be empty after failed registration, got count=%d", count)
	}
}

func TestRegister_Duplicate(t *testing.T) {
	registry := NewMemoryRegistry()
	provider1 := &mockProvider{name: "markdown", providerType: collection_domain.ProviderTypeStatic}
	provider2 := &mockProvider{name: "markdown", providerType: collection_domain.ProviderTypeDynamic}

	err := registry.Register(provider1)
	if err != nil {
		t.Fatalf("first Register failed: %v", err)
	}

	err = registry.Register(provider2)
	if err == nil {
		t.Fatal("expected error when registering duplicate provider")
	}

	expectedMessage := "provider 'markdown' is already registered"
	if err.Error() != expectedMessage {
		t.Errorf("expected error message %q, got %q", expectedMessage, err.Error())
	}

	if count := registry.count(); count != 1 {
		t.Errorf("registry should have count=1 after duplicate attempt, got %d", count)
	}
}

func TestGet_Found(t *testing.T) {
	registry := NewMemoryRegistry()
	expected := &mockProvider{name: "cms", providerType: collection_domain.ProviderTypeDynamic}

	err := registry.Register(expected)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	provider, ok := registry.Get("cms")
	if !ok {
		t.Fatal("Get returned ok=false for registered provider")
	}

	if provider == nil {
		t.Fatal("Get returned nil provider")
	}

	if provider.Name() != expected.Name() {
		t.Errorf("expected provider name %q, got %q", expected.Name(), provider.Name())
	}

	if provider.Type() != expected.Type() {
		t.Errorf("expected provider type %q, got %q", expected.Type(), provider.Type())
	}
}

func TestGet_NotFound(t *testing.T) {
	registry := NewMemoryRegistry()

	provider, ok := registry.Get("nonexistent")
	if ok {
		t.Error("Get returned ok=true for nonexistent provider")
	}

	if provider != nil {
		t.Error("Get returned non-nil provider for nonexistent name")
	}
}

func TestList_Empty(t *testing.T) {
	registry := NewMemoryRegistry()

	names := registry.List()
	if names == nil {
		t.Fatal("List returned nil")
	}

	if len(names) != 0 {
		t.Errorf("expected empty list, got %d names", len(names))
	}
}

func TestList_Multiple(t *testing.T) {
	registry := NewMemoryRegistry()

	providers := []*mockProvider{
		{name: "markdown", providerType: collection_domain.ProviderTypeStatic},
		{name: "cms", providerType: collection_domain.ProviderTypeDynamic},
		{name: "database", providerType: collection_domain.ProviderTypeHybrid},
	}

	for _, p := range providers {
		if err := registry.Register(p); err != nil {
			t.Fatalf("Register failed: %v", err)
		}
	}

	names := registry.List()
	if len(names) != 3 {
		t.Fatalf("expected 3 provider names, got %d", len(names))
	}

	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[name] = true
	}

	for _, p := range providers {
		if !nameMap[p.name] {
			t.Errorf("expected provider %q to be in list", p.name)
		}
	}
}

func TestHas(t *testing.T) {
	registry := NewMemoryRegistry()
	provider := &mockProvider{name: "markdown", providerType: collection_domain.ProviderTypeStatic}

	if registry.Has("markdown") {
		t.Error("Has returned true for unregistered provider")
	}

	err := registry.Register(provider)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if !registry.Has("markdown") {
		t.Error("Has returned false for registered provider")
	}

	if registry.Has("nonexistent") {
		t.Error("Has returned true for non-existent provider")
	}
}

func Test_count(t *testing.T) {
	registry := NewMemoryRegistry()

	if count := registry.count(); count != 0 {
		t.Errorf("expected count=0 for empty registry, got %d", count)
	}

	providers := []string{"markdown", "cms", "database", "api"}
	for _, name := range providers {
		p := &mockProvider{name: name, providerType: collection_domain.ProviderTypeStatic}
		if err := registry.Register(p); err != nil {
			t.Fatalf("Register failed: %v", err)
		}
	}

	if count := registry.count(); count != len(providers) {
		t.Errorf("expected count=%d, got %d", len(providers), count)
	}
}

func Test_clear(t *testing.T) {
	registry := NewMemoryRegistry()

	for i := range 5 {
		p := &mockProvider{
			name:         string(rune('a' + i)),
			providerType: collection_domain.ProviderTypeStatic,
		}
		if err := registry.Register(p); err != nil {
			t.Fatalf("Register failed: %v", err)
		}
	}

	if count := registry.count(); count != 5 {
		t.Fatalf("expected count=5 before clear, got %d", count)
	}

	registry.clear()

	if count := registry.count(); count != 0 {
		t.Errorf("expected count=0 after clear, got %d", count)
	}

	if names := registry.List(); len(names) != 0 {
		t.Errorf("expected empty list after clear, got %d names", len(names))
	}

	p := &mockProvider{name: "newprovider", providerType: collection_domain.ProviderTypeStatic}
	if err := registry.Register(p); err != nil {
		t.Errorf("Register failed after clear: %v", err)
	}

	if count := registry.count(); count != 1 {
		t.Errorf("expected count=1 after registering post-clear, got %d", count)
	}
}

func TestConcurrentRegister(t *testing.T) {
	registry := NewMemoryRegistry()
	const numGoroutines = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			p := &mockProvider{
				name:         string(rune('A' + id)),
				providerType: collection_domain.ProviderTypeStatic,
			}
			_ = registry.Register(p)
		}(i)
	}

	wg.Wait()

	if count := registry.count(); count != numGoroutines {
		t.Errorf("expected count=%d after concurrent registration, got %d", numGoroutines, count)
	}
}

func TestConcurrentGet(t *testing.T) {
	registry := NewMemoryRegistry()

	for i := range 10 {
		p := &mockProvider{
			name:         string(rune('a' + i)),
			providerType: collection_domain.ProviderTypeStatic,
		}
		if err := registry.Register(p); err != nil {
			t.Fatalf("Register failed: %v", err)
		}
	}

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			name := string(rune('a' + (id % 10)))
			provider, ok := registry.Get(name)
			if !ok {
				t.Errorf("Get returned ok=false for registered provider %q", name)
			}
			if provider == nil {
				t.Errorf("Get returned nil provider for %q", name)
			}
		}(i)
	}

	wg.Wait()
}

func TestConcurrentMixedOperations(t *testing.T) {
	registry := NewMemoryRegistry()

	for i := range 5 {
		p := &mockProvider{
			name:         string(rune('a' + i)),
			providerType: collection_domain.ProviderTypeStatic,
		}
		if err := registry.Register(p); err != nil {
			t.Fatalf("Register failed: %v", err)
		}
	}

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			switch id % 4 {
			case 0:
				name := string(rune('a' + (id % 5)))
				_, _ = registry.Get(name)
			case 1:
				_ = registry.List()
			case 2:
				name := string(rune('a' + (id % 5)))
				_ = registry.Has(name)
			case 3:
				_ = registry.count()
			}
		}(i)
	}

	wg.Wait()

	if count := registry.count(); count != 5 {
		t.Errorf("expected count=5 after concurrent mixed operations, got %d", count)
	}
}
