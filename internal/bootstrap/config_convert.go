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
	"time"

	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/lifecycle/lifecycle_domain"
	"piko.sh/piko/internal/logger/logger_adapters/driver_handlers"
	"piko.sh/piko/internal/security/security_adapters"
)

const (
	// defaultTraceSampleRate is the default OTLP trace sampling rate.
	defaultTraceSampleRate = 0.05

	// defaultRequestTimeoutSeconds is the default HTTP request timeout.
	defaultRequestTimeoutSeconds = 60

	// defaultMaxConcurrentRequests is the default limit on concurrent in-flight
	// requests, high enough for normal deployments but preventing goroutine
	// exhaustion under attack.
	defaultMaxConcurrentRequests = 10_000

	// defaultGlobalRequestsPerMinute is the default global rate limit.
	defaultGlobalRequestsPerMinute = 1000

	// defaultGlobalBurstSize is the default burst size for global rate limits.
	defaultGlobalBurstSize = 50

	// defaultActionRequestsPerMinute is the default action endpoint rate limit.
	defaultActionRequestsPerMinute = 100

	// defaultActionBurstSize is the default burst size for action rate limits.
	defaultActionBurstSize = 50

	// fontWeightNormal is the CSS font-weight value for normal (regular) text.
	fontWeightNormal = 400

	// fontWeightBold is the CSS font-weight value for bold text.
	fontWeightBold = 700
)

// NewDaemonConfig converts the pointer-based ServerConfig into a value-type
// DaemonConfig for the daemon service.
//
// Takes sc (*ServerConfig) which provides the server configuration
// values to convert.
//
// Returns daemon_domain.DaemonConfig which contains the resolved
// configuration values with defaults applied.
func NewDaemonConfig(sc *ServerConfig) daemon_domain.DaemonConfig {
	return daemon_domain.DaemonConfig{
		NetworkPort:         deref(sc.Network.Port, "8080"),
		NetworkAutoNextPort: deref(sc.Network.AutoNextPort, false),
		HealthEnabled:       deref(sc.HealthProbe.Enabled, true),
		HealthPort:          deref(sc.HealthProbe.Port, "9090"),
		HealthBindAddress:   deref(sc.HealthProbe.BindAddress, "127.0.0.1"),
		HealthAutoNextPort:  deref(sc.HealthProbe.AutoNextPort, false),
		HealthLivePath:      deref(sc.HealthProbe.LivePath, "/live"),
		HealthReadyPath:     deref(sc.HealthProbe.ReadyPath, "/ready"),
		ShutdownDrainDelay:  time.Duration(deref(sc.HealthProbe.ShutdownDrainDelay, 0)) * time.Second,
		TLS:                 NewTLSValues(&sc.Network.TLS),
		TLSRedirectHTTPPort: deref(sc.Network.TLS.RedirectHTTPPort, ""),
		HealthTLS:           NewHealthTLSValues(&sc.HealthProbe.TLS),
	}
}

// NewLifecyclePathsConfig converts the pointer-based ServerConfig into a
// value-type LifecyclePathsConfig for the lifecycle service.
//
// Takes sc (*ServerConfig) which provides the server configuration
// values to convert.
//
// Returns lifecycle_domain.LifecyclePathsConfig which contains the
// resolved source directory paths with defaults applied.
func NewLifecyclePathsConfig(sc *ServerConfig) lifecycle_domain.LifecyclePathsConfig {
	return lifecycle_domain.LifecyclePathsConfig{
		BaseDir:             deref(sc.Paths.BaseDir, "."),
		PagesSourceDir:      deref(sc.Paths.PagesSourceDir, "pages"),
		PartialsSourceDir:   deref(sc.Paths.PartialsSourceDir, "partials"),
		ComponentsSourceDir: deref(sc.Paths.ComponentsSourceDir, "components"),
		EmailsSourceDir:     deref(sc.Paths.EmailsSourceDir, "emails"),
		AssetsSourceDir:     deref(sc.Paths.AssetsSourceDir, "lib"),
		I18nSourceDir:       deref(sc.Paths.I18nSourceDir, "locales"),
	}
}

// NewAnnotatorPathsConfig converts the pointer-based ServerConfig into a
// value-type AnnotatorPathsConfig for the annotator service.
//
// Takes sc (*ServerConfig) which provides the server configuration
// values to convert.
//
// Returns annotator_domain.AnnotatorPathsConfig which contains the
// resolved annotator paths with defaults applied.
func NewAnnotatorPathsConfig(sc *ServerConfig) annotator_domain.AnnotatorPathsConfig {
	return annotator_domain.AnnotatorPathsConfig{
		PagesSourceDir:    deref(sc.Paths.PagesSourceDir, "pages"),
		EmailsSourceDir:   deref(sc.Paths.EmailsSourceDir, "emails"),
		PdfsSourceDir:     deref(sc.Paths.PdfsSourceDir, "pdfs"),
		PartialsSourceDir: deref(sc.Paths.PartialsSourceDir, "partials"),
		E2ESourceDir:      deref(sc.Paths.E2ESourceDir, "e2e"),
		AssetsSourceDir:   deref(sc.Paths.AssetsSourceDir, "lib"),
		PartialServePath:  deref(sc.Paths.PartialServePath, "/_piko/partials"),
		ArtefactServePath: deref(sc.Paths.ArtefactServePath, "/_piko/assets"),
	}
}

// NewGeneratorPathsConfig converts the pointer-based ServerConfig into a
// value-type GeneratorPathsConfig for the generator service.
//
// Takes sc (*ServerConfig) which provides the server configuration
// values to convert.
//
// Returns generator_domain.GeneratorPathsConfig which contains the
// resolved generator paths with defaults applied.
func NewGeneratorPathsConfig(sc *ServerConfig) generator_domain.GeneratorPathsConfig {
	return generator_domain.GeneratorPathsConfig{
		BaseDir:        deref(sc.Paths.BaseDir, "."),
		PagesSourceDir: deref(sc.Paths.PagesSourceDir, "pages"),
		E2ESourceDir:   deref(sc.Paths.E2ESourceDir, "e2e"),
		BaseServePath:  deref(sc.Paths.BaseServePath, "/"),
	}
}

// NewOtelSetupConfig converts the pointer-based ServerConfig into a value-type
// OtelSetupConfig for the OTEL setup.
//
// Takes sc (*ServerConfig) which provides the server configuration
// values to convert.
//
// Returns driver_handlers.OtelSetupConfig which contains the resolved
// OpenTelemetry settings with defaults applied.
func NewOtelSetupConfig(sc *ServerConfig) driver_handlers.OtelSetupConfig {
	return driver_handlers.OtelSetupConfig{
		Enabled:         deref(sc.Otlp.Enabled, false),
		Endpoint:        deref(sc.Otlp.Endpoint, "localhost:4317"),
		Protocol:        deref(sc.Otlp.Protocol, "http"),
		TraceSampleRate: deref(sc.Otlp.TraceSampleRate, defaultTraceSampleRate),
		TLSInsecure:     deref(sc.Otlp.TLS.Insecure, true),
		Headers:         sc.Otlp.Headers,
	}
}

// NewRouterConfig converts a ServerConfig into a RouterConfig for the HTTP
// router. The caller must supply pre-built security header values and the
// reporting configuration separately, because the CORP override and reporting
// source differ between build strategies.
//
// Takes sc (*ServerConfig) which provides the server configuration
// values to convert.
// Takes shValues (security_adapters.SecurityHeadersValues) which supplies
// the pre-built security header values.
// Takes reportingConfig (config.ReportingConfig) which supplies the
// reporting endpoint configuration.
//
// Returns *daemon_domain.RouterConfig which contains the resolved router
// configuration with defaults applied.
func NewRouterConfig(
	sc *ServerConfig,
	shValues security_adapters.SecurityHeadersValues,
	reportingConfig config.ReportingConfig,
) *daemon_domain.RouterConfig {
	forceHTTPS := deref(sc.Network.ForceHTTPS, false)
	if deref(sc.Network.TLS.Enabled, false) {
		forceHTTPS = true
	}

	return &daemon_domain.RouterConfig{
		Port:                  deref(sc.Network.Port, "8080"),
		PublicDomain:          deref(sc.Network.PublicDomain, "localhost:8080"),
		ForceHTTPS:            forceHTTPS,
		RequestTimeoutSeconds: deref(sc.Network.RequestTimeoutSeconds, defaultRequestTimeoutSeconds),
		MaxConcurrentRequests: deref(sc.Network.MaxConcurrentRequests, defaultMaxConcurrentRequests),
		DistServePath:         deref(sc.Paths.DistServePath, "/_piko/dist"),
		ArtefactServePath:     deref(sc.Paths.ArtefactServePath, "/_piko/assets"),
		SecurityHeaders:       shValues,
		RateLimit:             NewRateLimitValues(&sc.Security.RateLimit),
		Reporting:             NewReportingValues(new(reportingConfig)),
		WatchMode:             deref(sc.Build.WatchMode, false),
	}
}

// NewSecurityHeadersValues converts the pointer-based SecurityHeadersConfig
// into a value-type SecurityHeadersValues for the security middleware.
//
// Takes headersConfig (*config.SecurityHeadersConfig) which provides the security
// header settings to convert.
//
// Returns security_adapters.SecurityHeadersValues which contains the
// resolved security header values with defaults applied.
func NewSecurityHeadersValues(headersConfig *config.SecurityHeadersConfig) security_adapters.SecurityHeadersValues {
	return security_adapters.SecurityHeadersValues{
		XFrameOptions:             deref(headersConfig.XFrameOptions, "DENY"),
		XContentTypeOptions:       deref(headersConfig.XContentTypeOptions, "nosniff"),
		ReferrerPolicy:            deref(headersConfig.ReferrerPolicy, "strict-origin-when-cross-origin"),
		ContentSecurityPolicy:     deref(headersConfig.ContentSecurityPolicy, ""),
		StrictTransportSecurity:   deref(headersConfig.StrictTransportSecurity, "max-age=31536000; includeSubDomains"),
		CrossOriginOpenerPolicy:   deref(headersConfig.CrossOriginOpenerPolicy, "same-origin"),
		CrossOriginResourcePolicy: deref(headersConfig.CrossOriginResourcePolicy, "same-origin"),
		PermissionsPolicy:         deref(headersConfig.PermissionsPolicy, ""),
		Enabled:                   deref(headersConfig.Enabled, true),
		StripServerHeader:         deref(headersConfig.StripServerHeader, true),
		StripPoweredByHeader:      deref(headersConfig.StripPoweredByHeader, true),
	}
}

// NewCookieSecurityValues converts the pointer-based CookieSecurityConfig
// into a value-type CookieSecurityValues for the secure cookie writer.
//
// Takes cookieConfig (*config.CookieSecurityConfig) which provides the cookie
// security settings to convert.
//
// Returns security_adapters.CookieSecurityValues which contains the
// resolved cookie security values with defaults applied.
func NewCookieSecurityValues(cookieConfig *config.CookieSecurityConfig) security_adapters.CookieSecurityValues {
	return security_adapters.CookieSecurityValues{
		DefaultSameSite:    deref(cookieConfig.DefaultSameSite, "Lax"),
		ForceHTTPOnly:      deref(cookieConfig.ForceHTTPOnly, true),
		ForceSecureOnHTTPS: deref(cookieConfig.ForceSecureOnHTTPS, true),
	}
}

// NewReportingValues converts a ReportingConfig into a pre-built
// ReportingValues for the security middleware.
//
// Takes reportingConfig (*config.ReportingConfig) which provides the reporting
// endpoint configuration to convert.
//
// Returns security_adapters.ReportingValues which contains the
// pre-built reporting header value.
func NewReportingValues(reportingConfig *config.ReportingConfig) security_adapters.ReportingValues {
	return security_adapters.ReportingValues{
		HeaderValue: reportingConfig.BuildHeader(),
	}
}

// NewRateLimitValues converts the pointer-based RateLimitConfig into a
// value-type RateLimitValues for the rate limiting middleware.
//
// Takes rateLimitConfig (*config.RateLimitConfig) which provides the rate limit
// settings to convert.
//
// Returns security_adapters.RateLimitValues which contains the resolved
// rate limit values with defaults applied.
func NewRateLimitValues(rateLimitConfig *config.RateLimitConfig) security_adapters.RateLimitValues {
	return security_adapters.RateLimitValues{
		Storage:        deref(rateLimitConfig.Storage, "memory"),
		TrustedProxies: rateLimitConfig.TrustedProxies,
		ExemptPaths:    rateLimitConfig.ExemptPaths,
		Global: security_adapters.RateLimitTierValues{
			RequestsPerMinute: deref(rateLimitConfig.Global.RequestsPerMinute, defaultGlobalRequestsPerMinute),
			BurstSize:         deref(rateLimitConfig.Global.BurstSize, defaultGlobalBurstSize),
		},
		Actions: security_adapters.RateLimitTierValues{
			RequestsPerMinute: deref(rateLimitConfig.Actions.RequestsPerMinute, defaultActionRequestsPerMinute),
			BurstSize:         deref(rateLimitConfig.Actions.BurstSize, defaultActionBurstSize),
		},
		CloudflareEnabled: deref(rateLimitConfig.CloudflareEnabled, false),
		Enabled:           deref(rateLimitConfig.Enabled, false),
		HeadersEnabled:    deref(rateLimitConfig.HeadersEnabled, true),
	}
}
