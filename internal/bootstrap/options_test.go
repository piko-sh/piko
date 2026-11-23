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
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/crypto/crypto_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/daemon/daemon_frontend"
	"piko.sh/piko/internal/monitoring/monitoring_adapters"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/internal/profiler"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/seo/seo_domain"
	"piko.sh/safedisk"
)

type optionsStubCacheService struct{ cache_domain.Service }

type optionsStubSpanProcessor struct {
	monitoring_domain.SpanProcessor
}

type optionsStubWatchdogNotifier struct {
	monitoring_domain.WatchdogNotifier
}

type optionsStubProfileUploader struct {
	monitoring_domain.WatchdogProfileUploader
}

type optionsStubRouteSource struct{ seo_domain.RouteSource }

type optionsStubAuthProvider struct{ daemon_dto.AuthProvider }

func TestWithMonitoring_InstallsTheMonitoringService(t *testing.T) {
	testCases := []struct {
		name    string
		options []MonitoringOption
	}{
		{name: "defaults"},
		{name: "overridden address", options: []MonitoringOption{WithMonitoringAddress(":9191"), WithMonitoringBindAddress("127.0.0.1")}},
		{name: "profiling enabled", options: []MonitoringOption{WithMonitoringProfiling()}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewContainer(WithMonitoring(tc.options...))

			assert.NotNil(t, c.GetMonitoringService())
		})
	}
}

func TestWithAuthGuard_StoresACopyOfTheConfig(t *testing.T) {
	t.Parallel()

	guard := daemon_dto.AuthGuardConfig{LoginPath: "/sign-in", PublicPaths: []string{"/"}}
	c := NewContainer(WithAuthGuard(guard))
	guard.LoginPath = "/changed"

	require.NotNil(t, c.authGuardConfig)
	assert.Equal(t, "/sign-in", c.authGuardConfig.LoginPath)
	assert.Equal(t, []string{"/"}, c.authGuardConfig.PublicPaths)
}

func TestWithCSRFSecFetchSiteEnforcement_ReachesTheRouteSettings(t *testing.T) {
	t.Parallel()

	for _, enabled := range []bool{true, false} {
		c := NewContainer(WithCSRFSecFetchSiteEnforcement(enabled))

		require.NotNil(t, c.configServerOverrides)
		settings := buildRouteSettings(c.configServerOverrides)

		assert.Equal(t, enabled, settings.CSRFSecFetchSiteEnforcement)
	}
}

func TestWithEmailService_OverridesTheDefaultService(t *testing.T) {
	t.Parallel()

	override := &stubEmailService{}
	c := NewContainer(WithEmailService(override))

	service, err := c.GetEmailService()

	require.NoError(t, err)
	assert.Same(t, override, service)
}

func TestServiceOverrideOptions_AreReturnedByTheirGetters(t *testing.T) {
	t.Parallel()

	eventBus := &stubEventBus{}
	capability := &stubCapabilityService{}
	orchestrator := &stubOrchestratorService{}
	i18n := &stubI18nService{}
	image := &stubImageService{}
	cacheService := &optionsStubCacheService{}
	cryptoService := crypto_domain.NewDisabledCryptoService()
	search := &stubSearchServicePort{}
	parser := &stubMarkdownParser{}
	resolver := &stubResolverPort{}

	c := NewContainer(
		WithEventBus(eventBus),
		WithCapabilityService(capability),
		WithOrchestratorService(orchestrator),
		WithI18nService(i18n),
		WithImageService(image),
		WithCacheService(cacheService),
		WithCryptoService(cryptoService),
		WithSearchService(search),
		WithMarkdownParser(parser),
		WithResolver(resolver),
	)

	gotBus, err := c.GetEventBus()
	require.NoError(t, err)
	assert.Same(t, eventBus, gotBus)

	gotCapability, err := c.GetCapabilityService()
	require.NoError(t, err)
	assert.Same(t, capability, gotCapability)

	gotOrchestrator, err := c.GetOrchestratorService()
	require.NoError(t, err)
	assert.Same(t, orchestrator, gotOrchestrator)

	gotI18n, err := c.GetI18nService()
	require.NoError(t, err)
	assert.Same(t, i18n, gotI18n)

	gotImage, err := c.GetImageService()
	require.NoError(t, err)
	assert.Same(t, image, gotImage)

	gotCache, err := c.GetCacheService()
	require.NoError(t, err)
	assert.Same(t, cacheService, gotCache)

	gotCrypto, err := c.GetCryptoService()
	require.NoError(t, err)
	assert.Same(t, cryptoService, gotCrypto)

	gotSearch, err := c.GetSearchService()
	require.NoError(t, err)
	assert.Same(t, search, gotSearch)

	assert.Same(t, parser, c.GetMarkdownParser())

	gotResolver, err := c.GetResolver()
	require.NoError(t, err)
	assert.Same(t, resolver, gotResolver)
}

func TestWithPort_SetsTheNetworkPortOverride(t *testing.T) {
	t.Parallel()

	c := NewContainer(WithPort(8181))

	require.NotNil(t, c.configServerOverrides)
	assert.Equal(t, "8181", *c.configServerOverrides.Network.Port)
}

func TestWithShutdownDrainDelay_StoresWholeSeconds(t *testing.T) {
	t.Parallel()

	c := NewContainer(WithShutdownDrainDelay(2500 * time.Millisecond))

	require.NotNil(t, c.configServerOverrides)
	assert.Equal(t, 2, *c.configServerOverrides.HealthProbe.ShutdownDrainDelay)
}

func TestWithJSONTypeInspectorCache_InstallsAJSONCacheProvider(t *testing.T) {
	t.Parallel()

	c := NewContainer(WithJSONTypeInspectorCache())
	require.NotNil(t, c.typeDataProvider)

	sandbox, err := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })

	assert.NotNil(t, c.typeDataProvider(sandbox))
}

func TestWithFrontendModule_UsesTheFirstConfigAndIgnoresDuplicates(t *testing.T) {
	t.Parallel()

	c := NewContainer(
		WithFrontendModule(daemon_frontend.ModuleAnalytics, "first", "ignored"),
		WithFrontendModule(daemon_frontend.ModuleAnalytics, "second"),
	)

	modules := c.GetFrontendModules()
	require.Len(t, modules, 1)
	assert.Equal(t, daemon_frontend.ModuleAnalytics, modules[0].Module)
	assert.Equal(t, "first", modules[0].Config)
}

func TestWithFrontendModule_WithoutConfigStoresNil(t *testing.T) {
	t.Parallel()

	c := NewContainer(WithFrontendModule(daemon_frontend.ModuleAnalytics))

	modules := c.GetFrontendModules()
	require.Len(t, modules, 1)
	assert.Nil(t, modules[0].Config)
}

func TestWithCustomFrontendModule_RegistersTheModuleByName(t *testing.T) {
	t.Parallel()

	c := NewContainer(
		WithCustomFrontendModule("widget", []byte("console.log(1)"), map[string]any{"colour": "blue"}),
		WithCustomFrontendModule("bare", []byte("void 0")),
	)

	modules := c.GetCustomFrontendModules()
	require.Contains(t, modules, "widget")
	require.Contains(t, modules, "bare")
	assert.Equal(t, "widget", modules["widget"].Name)
	assert.Equal(t, map[string]any{"colour": "blue"}, modules["widget"].Config)
	assert.Nil(t, modules["bare"].Config)
}

func TestTelemetryOptions_IgnoreNilAndStoreValues(t *testing.T) {
	t.Parallel()

	exporter := &stubMetricsExporter{}
	processor := &optionsStubSpanProcessor{}
	observer := &captureObserver{}

	empty := NewContainer(WithMetricsExporter(nil), WithSpanProcessor(nil), WithQueryObserver(nil))
	assert.Nil(t, empty.GetMetricsExporter())
	assert.Empty(t, empty.GetSpanProcessors())
	assert.Nil(t, empty.GetQueryObserver())

	c := NewContainer(WithMetricsExporter(exporter), WithSpanProcessor(processor), WithQueryObserver(observer))
	assert.Same(t, exporter, c.GetMetricsExporter())
	require.Len(t, c.GetSpanProcessors(), 1)
	assert.Same(t, processor, c.GetSpanProcessors()[0])
	assert.Same(t, observer, c.GetQueryObserver())
}

func TestWithMonitoring_AppliesWatchdogAndDiagnosticDirectory(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	factories := monitoring_adapters.DefaultServiceFactories()
	c := NewContainer(
		WithDiagnosticDirectory(directory),
		WithMonitoring(
			WithMonitoringAutoNextPort(true),
			WithMonitoringTransport(nil),
			WithMonitoringOtelFactories(factories),
			WithMonitoringWatchdog(),
			WithWatchdogNotifier(&optionsStubWatchdogNotifier{}),
			WithWatchdogProfileUploader(&optionsStubProfileUploader{}),
		),
	)

	assert.NotNil(t, c.GetMonitoringService())
	assert.Equal(t, filepath.Join(directory, "crash.log"), c.crashOutputPath)
}

func TestBuildMonitoringServiceOpts_AddsOptionalSettings(t *testing.T) {
	t.Parallel()

	base := monitoring_domain.DefaultServiceConfig()
	baseOpts := buildMonitoringServiceOpts(&base)

	extended := monitoring_domain.DefaultServiceConfig()
	for _, opt := range []MonitoringOption{
		WithMonitoringAutoNextPort(true),
		WithMonitoringWatchdog(),
		WithWatchdogNotifier(&optionsStubWatchdogNotifier{}),
		WithWatchdogProfileUploader(&optionsStubProfileUploader{}),
	} {
		opt(&extended)
	}
	extendedOpts := buildMonitoringServiceOpts(&extended)

	assert.Len(t, extendedOpts, len(baseOpts)+4)
}

func TestResolveMonitoringFactories_PrefersSuppliedFactories(t *testing.T) {
	t.Parallel()

	supplied := monitoring_domain.ServiceFactories{}
	serviceConfig := monitoring_domain.DefaultServiceConfig()
	WithMonitoringOtelFactories(supplied)(&serviceConfig)

	resolved := resolveMonitoringFactories(&serviceConfig)
	assert.Nil(t, resolved.SpanProcessorFactory)
	assert.Nil(t, resolved.MetricsCollectorFactory)

	defaults := monitoring_domain.DefaultServiceConfig()
	resolvedDefaults := resolveMonitoringFactories(&defaults)
	assert.NotNil(t, resolvedDefaults.SpanProcessorFactory)
	assert.NotNil(t, resolvedDefaults.MetricsCollectorFactory)
}

func TestWithMonitoringWatchdog_AppliesEveryWatchdogOption(t *testing.T) {
	t.Parallel()

	serviceConfig := monitoring_domain.DefaultServiceConfig()
	WithMonitoringWatchdog(
		WithWatchdogHeapThresholdPercent(0.8),
		WithWatchdogHeapThresholdBytes(1024),
		WithWatchdogGoroutineThreshold(500),
		WithWatchdogCheckInterval(time.Second),
		WithWatchdogCooldown(time.Minute),
		WithWatchdogMaxProfilesPerType(3),
		WithWatchdogProfileDirectory("/profiles"),
		WithWatchdogDeltaProfiling(),
		WithWatchdogIncludeGoroutineStacks(true),
		WithWatchdogRSSThresholdPercent(0.9),
		WithWatchdogFDPressureThresholdPercent(0.7),
		WithWatchdogSchedulerLatencyP99Threshold(time.Millisecond),
		WithWatchdogMaxWarningsPerWindow(4),
		WithWatchdogContinuousProfiling(),
		WithWatchdogContinuousProfilingInterval(time.Hour),
		WithWatchdogContinuousProfilingTypes("heap", "goroutine"),
		WithWatchdogContinuousProfilingRetention(6),
		WithWatchdogContinuousProfilingNotify(),
		WithWatchdogContinuousProfilingUpload(),
		WithWatchdogContentionDiagnosticWindow(2*time.Second),
		WithWatchdogContentionDiagnosticAutoFire(),
		WithWatchdogContentionDiagnosticBlockProfileRate(10),
		WithWatchdogContentionDiagnosticMutexProfileFraction(20),
	)(&serviceConfig)

	watchdog := serviceConfig.WatchdogConfig
	require.NotNil(t, watchdog)
	assert.InDelta(t, 0.8, watchdog.HeapThresholdPercent, 0.0001)
	assert.Equal(t, uint64(1024), watchdog.HeapThresholdBytes)
	assert.Equal(t, 500, watchdog.GoroutineThreshold)
	assert.Equal(t, time.Second, watchdog.CheckInterval)
	assert.Equal(t, time.Minute, watchdog.Cooldown)
	assert.Equal(t, 3, watchdog.MaxProfilesPerType)
	assert.Equal(t, "/profiles", watchdog.ProfileDirectory)
	assert.True(t, watchdog.DeltaProfilingEnabled)
	assert.True(t, watchdog.IncludeGoroutineStacks)
	assert.InDelta(t, 0.9, watchdog.RSSThresholdPercent, 0.0001)
	assert.InDelta(t, 0.7, watchdog.FDPressureThresholdPercent, 0.0001)
	assert.Equal(t, time.Millisecond, watchdog.SchedulerLatencyP99Threshold)
	assert.Equal(t, 4, watchdog.MaxWarningsPerWindow)
	assert.True(t, watchdog.ContinuousProfilingEnabled)
	assert.Equal(t, time.Hour, watchdog.ContinuousProfilingInterval)
	assert.Equal(t, []string{"heap", "goroutine"}, watchdog.ContinuousProfilingTypes)
	assert.Equal(t, 6, watchdog.ContinuousProfilingRetention)
	assert.True(t, watchdog.ContinuousProfilingNotify)
	assert.True(t, watchdog.ContinuousProfilingUpload)
	assert.Equal(t, 2*time.Second, watchdog.ContentionDiagnosticWindowDuration)
	assert.True(t, watchdog.ContentionDiagnosticAutoFire)
	assert.Equal(t, 10, watchdog.ContentionDiagnosticBlockProfileRate)
	assert.Equal(t, 20, watchdog.ContentionDiagnosticMutexProfileFraction)
}

func TestWithDiagnosticDirectory_EmptyLeavesCrashOutputUnset(t *testing.T) {
	t.Parallel()

	c := NewContainer(WithDiagnosticDirectory(""))

	assert.Empty(t, c.crashOutputPath)
	assert.Empty(t, c.diagnosticDirectory)
}

func TestCrashOptions_StoreThePathAndLevel(t *testing.T) {
	t.Parallel()

	c := NewContainer(WithCrashOutput("/var/log/crash.log"), WithCrashTraceback("all"))

	assert.Equal(t, "/var/log/crash.log", c.crashOutputPath)
	assert.Equal(t, "all", c.crashTracebackLevel)
}

func TestWithProfiling_AppliesDefaultsAndOptions(t *testing.T) {
	t.Parallel()

	defaults := NewContainer(WithProfiling()).GetProfilingConfig()
	require.NotNil(t, defaults)
	assert.Equal(t, profiler.DefaultPort, defaults.Port)
	assert.Equal(t, profiler.DefaultBindAddress, defaults.BindAddress)
	assert.False(t, defaults.EnableRollingTrace)

	configured := NewContainer(WithProfiling(
		WithProfilingAutoNextPort(true),
		WithProfilingPort(7070),
		WithProfilingBindAddress("0.0.0.0"),
		WithProfilingBlockRate(5),
		WithProfilingMutexFraction(6),
		WithProfilingMemProfileRate(7),
		WithProfilingRollingTrace(),
	)).GetProfilingConfig()
	require.NotNil(t, configured)
	assert.True(t, configured.AutoNextPort)
	assert.Equal(t, 7070, configured.Port)
	assert.Equal(t, "0.0.0.0", configured.BindAddress)
	assert.Equal(t, 5, configured.BlockProfileRate)
	assert.Equal(t, 6, configured.MutexProfileFraction)
	assert.Equal(t, 7, configured.MemProfileRate)
	assert.True(t, configured.EnableRollingTrace)
	assert.Equal(t, profiler.DefaultRollingTraceMinAge, configured.RollingTraceMinAge)
	assert.Equal(t, uint64(profiler.DefaultRollingTraceMaxBytes), configured.RollingTraceMaxBytes)
}

func TestWithProfilingRollingTrace_KeepsExplicitLimits(t *testing.T) {
	t.Parallel()

	configured := NewContainer(WithProfiling(
		WithProfilingRollingTraceMinAge(time.Minute),
		WithProfilingRollingTraceMaxBytes(4096),
		WithProfilingRollingTrace(),
	)).GetProfilingConfig()

	require.NotNil(t, configured)
	assert.True(t, configured.EnableRollingTrace)
	assert.Equal(t, time.Minute, configured.RollingTraceMinAge)
	assert.Equal(t, uint64(4096), configured.RollingTraceMaxBytes)
}

func TestWithGeneratorProfiling_AppliesDefaultsAndOptions(t *testing.T) {
	t.Parallel()

	defaults := NewContainer(WithGeneratorProfiling()).GetGeneratorProfilingConfig()
	require.NotNil(t, defaults)
	assert.Equal(t, profiler.DefaultOutputDir, defaults.OutputDir)
	assert.Equal(t, profiler.CaptureBlockProfileRate, defaults.BlockProfileRate)

	configured := NewContainer(WithGeneratorProfiling(
		WithGeneratorProfilingOutputDir("/tmp/profiles"),
		WithGeneratorProfilingBlockRate(1),
		WithGeneratorProfilingMutexFraction(2),
		WithGeneratorProfilingMemProfileRate(3),
	)).GetGeneratorProfilingConfig()
	require.NotNil(t, configured)
	assert.Equal(t, "/tmp/profiles", configured.OutputDir)
	assert.Equal(t, 1, configured.BlockProfileRate)
	assert.Equal(t, 2, configured.MutexProfileFraction)
	assert.Equal(t, 3, configured.MemProfileRate)
}

func TestWithAutoMemoryLimit_StoresTheProvider(t *testing.T) {
	t.Parallel()

	c := NewContainer(WithAutoMemoryLimit(func() (int64, error) { return 42, nil }))

	require.NotNil(t, c.autoMemoryLimitFunc)
	limit, err := c.autoMemoryLimitFunc()
	require.NoError(t, err)
	assert.Equal(t, int64(42), limit)
}

func TestWithDatabase_RegistersTheDatabase(t *testing.T) {
	t.Parallel()

	registration := &DatabaseRegistration{}
	c := NewContainer(WithDatabase("primary", registration))

	assert.Same(t, registration, c.dbRegistrations["primary"])
}

func TestBuildFlagOptions_SetTheirFlags(t *testing.T) {
	t.Parallel()

	c := NewContainer(
		WithExperimentalDwarfLineDirectives(true),
		WithFormattedGeneratedCode(true),
		WithVerifiedGeneratedCode(true),
		WithStandardLoader(true),
		WithCSSTreeShaking(true),
		WithCSSTreeShakingSafelist([]string{"keep"}),
		WithCSSTreeShakingSafelist([]string{"also"}),
		WithStartupBanner(false),
		WithIAmACatPerson(),
		WithWatchMode(true),
		WithE2EMode(true),
		WithHybridCacheMaxBytes(10),
		WithLLMCacheMaxBytes(11),
		WithHybridCacheWriteExpiration(time.Minute),
		WithActionResponseCacheMaxBytes(12),
		WithDevWidget(),
		WithDevHotreload(),
		WithSRI(false),
		WithModuleName("example.com/site"),
		WithReleaseID("release-1"),
		WithInMemoryRuntimeStore(2048),
		WithEmbedScope(EmbedSourceOnly),
	)

	assert.True(t, c.experimentalDwarfLineDirectives)
	assert.True(t, c.formatGeneratedCode)
	assert.True(t, c.verifyGeneratedCode)
	assert.True(t, c.useStandardLoader)
	assert.True(t, c.IsCSSTreeShakingEnabled())
	assert.Equal(t, []string{"keep", "also"}, c.GetCSSTreeShakingSafelist())
	assert.False(t, c.IsStartupBannerEnabled())
	assert.True(t, c.IsIAmACatPerson())
	assert.True(t, *c.configServerOverrides.Build.WatchMode)
	assert.True(t, *c.configServerOverrides.Build.E2EMode)
	assert.Equal(t, uint64(10), c.hybridCacheMaxBytesOverride)
	assert.Equal(t, uint64(11), c.llmCacheMaxBytesOverride)
	assert.Equal(t, time.Minute, c.hybridCacheWriteExpirationOverride)
	assert.Equal(t, uint64(12), c.actionResponseCacheMaxBytesOverride)
	assert.True(t, c.IsDevWidgetEnabled())
	assert.True(t, c.IsDevHotreloadEnabled())
	assert.False(t, c.IsSRIEnabled())
	assert.Equal(t, "example.com/site", c.moduleNameOverride)
	assert.Equal(t, "release-1", c.releaseIDOverride)
	assert.True(t, c.inMemoryRuntimeStoreEnabled)
	assert.Equal(t, int64(2048), c.inMemoryRuntimeStoreBytes)
	assert.Equal(t, EmbedSourceOnly, c.embedScope)
}

func TestWithHybridCacheWriteExpiration_IgnoresNonPositiveValues(t *testing.T) {
	t.Parallel()

	c := NewContainer(WithHybridCacheWriteExpiration(0), WithHybridCacheWriteExpiration(-time.Second))

	assert.Zero(t, c.hybridCacheWriteExpirationOverride)
}

func TestWithCSSReset_SelectsTheResetStylesheet(t *testing.T) {
	t.Parallel()

	assert.Equal(t, render_domain.DefaultCSSResetSimple, NewContainer(WithCSSReset()).GetCSSResetCSS())
	assert.Equal(t, render_domain.DefaultCSSResetComplete, NewContainer(WithCSSReset(WithCSSResetComplete())).GetCSSResetCSS())
	assert.Equal(t, "p{margin:0}", NewContainer(WithCSSReset(WithCSSResetComplete(), WithCSSResetPKOverride("p{margin:0}"))).GetCSSResetCSS())
}

func TestWithSEOAndAssets_StoreTheirConfigs(t *testing.T) {
	t.Parallel()

	assets := config.AssetsConfig{}
	c := NewContainer(WithSEO(config.SEOConfig{}), WithAssets(assets))

	assert.NotNil(t, c.GetAssetsConfig())
}

func TestWithRouteSource_IgnoresNilAndAppendsSources(t *testing.T) {
	t.Parallel()

	source := &optionsStubRouteSource{}
	c := NewContainer(WithRouteSource(nil), WithRouteSource(source))

	require.Len(t, c.routeSources, 1)
	assert.Same(t, source, c.routeSources[0])
}

func TestWithAuthProviderAndAnalytics_StoreTheCollaborators(t *testing.T) {
	t.Parallel()

	provider := &optionsStubAuthProvider{}
	collector := &recordingAnalyticsCollector{name: "first"}
	c := NewContainer(WithAuthProvider(provider), WithBackendAnalytics(collector))

	assert.Same(t, provider, c.authProvider)
	require.Len(t, c.analyticsCollectors, 1)
	assert.Same(t, collector, c.analyticsCollectors[0])
}

func TestEmbeddedOptions_ToggleEmbeddedMode(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{"manifest.bin": &fstest.MapFile{Data: []byte("x")}}

	assert.False(t, NewContainer(WithEmbeddedPikoFolder(nil)).IsEmbeddedMode())

	embedded := NewContainer(WithEmbeddedPikoFolder(fsys), WithEmbeddedManifest([]byte("manifest")))
	assert.True(t, embedded.IsEmbeddedMode())
	assert.Equal(t, []byte("manifest"), embedded.embeddedManifest)

	disabled := NewContainer(WithEmbeddedPikoFolder(fsys), WithEmbeddedManifest([]byte("manifest")), WithoutEmbeddedRuntime())
	assert.False(t, disabled.IsEmbeddedMode())
	assert.Nil(t, disabled.embeddedManifest)
}
