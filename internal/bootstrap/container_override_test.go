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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/capabilities"
	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/events/events_domain"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/image/image_domain"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/pml/pml_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/seo/seo_domain"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/video/video_domain"
)

type mockStructValidator struct{}

func (*mockStructValidator) Struct(any) error { return nil }

type stubRegistryService struct {
	registry_domain.RegistryService
}
type stubEventsProvider struct{ events_domain.Provider }
type stubEventBus struct{ orchestrator_domain.EventBus }
type stubCapabilityService struct{ capabilities.Service }
type stubOrchestratorService struct {
	orchestrator_domain.OrchestratorService
}
type stubAnnotatorPort struct{ annotator_domain.AnnotatorPort }
type stubCoordinatorService struct {
	coordinator_domain.CoordinatorService
}
type stubBuildResultCachePort struct {
	coordinator_domain.BuildResultCachePort
}
type stubGeneratorService struct {
	generator_domain.GeneratorService
}
type stubResolverPort struct{ resolver_domain.ResolverPort }
type stubSearchServicePort struct {
	collection_domain.SearchServicePort
}
type stubCSRFTokenService struct {
	security_domain.CSRFTokenService
}
type stubPMLTransformer struct{ pml_domain.Transformer }
type stubRenderService struct{ render_domain.RenderService }
type stubI18nService struct{ i18n_domain.Service }
type stubImageService struct{ image_domain.Service }
type stubVideoService struct{ video_domain.Service }
type stubSEOService struct{ seo_domain.SEOService }
type stubStorageService struct{ storage_domain.Service }
type stubEmailService struct{ email_domain.Service }
type stubLLMService struct{ llm_domain.Service }
type stubRenderRegistryPort struct{ render_domain.RegistryPort }

type overrideTestCase struct {
	setup      func(c *Container) any
	callGetter func(c *Container) (any, error)
	name       string
}

func TestServiceGetterOverrideContract(t *testing.T) {
	t.Parallel()

	stubValidator := &mockStructValidator{}
	stubTypeBuilder := inspector_domain.NewTypeBuilder(
		inspector_dto.Config{},
	)

	testCases := []overrideTestCase{
		{
			name: "RegistryService",
			setup: func(c *Container) any {
				s := &stubRegistryService{}
				c.registryServiceOverride = s
				return s
			},
			callGetter: func(c *Container) (any, error) { return c.GetRegistryService() },
		},
		{
			name: "EventsProvider",
			setup: func(c *Container) any {
				s := &stubEventsProvider{}
				c.eventsProviderOverride = s
				return s
			},
			callGetter: func(c *Container) (any, error) { return c.GetEventsProvider() },
		},
		{
			name: "EventBus",
			setup: func(c *Container) any {
				s := &stubEventBus{}
				c.eventBusOverride = s
				return s
			},
			callGetter: func(c *Container) (any, error) { return c.GetEventBus(), nil },
		},
		{
			name: "CapabilityService",
			setup: func(c *Container) any {
				s := &stubCapabilityService{}
				c.capabilityServiceOverride = s
				return s
			},
			callGetter: func(c *Container) (any, error) { return c.GetCapabilityService() },
		},
		{
			name: "OrchestratorService",
			setup: func(c *Container) any {
				s := &stubOrchestratorService{}
				c.orchestratorServiceOverride = s
				return s
			},
			callGetter: func(c *Container) (any, error) { return c.GetOrchestratorService() },
		},
		{
			name: "AnnotatorService",
			setup: func(c *Container) any {
				s := &stubAnnotatorPort{}
				c.annotatorServiceOverride = s
				return s
			},
			callGetter: func(c *Container) (any, error) { return c.GetAnnotatorService() },
		},
		{
			name: "CoordinatorService",
			setup: func(c *Container) any {
				s := &stubCoordinatorService{}
				c.coordinatorServiceOverride = s
				return s
			},
			callGetter: func(c *Container) (any, error) { return c.GetCoordinatorService() },
		},
		{
			name: "CoordinatorCache",
			setup: func(c *Container) any {
				s := &stubBuildResultCachePort{}
				c.coordinatorCacheOverride = s
				return s
			},
			callGetter: func(c *Container) (any, error) { return c.GetCoordinatorCache() },
		},
		{
			name: "GeneratorService",
			setup: func(c *Container) any {
				s := &stubGeneratorService{}
				c.generatorServiceOverride = s
				return s
			},
			callGetter: func(c *Container) (any, error) { return c.GetGeneratorService() },
		},
		{
			name: "Resolver",
			setup: func(c *Container) any {
				s := &stubResolverPort{}
				c.resolverOverride = s
				return s
			},
			callGetter: func(c *Container) (any, error) { return c.GetResolver() },
		},
		{
			name: "SearchService",
			setup: func(c *Container) any {
				s := &stubSearchServicePort{}
				c.searchServiceOverride = s
				return s
			},
			callGetter: func(c *Container) (any, error) { return c.GetSearchService() },
		},
		{
			name: "CSRFService",
			setup: func(c *Container) any {
				s := &stubCSRFTokenService{}
				c.csrfServiceOverride = s
				return s
			},
			callGetter: func(c *Container) (any, error) { return c.GetCSRFService(), nil },
		},
		{
			name: "PMLTransformer",
			setup: func(c *Container) any {
				s := &stubPMLTransformer{}
				c.pmlTransformerOverride = s
				return s
			},
			callGetter: func(c *Container) (any, error) { return c.GetPMLTransformer(), nil },
		},
		{
			name: "Renderer",
			setup: func(c *Container) any {
				s := &stubRenderService{}
				c.rendererOverride = s
				return s
			},
			callGetter: func(c *Container) (any, error) { return c.GetRenderer(), nil },
		},
		{
			name: "I18nService",
			setup: func(c *Container) any {
				s := &stubI18nService{}
				c.i18nServiceOverride = s
				return s
			},
			callGetter: func(c *Container) (any, error) { return c.GetI18nService() },
		},
		{
			name: "ImageService",
			setup: func(c *Container) any {
				s := &stubImageService{}
				c.imageServiceOverride = s
				return s
			},
			callGetter: func(c *Container) (any, error) { return c.GetImageService() },
		},
		{
			name: "VideoService",
			setup: func(c *Container) any {
				s := &stubVideoService{}
				c.videoServiceOverride = s
				return s
			},
			callGetter: func(c *Container) (any, error) { return c.GetVideoService() },
		},
		{
			name: "SEOService",
			setup: func(c *Container) any {
				s := &stubSEOService{}
				c.seoServiceOverride = s
				return s
			},
			callGetter: func(c *Container) (any, error) { return c.GetSEOService() },
		},
		{
			name: "Validator",
			setup: func(c *Container) any {
				c.validatorOverride = stubValidator
				return stubValidator
			},
			callGetter: func(c *Container) (any, error) { return c.GetValidator(), nil },
		},
		{
			name: "StorageService",
			setup: func(c *Container) any {
				s := &stubStorageService{}
				c.storageServiceOverride = s
				return s
			},
			callGetter: func(c *Container) (any, error) { return c.GetStorageService() },
		},
		{
			name: "EmailService",
			setup: func(c *Container) any {
				s := &stubEmailService{}
				c.emailServiceOverride = s
				return s
			},
			callGetter: func(c *Container) (any, error) { return c.GetEmailService() },
		},
		{
			name: "LLMService",
			setup: func(c *Container) any {
				s := &stubLLMService{}
				c.llmServiceOverride = s
				return s
			},
			callGetter: func(c *Container) (any, error) { return c.GetLLMService() },
		},
		{
			name: "RenderRegistry",
			setup: func(c *Container) any {
				s := &stubRenderRegistryPort{}
				c.renderRegistryOverride = s
				return s
			},
			callGetter: func(c *Container) (any, error) { return c.GetRenderRegistry(), nil },
		},
		{
			name: "TypeInspectorManager",
			setup: func(c *Container) any {
				c.typeInspectorBuilderOverride = stubTypeBuilder
				return stubTypeBuilder
			},
			callGetter: func(c *Container) (any, error) { return c.GetTypeInspectorManager() },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := NewContainer(config.NewConfigProvider())

			stub := tc.setup(c)

			got, err := tc.callGetter(c)

			require.NoError(t, err, "getter should not return an error when override is set")
			assert.Same(t, stub, got, "getter should return the exact override instance")
		})
	}
}

func TestServiceGetterOverrideIdempotency(t *testing.T) {
	t.Parallel()

	stub := &stubEmailService{}
	c := NewContainer(config.NewConfigProvider())
	c.emailServiceOverride = stub

	first, err1 := c.GetEmailService()
	require.NoError(t, err1)

	second, err2 := c.GetEmailService()
	require.NoError(t, err2)

	assert.Same(t, first, second, "consecutive getter calls should return the same instance")
	assert.Same(t, stub, first, "both calls should return the override")
}
