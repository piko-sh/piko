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

package config

import (
	"time"

	"piko.sh/piko/internal/logger/logger_dto"
)

// ServerConfig is the root configuration object for the entire server. It
// aggregates all other configuration areas like paths, build settings, and
// networking.
type ServerConfig struct {
	// Security holds settings for security headers, rate limiting, and reporting.
	Security SecurityConfig `json:"security" yaml:"security"`

	// Paths specifies directory paths for pages, emails, partials, and other
	// source locations.
	Paths PathsConfig `json:"paths" yaml:"paths"`

	// HealthProbe configures the health check endpoint for liveness and readiness
	// probes.
	HealthProbe HealthProbeConfig `json:"healthProbe" yaml:"healthProbe"`

	// Network holds the network configuration for the server.
	Network NetworkConfig `json:"network" yaml:"network"`

	// Storage sets up the storage service, including presigned URL support.
	Storage StorageConfig `json:"storage" yaml:"storage"`

	// Database specifies the database connection settings.
	Database DatabaseConfig `json:"database" yaml:"database"`

	// Otlp holds the settings for the OpenTelemetry Protocol exporter used for
	// tracing and metrics.
	Otlp OtlpConfig `json:"otlp" yaml:"otlp"`

	// Build sets build options such as watch mode and asset pre-rendering.
	Build BuildModeConfig `json:"build" yaml:"build"`

	// I18nDefaultLocale specifies the default locale for internationalisation;
	// defaults to "en" if not set.
	I18nDefaultLocale *string `json:"i18nDefaultLocale" yaml:"i18nDefaultLocale" default:"en" env:"PIKO_I18N_DEFAULT_LOCALE" flag:"i18nDefaultLocale" usage:"Default locale for internationalization."`

	// CSRFSecret is the secret for CSRF token generation. If not set, a random
	// secret is generated and persisted to a temp file.
	CSRFSecret *string `json:"csrfSecret" yaml:"csrfSecret" env:"PIKO_CSRF_SECRET" flag:"csrfSecret" usage:"Secret for CSRF token generation. If not set, a random secret is generated and persisted to a temp file."`

	// Logger configures the application logging behaviour.
	Logger logger_dto.Config `json:"logger" yaml:"logger"`
}

// PathsConfig holds all file system and URL path settings for the project.
// Framework-internal paths (PikoInternalPath, RegistryPath, compiled target
// dirs, etc.) are defined as constants in internal_paths.go and are not
// user-configurable.
type PathsConfig struct {
	// BaseDir is the root directory of the website project.
	BaseDir *string `json:"baseDir" yaml:"baseDir" default:"." env:"PIKO_BASE_DIR" flag:"baseDir" usage:"Root directory of the website project." validate:"required"`

	// ComponentsSourceDir is the directory path for .pkc/.sfc component files.
	ComponentsSourceDir *string `json:"componentsSourceDir" yaml:"componentsSourceDir" default:"components" env:"PIKO_COMPONENTS_SOURCE_DIR" flag:"componentsDir" usage:"Directory for .pkc/.sfc components."`

	// PagesSourceDir is the path to page definition files relative to BaseDir.
	PagesSourceDir *string `json:"pagesSourceDir" yaml:"pagesSourceDir" default:"pages" env:"PIKO_PAGES_SOURCE_DIR" flag:"pagesDir" usage:"Directory for page definition files."`

	// PartialsSourceDir is the directory containing partial definition files.
	PartialsSourceDir *string `json:"partialsSourceDir" yaml:"partialsSourceDir" default:"partials" env:"PIKO_PARTIALS_SOURCE_DIR" flag:"partialsDir" usage:"Directory for partial definition files."`

	// EmailsSourceDir is the directory for email template source files.
	EmailsSourceDir *string `json:"emailsSourceDir" yaml:"emailsSourceDir" default:"emails" env:"PIKO_EMAILS_SOURCE_DIR" flag:"emailsDir" usage:"Directory for email definition files."`

	// PdfsSourceDir is the directory for PDF template source files.
	PdfsSourceDir *string `json:"pdfsSourceDir" yaml:"pdfsSourceDir" default:"pdfs" env:"PIKO_PDFS_SOURCE_DIR" flag:"pdfsDir" usage:"Directory for PDF definition files."`

	// E2ESourceDir is the directory for E2E test pages and partials, relative
	// to BaseDir. Only scanned when Build.E2EMode is enabled; these pages are
	// never included in production builds.
	E2ESourceDir *string `json:"e2eSourceDir" yaml:"e2eSourceDir" default:"e2e" env:"PIKO_E2E_SOURCE_DIR" flag:"e2eDir" usage:"Directory for E2E test pages and partials."`

	// AssetsSourceDir is the folder for asset files, relative to BaseDir.
	AssetsSourceDir *string `json:"assetsSourceDir" yaml:"assetsSourceDir" default:"lib" env:"PIKO_ASSETS_SOURCE_DIR" flag:"assetsDir" usage:"Directory for asset files."`

	// I18nSourceDir is the directory containing locale and translation JSON files.
	I18nSourceDir *string `json:"i18nSourceDir" yaml:"i18nSourceDir" default:"locales" env:"PIKO_I18N_SOURCE_DIR" flag:"i18nSourceDir" usage:"Directory containing locale/translation files."`

	// BaseServePath is the URL path prefix for serving pages.
	BaseServePath *string `json:"baseServePath" yaml:"baseServePath" default:"" env:"PIKO_BASE_SERVE_PATH" flag:"baseServePath" usage:"URL path prefix for serving pages."`

	// PartialServePath is the URL path prefix for serving partials.
	PartialServePath *string `json:"partialServePath" yaml:"partialServePath" default:"/_piko/partials" env:"PIKO_PARTIAL_SERVE_PATH" flag:"partialServePath" usage:"URL path prefix for serving partials."`

	// ActionServePath is the URL path prefix for server actions.
	ActionServePath *string `json:"actionServePath" yaml:"actionServePath" default:"/_piko/actions" env:"PIKO_ACTION_SERVE_PATH" flag:"actionServePath" usage:"URL path prefix for server actions."`

	// LibServePath is the URL path prefix for serving internal library files.
	LibServePath *string `json:"libServePath" yaml:"libServePath" default:"/_piko/lib" env:"PIKO_LIB_SERVE_PATH" flag:"libServePath" usage:"URL path prefix for internal library files."`

	// DistServePath is the URL path prefix for serving frontend files.
	DistServePath *string `json:"distServePath" yaml:"distServePath" default:"/_piko/dist" env:"PIKO_DIST_SERVE_PATH" flag:"distServePath" usage:"URL path prefix for frontend distribution files."`

	// ArtefactServePath is the URL path prefix for serving compiled assets.
	ArtefactServePath *string `json:"artefactServePath" yaml:"artefactServePath" default:"/_piko/assets" env:"PIKO_COMPILED_ASSETS_SERVE_PATH" flag:"assetsServePath" usage:"URL path prefix for compiled assets."`
}

// BuildModeConfig holds settings for the build and watch process.
type BuildModeConfig struct {
	// DefaultServeMode specifies whether to serve pages dynamically or as static
	// files; valid values are "preview" or "render"; defaults to "render".
	DefaultServeMode *string `json:"defaultServeMode" yaml:"defaultServeMode" default:"render" env:"PIKO_DEFAULT_SERVE_MODE" flag:"serveMode" usage:"Serve mode: 'preview' (dynamic) or 'render' (static)." validate:"oneof=preview render"`

	// WatchMode enables file system watching for hot-reloading, derived from
	// the run mode (dev = true, prod = false) and not loaded from YAML or
	// environment variables, with WithWatchMode() available to override.
	WatchMode *bool `json:"-" yaml:"-"`

	// E2EMode enables inclusion of E2E test pages and partials from
	// Paths.E2ESourceDir.
	//
	// When false (default), E2E pages are excluded from code generation and
	// cannot be served. When true, E2E pages are included but protected by a
	// runtime guard that returns 404 if disabled at runtime. This provides
	// defence in depth. WARNING: Never enable in production builds.
	// Set via WithE2EMode(); not loaded from YAML or environment variables.
	E2EMode *bool `json:"-" yaml:"-"`
}

// NetworkConfig holds server network-related configurations.
type NetworkConfig struct {
	// Port specifies the TCP port number for the HTTP server; default is "8080".
	Port *string `json:"port" yaml:"port" default:"8080" env:"PIKO_PORT" flag:"port" usage:"HTTP server port."`

	// PublicDomain is the public domain used for CORS allowed origins and
	// absolute URLs; empty allows all origins.
	PublicDomain *string `json:"publicDomain" yaml:"publicDomain" default:"localhost:8080" env:"PIKO_PUBLIC_DOMAIN" flag:"publicDomain" usage:"Public domain for CORS and absolute URLs."`

	// ActionMaxBodyBytes is the maximum size in bytes for action request bodies.
	ActionMaxBodyBytes *int64 `json:"actionMaxBodyBytes" yaml:"actionMaxBodyBytes" default:"1048576" env:"PIKO_ACTION_MAX_BODY_BYTES" flag:"actionMaxBodyBytes" usage:"Maximum size in bytes for action request bodies."`

	// AutoNextPort enables automatic port selection when the default port is in.
	// use.
	AutoNextPort *bool `json:"autoNextPort" yaml:"autoNextPort" default:"false" env:"PIKO_AUTO_PORT" flag:"autoNextPort" usage:"Automatically find the next port if the default is in use."`

	// ForceHTTPS enables redirection from HTTP to HTTPS; false allows plain HTTP.
	ForceHTTPS *bool `json:"forceHttps" yaml:"forceHttps" default:"false" env:"PIKO_FORCE_HTTPS" flag:"forceHttps" usage:"Redirect all HTTP requests to HTTPS."`

	// RequestTimeoutSeconds sets the maximum duration in seconds for dynamic
	// HTTP requests (pages, partials, actions), where 0 disables the
	// timeout middleware entirely (default: 60).
	RequestTimeoutSeconds *int `json:"requestTimeoutSeconds" yaml:"requestTimeoutSeconds" default:"60" env:"PIKO_REQUEST_TIMEOUT_SECONDS" flag:"requestTimeoutSeconds" usage:"Timeout in seconds for dynamic HTTP requests (0 to disable)."`

	// DefaultMaxSSEDurationSeconds is the maximum lifetime in seconds for SSE
	// connections that do not specify their own limit (default: 1800 / 30 min).
	// Set to 0 for unlimited; individual actions can override via
	// ResourceLimits.MaxSSEDuration.
	DefaultMaxSSEDurationSeconds *int `json:"defaultMaxSseDurationSeconds" yaml:"defaultMaxSseDurationSeconds" default:"1800" env:"PIKO_DEFAULT_MAX_SSE_DURATION_SECONDS" flag:"defaultMaxSseDurationSeconds" usage:"Default max SSE connection duration in seconds (0 for unlimited)."`

	// MaxMultipartFormBytes is the maximum in-memory size for multipart form
	// data, where files exceeding this limit are stored in temporary files
	// on disk (default: 33554432 / 32 MB).
	MaxMultipartFormBytes *int64 `json:"maxMultipartFormBytes" yaml:"maxMultipartFormBytes" default:"33554432" env:"PIKO_MAX_MULTIPART_FORM_BYTES" flag:"maxMultipartFormBytes" usage:"Maximum in-memory size for multipart form data in bytes."`

	// MaxConcurrentRequests is the maximum number of in-flight requests
	// the server will process simultaneously (default: 10000). Set to 0 to
	// disable the concurrency limit entirely.
	MaxConcurrentRequests *int `json:"maxConcurrentRequests" yaml:"maxConcurrentRequests" default:"10000" env:"PIKO_MAX_CONCURRENT_REQUESTS" flag:"maxConcurrentRequests" usage:"Maximum concurrent in-flight requests (0 for unlimited)."`

	// TLS holds server-side TLS/HTTPS configuration. When enabled, the server
	// terminates TLS directly using user-provided certificates.
	TLS TLSConfig `json:"tls" yaml:"tls"`
}

// TLSConfig holds server-side TLS settings for HTTPS termination, including
// certificate paths and mTLS client verification.
type TLSConfig struct {
	// Enabled controls whether TLS is active (default: false).
	Enabled *bool `json:"enabled" yaml:"enabled" default:"false" env:"PIKO_TLS_ENABLED" flag:"tlsEnabled" usage:"Enable TLS/HTTPS for the server."`

	// CertFile is the path to the PEM-encoded TLS certificate file.
	CertFile *string `json:"certFile" yaml:"certFile" env:"PIKO_TLS_CERT_FILE" flag:"tlsCertFile" usage:"Path to TLS certificate PEM file."`

	// KeyFile is the path to the PEM-encoded TLS private key file.
	KeyFile *string `json:"keyFile" yaml:"keyFile" env:"PIKO_TLS_KEY_FILE" flag:"tlsKeyFile" usage:"Path to TLS private key PEM file."`

	// ClientCAFile is the path to a PEM-encoded CA bundle for mTLS client
	// certificate verification. When set, the server requires valid client
	// certificates signed by one of these CAs.
	ClientCAFile *string `json:"clientCaFile" yaml:"clientCaFile" env:"PIKO_TLS_CLIENT_CA_FILE" flag:"tlsClientCaFile" usage:"Path to client CA PEM bundle for mTLS."`

	// ClientAuthType controls the client certificate verification mode.
	// Valid values: "none" (default), "request", "require", "verify",
	// "require_and_verify".
	ClientAuthType *string `json:"clientAuthType" yaml:"clientAuthType" default:"none" env:"PIKO_TLS_CLIENT_AUTH_TYPE" flag:"tlsClientAuthType" usage:"Client certificate auth mode (none, request, require, verify, require_and_verify)." validate:"oneof=none request require verify require_and_verify"`

	// MinVersion is the minimum TLS version. Valid values: "1.2" (default),
	// "1.3".
	MinVersion *string `json:"minVersion" yaml:"minVersion" default:"1.2" env:"PIKO_TLS_MIN_VERSION" flag:"tlsMinVersion" usage:"Minimum TLS version (1.2 or 1.3)." validate:"oneof=1.2 1.3"`

	// HotReload enables automatic reload of certificate and key files when
	// they change on disk, without requiring a server restart. Default: false.
	HotReload *bool `json:"hotReload" yaml:"hotReload" default:"false" env:"PIKO_TLS_HOT_RELOAD" flag:"tlsHotReload" usage:"Enable certificate hot-reload on file change."`

	// RedirectHTTPPort, when set, starts a plain HTTP listener on this port
	// that 301-redirects all requests to the HTTPS server. Disabled by default.
	RedirectHTTPPort *string `json:"redirectHttpPort" yaml:"redirectHttpPort" env:"PIKO_TLS_REDIRECT_HTTP_PORT" flag:"tlsRedirectHttpPort" usage:"Start a plain HTTP listener on this port that redirects to HTTPS."`
}

// HealthTLSConfig holds simplified TLS settings for the health probe server.
// When not configured, the health server remains plain HTTP regardless of the
// main server's TLS settings.
type HealthTLSConfig struct {
	// Enabled controls whether TLS is active for the health probe server.
	// Default: false.
	Enabled *bool `json:"enabled" yaml:"enabled" default:"false" env:"PIKO_HEALTH_TLS_ENABLED" flag:"healthTlsEnabled" usage:"Enable TLS for the health probe server."`

	// CertFile is the path to the PEM-encoded TLS certificate file.
	CertFile *string `json:"certFile" yaml:"certFile" env:"PIKO_HEALTH_TLS_CERT_FILE" flag:"healthTlsCertFile" usage:"Path to health server TLS certificate PEM file."`

	// KeyFile is the path to the PEM-encoded TLS private key file.
	KeyFile *string `json:"keyFile" yaml:"keyFile" env:"PIKO_HEALTH_TLS_KEY_FILE" flag:"healthTlsKeyFile" usage:"Path to health server TLS private key PEM file."`

	// MinVersion is the minimum TLS version. Valid values: "1.2" (default),
	// "1.3".
	MinVersion *string `json:"minVersion" yaml:"minVersion" default:"1.2" env:"PIKO_HEALTH_TLS_MIN_VERSION" flag:"healthTlsMinVersion" usage:"Minimum TLS version for health server (1.2 or 1.3)." validate:"oneof=1.2 1.3"`
}

// OtlpConfig holds configuration for OpenTelemetry Protocol exporting.
type OtlpConfig struct {
	// Headers contains key-value pairs sent as HTTP headers with OTLP requests.
	Headers map[string]string `json:"headers" yaml:"headers" env:"PIKO_OTLP_HEADERS" flag:"otlpHeaders" usage:"Key-value pairs for OTLP headers (e.g., 'key1:val1,key2:val2')." summary:"hide"`

	// Protocol specifies the transport protocol: "grpc", "http", or "https".
	Protocol *string `json:"protocol" yaml:"protocol" default:"http" env:"PIKO_OTLP_PROTOCOL" flag:"otlpProtocol" usage:"Protocol for OTLP exporter ('grpc' or 'http/protobuf')."`

	// Endpoint is the target address for the OTLP collector (e.g.
	// "localhost:4317").
	Endpoint *string `json:"endpoint" yaml:"endpoint" default:"localhost:4317" env:"PIKO_OTLP_ENDPOINT" flag:"otlpEndpoint" usage:"Target endpoint for OTLP traces (e.g., 'localhost:4317')."`

	// TraceSampleRate specifies the fraction of traces to sample (0.0 to 1.0).
	// Applies to all tracing regardless of whether OTLP export is enabled.
	TraceSampleRate *float64 `json:"traceSampleRate" yaml:"traceSampleRate" default:"0.05" env:"PIKO_TRACE_SAMPLE_RATE" flag:"traceSampleRate" usage:"Fraction of traces to sample (0.0-1.0, e.g., 0.05 for 5%)."`

	// Enabled controls whether OTLP exporting is active.
	Enabled *bool `json:"enabled" yaml:"enabled" default:"false" env:"PIKO_OTLP_ENABLED" flag:"otlpEnabled" usage:"Enable OpenTelemetry protocol exporting."`

	// TLS holds the TLS settings for the OTLP connection.
	TLS OtlpTLSConfig `json:"tls" yaml:"tls"`
}

// OtlpTLSConfig holds TLS settings for the OTLP exporter connection.
type OtlpTLSConfig struct {
	// Insecure disables TLS certificate verification when true; default is true.
	Insecure *bool `json:"insecure" yaml:"insecure" default:"true" env:"PIKO_OTLP_TLS_INSECURE" flag:"otlpTlsInsecure" usage:"Disable TLS certificate verification for OTLP endpoint."`
}

// SecurityConfig holds security-related settings, including cryptographic
// encryption, HTTP security headers, rate limiting, and cookie security
// defaults.
type SecurityConfig struct {
	// Headers configures HTTP security headers such as X-Frame-Options, CSP,
	// and HSTS. Enabled by default for secure-by-default behaviour.
	Headers SecurityHeadersConfig `json:"headers" yaml:"headers"`

	// GCPKMS holds Google Cloud KMS settings. Only used when CryptoProvider is
	// set to "gcp_kms".
	GCPKMS GCPKMSConfig `json:"gcpKms" yaml:"gcpKms"`

	// AWSKMS holds AWS KMS settings; only used when CryptoProvider is "aws_kms".
	AWSKMS AWSKMSConfig `json:"awsKms" yaml:"awsKms"`

	// Cookies configures secure defaults for HTTP cookies, including
	// HttpOnly, Secure, and SameSite attributes. Enabled by default
	// for secure-by-default behaviour.
	Cookies CookieSecurityConfig `json:"cookies" yaml:"cookies"`

	// EncryptionKey is the base64-encoded 32-byte (256-bit) encryption key
	// for the default local AES-GCM provider, preferably loaded from the
	// PIKO_ENCRYPTION_KEY environment variable in production.
	EncryptionKey *string `json:"encryptionKey" yaml:"encryptionKey" default:"" env:"PIKO_ENCRYPTION_KEY" flag:"encryptionKey" usage:"Base64-encoded encryption key for local AES-GCM provider." summary:"hide"`

	// CSRF configures CSRF token enforcement. Defaults provide
	// defence-in-depth by requiring tokens on browser requests.
	CSRF CSRFConfig `json:"csrf" yaml:"csrf"`

	// CryptoProvider specifies which encryption provider to use.
	// Options: "local_aes_gcm" (default, always available), "aws_kms", "gcp_kms"
	// (require explicit adapter registration).
	CryptoProvider *string `json:"cryptoProvider" yaml:"cryptoProvider" default:"local_aes_gcm" env:"PIKO_CRYPTO_PROVIDER" flag:"cryptoProvider" usage:"Encryption provider to use (local_aes_gcm, aws_kms, gcp_kms)."`

	// DataKeyCacheTTL specifies how long to cache decrypted data keys for
	// KMS providers, improving performance by reducing API calls, where
	// 0 disables caching and the default is 5 minutes.
	DataKeyCacheTTL *time.Duration `json:"dataKeyCacheTtl" yaml:"dataKeyCacheTtl" default:"5m" env:"PIKO_DATA_KEY_CACHE_TTL" flag:"dataKeyCacheTtl" usage:"TTL for cached data keys (e.g., 5m, 10m)."`

	// DataKeyCacheMaxSize is the maximum number of data keys to cache.
	// Each key is typically 32 bytes for AES-256; default is 100 keys.
	DataKeyCacheMaxSize *int `json:"dataKeyCacheMaxSize" yaml:"dataKeyCacheMaxSize" default:"100" env:"PIKO_DATA_KEY_CACHE_MAX_SIZE" flag:"dataKeyCacheMaxSize" usage:"Maximum number of data keys to cache."`

	// CaptchaProvider specifies which captcha provider to use.
	// Options: "hmac_challenge" (built-in test provider), "turnstile",
	// "recaptcha_v3", "hcaptcha" (require explicit adapter registration).
	CaptchaProvider *string `json:"captchaProvider" yaml:"captchaProvider" default:"" env:"PIKO_CAPTCHA_PROVIDER" flag:"captchaProvider" usage:"Captcha provider to use (hmac_challenge, turnstile, recaptcha_v3, hcaptcha)." validate:"omitempty,oneof=hmac_challenge turnstile recaptcha_v3 hcaptcha"`

	// CaptchaSiteKey is the site key / widget key for the configured captcha
	// provider. Required for Turnstile, reCAPTCHA, and hCaptcha.
	CaptchaSiteKey *string `json:"captchaSiteKey" yaml:"captchaSiteKey" default:"" env:"PIKO_CAPTCHA_SITE_KEY" flag:"captchaSiteKey" usage:"Site key for the captcha provider."`

	// CaptchaSecretKey is the secret key for server-side captcha verification.
	CaptchaSecretKey *string `json:"captchaSecretKey" yaml:"captchaSecretKey" default:"" env:"PIKO_CAPTCHA_SECRET_KEY" flag:"captchaSecretKey" usage:"Secret key for captcha verification." summary:"hide"`

	// CaptchaScoreThreshold is the minimum score (0.0-1.0) for score-based
	// providers like reCAPTCHA v3. Default 0.5.
	CaptchaScoreThreshold *float64 `json:"captchaScoreThreshold" yaml:"captchaScoreThreshold" default:"0.5" env:"PIKO_CAPTCHA_SCORE_THRESHOLD" flag:"captchaScoreThreshold" usage:"Minimum captcha score threshold (0.0-1.0)." validate:"omitempty,gte=0,lte=1"`

	// SpamDetectEnabled enables the built-in spam detection rules engine.
	// When true and no providers are registered via options, the built-in
	// rules engine is automatically configured.
	SpamDetectEnabled *bool `json:"spamDetectEnabled" yaml:"spamDetectEnabled" default:"false" env:"PIKO_SPAM_DETECT_ENABLED" flag:"spamDetectEnabled" usage:"Enable built-in spam detection."`

	// SpamDetectScoreThreshold is the composite score above which
	// submissions are rejected. Default 0.7.
	SpamDetectScoreThreshold *float64 `json:"spamDetectScoreThreshold" yaml:"spamDetectScoreThreshold" default:"0.7" env:"PIKO_SPAM_DETECT_SCORE_THRESHOLD" flag:"spamDetectScoreThreshold" usage:"Spam detection score threshold (0.0-1.0)." validate:"omitempty,gte=0,lte=1"`

	// SpamDetectBlocklistPatterns is a list of regex patterns for the
	// built-in blocklist rule.
	SpamDetectBlocklistPatterns []string `json:"spamDetectBlocklistPatterns" yaml:"spamDetectBlocklistPatterns" env:"PIKO_SPAM_DETECT_BLOCKLIST"`

	// RateLimit sets up request rate limiting with trusted proxy support. Disabled
	// by default to stop users from rate-limiting their own reverse proxy.
	RateLimit RateLimitConfig `json:"rateLimit" yaml:"rateLimit"`

	// Sandbox configures filesystem sandboxing for Piko internals using Go 1.24's
	// os.Root. Disabled by default; when enabled, restricts file access to
	// configured directories.
	Sandbox SandboxConfig `json:"sandbox" yaml:"sandbox"`

	// Reporting configures the Reporting-Endpoints HTTP header used by CSP
	// report-to directive and other reporting APIs. Disabled by default; when
	// enabled, allows configuring violation report destinations.
	Reporting ReportingConfig `json:"reporting" yaml:"reporting"`

	// DeprecatedKeyIDs lists key IDs that are still valid for decryption but
	// should not be used for new encryption. This supports gradual key rotation.
	DeprecatedKeyIDs []string `json:"deprecatedKeyIds" yaml:"deprecatedKeyIds" env:"PIKO_DEPRECATED_KEY_IDS"`
}

// AWSKMSConfig holds settings for AWS Key Management Service.
type AWSKMSConfig struct {
	// KeyID is the AWS KMS key identifier. Accepts a key ID, key ARN, alias
	// name (alias/my-key), or alias ARN.
	KeyID *string `json:"keyId" yaml:"keyId" env:"PIKO_AWS_KMS_KEY_ID" flag:"awsKmsKeyId" usage:"AWS KMS key ID or ARN."`

	// Region is the AWS region where the KMS key is located (e.g. "us-east-1").
	Region *string `json:"region" yaml:"region" env:"PIKO_AWS_KMS_REGION" flag:"awsKmsRegion" usage:"AWS region for KMS key."`

	// MaxRetries is the maximum number of retry attempts for KMS operations.
	// Default: 3.
	MaxRetries *int `json:"maxRetries" yaml:"maxRetries" default:"3" env:"PIKO_AWS_KMS_MAX_RETRIES" flag:"awsKmsMaxRetries" usage:"Maximum retry attempts for KMS operations."`

	// EnableMetrics enables CloudWatch metrics for KMS operations. Default: false.
	EnableMetrics *bool `json:"enableMetrics" yaml:"enableMetrics" default:"false" env:"PIKO_AWS_KMS_ENABLE_METRICS" flag:"awsKmsEnableMetrics" usage:"Enable CloudWatch metrics for KMS."`
}

// GCPKMSConfig holds settings for Google Cloud Key Management Service.
type GCPKMSConfig struct {
	// ProjectID is the Google Cloud project ID.
	ProjectID *string `json:"projectId" yaml:"projectId" env:"PIKO_GCP_KMS_PROJECT_ID" flag:"gcpKmsProjectId" usage:"Google Cloud project ID."`

	// Location is the key ring location (e.g., "global", "us-central1",
	// "europe-west1").
	Location *string `json:"location" yaml:"location" env:"PIKO_GCP_KMS_LOCATION" flag:"gcpKmsLocation" usage:"GCP KMS key ring location."`

	// KeyRing is the name of the key ring containing the key.
	KeyRing *string `json:"keyRing" yaml:"keyRing" env:"PIKO_GCP_KMS_KEY_RING" flag:"gcpKmsKeyRing" usage:"GCP KMS key ring name."`

	// KeyName is the name of the cryptographic key in GCP KMS.
	KeyName *string `json:"keyName" yaml:"keyName" env:"PIKO_GCP_KMS_KEY_NAME" flag:"gcpKmsKeyName" usage:"GCP KMS cryptographic key name."`

	// MaxRetries is the maximum number of retry attempts for KMS operations.
	// Default: 3.
	MaxRetries *int `json:"maxRetries" yaml:"maxRetries" default:"3" env:"PIKO_GCP_KMS_MAX_RETRIES" flag:"gcpKmsMaxRetries" usage:"Maximum retry attempts for KMS operations."`
}

// SecurityHeadersConfig configures HTTP security headers following OWASP
// best practices, protecting against clickjacking, XSS, and MIME sniffing.
// Enabled by default for secure-by-default behaviour.
type SecurityHeadersConfig struct {
	// XFrameOptions controls the X-Frame-Options header to prevent clickjacking.
	// Valid options are "DENY" or "SAMEORIGIN"; defaults to "DENY".
	XFrameOptions *string `json:"xFrameOptions" yaml:"xFrameOptions" default:"DENY" env:"PIKO_SECURITY_X_FRAME_OPTIONS" flag:"xFrameOptions" usage:"X-Frame-Options header value (DENY or SAMEORIGIN)." validate:"oneof=DENY SAMEORIGIN"`

	// XContentTypeOptions sets the X-Content-Type-Options header value to stop
	// browsers from guessing the content type. Default: "nosniff".
	XContentTypeOptions *string `json:"xContentTypeOptions" yaml:"xContentTypeOptions" default:"nosniff" env:"PIKO_SECURITY_X_CONTENT_TYPE_OPTIONS" flag:"xContentTypeOptions" usage:"X-Content-Type-Options header value."`

	// ReferrerPolicy sets the Referrer-Policy header value.
	// Default: "strict-origin-when-cross-origin".
	ReferrerPolicy *string `json:"referrerPolicy" yaml:"referrerPolicy" default:"strict-origin-when-cross-origin" env:"PIKO_SECURITY_REFERRER_POLICY" flag:"referrerPolicy" usage:"Referrer-Policy header value."`

	// ContentSecurityPolicy sets the Content-Security-Policy header value.
	//
	// When empty, Piko applies sensible defaults via the CSP builder that are
	// compatible with built-in features like font loading and inline styles.
	// Use piko.WithCSP() for type-safe customisation, or set this field for a
	// raw string override. To disable CSP entirely, use piko.WithCSPString("").
	ContentSecurityPolicy *string `json:"contentSecurityPolicy" yaml:"contentSecurityPolicy" env:"PIKO_SECURITY_CSP" flag:"contentSecurityPolicy" usage:"Content-Security-Policy header value. Leave empty for Piko defaults."`

	// StrictTransportSecurity sets the Strict-Transport-Security
	// (HSTS) header, only applied when ForceHTTPS is enabled in
	// NetworkConfig.
	//
	// Default: "max-age=31536000; includeSubDomains".
	StrictTransportSecurity *string `json:"strictTransportSecurity" yaml:"strictTransportSecurity" default:"max-age=31536000; includeSubDomains" env:"PIKO_SECURITY_HSTS" flag:"hsts" usage:"Strict-Transport-Security header value (HSTS)."`

	// CrossOriginOpenerPolicy sets the Cross-Origin-Opener-Policy header value.
	// Default: "same-origin".
	CrossOriginOpenerPolicy *string `json:"crossOriginOpenerPolicy" yaml:"crossOriginOpenerPolicy" default:"same-origin" env:"PIKO_SECURITY_COOP" flag:"crossOriginOpenerPolicy" usage:"Cross-Origin-Opener-Policy header value."`

	// CrossOriginResourcePolicy sets the Cross-Origin-Resource-Policy header.
	// Default is "same-origin".
	CrossOriginResourcePolicy *string `json:"crossOriginResourcePolicy" yaml:"crossOriginResourcePolicy" default:"same-origin" env:"PIKO_SECURITY_CORP" flag:"crossOriginResourcePolicy" usage:"Cross-Origin-Resource-Policy header value."`

	// PermissionsPolicy sets the Permissions-Policy header (formerly
	// Feature-Policy) which controls which browser features can be used. Default:
	// "" (not set).
	PermissionsPolicy *string `json:"permissionsPolicy" yaml:"permissionsPolicy" default:"" env:"PIKO_SECURITY_PERMISSIONS_POLICY" flag:"permissionsPolicy" usage:"Permissions-Policy header value."`

	// Enabled controls whether security headers are added to HTTP responses,
	// defaulting to true.
	Enabled *bool `json:"enabled" yaml:"enabled" default:"true" env:"PIKO_SECURITY_HEADERS_ENABLED" flag:"securityHeadersEnabled" usage:"Enable HTTP security headers."`

	// StripServerHeader removes the Server header from responses to prevent
	// information disclosure. Default: true.
	StripServerHeader *bool `json:"stripServerHeader" yaml:"stripServerHeader" default:"true" env:"PIKO_SECURITY_STRIP_SERVER" flag:"stripServerHeader" usage:"Remove Server header from responses."`

	// StripPoweredByHeader removes the X-Powered-By header from responses.
	// Default: true.
	StripPoweredByHeader *bool `json:"stripPoweredByHeader" yaml:"stripPoweredByHeader" default:"true" env:"PIKO_SECURITY_STRIP_POWERED_BY" flag:"stripPoweredByHeader" usage:"Remove X-Powered-By header from responses."`
}

// RateLimitConfig configures request rate limiting with trusted proxy support.
// Rate limiting is disabled by default to prevent users from accidentally
// rate-limiting their own reverse proxy when deployed behind one.
type RateLimitConfig struct {
	// Global sets the rate limit that applies to all requests across the service.
	Global RateLimitTierConfig `json:"global" yaml:"global"`

	// Actions sets rate limits for server action endpoints.
	Actions RateLimitTierConfig `json:"actions" yaml:"actions"`

	// Storage specifies the backend for rate limit counters; "memory" for single
	// instance or "redis" for distributed. Default is "memory".
	Storage *string `json:"storage" yaml:"storage" default:"memory" env:"PIKO_RATE_LIMIT_STORAGE" flag:"rateLimitStorage" usage:"Rate limit storage backend (memory or redis)." validate:"oneof=memory redis"`

	// CloudflareEnabled enables trust of the CF-Connecting-IP header from trusted
	// proxies; requires Cloudflare IP ranges in TrustedProxies. Default: false.
	CloudflareEnabled *bool `json:"cloudflareEnabled" yaml:"cloudflareEnabled" default:"false" env:"PIKO_RATE_LIMIT_CLOUDFLARE_ENABLED" flag:"rateLimitCloudflareEnabled" usage:"Trust CF-Connecting-IP header from trusted proxies (enable only with Cloudflare)."`

	// Enabled controls whether rate limiting is active,
	// disabled by default.
	//
	// Requires TrustedProxies configuration before enabling
	// when behind a reverse proxy.
	Enabled *bool `json:"enabled" yaml:"enabled" default:"false" env:"PIKO_RATE_LIMIT_ENABLED" flag:"rateLimitEnabled" usage:"Enable request rate limiting."`

	// HeadersEnabled controls whether rate limit headers
	// (X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset) are
	// included in responses, defaulting to true.
	HeadersEnabled *bool `json:"headersEnabled" yaml:"headersEnabled" default:"true" env:"PIKO_RATE_LIMIT_HEADERS_ENABLED" flag:"rateLimitHeaders" usage:"Include rate limit headers in responses."`

	// TrustedProxies is a list of CIDR ranges trusted to set X-Forwarded-For
	// headers. When a request comes from these CIDRs, forwarding headers are
	// used to extract the real client IP address.
	//
	// Common values include RFC 1918 private ranges: "10.0.0.0/8",
	// "172.16.0.0/12", "192.168.0.0/16". For Cloudflare, add their IP ranges
	// from https://www.cloudflare.com/ips/.
	TrustedProxies []string `json:"trustedProxies" yaml:"trustedProxies" env:"PIKO_RATE_LIMIT_TRUSTED_PROXIES" usage:"CIDR ranges of trusted proxies for X-Forwarded-For."`

	// ExemptPaths lists URL path prefixes that bypass rate limiting, such as
	// health checks and metrics endpoints. Default: ["/ping", "/_piko/health"].
	ExemptPaths []string `json:"exemptPaths" yaml:"exemptPaths" default:"/ping,/_piko/health" env:"PIKO_RATE_LIMIT_EXEMPT_PATHS" usage:"URL path prefixes exempt from rate limiting."`
}

// RateLimitTierConfig configures rate limits for a tier (global or actions).
type RateLimitTierConfig struct {
	// RequestsPerMinute is the maximum number of requests allowed per minute
	// per client IP. Default: 1000 for global, 100 for actions.
	RequestsPerMinute *int `json:"requestsPerMinute" yaml:"requestsPerMinute" default:"1000" env:"" flag:"" usage:"Maximum requests per minute per client."`

	// BurstSize is the maximum number of requests allowed in a single burst,
	// permitting short traffic spikes while still enforcing the overall rate.
	// Defaults to 50.
	BurstSize *int `json:"burstSize" yaml:"burstSize" default:"50" env:"" flag:"" usage:"Maximum burst size for rate limiting."`
}

// CookieSecurityConfig holds secure defaults for HTTP cookies. These settings
// help protect against session hijacking and CSRF attacks.
type CookieSecurityConfig struct {
	// DefaultSameSite specifies the default SameSite attribute for cookies.
	// Options: "Strict" (first-party only), "Lax" (default, top-level
	// navigations), or "None" (all contexts, requires Secure flag).
	DefaultSameSite *string `json:"defaultSameSite" yaml:"defaultSameSite" default:"Lax" env:"PIKO_COOKIE_SAMESITE" flag:"cookieSameSite" usage:"Default SameSite attribute for cookies." validate:"oneof=Strict Lax None"`

	// ForceHTTPOnly forces the HttpOnly flag on all cookies, which
	// prevents client-side JavaScript from accessing them. Default: true.
	ForceHTTPOnly *bool `json:"forceHttpOnly" yaml:"forceHttpOnly" default:"true" env:"PIKO_COOKIE_FORCE_HTTPONLY" flag:"cookieForceHttpOnly" usage:"Force HttpOnly flag on all cookies."`

	// ForceSecureOnHTTPS forces the Secure flag on cookies when served over
	// HTTPS, preventing them from being sent over unencrypted connections.
	// Default: true.
	ForceSecureOnHTTPS *bool `json:"forceSecureOnHttps" yaml:"forceSecureOnHttps" default:"true" env:"PIKO_COOKIE_FORCE_SECURE" flag:"cookieForceSecure" usage:"Force Secure flag on cookies when using HTTPS."`
}

// CSRFConfig holds CSRF token enforcement settings.
type CSRFConfig struct {
	// SecFetchSiteEnforcement controls whether browser requests identified by
	// the Sec-Fetch-Site header must include CSRF tokens (default: true).
	// Server-to-server calls without this header are unaffected.
	SecFetchSiteEnforcement *bool `json:"secFetchSiteEnforcement" yaml:"secFetchSiteEnforcement" default:"true" env:"PIKO_CSRF_SEC_FETCH_SITE_ENFORCEMENT" flag:"csrfSecFetchSiteEnforcement" usage:"Require CSRF tokens on browser requests identified by Sec-Fetch-Site header."`
}

// SandboxConfig configures filesystem sandboxing for Piko internals.
// It uses Go 1.24's os.Root to restrict file access, providing
// kernel-level protection against path traversal attacks.
type SandboxConfig struct {
	// Enabled controls whether filesystem sandboxing is active,
	// requiring Go 1.24 or later and defaulting to true.
	Enabled *bool `json:"enabled" yaml:"enabled" default:"true" env:"PIKO_SANDBOX_ENABLED" flag:"sandboxEnabled" usage:"Enable filesystem sandboxing (requires Go 1.24+)."`

	// AllowSymlinks controls whether symbolic links can be followed within
	// sandboxed directories, where disabling (the default) provides stronger
	// security but may break setups that rely on symlinks.
	AllowSymlinks *bool `json:"allowSymlinks" yaml:"allowSymlinks" default:"false" env:"PIKO_SANDBOX_ALLOW_SYMLINKS" flag:"sandboxAllowSymlinks" usage:"Allow following symbolic links in sandboxed directories."`

	// AllowedPaths is a list of additional absolute or relative paths
	// accessible to sandboxes, where CWD is always implicitly allowed
	// and an empty list restricts access to CWD and its subdirectories.
	AllowedPaths []string `json:"allowedPaths" yaml:"allowedPaths" env:"PIKO_SANDBOX_ALLOWED_PATHS" usage:"Additional paths accessible to sandboxed operations."`
}

// DatabaseConfig holds database connection configuration.
// Piko supports SQLite (default), PostgreSQL, and Cloudflare D1 backends.
type DatabaseConfig struct {
	// D1 holds the Cloudflare D1 database settings. Only used when Driver is "d1".
	D1 D1DatabaseConfig `json:"d1" yaml:"d1"`

	// Driver specifies which database backend to use.
	// Options: "sqlite" (default), "postgres", "d1".
	Driver *string `json:"driver" yaml:"driver" default:"sqlite" env:"PIKO_DATABASE_DRIVER" flag:"databaseDriver" usage:"Database backend driver (sqlite, postgres, or d1)." validate:"oneof=sqlite postgres d1"`

	// Postgres holds PostgreSQL settings. Only used when Driver is "postgres".
	Postgres PostgresDatabaseConfig `json:"postgres" yaml:"postgres"`
}

// PostgresDatabaseConfig holds PostgreSQL connection settings.
type PostgresDatabaseConfig struct {
	// URL is the PostgreSQL connection string.
	URL *string `json:"url" yaml:"url" default:"" env:"PIKO_DATABASE_POSTGRES_URL" flag:"databasePostgresUrl" usage:"PostgreSQL connection URL." summary:"hide"`

	// MaxConns is the maximum number of connections in the pool. Default is 10.
	MaxConns *int32 `json:"maxConns" yaml:"maxConns" default:"10" env:"PIKO_DATABASE_POSTGRES_MAX_CONNS" flag:"databasePostgresMaxConns" usage:"Maximum connections in PostgreSQL pool."`

	// MinConns is the minimum number of connections to keep in the pool.
	// Default is 2.
	MinConns *int32 `json:"minConns" yaml:"minConns" default:"2" env:"PIKO_DATABASE_POSTGRES_MIN_CONNS" flag:"databasePostgresMinConns" usage:"Minimum connections in PostgreSQL pool."`
}

// D1DatabaseConfig holds Cloudflare D1-specific settings.
// D1 is a serverless SQLite-compatible database for Cloudflare Workers.
type D1DatabaseConfig struct {
	// APIToken is the Cloudflare API token with D1 read and write permissions.
	// Create a token at https://dash.cloudflare.com/profile/api-tokens with the
	// required scopes: Account.D1:Read, Account.D1:Write.
	APIToken *string `json:"apiToken" yaml:"apiToken" default:"" env:"PIKO_DATABASE_D1_API_TOKEN" flag:"databaseD1ApiToken" usage:"Cloudflare API token for D1 access." summary:"hide"`

	// AccountID is the Cloudflare account identifier.
	// Find this in the Cloudflare dashboard under Account Home.
	AccountID *string `json:"accountId" yaml:"accountId" default:"" env:"PIKO_DATABASE_D1_ACCOUNT_ID" flag:"databaseD1AccountId" usage:"Cloudflare account ID."`

	// DatabaseID is the D1 database UUID.
	// Find this in the Cloudflare dashboard under Workers & Pages > D1.
	DatabaseID *string `json:"databaseId" yaml:"databaseId" default:"" env:"PIKO_DATABASE_D1_DATABASE_ID" flag:"databaseD1DatabaseId" usage:"D1 database UUID."`
}

// HealthProbeConfig holds configuration for the health check server and
// endpoints.
type HealthProbeConfig struct {
	// Port is the port number for the health check server. Default: "9090".
	Port *string `json:"port" yaml:"port" default:"9090" env:"PIKO_HEALTH_PROBE_PORT" flag:"healthProbePort" usage:"Port for the health check server."`

	// BindAddress is the network address to bind the health server to,
	// defaulting to "127.0.0.1" (localhost only) for security and
	// settable to "0.0.0.0" for external access such as Docker health
	// checks.
	BindAddress *string `json:"bindAddress" yaml:"bindAddress" default:"127.0.0.1" env:"PIKO_HEALTH_PROBE_BIND_ADDRESS" flag:"healthProbeBindAddress" usage:"Bind address for health check server."`

	// LivePath is the URL path for the liveness probe endpoint. Default: "/live".
	LivePath *string `json:"livePath" yaml:"livePath" default:"/live" env:"PIKO_HEALTH_PROBE_LIVE_PATH" flag:"healthProbeLivePath" usage:"URL path for liveness probe."`

	// ReadyPath is the URL path for the readiness probe endpoint. Default:
	// "/ready".
	ReadyPath *string `json:"readyPath" yaml:"readyPath" default:"/ready" env:"PIKO_HEALTH_PROBE_READY_PATH" flag:"healthProbeReadyPath" usage:"URL path for readiness probe."`

	// MetricsPath is the URL path for the Prometheus metrics endpoint. Default:
	// "/metrics".
	MetricsPath *string `json:"metricsPath" yaml:"metricsPath" default:"/metrics" env:"PIKO_HEALTH_PROBE_METRICS_PATH" flag:"healthProbeMetricsPath" usage:"URL path for Prometheus metrics."`

	// CheckTimeoutSeconds is the maximum time in seconds for each individual
	// health check. Default: 5.
	CheckTimeoutSeconds *int `json:"checkTimeoutSeconds" yaml:"checkTimeoutSeconds" default:"5" env:"PIKO_HEALTH_PROBE_CHECK_TIMEOUT" flag:"healthProbeCheckTimeout" usage:"Timeout in seconds for each health check."`

	// MetricsEnabled controls whether the Prometheus metrics endpoint is enabled.
	//
	// When true, OTEL metrics are exposed in Prometheus format at MetricsPath.
	// Defaults to true.
	MetricsEnabled *bool `json:"metricsEnabled" yaml:"metricsEnabled" default:"true" env:"PIKO_HEALTH_PROBE_METRICS_ENABLED" flag:"healthProbeMetricsEnabled" usage:"Enable Prometheus metrics endpoint."`

	// Enabled controls whether the health probe server starts. Default: true.
	Enabled *bool `json:"enabled" yaml:"enabled" default:"true" env:"PIKO_HEALTH_PROBE_ENABLED" flag:"healthProbeEnabled" usage:"Enable the health check server."`

	// AutoNextPort enables automatic port selection if the configured port
	// is already in use, trying subsequent ports up to 100 attempts
	// (default: false).
	AutoNextPort *bool `json:"autoNextPort" yaml:"autoNextPort" default:"false" env:"PIKO_HEALTH_PROBE_AUTO_PORT" flag:"healthProbeAutoNextPort" usage:"Automatically find the next port if the default is in use."`

	// ShutdownDrainDelay is the seconds to wait after marking the instance
	// as not ready before shutting down (default: 0, no delay).
	ShutdownDrainDelay *int `json:"shutdownDrainDelay" yaml:"shutdownDrainDelay" default:"0" env:"PIKO_SHUTDOWN_DRAIN_DELAY" flag:"shutdownDrainDelay" usage:"Seconds to wait after marking not-ready before shutting down (e.g. 3, 5)."`

	// TLS configures TLS for the health probe server. When not configured,
	// the health server remains plain HTTP regardless of the main server.
	TLS HealthTLSConfig `json:"tls" yaml:"tls"`
}

// StorageConfig holds settings for the storage service, including presigned URL
// support.
type StorageConfig struct {
	// PublicBaseURL is the base URL for generating public storage URLs,
	// where an empty value produces relative paths and a non-empty value
	// produces absolute URLs, required when the website and CMS/API run
	// on different ports or hosts (e.g., "http://localhost:8080").
	PublicBaseURL *string `json:"publicBaseUrl" yaml:"publicBaseUrl" env:"PIKO_STORAGE_PUBLIC_BASE_URL" flag:"storagePublicBaseUrl" usage:"Base URL for public storage URLs."`

	// Presign sets up presigned URL creation at service level for storage
	// providers that do not support native presigned URLs (e.g. disk provider).
	Presign StoragePresignConfig `json:"presign" yaml:"presign"`
}

// StoragePresignConfig configures presigned URL support for storage operations.
// It generates HMAC-signed tokens for storage providers that lack native
// presigned URL support, which the HTTP upload handler then validates.
type StoragePresignConfig struct {
	// Secret is the HMAC secret for signing presign tokens.
	//
	// Must be at least 32 bytes. If empty, a random secret is generated for
	// the process lifetime. For multi-instance deployments, set this to
	// ensure all instances share the same secret. Accepts hex, base64, or
	// raw string format.
	Secret *string `json:"secret" yaml:"secret" env:"PIKO_STORAGE_PRESIGN_SECRET" flag:"storagePresignSecret" usage:"HMAC secret for signing presign tokens (min 32 bytes)." summary:"hide"`

	// BaseURL is the base URL for generating presigned upload URLs,
	// defaulting to relative paths when empty and useful for CDN or
	// proxy setups where the upload endpoint is on a different domain
	// (e.g., "https://upload.example.com").
	BaseURL *string `json:"baseUrl" yaml:"baseUrl" env:"PIKO_STORAGE_PRESIGN_BASE_URL" flag:"storagePresignBaseUrl" usage:"Base URL for presigned upload URLs."`

	// DefaultExpiry specifies how long presigned URLs remain valid by default.
	// Requests can specify shorter durations but not longer than MaxExpiry.
	DefaultExpiry *time.Duration `json:"defaultExpiry" yaml:"defaultExpiry" default:"15m" env:"PIKO_STORAGE_PRESIGN_DEFAULT_EXPIRY" flag:"storagePresignDefaultExpiry" usage:"Default validity duration for presigned URLs."`

	// MaxExpiry is the maximum validity duration for presigned URLs; longer
	// requests are capped to this value. Format: Go duration (e.g., "1h",
	// "24h"); default: 1h.
	MaxExpiry *time.Duration `json:"maxExpiry" yaml:"maxExpiry" default:"1h" env:"PIKO_STORAGE_PRESIGN_MAX_EXPIRY" flag:"storagePresignMaxExpiry" usage:"Maximum validity duration for presigned URLs."`

	// DefaultMaxSize is the default maximum upload size in bytes (default: 100
	// MB). Individual presign requests can specify smaller limits.
	DefaultMaxSize *int64 `json:"defaultMaxSize" yaml:"defaultMaxSize" default:"104857600" env:"PIKO_STORAGE_PRESIGN_DEFAULT_MAX_SIZE" flag:"storagePresignDefaultMaxSize" usage:"Default maximum upload size in bytes."`

	// MaxMaxSize is the absolute maximum upload size in bytes; requests for larger
	// limits are capped to this value. Defaults to 1073741824 (1 GB).
	MaxMaxSize *int64 `json:"maxMaxSize" yaml:"maxMaxSize" default:"1073741824" env:"PIKO_STORAGE_PRESIGN_MAX_MAX_SIZE" flag:"storagePresignMaxMaxSize" usage:"Absolute maximum upload size in bytes."`

	// RateLimitPerMinute is the per-IP rate limit for presigned upload requests.
	// Set to 0 to disable rate limiting; default is 50.
	RateLimitPerMinute *int `json:"rateLimitPerMinute" yaml:"rateLimitPerMinute" default:"50" env:"PIKO_STORAGE_PRESIGN_RATE_LIMIT" flag:"storagePresignRateLimit" usage:"Per-IP rate limit for presigned uploads per minute."`
}
