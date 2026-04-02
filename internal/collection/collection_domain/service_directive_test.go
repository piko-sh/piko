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
	"testing"

	"piko.sh/piko/internal/collection/collection_dto"
)

type mockBasePathConfigurable struct {
	basePath string
	MockCollectionProvider
}

func (m *mockBasePathConfigurable) SetBasePath(_ context.Context, basePath string) {
	m.basePath = basePath
}

type mockContentModuleConfigurable struct {
	setContentModulePathFunc func(ctx context.Context, modulePath string) error
	MockCollectionProvider
}

func (m *mockContentModuleConfigurable) SetContentModulePath(ctx context.Context, modulePath string) error {
	if m.setContentModulePathFunc != nil {
		return m.setContentModulePathFunc(ctx, modulePath)
	}
	return nil
}

func TestConfigureContentSource(t *testing.T) {
	t.Run("ContentModulePath", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &mockContentModuleConfigurable{
			MockCollectionProvider: MockCollectionProvider{
				NameFunc: func() string { return "md" },
			},
		}
		_ = registry.Register(&provider.MockCollectionProvider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

		directive := &collection_dto.CollectionDirectiveInfo{
			ContentModulePath: "piko.sh/piko/docs",
		}
		err := service.configureContentSource(context.Background(), provider, directive)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("ProviderNotModuleConfigurable", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "plain" },
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

		directive := &collection_dto.CollectionDirectiveInfo{
			ContentModulePath: "piko.sh/piko/docs",
		}
		err := service.configureContentSource(context.Background(), provider, directive)
		if err == nil {
			t.Error("expected error for non-module-configurable provider")
		}
	})

	t.Run("ModuleSetupFails", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &mockContentModuleConfigurable{
			MockCollectionProvider: MockCollectionProvider{
				NameFunc: func() string { return "md" },
			},
			setContentModulePathFunc: func(_ context.Context, _ string) error {
				return errors.New("module resolve failed")
			},
		}
		_ = registry.Register(&provider.MockCollectionProvider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

		directive := &collection_dto.CollectionDirectiveInfo{
			ContentModulePath: "piko.sh/piko/docs",
		}
		err := service.configureContentSource(context.Background(), provider, directive)
		if err == nil {
			t.Error("expected error when module setup fails")
		}
	})

	t.Run("BasePathConfigurable", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &mockBasePathConfigurable{
			MockCollectionProvider: MockCollectionProvider{
				NameFunc: func() string { return "md" },
			},
		}
		_ = registry.Register(&provider.MockCollectionProvider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

		directive := &collection_dto.CollectionDirectiveInfo{
			BasePath: "/custom/path",
		}
		err := service.configureContentSource(context.Background(), provider, directive)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if provider.basePath != "/custom/path" {
			t.Errorf("expected basePath '/custom/path', got %q", provider.basePath)
		}
	})

	t.Run("NoSourceConfig", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "md" },
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

		directive := &collection_dto.CollectionDirectiveInfo{}
		err := service.configureContentSource(context.Background(), provider, directive)
		if err != nil {
			t.Fatalf("expected no error for empty source config, got %v", err)
		}
	})
}

func TestConfigureProviderBasePath(t *testing.T) {
	t.Run("Supported", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &mockBasePathConfigurable{
			MockCollectionProvider: MockCollectionProvider{
				NameFunc: func() string { return "md" },
			},
		}
		_ = registry.Register(&provider.MockCollectionProvider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

		service.configureProviderBasePath(context.Background(), provider, "/some/path")
		if provider.basePath != "/some/path" {
			t.Errorf("expected basePath '/some/path', got %q", provider.basePath)
		}
	})

	t.Run("NotSupported", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &MockCollectionProvider{
			NameFunc: func() string { return "md" },
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

		service.configureProviderBasePath(context.Background(), provider, "/some/path")
	})

	t.Run("EmptyPath", func(t *testing.T) {
		registry := newTestProviderRegistry()

		provider := &mockBasePathConfigurable{
			MockCollectionProvider: MockCollectionProvider{
				NameFunc: func() string { return "md" },
			},
		}
		_ = registry.Register(&provider.MockCollectionProvider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))

		service.configureProviderBasePath(context.Background(), provider, "")
		if provider.basePath != "" {
			t.Errorf("expected empty basePath for empty input, got %q", provider.basePath)
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
			FetchStaticContentFunc: func(_ context.Context, _ string) ([]collection_dto.ContentItem, error) {
				return []collection_dto.ContentItem{{ID: "1"}}, nil
			},
			ComputeETagFunc: func(_ context.Context, _ string) (string, error) {
				return "etag-1", nil
			},
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry, withEncoder(encoder)))

		directive := &collection_dto.CollectionDirectiveInfo{CollectionName: "blog"}
		items, etag, blob, err := service.fetchAndPrepareHybridContent(context.Background(), provider, directive)
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
			FetchStaticContentFunc: func(_ context.Context, _ string) ([]collection_dto.ContentItem, error) {
				return []collection_dto.ContentItem{{ID: "1"}}, nil
			},
			ComputeETagFunc: func(_ context.Context, _ string) (string, error) {
				return "", errors.New("etag failed")
			},
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry, withEncoder(encoder)))

		directive := &collection_dto.CollectionDirectiveInfo{CollectionName: "blog"}
		_, _, _, err := service.fetchAndPrepareHybridContent(context.Background(), provider, directive)
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
			FetchStaticContentFunc: func(_ context.Context, _ string) ([]collection_dto.ContentItem, error) {
				return []collection_dto.ContentItem{{ID: "1"}}, nil
			},
			ComputeETagFunc: func(_ context.Context, _ string) (string, error) {
				return "etag-1", nil
			},
		}
		_ = registry.Register(provider)
		service := mustCastToCollectionService(t, NewCollectionService(context.Background(), registry, withEncoder(encoder)))

		directive := &collection_dto.CollectionDirectiveInfo{CollectionName: "blog"}
		_, _, _, err := service.fetchAndPrepareHybridContent(context.Background(), provider, directive)
		if err == nil {
			t.Error("expected error from encode failure")
		}
	})
}
