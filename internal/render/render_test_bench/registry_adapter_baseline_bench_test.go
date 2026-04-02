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
	"bytes"
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"piko.sh/piko/internal/registry/registry_adapters"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/render/render_adapters"
)

var svgContentSizes = map[string]int{
	"tiny":   100,
	"small":  500,
	"medium": 2000,
	"large":  10000,
}

func createBaselineSVGContent(size int, index int) string {

	base := fmt.Sprintf(`<svg viewBox="0 0 24 24" fill="currentColor" class="icon-%d" xmlns="http://www.w3.org/2000/svg">`, index)
	closer := `</svg>`

	pathTemplate := `<path d="M12 2L2 7v10l10 5 10-5V7l-10-5z"></path>`
	circleTemplate := `<circle cx="12" cy="12" r="5"></circle>`

	var builder strings.Builder
	builder.WriteString(base)

	for builder.Len() < size-len(closer) {
		if builder.Len()%2 == 0 {
			builder.WriteString(pathTemplate)
		} else {
			builder.WriteString(circleTemplate)
		}
	}

	builder.WriteString(closer)
	return builder.String()
}

func createBaselineRegistryService(svgCount int, svgSize string) (registry_domain.RegistryService, *registry_adapters.MockBlobStore) {
	metaStore := registry_adapters.NewMockMetadataStore()
	blobStore := registry_adapters.NewMockBlobStore()

	blobStores := map[string]registry_domain.BlobStore{
		"default": blobStore,
	}

	ctx := context.Background()
	contentSize := svgContentSizes[svgSize]
	if contentSize == 0 {
		contentSize = svgContentSizes["small"]
	}

	for i := range svgCount {
		artefactID := fmt.Sprintf("svg-artefact-%d", i)
		storageKey := fmt.Sprintf("blobs/svg-%d.svg", i)

		svgContent := createBaselineSVGContent(contentSize, i)

		_ = blobStore.Put(ctx, storageKey, bytes.NewReader([]byte(svgContent)))

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

	components := []string{
		"my-card", "custom-button", "nav-menu", "footer-widget", "sidebar-panel",
		"data-table", "form-input", "modal-dialog", "toast-notification", "dropdown-menu",
	}
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

func BenchmarkBaseline_CacheHit_SVG(b *testing.B) {
	service, _ := createBaselineRegistryService(100, "small")
	adapter := render_adapters.NewDataLoaderRegistryAdapter(service, nil, "/dist")
	ctx := context.Background()

	_, _ = adapter.GetAssetRawSVG(ctx, "svg-artefact-0")

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, _ = adapter.GetAssetRawSVG(ctx, "svg-artefact-0")
	}
}

func BenchmarkBaseline_CacheHit_Component(b *testing.B) {
	service, _ := createBaselineRegistryService(10, "small")
	adapter := render_adapters.NewDataLoaderRegistryAdapter(service, nil, "/dist")
	ctx := context.Background()

	_, _ = adapter.GetComponentMetadata(ctx, "my-card")

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, _ = adapter.GetComponentMetadata(ctx, "my-card")
	}
}

func BenchmarkBaseline_CacheHit_SVG_BySizes(b *testing.B) {
	for sizeName := range svgContentSizes {
		b.Run(sizeName, func(b *testing.B) {
			service, _ := createBaselineRegistryService(10, sizeName)
			adapter := render_adapters.NewDataLoaderRegistryAdapter(service, nil, "/dist")
			ctx := context.Background()

			_, _ = adapter.GetAssetRawSVG(ctx, "svg-artefact-0")

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_, _ = adapter.GetAssetRawSVG(ctx, "svg-artefact-0")
			}
		})
	}
}

func BenchmarkBaseline_CacheMiss_SVG(b *testing.B) {

	service, _ := createBaselineRegistryService(100000, "small")
	adapter := render_adapters.NewDataLoaderRegistryAdapter(service, nil, "/dist")
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	i := 0
	for b.Loop() {

		_, _ = adapter.GetAssetRawSVG(ctx, fmt.Sprintf("svg-artefact-%d", i))
		i++
	}
}

func BenchmarkBaseline_CacheMiss_Component(b *testing.B) {

	service, _ := createBaselineRegistryService(100000, "small")
	adapter := render_adapters.NewDataLoaderRegistryAdapter(service, nil, "/dist")
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	i := 0
	for b.Loop() {

		_, _ = adapter.GetComponentMetadata(ctx, fmt.Sprintf("component-%d", i))
		i++
	}
}

func BenchmarkBaseline_CacheMiss_SVG_BySizes(b *testing.B) {
	for sizeName, sizeBytes := range svgContentSizes {
		b.Run(fmt.Sprintf("%s_%dB", sizeName, sizeBytes), func(b *testing.B) {
			service, _ := createBaselineRegistryService(100000, sizeName)
			adapter := render_adapters.NewDataLoaderRegistryAdapter(service, nil, "/dist")
			ctx := context.Background()

			b.ResetTimer()
			b.ReportAllocs()

			i := 0
			for b.Loop() {

				_, _ = adapter.GetAssetRawSVG(ctx, fmt.Sprintf("svg-artefact-%d", i))
				i++
			}
		})
	}
}

func BenchmarkBaseline_BulkLoad_SVGs(b *testing.B) {
	batchSizes := []int{1, 5, 10, 25, 50}

	for _, batchSize := range batchSizes {
		b.Run(fmt.Sprintf("Batch_%d", batchSize), func(b *testing.B) {
			service, _ := createBaselineRegistryService(100, "small")
			adapter, ok := render_adapters.NewDataLoaderRegistryAdapter(service, nil, "/dist").(*render_adapters.DataLoaderRegistryAdapter)
			if !ok {
				b.Fatal("NewDataLoaderRegistryAdapter() should return *DataLoaderRegistryAdapter")
			}
			ctx := context.Background()

			assetIDs := make([]string, batchSize)
			for i := range batchSize {
				assetIDs[i] = fmt.Sprintf("svg-artefact-%d", i)
			}

			_, _ = adapter.BulkGetAssetRawSVG(ctx, assetIDs)

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_, _ = adapter.BulkGetAssetRawSVG(ctx, assetIDs)
			}
		})
	}
}

func BenchmarkBaseline_BulkLoad_ColdCache(b *testing.B) {
	batchSizes := []int{10, 50}

	for _, batchSize := range batchSizes {
		b.Run(fmt.Sprintf("Batch_%d", batchSize), func(b *testing.B) {

			service, _ := createBaselineRegistryService(100000, "small")
			adapter, ok := render_adapters.NewDataLoaderRegistryAdapter(service, nil, "/dist").(*render_adapters.DataLoaderRegistryAdapter)
			if !ok {
				b.Fatal("NewDataLoaderRegistryAdapter() should return *DataLoaderRegistryAdapter")
			}
			ctx := context.Background()

			b.ResetTimer()
			b.ReportAllocs()

			i := 0
			for b.Loop() {

				assetIDs := make([]string, batchSize)
				offset := i * batchSize
				for j := range batchSize {
					assetIDs[j] = fmt.Sprintf("svg-artefact-%d", offset+j)
				}
				_, _ = adapter.BulkGetAssetRawSVG(ctx, assetIDs)
				i++
			}
		})
	}
}

func BenchmarkBaseline_Concurrent_SVG(b *testing.B) {
	parallelisms := []int{1, 4, 8, 16, 32}

	for _, p := range parallelisms {
		b.Run(fmt.Sprintf("P%d", p), func(b *testing.B) {
			service, _ := createBaselineRegistryService(100, "small")
			adapter := render_adapters.NewDataLoaderRegistryAdapter(service, nil, "/dist")
			ctx := context.Background()

			for i := range 50 {
				_, _ = adapter.GetAssetRawSVG(ctx, fmt.Sprintf("svg-artefact-%d", i))
			}

			b.SetParallelism(p)
			b.ResetTimer()
			b.ReportAllocs()

			var counter int64
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					index := atomic.AddInt64(&counter, 1)
					assetID := fmt.Sprintf("svg-artefact-%d", index%50)
					_, _ = adapter.GetAssetRawSVG(ctx, assetID)
				}
			})
		})
	}
}

func BenchmarkBaseline_Concurrent_Component(b *testing.B) {
	parallelisms := []int{1, 4, 8, 16, 32}
	tags := []string{"my-card", "custom-button", "nav-menu", "footer-widget", "sidebar-panel"}

	for _, p := range parallelisms {
		b.Run(fmt.Sprintf("P%d", p), func(b *testing.B) {
			service, _ := createBaselineRegistryService(10, "small")
			adapter := render_adapters.NewDataLoaderRegistryAdapter(service, nil, "/dist")
			ctx := context.Background()

			for _, tag := range tags {
				_, _ = adapter.GetComponentMetadata(ctx, tag)
			}

			b.SetParallelism(p)
			b.ResetTimer()
			b.ReportAllocs()

			var counter int64
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					index := atomic.AddInt64(&counter, 1)
					tag := tags[index%int64(len(tags))]
					_, _ = adapter.GetComponentMetadata(ctx, tag)
				}
			})
		})
	}
}

func BenchmarkBaseline_Concurrent_Mixed(b *testing.B) {
	service, _ := createBaselineRegistryService(100, "small")
	adapter := render_adapters.NewDataLoaderRegistryAdapter(service, nil, "/dist")
	ctx := context.Background()
	tags := []string{"my-card", "custom-button", "nav-menu"}

	for i := range 50 {
		_, _ = adapter.GetAssetRawSVG(ctx, fmt.Sprintf("svg-artefact-%d", i))
	}
	for _, tag := range tags {
		_, _ = adapter.GetComponentMetadata(ctx, tag)
	}

	b.SetParallelism(16)
	b.ResetTimer()
	b.ReportAllocs()

	var counter int64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			index := atomic.AddInt64(&counter, 1)
			if index%3 == 0 {
				tag := tags[index%int64(len(tags))]
				_, _ = adapter.GetComponentMetadata(ctx, tag)
			} else {
				assetID := fmt.Sprintf("svg-artefact-%d", index%50)
				_, _ = adapter.GetAssetRawSVG(ctx, assetID)
			}
		}
	})
}

func BenchmarkBaseline_MixedWorkload_TypicalPage(b *testing.B) {

	service, _ := createBaselineRegistryService(100, "small")
	adapter := render_adapters.NewDataLoaderRegistryAdapter(service, nil, "/dist")
	ctx := context.Background()

	svgPattern := []string{
		"svg-artefact-0", "svg-artefact-1", "svg-artefact-2",
		"svg-artefact-0", "svg-artefact-3", "svg-artefact-1",
		"svg-artefact-4", "svg-artefact-2", "svg-artefact-0",
		"svg-artefact-3", "svg-artefact-1", "svg-artefact-4",
	}

	seen := make(map[string]bool)
	for _, id := range svgPattern {
		if !seen[id] {
			_, _ = adapter.GetAssetRawSVG(ctx, id)
			seen[id] = true
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {

		for _, id := range svgPattern {
			_, _ = adapter.GetAssetRawSVG(ctx, id)
		}
	}

	b.ReportMetric(float64(len(svgPattern)), "lookups/op")
}

func BenchmarkBaseline_MixedWorkload_HeavySVGPage(b *testing.B) {

	service, _ := createBaselineRegistryService(100, "small")
	adapter := render_adapters.NewDataLoaderRegistryAdapter(service, nil, "/dist")
	ctx := context.Background()

	svgPattern := make([]string, 50)
	for i := range 50 {
		svgPattern[i] = fmt.Sprintf("svg-artefact-%d", i%25)
	}

	for i := range 25 {
		_, _ = adapter.GetAssetRawSVG(ctx, fmt.Sprintf("svg-artefact-%d", i))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		for _, id := range svgPattern {
			_, _ = adapter.GetAssetRawSVG(ctx, id)
		}
	}

	b.ReportMetric(float64(len(svgPattern)), "lookups/op")
}

func BenchmarkBaseline_NotFound_SVG(b *testing.B) {
	service, _ := createBaselineRegistryService(10, "small")
	adapter := render_adapters.NewDataLoaderRegistryAdapter(service, nil, "/dist")
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, _ = adapter.GetAssetRawSVG(ctx, "non-existent-svg")
	}
}

func BenchmarkBaseline_NotFound_Component(b *testing.B) {
	service, _ := createBaselineRegistryService(10, "small")
	adapter := render_adapters.NewDataLoaderRegistryAdapter(service, nil, "/dist")
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, _ = adapter.GetComponentMetadata(ctx, "non-existent-component")
	}
}

func BenchmarkBaseline_AdapterCreation(b *testing.B) {
	service, _ := createBaselineRegistryService(10, "small")

	configs := []struct {
		config *render_adapters.DataLoaderAdapterConfig
		name   string
	}{
		{name: "Default", config: nil},
		{name: "Small", config: &render_adapters.DataLoaderAdapterConfig{
			ComponentCacheCapacity: 10,
			ComponentCacheTTL:      1 * time.Minute,
			SVGCacheCapacity:       20,
			SVGCacheTTL:            5 * time.Minute,
		}},
		{name: "Large", config: &render_adapters.DataLoaderAdapterConfig{
			ComponentCacheCapacity: 1000,
			ComponentCacheTTL:      30 * time.Minute,
			SVGCacheCapacity:       2000,
			SVGCacheTTL:            60 * time.Minute,
		}},
	}

	for _, config := range configs {
		b.Run(config.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				_ = render_adapters.NewDataLoaderRegistryAdapter(service, config.config, "/dist")
			}
		})
	}
}

func BenchmarkBaseline_RealVsMock(b *testing.B) {
	ctx := context.Background()

	b.Run("RealAdapter", func(b *testing.B) {
		service, _ := createBaselineRegistryService(10, "small")
		adapter := render_adapters.NewDataLoaderRegistryAdapter(service, nil, "/dist")

		_, _ = adapter.GetAssetRawSVG(ctx, "svg-artefact-0")

		b.ResetTimer()
		b.ReportAllocs()

		for b.Loop() {
			_, _ = adapter.GetAssetRawSVG(ctx, "svg-artefact-0")
		}
	})

	b.Run("MockRegistry", func(b *testing.B) {
		registry := NewBenchmarkRegistry()

		b.ResetTimer()
		b.ReportAllocs()

		for b.Loop() {
			_, _ = registry.GetAssetRawSVG(ctx, "testmodule/lib/icon.svg")
		}
	})
}

func BenchmarkBaseline_MemoryPressure(b *testing.B) {

	service, _ := createBaselineRegistryService(100, "large")
	ctx := context.Background()

	b.Run("SustainedCacheMisses", func(b *testing.B) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		startAlloc := m.TotalAlloc

		b.ResetTimer()
		b.ReportAllocs()

		for b.Loop() {
			adapter := render_adapters.NewDataLoaderRegistryAdapter(service, nil, "/dist")

			for j := range 10 {
				_, _ = adapter.GetAssetRawSVG(ctx, fmt.Sprintf("svg-artefact-%d", j))
			}
		}

		runtime.ReadMemStats(&m)
		b.ReportMetric(float64(m.TotalAlloc-startAlloc)/float64(b.N), "total-bytes/op")
	})
}

func BenchmarkBaseline_LatencyDistribution(b *testing.B) {
	service, _ := createBaselineRegistryService(100, "small")
	adapter := render_adapters.NewDataLoaderRegistryAdapter(service, nil, "/dist")
	ctx := context.Background()

	for i := range 50 {
		_, _ = adapter.GetAssetRawSVG(ctx, fmt.Sprintf("svg-artefact-%d", i))
	}

	b.Run("P50_CacheHit", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = adapter.GetAssetRawSVG(ctx, "svg-artefact-0")
		}
	})

	b.Run("P50_CacheMiss", func(b *testing.B) {

		largeService, _ := createBaselineRegistryService(100000, "small")
		missAdapter := render_adapters.NewDataLoaderRegistryAdapter(largeService, nil, "/dist")
		b.ReportAllocs()
		i := 0
		for b.Loop() {

			_, _ = missAdapter.GetAssetRawSVG(ctx, fmt.Sprintf("svg-artefact-miss-%d", i))
			i++
		}
	})
}
