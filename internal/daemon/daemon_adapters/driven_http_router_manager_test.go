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
