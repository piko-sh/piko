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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/go-chi/chi/v5"
	"github.com/klauspost/compress/gzip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/render/render_dto"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func createTestArtefactCache(t *testing.T) *artefactMetadataCache {
	t.Helper()
	otterCache, err := provider_otter.OtterProviderFactory(cache_dto.Options[string, *registry_dto.ArtefactMeta]{
		Namespace:   "test-artefact-metadata",
		MaximumSize: 100,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = otterCache.Close(context.Background()) })
	return &artefactMetadataCache{cache: otterCache}
}

func Test_newArtefactMetadataCache_CreatesCache(t *testing.T) {
	t.Parallel()

	otterCache, err := provider_otter.OtterProviderFactory(cache_dto.Options[string, *registry_dto.ArtefactMeta]{
		Namespace:   "test-artefact-metadata",
		MaximumSize: 100,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = otterCache.Close(context.Background()) })

	registryService := &registry_domain.MockRegistryService{}
	cache := newArtefactMetadataCache(otterCache, registryService)

	require.NotNil(t, cache, "Cache should not be nil")
	require.NotNil(t, cache.cache, "Internal cache should not be nil")
}

func Test_artefactMetadataCache_Get_CacheMiss(t *testing.T) {
	t.Parallel()

	registryService := &registry_domain.MockRegistryService{}
	c := createTestArtefactCache(t)
	c.registryService = registryService

	artefact, ok := c.Get(context.Background(), "non-existent-artefact")

	assert.False(t, ok, "Should return false for cache miss")
	assert.Nil(t, artefact, "Should return nil artefact for cache miss")
}

func Test_artefactMetadataCache_GetOrLoad_LoadsFromRegistry(t *testing.T) {
	t.Parallel()

	expectedArtefact := &registry_dto.ArtefactMeta{
		ID:             "test-artefact",
		ActualVariants: []registry_dto.Variant{},
	}

	registryService := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
			if artefactID == "test-artefact" {
				return expectedArtefact, nil
			}
			return nil, errors.New("not found")
		},
	}
	c := createTestArtefactCache(t)
	c.registryService = registryService

	artefact, err := c.GetOrLoad(context.Background(), "test-artefact")

	require.NoError(t, err, "Should not return error")
	assert.Equal(t, expectedArtefact.ID, artefact.ID, "Should return correct artefact")
}

func Test_artefactMetadataCache_GetOrLoad_ReturnsErrorFromRegistry(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("registry error")
	registryService := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, expectedErr
		},
	}
	c := createTestArtefactCache(t)
	c.registryService = registryService

	artefact, err := c.GetOrLoad(context.Background(), "test-artefact")

	require.Error(t, err, "Should return error from registry")
	assert.Nil(t, artefact, "Should return nil artefact on error")
}

func Test_artefactMetadataCache_GetOrLoad_CachesResult(t *testing.T) {
	t.Parallel()

	expectedArtefact := &registry_dto.ArtefactMeta{
		ID: "cached-artefact",
	}

	registryService := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return expectedArtefact, nil
		},
	}
	c := createTestArtefactCache(t)
	c.registryService = registryService
	ctx := context.Background()

	artefact1, err1 := c.GetOrLoad(ctx, "cached-artefact")
	require.NoError(t, err1)
	assert.Equal(t, int64(1), atomic.LoadInt64(&registryService.GetArtefactCallCount), "First call should hit registry")
	assert.Equal(t, expectedArtefact.ID, artefact1.ID)

	artefact2, err2 := c.GetOrLoad(ctx, "cached-artefact")
	require.NoError(t, err2)
	assert.Equal(t, int64(1), atomic.LoadInt64(&registryService.GetArtefactCallCount), "Second call should use cache, not hit registry again")
	assert.Equal(t, expectedArtefact.ID, artefact2.ID)
}

func Test_artefactMetadataCache_Get_AfterGetOrLoad(t *testing.T) {
	t.Parallel()

	expectedArtefact := &registry_dto.ArtefactMeta{
		ID: "test-artefact",
	}

	registryService := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return expectedArtefact, nil
		},
	}
	c := createTestArtefactCache(t)
	c.registryService = registryService
	ctx := context.Background()

	_, err := c.GetOrLoad(ctx, "test-artefact")
	require.NoError(t, err)

	artefact, ok := c.Get(ctx, "test-artefact")

	assert.True(t, ok, "Should find artefact in cache after GetOrLoad")
	assert.Equal(t, expectedArtefact.ID, artefact.ID)
}

func Test_artefactMetadataCache_Invalidate(t *testing.T) {
	t.Parallel()

	expectedArtefact := &registry_dto.ArtefactMeta{
		ID: "invalidate-test",
	}

	registryService := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return expectedArtefact, nil
		},
	}
	c := createTestArtefactCache(t)
	c.registryService = registryService
	ctx := context.Background()

	_, err := c.GetOrLoad(ctx, "invalidate-test")
	require.NoError(t, err)

	_, ok := c.Get(ctx, "invalidate-test")
	assert.True(t, ok, "Should be in cache before invalidate")

	c.Invalidate(ctx, "invalidate-test")

	_, ok = c.Get(ctx, "invalidate-test")
	assert.False(t, ok, "Should not be in cache after invalidate")
}

func Test_artefactMetadataCache_Stats(t *testing.T) {
	t.Parallel()

	c := createTestArtefactCache(t)

	stats := c.Stats()

	assert.NotNil(t, stats, "Stats should not be nil")
}

func Test_newOSSignalNotifier_ReturnsNotNil(t *testing.T) {
	t.Parallel()

	notifier := newOSSignalNotifier()

	require.NotNil(t, notifier, "Should return non-nil notifier")
}

func Test_osSignalNotifier_NotifyContext_ReturnsValidContext(t *testing.T) {
	t.Parallel()

	notifier := newOSSignalNotifier()
	parent := context.Background()

	ctx, cancel := notifier.NotifyContext(parent)
	defer cancel()

	require.NotNil(t, ctx, "Should return non-nil context")
	require.NotNil(t, cancel, "Should return non-nil cancel function")

	select {
	case <-ctx.Done():
		t.Error("Context should not be cancelled initially")
	default:

	}
}

func Test_osSignalNotifier_NotifyContext_CancelWorksImmediately(t *testing.T) {
	t.Parallel()

	notifier := newOSSignalNotifier()
	parent := context.Background()

	ctx, cancel := notifier.NotifyContext(parent)

	cancel()

	select {
	case <-ctx.Done():

	case <-time.After(100 * time.Millisecond):
		t.Error("Context should be cancelled after calling cancel()")
	}
}

func Test_osSignalNotifier_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ daemon_domain.SignalNotifier = newOSSignalNotifier()
}

func TestNewRouterManager_CreatesManager(t *testing.T) {
	t.Parallel()

	routerManagerConfig := &RouterManagerConfig{
		CSRFService:      &security_domain.MockCSRFTokenService{},
		RegistryService:  &registry_domain.MockRegistryService{},
		VariantGenerator: &daemon_domain.MockOnDemandVariantGenerator{},
		RouteSettings:    RouteSettings{},
		CSPConfig:        security_dto.CSPRuntimeConfig{},
		Deps:             &daemon_domain.HTTPHandlerDependencies{},
		SiteSettings:     &config.WebsiteConfig{},
		Actions:          map[string]ActionHandlerEntry{},
		CacheMiddleware:  nil,
		AppRouter:        chi.NewRouter(),
		RouteProviders:   nil,
		RouterConfig:     &daemon_domain.RouterConfig{},
	}

	rm := NewRouterManager(routerManagerConfig)

	require.NotNil(t, rm, "RouterManager should not be nil")
	assert.Nil(t, rm.currentRouter, "currentRouter should be nil initially")
}

func TestRouterManager_ServeHTTP_ReturnsServiceUnavailable_WhenNoRouter(t *testing.T) {
	t.Parallel()

	artefactConfig := &RouterManagerConfig{
		CSRFService:      &security_domain.MockCSRFTokenService{},
		RegistryService:  &registry_domain.MockRegistryService{},
		VariantGenerator: &daemon_domain.MockOnDemandVariantGenerator{},
		RouteSettings:    RouteSettings{},
		CSPConfig:        security_dto.CSPRuntimeConfig{},
		Deps:             &daemon_domain.HTTPHandlerDependencies{},
		SiteSettings:     &config.WebsiteConfig{},
		Actions:          map[string]ActionHandlerEntry{},
		CacheMiddleware:  nil,
		AppRouter:        chi.NewRouter(),
		RouteProviders:   nil,
		RouterConfig:     &daemon_domain.RouterConfig{},
	}

	rm := NewRouterManager(artefactConfig)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	rm.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code, "Should return 503 when no router is set")
	assert.Contains(t, recorder.Body.String(), "Service is initialising", "Should return appropriate message")
}

func TestRouterManager_ServeHTTP_DelegatesToRouter(t *testing.T) {
	t.Parallel()

	artefactConfig := &RouterManagerConfig{
		CSRFService:      &security_domain.MockCSRFTokenService{},
		RegistryService:  &registry_domain.MockRegistryService{},
		VariantGenerator: &daemon_domain.MockOnDemandVariantGenerator{},
		RouteSettings:    RouteSettings{},
		CSPConfig:        security_dto.CSPRuntimeConfig{},
		Deps:             &daemon_domain.HTTPHandlerDependencies{},
		SiteSettings:     &config.WebsiteConfig{},
		Actions:          map[string]ActionHandlerEntry{},
		CacheMiddleware:  nil,
		AppRouter:        chi.NewRouter(),
		RouteProviders:   nil,
		RouterConfig:     &daemon_domain.RouterConfig{},
	}

	rm := NewRouterManager(artefactConfig)

	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	rm.mu.Lock()
	rm.currentRouter = mockHandler
	rm.mu.Unlock()

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	rm.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Should delegate to the current router")
	assert.Equal(t, "OK", recorder.Body.String())
}

func TestRouterManager_ReloadRoutes_ReturnsError_WhenBuildFails(t *testing.T) {
	t.Parallel()

	artefactConfig := &RouterManagerConfig{
		CSRFService:     &security_domain.MockCSRFTokenService{},
		RegistryService: &registry_domain.MockRegistryService{},

		RouteSettings:    RouteSettings{},
		CSPConfig:        security_dto.CSPRuntimeConfig{},
		Deps:             &daemon_domain.HTTPHandlerDependencies{},
		SiteSettings:     &config.WebsiteConfig{},
		Actions:          map[string]ActionHandlerEntry{},
		CacheMiddleware:  nil,
		AppRouter:        chi.NewRouter(),
		RouteProviders:   nil,
		VariantGenerator: &daemon_domain.MockOnDemandVariantGenerator{},
		RouterConfig:     &daemon_domain.RouterConfig{},
	}

	rm := NewRouterManager(artefactConfig)
	t.Cleanup(rm.Close)
	store := &templater_domain.MockManifestStoreView{}

	err := rm.ReloadRoutes(context.Background(), store)

	if err != nil {
		assert.Contains(t, err.Error(), "failed to build new router", "Error should mention router building")
	}
}

func Test_NewDriverHTTPServerAdapter_ReturnsNotNil(t *testing.T) {
	t.Parallel()

	adapter := NewDriverHTTPServerAdapter()

	require.NotNil(t, adapter, "Should return non-nil adapter")
}

func Test_driverHTTPServerAdapter_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ daemon_domain.ServerAdapter = NewDriverHTTPServerAdapter()
}

func Test_driverHTTPServerAdapter_Shutdown_ReturnsNil_WhenNoServer(t *testing.T) {
	t.Parallel()

	adapter := NewDriverHTTPServerAdapter()

	err := adapter.Shutdown(context.Background())

	assert.NoError(t, err, "Should return nil when no server to shutdown")
}

func Test_driverHTTPServerAdapter_Shutdown_WithTimeout(t *testing.T) {
	t.Parallel()

	adapter, ok := NewDriverHTTPServerAdapter().(*driverHTTPServerAdapter)
	require.True(t, ok, "NewDriverHTTPServerAdapter() should return *driverHTTPServerAdapter")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	adapter.server = &http.Server{
		Handler: testHandler,
	}

	ctx, cancel := context.WithTimeoutCause(context.Background(), 100*time.Millisecond, fmt.Errorf("test: Shutdown timeout"))
	defer cancel()

	err := adapter.Shutdown(ctx)

	assert.NoError(t, err, "Shutdown should complete without error")
}

func TestHTTPConstants_HaveExpectedValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 5*time.Second, defaultReadTimeout)
	assert.Equal(t, 10*time.Second, defaultWriteTimeout)
	assert.Equal(t, 120*time.Second, defaultIdleTimeout)
	assert.Equal(t, 2*time.Second, defaultReadHeaderTimeout)
	assert.Equal(t, 1<<20, defaultMaxHeaderBytes)

	assert.Equal(t, "Content-Type", headerContentType)
	assert.Equal(t, "Content-Encoding", headerContentEncoding)
	assert.Equal(t, "Cache-Control", headerCacheControl)
	assert.Equal(t, "Etag", headerETag)

	assert.Equal(t, "br", encodingBrotli)
	assert.Equal(t, "gzip", encodingGzip)

	assert.Equal(t, "application/json", contentTypeJSON)
	assert.Equal(t, "text/html; charset=utf-8", contentTypeHTML)
}

func TestNewHTTPRouterBuilder_ReturnsNotNil(t *testing.T) {
	t.Parallel()

	builder := NewHTTPRouterBuilder(nil)

	require.NotNil(t, builder, "Builder should not be nil")
}

func TestFindVariantByID_ReturnsVariant_WhenFound(t *testing.T) {
	t.Parallel()

	variants := []registry_dto.Variant{
		{VariantID: "source"},
		{VariantID: "minified"},
		{VariantID: "source_br"},
	}

	result := findVariantByID(variants, "minified")

	require.NotNil(t, result, "Should find variant")
	assert.Equal(t, "minified", result.VariantID)
}

func TestFindVariantByID_ReturnsNil_WhenNotFound(t *testing.T) {
	t.Parallel()

	variants := []registry_dto.Variant{
		{VariantID: "source"},
		{VariantID: "minified"},
	}

	result := findVariantByID(variants, "nonexistent")

	assert.Nil(t, result, "Should return nil when not found")
}

func TestFindVariantByID_ReturnsNil_ForEmptySlice(t *testing.T) {
	t.Parallel()

	result := findVariantByID([]registry_dto.Variant{}, "any")

	assert.Nil(t, result, "Should return nil for empty slice")
}

func TestFindVariantByStorageKey_ReturnsVariant_WhenFound(t *testing.T) {
	t.Parallel()

	variants := []registry_dto.Variant{
		{VariantID: "v1", StorageKey: "artefacts/1/source.js"},
		{VariantID: "v2", StorageKey: "artefacts/1/minified.js"},
	}

	result := findVariantByStorageKey(variants, "artefacts/1/minified.js")

	require.NotNil(t, result, "Should find variant")
	assert.Equal(t, "v2", result.VariantID)
}

func TestFindVariantByStorageKey_ReturnsNil_WhenNotFound(t *testing.T) {
	t.Parallel()

	variants := []registry_dto.Variant{
		{VariantID: "v1", StorageKey: "artefacts/1/source.js"},
	}

	result := findVariantByStorageKey(variants, "nonexistent")

	assert.Nil(t, result, "Should return nil when not found")
}

func TestParseFragmentParam_ReturnsTrue_WhenSetToTrue(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/?_f=true", nil)

	result := parseFragmentParam(request)

	assert.True(t, result, "Should return true when _f=true")
}

func TestParseFragmentParam_ReturnsTrue_WhenSetToOne(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/?_f=1", nil)

	result := parseFragmentParam(request)

	assert.True(t, result, "Should return true when _f=1")
}

func TestParseFragmentParam_ReturnsFalse_WhenSetToFalse(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/?_f=false", nil)

	result := parseFragmentParam(request)

	assert.False(t, result, "Should return false when _f=false")
}

func TestParseFragmentParam_ReturnsFalse_WhenNotSet(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/", nil)

	result := parseFragmentParam(request)

	assert.False(t, result, "Should return false when not set")
}

func TestParseFragmentParam_ReturnsFalse_WhenInvalidValue(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/?_f=invalid", nil)

	result := parseFragmentParam(request)

	assert.False(t, result, "Should return false for invalid value")
}

type mockLogger struct {
	logger_domain.Logger
	warnCalled     bool
	internalCalled bool
}

func (m *mockLogger) Warn(_ string, _ ...logger_domain.Attr) {
	m.warnCalled = true
}

func (m *mockLogger) Trace(_ string, _ ...logger_domain.Attr) {}

func (m *mockLogger) Internal(_ string, _ ...logger_domain.Attr) {
	m.internalCalled = true
}

func (m *mockLogger) Debug(_ string, _ ...logger_domain.Attr) {}

func (m *mockLogger) Info(_ string, _ ...logger_domain.Attr) {}

func (m *mockLogger) Notice(_ string, _ ...logger_domain.Attr) {}

func (m *mockLogger) Error(_ string, _ ...logger_domain.Attr) {}

func (m *mockLogger) Panic(_ string, _ ...logger_domain.Attr) {}

func (m *mockLogger) ReportError(_ trace.Span, _ error, _ string, _ ...logger_domain.Attr) {}

func TestValidateRedirectStatusCode_Returns301_For301(t *testing.T) {
	t.Parallel()

	result := validateRedirectStatusCode(context.Background(), http.StatusMovedPermanently)

	assert.Equal(t, http.StatusMovedPermanently, result)
}

func TestValidateRedirectStatusCode_Returns302_For302(t *testing.T) {
	t.Parallel()

	result := validateRedirectStatusCode(context.Background(), http.StatusFound)

	assert.Equal(t, http.StatusFound, result)
}

func TestValidateRedirectStatusCode_Returns303_For303(t *testing.T) {
	t.Parallel()

	result := validateRedirectStatusCode(context.Background(), http.StatusSeeOther)

	assert.Equal(t, http.StatusSeeOther, result)
}

func TestValidateRedirectStatusCode_Returns307_For307(t *testing.T) {
	t.Parallel()

	result := validateRedirectStatusCode(context.Background(), http.StatusTemporaryRedirect)

	assert.Equal(t, http.StatusTemporaryRedirect, result)
}

func TestValidateRedirectStatusCode_Returns302_ForInvalidCode(t *testing.T) {
	t.Parallel()

	result := validateRedirectStatusCode(context.Background(), http.StatusOK)

	assert.Equal(t, http.StatusFound, result, "Should default to 302 for invalid codes")
}

func TestValidateRedirectStatusCode_Returns302_For404(t *testing.T) {
	t.Parallel()

	result := validateRedirectStatusCode(context.Background(), http.StatusNotFound)

	assert.Equal(t, http.StatusFound, result, "Should default to 302 for invalid codes")
}

func TestGenerateCacheArtefactID_GeneratesConsistentID(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/test/page", nil)
	policy := templater_dto.CachePolicy{}

	id1 := generateCacheArtefactID(request, policy)
	id2 := generateCacheArtefactID(request, policy)

	assert.Equal(t, id1, id2, "Same request should generate same ID")
	assert.True(t, strings.HasPrefix(id1, "page:"), "ID should have page: prefix")
}

func TestGenerateCacheArtefactID_DiffersByPath(t *testing.T) {
	t.Parallel()

	req1 := httptest.NewRequest(http.MethodGet, "/path/one", nil)
	req2 := httptest.NewRequest(http.MethodGet, "/path/two", nil)
	policy := templater_dto.CachePolicy{}

	id1 := generateCacheArtefactID(req1, policy)
	id2 := generateCacheArtefactID(req2, policy)

	assert.NotEqual(t, id1, id2, "Different paths should generate different IDs")
}

func TestGenerateCacheArtefactID_DiffersByPolicyKey(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/test/page", nil)
	policy1 := templater_dto.CachePolicy{Key: "key1"}
	policy2 := templater_dto.CachePolicy{Key: "key2"}

	id1 := generateCacheArtefactID(request, policy1)
	id2 := generateCacheArtefactID(request, policy2)

	assert.NotEqual(t, id1, id2, "Different policy keys should generate different IDs")
}

func TestGenerateCacheArtefactID_DiffersByQueryString(t *testing.T) {
	t.Parallel()

	req1 := httptest.NewRequest(http.MethodGet, "/test?a=1", nil)
	req2 := httptest.NewRequest(http.MethodGet, "/test?a=2", nil)
	policy := templater_dto.CachePolicy{}

	id1 := generateCacheArtefactID(req1, policy)
	id2 := generateCacheArtefactID(req2, policy)

	assert.NotEqual(t, id1, id2, "Different query strings should generate different IDs")
}

func TestFindBestCompressedVariant_ReturnsBrotli_WhenAccepted(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source"},
			{VariantID: "source_br"},
			{VariantID: "source_gzip"},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "br, gzip, deflate")

	result := findBestCompressedVariant(request, artefact, "source")

	require.NotNil(t, result, "Should find brotli variant")
	assert.Equal(t, "source_br", result.VariantID)
}

func TestFindBestCompressedVariant_ReturnsGzip_WhenOnlyGzipAccepted(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source"},
			{VariantID: "source_br"},
			{VariantID: "source_gzip"},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "gzip, deflate")

	result := findBestCompressedVariant(request, artefact, "source")

	require.NotNil(t, result, "Should find gzip variant")
	assert.Equal(t, "source_gzip", result.VariantID)
}

func TestFindBestCompressedVariant_ReturnsBase_WhenNoEncodingAccepted(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source"},
			{VariantID: "source_br"},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	result := findBestCompressedVariant(request, artefact, "source")

	require.NotNil(t, result, "Should find source variant")
	assert.Equal(t, "source", result.VariantID)
}

func TestFindBestCompressedVariant_ReturnsNil_WhenNoVariantsMatch(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: "other"},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "br")

	result := findBestCompressedVariant(request, artefact, "source")

	assert.Nil(t, result, "Should return nil when no variants match")
}

func TestLogFieldConstants_HaveExpectedValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "path", logFieldPath)
	assert.Equal(t, "method", logFieldMethod)
	assert.Equal(t, "artefactID", logFieldArtefactID)
	assert.Equal(t, "variantID", logFieldVariantID)
	assert.Equal(t, "error", logFieldError)
	assert.Equal(t, "url", logFieldURL)
	assert.Equal(t, "originalPath", logFieldOriginalPath)
	assert.Equal(t, "routePattern", logFieldRoutePattern)
	assert.Equal(t, "actionCount", logFieldActionCount)
	assert.Equal(t, "actionName", logFieldActionName)
}

func TestCollectMissingVariantProfiles_ReturnsProfiles_ExcludingAlreadyGenerated(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		DesiredProfiles: []registry_dto.NamedProfile{
			{Name: "source"},
			{Name: "thumbnail"},
			{Name: "large"},
			{Name: "webp"},
		},
	}

	result := collectMissingVariantProfiles(artefact, "thumbnail")

	assert.Len(t, result, 2, "Should exclude source and the already generated profile")
	assert.Contains(t, result, "large")
	assert.Contains(t, result, "webp")
	assert.NotContains(t, result, "source")
	assert.NotContains(t, result, "thumbnail")
}

func TestCollectMissingVariantProfiles_ReturnsEmpty_WhenAllGenerated(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		DesiredProfiles: []registry_dto.NamedProfile{
			{Name: "source"},
		},
	}

	result := collectMissingVariantProfiles(artefact, "other")

	assert.Empty(t, result, "Should return empty when only source profile exists")
}

func TestCollectMissingVariantProfiles_ReturnsEmpty_ForEmptyProfiles(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		DesiredProfiles: []registry_dto.NamedProfile{},
	}

	result := collectMissingVariantProfiles(artefact, "any")

	assert.Empty(t, result)
}

func TestVariantExistsInArtefact_ReturnsTrue_WhenExists(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source"},
			{VariantID: "thumbnail"},
		},
	}

	assert.True(t, variantExistsInArtefact(artefact, "source"))
	assert.True(t, variantExistsInArtefact(artefact, "thumbnail"))
}

func TestVariantExistsInArtefact_ReturnsFalse_WhenNotExists(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source"},
		},
	}

	assert.False(t, variantExistsInArtefact(artefact, "nonexistent"))
}

func TestVariantExistsInArtefact_ReturnsFalse_ForEmptyVariants(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{},
	}

	assert.False(t, variantExistsInArtefact(artefact, "any"))
}

func TestGetMatchedPattern_ReturnsContextPattern_WhenSet(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{
		MatchedPattern: "/test/{id}",
	})
	request = request.WithContext(ctx)

	result := getMatchedPattern(request)

	assert.Equal(t, "/test/{id}", result)
}

func TestGetMatchedPattern_ReturnsEmpty_WhenNoContext(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/test", nil)

	result := getMatchedPattern(request)

	assert.Empty(t, result)
}

func TestApplyMiddlewares_AppliesCacheMiddleware(t *testing.T) {
	t.Parallel()

	called := false
	cacheMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			next.ServeHTTP(w, r)
		})
	}

	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	entry := &templater_domain.MockPageEntryView{}
	result := applyMiddlewares(baseHandler, entry, cacheMiddleware, nil)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	result.ServeHTTP(recorder, request)

	assert.True(t, called, "Cache middleware should have been called")
}

func TestApplyMiddlewares_AppliesEntryMiddlewares(t *testing.T) {
	t.Parallel()

	var order []string
	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw1")
			next.ServeHTTP(w, r)
		})
	}
	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw2")
			next.ServeHTTP(w, r)
		})
	}

	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	})

	entry := &templater_domain.MockPageEntryView{
		GetHasMiddlewareFunc: func() bool { return true },
		GetMiddlewaresFunc:   func() []func(http.Handler) http.Handler { return []func(http.Handler) http.Handler{mw1, mw2} },
	}
	result := applyMiddlewares(baseHandler, entry, nil, nil)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	result.ServeHTTP(recorder, request)

	assert.Equal(t, []string{"mw1", "mw2", "handler"}, order)
}

func TestApplyMiddlewares_ReturnsBaseHandler_WhenNoMiddlewares(t *testing.T) {
	t.Parallel()

	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	entry := &templater_domain.MockPageEntryView{}
	result := applyMiddlewares(baseHandler, entry, nil, nil)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	result.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestBuildHeaders_BasicHeader(t *testing.T) {
	t.Parallel()

	lh := render_dto.LinkHeader{
		URL: "/styles.css",
		Rel: "preload",
	}

	result := buildHeaders(lh)

	assert.Equal(t, "</styles.css>; rel=preload", result)
}

func TestBuildHeaders_WithAs(t *testing.T) {
	t.Parallel()

	lh := render_dto.LinkHeader{
		URL: "/styles.css",
		Rel: "preload",
		As:  "style",
	}

	result := buildHeaders(lh)

	assert.Equal(t, "</styles.css>; rel=preload; as=style", result)
}

func TestBuildHeaders_WithType(t *testing.T) {
	t.Parallel()

	lh := render_dto.LinkHeader{
		URL:  "/font.woff2",
		Rel:  "preload",
		As:   "font",
		Type: "font/woff2",
	}

	result := buildHeaders(lh)

	assert.Equal(t, `</font.woff2>; rel=preload; as=font; type="font/woff2"`, result)
}

func TestBuildHeaders_WithCrossOriginAnonymous(t *testing.T) {
	t.Parallel()

	lh := render_dto.LinkHeader{
		URL:         "/api/data",
		Rel:         "preconnect",
		CrossOrigin: "anonymous",
	}

	result := buildHeaders(lh)

	assert.Equal(t, "</api/data>; rel=preconnect; crossorigin=anonymous", result)
}

func TestBuildHeaders_WithCrossOriginUseCredentials(t *testing.T) {
	t.Parallel()

	lh := render_dto.LinkHeader{
		URL:         "/api/data",
		Rel:         "preconnect",
		CrossOrigin: "use-credentials",
	}

	result := buildHeaders(lh)

	assert.Equal(t, "</api/data>; rel=preconnect; crossorigin=use-credentials", result)
}

func TestBuildHeaders_WithCrossOriginOther(t *testing.T) {
	t.Parallel()

	lh := render_dto.LinkHeader{
		URL:         "/api/data",
		Rel:         "preconnect",
		CrossOrigin: "other",
	}

	result := buildHeaders(lh)

	assert.Equal(t, "</api/data>; rel=preconnect; crossorigin", result)
}

func TestDetermineCompression_ReturnsBrotli_WhenBrAccepted(t *testing.T) {
	t.Parallel()

	encoding, _ := determineCompression("br, gzip, deflate")

	assert.Equal(t, "br", encoding)
}

func TestDetermineCompression_ReturnsGzip_WhenOnlyGzipAccepted(t *testing.T) {
	t.Parallel()

	encoding, _ := determineCompression("gzip, deflate")

	assert.Equal(t, "gzip", encoding)
}

func TestDetermineCompression_ReturnsEmpty_WhenNoCompressionAccepted(t *testing.T) {
	t.Parallel()

	encoding, _ := determineCompression("deflate")

	assert.Empty(t, encoding)
}

func TestDetermineCompression_ReturnsEmpty_ForEmptyHeader(t *testing.T) {
	t.Parallel()

	encoding, _ := determineCompression("")

	assert.Empty(t, encoding)
}

func TestSelectBestVariantForRequest_ReturnsBrotli_WhenAccepted(t *testing.T) {
	t.Parallel()

	var brTags registry_dto.Tags
	brTags.SetByName("contentEncoding", "br")

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source"},
			{VariantID: "brotli", MetadataTags: brTags},
		},
	}

	result := selectBestVariantForRequest(artefact, "br, gzip")

	assert.NotNil(t, result.variant)
	assert.Equal(t, "brotli", result.variantName)
}

func TestSelectBestVariantForRequest_ReturnsGzip_WhenBrotliNotAvailable(t *testing.T) {
	t.Parallel()

	var gzipTags registry_dto.Tags
	gzipTags.SetByName("contentEncoding", "gzip")

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source"},
			{VariantID: "gzip_variant", MetadataTags: gzipTags},
		},
	}

	result := selectBestVariantForRequest(artefact, "br, gzip")

	assert.NotNil(t, result.variant)
	assert.Equal(t, "gzip", result.variantName)
}

func TestSelectBestVariantForRequest_ReturnsMinified_WhenNoCompression(t *testing.T) {
	t.Parallel()

	var minTags registry_dto.Tags
	minTags.SetByName("type", "minified-html")

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source"},
			{VariantID: "minified", MetadataTags: minTags},
		},
	}

	result := selectBestVariantForRequest(artefact, "")

	assert.NotNil(t, result.variant)
	assert.Equal(t, "minified-html", result.variantName)
}

func TestSelectBestVariantForRequest_ReturnsSource_AsFallback(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source"},
		},
	}

	result := selectBestVariantForRequest(artefact, "")

	assert.NotNil(t, result.variant)
	assert.Equal(t, "source", result.variantName)
}

func TestSelectBestVariantForRequest_ReturnsNil_WhenNoVariants(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{},
	}

	result := selectBestVariantForRequest(artefact, "br")

	assert.Nil(t, result.variant)
}

func TestFindVariantByTag_ReturnsVariant_WhenTagMatches(t *testing.T) {
	t.Parallel()

	var tags registry_dto.Tags
	tags.SetByName("type", "minified")

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source"},
			{VariantID: "minified", MetadataTags: tags},
		},
	}

	result := findVariantByTag(artefact, "type", "minified")

	require.NotNil(t, result)
	assert.Equal(t, "minified", result.VariantID)
}

func TestFindVariantByTag_ReturnsNil_WhenTagNotFound(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source"},
		},
	}

	result := findVariantByTag(artefact, "type", "minified")

	assert.Nil(t, result)
}

func TestFindVariantByTag_ReturnsNil_WhenTagValueDiffers(t *testing.T) {
	t.Parallel()

	var tags registry_dto.Tags
	tags.SetByName("type", "other")

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: "variant", MetadataTags: tags},
		},
	}

	result := findVariantByTag(artefact, "type", "minified")

	assert.Nil(t, result)
}

func TestNewPipeResponseWriter_CreatesWriter(t *testing.T) {
	t.Parallel()

	pr, pw := io.Pipe()
	defer func() { _ = pr.Close() }()
	defer func() { _ = pw.Close() }()

	writer := newPipeResponseWriter(pw)

	assert.NotNil(t, writer)
	assert.Equal(t, http.StatusOK, writer.statusCode)
	assert.NotNil(t, writer.header)
}

func TestPipeResponseWriter_Header_ReturnsHeader(t *testing.T) {
	t.Parallel()

	pr, pw := io.Pipe()
	defer func() { _ = pr.Close() }()
	defer func() { _ = pw.Close() }()

	writer := newPipeResponseWriter(pw)
	writer.Header().Set("X-Custom", "value")

	assert.Equal(t, "value", writer.Header().Get("X-Custom"))
}

func TestPipeResponseWriter_WriteHeader_SetsStatusCode(t *testing.T) {
	t.Parallel()

	pr, pw := io.Pipe()
	defer func() { _ = pr.Close() }()
	defer func() { _ = pw.Close() }()

	writer := newPipeResponseWriter(pw)
	writer.WriteHeader(http.StatusNotFound)

	assert.Equal(t, http.StatusNotFound, writer.statusCode)
}

func TestPipeResponseWriter_Write_WritesToPipe(t *testing.T) {
	t.Parallel()

	pr, pw := io.Pipe()
	writer := newPipeResponseWriter(pw)

	go func() {
		defer func() { _ = pw.Close() }()
		_, _ = writer.Write([]byte("test content"))
	}()

	buffer := make([]byte, 12)
	n, err := pr.Read(buffer)

	require.NoError(t, err)
	assert.Equal(t, 12, n)
	assert.Equal(t, "test content", string(buffer))
}

func TestCompressedResponseWriter_WritesToCompressor(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	buffer := &bytes.Buffer{}

	crw := &compressedResponseWriter{
		ResponseWriter: recorder,
		compressor:     nopWriteCloser{buffer},
	}

	n, err := crw.Write([]byte("test"))

	require.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, "test", buffer.String())
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

func TestStaticArtefactConfig_FieldsWork(t *testing.T) {
	t.Parallel()

	artefactConfig := staticArtefactConfig{
		artefactID:      "test.css",
		defaultMimeType: "text/css",
		cacheMaxAge:     "public, no-cache",
		preferredType:   "minified",
		useCompression:  true,
	}

	assert.Equal(t, "test.css", artefactConfig.artefactID)
	assert.Equal(t, "text/css", artefactConfig.defaultMimeType)
	assert.Equal(t, "public, no-cache", artefactConfig.cacheMaxAge)
	assert.Equal(t, "minified", artefactConfig.preferredType)
	assert.True(t, artefactConfig.useCompression)
}

func TestExtractOTelContext_ReturnsContext(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("traceparent", "00-abc123-def456-01")

	ctx := extractOTelContext(request)

	assert.NotNil(t, ctx)
}

func TestExtractOTelContext_HandlesMissingHeaders(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/", nil)

	ctx := extractOTelContext(request)

	assert.NotNil(t, ctx)
}

func TestExtractOTelContextFromRequest_ReturnsContext(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("traceparent", "00-abc123-def456-01")

	ctx, _ := extractOTelContextFromRequest(request)

	assert.NotNil(t, ctx)
}

func TestSendSpecificEarlyHints_ReturnsFalse_WhenNoHeaders(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	result := sendSpecificEarlyHints(recorder, request, []render_dto.LinkHeader{})

	assert.False(t, result)
}

func TestSendSpecificEarlyHints_AddsLinkHeaders(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	headers := []render_dto.LinkHeader{
		{URL: "/style.css", Rel: "preload", As: "style"},
	}

	_ = sendSpecificEarlyHints(recorder, request, headers)

	assert.Contains(t, recorder.Header().Get("Link"), "/style.css")
}

func TestTryGenerateVariantOnDemand_ReturnsNil_WhenGeneratorIsNil(t *testing.T) {
	t.Parallel()

	builder := &HTTPRouterBuilder{}
	artefact := &registry_dto.ArtefactMeta{ID: "test"}

	result := builder.tryGenerateVariantOnDemand(context.Background(), nil, artefact, "profile")

	assert.Nil(t, result)
}

func TestTryGenerateVariantOnDemand_ReturnsNil_WhenProfileInvalid(t *testing.T) {
	t.Parallel()

	builder := &HTTPRouterBuilder{}
	generator := &daemon_domain.MockOnDemandVariantGenerator{}
	artefact := &registry_dto.ArtefactMeta{ID: "test"}

	result := builder.tryGenerateVariantOnDemand(context.Background(), generator, artefact, "invalid")

	assert.Nil(t, result)
}

func TestTryGenerateVariantOnDemand_ReturnsNil_WhenGenerationFails(t *testing.T) {
	t.Parallel()

	builder := &HTTPRouterBuilder{}
	generator := &daemon_domain.MockOnDemandVariantGenerator{
		ParseProfileNameFunc: func(_ string) *daemon_domain.ParsedImageProfile {
			return &daemon_domain.ParsedImageProfile{}
		},
		GenerateVariantFunc: func(_ context.Context, _ *registry_dto.ArtefactMeta, _ string) (*registry_dto.Variant, error) {
			return nil, errors.New("generation failed")
		},
	}
	artefact := &registry_dto.ArtefactMeta{ID: "test"}

	result := builder.tryGenerateVariantOnDemand(context.Background(), generator, artefact, "profile")

	assert.Nil(t, result)
}

func TestTryGenerateVariantOnDemand_ReturnsVariant_WhenSuccess(t *testing.T) {
	t.Parallel()

	builder := &HTTPRouterBuilder{}
	expectedVariant := &registry_dto.Variant{VariantID: "generated"}
	generator := &daemon_domain.MockOnDemandVariantGenerator{
		ParseProfileNameFunc: func(_ string) *daemon_domain.ParsedImageProfile {
			return &daemon_domain.ParsedImageProfile{}
		},
		GenerateVariantFunc: func(_ context.Context, _ *registry_dto.ArtefactMeta, _ string) (*registry_dto.Variant, error) {
			return expectedVariant, nil
		},
	}
	artefact := &registry_dto.ArtefactMeta{ID: "test"}

	result := builder.tryGenerateVariantOnDemand(context.Background(), generator, artefact, "profile")

	assert.Equal(t, expectedVariant, result)
}

func TestSelectStaticVariant_WithCompression_ReturnsBestCompressed(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source"},
			{VariantID: "source_br"},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "br")
	artefactConfig := staticArtefactConfig{
		preferredType:  "source",
		useCompression: true,
	}

	result := selectStaticVariant(request, artefact, artefactConfig)

	require.NotNil(t, result)
	assert.Equal(t, "source_br", result.VariantID)
}

func TestSelectStaticVariant_NoCompression_ReturnsPreferred(t *testing.T) {
	t.Parallel()

	var minTags registry_dto.Tags
	minTags.SetByName("type", "minified")

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source"},
			{VariantID: "minified", MetadataTags: minTags},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	artefactConfig := staticArtefactConfig{
		preferredType:  "minified",
		useCompression: false,
	}

	result := selectStaticVariant(request, artefact, artefactConfig)

	require.NotNil(t, result)
	assert.Equal(t, "minified", result.VariantID)
}

func TestSelectStaticVariant_FallsBackToSource(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source"},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	artefactConfig := staticArtefactConfig{
		preferredType:  "nonexistent",
		useCompression: false,
	}

	result := selectStaticVariant(request, artefact, artefactConfig)

	require.NotNil(t, result)
	assert.Equal(t, "source", result.VariantID)
}

func TestCacheErrors_HaveExpectedMessages(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "handler returned empty body on 200 OK", errEmptyBody.Error())
	assert.Equal(t, "upstream handler returned non-200 status code", errHandlerNonSuccess.Error())
}

func TestJitResult_FieldsWork(t *testing.T) {
	t.Parallel()

	result := jitResult{
		Content:      []byte("test content"),
		Encoding:     "br",
		ETag:         `"test-etag"`,
		CacheControl: "public, max-age=3600",
		StatusCode:   http.StatusOK,
	}

	assert.Equal(t, []byte("test content"), result.Content)
	assert.Equal(t, "br", result.Encoding)
	assert.Equal(t, `"test-etag"`, result.ETag)
	assert.Equal(t, "public, max-age=3600", result.CacheControl)
	assert.Equal(t, http.StatusOK, result.StatusCode)
}

func TestArtefactLookupResult_FieldsWork(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{ID: "test"}
	result := artefactLookupResult{
		artefact:          artefact,
		err:               nil,
		httpStatus:        http.StatusOK,
		foundByStorageKey: true,
	}

	assert.Equal(t, artefact, result.artefact)
	assert.Nil(t, result.err)
	assert.Equal(t, http.StatusOK, result.httpStatus)
	assert.True(t, result.foundByStorageKey)
}

func TestMountRoutesConfig_FieldsWork(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	deps := &daemon_domain.HTTPHandlerDependencies{}
	store := &templater_domain.MockManifestStoreView{}
	csrf := &security_domain.MockCSRFTokenService{}
	siteSettings := &config.WebsiteConfig{}
	actions := map[string]ActionHandlerEntry{}

	mountConfig := MountRoutesConfig{
		Router:        router,
		Deps:          deps,
		Store:         store,
		CSRFService:   csrf,
		RouteSettings: RouteSettings{},
		SiteSettings:  siteSettings,
		Actions:       actions,
	}

	assert.NotNil(t, mountConfig.Router)
	assert.NotNil(t, mountConfig.Deps)
	assert.NotNil(t, mountConfig.Store)
	assert.NotNil(t, mountConfig.CSRFService)
	assert.NotNil(t, mountConfig.SiteSettings)
	assert.NotNil(t, mountConfig.Actions)
}

func TestCacheMiddlewareConfig_FieldsWork(t *testing.T) {
	t.Parallel()

	cacheMiddlewareConfig := CacheMiddlewareConfig{
		StreamCompressionLevel: 6,
		CacheWriteConcurrency:  10,
	}

	assert.Equal(t, 6, cacheMiddlewareConfig.StreamCompressionLevel)
	assert.Equal(t, 10, cacheMiddlewareConfig.CacheWriteConcurrency)
}

func TestVariantSelectionResult_FieldsWork(t *testing.T) {
	t.Parallel()

	variant := &registry_dto.Variant{VariantID: "test"}
	result := variantSelectionResult{
		variant:     variant,
		variantName: "test-variant",
	}

	assert.Equal(t, variant, result.variant)
	assert.Equal(t, "test-variant", result.variantName)
}

func TestRouteRegistrationDeps_FieldsWork(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	deps := &daemon_domain.HTTPHandlerDependencies{}
	store := &templater_domain.MockManifestStoreView{}
	siteSettings := &config.WebsiteConfig{}

	regDeps := routeRegistrationDeps{
		router:       router,
		deps:         deps,
		store:        store,
		siteSettings: siteSettings,
	}

	assert.NotNil(t, regDeps.router)
	assert.NotNil(t, regDeps.deps)
	assert.NotNil(t, regDeps.store)
	assert.NotNil(t, regDeps.siteSettings)
}

func TestNewCacheMiddleware_CreatesMiddleware(t *testing.T) {
	t.Parallel()

	artefactConfig := CacheMiddlewareConfig{
		StreamCompressionLevel: 4,
		CacheWriteConcurrency:  5,
	}
	manifest := &templater_domain.MockManifestStoreView{}
	registry := &registry_domain.MockRegistryService{}
	partialServePath := "/partials"

	mw := NewCacheMiddleware(artefactConfig, manifest, registry, nil, partialServePath)

	assert.NotNil(t, mw)
	assert.Equal(t, registry, mw.registryService)
	assert.Equal(t, manifest, mw.manifest)
}

func TestNewCacheMiddleware_SetsDefaults_WhenConcurrencyZero(t *testing.T) {
	t.Parallel()

	artefactConfig := CacheMiddlewareConfig{
		CacheWriteConcurrency: 0,
	}
	manifest := &templater_domain.MockManifestStoreView{}
	registry := &registry_domain.MockRegistryService{}
	mw := NewCacheMiddleware(artefactConfig, manifest, registry, nil, "")

	assert.NotNil(t, mw)
	assert.Equal(t, defaultCacheWriteConcurrency, mw.config.CacheWriteConcurrency)
}

func TestPageCacheProfiles_ReturnsTwoProfiles(t *testing.T) {
	t.Parallel()

	profiles := pageCacheProfiles

	assert.Len(t, profiles, 2)
	assert.Equal(t, "brotli_variant", profiles[0].Name)
	assert.Equal(t, "gzip_variant", profiles[1].Name)
}

func TestPipeResponseWriter_Flush_DoesNotPanic(t *testing.T) {
	t.Parallel()

	pr, pw := io.Pipe()
	defer func() { _ = pr.Close() }()
	defer func() { _ = pw.Close() }()

	writer := newPipeResponseWriter(pw)

	assert.NotPanics(t, func() {
		writer.Flush()
	})
}

func TestContextKey_TypeWorks(t *testing.T) {
	t.Parallel()

	var key contextKey = "test_key"

	assert.Equal(t, contextKey("test_key"), key)
	assert.Equal(t, "test_key", string(key))
}

func TestPikoRequestCtx_CanBeStoredInContext(t *testing.T) {
	t.Parallel()

	pctx := &daemon_dto.PikoRequestCtx{
		Locale:         "en",
		MatchedPattern: "/test/{id}",
	}
	ctx := daemon_dto.WithPikoRequestCtx(context.Background(), pctx)

	got := daemon_dto.PikoRequestCtxFromContext(ctx)
	assert.NotNil(t, got)
	assert.Equal(t, "en", got.Locale)
	assert.Equal(t, "/test/{id}", got.MatchedPattern)
}

func TestReleaseCompressor_HandlesNilGracefully(t *testing.T) {

	t.Parallel()

	_ = releaseCompressor
}

func TestServeSitemapChunk_CreatesCorrectArtefactID(t *testing.T) {
	t.Parallel()

	builder := &HTTPRouterBuilder{}

	assert.NotNil(t, builder)
}

func TestBrotliWriterPool_ReturnsWriter(t *testing.T) {
	t.Parallel()

	writer := brotliWriterPool.Get()
	assert.NotNil(t, writer)
	brotliWriterPool.Put(writer)
}

func TestGzipWriterPool_ReturnsWriter(t *testing.T) {
	t.Parallel()

	writer := gzipWriterPool.Get()
	assert.NotNil(t, writer)
	gzipWriterPool.Put(writer)
}

type noopSpan struct {
	trace.Span
	statusDesc    string
	attributesSet []attribute.KeyValue
	statusCode    codes.Code
	statusSet     bool
	errorRecorded bool
}

func (n *noopSpan) End(...trace.SpanEndOption) {}

func (n *noopSpan) SetStatus(code codes.Code, description string) {
	n.statusSet = true
	n.statusCode = code
	n.statusDesc = description
}

func (n *noopSpan) RecordError(_ error, _ ...trace.EventOption) {
	n.errorRecorded = true
}

func (n *noopSpan) SetAttributes(kv ...attribute.KeyValue) {
	n.attributesSet = append(n.attributesSet, kv...)
}

func (n *noopSpan) SpanContext() trace.SpanContext {
	return trace.SpanContext{}
}

func (n *noopSpan) IsRecording() bool {
	return true
}

func newNoopSpan() *noopSpan {
	return &noopSpan{attributesSet: make([]attribute.KeyValue, 0)}
}

func TestFetchStaticArtefact_ReturnsArtefact_WhenFound(t *testing.T) {
	t.Parallel()

	expectedArtefact := &registry_dto.ArtefactMeta{
		ID: "test-artefact-123",
	}
	registryService := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
			if artefactID == "test-artefact-123" {
				return expectedArtefact, nil
			}
			return nil, errors.New("not found")
		},
	}
	span := newNoopSpan()

	artefact, ok := fetchStaticArtefact(context.Background(), registryService, "test-artefact-123", span)

	assert.True(t, ok, "Should return true when artefact found")
	assert.NotNil(t, artefact, "Should return artefact")
	assert.Equal(t, "test-artefact-123", artefact.ID)
}

func TestFetchStaticArtefact_ReturnsFalse_WhenArtefactNotFound(t *testing.T) {
	t.Parallel()

	registryService := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, registry_domain.ErrArtefactNotFound
		},
	}
	span := newNoopSpan()

	artefact, ok := fetchStaticArtefact(context.Background(), registryService, "missing-artefact", span)

	assert.False(t, ok, "Should return false when artefact not found")
	assert.Nil(t, artefact, "Should return nil artefact")
}

func TestFetchStaticArtefact_ReturnsFalse_OnOtherError(t *testing.T) {
	t.Parallel()

	registryService := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, errors.New("database connection failed")
		},
	}
	span := newNoopSpan()

	artefact, ok := fetchStaticArtefact(context.Background(), registryService, "some-artefact", span)

	assert.False(t, ok, "Should return false on error")
	assert.Nil(t, artefact, "Should return nil artefact on error")
}

func TestGenerateBackgroundVariant_CallsGenerator(t *testing.T) {
	t.Parallel()

	mockGen := &daemon_domain.MockOnDemandVariantGenerator{
		GenerateVariantFunc: func(_ context.Context, _ *registry_dto.ArtefactMeta, _ string) (*registry_dto.Variant, error) {
			return &registry_dto.Variant{VariantID: "generated-variant"}, nil
		},
	}
	artefact := &registry_dto.ArtefactMeta{ID: "test-artefact"}

	generateBackgroundVariant(context.Background(), mockGen, artefact, "test_profile")

	assert.Equal(t, int64(1), atomic.LoadInt64(&mockGen.GenerateVariantCallCount), "Should call generator once")
}

func TestGenerateBackgroundVariant_LogsWarning_OnError(t *testing.T) {
	t.Parallel()

	mockGen := &daemon_domain.MockOnDemandVariantGenerator{
		GenerateVariantFunc: func(_ context.Context, _ *registry_dto.ArtefactMeta, _ string) (*registry_dto.Variant, error) {
			return nil, errors.New("generation failed")
		},
	}
	artefact := &registry_dto.ArtefactMeta{ID: "test-artefact"}

	generateBackgroundVariant(context.Background(), mockGen, artefact, "test_profile")

	assert.Equal(t, int64(1), atomic.LoadInt64(&mockGen.GenerateVariantCallCount), "Should call generator once")
}

func TestSetupBrotliCompressor_ReturnsWriter(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()

	compressor, ok := setupBrotliCompressor(context.Background(), recorder)

	assert.True(t, ok, "Should return true on success")
	assert.NotNil(t, compressor, "Should return compressor")
	assert.Equal(t, encodingBrotli, recorder.Header().Get(headerContentEncoding), "Should set Content-Encoding header")

	if compressor != nil {
		_ = compressor.Close()
	}
}

func TestSetupGzipCompressor_ReturnsWriter(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()

	compressor, ok := setupGzipCompressor(context.Background(), recorder)

	assert.True(t, ok, "Should return true on success")
	assert.NotNil(t, compressor, "Should return compressor")
	assert.Equal(t, encodingGzip, recorder.Header().Get(headerContentEncoding), "Should set Content-Encoding header")

	if compressor != nil {
		_ = compressor.Close()
	}
}

func TestReleaseCompressor_ReleasesBrotli(t *testing.T) {
	t.Parallel()

	bw, ok := brotliWriterPool.Get().(*brotli.Writer)
	require.True(t, ok, "brotliWriterPool.Get() should return *brotli.Writer")
	buffer := &bytes.Buffer{}
	bw.Reset(buffer)

	_, err := bw.Write([]byte("test data"))
	require.NoError(t, err)

	releaseCompressor(bw, true)

	bw2 := brotliWriterPool.Get()
	assert.NotNil(t, bw2, "Should be able to get writer from pool after release")
	brotliWriterPool.Put(bw2)
}

func TestReleaseCompressor_ReleasesGzip(t *testing.T) {
	t.Parallel()

	gw, ok := gzipWriterPool.Get().(*gzip.Writer)
	require.True(t, ok, "gzipWriterPool.Get() should return *gzip.Writer")
	buffer := &bytes.Buffer{}
	gw.Reset(buffer)

	_, err := gw.Write([]byte("test data"))
	require.NoError(t, err)

	releaseCompressor(gw, false)

	gw2 := gzipWriterPool.Get()
	assert.NotNil(t, gw2, "Should be able to get writer from pool after release")
	gzipWriterPool.Put(gw2)
}

func TestSelectStaticVariant_ReturnsCompressedVariant_WhenCompressionEnabled(t *testing.T) {
	t.Parallel()

	var sourceTags registry_dto.Tags
	sourceTags.SetByName("type", "source")
	var brTags registry_dto.Tags
	brTags.SetByName("contentEncoding", "br")

	artefact := &registry_dto.ArtefactMeta{
		ID: "test-artefact",
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source", MetadataTags: sourceTags},
			{VariantID: "br", MetadataTags: brTags},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	request.Header.Set("Accept-Encoding", "br, gzip")
	artefactConfig := staticArtefactConfig{
		useCompression: true,
		preferredType:  "",
	}

	variant := selectStaticVariant(request, artefact, artefactConfig)

	assert.NotNil(t, variant)
}

func TestSelectStaticVariant_ReturnsSourceVariant_WhenNoCompression(t *testing.T) {
	t.Parallel()

	var sourceTags registry_dto.Tags
	sourceTags.SetByName("type", "source")

	artefact := &registry_dto.ArtefactMeta{
		ID: "test-artefact",
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source", MetadataTags: sourceTags},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	artefactConfig := staticArtefactConfig{
		useCompression: false,
		preferredType:  "",
	}

	variant := selectStaticVariant(request, artefact, artefactConfig)

	require.NotNil(t, variant)
	assert.Equal(t, "source", variant.VariantID)
}

func TestSelectStaticVariant_ReturnsPreferredType_WhenSpecified(t *testing.T) {
	t.Parallel()

	var sourceTags registry_dto.Tags
	sourceTags.SetByName("type", "source")
	var webpTags registry_dto.Tags
	webpTags.SetByName("type", "webp")

	artefact := &registry_dto.ArtefactMeta{
		ID: "test-artefact",
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source", MetadataTags: sourceTags},
			{VariantID: "webp", MetadataTags: webpTags},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	artefactConfig := staticArtefactConfig{
		useCompression: false,
		preferredType:  "webp",
	}

	variant := selectStaticVariant(request, artefact, artefactConfig)

	require.NotNil(t, variant)
	assert.Equal(t, "webp", variant.VariantID)
}

func TestRegisterRoutesFromStore_ReturnsZeroCounts_ForEmptyStore(t *testing.T) {
	t.Parallel()

	store := &templater_domain.MockManifestStoreView{}
	deps := &routeRegistrationDeps{}

	pageCount, partialCount := registerRoutesFromStore(context.Background(), deps, store)

	assert.Equal(t, 0, pageCount)
	assert.Equal(t, 0, partialCount)
}

func TestSendSpecificEarlyHints_WritesLinkHeader_ForSingleHint(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	hints := []render_dto.LinkHeader{
		{
			URL: "/styles.css",
			Rel: "preload",
			As:  "style",
		},
	}

	sendSpecificEarlyHints(recorder, request, hints)

	linkHeader := recorder.Header().Get("Link")
	assert.Contains(t, linkHeader, "</styles.css>")
	assert.Contains(t, linkHeader, "rel=preload")
	assert.Contains(t, linkHeader, "as=style")
}

func TestSendSpecificEarlyHints_WritesMultipleHeaders_ForMultipleHints(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	hints := []render_dto.LinkHeader{
		{URL: "/styles.css", Rel: "preload", As: "style"},
		{URL: "/script.js", Rel: "preload", As: "script"},
	}

	sendSpecificEarlyHints(recorder, request, hints)

	linkHeaders := recorder.Header().Values("Link")
	assert.Len(t, linkHeaders, 2)
}

func TestSendSpecificEarlyHints_ReturnsFalse_ForEmptyHints(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	hints := []render_dto.LinkHeader{}

	result := sendSpecificEarlyHints(recorder, request, hints)

	assert.False(t, result, "Should return false for empty hints")
}

func TestParseFragmentParam_ReturnsTrue_When1(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/test?_f=1", nil)

	result := parseFragmentParam(request)

	assert.True(t, result)
}

func TestParseFragmentParam_ReturnsFalse_WhenNotPresent(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/test", nil)

	result := parseFragmentParam(request)

	assert.False(t, result)
}

func TestFindBestCompressedVariant_ReturnsBrotli_WhenBothAvailable(t *testing.T) {
	t.Parallel()

	var brTags registry_dto.Tags
	brTags.SetByName("contentEncoding", "br")
	var gzTags registry_dto.Tags
	gzTags.SetByName("contentEncoding", "gzip")

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source"},
			{VariantID: "brotli", MetadataTags: brTags},
			{VariantID: "gzip", MetadataTags: gzTags},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	request.Header.Set("Accept-Encoding", "br, gzip")

	result := findBestCompressedVariant(request, artefact, "")

	require.NotNil(t, result)
	assert.Equal(t, "brotli", result.VariantID)
}

func TestGenerateCacheArtefactID_ReturnsConsistentID(t *testing.T) {
	t.Parallel()

	req1 := httptest.NewRequest(http.MethodGet, "/path", nil)
	req2 := httptest.NewRequest(http.MethodGet, "/path", nil)
	policy := templater_dto.CachePolicy{Key: "key1"}

	id1 := generateCacheArtefactID(req1, policy)
	id2 := generateCacheArtefactID(req2, policy)

	assert.Equal(t, id1, id2, "Same inputs should produce same ID")
}

func TestGenerateCacheArtefactID_ReturnsDifferentIDs_ForDifferentPaths(t *testing.T) {
	t.Parallel()

	req1 := httptest.NewRequest(http.MethodGet, "/path1", nil)
	req2 := httptest.NewRequest(http.MethodGet, "/path2", nil)
	policy := templater_dto.CachePolicy{Key: "key1"}

	id1 := generateCacheArtefactID(req1, policy)
	id2 := generateCacheArtefactID(req2, policy)

	assert.NotEqual(t, id1, id2, "Different paths should produce different IDs")
}

func TestGenerateCacheArtefactID_ReturnsDifferentIDs_ForDifferentKeys(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/path", nil)
	policy1 := templater_dto.CachePolicy{Key: "key1"}
	policy2 := templater_dto.CachePolicy{Key: "key2"}

	id1 := generateCacheArtefactID(request, policy1)
	id2 := generateCacheArtefactID(request, policy2)

	assert.NotEqual(t, id1, id2, "Different keys should produce different IDs")
}

func TestExtractOTelContextFromRequest_ReturnsContextFromHeader(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/test", nil)

	request.Header.Set("traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")

	ctx, _ := extractOTelContextFromRequest(request)

	assert.NotNil(t, ctx)
}

func TestExtractOTelContextFromRequest_ReturnsContextWithoutHeader(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/test", nil)

	ctx, _ := extractOTelContextFromRequest(request)

	assert.NotNil(t, ctx)
}
