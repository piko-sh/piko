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

package piko

import (
	"time"

	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/capabilities"
	"piko.sh/piko/internal/captcha/captcha_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/crypto/crypto_domain"
	"piko.sh/piko/internal/daemon/daemon_frontend"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/events/events_domain"
	"piko.sh/piko/internal/highlight/highlight_domain"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/image/image_domain"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/logger/logger_dto"
	"piko.sh/piko/internal/markdown/markdown_domain"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/internal/notification/notification_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/pml/pml_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
	"piko.sh/piko/internal/video/video_domain"
)

const (
	// WatchdogPriorityNormal is informational; safe to ignore in alerting.
	WatchdogPriorityNormal = monitoring_domain.WatchdogPriorityNormal

	// WatchdogPriorityHigh warrants prompt investigation.
	WatchdogPriorityHigh = monitoring_domain.WatchdogPriorityHigh

	// WatchdogPriorityCritical indicates imminent system instability.
	WatchdogPriorityCritical = monitoring_domain.WatchdogPriorityCritical

	// ModuleAnalytics provides Google Analytics (GA4) support through hooks.
	// It tracks page views, navigation, server actions, modal opens, and errors.
	ModuleAnalytics = daemon_frontend.ModuleAnalytics

	// ModuleModals provides helper functions for modal dialogues. It exports
	// showModal, closeModal, updateModal, and reloadPartial.
	ModuleModals = daemon_frontend.ModuleModals

	// ModuleToasts provides toast notification helpers. It exports showToast which
	// takes a message, variant, and duration.
	ModuleToasts = daemon_frontend.ModuleToasts

	// defaultRegistryCacheMaxWeightMB is the default maximum cache size in
	// megabytes.
	defaultRegistryCacheMaxWeightMB = 256

	// defaultRegistryCacheTTLMinutes is the default TTL for cache entries in
	// minutes.
	defaultRegistryCacheTTLMinutes = 30

	// bytesPerMB is the number of bytes in a megabyte.
	bytesPerMB = 1024 * 1024

	// CORPSameOrigin restricts resources to same-origin requests only.
	// This is the default and most secure option.
	CORPSameOrigin = "same-origin"

	// CORPSameSite allows resources to be loaded by same-site requests.
	CORPSameSite = "same-site"

	// CORPCrossOrigin allows resources to be loaded by any origin.
	// Use this for headless CMS scenarios where resources are served to
	// frontends on different origins.
	CORPCrossOrigin = "cross-origin"
)

// CSSResetOption is a functional option for configuring the CSS reset feature.
// It is used as a sub-option of WithCSSReset.
type CSSResetOption = bootstrap.CSSResetOption

// FrontendModule represents a built-in Piko frontend module.
// These are optional JavaScript bundles that add features like analytics,
// modals, or toasts to the core framework.
type FrontendModule = daemon_frontend.FrontendModule

// AnalyticsConfig is an alias for the frontend analytics settings.
// It configures Google Analytics (GA4) integration.
type AnalyticsConfig = daemon_frontend.AnalyticsConfig

// ModalsConfig provides settings for the Modals module.
type ModalsConfig = daemon_frontend.ModalsConfig

// ToastsConfig provides settings for the Toasts module.
type ToastsConfig = daemon_frontend.ToastsConfig

// MonitoringOption sets up the gRPC monitoring service.
type MonitoringOption = bootstrap.MonitoringOption

// MetricsExporter is the interface for metrics exporters.
// Implementations provide OTEL metric reader integration and HTTP handlers.
type MetricsExporter = monitoring_domain.MetricsExporter

// MonitoringTLSOption configures TLS settings for the monitoring gRPC server.
type MonitoringTLSOption = bootstrap.MonitoringTLSOption

// ProfilingOption configures the pprof HTTP debug server.
type ProfilingOption = bootstrap.ProfilingOption

// GeneratorProfilingOption configures profiling for short-lived generator
// builds that capture profiles to disk.
type GeneratorProfilingOption = bootstrap.GeneratorProfilingOption

// RegistryMetadataCacheConfig configures the metadata cache for the Registry
// service. This cache stores artefact metadata to reduce database queries and
// improve performance.
type RegistryMetadataCacheConfig struct {
	// MaxWeight is the maximum total weight (in bytes) the cache may hold,
	// calculated by a custom weigher that estimates the memory footprint of
	// each artefact metadata entry and evicting the least recently used
	// entries when the limit is reached.
	// Default: 256 MB (256 * 1024 * 1024).
	MaxWeight uint64

	// TTL is the time-to-live for cached entries, using access-based
	// expiration that resets on every read to implement a sliding window
	// policy ideal for keeping frequently accessed artefacts in cache.
	// Default: 30 minutes.
	TTL time.Duration

	// StatsEnabled enables collection of cache performance statistics,
	// reserved for future functionality and currently having no effect
	// since statistics are always available via the cache's Stats() method.
	// Default: false.
	StatsEnabled bool
}

// CSPBuilder is a helper for creating Content-Security-Policy headers.
// See WithCSP for usage examples.
type CSPBuilder = security_domain.CSPBuilder

var (
	// CSPSelf allows resources from the same origin (scheme, host, and port).
	CSPSelf = security_domain.Self

	// CSPNone disallows all resources for the directive.
	CSPNone = security_domain.None

	// CSPUnsafeInline allows inline scripts and styles.
	// Warning: This significantly reduces CSP protection against XSS.
	CSPUnsafeInline = security_domain.UnsafeInline

	// CSPUnsafeEval allows use of eval() and similar dynamic code execution.
	// Warning: This significantly reduces CSP protection against XSS.
	CSPUnsafeEval = security_domain.UnsafeEval

	// CSPUnsafeHashes allows specific inline event handlers based on their hash.
	CSPUnsafeHashes = security_domain.UnsafeHashes

	// CSPStrictDynamic allows scripts loaded by trusted scripts to execute.
	// When present, 'self' and URL-based allowlists are ignored for script-src.
	CSPStrictDynamic = security_domain.StrictDynamic

	// CSPReportSample tells the browser to include a sample of the code that
	// broke the rules in violation reports.
	CSPReportSample = security_domain.ReportSample

	// CSPWasmUnsafeEval permits WebAssembly code to run.
	CSPWasmUnsafeEval = security_domain.WasmUnsafeEval

	// CSPData allows data: URIs (e.g., inline images, fonts).
	CSPData = security_domain.Data

	// CSPBlob allows blob: URIs such as URL.createObjectURL results.
	CSPBlob = security_domain.Blob

	// CSPHTTPS allows any resource loaded over HTTPS.
	CSPHTTPS = security_domain.HTTPS

	// CSPHTTP allows any resource loaded over HTTP.
	// Warning: Using HTTP reduces security; prefer HTTPS.
	CSPHTTP = security_domain.HTTP

	// CSPRequestToken is a placeholder for dynamic per-request tokens.
	//
	// When used, the security middleware generates a unique cryptographic token
	// for each request and replaces this placeholder in the CSP header. Use
	// CSPTokenAttr in templates to add the token attribute to inline scripts
	// and styles.
	CSPRequestToken = security_domain.RequestTokenPlaceholder

	// CSPHost creates a source from a host specification such as
	// "cdn.example.com", "*.example.com", or "https://cdn.example.com".
	CSPHost = security_domain.Host

	// CSPScheme creates a source from a URL scheme.
	// The scheme should include the trailing colon (e.g., "wss:").
	CSPScheme = security_domain.Scheme

	// CSPSHA256 creates a source from a base64-encoded SHA-256 hash. Use this for
	// allowing specific inline scripts or styles by their content hash.
	CSPSHA256 = security_domain.SHA256

	// CSPSHA384 creates a source from a base64-encoded SHA-384 hash.
	CSPSHA384 = security_domain.SHA384

	// CSPSHA512 creates a source from a base64-encoded SHA-512 hash.
	CSPSHA512 = security_domain.SHA512

	// CSPStaticToken creates a source from a specific token value.
	// For dynamic per-request tokens, use CSPRequestToken instead.
	CSPStaticToken = security_domain.RequestToken

	// CSPPolicyName creates a Trusted Types policy name source.
	// Policy names are unquoted in the CSP header output.
	//
	// Example:
	// builder.TrustedTypes(
	//     piko.CSPPolicyName("default"),
	//     piko.CSPPolicyName("dompurify"),
	// )
	// // outputs: trusted-types default dompurify
	CSPPolicyName = security_domain.PolicyName

	// CSPScript is the 'script' keyword for require-trusted-types-for directive.
	// This is used internally; prefer using RequireTrustedTypesFor() method.
	CSPScript = security_domain.Script

	// CSPAllowDuplicates is the 'allow-duplicates' keyword for the trusted-types
	// directive. This enables creating multiple policies with the same name.
	CSPAllowDuplicates = security_domain.AllowDuplicates

	// CSPWildcard is the wildcard (*) for trusted-types directive.
	// Allows creating policies with any unique names.
	CSPWildcard = security_domain.Wildcard

	// CSPSandboxAllowDownloads enables file downloads.
	CSPSandboxAllowDownloads = security_domain.SandboxAllowDownloads

	// CSPSandboxAllowForms allows form submission in sandboxed content.
	CSPSandboxAllowForms = security_domain.SandboxAllowForms

	// CSPSandboxAllowModals enables opening modal windows.
	CSPSandboxAllowModals = security_domain.SandboxAllowModals

	// CSPSandboxAllowOrientationLock allows the page to lock the screen direction.
	CSPSandboxAllowOrientationLock = security_domain.SandboxAllowOrientationLock

	// CSPSandboxAllowPointerLock enables the Pointer Lock API in sandboxed
	// content.
	CSPSandboxAllowPointerLock = security_domain.SandboxAllowPointerLock

	// CSPSandboxAllowPopups enables window.open() and similar.
	CSPSandboxAllowPopups = security_domain.SandboxAllowPopups

	// CSPSandboxAllowPopupsToEscapeSandbox enables popups to open unsandboxed
	// windows.
	CSPSandboxAllowPopupsToEscapeSandbox = security_domain.SandboxAllowPopupsToEscapeSandbox

	// CSPSandboxAllowPresentation enables the Presentation API in sandboxed
	// content.
	CSPSandboxAllowPresentation = security_domain.SandboxAllowPresentation

	// CSPSandboxAllowSameOrigin allows sandboxed content to be treated as being
	// from the same origin.
	CSPSandboxAllowSameOrigin = security_domain.SandboxAllowSameOrigin

	// CSPSandboxAllowScripts allows JavaScript to run in a sandboxed iframe.
	CSPSandboxAllowScripts = security_domain.SandboxAllowScripts

	// CSPSandboxAllowStorageAccessByUserActivation enables the Storage Access API
	// with user gesture.
	CSPSandboxAllowStorageAccessByUserActivation = security_domain.SandboxAllowStorageAccessByUserActivation

	// CSPSandboxAllowTopNavigation allows the page to move the top-level window
	// to a new address.
	CSPSandboxAllowTopNavigation = security_domain.SandboxAllowTopNavigation

	// CSPSandboxAllowTopNavigationByUserActivation enables top navigation with
	// user gesture.
	CSPSandboxAllowTopNavigationByUserActivation = security_domain.SandboxAllowTopNavigationByUserActivation

	// CSPSandboxAllowTopNavigationToCustomProtocols enables top navigation to
	// custom protocols.
	CSPSandboxAllowTopNavigationToCustomProtocols = security_domain.SandboxAllowTopNavigationToCustomProtocols
)

// ReportingEndpoint defines a single reporting endpoint for the
// Reporting-Endpoints header. These endpoints can be referenced by CSP
// report-to directives and other reporting APIs.
type ReportingEndpoint = config.ReportingEndpoint

// TLSOption configures TLS settings for the main server. Use with WithTLS.
type TLSOption = bootstrap.TLSOption

// HealthTLSOption configures TLS settings for the health probe server. Use
// with WithHealthTLS.
type HealthTLSOption = bootstrap.HealthTLSOption

// WithEventBus provides a custom EventBus implementation.
//
// Takes bus (orchestrator_domain.EventBus) which specifies the event bus to
// use.
//
// Returns Option which configures the application to use the provided event
// bus.
func WithEventBus(bus orchestrator_domain.EventBus) Option {
	return bootstrap.WithEventBus(bus)
}

// WithRegistryService provides a custom RegistryService implementation.
//
// Takes service (RegistryService) which provides the registry service to use.
//
// Returns Option which configures the application with the given registry.
func WithRegistryService(service registry_domain.RegistryService) Option {
	return bootstrap.WithRegistryService(service)
}

// WithCapabilityService provides a custom CapabilityService implementation.
//
// Takes service (capabilities.Service) which is the custom capability service to
// use.
//
// Returns Option which configures the bootstrap with the given service.
func WithCapabilityService(service capabilities.Service) Option {
	return bootstrap.WithCapabilityService(service)
}

// WithOrchestratorService provides a custom OrchestratorService implementation.
//
// Takes service (OrchestratorService) which is the service to use for
// orchestration.
//
// Returns Option which configures the application with the given service.
func WithOrchestratorService(service orchestrator_domain.OrchestratorService) Option {
	return bootstrap.WithOrchestratorService(service)
}

// WithI18nService provides a custom I18nService implementation.
//
// Takes service (i18n_domain.Service) which is the internationalisation service to
// use.
//
// Returns Option which configures the application to use the provided service.
func WithI18nService(service i18n_domain.Service) Option {
	return bootstrap.WithI18nService(service)
}

// WithMemoryRegistryCache configures the default RegistryService to be built
// with an in-memory cache.
//
// Returns Option which applies the memory cache configuration.
func WithMemoryRegistryCache() Option {
	return bootstrap.WithMemoryRegistryCache()
}

// WithJSONTypeInspectorCache configures the TypeInspectorManager to use a
// JSON file-based cache.
//
// Returns Option which applies the JSON cache configuration.
func WithJSONTypeInspectorCache() Option {
	return bootstrap.WithJSONTypeInspectorCache()
}

// WithCSRFSecret configures the CSRF service with a specific secret key.
//
// Takes key ([]byte) which is the secret key for CSRF token generation.
//
// Returns Option which applies the CSRF secret configuration.
func WithCSRFSecret(key []byte) Option {
	return bootstrap.WithCSRFSecret(key)
}

// WithConfigResolvers allows for injecting custom configuration resolvers.
// This is the primary mechanism for integrating with secret managers like
// AWS Secrets Manager, GCP Secret Manager, or HashiCorp Vault.
//
// Takes resolvers (...ConfigResolver) which provide custom resolution logic.
//
// Returns Option which configures the application with the given resolvers.
func WithConfigResolvers(resolvers ...ConfigResolver) Option {
	return bootstrap.WithConfigResolvers(resolvers...)
}

// WithShutdownDrainDelay sets the duration to wait after marking the instance
// as not ready (readiness returns 503) before shutting down the HTTP server.
// This gives load balancers time to deregister the instance during rolling
// deploys.
//
// Default: 0s in dev mode, 3s in production.
//
// Takes delay (time.Duration) which specifies how long to wait.
//
// Returns Option which configures the shutdown drain delay.
func WithShutdownDrainDelay(delay time.Duration) Option {
	return bootstrap.WithShutdownDrainDelay(delay)
}

// WithCacheProvider registers a named cache provider instance with the default
// cache service. If multiple providers are registered, use
// WithDefaultCacheProvider to specify which one is the default.
//
// Cache providers implement the Provider interface, which creates namespaced
// Cache[K, V] instances on demand, so a single provider (e.g., a Redis
// connection pool) can serve multiple typed caches.
//
// Takes name (string) which identifies this provider for later reference.
// Takes provider (cache_domain.Provider) which creates cache instances.
//
// Returns Option which configures the container with the cache provider.
//
// Example:
// import (
//
//	"piko.sh/piko"
//	cache_provider_otter "piko.sh/piko/internal/cache/cache_adapters/provider_otter"
//
// )
// otterProvider := cache_provider_otter.NewOtterProvider()
// app := piko.New(
//
//	piko.WithCacheProvider("otter", otterProvider),
//	piko.WithDefaultCacheProvider("otter"),
//
// )
func WithCacheProvider(name string, provider cache_domain.Provider) Option {
	return func(c *bootstrap.Container) {
		c.AddCacheProvider(name, provider)
	}
}

// WithDefaultCacheProvider sets the name of the provider to use for default
// cache creation. A provider with this name must be registered via
// WithCacheProvider.
//
// Takes name (string) which specifies the provider name to use as default.
//
// Returns Option which configures the container with the default cache
// provider.
//
// Example:
// app := piko.New(
//
//	piko.WithCacheProvider("redis", redisProvider),
//	piko.WithCacheProvider("otter", otterProvider),
//	piko.WithDefaultCacheProvider("redis"),
//
// )
func WithDefaultCacheProvider(name string) Option {
	return func(c *bootstrap.Container) {
		c.SetCacheDefaultProvider(name)
	}
}

// WithCacheService sets a fully configured cache service instance. This
// overrides the default cache service creation and provider registration.
//
// Takes service (cache_domain.Service) which is the configured cache service.
//
// Returns Option which applies the cache service to the container.
//
// Example:
// import (
//
//	"piko.sh/piko/wdk/cache"
//	"piko.sh/piko/wdk/cache/cache_provider_otter"
//
// )
// // Create a custom cache service
// cacheService := cache.NewService("otter")
// otterFactory := cache_provider_otter.NewOtterProviderFactory[string, []byte]()
// cacheService.RegisterProvider("otter", otterFactory)
// // Use it when initialising Piko
// server, err := piko.New(
//
//	piko.WithCacheService(cacheService),
//
// )
func WithCacheService(service cache_domain.Service) Option {
	return func(c *bootstrap.Container) {
		c.SetCacheService(service)
	}
}

// WithCSSTreeShaking enables CSS tree-shaking during scaffold generation.
//
// When enabled, unused CSS rules (based on static HTML analysis) are removed
// from the final output, reducing CSS bundle size.
//
// Returns Option which configures the server to enable CSS tree-shaking.
//
// WARNING: Tree-shaking removes CSS for classes that are not present in
// static HTML. Classes added dynamically via JavaScript (e.g., via
// classList.add()) will be removed unless added to the safelist using
// WithCSSTreeShakingSafelist.
//
// By default, CSS tree-shaking is DISABLED to prevent accidentally removing
// styles needed for dynamic functionality.
//
// Example:
// server := piko.New(
//
//	piko.WithCSSTreeShaking(),
//	piko.WithCSSTreeShakingSafelist("open", "active", "hidden", "visible"),
//
// )
func WithCSSTreeShaking() Option {
	return Option(bootstrap.WithCSSTreeShaking(true))
}

// WithCSSTreeShakingSafelist specifies CSS class names that should never be
// removed by tree-shaking, even when CSSTreeShaking is enabled.
//
// Use this for classes that are added dynamically via JavaScript, such as
// modal open states, visibility toggles, or animation classes.
//
// Class names should be specified without the leading dot.
//
// Example:
// server := piko.New(
//
//	piko.WithCSSTreeShaking(),
//	piko.WithCSSTreeShakingSafelist("open", "active", "hidden", "visible"),
//
// )
// This preserves CSS rules for .open, .active, .hidden, and .visible even if
// those classes don't appear in the static HTML.
//
// Takes classes (...string) which lists the CSS class names to preserve.
//
// Returns Option which configures the safelist on the server.
func WithCSSTreeShakingSafelist(classes ...string) Option {
	return Option(bootstrap.WithCSSTreeShakingSafelist(classes))
}

// WithCSSReset enables the CSS reset for PK files (pages, partials, emails).
// Without sub-options, the simple default reset is used which zeroes margins
// and padding and sets border-box sizing on all elements.
//
// Use WithCSSResetComplete to switch to the comprehensive legacy reset that
// includes element-level resets, typography defaults, heading sizes via theme
// variables, and focus-ring styles.
//
// Use WithCSSResetPKOverride to provide entirely custom CSS reset content.
//
// When WithCSSReset is not called, no CSS reset is included in the generated
// theme CSS.
//
// Takes opts (...CSSResetOption) which provides optional settings:
//   - WithCSSResetComplete(): selects the comprehensive legacy reset.
//   - WithCSSResetPKOverride(css): overrides with custom CSS content.
//
// Returns Option which configures the server's CSS reset behaviour.
//
// Example:
//
//	server := piko.New(
//	    piko.WithCSSReset(),
//	)
//
//	server := piko.New(
//	    piko.WithCSSReset(piko.WithCSSResetComplete()),
//	)
//
//	server := piko.New(
//	    piko.WithCSSReset(piko.WithCSSResetPKOverride(myCustomCSS)),
//	)
func WithCSSReset(opts ...CSSResetOption) Option {
	return Option(bootstrap.WithCSSReset(opts...))
}

// WithCSSResetComplete selects the comprehensive legacy CSS reset instead of
// the simple default. The comprehensive reset includes element-level resets,
// typography defaults, heading sizes via theme variables, and focus-ring
// styles.
//
// Returns CSSResetOption which switches to the complete reset preset.
func WithCSSResetComplete() CSSResetOption {
	return bootstrap.WithCSSResetComplete()
}

// WithCSSResetPKOverride replaces the default CSS reset with custom CSS
// content for PK files (pages, partials, emails).
//
// Takes css (string) which is the custom CSS reset content.
//
// Returns CSSResetOption which sets the custom CSS override.
func WithCSSResetPKOverride(css string) CSSResetOption {
	return bootstrap.WithCSSResetPKOverride(css)
}

// WithCryptoService provides a fully configured crypto service instance.
//
// Use it in advanced scenarios where you need complete control over service
// construction.
//
// For most use cases, prefer using WithCryptoProvider and
// WithDefaultCryptoProvider instead.
//
// Takes service (crypto_domain.CryptoServicePort) which is the custom crypto
// service to use.
//
// Returns Option which configures the server to use the provided service.
//
// Example:
// customService := crypto.NewService(myProvider, myConfig)
// server := piko.New(
//
//	piko.WithCryptoService(customService),
//
// )
func WithCryptoService(service crypto_domain.CryptoServicePort) Option {
	return bootstrap.WithCryptoService(service)
}

// WithCryptoProvider registers a named encryption provider instance with the
// default crypto service.
//
// Multiple providers can be registered (e.g., one for development, one for
// production). Use WithDefaultCryptoProvider to specify which provider is
// active.
//
// Takes name (string) which identifies the provider for later reference.
// Takes provider (crypto_domain.EncryptionProvider) which handles encryption
// operations.
//
// Returns Option which configures the container with the named provider.
//
// Example:
// import (
//
//	"piko.sh/piko"
//	"piko.sh/piko/wdk/crypto/crypto_provider_aws_kms"
//	"piko.sh/piko/wdk/crypto/crypto_provider_local_aes_gcm"
//
// )
// // Register multiple providers
// localProvider, _ := crypto_provider_local_aes_gcm.NewProvider(ctx, localConfig)
// awsProvider, _ := crypto_provider_aws_kms.NewAWSKMSProvider(ctx, awsConfig)
// server := piko.New(
//
//	piko.WithCryptoProvider("local", localProvider),
//	piko.WithCryptoProvider("aws_kms", awsProvider),
//	piko.WithDefaultCryptoProvider("aws_kms"), // Use AWS KMS as default
//
// )
func WithCryptoProvider(name string, provider crypto_domain.EncryptionProvider) Option {
	return func(c *bootstrap.Container) {
		c.AddCryptoProvider(name, provider)
	}
}

// WithDefaultCryptoProvider sets the name of the provider to use for
// encryption operations. A provider with this name must be registered via
// WithCryptoProvider.
//
// Use it to switch between providers (e.g., local for dev, KMS for production)
// without changing application code.
//
// Takes name (string) which specifies the registered provider to use.
//
// Returns Option which configures the default crypto provider.
//
// Example:
// server := piko.New(
//
//	piko.WithCryptoProvider("local", localProvider),
//	piko.WithCryptoProvider("aws_kms", awsProvider),
//	piko.WithDefaultCryptoProvider("aws_kms"), // Active provider
//
// )
func WithDefaultCryptoProvider(name string) Option {
	return func(c *bootstrap.Container) {
		c.SetCryptoDefaultProvider(name)
	}
}

// WithCaptchaProvider registers a named captcha provider instance with the
// default captcha service.
//
// Multiple providers can be registered. Use WithDefaultCaptchaProvider to
// specify which provider is active.
//
// Takes name (string) which identifies the provider for later reference.
// Takes provider (captcha_domain.CaptchaProvider) which handles captcha
// verification.
//
// Returns Option which configures the container with the captcha provider.
//
// Example:
//
//	import "piko.sh/piko/wdk/captcha/captcha_provider_turnstile"
//
//	provider, _ := captcha_provider_turnstile.NewProvider(config)
//	server := piko.New(
//	    piko.WithCaptchaProvider("turnstile", provider),
//	    piko.WithDefaultCaptchaProvider("turnstile"),
//	)
func WithCaptchaProvider(name string, provider captcha_domain.CaptchaProvider) Option {
	return func(c *bootstrap.Container) {
		if name == "" || provider == nil {
			return
		}
		c.AddCaptchaProvider(name, provider)
	}
}

// WithDefaultCaptchaProvider sets the name of the provider to use for captcha
// verification. A provider with this name must be registered via
// WithCaptchaProvider.
//
// Takes name (string) which specifies the provider name to use as default.
//
// Returns Option which configures the container with the default captcha
// provider.
func WithDefaultCaptchaProvider(name string) Option {
	return func(c *bootstrap.Container) {
		c.SetCaptchaDefaultProvider(name)
	}
}

// WithEmailService returns an option that configures the email service.
//
// Takes service (email_domain.Service) which provides email functionality.
//
// Returns Option which applies the email service configuration.
func WithEmailService(service email_domain.Service) Option {
	return bootstrap.WithEmailService(service)
}

// WithEmailProvider registers a named email provider instance with the
// default email service. If multiple providers are registered, use
// WithDefaultEmailProvider to specify which one is the default.
//
// Takes name (string) which identifies this provider for later reference.
// Takes provider (email_domain.EmailProviderPort) which handles email sending.
//
// Returns Option which configures the container with the email provider.
func WithEmailProvider(name string, provider email_domain.EmailProviderPort) Option {
	return func(c *bootstrap.Container) {
		c.AddEmailProvider(name, provider)
	}
}

// WithDefaultEmailProvider sets the name of the provider to use for default
// email sending. A provider with this name must be registered via
// WithEmailProvider.
//
// Takes name (string) which specifies the provider name to use as default.
//
// Returns Option which configures the container with the default email
// provider.
func WithDefaultEmailProvider(name string) Option {
	return func(c *bootstrap.Container) {
		c.SetEmailDefaultProvider(name)
	}
}

// WithEmailDispatcher enables and configures the background dispatcher for the
// default email service.
//
// Takes dispatcherConfig (email_dto.DispatcherConfig) which controls batching,
// retries, and queue sizes.
//
// Returns Option which applies the dispatcher configuration to the container.
func WithEmailDispatcher(dispatcherConfig email_dto.DispatcherConfig) Option {
	return func(c *bootstrap.Container) {
		c.SetEmailDispatcherConfig(&dispatcherConfig)
	}
}

// WithEmailDeadLetterQueue provides a custom dead letter queue implementation
// used by the dispatcher. If omitted, an in-memory DLQ will be used by default
// when a dispatcher is enabled.
//
// Takes dlq (email_domain.DeadLetterPort) which handles failed email delivery
// attempts.
//
// Returns Option which configures the container with the provided DLQ.
func WithEmailDeadLetterQueue(dlq email_domain.DeadLetterPort) Option {
	return func(c *bootstrap.Container) {
		c.SetEmailDeadLetterAdapter(dlq)
	}
}

// WithEventsProvider sets a custom events provider, overriding the default
// GoChannel provider.
//
// Use this to configure NATS JetStream, PostgreSQL, SQLite, or a custom
// provider.
//
// Example with NATS JetStream:
// import "piko.sh/piko/wdk/events/events_provider_nats"
// provider, _ := events_provider_nats.NewNATSProvider(ctx,
//
//	events_provider_nats.Config{URL: "nats://localhost:4222"},
//
// )
// provider.Start(ctx)
// app := piko.New(
//
//	piko.WithEventsProvider(provider),
//
// )
// Example with custom GoChannel configuration:
// import "piko.sh/piko/wdk/events/events_provider_gochannel"
// channelConfig := events_provider_gochannel.DefaultConfig()
// channelConfig.OutputChannelBuffer = 2048
// provider, _ := events_provider_gochannel.NewGoChannelProvider(channelConfig)
// provider.Start(ctx)
// app := piko.New(
//
//	piko.WithEventsProvider(provider),
//
// )
// The provider must be started (via Start()) before passing it to
// WithEventsProvider. If you pass an unstarted provider, the container will
// not start it automatically.
//
// Takes provider (events_domain.Provider) which is the events provider to use.
//
// Returns Option which configures the container's events provider.
func WithEventsProvider(provider events_domain.Provider) Option {
	return func(c *bootstrap.Container) {
		c.SetEventsProvider(provider)
	}
}

// WithFrontendModule enables a built-in frontend module to be loaded
// site-wide. The module's JavaScript will be automatically preloaded and
// executed on every page.
//
// The optional config parameter allows passing configuration to the module.
// Use the appropriate config type for each module:
//   - AnalyticsConfig for ModuleAnalytics
//   - ModalsConfig for ModuleModals
//   - ToastsConfig for ModuleToasts
//
// Example without config:
// server := piko.New(
//
//	piko.WithFrontendModule(piko.ModuleModals),
//
// )
// Example with config:
// server := piko.New(
//
//	piko.WithFrontendModule(piko.ModuleAnalytics, piko.AnalyticsConfig{
//	    TrackingIDs: []string{"G-XXXXXXXXXX", "G-YYYYYYYYYY"},
//	    DebugMode:   true,
//	}),
//
// )
//
// Takes module (FrontendModule) which specifies the frontend module to
// enable.
// Takes moduleConfig (...any) which provides optional configuration for the
// module.
//
// Returns Option which configures the server to load the specified module.
func WithFrontendModule(module FrontendModule, moduleConfig ...any) Option {
	return bootstrap.WithFrontendModule(module, moduleConfig...)
}

// WithCustomFrontendModule registers a custom frontend JavaScript module.
// The module will be served at /_piko/dist/ppframework.{name}.min.js and
// automatically included in all pages.
//
// The content should be provided via go:embed at compile time to ensure
// the module is bundled into the binary:
// //go:embed static/js/tracking.js
// var trackingJS []byte
// server := piko.New(
//
//	piko.WithCustomFrontendModule("tracking", trackingJS),
//
// )
// The optional config parameter allows passing configuration to the module:
// server := piko.New(
//
//	piko.WithCustomFrontendModule("tracking", trackingJS, map[string]any{
//	    "endpoint": "https://analytics.example.com",
//	    "debug":    true,
//	}),
//
// )
// The module can access its config via PPFramework.getModuleConfig("tracking").
//
// Takes name (string) which identifies the module in the URL and config lookup.
// Takes content ([]byte) which contains the JavaScript module source code.
// Takes moduleConfig (...map[string]any) which provides optional key-value
// settings accessible to the module at runtime.
//
// Returns Option which configures the server to serve this custom module.
func WithCustomFrontendModule(name string, content []byte, moduleConfig ...map[string]any) Option {
	return bootstrap.WithCustomFrontendModule(name, content, moduleConfig...)
}

// WithHighlighter configures syntax highlighting for markdown code blocks.
//
// When a highlighter is configured, code blocks in markdown content will be
// syntax-highlighted using the provided implementation. If no highlighter is
// configured, code blocks render as plain <pre><code> elements.
//
// Example with Chroma (recommended):
// import (
//
//	"piko.sh/piko"
//	"piko.sh/piko/wdk/highlight/highlight_chroma"
//
// )
//
//	highlighter := highlight_chroma.NewChromaHighlighter(highlight_chroma.Config{
//	    Style:       "dracula",
//	    WithClasses: true,
//	})
//
// server := piko.New(
//
//	piko.WithHighlighter(highlighter),
//
// )
// When using WithClasses: true, you must include the appropriate CSS
// styles in your page for the highlighting to be visible.
//
// Takes h (highlight_domain.Highlighter) which provides the syntax
// highlighting implementation.
//
// Returns Option which configures the server to use the given highlighter.
func WithHighlighter(h highlight_domain.Highlighter) Option {
	return func(c *bootstrap.Container) {
		c.SetHighlighter(h)
	}
}

// WithLLMProvider registers a named LLM provider instance with the default LLM
// service. If multiple providers are registered, use WithDefaultLLMProvider to
// specify which one is the default.
//
// LLM providers handle completion and streaming requests to language models.
// Multiple providers enable fallback scenarios and cost optimisation
// strategies.
//
// Takes name (string) which identifies this provider for later reference.
// Takes provider (llm_domain.LLMProviderPort) which handles LLM requests.
//
// Returns Option which configures the container with the LLM provider.
//
// Example:
// import (
//
//	"piko.sh/piko"
//	"piko.sh/piko/wdk/llm/llm_provider_anthropic"
//	"piko.sh/piko/wdk/llm/llm_provider_openai"
//
// )
// // Create providers
// anthropicProvider := llm_provider_anthropic.NewProvider(anthropicConfig)
// openaiProvider := llm_provider_openai.NewProvider(openaiConfig)
// // Register multiple providers with fallback capability
// app := piko.New(
//
//	piko.WithLLMProvider("anthropic", anthropicProvider),
//	piko.WithLLMProvider("openai", openaiProvider),
//	piko.WithDefaultLLMProvider("anthropic"),
//
// )
// // Later, use specific provider or fallback
// llmService, _ := llm.GetDefaultService()
// response, err := llmService.CompleteWithProvider(ctx, "openai", request)
func WithLLMProvider(name string, provider llm_domain.LLMProviderPort) Option {
	return func(c *bootstrap.Container) {
		c.AddLLMProvider(name, provider)
	}
}

// WithDefaultLLMProvider sets the name of the provider to use for default LLM
// operations. A provider with this name must be registered via WithLLMProvider.
//
// Takes name (string) which specifies the provider name to use as default.
//
// Returns Option which configures the container with the default LLM provider.
//
// Example:
// app := piko.New(
//
//	piko.WithLLMProvider("anthropic", anthropicProvider),
//	piko.WithLLMProvider("openai", openaiProvider),
//	piko.WithDefaultLLMProvider("anthropic"),
//
// )
// // Default provider is used when no provider is specified
// llmService, _ := llm.GetDefaultService()
// response, err := llmService.Complete(ctx, request) // Uses "anthropic"
func WithDefaultLLMProvider(name string) Option {
	return func(c *bootstrap.Container) {
		c.SetLLMDefaultProvider(name)
	}
}

// WithEmbeddingProvider registers a standalone embedding provider for
// embedding-only services such as Voyage AI.
//
// Unlike WithLLMProvider, this does not register a completion provider.
// When using an LLM provider that also supports embeddings (e.g. OpenAI,
// Ollama), embedding support is auto-detected and this option is not needed.
//
// Takes name (string) which identifies this provider.
// Takes provider (llm_domain.EmbeddingProviderPort) which handles embedding
// requests.
//
// Returns Option which configures the container with the embedding provider.
//
// Example:
// app := piko.New(
//
//	piko.WithLLMProvider("anthropic", anthropicProvider),
//	piko.WithDefaultLLMProvider("anthropic"),
//	piko.WithEmbeddingProvider("voyage", voyageProvider),
//	piko.WithDefaultEmbeddingProvider("voyage"),
//
// )
func WithEmbeddingProvider(name string, provider llm_domain.EmbeddingProviderPort) Option {
	return func(c *bootstrap.Container) {
		c.AddEmbeddingProvider(name, provider)
	}
}

// WithDefaultEmbeddingProvider sets the name of the default embedding provider.
// When set, this takes precedence over any auto-detected embedding support
// from the default LLM provider.
//
// A provider with this name must be registered via WithEmbeddingProvider or
// be an LLM provider that implements EmbeddingProviderPort.
//
// Takes name (string) which specifies the provider name to use as default.
//
// Returns Option which configures the container with the default embedding
// provider.
func WithDefaultEmbeddingProvider(name string) Option {
	return func(c *bootstrap.Container) {
		c.SetDefaultEmbeddingProvider(name)
	}
}

// WithLLMService sets a fully configured LLM service instance. This overrides
// the default LLM service creation and provider registration.
//
// Takes service (llm_domain.Service) which is the configured LLM service.
//
// Returns Option which applies the LLM service to the container.
//
// Example:
// import (
//
//	"piko.sh/piko"
//	"piko.sh/piko/wdk/llm"
//	"piko.sh/piko/wdk/llm/llm_provider_anthropic"
//
// )
// // Create and configure a custom LLM service
// llmService := llm.NewService("anthropic")
// anthropicProvider := llm_provider_anthropic.NewProvider(config)
// llmService.RegisterProvider("anthropic", anthropicProvider)
// llmService.SetDefaultProvider("anthropic")
// // Configure budgets and rate limits
//
//	llmService.SetBudget("user:123", &llm.BudgetConfig{
//	    MaxDailyCost: 10.0,
//	})
//
// llmService.SetRateLimits("global", 100, 100000)
// // Use the custom service
// app := piko.New(
//
//	piko.WithLLMService(llmService),
//
// )
func WithLLMService(service llm_domain.Service) Option {
	return func(c *bootstrap.Container) {
		c.SetLLMService(service)
	}
}

// WithMarkdownParser sets the markdown parser implementation used by the
// collection service for processing markdown content.
//
// When a parser is configured, the collection service will register a markdown
// content provider that uses it to parse .md files. If no parser is set, the
// markdown collection provider is not registered.
//
// Takes parser (markdown_domain.MarkdownParserPort) which provides the
// markdown parsing implementation.
//
// Returns Option which configures the server to use the given parser.
//
// Example usage with the goldmark provider:
//
//	import "piko.sh/piko/wdk/markdown/markdown_provider_goldmark"
//
//	server := piko.New(
//	    piko.WithMarkdownParser(markdown_provider_goldmark.NewParser()),
//	)
func WithMarkdownParser(parser markdown_domain.MarkdownParserPort) Option {
	return bootstrap.WithMarkdownParser(parser)
}

// WithImageService sets a custom image service for the application.
//
// This gives you full control over image transformation, including:
//   - Using different transformer adapters (e.g., vips for high performance)
//   - Configuring custom storage backends for image caching
//   - Setting custom security limits and allowed formats
//
// Most users should use WithImageTransformer instead to register individual
// transformers.
//
// Takes service (image_domain.Service) which provides the image transformation
// implementation.
//
// Returns Option which configures the application to use the given service.
//
// Example with vips transformer:
// import "piko.sh/piko/wdk/media/image_provider_vips"
// vipsTransformer, _ := image_provider_vips.NewProvider(config)
//
//	transformers := map[string]image_domain.TransformerPort{
//	    "vips": vipsTransformer,
//	}
//
// imageService, _ := image_domain.NewService(transformers, "vips", config)
// app := piko.New(piko.WithImageService(imageService))
func WithImageService(service image_domain.Service) Option {
	return bootstrap.WithImageService(service)
}

// WithImageProvider registers a named image provider with the image service.
// The first provider registered becomes the default unless
// WithDefaultImageProvider is called.
//
// If no providers are registered, the image service will be nil and
// components like piko:img will gracefully degrade to basic HTML output
// without responsive image features.
//
// Takes name (string) which identifies this provider.
// Takes provider (image_domain.TransformerPort) which provides image
// transformation capabilities.
//
// Returns Option which configures the container with the provider.
//
// Example usage:
// import imaging "piko.sh/piko/wdk/media/image_provider_imaging"
// imgProvider, _ := imaging.NewProvider(imaging.Config{})
// app := piko.New(
//
//	piko.WithImageProvider("imaging", imgProvider),
//
// )
func WithImageProvider(name string, provider image_domain.TransformerPort) Option {
	return func(c *bootstrap.Container) {
		c.AddImageTransformer(name, provider)
	}
}

// WithDefaultImageProvider sets which registered provider should be used as
// the default. A provider with this name must be registered via
// WithImageProvider.
//
// Takes name (string) which specifies the registered provider name.
//
// Returns Option which configures the default image provider.
func WithDefaultImageProvider(name string) Option {
	return func(c *bootstrap.Container) {
		c.SetDefaultImageTransformer(name)
	}
}

// WithImage configures the image service using an ImageConfig from the builder.
// This is the recommended way to configure image processing with full control.
//
// Takes imageConfig (*image_domain.ImageConfig) which specifies the image
// processing settings including provider, file size limits, and variants.
//
// Returns Option which applies the image configuration to the container.
//
// Example:
// import (
//
//	"piko.sh/piko/wdk/media"
//	"piko.sh/piko/wdk/media/image_provider_vips"
//
// )
// vipsProvider, _ := image_provider_vips.NewProvider(image_provider_vips.Config{})
// app := piko.New(
//
//	piko.WithImage(
//	    media.Image().
//	        Provider("vips", vipsProvider).
//	        MaxFileSizeMB(50).
//	        WithVariant("thumb", media.Variant().Size(200, 200).Cover().Build()).
//	        Build(),
//	),
//
// )
func WithImage(imageConfig *image_domain.ImageConfig) Option {
	return func(c *bootstrap.Container) {
		c.SetImageConfig(imageConfig)
	}
}

// WithVideoService sets a custom video service implementation.
//
// This gives full control over video transcoding, including the ability to:
//   - Use different transcoder adapters (e.g., astiav for FFmpeg-based
//     transcoding)
//   - Configure custom encoding profiles and quality settings
//   - Set custom security limits and allowed codecs
//
// Most users should use WithVideoTranscoder instead to register individual
// transcoders.
//
// Example with astiav transcoder:
// import "piko.sh/piko/wdk/media/video_provider_astiav"
// astiavTranscoder, _ := video_provider_astiav.NewProvider(config)
//
//	transcoders := map[string]video_domain.TranscoderPort{
//	    "astiav": astiavTranscoder,
//	}
//
// videoService, _ := video_domain.NewService(transcoders, "astiav", config)
// app := piko.New(piko.WithVideoService(videoService))
//
// Takes service (video_domain.Service) which provides the video transcoding
// implementation.
//
// Returns Option which configures the container with the video service.
func WithVideoService(service video_domain.Service) Option {
	return func(c *bootstrap.Container) {
		c.SetVideoService(service)
	}
}

// WithVideoProvider registers a named video provider with the video service.
// The first provider registered becomes the default unless
// WithDefaultVideoProvider is called.
//
// If no providers are registered, the video service will be nil and
// components like piko:video will gracefully degrade to basic HTML output
// without transcoding features.
//
// Takes name (string) which identifies the provider for later reference.
// Takes provider (video_domain.TranscoderPort) which provides the video
// transcoding implementation.
//
// Returns Option which configures the container with the video provider.
//
// Example usage:
// import astiav "piko.sh/piko/wdk/media/video_provider_astiav"
// vidProvider, _ := astiav.NewProvider(astiav.Config{})
// app := piko.New(
//
//	piko.WithVideoProvider("astiav", vidProvider),
//
// )
func WithVideoProvider(name string, provider video_domain.TranscoderPort) Option {
	return func(c *bootstrap.Container) {
		c.AddVideoTranscoder(name, provider)
	}
}

// WithDefaultVideoProvider sets which registered provider should be used as
// the default. A provider with this name must be registered via
// WithVideoProvider.
//
// Takes name (string) which is the identifier of the provider to use.
//
// Returns Option which configures the container's default video provider.
func WithDefaultVideoProvider(name string) Option {
	return func(c *bootstrap.Container) {
		c.SetDefaultVideoTranscoder(name)
	}
}

// WithNotificationProvider registers a named notification provider instance
// with the default notification service. If multiple providers are registered,
// use WithDefaultNotificationProvider to specify which one is the default.
//
// Takes name (string) which identifies this provider for later reference.
// Takes provider (notification_domain.NotificationProviderPort) which handles
// notification delivery.
//
// Returns Option which configures the container with the notification provider.
//
// Example:
// import (
//
//	"piko.sh/piko"
//	"piko.sh/piko/wdk/notification/notification_provider_webhook"
//
// )
// webhookProvider := notification_provider_webhook.NewProvider(config)
// app := piko.New(
//
//	piko.WithNotificationProvider("webhook", webhookProvider),
//	piko.WithDefaultNotificationProvider("webhook"),
//
// )
func WithNotificationProvider(name string, provider notification_domain.NotificationProviderPort) Option {
	return func(c *bootstrap.Container) {
		c.AddNotificationProvider(name, provider)
	}
}

// WithDefaultNotificationProvider sets the name of the provider to use for
// default notification sending. A provider with this name must be registered
// via WithNotificationProvider.
//
// Takes name (string) which specifies the provider name to use as default.
//
// Returns Option which configures the container with the default notification
// provider.
//
// Example:
// app := piko.New(
//
//	piko.WithNotificationProvider("webhook", webhookProvider),
//	piko.WithNotificationProvider("email", emailNotifier),
//	piko.WithDefaultNotificationProvider("webhook"),
//
// )
func WithDefaultNotificationProvider(name string) Option {
	return func(c *bootstrap.Container) {
		c.SetNotificationDefaultProvider(name)
	}
}

// WithMonitoringAddress sets the gRPC server listen address for monitoring.
// Default: ":9091".
//
// Takes addr (string) which specifies the address and port to listen on.
//
// Returns MonitoringOption which configures the monitoring server address.
func WithMonitoringAddress(addr string) MonitoringOption {
	return bootstrap.WithMonitoringAddress(addr)
}

// WithMonitoringBindAddress sets the network address for the monitoring server
// to bind to. Default is "127.0.0.1" (localhost only for security).
//
// Takes addr (string) which specifies the network address to bind to.
//
// Returns MonitoringOption which configures the monitoring bind address.
func WithMonitoringBindAddress(addr string) MonitoringOption {
	return bootstrap.WithMonitoringBindAddress(addr)
}

// WithMonitoringAutoNextPort enables automatic port selection for the
// monitoring server. When the configured port is already in use, the server
// tries consecutive ports up to 100 attempts.
//
// Takes enabled (bool) which controls whether auto-port selection is active.
//
// Returns MonitoringOption which configures auto-port selection.
func WithMonitoringAutoNextPort(enabled bool) MonitoringOption {
	return bootstrap.WithMonitoringAutoNextPort(enabled)
}

// WithMonitoringTLS enables TLS for the monitoring gRPC server. Sub-options
// configure certificate paths, mTLS, and hot-reload settings.
//
// Takes opts (...MonitoringTLSOption) which provides optional TLS settings:
//   - WithMonitoringTLSCertFile("/path/to/cert.pem"): sets the certificate.
//   - WithMonitoringTLSKeyFile("/path/to/key.pem"): sets the private key.
//   - WithMonitoringTLSClientCA("/path/to/ca.pem"): enables mTLS.
//   - WithMonitoringTLSClientAuth("require_and_verify"): sets client auth mode.
//   - WithMonitoringTLSMinVersion("1.3"): sets minimum TLS version.
//   - WithMonitoringTLSHotReload(true): enables certificate hot-reload.
//
// Returns MonitoringOption which configures TLS on the monitoring service.
func WithMonitoringTLS(opts ...MonitoringTLSOption) MonitoringOption {
	return bootstrap.WithMonitoringTLS(opts...)
}

// WithMonitoringTLSCertFile sets the certificate file for the monitoring server.
//
// Takes path (string) which specifies the certificate file path.
//
// Returns MonitoringTLSOption which sets the certificate path.
func WithMonitoringTLSCertFile(path string) MonitoringTLSOption {
	return bootstrap.WithMonitoringTLSCertFile(path)
}

// WithMonitoringTLSKeyFile sets the key file for the monitoring server.
//
// Takes path (string) which specifies the key file path.
//
// Returns MonitoringTLSOption which sets the key path.
func WithMonitoringTLSKeyFile(path string) MonitoringTLSOption {
	return bootstrap.WithMonitoringTLSKeyFile(path)
}

// WithMonitoringTLSClientCA sets the client CA file for mTLS on the monitoring
// server.
//
// Takes path (string) which specifies the client CA file path.
//
// Returns MonitoringTLSOption which sets the client CA path.
func WithMonitoringTLSClientCA(path string) MonitoringTLSOption {
	return bootstrap.WithMonitoringTLSClientCA(path)
}

// WithMonitoringTLSClientAuth sets the client certificate verification mode for
// the monitoring server.
//
// Takes authType (string) which specifies the auth mode.
//
// Returns MonitoringTLSOption which sets the client auth type.
func WithMonitoringTLSClientAuth(authType string) MonitoringTLSOption {
	return bootstrap.WithMonitoringTLSClientAuth(authType)
}

// WithMonitoringTLSMinVersion sets the minimum TLS version for the monitoring
// server.
//
// Takes version (string) which specifies the minimum version ("1.2" or "1.3").
//
// Returns MonitoringTLSOption which sets the minimum TLS version.
func WithMonitoringTLSMinVersion(version string) MonitoringTLSOption {
	return bootstrap.WithMonitoringTLSMinVersion(version)
}

// WithMonitoringTLSHotReload enables or disables automatic certificate reload
// for the monitoring server.
//
// Takes enabled (bool) which controls hot-reload behaviour.
//
// Returns MonitoringTLSOption which sets the hot-reload flag.
func WithMonitoringTLSHotReload(enabled bool) MonitoringTLSOption {
	return bootstrap.WithMonitoringTLSHotReload(enabled)
}

// WithMonitoring enables the gRPC monitoring server for TUI integration.
// The monitoring server exposes telemetry data (metrics, traces, system stats)
// via gRPC on port 9091 by default.
//
// When enabled, the following gRPC services are available:
//   - HealthService - Basic health checks
//   - MetricsService - OTEL metrics, traces, system stats, file descriptors
//   - OrchestratorInspectorService - Task and workflow status (if orchestrator
//     enabled)
//   - RegistryInspectorService - Artefact status (if registry enabled)
//
// The monitoring service also provides span processors and metric readers for
// integration with the OTEL SDK, allowing traces and metrics to be captured
// for the TUI without requiring external backends like Jaeger or Prometheus.
//
// Takes opts (...MonitoringOption) which configures the monitoring server.
//
// Returns Option which enables monitoring when passed to New.
//
// Example:
// server := piko.New(
//
//	piko.WithMonitoring(),
//
// )
// With custom configuration:
// server := piko.New(
//
//	piko.WithMonitoring(
//	    piko.WithMonitoringAddress(":9092"),
//	    piko.WithMonitoringBindAddress("0.0.0.0"),
//	),
//
// )
func WithMonitoring(opts ...MonitoringOption) Option {
	return bootstrap.WithMonitoring(opts...)
}

// WithMonitoringTransport sets the transport factory for the monitoring
// service.
//
// Example:
//
//	piko.WithMonitoring(
//	    piko.WithMonitoringTransport(monitoring_transport_grpc.Transport()),
//	)
//
// Takes factory (monitoring_domain.TransportFactory) which creates the
// transport server.
//
// Returns MonitoringOption which configures the transport on the service.
func WithMonitoringTransport(factory monitoring_domain.TransportFactory) MonitoringOption {
	return bootstrap.WithMonitoringTransport(factory)
}

// WithMonitoringOtelFactories overrides the default noop monitoring service
// factories with real OTEL SDK implementations. Use this with the factories
// from logger_otel_sdk.OtelServiceFactories() to enable SDK-backed span
// processing and metrics collection.
//
// Example:
//
//	piko.WithMonitoring(
//	    piko.WithMonitoringOtelFactories(logger_otel_sdk.OtelServiceFactories()),
//	)
//
// Takes factories (monitoring_domain.ServiceFactories) which provides the
// span processor and metrics collector factories.
//
// Returns MonitoringOption which configures the factories on the service.
func WithMonitoringOtelFactories(factories monitoring_domain.ServiceFactories) MonitoringOption {
	return bootstrap.WithMonitoringOtelFactories(factories)
}

// WithAutoMemoryLimit configures the Go runtime to set GOMEMLIMIT based on
// the container's cgroup memory limit. This prevents OOM kills in
// containerised deployments by making the garbage collector aware of the
// memory ceiling.
//
// Takes provider (func() (int64, error)) which detects and applies the
// memory limit. Use the automemlimit WDK module for a ready-made provider.
//
// Returns Option which configures automatic memory limit detection.
func WithAutoMemoryLimit(provider func() (int64, error)) Option {
	return bootstrap.WithAutoMemoryLimit(provider)
}

// WithDiagnosticDirectory sets a single root directory for all runtime
// diagnostic artefacts: the crash mirror file (dir/crash.log) and the
// watchdog's profile / sidecar / startup-history files (dir/profiles).
//
// Takes directory (string) which is the root directory for diagnostic
// artefacts.
//
// Returns Option which sets the diagnostic directory on the container.
func WithDiagnosticDirectory(directory string) Option {
	return bootstrap.WithDiagnosticDirectory(directory)
}

// WithCrashOutput configures runtime/debug.SetCrashOutput so the Go runtime
// mirrors fatal-error output (panics, stack overflows, concurrent map
// writes, OOM aborts) to the given file path. The file is opened in append
// mode so captures from earlier crashes are preserved, and the runtime
// retains ownership of the file descriptor for the process lifetime.
//
// When the file cannot be opened the feature is disabled silently and a
// warning is logged. Crash output must never block startup.
//
// Pass an empty path to leave the feature disabled (default).
//
// Takes path (string) which is the absolute file path for crash output.
//
// Returns Option which sets the crash-output path.
func WithCrashOutput(path string) Option {
	return bootstrap.WithCrashOutput(path)
}

// WithCrashTraceback sets the GOTRACEBACK level via
// runtime/debug.SetTraceback. Valid levels are "none", "single" (Go
// default), "all", "system", "crash" (raises SIGABRT after the traceback so
// the kernel or systemd-coredump can capture a coredump), and "wer"
// (Windows error reporting).
//
// Takes level (string) which is the traceback level.
//
// Returns Option which sets the crash-traceback level.
func WithCrashTraceback(level string) Option {
	return bootstrap.WithCrashTraceback(level)
}

// WithMonitoringProfiling enables the remote profiling gRPC service, allowing
// operators to toggle pprof on and off at runtime via the monitoring endpoint.
// Without this option, the ProfilingService is not registered and profiling
// cannot be controlled remotely.
//
// Usage:
//
//	piko.WithMonitoring(
//	    piko.WithMonitoringTransport(monitoring_transport_grpc.Transport()),
//	    piko.WithMonitoringProfiling(),
//	)
//
// Once enabled, operators can use the CLI to control profiling:
//
//	piko profiling enable 30m     # Start profiling for 30 minutes
//	piko profiling capture heap   # Capture a heap profile
//	piko profiling disable        # Stop profiling early
//
// Returns MonitoringOption which enables the profiling service.
func WithMonitoringProfiling() MonitoringOption {
	return bootstrap.WithMonitoringProfiling()
}

// WatchdogOption configures the runtime watchdog.
type WatchdogOption = bootstrap.WatchdogOption

// WithMonitoringWatchdog enables the runtime watchdog that monitors heap
// memory, goroutine counts, and GC pressure, automatically capturing
// diagnostic profiles when anomalies are detected.
//
// Takes opts (...WatchdogOption) which configure thresholds and behaviour.
//
// Returns MonitoringOption which enables the watchdog on the service.
func WithMonitoringWatchdog(opts ...WatchdogOption) MonitoringOption {
	return bootstrap.WithMonitoringWatchdog(opts...)
}

// WithWatchdogHeapThresholdPercent sets the heap threshold as a fraction of
// GOMEMLIMIT (0.0-1.0). Default: 0.85.
//
// Takes percent (float64) which is the threshold fraction.
//
// Returns WatchdogOption which configures the heap threshold.
func WithWatchdogHeapThresholdPercent(percent float64) WatchdogOption {
	return bootstrap.WithWatchdogHeapThresholdPercent(percent)
}

// WithWatchdogHeapThresholdBytes sets the absolute heap threshold in bytes,
// used when GOMEMLIMIT is not configured. Default: 512 MiB.
//
// Takes thresholdBytes (uint64) which is the threshold in bytes.
//
// Returns WatchdogOption which configures the heap threshold.
func WithWatchdogHeapThresholdBytes(thresholdBytes uint64) WatchdogOption {
	return bootstrap.WithWatchdogHeapThresholdBytes(thresholdBytes)
}

// WithWatchdogGoroutineThreshold sets the goroutine count that triggers a
// goroutine profile capture. Default: 10,000.
//
// Takes threshold (int) which is the goroutine count threshold.
//
// Returns WatchdogOption which configures the goroutine threshold.
func WithWatchdogGoroutineThreshold(threshold int) WatchdogOption {
	return bootstrap.WithWatchdogGoroutineThreshold(threshold)
}

// WithWatchdogCheckInterval sets how often the watchdog evaluates runtime
// metrics.
//
// Shorter intervals detect anomalies faster at negligible CPU cost.
// Default: 500ms.
//
// Takes interval (time.Duration) which is the check period.
//
// Returns WatchdogOption which configures the check interval.
func WithWatchdogCheckInterval(interval time.Duration) WatchdogOption {
	return bootstrap.WithWatchdogCheckInterval(interval)
}

// WithWatchdogMaxProfilesPerType sets the maximum number of stored profiles
// per type (heap, goroutine).
//
// Oldest profiles are rotated out. Default: 5.
//
// Takes count (int) which is the maximum profile count per type.
//
// Returns WatchdogOption which configures profile rotation.
func WithWatchdogMaxProfilesPerType(count int) WatchdogOption {
	return bootstrap.WithWatchdogMaxProfilesPerType(count)
}

// WithWatchdogCooldown sets the minimum duration between consecutive profile
// captures for the same metric type. Default: 2 minutes.
//
// Takes duration (time.Duration) which is the cooldown period.
//
// Returns WatchdogOption which configures the cooldown.
func WithWatchdogCooldown(duration time.Duration) WatchdogOption {
	return bootstrap.WithWatchdogCooldown(duration)
}

// WithWatchdogProfileDirectory sets the local directory for profile storage.
// Default: os.TempDir()/piko-watchdog.
//
// Takes directory (string) which is the directory path.
//
// Returns WatchdogOption which configures the profile directory.
func WithWatchdogProfileDirectory(directory string) WatchdogOption {
	return bootstrap.WithWatchdogProfileDirectory(directory)
}

// WithWatchdogDeltaProfiling enables storing a baseline heap profile alongside
// each capture so the user can compute a diff between consecutive captures
// using `go tool pprof -diff_base`.
//
// Returns WatchdogOption which enables delta profiling.
func WithWatchdogDeltaProfiling() WatchdogOption {
	return bootstrap.WithWatchdogDeltaProfiling()
}

// WithWatchdogIncludeGoroutineStacks toggles per-goroutine stack capture.
// When enabled, each goroutine profile firing also writes a human-readable
// .stacks.txt sidecar containing the full stack of every goroutine
// (pprof debug=2 output), alongside the existing aggregated .pb.gz binary
// profile.
//
// Useful when investigating goroutine leaks where you need to know the exact
// call site or closure-captured arguments (e.g. which channel a publisher is
// blocked on). Disabled by default because the sidecar can be tens of
// megabytes per dump for processes with many thousand goroutines.
//
// Takes enabled (bool) which toggles the feature.
//
// Returns WatchdogOption which configures the stacks sidecar capture.
func WithWatchdogIncludeGoroutineStacks(enabled bool) WatchdogOption {
	return bootstrap.WithWatchdogIncludeGoroutineStacks(enabled)
}

// WithWatchdogRSSThresholdPercent sets the fraction of the cgroup memory limit
// above which RSS triggers a profile capture. Default: 0.85.
//
// Takes percent (float64) which is the threshold fraction (0.0-1.0).
//
// Returns WatchdogOption which configures the RSS threshold.
func WithWatchdogRSSThresholdPercent(percent float64) WatchdogOption {
	return bootstrap.WithWatchdogRSSThresholdPercent(percent)
}

// WithWatchdogFDPressureThresholdPercent sets the fraction of the soft
// RLIMIT_NOFILE above which the watchdog emits an FD pressure warning.
//
// Default is 0.80; pass 0 to disable the rule.
//
// Takes percent (float64) which is the threshold fraction (0.0-1.0).
//
// Returns WatchdogOption which configures the FD pressure threshold.
func WithWatchdogFDPressureThresholdPercent(percent float64) WatchdogOption {
	return bootstrap.WithWatchdogFDPressureThresholdPercent(percent)
}

// WithWatchdogSchedulerLatencyP99Threshold sets the p99 scheduler latency
// above which the watchdog emits a scheduler-latency warning.
//
// Default is 10ms; pass zero to disable the rule.
//
// Takes threshold (time.Duration) which is the p99 latency threshold.
//
// Returns WatchdogOption which configures the scheduler-latency threshold.
func WithWatchdogSchedulerLatencyP99Threshold(threshold time.Duration) WatchdogOption {
	return bootstrap.WithWatchdogSchedulerLatencyP99Threshold(threshold)
}

// WithWatchdogMaxWarningsPerWindow sets the maximum number of warning-only
// events permitted within a single CaptureWindow.
//
// Default is 10. Warnings have their own budget separate from profile
// captures so flapping warnings cannot crowd out real captures.
//
// Takes count (int) which is the maximum warnings per window.
//
// Returns WatchdogOption which configures the warning budget.
func WithWatchdogMaxWarningsPerWindow(count int) WatchdogOption {
	return bootstrap.WithWatchdogMaxWarningsPerWindow(count)
}

// WithWatchdogContinuousProfiling enables the continuous-profiling loop
// which captures routine profile snapshots so post-mortem operators have
// recent profiles even when no threshold breach occurred. Default
// behaviour is disabled (opt-in).
//
// Returns WatchdogOption which enables continuous profiling.
func WithWatchdogContinuousProfiling() WatchdogOption {
	return bootstrap.WithWatchdogContinuousProfiling()
}

// WithWatchdogContinuousProfilingInterval sets the interval between routine
// profile captures.
//
// Default is 10 minutes; validation enforces a minimum of 1 minute.
//
// Takes interval (time.Duration) which is the period between captures.
//
// Returns WatchdogOption which configures the routine capture interval.
func WithWatchdogContinuousProfilingInterval(interval time.Duration) WatchdogOption {
	return bootstrap.WithWatchdogContinuousProfilingInterval(interval)
}

// WithWatchdogContinuousProfilingTypes sets the profile types captured each
// routine interval.
//
// Default is ["heap"]. Allowed values are heap, goroutine, and allocs.
//
// Takes types (...string) which is the list of profile types to capture.
//
// Returns WatchdogOption which configures the routine capture set.
func WithWatchdogContinuousProfilingTypes(types ...string) WatchdogOption {
	return bootstrap.WithWatchdogContinuousProfilingTypes(types...)
}

// WithWatchdogContinuousProfilingRetention sets the maximum number of
// routine profile files retained per type. Default: 6.
//
// Takes count (int) which is the retention cap.
//
// Returns WatchdogOption which configures routine profile retention.
func WithWatchdogContinuousProfilingRetention(count int) WatchdogOption {
	return bootstrap.WithWatchdogContinuousProfilingRetention(count)
}

// WithWatchdogContinuousProfilingNotify enables informational notifications
// for each routine capture. Default behaviour is suppression to avoid
// flooding notifiers.
//
// Returns WatchdogOption which enables routine-capture notifications.
func WithWatchdogContinuousProfilingNotify() WatchdogOption {
	return bootstrap.WithWatchdogContinuousProfilingNotify()
}

// WithWatchdogContentionDiagnosticWindow sets the duration during which
// block + mutex profiling are active during a contention diagnostic.
//
// Default is 60s; allowed range is 1s to 5m.
//
// Takes window (time.Duration) which is the diagnostic window duration.
//
// Returns WatchdogOption which configures the diagnostic window.
func WithWatchdogContentionDiagnosticWindow(window time.Duration) WatchdogOption {
	return bootstrap.WithWatchdogContentionDiagnosticWindow(window)
}

// WithWatchdogContentionDiagnosticAutoFire enables automatic
// contention-diagnostic firing when scheduler-latency events repeat.
// Default behaviour is manual (operator must call RunContentionDiagnostic).
//
// Returns WatchdogOption which enables automatic contention diagnostic
// firing.
func WithWatchdogContentionDiagnosticAutoFire() WatchdogOption {
	return bootstrap.WithWatchdogContentionDiagnosticAutoFire()
}

// WithWatchdogContentionDiagnosticBlockProfileRate sets the runtime block
// profile rate during a contention diagnostic. Default: 1e6 (one sample
// per 1ms of blocking).
//
// Takes rate (int) which is the runtime block profile rate.
//
// Returns WatchdogOption which configures the block profile rate.
func WithWatchdogContentionDiagnosticBlockProfileRate(rate int) WatchdogOption {
	return bootstrap.WithWatchdogContentionDiagnosticBlockProfileRate(rate)
}

// WithWatchdogContentionDiagnosticMutexProfileFraction sets the runtime
// mutex profile fraction during a contention diagnostic. Default: 100.
//
// Takes fraction (int) which is the runtime mutex profile fraction.
//
// Returns WatchdogOption which configures the mutex profile fraction.
func WithWatchdogContentionDiagnosticMutexProfileFraction(fraction int) WatchdogOption {
	return bootstrap.WithWatchdogContentionDiagnosticMutexProfileFraction(fraction)
}

// WatchdogNotifier delivers watchdog event notifications to external systems.
type WatchdogNotifier = monitoring_domain.WatchdogNotifier

// WatchdogProfileUploader uploads captured diagnostic profiles to remote
// storage.
type WatchdogProfileUploader = monitoring_domain.WatchdogProfileUploader

// WatchdogEvent describes a notable runtime event detected by the watchdog.
type WatchdogEvent = monitoring_domain.WatchdogEvent

// WatchdogEventType identifies the category of a watchdog event.
type WatchdogEventType = monitoring_domain.WatchdogEventType

// WatchdogEventPriority indicates the urgency of a watchdog event.
type WatchdogEventPriority = monitoring_domain.WatchdogEventPriority

// WithWatchdogNotifier sets the notification delivery mechanism for watchdog
// events. When set, the watchdog sends notifications to external systems when
// thresholds are breached or errors occur.
//
// Takes notifier (WatchdogNotifier) which delivers event notifications.
//
// Returns MonitoringOption which configures the notifier on the service.
func WithWatchdogNotifier(notifier WatchdogNotifier) MonitoringOption {
	return bootstrap.WithWatchdogNotifier(notifier)
}

// WithWatchdogProfileUploader sets the remote storage backend for watchdog
// profile uploads. When set, captured profiles are uploaded after being
// written to local disk.
//
// Takes uploader (WatchdogProfileUploader) which handles remote storage.
//
// Returns MonitoringOption which configures the uploader on the service.
func WithWatchdogProfileUploader(uploader WatchdogProfileUploader) MonitoringOption {
	return bootstrap.WithWatchdogProfileUploader(uploader)
}

// WithProfilingPort sets the port for the pprof HTTP server.
//
// Takes port (int) which specifies the port number to listen on.
//
// Returns ProfilingOption which configures the pprof server port.
func WithProfilingPort(port int) ProfilingOption {
	return bootstrap.WithProfilingPort(port)
}

// WithProfilingBindAddress sets the network address for the pprof server to
// bind to.
//
// Takes addr (string) which specifies the bind address.
//
// Returns ProfilingOption which configures the pprof server bind address.
func WithProfilingBindAddress(addr string) ProfilingOption {
	return bootstrap.WithProfilingBindAddress(addr)
}

// WithProfilingBlockRate sets the block profiling rate. After calling
// runtime.SetBlockProfileRate, the profiler samples one blocking event per
// this many nanoseconds of blocking.
//
// Takes rate (int) which specifies the sampling rate in nanoseconds.
//
// Returns ProfilingOption which configures the block profile rate.
func WithProfilingBlockRate(rate int) ProfilingOption {
	return bootstrap.WithProfilingBlockRate(rate)
}

// WithProfilingMutexFraction sets the mutex profiling fraction. On average
// 1/n mutex contention events are reported.
//
// Takes fraction (int) which specifies the sampling fraction.
//
// Returns ProfilingOption which configures the mutex profile fraction.
func WithProfilingMutexFraction(fraction int) ProfilingOption {
	return bootstrap.WithProfilingMutexFraction(fraction)
}

// WithProfilingMemProfileRate sets the memory profiling
// sample rate; 0 uses the Go runtime default of 512KB and
// lower values capture smaller allocations at higher cost.
//
// Takes rate (int) which specifies the sampling rate in bytes.
//
// Returns ProfilingOption which configures the memory profile rate.
func WithProfilingMemProfileRate(rate int) ProfilingOption {
	return bootstrap.WithProfilingMemProfileRate(rate)
}

// WithProfilingRollingTrace enables an in-memory rolling execution trace buffer
// for the profiling server. The trace window can later be downloaded from
// `/_piko/profiler/trace/recent`.
//
// Returns ProfilingOption which enables rolling trace capture with safe
// defaults.
func WithProfilingRollingTrace() ProfilingOption {
	return bootstrap.WithProfilingRollingTrace()
}

// WithProfilingRollingTraceMinAge sets how much recent trace history the rolling
// trace recorder should try to retain. Implicitly enables rolling trace capture
// if not already enabled.
//
// Takes minAge (time.Duration) which specifies the retention target.
//
// Returns ProfilingOption which configures the rolling trace minimum age.
func WithProfilingRollingTraceMinAge(minAge time.Duration) ProfilingOption {
	return bootstrap.WithProfilingRollingTraceMinAge(minAge)
}

// WithProfilingRollingTraceMaxBytes sets the in-memory budget hint for the
// rolling trace recorder. Implicitly enables rolling trace capture if not
// already enabled.
//
// Takes maxBytes (uint64) which specifies the approximate buffer size.
//
// Returns ProfilingOption which configures the rolling trace memory budget.
func WithProfilingRollingTraceMaxBytes(maxBytes uint64) ProfilingOption {
	return bootstrap.WithProfilingRollingTraceMaxBytes(maxBytes)
}

// WithProfiling enables the pprof HTTP debug server. The server exposes
// profiling endpoints at /_piko/debug/pprof/ on a dedicated port (default 6060).
//
// When enabled, block and mutex profiling rates are configured so that
// /_piko/debug/pprof/block and /_piko/debug/pprof/mutex return meaningful data.
//
// The server starts during application bootstrap and stops during graceful
// shutdown.
//
// Takes opts (...ProfilingOption) which provides optional settings:
//   - WithProfilingPort(6060): sets the HTTP listen port.
//   - WithProfilingBindAddress("localhost"): sets the bind address.
//   - WithProfilingBlockRate(1000): sets the block profile rate.
//   - WithProfilingMutexFraction(10): sets the mutex profile fraction.
//   - WithProfilingMemProfileRate(0): sets the memory profile sample rate.
//   - WithProfilingRollingTrace(): enables bounded rolling trace capture.
//   - WithProfilingRollingTraceMinAge(15 * time.Second): adjusts retained trace age.
//   - WithProfilingRollingTraceMaxBytes(16 * 1024 * 1024): adjusts trace buffer size.
//
// Returns Option which configures the container with profiling settings.
//
// Example:
//
//	server := piko.New(
//	    piko.WithProfiling(),
//	)
//
// With custom port:
//
//	server := piko.New(
//	    piko.WithProfiling(
//	        piko.WithProfilingPort(7070),
//	    ),
//	)
func WithProfiling(opts ...ProfilingOption) Option {
	return bootstrap.WithProfiling(opts...)
}

// WithGeneratorProfilingOutputDir sets the directory for captured profile
// files.
//
// Takes directory (string) which specifies the output directory path.
//
// Returns GeneratorProfilingOption which configures the output directory.
func WithGeneratorProfilingOutputDir(directory string) GeneratorProfilingOption {
	return bootstrap.WithGeneratorProfilingOutputDir(directory)
}

// WithGeneratorProfilingBlockRate sets the block profiling rate for generator
// profiling.
//
// Takes rate (int) which specifies the sampling rate in nanoseconds.
//
// Returns GeneratorProfilingOption which configures the block profile rate.
func WithGeneratorProfilingBlockRate(rate int) GeneratorProfilingOption {
	return bootstrap.WithGeneratorProfilingBlockRate(rate)
}

// WithGeneratorProfilingMutexFraction sets the mutex profiling fraction for
// generator profiling.
//
// Takes fraction (int) which specifies the sampling fraction.
//
// Returns GeneratorProfilingOption which configures the mutex profile fraction.
func WithGeneratorProfilingMutexFraction(fraction int) GeneratorProfilingOption {
	return bootstrap.WithGeneratorProfilingMutexFraction(fraction)
}

// WithGeneratorProfilingMemProfileRate sets the memory profiling sample rate
// for generator profiling. 0 uses the Go runtime default.
//
// Takes rate (int) which specifies the sampling rate in bytes.
//
// Returns GeneratorProfilingOption which configures the memory profile rate.
func WithGeneratorProfilingMemProfileRate(rate int) GeneratorProfilingOption {
	return bootstrap.WithGeneratorProfilingMemProfileRate(rate)
}

// WithGeneratorProfiling enables profiling for short-lived generator builds.
// CPU, trace, heap, block, mutex, goroutine, and allocs profiles are captured
// to disk in the specified output directory (default "./profiles").
//
// Unlike WithProfiling, this does not start an HTTP server. Instead, it wraps
// the build execution: profiling starts before the build and profiles are
// written when the build completes.
//
// Takes opts (...GeneratorProfilingOption) which provides optional settings:
//   - WithGeneratorProfilingOutputDir("./profiles"): sets the output directory.
//   - WithGeneratorProfilingBlockRate(1): sets the block profile rate.
//   - WithGeneratorProfilingMutexFraction(1): sets the mutex profile fraction.
//
// Returns Option which configures the container with generator profiling
// settings.
//
// Example:
//
//	server := piko.New(
//	    piko.WithGeneratorProfiling(),
//	)
//
// With custom output directory:
//
//	server := piko.New(
//	    piko.WithGeneratorProfiling(
//	        piko.WithGeneratorProfilingOutputDir("/tmp/profiles"),
//	    ),
//	)
func WithGeneratorProfiling(opts ...GeneratorProfilingOption) Option {
	return bootstrap.WithGeneratorProfiling(opts...)
}

// WithMetricsExporter configures a metrics exporter for the health probe
// server. When enabled, OTEL metrics are exposed at /metrics on port 9090 by
// default.
//
// The exporter integrates with the OTEL MeterProvider, so all metrics
// recorded through OTEL instrumentation will be available for scraping.
//
// Example with Prometheus:
// import prometheus "piko.sh/piko/wdk/metrics/metrics_exporter_prometheus"
// server := piko.New(
//
//	piko.WithMetricsExporter(prometheus.New()),
//
// )
// The metrics endpoint path can be configured via the healthProbe.metricsPath
// config option.
//
// Takes exporter (MetricsExporter) which provides the metrics export backend.
//
// Returns Option which configures the metrics exporter on the server.
func WithMetricsExporter(exporter MetricsExporter) Option {
	return bootstrap.WithMetricsExporter(exporter)
}

// WithDatabase registers a named SQL database connection for the registry or
// orchestrator subsystem. When a database named "registry" or "orchestrator"
// is registered, piko uses a SQL-backed DAL instead of the default otter
// in-memory backend for that subsystem.
//
// Available driver packages:
//   - db_driver_sqlite_cgo: CGO SQLite (maximum performance)
//   - db_driver_sqlite_nocgo: Pure-Go SQLite (no CGO required)
//   - db_driver_d1: Cloudflare D1
//
// Example:
//
//	import (
//	    "piko.sh/piko"
//	    "piko.sh/piko/wdk/db"
//	    "piko.sh/piko/wdk/db/db_driver_sqlite_cgo"
//	    "piko.sh/piko/wdk/db/db_engine_sqlite"
//	    "piko.sh/piko/wdk/db/db_schema_registry_sqlite"
//	    "piko.sh/piko/wdk/db/db_schema_orchestrator_sqlite"
//	)
//
//	app := piko.New(
//	    piko.WithDatabase(db.DatabaseNameRegistry, &db.DatabaseRegistration{
//	        DB:           db_driver_sqlite_cgo.Open("data/registry.db", db_driver_sqlite_cgo.Config{}),
//	        EngineConfig: db_engine_sqlite.SQLite(),
//	        MigrationFS:  db_schema_registry_sqlite.Migrations,
//	    }),
//	    piko.WithDatabase(db.DatabaseNameOrchestrator, &db.DatabaseRegistration{
//	        DB:           db_driver_sqlite_cgo.Open("data/orchestrator.db", db_driver_sqlite_cgo.Config{}),
//	        EngineConfig: db_engine_sqlite.SQLite(),
//	        MigrationFS:  db_schema_orchestrator_sqlite.Migrations,
//	    }),
//	)
//
// Takes name (string) which identifies the database (db.DatabaseNameRegistry
// or db.DatabaseNameOrchestrator for framework subsystems, or any custom name
// for application databases).
// Takes registration (*bootstrap.DatabaseRegistration) which provides the
// connection and migration configuration.
//
// Returns Option which registers the database with the application.
func WithDatabase(name string, registration *bootstrap.DatabaseRegistration) Option {
	return bootstrap.WithDatabase(name, registration)
}

// WithPMLTransformer sets a custom PML transformation engine.
//
// Use it to customise the PikoML to HTML transformation process. If not
// provided, a default transformer with all built-in components will be used.
//
// Takes transformer (pml_domain.Transformer) which handles the PikoML to HTML
// conversion.
//
// Returns Option which configures the container with the custom transformer.
func WithPMLTransformer(transformer pml_domain.Transformer) Option {
	return func(c *bootstrap.Container) {
		c.SetPMLTransformer(transformer)
	}
}

// DefaultRegistryMetadataCacheConfig returns the default configuration for the
// Registry metadata cache.
//
// Returns RegistryMetadataCacheConfig which provides sensible defaults with
// 256 MB max weight, 30 minute TTL, and stats disabled.
func DefaultRegistryMetadataCacheConfig() RegistryMetadataCacheConfig {
	return RegistryMetadataCacheConfig{
		MaxWeight:    defaultRegistryCacheMaxWeightMB * bytesPerMB,
		TTL:          defaultRegistryCacheTTLMinutes * time.Minute,
		StatsEnabled: false,
	}
}

// WithRegistryMetadataCacheConfig configures the Registry service's metadata
// cache. The metadata cache stores artefact metadata to reduce database queries
// and improve Registry performance, especially for frequently accessed
// artefacts.
//
// The cache uses weight-based eviction (calculated by the approximate memory
// size of artefact metadata) and access-based expiration (TTL resets on every
// read).
//
// Takes registryCacheConfig (RegistryMetadataCacheConfig) which specifies the
// cache settings including maximum weight, TTL, and whether stats are enabled.
//
// Returns Option which configures the metadata cache when applied.
//
// Example:
// import "piko.sh/piko"
// import "time"
// server, err := piko.New(
//
//	piko.WithRegistryMetadataCacheConfig(piko.RegistryMetadataCacheConfig{
//	    MaxWeight:    512 * 1024 * 1024, // 512 MB
//	    TTL:          time.Hour,           // 1 hour sliding window
//	    StatsEnabled: true,                // Enable observability
//	}),
//
// )
func WithRegistryMetadataCacheConfig(registryCacheConfig RegistryMetadataCacheConfig) Option {
	return func(c *bootstrap.Container) {
		c.SetRegistryMetadataCacheConfig(bootstrap.RegistryMetadataCacheConfig{
			MaxWeight:    registryCacheConfig.MaxWeight,
			TTL:          registryCacheConfig.TTL,
			StatsEnabled: registryCacheConfig.StatsEnabled,
		})
	}
}

// NewCSPBuilder creates a new CSP builder with no directives configured.
// This is the starting point for building a Content-Security-Policy.
//
// Returns *CSPBuilder which is ready for chaining directive methods.
//
// Example:
// builder := piko.NewCSPBuilder().
//
//	DefaultSrc(piko.CSPSelf).
//	ScriptSrc(piko.CSPSelf, piko.CSPHost("cdn.example.com")).
//	Build()
func NewCSPBuilder() *CSPBuilder {
	return security_domain.NewCSPBuilder()
}

// WithCSP configures the Content-Security-Policy header using a
// builder function.
//
// Takes configure (func(*CSPBuilder)) which is a callback that
// receives a CSPBuilder for configuring CSP directives.
//
// Returns Option which applies the CSP configuration to the server.
//
// The builder function receives a CSPBuilder that you configure
// with your desired directives. The resulting policy is applied
// to all HTTP responses.
//
// Example - Simple custom policy:
// server := piko.New(
//
//	piko.WithCSP(func(b *piko.CSPBuilder) {
//		b.DefaultSrc(piko.CSPSelf).
//			ScriptSrc(piko.CSPSelf, piko.CSPHost("cdn.example.com")).
//			StyleSrc(piko.CSPSelf, piko.CSPUnsafeInline)
//	}),
//
// )
// Example - Extend Piko defaults:
// server := piko.New(
//
//	piko.WithCSP(func(b *piko.CSPBuilder) {
//		b.WithPikoDefaults().
//			ScriptSrc(piko.CSPSelf, piko.CSPHost("analytics.example.com"))
//	}),
//
// )
// Example - Report-only mode for testing:
// server := piko.New(
//
//	piko.WithCSP(func(b *piko.CSPBuilder) {
//		b.WithPikoDefaults().
//			ReportOnly().
//			ReportToDirective("csp-violations")
//	}),
//
// )
// Example - Dynamic per-request tokens for strict CSP:
// server := piko.New(
//
//	piko.WithCSP(func(b *piko.CSPBuilder) {
//		b.DefaultSrc(piko.CSPSelf).
//			ScriptSrc(piko.CSPSelf, piko.CSPStrictDynamic, piko.CSPRequestToken).
//			StyleSrc(piko.CSPSelf, piko.CSPRequestToken)
//	}),
//
// )
// When using CSPRequestToken, add the token attribute to inline elements in
// templates:
// <script {{ .CSPTokenAttr }}>...</script>
// <style {{ .CSPTokenAttr }}>...</style>
func WithCSP(configure func(*CSPBuilder)) Option {
	return func(c *bootstrap.Container) {
		builder := security_domain.NewCSPBuilder()
		configure(builder)
		c.SetCSPConfig(builder)
	}
}

// WithCSPString sets the Content-Security-Policy header directly from a string.
// This is an escape hatch for complex cases that don't fit the builder pattern,
// or for migrating existing CSP configurations.
//
// Pass an empty string to disable the CSP header entirely.
//
// Takes policy (string) which specifies the raw CSP policy value.
//
// Returns Option which configures the CSP header on the container.
//
// Example:
// // Use a pre-existing policy string
// server := piko.New(
//
//	piko.WithCSPString("default-src 'self'; script-src 'self' cdn.example.com"),
//
// )
// // Disable CSP entirely
// server := piko.New(
//
//	piko.WithCSPString(""),
//
// )
func WithCSPString(policy string) Option {
	return func(c *bootstrap.Container) {
		c.SetCSPPolicyString(policy)
	}
}

// WithPikoDefaultCSP configures the Content-Security-Policy with Piko's
// recommended default settings. This provides a secure baseline that works
// with Piko's built-in features including font loading, inline styles,
// and the async font loader script.
//
// The default policy includes:
//   - default-src 'self'
//   - style-src 'self' 'unsafe-inline' https://fonts.googleapis.com
//   - script-src-attr 'unsafe-hashes' 'sha256-...' (for font loader)
//   - font-src 'self' https://fonts.gstatic.com data:
//   - img-src 'self' data: blob: https:
//   - connect-src 'self'
//
// Returns Option which applies the default CSP configuration to the server.
//
// Example:
// server := piko.New(
//
//	piko.WithPikoDefaultCSP(),
//
// )
func WithPikoDefaultCSP() Option {
	return func(c *bootstrap.Container) {
		builder := security_domain.NewCSPBuilder().WithPikoDefaults()
		c.SetCSPConfig(builder)
	}
}

// WithStrictCSP configures a strict Content Security Policy following Google's
// recommendations, using per-request tokens and 'strict-dynamic' for script
// execution to provide strong XSS protection while allowing dynamically loaded
// scripts.
//
// The policy includes:
//   - default-src 'self'
//   - script-src 'strict-dynamic' {{REQUEST_TOKEN}} (token-based)
//   - style-src 'self' {{REQUEST_TOKEN}} (token-based)
//   - object-src 'none' (blocks plugins like Flash)
//   - base-uri 'self' (prevents base tag hijacking)
//   - frame-ancestors 'self' (clickjacking protection)
//   - upgrade-insecure-requests
//
// Returns Option which applies the strict CSP configuration to the server.
//
// IMPORTANT: This policy uses request tokens. Templates must use
// {{ .CSPTokenAttr }} on inline script and style elements:
// <script {{ .CSPTokenAttr }}>console.log("safe");</script>
// <style {{ .CSPTokenAttr }}>.my-class { color: red; }</style>
// Example:
// server := piko.New(
//
//	piko.WithStrictCSP(),
//
// )
func WithStrictCSP() Option {
	return func(c *bootstrap.Container) {
		builder := security_domain.NewCSPBuilder().WithStrictPolicy()
		c.SetCSPConfig(builder)
	}
}

// WithRelaxedCSP configures a permissive CSP policy for legacy applications.
//
// Returns Option which applies a relaxed CSP policy to the server.
//
// WARNING: This policy allows 'unsafe-inline' and 'unsafe-eval', which
// significantly reduces XSS protection. Use only when migrating legacy
// code that cannot be updated to use token-based CSP or content hashes.
//
// The policy includes:
//   - default-src 'self'
//   - script-src 'self' 'unsafe-inline' 'unsafe-eval'
//   - style-src 'self' 'unsafe-inline'
//   - img-src 'self' data: https:
//   - font-src 'self' data:
//   - connect-src 'self'
//   - frame-ancestors 'self' (clickjacking protection)
//
// Example:
// server := piko.New(
//
//	piko.WithRelaxedCSP(),
//
// )
func WithRelaxedCSP() Option {
	return func(c *bootstrap.Container) {
		builder := security_domain.NewCSPBuilder().WithRelaxedPolicy()
		c.SetCSPConfig(builder)
	}
}

// WithAPICSP configures a minimal CSP policy for JSON API servers.
// This policy blocks all resource types, which protects against cases where
// the API accidentally serves HTML content (e.g., error pages with user input).
//
// The policy includes:
//   - default-src 'none' (blocks everything by default)
//   - frame-ancestors 'none' (cannot be embedded in frames)
//   - base-uri 'none' (no base element allowed)
//   - form-action 'none' (no form submissions)
//
// This is ideal for API endpoints that should only return JSON data.
//
// Returns Option which applies the API CSP configuration to the server.
//
// Example:
// server := piko.New(
//
//	piko.WithAPICSP(),
//
// )
func WithAPICSP() Option {
	return func(c *bootstrap.Container) {
		builder := security_domain.NewCSPBuilder().WithAPIPolicy()
		c.SetCSPConfig(builder)
	}
}

// WithReportingEndpoints enables the Reporting-Endpoints HTTP header and
// configures the specified reporting endpoints. These endpoints can be
// referenced by CSP report-to directives and other web platform reporting APIs
// (Network Error Logging, Deprecation Reports, Crash Reports).
//
// Takes endpoints (...ReportingEndpoint) which specifies the reporting
// endpoints to configure.
//
// Returns Option which configures the server with the reporting endpoints.
//
// Example:
// server := piko.New(
//
//	piko.WithReportingEndpoints(
//		piko.ReportingEndpoint{
//			Name: "csp-violations",
//			URL:  "https://monitoring.example.com/reports/csp",
//		},
//		piko.ReportingEndpoint{
//			Name: "deprecations",
//			URL:  "https://monitoring.example.com/reports/deprecations",
//		},
//	),
//	piko.WithCSP(func(b *piko.CSPBuilder) {
//		b.WithStrictPolicy().
//			ReportToDirective("csp-violations")
//	}),
//
// )
// The resulting headers will be:
// Reporting-Endpoints: csp-violations="...", deprecations="..."
// Content-Security-Policy: ...; report-to csp-violations
func WithReportingEndpoints(endpoints ...ReportingEndpoint) Option {
	return func(c *bootstrap.Container) {
		c.SetReportingEndpoints(endpoints)
	}
}

// WithCrossOriginResourcePolicy sets the Cross-Origin-Resource-Policy header
// value. This header controls which origins can load resources from this
// server.
//
// Takes policy (string) which specifies the CORP policy. Use one of:
//   - CORPSameOrigin (default): Only same-origin requests can load resources
//   - CORPSameSite: Same-site requests can load resources
//   - CORPCrossOrigin: Any origin can load resources (required for headless
//     CMS)
//
// Returns Option which configures the CORP header on the container.
//
// Example - Enable cross-origin resource sharing for headless CMS:
// server := piko.New(
//
//	piko.WithCrossOriginResourcePolicy(piko.CORPCrossOrigin),
//
// )
func WithCrossOriginResourcePolicy(policy string) Option {
	return func(c *bootstrap.Container) {
		c.SetCrossOriginResourcePolicy(policy)
	}
}

// WithTrustedProxies configures the CIDR ranges of reverse proxies trusted to
// set X-Forwarded-For headers. When a request arrives from one of these ranges,
// the real client IP is extracted from forwarding headers rather than using the
// connection IP directly.
//
// Common values include RFC 1918 private ranges: "10.0.0.0/8",
// "172.16.0.0/12", "192.168.0.0/16". For Cloudflare deployments, add their
// published IP ranges from https://www.cloudflare.com/ips/.
//
// Takes cidrs (...string) which are the CIDR ranges to trust.
//
// Returns Option which configures the trusted proxy list on the server.
//
// Example:
// server := piko.New(
//
//	piko.WithTrustedProxies("10.0.0.0/8", "172.16.0.0/12"),
//
// )
func WithTrustedProxies(cidrs ...string) Option {
	return bootstrap.WithTrustedProxies(cidrs...)
}

// WithCloudflareEnabled enables trust of the CF-Connecting-IP header from
// trusted proxies.
//
// When false (the default), the header is ignored even from trusted proxies.
// Enable this only when Cloudflare is your edge proxy and its IP ranges are
// listed in WithTrustedProxies.
//
// Takes enabled (bool) which controls whether CF-Connecting-IP is trusted.
//
// Returns Option which configures the Cloudflare setting on the server.
func WithCloudflareEnabled(enabled bool) Option {
	return bootstrap.WithCloudflareEnabled(enabled)
}

// WithRateLimitEnabled enables or disables HTTP request rate limiting.
// Rate limiting is disabled by default to prevent accidental self-limiting
// when deployed behind a reverse proxy without WithTrustedProxies configured.
//
// Takes enabled (bool) which controls whether rate limiting is active.
//
// Returns Option which configures rate limiting on the server.
//
// Example:
// server := piko.New(
//
//	piko.WithTrustedProxies("10.0.0.0/8"),
//	piko.WithRateLimitEnabled(true),
//
// )
func WithRateLimitEnabled(enabled bool) Option {
	return bootstrap.WithRateLimitEnabled(enabled)
}

// WithCSRFTokenMaxAge sets the fallback maximum age for CSRF tokens.
//
// The primary expiry mechanism is cookie rotation; this setting acts as a
// safety net for tokens backed by cookies that persist beyond their expected
// lifetime. When d is 0 or negative, the default of 30 days is used.
//
// Takes d (time.Duration) which specifies the maximum token age.
//
// Returns Option which configures the CSRF token max age on the server.
func WithCSRFTokenMaxAge(d time.Duration) Option {
	return bootstrap.WithCSRFTokenMaxAge(d)
}

// WithCSRFSecFetchSiteEnforcement controls whether browser requests must
// include CSRF tokens when the Sec-Fetch-Site header is present.
//
// Takes enabled (bool) which controls whether enforcement is active.
//
// Returns Option which configures the CSRF enforcement on the server.
func WithCSRFSecFetchSiteEnforcement(enabled bool) Option {
	return bootstrap.WithCSRFSecFetchSiteEnforcement(enabled)
}

// WithStartupBanner controls whether the startup information banner is
// displayed when the server starts. Defaults to true.
//
// Takes enabled (bool) which specifies whether the banner is shown.
//
// Returns Option which configures the startup banner setting.
func WithStartupBanner(enabled bool) Option {
	return bootstrap.WithStartupBanner(enabled)
}

// WithIAmACatPerson swaps the large pixel-art mascot in the startup banner
// for the small ASCII art version. Defaults to false.
//
// Returns Option which configures the mascot preference.
func WithIAmACatPerson() Option {
	return bootstrap.WithIAmACatPerson()
}

// WithWatchMode controls whether file system watching for hot-reloading is
// enabled. This is typically derived from the run mode (dev -> true,
// prod -> false) and does not need to be set manually.
//
// Takes enabled (bool) which controls whether watch mode is active.
//
// Returns Option which configures the watch mode setting.
func WithWatchMode(enabled bool) Option {
	return bootstrap.WithWatchMode(enabled)
}

// WithE2EMode controls whether E2E test pages and partials are included in
// the build. WARNING: Never enable in production.
//
// Takes enabled (bool) which controls whether E2E mode is active.
//
// Returns Option which configures the E2E mode setting.
func WithE2EMode(enabled bool) Option {
	return bootstrap.WithE2EMode(enabled)
}

// WithStorageService overrides the default storage service with a custom
// implementation. Use it for testing or providing a fully configured service
// with custom behaviour.
//
// Takes service (storage_domain.Service) which is the custom storage service.
//
// Returns Option which configures the container to use the provided service.
func WithStorageService(service storage_domain.Service) Option {
	return func(c *bootstrap.Container) {
		c.SetStorageService(service)
	}
}

// WithStorageProvider registers a named storage provider instance with the
// default storage service. If multiple providers are registered, use
// WithDefaultStorageProvider to specify which one is the default.
//
// Takes name (string) which identifies the storage provider.
// Takes provider (StorageProviderPort) which is the storage provider instance.
//
// Returns Option which configures the container with the storage provider.
func WithStorageProvider(name string, provider storage_domain.StorageProviderPort) Option {
	return func(c *bootstrap.Container) {
		c.AddStorageProvider(name, provider)
	}
}

// WithSystemStorageProvider registers a storage provider for Piko internal
// operations such as registry blob storage.
//
// If not set, Piko internals will use the "default" provider. Set it to
// separate application storage from Piko's internal storage needs.
//
// Example: Use GCS for app storage but keep Piko internals on local disk:
// piko.New(
//
//	piko.WithStorageProvider("default", gcsProvider),  // App uses GCS
//	piko.WithSystemStorageProvider(diskProvider),      // Piko uses disk
//
// )
//
// Takes provider (StorageProviderPort) which provides the storage
// backend for Piko internal operations.
//
// Returns Option which configures the container with the system storage
// provider.
func WithSystemStorageProvider(provider storage_domain.StorageProviderPort) Option {
	return func(c *bootstrap.Container) {
		c.AddStorageProvider(storage_dto.StorageProviderSystem, provider)
	}
}

// WithDefaultStorageProvider sets the name of the provider to use for default
// storage operations. A provider with this name must be registered via
// WithStorageProvider.
//
// Takes name (string) which specifies the registered provider name.
//
// Returns Option which configures the container's default storage provider.
func WithDefaultStorageProvider(name string) Option {
	return func(c *bootstrap.Container) {
		c.SetStorageDefaultProvider(name)
	}
}

// WithStorageDispatcher enables and configures the background dispatcher for
// the default storage service.
//
// Takes storageDispatcherConfig (storage_domain.DispatcherConfig) which
// controls batching, retries, and queue sizes.
//
// Returns Option which configures the dispatcher on the container.
func WithStorageDispatcher(storageDispatcherConfig storage_domain.DispatcherConfig) Option {
	return func(c *bootstrap.Container) {
		c.SetStorageDispatcherConfig(&storageDispatcherConfig)
	}
}

// WithStoragePresignBaseURL sets the base URL prefix for presigned storage
// URLs. This is essential when Piko is used as a headless CMS where content is
// consumed by a frontend on a different host or port.
//
// Without this option, presigned URLs will be relative paths that do not work
// cross-origin.
//
// Takes baseURL (string) which is the full base URL including scheme and host,
// e.g., "http://localhost:8080" or "https://cms.example.com".
//
// Returns Option which configures the presign base URL on the storage service.
func WithStoragePresignBaseURL(baseURL string) Option {
	return func(c *bootstrap.Container) {
		c.SetStoragePresignBaseURL(baseURL)
	}
}

// WithStoragePublicBaseURL sets the base URL for public storage URLs.
//
// When set, public URLs are generated as absolute URLs
// (e.g., "http://localhost:8080/_piko/storage/public/...").
// When empty, URLs are generated as relative paths
// (e.g., "/_piko/storage/public/...").
// This is required when the website and CMS/API run on different ports
// or hosts.
//
// Takes baseURL (string) which is the full base URL including scheme and
// host, e.g., "http://localhost:8080" or "https://cms.example.com".
//
// Returns Option which configures the public base URL on the storage service.
func WithStoragePublicBaseURL(baseURL string) Option {
	return func(c *bootstrap.Container) {
		c.SetStoragePublicBaseURL(baseURL)
	}
}

// WithValidator provides a struct validator implementation. Any type that
// implements the bootstrap.StructValidator interface (a single Struct(any) error
// method) can be used.
//
// The recommended implementation is the validation_provider_playground WDK
// module, which provides a go-playground/validator backed validator with
// Piko's built-in Money and Decimal validation rules pre-registered.
//
// When no validator is configured, struct validation is skipped entirely.
//
// Takes v (bootstrap.StructValidator) which is the validator to use.
//
// Returns Option which configures Piko to use the provided validator.
//
// Example:
//
//	import (
//	    "piko.sh/piko"
//	    playground "piko.sh/piko/wdk/validation/validation_provider_playground"
//	)
//	validator := playground.NewValidator()
//	server, err := piko.New(
//	    piko.WithValidator(validator),
//	)
func WithValidator(v bootstrap.StructValidator) Option {
	return bootstrap.WithValidator(v)
}

// WithJSONProvider activates a JSON encoding provider, replacing the default
// standard-library implementation for the entire application.
//
// Takes provider (json.Provider) which supplies the JSON implementation.
//
// Returns Option which activates the provider during bootstrap.
//
// Example:
//
//	import sonicjson "piko.sh/piko/wdk/json/json_provider_sonic"
//
//	ssr := piko.New(
//	    piko.WithJSONProvider(sonicjson.New()),
//	)
func WithJSONProvider(provider json.Provider) Option {
	return func(*bootstrap.Container) {
		provider.Activate()
	}
}

// WithPort sets the TCP port for the main HTTP server.
//
// Takes port (int) which specifies the port number (e.g. 8080, 443).
//
// Returns Option which sets the server port.
func WithPort(port int) Option {
	return bootstrap.WithPort(port)
}

// WithTLS enables TLS/HTTPS for the main server. Sub-options configure
// certificate paths, mTLS, and hot-reload settings.
//
// Takes opts (...TLSOption) which provides optional TLS configuration:
//   - WithTLSCertFile("/path/to/cert.pem"): sets the certificate path.
//   - WithTLSKeyFile("/path/to/key.pem"): sets the private key path.
//   - WithTLSClientCA("/path/to/ca.pem"): enables mTLS with client CA.
//   - WithTLSClientAuth("require_and_verify"): sets client auth mode.
//   - WithTLSMinVersion("1.3"): sets minimum TLS version.
//   - WithTLSHotReload(true): enables certificate hot-reload.
//   - WithTLSRedirectHTTP("80"): starts an HTTP-to-HTTPS redirect listener.
//
// Returns Option which configures the container with TLS settings.
func WithTLS(opts ...TLSOption) Option {
	return bootstrap.WithTLS(opts...)
}

// WithTLSCertFile sets the path to the PEM-encoded TLS certificate file.
//
// Takes path (string) which specifies the certificate file path.
//
// Returns TLSOption which sets the certificate path.
func WithTLSCertFile(path string) TLSOption {
	return bootstrap.WithTLSCertFile(path)
}

// WithTLSKeyFile sets the path to the PEM-encoded TLS private key file.
//
// Takes path (string) which specifies the key file path.
//
// Returns TLSOption which sets the key path.
func WithTLSKeyFile(path string) TLSOption {
	return bootstrap.WithTLSKeyFile(path)
}

// WithTLSClientCA sets the path to a PEM-encoded CA bundle for mTLS client
// certificate verification.
//
// Takes path (string) which specifies the client CA file path.
//
// Returns TLSOption which sets the client CA path.
func WithTLSClientCA(path string) TLSOption {
	return bootstrap.WithTLSClientCA(path)
}

// WithTLSClientAuth sets the client certificate verification mode. Valid
// values are "none", "request", "require", "verify", and
// "require_and_verify".
//
// Takes authType (string) which specifies the auth mode.
//
// Returns TLSOption which sets the client auth type.
func WithTLSClientAuth(authType string) TLSOption {
	return bootstrap.WithTLSClientAuth(authType)
}

// WithTLSMinVersion sets the minimum TLS version. Valid values are "1.2"
// and "1.3".
//
// Takes version (string) which specifies the minimum version.
//
// Returns TLSOption which sets the minimum TLS version.
func WithTLSMinVersion(version string) TLSOption {
	return bootstrap.WithTLSMinVersion(version)
}

// WithTLSHotReload enables or disables automatic certificate reload when
// certificate files change on disk.
//
// Takes enabled (bool) which controls hot-reload behaviour.
//
// Returns TLSOption which sets the hot-reload flag.
func WithTLSHotReload(enabled bool) TLSOption {
	return bootstrap.WithTLSHotReload(enabled)
}

// WithTLSRedirectHTTP starts a plain HTTP listener on the given port that
// 301-redirects all requests to the HTTPS server. Use this to redirect
// http://example.com to https://example.com.
//
// Takes port (int) which specifies the HTTP port to listen on (e.g. 80
// or 8080).
//
// Returns TLSOption which configures the redirect listener.
func WithTLSRedirectHTTP(port int) TLSOption {
	return bootstrap.WithTLSRedirectHTTP(port)
}

// WithHealthTLS enables TLS for the health probe server.
//
// Takes opts (...HealthTLSOption) which provides optional health TLS settings.
//
// Returns Option which configures the container with health server TLS.
func WithHealthTLS(opts ...HealthTLSOption) Option {
	return bootstrap.WithHealthTLS(opts...)
}

// WithHealthTLSCertFile sets the certificate file for the health probe server.
//
// Takes path (string) which specifies the certificate file path.
//
// Returns HealthTLSOption which sets the certificate path.
func WithHealthTLSCertFile(path string) HealthTLSOption {
	return bootstrap.WithHealthTLSCertFile(path)
}

// WithHealthTLSKeyFile sets the key file for the health probe server.
//
// Takes path (string) which specifies the key file path.
//
// Returns HealthTLSOption which sets the key path.
func WithHealthTLSKeyFile(path string) HealthTLSOption {
	return bootstrap.WithHealthTLSKeyFile(path)
}

// WithHealthTLSMinVersion sets the minimum TLS version for the health probe
// server.
//
// Takes version (string) which specifies the minimum version ("1.2" or "1.3").
//
// Returns HealthTLSOption which sets the minimum TLS version.
func WithHealthTLSMinVersion(version string) HealthTLSOption {
	return bootstrap.WithHealthTLSMinVersion(version)
}

// WithDevWidget enables the dev tools overlay widget in dev mode.
//
// Returns Option which enables the dev widget.
func WithDevWidget() Option {
	return bootstrap.WithDevWidget()
}

// WithDevHotreload enables automatic page refresh in dev mode.
//
// Returns Option which enables dev hot-reload.
func WithDevHotreload() Option {
	return bootstrap.WithDevHotreload()
}

// WithAuthProvider registers an authentication provider that Piko calls on
// every request to resolve the auth state. The resolved AuthContext is
// available via RequestData.Auth() in pages and ActionMetadata.Auth() in
// actions.
//
// Takes provider (AuthProvider) which resolves auth state from HTTP requests.
//
// Returns Option which configures the auth provider.
func WithAuthProvider(provider AuthProvider) Option {
	return bootstrap.WithAuthProvider(provider)
}

// WithAuthGuard enables route-level authentication enforcement.
//
// Requires WithAuthProvider to be set; ignored without it.
//
// Takes authGuardConfig (AuthGuardConfig) which specifies public paths, login
// redirect, and optional custom handler.
//
// Returns Option which configures the auth guard.
func WithAuthGuard(authGuardConfig AuthGuardConfig) Option {
	return bootstrap.WithAuthGuard(authGuardConfig)
}

// IsAuthenticated reports whether the request has a valid auth session.
// Returns false when r is nil, the context is missing, or no auth provider
// is configured.
//
// Takes r (*RequestData) which is the current request to check.
//
// Returns bool which is true if the request is authenticated.
func IsAuthenticated(r *RequestData) bool {
	if r == nil {
		return false
	}
	auth := r.Auth()
	return auth != nil && auth.IsAuthenticated()
}

// WithBackendAnalytics registers one or more backend analytics
// collectors. When at least one collector is registered, the analytics
// middleware is automatically installed in the HTTP request chain
// (after auth, before rate limiting) and fires page view events for
// every request.
//
// Takes collectors (...AnalyticsCollector) which handle event delivery
// to external analytics backends.
//
// Returns Option which registers the collectors.
func WithBackendAnalytics(collectors ...AnalyticsCollector) Option {
	return bootstrap.WithBackendAnalytics(collectors...)
}

// CaptchaOptions groups the captcha provider's per-deployment settings.
// The provider implementation itself is selected via WithDefaultCaptchaProvider
// and registered via WithCaptchaProvider.
type CaptchaOptions = bootstrap.CaptchaOptions

// WithPublicDomain sets the public domain used for CORS allowed origins and
// absolute URLs. Empty string allows all origins.
//
// Takes domain (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithPublicDomain(domain string) Option {
	return bootstrap.WithPublicDomain(domain)
}

// WithForceHTTPS enables redirection from HTTP to HTTPS.
//
// Takes enabled (bool) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithForceHTTPS(enabled bool) Option {
	return bootstrap.WithForceHTTPS(enabled)
}

// WithRequestTimeout sets the maximum duration for dynamic HTTP requests.
// Zero disables the timeout middleware.
//
// Takes d (time.Duration) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithRequestTimeout(d time.Duration) Option {
	return bootstrap.WithRequestTimeout(d)
}

// WithMaxConcurrentRequests sets the maximum number of in-flight requests
// the server will process simultaneously. Zero disables the limit.
//
// Takes n (int) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithMaxConcurrentRequests(n int) Option {
	return bootstrap.WithMaxConcurrentRequests(n)
}

// WithActionMaxBodyBytes sets the maximum size in bytes for action request
// bodies.
//
// Takes n (int64) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithActionMaxBodyBytes(n int64) Option {
	return bootstrap.WithActionMaxBodyBytes(n)
}

// WithMaxMultipartFormBytes sets the maximum in-memory size for multipart
// form data.
//
// Takes n (int64) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithMaxMultipartFormBytes(n int64) Option {
	return bootstrap.WithMaxMultipartFormBytes(n)
}

// WithDefaultMaxSSEDuration sets the maximum lifetime for SSE connections
// that do not specify their own limit. Zero means unlimited.
//
// Takes d (time.Duration) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithDefaultMaxSSEDuration(d time.Duration) Option {
	return bootstrap.WithDefaultMaxSSEDuration(d)
}

// WithAutoNextPort enables automatic port selection for the main HTTP server
// when the configured port is already in use.
//
// Takes enabled (bool) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithAutoNextPort(enabled bool) Option {
	return bootstrap.WithAutoNextPort(enabled)
}

// WithEncryptionKey sets the base64-encoded 32-byte encryption key for the
// default local AES-GCM crypto provider.
//
// Takes key (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithEncryptionKey(key string) Option {
	return bootstrap.WithEncryptionKey(key)
}

// WithDataKeyCacheTTL configures how long decrypted data keys are cached for
// KMS providers. Zero disables caching.
//
// Takes d (time.Duration) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithDataKeyCacheTTL(d time.Duration) Option {
	return bootstrap.WithDataKeyCacheTTL(d)
}

// WithDataKeyCacheMaxSize sets the maximum number of cached data keys.
//
// Takes n (int) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithDataKeyCacheMaxSize(n int) Option {
	return bootstrap.WithDataKeyCacheMaxSize(n)
}

// WithSecurityHeaders sets the HTTP security header policy. Pass a fully
// populated SecurityHeadersConfig.
//
// Takes headers (SecurityHeadersConfig) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithSecurityHeaders(headers SecurityHeadersConfig) Option {
	return bootstrap.WithSecurityHeaders(headers)
}

// WithCookieSecurity sets the secure cookie defaults applied to all cookies
// the framework writes.
//
// Takes cookies (CookieSecurityConfig) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithCookieSecurity(cookies CookieSecurityConfig) Option {
	return bootstrap.WithCookieSecurity(cookies)
}

// WithRateLimit sets the request rate limiting configuration. Disabled by
// default; pass Enabled=true to activate.
//
// Takes rl (RateLimitConfig) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithRateLimit(rl RateLimitConfig) Option {
	return bootstrap.WithRateLimit(rl)
}

// WithSandbox configures filesystem sandboxing for Piko internals.
//
// Takes s (SandboxConfig) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithSandbox(s SandboxConfig) Option {
	return bootstrap.WithSandbox(s)
}

// WithReporting configures the Reporting-Endpoints HTTP header.
//
// Takes r (ReportingConfig) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithReporting(r ReportingConfig) Option {
	return bootstrap.WithReporting(r)
}

// WithCaptcha sets the per-deployment captcha settings (site key, secret,
// score threshold).
//
// Takes opts (CaptchaOptions) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithCaptcha(opts CaptchaOptions) Option {
	return bootstrap.WithCaptcha(opts)
}

// WithAWSKMS configures AWS Key Management Service settings.
//
// Takes k (AWSKMSConfig) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithAWSKMS(k AWSKMSConfig) Option {
	return bootstrap.WithAWSKMS(k)
}

// WithGCPKMS configures Google Cloud KMS settings.
//
// Takes k (GCPKMSConfig) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithGCPKMS(k GCPKMSConfig) Option {
	return bootstrap.WithGCPKMS(k)
}

// WithDeprecatedKeyIDs lists key IDs that remain valid for decryption but
// are not used for new encryption.
//
// Takes ids (...string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithDeprecatedKeyIDs(ids ...string) Option {
	return bootstrap.WithDeprecatedKeyIDs(ids...)
}

// WithLogLevel sets the application log level. Accepts standard slog level
// strings: "debug", "info", "warn", "error".
//
// PIKO_LOG_LEVEL environment variable, when set, overrides this option for
// the bootstrap logger before any options apply.
//
// Takes level (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithLogLevel(level string) Option {
	return bootstrap.WithLogLevel(level)
}

// WithLogger replaces the entire logger configuration.
//
// Takes cfg (logger_dto.Config) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithLogger(cfg logger_dto.Config) Option {
	return bootstrap.WithLogger(cfg)
}

// WithDatabaseDriver selects the database backend.
// Valid values: "sqlite" (default), "postgres", "d1".
//
// Takes driver (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithDatabaseDriver(driver string) Option {
	return bootstrap.WithDatabaseDriver(driver)
}

// WithPostgresURL sets the PostgreSQL connection URL.
//
// Takes url (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithPostgresURL(url string) Option {
	return bootstrap.WithPostgresURL(url)
}

// WithPostgresMaxConns sets the maximum number of connections in the
// PostgreSQL pool.
//
// Takes n (int32) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithPostgresMaxConns(n int32) Option {
	return bootstrap.WithPostgresMaxConns(n)
}

// WithPostgresMinConns sets the minimum number of connections kept in the
// PostgreSQL pool.
//
// Takes n (int32) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithPostgresMinConns(n int32) Option {
	return bootstrap.WithPostgresMinConns(n)
}

// WithD1APIToken sets the Cloudflare API token used for D1 access.
//
// Takes token (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithD1APIToken(token string) Option {
	return bootstrap.WithD1APIToken(token)
}

// WithD1AccountID sets the Cloudflare account ID for D1 access.
//
// Takes id (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithD1AccountID(id string) Option {
	return bootstrap.WithD1AccountID(id)
}

// WithD1DatabaseID sets the D1 database UUID.
//
// Takes id (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithD1DatabaseID(id string) Option {
	return bootstrap.WithD1DatabaseID(id)
}

// WithOTLP replaces the entire OpenTelemetry Protocol exporter configuration.
//
// Takes o (OtlpConfig) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithOTLP(o OtlpConfig) Option {
	return bootstrap.WithOTLP(o)
}

// WithOTLPEnabled controls whether OTLP exporting is active.
//
// Takes enabled (bool) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithOTLPEnabled(enabled bool) Option {
	return bootstrap.WithOTLPEnabled(enabled)
}

// WithOTLPEndpoint sets the OTLP collector endpoint.
//
// Takes endpoint (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithOTLPEndpoint(endpoint string) Option {
	return bootstrap.WithOTLPEndpoint(endpoint)
}

// WithOTLPProtocol sets the OTLP transport protocol.
// Valid values: "grpc", "http", "https".
//
// Takes protocol (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithOTLPProtocol(protocol string) Option {
	return bootstrap.WithOTLPProtocol(protocol)
}

// WithOTLPTraceSampleRate sets the fraction of traces to sample (0.0 to 1.0).
//
// Takes rate (float64) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithOTLPTraceSampleRate(rate float64) Option {
	return bootstrap.WithOTLPTraceSampleRate(rate)
}

// WithOTLPHeaders sets the HTTP headers sent with OTLP requests.
//
// Takes headers (map[string]string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithOTLPHeaders(headers map[string]string) Option {
	return bootstrap.WithOTLPHeaders(headers)
}

// WithOTLPInsecureTLS controls whether TLS certificate verification is
// disabled for the OTLP connection.
//
// Takes insecure (bool) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithOTLPInsecureTLS(insecure bool) Option {
	return bootstrap.WithOTLPInsecureTLS(insecure)
}

// WithHealthEnabled controls whether the health probe server starts.
//
// Takes enabled (bool) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithHealthEnabled(enabled bool) Option {
	return bootstrap.WithHealthEnabled(enabled)
}

// WithHealthBindAddress sets the network address to bind the health probe
// server to.
//
// Takes addr (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithHealthBindAddress(addr string) Option {
	return bootstrap.WithHealthBindAddress(addr)
}

// WithHealthMetricsEnabled controls whether the Prometheus metrics endpoint
// is exposed on the health probe server.
//
// Takes enabled (bool) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithHealthMetricsEnabled(enabled bool) Option {
	return bootstrap.WithHealthMetricsEnabled(enabled)
}

// WithHealthMetricsPath sets the URL path for the Prometheus metrics
// endpoint.
//
// Takes path (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithHealthMetricsPath(path string) Option {
	return bootstrap.WithHealthMetricsPath(path)
}

// WithHealthLivePath sets the URL path for the liveness probe endpoint.
//
// Takes path (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithHealthLivePath(path string) Option {
	return bootstrap.WithHealthLivePath(path)
}

// WithHealthReadyPath sets the URL path for the readiness probe endpoint.
//
// Takes path (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithHealthReadyPath(path string) Option {
	return bootstrap.WithHealthReadyPath(path)
}

// WithHealthCheckTimeout sets the maximum time for each individual health
// check.
//
// Takes d (time.Duration) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithHealthCheckTimeout(d time.Duration) Option {
	return bootstrap.WithHealthCheckTimeout(d)
}

// WithHealthAutoNextPort enables automatic port selection for the health
// probe server.
//
// Takes enabled (bool) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithHealthAutoNextPort(enabled bool) Option {
	return bootstrap.WithHealthAutoNextPort(enabled)
}

// WithHealthProbePort sets the port for the health probe server.
//
// Takes port (int) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithHealthProbePort(port int) Option {
	return bootstrap.WithHealthProbePort(port)
}

// WithBaseDir sets the root directory of the website project.
//
// Takes dir (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithBaseDir(dir string) Option {
	return bootstrap.WithBaseDir(dir)
}

// WithComponentsSourceDir sets the directory for .pkc/.sfc component files.
//
// Takes dir (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithComponentsSourceDir(dir string) Option {
	return bootstrap.WithComponentsSourceDir(dir)
}

// WithPagesSourceDir sets the directory for page definition files.
//
// Takes dir (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithPagesSourceDir(dir string) Option {
	return bootstrap.WithPagesSourceDir(dir)
}

// WithPartialsSourceDir sets the directory for partial definition files.
//
// Takes dir (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithPartialsSourceDir(dir string) Option {
	return bootstrap.WithPartialsSourceDir(dir)
}

// WithEmailsSourceDir sets the directory for email template source files.
//
// Takes dir (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithEmailsSourceDir(dir string) Option {
	return bootstrap.WithEmailsSourceDir(dir)
}

// WithPdfsSourceDir sets the directory for PDF template source files.
//
// Takes dir (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithPdfsSourceDir(dir string) Option {
	return bootstrap.WithPdfsSourceDir(dir)
}

// WithE2ESourceDir sets the directory for E2E test pages and partials.
//
// Takes dir (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithE2ESourceDir(dir string) Option {
	return bootstrap.WithE2ESourceDir(dir)
}

// WithAssetsSourceDir sets the directory for asset files.
//
// Takes dir (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithAssetsSourceDir(dir string) Option {
	return bootstrap.WithAssetsSourceDir(dir)
}

// WithI18nSourceDir sets the directory containing locale and translation
// JSON files.
//
// Takes dir (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithI18nSourceDir(dir string) Option {
	return bootstrap.WithI18nSourceDir(dir)
}

// WithBaseServePath sets the URL path prefix for serving pages.
//
// Takes path (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithBaseServePath(path string) Option {
	return bootstrap.WithBaseServePath(path)
}

// WithPartialServePath sets the URL path prefix for serving partials.
//
// Takes path (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithPartialServePath(path string) Option {
	return bootstrap.WithPartialServePath(path)
}

// WithActionServePath sets the URL path prefix for server actions.
//
// Takes path (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithActionServePath(path string) Option {
	return bootstrap.WithActionServePath(path)
}

// WithLibServePath sets the URL path prefix for serving internal library
// files.
//
// Takes path (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithLibServePath(path string) Option {
	return bootstrap.WithLibServePath(path)
}

// WithDistServePath sets the URL path prefix for serving frontend
// distribution files.
//
// Takes path (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithDistServePath(path string) Option {
	return bootstrap.WithDistServePath(path)
}

// WithArtefactServePath sets the URL path prefix for serving compiled
// assets.
//
// Takes path (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithArtefactServePath(path string) Option {
	return bootstrap.WithArtefactServePath(path)
}

// WithDefaultServeMode selects the default page serving mode.
// Valid values: "preview" (dynamic) or "render" (static).
//
// Takes mode (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithDefaultServeMode(mode string) Option {
	return bootstrap.WithDefaultServeMode(mode)
}

// WithStoragePresign replaces the entire presigned URL configuration.
//
// Takes p (StoragePresignConfig) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithStoragePresign(p StoragePresignConfig) Option {
	return bootstrap.WithStoragePresign(p)
}

// WithStoragePresignSecret sets the HMAC secret for signing presign tokens.
//
// Takes secret (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithStoragePresignSecret(secret string) Option {
	return bootstrap.WithStoragePresignSecret(secret)
}

// WithStoragePresignDefaultExpiry sets the default validity duration for
// presigned URLs.
//
// Takes d (time.Duration) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithStoragePresignDefaultExpiry(d time.Duration) Option {
	return bootstrap.WithStoragePresignDefaultExpiry(d)
}

// WithStoragePresignMaxExpiry sets the maximum validity duration for
// presigned URLs.
//
// Takes d (time.Duration) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithStoragePresignMaxExpiry(d time.Duration) Option {
	return bootstrap.WithStoragePresignMaxExpiry(d)
}

// WithStoragePresignDefaultMaxSize sets the default maximum upload size in
// bytes.
//
// Takes size (int64) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithStoragePresignDefaultMaxSize(size int64) Option {
	return bootstrap.WithStoragePresignDefaultMaxSize(size)
}

// WithStoragePresignMaxMaxSize sets the absolute maximum upload size in
// bytes.
//
// Takes size (int64) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithStoragePresignMaxMaxSize(size int64) Option {
	return bootstrap.WithStoragePresignMaxMaxSize(size)
}

// WithStoragePresignRateLimit sets the per-IP rate limit for presigned upload
// requests.
//
// Takes rpm (int) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithStoragePresignRateLimit(rpm int) Option {
	return bootstrap.WithStoragePresignRateLimit(rpm)
}

// WithI18nDefaultLocale sets the default locale used for internationalisation.
//
// Takes locale (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithI18nDefaultLocale(locale string) Option {
	return bootstrap.WithI18nDefaultLocale(locale)
}

// WithSRI controls whether Subresource Integrity (SRI) hashes are added to
// script and link tags in rendered HTML. Enabled by default; disable for
// development environments where assets change frequently.
//
// Takes enabled (bool) which controls whether integrity attributes are emitted.
//
// Returns Option which configures the SRI setting.
func WithSRI(enabled bool) Option {
	return bootstrap.WithSRI(enabled)
}

// WithExperimentalPrerendering toggles prerendering of pages during build.
//
// When enabled, eligible pages are rendered to HTML at build time rather than
// per request. Email templates are never prerendered regardless of this
// setting.
//
// Takes enabled (bool) which specifies whether prerendering is active.
//
// Returns Option which the bootstrap consumes when applied.
func WithExperimentalPrerendering(enabled bool) Option {
	return bootstrap.WithExperimentalPrerendering(enabled)
}

// WithExperimentalCommentStripping toggles stripping of HTML comments from generated output.
//
// When enabled, <!-- ... --> comments are omitted from rendered HTML.
//
// Takes enabled (bool) which specifies whether comment stripping is active.
//
// Returns Option which the bootstrap consumes when applied.
func WithExperimentalCommentStripping(enabled bool) Option {
	return bootstrap.WithExperimentalCommentStripping(enabled)
}

// WithExperimentalDwarfLineDirectives toggles DWARF-compatible line
// directives in generated Go code.
//
// When enabled, the generator emits "//line file:line" (no space) which the
// Go compiler embeds in DWARF debug info, allowing debuggers like Delve to
// map breakpoints back to .pk source files. Disabled by default; the
// generator emits a plain "// line file:line" comment.
//
// Takes enabled (bool) which specifies whether DWARF line directives are
// active.
//
// Returns Option which the bootstrap consumes when applied.
func WithExperimentalDwarfLineDirectives(enabled bool) Option {
	return bootstrap.WithExperimentalDwarfLineDirectives(enabled)
}
