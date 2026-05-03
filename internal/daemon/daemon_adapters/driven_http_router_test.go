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
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/render/render_dto"
	"piko.sh/piko/internal/security/security_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestParseFragmentParam(t *testing.T) {
	testCases := []struct {
		name     string
		rawQuery string
		expected bool
	}{
		{
			name:     "empty query returns false",
			rawQuery: "",
			expected: false,
		},
		{
			name:     "_f=true returns true",
			rawQuery: "_f=true",
			expected: true,
		},
		{
			name:     "_f=1 returns true",
			rawQuery: "_f=1",
			expected: true,
		},
		{
			name:     "_f=false returns false",
			rawQuery: "_f=false",
			expected: false,
		},
		{
			name:     "_f=0 returns false",
			rawQuery: "_f=0",
			expected: false,
		},
		{
			name:     "other query param does not trigger fragment",
			rawQuery: "page=1&sort=desc",
			expected: false,
		},
		{
			name:     "_f param with other params",
			rawQuery: "page=1&_f=true&sort=desc",
			expected: true,
		},
		{
			name:     "_f with yes value (parsed by strconv)",
			rawQuery: "_f=yes",
			expected: false,
		},
		{
			name:     "_f with no value (parsed by strconv)",
			rawQuery: "_f=no",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/?"+tc.rawQuery, nil)
			result := parseFragmentParam(request)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetMatchedPattern(t *testing.T) {
	t.Run("returns pattern from PikoRequestCtx", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		ctx := daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{
			Locale:         "en",
			MatchedPattern: "/users/{id}",
		})
		request = request.WithContext(ctx)

		result := getMatchedPattern(request)
		assert.Equal(t, "/users/{id}", result)
	})

	t.Run("returns empty when no route data in context", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/test", nil)

		result := getMatchedPattern(request)
		assert.Empty(t, result)
	})

	t.Run("falls back to chi route context", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		chiCtx := chi.NewRouteContext()
		chiCtx.RoutePatterns = []string{"/api/*"}
		ctx := context.WithValue(request.Context(), chi.RouteCtxKey, chiCtx)
		request = request.WithContext(ctx)

		result := getMatchedPattern(request)
		assert.Equal(t, "/api/*", result)
	})

	t.Run("prefers custom pattern over chi pattern", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/test", nil)

		chiCtx := chi.NewRouteContext()
		chiCtx.RoutePatterns = []string{"/fallback"}
		ctx := context.WithValue(request.Context(), chi.RouteCtxKey, chiCtx)

		ctx = daemon_dto.WithPikoRequestCtx(ctx, &daemon_dto.PikoRequestCtx{
			MatchedPattern: "/preferred",
		})
		request = request.WithContext(ctx)

		result := getMatchedPattern(request)
		assert.Equal(t, "/preferred", result)
	})
}

func TestValidateRedirectStatusCode(t *testing.T) {
	testCases := []struct {
		name         string
		statusCode   int
		expected     int
		expectsValid bool
	}{
		{
			name:         "301 Moved Permanently is valid",
			statusCode:   http.StatusMovedPermanently,
			expected:     http.StatusMovedPermanently,
			expectsValid: true,
		},
		{
			name:         "302 Found is valid",
			statusCode:   http.StatusFound,
			expected:     http.StatusFound,
			expectsValid: true,
		},
		{
			name:         "303 See Other is valid",
			statusCode:   http.StatusSeeOther,
			expected:     http.StatusSeeOther,
			expectsValid: true,
		},
		{
			name:         "307 Temporary Redirect is valid",
			statusCode:   http.StatusTemporaryRedirect,
			expected:     http.StatusTemporaryRedirect,
			expectsValid: true,
		},
		{
			name:         "200 OK defaults to 302",
			statusCode:   http.StatusOK,
			expected:     http.StatusFound,
			expectsValid: false,
		},
		{
			name:         "500 Internal Server Error defaults to 302",
			statusCode:   http.StatusInternalServerError,
			expected:     http.StatusFound,
			expectsValid: false,
		},
		{
			name:         "308 Permanent Redirect defaults to 302",
			statusCode:   http.StatusPermanentRedirect,
			expected:     http.StatusFound,
			expectsValid: false,
		},
		{
			name:         "0 defaults to 302",
			statusCode:   0,
			expected:     http.StatusFound,
			expectsValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validateRedirectStatusCode(context.Background(), tc.statusCode)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestE2eGuardMiddleware(t *testing.T) {
	t.Run("returns 404 for any request", func(t *testing.T) {
		inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("should not reach here"))
		})

		handler := e2eGuardMiddleware(inner)
		request := httptest.NewRequest(http.MethodGet, "/e2e/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

func TestApplyMiddlewares(t *testing.T) {
	t.Run("returns base handler when no cache and no page middleware", func(t *testing.T) {
		called := false
		baseHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		entry := &templater_domain.MockPageEntryView{}
		result := applyMiddlewares(baseHandler, entry, nil, nil)

		request := httptest.NewRequest(http.MethodGet, "/", nil)
		recorder := httptest.NewRecorder()
		result.ServeHTTP(recorder, request)

		assert.True(t, called)
	})

	t.Run("applies cache middleware", func(t *testing.T) {
		baseHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		cacheApplied := false
		cacheMiddleware := func(next http.Handler) http.Handler {
			cacheApplied = true
			return next
		}

		entry := &templater_domain.MockPageEntryView{}
		_ = applyMiddlewares(baseHandler, entry, cacheMiddleware, nil)

		assert.True(t, cacheApplied)
	})

	t.Run("applies page middlewares in reverse order", func(t *testing.T) {
		order := []string{}
		baseHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			order = append(order, "base")
			w.WriteHeader(http.StatusOK)
		})

		middleware1 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "mw1")
				next.ServeHTTP(w, r)
			})
		}

		middleware2 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "mw2")
				next.ServeHTTP(w, r)
			})
		}

		entry := &templater_domain.MockPageEntryView{
			GetHasMiddlewareFunc: func() bool { return true },
			GetMiddlewaresFunc: func() []func(http.Handler) http.Handler {
				return []func(http.Handler) http.Handler{middleware1, middleware2}
			},
		}

		result := applyMiddlewares(baseHandler, entry, nil, nil)

		request := httptest.NewRequest(http.MethodGet, "/", nil)
		recorder := httptest.NewRecorder()
		result.ServeHTTP(recorder, request)

		assert.Equal(t, []string{"mw1", "mw2", "base"}, order)
	})
}

func TestBuildHeaders(t *testing.T) {
	testCases := []struct {
		name     string
		lh       render_dto.LinkHeader
		expected string
	}{
		{
			name: "basic preload link",
			lh: render_dto.LinkHeader{
				URL: "/assets/style.css",
				Rel: "preload",
			},
			expected: "</assets/style.css>; rel=preload",
		},
		{
			name: "preload with as attribute",
			lh: render_dto.LinkHeader{
				URL: "/assets/script.js",
				Rel: "preload",
				As:  "script",
			},
			expected: "</assets/script.js>; rel=preload; as=script",
		},
		{
			name: "preload with type attribute",
			lh: render_dto.LinkHeader{
				URL:  "/assets/font.woff2",
				Rel:  "preload",
				As:   "font",
				Type: "font/woff2",
			},
			expected: `</assets/font.woff2>; rel=preload; as=font; type="font/woff2"`,
		},
		{
			name: "preload with anonymous crossorigin",
			lh: render_dto.LinkHeader{
				URL:         "/api/data",
				Rel:         "preload",
				CrossOrigin: "anonymous",
			},
			expected: "</api/data>; rel=preload; crossorigin=anonymous",
		},
		{
			name: "preload with use-credentials crossorigin",
			lh: render_dto.LinkHeader{
				URL:         "/api/secure",
				Rel:         "preload",
				CrossOrigin: "use-credentials",
			},
			expected: "</api/secure>; rel=preload; crossorigin=use-credentials",
		},
		{
			name: "preload with boolean crossorigin",
			lh: render_dto.LinkHeader{
				URL:         "/external/resource",
				Rel:         "preload",
				CrossOrigin: "true",
			},
			expected: "</external/resource>; rel=preload; crossorigin",
		},
		{
			name: "full link header with all attributes",
			lh: render_dto.LinkHeader{
				URL:         "/assets/font.woff2",
				Rel:         "preload",
				As:          "font",
				Type:        "font/woff2",
				CrossOrigin: "anonymous",
			},
			expected: `</assets/font.woff2>; rel=preload; as=font; type="font/woff2"; crossorigin=anonymous`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := buildHeaders(tc.lh)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSendSpecificEarlyHints(t *testing.T) {
	t.Run("returns false for empty headers", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)

		result := sendSpecificEarlyHints(recorder, request, []render_dto.LinkHeader{})

		assert.False(t, result)
	})

	t.Run("adds link headers to response", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)

		headers := []render_dto.LinkHeader{
			{URL: "/style.css", Rel: "preload", As: "style"},
			{URL: "/script.js", Rel: "preload", As: "script"},
		}

		_ = sendSpecificEarlyHints(recorder, request, headers)

		linkHeaders := recorder.Header().Values("Link")
		require.Len(t, linkHeaders, 2)
		assert.Contains(t, linkHeaders[0], "/style.css")
		assert.Contains(t, linkHeaders[1], "/script.js")
	})
}

type mockPageEntryViewWithPath struct {
	originalPath string
	templater_domain.MockPageEntryView
}

func (m *mockPageEntryViewWithPath) GetOriginalPath() string {
	return m.originalPath
}

func TestHandleRedirect_ServerRedirect_ReturnsError(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	span := newNoopSpan()

	meta := &templater_dto.InternalMetadata{}
	meta.ServerRedirect = "/loop-target"

	handleRedirect(context.Background(), recorder, request, meta, span)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.True(t, span.statusSet)
	assert.Equal(t, codes.Error, span.statusCode)
}

func TestHandleRedirect_ClientRedirect_Redirects(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/old-page", nil)
	span := newNoopSpan()

	meta := &templater_dto.InternalMetadata{}
	meta.ClientRedirect = "/new-page"
	meta.RedirectStatus = http.StatusMovedPermanently

	handleRedirect(context.Background(), recorder, request, meta, span)

	assert.Equal(t, http.StatusMovedPermanently, recorder.Code)
	assert.Equal(t, "/new-page", recorder.Header().Get("Location"))
	assert.True(t, span.statusSet)
	assert.Equal(t, codes.Ok, span.statusCode)
}

func TestHandleRedirect_ClientRedirect_DefaultStatus302(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	span := newNoopSpan()

	meta := &templater_dto.InternalMetadata{}
	meta.ClientRedirect = "/target"

	handleRedirect(context.Background(), recorder, request, meta, span)

	assert.Equal(t, http.StatusFound, recorder.Code)
	assert.Equal(t, "/target", recorder.Header().Get("Location"))
}

func TestHandleRedirect_ClientRedirect_Uses302(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	span := newNoopSpan()

	meta := &templater_dto.InternalMetadata{}
	meta.ClientRedirect = "/target-url"

	handleRedirect(context.Background(), recorder, request, meta, span)

	assert.Equal(t, http.StatusFound, recorder.Code)
	assert.Equal(t, "/target-url", recorder.Header().Get("Location"))
}

func TestHandleRedirect_NoTarget_ReturnsError(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	span := newNoopSpan()

	meta := &templater_dto.InternalMetadata{}

	handleRedirect(context.Background(), recorder, request, meta, span)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.True(t, span.statusSet)
	assert.Equal(t, codes.Error, span.statusCode)
}

func TestHandleRedirect_InvalidStatusCode_Defaults302(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	span := newNoopSpan()

	meta := &templater_dto.InternalMetadata{}
	meta.ClientRedirect = "/target"
	meta.RedirectStatus = http.StatusOK

	handleRedirect(context.Background(), recorder, request, meta, span)

	assert.Equal(t, http.StatusFound, recorder.Code)
	assert.Equal(t, "/target", recorder.Header().Get("Location"))
}

func TestHandleRedirect_ClearsContentTypeHeader(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	recorder.Header().Set("Content-Type", "text/html")
	recorder.Header().Set("X-PP-Response-Support", "fragment-patch")
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	span := newNoopSpan()

	meta := &templater_dto.InternalMetadata{}
	meta.ClientRedirect = "/target"

	handleRedirect(context.Background(), recorder, request, meta, span)

	assert.Empty(t, recorder.Header().Get("Content-Type"))
	assert.Empty(t, recorder.Header().Get("X-PP-Response-Support"))
}

func TestHandleRedirect_EmptyClientRedirect_ReturnsError(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	span := newNoopSpan()

	meta := &templater_dto.InternalMetadata{}
	meta.ClientRedirect = ""

	handleRedirect(context.Background(), recorder, request, meta, span)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestHandleRedirect_SeeOtherStatus(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	span := newNoopSpan()

	meta := &templater_dto.InternalMetadata{}
	meta.ClientRedirect = "/dashboard"
	meta.RedirectStatus = http.StatusSeeOther

	handleRedirect(context.Background(), recorder, request, meta, span)

	assert.Equal(t, http.StatusSeeOther, recorder.Code)
	assert.Equal(t, "/dashboard", recorder.Header().Get("Location"))
}

func TestHandleRedirect_TemporaryRedirectStatus(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	span := newNoopSpan()

	meta := &templater_dto.InternalMetadata{}
	meta.ClientRedirect = "/temp-location"
	meta.RedirectStatus = http.StatusTemporaryRedirect

	handleRedirect(context.Background(), recorder, request, meta, span)

	assert.Equal(t, http.StatusTemporaryRedirect, recorder.Code)
	assert.Equal(t, "/temp-location", recorder.Header().Get("Location"))
}

func TestHandlePageRenderError_ContextCanceled_NoHTTPError(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	span := newNoopSpan()
	entry := &mockPageEntryViewWithPath{}

	handlePageRenderError(context.Background(), recorder, request, context.Canceled, pageErrorContext{Entry: entry, Store: &templater_domain.MockManifestStoreView{}, Span: span}, false)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestHandlePageRenderError_DeadlineExceeded_NoHTTPError(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	span := newNoopSpan()
	entry := &mockPageEntryViewWithPath{}

	handlePageRenderError(context.Background(), recorder, request, context.DeadlineExceeded, pageErrorContext{Entry: entry, Store: &templater_domain.MockManifestStoreView{}, Span: span}, false)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestHandlePageRenderError_GenericError_RecordsError(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	span := newNoopSpan()
	entry := &mockPageEntryViewWithPath{}

	handlePageRenderError(context.Background(), recorder, request, errors.New("render failed"), pageErrorContext{Entry: entry, Store: &templater_domain.MockManifestStoreView{}, Span: span}, false)

	assert.True(t, span.errorRecorded)
	assert.Equal(t, codes.Error, span.statusCode)
}

func TestHandlePageRenderError_RedirectError_Redirects(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/old-page", nil)
	span := newNoopSpan()
	entry := &mockPageEntryViewWithPath{}

	redirectMeta := templater_dto.InternalMetadata{}
	redirectMeta.ClientRedirect = "/new-page"
	redirectMeta.RedirectStatus = http.StatusMovedPermanently
	err := &templater_dto.RedirectRequired{Metadata: redirectMeta}

	handlePageRenderError(context.Background(), recorder, request, err, pageErrorContext{Entry: entry, Store: &templater_domain.MockManifestStoreView{}, Span: span}, false)

	assert.Equal(t, http.StatusMovedPermanently, recorder.Code)
	assert.Equal(t, "/new-page", recorder.Header().Get("Location"))
}

func TestHandlePageProbeError_SetsErrorStatus(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	span := newNoopSpan()
	entry := &mockPageEntryViewWithPath{}

	handlePageProbeError(context.Background(), recorder, request, errors.New("probe failed"), pageErrorContext{Entry: entry, Store: &templater_domain.MockManifestStoreView{}, Span: span})

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.True(t, span.statusSet)
	assert.Equal(t, codes.Error, span.statusCode)
	assert.True(t, span.errorRecorded)
}

func TestHandlePartialProbeError_SetsErrorStatus(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	span := newNoopSpan()
	entry := &mockPageEntryViewWithPath{}

	handlePartialProbeError(context.Background(), recorder, errors.New("probe failed"), entry, span)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.True(t, span.statusSet)
	assert.Equal(t, codes.Error, span.statusCode)
	assert.True(t, span.errorRecorded)
}

func TestHandlePartialRenderError_RecordsError(t *testing.T) {
	t.Parallel()

	span := newNoopSpan()
	entry := &mockPageEntryViewWithPath{}

	handlePartialRenderError(context.Background(), errors.New("render failed"), entry, span)

	assert.True(t, span.statusSet)
	assert.Equal(t, codes.Error, span.statusCode)
	assert.True(t, span.errorRecorded)
}

func TestMountActionRoutes_EmptyActions_NoError(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		r := chi.NewRouter()
		mountActionRoutes(&MountRoutesConfig{
			Router:          r,
			Actions:         map[string]ActionHandlerEntry{},
			RateLimitConfig: security_dto.RateLimitValues{},
		})
	})
}

func TestMountActionRoutes_WithActions_RegistersRoutes(t *testing.T) {
	t.Parallel()

	r := chi.NewRouter()
	actions := map[string]ActionHandlerEntry{
		"test.action": {
			Name:   "test.action",
			Method: http.MethodPost,
			Create: func() any { return struct{}{} },
			Invoke: func(_ context.Context, _ any, _ map[string]any) (any, error) { return "ok", nil },
		},
	}

	mountActionRoutes(&MountRoutesConfig{
		Router:          r,
		Actions:         actions,
		RateLimitConfig: security_dto.RateLimitValues{},
	})

	request := httptest.NewRequest(http.MethodPost, "/_piko/actions/test.action", nil)
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, request)

	assert.NotEqual(t, http.StatusNotFound, recorder.Code)
}

func TestMountActionRoutes_CustomBasePath(t *testing.T) {
	t.Parallel()

	r := chi.NewRouter()
	routeSettings := RouteSettings{ActionServePath: "/api/actions"}

	actions := map[string]ActionHandlerEntry{
		"test.action": {
			Name:   "test.action",
			Method: http.MethodPost,
			Create: func() any { return struct{}{} },
			Invoke: func(_ context.Context, _ any, _ map[string]any) (any, error) { return "ok", nil },
		},
	}

	mountActionRoutes(&MountRoutesConfig{
		Router:          r,
		RouteSettings:   routeSettings,
		Actions:         actions,
		RateLimitConfig: security_dto.RateLimitValues{},
	})

	request := httptest.NewRequest(http.MethodPost, "/api/actions/test.action", nil)
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, request)

	assert.NotEqual(t, http.StatusNotFound, recorder.Code)
}

func TestMountActionRoutes_CustomMaxBodyBytes(t *testing.T) {
	t.Parallel()

	r := chi.NewRouter()
	routeSettings := RouteSettings{ActionMaxBodyBytes: 5 * 1024}

	mountActionRoutes(&MountRoutesConfig{
		Router:        r,
		RouteSettings: routeSettings,
		Actions: map[string]ActionHandlerEntry{
			"test": {
				Name:   "test",
				Method: http.MethodPost,
				Create: func() any { return struct{}{} },
				Invoke: func(_ context.Context, _ any, _ map[string]any) (any, error) { return nil, nil },
			},
		},
		RateLimitConfig: security_dto.RateLimitValues{},
	})

	request := httptest.NewRequest(http.MethodPost, "/_piko/actions/test", nil)
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, request)
	assert.NotEqual(t, http.StatusNotFound, recorder.Code)
}

func TestRegisterRoutesFromStore_EmptyStore(t *testing.T) {
	t.Parallel()

	store := &templater_domain.MockManifestStoreView{}
	regDeps := &routeRegistrationDeps{
		router: chi.NewRouter(),
	}

	pageCount, partialCount := registerRoutesFromStore(context.Background(), regDeps, store)

	assert.Equal(t, 0, pageCount)
	assert.Equal(t, 0, partialCount)
}

func TestExtractOTelContext_ReturnsContextExtra(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := extractOTelContext(request)

	assert.NotNil(t, ctx)
}

func TestExtractOTelContext_WithTraceHeaders(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	ctx := extractOTelContext(request)

	assert.NotNil(t, ctx)
}

func TestGetMatchedPattern_WithPikoRequestCtx(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{
		MatchedPattern: "/users/{id}",
	})
	request = request.WithContext(ctx)

	result := getMatchedPattern(request)

	assert.Equal(t, "/users/{id}", result)
}

func TestGetMatchedPattern_EmptyPikoRequestCtx_FallsBackToChi(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{
		MatchedPattern: "",
	})
	request = request.WithContext(ctx)

	result := getMatchedPattern(request)
	assert.Equal(t, "", result)
}

func TestGetMatchedPattern_NoContextData(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/test", nil)

	result := getMatchedPattern(request)

	assert.Equal(t, "", result)
}

func TestE2EGuardMiddleware_Returns404(t *testing.T) {
	t.Parallel()

	innerCalled := false
	inner := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		innerCalled = true
	})

	guarded := e2eGuardMiddleware(inner)

	request := httptest.NewRequest(http.MethodGet, "/e2e/test", nil)
	recorder := httptest.NewRecorder()

	guarded.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.False(t, innerCalled, "Inner handler should not be called")
}

func TestContextKey_StringValue(t *testing.T) {
	t.Parallel()

	key := contextKey("test-key")
	assert.Equal(t, contextKey("test-key"), key)
}

func TestMountRoutesConfig_Fields(t *testing.T) {
	t.Parallel()

	mountConfig := MountRoutesConfig{
		Actions: map[string]ActionHandlerEntry{
			"test": {Name: "test"},
		},
	}

	assert.NotNil(t, mountConfig.Actions)
	assert.Contains(t, mountConfig.Actions, "test")
}

func TestExtractErrorStatusCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err        error
		name       string
		wantStatus int
	}{
		{
			name:       "plain error defaults to 500",
			err:        errors.New("something broke"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "NotFoundError returns 404",
			err:        daemon_dto.NotFound("user", "123"),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "UnauthorisedError returns 401",
			err:        daemon_dto.Unauthorised("session expired"),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "ForbiddenError returns 403",
			err:        daemon_dto.Forbidden("access denied"),
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "BadRequestError returns 400",
			err:        daemon_dto.BadRequest("missing header"),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "ConflictError returns 409",
			err:        daemon_dto.Conflict("already exists"),
			wantStatus: http.StatusConflict,
		},
		{
			name:       "ValidationError returns 422",
			err:        daemon_dto.ValidationField("email", "invalid"),
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "PageError with custom status code",
			err:        daemon_dto.PageError(http.StatusTooManyRequests, "rate limited"),
			wantStatus: http.StatusTooManyRequests,
		},
		{
			name:       "TeapotError returns 418",
			err:        daemon_dto.Teapot(""),
			wantStatus: http.StatusTeapot,
		},
		{
			name:       "wrapped NotFoundError still returns 404",
			err:        fmt.Errorf("loading profile: %w", daemon_dto.NotFound("user", "abc")),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "wrapped ForbiddenError still returns 403",
			err:        fmt.Errorf("authorisation check: %w", daemon_dto.Forbidden("not allowed")),
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := extractErrorStatusCode(tt.err)
			assert.Equal(t, tt.wantStatus, got)
		})
	}
}

func TestResponseTracker_Write(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	rt := &responseTracker{ResponseWriter: recorder}

	assert.False(t, rt.started)

	n, err := rt.Write([]byte("hello"))

	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.True(t, rt.started)
	assert.Equal(t, "hello", recorder.Body.String())
}

func TestResponseTracker_WriteHeader(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	rt := &responseTracker{ResponseWriter: recorder}

	assert.False(t, rt.started)

	rt.WriteHeader(http.StatusNotFound)

	assert.True(t, rt.started)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestResponseTracker_Flush(t *testing.T) {
	t.Parallel()

	t.Run("delegates to underlying flusher", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		rt := &responseTracker{ResponseWriter: recorder}

		assert.NotPanics(t, func() {
			rt.Flush()
		})
	})

	t.Run("does not panic when underlying writer is not a flusher", func(t *testing.T) {
		t.Parallel()

		rt := &responseTracker{ResponseWriter: &nonFlushWriter{}}

		assert.NotPanics(t, func() {
			rt.Flush()
		})
	})
}

func TestResponseTracker_Unwrap(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	rt := &responseTracker{ResponseWriter: recorder}

	unwrapped := rt.Unwrap()

	assert.Equal(t, recorder, unwrapped)
}

type nonFlushWriter struct{}

func (nonFlushWriter) Header() http.Header         { return http.Header{} }
func (nonFlushWriter) Write(b []byte) (int, error) { return len(b), nil }
func (nonFlushWriter) WriteHeader(_ int)           {}

func TestRenderErrorPage_NoErrorPageInStore(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/missing", nil)
	store := &templater_domain.MockManifestStoreView{
		FindErrorPageFunc: func(_ int, _ string) (templater_domain.PageEntryView, bool) {
			return nil, false
		},
	}

	ok := renderErrorPage(
		context.Background(), recorder, request,
		pageErrorContext{Deps: &daemon_domain.HTTPHandlerDependencies{}, Store: store, WebsiteConfig: &config.WebsiteConfig{}},
		errorPageRequest{StatusCode: 404, Message: "not found", OriginalPath: "/missing"},
	)

	assert.False(t, ok)
}

func TestRenderErrorPage_ProbePageFails(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/broken", nil)
	mockEntry := &templater_domain.MockPageEntryView{
		GetOriginalPathFunc: func() string { return "/pages/!404.pk" },
	}
	store := &templater_domain.MockManifestStoreView{
		FindErrorPageFunc: func(_ int, _ string) (templater_domain.PageEntryView, bool) {
			return mockEntry, true
		},
	}
	deps := &daemon_domain.HTTPHandlerDependencies{
		Templater: &templater_domain.MockTemplaterService{
			ProbePageFunc: func(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request, _ *config.WebsiteConfig) (*templater_dto.PageProbeResult, error) {
				return nil, errors.New("probe failed")
			},
		},
	}

	ok := renderErrorPage(
		context.Background(), recorder, request,
		pageErrorContext{Deps: deps, Store: store, WebsiteConfig: &config.WebsiteConfig{}},
		errorPageRequest{StatusCode: 500, Message: "internal error", OriginalPath: "/broken"},
	)

	assert.False(t, ok)
}

func TestRenderErrorPage_RenderPageFails(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/fail", nil)
	mockEntry := &templater_domain.MockPageEntryView{
		GetOriginalPathFunc: func() string { return "/pages/!500.pk" },
	}
	store := &templater_domain.MockManifestStoreView{
		FindErrorPageFunc: func(_ int, _ string) (templater_domain.PageEntryView, bool) {
			return mockEntry, true
		},
	}
	deps := &daemon_domain.HTTPHandlerDependencies{
		Templater: &templater_domain.MockTemplaterService{
			ProbePageFunc: func(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request, _ *config.WebsiteConfig) (*templater_dto.PageProbeResult, error) {
				return &templater_dto.PageProbeResult{}, nil
			},
			RenderPageFunc: func(_ context.Context, _ templater_domain.RenderRequest) error {
				return errors.New("render failed")
			},
		},
	}

	ok := renderErrorPage(
		context.Background(), recorder, request,
		pageErrorContext{Deps: deps, Store: store, WebsiteConfig: &config.WebsiteConfig{}},
		errorPageRequest{StatusCode: 500, Message: "internal error", OriginalPath: "/fail"},
	)

	assert.False(t, ok)
}

func TestRenderErrorPage_Success(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/not-here", nil)
	mockEntry := &templater_domain.MockPageEntryView{
		GetOriginalPathFunc: func() string { return "/pages/!404.pk" },
	}
	store := &templater_domain.MockManifestStoreView{
		FindErrorPageFunc: func(_ int, _ string) (templater_domain.PageEntryView, bool) {
			return mockEntry, true
		},
	}
	deps := &daemon_domain.HTTPHandlerDependencies{
		Templater: &templater_domain.MockTemplaterService{
			ProbePageFunc: func(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request, _ *config.WebsiteConfig) (*templater_dto.PageProbeResult, error) {
				return &templater_dto.PageProbeResult{}, nil
			},
			RenderPageFunc: func(_ context.Context, req templater_domain.RenderRequest) error {
				_, _ = req.Writer.Write([]byte("<h1>Not Found</h1>"))
				return nil
			},
		},
	}

	ok := renderErrorPage(
		context.Background(), recorder, request,
		pageErrorContext{Deps: deps, Store: store, WebsiteConfig: &config.WebsiteConfig{}},
		errorPageRequest{StatusCode: 404, Message: "page not found", OriginalPath: "/not-here"},
	)

	assert.True(t, ok)
	assert.Equal(t, 404, recorder.Code)
	assert.Equal(t, contentTypeHTML, recorder.Header().Get(headerContentType))
	assert.Contains(t, recorder.Body.String(), "Not Found")
}

func TestWriteDevErrorFallback(t *testing.T) {
	t.Parallel()

	t.Run("writes HTML error page in development mode", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		ctx := daemon_dto.WithPikoRequestCtx(context.Background(), &daemon_dto.PikoRequestCtx{
			DevelopmentMode: true,
		})

		ok := writeDevErrorFallback(ctx, recorder, http.StatusInternalServerError, "db connection failed: timeout after 5s")

		assert.True(t, ok)
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.Contains(t, recorder.Header().Get(headerContentType), "text/html")
		body := recorder.Body.String()
		assert.Contains(t, body, "500 Internal Server Error")
		assert.Contains(t, body, "db connection failed: timeout after 5s")
	})

	t.Run("returns false in production mode", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()

		ok := writeDevErrorFallback(context.Background(), recorder, http.StatusInternalServerError, "secret db error")

		assert.False(t, ok)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Empty(t, recorder.Body.String())
	})

	t.Run("escapes HTML in error message", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		ctx := daemon_dto.WithPikoRequestCtx(context.Background(), &daemon_dto.PikoRequestCtx{
			DevelopmentMode: true,
		})

		ok := writeDevErrorFallback(ctx, recorder, http.StatusBadRequest, `<script>alert("xss")</script>`)

		assert.True(t, ok)
		body := recorder.Body.String()
		assert.NotContains(t, body, "<script>")
		assert.Contains(t, body, "&lt;script&gt;")
	})
}

func TestAliasCatchAllParam(t *testing.T) {
	t.Parallel()

	t.Run("aliases inner catch-all when nested under parent", func(t *testing.T) {
		t.Parallel()

		rctx := chi.NewRouteContext()

		rctx.URLParams.Add("*", "docs/get-started/introduction")
		rctx.URLParams.Add("*", "get-started/introduction")

		req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(
			context.WithValue(context.Background(), chi.RouteCtxKey, rctx),
		)

		aliasCatchAllParam(req, "slug")

		got := chi.URLParam(req, "slug")
		assert.Equal(t, "get-started/introduction", got, "alias must reflect the inner subrouter's capture, not the parent's")
	})

	t.Run("no-op when chi route context is absent", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		assert.NotPanics(t, func() {
			aliasCatchAllParam(req, "slug")
		})
	})

	t.Run("no-op when wildcard not captured", func(t *testing.T) {
		t.Parallel()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "42")

		req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(
			context.WithValue(context.Background(), chi.RouteCtxKey, rctx),
		)

		aliasCatchAllParam(req, "slug")
		assert.Empty(t, chi.URLParam(req, "slug"))
	})

	t.Run("aliases deepest catch-all when three subrouters stack", func(t *testing.T) {
		t.Parallel()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("*", "outer/middle/inner")
		rctx.URLParams.Add("*", "middle/inner")
		rctx.URLParams.Add("*", "inner")

		req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(
			context.WithValue(context.Background(), chi.RouteCtxKey, rctx),
		)

		aliasCatchAllParam(req, "slug")
		assert.Equal(t, "inner", chi.URLParam(req, "slug"),
			"alias must reflect the deepest subrouter's capture")
	})
}

func TestTranslateCatchAllForChi(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		pattern     string
		wantPattern string
		wantAlias   string
	}{
		{
			name:        "no regex segment passes through",
			pattern:     "/blog/{slug}",
			wantPattern: "/blog/{slug}",
			wantAlias:   "",
		},
		{
			name:        "static path passes through",
			pattern:     "/about",
			wantPattern: "/about",
			wantAlias:   "",
		},
		{
			name:        "trailing dot-plus is translated",
			pattern:     "/docs/{slug:.+}",
			wantPattern: "/docs/*",
			wantAlias:   "slug",
		},
		{
			name:        "trailing dot-star is translated",
			pattern:     "/docs/{slug:.*}",
			wantPattern: "/docs/*",
			wantAlias:   "slug",
		},
		{
			name:        "non-greedy regex is translated",
			pattern:     "/docs/{slug:.+?}",
			wantPattern: "/docs/*",
			wantAlias:   "slug",
		},
		{
			name:        "character-class regex is translated",
			pattern:     "/files/{path:[a-zA-Z0-9/_-]+}",
			wantPattern: "/files/*",
			wantAlias:   "path",
		},
		{
			name:        "regex without name passes through",
			pattern:     "/blog/{:.+}",
			wantPattern: "/blog/{:.+}",
			wantAlias:   "",
		},
		{
			name:        "trailing brace without opener passes through",
			pattern:     "stray}",
			wantPattern: "stray}",
			wantAlias:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotPattern, gotAlias := translateCatchAllForChi(tt.pattern)
			assert.Equal(t, tt.wantPattern, gotPattern)
			assert.Equal(t, tt.wantAlias, gotAlias)
		})
	}
}
