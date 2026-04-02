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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/internal/tlscert"
)

func TestWithTLSCertFile_SetsPath(t *testing.T) {
	t.Parallel()

	tlsConfig := &config.TLSConfig{}
	opt := WithTLSCertFile("/certs/server.pem")
	opt(tlsConfig)

	require.NotNil(t, tlsConfig.CertFile)
	assert.Equal(t, "/certs/server.pem", *tlsConfig.CertFile)
}

func TestWithTLSKeyFile_SetsPath(t *testing.T) {
	t.Parallel()

	tlsConfig := &config.TLSConfig{}
	opt := WithTLSKeyFile("/certs/server-key.pem")
	opt(tlsConfig)

	require.NotNil(t, tlsConfig.KeyFile)
	assert.Equal(t, "/certs/server-key.pem", *tlsConfig.KeyFile)
}

func TestWithTLSClientCA_SetsPath(t *testing.T) {
	t.Parallel()

	tlsConfig := &config.TLSConfig{}
	opt := WithTLSClientCA("/certs/ca.pem")
	opt(tlsConfig)

	require.NotNil(t, tlsConfig.ClientCAFile)
	assert.Equal(t, "/certs/ca.pem", *tlsConfig.ClientCAFile)
}

func TestWithTLSClientAuth_SetsMode(t *testing.T) {
	t.Parallel()

	tlsConfig := &config.TLSConfig{}
	opt := WithTLSClientAuth("require_and_verify")
	opt(tlsConfig)

	require.NotNil(t, tlsConfig.ClientAuthType)
	assert.Equal(t, "require_and_verify", *tlsConfig.ClientAuthType)
}

func TestWithTLSMinVersion_SetsVersion(t *testing.T) {
	t.Parallel()

	tlsConfig := &config.TLSConfig{}
	opt := WithTLSMinVersion("1.3")
	opt(tlsConfig)

	require.NotNil(t, tlsConfig.MinVersion)
	assert.Equal(t, "1.3", *tlsConfig.MinVersion)
}

func TestWithTLSHotReload_SetsFlag(t *testing.T) {
	t.Parallel()

	tlsConfig := &config.TLSConfig{}
	opt := WithTLSHotReload(true)
	opt(tlsConfig)

	require.NotNil(t, tlsConfig.HotReload)
	assert.True(t, *tlsConfig.HotReload)
}

func TestWithHealthTLSCertFile_SetsPath(t *testing.T) {
	t.Parallel()

	healthTLSConfig := &config.HealthTLSConfig{}
	opt := WithHealthTLSCertFile("/certs/health.pem")
	opt(healthTLSConfig)

	require.NotNil(t, healthTLSConfig.CertFile)
	assert.Equal(t, "/certs/health.pem", *healthTLSConfig.CertFile)
}

func TestWithHealthTLSKeyFile_SetsPath(t *testing.T) {
	t.Parallel()

	healthTLSConfig := &config.HealthTLSConfig{}
	opt := WithHealthTLSKeyFile("/certs/health-key.pem")
	opt(healthTLSConfig)

	require.NotNil(t, healthTLSConfig.KeyFile)
	assert.Equal(t, "/certs/health-key.pem", *healthTLSConfig.KeyFile)
}

func TestWithHealthTLSMinVersion_SetsVersion(t *testing.T) {
	t.Parallel()

	healthTLSConfig := &config.HealthTLSConfig{}
	opt := WithHealthTLSMinVersion("1.3")
	opt(healthTLSConfig)

	require.NotNil(t, healthTLSConfig.MinVersion)
	assert.Equal(t, "1.3", *healthTLSConfig.MinVersion)
}

func TestWithMonitoringTLSCertFile_SetsPath(t *testing.T) {
	t.Parallel()

	v := &tlscert.TLSValues{}
	opt := WithMonitoringTLSCertFile("/certs/monitoring.pem")
	opt(v)

	assert.Equal(t, "/certs/monitoring.pem", v.CertFile)
}

func TestWithMonitoringTLSKeyFile_SetsPath(t *testing.T) {
	t.Parallel()

	v := &tlscert.TLSValues{}
	opt := WithMonitoringTLSKeyFile("/certs/monitoring-key.pem")
	opt(v)

	assert.Equal(t, "/certs/monitoring-key.pem", v.KeyFile)
}

func TestWithMonitoringTLSClientCA_SetsPath(t *testing.T) {
	t.Parallel()

	v := &tlscert.TLSValues{}
	opt := WithMonitoringTLSClientCA("/certs/ca.pem")
	opt(v)

	assert.Equal(t, "/certs/ca.pem", v.ClientCAFile)
}

func TestWithMonitoringTLSClientAuth_SetsMode(t *testing.T) {
	t.Parallel()

	v := &tlscert.TLSValues{}
	opt := WithMonitoringTLSClientAuth("require_and_verify")
	opt(v)

	assert.Equal(t, tls.RequireAndVerifyClientCert, v.ClientAuthType)
}

func TestWithMonitoringTLSMinVersion_SetsVersion(t *testing.T) {
	t.Parallel()

	v := &tlscert.TLSValues{}
	opt := WithMonitoringTLSMinVersion("1.3")
	opt(v)

	assert.Equal(t, uint16(tls.VersionTLS13), v.MinVersion)
}

func TestWithMonitoringTLSHotReload_SetsFlag(t *testing.T) {
	t.Parallel()

	v := &tlscert.TLSValues{}
	opt := WithMonitoringTLSHotReload(true)
	opt(v)

	assert.True(t, v.HotReload)
}

func TestWithMonitoringTLS_EnablesTLS(t *testing.T) {
	t.Parallel()

	monitoringConfig := &monitoring_domain.ServiceConfig{}
	opt := WithMonitoringTLS(
		WithMonitoringTLSCertFile("/certs/server.pem"),
		WithMonitoringTLSKeyFile("/certs/server-key.pem"),
	)
	opt(monitoringConfig)

	assert.True(t, monitoringConfig.TLS.Enabled())
	assert.Equal(t, tlscert.TLSModeCertFile, monitoringConfig.TLS.Mode)
	assert.Equal(t, "/certs/server.pem", monitoringConfig.TLS.CertFile)
	assert.Equal(t, "/certs/server-key.pem", monitoringConfig.TLS.KeyFile)
	assert.Equal(t, uint16(tls.VersionTLS12), monitoringConfig.TLS.MinVersion)
}

func TestWithMonitoringTLS_WithMTLS(t *testing.T) {
	t.Parallel()

	monitoringConfig := &monitoring_domain.ServiceConfig{}
	opt := WithMonitoringTLS(
		WithMonitoringTLSCertFile("/certs/server.pem"),
		WithMonitoringTLSKeyFile("/certs/server-key.pem"),
		WithMonitoringTLSClientCA("/certs/ca.pem"),
		WithMonitoringTLSClientAuth("require_and_verify"),
		WithMonitoringTLSMinVersion("1.3"),
		WithMonitoringTLSHotReload(true),
	)
	opt(monitoringConfig)

	assert.True(t, monitoringConfig.TLS.Enabled())
	assert.Equal(t, "/certs/ca.pem", monitoringConfig.TLS.ClientCAFile)
	assert.Equal(t, tls.RequireAndVerifyClientCert, monitoringConfig.TLS.ClientAuthType)
	assert.Equal(t, uint16(tls.VersionTLS13), monitoringConfig.TLS.MinVersion)
	assert.True(t, monitoringConfig.TLS.HotReload)
}
