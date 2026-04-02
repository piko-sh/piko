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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/security/security_dto"
)

func TestMiddleware_CORS_PreflightRequest(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)

	userRouter := chi.NewRouter()
	userRouter.Get("/api/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
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

	request := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	request.Header.Set("Origin", "http://localhost:8080")
	request.Header.Set("Access-Control-Request-Method", "GET")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)

	assert.NotEmpty(t, recorder.Header().Get("Access-Control-Allow-Origin"))
}

func TestMiddleware_CORS_AllowedHeaders(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)

	userRouter := chi.NewRouter()
	userRouter.Get("/api/data", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data": "test"}`))
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

	request := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	request.Header.Set("Origin", "http://localhost:8080")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	AssertStatus(t, recorder, http.StatusOK)

	assert.NotEmpty(t, recorder.Header().Get("Access-Control-Allow-Origin"))
}

func TestMiddleware_RequestID_Generated(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)

	var capturedRequestID string
	userRouter := chi.NewRouter()
	userRouter.Get("/test", func(w http.ResponseWriter, r *http.Request) {

		capturedRequestID = security_dto.RequestIDFromContext(r.Context())
		_, _ = w.Write([]byte("ok"))
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

	AssertStatus(t, recorder, http.StatusOK)

	assert.NotEmpty(t, capturedRequestID, "RequestID middleware should set a request ID in context")
}

func TestMiddleware_Heartbeat_Ping(t *testing.T) {
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

func TestMiddleware_RealIP_XForwardedFor(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)

	h.ServerConfig.Security.RateLimit.TrustedProxies = []string{"192.0.2.0/24"}

	var capturedIP string
	userRouter := chi.NewRouter()
	userRouter.Get("/ip-test", func(w http.ResponseWriter, r *http.Request) {
		capturedIP = r.RemoteAddr
		_, _ = w.Write([]byte("ok"))
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

	request := httptest.NewRequest(http.MethodGet, "/ip-test", nil)
	request.Header.Set("X-Forwarded-For", "192.168.1.100")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	AssertStatus(t, recorder, http.StatusOK)

	assert.Contains(t, capturedIP, "192.168.1.100")
}

func TestMiddleware_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)

	userRouter := chi.NewRouter()
	userRouter.Get("/only-get", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
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

	request := httptest.NewRequest(http.MethodPost, "/only-get", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	AssertStatus(t, recorder, http.StatusMethodNotAllowed)
}

func TestMiddleware_NotFound(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)

	userRouter := chi.NewRouter()
	userRouter.Get("/exists", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
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

	request := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	AssertStatus(t, recorder, http.StatusNotFound)
}

func TestRouterBuilder_CreatesRouter(t *testing.T) {
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
	assert.NotNil(t, router)
}

func TestRouterBuilder_WithUserRouter(t *testing.T) {
	t.Parallel()

	h := NewTestHarness(t)

	userRouter := chi.NewRouter()
	userRouter.Get("/custom", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("custom handler"))
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

	request := httptest.NewRequest(http.MethodGet, "/custom", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	AssertStatus(t, recorder, http.StatusOK)
	assert.Equal(t, "custom handler", recorder.Body.String())
}

func TestRouterBuilder_StaticRoutes(t *testing.T) {
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

	testCases := []struct {
		path           string
		expectedStatus int
	}{
		{path: "/ping", expectedStatus: http.StatusOK},
		{path: "/theme.css", expectedStatus: http.StatusNotFound},
		{path: "/robots.txt", expectedStatus: http.StatusNotFound},
		{path: "/sitemap.xml", expectedStatus: http.StatusNotFound},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tc.path, nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			AssertStatus(t, recorder, tc.expectedStatus)
		})
	}
}
