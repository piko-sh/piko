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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/logger/logger_dto"
)

func TestOverrideOptions_PopulateServerConfigOverrides(t *testing.T) {
	t.Parallel()

	headers := config.SecurityHeadersConfig{XFrameOptions: new("SAMEORIGIN")}
	cookies := config.CookieSecurityConfig{DefaultSameSite: new("Strict")}
	rateLimit := config.RateLimitConfig{Global: config.RateLimitTierConfig{RequestsPerMinute: new(42)}}
	sandbox := config.SandboxConfig{Enabled: new(false)}
	reporting := config.ReportingConfig{Enabled: new(true)}
	awsKMS := config.AWSKMSConfig{KeyID: new("aws-key")}
	gcpKMS := config.GCPKMSConfig{ProjectID: new("gcp-project")}
	otlpHeaders := map[string]string{"authorisation": "token"}

	c := NewContainer(
		WithPublicDomain("example.test"),
		WithForceHTTPS(true),
		WithRequestTimeout(30*time.Second),
		WithMaxConcurrentRequests(7),
		WithActionMaxBodyBytes(1024),
		WithMaxMultipartFormBytes(2048),
		WithDefaultMaxSSEDuration(time.Minute),
		WithAutoNextPort(true),
		WithBindAddress("127.0.0.1"),
		WithEncryptionKey("encryption-key"),
		WithDataKeyCacheTTL(time.Hour),
		WithDataKeyCacheMaxSize(99),
		WithSecurityHeaders(headers),
		WithCookieSecurity(cookies),
		WithRateLimit(rateLimit),
		WithSandbox(sandbox),
		WithReporting(reporting),
		WithCaptcha(CaptchaOptions{SiteKey: "site", SecretKey: "secret", ScoreThreshold: 0.7}),
		WithAWSKMS(awsKMS),
		WithGCPKMS(gcpKMS),
		WithDeprecatedKeyIDs("old-1", "old-2"),
		WithLogger(logger_dto.Config{Level: "warn", AddSource: true}),
		WithLogLevel("debug"),
		WithDatabaseDriver("postgres"),
		WithPostgresURL("postgres://localhost/db"),
		WithPostgresMaxConns(10),
		WithPostgresMinConns(2),
		WithD1APIToken("d1-token"),
		WithD1AccountID("d1-account"),
		WithD1DatabaseID("d1-database"),
		WithOTLP(config.OtlpConfig{Protocol: new("grpc")}),
		WithOTLPEnabled(true),
		WithOTLPEndpoint("otel:4317"),
		WithOTLPProtocol("http"),
		WithOTLPTraceSampleRate(0.25),
		WithOTLPHeaders(otlpHeaders),
		WithOTLPInsecureTLS(true),
		WithHealthEnabled(true),
		WithHealthBindAddress("0.0.0.0"),
		WithHealthMetricsEnabled(true),
		WithHealthMetricsPath("/metrics"),
		WithHealthLivePath("/live"),
		WithHealthReadyPath("/ready"),
		WithHealthCheckTimeout(5*time.Second),
		WithHealthAutoNextPort(true),
		WithHealthProbePort(9191),
		WithBaseDir("/srv/site"),
		WithComponentsSourceDir("components"),
		WithPagesSourceDir("pages"),
		WithPartialsSourceDir("partials"),
		WithEmailsSourceDir("emails"),
		WithPdfsSourceDir("pdfs"),
		WithE2ESourceDir("e2e"),
		WithAssetsSourceDir("assets"),
		WithI18nSourceDir("i18n"),
		WithBaseServePath("/base"),
		WithPartialServePath("/partials"),
		WithActionServePath("/actions"),
		WithActionCompression(true),
		WithMaxParallelBatchWorkers(3),
		WithLibServePath("/lib"),
		WithDistServePath("/dist"),
		WithArtefactServePath("/artefacts"),
		WithDefaultServeMode("render"),
		WithStoragePresign(config.StoragePresignConfig{Secret: new("initial")}),
		WithStoragePresignSecret("presign-secret"),
		WithStoragePresignDefaultExpiry(time.Minute),
		WithStoragePresignMaxExpiry(time.Hour),
		WithStoragePresignDefaultMaxSize(100),
		WithStoragePresignMaxMaxSize(200),
		WithStoragePresignRateLimit(60),
		WithI18nDefaultLocale("en-GB"),
		WithCSSImportMaxDepth(4),
		WithCSSImportMaxBytes(4096),
	)

	overrides := c.configServerOverrides
	require.NotNil(t, overrides)

	network := overrides.Network
	assert.Equal(t, "example.test", *network.PublicDomain)
	assert.True(t, *network.ForceHTTPS)
	assert.Equal(t, 30, *network.RequestTimeoutSeconds)
	assert.Equal(t, 7, *network.MaxConcurrentRequests)
	assert.Equal(t, int64(1024), *network.ActionMaxBodyBytes)
	assert.Equal(t, int64(2048), *network.MaxMultipartFormBytes)
	assert.Equal(t, 60, *network.DefaultMaxSSEDurationSeconds)
	assert.True(t, *network.AutoNextPort)
	assert.Equal(t, "127.0.0.1", *network.BindAddress)
	assert.True(t, *network.ActionCompression)
	assert.Equal(t, 3, *network.MaxParallelBatchWorkers)

	security := overrides.Security
	assert.Equal(t, "encryption-key", *security.EncryptionKey)
	assert.Equal(t, time.Hour, *security.DataKeyCacheTTL)
	assert.Equal(t, 99, *security.DataKeyCacheMaxSize)
	assert.Equal(t, headers, security.Headers)
	assert.Equal(t, cookies, security.Cookies)
	assert.Equal(t, rateLimit, security.RateLimit)
	assert.Equal(t, sandbox, security.Sandbox)
	assert.Equal(t, reporting, security.Reporting)
	assert.Equal(t, "site", *security.CaptchaSiteKey)
	assert.Equal(t, "secret", *security.CaptchaSecretKey)
	assert.InDelta(t, 0.7, *security.CaptchaScoreThreshold, 0.0001)
	assert.Equal(t, awsKMS, security.AWSKMS)
	assert.Equal(t, gcpKMS, security.GCPKMS)
	assert.Equal(t, []string{"old-1", "old-2"}, security.DeprecatedKeyIDs)

	assert.Equal(t, "debug", overrides.Logger.Level)
	assert.True(t, overrides.Logger.AddSource)

	database := overrides.Database
	assert.Equal(t, "postgres", *database.Driver)
	assert.Equal(t, "postgres://localhost/db", *database.Postgres.URL)
	assert.Equal(t, int32(10), *database.Postgres.MaxConns)
	assert.Equal(t, int32(2), *database.Postgres.MinConns)
	assert.Equal(t, "d1-token", *database.D1.APIToken)
	assert.Equal(t, "d1-account", *database.D1.AccountID)
	assert.Equal(t, "d1-database", *database.D1.DatabaseID)

	otlp := overrides.Otlp
	assert.True(t, *otlp.Enabled)
	assert.Equal(t, "otel:4317", *otlp.Endpoint)
	assert.Equal(t, "http", *otlp.Protocol)
	assert.InDelta(t, 0.25, *otlp.TraceSampleRate, 0.0001)
	assert.Equal(t, otlpHeaders, otlp.Headers)
	assert.True(t, *otlp.TLS.Insecure)

	health := overrides.HealthProbe
	assert.True(t, *health.Enabled)
	assert.Equal(t, "0.0.0.0", *health.BindAddress)
	assert.True(t, *health.MetricsEnabled)
	assert.Equal(t, "/metrics", *health.MetricsPath)
	assert.Equal(t, "/live", *health.LivePath)
	assert.Equal(t, "/ready", *health.ReadyPath)
	assert.Equal(t, 5, *health.CheckTimeoutSeconds)
	assert.True(t, *health.AutoNextPort)
	assert.Equal(t, "9191", *health.Port)

	paths := overrides.Paths
	assert.Equal(t, "/srv/site", *paths.BaseDir)
	assert.Equal(t, "components", *paths.ComponentsSourceDir)
	assert.Equal(t, "pages", *paths.PagesSourceDir)
	assert.Equal(t, "partials", *paths.PartialsSourceDir)
	assert.Equal(t, "emails", *paths.EmailsSourceDir)
	assert.Equal(t, "pdfs", *paths.PdfsSourceDir)
	assert.Equal(t, "e2e", *paths.E2ESourceDir)
	assert.Equal(t, "assets", *paths.AssetsSourceDir)
	assert.Equal(t, "i18n", *paths.I18nSourceDir)
	assert.Equal(t, "/base", *paths.BaseServePath)
	assert.Equal(t, "/partials", *paths.PartialServePath)
	assert.Equal(t, "/actions", *paths.ActionServePath)
	assert.Equal(t, "/lib", *paths.LibServePath)
	assert.Equal(t, "/dist", *paths.DistServePath)
	assert.Equal(t, "/artefacts", *paths.ArtefactServePath)

	assert.Equal(t, "render", *overrides.Build.DefaultServeMode)
	assert.Equal(t, 4, *overrides.Build.CSSImportMaxDepth)
	assert.Equal(t, 4096, *overrides.Build.CSSImportMaxBytes)

	presign := overrides.Storage.Presign
	assert.Equal(t, "presign-secret", *presign.Secret)
	assert.Equal(t, time.Minute, *presign.DefaultExpiry)
	assert.Equal(t, time.Hour, *presign.MaxExpiry)
	assert.Equal(t, int64(100), *presign.DefaultMaxSize)
	assert.Equal(t, int64(200), *presign.MaxMaxSize)
	assert.Equal(t, 60, *presign.RateLimitPerMinute)

	assert.Equal(t, "en-GB", *overrides.I18nDefaultLocale)
}

func TestWithCaptcha_ZeroScoreThresholdKeepsDefault(t *testing.T) {
	t.Parallel()

	c := NewContainer(WithCaptcha(CaptchaOptions{SiteKey: "site", SecretKey: "secret"}))

	require.NotNil(t, c.configServerOverrides)
	assert.Equal(t, "site", *c.configServerOverrides.Security.CaptchaSiteKey)
	assert.Nil(t, c.configServerOverrides.Security.CaptchaScoreThreshold)
}
