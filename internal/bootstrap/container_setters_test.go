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
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/daemon/daemon_frontend"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/highlight/highlight_domain"
	"piko.sh/piko/internal/image/image_domain"
	"piko.sh/piko/internal/image/image_dto"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/markdown/markdown_domain"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/internal/notification/notification_domain"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/video/video_domain"
)

type stubMetricsExporter struct {
	monitoring_domain.MetricsExporter
}

type stubMonitoringService struct {
	monitoring_domain.MonitoringService
}

type stubHighlighter struct{ highlight_domain.Highlighter }

type stubImageTransformerPort struct{}

func (s *stubImageTransformerPort) Transform(_ context.Context, _ io.Reader, _ io.Writer, _ image_dto.TransformationSpec) (string, error) {
	return "", nil
}
func (s *stubImageTransformerPort) GetSupportedFormats() []string { return nil }
func (s *stubImageTransformerPort) GetDimensions(_ context.Context, _ io.Reader) (int, int, error) {
	return 0, 0, nil
}
func (s *stubImageTransformerPort) GetSupportedModifiers() []string { return nil }

type stubVideoTranscoderPort struct{ video_domain.TranscoderPort }

type stubCacheProvider struct{ cache_domain.Provider }

type stubLLMProviderPort struct{ llm_domain.LLMProviderPort }

type stubNotificationProviderPort struct {
	notification_domain.NotificationProviderPort
}

type stubEmailTemplateService struct {
	templater_domain.EmailTemplateService
}

type stubCSRFCookieSourceAdapter struct {
	security_domain.CSRFCookieSourceAdapter
}

type stubMarkdownParser struct {
	markdown_domain.MarkdownParserPort
}

func TestSetGetMetricsExporter(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Nil(t, c.GetMetricsExporter(), "should be nil by default")

	exporter := &stubMetricsExporter{}
	c.SetMetricsExporter(exporter)

	assert.Same(t, exporter, c.GetMetricsExporter())
}

func TestSetGetMonitoringService(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Nil(t, c.GetMonitoringService(), "should be nil by default")

	service := &stubMonitoringService{}
	c.SetMonitoringService(service)

	assert.Same(t, service, c.GetMonitoringService())
}

func TestGetOrchestratorInspector_NilByDefault(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Nil(t, c.GetOrchestratorInspector())
}

func TestGetRegistryInspector_NilByDefault(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Nil(t, c.GetRegistryInspector())
}

func TestSetCSPPolicyString(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	policy, wasSet := c.GetCSPPolicyString()
	assert.Empty(t, policy)
	assert.False(t, wasSet, "should not be set by default")

	c.SetCSPPolicyString("default-src 'self'")

	policy, wasSet = c.GetCSPPolicyString()
	assert.Equal(t, "default-src 'self'", policy)
	assert.True(t, wasSet)
}

func TestSetCSPPolicyString_EmptyMarksAsSet(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	c.SetCSPPolicyString("")

	policy, wasSet := c.GetCSPPolicyString()
	assert.Empty(t, policy)
	assert.True(t, wasSet, "setting to empty should still mark as set")
}

func TestSetGetCrossOriginResourcePolicy(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Empty(t, c.GetCrossOriginResourcePolicy(), "should be empty by default")

	c.SetCrossOriginResourcePolicy("cross-origin")

	assert.Equal(t, "cross-origin", c.GetCrossOriginResourcePolicy())
}

func TestSetGetCSPConfig(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Nil(t, c.GetCSPConfig(), "should be nil by default")

	builder := security_domain.NewCSPBuilder().WithPikoDefaults()
	c.SetCSPConfig(builder)

	assert.Same(t, builder, c.GetCSPConfig())
}

func TestSetReportingEndpoints(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	endpoints := []config.ReportingEndpoint{
		{Name: "csp", URL: "https://example.com/csp-report"},
	}

	c.SetReportingEndpoints(endpoints)

	reportingConfig := c.GetReportingConfig()
	assert.True(t, deref(reportingConfig.Enabled, false))
	require.Len(t, reportingConfig.Endpoints, 1)
	assert.Equal(t, "csp", reportingConfig.Endpoints[0].Name)
	assert.Equal(t, "https://example.com/csp-report", reportingConfig.Endpoints[0].URL)
}

func TestGetReportingConfig_FallsBackToServerConfig(t *testing.T) {
	t.Parallel()
	c := NewContainer()
	c.serverConfig.Security.Reporting = config.ReportingConfig{
		Enabled: new(true),
		Endpoints: []config.ReportingEndpoint{
			{Name: "default", URL: "https://example.com/default"},
		},
	}

	reportingConfig := c.GetReportingConfig()
	assert.True(t, deref(reportingConfig.Enabled, false))
	require.Len(t, reportingConfig.Endpoints, 1)
	assert.Equal(t, "default", reportingConfig.Endpoints[0].Name)
}

func TestSetReportingEndpoints_OverridesServerConfig(t *testing.T) {
	t.Parallel()
	c := NewContainer()
	c.serverConfig.Security.Reporting = config.ReportingConfig{
		Enabled: new(true),
		Endpoints: []config.ReportingEndpoint{
			{Name: "from-config", URL: "https://example.com/config"},
		},
	}

	c.SetReportingEndpoints([]config.ReportingEndpoint{
		{Name: "from-code", URL: "https://example.com/code"},
	})

	reportingConfig := c.GetReportingConfig()
	require.Len(t, reportingConfig.Endpoints, 1)
	assert.Equal(t, "from-code", reportingConfig.Endpoints[0].Name,
		"code endpoints should take precedence over config")
}

func TestGetCSPPolicy_RawPolicyTakesPrecedence(t *testing.T) {
	t.Parallel()
	c := NewContainer()
	c.SetCSPPolicyString("raw-policy")

	assert.Equal(t, "raw-policy", c.GetCSPPolicy())
}

func TestGetCSPPolicy_BuilderUsedWhenNoRawPolicy(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	builder := security_domain.NewCSPBuilder().WithPikoDefaults()
	c.SetCSPConfig(builder)

	assert.Equal(t, builder.Build(), c.GetCSPPolicy())
}

func TestGetCSPPolicy_ConfigValueUsedWhenNoBuilderOrRaw(t *testing.T) {
	t.Parallel()
	c := NewContainer()
	c.serverConfig.Security.Headers.ContentSecurityPolicy = new("config-policy")

	assert.Equal(t, "config-policy", c.GetCSPPolicy())
}

func TestGetCSPPolicy_DefaultWhenNothingSet(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	policy := c.GetCSPPolicy()
	assert.NotEmpty(t, policy, "should return Piko default CSP")
}

func TestGetCSPPolicy_EmptyRawPolicyDisablesCSP(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	c.SetCSPPolicyString("")

	assert.Empty(t, c.GetCSPPolicy(), "empty raw policy should disable CSP")
}

func TestGetCSPPolicy_RawPolicyBeatsBuilder(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	builder := security_domain.NewCSPBuilder().WithPikoDefaults()
	c.SetCSPConfig(builder)
	c.SetCSPPolicyString("raw-wins")

	assert.Equal(t, "raw-wins", c.GetCSPPolicy(),
		"raw policy string should take precedence over builder")
}

func TestSetGetHighlighter(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Nil(t, c.GetHighlighter(), "should be nil by default")

	h := &stubHighlighter{}
	c.SetHighlighter(h)

	assert.Same(t, h, c.GetHighlighter())
}

func TestAddFrontendModule(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Empty(t, c.GetFrontendModules())

	c.AddFrontendModule(daemon_frontend.ModuleAnalytics, nil)

	modules := c.GetFrontendModules()
	require.Len(t, modules, 1)
	assert.Equal(t, daemon_frontend.ModuleAnalytics, modules[0].Module)
	assert.Nil(t, modules[0].Config)
}

func TestAddFrontendModule_DeduplicatesSameModule(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	c.AddFrontendModule(daemon_frontend.ModuleAnalytics, nil)
	c.AddFrontendModule(daemon_frontend.ModuleAnalytics, nil)

	assert.Len(t, c.GetFrontendModules(), 1, "duplicate should be silently ignored")
}

func TestAddFrontendModule_WithConfig(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	moduleConfig := map[string]string{"key": "value"}
	c.AddFrontendModule(daemon_frontend.ModuleAnalytics, moduleConfig)

	modules := c.GetFrontendModules()
	require.Len(t, modules, 1)
	assert.Equal(t, moduleConfig, modules[0].Config)
}

func TestAddCustomFrontendModule(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Nil(t, c.GetCustomFrontendModules())

	content := []byte("console.log('hello');")
	c.AddCustomFrontendModule("my-module", content, nil)

	modules := c.GetCustomFrontendModules()
	require.Contains(t, modules, "my-module")
	assert.NotNil(t, modules["my-module"])
}

func TestAddCustomFrontendModule_MultipleModules(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	c.AddCustomFrontendModule("alpha", []byte("a"), nil)
	c.AddCustomFrontendModule("beta", []byte("b"), nil)

	modules := c.GetCustomFrontendModules()
	assert.Len(t, modules, 2)
	assert.Contains(t, modules, "alpha")
	assert.Contains(t, modules, "beta")
}

func TestAddImageTransformer_SetsDefaultOnFirst(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	c.AddImageTransformer("sharp", &stubImageTransformerPort{})

	assert.Equal(t, "sharp", c.defaultImageTransformer)
}

func TestAddImageTransformer_DoesNotOverrideDefault(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	c.AddImageTransformer("first", &stubImageTransformerPort{})
	c.AddImageTransformer("second", &stubImageTransformerPort{})

	assert.Equal(t, "first", c.defaultImageTransformer,
		"default should remain as the first registered transformer")
}

func TestAddImageTransformer_StoresInMap(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	transformer := &stubImageTransformerPort{}
	c.AddImageTransformer("vips", transformer)

	require.Contains(t, c.imageTransformers, "vips")
	assert.Same(t, transformer, c.imageTransformers["vips"])
}

func TestSetDefaultImageTransformer(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	c.AddImageTransformer("first", &stubImageTransformerPort{})
	c.AddImageTransformer("second", &stubImageTransformerPort{})
	c.SetDefaultImageTransformer("second")

	assert.Equal(t, "second", c.defaultImageTransformer)
}

func TestAddVideoTranscoder_SetsDefaultOnFirst(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	c.AddVideoTranscoder("ffmpeg", &stubVideoTranscoderPort{})

	assert.Equal(t, "ffmpeg", c.defaultVideoTranscoder)
}

func TestAddVideoTranscoder_DoesNotOverrideDefault(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	c.AddVideoTranscoder("first", &stubVideoTranscoderPort{})
	c.AddVideoTranscoder("second", &stubVideoTranscoderPort{})

	assert.Equal(t, "first", c.defaultVideoTranscoder,
		"default should remain as the first registered transcoder")
}

func TestAddVideoTranscoder_StoresInMap(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	transcoder := &stubVideoTranscoderPort{}
	c.AddVideoTranscoder("ffmpeg", transcoder)

	require.Contains(t, c.videoTranscoders, "ffmpeg")
	assert.Same(t, transcoder, c.videoTranscoders["ffmpeg"])
}

func TestSetDefaultVideoTranscoder(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	c.SetDefaultVideoTranscoder("libx264")

	assert.Equal(t, "libx264", c.defaultVideoTranscoder)
}

func TestGetImagePredefinedVariants_NilByDefault(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Nil(t, c.GetImagePredefinedVariants())
}

func TestGetImagePredefinedVariants_AfterDirectSet(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	variants := map[string]image_dto.TransformationSpec{
		"thumbnail": {Width: 100, Height: 100},
	}
	c.imagePredefinedVariants = variants

	got := c.GetImagePredefinedVariants()
	require.Contains(t, got, "thumbnail")
	assert.Equal(t, 100, got["thumbnail"].Width)
	assert.Equal(t, 100, got["thumbnail"].Height)
}

func TestSetImageConfig_NilIsNoOp(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	c.SetImageConfig(nil)

	assert.Nil(t, c.imageTransformers, "nil config should not initialise the map")
	assert.Empty(t, c.defaultImageTransformer)
}

func TestSetImageConfig_RegistersProviders(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	imgConfig := &image_domain.ImageConfig{
		Providers: map[string]image_domain.TransformerPort{
			"vips": &stubImageTransformerPort{},
		},
		DefaultProvider: "vips",
		PredefinedVariants: map[string]image_dto.TransformationSpec{
			"thumb": {Width: 200, Height: 200},
		},
	}
	c.SetImageConfig(imgConfig)

	require.Contains(t, c.imageTransformers, "vips")
	assert.Equal(t, "vips", c.defaultImageTransformer)
	require.Contains(t, c.imagePredefinedVariants, "thumb")
	assert.NotNil(t, c.imageServiceConfigOverride)
}

func TestAddCacheProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	provider := &stubCacheProvider{}
	c.AddCacheProvider("redis", provider)

	require.Contains(t, c.cacheProviders, "redis")
	assert.Same(t, provider, c.cacheProviders["redis"])
}

func TestSetCacheDefaultProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	c.SetCacheDefaultProvider("redis")

	assert.Equal(t, "redis", c.cacheDefaultProvider)
}

func TestAddLLMProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	provider := &stubLLMProviderPort{}
	c.AddLLMProvider("openai", provider)

	require.Contains(t, c.llmProviders, "openai")
	assert.Same(t, provider, c.llmProviders["openai"])
}

func TestSetLLMDefaultProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	c.SetLLMDefaultProvider("openai")

	assert.Equal(t, "openai", c.llmDefaultProvider)
}

func TestAddCryptoProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	provider := &stubEncryptionProvider{}
	c.AddCryptoProvider("aes", provider)

	require.Contains(t, c.cryptoProviders, "aes")
	assert.Same(t, provider, c.cryptoProviders["aes"])
}

func TestAddNotificationProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	provider := &stubNotificationProviderPort{}
	c.AddNotificationProvider("slack", provider)

	require.Contains(t, c.notificationProviders, "slack")
	assert.Same(t, provider, c.notificationProviders["slack"])
}

func TestSetNotificationDefaultProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	c.SetNotificationDefaultProvider("slack")

	assert.Equal(t, "slack", c.notificationDefaultProvider)
}

func TestSetEmailDispatcherConfig(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.False(t, c.hasEmailDispatcher, "should be false by default")

	dispatcherConfig := &email_dto.DispatcherConfig{}
	c.SetEmailDispatcherConfig(dispatcherConfig)

	assert.True(t, c.hasEmailDispatcher)
	assert.Same(t, dispatcherConfig, c.emailDispatcherConfig)
}

func TestSetGetEmailTemplateService(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Nil(t, c.GetEmailTemplateService(), "should be nil by default")

	service := &stubEmailTemplateService{}
	c.SetEmailTemplateService(service)

	assert.Same(t, service, c.GetEmailTemplateService())
}

func TestSetStoragePresignBaseURL(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Empty(t, c.storagePresignBaseURL)

	c.SetStoragePresignBaseURL("https://cdn.example.com")

	assert.Equal(t, "https://cdn.example.com", c.storagePresignBaseURL)
}

func TestSetStoragePublicBaseURL(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Empty(t, c.storagePublicBaseURL)

	c.SetStoragePublicBaseURL("https://public.example.com")

	assert.Equal(t, "https://public.example.com", c.storagePublicBaseURL)
}

func TestSetStorageDispatcherConfig(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.False(t, c.hasStorageDispatcher, "should be false by default")

	dispatcherConfig := &storage_domain.DispatcherConfig{}
	c.SetStorageDispatcherConfig(dispatcherConfig)

	assert.True(t, c.hasStorageDispatcher)
	assert.Same(t, dispatcherConfig, c.storageDispatcherConfig)
}

func TestSetCoordinatorCodeEmitterOverride(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Nil(t, c.coordinatorCodeEmitterOverride)

	type emitter struct {
		coordinator_domain.CodeEmitterPort
	}
	e := &emitter{}
	c.SetCoordinatorCodeEmitterOverride(e)

	assert.Same(t, e, c.coordinatorCodeEmitterOverride)
}

func TestSetCoordinatorDiagnosticOutputOverride(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Nil(t, c.coordinatorDiagnosticOutputOverride)

	type output struct {
		coordinator_domain.DiagnosticOutputPort
	}
	o := &output{}
	c.SetCoordinatorDiagnosticOutputOverride(o)

	assert.Same(t, o, c.coordinatorDiagnosticOutputOverride)
}

func TestSetFSReaderOverride(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Nil(t, c.coordinatorFSReaderOverride)

	type reader struct{ annotator_domain.FSReaderPort }
	r := &reader{}
	c.SetFSReaderOverride(r)

	assert.Same(t, r, c.coordinatorFSReaderOverride)
}

func TestSetResolverOverride(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Nil(t, c.resolverOverride)

	resolver := &stubResolverPort{}
	c.SetResolverOverride(resolver)

	assert.Same(t, resolver, c.resolverOverride)
}

func TestSetSearchService(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Nil(t, c.searchServiceOverride)

	service := &stubSearchServicePort{}
	c.SetSearchService(service)

	assert.Same(t, service, c.searchServiceOverride)
}

func TestSetEventsProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Nil(t, c.eventsProviderOverride)

	provider := &stubEventsProvider{}
	c.SetEventsProvider(provider)

	assert.Same(t, provider, c.eventsProviderOverride)
}

func TestSetPMLTransformer(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	transformer := &stubPMLTransformer{}
	c.SetPMLTransformer(transformer)

	assert.Same(t, transformer, c.pmlTransformerOverride,
		"override field should be set")
	assert.Same(t, transformer, c.pmlTransformer,
		"active transformer field should be set immediately")
}

func TestSetCSRFCookieSource(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	source := &stubCSRFCookieSourceAdapter{}
	c.SetCSRFCookieSource(source)

	assert.Same(t, source, c.csrfCookieSourceOverride)
	assert.Same(t, source, c.csrfCookieSource)
}

func TestSetRegistryMetadataCacheConfig(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Nil(t, c.registryMetadataCacheConfig, "should be nil by default")

	cacheConfig := RegistryMetadataCacheConfig{
		MaxWeight:    1024,
		TTL:          5 * time.Minute,
		StatsEnabled: true,
	}
	c.SetRegistryMetadataCacheConfig(cacheConfig)

	require.NotNil(t, c.registryMetadataCacheConfig)
	assert.Equal(t, uint64(1024), c.registryMetadataCacheConfig.MaxWeight)
	assert.Equal(t, 5*time.Minute, c.registryMetadataCacheConfig.TTL)
	assert.True(t, c.registryMetadataCacheConfig.StatsEnabled)
}

func TestGetServerConfig(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.NotNil(t, c.GetServerConfig())
	assert.NotNil(t, c.GetWebsiteConfig())
}

func TestGetEmailDispatcher_NilByDefault(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Nil(t, c.GetEmailDispatcher())
}

func TestGetNotificationDispatcher_NilByDefault(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Nil(t, c.GetNotificationDispatcher())
}

func TestGetDispatcherInspector_NilWhenNoDispatchers(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Nil(t, c.GetDispatcherInspector(),
		"should return nil when neither email nor notification dispatchers are configured")
}

func TestDefaultMetadataCacheProvider_ReturnsNil(t *testing.T) {
	t.Parallel()

	result := defaultMetadataCacheProvider()

	assert.Nil(t, result)
}

func TestSetGetMarkdownParser(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Nil(t, c.GetMarkdownParser(), "should be nil by default")

	parser := &stubMarkdownParser{}
	c.SetMarkdownParser(parser)

	assert.Same(t, parser, c.GetMarkdownParser())
}

func TestSetValidator_NilClearsOverride(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	c.SetValidator(nil)

	assert.Nil(t, c.validatorOverride)
}

func TestSetValidator_SetsOverride(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	v := &stubStructValidator{}
	c.SetValidator(v)

	assert.Same(t, v, c.validatorOverride)
}

type stubStructValidator struct{}

func (*stubStructValidator) Struct(any) error { return nil }
