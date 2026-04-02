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
	"crypto/tls"
	"strconv"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/internal/tlscert"
)

// TLSOption configures TLS settings for the server. Each option mutates
// the TLS config applied during bootstrap.
type TLSOption func(*config.TLSConfig)

// HealthTLSOption configures TLS settings for the health probe server.
type HealthTLSOption func(*config.HealthTLSConfig)

// MonitoringTLSOption configures TLS settings for the monitoring gRPC server.
type MonitoringTLSOption func(*tlscert.TLSValues)

// WithTLSCertFile sets the path to the PEM-encoded TLS certificate file.
//
// Takes path (string) which specifies the certificate file path.
//
// Returns TLSOption which sets the certificate path.
func WithTLSCertFile(path string) TLSOption {
	return func(tlsConfig *config.TLSConfig) {
		tlsConfig.CertFile = &path
	}
}

// WithTLSKeyFile sets the path to the PEM-encoded TLS private key file.
//
// Takes path (string) which specifies the key file path.
//
// Returns TLSOption which sets the key path.
func WithTLSKeyFile(path string) TLSOption {
	return func(tlsConfig *config.TLSConfig) {
		tlsConfig.KeyFile = &path
	}
}

// WithTLSClientCA sets the path to a PEM-encoded CA bundle for mTLS client
// certificate verification.
//
// Takes path (string) which specifies the client CA file path.
//
// Returns TLSOption which sets the client CA path.
func WithTLSClientCA(path string) TLSOption {
	return func(tlsConfig *config.TLSConfig) {
		tlsConfig.ClientCAFile = &path
	}
}

// WithTLSClientAuth sets the client certificate verification mode. Valid
// values are "none", "request", "require", "verify", and
// "require_and_verify".
//
// Takes authType (string) which specifies the auth mode.
//
// Returns TLSOption which sets the client auth type.
func WithTLSClientAuth(authType string) TLSOption {
	return func(tlsConfig *config.TLSConfig) {
		tlsConfig.ClientAuthType = &authType
	}
}

// WithTLSMinVersion sets the minimum TLS version. Valid values are "1.2"
// and "1.3".
//
// Takes version (string) which specifies the minimum version.
//
// Returns TLSOption which sets the minimum TLS version.
func WithTLSMinVersion(version string) TLSOption {
	return func(tlsConfig *config.TLSConfig) {
		tlsConfig.MinVersion = &version
	}
}

// WithTLSHotReload enables or disables automatic certificate reload when
// certificate files change on disk.
//
// Takes enabled (bool) which controls hot-reload behaviour.
//
// Returns TLSOption which sets the hot-reload flag.
func WithTLSHotReload(enabled bool) TLSOption {
	return func(tlsConfig *config.TLSConfig) {
		tlsConfig.HotReload = &enabled
	}
}

// WithTLSRedirectHTTP starts a plain HTTP listener on the given port that
// 301-redirects all requests to the HTTPS server. This is useful for
// redirecting http://example.com to https://example.com.
//
// Takes port (int) which specifies the port to listen on (e.g. 80 or 8081).
//
// Returns TLSOption which sets the redirect port.
func WithTLSRedirectHTTP(port int) TLSOption {
	return func(tlsConfig *config.TLSConfig) {
		s := strconv.Itoa(port)
		tlsConfig.RedirectHTTPPort = &s
	}
}

// WithTLS enables TLS/HTTPS for the server. Sub-options configure certificate
// paths, mTLS, and hot-reload settings.
//
// Takes opts (...TLSOption) which provides optional TLS configuration:
//   - WithTLSCertFile("/path/to/cert.pem"): sets the certificate path.
//   - WithTLSKeyFile("/path/to/key.pem"): sets the private key path.
//   - WithTLSClientCA("/path/to/ca.pem"): enables mTLS with client CA.
//   - WithTLSClientAuth("require_and_verify"): sets client auth mode.
//   - WithTLSMinVersion("1.3"): sets minimum TLS version.
//   - WithTLSHotReload(true): enables certificate hot-reload.
//
// Returns Option which configures the container with TLS settings.
func WithTLS(opts ...TLSOption) Option {
	return func(c *Container) {
		overrides := c.ensureOverrides()
		overrides.Network.TLS.Enabled = new(true)
		for _, opt := range opts {
			opt(&overrides.Network.TLS)
		}
	}
}

// WithHealthTLSCertFile sets the certificate file for the health probe server.
//
// Takes path (string) which specifies the certificate file path.
//
// Returns HealthTLSOption which sets the certificate path.
func WithHealthTLSCertFile(path string) HealthTLSOption {
	return func(healthTLSConfig *config.HealthTLSConfig) {
		healthTLSConfig.CertFile = &path
	}
}

// WithHealthTLSKeyFile sets the key file for the health probe server.
//
// Takes path (string) which specifies the key file path.
//
// Returns HealthTLSOption which sets the key path.
func WithHealthTLSKeyFile(path string) HealthTLSOption {
	return func(healthTLSConfig *config.HealthTLSConfig) {
		healthTLSConfig.KeyFile = &path
	}
}

// WithHealthTLSMinVersion sets the minimum TLS version for the health probe
// server.
//
// Takes version (string) which specifies the minimum version ("1.2" or "1.3").
//
// Returns HealthTLSOption which sets the minimum TLS version.
func WithHealthTLSMinVersion(version string) HealthTLSOption {
	return func(healthTLSConfig *config.HealthTLSConfig) {
		healthTLSConfig.MinVersion = &version
	}
}

// WithHealthTLS enables TLS for the health probe server.
//
// Takes opts (...HealthTLSOption) which provides optional health TLS settings.
//
// Returns Option which configures the container with health server TLS.
func WithHealthTLS(opts ...HealthTLSOption) Option {
	return func(c *Container) {
		overrides := c.ensureOverrides()
		overrides.HealthProbe.TLS.Enabled = new(true)
		for _, opt := range opts {
			opt(&overrides.HealthProbe.TLS)
		}
	}
}

// WithMonitoringTLSCertFile sets the certificate file for the monitoring server.
//
// Takes path (string) which specifies the certificate file path.
//
// Returns MonitoringTLSOption which sets the certificate path.
func WithMonitoringTLSCertFile(path string) MonitoringTLSOption {
	return func(v *tlscert.TLSValues) {
		v.CertFile = path
	}
}

// WithMonitoringTLSKeyFile sets the key file for the monitoring server.
//
// Takes path (string) which specifies the key file path.
//
// Returns MonitoringTLSOption which sets the key path.
func WithMonitoringTLSKeyFile(path string) MonitoringTLSOption {
	return func(v *tlscert.TLSValues) {
		v.KeyFile = path
	}
}

// WithMonitoringTLSClientCA sets the client CA file for mTLS on the monitoring
// server.
//
// Takes path (string) which specifies the client CA file path.
//
// Returns MonitoringTLSOption which sets the client CA path.
func WithMonitoringTLSClientCA(path string) MonitoringTLSOption {
	return func(v *tlscert.TLSValues) {
		v.ClientCAFile = path
	}
}

// WithMonitoringTLSClientAuth sets the client certificate verification mode for
// the monitoring server. Valid values are "none", "request", "require",
// "verify", and "require_and_verify".
//
// Takes authType (string) which specifies the auth mode.
//
// Returns MonitoringTLSOption which sets the client auth type.
func WithMonitoringTLSClientAuth(authType string) MonitoringTLSOption {
	return func(v *tlscert.TLSValues) {
		v.ClientAuthType = parseTLSClientAuth(authType)
	}
}

// WithMonitoringTLSMinVersion sets the minimum TLS version for the monitoring
// server. Valid values are "1.2" and "1.3".
//
// Takes version (string) which specifies the minimum version.
//
// Returns MonitoringTLSOption which sets the minimum TLS version.
func WithMonitoringTLSMinVersion(version string) MonitoringTLSOption {
	return func(v *tlscert.TLSValues) {
		v.MinVersion = parseTLSMinVersion(version)
	}
}

// WithMonitoringTLSHotReload enables or disables automatic certificate reload
// for the monitoring server.
//
// Takes enabled (bool) which controls hot-reload behaviour.
//
// Returns MonitoringTLSOption which sets the hot-reload flag.
func WithMonitoringTLSHotReload(enabled bool) MonitoringTLSOption {
	return func(v *tlscert.TLSValues) {
		v.HotReload = enabled
	}
}

// WithMonitoringTLS enables TLS for the monitoring gRPC server. Sub-options
// configure certificate paths, mTLS, and hot-reload settings.
//
// Takes opts (...MonitoringTLSOption) which provides optional TLS settings.
//
// Returns MonitoringOption which configures TLS on the monitoring service.
func WithMonitoringTLS(opts ...MonitoringTLSOption) MonitoringOption {
	return func(c *monitoring_domain.ServiceConfig) {
		c.TLS = tlscert.TLSValues{
			Mode:       tlscert.TLSModeCertFile,
			MinVersion: tls.VersionTLS12,
		}
		for _, opt := range opts {
			opt(&c.TLS)
		}
	}
}
