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
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/analytics/analytics_domain"
	"piko.sh/piko/internal/analytics/analytics_dto"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
	"piko.sh/piko/internal/templater/templater_domain"
)

const (
	testUserHeader = "X-Test-User"
)

type headerAuthContext struct {
	userID string
}

func (a headerAuthContext) IsAuthenticated() bool { return a.userID != "" }

func (a headerAuthContext) UserID() string { return a.userID }

func (headerAuthContext) Get(string) any { return nil }

type headerAuthProvider struct{}

func (headerAuthProvider) Authenticate(_ context.Context, r *http.Request) (daemon_dto.AuthContext, error) {
	user := r.Header.Get(testUserHeader)
	if user == "" {
		return nil, nil
	}
	return headerAuthContext{userID: user}, nil
}

type pageRouteProvider struct{}

func (pageRouteProvider) MountRoutes(router *chi.Mux) {
	router.Get("/members", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("members"))
	})
}

type recordingCollector struct {
	events []analytics_dto.Event
	mu     sync.Mutex
}

func (c *recordingCollector) Collect(_ context.Context, ev *analytics_dto.Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, *ev)
	return nil
}

func (*recordingCollector) Start(context.Context) {}

func (*recordingCollector) Flush(context.Context) error { return nil }

func (*recordingCollector) Close(context.Context) error { return nil }

func (*recordingCollector) Name() string { return "recording" }

func (c *recordingCollector) userIDs() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	ids := make([]string, 0, len(c.events))
	for i := range c.events {
		ids = append(ids, c.events[i].UserID)
	}
	return ids
}

func newParityRouterManagerConfig(
	rateLimitService security_domain.RateLimitService,
	analyticsService *analytics_domain.Service,
) *RouterManagerConfig {
	routerConfig := &daemon_domain.RouterConfig{}
	routerConfig.RateLimit = security_dto.RateLimitValues{
		Enabled: true,
		Global:  security_dto.RateLimitTierValues{RequestsPerMinute: 600, BurstSize: 60},
		Actions: security_dto.RateLimitTierValues{RequestsPerMinute: 60, BurstSize: 6},
	}
	routerConfig.MaxConcurrentRequests = 16
	routerConfig.RequestTimeoutSeconds = 30

	managerConfig := &RouterManagerConfig{}
	managerConfig.CSRFService = &security_domain.MockCSRFTokenService{}
	managerConfig.RegistryService = &registry_domain.MockRegistryService{}
	managerConfig.VariantGenerator = &daemon_domain.MockOnDemandVariantGenerator{}
	managerConfig.Deps = &daemon_domain.HTTPHandlerDependencies{}
	managerConfig.SiteSettings = &config.WebsiteConfig{}
	managerConfig.Actions = map[string]ActionHandlerEntry{}
	managerConfig.AppRouter = chi.NewRouter()
	managerConfig.RouteProviders = []daemon_domain.RouteProvider{pageRouteProvider{}}
	managerConfig.RouterConfig = routerConfig
	managerConfig.AuthProvider = headerAuthProvider{}
	managerConfig.AuthGuardConfig = &daemon_dto.AuthGuardConfig{LoginPath: "/login"}
	managerConfig.AnalyticsService = analyticsService
	managerConfig.RateLimitService = rateLimitService
	managerConfig.RateLimitConfig = routerConfig.RateLimit
	managerConfig.PublicDownloadHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("public"))
	})
	managerConfig.InstanceRelease = "release-7"

	return managerConfig
}

func serveThroughCarrier(handler http.Handler, request *http.Request) *httptest.ResponseRecorder {
	pctx := daemon_dto.AcquirePikoRequestCtx()
	defer daemon_dto.ReleasePikoRequestCtx(pctx)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request.WithContext(daemon_dto.WithPikoRequestCtx(request.Context(), pctx)))

	return recorder
}

func TestRouterManager_ReloadRoutesInstallsTheRequestPathServices(t *testing.T) {
	t.Parallel()

	rateLimitService := &security_domain.MockRateLimitService{
		CheckLimitFunc: func(context.Context, string, int, time.Duration) (ratelimiter_dto.Result, error) {
			return ratelimiter_dto.Result{Allowed: true, Limit: 600, Remaining: 599}, nil
		},
	}
	collector := &recordingCollector{}
	analyticsService := analytics_domain.NewService([]analytics_domain.Collector{collector})
	analyticsService.Start(context.Background())

	manager := NewRouterManager(newParityRouterManagerConfig(rateLimitService, analyticsService))
	t.Cleanup(manager.Close)
	require.NoError(t, manager.ReloadRoutes(context.Background(), &templater_domain.MockManifestStoreView{}))
	assert.Equal(t, "release-7", manager.instanceRelease)

	testCases := []struct {
		name         string
		path         string
		user         string
		wantBody     string
		wantLocation string
		wantStatus   int
	}{
		{
			name:         "the auth guard redirects an anonymous visitor",
			path:         "/members",
			wantStatus:   http.StatusSeeOther,
			wantLocation: "/login",
		},
		{
			name:       "an authenticated visitor reaches the page",
			path:       "/members",
			user:       "user-1",
			wantStatus: http.StatusOK,
			wantBody:   "members",
		},
		{
			name:       "public storage downloads are served",
			path:       "/_piko/storage/public/file.txt",
			wantStatus: http.StatusOK,
			wantBody:   "public",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if tc.user != "" {
				request.Header.Set(testUserHeader, tc.user)
			}

			response := serveThroughCarrier(manager, request)

			assert.Equal(t, tc.wantStatus, response.Code)
			if tc.wantBody != "" {
				assert.Equal(t, tc.wantBody, response.Body.String())
			}
			if tc.wantLocation != "" {
				assert.Contains(t, response.Header().Get("Location"), tc.wantLocation)
			}
		})
	}

	require.NoError(t, analyticsService.Close(context.Background()))
	assert.Contains(t, collector.userIDs(), "user-1", "analytics records the authenticated visitor")
	assert.Positive(t, rateLimitService.CheckLimitCallCount.Load(), "rate limiting checks each request")
}

func TestRouterManager_ReloadRoutesRefusesRateLimitingWithoutAService(t *testing.T) {
	t.Parallel()

	manager := NewRouterManager(newParityRouterManagerConfig(nil, nil))
	t.Cleanup(manager.Close)

	err := manager.ReloadRoutes(context.Background(), &templater_domain.MockManifestStoreView{})

	require.ErrorIs(t, err, errRateLimitServiceMissing)
	response := serveThroughCarrier(manager, httptest.NewRequest(http.MethodGet, "/members", nil))
	assert.Equal(t, http.StatusServiceUnavailable, response.Code, "no router is installed after a failed build")
}

func routeTrace(label string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Trace", label)
			next.ServeHTTP(w, r)
		})
	}
}

func applicationRouteFixture() *chi.Mux {
	router := chi.NewRouter()
	router.Use(routeTrace("root"))
	router.Get("/custom/{id}", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(chi.URLParam(r, "id") + ":" + r.URL.Query().Get("q")))
	})
	router.With(routeTrace("inline")).Get("/inline", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	router.Route("/api", func(child chi.Router) {
		child.Use(routeTrace("child"))
		child.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
		child.Get("/{id}", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(chi.URLParam(r, "id"))) })
		child.NotFound(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusGone) })
		child.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotImplemented) })
	})
	router.Mount("/opaque", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.URL.Path))
	}))
	router.Route("/number/{id:[0-9]+}", func(child chi.Router) {
		child.Get("/", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(chi.URLParam(r, "id"))) })
	})
	router.Get("/api", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusCreated) })
	router.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) })
	router.HandleFunc("/override", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusAccepted) })
	router.Get("/override", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusCreated) })
	return router
}

func TestRouterManagerReloadPreservesApplicationRoutes(t *testing.T) {
	t.Parallel()
	managerConfig := newParityRouterManagerConfig(nil, nil)
	managerConfig.AppRouter = applicationRouteFixture()
	manager := NewRouterManager(managerConfig)
	t.Cleanup(manager.Close)
	cases := []struct{ method, path string }{
		{http.MethodGet, "/custom/value?q=query"},
		{http.MethodGet, "/custom/encoded%2Fvalue?q=query"},
		{http.MethodGet, "/inline"},
		{http.MethodGet, "/api"},
		{http.MethodGet, "/number/123"},
		{http.MethodGet, "/number/123/"},
		{http.MethodGet, "/number/123/missing"},
		{http.MethodGet, "/api/"},
		{http.MethodGet, "/api/value"},
		{http.MethodGet, "/api/absent/path"},
		{http.MethodPost, "/api/value"},
		{http.MethodGet, "/opaque"},
		{http.MethodGet, "/opaque/"},
		{http.MethodGet, "/opaque/file"},
		{http.MethodGet, "/override"},
		{http.MethodPost, "/override"},
		{http.MethodPost, "/custom/value"},
	}
	for range 3 {
		reloaded, _ := manager.buildReloadedAppRouter(t.Context(), &templater_domain.MockManifestStoreView{})
		for _, test := range cases {
			original := httptest.NewRecorder()
			managerConfig.AppRouter.ServeHTTP(original, httptest.NewRequest(test.method, test.path, nil))
			actual := httptest.NewRecorder()
			reloaded.ServeHTTP(actual, httptest.NewRequest(test.method, test.path, nil))
			require.Equal(t, original.Code, actual.Code, "%s %s", test.method, test.path)
			require.Equal(t, original.Body.String(), actual.Body.String(), "%s %s", test.method, test.path)
			require.Equal(t, original.Header(), actual.Header(), "%s %s", test.method, test.path)
		}
	}
}

func TestRouterManagerReloadProtectsCustomRoutesWithTheAuthGuard(t *testing.T) {
	t.Parallel()
	managerConfig := newParityRouterManagerConfig(nil, nil)
	managerConfig.RouterConfig.RateLimit.Enabled = false
	managerConfig.RateLimitConfig.Enabled = false
	managerConfig.AppRouter = applicationRouteFixture()
	manager := NewRouterManager(managerConfig)
	t.Cleanup(manager.Close)
	for range 2 {
		require.NoError(t, manager.ReloadRoutes(context.Background(), &templater_domain.MockManifestStoreView{}))
		request := httptest.NewRequest(http.MethodGet, "/custom/value", nil)
		response := serveThroughCarrier(manager, request)
		require.Equal(t, http.StatusSeeOther, response.Code)
		require.Contains(t, response.Header().Get("Location"), "/login")
		request.Header.Set(testUserHeader, "user")
		response = serveThroughCarrier(manager, request)
		require.Equal(t, http.StatusOK, response.Code)
		require.Equal(t, "value:", response.Body.String())
	}
}

type changingRouteProvider struct{ path string }

func (provider *changingRouteProvider) MountRoutes(router *chi.Mux) {
	router.Get(provider.path, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("generated"))
	})
}

func TestRouterManagerReloadKeepsGeneratedRoutersIndependent(t *testing.T) {
	t.Parallel()
	managerConfig := newParityRouterManagerConfig(nil, nil)
	managerConfig.AppRouter = applicationRouteFixture()
	provider := &changingRouteProvider{path: "/before"}
	managerConfig.RouteProviders = []daemon_domain.RouteProvider{provider}
	manager := NewRouterManager(managerConfig)
	t.Cleanup(manager.Close)
	before, _ := manager.buildReloadedAppRouter(t.Context(), &templater_domain.MockManifestStoreView{})
	provider.path = "/after"
	after, _ := manager.buildReloadedAppRouter(t.Context(), &templater_domain.MockManifestStoreView{})
	for _, test := range []struct {
		router http.Handler
		path   string
		status int
	}{
		{before, "/before", http.StatusOK},
		{before, "/after", http.StatusNotFound},
		{after, "/before", http.StatusNotFound},
		{after, "/after", http.StatusOK},
		{managerConfig.AppRouter, "/before", http.StatusNotFound},
		{managerConfig.AppRouter, "/after", http.StatusNotFound},
	} {
		recorder := httptest.NewRecorder()
		test.router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
		require.Equal(t, test.status, recorder.Code, test.path)
		require.Equal(t, []string{"root"}, recorder.Header().Values("X-Trace"))
	}
}

func TestRouterManagerReloadKeepsApplicationMethodsAtGeneratedPaths(t *testing.T) {
	t.Parallel()
	managerConfig := newParityRouterManagerConfig(nil, nil)
	managerConfig.AppRouter = applicationRouteFixture()
	managerConfig.AppRouter.Post("/override", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	managerConfig.RouteProviders = []daemon_domain.RouteProvider{&changingRouteProvider{path: "/override"}}
	manager := NewRouterManager(managerConfig)
	t.Cleanup(manager.Close)
	reloaded, _ := manager.buildReloadedAppRouter(t.Context(), &templater_domain.MockManifestStoreView{})
	response := httptest.NewRecorder()
	reloaded.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/override", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "generated", response.Body.String())
	require.Equal(t, []string{"root"}, response.Header().Values("X-Trace"))
	response = httptest.NewRecorder()
	reloaded.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/override", nil))
	require.Equal(t, http.StatusNoContent, response.Code)
	require.Equal(t, []string{"root"}, response.Header().Values("X-Trace"))
}

func TestRouterManagerReloadPreservesConcurrentApplicationRequests(t *testing.T) {
	t.Parallel()
	managerConfig := newParityRouterManagerConfig(nil, nil)
	managerConfig.RouterConfig.RateLimit.Enabled = false
	managerConfig.RateLimitConfig.Enabled = false
	managerConfig.AppRouter = applicationRouteFixture()
	manager := NewRouterManager(managerConfig)
	t.Cleanup(manager.Close)
	require.NoError(t, manager.ReloadRoutes(t.Context(), &templater_domain.MockManifestStoreView{}))
	const workers = 4
	const requests = 20
	responses := make(chan *httptest.ResponseRecorder, workers*requests)
	var group sync.WaitGroup
	for range workers {
		group.Go(func() {
			for range requests {
				request := httptest.NewRequest(http.MethodGet, "/custom/value?q=query", nil)
				request.Header.Set(testUserHeader, "user")
				responses <- serveThroughCarrier(manager, request)
			}
		})
	}
	for range 4 {
		require.NoError(t, manager.ReloadRoutes(t.Context(), &templater_domain.MockManifestStoreView{}))
	}
	group.Wait()
	close(responses)
	for response := range responses {
		require.Equal(t, http.StatusOK, response.Code)
		require.Equal(t, "value:query", response.Body.String())
		require.Equal(t, []string{"root"}, response.Header().Values("X-Trace"))
	}
}

func TestRouterManagerReloadPreservesChiRoutePrecedence(t *testing.T) {
	t.Parallel()
	cases := []struct {
		application, generated, path, body string
	}{
		{"/status", "/{page}", "/status", "application"},
		{"/{page}", "/status", "/status", "generated"},
		{"/api/*", "/api/status", "/api/status", "generated"},
		{"/api/status", "/api/*", "/api/status", "application"},
		{"/{id:[0-9]+}", "/{page}", "/123", "application"},
		{"/same/{first}", "/same/{second}", "/same/value", "generated"},
	}
	for _, test := range cases {
		t.Run(test.application+"_"+test.generated, func(t *testing.T) {
			t.Parallel()
			managerConfig := newParityRouterManagerConfig(nil, nil)
			managerConfig.AppRouter = chi.NewRouter()
			managerConfig.AppRouter.Get(test.application, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("application"))
			})
			managerConfig.RouteProviders = []daemon_domain.RouteProvider{&changingRouteProvider{path: test.generated}}
			manager := NewRouterManager(managerConfig)
			t.Cleanup(manager.Close)
			router, _ := manager.buildReloadedAppRouter(t.Context(), &templater_domain.MockManifestStoreView{})
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
			require.Equal(t, http.StatusOK, response.Code)
			require.Equal(t, test.body, response.Body.String())
		})
	}
}
