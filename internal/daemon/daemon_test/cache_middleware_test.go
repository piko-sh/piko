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

package daemon_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/capabilities/capabilities_domain"

	"piko.sh/piko/internal/daemon/daemon_adapters"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/templater/templater_dto"
)

func NewMockCapabilityService() *capabilities_domain.MockCapabilityService {
	return &capabilities_domain.MockCapabilityService{
		ExecuteFunc: func(_ context.Context, _ string, inputData io.Reader, _ capabilities_domain.CapabilityParams) (io.Reader, error) {
			data, err := io.ReadAll(inputData)
			if err != nil {
				return nil, err
			}
			return bytes.NewReader(data), nil
		},
	}
}

type CacheTestHarness struct {
	T                 *testing.T
	Router            *chi.Mux
	RegistryService   *testRegistryService
	ManifestStore     *testManifestStoreView
	CapabilityService *capabilities_domain.MockCapabilityService
	PartialServePath  string
	CacheMiddleware   *daemon_adapters.CacheMiddleware
	HandlerBody       string
	HandlerCalls      atomic.Int32
}

func NewCacheTestHarness(t *testing.T) *CacheTestHarness {
	t.Helper()

	h := &CacheTestHarness{
		T:                 t,
		Router:            chi.NewRouter(),
		RegistryService:   newTestRegistryService(),
		ManifestStore:     newTestManifestStoreView(),
		CapabilityService: NewMockCapabilityService(),
		PartialServePath:  "/_piko/partials",
		HandlerBody:       "<html><body>Test Page</body></html>",
	}

	return h
}

func (h *CacheTestHarness) SetupCacheMiddleware() {
	h.CacheMiddleware = daemon_adapters.NewCacheMiddleware(
		daemon_adapters.CacheMiddlewareConfig{
			StreamCompressionLevel: 4,
			CacheWriteConcurrency:  10,
		},
		h.ManifestStore,
		h.RegistryService,
		h.CapabilityService,
		h.PartialServePath,
	)
}

func (h *CacheTestHarness) AddCacheablePage(path, routePattern string, policy templater_dto.CachePolicy) {
	entry := newTestPageEntryView()
	entry.OriginalPath_ = path
	entry.RoutePatterns_ = map[string]string{"en": routePattern}
	entry.IsPage_ = true
	entry.HasCachePolicy_ = true
	entry.CachePolicy_ = policy
	h.ManifestStore.AddEntry(path, entry)
}

func (h *CacheTestHarness) CreateTestHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		h.HandlerCalls.Add(1)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(h.HandlerBody))
	})
}

func (h *CacheTestHarness) MountRoute(pattern string) {
	h.SetupCacheMiddleware()
	h.Router.Method(http.MethodGet, pattern, h.CacheMiddleware.Handle(h.CreateTestHandler()))
}

func (h *CacheTestHarness) DoRequest(request *http.Request) *httptest.ResponseRecorder {
	h.T.Helper()
	recorder := httptest.NewRecorder()
	h.Router.ServeHTTP(recorder, request)
	return recorder
}

func (h *CacheTestHarness) CreateCachedArtefact(artefactID string, updatedAt time.Time, variants ...registry_dto.Variant) {
	artefact := &registry_dto.ArtefactMeta{
		ID:             artefactID,
		UpdatedAt:      updatedAt,
		ActualVariants: variants,
		Status:         registry_dto.VariantStatusReady,
	}
	h.RegistryService.AddArtefact(artefact)
}

func CreateVariantWithTags(variantID, storageKey string, tags map[string]string, data []byte) registry_dto.Variant {
	v := registry_dto.Variant{
		VariantID:  variantID,
		StorageKey: storageKey,
		MimeType:   "text/html",
		Status:     registry_dto.VariantStatusReady,
		SizeBytes:  int64(len(data)),
	}
	for k, value := range tags {
		v.MetadataTags.SetByName(k, value)
	}
	return v
}

func TestCacheMiddleware_CacheHit_ServesFromCache(t *testing.T) {
	t.Parallel()

	h := NewCacheTestHarness(t)

	h.AddCacheablePage("pages/home", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		MaxAgeSeconds: 3600,
	})

	h.MountRoute("/")

	cachedHTML := "<html><body>Cached Content</body></html>"
	variant := CreateVariantWithTags("source", "cache/home/source", map[string]string{
		"type": "source",
		"etag": `"abc123"`,
	}, []byte(cachedHTML))
	h.RegistryService.AddVariantData("cache/home/source", []byte(cachedHTML))

	h.RegistryService.GetArtefactFunc = func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
		return &registry_dto.ArtefactMeta{
			ID:             artefactID,
			UpdatedAt:      time.Now(),
			ActualVariants: []registry_dto.Variant{variant},
			Status:         registry_dto.VariantStatusReady,
		}, nil
	}

	request := Get("/").Build()
	recorder := h.DoRequest(request)

	AssertStatus(t, recorder, http.StatusOK)
	AssertHeader(t, recorder, "X-Cache-Status", "HIT")

	assert.Equal(t, int32(0), h.HandlerCalls.Load(), "handler should not be called on cache hit")

	AssertBodyEquals(t, recorder, cachedHTML)
}

func TestCacheMiddleware_CacheHit_SelectsBrotliVariant(t *testing.T) {
	t.Parallel()

	h := NewCacheTestHarness(t)

	h.AddCacheablePage("pages/home", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		MaxAgeSeconds: 3600,
	})

	h.MountRoute("/")

	sourceHTML := "<html><body>Source</body></html>"
	brotliData := CompressBrotli([]byte(sourceHTML))
	gzipData := CompressGzip([]byte(sourceHTML))

	sourceVariant := CreateVariantWithTags("source", "cache/home/source", map[string]string{
		"type": "source",
		"etag": `"source-etag"`,
	}, []byte(sourceHTML))

	brotliVariant := CreateVariantWithTags("brotli", "cache/home/brotli", map[string]string{
		"type":            "cached-page",
		"contentEncoding": "br",
		"etag":            `"brotli-etag"`,
	}, brotliData)

	gzipVariant := CreateVariantWithTags("gzip", "cache/home/gzip", map[string]string{
		"type":            "cached-page",
		"contentEncoding": "gzip",
		"etag":            `"gzip-etag"`,
	}, gzipData)

	h.RegistryService.AddVariantData("cache/home/source", []byte(sourceHTML))
	h.RegistryService.AddVariantData("cache/home/brotli", brotliData)
	h.RegistryService.AddVariantData("cache/home/gzip", gzipData)

	h.RegistryService.GetArtefactFunc = func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
		return &registry_dto.ArtefactMeta{
			ID:             artefactID,
			UpdatedAt:      time.Now(),
			ActualVariants: []registry_dto.Variant{sourceVariant, gzipVariant, brotliVariant},
			Status:         registry_dto.VariantStatusReady,
		}, nil
	}

	request := Get("/").WithAcceptEncoding("br, gzip, deflate").Build()
	recorder := h.DoRequest(request)

	AssertStatus(t, recorder, http.StatusOK)
	AssertHeader(t, recorder, "X-Cache-Status", "HIT")
	AssertHeader(t, recorder, "Content-Encoding", "br")
	AssertHeader(t, recorder, "ETag", `"brotli-etag"`)

	decompressed := DecompressResponse(t, recorder)
	assert.Equal(t, sourceHTML, string(decompressed))
}

func TestCacheMiddleware_CacheHit_SelectsGzipVariant_WhenNoBrotli(t *testing.T) {
	t.Parallel()

	h := NewCacheTestHarness(t)

	h.AddCacheablePage("pages/home", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		MaxAgeSeconds: 3600,
	})

	h.MountRoute("/")

	sourceHTML := "<html><body>Source</body></html>"
	gzipData := CompressGzip([]byte(sourceHTML))

	sourceVariant := CreateVariantWithTags("source", "cache/home/source", map[string]string{
		"type": "source",
		"etag": `"source-etag"`,
	}, []byte(sourceHTML))

	gzipVariant := CreateVariantWithTags("gzip", "cache/home/gzip", map[string]string{
		"type":            "cached-page",
		"contentEncoding": "gzip",
		"etag":            `"gzip-etag"`,
	}, gzipData)

	h.RegistryService.AddVariantData("cache/home/source", []byte(sourceHTML))
	h.RegistryService.AddVariantData("cache/home/gzip", gzipData)

	h.RegistryService.GetArtefactFunc = func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
		return &registry_dto.ArtefactMeta{
			ID:             artefactID,
			UpdatedAt:      time.Now(),
			ActualVariants: []registry_dto.Variant{sourceVariant, gzipVariant},
			Status:         registry_dto.VariantStatusReady,
		}, nil
	}

	request := Get("/").WithAcceptEncoding("gzip, deflate").Build()
	recorder := h.DoRequest(request)

	AssertStatus(t, recorder, http.StatusOK)
	AssertHeader(t, recorder, "X-Cache-Status", "HIT")
	AssertHeader(t, recorder, "Content-Encoding", "gzip")
	AssertHeader(t, recorder, "ETag", `"gzip-etag"`)
}

func TestCacheMiddleware_CacheMiss_GeneratesAndCaches(t *testing.T) {
	t.Parallel()

	h := NewCacheTestHarness(t)

	h.AddCacheablePage("pages/home", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		MaxAgeSeconds: 3600,
	})

	h.MountRoute("/")

	request := Get("/").Build()
	recorder := h.DoRequest(request)

	AssertStatus(t, recorder, http.StatusOK)
	AssertHeader(t, recorder, "X-Cache-Status", "MISS")

	assert.Equal(t, int32(1), h.HandlerCalls.Load(), "handler should be called on cache miss")

	AssertBodyEquals(t, recorder, h.HandlerBody)

	require.Eventually(t, func() bool {
		return atomic.LoadInt64(&h.RegistryService.UpsertArtefactCallCount) >= 1
	}, time.Second, 5*time.Millisecond, "should have called upsert for caching")
}

func TestCacheMiddleware_CacheMiss_SetsCorrectHeaders(t *testing.T) {
	t.Parallel()

	h := NewCacheTestHarness(t)

	h.AddCacheablePage("pages/home", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		MaxAgeSeconds: 3600,
	})

	h.MountRoute("/")

	request := Get("/").Build()
	recorder := h.DoRequest(request)

	AssertStatus(t, recorder, http.StatusOK)
	AssertHeader(t, recorder, "X-Cache-Status", "MISS")
	AssertHeader(t, recorder, "Content-Type", "text/html; charset=utf-8")
	AssertHeaderExists(t, recorder, "ETag")
	AssertHeaderExists(t, recorder, "Cache-Control")
}

func TestCacheMiddleware_CacheStale_RegeneratesContent(t *testing.T) {
	t.Parallel()

	h := NewCacheTestHarness(t)

	h.AddCacheablePage("pages/home", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		MaxAgeSeconds: 60,
	})

	h.MountRoute("/")

	staleTime := time.Now().Add(-120 * time.Second)
	cachedHTML := "<html><body>Stale Content</body></html>"
	variant := CreateVariantWithTags("source", "cache/home/source", map[string]string{
		"type": "source",
		"etag": `"stale-etag"`,
	}, []byte(cachedHTML))

	h.RegistryService.GetArtefactFunc = func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
		return &registry_dto.ArtefactMeta{
			ID:             artefactID,
			UpdatedAt:      staleTime,
			ActualVariants: []registry_dto.Variant{variant},
			Status:         registry_dto.VariantStatusReady,
		}, nil
	}

	request := Get("/").Build()
	recorder := h.DoRequest(request)

	AssertStatus(t, recorder, http.StatusOK)
	AssertHeader(t, recorder, "X-Cache-Status", "STALE")

	assert.Equal(t, int32(1), h.HandlerCalls.Load(), "handler should be called when cache is stale")

	AssertBodyEquals(t, recorder, h.HandlerBody)
}

func TestCacheMiddleware_Bypass_WhenDisabled(t *testing.T) {
	t.Parallel()

	h := NewCacheTestHarness(t)

	h.AddCacheablePage("pages/home", "/", templater_dto.CachePolicy{
		Enabled:       false,
		Static:        true,
		MaxAgeSeconds: 3600,
	})

	h.MountRoute("/")

	request := Get("/").Build()
	recorder := h.DoRequest(request)

	AssertStatus(t, recorder, http.StatusOK)

	AssertHeaderNotExists(t, recorder, "X-Cache-Status")

	assert.Equal(t, int32(1), h.HandlerCalls.Load())
}

func TestCacheMiddleware_Bypass_WhenNoStore(t *testing.T) {
	t.Parallel()

	h := NewCacheTestHarness(t)

	h.AddCacheablePage("pages/home", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		NoStore:       true,
		MaxAgeSeconds: 3600,
	})

	h.MountRoute("/")

	request := Get("/").Build()
	recorder := h.DoRequest(request)

	AssertStatus(t, recorder, http.StatusOK)
	AssertHeaderNotExists(t, recorder, "X-Cache-Status")
	assert.Equal(t, int32(1), h.HandlerCalls.Load())
}

func TestCacheMiddleware_Bypass_WhenNotStatic(t *testing.T) {
	t.Parallel()

	h := NewCacheTestHarness(t)

	h.AddCacheablePage("pages/home", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        false,
		MaxAgeSeconds: 3600,
	})

	h.MountRoute("/")

	request := Get("/").Build()
	recorder := h.DoRequest(request)

	AssertStatus(t, recorder, http.StatusOK)
	AssertHeaderNotExists(t, recorder, "X-Cache-Status")
	assert.Equal(t, int32(1), h.HandlerCalls.Load())
}

func TestCacheMiddleware_Bypass_WhenRouteNotMapped(t *testing.T) {
	t.Parallel()

	h := NewCacheTestHarness(t)

	h.SetupCacheMiddleware()

	h.Router.Method(http.MethodGet, "/unmapped", h.CacheMiddleware.Handle(h.CreateTestHandler()))

	request := Get("/unmapped").Build()
	recorder := h.DoRequest(request)

	AssertStatus(t, recorder, http.StatusOK)

	AssertHeaderNotExists(t, recorder, "X-Cache-Status")
	assert.Equal(t, int32(1), h.HandlerCalls.Load())
}

func TestCacheMiddleware_ETag_Returns304_WhenMatched(t *testing.T) {
	t.Parallel()

	h := NewCacheTestHarness(t)

	h.AddCacheablePage("pages/home", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		MaxAgeSeconds: 3600,
	})

	h.MountRoute("/")

	cachedHTML := "<html><body>Cached Content</body></html>"
	etag := `"matched-etag"`
	variant := CreateVariantWithTags("source", "cache/home/source", map[string]string{
		"type": "source",
		"etag": etag,
	}, []byte(cachedHTML))
	h.RegistryService.AddVariantData("cache/home/source", []byte(cachedHTML))

	h.RegistryService.GetArtefactFunc = func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
		return &registry_dto.ArtefactMeta{
			ID:             artefactID,
			UpdatedAt:      time.Now(),
			ActualVariants: []registry_dto.Variant{variant},
			Status:         registry_dto.VariantStatusReady,
		}, nil
	}

	request := Get("/").WithIfNoneMatch(etag).Build()
	recorder := h.DoRequest(request)

	AssertNotModified(t, recorder)

	assert.Equal(t, int32(0), h.HandlerCalls.Load())
}

func TestCacheMiddleware_ETag_Returns200_WhenNotMatched(t *testing.T) {
	t.Parallel()

	h := NewCacheTestHarness(t)

	h.AddCacheablePage("pages/home", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		MaxAgeSeconds: 3600,
	})

	h.MountRoute("/")

	cachedHTML := "<html><body>Cached Content</body></html>"
	variant := CreateVariantWithTags("source", "cache/home/source", map[string]string{
		"type": "source",
		"etag": `"current-etag"`,
	}, []byte(cachedHTML))
	h.RegistryService.AddVariantData("cache/home/source", []byte(cachedHTML))

	h.RegistryService.GetArtefactFunc = func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
		return &registry_dto.ArtefactMeta{
			ID:             artefactID,
			UpdatedAt:      time.Now(),
			ActualVariants: []registry_dto.Variant{variant},
			Status:         registry_dto.VariantStatusReady,
		}, nil
	}

	request := Get("/").WithIfNoneMatch(`"old-etag"`).Build()
	recorder := h.DoRequest(request)

	AssertStatus(t, recorder, http.StatusOK)
	AssertHeader(t, recorder, "X-Cache-Status", "HIT")
	AssertBodyEquals(t, recorder, cachedHTML)
}

func TestCacheMiddleware_VariantSelection_FallsBackToSource(t *testing.T) {
	t.Parallel()

	h := NewCacheTestHarness(t)

	h.AddCacheablePage("pages/home", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		MaxAgeSeconds: 3600,
	})

	h.MountRoute("/")

	sourceHTML := "<html><body>Source Only</body></html>"
	sourceVariant := CreateVariantWithTags("source", "cache/home/source", map[string]string{
		"type": "source",
		"etag": `"source-etag"`,
	}, []byte(sourceHTML))
	h.RegistryService.AddVariantData("cache/home/source", []byte(sourceHTML))

	h.RegistryService.GetArtefactFunc = func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
		return &registry_dto.ArtefactMeta{
			ID:             artefactID,
			UpdatedAt:      time.Now(),
			ActualVariants: []registry_dto.Variant{sourceVariant},
			Status:         registry_dto.VariantStatusReady,
		}, nil
	}

	request := Get("/").WithAcceptEncoding("br, gzip").Build()
	recorder := h.DoRequest(request)

	AssertStatus(t, recorder, http.StatusOK)
	AssertHeader(t, recorder, "X-Cache-Status", "HIT")

	AssertBodyEquals(t, recorder, sourceHTML)
}

func TestCacheMiddleware_VariantSelection_MinifiedHTML_WhenNoCompression(t *testing.T) {
	t.Parallel()

	h := NewCacheTestHarness(t)

	h.AddCacheablePage("pages/home", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		MaxAgeSeconds: 3600,
	})

	h.MountRoute("/")

	sourceHTML := "<html><body>Source</body></html>"
	minifiedHTML := "<html><body>Minified</body></html>"

	sourceVariant := CreateVariantWithTags("source", "cache/home/source", map[string]string{
		"type": "source",
		"etag": `"source-etag"`,
	}, []byte(sourceHTML))

	minifiedVariant := CreateVariantWithTags("minified", "cache/home/minified", map[string]string{
		"type": "minified-html",
		"etag": `"minified-etag"`,
	}, []byte(minifiedHTML))

	h.RegistryService.AddVariantData("cache/home/source", []byte(sourceHTML))
	h.RegistryService.AddVariantData("cache/home/minified", []byte(minifiedHTML))

	h.RegistryService.GetArtefactFunc = func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
		return &registry_dto.ArtefactMeta{
			ID:             artefactID,
			UpdatedAt:      time.Now(),
			ActualVariants: []registry_dto.Variant{sourceVariant, minifiedVariant},
			Status:         registry_dto.VariantStatusReady,
		}, nil
	}

	request := Get("/").Build()
	recorder := h.DoRequest(request)

	AssertStatus(t, recorder, http.StatusOK)
	AssertHeader(t, recorder, "X-Cache-Status", "HIT")
	AssertBodyEquals(t, recorder, minifiedHTML)
}

func TestCacheMiddleware_Singleflight_CoalescesConcurrentRequests(t *testing.T) {
	t.Parallel()

	h := NewCacheTestHarness(t)

	h.AddCacheablePage("pages/home", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		MaxAgeSeconds: 3600,
	})

	var handlerCalls atomic.Int32
	slowHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handlerCalls.Add(1)
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html>Slow Content</html>"))
	})

	h.SetupCacheMiddleware()
	h.Router.Method(http.MethodGet, "/", h.CacheMiddleware.Handle(slowHandler))

	const concurrentRequests = 5
	var wg sync.WaitGroup
	responses := make([]*httptest.ResponseRecorder, concurrentRequests)

	for i := range concurrentRequests {
		index := i
		wg.Go(func() {
			request := Get("/").Build()
			responses[index] = h.DoRequest(request)
		})
	}

	wg.Wait()

	for i, recorder := range responses {
		AssertStatus(t, recorder, http.StatusOK)
		assert.NotEmpty(t, recorder.Body.String(), "response %d should have content", i)
	}

	actualCalls := handlerCalls.Load()
	assert.Equal(t, int32(1), actualCalls,
		"singleflight should coalesce concurrent requests; got %d calls", actualCalls)
}

func TestCacheMiddleware_NonCacheable_StreamsBrotliCompression(t *testing.T) {
	t.Parallel()

	h := NewCacheTestHarness(t)

	h.AddCacheablePage("pages/dynamic", "/dynamic", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        false,
		MaxAgeSeconds: 0,
	})

	h.MountRoute("/dynamic")

	request := Get("/dynamic").WithAcceptEncoding("br, gzip").Build()
	recorder := h.DoRequest(request)

	AssertStatus(t, recorder, http.StatusOK)

	AssertHeader(t, recorder, "Content-Encoding", "br")
	AssertHeaderNotExists(t, recorder, "X-Cache-Status")

	assert.Equal(t, int32(1), h.HandlerCalls.Load())

	decompressed := DecompressResponse(t, recorder)
	assert.Equal(t, h.HandlerBody, string(decompressed))
}

func TestCacheMiddleware_NonCacheable_StreamsGzipCompression(t *testing.T) {
	t.Parallel()

	h := NewCacheTestHarness(t)

	h.AddCacheablePage("pages/dynamic", "/dynamic", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        false,
		MaxAgeSeconds: 0,
	})

	h.MountRoute("/dynamic")

	request := Get("/dynamic").WithAcceptEncoding("gzip, deflate").Build()
	recorder := h.DoRequest(request)

	AssertStatus(t, recorder, http.StatusOK)
	AssertHeader(t, recorder, "Content-Encoding", "gzip")
	AssertHeaderNotExists(t, recorder, "X-Cache-Status")

	decompressed := DecompressResponse(t, recorder)
	assert.Equal(t, h.HandlerBody, string(decompressed))
}

func TestCacheMiddleware_NonCacheable_NoCompression_WhenNotSupported(t *testing.T) {
	t.Parallel()

	h := NewCacheTestHarness(t)

	h.AddCacheablePage("pages/dynamic", "/dynamic", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        false,
		MaxAgeSeconds: 0,
	})

	h.MountRoute("/dynamic")

	request := Get("/dynamic").Build()
	recorder := h.DoRequest(request)

	AssertStatus(t, recorder, http.StatusOK)

	AssertHeaderNotExists(t, recorder, "Content-Encoding")
	AssertHeaderNotExists(t, recorder, "X-Cache-Status")

	AssertBodyEquals(t, recorder, h.HandlerBody)
}

func TestCacheMiddleware_CacheKey_IncludesPolicyKey(t *testing.T) {
	t.Parallel()

	h := NewCacheTestHarness(t)

	h.AddCacheablePage("pages/user", "/user", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		MaxAgeSeconds: 3600,
		Key:           "user-variant-a",
	})

	h.MountRoute("/user")

	var requestedArtefactIDs []string
	var mu sync.Mutex
	h.RegistryService.GetArtefactFunc = func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
		mu.Lock()
		requestedArtefactIDs = append(requestedArtefactIDs, artefactID)
		mu.Unlock()
		return nil, registry_domain.ErrArtefactNotFound
	}

	request := Get("/user").Build()
	_ = h.DoRequest(request)

	entry, ok := h.ManifestStore.Entries["pages/user"].(*testPageEntryView)
	require.True(t, ok, "expected entry to be *testPageEntryView")
	entry.CachePolicy_.Key = "user-variant-b"

	request = Get("/user").Build()
	_ = h.DoRequest(request)

	time.Sleep(50 * time.Millisecond)

	require.Len(t, requestedArtefactIDs, 2, "should have made 2 requests")
	assert.NotEqual(t, requestedArtefactIDs[0], requestedArtefactIDs[1],
		"different policy keys should produce different cache keys")
}

func TestCacheMiddleware_HandlesEmptyHandlerResponse(t *testing.T) {
	t.Parallel()

	h := NewCacheTestHarness(t)

	h.AddCacheablePage("pages/empty", "/empty", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		MaxAgeSeconds: 3600,
	})

	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)

	})

	h.SetupCacheMiddleware()
	h.Router.Method(http.MethodGet, "/empty", h.CacheMiddleware.Handle(emptyHandler))

	request := Get("/empty").Build()
	recorder := h.DoRequest(request)

	AssertStatus(t, recorder, http.StatusNoContent)
}

func TestCacheMiddleware_HandlesNonSuccessStatus(t *testing.T) {
	t.Parallel()

	h := NewCacheTestHarness(t)

	h.AddCacheablePage("pages/error", "/error", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		MaxAgeSeconds: 3600,
	})

	errorHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Error"))
	})

	h.SetupCacheMiddleware()
	h.Router.Method(http.MethodGet, "/error", h.CacheMiddleware.Handle(errorHandler))

	request := Get("/error").Build()
	recorder := h.DoRequest(request)

	AssertStatus(t, recorder, http.StatusInternalServerError)

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int64(0), atomic.LoadInt64(&h.RegistryService.UpsertArtefactCallCount),
		"should not cache non-success responses")
}

func TestCacheMiddleware_CacheHit_SetsMaxAgeHeader(t *testing.T) {
	t.Parallel()

	h := NewCacheTestHarness(t)

	maxAge := 7200
	h.AddCacheablePage("pages/home", "/", templater_dto.CachePolicy{
		Enabled:       true,
		Static:        true,
		MaxAgeSeconds: maxAge,
	})

	h.MountRoute("/")

	cachedHTML := "<html><body>Cached</body></html>"
	variant := CreateVariantWithTags("source", "cache/home/source", map[string]string{
		"type": "source",
		"etag": `"test-etag"`,
	}, []byte(cachedHTML))
	h.RegistryService.AddVariantData("cache/home/source", []byte(cachedHTML))

	h.RegistryService.GetArtefactFunc = func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
		return &registry_dto.ArtefactMeta{
			ID:             artefactID,
			UpdatedAt:      time.Now(),
			ActualVariants: []registry_dto.Variant{variant},
			Status:         registry_dto.VariantStatusReady,
		}, nil
	}

	request := Get("/").Build()
	recorder := h.DoRequest(request)

	AssertStatus(t, recorder, http.StatusOK)

	cacheControl := recorder.Header().Get("Cache-Control")
	require.Contains(t, cacheControl, "public")
	require.Contains(t, cacheControl, "max-age=7200")
}
