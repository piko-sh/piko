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
	"strings"
	"testing"

	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/wdk/safedisk"
)

func TestResolveContentSource(t *testing.T) {
	t.Run("EmptyDirective", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "md" },
		}
		_ = registry.Register(provider)
		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		service := mustCastToCollectionService(t, NewCollectionService(
			context.Background(), registry, WithDefaultSandbox(sandbox)))

		directive := &collection_dto.CollectionDirectiveInfo{}
		source, err := service.resolveContentSource(context.Background(), directive)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if source.IsExternal {
			t.Error("expected IsExternal=false for empty directive")
		}
		if source.Sandbox != sandbox {
			t.Error("expected default sandbox for local content")
		}
	})

	t.Run("LocalWithBasePath", func(t *testing.T) {
		registry := newTestProviderRegistry()
		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		service := mustCastToCollectionService(t, NewCollectionService(
			context.Background(), registry, WithDefaultSandbox(sandbox)))

		directive := &collection_dto.CollectionDirectiveInfo{
			BasePath: "/custom/path",
		}
		source, err := service.resolveContentSource(context.Background(), directive)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if source.IsExternal {
			t.Error("expected IsExternal=false")
		}
		if source.BasePath != "/custom/path" {
			t.Errorf("expected BasePath '/custom/path', got %q", source.BasePath)
		}
	})

	t.Run("NilResolverWithModulePath", func(t *testing.T) {
		registry := newTestProviderRegistry()
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

		directive := &collection_dto.CollectionDirectiveInfo{
			ContentModulePath: "piko.sh/piko/docs",
		}
		_, err := service.resolveContentSource(context.Background(), directive)
		if err == nil {
			t.Fatal("expected error when resolver is nil")
		}
		if !strings.Contains(err.Error(), "resolver not configured") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("FindModuleBoundaryFails", func(t *testing.T) {
		registry := newTestProviderRegistry()
		resolver := &mockResolverPort{
			FindModuleBoundaryFunc: func(_ context.Context, _ string) (string, string, error) {
				return "", "", errors.New("unknown module")
			},
		}
		service := mustCastToCollectionService(t, NewCollectionService(
			context.Background(), registry, WithResolver(resolver)))

		directive := &collection_dto.CollectionDirectiveInfo{
			ContentModulePath: "piko.sh/piko/docs",
		}
		_, err := service.resolveContentSource(context.Background(), directive)
		if err == nil {
			t.Fatal("expected error from FindModuleBoundary failure")
		}
		if !strings.Contains(err.Error(), "finding module boundary") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("GetModuleDirFails", func(t *testing.T) {
		registry := newTestProviderRegistry()
		resolver := &mockResolverPort{
			FindModuleBoundaryFunc: func(_ context.Context, _ string) (string, string, error) {
				return "piko.sh/piko", "docs", nil
			},
			GetModuleDirFunc: func(_ context.Context, _ string) (string, error) {
				return "", errors.New("module not downloaded")
			},
		}
		service := mustCastToCollectionService(t, NewCollectionService(
			context.Background(), registry, WithResolver(resolver)))

		directive := &collection_dto.CollectionDirectiveInfo{
			ContentModulePath: "piko.sh/piko/docs",
		}
		_, err := service.resolveContentSource(context.Background(), directive)
		if err == nil {
			t.Fatal("expected error from GetModuleDir failure")
		}
		if !strings.Contains(err.Error(), "resolving module directory") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("ExternalModuleSuccess", func(t *testing.T) {
		tmpDir := t.TempDir()
		registry := newTestProviderRegistry()
		resolver := &mockResolverPort{
			FindModuleBoundaryFunc: func(_ context.Context, _ string) (string, string, error) {
				return "piko.sh/piko", "", nil
			},
			GetModuleDirFunc: func(_ context.Context, _ string) (string, error) {
				return tmpDir, nil
			},
		}
		service := mustCastToCollectionService(t, NewCollectionService(
			context.Background(), registry, WithResolver(resolver)))

		directive := &collection_dto.CollectionDirectiveInfo{
			ContentModulePath: "piko.sh/piko",
		}
		source, err := service.resolveContentSource(context.Background(), directive)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !source.IsExternal {
			t.Error("expected IsExternal=true for module content")
		}
		if source.Sandbox == nil {
			t.Error("expected non-nil sandbox for external module")
		}
		if source.BasePath != tmpDir {
			t.Errorf("expected BasePath %q, got %q", tmpDir, source.BasePath)
		}

		if len(service.externalSandboxes) != 1 {
			t.Errorf("expected 1 tracked sandbox, got %d", len(service.externalSandboxes))
		}
	})

	t.Run("ExternalModuleSandboxTrackedAndClosed", func(t *testing.T) {
		tmpDir := t.TempDir()
		registry := newTestProviderRegistry()
		resolver := &mockResolverPort{
			FindModuleBoundaryFunc: func(_ context.Context, _ string) (string, string, error) {
				return "piko.sh/piko", "", nil
			},
			GetModuleDirFunc: func(_ context.Context, _ string) (string, error) {
				return tmpDir, nil
			},
		}
		svc := NewCollectionService(context.Background(), registry, WithResolver(resolver))
		service := mustCastToCollectionService(t, svc)

		directive := &collection_dto.CollectionDirectiveInfo{
			ContentModulePath: "piko.sh/piko",
		}
		_, err := service.resolveContentSource(context.Background(), directive)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := svc.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}
		if len(service.externalSandboxes) != 0 {
			t.Errorf("expected empty externalSandboxes after Close, got %d", len(service.externalSandboxes))
		}
	})
}

func TestCreateModuleSandbox(t *testing.T) {
	t.Run("ValidPath", func(t *testing.T) {
		tmpDir := t.TempDir()
		sandbox, err := createModuleSandbox("piko.sh/piko", tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sandbox == nil {
			t.Fatal("expected non-nil sandbox")
		}
		_ = sandbox.Close()
	})

	t.Run("InvalidPath", func(t *testing.T) {
		_, err := createModuleSandbox("piko.sh/piko", "/nonexistent/path/that/does/not/exist")
		if err == nil {
			t.Fatal("expected error for invalid path")
		}
	})
}

func TestFetchAndPrepareHybridContent(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		registry := newTestProviderRegistry()
		encoder := &MockEncoder{
			EncodeCollectionFunc: func(_ []collection_dto.ContentItem) ([]byte, error) {
				return []byte("encoded"), nil
			},
		}

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "md" },
			FetchStaticContentFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) ([]collection_dto.ContentItem, error) {
				return []collection_dto.ContentItem{{ID: "1"}}, nil
			},
			ComputeETagFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) (string, error) {
				return "etag-1", nil
			},
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry, withEncoder(encoder)))

		directive := &collection_dto.CollectionDirectiveInfo{CollectionName: "blog"}
		items, etag, blob, err := service.fetchAndPrepareHybridContent(context.Background(), provider, directive, collection_dto.ContentSource{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 1 {
			t.Errorf("expected 1 item, got %d", len(items))
		}
		if etag != "etag-1" {
			t.Errorf("expected etag 'etag-1', got %q", etag)
		}
		if len(blob) == 0 {
			t.Error("expected non-empty blob")
		}
	})

	t.Run("ETagError", func(t *testing.T) {
		registry := newTestProviderRegistry()
		encoder := &MockEncoder{
			EncodeCollectionFunc: func(_ []collection_dto.ContentItem) ([]byte, error) {
				return []byte("encoded"), nil
			},
		}

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "md" },
			FetchStaticContentFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) ([]collection_dto.ContentItem, error) {
				return []collection_dto.ContentItem{{ID: "1"}}, nil
			},
			ComputeETagFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) (string, error) {
				return "", errors.New("etag failed")
			},
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry, withEncoder(encoder)))

		directive := &collection_dto.CollectionDirectiveInfo{CollectionName: "blog"}
		_, _, _, err := service.fetchAndPrepareHybridContent(context.Background(), provider, directive, collection_dto.ContentSource{})
		if err == nil {
			t.Error("expected error from ETag failure")
		}
	})

	t.Run("EncodeError", func(t *testing.T) {
		registry := newTestProviderRegistry()
		encoder := &MockEncoder{
			EncodeCollectionFunc: func(_ []collection_dto.ContentItem) ([]byte, error) {
				return nil, errors.New("encode failed")
			},
		}

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "md" },
			FetchStaticContentFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) ([]collection_dto.ContentItem, error) {
				return []collection_dto.ContentItem{{ID: "1"}}, nil
			},
			ComputeETagFunc: func(_ context.Context, _ string, _ collection_dto.ContentSource) (string, error) {
				return "etag-1", nil
			},
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry, withEncoder(encoder)))

		directive := &collection_dto.CollectionDirectiveInfo{CollectionName: "blog"}
		_, _, _, err := service.fetchAndPrepareHybridContent(context.Background(), provider, directive, collection_dto.ContentSource{})
		if err == nil {
			t.Error("expected error from encode failure")
		}
	})
}
