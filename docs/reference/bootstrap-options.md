---
title: Bootstrap options
description: Every With* option that customises a Piko server at construction time.
nav:
  sidebar:
    section: "reference"
    subsection: "bootstrap"
    order: 10
---

# Bootstrap options

`piko.New` accepts a variadic list of `piko.Option` values. Each option configures one aspect of the server. Examples include provider selection for storage or caching, frontend module registration, TLS handling, and metrics transport. This page groups every shipped option by concern. Source of truth: [`options.go`](https://github.com/piko-sh/piko/blob/master/options.go).

<p align="center">
  <img src="../diagrams/bootstrap-wiring.svg"
       alt="The piko-New constructor at the centre with eight With-options radiating out for cache, storage, email, LLM, image, notification, crypto, and captcha providers. Each option registers a named adapter, and a matching WithDefault option marks one provider as the default for unqualified calls."
       width="600"/>
</p>

For the ports-and-adapters mental model that motivates every `With*Provider` option, see [About the hexagonal architecture](../explanation/about-the-hexagonal-architecture.md).

Piko reads no configuration files and no environment variables apart from `PIKO_LOG_LEVEL` (which raises the bootstrap logger before any options apply). Every other knob is one of the `With*` options listed below. See [Configuration philosophy](../explanation/about-configuration.md) for the rationale.

## Server

| Option | Purpose |
|---|---|
| `WithPort(port int)` | HTTP listen port. Defaults to `8080`. |
| `WithAutoNextPort(enabled bool)` | Pick the next available port if another process is already bound. |
| `WithPublicDomain(domain string)` | Public hostname used to build absolute URLs. |
| `WithForceHTTPS(enabled bool)` | Redirect HTTP requests to HTTPS. |
| `WithRequestTimeout(d time.Duration)` | Per-request timeout. |
| `WithMaxConcurrentRequests(n int)` | Cap on in-flight HTTP requests. |
| `WithActionMaxBodyBytes(n int64)` | Maximum body size for server actions. |
| `WithMaxMultipartFormBytes(n int64)` | Maximum multipart-form body size. |
| `WithDefaultMaxSSEDuration(d time.Duration)` | Default SSE stream lifetime. |
| `WithTLS(opts ...TLSOption)` | Enables HTTPS. See [TLS](#tls-options). |
| `WithWatchMode(enabled bool)` | Enables the dev watcher. Normally set automatically by `piko dev`. |
| `WithE2EMode(enabled bool)` | Enables test hooks and fixed timestamps for E2E tests. |
| `WithDefaultServeMode(mode string)` | Initial serve mode (`dev`, `prod`). |
| `WithStartupBanner(enabled bool)` | Toggles the ASCII banner emitted on start-up. |
| `WithShutdownDrainDelay(delay time.Duration)` | Adds a grace period before the server stops draining connections. |
| `WithIAmACatPerson()` | Swap the large pixel-art mascot in the startup banner for the small ASCII art version. |

### Logger

| Option | Purpose |
|---|---|
| `WithLogLevel(level string)` | Set the log level (`trace`, `debug`, `info`, `warn`, `error`). Overrides the `PIKO_LOG_LEVEL` env var. |
| `WithLogger(cfg logger_dto.Config)` | Replace the default logger configuration wholesale. |

`PIKO_LOG_LEVEL` is the single environment variable Piko itself consults. Piko reads it once in `piko.New` so you can raise verbosity without rebuilding the binary. An explicit `WithLogLevel(...)` always wins.

### TLS options

Passed inside `WithTLS(...)`:

| Option | Purpose |
|---|---|
| `WithTLSCertFile(path)` | Path to the PEM-encoded certificate chain. |
| `WithTLSKeyFile(path)` | Path to the PEM-encoded private key. |
| `WithTLSClientCA(path)` | Enables mTLS; path to the CA used to validate client certs. |
| `WithTLSClientAuth(mode)` | Client-cert policy. One of `"none"`, `"request"`, `"require"`, `"verify"`, `"require_and_verify"` (literal `_`, not hyphen). |
| `WithTLSMinVersion(version)` | `"1.2"` or `"1.3"`. |
| `WithTLSHotReload(enabled bool)` | Watches cert files and reloads on change. |
| `WithTLSRedirectHTTP(port int)` | Redirects HTTP requests on the given port to HTTPS. |

## Dependency injection

Piko's hexagonal architecture lets a project substitute any backend at construction. Most ports offer three options. The first registers a named provider, the second marks one as the default, and the third overrides the service wrapper entirely.

### Cache

| Option | Purpose |
|---|---|
| `WithCacheProvider(name, provider)` | Register a named cache backend. |
| `WithDefaultCacheProvider(name)` | Select the default backend by name. |
| `WithCacheService(service)` | Replace the entire cache service. |
| `WithMemoryRegistryCache()` | In-memory metadata cache for the registry. |
| `WithJSONTypeInspectorCache()` | Cache the JSON type inspector to avoid re-reflection per request. |

### Crypto

| Option | Purpose |
|---|---|
| `WithCryptoProvider(name, provider)` | Register a named encryption provider. |
| `WithDefaultCryptoProvider(name)` | Select the default crypto backend. |
| `WithCryptoService(service)` | Replace the entire crypto service. |
| `WithEncryptionKey(key string)` | Base64-encoded 32-byte key for the default local AES-GCM provider. |
| `WithDataKeyCacheTTL(d time.Duration)` | How long decrypted KMS data keys stay cached. Zero disables the cache. |
| `WithDataKeyCacheMaxSize(n int)` | Maximum number of cached KMS data keys. |
| `WithAWSKMS(k AWSKMSConfig)` | AWS Key Management Service configuration. |
| `WithGCPKMS(k GCPKMSConfig)` | Google Cloud KMS configuration. |
| `WithDeprecatedKeyIDs(ids ...string)` | Key IDs that remain valid for decryption but are no longer used to encrypt. |

### Email

| Option | Purpose |
|---|---|
| `WithEmailProvider(name, provider)` | Register a named email transport (SES, SendGrid, SMTP, etc.). |
| `WithDefaultEmailProvider(name)` | Select the default transport. |
| `WithEmailService(service)` | Replace the entire email service. |
| `WithEmailDispatcher(config)` | Configure the queue-backed dispatcher. |
| `WithEmailDeadLetterQueue(dlq)` | Route failed emails to a dead-letter queue. |

### Storage

| Option | Purpose |
|---|---|
| `WithStorageProvider(name, provider)` | Register a named storage provider (S3, R2, GCS, etc.). |
| `WithDefaultStorageProvider(name)` | Select the default provider. |
| `WithSystemStorageProvider(provider)` | Register the internal-system storage backend. |
| `WithStorageService(service)` | Replace the entire storage service. |
| `WithStorageDispatcher(config)` | Configure the storage dispatcher. |
| `WithStoragePresignBaseURL(url)` | Base URL for generated presigned URLs. |
| `WithStoragePublicBaseURL(url)` | Base URL for public download links. |

### LLM and embedding

| Option | Purpose |
|---|---|
| `WithLLMProvider(name, provider)` | Register a named LLM provider (OpenAI, Anthropic, local). |
| `WithDefaultLLMProvider(name)` | Select the default LLM. |
| `WithLLMService(service)` | Replace the entire LLM service. |
| `WithEmbeddingProvider(name, provider)` | Register a named embedding backend. |
| `WithDefaultEmbeddingProvider(name)` | Select the default embedding backend. |

### Captcha

| Option | Purpose |
|---|---|
| `WithCaptchaProvider(name, provider)` | Register a named captcha provider (Turnstile, hCaptcha, etc.). |
| `WithDefaultCaptchaProvider(name)` | Select the default. |

### Image and video

| Option | Purpose |
|---|---|
| `WithImageProvider(name, provider)` | Register a named image transformer (vips, imaginary, etc.). |
| `WithDefaultImageProvider(name)` | Select the default image backend. |
| `WithImageService(service)` | Replace the entire image service. |
| `WithImage(config)` | Register additional image profiles. |
| `WithVideoProvider(name, provider)` | Register a named video transcoder (ffmpeg, etc.). |
| `WithDefaultVideoProvider(name)` | Select the default video backend. |
| `WithVideoService(service)` | Replace the entire video service. |

### Notifications and events

| Option | Purpose |
|---|---|
| `WithNotificationProvider(name, provider)` | Register a named notification provider. |
| `WithDefaultNotificationProvider(name)` | Select the default notification backend. |
| `WithEventsProvider(provider)` | Provider for the internal event bus. |
| `WithEventBus(bus)` | Replace the event bus implementation. |

### Other services

| Option | Purpose |
|---|---|
| `WithRegistryService(service)` | Replace the route registry. |
| `WithCapabilityService(service)` | Replace the capability service. |
| `WithOrchestratorService(service)` | Replace the background-task orchestrator. |
| `WithI18nService(service)` | Replace the i18n service. |
| `WithHighlighter(h)` | Syntax highlighter for code blocks in markdown. |
| `WithMarkdownParser(parser)` | Markdown parser implementation. |
| `WithPMLTransformer(t)` | `Piko Markup Language` transformer. |

## Security

| Option | Purpose |
|---|---|
| `WithCSRFSecret(key []byte)` | 32-byte key used to sign CSRF tokens. |
| `WithCSRFTokenMaxAge(d)` | Token validity period. |
| `WithCSRFSecFetchSiteEnforcement(enabled)` | Enforce `Sec-Fetch-Site` header alongside the token. |
| `WithConfigResolvers(resolvers...)` | Register resolvers that populate `Secret[T]` from external sources. |
| `WithAuthProvider(provider)` | Authentication provider. |
| `WithAuthGuard(config)` | Route-level authorisation rules. |
| `WithTrustedProxies(cidrs...)` | CIDRs of reverse proxies whose `X-Forwarded-*` headers Piko should honour. |
| `WithCloudflareEnabled(enabled)` | Trusts Cloudflare's CF-Connecting-IP header. |
| `WithSecurityHeaders(headers SecurityHeadersConfig)` | HTTP security header policy (HSTS, X-Frame-Options, Permissions-Policy, etc.). |
| `WithCookieSecurity(cookies CookieSecurityConfig)` | Default `Secure`, `HttpOnly`, and `SameSite` flags applied to cookies the framework writes. |
| `WithRateLimit(rl RateLimitConfig)` | Full rate-limiter configuration (window, burst, per-IP/user, exclusions). |
| `WithRateLimitEnabled(enabled)` | Toggle the rate limiter without supplying a full config. |
| `WithSandbox(s SandboxConfig)` | Filesystem sandbox for Piko internals (allowed roots, deny lists). |
| `WithReporting(r ReportingConfig)` | Configure the `Reporting-Endpoints` HTTP header. |
| `WithCaptcha(opts CaptchaOptions)` | Per-deployment captcha settings (site key, secret, score threshold). |

### `Content Security Policy`

| Option | Purpose |
|---|---|
| `WithCSP(configure)` | Build a CSP programmatically. |
| `WithCSPString(policy)` | Supply a raw CSP string. |
| `WithPikoDefaultCSP()` | Use Piko's recommended default policy. |
| `WithStrictCSP()` | Strict policy (no inline scripts, no eval). |
| `WithRelaxedCSP()` | Permissive policy for legacy content. |
| `WithAPICSP()` | Minimal policy for API-only endpoints. |
| `WithReportingEndpoints(endpoints...)` | CSP violation reporting targets. |
| `WithCrossOriginResourcePolicy(policy)` | `Cross-Origin-Resource-Policy` header value. |

## Frontend

| Option | Purpose |
|---|---|
| `WithFrontendModule(module, config...)` | Enable a built-in frontend module (`ModuleAnalytics`, `ModuleToasts`, `ModuleModals`, etc.). |
| `WithCustomFrontendModule(name, content, config...)` | Ship a custom JS bundle as a first-class module. |
| `WithComponents(defs...)` | Register external `.pkc` components. |
| `WithDevWidget()` | Show the dev-mode overlay. |
| `WithDevHotreload()` | Enable hot reload in dev mode. |

## Performance and caching

| Option | Purpose |
|---|---|
| `WithCSSTreeShaking()` | Eliminate unused CSS at build time. |
| `WithCSSTreeShakingSafelist(classes...)` | Classes to exempt from tree-shaking. |
| `WithCSSReset(opts...)` | Apply the CSS reset. |
| `WithCSSResetComplete()` | Use the full reset variant. |
| `WithCSSResetPKOverride(css)` | Override the reset with project-specific CSS. |
| `WithAutoMemoryLimit(provider)` | Set a dynamic `GOMEMLIMIT` based on the environment. |
| `WithRegistryMetadataCacheConfig(config)` | Tune the metadata cache's TTL and size. |

## Monitoring

| Option | Purpose |
|---|---|
| `WithMonitoring(opts...)` | Enables the gRPC monitoring endpoint. |
| `WithMonitoringAddress(addr)` | Full address (host:port). |
| `WithMonitoringBindAddress(addr)` | Bind address override. |
| `WithMonitoringAutoNextPort(enabled)` | Pick the next available port if another process already holds the primary. |
| `WithMonitoringTLS(opts...)` | TLS for the monitoring endpoint. See [monitoring TLS options](#monitoring-tls-options). |
| `WithMonitoringTransport(factory)` | Transport factory (OTLP, Prometheus, etc.). |
| `WithMonitoringOtelFactories(factories)` | OpenTelemetry service factories. |
| `WithMonitoringProfiling()` | Attach profiling handlers to the monitoring endpoint. |
| `WithMonitoringWatchdog(opts...)` | Enable the runtime watchdog that monitors heap, goroutines, RSS, FDs, and scheduler latency. See [watchdog options](#watchdog-options). |
| `WithWatchdogNotifier(notifier)` | Set the notification delivery mechanism for watchdog events. |
| `WithWatchdogProfileUploader(uploader)` | Set the remote storage backend for watchdog profile uploads. |
| `WithProfiling(opts...)` | Enables pprof HTTP server (separate from monitoring). |
| `WithGeneratorProfiling(opts...)` | Profile the project generator during `go run ./cmd/generator/main.go all`. See [generator profiling options](#generator-profiling-options). |
| `WithMetricsExporter(exporter)` | Custom metrics exporter. |

### Monitoring TLS options

Passed inside `WithMonitoringTLS(...)`:

| Option | Purpose |
|---|---|
| `WithMonitoringTLSCertFile(path string)` | Certificate file for the monitoring server. |
| `WithMonitoringTLSKeyFile(path string)` | Private key file for the monitoring server. |
| `WithMonitoringTLSClientCA(path string)` | Client CA file enabling mTLS. |
| `WithMonitoringTLSClientAuth(authType string)` | Client certificate verification mode (for example `"require_and_verify"`). |
| `WithMonitoringTLSMinVersion(version string)` | Minimum TLS version (`"1.2"` or `"1.3"`). |
| `WithMonitoringTLSHotReload(enabled bool)` | Enable automatic certificate hot-reload. |

### Profiling options

Passed inside `WithProfiling(...)`:

| Option | Purpose |
|---|---|
| `WithProfilingPort(port int)` | Port for the pprof HTTP server. |
| `WithProfilingBindAddress(addr string)` | Bind address. |
| `WithProfilingBlockRate(rate int)` | pprof block profiler rate. |
| `WithProfilingMutexFraction(fraction int)` | Mutex profiler fraction. |
| `WithProfilingMemProfileRate(rate int)` | Memory profile sampling rate. |
| `WithProfilingRollingTrace()` | Enable rolling execution traces. |
| `WithProfilingRollingTraceMinAge(d time.Duration)` | Minimum age before Piko retains a trace. |
| `WithProfilingRollingTraceMaxBytes(n uint64)` | Max size of the rolling trace buffer. |

### Watchdog options

Passed inside `WithMonitoringWatchdog(...)`:

| Option | Purpose |
|---|---|
| `WithWatchdogHeapThresholdPercent(percent float64)` | Heap threshold as a fraction of `GOMEMLIMIT`. |
| `WithWatchdogHeapThresholdBytes(bytes uint64)` | Absolute heap threshold in bytes. |
| `WithWatchdogGoroutineThreshold(threshold int)` | Goroutine count that triggers a profile capture. |
| `WithWatchdogCheckInterval(interval time.Duration)` | How often the watchdog evaluates runtime state. |
| `WithWatchdogCooldown(duration time.Duration)` | Minimum duration between consecutive captures. |
| `WithWatchdogMaxProfilesPerType(count int)` | Maximum number of stored profiles per type. |
| `WithWatchdogProfileDirectory(directory string)` | Local directory for profile storage. |
| `WithWatchdogDeltaProfiling()` | Store a baseline heap profile alongside each capture for delta analysis. |
| `WithWatchdogRSSThresholdPercent(percent float64)` | Fraction of the cgroup memory limit that triggers a capture. |
| `WithWatchdogFDPressureThresholdPercent(percent float64)` | Fraction of the soft file-descriptor limit that triggers a capture. |
| `WithWatchdogSchedulerLatencyP99Threshold(threshold time.Duration)` | p99 scheduler latency threshold. |
| `WithWatchdogMaxWarningsPerWindow(count int)` | Maximum warning-only events per window. |
| `WithWatchdogContinuousProfiling()` | Enable the continuous-profiling loop. |
| `WithWatchdogContinuousProfilingInterval(interval time.Duration)` | Interval between routine continuous-profiling captures. |
| `WithWatchdogContinuousProfilingTypes(types ...string)` | Profile types captured each cycle. |
| `WithWatchdogContinuousProfilingRetention(count int)` | Maximum number of continuous-profiling artefacts retained. |
| `WithWatchdogContinuousProfilingNotify()` | Notify on each continuous-profiling capture. |
| `WithWatchdogContentionDiagnosticWindow(window time.Duration)` | Diagnostic window for contention events. |
| `WithWatchdogContentionDiagnosticAutoFire()` | Auto-fire contention diagnostics on threshold breach. |
| `WithWatchdogContentionDiagnosticBlockProfileRate(rate int)` | Block profile rate during contention diagnostics. |
| `WithWatchdogContentionDiagnosticMutexProfileFraction(fraction int)` | Mutex profile fraction during contention diagnostics. |

### Generator profiling options

Passed inside `WithGeneratorProfiling(...)`:

| Option | Purpose |
|---|---|
| `WithGeneratorProfilingOutputDir(directory string)` | Directory for captured generator profiles. |
| `WithGeneratorProfilingBlockRate(rate int)` | Block profiling rate during generator runs. |
| `WithGeneratorProfilingMutexFraction(fraction int)` | Mutex profiling fraction during generator runs. |
| `WithGeneratorProfilingMemProfileRate(rate int)` | Memory profiling sample rate during generator runs. |

### Diagnostics

| Option | Purpose |
|---|---|
| `WithDiagnosticDirectory(directory string)` | Single root directory for all runtime diagnostic artefacts (profiles, traces, crash dumps). |
| `WithCrashOutput(path string)` | Mirror Go's crash output (`runtime/debug.SetCrashOutput`) to the given path. |
| `WithCrashTraceback(level string)` | Set `GOTRACEBACK` level. |

## Health

| Option | Purpose |
|---|---|
| `WithCustomHealthProbe(probe)` | Register a custom health probe. |
| `WithHealthTLS(opts...)` | TLS for the health endpoint. |
| `WithHealthTLSCertFile(path)` | Cert path. |
| `WithHealthTLSKeyFile(path)` | Key path. |
| `WithHealthTLSMinVersion(version)` | `"1.2"` or `"1.3"`. |

## SEO and assets

| Option | Purpose |
|---|---|
| `WithSEO(config)` | Generate `sitemap.xml` and `robots.txt`. |
| `WithAssets(config)` | Image profiles, breakpoints, densities. |
| `WithWebsiteConfig(config piko.WebsiteConfig)` | Theme colours, fonts, favicons, supported locales, i18n strategy, site name. The single source of website-level metadata. |

## Analytics

| Option | Purpose |
|---|---|
| `WithBackendAnalytics(collectors...)` | Register backend analytics collectors (see [analytics reference](analytics-api.md)). |

## Database

| Option | Purpose |
|---|---|
| `WithDatabase(name, registration)` | Register a database connection by name. |
| `WithDatabaseDriver(driver string)` | Default driver name (`postgres`, `d1`, and so on). |
| `WithPostgresURL(url string)` | Postgres DSN. |
| `WithPostgresMaxConns(n int32)` | Maximum pool size. |
| `WithPostgresMinConns(n int32)` | Minimum pool size. |
| `WithD1APIToken(token string)` | Cloudflare D1 API token. |
| `WithD1AccountID(id string)` | Cloudflare account ID. |
| `WithD1DatabaseID(id string)` | Cloudflare D1 database ID. |

## Storage URLs and presigning

| Option | Purpose |
|---|---|
| `WithStoragePublicBaseURL(url)` | Base URL for public download links. |
| `WithStoragePresignBaseURL(url)` | Base URL for generated presigned URLs. |
| `WithStoragePresign(cfg piko.StoragePresignConfig)` | Group option configuring presign secret, expiry, max body size, and rate limits. |
| `WithStoragePresignSecret(secret string)` | HMAC secret used to sign presigned URLs. |
| `WithStoragePresignDefaultExpiry(d time.Duration)` | Default expiry for presigned URLs. |
| `WithStoragePresignMaxExpiry(d time.Duration)` | Maximum allowed expiry. |
| `WithStoragePresignDefaultMaxSize(bytes int64)` | Default upload size cap. |
| `WithStoragePresignMaxMaxSize(bytes int64)` | Hard ceiling for upload size. |
| `WithStoragePresignRateLimit(n int)` | Per-IP requests per minute for presign issuance. |

## OpenTelemetry (OTLP)

`OTLP` exports replace the old environment-driven configuration. Pass a single `WithOTLP(piko.OtlpConfig{...})` for grouped configuration, or use the per-field options below.

| Option | Purpose |
|---|---|
| `WithOTLP(cfg piko.OtlpConfig)` | Group option configuring all OTLP fields. |
| `WithOTLPEnabled(enabled bool)` | Toggle OTLP export. |
| `WithOTLPEndpoint(url string)` | Collector endpoint. |
| `WithOTLPProtocol(protocol string)` | `grpc` or `http/protobuf`. |
| `WithOTLPTraceSampleRate(rate float64)` | Trace sampling rate (0.0-1.0). |
| `WithOTLPHeaders(headers map[string]string)` | Additional gRPC/HTTP headers. |
| `WithOTLPInsecureTLS(enabled bool)` | Skip server cert verification (test environments only). |

## Health probes

| Option | Purpose |
|---|---|
| `WithHealthEnabled(enabled bool)` | Toggle the health probe HTTP server. |
| `WithHealthProbePort(port int)` | Health server port. |
| `WithHealthBindAddress(addr string)` | Bind address override. |
| `WithHealthMetricsEnabled(enabled bool)` | Expose Prometheus metrics on the health server. |
| `WithHealthMetricsPath(path string)` | Path for the metrics endpoint. |
| `WithHealthLivePath(path string)` | Liveness probe path. |
| `WithHealthReadyPath(path string)` | Readiness probe path. |
| `WithHealthCheckTimeout(d time.Duration)` | Per-probe timeout. |
| `WithHealthAutoNextPort(enabled bool)` | Pick the next available port if another process holds the primary. |

## Source paths

| Option | Purpose |
|---|---|
| `WithBaseDir(path string)` | Project root directory. |
| `WithPagesSourceDir(path string)` | Pages directory (default `pages/`). |
| `WithPartialsSourceDir(path string)` | Partials directory. |
| `WithComponentsSourceDir(path string)` | Components directory. |
| `WithEmailsSourceDir(path string)` | Email templates directory. |
| `WithPdfsSourceDir(path string)` | PDF templates directory. |
| `WithE2ESourceDir(path string)` | E2E tests directory. |
| `WithAssetsSourceDir(path string)` | Static assets directory. |
| `WithI18nSourceDir(path string)` | Locale files directory. |
| `WithBaseServePath(path string)` | URL prefix for routed pages. |
| `WithPartialServePath(path string)` | URL prefix for partial responses. |
| `WithActionServePath(path string)` | URL prefix for server actions. |
| `WithLibServePath(path string)` | URL prefix for the runtime library. |
| `WithDistServePath(path string)` | URL prefix for the build output. |
| `WithArtefactServePath(path string)` | URL prefix for build artefacts. |

## I18n

| Option | Purpose |
|---|---|
| `WithI18nDefaultLocale(locale string)` | Override the default locale (also settable via `WithWebsiteConfig`). |

## Miscellaneous

| Option | Purpose |
|---|---|
| `WithValidator(v)` | Override the struct validator used for action parameters. |
| `WithJSONProvider(provider)` | Override the JSON implementation. |
| `WithStandardLoader(enabled bool)` | Fallback type inspector; slower but stable. |

## Server-instance methods (not options)

The following are methods on `*SSRServer` returned by `piko.New(...)`, not `piko.Option` values. Call them on the server before `Run`:

| Method | Purpose |
|---|---|
| `(*SSRServer).WithSymbols(symbols templater_domain.SymbolExports)` | Register custom Go symbols for interpreted mode. See [runtime symbols reference](runtime-symbols.md#register-custom-symbols). |
| `(*SSRServer).WithInterpreterProvider(provider)` | Override the interpreter for `dev-i` mode. See [runtime symbols reference](runtime-symbols.md#register-custom-symbols). |
| `(*SSRServer).RegisterLifecycle(component LifecycleComponent)` | Register a component for managed startup and shutdown. See [lifecycle API reference](lifecycle-api.md). |

## See also

- [Configuration philosophy](../explanation/about-configuration.md) for the reasoning behind code-only configuration.
- [How to secrets](../how-to/secrets.md) for `WithConfigResolvers`.
- [How to health checks](../how-to/health-checks.md) for `WithCustomHealthProbe`.
- [How to analytics](../how-to/analytics.md) for `WithBackendAnalytics`.
- [Secrets API reference](secrets-api.md).
- [Lifecycle API reference](lifecycle-api.md).
- [Analytics API reference](analytics-api.md).

Source file: [`options.go`](https://github.com/piko-sh/piko/blob/master/options.go).
