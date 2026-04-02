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

//go:build bench

package render_test_bench

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"piko.sh/piko/internal/registry/registry_adapters"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/render/render_adapters"
)

func newRealRegistryService(svgCount int) (registry_domain.RegistryService, *registry_adapters.MockBlobStore) {
	metaStore := registry_adapters.NewMockMetadataStore()
	blobStore := registry_adapters.NewMockBlobStore()

	blobStores := map[string]registry_domain.BlobStore{
		"default": blobStore,
	}

	ctx := context.Background()

	for i := range svgCount {
		artefactID := fmt.Sprintf("svg-artefact-%d", i)
		storageKey := fmt.Sprintf("blobs/svg-%d.svg", i)

		svgContent := fmt.Sprintf(`<svg viewBox="0 0 24 24" fill="currentColor" class="icon-%d">
			<path d="M12 2L2 7v10l10 5 10-5V7l-10-5z"></path>
			<circle cx="12" cy="12" r="5"></circle>
		</svg>`, i)

		_ = blobStore.Put(ctx, storageKey, strings.NewReader(svgContent))

		artefact := &registry_dto.ArtefactMeta{
			ID:         artefactID,
			SourcePath: fmt.Sprintf("assets/icons/icon-%d.svg", i),
			ActualVariants: []registry_dto.Variant{
				{
					VariantID:        fmt.Sprintf("variant-%d", i),
					StorageBackendID: "default",
					StorageKey:       storageKey,
					MimeType:         "image/svg+xml",
					SizeBytes:        int64(len(svgContent)),
					Status:           registry_dto.VariantStatusReady,
					MetadataTags: registry_dto.TagsFromMap(map[string]string{
						"type": "minified-svg",
					}),
				},
			},
		}

		_ = metaStore.AtomicUpdate(ctx, []registry_dto.AtomicAction{
			{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: artefact},
		})
	}

	components := []string{"my-card", "custom-button", "nav-menu", "footer-widget", "sidebar-panel"}
	for _, tagName := range components {
		artefactID := fmt.Sprintf("component-%s", tagName)
		storageKey := fmt.Sprintf("dist/components/%s.js", tagName)

		artefact := &registry_dto.ArtefactMeta{
			ID:         artefactID,
			SourcePath: fmt.Sprintf("src/components/%s/%s.ts", tagName, tagName),
			ActualVariants: []registry_dto.Variant{
				{
					VariantID:        fmt.Sprintf("variant-%s", tagName),
					StorageBackendID: "default",
					StorageKey:       storageKey,
					MimeType:         "application/javascript",
					SizeBytes:        1024,
					Status:           registry_dto.VariantStatusReady,
					MetadataTags: registry_dto.TagsFromMap(map[string]string{
						"type":    "component-js",
						"role":    "entrypoint",
						"tagName": tagName,
					}),
				},
			},
		}

		_ = metaStore.AtomicUpdate(ctx, []registry_dto.AtomicAction{
			{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: artefact},
		})
	}

	service := registry_domain.NewRegistryService(metaStore, blobStores, nil, nil)
	return service, blobStore
}

func BenchmarkRegistryAdapter_GetComponentMetadata(b *testing.B) {
	service, _ := newRealRegistryService(10)
	adapter := render_adapters.NewDataLoaderRegistryAdapter(service, nil, "/dist")
	ctx := context.Background()

	b.Run("CacheHit", func(b *testing.B) {

		_, _ = adapter.GetComponentMetadata(ctx, "my-card")

		b.ResetTimer()
		b.ReportAllocs()

		for b.Loop() {
			_, _ = adapter.GetComponentMetadata(ctx, "my-card")
		}
	})

	b.Run("CacheMiss", func(b *testing.B) {

		b.ResetTimer()
		b.ReportAllocs()

		for b.Loop() {
			freshAdapter := render_adapters.NewDataLoaderRegistryAdapter(service, nil, "/dist")
			_, _ = freshAdapter.GetComponentMetadata(ctx, "custom-button")
		}
	})

	b.Run("NotFound", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for b.Loop() {
			_, _ = adapter.GetComponentMetadata(ctx, "non-existent-component")
		}
	})
}

func BenchmarkRegistryAdapter_GetAssetRawSVG(b *testing.B) {
	service, _ := newRealRegistryService(100)
	adapter := render_adapters.NewDataLoaderRegistryAdapter(service, nil, "/dist")
	ctx := context.Background()

	b.Run("CacheHit", func(b *testing.B) {

		_, _ = adapter.GetAssetRawSVG(ctx, "svg-artefact-0")

		b.ResetTimer()
		b.ReportAllocs()

		for b.Loop() {
			_, _ = adapter.GetAssetRawSVG(ctx, "svg-artefact-0")
		}
	})

	b.Run("CacheMiss_SingleItem", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for b.Loop() {
			freshAdapter := render_adapters.NewDataLoaderRegistryAdapter(service, nil, "/dist")
			_, _ = freshAdapter.GetAssetRawSVG(ctx, "svg-artefact-1")
		}
	})

	b.Run("NotFound", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for b.Loop() {
			_, _ = adapter.GetAssetRawSVG(ctx, "non-existent-svg")
		}
	})
}

func BenchmarkRegistryAdapter_CacheConfig(b *testing.B) {
	service, _ := newRealRegistryService(100)
	ctx := context.Background()

	configs := []struct {
		config *render_adapters.DataLoaderAdapterConfig
		name   string
	}{
		{
			name:   "DefaultConfig",
			config: nil,
		},
		{
			name: "SmallCache",
			config: &render_adapters.DataLoaderAdapterConfig{
				ComponentCacheCapacity: 10,
				ComponentCacheTTL:      1 * time.Minute,
				SVGCacheCapacity:       20,
				SVGCacheTTL:            5 * time.Minute,
			},
		},
		{
			name: "LargeCache",
			config: &render_adapters.DataLoaderAdapterConfig{
				ComponentCacheCapacity: 1000,
				ComponentCacheTTL:      30 * time.Minute,
				SVGCacheCapacity:       2000,
				SVGCacheTTL:            60 * time.Minute,
			},
		},
	}

	for _, config := range configs {
		b.Run(config.name, func(b *testing.B) {
			adapter := render_adapters.NewDataLoaderRegistryAdapter(service, config.config, "/dist")

			_, _ = adapter.GetAssetRawSVG(ctx, "svg-artefact-0")

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_, _ = adapter.GetAssetRawSVG(ctx, "svg-artefact-0")
			}
		})
	}
}

func BenchmarkRegistryAdapter_Concurrent(b *testing.B) {
	service, _ := newRealRegistryService(100)
	adapter := render_adapters.NewDataLoaderRegistryAdapter(service, nil, "/dist")
	ctx := context.Background()

	for i := range 50 {
		_, _ = adapter.GetAssetRawSVG(ctx, fmt.Sprintf("svg-artefact-%d", i))
	}
	for _, tag := range []string{"my-card", "custom-button", "nav-menu"} {
		_, _ = adapter.GetComponentMetadata(ctx, tag)
	}

	b.Run("ConcurrentSVGReads", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				assetID := fmt.Sprintf("svg-artefact-%d", i%50)
				_, _ = adapter.GetAssetRawSVG(ctx, assetID)
				i++
			}
		})
	})

	b.Run("ConcurrentComponentReads", func(b *testing.B) {
		tags := []string{"my-card", "custom-button", "nav-menu"}

		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				tag := tags[i%len(tags)]
				_, _ = adapter.GetComponentMetadata(ctx, tag)
				i++
			}
		})
	})

	b.Run("ConcurrentMixedReads", func(b *testing.B) {
		tags := []string{"my-card", "custom-button", "nav-menu"}

		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				if i%2 == 0 {
					assetID := fmt.Sprintf("svg-artefact-%d", i%50)
					_, _ = adapter.GetAssetRawSVG(ctx, assetID)
				} else {
					tag := tags[i%len(tags)]
					_, _ = adapter.GetComponentMetadata(ctx, tag)
				}
				i++
			}
		})
	})
}

func BenchmarkRegistryAdapter_RealVsMock(b *testing.B) {
	ctx := context.Background()

	b.Run("RealAdapter_CacheHit", func(b *testing.B) {
		service, _ := newRealRegistryService(10)
		adapter := render_adapters.NewDataLoaderRegistryAdapter(service, nil, "/dist")

		_, _ = adapter.GetAssetRawSVG(ctx, "svg-artefact-0")

		b.ResetTimer()
		b.ReportAllocs()

		for b.Loop() {
			_, _ = adapter.GetAssetRawSVG(ctx, "svg-artefact-0")
		}
	})

	b.Run("MockRegistry_CacheHit", func(b *testing.B) {
		registry := NewBenchmarkRegistry()

		b.ResetTimer()
		b.ReportAllocs()

		for b.Loop() {
			_, _ = registry.GetAssetRawSVG(ctx, "testmodule/lib/icon.svg")
		}
	})
}
