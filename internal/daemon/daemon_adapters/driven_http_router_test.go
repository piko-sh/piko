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
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/daemon/daemon_frontend"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
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

	redirectMeta := templater_dto.InternalMetadata{
		ClientRedirect: "/new-page",
		RedirectStatus: http.StatusMovedPermanently}
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
	ctx := daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{})
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

func TestNamedStarRouteCarriesAllCapturedSegments(t *testing.T) {
	router := chi.NewRouter()
	pattern, alias := translateCatchAllForChi("/bulletins/archive/{rest}*")
	router.Get(pattern, func(w http.ResponseWriter, r *http.Request) {
		aliasCatchAllParam(r, alias)
		_, _ = w.Write([]byte(chi.URLParam(r, "rest")))
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/bulletins/archive/2026/march", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "2026/march", response.Body.String())
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

func newAutoHeadRequest(locale, path string, params map[string]string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "https://example.com"+path, nil)
	ctx := daemon_dto.WithPikoRequestCtx(req.Context(), &daemon_dto.PikoRequestCtx{Locale: locale})
	if params != nil {
		routeCtx := chi.NewRouteContext()
		for key, value := range params {
			routeCtx.URLParams.Add(key, value)
		}
		ctx = context.WithValue(ctx, chi.RouteCtxKey, routeCtx)
	}
	return req.WithContext(ctx)
}

func TestComputeAutoLocaleHead_LocalisedStaticPage(t *testing.T) {
	t.Parallel()

	entry := &templater_domain.MockPageEntryView{
		GetRoutePatternsFunc: func() map[string]string {
			return map[string]string{"en": "/what-we-do/", "fr": "/fr/what-we-do/"}
		},
	}
	cfg := &config.WebsiteConfig{I18n: config.I18nConfig{
		Locales:       []string{"en", "fr"},
		DefaultLocale: "en",
		Strategy:      "prefix_except_default",
	}}
	req := newAutoHeadRequest("fr", "/fr/what-we-do/", nil)

	head := computeAutoLocaleHead(req, entry, cfg)

	require.NotNil(t, head)
	assert.Equal(t, "fr", head.Language)
	assert.Equal(t, "https://example.com/fr/what-we-do/", head.CanonicalURL, "a translated page must be its own canonical")
	require.Len(t, head.AlternateLinks, 3)
	assert.Equal(t, map[string]string{"hreflang": "en", "href": "https://example.com/what-we-do/"}, head.AlternateLinks[0])
	assert.Equal(t, map[string]string{"hreflang": "fr", "href": "https://example.com/fr/what-we-do/"}, head.AlternateLinks[1])
	assert.Equal(t, map[string]string{"hreflang": "x-default", "href": "https://example.com/what-we-do/"}, head.AlternateLinks[2])
}

func TestComputeAutoLocaleHead_DefaultLocaleIsSelfCanonical(t *testing.T) {
	t.Parallel()

	entry := &templater_domain.MockPageEntryView{
		GetRoutePatternsFunc: func() map[string]string {
			return map[string]string{"en": "/what-we-do/", "fr": "/fr/what-we-do/"}
		},
	}
	cfg := &config.WebsiteConfig{I18n: config.I18nConfig{Locales: []string{"en", "fr"}, DefaultLocale: "en"}}

	head := computeAutoLocaleHead(newAutoHeadRequest("en", "/what-we-do/", nil), entry, cfg)

	require.NotNil(t, head)
	assert.Equal(t, "https://example.com/what-we-do/", head.CanonicalURL)
	assert.Equal(t, map[string]string{"hreflang": "x-default", "href": "https://example.com/what-we-do/"}, head.AlternateLinks[2])
}

func TestComputeAutoLocaleHead_UnknownLocaleFallsBackToDefaultCanonical(t *testing.T) {
	t.Parallel()

	entry := &templater_domain.MockPageEntryView{
		GetRoutePatternsFunc: func() map[string]string {
			return map[string]string{"en": "/what-we-do/", "fr": "/fr/what-we-do/"}
		},
	}
	cfg := &config.WebsiteConfig{I18n: config.I18nConfig{Locales: []string{"en", "fr"}, DefaultLocale: "en"}}

	head := computeAutoLocaleHead(newAutoHeadRequest("de", "/what-we-do/", nil), entry, cfg)

	require.NotNil(t, head)
	assert.Equal(t, "https://example.com/what-we-do/", head.CanonicalURL)
}

func TestComputeAutoLocaleHead_SingleLocaleEmitsSelfCanonicalOnly(t *testing.T) {
	t.Parallel()

	entry := &templater_domain.MockPageEntryView{
		GetRoutePatternsFunc: func() map[string]string { return map[string]string{"en": "/about"} },
	}
	cfg := &config.WebsiteConfig{I18n: config.I18nConfig{Locales: []string{"en"}, DefaultLocale: "en"}}

	head := computeAutoLocaleHead(newAutoHeadRequest("en", "/about", nil), entry, cfg)

	require.NotNil(t, head)
	assert.Equal(t, "https://example.com/about", head.CanonicalURL)
	assert.Empty(t, head.AlternateLinks)
}

func TestComputeAutoLocaleHead_NoRoutePatternsReturnsNil(t *testing.T) {
	t.Parallel()

	entry := &templater_domain.MockPageEntryView{
		GetRoutePatternsFunc: func() map[string]string { return map[string]string{} },
	}
	cfg := &config.WebsiteConfig{I18n: config.I18nConfig{Locales: []string{"en"}, DefaultLocale: "en"}}

	assert.Nil(t, computeAutoLocaleHead(newAutoHeadRequest("en", "/about", nil), entry, cfg))
}

func TestComputeAutoLocaleHead_SubstitutesRouteParams(t *testing.T) {
	t.Parallel()

	entry := &templater_domain.MockPageEntryView{
		GetRoutePatternsFunc: func() map[string]string {
			return map[string]string{"en": "/articles/{slug}", "fr": "/fr/articles/{slug}"}
		},
	}
	cfg := &config.WebsiteConfig{I18n: config.I18nConfig{Locales: []string{"en", "fr"}, DefaultLocale: "en"}}
	req := newAutoHeadRequest("en", "/articles/hello", map[string]string{"slug": "hello"})

	head := computeAutoLocaleHead(req, entry, cfg)

	require.NotNil(t, head)
	assert.Equal(t, "https://example.com/articles/hello", head.CanonicalURL)
	assert.Equal(t, "https://example.com/fr/articles/hello", head.AlternateLinks[1]["href"])
}

func TestSubstituteRouteParams(t *testing.T) {
	t.Parallel()

	params := map[string]string{"slug": "hello", "locationslug": "/london"}

	assert.Equal(t, "/articles/hello", substituteRouteParams("/articles/{slug}", params))
	assert.Equal(t, "/docs/hello", substituteRouteParams("/docs/{slug:.+}", params))
	assert.Equal(t, "/mcs/london/k8s", substituteRouteParams("/mcs{locationslug}/k8s", params))
	assert.Equal(t, "/x/{unknown}", substituteRouteParams("/x/{unknown}", params))
	assert.Equal(t, "/static", substituteRouteParams("/static", params))
}

func TestToggleTrailingSlash(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "/x/", toggleTrailingSlash("/x"))
	assert.Equal(t, "/x", toggleTrailingSlash("/x/"))
	assert.Equal(t, "/a/b/", toggleTrailingSlash("/a/b"))
	assert.Equal(t, "/", toggleTrailingSlash("/"))
	assert.Equal(t, "", toggleTrailingSlash(""))
}

func TestRedirectToCanonicalSlash(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	ok := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
	router.Get("/managed-cloud-services/", ok)
	router.Get("/fr/managed-cloud-services/", ok)
	router.Get("/managed-cloud-services/aws", ok)

	cases := []struct {
		path     string
		target   string
		redirect bool
	}{
		{"/fr/managed-cloud-services", "/fr/managed-cloud-services/", true},
		{"/managed-cloud-services", "/managed-cloud-services/", true},
		{"/managed-cloud-services/aws/", "/managed-cloud-services/aws", true},
		{"/managed-cloud-services/", "", false},
		{"/totally-unknown", "", false},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "https://example.com"+tc.path, nil)
		rec := httptest.NewRecorder()
		got := redirectToCanonicalSlash(router, rec, req)
		assert.Equal(t, tc.redirect, got, "path %s", tc.path)
		if tc.redirect {
			assert.Equal(t, http.StatusPermanentRedirect, rec.Code, "path %s", tc.path)
			assert.Equal(t, tc.target, rec.Header().Get("Location"), "path %s", tc.path)
		}
	}
}

func TestRedirectToCanonicalSlash_PreservesQuery(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	router.Get("/fr/articles/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "https://example.com/fr/articles?page=2", nil)
	rec := httptest.NewRecorder()

	assert.True(t, redirectToCanonicalSlash(router, rec, req))
	assert.Equal(t, "/fr/articles/?page=2", rec.Header().Get("Location"))
}

const (
	customNotFoundBody = "<h1>Custom 404</h1>"
)

func storeWithErrorPage() *templater_domain.MockManifestStoreView {
	entry := &templater_domain.MockPageEntryView{
		GetOriginalPathFunc: func() string { return "/pages/!404.pk" },
	}
	return &templater_domain.MockManifestStoreView{
		FindErrorPageFunc: func(_ int, _ string) (templater_domain.PageEntryView, bool) {
			return entry, true
		},
	}
}

func storeWithoutErrorPage() *templater_domain.MockManifestStoreView {
	return &templater_domain.MockManifestStoreView{
		FindErrorPageFunc: func(_ int, _ string) (templater_domain.PageEntryView, bool) {
			return nil, false
		},
	}
}

func depsRenderingBody(body string) *daemon_domain.HTTPHandlerDependencies {
	return &daemon_domain.HTTPHandlerDependencies{
		Templater: &templater_domain.MockTemplaterService{
			ProbePageFunc: func(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request, _ *config.WebsiteConfig) (*templater_dto.PageProbeResult, error) {
				return &templater_dto.PageProbeResult{}, nil
			},
			RenderPageFunc: func(_ context.Context, request templater_domain.RenderRequest) error {
				_, _ = request.Writer.Write([]byte(body))
				return nil
			},
		},
	}
}

func mountAppRouterUnderCatchAll(notFound http.Handler) chi.Router {
	appRouter := chi.NewRouter()
	appRouter.NotFound(notFound.ServeHTTP)

	parent := chi.NewRouter()
	parent.Group(func(r chi.Router) {
		r.Handle("/*", appRouter)
	})
	parent.NotFound(http.NotFound)
	return parent
}

func TestNewNotFoundHandler_AppRouterThroughCatchAll_RendersCustomPage(t *testing.T) {
	t.Parallel()

	handler := NewNotFoundHandler(
		chi.NewRouter(),
		depsRenderingBody(customNotFoundBody),
		storeWithErrorPage(),
		&config.WebsiteConfig{},
	)
	parent := mountAppRouterUnderCatchAll(handler)

	request := httptest.NewRequest(http.MethodGet, "/no-such-route", nil)
	recorder := httptest.NewRecorder()
	parent.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Contains(t, recorder.Body.String(), customNotFoundBody)
}

func TestNewNotFoundHandler_AppRouterThroughCatchAll_PlainWhenNoErrorPage(t *testing.T) {
	t.Parallel()

	handler := NewNotFoundHandler(
		chi.NewRouter(),
		depsRenderingBody(customNotFoundBody),
		storeWithoutErrorPage(),
		&config.WebsiteConfig{},
	)
	parent := mountAppRouterUnderCatchAll(handler)

	request := httptest.NewRequest(http.MethodGet, "/no-such-route", nil)
	recorder := httptest.NewRecorder()
	parent.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), customNotFoundBody)
	assert.Contains(t, recorder.Body.String(), "404 page not found")
}

func TestMountRoutesFromManifest_NotFound_RendersCustomPage(t *testing.T) {
	t.Parallel()

	appRouter := chi.NewRouter()
	MountRoutesFromManifest(context.Background(), &MountRoutesConfig{
		Router:        appRouter,
		Deps:          depsRenderingBody(customNotFoundBody),
		Store:         storeWithErrorPage(),
		SiteSettings:  &config.WebsiteConfig{},
		RouteSettings: RouteSettings{},
	})

	request := httptest.NewRequest(http.MethodGet, "/no-such-route", nil)
	recorder := httptest.NewRecorder()
	appRouter.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Contains(t, recorder.Body.String(), customNotFoundBody)
}

func TestMountRoutesFromManifest_NotFound_PlainWhenNoErrorPage(t *testing.T) {
	t.Parallel()

	appRouter := chi.NewRouter()
	MountRoutesFromManifest(context.Background(), &MountRoutesConfig{
		Router:        appRouter,
		Deps:          depsRenderingBody(customNotFoundBody),
		Store:         storeWithoutErrorPage(),
		SiteSettings:  &config.WebsiteConfig{},
		RouteSettings: RouteSettings{},
	})

	request := httptest.NewRequest(http.MethodGet, "/no-such-route", nil)
	recorder := httptest.NewRecorder()
	appRouter.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), customNotFoundBody)
	assert.Contains(t, recorder.Body.String(), "404 page not found")
}

func TestBuildRouter_InstallsNotFoundHandlerOnMainRouter(t *testing.T) {
	t.Parallel()

	notFoundHandler := NewNotFoundHandler(
		chi.NewRouter(),
		depsRenderingBody(customNotFoundBody),
		storeWithErrorPage(),
		&config.WebsiteConfig{},
	)

	builder := NewHTTPRouterBuilder(nil, "")
	mainRouter, err := builder.BuildRouter(
		&daemon_domain.RouterConfig{
			DistServePath:     "/_piko/dist",
			ArtefactServePath: "/_piko/artefacts",
			SecurityHeaders:   security_dto.SecurityHeadersValues{},
			RateLimit:         security_dto.RateLimitValues{},
		},
		daemon_domain.RouterDependencies{
			NotFoundHandler: notFoundHandler,
		},
	)
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/no-such-route", nil)
	recorder := httptest.NewRecorder()
	mainRouter.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Contains(t, recorder.Body.String(), customNotFoundBody)
}

func TestBuildRouter_NotFoundNamesTheRuntimeVersion(t *testing.T) {
	t.Parallel()

	require.NotEmpty(t, daemon_frontend.RuntimeVersion(), "the embedded bundles must have a runtime version")

	testCases := []struct {
		notFoundHandler http.Handler
		name            string
		build           daemon_frontend.CoreRuntimeBuild
	}{
		{name: "the app's not found page on a production server", notFoundHandler: NewNotFoundHandler(
			chi.NewRouter(), depsRenderingBody(customNotFoundBody), storeWithErrorPage(), &config.WebsiteConfig{}),
			build: daemon_frontend.CoreRuntimeBuildProduction},
		{name: "the plain not found response on a dev server", notFoundHandler: nil, build: daemon_frontend.CoreRuntimeBuildDevelopment},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			builder := NewHTTPRouterBuilder(nil, "")
			mainRouter, err := builder.BuildRouter(
				&daemon_domain.RouterConfig{DistServePath: "/_piko/dist", ArtefactServePath: "/_piko/artefacts", CoreRuntimeBuild: testCase.build},
				daemon_domain.RouterDependencies{NotFoundHandler: testCase.notFoundHandler},
			)
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			mainRouter.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/no-such-route", nil))

			assert.Equal(t, http.StatusNotFound, recorder.Code)
			assert.Equal(t, testCase.build.RuntimeVersion(), recorder.Header().Get(daemon_frontend.RuntimeVersionHeader),
				"the header must name the version in the core URL the server's pages load")
		})
	}
}

func TestApplyRequestTimeoutBoundsARoute(t *testing.T) {
	t.Parallel()

	t.Run("a bounded route sees a deadline", func(t *testing.T) {
		t.Parallel()

		var (
			hadDeadline bool
			cause       error
		)
		handler := applyRequestTimeout(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			_, hadDeadline = request.Context().Deadline()
			<-request.Context().Done()
			cause = context.Cause(request.Context())
		}), 10*time.Millisecond)

		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

		assert.True(t, hadDeadline, "the route must carry the configured deadline")
		require.Error(t, cause)
		assert.Contains(t, cause.Error(), "request exceeded")
	})

	t.Run("zero leaves the route unbounded", func(t *testing.T) {
		t.Parallel()

		var hadDeadline bool
		handler := applyRequestTimeout(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			_, hadDeadline = request.Context().Deadline()
		}), 0)

		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

		assert.False(t, hadDeadline)
	})
}

func TestRouteRequestTimeout(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 30*time.Second, routeRequestTimeout(RouteSettings{RequestTimeoutSeconds: 30}))
	assert.Zero(t, routeRequestTimeout(RouteSettings{}))
	assert.Zero(t, routeRequestTimeout(RouteSettings{RequestTimeoutSeconds: -1}))
}

func TestApplyMiddlewaresEnforcesAuthPolicyWithoutAGuard(t *testing.T) {
	t.Parallel()

	protectedPage := func() *templater_domain.MockPageEntryView {
		return &templater_domain.MockPageEntryView{
			GetHasAuthPolicyFunc: func() bool { return true },
			GetAuthPolicyFunc: func(*templater_dto.RequestData) daemon_dto.AuthPolicy {
				return daemon_dto.AuthPolicy{Required: true}
			},
		}
	}

	served := false
	base := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		served = true
		writer.WriteHeader(http.StatusOK)
	})

	handler := applyMiddlewares(base, protectedPage(), nil, nil)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/dashboard", nil))

	assert.False(t, served,
		"a page declaring AuthPolicy{Required: true} must not be served to an unauthenticated caller "+
			"just because the application configured no auth guard")
	assert.Equal(t, http.StatusSeeOther, recorder.Code)
}

func TestApplyMiddlewaresDoesNotCacheAnIdentityVaryingPage(t *testing.T) {
	t.Parallel()

	cacheApplied := false
	cacheMiddleware := func(next http.Handler) http.Handler {
		cacheApplied = true

		return next
	}

	protected := &templater_domain.MockPageEntryView{
		GetHasAuthPolicyFunc: func() bool { return true },
		GetAuthPolicyFunc: func(*templater_dto.RequestData) daemon_dto.AuthPolicy {
			return daemon_dto.AuthPolicy{Required: true}
		},
	}

	base := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	applyMiddlewares(base, protected, cacheMiddleware, nil)

	assert.False(t, cacheApplied,
		"the shared cache key does not vary on identity, so a page behind an AuthPolicy must bypass it")

	public := &templater_domain.MockPageEntryView{GetHasAuthPolicyFunc: func() bool { return false }}
	applyMiddlewares(base, public, cacheMiddleware, nil)

	assert.True(t, cacheApplied, "an ordinary page is still cached")
}

func TestApplyCachePolicyMiddlewareSendsNoStore(t *testing.T) {
	t.Parallel()

	passThroughCache := func(next http.Handler) http.Handler { return next }
	noStorePage := func(hasAuthPolicy bool) *templater_domain.MockPageEntryView {
		return &templater_domain.MockPageEntryView{
			GetHasAuthPolicyFunc:  func() bool { return hasAuthPolicy },
			GetHasCachePolicyFunc: func() bool { return true },
			GetCachePolicyFunc: func(*templater_dto.RequestData) templater_dto.CachePolicy {
				return templater_dto.CachePolicy{NoStore: true}
			},
		}
	}

	testCases := []struct {
		cacheMiddleware  func(http.Handler) http.Handler
		entry            *templater_domain.MockPageEntryView
		name             string
		wantCacheControl string
	}{
		{
			name:             "page behind an auth policy",
			cacheMiddleware:  passThroughCache,
			entry:            noStorePage(true),
			wantCacheControl: cacheControlNoStore,
		},
		{
			name:             "static page cache off",
			entry:            noStorePage(false),
			wantCacheControl: cacheControlNoStore,
		},
		{
			name:             "page without a cache policy",
			entry:            &templater_domain.MockPageEntryView{},
			wantCacheControl: cacheControlPrivateNoCache,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			page := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				setDefaultCacheControl(w.Header(), headerValCachePrivateNoCache)
				w.WriteHeader(http.StatusOK)
			})
			handler := applyCachePolicyMiddleware(page, testCase.entry, testCase.cacheMiddleware)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, newRoutedRequest(t, "/account"))

			assert.Equal(t, []string{testCase.wantCacheControl}, recorder.Header().Values("Cache-Control"))
		})
	}
}

func TestRolesFromAuthValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		value any
		want  []string
	}{
		{name: "a typed slice", value: []string{"admin", "editor"}, want: []string{"admin", "editor"}},
		{name: "a single role", value: "admin", want: []string{"admin"}},
		{name: "a decoded JSON claim", value: []any{"admin", "editor"}, want: []string{"admin", "editor"}},
		{name: "a mixed claim keeps the strings", value: []any{"admin", 7}, want: []string{"admin"}},
		{name: "no claim", value: nil, want: nil},
		{name: "an unusable claim", value: 42, want: nil},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, rolesFromAuthValue(testCase.value))
		})
	}
}

func TestProductionFallbackUsesRenderedErrorStatus(t *testing.T) {
	tests := []struct {
		err    error
		status int
	}{
		{err: daemon_dto.NotFound("book", "private-id"), status: http.StatusNotFound},
		{err: daemon_dto.Forbidden("private detail"), status: http.StatusForbidden},
		{err: errors.New("private detail"), status: http.StatusInternalServerError},
	}
	for _, phase := range []string{"probe", "render"} {
		for _, testCase := range tests {
			t.Run(fmt.Sprintf("%s/%d", phase, testCase.status), func(t *testing.T) {
				page := pageErrorContext{Entry: &mockPageEntryViewWithPath{}, Store: &templater_domain.MockManifestStoreView{}, Span: newNoopSpan()}
				request := httptest.NewRequest(http.MethodGet, "/book", nil)
				recorder := httptest.NewRecorder()
				if phase == "probe" {
					handlePageProbeError(request.Context(), recorder, request, testCase.err, page)
				} else {
					handlePageRenderError(request.Context(), recorder, request, testCase.err, page, false)
				}
				assert.Equal(t, testCase.status, recorder.Code)
				assert.Equal(t, http.StatusText(testCase.status)+"\n", recorder.Body.String(), "the body must not leak the error detail")
			})
		}
	}
}

const (
	cacheTestPagePath      = "/about"
	cacheTestPartialPath   = "/_piko/partials/card"
	cacheTestErrorPagePath = "/pages/!error.pk"
	cacheTestPageBody      = "<h1>About</h1>"
	cacheTestErrorBody     = "<h1>Something went wrong</h1>"
	cacheTestPagePolicy    = "public, max-age=60"
	cacheTestAppPolicy     = "no-store"
	cacheTestDefaultPolicy = "private, no-cache"
)

type cacheTestRoutes struct {
	templater       *templater_domain.MockTemplaterService
	appMiddleware   func(http.Handler) http.Handler
	pageMiddleware  func(http.Handler) http.Handler
	cacheMiddleware func(http.Handler) http.Handler
	pagePolicy      templater_dto.CachePolicy
	pageRoles       []string
}

func setsCacheControl(value string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			w.Header().Set("Cache-Control", value)
			next.ServeHTTP(w, request)
		})
	}
}

func renderingTemplater() *templater_domain.MockTemplaterService {
	probe := func(context.Context, templater_dto.PageDefinition, *http.Request, *config.WebsiteConfig) (*templater_dto.PageProbeResult, error) {
		return &templater_dto.PageProbeResult{}, nil
	}
	render := func(_ context.Context, request templater_domain.RenderRequest) error {
		body := cacheTestPageBody
		if request.Page.OriginalPath == cacheTestErrorPagePath {
			body = cacheTestErrorBody
		}
		_, err := request.Writer.Write([]byte(body))
		return err
	}
	return &templater_domain.MockTemplaterService{
		ProbePageFunc:    probe,
		ProbePartialFunc: probe,
		RenderPageFunc:   render,
		RenderPartialFunc: func(_ context.Context, request templater_domain.RenderRequest) error {
			_, err := request.Writer.Write([]byte(cacheTestPageBody))
			return err
		},
	}
}

func newCacheTestStore(routes cacheTestRoutes) *templater_domain.MockManifestStoreView {
	page := &templater_domain.MockPageEntryView{
		GetIsPageFunc:        func() bool { return true },
		GetOriginalPathFunc:  func() string { return "/pages/about.pk" },
		GetRoutePatternFunc:  func() string { return cacheTestPagePath },
		GetRoutePatternsFunc: func() map[string]string { return map[string]string{"en": cacheTestPagePath} },
		GetCachePolicyFunc:   func(*templater_dto.RequestData) templater_dto.CachePolicy { return routes.pagePolicy },
	}
	if routes.pageMiddleware != nil {
		page.GetHasMiddlewareFunc = func() bool { return true }
		page.GetMiddlewaresFunc = func() []func(http.Handler) http.Handler {
			return []func(http.Handler) http.Handler{routes.pageMiddleware}
		}
	}
	if len(routes.pageRoles) > 0 {
		page.GetHasAuthPolicyFunc = func() bool { return true }
		page.GetAuthPolicyFunc = func(*templater_dto.RequestData) daemon_dto.AuthPolicy {
			return daemon_dto.AuthPolicy{Required: true, Roles: routes.pageRoles}
		}
	}
	partial := &templater_domain.MockPageEntryView{
		GetOriginalPathFunc: func() string { return "/partials/card.pk" },
		GetRoutePatternFunc: func() string { return cacheTestPartialPath },
	}
	errorPage := &templater_domain.MockPageEntryView{
		GetOriginalPathFunc: func() string { return cacheTestErrorPagePath },
	}
	entries := map[string]templater_domain.PageEntryView{"pages/about.pk": page, "partials/card.pk": partial}

	return &templater_domain.MockManifestStoreView{
		GetKeysFunc: func() []string { return []string{"pages/about.pk", "partials/card.pk"} },
		GetPageEntryFunc: func(key string) (templater_domain.PageEntryView, bool) {
			entry, ok := entries[key]
			return entry, ok
		},
		FindErrorPageFunc: func(int, string) (templater_domain.PageEntryView, bool) { return errorPage, true },
	}
}

func newCacheTestRouter(t *testing.T, routes cacheTestRoutes) chi.Router {
	t.Helper()

	router := chi.NewRouter()
	if routes.appMiddleware != nil {
		router.Use(routes.appMiddleware)
	}
	MountRoutesFromManifest(t.Context(), &MountRoutesConfig{
		Router:          router,
		Deps:            &daemon_domain.HTTPHandlerDependencies{Templater: routes.templater},
		Store:           newCacheTestStore(routes),
		SiteSettings:    &config.WebsiteConfig{},
		CacheMiddleware: routes.cacheMiddleware,
		RouteSettings:   RouteSettings{},
	})
	return router
}

func serveCacheTestRequest(t *testing.T, router http.Handler, target string, auth daemon_dto.AuthContext) *httptest.ResponseRecorder {
	t.Helper()

	pctx := daemon_dto.AcquirePikoRequestCtx()
	t.Cleanup(func() { daemon_dto.ReleasePikoRequestCtx(pctx) })
	if auth != nil {
		pctx.CachedAuth = auth
	}
	request := httptest.NewRequestWithContext(daemon_dto.WithPikoRequestCtx(t.Context(), pctx), http.MethodGet, target, nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestPageRoutesDefaultCacheControl(t *testing.T) {
	t.Parallel()

	redirecting := renderingTemplater()
	redirecting.RenderPageFunc = func(context.Context, templater_domain.RenderRequest) error {
		return &templater_dto.RedirectRequired{Metadata: templater_dto.InternalMetadata{
			Metadata: templater_dto.Metadata{ClientRedirect: "/login"},
		}}
	}

	failingRender := renderingTemplater()
	failingRender.RenderPageFunc = func(_ context.Context, request templater_domain.RenderRequest) error {
		if request.Page.OriginalPath == cacheTestErrorPagePath {
			_, err := request.Writer.Write([]byte(cacheTestErrorBody))
			return err
		}
		return errors.New("render failed")
	}

	failingProbe := renderingTemplater()
	failingProbe.ProbePageFunc = func(
		_ context.Context, page templater_dto.PageDefinition, _ *http.Request, _ *config.WebsiteConfig,
	) (*templater_dto.PageProbeResult, error) {
		if page.OriginalPath == cacheTestErrorPagePath {
			return &templater_dto.PageProbeResult{}, nil
		}
		return nil, errors.New("probe failed")
	}

	testCases := []struct {
		auth             daemon_dto.AuthContext
		name             string
		target           string
		wantCacheControl string
		wantBody         string
		routes           cacheTestRoutes
		wantStatus       int
	}{
		{
			name:             "page without a policy gets the default",
			routes:           cacheTestRoutes{templater: renderingTemplater()},
			target:           cacheTestPagePath,
			wantStatus:       http.StatusOK,
			wantCacheControl: cacheTestDefaultPolicy,
			wantBody:         cacheTestPageBody,
		},
		{
			name:             "fragment request gets the default",
			routes:           cacheTestRoutes{templater: renderingTemplater()},
			target:           cacheTestPagePath + "?_f=1",
			wantStatus:       http.StatusOK,
			wantCacheControl: cacheTestDefaultPolicy,
			wantBody:         cacheTestPageBody,
		},
		{
			name:             "partial gets the default",
			routes:           cacheTestRoutes{templater: renderingTemplater()},
			target:           cacheTestPartialPath,
			wantStatus:       http.StatusOK,
			wantCacheControl: cacheTestDefaultPolicy,
			wantBody:         cacheTestPageBody,
		},
		{
			name:             "page that sets its own policy keeps it",
			routes:           cacheTestRoutes{templater: renderingTemplater(), pageMiddleware: setsCacheControl(cacheTestPagePolicy)},
			target:           cacheTestPagePath,
			wantStatus:       http.StatusOK,
			wantCacheControl: cacheTestPagePolicy,
			wantBody:         cacheTestPageBody,
		},
		{
			name:             "app that sets its own policy keeps it",
			routes:           cacheTestRoutes{templater: renderingTemplater(), appMiddleware: setsCacheControl(cacheTestAppPolicy)},
			target:           cacheTestPagePath,
			wantStatus:       http.StatusOK,
			wantCacheControl: cacheTestAppPolicy,
			wantBody:         cacheTestPageBody,
		},
		{
			name:             "app policy also wins on a partial",
			routes:           cacheTestRoutes{templater: renderingTemplater(), appMiddleware: setsCacheControl(cacheTestAppPolicy)},
			target:           cacheTestPartialPath,
			wantStatus:       http.StatusOK,
			wantCacheControl: cacheTestAppPolicy,
			wantBody:         cacheTestPageBody,
		},
		{
			name:             "redirect gets the default",
			routes:           cacheTestRoutes{templater: redirecting},
			target:           cacheTestPagePath,
			wantStatus:       http.StatusFound,
			wantCacheControl: cacheTestDefaultPolicy,
		},
		{
			name:             "error page after a failed render gets the default",
			routes:           cacheTestRoutes{templater: failingRender},
			target:           cacheTestPagePath,
			wantStatus:       http.StatusInternalServerError,
			wantCacheControl: cacheTestDefaultPolicy,
			wantBody:         cacheTestErrorBody,
		},
		{
			name:             "error page after a failed probe gets the default",
			routes:           cacheTestRoutes{templater: failingProbe},
			target:           cacheTestPagePath,
			wantStatus:       http.StatusInternalServerError,
			wantCacheControl: cacheTestDefaultPolicy,
			wantBody:         cacheTestErrorBody,
		},
		{
			name:             "not found page gets the default",
			routes:           cacheTestRoutes{templater: renderingTemplater()},
			target:           "/no-such-page",
			wantStatus:       http.StatusNotFound,
			wantCacheControl: cacheTestDefaultPolicy,
			wantBody:         cacheTestErrorBody,
		},
		{
			name:             "page refused for a missing role gets the default",
			routes:           cacheTestRoutes{templater: renderingTemplater(), pageRoles: []string{"admin"}},
			auth:             headerAuthContext{userID: "ada"},
			target:           cacheTestPagePath,
			wantStatus:       http.StatusForbidden,
			wantCacheControl: cacheTestDefaultPolicy,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			router := newCacheTestRouter(t, testCase.routes)

			recorder := serveCacheTestRequest(t, router, testCase.target, testCase.auth)

			assert.Equal(t, testCase.wantStatus, recorder.Code)
			assert.Equal(t, []string{testCase.wantCacheControl}, recorder.Header().Values("Cache-Control"))
			if testCase.wantBody != "" {
				assert.Contains(t, recorder.Body.String(), testCase.wantBody)
			}
		})
	}
}

func TestPageRoutesStaticCachedPageKeepsItsOwnCacheControl(t *testing.T) {
	t.Parallel()

	routes := cacheTestRoutes{
		templater:  renderingTemplater(),
		pagePolicy: templater_dto.CachePolicy{Enabled: true, Static: true, MaxAgeSeconds: 3600},
	}
	store := newCacheTestStore(routes)
	cacheMiddleware := NewCacheMiddleware(
		CacheMiddlewareConfig{PersistenceDisabled: true},
		store,
		&registry_domain.MockRegistryService{
			GetArtefactFunc: func(context.Context, string) (*registry_dto.ArtefactMeta, error) {
				return nil, errors.New("not cached yet")
			},
		},
		nil,
		"",
	)
	routes.cacheMiddleware = cacheMiddleware.Handle
	router := newCacheTestRouter(t, routes)

	recorder := serveCacheTestRequest(t, router, cacheTestPagePath, nil)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "MISS", recorder.Header().Get("X-Cache-Status"))
	assert.Equal(t, []string{"public, max-age=0, s-maxage=3600, must-revalidate"}, recorder.Header().Values("Cache-Control"))
	assert.Equal(t, cacheTestPageBody, recorder.Body.String())
}
