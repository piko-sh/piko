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

package daemon_adapters

import (
	"bytes"
	"context"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/daemon/daemon_frontend"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/security/security_dto"
)

func TestBuildAllowedOrigins_EmptyPublicDomain_ReturnsNil(t *testing.T) {
	t.Parallel()

	config := &daemon_domain.RouterConfig{}

	result := buildAllowedOrigins(config)

	assert.Nil(t, result)
}

func TestBuildAllowedOrigins_WithPublicDomain_ForceHTTPS(t *testing.T) {
	t.Parallel()

	config := &daemon_domain.RouterConfig{
		PublicDomain: "example.com",
		ForceHTTPS:   true,
	}

	result := buildAllowedOrigins(config)

	require.Len(t, result, 1)
	assert.Equal(t, "https://example.com", result[0])
}

func TestBuildAllowedOrigins_WithPublicDomain_NoForceHTTPS(t *testing.T) {
	t.Parallel()

	config := &daemon_domain.RouterConfig{
		PublicDomain: "example.com",
		ForceHTTPS:   false,
	}

	result := buildAllowedOrigins(config)

	require.Len(t, result, 2)
	assert.Equal(t, "https://example.com", result[0])
	assert.Equal(t, "http://example.com", result[1])
}

func TestBuildAllowedOrigins_SubdomainDomain(t *testing.T) {
	t.Parallel()

	config := &daemon_domain.RouterConfig{
		PublicDomain: "app.example.com",
		ForceHTTPS:   true,
	}

	result := buildAllowedOrigins(config)

	require.Len(t, result, 1)
	assert.Equal(t, "https://app.example.com", result[0])
}

func TestFindChunkByID_Found(t *testing.T) {
	t.Parallel()

	variant := &registry_dto.Variant{
		Chunks: []registry_dto.VariantChunk{
			{ChunkID: "chunk-001", MimeType: "video/mp2t", SizeBytes: 1024},
			{ChunkID: "chunk-002", MimeType: "video/mp2t", SizeBytes: 2048},
			{ChunkID: "chunk-003", MimeType: "video/mp2t", SizeBytes: 512},
		},
	}

	result := findChunkByID(variant, "chunk-002")
	require.NotNil(t, result)
	assert.Equal(t, "chunk-002", result.ChunkID)
	assert.Equal(t, int64(2048), result.SizeBytes)
}

func TestFindChunkByID_NotFound(t *testing.T) {
	t.Parallel()

	variant := &registry_dto.Variant{
		Chunks: []registry_dto.VariantChunk{
			{ChunkID: "chunk-001"},
		},
	}

	result := findChunkByID(variant, "nonexistent")
	assert.Nil(t, result)
}

func TestFindChunkByID_EmptyChunks(t *testing.T) {
	t.Parallel()

	variant := &registry_dto.Variant{
		Chunks: []registry_dto.VariantChunk{},
	}

	result := findChunkByID(variant, "chunk-001")
	assert.Nil(t, result)
}

func TestFindChunkByID_FirstChunk(t *testing.T) {
	t.Parallel()

	variant := &registry_dto.Variant{
		Chunks: []registry_dto.VariantChunk{
			{ChunkID: "first-chunk"},
			{ChunkID: "second-chunk"},
		},
	}

	result := findChunkByID(variant, "first-chunk")
	require.NotNil(t, result)
	assert.Equal(t, "first-chunk", result.ChunkID)
}

func TestFindChunkByID_LastChunk(t *testing.T) {
	t.Parallel()

	variant := &registry_dto.Variant{
		Chunks: []registry_dto.VariantChunk{
			{ChunkID: "first-chunk"},
			{ChunkID: "last-chunk"},
		},
	}

	result := findChunkByID(variant, "last-chunk")
	require.NotNil(t, result)
	assert.Equal(t, "last-chunk", result.ChunkID)
}

func TestBuildMasterPlaylist_NoQualityProfiles(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		DesiredProfiles: []registry_dto.NamedProfile{
			{Name: "thumbnail"},
			{Name: "non_hls_profile"},
		},
	}

	result := buildMasterPlaylist(artefact)
	assert.Equal(t, "#EXTM3U\n", result)
}

func TestBuildMasterPlaylist_SingleQuality(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		DesiredProfiles: []registry_dto.NamedProfile{
			{Name: "hls_720p"},
		},
	}

	result := buildMasterPlaylist(artefact)

	assert.Contains(t, result, "#EXTM3U\n")
	assert.Contains(t, result, "#EXT-X-STREAM-INF:BANDWIDTH=2500000,RESOLUTION=1280x720")
	assert.Contains(t, result, "720p/manifest.m3u8")
}

func TestBuildMasterPlaylist_MultipleQualities(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		DesiredProfiles: []registry_dto.NamedProfile{
			{Name: "hls_1080p"},
			{Name: "hls_720p"},
			{Name: "hls_480p"},
		},
	}

	result := buildMasterPlaylist(artefact)

	assert.Contains(t, result, "#EXTM3U\n")
	assert.Contains(t, result, "BANDWIDTH=5000000,RESOLUTION=1920x1080")
	assert.Contains(t, result, "1080p/manifest.m3u8")
	assert.Contains(t, result, "BANDWIDTH=2500000,RESOLUTION=1280x720")
	assert.Contains(t, result, "720p/manifest.m3u8")
	assert.Contains(t, result, "BANDWIDTH=1000000,RESOLUTION=854x480")
	assert.Contains(t, result, "480p/manifest.m3u8")
}

func TestBuildMasterPlaylist_UnknownQuality_Skipped(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		DesiredProfiles: []registry_dto.NamedProfile{
			{Name: "hls_unknownp"},
			{Name: "hls_720p"},
		},
	}

	result := buildMasterPlaylist(artefact)

	assert.NotContains(t, result, "unknownp")
	assert.Contains(t, result, "720p/manifest.m3u8")
}

func TestBuildMasterPlaylist_EmptyProfiles(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{}

	result := buildMasterPlaylist(artefact)
	assert.Equal(t, "#EXTM3U\n", result)
}

func TestBuildMasterPlaylist_AllSupportedQualities(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		DesiredProfiles: []registry_dto.NamedProfile{
			{Name: "hls_2160p"},
			{Name: "hls_1440p"},
			{Name: "hls_1080p"},
			{Name: "hls_720p"},
			{Name: "hls_480p"},
			{Name: "hls_360p"},
		},
	}

	result := buildMasterPlaylist(artefact)

	assert.Contains(t, result, "BANDWIDTH=15000000,RESOLUTION=3840x2160")
	assert.Contains(t, result, "BANDWIDTH=8000000,RESOLUTION=2560x1440")
	assert.Contains(t, result, "BANDWIDTH=5000000,RESOLUTION=1920x1080")
	assert.Contains(t, result, "BANDWIDTH=2500000,RESOLUTION=1280x720")
	assert.Contains(t, result, "BANDWIDTH=1000000,RESOLUTION=854x480")
	assert.Contains(t, result, "BANDWIDTH=500000,RESOLUTION=640x360")
}

func TestBuildVariantPlaylist_NoChunks(t *testing.T) {
	t.Parallel()

	variant := &registry_dto.Variant{
		Chunks: []registry_dto.VariantChunk{},
	}

	result := buildVariantPlaylist(variant)

	assert.Contains(t, result, "#EXTM3U\n")
	assert.Contains(t, result, "#EXT-X-VERSION:3\n")
	assert.Contains(t, result, "#EXT-X-ENDLIST\n")
	assert.Contains(t, result, "#EXT-X-TARGETDURATION:10")
}

func TestBuildVariantPlaylist_SingleVideoChunk(t *testing.T) {
	t.Parallel()

	variant := &registry_dto.Variant{
		Chunks: []registry_dto.VariantChunk{
			{ChunkID: "seg-001.ts", MimeType: "video/mp2t", DurationSeconds: new(6.5)},
		},
	}

	result := buildVariantPlaylist(variant)

	assert.Contains(t, result, "#EXTM3U\n")
	assert.Contains(t, result, "#EXT-X-VERSION:3\n")
	assert.Contains(t, result, "#EXT-X-TARGETDURATION:7\n")
	assert.Contains(t, result, "#EXTINF:6.500000,\n")
	assert.Contains(t, result, "seg-001.ts\n")
	assert.Contains(t, result, "#EXT-X-ENDLIST\n")
}

func TestBuildVariantPlaylist_MultipleChunks(t *testing.T) {
	t.Parallel()

	variant := &registry_dto.Variant{
		Chunks: []registry_dto.VariantChunk{
			{ChunkID: "seg-001.ts", MimeType: "video/mp2t", DurationSeconds: new(10.0)},
			{ChunkID: "seg-002.ts", MimeType: "video/mp2t", DurationSeconds: new(8.5)},
			{ChunkID: "seg-003.ts", MimeType: "video/mp2t", DurationSeconds: new(5.0)},
		},
	}

	result := buildVariantPlaylist(variant)

	assert.Contains(t, result, "#EXT-X-TARGETDURATION:10\n")
	assert.Contains(t, result, "#EXTINF:10.000000,\nseg-001.ts")
	assert.Contains(t, result, "#EXTINF:8.500000,\nseg-002.ts")
	assert.Contains(t, result, "#EXTINF:5.000000,\nseg-003.ts")
}

func TestBuildVariantPlaylist_NilDuration_UsesDefault(t *testing.T) {
	t.Parallel()

	variant := &registry_dto.Variant{
		Chunks: []registry_dto.VariantChunk{
			{ChunkID: "seg-001.ts", MimeType: "video/mp2t", DurationSeconds: nil},
		},
	}

	result := buildVariantPlaylist(variant)

	assert.Contains(t, result, "#EXTINF:10.000000,\nseg-001.ts")
}

func TestBuildVariantPlaylist_SkipsNonVideoChunks(t *testing.T) {
	t.Parallel()

	variant := &registry_dto.Variant{
		Chunks: []registry_dto.VariantChunk{
			{ChunkID: "thumbnail.jpg", MimeType: "image/jpeg", DurationSeconds: nil},
			{ChunkID: "seg-001.ts", MimeType: "video/mp2t", DurationSeconds: new(5.0)},
		},
	}

	result := buildVariantPlaylist(variant)

	assert.NotContains(t, result, "thumbnail.jpg")
	assert.Contains(t, result, "seg-001.ts")
}

func TestBuildVariantPlaylist_AcceptsMP2TUppercase(t *testing.T) {
	t.Parallel()

	variant := &registry_dto.Variant{
		Chunks: []registry_dto.VariantChunk{
			{ChunkID: "seg-001.ts", MimeType: "video/MP2T", DurationSeconds: new(7.0)},
		},
	}

	result := buildVariantPlaylist(variant)
	assert.Contains(t, result, "seg-001.ts")
}

func TestStreamChunkToResponse_WritesHeaders(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	chunk := &registry_dto.VariantChunk{
		MimeType:  "video/mp2t",
		SizeBytes: 1234,
	}
	data := io.NopCloser(strings.NewReader("chunk-data"))

	streamChunkToResponse(context.Background(), recorder, data, chunk)

	assert.Equal(t, "video/mp2t", recorder.Header().Get("Content-Type"))
	assert.Equal(t, "public, max-age=31536000, immutable", recorder.Header().Get("Cache-Control"))
	assert.Equal(t, "1234", recorder.Header().Get("Content-Length"))
	assert.Equal(t, "chunk-data", recorder.Body.String())
}

func TestStreamChunkToResponse_ClosesReader(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	chunk := &registry_dto.VariantChunk{
		MimeType: "video/mp2t",
	}
	closed := false
	data := &testReadCloser{
		Reader: strings.NewReader("test"),
		closeFunc: func() error {
			closed = true
			return nil
		},
	}
	streamChunkToResponse(context.Background(), recorder, data, chunk)

	assert.True(t, closed, "Reader should be closed after streaming")
}

type testReadCloser struct {
	io.Reader
	closeFunc func() error
}

func (r *testReadCloser) Close() error {
	if r.closeFunc != nil {
		return r.closeFunc()
	}
	return nil
}

func TestLookupArtefactByStorageKey_Found(t *testing.T) {
	t.Parallel()

	expectedArtefact := &registry_dto.ArtefactMeta{ID: "found-artefact"}
	registryService := &registry_domain.MockRegistryService{
		FindArtefactByVariantStorageKeyFunc: func(_ context.Context, key string) (*registry_dto.ArtefactMeta, error) {
			if key == "my-storage-key" {
				return expectedArtefact, nil
			}
			return nil, registry_domain.ErrArtefactNotFound
		},
	}
	span := newNoopSpan()

	result := lookupArtefactByStorageKey(context.Background(), span, registryService, "my-storage-key")

	assert.Nil(t, result.err)
	assert.Equal(t, 0, result.httpStatus)
	assert.True(t, result.foundByStorageKey)
	assert.Equal(t, "found-artefact", result.artefact.ID)
}

func TestLookupArtefactByStorageKey_NotFound(t *testing.T) {
	t.Parallel()

	registryService := &registry_domain.MockRegistryService{
		FindArtefactByVariantStorageKeyFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, registry_domain.ErrArtefactNotFound
		},
	}
	span := newNoopSpan()

	result := lookupArtefactByStorageKey(context.Background(), span, registryService, "missing-key")

	assert.NotNil(t, result.err)
	assert.Equal(t, http.StatusNotFound, result.httpStatus)
	assert.False(t, result.foundByStorageKey)
	assert.Nil(t, result.artefact)
}

func TestLookupArtefactByStorageKey_OtherError(t *testing.T) {
	t.Parallel()

	registryService := &registry_domain.MockRegistryService{
		FindArtefactByVariantStorageKeyFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, errors.New("database error")
		},
	}
	span := newNoopSpan()

	result := lookupArtefactByStorageKey(context.Background(), span, registryService, "key")

	assert.NotNil(t, result.err)
	assert.Equal(t, http.StatusInternalServerError, result.httpStatus)
	assert.False(t, result.foundByStorageKey)
	assert.Nil(t, result.artefact)
}

func TestNewHTTPRouterBuilder_ReturnsNonNil(t *testing.T) {
	t.Parallel()

	builder := NewHTTPRouterBuilder(nil, "")
	require.NotNil(t, builder)
}

func TestNewHTTPRouterBuilder_ImplementsInterface(t *testing.T) {
	t.Parallel()

	builder := NewHTTPRouterBuilder(nil, "")
	_, ok := builder.(*HTTPRouterBuilder)
	assert.True(t, ok)
}

func TestHTTPRouterBuilder_Close_NilCache_DoesNotPanic(t *testing.T) {
	t.Parallel()

	builder := &HTTPRouterBuilder{}
	assert.NotPanics(t, func() {
		builder.Close()
	})
}

func TestSelectStaticVariant_PreferMinified(t *testing.T) {
	t.Parallel()

	var minifiedTags registry_dto.Tags
	minifiedTags.SetByName("type", "minified")

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source"},
			{VariantID: "minified-variant", MetadataTags: minifiedTags},
		},
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	config := staticArtefactConfig{
		preferredType:  "minified",
		useCompression: false,
	}

	result := selectStaticVariant(request, artefact, config)
	require.NotNil(t, result)
	assert.Equal(t, "minified-variant", result.VariantID)
}

func TestSelectStaticVariant_FallsBackToSourceExtra(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source"},
		},
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	config := staticArtefactConfig{}

	result := selectStaticVariant(request, artefact, config)
	require.NotNil(t, result)
	assert.Equal(t, "source", result.VariantID)
}

func TestSelectStaticVariant_CompressedVariantPreferred(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source"},
			{VariantID: "source_br", MetadataTags: func() registry_dto.Tags {
				var tags registry_dto.Tags
				tags.SetByName("contentEncoding", "br")
				return tags
			}()},
		},
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "br, gzip")
	config := staticArtefactConfig{
		preferredType:  "",
		useCompression: true,
	}

	result := selectStaticVariant(request, artefact, config)
	require.NotNil(t, result)
	assert.Equal(t, "source_br", result.VariantID)
}

func TestStaticArtefactConfig_Fields(t *testing.T) {
	t.Parallel()

	config := staticArtefactConfig{
		artefactID:      "test.css",
		defaultMimeType: "text/css",
		cacheMaxAge:     "public, no-cache",
		preferredType:   "minified",
		useCompression:  true,
	}

	assert.Equal(t, "test.css", config.artefactID)
	assert.Equal(t, "text/css", config.defaultMimeType)
	assert.Equal(t, "public, no-cache", config.cacheMaxAge)
	assert.Equal(t, "minified", config.preferredType)
	assert.True(t, config.useCompression)
}

func TestArtefactLookupResult_SuccessFields(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{ID: "test"}
	result := artefactLookupResult{
		err:               nil,
		artefact:          artefact,
		httpStatus:        0,
		foundByStorageKey: true,
	}

	assert.Nil(t, result.err)
	assert.Equal(t, "test", result.artefact.ID)
	assert.Equal(t, 0, result.httpStatus)
	assert.True(t, result.foundByStorageKey)
}

func TestArtefactLookupResult_ErrorFields(t *testing.T) {
	t.Parallel()

	result := artefactLookupResult{
		err:               errors.New("lookup failed"),
		artefact:          nil,
		httpStatus:        http.StatusNotFound,
		foundByStorageKey: false,
	}

	assert.NotNil(t, result.err)
	assert.Nil(t, result.artefact)
	assert.Equal(t, http.StatusNotFound, result.httpStatus)
	assert.False(t, result.foundByStorageKey)
}

func mountProdAssetHandler(t *testing.T, registry registry_domain.RegistryService) http.Handler {
	t.Helper()
	builder := &HTTPRouterBuilder{}
	t.Cleanup(builder.Close)
	router := chi.NewRouter()
	router.Get("/_piko/assets/*", builder.serveArtefact(registry, daemon_domain.OnDemandVariantGenerator(nil), false))
	return router
}

func staticByIDRegistry(artefactID, mimeType, etag, body string) *registry_domain.MockRegistryService {
	tags := registry_dto.Tags{}
	tags.Set(registry_dto.TagEtag, etag)
	artefact := &registry_dto.ArtefactMeta{
		ID:     artefactID,
		Status: registry_dto.VariantStatusReady,
		ActualVariants: []registry_dto.Variant{
			{
				VariantID:    variantSource,
				MimeType:     mimeType,
				StorageKey:   "store/" + artefactID,
				SizeBytes:    int64(len(body)),
				Status:       registry_dto.VariantStatusReady,
				MetadataTags: tags,
			},
		},
	}
	return &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, id string) (*registry_dto.ArtefactMeta, error) {
			if id != artefactID {
				return nil, registry_domain.ErrArtefactNotFound
			}
			return artefact, nil
		},
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(body)), nil
		},
	}
}

func contentAddressedRegistry(storageKey, mimeType, body string) *registry_domain.MockRegistryService {
	tags := registry_dto.Tags{}
	tags.Set(registry_dto.TagEtag, `"content-addressed"`)
	artefact := &registry_dto.ArtefactMeta{
		ID:     "logical-id",
		Status: registry_dto.VariantStatusReady,
		ActualVariants: []registry_dto.Variant{
			{
				VariantID:    variantSource,
				MimeType:     mimeType,
				StorageKey:   storageKey,
				SizeBytes:    int64(len(body)),
				Status:       registry_dto.VariantStatusReady,
				MetadataTags: tags,
			},
		},
	}
	return &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, registry_domain.ErrArtefactNotFound
		},
		FindArtefactByVariantStorageKeyFunc: func(_ context.Context, key string) (*registry_dto.ArtefactMeta, error) {
			if key != storageKey {
				return nil, registry_domain.ErrArtefactNotFound
			}
			return artefact, nil
		},
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(body)), nil
		},
	}
}

func TestServeArtefact_ContentAddressedURLIsImmutableInProduction(t *testing.T) {
	t.Parallel()

	const storageKey = "generated/app_a1b2c3d4e5f60718.js"
	const body = "export const value = 1;"

	registry := contentAddressedRegistry(storageKey, "application/javascript", body)

	builder := &HTTPRouterBuilder{}
	t.Cleanup(builder.Close)
	router := chi.NewRouter()
	router.Get("/_piko/assets/*", builder.serveArtefact(registry, daemon_domain.OnDemandVariantGenerator(nil), false))

	request := httptest.NewRequest(http.MethodGet, "/_piko/assets/"+storageKey, nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, cacheControlImmutable, recorder.Header().Get(headerCacheControl),
		"a URL that names its own bytes can never need revalidating")
	assert.Equal(t, headerAcceptEncoding, recorder.Header().Get("Vary"),
		"a compressed sibling may be served for the same URL, so Vary is required")

	etag := recorder.Header().Get(headerETag)
	require.NotEmpty(t, etag)

	conditional := httptest.NewRequest(http.MethodGet, "/_piko/assets/"+storageKey, nil)
	conditional.Header.Set(headerIfNoneMatch, etag)
	conditionalRecorder := httptest.NewRecorder()
	router.ServeHTTP(conditionalRecorder, conditional)

	require.Equal(t, http.StatusNotModified, conditionalRecorder.Code)
	assert.Equal(t, cacheControlImmutable, conditionalRecorder.Header().Get(headerCacheControl),
		"the 304 must carry the same policy as the 200")
}

func TestServeArtefact_ContentAddressedURLRevalidatesInDevelopment(t *testing.T) {
	t.Parallel()

	const storageKey = "generated/app_a1b2c3d4e5f60718.js"
	registry := contentAddressedRegistry(storageKey, "application/javascript", "export const value = 1;")

	builder := &HTTPRouterBuilder{}
	t.Cleanup(builder.Close)
	router := chi.NewRouter()
	router.Get("/_piko/assets/*", builder.serveArtefact(registry, daemon_domain.OnDemandVariantGenerator(nil), true))

	request := httptest.NewRequest(http.MethodGet, "/_piko/assets/"+storageKey, nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, cacheControlNoCache, recorder.Header().Get(headerCacheControl),
		"dev mode must not pin an asset for a year")
}

func TestServeArtefact_CacheControlAndVaryHeaders(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		artefactID       string
		mimeType         string
		wantCacheControl string
	}{
		{
			name:             "stable-URL pk-js bundle revalidates",
			artefactID:       "pk-js/pk/actions.gen.js",
			mimeType:         "application/javascript",
			wantCacheControl: jitCacheControl,
		},
		{
			name:             "stable-URL @/-aliased module JS revalidates",
			artefactID:       "module/lib/utils.js",
			mimeType:         "application/javascript",
			wantCacheControl: jitCacheControl,
		},
		{
			name:             "image served by ID uses bounded media tier",
			artefactID:       "module/lib/logo.png",
			mimeType:         "image/png",
			wantCacheControl: cacheControlMutableAsset,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			registry := staticByIDRegistry(tc.artefactID, tc.mimeType, `"etag-1"`, "body-"+tc.artefactID)
			handler := mountProdAssetHandler(t, registry)

			request := httptest.NewRequest(http.MethodGet, "/_piko/assets/"+tc.artefactID, nil)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, tc.wantCacheControl, recorder.Header().Get("Cache-Control"))
			assert.Equal(t, "Accept-Encoding", recorder.Header().Get("Vary"))
			assert.Equal(t, `"etag-1"`, recorder.Header().Get("Etag"))
		})
	}
}

func TestServeArtefact_NotModifiedCarriesCacheHeaders(t *testing.T) {
	t.Parallel()

	const artefactID = "pk-js/pk/actions.gen.js"
	registry := staticByIDRegistry(artefactID, "application/javascript", `"etag-1"`, "actions-body")
	handler := mountProdAssetHandler(t, registry)

	request := httptest.NewRequest(http.MethodGet, "/_piko/assets/"+artefactID, nil)
	request.Header.Set("If-None-Match", `"etag-1"`)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusNotModified, recorder.Code)
	assert.Empty(t, recorder.Body.String(), "304 must not carry a body")
	assert.Equal(t, jitCacheControl, recorder.Header().Get("Cache-Control"),
		"304 must refresh the client's stored Cache-Control so it never goes stale")
	assert.Equal(t, `"etag-1"`, recorder.Header().Get("Etag"))
	assert.Equal(t, "Accept-Encoding", recorder.Header().Get("Vary"))
}

func TestHlsQualityConfigs_AllQualities(t *testing.T) {
	t.Parallel()

	expected := map[string]videoQualityInfo{
		"2160p": {resolution: "3840x2160", bandwidth: 15000000},
		"1440p": {resolution: "2560x1440", bandwidth: 8000000},
		"1080p": {resolution: "1920x1080", bandwidth: 5000000},
		"720p":  {resolution: "1280x720", bandwidth: 2500000},
		"480p":  {resolution: "854x480", bandwidth: 1000000},
		"360p":  {resolution: "640x360", bandwidth: 500000},
	}

	assert.Equal(t, len(expected), len(hlsQualityConfigs))
	for quality, info := range expected {
		got, ok := hlsQualityConfigs[quality]
		assert.True(t, ok, "Quality %s should exist", quality)
		assert.Equal(t, info.resolution, got.resolution, "Resolution mismatch for %s", quality)
		assert.Equal(t, info.bandwidth, got.bandwidth, "Bandwidth mismatch for %s", quality)
	}
}

func TestServeJITResult_WritesContent(t *testing.T) {
	t.Parallel()

	m := &CacheMiddleware{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	result := &jitResult{
		Content:    []byte("<html>test</html>"),
		Encoding:   "br",
		ETag:       `"jit-abc123"`,
		StatusCode: http.StatusOK,
	}

	m.serveJITResult(recorder, request, result, 600)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "text/html; charset=utf-8", recorder.Header().Get("Content-Type"))
	assert.Equal(t, "br", recorder.Header().Get("Content-Encoding"))
	assert.Equal(t, `"jit-abc123"`, recorder.Header().Get("ETag"))
	assert.Equal(t, "public, max-age=0, s-maxage=600, must-revalidate", recorder.Header().Get("Cache-Control"))
	assert.Equal(t, "<html>test</html>", recorder.Body.String())
}

func TestServeJITResult_ETagMatch_Returns304(t *testing.T) {
	t.Parallel()

	m := &CacheMiddleware{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("If-None-Match", `"jit-abc123"`)

	result := &jitResult{
		Content:    []byte("<html>test</html>"),
		Encoding:   "br",
		ETag:       `"jit-abc123"`,
		StatusCode: http.StatusOK,
	}

	m.serveJITResult(recorder, request, result, 600)

	assert.Equal(t, http.StatusNotModified, recorder.Code)
	assert.Empty(t, recorder.Body.String())
}

func TestServeJITResult_NoETagMatch_Returns200(t *testing.T) {
	t.Parallel()

	m := &CacheMiddleware{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("If-None-Match", `"different-etag"`)

	result := &jitResult{
		Content:    []byte("content"),
		ETag:       `"jit-abc123"`,
		StatusCode: http.StatusOK,
	}

	m.serveJITResult(recorder, request, result, 600)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "content", recorder.Body.String())
}

func TestServeJITResult_NoIfNoneMatch_Returns200(t *testing.T) {
	t.Parallel()

	m := &CacheMiddleware{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	result := &jitResult{
		Content:    []byte("fresh content"),
		ETag:       `"jit-abc123"`,
		StatusCode: http.StatusOK,
	}

	m.serveJITResult(recorder, request, result, 600)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "fresh content", recorder.Body.String())
}

func TestVariantETag(t *testing.T) {
	t.Parallel()

	var stored registry_dto.Tags
	stored.Set(registry_dto.TagEtag, `"stored"`)

	testCases := []struct {
		name    string
		variant *registry_dto.Variant
		want    string
	}{
		{
			name:    "prefers stored etag tag",
			variant: &registry_dto.Variant{MetadataTags: stored, ContentHash: "deadbeef"},
			want:    `"stored"`,
		},
		{
			name:    "falls back to content hash",
			variant: &registry_dto.Variant{ContentHash: "deadbeef"},
			want:    `"deadbeef"`,
		},
		{
			name:    "empty when neither present",
			variant: &registry_dto.Variant{},
			want:    "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, variantETag(tc.variant))
		})
	}
}

func TestServeFromCache_EmptyETagNoConditional_Returns200(t *testing.T) {
	t.Parallel()

	registryService := &registry_domain.MockRegistryService{
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("CACHED BODY")), nil
		},
	}
	m := &CacheMiddleware{registryService: registryService}

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: variantSource, MimeType: "text/html", ContentHash: "abc123"},
		},
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	m.serveFromCache(recorder, request, artefact, 0)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "CACHED BODY", recorder.Body.String())
	assert.Equal(t, `"abc123"`, recorder.Header().Get("Etag"))
}

func TestServeFromCache_ContentHashConditional_Returns304(t *testing.T) {
	t.Parallel()

	registryService := &registry_domain.MockRegistryService{
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("CACHED BODY")), nil
		},
	}
	m := &CacheMiddleware{registryService: registryService}

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: variantSource, MimeType: "text/html", ContentHash: "abc123"},
		},
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("If-None-Match", `"abc123"`)

	m.serveFromCache(recorder, request, artefact, 0)

	assert.Equal(t, http.StatusNotModified, recorder.Code)
	assert.Empty(t, recorder.Body.String())
}

func TestHandleNonCacheableStream_NoCompression_PassesThrough(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("plain text"))
	})

	handleNonCacheableStream(recorder, request, next)

	assert.True(t, nextCalled)
	assert.Equal(t, "plain text", recorder.Body.String())
}

func TestHandleNonCacheableStream_BrotliCompression(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "br")

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("compressed content"))
	})

	handleNonCacheableStream(recorder, request, next)

	assert.Equal(t, "br", recorder.Header().Get("Content-Encoding"))

	assert.Greater(t, recorder.Body.Len(), 0)
}

func TestHandleNonCacheableStream_GzipCompression(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "gzip")

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("compressed content"))
	})

	handleNonCacheableStream(recorder, request, next)

	assert.Equal(t, "gzip", recorder.Header().Get("Content-Encoding"))
	assert.Greater(t, recorder.Body.Len(), 0)
}

func TestGetCompressedResponseWriter_SetsFields(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	compressor := nopWriteCloser{&bytes.Buffer{}}

	crw := getCompressedResponseWriter(recorder, compressor)
	defer releaseCompressedResponseWriter(crw)

	assert.NotNil(t, crw)
	assert.NotNil(t, crw.ResponseWriter)
	assert.NotNil(t, crw.compressor)
}

func TestReleaseCompressedResponseWriter_ClearsFields(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	compressor := nopWriteCloser{&bytes.Buffer{}}

	crw := getCompressedResponseWriter(recorder, compressor)

	assert.NotNil(t, crw.ResponseWriter)
	assert.NotNil(t, crw.compressor)

	releaseCompressedResponseWriter(crw)
}

func TestGetPipeResponseWriter_SetsDefaults(t *testing.T) {
	t.Parallel()

	_, pw := io.Pipe()
	defer func() { _ = pw.Close() }()

	prw := getPipeResponseWriter(pw)
	defer releasePipeResponseWriter(prw)

	assert.NotNil(t, prw)
	assert.Equal(t, http.StatusOK, prw.statusCode)
	assert.NotNil(t, prw.header)
}

func TestReleasePipeResponseWriter_ClearsWriter(t *testing.T) {
	t.Parallel()

	_, pw := io.Pipe()
	defer func() { _ = pw.Close() }()

	prw := getPipeResponseWriter(pw)

	assert.NotNil(t, prw.Writer)

	releasePipeResponseWriter(prw)
}

func TestNewPipeResponseWriter_SetStatusOK(t *testing.T) {
	t.Parallel()

	_, pw := io.Pipe()
	defer func() { _ = pw.Close() }()

	prw := newPipeResponseWriter(pw)

	assert.Equal(t, http.StatusOK, prw.statusCode)
	assert.NotNil(t, prw.header)
}

type generatedVariant struct {
	cause   error
	carrier *daemon_dto.PikoRequestCtx
	profile string
}

func TestQueueRemainingVariants_GeneratesMissingProfilesAfterTheRequest(t *testing.T) {
	generated := make(chan generatedVariant, 4)
	requestFinished := make(chan struct{})
	generator := &daemon_domain.MockOnDemandVariantGenerator{
		GenerateVariantFunc: func(ctx context.Context, _ *registry_dto.ArtefactMeta, profileName string) (*registry_dto.Variant, error) {
			<-requestFinished
			generated <- generatedVariant{
				cause:   context.Cause(ctx),
				carrier: daemon_dto.PikoRequestCtxFromContext(ctx),
				profile: profileName,
			}
			return nil, errors.New("generation failed")
		},
	}
	registry := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(context.Context, string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{ActualVariants: []registry_dto.Variant{{VariantID: "existing"}}}, nil
		},
	}
	artefact := &registry_dto.ArtefactMeta{
		ID: "images/photo.jpg",
		DesiredProfiles: []registry_dto.NamedProfile{
			{Name: "thumb"}, {Name: "existing"}, {Name: "webp"}, {Name: variantSource},
		},
	}
	builder, ok := NewHTTPRouterBuilder(nil, "").(*HTTPRouterBuilder)
	require.True(t, ok)

	pctx := daemon_dto.AcquirePikoRequestCtx()
	pctx.ClientIP = "10.0.0.1"
	requestCtx, cancelRequest := context.WithCancelCause(daemon_dto.WithPikoRequestCtx(context.Background(), pctx))

	builder.queueRemainingVariants(requestCtx, registry, generator, artefact, "webp")
	cancelRequest(errors.New("request finished"))
	daemon_dto.ReleasePikoRequestCtx(pctx)
	close(requestFinished)

	got := <-generated
	assert.Equal(t, "thumb", got.profile, "existing and already generated profiles are skipped")
	require.NoError(t, got.cause, "generation must outlive the request")
	require.NotNil(t, got.carrier)
	assert.NotSame(t, pctx, got.carrier, "generation must not read the pooled carrier")
	assert.Equal(t, "10.0.0.1", got.carrier.ClientIP)
	require.Eventually(t, func() bool { return generator.GenerateVariantCallCount.Load() == 1 },
		time.Second, time.Millisecond)
}

func TestQueueRemainingVariants_DoesNothingWithoutMissingProfiles(t *testing.T) {
	generator := &daemon_domain.MockOnDemandVariantGenerator{}
	builder, ok := NewHTTPRouterBuilder(nil, "").(*HTTPRouterBuilder)
	require.True(t, ok)
	artefact := &registry_dto.ArtefactMeta{DesiredProfiles: []registry_dto.NamedProfile{{Name: "webp"}}}

	builder.queueRemainingVariants(context.Background(), &registry_domain.MockRegistryService{}, generator, artefact, "webp")

	assert.Zero(t, generator.GenerateVariantCallCount.Load())
}

func videoRegistry(chunkBody string) *registry_domain.MockRegistryService {
	artefact := &registry_dto.ArtefactMeta{
		ID: "videos/intro.mp4",
		DesiredProfiles: []registry_dto.NamedProfile{
			{Name: hlsVariantPrefix + "720p"},
		},
		ActualVariants: []registry_dto.Variant{
			{
				VariantID: hlsVariantPrefix + "720p",
				Chunks: []registry_dto.VariantChunk{
					{ChunkID: "seg-001.ts", MimeType: "video/mp2t", DurationSeconds: new(6.0), SizeBytes: int64(len(chunkBody))},
				},
			},
		},
	}
	return &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, id string) (*registry_dto.ArtefactMeta, error) {
			if id != "intro" {
				return nil, registry_domain.ErrArtefactNotFound
			}
			return artefact, nil
		},
		GetVariantChunkFunc: func(_ context.Context, _ *registry_dto.Variant, chunkID string) (io.ReadCloser, error) {
			if chunkID == "broken.ts" {
				return nil, errors.New("storage unavailable")
			}
			return io.NopCloser(strings.NewReader(chunkBody)), nil
		},
	}
}

func mountStaticRoutes(t *testing.T, routerConfig *daemon_domain.RouterConfig, registry registry_domain.RegistryService) http.Handler {
	t.Helper()

	builder := &HTTPRouterBuilder{}
	t.Cleanup(builder.Close)
	router := chi.NewRouter()
	builder.setupBaseMiddleware(router, routerConfig, security_dto.CSPRuntimeConfig{})
	builder.setupStaticRoutes(router, routerConfig, registry, nil, nil)
	return router
}

func serveRoute(t *testing.T, handler http.Handler, target string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	maps.Copy(request.Header, header)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestStaticRoutes_ServeVideo(t *testing.T) {
	t.Parallel()

	const chunkBody = "video-bytes"
	handler := mountStaticRoutes(t, &daemon_domain.RouterConfig{
		DistServePath:     "/_piko/dist",
		ArtefactServePath: "/_piko/assets",
	}, videoRegistry(chunkBody))

	testCases := []struct {
		name       string
		target     string
		wantBody   string
		wantType   string
		wantStatus int
	}{
		{name: "master playlist", target: "/_piko/video/intro/master.m3u8", wantStatus: http.StatusOK, wantBody: "#EXTM3U", wantType: contentTypeMPEGURL},
		{name: "master playlist for a missing video", target: "/_piko/video/missing/master.m3u8", wantStatus: http.StatusNotFound},
		{name: "variant playlist", target: "/_piko/video/intro/720p/manifest.m3u8", wantStatus: http.StatusOK, wantBody: "seg-001.ts", wantType: contentTypeMPEGURL},
		{name: "variant playlist for a missing video", target: "/_piko/video/missing/720p/manifest.m3u8", wantStatus: http.StatusNotFound},
		{name: "variant playlist for a missing quality", target: "/_piko/video/intro/1080p/manifest.m3u8", wantStatus: http.StatusNotFound},
		{name: "chunk", target: "/_piko/video/intro/720p/seg-001.ts", wantStatus: http.StatusOK, wantBody: chunkBody, wantType: "video/mp2t"},
		{name: "chunk for a missing video", target: "/_piko/video/missing/720p/seg-001.ts", wantStatus: http.StatusNotFound},
		{name: "chunk for a missing quality", target: "/_piko/video/intro/1080p/seg-001.ts", wantStatus: http.StatusNotFound},
		{name: "chunk that does not exist", target: "/_piko/video/intro/720p/seg-999.ts", wantStatus: http.StatusNotFound},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			recorder := serveRoute(t, handler, testCase.target, nil)

			assert.Equal(t, testCase.wantStatus, recorder.Code)
			if testCase.wantBody != "" {
				assert.Contains(t, recorder.Body.String(), testCase.wantBody)
			}
			if testCase.wantType != "" {
				assert.Equal(t, testCase.wantType, recorder.Header().Get(headerContentType))
			}
		})
	}
}

func TestServeVideoChunk_StorageFailureAnswersInternalServerError(t *testing.T) {
	t.Parallel()

	registry := videoRegistry("video-bytes")
	artefact, err := registry.GetArtefact(t.Context(), "intro")
	require.NoError(t, err)
	artefact.ActualVariants[0].Chunks = append(artefact.ActualVariants[0].Chunks,
		registry_dto.VariantChunk{ChunkID: "broken.ts", MimeType: "video/mp2t"})

	handler := mountStaticRoutes(t, &daemon_domain.RouterConfig{ArtefactServePath: "/_piko/assets"}, registry)
	recorder := serveRoute(t, handler, "/_piko/video/intro/720p/broken.ts", nil)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestServeVideoHandlers_RejectMissingParameters(t *testing.T) {
	t.Parallel()

	builder := &HTTPRouterBuilder{}
	registry := videoRegistry("video-bytes")

	handlers := map[string]http.HandlerFunc{
		"master playlist":  builder.serveVideoMasterPlaylist(registry, false),
		"variant playlist": builder.serveVideoVariantPlaylist(registry, false),
		"chunk":            builder.serveVideoChunk(registry),
	}

	for name, handler := range handlers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			recorder := serveRoute(t, handler, "/video", nil)

			assert.Equal(t, http.StatusNotFound, recorder.Code)
			assert.Zero(t, registry.GetArtefactCallCount.Load())
		})
	}
}

func TestStaticRoutes_ServeSitemapAndRobots(t *testing.T) {
	t.Parallel()

	sitemapRegistry := staticByIDRegistry("sitemap-2.xml", "application/xml", `"sitemap-etag"`, "<urlset/>")
	handler := mountStaticRoutes(t, &daemon_domain.RouterConfig{ArtefactServePath: "/_piko/assets"}, sitemapRegistry)

	recorder := serveRoute(t, handler, "/sitemap-2.xml", nil)
	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "<urlset/>", recorder.Body.String())
	assert.Equal(t, contentTypeXML, recorder.Header().Get(headerContentType))
	assert.Equal(t, `"sitemap-etag"`, recorder.Header().Get(headerETag))

	notModified := serveRoute(t, handler, "/sitemap-2.xml", http.Header{headerIfNoneMatch: {`"sitemap-etag"`}})
	assert.Equal(t, http.StatusNotModified, notModified.Code)

	missing := serveRoute(t, handler, "/sitemap-9.xml", nil)
	assert.Equal(t, http.StatusNotFound, missing.Code)

	missingRobots := serveRoute(t, handler, "/robots.txt", nil)
	assert.Equal(t, http.StatusNotFound, missingRobots.Code)
	assert.Empty(t, missingRobots.Header().Get("X-Robots-Tag"))
}

func TestStaticRoutes_RobotsBlockWithholdsTheSite(t *testing.T) {
	t.Parallel()

	block := []byte("User-agent: *\nDisallow: /\n")
	handler := mountStaticRoutes(t, &daemon_domain.RouterConfig{
		ArtefactServePath: "/_piko/assets",
		RobotsTxtBlock:    block,
	}, &registry_domain.MockRegistryService{})

	recorder := serveRoute(t, handler, "/robots.txt", nil)
	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, string(block), recorder.Body.String())
	assert.Equal(t, "text/plain; charset=utf-8", recorder.Header().Get(headerContentType))
	assert.NotEmpty(t, recorder.Header().Get("X-Robots-Tag"), "every response is marked as not for indexing")

	etag := recorder.Header().Get(headerETag)
	require.NotEmpty(t, etag)
	notModified := serveRoute(t, handler, "/robots.txt", http.Header{headerIfNoneMatch: {etag}})
	assert.Equal(t, http.StatusNotModified, notModified.Code)

	ping := serveRoute(t, handler, "/ping", nil)
	assert.Equal(t, http.StatusOK, ping.Code)
	assert.NotEmpty(t, ping.Header().Get("X-Robots-Tag"))
}

func TestStaticRoutes_FrameworkBundles(t *testing.T) {
	t.Parallel()

	handler := mountStaticRoutes(t, &daemon_domain.RouterConfig{
		DistServePath:     "/_piko/dist",
		ArtefactServePath: "/_piko/assets",
	}, &registry_domain.MockRegistryService{})

	testCases := []struct {
		name           string
		file           string
		acceptEncoding string
		storedSuffix   string
	}{
		{name: "core runtime", file: "ppframework.core.es.js"},
		{name: "core runtime as brotli", file: "ppframework.core.es.js", acceptEncoding: "br, gzip", storedSuffix: ".br"},
		{name: "core runtime as gzip", file: "ppframework.core.es.js", acceptEncoding: "gzip", storedSuffix: ".gz"},
		{name: "minified core runtime", file: "ppframework.core.min.es.js"},
		{name: "component runtime", file: "ppframework.components.es.js"},
		{name: "minified component runtime", file: "ppframework.components.min.es.js"},
		{name: "modals module", file: "ppframework.modals.min.es.js"},
		{name: "toasts module", file: "ppframework.toasts.min.es.js"},
		{name: "analytics module", file: "ppframework.analytics.min.es.js"},
		{name: "dev module", file: "ppframework.dev.min.es.js"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			asset, found := daemon_frontend.GetAsset(t.Context(), "built/"+testCase.file+testCase.storedSuffix)
			require.True(t, found, "the embedded build must contain the bundle")

			request := http.Header{}
			if testCase.acceptEncoding != "" {
				request.Set(headerAcceptEncoding, testCase.acceptEncoding)
			}

			assertCachedForGood(t, handler, versionedBundleURL(testCase.file), request, asset.ETag, asset.Content)
			assertReexportsVersionedModule(t, handler, testCase.file, request)
		})
	}
}

func TestStaticRoutes_UnversionedClassicBundleAlwaysRevalidates(t *testing.T) {
	t.Parallel()

	handler := mountStaticRoutes(t, &daemon_domain.RouterConfig{
		DistServePath:     "/_piko/dist",
		ArtefactServePath: "/_piko/assets",
	}, &registry_domain.MockRegistryService{})

	asset, found := daemon_frontend.GetAsset(t.Context(), "built/ppframework.core.min.umd.js")
	require.True(t, found)

	assertAlwaysRevalidated(t, handler, "/_piko/dist/ppframework.core.min.umd.js", http.Header{}, asset.ETag, asset.Content)
}

func TestStaticRoutes_DevModeKeepsVersionedURLsButRevalidates(t *testing.T) {
	t.Parallel()

	handler := mountStaticRoutes(t, &daemon_domain.RouterConfig{
		DistServePath:     "/_piko/dist",
		ArtefactServePath: "/_piko/assets",
		WatchMode:         true,
		DisableHTTPCache:  true,
	}, &registry_domain.MockRegistryService{})

	asset, found := daemon_frontend.GetAsset(t.Context(), "built/ppframework.core.es.js")
	require.True(t, found)

	assertAlwaysRevalidated(t, handler, daemon_frontend.CoreRuntimeBuildDevelopment.CoreModuleURL(), http.Header{}, asset.ETag, asset.Content)
}

func TestStaticRoutes_RuntimeRecoveryScriptAlwaysRevalidates(t *testing.T) {
	t.Parallel()

	handler := mountStaticRoutes(t, &daemon_domain.RouterConfig{
		DistServePath:     "/_piko/dist",
		ArtefactServePath: "/_piko/assets",
	}, &registry_domain.MockRegistryService{})

	asset, found := daemon_frontend.GetAsset(t.Context(), daemon_frontend.RuntimeRecoveryAssetPath)
	require.True(t, found, "the embedded build must contain the runtime recovery script")

	assertAlwaysRevalidated(t, handler, daemon_frontend.RuntimeRecoveryURL, http.Header{}, asset.ETag, asset.Content)
}

func assertCachedForGood(
	t *testing.T,
	handler http.Handler,
	target string,
	request http.Header,
	wantETag string,
	wantBody []byte,
) {
	t.Helper()

	fresh := serveRoute(t, handler, target, request)
	require.Equal(t, http.StatusOK, fresh.Code)
	assert.Equal(t, cacheControlImmutable, fresh.Header().Get(headerCacheControl))
	assert.Equal(t, wantETag, fresh.Header().Get(headerETag))
	assert.Contains(t, fresh.Header().Values(headerVary), headerAcceptEncoding)
	assert.Equal(t, wantBody, fresh.Body.Bytes())

	conditional := request.Clone()
	conditional.Set(headerIfNoneMatch, wantETag)
	notModified := serveRoute(t, handler, target, conditional)
	require.Equal(t, http.StatusNotModified, notModified.Code)
	assert.Empty(t, notModified.Body.Bytes(), "a 304 must not carry a body")
}

func assertReexportsVersionedModule(t *testing.T, handler http.Handler, fileName string, request http.Header) {
	t.Helper()

	version := versionSegmentFor(fileName)
	wantETag := `"reexport-` + version + `"`

	fresh := serveRoute(t, handler, "/_piko/dist/"+fileName, request)
	require.Equal(t, http.StatusOK, fresh.Code)
	assert.Equal(t, cacheControlNoCache, fresh.Header().Get(headerCacheControl))
	assert.Equal(t, wantETag, fresh.Header().Get(headerETag))
	assert.Empty(t, fresh.Header().Get(headerContentEncoding), "the re-export module is never sent compressed")
	assert.Equal(t, "export * from \"./"+version+"/"+fileName+"\";\n", fresh.Body.String())

	stale := request.Clone()
	stale.Set(headerIfNoneMatch, `"etag-from-an-earlier-release"`)
	replaced := serveRoute(t, handler, "/_piko/dist/"+fileName, stale)
	require.Equal(t, http.StatusOK, replaced.Code, "a copy from an earlier release must be replaced in full")

	conditional := request.Clone()
	conditional.Set(headerIfNoneMatch, wantETag)
	notModified := serveRoute(t, handler, "/_piko/dist/"+fileName, conditional)
	assert.Equal(t, http.StatusNotModified, notModified.Code)
}

func TestStaticRoutes_ThemeStylesheetAlwaysRevalidates(t *testing.T) {
	t.Parallel()

	const (
		themeBody       = "body{color:red}"
		themeETag       = `"theme-minified"`
		themeBrotliBody = "brotli-theme-bytes"
		themeBrotliETag = `"theme-minified-br"`
	)

	handler := mountStaticRoutes(t, &daemon_domain.RouterConfig{
		DistServePath:     "/_piko/dist",
		ArtefactServePath: "/_piko/assets",
	}, themeRegistry(
		themeFixture{variantID: "minified", etag: themeETag, body: themeBody},
		themeFixture{variantID: "minified_br", etag: themeBrotliETag, encoding: encodingBrotli, body: themeBrotliBody},
	))

	testCases := []struct {
		name           string
		acceptEncoding string
		wantETag       string
		wantBody       string
	}{
		{name: "uncompressed stylesheet", wantETag: themeETag, wantBody: themeBody},
		{name: "brotli stylesheet", acceptEncoding: "br, gzip", wantETag: themeBrotliETag, wantBody: themeBrotliBody},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			request := http.Header{}
			if testCase.acceptEncoding != "" {
				request.Set(headerAcceptEncoding, testCase.acceptEncoding)
			}

			assertAlwaysRevalidated(t, handler, "/theme.css", request, testCase.wantETag, []byte(testCase.wantBody))
		})
	}
}

func assertAlwaysRevalidated(
	t *testing.T,
	handler http.Handler,
	target string,
	request http.Header,
	wantETag string,
	wantBody []byte,
) {
	t.Helper()

	fresh := serveRoute(t, handler, target, request)
	require.Equal(t, http.StatusOK, fresh.Code)
	assertRevalidatingCacheHeaders(t, fresh, wantETag)
	assert.Equal(t, wantBody, fresh.Body.Bytes())

	for _, ifNoneMatch := range []string{wantETag, "W/" + wantETag, `"another-tag", ` + wantETag} {
		conditional := request.Clone()
		conditional.Set(headerIfNoneMatch, ifNoneMatch)

		notModified := serveRoute(t, handler, target, conditional)
		require.Equal(t, http.StatusNotModified, notModified.Code, "If-None-Match %s must match", ifNoneMatch)
		assert.Empty(t, notModified.Body.Bytes(), "a 304 must not carry a body")
		assertRevalidatingCacheHeaders(t, notModified, wantETag)
	}

	stale := request.Clone()
	stale.Set(headerIfNoneMatch, `"etag-from-an-earlier-release"`)
	replaced := serveRoute(t, handler, target, stale)
	require.Equal(t, http.StatusOK, replaced.Code)
	assertRevalidatingCacheHeaders(t, replaced, wantETag)
	assert.Equal(t, wantBody, replaced.Body.Bytes(), "an outdated copy must be replaced in full")
}

func assertRevalidatingCacheHeaders(t *testing.T, recorder *httptest.ResponseRecorder, wantETag string) {
	t.Helper()

	assert.Equal(t, cacheControlNoCache, recorder.Header().Get(headerCacheControl))
	assert.Equal(t, wantETag, recorder.Header().Get(headerETag))
	assert.Contains(t, recorder.Header().Values(headerVary), headerAcceptEncoding)
}

type themeFixture struct {
	variantID string
	etag      string
	encoding  string
	body      string
}

func themeRegistry(fixtures ...themeFixture) *registry_domain.MockRegistryService {
	bodies := make(map[string]string, len(fixtures))
	variants := make([]registry_dto.Variant, 0, len(fixtures))
	for _, fixture := range fixtures {
		tags := registry_dto.Tags{}
		tags.Set(registry_dto.TagType, "minified")
		tags.Set(registry_dto.TagEtag, fixture.etag)
		if fixture.encoding != "" {
			tags.Set(registry_dto.TagContentEncoding, fixture.encoding)
		}
		variants = append(variants, registry_dto.Variant{
			VariantID:    fixture.variantID,
			MimeType:     contentTypeCSS,
			StorageKey:   "store/theme/" + fixture.variantID,
			SizeBytes:    int64(len(fixture.body)),
			Status:       registry_dto.VariantStatusReady,
			MetadataTags: tags,
		})
		bodies[fixture.variantID] = fixture.body
	}
	artefact := &registry_dto.ArtefactMeta{
		ID:             "theme.css",
		Status:         registry_dto.VariantStatusReady,
		ActualVariants: variants,
	}
	return &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, id string) (*registry_dto.ArtefactMeta, error) {
			if id != artefact.ID {
				return nil, registry_domain.ErrArtefactNotFound
			}
			return artefact, nil
		},
		GetVariantDataFunc: func(_ context.Context, variant *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(bodies[variant.VariantID])), nil
		},
	}
}

func TestServeStaticArtefactHandler_Failures(t *testing.T) {
	t.Parallel()

	config := staticArtefactConfig{
		artefactID:      "theme.css",
		defaultMimeType: contentTypeCSS,
		cacheMaxAge:     cacheControlMutableAsset,
		preferredType:   variantSource,
		useCompression:  false,
	}

	testCases := []struct {
		registry   *registry_domain.MockRegistryService
		name       string
		wantStatus int
	}{
		{
			name: "registry failure",
			registry: &registry_domain.MockRegistryService{
				GetArtefactFunc: func(context.Context, string) (*registry_dto.ArtefactMeta, error) {
					return nil, errors.New("registry unavailable")
				},
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "artefact without a source variant",
			registry: &registry_domain.MockRegistryService{
				GetArtefactFunc: func(context.Context, string) (*registry_dto.ArtefactMeta, error) {
					return &registry_dto.ArtefactMeta{ID: "theme.css"}, nil
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "variant data cannot be read",
			registry: &registry_domain.MockRegistryService{
				GetArtefactFunc: func(context.Context, string) (*registry_dto.ArtefactMeta, error) {
					return &registry_dto.ArtefactMeta{
						ID:             "theme.css",
						ActualVariants: []registry_dto.Variant{{VariantID: variantSource}},
					}, nil
				},
				GetVariantDataFunc: func(context.Context, *registry_dto.Variant) (io.ReadCloser, error) {
					return nil, errors.New("blob missing")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			handler := serveStaticArtefactHandler(testCase.registry, config, "serveTest")
			recorder := serveRoute(t, handler, "/theme.css", nil)

			assert.Equal(t, testCase.wantStatus, recorder.Code)
		})
	}
}

func TestServeStaticArtefactHandler_WritesVariantHeaders(t *testing.T) {
	t.Parallel()

	tags := registry_dto.Tags{}
	tags.Set(registry_dto.TagEtag, `"theme-etag"`)
	tags.Set(registry_dto.TagMimeType, "text/css; charset=utf-8")
	tags.Set(registry_dto.TagContentEncoding, "br")
	registry := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(context.Context, string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID:             "theme.css",
				ActualVariants: []registry_dto.Variant{{VariantID: variantSource, SizeBytes: 4, MetadataTags: tags}},
			}, nil
		},
		GetVariantDataFunc: func(context.Context, *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("body")), nil
		},
	}

	handler := serveStaticArtefactHandler(registry, staticArtefactConfig{
		artefactID:      "theme.css",
		defaultMimeType: contentTypeCSS,
		cacheMaxAge:     cacheControlMutableAsset,
		preferredType:   variantSource,
		useCompression:  false,
	}, "serveTest")
	recorder := serveRoute(t, handler, "/theme.css", nil)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "body", recorder.Body.String())
	assert.Equal(t, "text/css; charset=utf-8", recorder.Header().Get(headerContentType))
	assert.Equal(t, "br", recorder.Header().Get(headerContentEncoding))
	assert.Equal(t, "4", recorder.Header().Get(headerContentLength))
	assert.Equal(t, cacheControlMutableAsset, recorder.Header().Get(headerCacheControl))
}

func pendingArtefactRegistry() *registry_domain.MockRegistryService {
	artefact := &registry_dto.ArtefactMeta{
		ID:     "images/photo.jpg",
		Status: registry_dto.VariantStatusPending,
	}
	return &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, id string) (*registry_dto.ArtefactMeta, error) {
			if id != artefact.ID {
				return nil, registry_domain.ErrArtefactNotFound
			}
			return artefact, nil
		},
		GetVariantDataFunc: func(context.Context, *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("image-bytes")), nil
		},
	}
}

func mountArtefactRoute(t *testing.T, registry registry_domain.RegistryService, generator daemon_domain.OnDemandVariantGenerator) http.Handler {
	t.Helper()

	builder := &HTTPRouterBuilder{}
	t.Cleanup(builder.Close)
	router := chi.NewRouter()
	router.Get("/_piko/assets/*", builder.serveArtefact(registry, generator, false))
	return router
}

func TestServeArtefact_PendingArtefactGeneratesItsFirstVariant(t *testing.T) {
	t.Parallel()

	generator := &daemon_domain.MockOnDemandVariantGenerator{
		ParseProfileNameFunc: func(string) *daemon_domain.ParsedImageProfile {
			return &daemon_domain.ParsedImageProfile{}
		},
		GenerateVariantFunc: func(_ context.Context, _ *registry_dto.ArtefactMeta, profileName string) (*registry_dto.Variant, error) {
			return &registry_dto.Variant{VariantID: profileName, MimeType: "image/webp", StorageKey: "store/photo.webp"}, nil
		},
	}

	handler := mountArtefactRoute(t, pendingArtefactRegistry(), generator)
	recorder := serveRoute(t, handler, "/_piko/assets/images/photo.jpg?v=webp", nil)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, int64(1), generator.GenerateVariantCallCount.Load())
}

func TestServeArtefact_PendingArtefactAnswersUnavailableWhenGenerationFails(t *testing.T) {
	t.Parallel()

	generator := &daemon_domain.MockOnDemandVariantGenerator{
		ParseProfileNameFunc: func(string) *daemon_domain.ParsedImageProfile {
			return &daemon_domain.ParsedImageProfile{}
		},
		GenerateVariantFunc: func(context.Context, *registry_dto.ArtefactMeta, string) (*registry_dto.Variant, error) {
			return nil, errors.New("encoder crashed")
		},
	}

	handler := mountArtefactRoute(t, pendingArtefactRegistry(), generator)
	recorder := serveRoute(t, handler, "/_piko/assets/images/photo.jpg", nil)

	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
}

func TestServeArtefact_PendingArtefactWithoutAGeneratorAnswersUnavailable(t *testing.T) {
	t.Parallel()

	handler := mountArtefactRoute(t, pendingArtefactRegistry(), nil)
	recorder := serveRoute(t, handler, "/_piko/assets/images/photo.jpg", nil)

	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
}

func TestServeArtefact_ResolvesVariantsByArtefactID(t *testing.T) {
	t.Parallel()

	readyArtefact := &registry_dto.ArtefactMeta{
		ID:     "images/logo.png",
		Status: registry_dto.VariantStatusReady,
		ActualVariants: []registry_dto.Variant{
			{VariantID: variantSource, MimeType: "image/png", StorageKey: "store/logo.png", Status: registry_dto.VariantStatusReady},
		},
	}
	noSourceArtefact := &registry_dto.ArtefactMeta{
		ID:             "images/empty.png",
		Status:         registry_dto.VariantStatusReady,
		ActualVariants: []registry_dto.Variant{{VariantID: "thumb", StorageKey: "store/empty-thumb.png", Status: registry_dto.VariantStatusReady}},
	}
	registry := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, id string) (*registry_dto.ArtefactMeta, error) {
			switch id {
			case readyArtefact.ID:
				return readyArtefact, nil
			case noSourceArtefact.ID:
				return noSourceArtefact, nil
			default:
				return nil, registry_domain.ErrArtefactNotFound
			}
		},
		FindArtefactByVariantStorageKeyFunc: func(context.Context, string) (*registry_dto.ArtefactMeta, error) {
			return nil, registry_domain.ErrArtefactNotFound
		},
		GetVariantDataFunc: func(context.Context, *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("image-bytes")), nil
		},
	}
	generator := &daemon_domain.MockOnDemandVariantGenerator{
		ParseProfileNameFunc: func(string) *daemon_domain.ParsedImageProfile { return nil },
	}
	handler := mountArtefactRoute(t, registry, generator)

	testCases := []struct {
		name       string
		target     string
		wantStatus int
	}{
		{name: "an unknown variant falls back to the source", target: "/_piko/assets/images/logo.png?v=huge", wantStatus: http.StatusOK},
		{name: "an artefact without a source variant is not found", target: "/_piko/assets/images/empty.png", wantStatus: http.StatusNotFound},
		{name: "an unknown artefact is not found", target: "/_piko/assets/images/missing.png", wantStatus: http.StatusNotFound},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			recorder := serveRoute(t, handler, testCase.target, nil)

			assert.Equal(t, testCase.wantStatus, recorder.Code)
		})
	}
}

func versionedBundleURL(fileName string) string {
	return "/_piko/dist/" + versionSegmentFor(fileName) + "/" + fileName
}

func versionSegmentFor(fileName string) string {
	if daemon_frontend.HasBuildVariants(fileName) {
		return daemon_frontend.CoreRuntimeBuildDevelopment.RuntimeVersion()
	}
	return daemon_frontend.RuntimeVersion()
}
