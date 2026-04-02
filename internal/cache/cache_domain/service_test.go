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

package cache_domain

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

type mockTestProvider struct {
	createErr  error
	namespaces map[string]any
	name       string
	mu         sync.RWMutex
}

func newMockTestProvider(name string) *mockTestProvider {
	return &mockTestProvider{
		name:       name,
		namespaces: make(map[string]any),
	}
}

func newMockTestProviderWithError(name string, err error) *mockTestProvider {
	return &mockTestProvider{
		name:       name,
		namespaces: make(map[string]any),
		createErr:  err,
	}
}

func (p *mockTestProvider) CreateNamespaceTyped(namespace string, options any) (any, error) {
	if p.createErr != nil {
		return nil, p.createErr
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if namespace == "" {
		namespace = "default"
	}

	if existing, exists := p.namespaces[namespace]; exists {
		return existing, nil
	}

	switch opts := options.(type) {
	case cache_dto.Options[string, string]:
		cache := &mockCache[string, string]{data: make(map[string]string)}
		p.namespaces[namespace] = cache
		_ = opts
		return cache, nil
	case cache_dto.Options[int, int]:
		cache := &mockCache[int, int]{data: make(map[int]int)}
		p.namespaces[namespace] = cache
		_ = opts
		return cache, nil
	case cache_dto.Options[string, []byte]:
		cache := &mockCache[string, []byte]{data: make(map[string][]byte)}
		p.namespaces[namespace] = cache
		_ = opts
		return cache, nil
	default:
		return nil, fmt.Errorf("unsupported options type: %T", options)
	}
}

func (p *mockTestProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.namespaces = make(map[string]any)
	return nil
}

func (p *mockTestProvider) Name() string {
	return p.name
}

type mockCache[K comparable, V any] struct {
	data map[K]V
	mu   sync.RWMutex
}

func (m *mockCache[K, V]) GetIfPresent(_ context.Context, key K) (V, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[key]
	return v, ok, nil
}

func (m *mockCache[K, V]) Get(ctx context.Context, key K, loader cache_dto.Loader[K, V]) (V, error) {
	v, ok, _ := m.GetIfPresent(ctx, key)
	if ok {
		return v, nil
	}
	if loader == nil {
		var zero V
		return zero, errors.New("key not found and no loader provided")
	}
	return loader.Load(ctx, key)
}

func (m *mockCache[K, V]) Set(_ context.Context, key K, value V, tags ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	return nil
}

func (m *mockCache[K, V]) SetWithTTL(ctx context.Context, key K, value V, ttl time.Duration, tags ...string) error {
	_ = m.Set(ctx, key, value, tags...)
	return nil
}

func (m *mockCache[K, V]) BulkSet(ctx context.Context, items map[K]V, tags ...string) error {
	for key, value := range items {
		_ = m.Set(ctx, key, value, tags...)
	}
	return nil
}

func (m *mockCache[K, V]) Invalidate(_ context.Context, key K) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *mockCache[K, V]) InvalidateAll(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[K]V)
	return nil
}

func (m *mockCache[K, V]) Close(_ context.Context) error { return nil }

func (m *mockCache[K, V]) BulkGet(ctx context.Context, keys []K, bulkLoader cache_dto.BulkLoader[K, V]) (map[K]V, error) {
	return nil, nil
}
func (m *mockCache[K, V]) BulkRefresh(ctx context.Context, keys []K, bulkLoader cache_dto.BulkLoader[K, V]) {
}
func (m *mockCache[K, V]) Refresh(ctx context.Context, key K, loader cache_dto.Loader[K, V]) <-chan cache_dto.LoadResult[V] {
	return nil
}
func (m *mockCache[K, V]) Compute(_ context.Context, key K, computeFunction func(oldValue V, found bool) (newValue V, action cache_dto.ComputeAction)) (V, bool, error) {
	var zero V
	return zero, false, nil
}
func (m *mockCache[K, V]) ComputeIfAbsent(_ context.Context, key K, computeFunction func() V) (V, bool, error) {
	var zero V
	return zero, false, nil
}
func (m *mockCache[K, V]) ComputeIfPresent(_ context.Context, key K, computeFunction func(oldValue V) (newValue V, action cache_dto.ComputeAction)) (V, bool, error) {
	var zero V
	return zero, false, nil
}
func (m *mockCache[K, V]) ComputeWithTTL(_ context.Context, key K, computeFunction func(oldValue V, found bool) cache_dto.ComputeResult[V]) (V, bool, error) {
	var zero V
	return zero, false, nil
}
func (m *mockCache[K, V]) InvalidateByTags(_ context.Context, tags ...string) (int, error) {
	return 0, nil
}
func (m *mockCache[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		m.mu.RLock()
		defer m.mu.RUnlock()
		for k, v := range m.data {
			if !yield(k, v) {
				return
			}
		}
	}
}
func (m *mockCache[K, V]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		m.mu.RLock()
		defer m.mu.RUnlock()
		for k := range m.data {
			if !yield(k) {
				return
			}
		}
	}
}
func (m *mockCache[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		m.mu.RLock()
		defer m.mu.RUnlock()
		for _, v := range m.data {
			if !yield(v) {
				return
			}
		}
	}
}
func (m *mockCache[K, V]) GetEntry(_ context.Context, key K) (cache_dto.Entry[K, V], bool, error) {
	return cache_dto.Entry[K, V]{}, false, nil
}
func (m *mockCache[K, V]) ProbeEntry(_ context.Context, key K) (cache_dto.Entry[K, V], bool, error) {
	return cache_dto.Entry[K, V]{}, false, nil
}
func (m *mockCache[K, V]) Stats() cache_dto.Stats { return cache_dto.Stats{} }
func (m *mockCache[K, V]) EstimatedSize() int     { return len(m.data) }
func (m *mockCache[K, V]) GetMaximum() uint64     { return 0 }
func (m *mockCache[K, V]) SetMaximum(size uint64) {}
func (m *mockCache[K, V]) WeightedSize() uint64   { return 0 }
func (m *mockCache[K, V]) SetExpiresAfter(_ context.Context, key K, expiresAfter time.Duration) error {
	return nil
}
func (m *mockCache[K, V]) SetRefreshableAfter(_ context.Context, key K, refreshableAfter time.Duration) error {
	return nil
}
func (m *mockCache[K, V]) Search(ctx context.Context, query string, opts *cache_dto.SearchOptions) (cache_dto.SearchResult[K, V], error) {
	return cache_dto.SearchResult[K, V]{}, nil
}
func (m *mockCache[K, V]) Query(ctx context.Context, opts *cache_dto.QueryOptions) (cache_dto.SearchResult[K, V], error) {
	return cache_dto.SearchResult[K, V]{}, nil
}
func (m *mockCache[K, V]) SupportsSearch() bool               { return false }
func (m *mockCache[K, V]) GetSchema() *cache_dto.SearchSchema { return nil }

func TestNewService(t *testing.T) {
	tests := []struct {
		name            string
		defaultProvider string
	}{
		{name: "with default provider", defaultProvider: "mock"},
		{name: "without default provider", defaultProvider: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			service := NewService(tc.defaultProvider)

			if service == nil {
				t.Fatal("expected non-nil service")
			}

			var _ = service

			providers := service.GetProviders()
			if len(providers) != 0 {
				t.Errorf("expected 0 providers, got %d", len(providers))
			}
		})
	}
}

func TestRegisterProvider_Success(t *testing.T) {
	service := NewService("")

	provider := newMockTestProvider("mock")

	err := service.RegisterProvider(context.Background(), "mock", provider)
	if err != nil {
		t.Fatalf("unexpected error registering provider: %v", err)
	}

	providers := service.GetProviders()
	if len(providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(providers))
	}
	if providers[0] != "mock" {
		t.Errorf("expected provider name 'mock', got '%s'", providers[0])
	}
}

func TestRegisterProvider_DuplicateName(t *testing.T) {
	service := NewService("")

	provider1 := newMockTestProvider("mock")
	provider2 := newMockTestProvider("mock")

	err := service.RegisterProvider(context.Background(), "mock", provider1)
	if err != nil {
		t.Fatalf("unexpected error on first registration: %v", err)
	}

	err = service.RegisterProvider(context.Background(), "mock", provider2)
	if err == nil {
		t.Fatal("expected error when registering duplicate provider name, got nil")
	}

	expectedMessage := "already registered"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error message should contain %q, got: %v", expectedMessage, err)
	}
}

func TestRegisterProvider_EmptyName(t *testing.T) {
	service := NewService("")

	provider := newMockTestProvider("mock")

	err := service.RegisterProvider(context.Background(), "", provider)
	if err == nil {
		t.Fatal("expected error when registering with empty name, got nil")
	}

	expectedMessage := "cannot be empty"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error message should contain %q, got: %v", expectedMessage, err)
	}
}

func TestRegisterProvider_NilProvider(t *testing.T) {
	service := NewService("")

	err := service.RegisterProvider(context.Background(), "mock", nil)
	if err == nil {
		t.Fatal("expected error when registering nil provider, got nil")
	}

	expectedMessage := "cannot be nil"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error message should contain %q, got: %v", expectedMessage, err)
	}
}

func TestGetProviders(t *testing.T) {
	tests := []struct {
		name      string
		providers []string
		want      []string
	}{
		{name: "no providers", providers: []string{}, want: []string{}},
		{name: "single provider", providers: []string{"mock"}, want: []string{"mock"}},
		{name: "multiple providers", providers: []string{"redis", "otter", "memory"}, want: []string{"memory", "otter", "redis"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			service := NewService("")

			for _, name := range tc.providers {
				provider := newMockTestProvider(name)
				err := service.RegisterProvider(context.Background(), name, provider)
				if err != nil {
					t.Fatalf("failed to register provider %q: %v", name, err)
				}
			}

			got := service.GetProviders()

			if len(got) != len(tc.want) {
				t.Fatalf("provider count: got %d, want %d", len(got), len(tc.want))
			}

			for i, name := range tc.want {
				if got[i] != name {
					t.Errorf("provider[%d]: got %q, want %q", i, got[i], name)
				}
			}
		})
	}
}

func TestNewCache_Success(t *testing.T) {
	service := NewService("mock")

	provider := newMockTestProvider("mock")
	err := service.RegisterProvider(context.Background(), "mock", provider)
	if err != nil {
		t.Fatalf("failed to register provider: %v", err)
	}

	options := cache_dto.Options[string, string]{
		Provider: "mock",
	}

	cache, err := NewCache[string, string](service, options)
	if err != nil {
		t.Fatalf("unexpected error creating cache: %v", err)
	}
	if cache == nil {
		t.Fatal("expected non-nil cache")
	}

	defer func() { _ = cache.Close(context.Background()) }()
}

func TestNewCache_UseDefaultProvider(t *testing.T) {
	service := NewService("mock")

	provider := newMockTestProvider("mock")
	err := service.RegisterProvider(context.Background(), "mock", provider)
	if err != nil {
		t.Fatalf("failed to register provider: %v", err)
	}

	options := cache_dto.Options[string, string]{}

	cache, err := NewCache[string, string](service, options)
	if err != nil {
		t.Fatalf("unexpected error creating cache: %v", err)
	}
	if cache == nil {
		t.Fatal("expected non-nil cache")
	}

	defer func() { _ = cache.Close(context.Background()) }()
}

func TestNewCache_ProviderNotFound(t *testing.T) {
	service := NewService("")

	options := cache_dto.Options[string, string]{
		Provider: "nonexistent",
	}

	cache, err := NewCache[string, string](service, options)
	if err == nil {
		t.Fatal("expected error for nonexistent provider, got nil")
	}
	if cache != nil {
		t.Error("expected nil cache on error")
	}

	if !errors.Is(err, ErrProviderNotFound) {
		t.Errorf("expected ErrProviderNotFound, got: %v", err)
	}
}

func TestNewCache_NoDefaultProvider(t *testing.T) {
	service := NewService("")

	options := cache_dto.Options[string, string]{}

	cache, err := NewCache[string, string](service, options)
	if err == nil {
		t.Fatal("expected error when no provider specified and no default, got nil")
	}
	if cache != nil {
		t.Error("expected nil cache on error")
	}

	if !errors.Is(err, ErrProviderNotFound) {
		t.Errorf("expected ErrProviderNotFound, got: %v", err)
	}
}

func TestNewCache_UnsupportedType(t *testing.T) {
	service := NewService("")

	provider := newMockTestProvider("mock")
	err := service.RegisterProvider(context.Background(), "mock", provider)
	if err != nil {
		t.Fatalf("failed to register provider: %v", err)
	}

	options := cache_dto.Options[string, int]{
		Provider: "mock",
	}

	cache, err := NewCache[string, int](service, options)
	if err == nil {
		t.Fatal("expected error for unsupported type, got nil")
	}
	if cache != nil {
		t.Error("expected nil cache on error")
	}

	expectedMessage := "unsupported options type"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error should contain %q, got: %v", expectedMessage, err)
	}
}

func TestNewCache_ProviderError(t *testing.T) {
	service := NewService("")

	providerErr := errors.New("provider initialisation failed")
	provider := newMockTestProviderWithError("mock", providerErr)

	err := service.RegisterProvider(context.Background(), "mock", provider)
	if err != nil {
		t.Fatalf("failed to register provider: %v", err)
	}

	options := cache_dto.Options[string, string]{
		Provider: "mock",
	}

	cache, err := NewCache[string, string](service, options)
	if err == nil {
		t.Fatal("expected provider error to be propagated, got nil")
	}
	if cache != nil {
		t.Error("expected nil cache on error")
	}

	expectedMessage := "provider initialisation failed"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error should contain %q, got: %v", expectedMessage, err)
	}
}

func TestNewCache_InvalidOptions(t *testing.T) {
	service := NewService("")

	provider := newMockTestProvider("mock")
	err := service.RegisterProvider(context.Background(), "mock", provider)
	if err != nil {
		t.Fatalf("failed to register provider: %v", err)
	}

	options := cache_dto.Options[string, string]{
		Provider:      "mock",
		MaximumSize:   100,
		MaximumWeight: 1000,
	}

	cache, err := NewCache[string, string](service, options)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if cache != nil {
		t.Error("expected nil cache on error")
	}

	if !errors.Is(err, errInvalidConfiguration) {
		t.Errorf("expected errInvalidConfiguration, got: %v", err)
	}
}

func TestConcurrentRegistration(t *testing.T) {
	service := NewService("")

	const numGoroutines = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(index int) {
			defer wg.Done()

			providerName := fmt.Sprintf("provider-%d", index)
			provider := newMockTestProvider(providerName)

			err := service.RegisterProvider(context.Background(), providerName, provider)
			if err != nil {
				t.Errorf("failed to register provider %q: %v", providerName, err)
			}
		}(i)
	}

	wg.Wait()

	providers := service.GetProviders()
	if len(providers) != numGoroutines {
		t.Errorf("expected %d providers, got %d", numGoroutines, len(providers))
	}
}

func TestConcurrentGetProviders(t *testing.T) {
	service := NewService("")

	for i := range 5 {
		provider := newMockTestProvider(fmt.Sprintf("provider-%d", i))
		err := service.RegisterProvider(context.Background(), fmt.Sprintf("provider-%d", i), provider)
		if err != nil {
			t.Fatalf("failed to register provider: %v", err)
		}
	}

	const numGoroutines = 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()

			providers := service.GetProviders()
			if len(providers) != 5 {
				t.Errorf("expected 5 providers, got %d", len(providers))
			}
		}()
	}

	wg.Wait()
}

type mockFailingProvider struct {
	closeErr error
	name     string
}

func (p *mockFailingProvider) CreateNamespaceTyped(_ string, _ any) (any, error) {
	return nil, errors.New("not implemented")
}

func (p *mockFailingProvider) Close() error {
	return p.closeErr
}

func (p *mockFailingProvider) Name() string {
	return p.name
}

func TestService_Name(t *testing.T) {
	service, ok := NewService("").(*service)
	require.True(t, ok, "expected NewService to return *service")
	if service.Name() != "CacheService" {
		t.Errorf("expected 'CacheService', got %q", service.Name())
	}
}

func TestService_Check_WithProviders(t *testing.T) {
	cacheService := NewService("")
	provider := newMockTestProvider("mock")
	if err := cacheService.RegisterProvider(context.Background(), "mock", provider); err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	s, ok := cacheService.(*service)
	require.True(t, ok, "expected service to be *service")
	status := s.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)
	if status.State != healthprobe_dto.StateHealthy {
		t.Errorf("expected StateHealthy, got %q", status.State)
	}
	if status.Name != "CacheService" {
		t.Errorf("expected 'CacheService', got %q", status.Name)
	}
	if !contains(status.Message, "1 provider") {
		t.Errorf("unexpected message: %q", status.Message)
	}
}

func TestService_Check_NoProviders(t *testing.T) {
	cacheService := NewService("")
	s, ok := cacheService.(*service)
	require.True(t, ok, "expected service to be *service")
	status := s.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)
	if status.State != healthprobe_dto.StateHealthy {
		t.Errorf("expected StateHealthy, got %q", status.State)
	}
	if !contains(status.Message, "No cache providers") {
		t.Errorf("unexpected message: %q", status.Message)
	}
}

func TestService_SetDefaultProvider_Success(t *testing.T) {
	service := NewService("")
	provider := newMockTestProvider("mock")
	if err := service.RegisterProvider(context.Background(), "mock", provider); err != nil {
		t.Fatalf("registration failed: %v", err)
	}
	if err := service.SetDefaultProvider(context.Background(), "mock"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if service.GetDefaultProvider() != "mock" {
		t.Errorf("expected default 'mock', got %q", service.GetDefaultProvider())
	}
}

func TestService_SetDefaultProvider_NotRegistered(t *testing.T) {
	service := NewService("")
	err := service.SetDefaultProvider(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for unregistered provider")
	}
	if !errors.Is(err, ErrProviderNotFound) {
		t.Errorf("expected ErrProviderNotFound, got: %v", err)
	}
}

func TestService_GetDefaultProvider(t *testing.T) {
	service := NewService("initial")
	if service.GetDefaultProvider() != "initial" {
		t.Errorf("expected 'initial', got %q", service.GetDefaultProvider())
	}
}

func TestService_GetProvider_Success(t *testing.T) {
	service := NewService("")
	provider := newMockTestProvider("mock")
	if err := service.RegisterProvider(context.Background(), "mock", provider); err != nil {
		t.Fatalf("registration failed: %v", err)
	}
	p, err := service.GetProvider("mock")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "mock" {
		t.Errorf("expected provider name 'mock', got %q", p.Name())
	}
}

func TestService_GetProvider_NotFound(t *testing.T) {
	service := NewService("")
	_, err := service.GetProvider("missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrProviderNotFound) {
		t.Errorf("expected ErrProviderNotFound, got: %v", err)
	}
}

func TestService_Close_Success(t *testing.T) {
	service := NewService("")
	provider := newMockTestProvider("mock")
	if err := service.RegisterProvider(context.Background(), "mock", provider); err != nil {
		t.Fatalf("registration failed: %v", err)
	}
	err := service.Close(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestService_Close_WithErrors(t *testing.T) {
	service := NewService("")
	provider := &mockFailingProvider{
		name:     "failing",
		closeErr: errors.New("close failed"),
	}
	if err := service.RegisterProvider(context.Background(), "failing", provider); err != nil {
		t.Fatalf("registration failed: %v", err)
	}
	err := service.Close(context.Background())
	if err == nil {
		t.Fatal("expected error from Close")
	}
	if !contains(err.Error(), "close failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestService_Close_Empty(t *testing.T) {
	service := NewService("")
	err := service.Close(context.Background())
	if err != nil {
		t.Fatalf("unexpected error closing empty service: %v", err)
	}
}

func TestCreateNamespace_Success(t *testing.T) {
	service := NewService("")
	provider := newMockTestProvider("mock")
	if err := service.RegisterProvider(context.Background(), "mock", provider); err != nil {
		t.Fatalf("registration failed: %v", err)
	}
	cache, err := CreateNamespace[string, string](context.Background(), service, "mock", "ns1", cache_dto.Options[string, string]{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache == nil {
		t.Fatal("expected non-nil cache")
	}
}

func TestCreateNamespace_ProviderNotFound(t *testing.T) {
	service := NewService("")
	_, err := CreateNamespace[string, string](context.Background(), service, "nonexistent", "ns", cache_dto.Options[string, string]{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrProviderNotFound) {
		t.Errorf("expected ErrProviderNotFound, got: %v", err)
	}
}

func TestCreateNamespace_InvalidOptions(t *testing.T) {
	service := NewService("")
	provider := newMockTestProvider("mock")
	if err := service.RegisterProvider(context.Background(), "mock", provider); err != nil {
		t.Fatalf("registration failed: %v", err)
	}
	opts := cache_dto.Options[string, string]{
		MaximumSize:   100,
		MaximumWeight: 1000,
	}
	_, err := CreateNamespace[string, string](context.Background(), service, "mock", "ns", opts)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !errors.Is(err, errInvalidConfiguration) {
		t.Errorf("expected errInvalidConfiguration, got: %v", err)
	}
}

func TestRegisterProvider_NonProviderType(t *testing.T) {
	service := NewService("")
	err := service.RegisterProvider(context.Background(), "bad", "not-a-provider")
	if err == nil {
		t.Fatal("expected error for non-Provider type")
	}
	if !contains(err.Error(), "must implement") {
		t.Errorf("unexpected error: %v", err)
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
