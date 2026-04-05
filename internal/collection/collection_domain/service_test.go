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
	"go/ast"
	"sync"
	"testing"
	"time"

	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safedisk"
)

func mustCastToCollectionService(t *testing.T, service CollectionService) *collectionService {
	t.Helper()
	cs, ok := service.(*collectionService)
	if !ok {
		t.Fatal("expected *collectionService")
	}
	return cs
}

func newTestProviderRegistry() *MockProviderRegistry {
	providers := make(map[string]CollectionProvider)
	return &MockProviderRegistry{
		RegisterFunc: func(provider CollectionProvider) error {
			providers[provider.Name()] = provider
			return nil
		},
		GetFunc: func(name string) (CollectionProvider, bool) {
			p, ok := providers[name]
			return p, ok
		},
		ListFunc: func() []string {
			names := make([]string, 0, len(providers))
			for name := range providers {
				names = append(names, name)
			}
			return names
		},
		HasFunc: func(name string) bool {
			_, ok := providers[name]
			return ok
		},
	}
}

func newTestHybridRegistry() *MockHybridRegistry {
	type entry struct {
		etag   string
		blob   []byte
		config collection_dto.HybridConfig
	}
	registered := make(map[string]entry)
	return &MockHybridRegistry{
		RegisterFunc: func(_ context.Context, providerName, collectionName string, blob []byte, etag string, config collection_dto.HybridConfig) {
			key := providerName + ":" + collectionName
			registered[key] = entry{blob: blob, etag: etag, config: config}
		},
		GetBlobFunc: func(_ context.Context, providerName, collectionName string) ([]byte, bool) {
			key := providerName + ":" + collectionName
			if e, ok := registered[key]; ok {
				return e.blob, false
			}
			return nil, false
		},
		GetETagFunc: func(providerName, collectionName string) string {
			key := providerName + ":" + collectionName
			if e, ok := registered[key]; ok {
				return e.etag
			}
			return ""
		},
		HasFunc: func(providerName, collectionName string) bool {
			key := providerName + ":" + collectionName
			_, ok := registered[key]
			return ok
		},
		ListFunc: func() []string {
			keys := make([]string, 0, len(registered))
			for key := range registered {
				keys = append(keys, key)
			}
			return keys
		},
		TriggerRevalidationFunc: func(_ context.Context, _, _ string) {},
	}
}

type mockResolverPort struct {
	FindModuleBoundaryFunc func(ctx context.Context, importPath string) (string, string, error)
	GetModuleDirFunc       func(ctx context.Context, modulePath string) (string, error)
}

var _ resolver_domain.ResolverPort = (*mockResolverPort)(nil)

func (*mockResolverPort) DetectLocalModule(_ context.Context) error { return nil }
func (*mockResolverPort) GetModuleName() string                     { return "" }
func (*mockResolverPort) GetBaseDir() string                        { return "" }
func (*mockResolverPort) ResolvePKPath(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (*mockResolverPort) ResolveCSSPath(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (*mockResolverPort) ResolveAssetPath(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (*mockResolverPort) ConvertEntryPointPathToManifestKey(_ string) string { return "" }

func (m *mockResolverPort) GetModuleDir(ctx context.Context, modulePath string) (string, error) {
	if m.GetModuleDirFunc != nil {
		return m.GetModuleDirFunc(ctx, modulePath)
	}
	return "", nil
}

func (m *mockResolverPort) FindModuleBoundary(ctx context.Context, importPath string) (string, string, error) {
	if m.FindModuleBoundaryFunc != nil {
		return m.FindModuleBoundaryFunc(ctx, importPath)
	}
	return "", "", nil
}

func TestNewCollectionService(t *testing.T) {
	registry := newTestProviderRegistry()

	service := NewCollectionService(context.Background(), registry)
	if service == nil {
		t.Fatal("NewCollectionService() returned nil")
	}
}

func TestNewCollectionService_withHybridRegistry(t *testing.T) {
	registry := newTestProviderRegistry()
	hybridRegistry := newTestHybridRegistry()

	service := NewCollectionService(context.Background(), registry, withHybridRegistry(hybridRegistry))
	if service == nil {
		t.Fatal("NewCollectionService() returned nil")
	}

	s := mustCastToCollectionService(t, service)
	if s.hybridRegistry != hybridRegistry {
		t.Error("withHybridRegistry did not inject the mock registry")
	}
}

func TestNewCollectionService_withEncoder(t *testing.T) {
	registry := newTestProviderRegistry()
	encoder := &MockEncoder{}

	service := NewCollectionService(context.Background(), registry, withEncoder(encoder))
	if service == nil {
		t.Fatal("NewCollectionService() returned nil")
	}

	s := mustCastToCollectionService(t, service)
	if s.encoder != encoder {
		t.Error("withEncoder did not inject the mock encoder")
	}
}

func TestNewCollectionService_withServiceClock(t *testing.T) {
	registry := newTestProviderRegistry()

	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(baseTime)

	service := NewCollectionService(context.Background(), registry, withServiceClock(mockClock))
	if service == nil {
		t.Fatal("NewCollectionService() returned nil")
	}

	s := mustCastToCollectionService(t, service)

	if s.clock.Now() != baseTime {
		t.Errorf("Expected time %v, got %v", baseTime, s.clock.Now())
	}
}

func TestNewCollectionService_WithDefaultSandbox(t *testing.T) {
	registry := newTestProviderRegistry()
	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	defer func() { _ = sandbox.Close() }()

	service := NewCollectionService(context.Background(), registry, WithDefaultSandbox(sandbox))
	s := mustCastToCollectionService(t, service)
	if s.defaultSandbox != sandbox {
		t.Error("WithDefaultSandbox did not inject the mock sandbox")
	}
}

func TestNewCollectionService_WithResolver(t *testing.T) {
	registry := newTestProviderRegistry()
	resolver := &mockResolverPort{}

	service := NewCollectionService(context.Background(), registry, WithResolver(resolver))
	s := mustCastToCollectionService(t, service)
	if s.resolver != resolver {
		t.Error("WithResolver did not inject the mock resolver")
	}
}

func TestDefaultContentSource(t *testing.T) {
	registry := newTestProviderRegistry()
	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	defer func() { _ = sandbox.Close() }()

	service := NewCollectionService(context.Background(), registry, WithDefaultSandbox(sandbox))
	s := mustCastToCollectionService(t, service)

	source := s.defaultContentSource()
	if source.IsExternal {
		t.Error("expected IsExternal=false")
	}
	if source.Sandbox != sandbox {
		t.Error("expected defaultSandbox to be returned")
	}
}

func TestServiceClose_NoSandboxes(t *testing.T) {
	registry := newTestProviderRegistry()
	service := NewCollectionService(context.Background(), registry)

	err := service.Close()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestServiceClose_ClosesAllSandboxes(t *testing.T) {
	registry := newTestProviderRegistry()
	service := NewCollectionService(context.Background(), registry)
	s := mustCastToCollectionService(t, service)

	sb1 := safedisk.NewMockSandbox("/mod1", safedisk.ModeReadOnly)
	sb2 := safedisk.NewMockSandbox("/mod2", safedisk.ModeReadOnly)

	s.trackExternalSandbox(sb1)
	s.trackExternalSandbox(sb2)

	err := service.Close()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if sb1.CallCounts["Close"] != 1 {
		t.Errorf("expected sb1.Close called once, got %d", sb1.CallCounts["Close"])
	}
	if sb2.CallCounts["Close"] != 1 {
		t.Errorf("expected sb2.Close called once, got %d", sb2.CallCounts["Close"])
	}

	if len(s.externalSandboxes) != 0 {
		t.Errorf("expected externalSandboxes to be empty after Close, got %d", len(s.externalSandboxes))
	}
}

func TestServiceClose_ReturnsFirstError(t *testing.T) {
	registry := newTestProviderRegistry()
	service := NewCollectionService(context.Background(), registry)
	s := mustCastToCollectionService(t, service)

	sb1 := safedisk.NewMockSandbox("/mod1", safedisk.ModeReadOnly)
	sb1.CloseErr = errors.New("close failed")
	sb2 := safedisk.NewMockSandbox("/mod2", safedisk.ModeReadOnly)

	s.trackExternalSandbox(sb1)
	s.trackExternalSandbox(sb2)

	err := service.Close()
	if err == nil {
		t.Fatal("expected error from Close")
	}
	if err.Error() != "close failed" {
		t.Errorf("expected 'close failed', got %q", err.Error())
	}

	if sb2.CallCounts["Close"] != 1 {
		t.Errorf("expected sb2.Close called even after sb1 error, got %d", sb2.CallCounts["Close"])
	}
}

func TestTrackExternalSandbox_Concurrent(t *testing.T) {
	registry := newTestProviderRegistry()
	service := NewCollectionService(context.Background(), registry)
	s := mustCastToCollectionService(t, service)

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			sb := safedisk.NewMockSandbox("/mod", safedisk.ModeReadOnly)
			s.trackExternalSandbox(sb)
		}()
	}
	wg.Wait()

	if len(s.externalSandboxes) != goroutines {
		t.Errorf("expected %d sandboxes, got %d", goroutines, len(s.externalSandboxes))
	}
}

func TestProcessCollectionDirective_ProviderNotFound(t *testing.T) {
	registry := newTestProviderRegistry()
	service := NewCollectionService(context.Background(), registry)

	directive := &collection_dto.CollectionDirectiveInfo{
		ProviderName:   "nonexistent",
		CollectionName: "blog",
		LayoutPath:     "/pages/blog/{slug}.pk",
	}

	_, err := service.ProcessCollectionDirective(context.Background(), directive)
	if err == nil {
		t.Error("Expected error for nonexistent provider")
	}
}

func TestProcessCollectionDirective_StaticProvider(t *testing.T) {
	registry := newTestProviderRegistry()

	provider := &MockCollectionProvider{
		NameFunc: func() string { return "static-test" },
		TypeFunc: func() ProviderType { return ProviderTypeStatic },
		FetchStaticContentFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) ([]collection_dto.ContentItem, error) {
			return []collection_dto.ContentItem{
				{
					ID:       "post-1",
					URL:      "/blog/post-1",
					Slug:     "post-1",
					Metadata: map[string]any{"title": "Post 1"},
				},
				{
					ID:       "post-2",
					URL:      "/blog/post-2",
					Slug:     "post-2",
					Metadata: map[string]any{"title": "Post 2"},
				},
			}, nil
		},
	}
	_ = registry.Register(provider)

	service := NewCollectionService(context.Background(), registry)

	directive := &collection_dto.CollectionDirectiveInfo{
		ProviderName:   "static-test",
		CollectionName: "blog",
		LayoutPath:     "/pages/blog/{slug}.pk",
	}

	entryPoints, err := service.ProcessCollectionDirective(context.Background(), directive)
	if err != nil {
		t.Fatalf("ProcessCollectionDirective() failed: %v", err)
	}

	if len(entryPoints) != 2 {
		t.Fatalf("Expected 2 entry points, got %d", len(entryPoints))
	}

	for _, ep := range entryPoints {
		if !ep.IsVirtual {
			t.Error("Expected IsVirtual to be true")
		}
		if ep.IsDynamic {
			t.Error("Expected IsDynamic to be false for static provider")
		}
		if ep.Path != "/pages/blog/{slug}.pk" {
			t.Errorf("Expected path '/pages/blog/{slug}.pk', got %q", ep.Path)
		}
	}
}

func TestProcessCollectionDirective_DynamicProvider(t *testing.T) {
	registry := newTestProviderRegistry()

	provider := &MockCollectionProvider{
		NameFunc: func() string { return "dynamic-test" },
		TypeFunc: func() ProviderType { return ProviderTypeDynamic },
	}
	_ = registry.Register(provider)

	service := NewCollectionService(context.Background(), registry)

	directive := &collection_dto.CollectionDirectiveInfo{
		ProviderName:   "dynamic-test",
		CollectionName: "products",
		LayoutPath:     "/pages/products/{id}.pk",
		RoutePath:      "/products/{id}",
	}

	entryPoints, err := service.ProcessCollectionDirective(context.Background(), directive)
	if err != nil {
		t.Fatalf("ProcessCollectionDirective() failed: %v", err)
	}

	if len(entryPoints) != 1 {
		t.Fatalf("Expected 1 entry point for dynamic provider, got %d", len(entryPoints))
	}

	ep := entryPoints[0]
	if !ep.IsDynamic {
		t.Error("Expected IsDynamic to be true for dynamic provider")
	}
	if ep.RoutePatternOverride != "/products/{id}" {
		t.Errorf("Expected route '/products/{id}', got %q", ep.RoutePatternOverride)
	}
}

func TestProcessCollectionDirective_HybridProvider(t *testing.T) {
	ResetHybridRegistry()

	registry := newTestProviderRegistry()
	hybridRegistry := newTestHybridRegistry()
	encoder := &MockEncoder{
		EncodeCollectionFunc: func(_ []collection_dto.ContentItem) ([]byte, error) {
			return []byte("encoded"), nil
		},
	}

	provider := &MockCollectionProvider{
		NameFunc: func() string { return "hybrid-test" },
		TypeFunc: func() ProviderType { return ProviderTypeHybrid },
		FetchStaticContentFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) ([]collection_dto.ContentItem, error) {
			return []collection_dto.ContentItem{
				{
					ID:       "item-1",
					URL:      "/hybrid/item-1",
					Slug:     "item-1",
					Metadata: map[string]any{"title": "Hybrid Item 1"},
				},
			}, nil
		},
		ComputeETagFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) (string, error) {
			return "etag-123", nil
		},
	}
	_ = registry.Register(provider)

	service := NewCollectionService(context.Background(), registry, withHybridRegistry(hybridRegistry), withEncoder(encoder))

	directive := &collection_dto.CollectionDirectiveInfo{
		ProviderName:   "hybrid-test",
		CollectionName: "content",
		LayoutPath:     "/pages/content/{slug}.pk",
	}

	entryPoints, err := service.ProcessCollectionDirective(context.Background(), directive)
	if err != nil {
		t.Fatalf("ProcessCollectionDirective() failed: %v", err)
	}

	if len(entryPoints) != 1 {
		t.Fatalf("Expected 1 entry point, got %d", len(entryPoints))
	}

	ep := entryPoints[0]
	if !ep.IsHybrid {
		t.Error("Expected IsHybrid to be true for hybrid provider")
	}

	if !hybridRegistry.Has("hybrid-test", "content") {
		t.Error("Expected hybrid snapshot to be registered")
	}
}

func TestProcessCollectionDirective_StaticFetchError(t *testing.T) {
	registry := newTestProviderRegistry()

	expectedErr := errors.New("fetch failed")
	provider := &MockCollectionProvider{
		NameFunc: func() string { return "error-test" },
		TypeFunc: func() ProviderType { return ProviderTypeStatic },
		FetchStaticContentFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) ([]collection_dto.ContentItem, error) {
			return nil, expectedErr
		},
	}
	_ = registry.Register(provider)

	service := NewCollectionService(context.Background(), registry)

	directive := &collection_dto.CollectionDirectiveInfo{
		ProviderName:   "error-test",
		CollectionName: "blog",
		LayoutPath:     "/pages/blog/{slug}.pk",
	}

	_, err := service.ProcessCollectionDirective(context.Background(), directive)
	if err == nil {
		t.Error("Expected error from fetch failure")
	}
}

func TestProcessCollectionDirective_UnknownProviderType(t *testing.T) {
	registry := newTestProviderRegistry()

	provider := &MockCollectionProvider{
		NameFunc: func() string { return "unknown-type" },
		TypeFunc: func() ProviderType { return ProviderType("invalid") },
	}
	_ = registry.Register(provider)

	service := NewCollectionService(context.Background(), registry)

	directive := &collection_dto.CollectionDirectiveInfo{
		ProviderName:   "unknown-type",
		CollectionName: "test",
		LayoutPath:     "/pages/test.pk",
	}

	_, err := service.ProcessCollectionDirective(context.Background(), directive)
	if err == nil {
		t.Error("Expected error for unknown provider type")
	}
}

func TestValidateConfiguration_NotImplemented(t *testing.T) {
	registry := newTestProviderRegistry()
	service := NewCollectionService(context.Background(), registry)

	err := service.ValidateConfiguration(context.Background(), nil)
	if err != nil {
		t.Errorf("ValidateConfiguration() returned unexpected error: %v", err)
	}
}

func TestGetCacheKey(t *testing.T) {
	registry := newTestProviderRegistry()
	service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

	key := service.getCacheKey("provider", "collection")
	expected := "provider:collection"

	if key != expected {
		t.Errorf("Expected key %q, got %q", expected, key)
	}
}

func TestCacheOperations(t *testing.T) {
	registry := newTestProviderRegistry()
	service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

	_, found := service.getCachedContent("provider", "collection")
	if found {
		t.Error("Expected no cached content initially")
	}

	items := []collection_dto.ContentItem{
		{ID: "item-1"},
	}
	service.setCachedContent("provider", "collection", items)

	cached, found := service.getCachedContent("provider", "collection")
	if !found {
		t.Error("Expected to find cached content")
	}

	if len(cached) != 1 {
		t.Errorf("Expected 1 cached item, got %d", len(cached))
	}
}

func TestParseGetCollectionOptions_Nil(t *testing.T) {
	registry := newTestProviderRegistry()
	service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

	options, err := service.parseGetCollectionOptions(nil)
	if err != nil {
		t.Fatalf("parseGetCollectionOptions(nil) failed: %v", err)
	}

	if options.ProviderName != "" {
		t.Errorf("Expected empty ProviderName, got %q", options.ProviderName)
	}
}

func TestParseGetCollectionOptions_ValidOptions(t *testing.T) {
	registry := newTestProviderRegistry()
	service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

	input := collection_dto.FetchOptions{
		ProviderName: "test-provider",
		Locale:       "en-GB",
	}

	options, err := service.parseGetCollectionOptions(input)
	if err != nil {
		t.Fatalf("parseGetCollectionOptions() failed: %v", err)
	}

	if options.ProviderName != "test-provider" {
		t.Errorf("Expected ProviderName 'test-provider', got %q", options.ProviderName)
	}
}

func TestParseGetCollectionOptions_InvalidType(t *testing.T) {
	registry := newTestProviderRegistry()
	service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

	_, err := service.parseGetCollectionOptions("not a FetchOptions")
	if err == nil {
		t.Error("Expected error for invalid type")
	}
}

func TestResolveProviderName_WithExplicitName(t *testing.T) {
	registry := newTestProviderRegistry()
	service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

	options := &collection_dto.FetchOptions{ProviderName: "explicit"}
	name := service.resolveProviderName(context.Background(), options)

	if name != "explicit" {
		t.Errorf("Expected 'explicit', got %q", name)
	}
}

func TestResolveProviderName_Default(t *testing.T) {
	registry := newTestProviderRegistry()
	service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

	options := &collection_dto.FetchOptions{}
	name := service.resolveProviderName(context.Background(), options)

	if name != "markdown" {
		t.Errorf("Expected default 'markdown', got %q", name)
	}
}

func TestLookupProvider_NotFound(t *testing.T) {
	registry := newTestProviderRegistry()
	service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

	_, err := service.lookupProvider("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent provider")
	}
}

func TestLookupProvider_Found(t *testing.T) {
	registry := newTestProviderRegistry()

	provider := &MockCollectionProvider{
		NameFunc: func() string { return "found" },
	}
	_ = registry.Register(provider)

	service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

	found, err := service.lookupProvider("found")
	if err != nil {
		t.Fatalf("lookupProvider() failed: %v", err)
	}

	if found.Name() != "found" {
		t.Errorf("Expected provider 'found', got %q", found.Name())
	}
}

func TestConvertItemMetadata(t *testing.T) {
	item := &collection_dto.ContentItem{
		ID:             "item-123",
		Slug:           "test-slug",
		Locale:         "en-GB",
		TranslationKey: "key-123",
		URL:            "/test/url",
		ReadingTime:    5,
		CreatedAt:      "2024-01-01T00:00:00Z",
		UpdatedAt:      "2024-01-02T00:00:00Z",
		PublishedAt:    "2024-01-03T00:00:00Z",
		Metadata: map[string]any{
			"title":  "Test Title",
			"tags":   []string{"go", "test"},
			"custom": 123,
		},
	}

	metadata := convertItemMetadata(item)

	if metadata["ID"] != "item-123" {
		t.Errorf("Expected ID 'item-123', got %v", metadata["ID"])
	}
	if metadata["Slug"] != "test-slug" {
		t.Errorf("Expected Slug 'test-slug', got %v", metadata["Slug"])
	}
	if metadata["Locale"] != "en-GB" {
		t.Errorf("Expected Locale 'en-GB', got %v", metadata["Locale"])
	}
	if metadata["TranslationKey"] != "key-123" {
		t.Errorf("Expected TranslationKey 'key-123', got %v", metadata["TranslationKey"])
	}
	if metadata["URL"] != "/test/url" {
		t.Errorf("Expected URL '/test/url', got %v", metadata["URL"])
	}
	if metadata["ReadingTime"] != 5 {
		t.Errorf("Expected ReadingTime 5, got %v", metadata["ReadingTime"])
	}

	if metadata["title"] != "Test Title" {
		t.Errorf("Expected title 'Test Title', got %v", metadata["title"])
	}
	if metadata["custom"] != 123 {
		t.Errorf("Expected custom 123, got %v", metadata["custom"])
	}
}

func TestBuildHybridConfigFromDirective_Default(t *testing.T) {
	directive := &collection_dto.CollectionDirectiveInfo{}

	config := buildHybridConfigFromDirective(directive)

	defaultConfig := collection_dto.DefaultHybridConfig()
	if config.RevalidationTTL != defaultConfig.RevalidationTTL {
		t.Errorf("Expected default TTL %v, got %v", defaultConfig.RevalidationTTL, config.RevalidationTTL)
	}
}

func TestBuildHybridConfigFromDirective_WithTTL(t *testing.T) {
	directive := &collection_dto.CollectionDirectiveInfo{
		CacheConfig: &collection_dto.CacheConfig{
			TTL: 60,
		},
	}

	config := buildHybridConfigFromDirective(directive)

	expected := 60 * time.Second
	if config.RevalidationTTL != expected {
		t.Errorf("Expected TTL %v, got %v", expected, config.RevalidationTTL)
	}
}

func TestBuildHybridConfigFromDirective_NoCache(t *testing.T) {
	directive := &collection_dto.CollectionDirectiveInfo{
		CacheConfig: &collection_dto.CacheConfig{
			Strategy: "no-cache",
		},
	}

	config := buildHybridConfigFromDirective(directive)

	if config.RevalidationTTL != 0 {
		t.Errorf("Expected TTL 0 for no-cache, got %v", config.RevalidationTTL)
	}
}

func TestProcessGetCollectionCall(t *testing.T) {
	makeTargetType := func() ast.Expr {
		return &ast.Ident{Name: "BlogPost"}
	}

	t.Run("StaticProvider", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "markdown" },
			TypeFunc: func() ProviderType { return ProviderTypeStatic },
			FetchStaticContentFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) ([]collection_dto.ContentItem, error) {
				return []collection_dto.ContentItem{
					{ID: "1", Slug: "post-1", URL: "/blog/post-1", Metadata: map[string]any{"title": "Post"}},
				}, nil
			},
		}
		_ = registry.Register(provider)
		service := NewCollectionService(context.Background(), registry)

		annotation, err := service.ProcessGetCollectionCall(
			context.Background(), "blog", "BlogPost", makeTargetType(), nil,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if annotation == nil {
			t.Fatal("expected non-nil annotation")
		}
		if !annotation.IsStatic {
			t.Error("expected IsStatic=true for static provider")
		}
		if !annotation.IsCollectionCall {
			t.Error("expected IsCollectionCall=true")
		}
	})

	t.Run("DynamicProvider", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "markdown" },
			TypeFunc: func() ProviderType { return ProviderTypeDynamic },
			GenerateRuntimeFetcherFunc: func(_ context.Context, _ string, _ ast.Expr, _ collection_dto.FetchOptions) (*collection_dto.RuntimeFetcherCode, error) {
				return &collection_dto.RuntimeFetcherCode{
					FetcherFunc: &ast.FuncDecl{Name: ast.NewIdent("GetBlogPosts")},
				}, nil
			},
		}
		_ = registry.Register(provider)
		service := NewCollectionService(context.Background(), registry)

		annotation, err := service.ProcessGetCollectionCall(
			context.Background(), "blog", "BlogPost", makeTargetType(), nil,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if annotation == nil {
			t.Fatal("expected non-nil annotation")
		}
		if annotation.IsStatic {
			t.Error("expected IsStatic=false for dynamic provider")
		}
	})

	t.Run("HybridProvider", func(t *testing.T) {
		ResetHybridRegistry()

		registry := newTestProviderRegistry()
		hybridRegistry := newTestHybridRegistry()
		encoder := &MockEncoder{
			EncodeCollectionFunc: func(_ []collection_dto.ContentItem) ([]byte, error) {
				return []byte("encoded"), nil
			},
		}

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "markdown" },
			TypeFunc: func() ProviderType { return ProviderTypeHybrid },
			FetchStaticContentFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) ([]collection_dto.ContentItem, error) {
				return []collection_dto.ContentItem{
					{ID: "1", Slug: "post-1", URL: "/blog/post-1", Metadata: map[string]any{"title": "Post"}},
				}, nil
			},
			ComputeETagFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) (string, error) {
				return "etag-abc", nil
			},
		}
		_ = registry.Register(provider)
		service := NewCollectionService(context.Background(), registry, withHybridRegistry(hybridRegistry), withEncoder(encoder))

		annotation, err := service.ProcessGetCollectionCall(
			context.Background(), "blog", "BlogPost", makeTargetType(), nil,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if annotation == nil {
			t.Fatal("expected non-nil annotation")
		}
		if !annotation.IsHybridCollection {
			t.Error("expected IsHybridCollection=true for hybrid provider")
		}
	})

	t.Run("NilOptions", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "markdown" },
			TypeFunc: func() ProviderType { return ProviderTypeStatic },
			FetchStaticContentFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) ([]collection_dto.ContentItem, error) {
				return nil, nil
			},
		}
		_ = registry.Register(provider)
		service := NewCollectionService(context.Background(), registry)

		annotation, err := service.ProcessGetCollectionCall(
			context.Background(), "blog", "BlogPost", makeTargetType(), nil,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if annotation == nil {
			t.Fatal("expected non-nil annotation")
		}
	})

	t.Run("InvalidOptionsType", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "markdown" },
			TypeFunc: func() ProviderType { return ProviderTypeStatic },
		}
		_ = registry.Register(provider)
		service := NewCollectionService(context.Background(), registry)

		_, err := service.ProcessGetCollectionCall(
			context.Background(), "blog", "BlogPost", makeTargetType(), "not-fetch-options",
		)
		if err == nil {
			t.Error("expected error for invalid options type")
		}
	})

	t.Run("ProviderNotFound", func(t *testing.T) {
		registry := newTestProviderRegistry()
		service := NewCollectionService(context.Background(), registry)

		_, err := service.ProcessGetCollectionCall(
			context.Background(), "blog", "BlogPost", makeTargetType(), nil,
		)
		if err == nil {
			t.Error("expected error for missing provider")
		}
	})

	t.Run("TargetTypeValidationFails", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "markdown" },
			TypeFunc: func() ProviderType { return ProviderTypeStatic },
			ValidateTargetTypeFunc: func(_ ast.Expr) error {
				return errors.New("invalid target type")
			},
		}
		_ = registry.Register(provider)
		service := NewCollectionService(context.Background(), registry)

		_, err := service.ProcessGetCollectionCall(
			context.Background(), "blog", "BlogPost", makeTargetType(), nil,
		)
		if err == nil {
			t.Error("expected error for invalid target type")
		}
	})

	t.Run("UnknownProviderType", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "markdown" },
			TypeFunc: func() ProviderType { return ProviderType("unknown") },
		}
		_ = registry.Register(provider)
		service := NewCollectionService(context.Background(), registry)

		_, err := service.ProcessGetCollectionCall(
			context.Background(), "blog", "BlogPost", makeTargetType(), nil,
		)
		if err == nil {
			t.Error("expected error for unknown provider type")
		}
	})
}

func TestValidateConfiguration(t *testing.T) {
	t.Run("NilConfig", func(t *testing.T) {
		registry := newTestProviderRegistry()
		service := NewCollectionService(context.Background(), registry)

		if err := service.ValidateConfiguration(context.Background(), nil); err != nil {
			t.Errorf("expected nil error for nil config, got %v", err)
		}
	})

	t.Run("DefaultProviderNotRegistered", func(t *testing.T) {
		registry := newTestProviderRegistry()
		service := NewCollectionService(context.Background(), registry)

		config := &Config{
			DefaultProvider: "nonexistent",
		}
		err := service.ValidateConfiguration(context.Background(), config)
		if err == nil {
			t.Error("expected error for unregistered default provider")
		}
	})

	t.Run("ProviderNotRegistered", func(t *testing.T) {
		registry := newTestProviderRegistry()
		service := NewCollectionService(context.Background(), registry)

		config := &Config{
			Providers: map[string]ProviderConfigEntry{
				"missing": {Enabled: true},
			},
		}
		err := service.ValidateConfiguration(context.Background(), config)
		if err == nil {
			t.Error("expected error for unregistered provider")
		}
	})

	t.Run("DisabledProviderSkipped", func(t *testing.T) {
		registry := newTestProviderRegistry()
		service := NewCollectionService(context.Background(), registry)

		config := &Config{
			Providers: map[string]ProviderConfigEntry{
				"missing": {Enabled: false},
			},
		}
		err := service.ValidateConfiguration(context.Background(), config)
		if err != nil {
			t.Errorf("expected no error for disabled provider, got %v", err)
		}
	})

	t.Run("CollectionProviderNotRegistered", func(t *testing.T) {
		registry := newTestProviderRegistry()
		service := NewCollectionService(context.Background(), registry)

		config := &Config{
			Collections: map[string]CollectionConfigEntry{
				"blog": {Provider: "nonexistent", Enabled: true},
			},
		}
		err := service.ValidateConfiguration(context.Background(), config)
		if err == nil {
			t.Error("expected error for collection with unregistered provider")
		}
	})

	t.Run("CollectionUsesDefaultProvider", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "markdown" },
		}
		_ = registry.Register(provider)
		service := NewCollectionService(context.Background(), registry)

		config := &Config{
			DefaultProvider: "markdown",
			Collections: map[string]CollectionConfigEntry{
				"blog": {Enabled: true},
			},
		}
		err := service.ValidateConfiguration(context.Background(), config)
		if err != nil {
			t.Errorf("expected no error when collection uses default, got %v", err)
		}
	})

	t.Run("CollectionNoProviderNoDefault", func(t *testing.T) {
		registry := newTestProviderRegistry()
		service := NewCollectionService(context.Background(), registry)

		config := &Config{
			Collections: map[string]CollectionConfigEntry{
				"blog": {Enabled: true},
			},
		}
		err := service.ValidateConfiguration(context.Background(), config)
		if err == nil {
			t.Error("expected error when collection has no provider and no default")
		}
	})

	t.Run("DisabledCollectionSkipped", func(t *testing.T) {
		registry := newTestProviderRegistry()
		service := NewCollectionService(context.Background(), registry)

		config := &Config{
			Collections: map[string]CollectionConfigEntry{
				"blog": {Provider: "nonexistent", Enabled: false},
			},
		}
		err := service.ValidateConfiguration(context.Background(), config)
		if err != nil {
			t.Errorf("expected no error for disabled collection, got %v", err)
		}
	})

	t.Run("MultipleErrors", func(t *testing.T) {
		registry := newTestProviderRegistry()
		service := NewCollectionService(context.Background(), registry)

		config := &Config{
			DefaultProvider: "nonexistent",
			Providers: map[string]ProviderConfigEntry{
				"also-missing": {Enabled: true},
			},
			Collections: map[string]CollectionConfigEntry{
				"blog": {Provider: "gone", Enabled: true},
			},
		}
		err := service.ValidateConfiguration(context.Background(), config)
		if err == nil {
			t.Error("expected error for multiple validation failures")
		}
	})

	t.Run("AllValid", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "markdown" },
		}
		_ = registry.Register(provider)
		service := NewCollectionService(context.Background(), registry)

		config := &Config{
			DefaultProvider: "markdown",
			Providers: map[string]ProviderConfigEntry{
				"markdown": {Enabled: true},
			},
			Collections: map[string]CollectionConfigEntry{
				"blog": {Provider: "markdown", Enabled: true},
			},
		}
		err := service.ValidateConfiguration(context.Background(), config)
		if err != nil {
			t.Errorf("expected no error for valid config, got %v", err)
		}
	})
}

func TestDispatchProviderAnnotation(t *testing.T) {
	makeTargetType := func() ast.Expr {
		return &ast.Ident{Name: "BlogPost"}
	}

	t.Run("Static", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "test" },
			TypeFunc: func() ProviderType { return ProviderTypeStatic },
			FetchStaticContentFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) ([]collection_dto.ContentItem, error) {
				return []collection_dto.ContentItem{
					{ID: "1", Slug: "p", URL: "/p", Metadata: map[string]any{"title": "P"}},
				}, nil
			},
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

		opts := &collection_dto.FetchOptions{}
		annotation, err := service.dispatchProviderAnnotation(context.Background(), provider, "blog", makeTargetType(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !annotation.IsStatic {
			t.Error("expected IsStatic=true")
		}
	})

	t.Run("Dynamic", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "test" },
			TypeFunc: func() ProviderType { return ProviderTypeDynamic },
			GenerateRuntimeFetcherFunc: func(_ context.Context, _ string, _ ast.Expr, _ collection_dto.FetchOptions) (*collection_dto.RuntimeFetcherCode, error) {
				return &collection_dto.RuntimeFetcherCode{
					FetcherFunc: &ast.FuncDecl{Name: ast.NewIdent("GetPosts")},
				}, nil
			},
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

		opts := &collection_dto.FetchOptions{}
		annotation, err := service.dispatchProviderAnnotation(context.Background(), provider, "blog", makeTargetType(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if annotation.IsStatic {
			t.Error("expected IsStatic=false for dynamic")
		}
	})

	t.Run("Hybrid", func(t *testing.T) {
		ResetHybridRegistry()

		registry := newTestProviderRegistry()
		hybridRegistry := newTestHybridRegistry()
		encoder := &MockEncoder{
			EncodeCollectionFunc: func(_ []collection_dto.ContentItem) ([]byte, error) {
				return []byte("encoded"), nil
			},
		}

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "test" },
			TypeFunc: func() ProviderType { return ProviderTypeHybrid },
			FetchStaticContentFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) ([]collection_dto.ContentItem, error) {
				return []collection_dto.ContentItem{
					{ID: "1", Slug: "p", URL: "/p", Metadata: map[string]any{"title": "P"}},
				}, nil
			},
			ComputeETagFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) (string, error) {
				return "etag-1", nil
			},
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry, withHybridRegistry(hybridRegistry), withEncoder(encoder)))

		opts := &collection_dto.FetchOptions{}
		annotation, err := service.dispatchProviderAnnotation(context.Background(), provider, "blog", makeTargetType(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !annotation.IsHybridCollection {
			t.Error("expected IsHybridCollection=true")
		}
	})

	t.Run("UnknownType", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "test" },
			TypeFunc: func() ProviderType { return ProviderType("bogus") },
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

		opts := &collection_dto.FetchOptions{}
		_, err := service.dispatchProviderAnnotation(context.Background(), provider, "blog", makeTargetType(), opts)
		if err == nil {
			t.Error("expected error for unknown provider type")
		}
	})
}
