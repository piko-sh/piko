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
	"piko.sh/piko/internal/config"
)

func TestNewDaemonConfig(t *testing.T) {
	t.Run("all pointers set", func(t *testing.T) {
		sc := &config.ServerConfig{
			Network: config.NetworkConfig{
				Port:         new("3000"),
				AutoNextPort: new(true),
			},
			HealthProbe: config.HealthProbeConfig{
				Enabled:      new(true),
				Port:         new("9999"),
				BindAddress:  new("0.0.0.0"),
				AutoNextPort: new(true),
				LivePath:     new("/healthz"),
				ReadyPath:    new("/readyz"),
			},
		}

		dc := NewDaemonConfig(sc)

		assert.Equal(t, "3000", dc.NetworkPort)
		assert.True(t, dc.NetworkAutoNextPort)
		assert.True(t, dc.HealthEnabled)
		assert.Equal(t, "9999", dc.HealthPort)
		assert.Equal(t, "0.0.0.0", dc.HealthBindAddress)
		assert.True(t, dc.HealthAutoNextPort)
		assert.Equal(t, "/healthz", dc.HealthLivePath)
		assert.Equal(t, "/readyz", dc.HealthReadyPath)
	})

	t.Run("all pointers nil uses defaults", func(t *testing.T) {
		sc := &config.ServerConfig{}

		dc := NewDaemonConfig(sc)

		assert.Equal(t, "8080", dc.NetworkPort)
		assert.False(t, dc.NetworkAutoNextPort)
		assert.True(t, dc.HealthEnabled)
		assert.Equal(t, "9090", dc.HealthPort)
		assert.Equal(t, "127.0.0.1", dc.HealthBindAddress)
		assert.False(t, dc.HealthAutoNextPort)
		assert.Equal(t, "/live", dc.HealthLivePath)
		assert.Equal(t, "/ready", dc.HealthReadyPath)
	})

	t.Run("explicit false overrides true default", func(t *testing.T) {
		sc := &config.ServerConfig{
			HealthProbe: config.HealthProbeConfig{
				Enabled: new(false),
			},
		}

		dc := NewDaemonConfig(sc)

		assert.False(t, dc.HealthEnabled)
	})

	t.Run("ShutdownDrainDelay nil defaults to zero", func(t *testing.T) {
		sc := &config.ServerConfig{}

		dc := NewDaemonConfig(sc)

		assert.Equal(t, time.Duration(0), dc.ShutdownDrainDelay)
	})

	t.Run("ShutdownDrainDelay explicit value preserved", func(t *testing.T) {
		sc := &config.ServerConfig{
			HealthProbe: config.HealthProbeConfig{
				ShutdownDrainDelay: new(5),
			},
		}

		dc := NewDaemonConfig(sc)

		assert.Equal(t, 5*time.Second, dc.ShutdownDrainDelay)
	})
}

func TestNewLifecyclePathsConfig(t *testing.T) {
	t.Run("all pointers set", func(t *testing.T) {
		sc := &config.ServerConfig{
			Paths: config.PathsConfig{
				BaseDir:             new("/app"),
				PagesSourceDir:      new("views"),
				PartialsSourceDir:   new("fragments"),
				ComponentsSourceDir: new("widgets"),
				EmailsSourceDir:     new("mail"),
				AssetsSourceDir:     new("assets"),
				I18nSourceDir:       new("translations"),
			},
		}

		lc := NewLifecyclePathsConfig(sc)

		assert.Equal(t, "/app", lc.BaseDir)
		assert.Equal(t, "views", lc.PagesSourceDir)
		assert.Equal(t, "fragments", lc.PartialsSourceDir)
		assert.Equal(t, "widgets", lc.ComponentsSourceDir)
		assert.Equal(t, "mail", lc.EmailsSourceDir)
		assert.Equal(t, "assets", lc.AssetsSourceDir)
		assert.Equal(t, "translations", lc.I18nSourceDir)
	})

	t.Run("all pointers nil uses defaults", func(t *testing.T) {
		sc := &config.ServerConfig{}

		lc := NewLifecyclePathsConfig(sc)

		assert.Equal(t, ".", lc.BaseDir)
		assert.Equal(t, "pages", lc.PagesSourceDir)
		assert.Equal(t, "partials", lc.PartialsSourceDir)
		assert.Equal(t, "components", lc.ComponentsSourceDir)
		assert.Equal(t, "emails", lc.EmailsSourceDir)
		assert.Equal(t, "lib", lc.AssetsSourceDir)
		assert.Equal(t, "locales", lc.I18nSourceDir)
	})
}

func TestNewAnnotatorPathsConfig(t *testing.T) {
	t.Run("all pointers set", func(t *testing.T) {
		sc := &config.ServerConfig{
			Paths: config.PathsConfig{
				PagesSourceDir:    new("views"),
				EmailsSourceDir:   new("mail"),
				PartialsSourceDir: new("fragments"),
				E2ESourceDir:      new("tests"),
				AssetsSourceDir:   new("assets"),
				PartialServePath:  new("/parts"),
				ArtefactServePath: new("/compiled"),
			},
		}

		ac := NewAnnotatorPathsConfig(sc)

		assert.Equal(t, "views", ac.PagesSourceDir)
		assert.Equal(t, "mail", ac.EmailsSourceDir)
		assert.Equal(t, "fragments", ac.PartialsSourceDir)
		assert.Equal(t, "tests", ac.E2ESourceDir)
		assert.Equal(t, "assets", ac.AssetsSourceDir)
		assert.Equal(t, "/parts", ac.PartialServePath)
		assert.Equal(t, "/compiled", ac.ArtefactServePath)
	})

	t.Run("all pointers nil uses defaults", func(t *testing.T) {
		sc := &config.ServerConfig{}

		ac := NewAnnotatorPathsConfig(sc)

		assert.Equal(t, "pages", ac.PagesSourceDir)
		assert.Equal(t, "emails", ac.EmailsSourceDir)
		assert.Equal(t, "partials", ac.PartialsSourceDir)
		assert.Equal(t, "e2e", ac.E2ESourceDir)
		assert.Equal(t, "lib", ac.AssetsSourceDir)
		assert.Equal(t, "/_piko/partials", ac.PartialServePath)
		assert.Equal(t, "/_piko/assets", ac.ArtefactServePath)
	})
}

func TestNewOtelSetupConfig(t *testing.T) {
	t.Run("all pointers set", func(t *testing.T) {
		sc := &config.ServerConfig{
			Otlp: config.OtlpConfig{
				Enabled:         new(true),
				Endpoint:        new("otel.example.com:4317"),
				Protocol:        new("grpc"),
				TraceSampleRate: new(0.1),
				TLS:             config.OtlpTLSConfig{Insecure: new(false)},
				Headers:         map[string]string{"Authorization": "Bearer token"},
			},
		}

		oc := NewOtelSetupConfig(sc)

		assert.True(t, oc.Enabled)
		assert.Equal(t, "otel.example.com:4317", oc.Endpoint)
		assert.Equal(t, "grpc", oc.Protocol)
		assert.InDelta(t, 0.1, oc.TraceSampleRate, 1e-9)
		assert.False(t, oc.TLSInsecure)
		assert.Equal(t, map[string]string{"Authorization": "Bearer token"}, oc.Headers)
	})

	t.Run("all pointers nil uses defaults", func(t *testing.T) {
		sc := &config.ServerConfig{}

		oc := NewOtelSetupConfig(sc)

		assert.False(t, oc.Enabled)
		assert.Equal(t, "localhost:4317", oc.Endpoint)
		assert.Equal(t, "http", oc.Protocol)
		assert.InDelta(t, 0.05, oc.TraceSampleRate, 1e-9)
		assert.True(t, oc.TLSInsecure)
		assert.Nil(t, oc.Headers)
	})
}

func TestNewSecurityHeadersValues(t *testing.T) {
	t.Run("all pointers set", func(t *testing.T) {
		headersConfig := &config.SecurityHeadersConfig{
			XFrameOptions:             new("SAMEORIGIN"),
			XContentTypeOptions:       new("nosniff"),
			ReferrerPolicy:            new("no-referrer"),
			ContentSecurityPolicy:     new("default-src 'self'"),
			StrictTransportSecurity:   new("max-age=63072000"),
			CrossOriginOpenerPolicy:   new("same-origin-allow-popups"),
			CrossOriginResourcePolicy: new("cross-origin"),
			PermissionsPolicy:         new("camera=()"),
			Enabled:                   new(false),
			StripServerHeader:         new(false),
			StripPoweredByHeader:      new(false),
		}

		sv := NewSecurityHeadersValues(headersConfig)

		assert.Equal(t, "SAMEORIGIN", sv.XFrameOptions)
		assert.Equal(t, "nosniff", sv.XContentTypeOptions)
		assert.Equal(t, "no-referrer", sv.ReferrerPolicy)
		assert.Equal(t, "default-src 'self'", sv.ContentSecurityPolicy)
		assert.Equal(t, "max-age=63072000", sv.StrictTransportSecurity)
		assert.Equal(t, "same-origin-allow-popups", sv.CrossOriginOpenerPolicy)
		assert.Equal(t, "cross-origin", sv.CrossOriginResourcePolicy)
		assert.Equal(t, "camera=()", sv.PermissionsPolicy)
		assert.False(t, sv.Enabled)
		assert.False(t, sv.StripServerHeader)
		assert.False(t, sv.StripPoweredByHeader)
	})

	t.Run("all pointers nil uses secure defaults", func(t *testing.T) {
		headersConfig := &config.SecurityHeadersConfig{}

		sv := NewSecurityHeadersValues(headersConfig)

		assert.Equal(t, "DENY", sv.XFrameOptions)
		assert.True(t, sv.Enabled)
		assert.True(t, sv.StripServerHeader)
		assert.True(t, sv.StripPoweredByHeader)
	})
}

func TestNewCookieSecurityValues(t *testing.T) {
	t.Run("all pointers set", func(t *testing.T) {
		cookieConfig := &config.CookieSecurityConfig{
			DefaultSameSite:    new("Strict"),
			ForceHTTPOnly:      new(false),
			ForceSecureOnHTTPS: new(false),
		}

		cv := NewCookieSecurityValues(cookieConfig)

		assert.Equal(t, "Strict", cv.DefaultSameSite)
		assert.False(t, cv.ForceHTTPOnly)
		assert.False(t, cv.ForceSecureOnHTTPS)
	})

	t.Run("all pointers nil uses secure defaults", func(t *testing.T) {
		cookieConfig := &config.CookieSecurityConfig{}

		cv := NewCookieSecurityValues(cookieConfig)

		assert.Equal(t, "Lax", cv.DefaultSameSite)
		assert.True(t, cv.ForceHTTPOnly)
		assert.True(t, cv.ForceSecureOnHTTPS)
	})
}

func TestNewReportingValues(t *testing.T) {
	t.Run("enabled with endpoints", func(t *testing.T) {
		reportingConfig := &config.ReportingConfig{
			Enabled: new(true),
			Endpoints: []config.ReportingEndpoint{
				{Name: "csp", URL: "https://example.com/csp"},
			},
		}

		rv := NewReportingValues(reportingConfig)

		assert.Equal(t, `csp="https://example.com/csp"`, rv.HeaderValue)
	})

	t.Run("disabled returns empty", func(t *testing.T) {
		reportingConfig := &config.ReportingConfig{
			Enabled: new(false),
		}

		rv := NewReportingValues(reportingConfig)

		assert.Equal(t, "", rv.HeaderValue)
	})

	t.Run("nil enabled returns empty", func(t *testing.T) {
		reportingConfig := &config.ReportingConfig{}

		rv := NewReportingValues(reportingConfig)

		assert.Equal(t, "", rv.HeaderValue)
	})
}
