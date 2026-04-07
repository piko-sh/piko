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

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/collection/collection_dto"
)

func TestGenerateStaticAnnotation(t *testing.T) {
	makeTargetType := func() ast.Expr { return &ast.Ident{Name: "Post"} }

	t.Run("Success", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "md" },
			TypeFunc: func() ProviderType { return ProviderTypeStatic },
			FetchStaticContentFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) ([]collection_dto.ContentItem, error) {
				return []collection_dto.ContentItem{
					{ID: "1", Slug: "a", URL: "/a", Metadata: map[string]any{"title": "A"}},
				}, nil
			},
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

		opts := &collection_dto.FetchOptions{}
		annotation, err := service.generateStaticCollectionAnnotation(context.Background(), provider, "blog", makeTargetType(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !annotation.IsStatic {
			t.Error("expected IsStatic=true")
		}
		if !annotation.IsCollectionCall {
			t.Error("expected IsCollectionCall=true")
		}
		if annotation.StaticCollectionLiteral == nil {
			t.Error("expected non-nil StaticCollectionLiteral")
		}
	})

	t.Run("FetchError", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "md" },
			TypeFunc: func() ProviderType { return ProviderTypeStatic },
			FetchStaticContentFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) ([]collection_dto.ContentItem, error) {
				return nil, errors.New("fetch failed")
			},
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

		opts := &collection_dto.FetchOptions{}
		_, err := service.generateStaticCollectionAnnotation(context.Background(), provider, "blog", makeTargetType(), opts)
		if err == nil {
			t.Error("expected error for fetch failure")
		}
	})

	t.Run("CacheHit", func(t *testing.T) {
		registry := newTestProviderRegistry()

		callCount := 0
		provider := &MockCollectionProvider{
			NameFunc: func() string { return "md" },
			TypeFunc: func() ProviderType { return ProviderTypeStatic },
			FetchStaticContentFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) ([]collection_dto.ContentItem, error) {
				callCount++
				return []collection_dto.ContentItem{
					{ID: "1", Slug: "a", URL: "/a", Metadata: map[string]any{"title": "A"}},
				}, nil
			},
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

		opts := &collection_dto.FetchOptions{}
		_, err := service.generateStaticCollectionAnnotation(context.Background(), provider, "blog", makeTargetType(), opts)
		if err != nil {
			t.Fatalf("first call: %v", err)
		}

		_, err = service.generateStaticCollectionAnnotation(context.Background(), provider, "blog", makeTargetType(), opts)
		if err != nil {
			t.Fatalf("second call: %v", err)
		}

		if callCount != 1 {
			t.Errorf("expected fetch called once (cached), got %d", callCount)
		}
	})

	t.Run("EmptyCollection", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "md" },
			TypeFunc: func() ProviderType { return ProviderTypeStatic },
			FetchStaticContentFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) ([]collection_dto.ContentItem, error) {
				return nil, nil
			},
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

		opts := &collection_dto.FetchOptions{}
		annotation, err := service.generateStaticCollectionAnnotation(context.Background(), provider, "blog", makeTargetType(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !annotation.IsStatic {
			t.Error("expected IsStatic=true for empty collection")
		}
	})

	t.Run("WithQueryFiltering", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "md" },
			TypeFunc: func() ProviderType { return ProviderTypeStatic },
			FetchStaticContentFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) ([]collection_dto.ContentItem, error) {
				return []collection_dto.ContentItem{
					{ID: "1", Locale: "en", Slug: "a", URL: "/a", Metadata: map[string]any{"title": "A"}},
					{ID: "2", Locale: "fr", Slug: "b", URL: "/b", Metadata: map[string]any{"title": "B"}},
				}, nil
			},
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

		opts := &collection_dto.FetchOptions{Locale: "en", Filters: map[string]any{}}
		annotation, err := service.generateStaticCollectionAnnotation(context.Background(), provider, "blog", makeTargetType(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(annotation.StaticCollectionData) != 1 {
			t.Errorf("expected 1 filtered item, got %d", len(annotation.StaticCollectionData))
		}
	})
}

func TestFetchOrCacheStaticContent(t *testing.T) {
	t.Run("FirstFetch", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "md" },
			FetchStaticContentFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) ([]collection_dto.ContentItem, error) {
				return []collection_dto.ContentItem{{ID: "1"}}, nil
			},
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

		items, err := service.fetchOrCacheStaticContent(context.Background(), provider, "blog", collection_dto.ContentSource{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 1 {
			t.Errorf("expected 1 item, got %d", len(items))
		}
	})

	t.Run("CachedRetrieval", func(t *testing.T) {
		registry := newTestProviderRegistry()

		callCount := 0
		provider := &MockCollectionProvider{
			NameFunc: func() string { return "md" },
			FetchStaticContentFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) ([]collection_dto.ContentItem, error) {
				callCount++
				return []collection_dto.ContentItem{{ID: "1"}}, nil
			},
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

		_, _ = service.fetchOrCacheStaticContent(context.Background(), provider, "blog", collection_dto.ContentSource{})
		_, _ = service.fetchOrCacheStaticContent(context.Background(), provider, "blog", collection_dto.ContentSource{})

		if callCount != 1 {
			t.Errorf("expected 1 fetch call, got %d", callCount)
		}
	})

	t.Run("FetchError", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "md" },
			FetchStaticContentFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) ([]collection_dto.ContentItem, error) {
				return nil, errors.New("broken")
			},
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

		_, err := service.fetchOrCacheStaticContent(context.Background(), provider, "blog", collection_dto.ContentSource{})
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestGenerateDynamicAnnotation(t *testing.T) {
	makeTargetType := func() ast.Expr { return &ast.Ident{Name: "Post"} }

	t.Run("Success", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "api" },
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
		annotation, err := service.generateDynamicAnnotation(context.Background(), provider, "blog", makeTargetType(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if annotation.IsStatic {
			t.Error("expected IsStatic=false")
		}
		if !annotation.IsCollectionCall {
			t.Error("expected IsCollectionCall=true")
		}
		if annotation.DynamicCollectionInfo == nil {
			t.Error("expected non-nil DynamicCollectionInfo")
		}
	})

	t.Run("FetcherError", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "api" },
			TypeFunc: func() ProviderType { return ProviderTypeDynamic },
			GenerateRuntimeFetcherFunc: func(_ context.Context, _ string, _ ast.Expr, _ collection_dto.FetchOptions) (*collection_dto.RuntimeFetcherCode, error) {
				return nil, errors.New("generator broke")
			},
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

		opts := &collection_dto.FetchOptions{}
		_, err := service.generateDynamicAnnotation(context.Background(), provider, "blog", makeTargetType(), opts)
		if err == nil {
			t.Error("expected error from fetcher generation failure")
		}
	})
}

func TestBuildStaticAnnotation(t *testing.T) {
	service := newTestCollectionService(t)
	targetType := &ast.Ident{Name: "Post"}
	sliceLit := &ast.CompositeLit{}
	items := []collection_dto.ContentItem{{ID: "1"}, {ID: "2"}}

	annotation := service.buildStaticAnnotation(targetType, sliceLit, items)

	if !annotation.IsStatic {
		t.Error("expected IsStatic=true")
	}
	if !annotation.IsCollectionCall {
		t.Error("expected IsCollectionCall=true")
	}
	if annotation.IsHybridCollection {
		t.Error("expected IsHybridCollection=false")
	}
	if annotation.StaticCollectionLiteral != sliceLit {
		t.Error("expected StaticCollectionLiteral to match")
	}
	if len(annotation.StaticCollectionData) != 2 {
		t.Errorf("expected 2 items in StaticCollectionData, got %d", len(annotation.StaticCollectionData))
	}
	if annotation.ResolvedType == nil {
		t.Error("expected non-nil ResolvedType")
	}
}

func TestBuildDynamicAnnotation(t *testing.T) {
	service := newTestCollectionService(t)
	targetType := &ast.Ident{Name: "Post"}
	dynamicInfo := &collection_dto.DynamicCollectionInfo{
		ProviderName:   "api",
		CollectionName: "blog",
	}

	annotation := service.buildDynamicAnnotation(targetType, dynamicInfo)

	if annotation.IsStatic {
		t.Error("expected IsStatic=false")
	}
	if !annotation.IsCollectionCall {
		t.Error("expected IsCollectionCall=true")
	}
	if annotation.DynamicCollectionInfo != dynamicInfo {
		t.Error("expected DynamicCollectionInfo to match")
	}
	if annotation.StaticCollectionLiteral != nil {
		t.Error("expected nil StaticCollectionLiteral for dynamic")
	}
}

func TestBuildDynamicCollectionInfo(t *testing.T) {
	targetType := &ast.Ident{Name: "Post"}
	fetcherCode := &collection_dto.RuntimeFetcherCode{
		FetcherFunc: &ast.FuncDecl{Name: ast.NewIdent("Fetch")},
	}

	info := buildDynamicCollectionInfo(fetcherCode, targetType, "api", "blog")

	if info.ProviderName != "api" {
		t.Errorf("expected ProviderName 'api', got %q", info.ProviderName)
	}
	if info.CollectionName != "blog" {
		t.Errorf("expected CollectionName 'blog', got %q", info.CollectionName)
	}
	if info.HybridMode {
		t.Error("expected HybridMode=false for dynamic")
	}
	if info.FetcherCode != fetcherCode {
		t.Error("expected FetcherCode to match")
	}
	if info.TargetType != targetType {
		t.Error("expected TargetType to match")
	}
}

func TestCreateSliceTypeInfo(t *testing.T) {
	service := newTestCollectionService(t)
	targetType := &ast.Ident{Name: "Post"}

	resolvedType := service.createSliceTypeInfo(targetType)

	require.NotNil(t, resolvedType, "expected non-nil ResolvedTypeInfo")
	arrayType, ok := resolvedType.TypeExpression.(*ast.ArrayType)
	if !ok {
		t.Fatal("expected ArrayType expression")
	}
	if arrayType.Elt != targetType {
		t.Error("expected element type to match target type")
	}
}

func TestConvertItemsToAny(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		result := convertItemsToAny(nil)
		if len(result) != 0 {
			t.Errorf("expected empty slice, got %d", len(result))
		}
	})

	t.Run("Multiple", func(t *testing.T) {
		items := []collection_dto.ContentItem{{ID: "1"}, {ID: "2"}}
		result := convertItemsToAny(items)
		if len(result) != 2 {
			t.Errorf("expected 2 items, got %d", len(result))
		}
		first, ok := result[0].(collection_dto.ContentItem)
		if !ok {
			t.Fatal("expected ContentItem type")
		}
		if first.ID != "1" {
			t.Errorf("expected ID '1', got %q", first.ID)
		}
	})
}

func TestLogStaticAnnotationDiagnostics(t *testing.T) {
	service := newTestCollectionService(t)
	targetType := &ast.Ident{Name: "Post"}

	t.Run("WithLiteral", func(t *testing.T) {
		annotation := service.buildStaticAnnotation(targetType, &ast.CompositeLit{}, nil)
		service.logStaticAnnotationDiagnostics(context.Background(), annotation, nil)
	})

	t.Run("WithoutLiteral", func(t *testing.T) {
		annotation := service.buildStaticAnnotation(targetType, nil, nil)
		annotation.StaticCollectionLiteral = nil
		service.logStaticAnnotationDiagnostics(context.Background(), annotation, nil)
	})
}

func TestLogDynamicAnnotationDiagnostics(t *testing.T) {
	service := newTestCollectionService(t)
	targetType := &ast.Ident{Name: "Post"}

	fetcherCode := &collection_dto.RuntimeFetcherCode{
		FetcherFunc: &ast.FuncDecl{Name: ast.NewIdent("Fetch")},
	}
	dynamicInfo := buildDynamicCollectionInfo(fetcherCode, targetType, "api", "blog")
	annotation := service.buildDynamicAnnotation(targetType, dynamicInfo)

	service.logDynamicAnnotationDiagnostics(context.Background(), annotation, dynamicInfo, "api", "blog", fetcherCode)
}
