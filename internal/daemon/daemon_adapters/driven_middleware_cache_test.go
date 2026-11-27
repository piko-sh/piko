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
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

type persistedArtefact struct {
	cause      error
	carrier    *daemon_dto.PikoRequestCtx
	artefactID string
	body       string
}

func newTestCacheMiddleware(
	registry registry_domain.RegistryService,
	capabilities capabilities_domain.CapabilityService,
) *CacheMiddleware {
	return NewCacheMiddleware(CacheMiddlewareConfig{}, &templater_domain.MockManifestStoreView{}, registry, capabilities, "")
}

func newCarrierRequest(t *testing.T, acceptEncoding string) (*http.Request, *daemon_dto.PikoRequestCtx) {
	t.Helper()

	pctx := daemon_dto.AcquirePikoRequestCtx()
	pctx.ClientIP = "10.0.0.1"
	request := httptest.NewRequest(http.MethodGet, "/blog/post", nil)
	request.Header.Set("Accept-Encoding", acceptEncoding)

	return request.WithContext(daemon_dto.WithPikoRequestCtx(request.Context(), pctx)), pctx
}

func TestGenerateAndCacheResponse_UpstreamFinishesBeforeReturning(t *testing.T) {
	compressionFailure := errors.New("compressor unavailable")

	testCases := []struct {
		capabilities   capabilities_domain.CapabilityService
		upstream       func(finished chan<- struct{}) http.HandlerFunc
		wantErr        error
		name           string
		acceptEncoding string
	}{
		{
			name:           "compression fails while the upstream is still writing",
			acceptEncoding: "gzip",
			capabilities: &capabilities_domain.MockCapabilityService{
				ExecuteFunc: func(context.Context, string, io.Reader, capabilities_domain.CapabilityParams) (io.Reader, error) {
					return nil, compressionFailure
				},
			},
			upstream: func(finished chan<- struct{}) http.HandlerFunc {
				return func(w http.ResponseWriter, _ *http.Request) {
					defer close(finished)
					chunk := bytes.Repeat([]byte("x"), 4096)
					for range 64 {
						if _, err := w.Write(chunk); err != nil {
							return
						}
					}
				}
			},
			wantErr: compressionFailure,
		},
		{
			name:           "upstream panics",
			acceptEncoding: "",
			upstream: func(finished chan<- struct{}) http.HandlerFunc {
				return func(http.ResponseWriter, *http.Request) {
					defer close(finished)
					panic("render failed")
				}
			},
			wantErr: errHandlerNonSuccess,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			upstreamFinished := make(chan struct{})
			middleware := newTestCacheMiddleware(&registry_domain.MockRegistryService{}, tc.capabilities)
			request, pctx := newCarrierRequest(t, tc.acceptEncoding)
			defer daemon_dto.ReleasePikoRequestCtx(pctx)

			result, err := middleware.generateAndCacheResponse(request, tc.upstream(upstreamFinished), "artefact-1")

			require.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, result)
			select {
			case <-upstreamFinished:
			default:
				assert.Fail(t, "the upstream handler was still running after the response was abandoned")
			}
		})
	}
}

func TestGenerateAndCacheResponse_PersistsUnderADetachedContext(t *testing.T) {
	persisted := make(chan persistedArtefact, 1)
	requestFinished := make(chan struct{})
	registry := &registry_domain.MockRegistryService{
		UpsertArtefactFunc: func(
			ctx context.Context, artefactID, _ string, sourceData io.Reader, _ string, _ []registry_dto.NamedProfile,
		) (*registry_dto.ArtefactMeta, error) {
			<-requestFinished
			body, err := io.ReadAll(sourceData)
			if err != nil {
				return nil, err
			}
			persisted <- persistedArtefact{
				cause:      context.Cause(ctx),
				carrier:    daemon_dto.PikoRequestCtxFromContext(ctx),
				artefactID: artefactID,
				body:       string(body),
			}
			return &registry_dto.ArtefactMeta{}, nil
		},
	}
	middleware := newTestCacheMiddleware(registry, nil)

	request, pctx := newCarrierRequest(t, "")
	requestCtx, cancelRequest := context.WithCancelCause(request.Context())
	request = request.WithContext(requestCtx)
	upstream := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<p>rendered</p>"))
	})

	result, err := middleware.generateAndCacheResponse(request, upstream, "artefact-1")
	require.NoError(t, err)
	assert.Equal(t, "<p>rendered</p>", string(result.Content))

	cancelRequest(errors.New("client went away"))
	daemon_dto.ReleasePikoRequestCtx(pctx)
	next := daemon_dto.AcquirePikoRequestCtx()
	next.ClientIP = "192.168.9.9"
	defer daemon_dto.ReleasePikoRequestCtx(next)
	close(requestFinished)

	got := <-persisted
	assert.Equal(t, "artefact-1", got.artefactID)
	assert.Equal(t, "<p>rendered</p>", got.body)
	require.NoError(t, got.cause, "persistence must outlive the request")
	require.NotNil(t, got.carrier)
	assert.NotSame(t, pctx, got.carrier, "persistence must not read the pooled carrier")
	assert.Equal(t, "10.0.0.1", got.carrier.ClientIP)
}

const (
	staticTestRoute         = "/pricing"
	staticTestBody          = "<h1>Pricing</h1>"
	staticTestETag          = `"pricing-v1"`
	staticTestMaxAgeSeconds = 3600
	staticTestPolicy        = "public, max-age=0, s-maxage=3600, must-revalidate"
	staticTestPagePolicy    = "public, max-age=60"
)

func cachedStaticPageRegistry() *registry_domain.MockRegistryService {
	var tags registry_dto.Tags
	tags.Set(registry_dto.TagEtag, staticTestETag)
	variant := registry_dto.Variant{VariantID: variantSource, MetadataTags: tags, Status: registry_dto.VariantStatusReady}

	return &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID:             artefactID,
				UpdatedAt:      time.Now(),
				ActualVariants: []registry_dto.Variant{variant},
				Status:         registry_dto.VariantStatusReady,
			}, nil
		},
		GetVariantDataFunc: func(context.Context, *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(staticTestBody)), nil
		},
	}
}

func emptyStaticPageRegistry() *registry_domain.MockRegistryService {
	return &registry_domain.MockRegistryService{
		GetArtefactFunc: func(context.Context, string) (*registry_dto.ArtefactMeta, error) {
			return nil, errors.New("not cached")
		},
	}
}

type cachedPageRouteOptions struct {
	registry      registry_domain.RegistryService
	capabilities  capabilities_domain.CapabilityService
	appMiddleware func(http.Handler) http.Handler
	page          http.Handler
	policy        templater_dto.CachePolicy
	persist       bool
}

func renderStaticTestPage(w http.ResponseWriter, _ *http.Request) {
	setDefaultCacheControl(w.Header(), headerValCachePrivateNoCache)
	w.Header()[headerXPPResponseSupport] = headerValFragmentPatch
	_, _ = w.Write([]byte(staticTestBody))
}

func newCachedPageRouter(t *testing.T, options cachedPageRouteOptions) *chi.Mux {
	t.Helper()

	entry := &templater_domain.MockPageEntryView{
		GetIsPageFunc:       func() bool { return true },
		GetRoutePatternFunc: func() string { return staticTestRoute },
		GetCachePolicyFunc: func(*templater_dto.RequestData) templater_dto.CachePolicy {
			return options.policy
		},
	}
	store := &templater_domain.MockManifestStoreView{
		GetKeysFunc: func() []string { return []string{"pages/pricing.pk"} },
		GetPageEntryFunc: func(string) (templater_domain.PageEntryView, bool) {
			return entry, true
		},
	}
	config := CacheMiddlewareConfig{PersistenceDisabled: !options.persist}
	middleware := NewCacheMiddleware(config, store, options.registry, options.capabilities, "")

	handler := middleware.Handle(options.page)
	if options.appMiddleware != nil {
		handler = options.appMiddleware(handler)
	}

	router := chi.NewRouter()
	router.Handle(staticTestRoute, handler)
	return router
}

func newStaticPageRouter(
	t *testing.T, registry registry_domain.RegistryService, pageMiddleware func(http.Handler) http.Handler,
) *chi.Mux {
	t.Helper()

	return newCachedPageRouter(t, cachedPageRouteOptions{
		registry:      registry,
		appMiddleware: pageMiddleware,
		page:          http.HandlerFunc(renderStaticTestPage),
		policy:        templater_dto.CachePolicy{Enabled: true, Static: true, MaxAgeSeconds: staticTestMaxAgeSeconds},
	})
}

func setsResponseHeader(key, value string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			w.Header().Set(key, value)
			next.ServeHTTP(w, request)
		})
	}
}

func withLocale(ctx context.Context, locale string) context.Context {
	if locale == "" {
		return ctx
	}
	return daemon_dto.WithPikoRequestCtx(ctx, &daemon_dto.PikoRequestCtx{Locale: locale})
}

func serveStaticTestRequest(t *testing.T, router http.Handler, configure func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, staticTestRoute, nil)
	if configure != nil {
		configure(request)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestCacheMiddlewareStaticPageHeaders(t *testing.T) {
	t.Parallel()

	jitETag := formatJITETag(xxhash.Sum64String(staticTestBody))
	setsPagePolicy := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			w.Header().Set("Cache-Control", staticTestPagePolicy)
			next.ServeHTTP(w, request)
		})
	}

	testCases := []struct {
		registry         registry_domain.RegistryService
		pageMiddleware   func(http.Handler) http.Handler
		name             string
		ifNoneMatch      string
		wantCacheControl string
		wantETag         string
		wantBody         string
		wantStatus       int
	}{
		{
			name:             "cache hit lets only shared caches keep the page",
			registry:         cachedStaticPageRegistry(),
			wantStatus:       http.StatusOK,
			wantCacheControl: staticTestPolicy,
			wantETag:         staticTestETag,
			wantBody:         staticTestBody,
		},
		{
			name:             "cache miss lets only shared caches keep the page",
			registry:         emptyStaticPageRegistry(),
			wantStatus:       http.StatusOK,
			wantCacheControl: staticTestPolicy,
			wantETag:         jitETag,
			wantBody:         staticTestBody,
		},
		{
			name:             "revalidated cache hit keeps its headers",
			registry:         cachedStaticPageRegistry(),
			ifNoneMatch:      staticTestETag,
			wantStatus:       http.StatusNotModified,
			wantCacheControl: staticTestPolicy,
			wantETag:         staticTestETag,
		},
		{
			name:             "cache hit revalidated with a weak tag keeps its headers",
			registry:         cachedStaticPageRegistry(),
			ifNoneMatch:      `W/"pricing-v1"`,
			wantStatus:       http.StatusNotModified,
			wantCacheControl: staticTestPolicy,
			wantETag:         staticTestETag,
		},
		{
			name:             "revalidated cache miss keeps its headers",
			registry:         emptyStaticPageRegistry(),
			ifNoneMatch:      jitETag,
			wantStatus:       http.StatusNotModified,
			wantCacheControl: staticTestPolicy,
			wantETag:         jitETag,
		},
		{
			name:             "page policy wins on a cache hit",
			registry:         cachedStaticPageRegistry(),
			pageMiddleware:   setsPagePolicy,
			wantStatus:       http.StatusOK,
			wantCacheControl: staticTestPagePolicy,
			wantETag:         staticTestETag,
			wantBody:         staticTestBody,
		},
		{
			name:             "page policy wins on a cache miss",
			registry:         emptyStaticPageRegistry(),
			pageMiddleware:   setsPagePolicy,
			wantStatus:       http.StatusOK,
			wantCacheControl: staticTestPagePolicy,
			wantETag:         jitETag,
			wantBody:         staticTestBody,
		},
		{
			name:             "page policy wins on a revalidated cache hit",
			registry:         cachedStaticPageRegistry(),
			pageMiddleware:   setsPagePolicy,
			ifNoneMatch:      staticTestETag,
			wantStatus:       http.StatusNotModified,
			wantCacheControl: staticTestPagePolicy,
			wantETag:         staticTestETag,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			router := newStaticPageRouter(t, testCase.registry, testCase.pageMiddleware)
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, staticTestRoute, nil)
			if testCase.ifNoneMatch != "" {
				request.Header.Set("If-None-Match", testCase.ifNoneMatch)
			}
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			assert.Equal(t, testCase.wantStatus, recorder.Code)
			assert.Equal(t, []string{testCase.wantCacheControl}, recorder.Header().Values("Cache-Control"))
			assert.Equal(t, testCase.wantETag, recorder.Header().Get("ETag"))
			assert.Equal(t, "Accept-Encoding", recorder.Header().Get("Vary"))
			assert.Equal(t, testCase.wantBody, recorder.Body.String())
		})
	}
}

func TestStaticPageCacheControl(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		want          string
		maxAgeSeconds int
	}{
		{name: "common age", maxAgeSeconds: 3600, want: "public, max-age=0, s-maxage=3600, must-revalidate"},
		{name: "uncommon age", maxAgeSeconds: 12345, want: "public, max-age=0, s-maxage=12345, must-revalidate"},
		{name: "zero", maxAgeSeconds: 0, want: "public, max-age=0, s-maxage=0, must-revalidate"},
		{name: "negative age counts as zero", maxAgeSeconds: -5, want: "public, max-age=0, s-maxage=0, must-revalidate"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, staticPageCacheControl(testCase.maxAgeSeconds))
		})
	}
}

func TestStaticPageVary(t *testing.T) {
	t.Parallel()

	localised := daemon_dto.AcquirePikoRequestCtx()
	localised.Locale = "fr"
	t.Cleanup(func() { daemon_dto.ReleasePikoRequestCtx(localised) })

	assert.Equal(t, []string{"Accept-Encoding"}, staticPageVary(t.Context()))
	assert.Equal(t, []string{"Accept-Encoding, Accept-Language"},
		staticPageVary(daemon_dto.WithPikoRequestCtx(t.Context(), localised)))
}

func TestCacheMiddlewareForwardsStaticPageRedirect(t *testing.T) {
	t.Parallel()

	redirectsTo := func(target string, status int, vary string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			setDefaultCacheControl(w.Header(), headerValCachePrivateNoCache)
			if vary != "" {
				w.Header().Set("Vary", vary)
			}
			w.Header().Set("Location", target)
			w.WriteHeader(status)
		})
	}

	testCases := []struct {
		appMiddleware    func(http.Handler) http.Handler
		name             string
		wantLocation     string
		wantCacheControl string
		wantVary         string
		pageVary         string
		status           int
	}{
		{
			name:             "temporary redirect",
			status:           http.StatusFound,
			wantLocation:     "/login",
			wantCacheControl: cacheControlPrivateNoCache,
		},
		{
			name:             "permanent redirect",
			status:           http.StatusMovedPermanently,
			wantLocation:     "/login",
			wantCacheControl: cacheControlPrivateNoCache,
		},
		{
			name:             "see other",
			status:           http.StatusSeeOther,
			wantLocation:     "/login",
			wantCacheControl: cacheControlPrivateNoCache,
		},
		{
			name:             "app Cache-Control wins over the page's",
			appMiddleware:    setsResponseHeader("Cache-Control", "no-store"),
			status:           http.StatusFound,
			wantLocation:     "/login",
			wantCacheControl: "no-store",
		},
		{
			name:             "app Vary is merged with the page's",
			appMiddleware:    setsResponseHeader("Vary", "Origin"),
			pageVary:         "Accept-Language, origin",
			status:           http.StatusFound,
			wantLocation:     "/login",
			wantCacheControl: cacheControlPrivateNoCache,
			wantVary:         "Origin, Accept-Language",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				registry := emptyStaticPageRegistry()
				router := newCachedPageRouter(t, cachedPageRouteOptions{
					registry:      registry,
					appMiddleware: testCase.appMiddleware,
					page:          redirectsTo(testCase.wantLocation, testCase.status, testCase.pageVary),
					policy:        templater_dto.CachePolicy{Enabled: true, Static: true, MaxAgeSeconds: staticTestMaxAgeSeconds},
					persist:       true,
				})

				recorder := serveStaticTestRequest(t, router, nil)
				synctest.Wait()

				assert.Equal(t, testCase.status, recorder.Code)
				assert.Equal(t, testCase.wantLocation, recorder.Header().Get("Location"))
				assert.Equal(t, []string{testCase.wantCacheControl}, recorder.Header().Values("Cache-Control"))
				assert.Equal(t, testCase.wantVary, recorder.Header().Get("Vary"))
				assert.Empty(t, recorder.Header().Get("ETag"), "a redirect is not the page, so it has no page validator")
				assert.Empty(t, recorder.Body.String())
				assert.Zero(t, registry.UpsertArtefactCallCount.Load(), "a redirect must not be stored as the page")
			})
		})
	}
}

func TestCacheMiddlewareNoStore(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		appMiddleware    func(http.Handler) http.Handler
		name             string
		wantCacheControl string
		policy           templater_dto.CachePolicy
		wantCacheLookup  bool
	}{
		{
			name:             "static page with NoStore",
			policy:           templater_dto.CachePolicy{Enabled: true, Static: true, NoStore: true, MaxAgeSeconds: staticTestMaxAgeSeconds},
			wantCacheControl: cacheControlNoStore,
		},
		{
			name:             "caching disabled with NoStore",
			policy:           templater_dto.CachePolicy{NoStore: true},
			wantCacheControl: cacheControlNoStore,
		},
		{
			name:             "app Cache-Control wins over NoStore",
			appMiddleware:    setsResponseHeader("Cache-Control", "private, max-age=10"),
			policy:           templater_dto.CachePolicy{NoStore: true},
			wantCacheControl: "private, max-age=10",
		},
		{
			name:             "caching disabled without NoStore keeps the page default",
			policy:           templater_dto.CachePolicy{},
			wantCacheControl: cacheControlPrivateNoCache,
		},
		{
			name:             "static page without NoStore is cached",
			policy:           templater_dto.CachePolicy{Enabled: true, Static: true, MaxAgeSeconds: staticTestMaxAgeSeconds},
			wantCacheControl: staticTestPolicy,
			wantCacheLookup:  true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			registry := emptyStaticPageRegistry()
			router := newCachedPageRouter(t, cachedPageRouteOptions{
				registry:      registry,
				appMiddleware: testCase.appMiddleware,
				page:          http.HandlerFunc(renderStaticTestPage),
				policy:        testCase.policy,
			})

			recorder := serveStaticTestRequest(t, router, nil)

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, staticTestBody, recorder.Body.String())
			assert.Equal(t, []string{testCase.wantCacheControl}, recorder.Header().Values("Cache-Control"))
			assert.Equal(t, testCase.wantCacheLookup, registry.GetArtefactCallCount.Load() > 0)
		})
	}
}

func TestCacheMiddlewareStaticPageVary(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		appVary  string
		wantVary string
	}{
		{name: "no app Vary", wantVary: "Accept-Encoding"},
		{name: "app Vary is kept", appVary: "Origin", wantVary: "Origin, Accept-Encoding"},
		{name: "app token in another case is not repeated", appVary: "accept-encoding, Cookie", wantVary: "accept-encoding, Cookie"},
		{name: "app Vary of any is kept", appVary: "*", wantVary: "*"},
	}

	registries := []struct {
		registry func() *registry_domain.MockRegistryService
		name     string
	}{
		{name: "cache hit", registry: cachedStaticPageRegistry},
		{name: "cache miss", registry: emptyStaticPageRegistry},
	}

	for _, testCase := range testCases {
		for _, source := range registries {
			t.Run(source.name+" "+testCase.name, func(t *testing.T) {
				t.Parallel()

				var appMiddleware func(http.Handler) http.Handler
				if testCase.appVary != "" {
					appMiddleware = setsResponseHeader("Vary", testCase.appVary)
				}
				router := newStaticPageRouter(t, source.registry(), appMiddleware)

				recorder := serveStaticTestRequest(t, router, nil)

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Equal(t, []string{testCase.wantVary}, recorder.Header().Values("Vary"))
			})
		}
	}
}

func TestCacheMiddlewareStaticPageHitAndMissSendSameHeaders(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		appMiddleware func(http.Handler) http.Handler
		name          string
	}{
		{name: "page defaults"},
		{name: "app Vary", appMiddleware: setsResponseHeader("Vary", "Origin")},
		{name: "app Cache-Control", appMiddleware: setsResponseHeader("Cache-Control", staticTestPagePolicy)},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			hit := serveStaticTestRequest(t, newStaticPageRouter(t, cachedStaticPageRegistry(), testCase.appMiddleware), nil)
			miss := serveStaticTestRequest(t, newStaticPageRouter(t, emptyStaticPageRegistry(), testCase.appMiddleware), nil)

			require.Equal(t, http.StatusOK, hit.Code)
			require.Equal(t, http.StatusOK, miss.Code)
			assert.Equal(t, hit.Body.String(), miss.Body.String())
			assert.Equal(t, "fragment-patch", hit.Header().Get("X-PP-Response-Support"),
				"the client reloads a fragment that does not carry this header")
			assert.NotContains(t, miss.Header(), "Content-Encoding", "a body sent as it is has no content encoding")
			assert.Equal(t, comparableStaticPageHeaders(hit.Header()), comparableStaticPageHeaders(miss.Header()))
		})
	}
}

func comparableStaticPageHeaders(header http.Header) http.Header {
	headers := header.Clone()
	headers.Del(headerXCacheStatus)
	headers.Del(headerETag)
	return headers
}

func TestCacheMiddlewareStaticPageAfterEarlyHints(t *testing.T) {
	t.Parallel()

	router := newCachedPageRouter(t, cachedPageRouteOptions{
		registry: emptyStaticPageRegistry(),
		page: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Add("Link", "</theme.css>; rel=preload; as=style")
			w.WriteHeader(http.StatusEarlyHints)
			_, _ = w.Write([]byte(staticTestBody))
		}),
		policy: templater_dto.CachePolicy{Enabled: true, Static: true, MaxAgeSeconds: staticTestMaxAgeSeconds},
	})

	recorder := serveStaticTestRequest(t, router, nil)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, staticTestBody, recorder.Body.String())
}

func TestPipeResponseWriterResponse(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		statuses      []int
		links         []string
		wantLinks     []string
		wantStatus    int
		wantForwarded bool
	}{
		{name: "body without a status", wantStatus: http.StatusOK},
		{name: "early hints then a body", statuses: []int{http.StatusEarlyHints}, wantStatus: http.StatusOK},
		{
			name:          "early hints then not found",
			statuses:      []int{http.StatusEarlyHints, http.StatusNotFound},
			wantStatus:    http.StatusNotFound,
			wantForwarded: true,
		},
		{name: "switching protocols is final", statuses: []int{http.StatusSwitchingProtocols}, wantStatus: http.StatusSwitchingProtocols, wantForwarded: true},
		{name: "redirect", statuses: []int{http.StatusFound}, wantStatus: http.StatusFound, wantForwarded: true},
		{name: "server error", statuses: []int{http.StatusInternalServerError}, wantStatus: http.StatusInternalServerError, wantForwarded: true},
		{
			name:       "page links are kept",
			links:      []string{"</a.css>; rel=preload", "</b.js>; rel=modulepreload"},
			wantLinks:  []string{"</a.css>; rel=preload", "</b.js>; rel=modulepreload"},
			wantStatus: http.StatusOK,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			writer := newPipeResponseWriter(nil)
			writer.Header().Set("X-Page", "kept")
			for _, link := range testCase.links {
				writer.Header().Add("Link", link)
			}
			for _, status := range testCase.statuses {
				writer.WriteHeader(status)
			}

			response := writer.response()
			writer.Header().Set("X-Page", "changed")
			writer.Header().Del("Link")

			assert.Equal(t, testCase.wantStatus, response.statusCode)
			assert.Equal(t, testCase.wantForwarded, response.isForwarded())
			assert.Equal(t, testCase.wantLinks, response.links, "the links must outlive the pooled writer")
			if testCase.wantForwarded {
				assert.Equal(t, "kept", response.forwardedHeader.Get("X-Page"),
					"the forwarded headers must outlive the pooled writer")
			}
		})
	}
}

func TestCacheMiddlewareForwardsStaticPageErrorResponse(t *testing.T) {
	t.Parallel()

	const errorBody = "<h1>Not here</h1>"
	errorPage := func(status int, body string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			setDefaultCacheControl(w.Header(), headerValCachePrivateNoCache)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("Content-Length", "999")
			w.Header().Set("X-Error-Page", "yes")
			w.WriteHeader(status)
			_, _ = io.WriteString(w, body)
		})
	}
	prefixingCompressor := &capabilities_domain.MockCapabilityService{
		ExecuteFunc: func(_ context.Context, _ string, input io.Reader, _ capabilities_domain.CapabilityParams) (io.Reader, error) {
			return io.MultiReader(strings.NewReader("gz:"), input), nil
		},
	}

	testCases := []struct {
		capabilities   capabilities_domain.CapabilityService
		name           string
		acceptEncoding string
		body           string
		wantBody       string
		wantEncoding   string
		wantVary       string
		locale         string
		status         int
	}{
		{name: "not found page", status: http.StatusNotFound, body: errorBody, wantBody: errorBody},
		{name: "server error the page set", status: http.StatusInternalServerError, body: errorBody, wantBody: errorBody},
		{name: "error without a body", status: http.StatusGone},
		{
			name:           "compressed error page",
			capabilities:   prefixingCompressor,
			acceptEncoding: "gzip",
			status:         http.StatusNotFound,
			body:           errorBody,
			wantBody:       "gz:" + errorBody,
			wantEncoding:   "gzip",
			wantVary:       "Accept-Encoding",
		},
		{
			name:           "compressed error page on a localised route",
			capabilities:   prefixingCompressor,
			acceptEncoding: "gzip",
			locale:         "fr",
			status:         http.StatusNotFound,
			body:           errorBody,
			wantBody:       "gz:" + errorBody,
			wantEncoding:   "gzip",
			wantVary:       "Accept-Encoding, Accept-Language",
		},
		{
			name:           "error without a body is not marked compressed",
			capabilities:   prefixingCompressor,
			acceptEncoding: "gzip",
			status:         http.StatusGone,
			wantBody:       "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				registry := emptyStaticPageRegistry()
				router := newCachedPageRouter(t, cachedPageRouteOptions{
					registry:     registry,
					capabilities: testCase.capabilities,
					page:         errorPage(testCase.status, testCase.body),
					policy:       templater_dto.CachePolicy{Enabled: true, Static: true, MaxAgeSeconds: staticTestMaxAgeSeconds},
					persist:      true,
				})

				request := httptest.NewRequestWithContext(withLocale(t.Context(), testCase.locale), http.MethodGet, staticTestRoute, nil)
				request.Header.Set("Accept-Encoding", testCase.acceptEncoding)
				recorder := httptest.NewRecorder()
				router.ServeHTTP(recorder, request)
				synctest.Wait()

				assert.Equal(t, testCase.status, recorder.Code)
				assert.Equal(t, testCase.wantBody, recorder.Body.String())
				assert.Equal(t, "text/plain; charset=utf-8", recorder.Header().Get("Content-Type"))
				assert.Equal(t, "yes", recorder.Header().Get("X-Error-Page"))
				assert.Equal(t, []string{cacheControlPrivateNoCache}, recorder.Header().Values("Cache-Control"))
				assert.Equal(t, testCase.wantEncoding, recorder.Header().Get("Content-Encoding"))
				assert.Equal(t, testCase.wantVary, recorder.Header().Get("Vary"))
				assert.Empty(t, recorder.Header().Get("Content-Length"), "the page length no longer matches a compressed body")
				assert.Empty(t, recorder.Header().Get("ETag"))
				assert.Zero(t, registry.UpsertArtefactCallCount.Load(), "an error response must not be stored as the page")
			})
		})
	}
}

func TestCacheMiddlewareStaticPageLinks(t *testing.T) {
	t.Parallel()

	pageLinks := []string{"</theme.css>; rel=preload; as=style", "</app.js>; rel=modulepreload"}
	linkingPage := http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		for _, link := range pageLinks {
			w.Header().Add("Link", link)
		}
		renderStaticTestPage(w, request)
	})
	appLink := setsResponseHeader("Link", "</font.woff2>; rel=preload; as=font")

	persisted := make(chan []registry_dto.NamedProfile, 1)
	missRegistry := emptyStaticPageRegistry()
	missRegistry.UpsertArtefactFunc = func(
		_ context.Context, _ string, _ string, _ io.Reader, _ string, profiles []registry_dto.NamedProfile,
	) (*registry_dto.ArtefactMeta, error) {
		persisted <- profiles
		return &registry_dto.ArtefactMeta{}, nil
	}
	miss := serveStaticTestRequest(t, newCachedPageRouter(t, cachedPageRouteOptions{
		registry:      missRegistry,
		appMiddleware: appLink,
		page:          linkingPage,
		policy:        templater_dto.CachePolicy{Enabled: true, Static: true, MaxAgeSeconds: staticTestMaxAgeSeconds},
		persist:       true,
	}), nil)

	wantLinks := append([]string{"</font.woff2>; rel=preload; as=font"}, pageLinks...)
	require.Equal(t, http.StatusOK, miss.Code)
	assert.Equal(t, wantLinks, miss.Header().Values("Link"))

	storedProfiles := <-persisted
	hitRegistry := cachedStaticPageRegistry()
	cachedArtefact := hitRegistry.GetArtefactFunc
	hitRegistry.GetArtefactFunc = func(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
		artefact, err := cachedArtefact(ctx, artefactID)
		if err != nil {
			return nil, err
		}
		artefact.DesiredProfiles = storedProfiles
		return artefact, nil
	}
	hit := serveStaticTestRequest(t, newStaticPageRouter(t, hitRegistry, appLink), nil)

	require.Equal(t, http.StatusOK, hit.Code)
	assert.Equal(t, miss.Header().Values("Link"), hit.Header().Values("Link"))
	assert.Equal(t, comparableStaticPageHeaders(hit.Header()), comparableStaticPageHeaders(miss.Header()))

	oldEntry := serveStaticTestRequest(t, newStaticPageRouter(t, cachedStaticPageRegistry(), nil), nil)
	assert.Equal(t, http.StatusOK, oldEntry.Code)
	assert.Equal(t, staticTestBody, oldEntry.Body.String())
	assert.Empty(t, oldEntry.Header().Values("Link"), "a page cached before links were kept is sent without them")
}

func TestPageCacheProfilesWithLinks(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		wantErr   error
		name      string
		links     []string
		wantLinks []string
	}{
		{name: "no links keeps the shared profiles"},
		{name: "links are kept", links: []string{"</a.css>; rel=preload", "</b.js>"}, wantLinks: []string{"</a.css>; rel=preload", "</b.js>"}},
		{name: "links too long are dropped", links: []string{strings.Repeat("x", maxStoredPageLinkBytes+1)}, wantErr: errPageLinksTooLong},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			profiles, err := pageCacheProfilesWithLinks(testCase.links)

			require.ErrorIs(t, err, testCase.wantErr)
			require.Len(t, profiles, len(pageCacheProfiles))
			assert.Equal(t, testCase.wantLinks, storedPageLinks(&registry_dto.ArtefactMeta{DesiredProfiles: profiles}))
			assert.Nil(t, storedPageLinks(&registry_dto.ArtefactMeta{DesiredProfiles: pageCacheProfiles}),
				"the shared profiles must never carry one page's links")
		})
	}
}

func echoesPostedName(w http.ResponseWriter, request *http.Request) {
	_, _ = io.WriteString(w, "posted:"+request.FormValue("name"))
}

func TestCacheMiddlewareStaticPageMethodsOtherThanGetReachThePage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		registry         func() *registry_domain.MockRegistryService
		name             string
		method           string
		wantCacheControl string
		policy           templater_dto.CachePolicy
	}{
		{
			name:     "a form POST on a cache miss is neither shared nor stored",
			registry: emptyStaticPageRegistry,
			method:   http.MethodPost,
			policy:   templater_dto.CachePolicy{Enabled: true, Static: true, MaxAgeSeconds: staticTestMaxAgeSeconds},
		},
		{
			name:     "a form POST is not answered with the cached GET page",
			registry: cachedStaticPageRegistry,
			method:   http.MethodPost,
			policy:   templater_dto.CachePolicy{Enabled: true, Static: true, MaxAgeSeconds: staticTestMaxAgeSeconds},
		},
		{
			name:     "a PUT reaches the page",
			registry: cachedStaticPageRegistry,
			method:   http.MethodPut,
			policy:   templater_dto.CachePolicy{Enabled: true, Static: true, MaxAgeSeconds: staticTestMaxAgeSeconds},
		},
		{
			name:             "a form POST to a NoStore page still forbids storing",
			registry:         emptyStaticPageRegistry,
			method:           http.MethodPost,
			policy:           templater_dto.CachePolicy{Enabled: true, Static: true, NoStore: true},
			wantCacheControl: cacheControlNoStore,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				registry := testCase.registry()
				router := newCachedPageRouter(t, cachedPageRouteOptions{
					registry: registry,
					page:     http.HandlerFunc(echoesPostedName),
					policy:   testCase.policy,
					persist:  true,
				})

				form := url.Values{"name": {"Ada"}}.Encode()
				request := httptest.NewRequestWithContext(t.Context(), testCase.method, staticTestRoute, strings.NewReader(form))
				request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				recorder := httptest.NewRecorder()
				router.ServeHTTP(recorder, request)
				synctest.Wait()

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Equal(t, "posted:Ada", recorder.Body.String(), "the page must receive the form")
				assert.Zero(t, registry.GetArtefactCallCount.Load(), "the cache must not be consulted")
				assert.Zero(t, registry.UpsertArtefactCallCount.Load(), "the response must not be stored for GET visitors")
				assert.Equal(t, testCase.wantCacheControl, recorder.Header().Get("Cache-Control"))
				assert.Empty(t, recorder.Header().Get(headerXCacheStatus))
			})
		})
	}
}

func TestCacheMiddlewareStaticPageAnswersHeadWithoutABody(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		registry       func() *registry_domain.MockRegistryService
		name           string
		wantETag       string
		wantCacheState string
	}{
		{name: "from the cache", registry: cachedStaticPageRegistry, wantETag: staticTestETag, wantCacheState: "HIT"},
		{
			name:           "rendered on a cache miss",
			registry:       emptyStaticPageRegistry,
			wantETag:       formatJITETag(xxhash.Sum64String(staticTestBody)),
			wantCacheState: "MISS",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			registry := testCase.registry()
			recorder := serveStaticTestRequest(t, newStaticPageRouter(t, registry, nil), func(request *http.Request) {
				request.Method = http.MethodHead
			})

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.Empty(t, recorder.Body.String(), "a HEAD response has no body")
			assert.Equal(t, testCase.wantETag, recorder.Header().Get("ETag"))
			assert.Equal(t, staticTestPolicy, recorder.Header().Get("Cache-Control"))
			assert.Equal(t, testCase.wantCacheState, recorder.Header().Get(headerXCacheStatus))
			assert.Zero(t, registry.GetVariantDataCallCount.Load(), "the cached body must not be read for HEAD")
		})
	}
}

func TestCacheMiddlewareSharedRenderKeepsCookiesWithTheRequestThatRendered(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		status int
	}{
		{name: "redirect", status: http.StatusFound},
		{name: "error page", status: http.StatusNotFound},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				release := make(chan struct{})
				router := newCachedPageRouter(t, cachedPageRouteOptions{
					registry: emptyStaticPageRegistry(),
					page: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						<-release
						http.SetCookie(w, &http.Cookie{Name: "session", Value: "rendering-visitor"})
						w.Header().Set("Location", "/login")
						w.WriteHeader(testCase.status)
					}),
					policy: templater_dto.CachePolicy{Enabled: true, Static: true, MaxAgeSeconds: staticTestMaxAgeSeconds},
				})

				rendering, joining := httptest.NewRecorder(), httptest.NewRecorder()
				var requests sync.WaitGroup
				requests.Go(func() {
					router.ServeHTTP(rendering, httptest.NewRequestWithContext(t.Context(), http.MethodGet, staticTestRoute, nil))
				})
				synctest.Wait()
				requests.Go(func() {
					router.ServeHTTP(joining, httptest.NewRequestWithContext(t.Context(), http.MethodGet, staticTestRoute, nil))
				})
				synctest.Wait()
				close(release)
				requests.Wait()

				assert.Equal(t, testCase.status, rendering.Code)
				assert.Equal(t, testCase.status, joining.Code)
				assert.Equal(t, "/login", joining.Header().Get("Location"))
				assert.Equal(t, []string{"session=rendering-visitor"}, rendering.Header().Values("Set-Cookie"))
				assert.Empty(t, joining.Header().Values("Set-Cookie"), "another visitor's cookie must never be sent on")
			})
		})
	}
}

func TestJITResultSharedWithFollower(t *testing.T) {
	t.Parallel()

	withCookie := &jitResult{
		ForwardedHeader: http.Header{"Set-Cookie": {"a=1"}, "Location": {"/login"}},
		StatusCode:      http.StatusFound,
	}
	withoutCookie := &jitResult{ForwardedHeader: http.Header{"Location": {"/login"}}, StatusCode: http.StatusFound}
	page := &jitResult{Content: []byte(staticTestBody), StatusCode: http.StatusOK}

	follower := withCookie.sharedWithFollower()
	assert.NotContains(t, follower.ForwardedHeader, "Set-Cookie")
	assert.Equal(t, "/login", follower.ForwardedHeader.Get("Location"))
	assert.Equal(t, []string{"a=1"}, withCookie.ForwardedHeader.Values("Set-Cookie"), "the shared result must not change")
	assert.Same(t, withoutCookie, withoutCookie.sharedWithFollower())
	assert.Same(t, page, page.sharedWithFollower())
}

type closeRecordingReader struct {
	io.Reader
	closed atomic.Bool
}

func (reader *closeRecordingReader) Close() error {
	reader.closed.Store(true)
	return nil
}

func TestCacheMiddlewareClosesTheCachedPageBody(t *testing.T) {
	t.Parallel()

	body := &closeRecordingReader{Reader: strings.NewReader(staticTestBody)}
	registry := cachedStaticPageRegistry()
	registry.GetVariantDataFunc = func(context.Context, *registry_dto.Variant) (io.ReadCloser, error) {
		return body, nil
	}

	recorder := serveStaticTestRequest(t, newStaticPageRouter(t, registry, nil), nil)

	assert.Equal(t, staticTestBody, recorder.Body.String())
	assert.True(t, body.closed.Load(), "the blob stream must be closed after a cache hit")
}

func TestGenerateAndCacheResponse_SkipsPersistenceWhenEveryWriteSlotIsBusy(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		registry := &registry_domain.MockRegistryService{}
		middleware := NewCacheMiddleware(CacheMiddlewareConfig{CacheWriteConcurrency: 1},
			&templater_domain.MockManifestStoreView{}, registry, nil, "")
		middleware.writeLimiter <- struct{}{}

		request, pctx := newCarrierRequest(t, "")
		defer daemon_dto.ReleasePikoRequestCtx(pctx)
		upstream := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(staticTestBody))
		})

		result, err := middleware.generateAndCacheResponse(request, upstream, "artefact-1")
		require.NoError(t, err)
		assert.Equal(t, staticTestBody, string(result.Content), "the page is still served")

		synctest.Wait()
		<-middleware.writeLimiter
		synctest.Wait()

		assert.Zero(t, registry.UpsertArtefactCallCount.Load(), "no goroutine may wait for a write slot")
	})
}

func TestCacheMiddlewareReportsAGenerationFailureOnce(t *testing.T) {
	t.Parallel()

	logHandler := &recordingLogHandler{}
	ctx := logger_domain.WithLogger(t.Context(), logger_domain.New(slog.New(logHandler), "test"))
	router := newCachedPageRouter(t, cachedPageRouteOptions{
		registry: emptyStaticPageRegistry(),
		capabilities: &capabilities_domain.MockCapabilityService{
			ExecuteFunc: func(context.Context, string, io.Reader, capabilities_domain.CapabilityParams) (io.Reader, error) {
				return nil, errors.New("compressor unavailable")
			},
		},
		page:   http.HandlerFunc(renderStaticTestPage),
		policy: templater_dto.CachePolicy{Enabled: true, Static: true, MaxAgeSeconds: staticTestMaxAgeSeconds},
	})

	request := httptest.NewRequestWithContext(ctx, http.MethodGet, staticTestRoute, nil)
	request.Header.Set("Accept-Encoding", "gzip")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Len(t, logHandler.atLevel(slog.LevelError), 1, "the failure must be reported once")
	assert.Empty(t, logHandler.atLevel(slog.LevelWarn))
}
