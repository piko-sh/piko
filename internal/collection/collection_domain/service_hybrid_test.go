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
	"testing"
	"time"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/collection/collection_dto"
)

func TestGenerateHybridAnnotation(t *testing.T) {
	makeTargetType := func() ast.Expr { return &ast.Ident{Name: "Post"} }

	makeProvider := func(
		fetchErr error,
		etagErr error,
	) *MockCollectionProvider {
		return &MockCollectionProvider{
			NameFunc: func() string { return "hybrid" },
			TypeFunc: func() ProviderType { return ProviderTypeHybrid },
			FetchStaticContentFunc: func(_ context.Context, _ string) ([]collection_dto.ContentItem, error) {
				if fetchErr != nil {
					return nil, fetchErr
				}
				return []collection_dto.ContentItem{
					{ID: "1", Slug: "p", URL: "/p", Metadata: map[string]any{"title": "P"}},
				}, nil
			},
			ComputeETagFunc: func(_ context.Context, _ string) (string, error) {
				if etagErr != nil {
					return "", etagErr
				}
				return "etag-abc", nil
			},
		}
	}

	t.Run("Success", func(t *testing.T) {
		ResetHybridRegistry()
		registry := newTestProviderRegistry()
		hybridRegistry := newTestHybridRegistry()
		encoder := &MockEncoder{
			EncodeCollectionFunc: func(_ []collection_dto.ContentItem) ([]byte, error) {
				return []byte("encoded"), nil
			},
		}

		provider := makeProvider(nil, nil)
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry, withHybridRegistry(hybridRegistry), withEncoder(encoder)))

		opts := &collection_dto.FetchOptions{}
		annotation, err := service.generateHybridAnnotation(context.Background(), provider, "blog", makeTargetType(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !annotation.IsHybridCollection {
			t.Error("expected IsHybridCollection=true")
		}
		if !annotation.IsCollectionCall {
			t.Error("expected IsCollectionCall=true")
		}
		if annotation.StaticCollectionLiteral == nil {
			t.Error("expected non-nil StaticCollectionLiteral")
		}
		if annotation.DynamicCollectionInfo == nil {
			t.Error("expected non-nil DynamicCollectionInfo")
		}
		if !hybridRegistry.Has("hybrid", "blog") {
			t.Error("expected snapshot registered in hybrid registry")
		}
	})

	t.Run("ETagFailure_FallbackToStatic", func(t *testing.T) {
		ResetHybridRegistry()
		registry := newTestProviderRegistry()
		hybridRegistry := newTestHybridRegistry()
		encoder := &MockEncoder{
			EncodeCollectionFunc: func(_ []collection_dto.ContentItem) ([]byte, error) {
				return []byte("encoded"), nil
			},
		}

		provider := makeProvider(nil, errors.New("etag failed"))
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry, withHybridRegistry(hybridRegistry), withEncoder(encoder)))

		opts := &collection_dto.FetchOptions{}
		annotation, err := service.generateHybridAnnotation(context.Background(), provider, "blog", makeTargetType(), opts)
		if err != nil {
			t.Fatalf("unexpected error on fallback: %v", err)
		}

		if annotation.IsHybridCollection {
			t.Error("expected IsHybridCollection=false after fallback to static")
		}
		if !annotation.IsStatic {
			t.Error("expected IsStatic=true after fallback")
		}
	})

	t.Run("EncodeFailure_FallbackToStatic", func(t *testing.T) {
		ResetHybridRegistry()
		registry := newTestProviderRegistry()
		hybridRegistry := newTestHybridRegistry()
		encoder := &MockEncoder{
			EncodeCollectionFunc: func(_ []collection_dto.ContentItem) ([]byte, error) {
				return nil, errors.New("encode failed")
			},
		}

		provider := makeProvider(nil, nil)
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry, withHybridRegistry(hybridRegistry), withEncoder(encoder)))

		opts := &collection_dto.FetchOptions{}
		annotation, err := service.generateHybridAnnotation(context.Background(), provider, "blog", makeTargetType(), opts)
		if err != nil {
			t.Fatalf("unexpected error on fallback: %v", err)
		}
		if annotation.IsHybridCollection {
			t.Error("expected IsHybridCollection=false after encode failure fallback")
		}
	})

	t.Run("FetchError_FallbackToStatic", func(t *testing.T) {
		ResetHybridRegistry()
		registry := newTestProviderRegistry()
		hybridRegistry := newTestHybridRegistry()
		encoder := &MockEncoder{
			EncodeCollectionFunc: func(_ []collection_dto.ContentItem) ([]byte, error) {
				return []byte("encoded"), nil
			},
		}

		fetchCallCount := 0
		provider := &MockCollectionProvider{
			NameFunc: func() string { return "hybrid" },
			TypeFunc: func() ProviderType { return ProviderTypeHybrid },
			FetchStaticContentFunc: func(_ context.Context, _ string) ([]collection_dto.ContentItem, error) {
				fetchCallCount++
				if fetchCallCount == 1 {
					return nil, errors.New("fetch failed")
				}
				return []collection_dto.ContentItem{
					{ID: "1", Slug: "p", URL: "/p", Metadata: map[string]any{"title": "P"}},
				}, nil
			},
			ComputeETagFunc: func(_ context.Context, _ string) (string, error) {
				return "etag-1", nil
			},
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry, withHybridRegistry(hybridRegistry), withEncoder(encoder)))

		opts := &collection_dto.FetchOptions{}
		annotation, err := service.generateHybridAnnotation(context.Background(), provider, "blog", makeTargetType(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if annotation.IsHybridCollection {
			t.Error("expected fallback to static on fetch error")
		}
	})
}

func TestPrepareHybridContent(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "md" },
			FetchStaticContentFunc: func(_ context.Context, _ string) ([]collection_dto.ContentItem, error) {
				return []collection_dto.ContentItem{{ID: "1"}}, nil
			},
			ComputeETagFunc: func(_ context.Context, _ string) (string, error) {
				return "etag-abc", nil
			},
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

		opts := &collection_dto.FetchOptions{}
		items, etag, err := service.prepareHybridContent(context.Background(), provider, "blog", opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 1 {
			t.Errorf("expected 1 item, got %d", len(items))
		}
		if etag != "etag-abc" {
			t.Errorf("expected etag 'etag-abc', got %q", etag)
		}
	})

	t.Run("FetchError", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "md" },
			FetchStaticContentFunc: func(_ context.Context, _ string) ([]collection_dto.ContentItem, error) {
				return nil, errors.New("fetch failed")
			},
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

		opts := &collection_dto.FetchOptions{}
		_, _, err := service.prepareHybridContent(context.Background(), provider, "blog", opts)
		if err == nil {
			t.Error("expected error from fetch failure")
		}
	})

	t.Run("ETagError", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "md" },
			FetchStaticContentFunc: func(_ context.Context, _ string) ([]collection_dto.ContentItem, error) {
				return []collection_dto.ContentItem{{ID: "1"}}, nil
			},
			ComputeETagFunc: func(_ context.Context, _ string) (string, error) {
				return "", errors.New("etag broke")
			},
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

		opts := &collection_dto.FetchOptions{}
		_, _, err := service.prepareHybridContent(context.Background(), provider, "blog", opts)
		if err == nil {
			t.Error("expected error from ETag failure")
		}
	})
}

func TestRegisterHybridSnapshotService(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		registry := newTestProviderRegistry()
		hybridRegistry := newTestHybridRegistry()
		encoder := &MockEncoder{
			EncodeCollectionFunc: func(_ []collection_dto.ContentItem) ([]byte, error) {
				return []byte("encoded"), nil
			},
		}

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "md" },
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry, withHybridRegistry(hybridRegistry), withEncoder(encoder)))

		items := []collection_dto.ContentItem{{ID: "1"}}
		err := service.registerHybridSnapshot(context.Background(), provider, "blog", items, "etag-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !hybridRegistry.Has("md", "blog") {
			t.Error("expected snapshot registered")
		}
	})

	t.Run("EncodeError", func(t *testing.T) {
		registry := newTestProviderRegistry()
		hybridRegistry := newTestHybridRegistry()
		encoder := &MockEncoder{
			EncodeCollectionFunc: func(_ []collection_dto.ContentItem) ([]byte, error) {
				return nil, errors.New("encode failed")
			},
		}

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "md" },
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry, withHybridRegistry(hybridRegistry), withEncoder(encoder)))

		items := []collection_dto.ContentItem{{ID: "1"}}
		err := service.registerHybridSnapshot(context.Background(), provider, "blog", items, "etag-1")
		if err == nil {
			t.Error("expected error from encode failure")
		}
	})
}

func TestBuildHybridConfigFromOptions(t *testing.T) {
	service := newTestCollectionService(t)

	t.Run("NilOptions", func(t *testing.T) {
		config := service.buildHybridConfigFromOptions(nil)
		defaultConfig := collection_dto.DefaultHybridConfig()
		if config.RevalidationTTL != defaultConfig.RevalidationTTL {
			t.Errorf("expected default TTL, got %v", config.RevalidationTTL)
		}
	})

	t.Run("NilCache", func(t *testing.T) {
		opts := &collection_dto.FetchOptions{}
		config := service.buildHybridConfigFromOptions(opts)
		defaultConfig := collection_dto.DefaultHybridConfig()
		if config.RevalidationTTL != defaultConfig.RevalidationTTL {
			t.Errorf("expected default TTL, got %v", config.RevalidationTTL)
		}
	})

	t.Run("WithTTL", func(t *testing.T) {
		opts := &collection_dto.FetchOptions{
			Cache: &collection_dto.CacheConfig{TTL: 120},
		}
		config := service.buildHybridConfigFromOptions(opts)
		expected := 120 * time.Second
		if config.RevalidationTTL != expected {
			t.Errorf("expected TTL %v, got %v", expected, config.RevalidationTTL)
		}
	})

	t.Run("StaleWhileRevalidate", func(t *testing.T) {
		opts := &collection_dto.FetchOptions{
			Cache: &collection_dto.CacheConfig{Strategy: "stale-while-revalidate"},
		}
		config := service.buildHybridConfigFromOptions(opts)
		if !config.StaleIfError {
			t.Error("expected StaleIfError=true for stale-while-revalidate strategy")
		}
	})

	t.Run("NoCache", func(t *testing.T) {
		opts := &collection_dto.FetchOptions{
			Cache: &collection_dto.CacheConfig{Strategy: "no-cache"},
		}
		config := service.buildHybridConfigFromOptions(opts)
		if config.RevalidationTTL != 0 {
			t.Errorf("expected TTL 0 for no-cache, got %v", config.RevalidationTTL)
		}
	})
}

func TestBuildHybridDynamicInfo(t *testing.T) {
	service := newTestCollectionService(t)
	targetType := &ast.Ident{Name: "Post"}
	info := service.buildHybridDynamicInfo("md", "blog", targetType, "etag-1", new(collection_dto.DefaultHybridConfig()))

	if !info.HybridMode {
		t.Error("expected HybridMode=true")
	}
	if info.ProviderName != "md" {
		t.Errorf("expected ProviderName 'md', got %q", info.ProviderName)
	}
	if info.CollectionName != "blog" {
		t.Errorf("expected CollectionName 'blog', got %q", info.CollectionName)
	}
	if info.SnapshotETag != "etag-1" {
		t.Errorf("expected SnapshotETag 'etag-1', got %q", info.SnapshotETag)
	}
	if info.HybridConfig == nil {
		t.Error("expected non-nil HybridConfig")
	}
	if info.FetcherCode != nil {
		t.Error("expected nil FetcherCode for hybrid dynamic info")
	}
}

func TestBuildHybridAnnotation(t *testing.T) {
	service := newTestCollectionService(t)
	targetType := &ast.Ident{Name: "Post"}
	sliceLit := &ast.CompositeLit{}
	dynamicInfo := &collection_dto.DynamicCollectionInfo{
		HybridMode: true,
	}
	items := []collection_dto.ContentItem{{ID: "1"}}

	annotation := service.buildHybridAnnotation(targetType, sliceLit, dynamicInfo, items)

	if !annotation.IsHybridCollection {
		t.Error("expected IsHybridCollection=true")
	}
	if !annotation.IsCollectionCall {
		t.Error("expected IsCollectionCall=true")
	}
	if !annotation.IsStatic {
		t.Error("expected IsStatic=true for hybrid (has static literal)")
	}
	if annotation.StaticCollectionLiteral != sliceLit {
		t.Error("expected StaticCollectionLiteral to match")
	}
	if annotation.DynamicCollectionInfo != dynamicInfo {
		t.Error("expected DynamicCollectionInfo to match")
	}
}

func TestLogHybridAnnotationDiagnostics(t *testing.T) {
	service := newTestCollectionService(t)
	targetType := &ast.Ident{Name: "Post"}

	t.Run("NilDynamicInfo", func(t *testing.T) {
		annotation := &ast_domain.GoGeneratorAnnotation{}

		service.logHybridAnnotationDiagnostics(context.Background(), annotation, nil, nil)
	})

	t.Run("WithDynamicInfo", func(t *testing.T) {
		dynamicInfo := &collection_dto.DynamicCollectionInfo{
			ProviderName:   "md",
			CollectionName: "blog",
			SnapshotETag:   "etag-1",
			HybridMode:     true,
		}
		annotation := service.buildHybridAnnotation(targetType, &ast.CompositeLit{}, dynamicInfo, nil)

		service.logHybridAnnotationDiagnostics(context.Background(), annotation, dynamicInfo, nil)
	})

	t.Run("WithHybridConfig", func(t *testing.T) {
		dynamicInfo := &collection_dto.DynamicCollectionInfo{
			ProviderName:   "md",
			CollectionName: "blog",
			HybridMode:     true,
			HybridConfig:   new(collection_dto.DefaultHybridConfig()),
		}
		annotation := service.buildHybridAnnotation(targetType, &ast.CompositeLit{}, dynamicInfo, nil)

		service.logHybridAnnotationDiagnostics(context.Background(), annotation, dynamicInfo, nil)
	})

	t.Run("WithStaticLiteral", func(t *testing.T) {
		dynamicInfo := &collection_dto.DynamicCollectionInfo{
			ProviderName:   "md",
			CollectionName: "blog",
		}
		annotation := service.buildHybridAnnotation(targetType, &ast.CompositeLit{}, dynamicInfo, nil)

		service.logHybridAnnotationDiagnostics(context.Background(), annotation, dynamicInfo, nil)
	})
}
