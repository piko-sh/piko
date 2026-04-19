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

package daemon_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/daemon/daemon_adapters"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/templater/templater_dto"
)

type BenchCacheHarness struct {
	Router            *chi.Mux
	RegistryService   *testRegistryService
	ManifestStore     *testManifestStoreView
	CapabilityService *capabilities_domain.MockCapabilityService
	ServerConfig      *bootstrap.ServerConfig
	CacheMiddleware   *daemon_adapters.CacheMiddleware
	HandlerBody       []byte
}

func NewBenchCacheHarness(b *testing.B) *BenchCacheHarness {
	b.Helper()

	serverConfig := &bootstrap.ServerConfig{
		Paths: config.PathsConfig{
			BaseDir:          new("/test"),
			PartialServePath: new("/_piko/partials"),
		},
		Network: config.NetworkConfig{
			Port: new("8080"),
		},
	}

	handlerBody := bytes.Repeat([]byte("<div class=\"content\">Lorem ipsum dolor sit amet</div>\n"), 80)

	h := &BenchCacheHarness{
		Router:            chi.NewRouter(),
		RegistryService:   newTestRegistryService(),
		ManifestStore:     newTestManifestStoreView(),
		CapabilityService: NewMockCapabilityService(),
		ServerConfig:      serverConfig,
		HandlerBody:       handlerBody,
	}

	return h
}

func (h *BenchCacheHarness) SetupCacheMiddleware() {
	h.CacheMiddleware = daemon_adapters.NewCacheMiddleware(
		daemon_adapters.CacheMiddlewareConfig{
			StreamCompressionLevel: 4,
			CacheWriteConcurrency:  10,
		},
		h.ManifestStore,
		h.RegistryService,
		h.CapabilityService,
		*h.ServerConfig.Paths.PartialServePath,
	)
}

func (h *BenchCacheHarness) AddCacheablePage(path, routePattern string, policy templater_dto.CachePolicy) {
	entry := newTestPageEntryView()
	entry.OriginalPath_ = path
	entry.RoutePatterns_ = map[string]string{"en": routePattern}
	entry.IsPage_ = true
	entry.HasCachePolicy_ = true
	entry.CachePolicy_ = policy
	h.ManifestStore.AddEntry(path, entry)
}

func (h *BenchCacheHarness) CreateTestHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(h.HandlerBody)
	})
}

func (h *BenchCacheHarness) MountRoute(pattern string) {
	h.SetupCacheMiddleware()
	h.Router.Method(http.MethodGet, pattern, h.CacheMiddleware.Handle(h.CreateTestHandler()))
}

func BenchmarkCacheMiddleware_CacheHit(b *testing.B) {
	h := NewBenchCacheHarness(b)

	h.AddCacheablePage("pages/home", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		MaxAgeSeconds: 3600,
	})

	h.MountRoute("/")

	cachedHTML := h.HandlerBody
	brotliData := CompressBrotli(cachedHTML)

	sourceVariant := CreateVariantWithTags("source", "cache/home/source", map[string]string{
		"type": "source",
		"etag": `"bench-etag"`,
	}, cachedHTML)

	brotliVariant := CreateVariantWithTags("brotli", "cache/home/brotli", map[string]string{
		"type":            "cached-page",
		"contentEncoding": "br",
		"etag":            `"brotli-etag"`,
	}, brotliData)

	h.RegistryService.AddVariantData("cache/home/source", cachedHTML)
	h.RegistryService.AddVariantData("cache/home/brotli", brotliData)

	h.RegistryService.GetArtefactFunc = func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
		return &registry_dto.ArtefactMeta{
			ID:             artefactID,
			UpdatedAt:      time.Now(),
			ActualVariants: []registry_dto.Variant{sourceVariant, brotliVariant},
			Status:         registry_dto.VariantStatusReady,
		}, nil
	}

	request := Get("/").WithAcceptEncoding("br, gzip").Build()

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		recorder := httptest.NewRecorder()
		h.Router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", recorder.Code)
		}
	}
}

func BenchmarkCacheMiddleware_CacheHit_NoCompression(b *testing.B) {
	h := NewBenchCacheHarness(b)

	h.AddCacheablePage("pages/home", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		MaxAgeSeconds: 3600,
	})

	h.MountRoute("/")

	cachedHTML := h.HandlerBody
	sourceVariant := CreateVariantWithTags("source", "cache/home/source", map[string]string{
		"type": "source",
		"etag": `"bench-etag"`,
	}, cachedHTML)

	h.RegistryService.AddVariantData("cache/home/source", cachedHTML)

	h.RegistryService.GetArtefactFunc = func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
		return &registry_dto.ArtefactMeta{
			ID:             artefactID,
			UpdatedAt:      time.Now(),
			ActualVariants: []registry_dto.Variant{sourceVariant},
			Status:         registry_dto.VariantStatusReady,
		}, nil
	}

	request := Get("/").Build()

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		recorder := httptest.NewRecorder()
		h.Router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", recorder.Code)
		}
	}
}

func BenchmarkCacheMiddleware_CacheHit_ETagMatch(b *testing.B) {
	h := NewBenchCacheHarness(b)

	h.AddCacheablePage("pages/home", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		MaxAgeSeconds: 3600,
	})

	h.MountRoute("/")

	cachedHTML := h.HandlerBody
	etag := `"bench-etag"`
	sourceVariant := CreateVariantWithTags("source", "cache/home/source", map[string]string{
		"type": "source",
		"etag": etag,
	}, cachedHTML)

	h.RegistryService.AddVariantData("cache/home/source", cachedHTML)

	h.RegistryService.GetArtefactFunc = func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
		return &registry_dto.ArtefactMeta{
			ID:             artefactID,
			UpdatedAt:      time.Now(),
			ActualVariants: []registry_dto.Variant{sourceVariant},
			Status:         registry_dto.VariantStatusReady,
		}, nil
	}

	request := Get("/").WithIfNoneMatch(etag).Build()

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		recorder := httptest.NewRecorder()
		h.Router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusNotModified {
			b.Fatalf("unexpected status: %d", recorder.Code)
		}
	}
}

func BenchmarkCacheMiddleware_CacheMiss(b *testing.B) {
	h := NewBenchCacheHarness(b)

	h.AddCacheablePage("pages/home", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		MaxAgeSeconds: 3600,
	})

	h.MountRoute("/")

	h.RegistryService.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
		return nil, registry_domain.ErrArtefactNotFound
	}

	request := Get("/").Build()

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		recorder := httptest.NewRecorder()
		h.Router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", recorder.Code)
		}
	}
}

func BenchmarkCacheMiddleware_CacheMiss_WithBrotli(b *testing.B) {
	h := NewBenchCacheHarness(b)

	h.AddCacheablePage("pages/home", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		MaxAgeSeconds: 3600,
	})

	h.MountRoute("/")

	h.RegistryService.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
		return nil, registry_domain.ErrArtefactNotFound
	}

	request := Get("/").WithAcceptEncoding("br, gzip").Build()

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		recorder := httptest.NewRecorder()
		h.Router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", recorder.Code)
		}
	}
}

func BenchmarkCacheMiddleware_Bypass_NonStatic(b *testing.B) {
	h := NewBenchCacheHarness(b)

	h.AddCacheablePage("pages/dynamic", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        false,
		MaxAgeSeconds: 0,
	})

	h.MountRoute("/")

	request := Get("/").Build()

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		recorder := httptest.NewRecorder()
		h.Router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", recorder.Code)
		}
	}
}

func BenchmarkCacheMiddleware_Bypass_WithBrotliStreaming(b *testing.B) {
	h := NewBenchCacheHarness(b)

	h.AddCacheablePage("pages/dynamic", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        false,
		MaxAgeSeconds: 0,
	})

	h.MountRoute("/")

	request := Get("/").WithAcceptEncoding("br, gzip").Build()

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		recorder := httptest.NewRecorder()
		h.Router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", recorder.Code)
		}
	}
}

func BenchmarkCacheMiddleware_Singleflight(b *testing.B) {
	h := NewBenchCacheHarness(b)

	h.AddCacheablePage("pages/home", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		MaxAgeSeconds: 3600,
	})

	h.MountRoute("/")

	h.RegistryService.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
		return nil, registry_domain.ErrArtefactNotFound
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		request := Get("/").Build()
		for pb.Next() {
			recorder := httptest.NewRecorder()
			h.Router.ServeHTTP(recorder, request)
		}
	})
}

func BenchmarkCacheMiddleware_VariantSelection(b *testing.B) {
	h := NewBenchCacheHarness(b)

	h.AddCacheablePage("pages/home", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		MaxAgeSeconds: 3600,
	})

	h.MountRoute("/")

	cachedHTML := h.HandlerBody
	brotliData := CompressBrotli(cachedHTML)
	gzipData := CompressGzip(cachedHTML)

	variants := []registry_dto.Variant{
		CreateVariantWithTags("source", "cache/home/source", map[string]string{
			"type": "source",
			"etag": `"source-etag"`,
		}, cachedHTML),
		CreateVariantWithTags("minified", "cache/home/minified", map[string]string{
			"type": "minified-html",
			"etag": `"minified-etag"`,
		}, cachedHTML),
		CreateVariantWithTags("gzip", "cache/home/gzip", map[string]string{
			"type":            "cached-page",
			"contentEncoding": "gzip",
			"etag":            `"gzip-etag"`,
		}, gzipData),
		CreateVariantWithTags("brotli", "cache/home/brotli", map[string]string{
			"type":            "cached-page",
			"contentEncoding": "br",
			"etag":            `"brotli-etag"`,
		}, brotliData),
	}

	h.RegistryService.AddVariantData("cache/home/source", cachedHTML)
	h.RegistryService.AddVariantData("cache/home/minified", cachedHTML)
	h.RegistryService.AddVariantData("cache/home/gzip", gzipData)
	h.RegistryService.AddVariantData("cache/home/brotli", brotliData)

	h.RegistryService.GetArtefactFunc = func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
		return &registry_dto.ArtefactMeta{
			ID:             artefactID,
			UpdatedAt:      time.Now(),
			ActualVariants: variants,
			Status:         registry_dto.VariantStatusReady,
		}, nil
	}

	request := Get("/").WithAcceptEncoding("br, gzip, deflate").Build()

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		recorder := httptest.NewRecorder()
		h.Router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", recorder.Code)
		}
	}
}

func BenchmarkBrotliCompression(b *testing.B) {
	data := bytes.Repeat([]byte("<div class=\"content\">Lorem ipsum dolor sit amet</div>\n"), 80)

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	for b.Loop() {
		_ = CompressBrotli(data)
	}
}

func BenchmarkGzipCompression(b *testing.B) {
	data := bytes.Repeat([]byte("<div class=\"content\">Lorem ipsum dolor sit amet</div>\n"), 80)

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	for b.Loop() {
		_ = CompressGzip(data)
	}
}

func BenchmarkMockCapabilityService_Execute(b *testing.B) {
	service := NewMockCapabilityService()
	data := bytes.Repeat([]byte("test data"), 100)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		reader := bytes.NewReader(data)
		_, _ = service.Execute(context.Background(), "compress_brotli", reader, nil)
	}
}

func BenchmarkRouteMapLookup(b *testing.B) {
	routeMap := make(map[string]string)
	for i := range 100 {
		routeMap["/page/"+string(rune('a'+i%26))+"/"+string(rune('0'+i%10))] = "pages/page" + string(rune('0'+i%10))
	}

	pattern := "/page/m/5"

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_ = routeMap[pattern]
	}
}

func BenchmarkCacheMiddleware_Allocations_CacheHit(b *testing.B) {
	h := NewBenchCacheHarness(b)

	h.AddCacheablePage("pages/home", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		MaxAgeSeconds: 3600,
	})

	h.MountRoute("/")

	cachedHTML := []byte("<html>Small cached content</html>")
	sourceVariant := CreateVariantWithTags("source", "cache/home/source", map[string]string{
		"type": "source",
		"etag": `"small-etag"`,
	}, cachedHTML)

	h.RegistryService.AddVariantData("cache/home/source", cachedHTML)

	h.RegistryService.GetArtefactFunc = func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
		return &registry_dto.ArtefactMeta{
			ID:             artefactID,
			UpdatedAt:      time.Now(),
			ActualVariants: []registry_dto.Variant{sourceVariant},
			Status:         registry_dto.VariantStatusReady,
		}, nil
	}

	request := Get("/").Build()

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		recorder := httptest.NewRecorder()
		h.Router.ServeHTTP(recorder, request)
	}
}

func BenchmarkCacheMiddleware_Concurrent_CacheHit(b *testing.B) {
	h := NewBenchCacheHarness(b)

	h.AddCacheablePage("pages/home", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		MaxAgeSeconds: 3600,
	})

	h.MountRoute("/")

	cachedHTML := h.HandlerBody
	sourceVariant := CreateVariantWithTags("source", "cache/home/source", map[string]string{
		"type": "source",
		"etag": `"bench-etag"`,
	}, cachedHTML)

	h.RegistryService.AddVariantData("cache/home/source", cachedHTML)

	var mu sync.Mutex
	h.RegistryService.GetArtefactFunc = func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
		mu.Lock()
		defer mu.Unlock()
		return &registry_dto.ArtefactMeta{
			ID:             artefactID,
			UpdatedAt:      time.Now(),
			ActualVariants: []registry_dto.Variant{sourceVariant},
			Status:         registry_dto.VariantStatusReady,
		}, nil
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		request := Get("/").Build()
		for pb.Next() {
			recorder := httptest.NewRecorder()
			h.Router.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusOK {
				b.Fatalf("unexpected status: %d", recorder.Code)
			}
		}
	})
}
