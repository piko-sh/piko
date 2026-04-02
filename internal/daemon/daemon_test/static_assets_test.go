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
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/security/security_dto"
)

func TestServeTheme_Success(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)
	cssContent := []byte("body { color: red; }")

	variant := CreateTestVariant("source", "theme.css/source",
		WithVariantMimeType("text/css"),
		WithVariantTag("type", "source"),
		WithVariantTag("etag", `"abc123"`),
	)
	artefact := CreateTestArtefact("theme.css", variant)
	h.RegistryService.AddArtefact(artefact)
	h.RegistryService.AddVariantData("theme.css/source", cssContent)

	builder := NewTestRouterBuilder(t)
	router, err := builder.BuildRouter(h.RouterConfig(), daemon_domain.RouterDependencies{
		RegistryService:  h.RegistryService,
		VariantGenerator: h.VariantGenerator,
		CSPConfig:        h.CSPConfig,
		RateLimitService: h.RateLimitService,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/theme.css", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	AssertStatus(t, recorder, http.StatusOK)
	assert.Contains(t, recorder.Header().Get("Content-Type"), "text/css")
	assert.NotEmpty(t, recorder.Header().Get("ETag"))
	assert.Contains(t, recorder.Header().Get("Cache-Control"), "max-age=3600")
	assert.Equal(t, string(cssContent), recorder.Body.String())
}

func TestServeTheme_NotFound(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)

	builder := NewTestRouterBuilder(t)
	router, err := builder.BuildRouter(h.RouterConfig(), daemon_domain.RouterDependencies{
		RegistryService:  h.RegistryService,
		VariantGenerator: h.VariantGenerator,
		CSPConfig:        h.CSPConfig,
		RateLimitService: h.RateLimitService,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/theme.css", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	AssertStatus(t, recorder, http.StatusNotFound)
}

func TestServeTheme_ETagMatch_304(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)
	cssContent := []byte("body { color: red; }")
	etag := `"abc123"`

	variant := CreateTestVariant("source", "theme.css/source",
		WithVariantMimeType("text/css"),
		WithVariantTag("type", "source"),
		WithVariantTag("etag", etag),
	)
	artefact := CreateTestArtefact("theme.css", variant)
	h.RegistryService.AddArtefact(artefact)
	h.RegistryService.AddVariantData("theme.css/source", cssContent)

	builder := NewTestRouterBuilder(t)
	router, err := builder.BuildRouter(h.RouterConfig(), daemon_domain.RouterDependencies{
		RegistryService:  h.RegistryService,
		VariantGenerator: h.VariantGenerator,
		CSPConfig:        h.CSPConfig,
		RateLimitService: h.RateLimitService,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/theme.css", nil)
	request.Header.Set("If-None-Match", etag)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	AssertNotModified(t, recorder)
}

func TestServeTheme_BrotliCompression(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)
	cssContent := []byte("body { color: red; }")
	brotliContent := CompressBrotli(cssContent)

	sourceVariant := CreateTestVariant("source", "theme.css/source",
		WithVariantMimeType("text/css"),
		WithVariantTag("type", "source"),
		WithVariantTag("etag", `"source123"`),
	)
	brotliVariant := CreateTestVariant("minified-br", "theme.css/minified-br",
		WithVariantMimeType("text/css"),
		WithVariantTag("type", "minified"),
		WithVariantTag("encoding", "br"),
		WithVariantTag("etag", `"br123"`),
	)

	artefact := CreateTestArtefact("theme.css", sourceVariant, brotliVariant)
	h.RegistryService.AddArtefact(artefact)
	h.RegistryService.AddVariantData("theme.css/source", cssContent)
	h.RegistryService.AddVariantData("theme.css/minified-br", brotliContent)

	builder := NewTestRouterBuilder(t)
	router, err := builder.BuildRouter(h.RouterConfig(), daemon_domain.RouterDependencies{
		RegistryService:  h.RegistryService,
		VariantGenerator: h.VariantGenerator,
		CSPConfig:        h.CSPConfig,
		RateLimitService: h.RateLimitService,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/theme.css", nil)
	request.Header.Set("Accept-Encoding", "br, gzip")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	AssertStatus(t, recorder, http.StatusOK)

}

func TestServeTheme_GzipCompression(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)
	cssContent := []byte("body { color: red; }")
	gzipContent := CompressGzip(cssContent)

	sourceVariant := CreateTestVariant("source", "theme.css/source",
		WithVariantMimeType("text/css"),
		WithVariantTag("type", "source"),
		WithVariantTag("etag", `"source123"`),
	)
	gzipVariant := CreateTestVariant("minified-gzip", "theme.css/minified-gzip",
		WithVariantMimeType("text/css"),
		WithVariantTag("type", "minified"),
		WithVariantTag("encoding", "gzip"),
		WithVariantTag("etag", `"gzip123"`),
	)

	artefact := CreateTestArtefact("theme.css", sourceVariant, gzipVariant)
	h.RegistryService.AddArtefact(artefact)
	h.RegistryService.AddVariantData("theme.css/source", cssContent)
	h.RegistryService.AddVariantData("theme.css/minified-gzip", gzipContent)

	builder := NewTestRouterBuilder(t)
	router, err := builder.BuildRouter(h.RouterConfig(), daemon_domain.RouterDependencies{
		RegistryService:  h.RegistryService,
		VariantGenerator: h.VariantGenerator,
		CSPConfig:        h.CSPConfig,
		RateLimitService: h.RateLimitService,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/theme.css", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	AssertStatus(t, recorder, http.StatusOK)
}

func TestServeSitemap_Success(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)
	sitemapContent := []byte(`<?xml version="1.0" encoding="UTF-8"?><urlset></urlset>`)

	variant := CreateTestVariant("source", "sitemap.xml/source",
		WithVariantMimeType("application/xml"),
		WithVariantTag("type", "source"),
		WithVariantTag("etag", `"sitemap123"`),
	)
	artefact := CreateTestArtefact("sitemap.xml", variant)
	h.RegistryService.AddArtefact(artefact)
	h.RegistryService.AddVariantData("sitemap.xml/source", sitemapContent)

	builder := NewTestRouterBuilder(t)
	router, err := builder.BuildRouter(h.RouterConfig(), daemon_domain.RouterDependencies{
		RegistryService:  h.RegistryService,
		VariantGenerator: h.VariantGenerator,
		CSPConfig:        h.CSPConfig,
		RateLimitService: h.RateLimitService,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	AssertStatus(t, recorder, http.StatusOK)
	assert.Contains(t, recorder.Header().Get("Content-Type"), "xml")
	assert.Contains(t, recorder.Body.String(), "urlset")
}

func TestServeSitemap_NotFound(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)

	builder := NewTestRouterBuilder(t)
	router, err := builder.BuildRouter(h.RouterConfig(), daemon_domain.RouterDependencies{
		RegistryService:  h.RegistryService,
		VariantGenerator: h.VariantGenerator,
		CSPConfig:        h.CSPConfig,
		RateLimitService: h.RateLimitService,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	AssertStatus(t, recorder, http.StatusNotFound)
}

func TestServeSitemapChunk_Valid(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)
	chunk1Content := []byte(`<?xml version="1.0"?><urlset><url><loc>http://example.com/page1</loc></url></urlset>`)

	variant := CreateTestVariant("source", "sitemap-1.xml/source",
		WithVariantMimeType("application/xml"),
		WithVariantTag("type", "source"),
		WithVariantTag("etag", `"chunk1"`),
	)
	artefact := CreateTestArtefact("sitemap-1.xml", variant)
	h.RegistryService.AddArtefact(artefact)
	h.RegistryService.AddVariantData("sitemap-1.xml/source", chunk1Content)

	builder := NewTestRouterBuilder(t)
	router, err := builder.BuildRouter(h.RouterConfig(), daemon_domain.RouterDependencies{
		RegistryService:  h.RegistryService,
		VariantGenerator: h.VariantGenerator,
		CSPConfig:        h.CSPConfig,
		RateLimitService: h.RateLimitService,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/sitemap-1.xml", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	AssertStatus(t, recorder, http.StatusOK)
	assert.Contains(t, recorder.Body.String(), "page1")
}

func TestServeSitemapChunk_NotFound(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)

	builder := NewTestRouterBuilder(t)
	router, err := builder.BuildRouter(h.RouterConfig(), daemon_domain.RouterDependencies{
		RegistryService:  h.RegistryService,
		VariantGenerator: h.VariantGenerator,
		CSPConfig:        h.CSPConfig,
		RateLimitService: h.RateLimitService,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/sitemap-999.xml", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	AssertStatus(t, recorder, http.StatusNotFound)
}

func TestServeRobotsTxt_Success(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)
	robotsContent := []byte("User-agent: *\nAllow: /")

	variant := CreateTestVariant("source", "robots.txt/source",
		WithVariantMimeType("text/plain; charset=utf-8"),
		WithVariantTag("type", "source"),
		WithVariantTag("etag", `"robots123"`),
	)
	artefact := CreateTestArtefact("robots.txt", variant)
	h.RegistryService.AddArtefact(artefact)
	h.RegistryService.AddVariantData("robots.txt/source", robotsContent)

	builder := NewTestRouterBuilder(t)
	router, err := builder.BuildRouter(h.RouterConfig(), daemon_domain.RouterDependencies{
		RegistryService:  h.RegistryService,
		VariantGenerator: h.VariantGenerator,
		CSPConfig:        h.CSPConfig,
		RateLimitService: h.RateLimitService,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	AssertStatus(t, recorder, http.StatusOK)
	assert.Contains(t, recorder.Header().Get("Content-Type"), "text/plain")
	assert.Equal(t, string(robotsContent), recorder.Body.String())
}

func TestServeRobotsTxt_NotFound(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)

	builder := NewTestRouterBuilder(t)
	router, err := builder.BuildRouter(h.RouterConfig(), daemon_domain.RouterDependencies{
		RegistryService:  h.RegistryService,
		VariantGenerator: h.VariantGenerator,
		CSPConfig:        h.CSPConfig,
		RateLimitService: h.RateLimitService,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	AssertStatus(t, recorder, http.StatusNotFound)
}

func TestServeRobotsTxt_ETagMatch_304(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)
	robotsContent := []byte("User-agent: *\nAllow: /")
	etag := `"robots123"`

	variant := CreateTestVariant("source", "robots.txt/source",
		WithVariantMimeType("text/plain; charset=utf-8"),
		WithVariantTag("type", "source"),
		WithVariantTag("etag", etag),
	)
	artefact := CreateTestArtefact("robots.txt", variant)
	h.RegistryService.AddArtefact(artefact)
	h.RegistryService.AddVariantData("robots.txt/source", robotsContent)

	builder := NewTestRouterBuilder(t)
	router, err := builder.BuildRouter(h.RouterConfig(), daemon_domain.RouterDependencies{
		RegistryService:  h.RegistryService,
		VariantGenerator: h.VariantGenerator,
		CSPConfig:        h.CSPConfig,
		RateLimitService: h.RateLimitService,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	request.Header.Set("If-None-Match", etag)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	AssertNotModified(t, recorder)
}

func TestHeartbeat_Ping(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)

	builder := NewTestRouterBuilder(t)
	router, err := builder.BuildRouter(h.RouterConfig(), daemon_domain.RouterDependencies{
		RegistryService:  h.RegistryService,
		VariantGenerator: h.VariantGenerator,
		CSPConfig:        h.CSPConfig,
		RateLimitService: h.RateLimitService,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/ping", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	AssertStatus(t, recorder, http.StatusOK)
	assert.Equal(t, ".", recorder.Body.String())
}

func TestFindVariantByTag(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		tagKey        string
		tagValue      string
		expectedID    string
		variants      []registry_dto.Variant
		expectVariant bool
	}{
		{
			name: "finds matching variant",
			variants: []registry_dto.Variant{
				CreateTestVariant("source", "key/source", WithVariantTag("type", "source")),
				CreateTestVariant("minified", "key/minified", WithVariantTag("type", "minified")),
			},
			tagKey:        "type",
			tagValue:      "minified",
			expectVariant: true,
			expectedID:    "minified",
		},
		{
			name: "returns nil when no match",
			variants: []registry_dto.Variant{
				CreateTestVariant("source", "key/source", WithVariantTag("type", "source")),
			},
			tagKey:        "type",
			tagValue:      "compressed",
			expectVariant: false,
		},
		{
			name:          "returns nil for empty variants",
			variants:      []registry_dto.Variant{},
			tagKey:        "type",
			tagValue:      "source",
			expectVariant: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			artefact := &registry_dto.ArtefactMeta{
				ID:             "test-artefact",
				ActualVariants: tc.variants,
			}

			result := findVariantByTagHelper(artefact, tc.tagKey, tc.tagValue)

			if tc.expectVariant {
				require.NotNil(t, result, "expected to find variant")
				assert.Equal(t, tc.expectedID, result.VariantID)
			} else {
				assert.Nil(t, result, "expected no variant")
			}
		})
	}
}

func findVariantByTagHelper(artefact *registry_dto.ArtefactMeta, key, value string) *registry_dto.Variant {
	for i := range artefact.ActualVariants {
		if tagValue, ok := artefact.ActualVariants[i].MetadataTags.GetByName(key); ok && tagValue == value {
			return &artefact.ActualVariants[i]
		}
	}
	return nil
}

func findVariantByIDHelper(variants []registry_dto.Variant, id string) *registry_dto.Variant {
	for i := range variants {
		if variants[i].VariantID == id {
			return &variants[i]
		}
	}
	return nil
}

func TestFindVariantByID(t *testing.T) {
	t.Parallel()

	variants := []registry_dto.Variant{
		CreateTestVariant("source", "key/source"),
		CreateTestVariant("thumbnail", "key/thumbnail"),
	}

	t.Run("finds existing variant", func(t *testing.T) {
		result := findVariantByIDHelper(variants, "thumbnail")
		require.NotNil(t, result)
		assert.Equal(t, "thumbnail", result.VariantID)
	})

	t.Run("returns nil for non-existent variant", func(t *testing.T) {
		result := findVariantByIDHelper(variants, "nonexistent")
		assert.Nil(t, result)
	})
}

func TestServeTheme_RegistryError(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)

	h.RegistryService.GetArtefactFunc = func(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
		return nil, registry_domain.ErrArtefactNotFound
	}

	builder := NewTestRouterBuilder(t)
	router, err := builder.BuildRouter(h.RouterConfig(), daemon_domain.RouterDependencies{
		RegistryService:  h.RegistryService,
		VariantGenerator: h.VariantGenerator,
		CSPConfig:        h.CSPConfig,
		RateLimitService: h.RateLimitService,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/theme.css", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	AssertStatus(t, recorder, http.StatusNotFound)
}

func TestServeTheme_VariantDataError(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)

	variant := CreateTestVariant("source", "theme.css/source",
		WithVariantMimeType("text/css"),
		WithVariantTag("type", "source"),
		WithVariantTag("etag", `"abc123"`),
	)
	artefact := CreateTestArtefact("theme.css", variant)
	h.RegistryService.AddArtefact(artefact)

	h.RegistryService.GetVariantDataFunc = func(ctx context.Context, variant *registry_dto.Variant) (io.ReadCloser, error) {
		return nil, registry_domain.ErrVariantNotFound
	}

	builder := NewTestRouterBuilder(t)
	router, err := builder.BuildRouter(h.RouterConfig(), daemon_domain.RouterDependencies{
		RegistryService:  h.RegistryService,
		VariantGenerator: h.VariantGenerator,
		CSPConfig:        h.CSPConfig,
		RateLimitService: h.RateLimitService,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/theme.css", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	AssertStatus(t, recorder, http.StatusInternalServerError)
}

func TestServeArtefact_Success(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)

	h.ServerConfig.Paths.ArtefactServePath = new("/_piko/assets")

	imageContent := []byte("fake-image-bytes")
	variant := CreateTestVariant("source", "image.png/source",
		WithVariantMimeType("image/png"),
		WithVariantTag("type", "source"),
		WithVariantTag("etag", `"img123"`),
	)
	artefact := CreateTestArtefact("image.png", variant)
	h.RegistryService.AddArtefact(artefact)
	h.RegistryService.AddVariantData("image.png/source", imageContent)

	builder := NewTestRouterBuilder(t)
	router, err := builder.BuildRouter(h.RouterConfig(), daemon_domain.RouterDependencies{
		RegistryService:  h.RegistryService,
		VariantGenerator: h.VariantGenerator,
		CSPConfig:        h.CSPConfig,
		RateLimitService: h.RateLimitService,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/_piko/assets/image.png", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	AssertStatus(t, recorder, http.StatusOK)
}

func TestServeArtefact_NotFound(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)
	h.ServerConfig.Paths.ArtefactServePath = new("/_piko/assets")

	builder := NewTestRouterBuilder(t)
	router, err := builder.BuildRouter(h.RouterConfig(), daemon_domain.RouterDependencies{
		RegistryService:  h.RegistryService,
		VariantGenerator: h.VariantGenerator,
		CSPConfig:        h.CSPConfig,
		RateLimitService: h.RateLimitService,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/_piko/assets/nonexistent.png", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	AssertStatus(t, recorder, http.StatusNotFound)
}

func TestServeArtefact_WithVariantParam(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)
	h.ServerConfig.Paths.ArtefactServePath = new("/_piko/assets")

	sourceContent := []byte("original-image")
	thumbnailContent := []byte("thumbnail-image")

	sourceVariant := CreateTestVariant("source", "image.png/source",
		WithVariantMimeType("image/png"),
		WithVariantTag("type", "source"),
		WithVariantTag("etag", `"source"`),
	)
	thumbnailVariant := CreateTestVariant("thumbnail", "image.png/thumbnail",
		WithVariantMimeType("image/png"),
		WithVariantTag("type", "thumbnail"),
		WithVariantTag("etag", `"thumb"`),
	)

	artefact := CreateTestArtefact("image.png", sourceVariant, thumbnailVariant)
	h.RegistryService.AddArtefact(artefact)
	h.RegistryService.AddVariantData("image.png/source", sourceContent)
	h.RegistryService.AddVariantData("image.png/thumbnail", thumbnailContent)

	builder := NewTestRouterBuilder(t)
	router, err := builder.BuildRouter(h.RouterConfig(), daemon_domain.RouterDependencies{
		RegistryService:  h.RegistryService,
		VariantGenerator: h.VariantGenerator,
		CSPConfig:        h.CSPConfig,
		RateLimitService: h.RateLimitService,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/_piko/assets/image.png?v=thumbnail", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	AssertStatus(t, recorder, http.StatusOK)
}

func TestBuildRouter_CORS(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)
	h.ServerConfig.Network.PublicDomain = new("example.com")

	userRouter := chi.NewRouter()
	userRouter.Get("/test-cors", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	builder := NewTestRouterBuilder(t)
	router, err := builder.BuildRouter(h.RouterConfig(), daemon_domain.RouterDependencies{
		RegistryService:  h.RegistryService,
		UserRouter:       userRouter,
		VariantGenerator: h.VariantGenerator,
		CSPConfig:        h.CSPConfig,
		RateLimitService: h.RateLimitService,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodOptions, "/test-cors", nil)
	request.Header.Set("Origin", "https://example.com")
	request.Header.Set("Access-Control-Request-Method", "GET")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.NotEmpty(t, recorder.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, recorder.Header().Get("Access-Control-Allow-Methods"), "GET")
}

func TestBuildRouter_RequestIDMiddleware(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)

	userRouter := chi.NewRouter()
	userRouter.Get("/test", func(w http.ResponseWriter, r *http.Request) {

		reqID := security_dto.RequestIDFromContext(r.Context())
		_, _ = w.Write([]byte(reqID))
	})

	builder := NewTestRouterBuilder(t)
	router, err := builder.BuildRouter(h.RouterConfig(), daemon_domain.RouterDependencies{
		RegistryService:  h.RegistryService,
		UserRouter:       userRouter,
		VariantGenerator: h.VariantGenerator,
		CSPConfig:        h.CSPConfig,
		RateLimitService: h.RateLimitService,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.NotEmpty(t, recorder.Body.String())
}
