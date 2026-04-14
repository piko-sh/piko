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

// This file provides a lightweight, manual Dependency Injection (DI) container
// for the Piko application. It centralises the creation and wiring of all major
// services, ensuring they are initialised lazily and used as singletons.
//
// This container is designed for flexibility using the Functional Options
// pattern. The NewContainer constructor accepts a series of Option functions
// that can override default service implementations or configure how they are
// built. This approach avoids external DI frameworks while providing clean,
// testable, and highly maintainable service management.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"piko.sh/piko/internal/analytics/analytics_domain"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/capabilities"
	"piko.sh/piko/internal/captcha/captcha_domain"
	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/component/component_adapters"
	"piko.sh/piko/internal/component/component_domain"
	"piko.sh/piko/internal/component/component_dto"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/config/config_domain"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/crypto/crypto_domain"
	"piko.sh/piko/internal/daemon/daemon_adapters"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/daemon/daemon_frontend"
	"piko.sh/piko/internal/dispatcher/dispatcher_adapters"
	"piko.sh/piko/internal/dispatcher/dispatcher_domain"
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
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/markdown/markdown_domain"
	"piko.sh/piko/internal/monitoring/monitoring_adapters"
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
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/video/video_domain"
	"piko.sh/piko/wdk/safedisk"
)

// Option configures a Container during initialisation.
type Option func(*Container)

const (
	// errCreateSandboxFactory is the error message format used when sandbox
	// factory creation fails.
	errCreateSandboxFactory = "failed to create sandbox factory: %w"

	// errCreateSourceSandbox is the error format used when sandbox creation fails.
	errCreateSourceSandbox = "failed to create source sandbox: %w"

	// logMessageAutoRegisteredShutdown is the log message used when a service is
	// registered for shutdown handling.
	logMessageAutoRegisteredShutdown = "Auto-registered shutdown for user-provided service"

	// logKeyService is the log key for the service name in shutdown messages.
	logKeyService = "service"

	// logKeyMethod is the log key for the shutdown method name.
	logKeyMethod = "method"

	// logKeyPath is the structured log key for file system paths.
	logKeyPath = "path"
)

// SandboxFactory is a function type that creates sandboxes. Inject a custom
// factory to use mock sandboxes for testing.
//
// Takes name (string) which identifies the sandbox instance.
// Takes baseDir (string) which specifies the root directory for the sandbox.
// Takes mode (safedisk.Mode) which defines the access permissions.
//
// Returns safedisk.Sandbox which provides controlled filesystem access.
// Returns error when sandbox creation fails.
type SandboxFactory func(name, baseDir string, mode safedisk.Mode) (safedisk.Sandbox, error)

// RegistryMetadataCacheConfig configures the metadata cache for the Registry
// service.
type RegistryMetadataCacheConfig struct {
	// MaxWeight is the maximum cache size in bytes.
	MaxWeight uint64

	// TTL is the time-to-live for cache entries; 0 means entries never expire.
	TTL time.Duration

	// StatsEnabled enables the collection of cache statistics when true.
	StatsEnabled bool
}

// Container holds all singleton services and dependencies for the application.
// Fields are unexported to prevent direct modification; configure via Options
// passed to NewContainer.
type Container struct {
	// emailErr holds any error from email service setup.
	emailErr error

	// llmErr holds any error from creating the LLM service.
	llmErr error

	// eventsProvider holds the cached events provider instance.
	eventsProvider events_domain.Provider

	// eventBus sends and receives domain events using publish-subscribe messaging.
	eventBus orchestrator_domain.EventBus

	// registryService holds the registry service for registry operations.
	registryService registry_domain.RegistryService

	// registryMetaStore stores registry metadata and is closed on shutdown.
	registryMetaStore registry_domain.MetadataStore

	// registryMetaCache stores cached metadata for the registry service.
	registryMetaCache registry_domain.MetadataCache

	// typeInspectorBuilderErr holds any error from creating the type inspector.
	typeInspectorBuilderErr error

	// healthProbeErr holds any error from creating the health probe service.
	healthProbeErr error

	// collectionServiceErr holds any error from collection service creation.
	collectionServiceErr error

	// searchServiceErr holds any error that occurred while setting up the search
	// service.
	searchServiceErr error

	// cryptoErr holds any error from creating the crypto service.
	cryptoErr error

	// captchaErr holds any error from creating the captcha service.
	captchaErr error

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

	// capabilityService stores the capability service after it is created.
	capabilityService capabilities.Service

	// orchestratorService holds the orchestrator service instance.
	orchestratorService orchestrator_domain.OrchestratorService

	// resolverErr holds any error that occurred during resolver setup.
	resolverErr error

	// csrfService handles CSRF token creation and validation.
	csrfService security_domain.CSRFTokenService

	// csrfCookieSource stores and reads CSRF tokens from cookies.
	csrfCookieSource security_domain.CSRFCookieSourceAdapter

	// pmlTransformer holds the service that transforms PML documents.
	pmlTransformer pml_domain.Transformer

	// renderer holds the service that produces rendered output.
	renderer render_domain.RenderService

	// i18nService is the cached translation service instance.
	i18nService i18n_domain.Service

	// annotatorService holds the annotator service instance, created when first
	// needed.
	annotatorService annotator_domain.AnnotatorPort

	// coordinatorCache stores build results for reuse.
	coordinatorCache coordinator_domain.BuildResultCachePort

	// coordinatorCacheErr holds any error from initialising the coordinator cache.
	coordinatorCacheErr error

	// coordinatorService is the coordinator service instance.
	coordinatorService coordinator_domain.CoordinatorService

	// generatorService holds the generator service instance.
	generatorService generator_domain.GeneratorService

	// resolver holds the lazily initialised symbol resolver instance.
	resolver resolver_domain.ResolverPort

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

	// collectionService handles collection operations.
	collectionService collection_domain.CollectionService

	// searchService is the search service for collections.
	searchService collection_domain.SearchServicePort

	// healthProbeService provides health check methods for the container.
	healthProbeService healthprobe_domain.Service

	// rateLimitService is the service that limits request rates.
	rateLimitService security_domain.RateLimitService

	// rateLimitServiceErr holds any error from creating the rate limit service.
	rateLimitServiceErr error

	// rateLimiterErr holds any error from creating the centralised rate limiter.
	rateLimiterErr error

	// querierDBErr holds any error from querier database service creation.
	querierDBErr error

	// dbProviderErr holds any error from database provider setup.
	dbProviderErr error

	// emailService holds the cached email service instance.
	emailService email_domain.Service

	// llmService holds the cached LLM service instance.
	llmService llm_domain.Service

	// eventBusOverride is a custom EventBus set via WithEventBus; nil uses
	// the default.
	eventBusOverride orchestrator_domain.EventBus

	// registryServiceOverride is a custom registry service; nil uses the default.
	registryServiceOverride registry_domain.RegistryService

	// capabilityServiceOverride holds a custom capability service; nil uses the
	// default.
	capabilityServiceOverride capabilities.Service

	// orchestratorServiceOverride holds an optional replacement for the default
	// orchestrator service; nil uses the default.
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

	// coordinatorCacheOverride is a custom coordinator cache; nil uses the
	// default.
	coordinatorCacheOverride coordinator_domain.BuildResultCachePort

	// introspectionCacheOverride is a custom introspection cache (Tier 1);
	// nil uses the default.
	introspectionCacheOverride coordinator_domain.IntrospectionCachePort

	// coordinatorCodeEmitterOverride overrides the code emitter; used for testing.
	coordinatorCodeEmitterOverride coordinator_domain.CodeEmitterPort

	// coordinatorClientScriptEmitterOverride overrides the client-side
	// script emitter used by the coordinator in dev-i mode; nil disables
	// emission (which skips per-component <script> tags in the rendered
	// page).
	coordinatorClientScriptEmitterOverride coordinator_domain.ClientScriptEmitterPort

	// coordinatorDiagnosticOutputOverride replaces the default diagnostic
	// output; nil uses CLIDiagnosticOutput.
	coordinatorDiagnosticOutputOverride coordinator_domain.DiagnosticOutputPort

	// coordinatorFSReaderOverride is a custom file system reader; nil uses the
	// default.
	coordinatorFSReaderOverride annotator_domain.FSReaderPort

	// coordinatorFileHashCacheOverride is a custom file hash cache; nil uses the
	// default.
	coordinatorFileHashCacheOverride coordinator_domain.FileHashCachePort

	// highlighter provides syntax highlighting for code blocks.
	highlighter highlight_domain.Highlighter

	// generatorServiceOverride is a custom generator service; nil uses the default.
	generatorServiceOverride generator_domain.GeneratorService

	// emailServiceOverride holds a custom email service for testing; nil uses the
	// default.
	emailServiceOverride email_domain.Service

	// llmServiceOverride is a custom LLM service for testing; nil uses the default.
	llmServiceOverride llm_domain.Service

	// imageServiceOverride is a custom image service; nil uses the default.
	imageServiceOverride image_domain.Service

	// videoServiceOverride holds a custom video service; nil uses the default.
	videoServiceOverride video_domain.Service

	// storageServiceOverride holds a user-provided storage service; nil uses the
	// default.
	storageServiceOverride storage_domain.Service

	// seoServiceOverride holds a custom SEO service; nil uses the default.
	seoServiceOverride seo_domain.SEOService

	// searchServiceOverride holds a custom search service; nil uses the default.
	searchServiceOverride collection_domain.SearchServicePort

	// emailDeadLetterAdapter stores failed email messages for later retry.
	emailDeadLetterAdapter email_domain.DeadLetterPort

	// emailDispatcher holds the email dispatcher for monitoring inspection.
	emailDispatcher email_domain.EmailDispatcherPort

	// notificationDispatcher holds the notification dispatcher for monitoring
	// inspection.
	notificationDispatcher notification_domain.NotificationDispatcherPort

	// coordinatorServiceOverride is a custom coordinator service; nil uses the
	// default.
	coordinatorServiceOverride coordinator_domain.CoordinatorService

	// eventsProviderOverride holds a custom events provider; nil uses the default.
	eventsProviderOverride events_domain.Provider

	// renderRegistry holds the render registry instance.
	renderRegistry render_domain.RegistryPort

	// appCtx is the application-level context; it is cancelled during shutdown.
	appCtx context.Context

	// generatorErr holds any error from creating the generator service.
	generatorErr error

	// coordinatorErr stores any error that occurred when creating the
	// coordinator service.
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

	// eventsProviderErr holds any error that occurred when creating the events
	// provider.
	eventsProviderErr error

	// monitoringService holds the monitoring service; nil when disabled.
	monitoringService monitoring_domain.MonitoringService

	// metricsExporter holds the metrics exporter (e.g., Prometheus).
	// Nil when disabled.
	metricsExporter monitoring_domain.MetricsExporter

	// orchestratorInspector holds the orchestrator inspector for monitoring; nil
	// when not available.
	orchestratorInspector orchestrator_domain.OrchestratorInspector

	// registryInspector holds the registry inspector for monitoring; nil when not
	// available.
	registryInspector registry_domain.RegistryInspector

	// componentRegistry holds the component registry for tag lookup.
	componentRegistry component_domain.ComponentRegistry

	// validatorOverride is a custom validator instance; nil uses the default.
	validatorOverride StructValidator

	// validator stores the validation instance for struct and field checks.
	validator StructValidator

	// authProvider resolves authentication state from HTTP requests.
	// Nil means no auth middleware is installed.
	authProvider daemon_dto.AuthProvider

	// analyticsCollectors holds user-registered backend analytics
	// collectors. Empty means no analytics middleware is installed.
	analyticsCollectors []analytics_domain.Collector

	// sandboxFactoryInstance is the lazily-created, cached safedisk.Factory
	// built from the server config. All production sandbox creation goes
	// through this single factory so that path validation, the Enabled flag,
	// and the purpose string are applied consistently.
	sandboxFactoryInstance safedisk.Factory

	// sandboxFactoryErr records any error from creating the cached factory.
	sandboxFactoryErr error

	// markdownParser holds the user-provided markdown parser implementation.
	markdownParser markdown_domain.MarkdownParserPort

	// dbProvider stores the otter persistence provider for the default
	// in-memory backend. Only used when no SQL database is registered via
	// AddDatabase.
	dbProvider *persistence.Provider

	// registryCacheOverride is an optional cache provider for the registry
	// DAL. When set, the registry uses this provider instead of the default
	// otter in-memory backend. This enables serverless deployments where the
	// registry is backed by DynamoDB, Firestore, or another cache provider.
	registryCacheOverride cache_domain.ProviderPort[string, *registry_dto.ArtefactMeta]

	// orchestratorCacheOverride is an optional cache provider for the
	// orchestrator DAL. When set, the orchestrator uses this provider instead
	// of the default otter in-memory backend.
	orchestratorCacheOverride cache_domain.ProviderPort[string, *orchestrator_domain.Task]

	// profilingConfig holds pprof server settings; nil when profiling is
	// disabled.
	profilingConfig *profiler.Config

	// querierDBService holds the querier database service for named SQL
	// connections and migrations.
	querierDBService *databaseService

	// generatorProfilingConfig holds capture-to-disk profiling settings; nil
	// when generator profiling is disabled.
	generatorProfilingConfig *profiler.Config

	// rateLimiter is the centralised rate limiter shared across all domains.
	rateLimiter *ratelimiter_domain.Limiter

	// cspBuilder holds the Content-Security-Policy builder; nil means no CSP is
	// set.
	cspBuilder *security_domain.CSPBuilder

	// dbRegistrations maps names to database registration configs. Populated
	// by AddDatabase and consumed lazily by GetDatabaseService.
	dbRegistrations map[string]*DatabaseRegistration

	// storageProviders maps names to storage provider instances for data storage.
	storageProviders map[string]storage_domain.StorageProviderPort

	// typeDataProvider creates a TypeDataProvider for the given cache sandbox.
	typeDataProvider func(sandbox safedisk.Sandbox) inspector_domain.TypeDataProvider

	// sandboxFactory creates sandboxes for filesystem operations; nil uses the
	// default factory from config.
	sandboxFactory SandboxFactory

	// typeInspectorBuilderOverride is a custom type inspector builder; nil uses
	// the default.
	typeInspectorBuilderOverride *inspector_domain.TypeBuilder

	// config provides access to server and website configuration loading.
	config *config.Provider

	// configServerDefaults holds the default values for server settings.
	// These are the lowest-precedence values, overwritten by YAML/env/flags.
	configServerDefaults *config.ServerConfig

	// configServerOverrides holds programmatic overrides that always win over
	// YAML, env vars, and flags. Set by WithXxx options that modify config values.
	configServerOverrides *config.ServerConfig

	// artefactBridge links artefact processing to the orchestrator workflow.
	artefactBridge *orchestrator_adapters.ArtefactWorkflowBridge

	// typeInspectorBuilder holds the type inspector used to analyse Go types.
	typeInspectorBuilder *inspector_domain.TypeBuilder

	// customFrontendModules holds custom frontend modules keyed by name.
	customFrontendModules map[string]*daemon_frontend.CustomFrontendModule

	// emailProviders maps provider names to their email delivery handlers.
	emailProviders map[string]email_domain.EmailProviderPort

	// llmProviders maps provider names to their LLM provider handlers.
	llmProviders map[string]llm_domain.LLMProviderPort

	// llmEmbeddingProviders maps names to standalone embedding-only providers
	// registered via AddEmbeddingProvider.
	llmEmbeddingProviders map[string]llm_domain.EmbeddingProviderPort

	// cryptoProviders maps provider names to their encryption handlers.
	cryptoProviders map[string]crypto_domain.EncryptionProvider

	// captchaProviders maps provider names to their captcha handlers.
	captchaProviders map[string]captcha_domain.CaptchaProvider

	// notificationProviders maps provider names to their notification handlers.
	notificationProviders map[string]notification_domain.NotificationProviderPort

	// cacheProviders maps provider names to cache provider instances.
	cacheProviders map[string]cache_domain.Provider

	// imageTransformers maps provider names to their image transformer instances.
	imageTransformers map[string]image_domain.TransformerPort

	// imagePredefinedVariants maps variant names to their transformation settings.
	imagePredefinedVariants map[string]image_dto.TransformationSpec

	// imageServiceConfigOverride holds a custom image service config; nil uses
	// defaults.
	imageServiceConfigOverride *image_domain.ServiceConfig

	// seoConfigOverride holds a custom SEO config; nil skips SEO service creation.
	seoConfigOverride *config.SEOConfig

	// assetsConfigOverride holds asset profiles and responsive image settings;
	// nil uses an empty config (no profiles).
	assetsConfigOverride *config.AssetsConfig

	// websiteConfigOverride holds a programmatic website configuration
	// provided via WithWebsiteConfig; nil uses the file-based config.json.
	websiteConfigOverride *config.WebsiteConfig

	// crashOutputPath is the file path the Go runtime should mirror crash
	// output to via runtime/debug.SetCrashOutput; empty disables the
	// feature and the default behaviour stays in place.
	crashOutputPath string

	// crashTracebackLevel is the GOTRACEBACK level applied via
	// runtime/debug.SetTraceback at startup; empty keeps the runtime
	// default in place.
	crashTracebackLevel string

	// diagnosticDirectory is the unified root for runtime-diagnostic
	// artefacts (crash mirror, watchdog profiles, sidecars, startup
	// history).
	diagnosticDirectory string

	// videoTranscoders maps provider names to video transcoder instances.
	videoTranscoders map[string]video_domain.TranscoderPort

	// csrfSecretKeyProvider returns the secret key used for CSRF token creation.
	csrfSecretKeyProvider func() []byte

	// metadataCacheProvider provides the metadata cache for the registry service.
	metadataCacheProvider func() registry_domain.MetadataCache

	// appCancel cancels the application context during shutdown.
	appCancel context.CancelCauseFunc

	// registryMetadataCacheConfig holds the settings for the registry metadata
	// cache; nil means no cache is used.
	registryMetadataCacheConfig *RegistryMetadataCacheConfig

	// storageDispatcherConfig holds settings for async storage operations.
	storageDispatcherConfig *storage_domain.DispatcherConfig

	// emailDispatcherConfig holds settings for async email sending; nil uses
	// defaults.
	emailDispatcherConfig *email_dto.DispatcherConfig

	// startupBannerEnabled controls whether the startup banner is displayed.
	// nil means not set (defaults to true).
	startupBannerEnabled *bool

	// iAmACatPerson swaps the large pixel-art mascot for the small ASCII art
	// version. nil means not set (defaults to false).
	iAmACatPerson *bool

	// compilerDebugLogsEnabled overrides the default for compiler debug log
	// files. nil means use the constant default (true).
	compilerDebugLogsEnabled *bool

	// autoMemoryLimitFunc is called during bootstrap to configure GOMEMLIMIT
	// based on the container's cgroup memory limit. Nil means disabled.
	autoMemoryLimitFunc func() (int64, error)

	// authGuardConfig controls route-level authentication enforcement.
	// Nil means no route protection middleware is installed.
	authGuardConfig *daemon_dto.AuthGuardConfig

	// sriEnabled controls whether Subresource Integrity (SRI) hashes are
	// added to script and link tags. Nil means use the default (enabled).
	sriEnabled *bool

	// onServerBound is an optional callback invoked after the main HTTP server
	// binds to a port. Used to print the startup banner with the actual port.
	onServerBound func(address string)

	// onHealthBound is an optional callback invoked after the health server
	// binds to a port.
	onHealthBound func(address string)

	// cspPolicyString holds a raw CSP policy string for complex cases that the
	// structured API cannot handle.
	cspPolicyString string

	// emailDefaultProvider is the name of the default email provider.
	emailDefaultProvider string

	// llmDefaultProvider is the name of the default LLM provider.
	llmDefaultProvider string

	// llmDefaultEmbeddingProvider is the name of the default embedding
	// provider. When empty, falls back to auto-detected embedding support
	// from the default LLM provider.
	llmDefaultEmbeddingProvider string

	// cryptoDefaultProvider is the name of the default encryption provider.
	cryptoDefaultProvider string

	// captchaDefaultProvider is the name of the default captcha provider.
	captchaDefaultProvider string

	// notificationDefaultProvider is the name of the provider to use by default.
	notificationDefaultProvider string

	// cacheDefaultProvider is the name of the default cache provider; empty
	// means the first registered provider becomes the default.
	cacheDefaultProvider string

	// storageDefaultProvider is the name of the default storage provider.
	storageDefaultProvider string

	// storagePresignBaseURL is the base URL for presigned storage URLs.
	// This is needed for headless CMS setups where the frontend runs on a
	// different host from the storage service.
	storagePresignBaseURL string

	// storagePublicBaseURL is the base URL for public storage URLs, making
	// them absolute when set or relative when empty, needed for headless CMS
	// setups where the frontend runs on a different host from the storage
	// service.
	storagePublicBaseURL string

	// defaultImageTransformer is the name of the default image transformer.
	// If empty, the first registered transformer becomes the default.
	defaultImageTransformer string

	// defaultVideoTranscoder is the name of the transcoder to use as the default.
	defaultVideoTranscoder string

	// crossOriginResourcePolicy overrides the default CORP header value.
	// Empty means use the config default ("same-origin").
	crossOriginResourcePolicy string

	// cssResetCSS holds the resolved CSS reset content for PK files. When
	// empty, no CSS reset is included in the generated theme CSS.
	cssResetCSS string

	// reportingEndpoints holds the configured reporting endpoints for the
	// Reporting-Endpoints header.
	reportingEndpoints []config.ReportingEndpoint

	// frontendModules stores the registered frontend modules and their settings.
	frontendModules []daemon_frontend.ModuleEntry

	// configResolvers holds the resolvers that process configuration during
	// bootstrap.
	configResolvers []config_domain.Resolver

	// customHealthProbes stores health probes provided by the application for
	// custom checks.
	customHealthProbes []healthprobe_domain.Probe

	// externalComponents holds component definitions added via WithComponents.
	externalComponents []component_dto.ComponentDefinition

	// cssTreeShakingSafelist lists CSS class names preserved during tree-shaking.
	cssTreeShakingSafelist []string

	// csrfTokenMaxAge overrides the default CSRF token maximum age when positive.
	csrfTokenMaxAge time.Duration

	// sandboxFactoryOnce guards single initialisation of the cached sandbox
	// factory.
	sandboxFactoryOnce sync.Once

	// annotatorOnce guards single initialisation of the annotator service.
	annotatorOnce sync.Once

	// querierDBOnce guards single initialisation of the querier database
	// service.
	querierDBOnce sync.Once

	// dbProviderOnce guards single initialisation of the database provider.
	dbProviderOnce sync.Once

	// i18nOnce guards single initialisation of the i18n service.
	i18nOnce sync.Once

	// generatorOnce guards single initialisation of the generator service.
	generatorOnce sync.Once

	// registryOnce guards single initialisation of the registry service.
	registryOnce sync.Once

	// capabilityOnce guards single initialisation of the capability service.
	capabilityOnce sync.Once

	// orchestratorOnce guards single initialisation of the orchestrator service.
	orchestratorOnce sync.Once

	// renderRegOnce guards single initialisation of the render registry.
	renderRegOnce sync.Once

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

	// videoOnce guards single initialisation of the video service.
	videoOnce sync.Once

	// storageOnce guards single initialisation of the storage service.
	storageOnce sync.Once

	// seoOnce guards single initialisation of the SEO service.
	seoOnce sync.Once

	// cacheOnce guards single initialisation of the cache service.
	cacheOnce sync.Once

	// cryptoOnce guards single initialisation of the crypto service.
	cryptoOnce sync.Once

	// captchaOnce guards single initialisation of the captcha service.
	captchaOnce sync.Once

	// validatorOnce guards single initialisation of the validator.
	validatorOnce sync.Once

	// collectionServiceOnce guards single initialisation of the collection
	// service.
	collectionServiceOnce sync.Once

	// searchServiceOnce guards single initialisation of the search service.
	searchServiceOnce sync.Once

	// healthProbeOnce guards single initialisation of the health probe service.
	healthProbeOnce sync.Once

	// rateLimitServiceOnce guards single initialisation of the rate limit service.
	rateLimitServiceOnce sync.Once

	// rateLimiterOnce guards single initialisation of the centralised rate
	// limiter.
	rateLimiterOnce sync.Once

	// componentRegistryOnce guards single initialisation of the component
	// registry.
	componentRegistryOnce sync.Once

	// cspPolicyStringSet tracks whether SetCSPPolicyString was called.
	// This tells apart "not set" from "set to empty string".
	cspPolicyStringSet bool

	// hasEmailDispatcher indicates whether an email dispatcher has been set up.
	hasEmailDispatcher bool

	// hasStorageDispatcher indicates whether a storage dispatcher has been set up.
	hasStorageDispatcher bool

	// hasNotificationDispatcher indicates whether a notification dispatcher has
	// been set up.
	hasNotificationDispatcher bool

	// cssTreeShaking enables CSS tree-shaking during scaffold generation.
	cssTreeShaking bool

	// experimentalPrerendering enables static HTML prerendering at generation
	// time.
	experimentalPrerendering bool

	// experimentalCommentStripping removes HTML comments from generated output.
	experimentalCommentStripping bool

	// experimentalDwarfLineDirectives enables valid DWARF //line directives
	// in generated code. When false (default), directives use "// line" (with
	// a space) which the Go compiler treats as a plain comment.
	experimentalDwarfLineDirectives bool

	// useStandardLoader causes the type inspector to use the standard
	// golang.org/x/tools/go/packages.Load instead of the faster
	// quickpackages.Load. This is slower but always stable as a fallback.
	useStandardLoader bool

	// devWidgetEnabled controls whether the dev tools overlay widget is
	// rendered on pages in dev mode.
	devWidgetEnabled bool

	// devHotreloadEnabled controls whether the SSE hot-reload JS module is
	// loaded in dev mode to trigger automatic page refreshes on rebuild.
	devHotreloadEnabled bool
}

// NewContainer creates a new dependency injection container.
//
// It sets sensible defaults and then applies any options provided.
//
// Takes configProvider (*config.Provider) which provides access
// to configuration values.
// Takes opts (...Option) which are options to change behaviour.
//
// Returns *Container which is the configured dependency injection container.
func NewContainer(configProvider *config.Provider, opts ...Option) *Container {
	c := &Container{
		config:                configProvider,
		metadataCacheProvider: defaultMetadataCacheProvider,
		csrfSecretKeyProvider: func() []byte {
			return resolveCSRFSecret(deref(configProvider.ServerConfig.CSRFSecret, ""))
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// GetAppContext returns the application-wide context that lives until shutdown.
//
// Returns context.Context which stays active until the application shuts down.
func (c *Container) GetAppContext() context.Context {
	c.appCtxOnce.Do(func() {
		c.appCtx, c.appCancel = context.WithCancelCause(context.Background())
		shutdown.Register(c.appCtx, "AppCancel", func(_ context.Context) error {
			c.appCancel(errors.New("application shutting down"))
			return nil
		})
	})
	return c.appCtx
}

// GetConfigProvider returns the application configuration provider.
//
// Returns *config.Provider which provides access to application settings.
func (c *Container) GetConfigProvider() *config.Provider {
	return c.config
}

// GetSandboxFactory returns the cached safedisk.Factory built from the server
// configuration. The factory is created lazily on first call and reused for all
// subsequent calls, ensuring consistent path validation and sandbox mode across
// the application.
//
// Returns safedisk.Factory which creates sandboxes with the configured allowed
// paths and enabled/disabled mode.
// Returns error when the factory cannot be created from the server config.
func (c *Container) GetSandboxFactory() (safedisk.Factory, error) {
	c.sandboxFactoryOnce.Do(func() {
		serverConfig := c.config.ServerConfig
		c.sandboxFactoryInstance, c.sandboxFactoryErr = safedisk.NewFactory(safedisk.FactoryConfig{
			Enabled:      deref(serverConfig.Security.Sandbox.Enabled, true),
			AllowedPaths: serverConfig.Security.Sandbox.AllowedPaths,
			CWD:          deref(serverConfig.Paths.BaseDir, "."),
		})
	})
	return c.sandboxFactoryInstance, c.sandboxFactoryErr
}

// IsDevWidgetEnabled reports whether the dev tools overlay widget is enabled.
//
// Returns bool which is true when the widget should be rendered in dev mode.
func (c *Container) IsDevWidgetEnabled() bool { return c.devWidgetEnabled }

// IsDevHotreloadEnabled reports whether the SSE hot-reload JS module is
// enabled.
//
// Returns bool which is true when automatic page refresh should be active in
// dev mode.
func (c *Container) IsDevHotreloadEnabled() bool { return c.devHotreloadEnabled }

// SetOnServerBound stores a callback to invoke when the main HTTP server
// binds to a port.
//
// Takes fn (func(address string)) which is the callback receiving the resolved
// listen address.
func (c *Container) SetOnServerBound(fn func(address string)) { c.onServerBound = fn }

// OnServerBound returns the stored server-bound callback, or nil.
//
// Returns func(address string) which is the callback, or nil if not set.
func (c *Container) OnServerBound() func(address string) { return c.onServerBound }

// SetOnHealthBound stores a callback to invoke when the health server
// binds to a port.
//
// Takes fn (func(address string)) which is the callback receiving the resolved
// listen address.
func (c *Container) SetOnHealthBound(fn func(address string)) { c.onHealthBound = fn }

// OnHealthBound returns the stored health-bound callback, or nil.
//
// Returns func(address string) which is the callback, or nil if not set.
func (c *Container) OnHealthBound() func(address string) { return c.onHealthBound }

// IsSRIEnabled reports whether Subresource Integrity hashes should be added
// to script and link tags. Returns true by default unless explicitly disabled
// via WithSRI(false).
//
// Returns bool which is true when SRI integrity attributes should be emitted.
func (c *Container) IsSRIEnabled() bool {
	if c.sriEnabled != nil {
		return *c.sriEnabled
	}
	return true
}

// SetCompilerDebugLogsEnabled overrides the default for compiler debug log
// files. Use this to disable debug log files in contexts like the LSP where
// they are not needed.
//
// Takes enabled (bool) which controls whether debug log files are written.
func (c *Container) SetCompilerDebugLogsEnabled(enabled bool) {
	c.compilerDebugLogsEnabled = &enabled
}

// IsStartupBannerEnabled returns whether the startup banner should be
// displayed. Defaults to true when not explicitly set.
//
// Returns bool which is true when the banner should be shown.
func (c *Container) IsStartupBannerEnabled() bool {
	if c.startupBannerEnabled != nil {
		return *c.startupBannerEnabled
	}
	return true
}

// IsIAmACatPerson returns whether the large pixel-art mascot should be
// replaced with the small ASCII art version. Defaults to false.
//
// Returns bool which is true when the small mascot should be used.
func (c *Container) IsIAmACatPerson() bool {
	if c.iAmACatPerson != nil {
		return *c.iAmACatPerson
	}
	return false
}

// IsCSSTreeShakingEnabled returns whether CSS tree-shaking is enabled.
//
// Returns bool which is true when CSS tree-shaking is active.
func (c *Container) IsCSSTreeShakingEnabled() bool {
	return c.cssTreeShaking
}

// GetCSSTreeShakingSafelist returns the CSS classes preserved during
// tree-shaking.
//
// Returns []string which lists CSS class names that are never removed.
func (c *Container) GetCSSTreeShakingSafelist() []string {
	return c.cssTreeShakingSafelist
}

// GetCSSResetCSS returns the resolved CSS reset content for PK files. When
// empty, no CSS reset should be included in theme CSS output.
//
// Returns string which is the CSS reset content, or empty when disabled.
func (c *Container) GetCSSResetCSS() string {
	return c.cssResetCSS
}

// IsExperimentalPrerenderingEnabled returns whether static HTML prerendering
// is enabled at generation time.
//
// Returns bool which is true when prerendering is active.
func (c *Container) IsExperimentalPrerenderingEnabled() bool {
	return c.experimentalPrerendering
}

// IsExperimentalCommentStrippingEnabled returns whether HTML comment stripping
// is enabled for generated output.
//
// Returns bool which is true when comment stripping is active.
func (c *Container) IsExperimentalCommentStrippingEnabled() bool {
	return c.experimentalCommentStripping
}

// IsExperimentalDwarfLineDirectivesEnabled returns whether valid DWARF //line
// directives are emitted in generated Go code.
//
// Returns bool which is true when DWARF line directives are active.
func (c *Container) IsExperimentalDwarfLineDirectivesEnabled() bool {
	return c.experimentalDwarfLineDirectives
}

// GetActionRegistry returns the action registry.
//
// Returns the global registry populated by auto-generated init() functions.
// Actions are discovered automatically during annotation and generate
// registry.go files that register actions via init().
//
// Returns map[string]daemon_adapters.ActionHandlerEntry which maps action
// names to their handler entries.
func (*Container) GetActionRegistry() map[string]daemon_adapters.ActionHandlerEntry {
	return daemon_adapters.GetGlobalActionRegistry()
}

// GetComponentRegistry returns the PKC component registry for deterministic
// tag lookup during template processing.
//
// The registry is lazily initialised on first access. It contains:
//   - External components registered via WithComponents()
//   - Local components discovered from the components/ folder
//
// Returns component_domain.ComponentRegistry which provides tag name lookups.
func (c *Container) GetComponentRegistry() component_domain.ComponentRegistry {
	c.componentRegistryOnce.Do(func() {
		c.componentRegistry = component_adapters.NewInMemoryRegistry()

		_, l := logger_domain.From(c.GetAppContext(), log)
		for _, definition := range c.externalComponents {
			if err := c.componentRegistry.Register(definition); err != nil {
				l.Error("Failed to register external component",
					logger_domain.String("tag_name", definition.TagName),
					logger_domain.Error(err),
				)
			}
		}

		if len(c.externalComponents) > 0 {
			l.Internal("Registered external components",
				logger_domain.Int("count", len(c.externalComponents)),
			)
		}

		c.discoverLocalComponents()
	})
	return c.componentRegistry
}

// GetMetricsExporter returns the metrics exporter, if configured.
// Returns nil if metrics export was not enabled.
//
// Returns monitoring_domain.MetricsExporter which provides the metrics handler.
func (c *Container) GetMetricsExporter() monitoring_domain.MetricsExporter {
	return c.metricsExporter
}

// SetMetricsExporter sets the metrics exporter. This is called during container
// initialisation when metrics are enabled.
//
// Takes exporter (monitoring_domain.MetricsExporter) which is the exporter to
// use.
func (c *Container) SetMetricsExporter(exporter monitoring_domain.MetricsExporter) {
	c.metricsExporter = exporter
}

// GetMonitoringService returns the full monitoring service, if configured.
// Returns nil if monitoring was not enabled via WithMonitoring().
//
// Returns monitoring_domain.MonitoringService which provides gRPC access and
// OTEL integration.
func (c *Container) GetMonitoringService() monitoring_domain.MonitoringService {
	return c.monitoringService
}

// SetMonitoringService sets the full monitoring service. This is called by
// WithMonitoring during container initialisation.
//
// Takes service (monitoring_domain.MonitoringService) which is the service to
// use.
func (c *Container) SetMonitoringService(service monitoring_domain.MonitoringService) {
	c.monitoringService = service
}

// GetProfilingConfig returns the pprof server configuration, if set.
// Returns nil when profiling was not enabled via WithProfiling().
//
// Returns *profiler.Config which holds the server profiling settings.
func (c *Container) GetProfilingConfig() *profiler.Config {
	return c.profilingConfig
}

// SetProfilingConfig stores the pprof server configuration. This is called
// by WithProfiling during container initialisation.
//
// Takes profilingConfig (*profiler.Config) which provides the profiling settings.
func (c *Container) SetProfilingConfig(profilingConfig *profiler.Config) {
	c.profilingConfig = profilingConfig
}

// GetGeneratorProfilingConfig returns the generator profiling configuration,
// if set. Returns nil when generator profiling was not enabled via
// WithGeneratorProfiling().
//
// Returns *profiler.Config which holds the capture-to-disk settings.
func (c *Container) GetGeneratorProfilingConfig() *profiler.Config {
	return c.generatorProfilingConfig
}

// SetGeneratorProfilingConfig stores the generator profiling configuration.
// This is called by WithGeneratorProfiling during container initialisation.
//
// Takes profilingConfig (*profiler.Config) which provides the
// generator profiling settings.
func (c *Container) SetGeneratorProfilingConfig(profilingConfig *profiler.Config) {
	c.generatorProfilingConfig = profilingConfig
}

// GetOrchestratorInspector returns the orchestrator inspector for monitoring.
// Returns nil if the orchestrator has not been initialised.
//
// Returns orchestrator_domain.OrchestratorInspector which provides read-only
// task data.
func (c *Container) GetOrchestratorInspector() orchestrator_domain.OrchestratorInspector {
	return c.orchestratorInspector
}

// GetRegistryInspector returns the registry inspector for monitoring.
// Returns nil if the registry has not been initialised.
//
// Returns registry_domain.RegistryInspector which provides read-only artefact
// data.
func (c *Container) GetRegistryInspector() registry_domain.RegistryInspector {
	return c.registryInspector
}

// GetMonitoringHealthProbeService returns the health probe service adapted
// for monitoring.
//
// Returns monitoring_domain.HealthProbeService for gRPC health reporting.
// Returns nil when the health probe service is not available.
func (c *Container) GetMonitoringHealthProbeService() monitoring_domain.HealthProbeService {
	healthService, err := c.GetHealthProbeService()
	if err != nil || healthService == nil {
		return nil
	}
	return monitoring_adapters.NewHealthProbeAdapter(healthService)
}

// GetEmailDispatcher returns the email dispatcher, if configured.
// Returns nil if no email dispatcher has been set up.
//
// Returns email_domain.EmailDispatcherPort which provides email dispatch and
// DLQ access.
func (c *Container) GetEmailDispatcher() email_domain.EmailDispatcherPort {
	return c.emailDispatcher
}

// GetNotificationDispatcher returns the notification dispatcher, if configured.
// Returns nil if no notification dispatcher has been set up.
//
// Returns notification_domain.NotificationDispatcherPort which provides
// notification dispatch and DLQ access.
func (c *Container) GetNotificationDispatcher() notification_domain.NotificationDispatcherPort {
	return c.notificationDispatcher
}

// GetDispatcherInspector returns a dispatcher inspector that provides
// read-only access to email and notification dispatcher state and DLQs.
//
// Returns dispatcher_domain.DispatcherInspector which provides unified DLQ
// monitoring, or nil if neither dispatcher is configured.
func (c *Container) GetDispatcherInspector() dispatcher_domain.DispatcherInspector {
	if c.emailDispatcher == nil && c.notificationDispatcher == nil {
		return nil
	}
	return dispatcher_adapters.NewInspector(c.emailDispatcher, c.notificationDispatcher)
}

// GetRateLimiterInspector returns the rate limiter as a RateLimiterInspector,
// or nil if the rate limiter is not available.
//
// Returns ratelimiter_domain.RateLimiterInspector which provides rate limiter
// state inspection, or nil.
func (c *Container) GetRateLimiterInspector() ratelimiter_domain.RateLimiterInspector {
	limiter, err := c.GetRateLimiter()
	if err != nil || limiter == nil {
		return nil
	}
	return limiter
}

// StartMonitoringService starts the monitoring gRPC server if configured.
// This should be called after SetInspectors() has been called to wire
// the orchestrator and registry inspectors.
//
// If the server fails to start, an error is logged but the application
// continues (monitoring is optional observability).
//
// Spawns a goroutine that runs the monitoring gRPC server
// until the application context is cancelled. The server is registered
// for graceful shutdown.
func (c *Container) StartMonitoringService() {
	monitoringService := c.GetMonitoringService()
	if monitoringService == nil {
		return
	}

	appCtx := c.GetAppContext()
	appCtx, l := logger_domain.From(appCtx, log)

	go func() {
		if err := monitoringService.Start(appCtx); err != nil {
			if appCtx.Err() == nil {
				l.Error("Monitoring gRPC server failed",
					logger_domain.Error(err))
			}
		}
	}()

	shutdown.Register(appCtx, "MonitoringService", func(ctx context.Context) error {
		monitoringService.Stop(ctx)
		return nil
	})

	l.Internal(
		"Monitoring gRPC service started",
		logger_domain.String("address", monitoringService.Address()),
	)
}

// StartProfilingServer starts the pprof HTTP server if profiling was enabled
// via WithProfiling. It configures runtime block and mutex profiling rates,
// checks for problematic build flags, and starts the server in a background
// goroutine with graceful shutdown.
//
// This is a no-op when profiling is not configured.
func (c *Container) StartProfilingServer() {
	profilingConfig := c.GetProfilingConfig()
	if profilingConfig == nil {
		return
	}

	_, l := logger_domain.From(c.GetAppContext(), log)

	profiler.SetRuntimeRates(*profilingConfig)

	if warning := profiler.CheckBuildFlags(); warning != "" {
		l.Warn(warning)
	}

	server, err := profiler.StartServer(*profilingConfig)
	if err != nil {
		l.Error("Failed to start profiling server",
			logger_domain.Error(err))
		return
	}
	server.SetErrorHandler(func(err error) {
		l.Error("Profiling server error", logger_domain.Error(err))
	})
	addr := profiler.ServerAddress(*profilingConfig)

	shutdown.Register(c.GetAppContext(), "ProfilingServer", func(ctx context.Context) error {
		return server.Shutdown(ctx)
	})

	base := "http://" + addr + profiler.BasePath

	noticeAttrs := []logger_domain.Attr{
		logger_domain.String("address", base+"/debug/pprof/"),
		logger_domain.Int("block_profile_rate", profilingConfig.BlockProfileRate),
		logger_domain.Int("mutex_profile_fraction", profilingConfig.MutexProfileFraction),
		logger_domain.Int("goroutine_count", profiler.GoroutineCount()),
	}
	if profilingConfig.EnableRollingTrace {
		noticeAttrs = append(noticeAttrs,
			logger_domain.String("rolling_trace_min_age", profilingConfig.RollingTraceMinAge.String()),
			logger_domain.String("rolling_trace_max_bytes", fmt.Sprintf("%.1f MiB", float64(profilingConfig.RollingTraceMaxBytes)/(1024*1024))),
		)
	}

	l.Notice("Profiling server started", noticeAttrs...)
	logProfilingEndpoints(l, base, addr, profilingConfig.EnableRollingTrace)
}

// logProfilingEndpoints logs the available pprof and profiler endpoints.
//
// Takes l (logger_domain.Logger) which is the logger for output.
// Takes base (string) which is the base URL prefix for pprof endpoints.
// Takes addr (string) which is the listen address for status endpoints.
// Takes rollingTraceEnabled (bool) which controls whether the rolling trace
// endpoint is included.
func logProfilingEndpoints(l logger_domain.Logger, base, addr string, rollingTraceEnabled bool) {
	internalAttrs := []logger_domain.Attr{
		logger_domain.String("cpu", base+"/debug/pprof/profile?seconds=30"),
		logger_domain.String("heap", base+"/debug/pprof/heap"),
		logger_domain.String("allocs", base+"/debug/pprof/allocs"),
		logger_domain.String("goroutine", base+"/debug/pprof/goroutine"),
		logger_domain.String("block", base+"/debug/pprof/block"),
		logger_domain.String("mutex", base+"/debug/pprof/mutex"),
		logger_domain.String("trace", base+"/debug/pprof/trace?seconds=5"),
		logger_domain.String("status", "http://"+addr+profiler.ProfilerStatusPath),
	}
	if rollingTraceEnabled {
		internalAttrs = append(internalAttrs,
			logger_domain.String("rolling_trace", "http://"+addr+profiler.RollingTracePath),
		)
	}

	l.Internal("Available pprof endpoints", internalAttrs...)
}

// StartGeneratorProfiling begins capture-to-disk profiling if generator
// profiling was enabled via WithGeneratorProfiling. It configures runtime
// rates, checks build flags, and starts capturing CPU and trace profiles.
//
// Returns func() which must be called (typically via defer) to stop profiling
// and write all profile files. Returns nil when generator profiling is not
// configured.
func (c *Container) StartGeneratorProfiling() func() {
	generatorProfilingConfig := c.GetGeneratorProfilingConfig()
	if generatorProfilingConfig == nil {
		return nil
	}

	_, l := logger_domain.From(c.GetAppContext(), log)

	profiler.SetRuntimeRates(*generatorProfilingConfig)

	if warning := profiler.CheckBuildFlags(); warning != "" {
		l.Warn(warning)
	}

	if generatorProfilingConfig.Sandbox == nil {
		sandbox, sandboxErr := c.createSandbox("profiler-capture", generatorProfilingConfig.OutputDir, safedisk.ModeReadWrite)
		if sandboxErr != nil {
			l.Warn("Failed to create profiler sandbox, using fallback",
				logger_domain.Error(sandboxErr))
		} else {
			generatorProfilingConfig.Sandbox = sandbox
		}
	}

	cleanup, err := profiler.StartCapture(*generatorProfilingConfig)
	if err != nil {
		l.Error("Failed to start generator profiling",
			logger_domain.Error(err))
		return nil
	}

	l.Notice("Generator profiling started",
		logger_domain.String("output_dir", generatorProfilingConfig.OutputDir),
		logger_domain.Int("goroutine_count", profiler.GoroutineCount()),
	)

	return func() {
		cleanup()
		l.Notice("Generator profiling complete",
			logger_domain.String("output_dir", generatorProfilingConfig.OutputDir),
		)
	}
}

// applyAutoMemoryLimit calls the configured auto memory limit function to
// set GOMEMLIMIT based on the container's cgroup memory limit.
//
// This is a no-op when no auto memory limit provider is configured.
func (c *Container) applyAutoMemoryLimit(ctx context.Context) {
	if c.autoMemoryLimitFunc == nil {
		return
	}

	_, l := logger_domain.From(ctx, log)

	limit, err := c.autoMemoryLimitFunc()
	if err != nil {
		l.Warn("Auto memory limit detection skipped", logger_domain.Error(err))
		return
	}

	l.Info("Auto memory limit applied",
		logger_domain.String("GOMEMLIMIT", fmt.Sprintf("%d MiB", limit/(1024*1024))))
}

// ensureOverrides lazily initialises the configServerOverrides struct so that
// option functions can write individual fields without nil-checking.
//
// Returns *config.ServerConfig which provides the override settings to modify.
func (c *Container) ensureOverrides() *config.ServerConfig {
	if c.configServerOverrides == nil {
		c.configServerOverrides = &config.ServerConfig{}
	}
	return c.configServerOverrides
}

// discoverLocalComponents walks the components folder and registers all .pkc
// files in the component registry for deterministic tag lookup.
//
// Component tag names come from the filename without the .pkc extension. For
// example, "my-button.pkc" registers as tag name "my-button".
func (c *Container) discoverLocalComponents() {
	_, l := logger_domain.From(c.GetAppContext(), log)
	serverConfig := c.config.ServerConfig
	componentsDir := deref(serverConfig.Paths.ComponentsSourceDir, "components")
	if componentsDir == "" {
		l.Internal("No components directory configured, skipping local component discovery")
		return
	}

	absDir := filepath.Join(deref(serverConfig.Paths.BaseDir, "."), componentsDir)

	if !validateComponentsDirectory(c.GetAppContext(), absDir) {
		return
	}

	baseDir := deref(serverConfig.Paths.BaseDir, ".")
	registered, regErrors := c.walkAndRegisterComponents(absDir, baseDir)
	logComponentDiscoveryResults(c.GetAppContext(), registered, regErrors, componentsDir)
}

// walkAndRegisterComponents walks the directory and registers .pkc files.
//
// Takes absDir (string) which is the absolute path to the directory to walk.
// Takes baseDir (string) which is the base path for computing relative paths.
//
// Returns int which is the count of successfully registered components.
// Returns []string which contains error messages for any failed registrations.
func (c *Container) walkAndRegisterComponents(absDir, baseDir string) (int, []string) {
	_, l := logger_domain.From(c.GetAppContext(), log)
	var registered int
	var regErrors []string

	walkErr := filepath.WalkDir(absDir, func(absPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walking component directory %q: %w", absPath, err)
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".pkc") {
			return nil
		}

		tagName := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		relPath, _ := filepath.Rel(baseDir, absPath)

		definition := component_dto.ComponentDefinition{
			TagName:    tagName,
			SourcePath: relPath,
			IsExternal: false,
		}

		if regErr := c.componentRegistry.Register(definition); regErr != nil {
			regErrors = append(regErrors, fmt.Sprintf("%s: %v", tagName, regErr))
		} else {
			registered++
		}
		return nil
	})

	if walkErr != nil {
		l.Warn("Error walking components directory",
			logger_domain.String(logKeyPath, absDir),
			logger_domain.Error(walkErr))
	}

	return registered, regErrors
}

// contextCloser defines a service that can be closed with a context for
// timeout control. Services set via Set* or Add* methods are registered
// for shutdown if they implement this interface.
type contextCloser interface {
	// Close releases resources held by the service.
	//
	// Returns error when the close operation fails.
	Close(ctx context.Context) error
}

// contextShutdown provides a way to shut down a service with a context.
type contextShutdown interface {
	// Shutdown stops the service in a controlled way.
	//
	// Returns error when the shutdown fails or the context is cancelled.
	Shutdown(ctx context.Context) error
}

// contextStopper defines a service that can be stopped with a context.
type contextStopper interface {
	// Stop signals the component to stop and release resources.
	//
	// Returns error when the shutdown fails.
	Stop(ctx context.Context) error
}

// validateComponentsDirectory checks that the components directory exists and
// is valid.
//
// Takes absDir (string) which is the absolute path to the components directory.
//
// Returns bool which is true if the directory is valid and discovery should
// proceed.
func validateComponentsDirectory(ctx context.Context, absDir string) bool {
	_, l := logger_domain.From(ctx, log)
	info, err := os.Stat(absDir)
	if err != nil {
		if os.IsNotExist(err) {
			l.Internal("Components directory does not exist, skipping discovery",
				logger_domain.String(logKeyPath, absDir))
			return false
		}
		l.Warn("Failed to stat components directory",
			logger_domain.String(logKeyPath, absDir),
			logger_domain.Error(err))
		return false
	}
	if !info.IsDir() {
		l.Warn("Components path is not a directory",
			logger_domain.String(logKeyPath, absDir))
		return false
	}
	return true
}

// logComponentDiscoveryResults logs the outcome of component discovery.
//
// Takes registered (int) which is the number of components registered.
// Takes regErrors ([]string) which lists any errors that occurred.
// Takes componentsDir (string) which is the directory that was scanned.
func logComponentDiscoveryResults(ctx context.Context, registered int, regErrors []string, componentsDir string) {
	_, l := logger_domain.From(ctx, log)
	if len(regErrors) > 0 {
		l.Warn("Some components failed to register",
			logger_domain.Int("failed_count", len(regErrors)),
			logger_domain.Strings("errors", regErrors))
	}

	if registered > 0 {
		l.Internal("Discovered and registered local components",
			logger_domain.Int("count", registered),
			logger_domain.String("dir", componentsDir))
	}
}

// defaultMetadataCacheProvider returns a no-op cache by default.
// Use WithMemoryRegistryCache to enable caching.
//
// Returns registry_domain.MetadataCache which is nil to disable caching.
func defaultMetadataCacheProvider() registry_domain.MetadataCache {
	return nil
}

// registerCloseableForShutdown registers a service for graceful shutdown if
// it uses a known shutdown interface, so user-provided services are cleaned up
// without the need for manual shutdown setup.
//
// The following shutdown patterns are checked (in order of priority):
//   - Close(context.Context) error
//   - Shutdown(context.Context) error
//   - Stop(context.Context) error
//   - io.Closer (Close() error)
//
// If the service does not use any shutdown interface, this function does
// nothing.
//
// Takes name (string) which identifies the service in shutdown logs.
// Takes service (any) which is the service to check for shutdown support.
func registerCloseableForShutdown(ctx context.Context, name string, service any) {
	if service == nil {
		return
	}

	_, l := logger_domain.From(ctx, log)
	shutdownName := fmt.Sprintf("%s-Override", name)

	if closer, ok := service.(contextCloser); ok {
		shutdown.Register(ctx, shutdownName, func(ctx context.Context) error {
			return closer.Close(ctx)
		})
		l.Internal(logMessageAutoRegisteredShutdown,
			logger_domain.String(logKeyService, name),
			logger_domain.String(logKeyMethod, "Close(ctx)"))
		return
	}

	if shutdowner, ok := service.(contextShutdown); ok {
		shutdown.Register(ctx, shutdownName, func(ctx context.Context) error {
			return shutdowner.Shutdown(ctx)
		})
		l.Internal(logMessageAutoRegisteredShutdown,
			logger_domain.String(logKeyService, name),
			logger_domain.String(logKeyMethod, "Shutdown(ctx)"))
		return
	}

	if stopper, ok := service.(contextStopper); ok {
		shutdown.Register(ctx, shutdownName, func(ctx context.Context) error {
			return stopper.Stop(ctx)
		})
		l.Internal(logMessageAutoRegisteredShutdown,
			logger_domain.String(logKeyService, name),
			logger_domain.String(logKeyMethod, "Stop(ctx)"))
		return
	}

	if closer, ok := service.(io.Closer); ok {
		shutdown.Register(ctx, shutdownName, func(_ context.Context) error {
			return closer.Close()
		})
		l.Internal(logMessageAutoRegisteredShutdown,
			logger_domain.String(logKeyService, name),
			logger_domain.String(logKeyMethod, "Close()"))
		return
	}

	l.Trace("Service does not implement shutdown interface",
		logger_domain.String("service", name))
}
