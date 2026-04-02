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

package render_domain

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pml/pml_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/render/render_dto"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
)

type TestRenderContext = renderContext

type TestRenderContextBuilder struct {
	ctx           context.Context
	registry      RegistryPort
	csrfService   security_domain.CSRFTokenService
	httpRequest   *http.Request
	httpResponse  http.ResponseWriter
	pageID        string
	locale        string
	i18nStrategy  string
	defaultLocale string
}

func NewTestRenderContextBuilder() *TestRenderContextBuilder {
	return &TestRenderContextBuilder{
		ctx:           context.Background(),
		registry:      defaultTestRegistry(),
		locale:        "en",
		defaultLocale: "en",
	}
}

func (b *TestRenderContextBuilder) WithContext(ctx context.Context) *TestRenderContextBuilder {
	b.ctx = ctx
	return b
}

func (b *TestRenderContextBuilder) WithRegistry(r RegistryPort) *TestRenderContextBuilder {
	b.registry = r
	return b
}

func (b *TestRenderContextBuilder) WithCSRFService(s security_domain.CSRFTokenService) *TestRenderContextBuilder {
	b.csrfService = s
	return b
}

func (b *TestRenderContextBuilder) WithHTTPRequest(r *http.Request) *TestRenderContextBuilder {
	b.httpRequest = r
	return b
}

func (b *TestRenderContextBuilder) WithHTTPResponse(w http.ResponseWriter) *TestRenderContextBuilder {
	b.httpResponse = w
	return b
}

func (b *TestRenderContextBuilder) WithLocale(locale string) *TestRenderContextBuilder {
	b.locale = locale
	return b
}

func (b *TestRenderContextBuilder) WithI18nStrategy(strategy string) *TestRenderContextBuilder {
	b.i18nStrategy = strategy
	return b
}

func (b *TestRenderContextBuilder) WithDefaultLocale(locale string) *TestRenderContextBuilder {
	b.defaultLocale = locale
	return b
}

func (b *TestRenderContextBuilder) WithPageID(pageID string) *TestRenderContextBuilder {
	b.pageID = pageID
	return b
}

func (b *TestRenderContextBuilder) Build() *TestRenderContext {

	response := b.httpResponse
	if response == nil && b.httpRequest != nil {
		response = httptest.NewRecorder()
	}
	return &renderContext{
		originalCtx:               b.ctx,
		registry:                  b.registry,
		csrfService:               b.csrfService,
		httpRequest:               b.httpRequest,
		httpResponse:              response,
		collectedCustomComponents: make(map[string]struct{}, 16),
		requiredSvgSymbols:        make([]svgSymbolEntry, 0, 16),
		customTags:                make(map[string]struct{}, 16),
		mergedAttrsCache:          make(map[svgCacheKey]string, 16),
		registeredDynamicAssets:   make(map[string]*registry_dto.ArtefactMeta, 8),
		srcsetCache:               make(map[srcsetCacheKey]string, 8),
		linkHeaderSet:             make(map[linkHeaderKey]struct{}, 16),
		collectedLinkHeaders:      make([]render_dto.LinkHeader, 0, 16),
		frozenBuffers:             make([]*[]byte, 0, 8),
		pageID:                    b.pageID,
		currentLocale:             b.locale,
		i18nStrategy:              b.i18nStrategy,
		defaultLocale:             b.defaultLocale,
		diagnostics:               renderDiagnostics{},
		muCollectedLinkHeaders:    sync.Mutex{},
		muDiagnostics:             sync.Mutex{},
		csrfOnce:                  sync.Once{},
	}
}

type TestOrchestratorBuilder struct {
	registry          RegistryPort
	csrfService       security_domain.CSRFTokenService
	pmlEngine         pml_domain.Transformer
	cssResetCSS       string
	transforms        []TransformationPort
	stripHTMLComments bool
}

func NewTestOrchestratorBuilder() *TestOrchestratorBuilder {
	return &TestOrchestratorBuilder{
		registry:   defaultTestRegistry(),
		transforms: make([]TransformationPort, 0),
		pmlEngine:  &pml_domain.MockTransformer{},
	}
}

func (b *TestOrchestratorBuilder) WithRegistry(r RegistryPort) *TestOrchestratorBuilder {
	b.registry = r
	return b
}

func (b *TestOrchestratorBuilder) WithCSRFService(s security_domain.CSRFTokenService) *TestOrchestratorBuilder {
	b.csrfService = s
	return b
}

func (b *TestOrchestratorBuilder) WithTransforms(transforms ...TransformationPort) *TestOrchestratorBuilder {
	b.transforms = transforms
	return b
}

func (b *TestOrchestratorBuilder) WithPmlEngine(engine pml_domain.Transformer) *TestOrchestratorBuilder {
	b.pmlEngine = engine
	return b
}

func (b *TestOrchestratorBuilder) WithStripHTMLComments(strip bool) *TestOrchestratorBuilder {
	b.stripHTMLComments = strip
	return b
}

func (b *TestOrchestratorBuilder) WithCSSResetCSS(css string) *TestOrchestratorBuilder {
	b.cssResetCSS = css
	return b
}

func (b *TestOrchestratorBuilder) Build() *RenderOrchestrator {
	return &RenderOrchestrator{
		registry:          b.registry,
		csrfService:       b.csrfService,
		transformSteps:    b.transforms,
		pmlEngine:         b.pmlEngine,
		stripHTMLComments: b.stripHTMLComments,
		cssResetCSS:       b.cssResetCSS,
	}
}

func defaultTestRegistry() *MockRegistryPort {
	return &MockRegistryPort{
		UpsertArtefactFunc: func(_ context.Context, artefactID string, _ string, _ io.Reader, _ string, desiredProfiles []registry_dto.NamedProfile) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID:              artefactID,
				DesiredProfiles: desiredProfiles,
			}, nil
		},
	}
}

func testHTTPRequest() *http.Request {
	return httptest.NewRequest(http.MethodGet, "/test", nil)
}

func newTestCSRFMock() *security_domain.MockCSRFTokenService {
	return &security_domain.MockCSRFTokenService{
		GenerateCSRFPairFunc: func(_ http.ResponseWriter, _ *http.Request, buffer *bytes.Buffer) (security_dto.CSRFPair, error) {
			buffer.Reset()
			buffer.WriteString("test-action-token")
			return security_dto.CSRFPair{
				RawEphemeralToken: "test-ephemeral-token",
				ActionToken:       buffer.Bytes(),
			}, nil
		},
	}
}

func newTestCSRFMockWithTokens(ephemeral string, action []byte) *security_domain.MockCSRFTokenService {
	return &security_domain.MockCSRFTokenService{
		GenerateCSRFPairFunc: func(_ http.ResponseWriter, _ *http.Request, buffer *bytes.Buffer) (security_dto.CSRFPair, error) {
			buffer.Reset()
			buffer.Write(action)
			return security_dto.CSRFPair{
				RawEphemeralToken: ephemeral,
				ActionToken:       buffer.Bytes(),
			}, nil
		},
	}
}

func newTestCSRFMockWithError(err error) *security_domain.MockCSRFTokenService {
	return &security_domain.MockCSRFTokenService{
		GenerateCSRFPairFunc: func(_ http.ResponseWriter, _ *http.Request, _ *bytes.Buffer) (security_dto.CSRFPair, error) {
			return security_dto.CSRFPair{}, err
		},
	}
}

type testRegistryBuilder struct {
	components     map[string]*render_dto.ComponentMetadata
	svgData        map[string]*ParsedSvgData
	componentError error
	svgError       error
}

func newTestRegistryBuilder() *testRegistryBuilder {
	return &testRegistryBuilder{
		components: make(map[string]*render_dto.ComponentMetadata),
		svgData:    make(map[string]*ParsedSvgData),
	}
}

func (b *testRegistryBuilder) withSVG(id, innerHTML string, attrs ...ast_domain.HTMLAttribute) *testRegistryBuilder {
	parsedData := &ParsedSvgData{
		InnerHTML:  innerHTML,
		Attributes: attrs,
	}
	parsedData.CachedSymbol = ComputeSymbolString(id, parsedData)
	b.svgData[id] = parsedData
	return b
}

func (b *testRegistryBuilder) withComponent(name string, meta *render_dto.ComponentMetadata) *testRegistryBuilder {
	b.components[name] = meta
	return b
}

func (b *testRegistryBuilder) withSVGError(err error) *testRegistryBuilder {
	b.svgError = err
	return b
}

func (b *testRegistryBuilder) withComponentError(err error) *testRegistryBuilder {
	b.componentError = err
	return b
}

func (b *testRegistryBuilder) build() *MockRegistryPort {
	components := b.components
	svgData := b.svgData
	componentError := b.componentError
	svgError := b.svgError

	return &MockRegistryPort{
		GetComponentMetadataFunc: func(_ context.Context, componentType string) (*render_dto.ComponentMetadata, error) {
			if componentError != nil {
				return nil, componentError
			}
			if meta, ok := components[componentType]; ok {
				return meta, nil
			}
			return nil, nil
		},
		BulkGetComponentMetadataFunc: func(_ context.Context, componentTypes []string) (map[string]*render_dto.ComponentMetadata, error) {
			if componentError != nil {
				return nil, componentError
			}
			results := make(map[string]*render_dto.ComponentMetadata, len(componentTypes))
			for _, ct := range componentTypes {
				if meta, ok := components[ct]; ok {
					results[ct] = meta
				}
			}
			return results, nil
		},
		GetAssetRawSVGFunc: func(_ context.Context, assetID string) (*ParsedSvgData, error) {
			if svgError != nil {
				return nil, svgError
			}
			if svg, ok := svgData[assetID]; ok {
				return svg, nil
			}
			return nil, nil
		},
		BulkGetAssetRawSVGFunc: func(_ context.Context, assetIDs []string) (map[string]*ParsedSvgData, error) {
			if svgError != nil {
				return nil, svgError
			}
			results := make(map[string]*ParsedSvgData, len(assetIDs))
			for _, id := range assetIDs {
				if svg, ok := svgData[id]; ok {
					results[id] = svg
				}
			}
			return results, nil
		},
		GetStatsFunc: func() RegistryAdapterStats {
			return RegistryAdapterStats{
				ComponentCacheSize: len(components),
				SVGCacheSize:       len(svgData),
			}
		},
		ClearComponentCacheFunc: func(_ context.Context, componentType string) {
			delete(components, componentType)
		},
		ClearSvgCacheFunc: func(_ context.Context, svgID string) {
			delete(svgData, svgID)
		},
		UpsertArtefactFunc: func(_ context.Context, artefactID string, _ string, _ io.Reader, _ string, desiredProfiles []registry_dto.NamedProfile) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID:              artefactID,
				DesiredProfiles: desiredProfiles,
				ActualVariants:  []registry_dto.Variant{},
				Status:          registry_dto.VariantStatusPending,
			}, nil
		},
	}
}
