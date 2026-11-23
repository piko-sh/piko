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

package main_test

import (
	"bytes"
	"context"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_adapters"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/generator/generator_adapters"
	"piko.sh/piko/internal/i18n/i18n_adapters"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/render/render_adapters"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/security/security_adapters"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/templater/templater_adapters"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
	"piko.sh/safedisk"

	_ "example.org/piko-fixture/dist"
)

func BenchmarkCompiledRender(b *testing.B) {
	ctx := b.Context()
	page := os.Getenv("PIKO_BENCH_TARGET_PAGE")
	require.NotEmpty(b, page)
	store, err := templater_adapters.NewManifestStore(ctx, generator_adapters.NewFlatBufferManifestProvider(filepath.Join("dist", "manifest.bin")))
	require.NoError(b, err)
	sandbox, err := safedisk.NewNoOpSandbox(".", safedisk.ModeReadOnly)
	require.NoError(b, err)
	b.Cleanup(func() { _ = sandbox.Close() })
	i18n, err := i18n_adapters.NewFSService(ctx, sandbox, "en", "")
	require.NoError(b, err)

	writer := security_adapters.NewSecureCookieWriter(security_adapters.CookieSecurityValues{}, false)
	csrf, err := security_domain.NewCSRFTokenService(security_domain.SecurityConfig{HMACSecretKey: []byte("test-secret-key-32-bytes-long")}, security_adapters.NewIPBinderAdapter(), security_adapters.NewCookieCSRFSourceAdapter(writer, 0))
	require.NoError(b, err)
	registry := &registry_domain.MockRegistryService{GetArtefactFunc: func(context.Context, string) (*registry_dto.ArtefactMeta, error) {
		return nil, registry_domain.ErrArtefactNotFound
	}}
	renderRegistry := render_adapters.NewDataLoaderRegistryAdapter(registry, nil, "")
	renderer := render_domain.NewRenderOrchestrator(nil, nil, renderRegistry, csrf)
	compiled := templater_adapters.NewCompiledManifestRunner(store, i18n, "en")
	cache, err := ast_adapters.NewASTCacheService(ast_adapters.ASTCacheConfig{L1CacheCapacity: 100, L1CacheTTL: 5 * time.Minute, L2CacheBaseDir: b.TempDir()})
	require.NoError(b, err)
	b.Cleanup(func() { cache.Shutdown(context.Background()) })
	service := templater_domain.NewTemplaterService(templater_adapters.NewCachingManifestRunner(compiled, cache), templater_adapters.NewDrivenRenderer(renderer), i18n)
	req := httptest.NewRequest("GET", "/test-path", nil).WithContext(context.WithValue(ctx, chi.RouteCtxKey, chi.NewRouteContext()))
	response := httptest.NewRecorder()
	r := templater_domain.RenderRequest{Request: req, Response: response, Page: templater_dto.PageDefinition{OriginalPath: page}, WebsiteConfig: &config.WebsiteConfig{}}
	var first bytes.Buffer
	r.Writer = &first
	require.NoError(b, service.RenderPage(ctx, r))
	require.Contains(b, first.String(), os.Getenv("PIKO_BENCH_EXPECT_TEXT"))
	r.Writer = io.Discard
	b.ReportAllocs()
	for b.Loop() {
		if err := service.RenderPage(ctx, r); err != nil {
			b.Fatal(err)
		}
	}
}
