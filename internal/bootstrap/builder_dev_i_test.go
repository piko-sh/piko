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

package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/analytics/analytics_dto"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/security/security_domain"
)

type noAuthProvider struct{}

func (noAuthProvider) Authenticate(context.Context, *http.Request) (daemon_dto.AuthContext, error) {
	return nil, nil
}

type discardingCollector struct{}

func (discardingCollector) Collect(context.Context, *analytics_dto.Event) error { return nil }

func (discardingCollector) Start(context.Context) {}

func (discardingCollector) Flush(context.Context) error { return nil }

func (discardingCollector) Close(context.Context) error { return nil }

func (discardingCollector) Name() string { return "discarding" }

func TestInterpretedDaemonBuilder_RouterManagerConfigMatchesTheCompiledRouter(t *testing.T) {
	c := NewContainer()
	c.authProvider = noAuthProvider{}
	c.authGuardConfig = &daemon_dto.AuthGuardConfig{LoginPath: "/login"}
	c.releaseIDOverride = "release-override"
	c.SetStorageService(&presignConfiguredStorage{config: newPresignConfig("secret", 0)})
	rateLimitService := &security_domain.MockRateLimitService{}
	c.rateLimitServiceOnce.Do(func() {})
	c.rateLimitService = rateLimitService
	c.cacheOnce.Do(func() {})
	c.cacheErr = errors.New("cache unavailable")
	c.AddAnalyticsCollector(discardingCollector{})
	analyticsService := c.GetAnalyticsService()
	require.NotNil(t, analyticsService)
	t.Cleanup(func() { _ = analyticsService.Close(context.Background()) })

	builder := &interpretedDaemonBuilder{}
	builder.c = c
	builder.deps = &Dependencies{AppRouter: chi.NewRouter()}

	managerConfig, err := builder.newRouterManagerConfig(context.Background())
	require.NoError(t, err)

	assert.Equal(t, c.authProvider, managerConfig.AuthProvider)
	assert.Same(t, c.authGuardConfig, managerConfig.AuthGuardConfig)
	assert.Same(t, analyticsService, managerConfig.AnalyticsService)
	assert.Same(t, rateLimitService, managerConfig.RateLimitService)
	assert.Equal(t, NewRateLimitValues(&c.serverConfig.Security.RateLimit), managerConfig.RateLimitConfig)
	assert.NotNil(t, managerConfig.PresignUploadHandler)
	assert.NotNil(t, managerConfig.PresignDownloadHandler)
	assert.NotNil(t, managerConfig.PublicDownloadHandler)
	assert.Equal(t, "release-override", managerConfig.InstanceRelease)
	assert.Same(t, builder.deps.AppRouter, managerConfig.AppRouter)
	require.NotNil(t, managerConfig.RouterConfig)
	assert.True(t, managerConfig.RouterConfig.DisableHTTPCache)
	assert.Nil(t, managerConfig.ArtefactCache, "no cache service means no metadata cache")
	assert.Nil(t, managerConfig.ActionResponseCache, "no cache service means no response cache")
}

func TestInterpretedDaemonBuilder_RouterManagerConfigNeedsTheRateLimitService(t *testing.T) {
	c := NewContainer()
	c.rateLimitServiceOnce.Do(func() {})
	c.rateLimitServiceErr = errors.New("limiter unavailable")

	builder := &interpretedDaemonBuilder{}
	builder.c = c
	builder.deps = &Dependencies{AppRouter: chi.NewRouter()}

	managerConfig, err := builder.newRouterManagerConfig(context.Background())

	require.Error(t, err)
	assert.Nil(t, managerConfig)
}
