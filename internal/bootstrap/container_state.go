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

// This file defines the embedded sub-structs that group the Container's state by concern.
// NewContainer starts every sub-struct from its zero value except containerLifecycle,
// whose constructor installs the default registry metadata cache provider.

import (
	"context"
	"io/fs"
	"sync"
	"time"

	"piko.sh/piko/internal/analytics/analytics_domain"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/capabilities"
	"piko.sh/piko/internal/captcha/captcha_domain"
	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/component/component_domain"
	"piko.sh/piko/internal/component/component_dto"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/config/config_domain"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/crypto/crypto_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/daemon/daemon_frontend"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/events/events_domain"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/healthprobe/healthprobe_domain"
	"piko.sh/piko/internal/highlight/highlight_domain"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/image/image_domain"
	"piko.sh/piko/internal/image/image_dto"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/markdown/markdown_domain"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/internal/notification/notification_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_adapters"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/persistence"
	"piko.sh/piko/internal/pml/pml_domain"
	"piko.sh/piko/internal/profiler"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/seo/seo_domain"
	"piko.sh/piko/internal/spamdetect/spamdetect_domain"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/video/video_domain"
	"piko.sh/piko/wdk/safedisk"
)

// containerOnceGuards holds the sync.Once guards that make each lazily created service a
// singleton.
type containerOnceGuards struct {
	// capabilityOnce guards single initialisation of the capability service.
	capabilityOnce sync.Once

	// storageOnce guards single initialisation of the storage service.
	storageOnce sync.Once

	// videoOnce guards single initialisation of the video service.
	videoOnce sync.Once

	// analyticsOnce guards single initialisation of the analytics service.
	analyticsOnce sync.Once

	// querierDBOnce guards single initialisation of the querier database service.
	querierDBOnce sync.Once

	// dbProviderOnce guards single initialisation of the database provider.
	dbProviderOnce sync.Once

	// i18nOnce guards single initialisation of the i18n service.
	i18nOnce sync.Once

	// generatorOnce guards single initialisation of the generator service.
	generatorOnce sync.Once

	// registryOnce guards single initialisation of the registry service.
	registryOnce sync.Once

	// componentRegistryOnce guards single initialisation of the component registry.
	componentRegistryOnce sync.Once

	// orchestratorOnce guards single initialisation of the orchestrator service.
	orchestratorOnce sync.Once

	// sandboxFactoryOnce guards single initialisation of the cached sandbox factory.
	sandboxFactoryOnce sync.Once

	// csrfOnce guards single initialisation of the CSRF service.
	csrfOnce sync.Once

	// pmlTransformerOnce guards single initialisation of the PML transformer.
	pmlTransformerOnce sync.Once

	// rendererOnce guards single initialisation of the renderer service.
	rendererOnce sync.Once

	// coordinatorCacheOnce guards single initialisation of coordinatorCache.
	coordinatorCacheOnce sync.Once

	// typeInspectorBuilderOnce guards single initialisation of the type inspector.
	typeInspectorBuilderOnce sync.Once

	// eventsProviderOnce guards the lazy initialisation of the events provider.
	eventsProviderOnce sync.Once

	// coordinatorOnce guards single initialisation of the coordinator service.
	coordinatorOnce sync.Once

	// appCtxOnce guards single initialisation of appCtx.
	appCtxOnce sync.Once

	// eventBusOnce guards single initialisation of the event bus.
	eventBusOnce sync.Once

	// resolverOnce guards single initialisation of the resolver.
	resolverOnce sync.Once

	// emailOnce guards single initialisation of the email service.
	emailOnce sync.Once

	// llmOnce guards single initialisation of the LLM service.
	llmOnce sync.Once

	// imageOnce guards single initialisation of the image service.
	imageOnce sync.Once

	// annotatorOnce guards single initialisation of the annotator service.
	annotatorOnce sync.Once

	// renderRegOnce guards single initialisation of the render registry.
	renderRegOnce sync.Once

	// seoOnce guards single initialisation of the SEO service.
	seoOnce sync.Once

	// cacheOnce guards single initialisation of the cache service.
	cacheOnce sync.Once

	// cryptoOnce guards single initialisation of the crypto service.
	cryptoOnce sync.Once

	// captchaOnce guards single initialisation of the captcha service.
	captchaOnce sync.Once

	// spamdetectOnce guards single initialisation of the spam detection service.
	spamdetectOnce sync.Once

	// validatorOnce guards single initialisation of the validator.
	validatorOnce sync.Once

	// collectionServiceOnce guards single initialisation of the collection service.
	collectionServiceOnce sync.Once

	// searchServiceOnce guards single initialisation of the search service.
	searchServiceOnce sync.Once

	// healthProbeOnce guards single initialisation of the health probe service.
	healthProbeOnce sync.Once

	// rateLimitServiceOnce guards single initialisation of the rate limit service.
	rateLimitServiceOnce sync.Once

	// rateLimiterOnce guards single initialisation of the centralised rate limiter.
	rateLimiterOnce sync.Once
}

// containerServiceErrors holds the errors recorded when a lazily created service fails to
// initialise, so later calls return the same failure.
type containerServiceErrors struct {
	// llmErr holds any error from creating the LLM service.
	llmErr error

	// typeInspectorBuilderErr holds any error from creating the type inspector.
	typeInspectorBuilderErr error

	// healthProbeErr holds any error from creating the health probe service.
	healthProbeErr error

	// collectionServiceErr holds any error from collection service creation.
	collectionServiceErr error

	// searchServiceErr holds any error that occurred while setting up the search service.
	searchServiceErr error

	// cryptoErr holds any error from creating the crypto service.
	cryptoErr error

	// captchaErr holds any error from creating the captcha service.
	captchaErr error

	// spamdetectErr holds any error from creating the spam detection service.
	spamdetectErr error

	// cacheErr holds any error that occurred when setting up the cache service.
	cacheErr error

	// seoErr holds any error that occurred when creating the SEO service.
	seoErr error

	// storageErr holds any error from creating the storage service.
	storageErr error

	// imageErr holds any error from creating the image service.
	imageErr error

	// videoErr holds any error from video service creation.
	videoErr error

	// resolverErr holds any error that occurred during resolver setup.
	resolverErr error

	// coordinatorCacheErr holds any error from initialising the coordinator cache.
	coordinatorCacheErr error

	// rateLimitServiceErr holds any error from creating the rate limit service.
	rateLimitServiceErr error

	// rateLimiterErr holds any error from creating the centralised rate limiter.
	rateLimiterErr error

	// querierDBErr holds any error from querier database service creation.
	querierDBErr error

	// dbProviderErr holds any error from database provider setup.
	dbProviderErr error

	// generatorErr holds any error from creating the generator service.
	generatorErr error

	// coordinatorErr stores any error that occurred when creating the coordinator service.
	coordinatorErr error

	// annotatorErr stores any error from creating the annotator service.
	annotatorErr error

	// i18nErr holds any error that occurred when creating the i18n service.
	i18nErr error

	// orchestratorErr holds any error from creating the orchestrator service.
	orchestratorErr error

	// capabilityErr holds any error from creating the capability service.
	capabilityErr error

	// registryErr holds any error from creating the registry service.
	registryErr error

	// eventsProviderErr holds any error that occurred when creating the events provider.
	eventsProviderErr error

	// eventBusErr holds any error captured while initialising the event bus.
	eventBusErr error

	// sandboxFactoryErr records any error from creating the cached factory.
	sandboxFactoryErr error

	// emailErr holds any error from email service setup.
	emailErr error
}

// containerCoreServices holds the lazily created services that make up the build and
// render pipeline, such as the registry, orchestrator, and coordinator.
type containerCoreServices struct {
	// eventsProvider holds the cached events provider instance.
	eventsProvider events_domain.Provider

	// eventBus sends and receives domain events using publish-subscribe messaging.
	eventBus orchestrator_domain.EventBus

	// registryService holds the registry service for registry operations.
	registryService registry_domain.RegistryService

	// registryMetaStore stores registry metadata and is closed on shutdown.
	registryMetaStore registry_domain.MetadataStore

	// registryReleaseOverlay is the layer-aware writable overlay used for release publish,
	// heartbeat, and retire. It differs from registryMetaStore on an embedded deploy, where
	// the meta store is a union that does not expose the ReleasePublisher capability the
	// release lifecycle needs; release operations must always go to the overlay directly.
	registryReleaseOverlay registry_domain.MetadataStore

	// registryBlobOverlay is the writable shared blob provider layered over the embedded
	// base, retained so release publish can replicate the base's bytes into the shared
	// store; nil when no writable shared blob store is configured.
	registryBlobOverlay storage_domain.StorageProviderPort

	// registryMetaCache stores cached metadata for the registry service.
	registryMetaCache registry_domain.MetadataCache

	// capabilityService stores the capability service after it is created.
	capabilityService capabilities.Service

	// orchestratorService holds the orchestrator service instance.
	orchestratorService orchestrator_domain.OrchestratorService

	// pmlTransformer holds the service that transforms PML documents.
	pmlTransformer pml_domain.Transformer

	// renderer holds the service that produces rendered output.
	renderer render_domain.RenderService

	// i18nService is the cached translation service instance.
	i18nService i18n_domain.Service

	// annotatorService holds the annotator service instance, created when first needed.
	annotatorService annotator_domain.AnnotatorPort

	// coordinatorCache stores build results for reuse.
	coordinatorCache coordinator_domain.BuildResultCachePort

	// coordinatorService is the coordinator service instance.
	coordinatorService coordinator_domain.CoordinatorService

	// generatorService holds the generator service instance.
	generatorService generator_domain.GeneratorService

	// resolver holds the lazily initialised symbol resolver instance.
	resolver resolver_domain.ResolverPort

	// renderRegistry holds the render registry instance.
	renderRegistry render_domain.RegistryPort

	// componentRegistry holds the component registry for tag lookup.
	componentRegistry component_domain.ComponentRegistry

	// typeInspectorBuilder holds the type inspector used to analyse Go types.
	typeInspectorBuilder *inspector_domain.TypeBuilder

	// artefactBridge links artefact processing to the orchestrator workflow.
	artefactBridge *orchestrator_adapters.ArtefactWorkflowBridge

	// orchestratorInspector holds the orchestrator inspector for monitoring; nil when not
	// available.
	orchestratorInspector orchestrator_domain.OrchestratorInspector

	// registryInspector holds the registry inspector for monitoring; nil when not available.
	registryInspector registry_domain.RegistryInspector

	// validator stores the validation instance for struct and field checks.
	validator StructValidator

	// sandboxFactoryInstance is the lazily-created, cached safedisk.Factory built from the
	// server config. All production sandbox creation goes through this single factory so
	// that path validation, the Enabled flag, and the purpose string are applied
	// consistently.
	sandboxFactoryInstance safedisk.Factory
}

// containerFeatureServices holds the lazily created application feature services, such as
// email, storage, caching, security, and monitoring.
type containerFeatureServices struct {
	// analyticsService distributes backend analytics events to the registered collectors;
	// nil until first requested, and stays nil when no collectors are registered.
	analyticsService *analytics_domain.Service

	// csrfService handles CSRF token creation and validation.
	csrfService security_domain.CSRFTokenService

	// csrfCookieSource stores and reads CSRF tokens from cookies.
	csrfCookieSource security_domain.CSRFCookieSourceAdapter

	// emailTemplateService holds the email template service; nil if not yet set.
	emailTemplateService templater_domain.EmailTemplateService

	// pdfWriterService holds the PDF writer service; nil if not yet set.
	pdfWriterService pdfwriter_domain.PdfWriterService

	// imageService is the cached image service instance.
	imageService image_domain.Service

	// videoService is the cached video service instance.
	videoService video_domain.Service

	// storageService holds the storage service instance.
	storageService storage_domain.Service

	// seoService holds the SEO service instance; nil when SEO is disabled.
	seoService seo_domain.SEOService

	// cacheService holds the cache service for storing and retrieving data.
	cacheService cache_domain.Service

	// cryptoService handles encryption and decryption; nil means not yet created.
	cryptoService crypto_domain.CryptoServicePort

	// captchaService handles captcha verification; nil means not yet created.
	captchaService captcha_domain.CaptchaServicePort

	// spamdetectService handles spam detection; nil means not yet created.
	spamdetectService spamdetect_domain.SpamDetectServicePort

	// collectionService handles collection operations.
	collectionService collection_domain.CollectionService

	// searchService is the search service for collections.
	searchService collection_domain.SearchServicePort

	// healthProbeService provides health check methods for the container.
	healthProbeService healthprobe_domain.Service

	// rateLimitService is the service that limits request rates.
	rateLimitService security_domain.RateLimitService

	// rateLimiter is the centralised rate limiter shared across all domains.
	rateLimiter *ratelimiter_domain.Limiter

	// emailService holds the cached email service instance.
	emailService email_domain.Service

	// llmService holds the cached LLM service instance.
	llmService llm_domain.Service

	// emailDeadLetterAdapter stores failed email messages for later retry.
	emailDeadLetterAdapter email_domain.DeadLetterPort

	// emailDispatcher holds the email dispatcher for monitoring inspection.
	emailDispatcher email_domain.EmailDispatcherPort

	// notificationDispatcher holds the notification dispatcher for monitoring inspection.
	notificationDispatcher notification_domain.NotificationDispatcherPort

	// querierDBService holds the querier database service for named SQL connections and
	// migrations.
	querierDBService *databaseService

	// dbProvider stores the otter persistence provider for the default in-memory backend.
	// Only used when no SQL database is registered via AddDatabase.
	dbProvider *persistence.Provider

	// monitoringService holds the monitoring service; nil when disabled.
	monitoringService monitoring_domain.MonitoringService

	// metricsExporter holds the metrics exporter (e.g., Prometheus). Nil when disabled.
	metricsExporter monitoring_domain.MetricsExporter
}

// containerOverrides holds user-supplied replacements for the default services; a nil
// override means the container builds the default implementation.
type containerOverrides struct {
	// eventBusOverride is a custom EventBus set via WithEventBus; nil uses the default.
	eventBusOverride orchestrator_domain.EventBus

	// registryServiceOverride is a custom registry service; nil uses the default.
	registryServiceOverride registry_domain.RegistryService

	// capabilityServiceOverride holds a custom capability service; nil uses the default.
	capabilityServiceOverride capabilities.Service

	// orchestratorServiceOverride holds an optional replacement for the default orchestrator
	// service; nil uses the default.
	orchestratorServiceOverride orchestrator_domain.OrchestratorService

	// renderRegistryOverride is an optional registry used instead of the default.
	renderRegistryOverride render_domain.RegistryPort

	// csrfServiceOverride replaces the default CSRF token service when set.
	csrfServiceOverride security_domain.CSRFTokenService

	// csrfCookieSourceOverride replaces the default CSRF cookie source when set.
	csrfCookieSourceOverride security_domain.CSRFCookieSourceAdapter

	// pmlTransformerOverride holds a custom PML transformer; nil uses the default.
	pmlTransformerOverride pml_domain.Transformer

	// rendererOverride is a custom renderer; nil uses the default.
	rendererOverride render_domain.RenderService

	// i18nServiceOverride holds a custom i18n service; nil uses the default.
	i18nServiceOverride i18n_domain.Service

	// resolverOverride is a custom resolver; nil uses the default.
	resolverOverride resolver_domain.ResolverPort

	// annotatorServiceOverride is a custom annotator service; nil uses the default.
	annotatorServiceOverride annotator_domain.AnnotatorPort

	// coordinatorCacheOverride is a custom coordinator cache; nil uses the default.
	coordinatorCacheOverride coordinator_domain.BuildResultCachePort

	// introspectionCacheOverride is a custom introspection cache (Tier 1); nil uses the
	// default.
	introspectionCacheOverride coordinator_domain.IntrospectionCachePort

	// coordinatorCodeEmitterOverride overrides the code emitter; used for testing.
	coordinatorCodeEmitterOverride coordinator_domain.CodeEmitterPort

	// coordinatorClientScriptEmitterOverride overrides the client-side script emitter used
	// by the coordinator in dev-i mode; nil disables emission (which skips per-component
	// <script> tags in the rendered page).
	coordinatorClientScriptEmitterOverride coordinator_domain.ClientScriptEmitterPort

	// coordinatorDiagnosticOutputOverride replaces the default diagnostic output; nil uses
	// CLIDiagnosticOutput.
	coordinatorDiagnosticOutputOverride coordinator_domain.DiagnosticOutputPort

	// coordinatorFSReaderOverride is a custom file system reader; nil uses the default.
	coordinatorFSReaderOverride annotator_domain.FSReaderPort

	// coordinatorFileHashCacheOverride is a custom file hash cache; nil uses the default.
	coordinatorFileHashCacheOverride coordinator_domain.FileHashCachePort

	// generatorServiceOverride is a custom generator service; nil uses the default.
	generatorServiceOverride generator_domain.GeneratorService

	// emailServiceOverride holds a custom email service for testing; nil uses the default.
	emailServiceOverride email_domain.Service

	// llmServiceOverride is a custom LLM service for testing; nil uses the default.
	llmServiceOverride llm_domain.Service

	// imageServiceOverride is a custom image service; nil uses the default.
	imageServiceOverride image_domain.Service

	// videoServiceOverride holds a custom video service; nil uses the default.
	videoServiceOverride video_domain.Service

	// storageServiceOverride holds a user-provided storage service; nil uses the default.
	storageServiceOverride storage_domain.Service

	// seoServiceOverride holds a custom SEO service; nil uses the default.
	seoServiceOverride seo_domain.SEOService

	// searchServiceOverride holds a custom search service; nil uses the default.
	searchServiceOverride collection_domain.SearchServicePort

	// coordinatorServiceOverride is a custom coordinator service; nil uses the default.
	coordinatorServiceOverride coordinator_domain.CoordinatorService

	// eventsProviderOverride holds a custom events provider; nil uses the default.
	eventsProviderOverride events_domain.Provider

	// validatorOverride is a custom validator instance; nil uses the default.
	validatorOverride StructValidator

	// typeInspectorBuilderOverride is a custom type inspector builder; nil uses the default.
	typeInspectorBuilderOverride *inspector_domain.TypeBuilder
}

// containerProviders holds the providers, adapters, and extensions registered through
// options, together with the names of the default providers.
type containerProviders struct {
	// markdownParser holds the user-provided markdown parser implementation.
	markdownParser markdown_domain.MarkdownParserPort

	// queryObserver receives a QueryObservation after each instrumented database statement
	// (registered via WithQueryObserver). Nil when no observer was registered.
	queryObserver monitoring_domain.QueryObserver

	// seoURLProvider supplies additional sitemap URLs at build time, in-process; nil means
	// no extra build-time URLs.
	seoURLProvider seo_domain.SitemapURLProvider

	// authProvider resolves authentication state from HTTP requests. Nil means no auth
	// middleware is installed.
	authProvider daemon_dto.AuthProvider

	// spamdetectFeedbackStore holds a deferred feedback store applied when the spam
	// detection service is lazily created.
	spamdetectFeedbackStore spamdetect_domain.FeedbackStore

	// highlighter provides syntax highlighting for code blocks.
	highlighter highlight_domain.Highlighter

	// imageTransformers maps provider names to their image transformer instances.
	imageTransformers map[string]image_domain.TransformerPort

	// notificationProviders maps provider names to their notification handlers.
	notificationProviders map[string]notification_domain.NotificationProviderPort

	// cacheProviders maps provider names to cache provider instances.
	cacheProviders map[string]cache_domain.Provider

	// storageProviders maps names to storage provider instances for data storage.
	storageProviders map[string]storage_domain.StorageProviderPort

	// imagePredefinedVariants maps variant names to their transformation settings.
	imagePredefinedVariants map[string]image_dto.TransformationSpec

	// videoTranscoders maps provider names to video transcoder instances.
	videoTranscoders map[string]video_domain.TranscoderPort

	// customFrontendModules holds custom frontend modules keyed by name.
	customFrontendModules map[string]*daemon_frontend.CustomFrontendModule

	// dbRegistrations maps names to database registration configs. Populated by AddDatabase
	// and consumed lazily by GetDatabaseService.
	dbRegistrations map[string]*DatabaseRegistration

	// spamdetectDetectors maps detector names to their spam detection handlers.
	spamdetectDetectors map[string]spamdetect_domain.Detector

	// captchaProviders maps provider names to their captcha handlers.
	captchaProviders map[string]captcha_domain.CaptchaProvider

	// cryptoProviders maps provider names to their encryption handlers.
	cryptoProviders map[string]crypto_domain.EncryptionProvider

	// llmEmbeddingProviders maps names to standalone embedding-only providers registered via
	// AddEmbeddingProvider.
	llmEmbeddingProviders map[string]llm_domain.EmbeddingProviderPort

	// llmProviders maps provider names to their LLM provider handlers.
	llmProviders map[string]llm_domain.LLMProviderPort

	// emailProviders maps provider names to their email delivery handlers.
	emailProviders map[string]email_domain.EmailProviderPort

	// emailDefaultProvider is the name of the default email provider.
	emailDefaultProvider string

	// notificationDefaultProvider is the name of the provider to use by default.
	notificationDefaultProvider string

	// defaultImageTransformer is the name of the default image transformer. If empty, the
	// first registered transformer becomes the default.
	defaultImageTransformer string

	// storageDefaultProvider is the name of the default storage provider.
	storageDefaultProvider string

	// cacheDefaultProvider is the name of the default cache provider; empty means the first
	// registered provider becomes the default.
	cacheDefaultProvider string

	// captchaDefaultProvider is the name of the default captcha provider.
	captchaDefaultProvider string

	// cryptoDefaultProvider is the name of the default encryption provider.
	cryptoDefaultProvider string

	// defaultVideoTranscoder is the name of the transcoder to use as the default.
	defaultVideoTranscoder string

	// llmDefaultEmbeddingProvider is the name of the default embedding provider. When empty,
	// falls back to auto-detected embedding support from the default LLM provider.
	llmDefaultEmbeddingProvider string

	// llmDefaultProvider is the name of the default LLM provider.
	llmDefaultProvider string

	// analyticsCollectors holds user-registered backend analytics collectors. Empty means no
	// analytics middleware is installed.
	analyticsCollectors []analytics_domain.Collector

	// routeSources enumerate the concrete URLs for pages bound to a p-route-source
	// directive; registered via WithRouteSource and composable across calls.
	routeSources []seo_domain.RouteSource

	// extraSpanProcessors holds additional OTEL span processors registered via
	// WithSpanProcessor, appended to the tracer provider during OTEL setup alongside the
	// monitoring service's own processor.
	extraSpanProcessors []monitoring_domain.SpanProcessor

	// customHealthProbes stores health probes provided by the application for custom checks.
	customHealthProbes []healthprobe_domain.Probe

	// configResolvers holds the resolvers that process configuration during bootstrap.
	configResolvers []config_domain.Resolver

	// externalComponents holds component definitions added via WithComponents.
	externalComponents []component_dto.ComponentDefinition

	// frontendModules stores the registered frontend modules and their settings.
	frontendModules []daemon_frontend.ModuleEntry
}

// containerSettings holds the configuration values supplied through options and the
// resolved server and website configuration.
type containerSettings struct {
	// embeddedPikoFS holds the embedded .piko filesystem. When set, the container serves
	// runtime data from this filesystem instead of disk, enabling single-binary deployments
	// for static sites.
	embeddedPikoFS fs.FS

	// registryMetadataCacheConfig holds the settings for the registry metadata cache; nil
	// means no cache is used.
	registryMetadataCacheConfig *RegistryMetadataCacheConfig

	// generatorProfilingConfig holds capture-to-disk profiling settings; nil when generator
	// profiling is disabled.
	generatorProfilingConfig *profiler.Config

	// authGuardConfig controls route-level authentication enforcement. Nil means no route
	// protection middleware is installed.
	authGuardConfig *daemon_dto.AuthGuardConfig

	// cspBuilder holds the Content-Security-Policy builder; nil means no CSP is set.
	cspBuilder *security_domain.CSPBuilder

	// profilingConfig holds pprof server settings; nil when profiling is disabled.
	profilingConfig *profiler.Config

	// configServerOverrides holds programmatic config values supplied via individual With*
	// options. These are the highest-precedence values merged into serverConfig during
	// bootstrap.
	configServerOverrides *ServerConfig

	// storageDispatcherConfig holds settings for async storage operations.
	storageDispatcherConfig *storage_domain.DispatcherConfig

	// emailDispatcherConfig holds settings for async email sending; nil uses defaults.
	emailDispatcherConfig *email_dto.DispatcherConfig

	// imageServiceConfigOverride holds a custom image service config; nil uses defaults.
	imageServiceConfigOverride *image_domain.ServiceConfig

	// seoConfigOverride holds a custom SEO config; nil skips SEO service creation.
	seoConfigOverride *config.SEOConfig

	// assetsConfigOverride holds asset profiles and responsive image settings; nil uses an
	// empty config (no profiles).
	assetsConfigOverride *config.AssetsConfig

	// websiteConfigOverride holds a programmatic website configuration provided via
	// WithWebsiteConfig; nil uses the file-based config.json.
	websiteConfigOverride *config.WebsiteConfig

	// cssResetCSS holds the resolved CSS reset content for PK files. When empty, no CSS
	// reset is included in the generated theme CSS.
	cssResetCSS string

	// storagePresignBaseURL is the base URL for presigned storage URLs. This is needed for
	// headless CMS setups where the frontend runs on a different host from the storage
	// service.
	storagePresignBaseURL string

	// storagePublicBaseURL is the base URL for public storage URLs, making them absolute
	// when set or relative when empty, needed for headless CMS setups where the frontend
	// runs on a different host from the storage service.
	storagePublicBaseURL string

	// releaseIDOverride sets the release identifier stamped on build-origin registry
	// variants during generation.
	//
	// When empty the release defaults to the VCS revision. An explicit identifier lets
	// deploys tag releases (canary, A/B) independently of the commit, so coexisting releases
	// are distinguishable.
	releaseIDOverride string

	// moduleNameOverride supplies the Go module name when no go.mod is readable at runtime
	// (single-binary / distroless deploys), so the favicon "@/" alias still resolves. Used
	// only as a fallback when the resolver finds no go.mod.
	moduleNameOverride string

	// crossOriginResourcePolicy overrides the default CORP header value. Empty means use the
	// config default ("same-origin").
	crossOriginResourcePolicy string

	// crashTracebackLevel is the GOTRACEBACK level applied via runtime/debug.SetTraceback at
	// startup; empty keeps the runtime default in place.
	crashTracebackLevel string

	// crashOutputPath is the file path the Go runtime should mirror crash output to via
	// runtime/debug.SetCrashOutput; empty disables the feature and the default behaviour
	// stays in place.
	crashOutputPath string

	// diagnosticDirectory is the unified root for runtime-diagnostic artefacts (crash
	// mirror, watchdog profiles, sidecars, startup history).
	diagnosticDirectory string

	// cspPolicyString holds a raw CSP policy string for complex cases that the structured
	// API cannot handle.
	cspPolicyString string

	// websiteConfig holds the user-facing website metadata supplied via WithWebsiteConfig.
	// Empty when no override is set.
	websiteConfig config.WebsiteConfig

	// registryReleaseSeed is the build seed published as a release layer after the blob
	// stores are built; nil when this deployment publishes nothing.
	registryReleaseSeed []*registry_dto.ArtefactMeta

	// cssTreeShakingSafelist lists CSS class names preserved during tree-shaking.
	cssTreeShakingSafelist []string

	// reportingEndpoints holds the configured reporting endpoints for the
	// Reporting-Endpoints header.
	reportingEndpoints []config.ReportingEndpoint

	// embeddedManifest holds the compiled dist/manifest.bin bytes. When set, the manifest is
	// served from memory instead of read from disk, completing the single-binary story (the
	// manifest lives in dist/, outside .piko).
	embeddedManifest []byte

	// serverConfig holds the resolved server configuration. It is populated during bootstrap
	// by merging configServerOverrides (set by With* options) with struct-tag defaults via
	// config_domain.Load.
	serverConfig ServerConfig

	// embedScope selects how much of the runtime payload generation copies into the embed
	// package. Zero value is EmbedAll.
	embedScope EmbedScope

	// actionResponseCacheMaxBytesOverride is the operator-supplied byte cap on the
	// action-response cache.
	actionResponseCacheMaxBytesOverride uint64

	// csrfTokenMaxAge overrides the default CSRF token maximum age when positive.
	csrfTokenMaxAge time.Duration

	// hybridCacheWriteExpirationOverride is the operator-supplied write expiration on the
	// hybrid-collections cache.
	hybridCacheWriteExpirationOverride time.Duration

	// hybridCacheMaxBytesOverride is the operator-supplied byte cap on the
	// hybrid-collections cache.
	hybridCacheMaxBytesOverride uint64

	// inMemoryRuntimeStoreBytes is the byte budget for the opt-in in-memory registry blob
	// overlay, used only when inMemoryRuntimeStoreEnabled is set.
	inMemoryRuntimeStoreBytes int64
}

// containerFeatureFlags holds the boolean toggles and state flags that switch container
// behaviour on or off.
type containerFeatureFlags struct {
	// iAmACatPerson swaps the large pixel-art mascot for the small ASCII art version. nil
	// means not set (defaults to false).
	iAmACatPerson *bool

	// sriEnabled controls whether Subresource Integrity (SRI) hashes are added to script and
	// link tags. Nil means use the default (enabled).
	sriEnabled *bool

	// seoProductionMode reports whether SEO artefacts are generated for a production run.
	//
	// Nil means unknown, which the SEO service treats as production so a missing wiring path
	// fails open (a live site keeps indexing) rather than de-indexing it. Bootstrap sets it
	// from the daemon run mode.
	seoProductionMode *bool

	// compilerDebugLogsEnabled overrides the default for compiler debug log files. nil means
	// use the constant default (true).
	compilerDebugLogsEnabled *bool

	// startupBannerEnabled controls whether the startup banner is displayed. nil means not
	// set (defaults to true).
	startupBannerEnabled *bool

	// experimentalPrerendering enables static HTML prerendering at generation time.
	experimentalPrerendering bool

	// devWidgetEnabled controls whether the dev tools overlay widget is rendered on pages in
	// dev mode.
	devWidgetEnabled bool

	// hasEmailDispatcher indicates whether an email dispatcher has been set up.
	hasEmailDispatcher bool

	// hasStorageDispatcher indicates whether a storage dispatcher has been set up.
	hasStorageDispatcher bool

	// inMemoryRuntimeStoreEnabled reports whether the opt-in bounded in-memory registry blob
	// overlay is used when no other writable blob store is configured.
	inMemoryRuntimeStoreEnabled bool

	// hasNotificationDispatcher indicates whether a notification dispatcher has been set up.
	hasNotificationDispatcher bool

	// cssTreeShaking enables CSS tree-shaking during scaffold generation.
	cssTreeShaking bool

	// experimentalDwarfLineDirectives enables valid DWARF //line directives in generated
	// code. When false (default), directives use "// line" (with a space) which the Go
	// compiler treats as a plain comment.
	experimentalDwarfLineDirectives bool

	// isBuildTime is set during BuildProject (generation).
	//
	// While set, the registry and orchestrator metadata stores route to the local otter/file
	// DAL regardless of any registered SQL database, so generation never opens a DB
	// connection (CI has none). The SQL codegen path is unaffected.
	isBuildTime bool

	// cspPolicyStringSet tracks whether SetCSPPolicyString was called. This tells apart "not
	// set" from "set to empty string".
	cspPolicyStringSet bool

	// formatGeneratedCode runs go/format.Source on generated code so it is gofmt-canonical.
	formatGeneratedCode bool

	// verifyGeneratedCode re-parses each generated file to confirm it is valid Go.
	verifyGeneratedCode bool

	// useStandardLoader causes the type inspector to use the standard
	// golang.org/x/tools/go/packages.Load instead of the faster quickpackages.Load. This is
	// slower but always stable as a fallback.
	useStandardLoader bool

	// experimentalCommentStripping removes HTML comments from generated output.
	experimentalCommentStripping bool

	// devHotreloadEnabled controls whether the SSE hot-reload JS module is loaded in dev
	// mode to trigger automatic page refreshes on rebuild.
	devHotreloadEnabled bool
}

// containerLifecycle holds the application context, the lifecycle callbacks, and the
// factory functions the container uses while running.
type containerLifecycle struct {
	// appCtx is the application-level context; it is cancelled during shutdown.
	appCtx context.Context

	// appCancel cancels the application context during shutdown.
	appCancel context.CancelCauseFunc

	// profilingServer is the running pprof server; nil until it starts.
	profilingServer *profiler.ServerHandle

	// onServerBound is an optional callback invoked after the main HTTP server binds to a
	// port. Used to print the startup banner with the actual port.
	onServerBound func(address string)

	// onHealthBound is an optional callback invoked after the health server binds to a port.
	onHealthBound func(address string)

	// autoMemoryLimitFunc is called during bootstrap to configure GOMEMLIMIT based on the
	// container's cgroup memory limit. Nil means disabled.
	autoMemoryLimitFunc func() (int64, error)

	// csrfSecretKeyProvider returns the secret key used for CSRF token creation.
	csrfSecretKeyProvider func() []byte

	// metadataCacheProvider provides the metadata cache for the registry service.
	metadataCacheProvider func() registry_domain.MetadataCache

	// typeDataProvider creates a TypeDataProvider for the given cache sandbox.
	typeDataProvider func(sandbox safedisk.Sandbox) inspector_domain.TypeDataProvider

	// sandboxFactory creates sandboxes for filesystem operations; nil uses the default
	// factory from config.
	sandboxFactory SandboxFactory

	// readinessInfoKeyFilter reports whether a provider info key is sensitive and must be
	// dropped before readiness info egresses off-box (registered via
	// WithReadinessInfoKeyFilter). Nil when the built-in default filter applies.
	readinessInfoKeyFilter func(string) bool
}

// newContainerLifecycle creates the lifecycle state with the default registry metadata
// cache provider and no callbacks set.
//
// Returns containerLifecycle which holds the initial lifecycle state.
func newContainerLifecycle() containerLifecycle {
	lifecycle := containerLifecycle{}
	lifecycle.metadataCacheProvider = defaultMetadataCacheProvider
	return lifecycle
}
