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
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

type testRegistryService struct {
	*registry_domain.MockRegistryService
	artefacts   map[string]*registry_dto.ArtefactMeta
	variantData map[string][]byte
	mu          sync.RWMutex
}

func newTestRegistryService() *testRegistryService {
	t := &testRegistryService{
		MockRegistryService: &registry_domain.MockRegistryService{},
		artefacts:           make(map[string]*registry_dto.ArtefactMeta),
		variantData:         make(map[string][]byte),
	}

	t.GetArtefactFunc = t.defaultGetArtefact
	t.FindArtefactByVariantStorageKeyFunc = t.defaultFindByStorageKey
	t.GetVariantDataFunc = t.defaultGetVariantData

	return t
}

func (t *testRegistryService) AddArtefact(artefact *registry_dto.ArtefactMeta) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.artefacts[artefact.ID] = artefact
}

func (t *testRegistryService) AddVariantData(storageKey string, data []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.variantData[storageKey] = data
}

func (t *testRegistryService) defaultGetArtefact(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if artefact, ok := t.artefacts[artefactID]; ok {
		return artefact, nil
	}
	return nil, registry_domain.ErrArtefactNotFound
}

func (t *testRegistryService) defaultFindByStorageKey(_ context.Context, storageKey string) (*registry_dto.ArtefactMeta, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, artefact := range t.artefacts {
		for _, variant := range artefact.ActualVariants {
			if variant.StorageKey == storageKey {
				return artefact, nil
			}
		}
	}
	return nil, registry_domain.ErrArtefactNotFound
}

func (t *testRegistryService) defaultGetVariantData(_ context.Context, variant *registry_dto.Variant) (io.ReadCloser, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if data, ok := t.variantData[variant.StorageKey]; ok {
		return io.NopCloser(bytes.NewReader(data)), nil
	}
	return nil, registry_domain.ErrVariantNotFound
}

var _ registry_domain.RegistryService = (*testRegistryService)(nil)

func newMockCSRFService() *security_domain.MockCSRFTokenService {
	return &security_domain.MockCSRFTokenService{
		GenerateCSRFPairFunc: func(_ http.ResponseWriter, _ *http.Request, _ *bytes.Buffer) (security_dto.CSRFPair, error) {
			return security_dto.CSRFPair{
				ActionToken:       []byte("mock-action-token"),
				RawEphemeralToken: "mock-ephemeral-token",
			}, nil
		},
		ValidateCSRFPairFunc: func(_ *http.Request, _ string, _ []byte) (bool, error) {
			return true, nil
		},
		NameFunc: func() string {
			return "mock-csrf-service"
		},
		CheckFunc: func(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
			return healthprobe_dto.Status{
				Name:  "mock-csrf-service",
				State: healthprobe_dto.StateHealthy,
			}
		},
	}
}

func newMockOnDemandVariantGenerator() *daemon_domain.MockOnDemandVariantGenerator {
	return &daemon_domain.MockOnDemandVariantGenerator{
		GenerateVariantFunc: func(_ context.Context, _ *registry_dto.ArtefactMeta, profileName string) (*registry_dto.Variant, error) {
			return &registry_dto.Variant{VariantID: "generated-" + profileName}, nil
		},
		ParseProfileNameFunc: func(_ string) *daemon_domain.ParsedImageProfile {
			return &daemon_domain.ParsedImageProfile{
				Format:  "webp",
				Width:   800,
				Quality: 80,
			}
		},
	}
}

func newMockTemplaterService() *templater_domain.MockTemplaterService {
	return &templater_domain.MockTemplaterService{
		ProbePageFunc: func(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request, _ *config.WebsiteConfig) (*templater_dto.PageProbeResult, error) {
			return &templater_dto.PageProbeResult{LinkHeaders: nil}, nil
		},
		RenderPageFunc: func(_ context.Context, request templater_domain.RenderRequest) error {
			_, err := request.Writer.Write([]byte("<html><body>Mock Page</body></html>"))
			return err
		},
		ProbePartialFunc: func(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request, _ *config.WebsiteConfig) (*templater_dto.PageProbeResult, error) {
			return &templater_dto.PageProbeResult{LinkHeaders: nil}, nil
		},
		RenderPartialFunc: func(_ context.Context, request templater_domain.RenderRequest) error {
			_, err := request.Writer.Write([]byte("<div>Mock Partial</div>"))
			return err
		},
	}
}

type testPageEntryView struct {
	*templater_domain.MockPageEntryView
	RoutePatterns_      map[string]string
	OriginalPath_       string
	MiddlewareFuncName_ string
	I18nStrategy_       string
	Styling_            string
	JSScriptMetas_      []templater_dto.JSScriptMeta
	Middlewares_        []func(http.Handler) http.Handler
	SupportedLocales_   []string
	CachePolicy_        templater_dto.CachePolicy
	IsPage_             bool
	HasCachePolicy_     bool
	HasMiddleware_      bool
	IsE2EOnly_          bool
}

func newTestPageEntryView() *testPageEntryView {
	t := &testPageEntryView{
		MockPageEntryView: &templater_domain.MockPageEntryView{},
		RoutePatterns_:    make(map[string]string),
		IsPage_:           true,
		SupportedLocales_: []string{"en"},
		I18nStrategy_:     "prefix",
	}
	t.wireDefaults()
	return t
}

func (t *testPageEntryView) wireDefaults() {
	t.GetHasMiddlewareFunc = func() bool { return t.HasMiddleware_ }
	t.GetMiddlewareFuncNameFunc = func() string { return t.MiddlewareFuncName_ }
	t.GetHasCachePolicyFunc = func() bool { return t.HasCachePolicy_ }
	t.GetCachePolicyFunc = func(_ *templater_dto.RequestData) templater_dto.CachePolicy {
		return t.CachePolicy_
	}
	t.GetCachePolicyFuncNameFunc = func() string { return "" }
	t.GetMiddlewaresFunc = func() []func(http.Handler) http.Handler {
		return t.Middlewares_
	}
	t.GetIsPageFunc = func() bool { return t.IsPage_ }
	t.GetRoutePatternFunc = func() string {
		for _, pattern := range t.RoutePatterns_ {
			return pattern
		}
		return ""
	}
	t.GetRoutePatternsFunc = func() map[string]string { return t.RoutePatterns_ }
	t.GetI18nStrategyFunc = func() string { return t.I18nStrategy_ }
	t.GetOriginalPathFunc = func() string { return t.OriginalPath_ }
	t.GetStylingFunc = func() string { return t.Styling_ }
	t.GetSupportedLocalesFunc = func() []string { return t.SupportedLocales_ }
	t.GetJSScriptMetasFunc = func() []templater_dto.JSScriptMeta { return t.JSScriptMetas_ }
	t.GetIsE2EOnlyFunc = func() bool { return t.IsE2EOnly_ }
	t.GetStaticMetadataFunc = func() *templater_dto.InternalMetadata {
		return &templater_dto.InternalMetadata{
			SupportedLocales: t.SupportedLocales_,
		}
	}
}

type testManifestStoreView struct {
	*templater_domain.MockManifestStoreView
	Entries map[string]templater_domain.PageEntryView
}

func newTestManifestStoreView() *testManifestStoreView {
	t := &testManifestStoreView{
		MockManifestStoreView: &templater_domain.MockManifestStoreView{},
		Entries:               make(map[string]templater_domain.PageEntryView),
	}

	t.GetKeysFunc = func() []string {
		keys := make([]string, 0, len(t.Entries))
		for k := range t.Entries {
			keys = append(keys, k)
		}
		return keys
	}
	t.GetPageEntryFunc = func(path string) (templater_domain.PageEntryView, bool) {
		entry, ok := t.Entries[path]
		return entry, ok
	}

	return t
}

func (t *testManifestStoreView) AddEntry(path string, entry templater_domain.PageEntryView) {
	t.Entries[path] = entry
}

func newMockRateLimitService() *security_domain.MockRateLimitService {
	return &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ string, limit int, window time.Duration) (ratelimiter_dto.Result, error) {
			return ratelimiter_dto.Result{
				Allowed:   true,
				Remaining: limit - 1,
				ResetAt:   time.Now().Add(window),
			}, nil
		},
	}
}
