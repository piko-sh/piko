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
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/analytics/analytics_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

type registeringInterpreterPool struct{}

type registeringInterpreter struct{}

func (*registeringInterpreterPool) LoadModules(context.Context) error { return nil }

func (*registeringInterpreterPool) Get() (templater_domain.InterpreterPort, error) {
	return &registeringInterpreter{}, nil
}

func (*registeringInterpreter) CompileAndExecute(_ context.Context, modulePath string, packages map[string]map[string]string) error {
	for packagePath := range packages {
		templater_domain.RegisterASTFunc(modulePath+"/"+packagePath, func(*templater_dto.RequestData, any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
			return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, nil
		})
	}
	return nil
}

func (*registeringInterpreter) HasRegisteredPackage(string) bool { return false }

func (*registeringInterpreter) CompilesFile(string, []byte) (bool, error) { return true, nil }

func TestBuildDevInterpretedDaemon_AssemblesWithAnInterpreterPool(t *testing.T) {
	baseDir := newTestProject(t, map[string]string{"pages/index.pk": testProjectIndexPage})
	c := newResolvedContainer(t, baseDir, WithDevWidget(), WithDevHotreload())

	daemon, err := Daemon(context.Background(), runModeDevInterpreted, c, &Dependencies{
		AppRouter:       chi.NewRouter(),
		InterpreterPool: &registeringInterpreterPool{},
	})

	require.NoError(t, err)
	assert.NotNil(t, daemon)
}

func TestBuildDevInterpretedDaemon_FailsWhenThePagesCannotBeLinked(t *testing.T) {
	baseDir := newTestProject(t, map[string]string{"pages/index.pk": testProjectIndexPage})
	c := newResolvedContainer(t, baseDir)

	daemon, err := Daemon(context.Background(), runModeDevInterpreted, c, &Dependencies{AppRouter: chi.NewRouter()})

	require.Error(t, err)
	assert.Nil(t, daemon)
	assert.Contains(t, err.Error(), "initial blocking JIT build failed")
}

func TestInterpretedStartupDiscoversConfiguredPDFDirectory(t *testing.T) {
	root, pdfs := t.TempDir(), "invoices"
	require.NoError(t, os.Mkdir(filepath.Join(root, pdfs), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(root, pdfs, "order.pk"), []byte("<template><p>Invoice</p></template>"), 0600))
	c := NewContainer()
	c.serverConfig.Paths.BaseDir, c.serverConfig.Paths.PdfsSourceDir = &root, &pdfs
	b := interpretedDaemonBuilder{c: c, resolver: &resolver_domain.MockResolver{GetModuleNameFunc: func() string { return "example.local/shop" }}}
	entries, err := b.discoverInitialEntryPoints(t.Context())
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "example.local/shop/invoices/order.pk", entries[0].Path)
	assert.True(t, entries[0].IsPdf)
	assert.False(t, entries[0].IsPublic || entries[0].IsPage || entries[0].IsEmail)
}

type invalidatingRenderRegistry struct {
	render_domain.MockRegistryPort
}

func (*invalidatingRenderRegistry) InvalidateArtefact(context.Context, string) {}

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
	eventBus := &orchestrator_domain.MockEventBus{}
	c.eventBusOverride = eventBus
	c.SetRenderRegistryOverride(&invalidatingRenderRegistry{})
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
	assert.Equal(t, int64(len(registry_domain.ArtefactTopics)), eventBus.SubscribeWithHandlerCallCount.Load(),
		"artefact events must still reach the render registry's caches without a metadata cache")
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

func TestResolveRenderingServicesFailsWhenCSRFServiceFails(t *testing.T) {
	t.Parallel()

	setupErr := errors.New("csrf setup failed")
	c := NewContainer()
	c.rendererOverride = &stubRenderService{}
	c.renderRegistryOverride = &stubRenderRegistryPort{}
	c.csrfOnce.Do(func() { c.csrfErr = setupErr })

	builder := &interpretedDaemonBuilder{c: c}

	err := builder.resolveRenderingServices()

	require.ErrorIs(t, err, setupErr)
	assert.Nil(t, builder.csrfService)
}
