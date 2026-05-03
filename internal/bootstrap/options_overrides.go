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
	"strconv"
	"time"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/logger/logger_dto"
)

// CaptchaOptions groups the captcha provider's per-deployment settings: site
// key, secret key, and score threshold. The provider implementation itself is
// selected via WithDefaultCaptchaProvider and registered via
// WithCaptchaProvider.
type CaptchaOptions struct {
	// SiteKey is the public site key issued by the captcha provider.
	SiteKey string

	// SecretKey is the server-side verification secret.
	SecretKey string

	// ScoreThreshold is the minimum acceptable score for score-based providers
	// such as reCAPTCHA v3 (range 0.0 to 1.0). Zero means use provider default.
	ScoreThreshold float64
}

// WithPublicDomain sets the public domain used for CORS allowed origins and
// absolute URLs. Empty string allows all origins.
//
// Takes domain (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithPublicDomain(domain string) Option {
	return func(c *Container) {
		c.ensureOverrides().Network.PublicDomain = &domain
	}
}

// WithForceHTTPS enables redirection from HTTP to HTTPS.
//
// Takes enabled (bool) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithForceHTTPS(enabled bool) Option {
	return func(c *Container) {
		c.ensureOverrides().Network.ForceHTTPS = &enabled
	}
}

// WithRequestTimeout sets the maximum duration for dynamic HTTP requests
// (pages, partials, actions). Zero disables the timeout middleware.
//
// Takes d (time.Duration) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithRequestTimeout(d time.Duration) Option {
	return func(c *Container) {
		c.ensureOverrides().Network.RequestTimeoutSeconds = new(int(d.Seconds()))
	}
}

// WithMaxConcurrentRequests sets the maximum number of in-flight requests the
// server will process simultaneously. Zero disables the concurrency limit.
//
// Takes n (int) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithMaxConcurrentRequests(n int) Option {
	return func(c *Container) {
		c.ensureOverrides().Network.MaxConcurrentRequests = &n
	}
}

// WithActionMaxBodyBytes sets the maximum size in bytes for action request
// bodies.
//
// Takes n (int64) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithActionMaxBodyBytes(n int64) Option {
	return func(c *Container) {
		c.ensureOverrides().Network.ActionMaxBodyBytes = &n
	}
}

// WithMaxMultipartFormBytes sets the maximum in-memory size for multipart form
// data. Files exceeding this limit are stored in temporary files on disk.
//
// Takes n (int64) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithMaxMultipartFormBytes(n int64) Option {
	return func(c *Container) {
		c.ensureOverrides().Network.MaxMultipartFormBytes = &n
	}
}

// WithDefaultMaxSSEDuration sets the maximum lifetime for SSE connections
// that do not specify their own limit. Zero means unlimited.
//
// Takes d (time.Duration) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithDefaultMaxSSEDuration(d time.Duration) Option {
	return func(c *Container) {
		c.ensureOverrides().Network.DefaultMaxSSEDurationSeconds = new(int(d.Seconds()))
	}
}

// WithAutoNextPort enables automatic port selection when the configured port
// is already in use. Applies to the main HTTP server.
//
// Takes enabled (bool) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithAutoNextPort(enabled bool) Option {
	return func(c *Container) {
		c.ensureOverrides().Network.AutoNextPort = &enabled
	}
}

// WithEncryptionKey sets the base64-encoded 32-byte (256-bit) encryption key
// used by the default local AES-GCM crypto provider. In production prefer
// loading this from a secret manager rather than hard-coding.
//
// Takes key (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithEncryptionKey(key string) Option {
	return func(c *Container) {
		c.ensureOverrides().Security.EncryptionKey = &key
	}
}

// WithDataKeyCacheTTL configures how long decrypted data keys are cached
// for KMS providers. Zero disables caching.
//
// Takes d (time.Duration) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithDataKeyCacheTTL(d time.Duration) Option {
	return func(c *Container) {
		c.ensureOverrides().Security.DataKeyCacheTTL = &d
	}
}

// WithDataKeyCacheMaxSize sets the maximum number of cached data keys.
//
// Takes n (int) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithDataKeyCacheMaxSize(n int) Option {
	return func(c *Container) {
		c.ensureOverrides().Security.DataKeyCacheMaxSize = &n
	}
}

// WithSecurityHeaders sets the HTTP security header policy. Pass a fully
// populated config.SecurityHeadersConfig; unset (nil) fields fall back to
// secure-by-default values.
//
// Takes headers (config.SecurityHeadersConfig) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithSecurityHeaders(headers config.SecurityHeadersConfig) Option {
	return func(c *Container) {
		c.ensureOverrides().Security.Headers = headers
	}
}

// WithCookieSecurity sets the secure cookie defaults applied to all cookies
// the framework writes.
//
// Takes cookies (config.CookieSecurityConfig) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithCookieSecurity(cookies config.CookieSecurityConfig) Option {
	return func(c *Container) {
		c.ensureOverrides().Security.Cookies = cookies
	}
}

// WithRateLimit sets the request rate limiting configuration.
//
// Disabled by default. Pass Enabled=true to activate. When deployed behind
// a reverse proxy, set TrustedProxies to extract real client IPs from
// forwarding headers instead of rate-limiting the proxy itself.
//
// Takes rl (config.RateLimitConfig) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithRateLimit(rl config.RateLimitConfig) Option {
	return func(c *Container) {
		c.ensureOverrides().Security.RateLimit = rl
	}
}

// WithSandbox configures filesystem sandboxing for Piko internals. Sandboxing
// uses Go 1.24's os.Root for kernel-level path traversal protection.
//
// Takes s (config.SandboxConfig) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithSandbox(s config.SandboxConfig) Option {
	return func(c *Container) {
		c.ensureOverrides().Security.Sandbox = s
	}
}

// WithReporting configures the Reporting-Endpoints HTTP header used for CSP
// violation reports and other reporting APIs. Disabled by default.
//
// Takes r (config.ReportingConfig) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithReporting(r config.ReportingConfig) Option {
	return func(c *Container) {
		c.ensureOverrides().Security.Reporting = r
	}
}

// WithCaptcha sets the per-deployment captcha settings (site key, secret,
// score threshold). The provider implementation is selected separately via
// WithDefaultCaptchaProvider.
//
// Takes opts (CaptchaOptions) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithCaptcha(opts CaptchaOptions) Option {
	return func(c *Container) {
		ov := c.ensureOverrides()
		ov.Security.CaptchaSiteKey = &opts.SiteKey
		ov.Security.CaptchaSecretKey = &opts.SecretKey
		if opts.ScoreThreshold > 0 {
			ov.Security.CaptchaScoreThreshold = &opts.ScoreThreshold
		}
	}
}

// WithAWSKMS configures AWS Key Management Service settings. Used only when
// the crypto provider is "aws_kms".
//
// Takes k (config.AWSKMSConfig) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithAWSKMS(k config.AWSKMSConfig) Option {
	return func(c *Container) {
		c.ensureOverrides().Security.AWSKMS = k
	}
}

// WithGCPKMS configures Google Cloud KMS settings. Used only when the crypto
// provider is "gcp_kms".
//
// Takes k (config.GCPKMSConfig) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithGCPKMS(k config.GCPKMSConfig) Option {
	return func(c *Container) {
		c.ensureOverrides().Security.GCPKMS = k
	}
}

// WithDeprecatedKeyIDs lists key IDs that remain valid for decryption but are
// not used for new encryption. Supports gradual key rotation.
//
// Takes ids (...string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithDeprecatedKeyIDs(ids ...string) Option {
	return func(c *Container) {
		c.ensureOverrides().Security.DeprecatedKeyIDs = ids
	}
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
	return func(c *Container) {
		c.ensureOverrides().Logger.Level = level
	}
}

// WithLogger replaces the entire logger configuration.
//
// Takes cfg (logger_dto.Config) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithLogger(cfg logger_dto.Config) Option {
	return func(c *Container) {
		c.ensureOverrides().Logger = cfg
	}
}

// WithDatabaseDriver selects the database backend. Valid values: "sqlite"
// (default), "postgres", "d1".
//
// Takes driver (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithDatabaseDriver(driver string) Option {
	return func(c *Container) {
		c.ensureOverrides().Database.Driver = &driver
	}
}

// WithPostgresURL sets the PostgreSQL connection URL. Required when the
// driver is "postgres".
//
// Takes url (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithPostgresURL(url string) Option {
	return func(c *Container) {
		c.ensureOverrides().Database.Postgres.URL = &url
	}
}

// WithPostgresMaxConns sets the maximum number of connections in the
// PostgreSQL pool.
//
// Takes n (int32) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithPostgresMaxConns(n int32) Option {
	return func(c *Container) {
		c.ensureOverrides().Database.Postgres.MaxConns = &n
	}
}

// WithPostgresMinConns sets the minimum number of connections held in the
// PostgreSQL pool.
//
// Takes n (int32) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithPostgresMinConns(n int32) Option {
	return func(c *Container) {
		c.ensureOverrides().Database.Postgres.MinConns = &n
	}
}

// WithD1APIToken sets the Cloudflare API token used for D1 access. Required
// when the driver is "d1".
//
// Takes token (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithD1APIToken(token string) Option {
	return func(c *Container) {
		c.ensureOverrides().Database.D1.APIToken = &token
	}
}

// WithD1AccountID sets the Cloudflare account ID for D1 access.
//
// Takes id (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithD1AccountID(id string) Option {
	return func(c *Container) {
		c.ensureOverrides().Database.D1.AccountID = &id
	}
}

// WithD1DatabaseID sets the D1 database UUID.
//
// Takes id (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithD1DatabaseID(id string) Option {
	return func(c *Container) {
		c.ensureOverrides().Database.D1.DatabaseID = &id
	}
}

// WithOTLP replaces the entire OpenTelemetry Protocol exporter configuration.
// Use this when you want to set multiple OTLP fields at once; otherwise the
// per-field WithOTLP* options below are more ergonomic.
//
// Takes o (config.OtlpConfig) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithOTLP(o config.OtlpConfig) Option {
	return func(c *Container) {
		c.ensureOverrides().Otlp = o
	}
}

// WithOTLPEnabled controls whether OTLP exporting is active.
//
// Takes enabled (bool) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithOTLPEnabled(enabled bool) Option {
	return func(c *Container) {
		c.ensureOverrides().Otlp.Enabled = &enabled
	}
}

// WithOTLPEndpoint sets the OTLP collector endpoint
// (e.g. "localhost:4317").
//
// Takes endpoint (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithOTLPEndpoint(endpoint string) Option {
	return func(c *Container) {
		c.ensureOverrides().Otlp.Endpoint = &endpoint
	}
}

// WithOTLPProtocol sets the OTLP transport protocol. Valid values: "grpc",
// "http", "https".
//
// Takes protocol (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithOTLPProtocol(protocol string) Option {
	return func(c *Container) {
		c.ensureOverrides().Otlp.Protocol = &protocol
	}
}

// WithOTLPTraceSampleRate sets the fraction of traces to sample (0.0 to 1.0).
// Applies regardless of whether OTLP export is enabled.
//
// Takes rate (float64) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithOTLPTraceSampleRate(rate float64) Option {
	return func(c *Container) {
		c.ensureOverrides().Otlp.TraceSampleRate = &rate
	}
}

// WithOTLPHeaders sets the HTTP headers sent with OTLP requests. Useful for
// adding authentication headers to managed collectors.
//
// Takes headers (map[string]string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithOTLPHeaders(headers map[string]string) Option {
	return func(c *Container) {
		c.ensureOverrides().Otlp.Headers = headers
	}
}

// WithOTLPInsecureTLS controls whether TLS certificate verification is
// disabled for the OTLP connection.
//
// Takes insecure (bool) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithOTLPInsecureTLS(insecure bool) Option {
	return func(c *Container) {
		c.ensureOverrides().Otlp.TLS.Insecure = &insecure
	}
}

// WithHealthEnabled controls whether the health probe server starts.
//
// Takes enabled (bool) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithHealthEnabled(enabled bool) Option {
	return func(c *Container) {
		c.ensureOverrides().HealthProbe.Enabled = &enabled
	}
}

// WithHealthBindAddress sets the network address to bind the health probe
// server to.
//
// Use "0.0.0.0" to expose externally (e.g. for Docker health checks). The
// default is "127.0.0.1".
//
// Takes addr (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithHealthBindAddress(addr string) Option {
	return func(c *Container) {
		c.ensureOverrides().HealthProbe.BindAddress = &addr
	}
}

// WithHealthMetricsEnabled controls whether the Prometheus metrics endpoint
// is exposed on the health probe server.
//
// Takes enabled (bool) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithHealthMetricsEnabled(enabled bool) Option {
	return func(c *Container) {
		c.ensureOverrides().HealthProbe.MetricsEnabled = &enabled
	}
}

// WithHealthMetricsPath sets the URL path for the Prometheus metrics
// endpoint.
//
// Takes path (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithHealthMetricsPath(path string) Option {
	return func(c *Container) {
		c.ensureOverrides().HealthProbe.MetricsPath = &path
	}
}

// WithHealthLivePath sets the URL path for the liveness probe endpoint.
//
// Takes path (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithHealthLivePath(path string) Option {
	return func(c *Container) {
		c.ensureOverrides().HealthProbe.LivePath = &path
	}
}

// WithHealthReadyPath sets the URL path for the readiness probe endpoint.
//
// Takes path (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithHealthReadyPath(path string) Option {
	return func(c *Container) {
		c.ensureOverrides().HealthProbe.ReadyPath = &path
	}
}

// WithHealthCheckTimeout sets the maximum time for each individual health
// check.
//
// Takes d (time.Duration) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithHealthCheckTimeout(d time.Duration) Option {
	return func(c *Container) {
		c.ensureOverrides().HealthProbe.CheckTimeoutSeconds = new(int(d.Seconds()))
	}
}

// WithHealthAutoNextPort enables automatic port selection for the health
// probe server when the configured port is already in use.
//
// Takes enabled (bool) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithHealthAutoNextPort(enabled bool) Option {
	return func(c *Container) {
		c.ensureOverrides().HealthProbe.AutoNextPort = &enabled
	}
}

// WithHealthProbePort sets the port for the health probe server.
//
// Takes port (int) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithHealthProbePort(port int) Option {
	return func(c *Container) {
		c.ensureOverrides().HealthProbe.Port = new(strconv.Itoa(port))
	}
}

// WithBaseDir sets the root directory of the website project.
//
// Takes dir (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithBaseDir(dir string) Option {
	return func(c *Container) {
		c.ensureOverrides().Paths.BaseDir = &dir
	}
}

// WithComponentsSourceDir sets the directory for .pkc/.sfc component files.
//
// Takes dir (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithComponentsSourceDir(dir string) Option {
	return func(c *Container) {
		c.ensureOverrides().Paths.ComponentsSourceDir = &dir
	}
}

// WithPagesSourceDir sets the directory for page definition files.
//
// Takes dir (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithPagesSourceDir(dir string) Option {
	return func(c *Container) {
		c.ensureOverrides().Paths.PagesSourceDir = &dir
	}
}

// WithPartialsSourceDir sets the directory for partial definition files.
//
// Takes dir (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithPartialsSourceDir(dir string) Option {
	return func(c *Container) {
		c.ensureOverrides().Paths.PartialsSourceDir = &dir
	}
}

// WithEmailsSourceDir sets the directory for email template source files.
//
// Takes dir (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithEmailsSourceDir(dir string) Option {
	return func(c *Container) {
		c.ensureOverrides().Paths.EmailsSourceDir = &dir
	}
}

// WithPdfsSourceDir sets the directory for PDF template source files.
//
// Takes dir (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithPdfsSourceDir(dir string) Option {
	return func(c *Container) {
		c.ensureOverrides().Paths.PdfsSourceDir = &dir
	}
}

// WithE2ESourceDir sets the directory for E2E test pages and partials. Only
// scanned when E2E mode is enabled.
//
// Takes dir (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithE2ESourceDir(dir string) Option {
	return func(c *Container) {
		c.ensureOverrides().Paths.E2ESourceDir = &dir
	}
}

// WithAssetsSourceDir sets the directory for asset files.
//
// Takes dir (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithAssetsSourceDir(dir string) Option {
	return func(c *Container) {
		c.ensureOverrides().Paths.AssetsSourceDir = &dir
	}
}

// WithI18nSourceDir sets the directory containing locale and translation
// JSON files.
//
// Takes dir (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithI18nSourceDir(dir string) Option {
	return func(c *Container) {
		c.ensureOverrides().Paths.I18nSourceDir = &dir
	}
}

// WithBaseServePath sets the URL path prefix for serving pages.
//
// Takes path (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithBaseServePath(path string) Option {
	return func(c *Container) {
		c.ensureOverrides().Paths.BaseServePath = &path
	}
}

// WithPartialServePath sets the URL path prefix for serving partials.
//
// Takes path (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithPartialServePath(path string) Option {
	return func(c *Container) {
		c.ensureOverrides().Paths.PartialServePath = &path
	}
}

// WithActionServePath sets the URL path prefix for server actions.
//
// Takes path (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithActionServePath(path string) Option {
	return func(c *Container) {
		c.ensureOverrides().Paths.ActionServePath = &path
	}
}

// WithLibServePath sets the URL path prefix for serving internal library
// files.
//
// Takes path (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithLibServePath(path string) Option {
	return func(c *Container) {
		c.ensureOverrides().Paths.LibServePath = &path
	}
}

// WithDistServePath sets the URL path prefix for serving frontend
// distribution files.
//
// Takes path (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithDistServePath(path string) Option {
	return func(c *Container) {
		c.ensureOverrides().Paths.DistServePath = &path
	}
}

// WithArtefactServePath sets the URL path prefix for serving compiled
// assets.
//
// Takes path (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithArtefactServePath(path string) Option {
	return func(c *Container) {
		c.ensureOverrides().Paths.ArtefactServePath = &path
	}
}

// WithDefaultServeMode selects the default page serving mode.
// Valid values: "preview" (dynamic) or "render" (static).
//
// Takes mode (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithDefaultServeMode(mode string) Option {
	return func(c *Container) {
		c.ensureOverrides().Build.DefaultServeMode = &mode
	}
}

// WithStoragePresign replaces the entire presigned URL configuration.
//
// Takes p (config.StoragePresignConfig) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithStoragePresign(p config.StoragePresignConfig) Option {
	return func(c *Container) {
		c.ensureOverrides().Storage.Presign = p
	}
}

// WithStoragePresignSecret sets the HMAC secret for signing presign tokens.
//
// The secret must be at least 32 bytes. For multi-instance deployments,
// set this so all instances share the same secret.
//
// Takes secret (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithStoragePresignSecret(secret string) Option {
	return func(c *Container) {
		c.ensureOverrides().Storage.Presign.Secret = &secret
	}
}

// WithStoragePresignDefaultExpiry sets the default validity duration for
// presigned URLs.
//
// Takes d (time.Duration) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithStoragePresignDefaultExpiry(d time.Duration) Option {
	return func(c *Container) {
		c.ensureOverrides().Storage.Presign.DefaultExpiry = &d
	}
}

// WithStoragePresignMaxExpiry sets the maximum validity duration for
// presigned URLs.
//
// Takes d (time.Duration) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithStoragePresignMaxExpiry(d time.Duration) Option {
	return func(c *Container) {
		c.ensureOverrides().Storage.Presign.MaxExpiry = &d
	}
}

// WithStoragePresignDefaultMaxSize sets the default maximum upload size in
// bytes.
//
// Takes size (int64) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithStoragePresignDefaultMaxSize(size int64) Option {
	return func(c *Container) {
		c.ensureOverrides().Storage.Presign.DefaultMaxSize = &size
	}
}

// WithStoragePresignMaxMaxSize sets the absolute maximum upload size in
// bytes.
//
// Takes size (int64) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithStoragePresignMaxMaxSize(size int64) Option {
	return func(c *Container) {
		c.ensureOverrides().Storage.Presign.MaxMaxSize = &size
	}
}

// WithStoragePresignRateLimit sets the per-IP rate limit for presigned upload
// requests, in requests per minute. Zero disables rate limiting.
//
// Takes rpm (int) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithStoragePresignRateLimit(rpm int) Option {
	return func(c *Container) {
		c.ensureOverrides().Storage.Presign.RateLimitPerMinute = &rpm
	}
}

// WithI18nDefaultLocale sets the default locale used for internationalisation
// when a translation is missing or no locale is selected.
//
// Takes locale (string) which is the value to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithI18nDefaultLocale(locale string) Option {
	return func(c *Container) {
		c.ensureOverrides().I18nDefaultLocale = &locale
	}
}
