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

package daemon_adapters

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/tlscert"
)

func TestNewServerAdapterFromTLSConfig_Disabled_ReturnsPlainAdapter(t *testing.T) {
	t.Parallel()

	tlsValues := tlscert.TLSValues{Mode: tlscert.TLSModeOff}

	adapter, cleanup, err := NewServerAdapterFromTLSConfig(context.Background(), tlsValues, serverPurposeMain, nil)
	require.NoError(t, err)
	require.NotNil(t, adapter)
	require.NotNil(t, cleanup)

	a, ok := adapter.(*driverHTTPServerAdapter)
	require.True(t, ok)
	assert.Nil(t, a.tlsConfig)
	assert.Equal(t, serverPurposeMain, a.purpose)

	assert.NoError(t, cleanup())
}

func TestNewServerAdapterFromTLSConfig_CertFile_ReturnsTLSAdapter(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	certPath, keyPath := generateTestCertPair(t, directory, 365*24*time.Hour)

	tlsValues := tlscert.TLSValues{
		Mode:       tlscert.TLSModeCertFile,
		CertFile:   certPath,
		KeyFile:    keyPath,
		MinVersion: tls.VersionTLS12,
	}

	adapter, cleanup, err := NewServerAdapterFromTLSConfig(context.Background(), tlsValues, serverPurposeMain, nil)
	require.NoError(t, err)
	require.NotNil(t, adapter)

	a, ok := adapter.(*driverHTTPServerAdapter)
	require.True(t, ok)
	assert.NotNil(t, a.tlsConfig)
	assert.Equal(t, uint16(tls.VersionTLS12), a.tlsConfig.MinVersion)
	assert.Contains(t, a.tlsConfig.NextProtos, "h2")
	assert.Contains(t, a.tlsConfig.NextProtos, "http/1.1")

	assert.NoError(t, cleanup())
}

func TestNewServerAdapterFromTLSConfig_CertFile_WithClientCA(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	certPath, keyPath := generateTestCertPair(t, directory, 365*24*time.Hour)
	caPath := generateTestCA(t, directory)

	tlsValues := tlscert.TLSValues{
		Mode:           tlscert.TLSModeCertFile,
		CertFile:       certPath,
		KeyFile:        keyPath,
		ClientCAFile:   caPath,
		ClientAuthType: tls.RequireAndVerifyClientCert,
		MinVersion:     tls.VersionTLS13,
	}

	adapter, cleanup, err := NewServerAdapterFromTLSConfig(context.Background(), tlsValues, serverPurposeMain, nil)
	require.NoError(t, err)
	require.NotNil(t, adapter)

	a, ok := adapter.(*driverHTTPServerAdapter)
	require.True(t, ok)
	assert.NotNil(t, a.tlsConfig.ClientCAs)
	assert.Equal(t, tls.RequireAndVerifyClientCert, a.tlsConfig.ClientAuth)

	assert.NoError(t, cleanup())
}

func TestNewServerAdapterFromTLSConfig_InvalidCert_ReturnsError(t *testing.T) {
	t.Parallel()

	tlsValues := tlscert.TLSValues{
		Mode:     tlscert.TLSModeCertFile,
		CertFile: "/nonexistent/cert.pem",
		KeyFile:  "/nonexistent/key.pem",
	}

	_, _, err := NewServerAdapterFromTLSConfig(context.Background(), tlsValues, serverPurposeMain, nil)
	assert.Error(t, err)
}

func TestNewServerAdapterFromTLSConfig_InvalidClientCA_ReturnsError(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	certPath, keyPath := generateTestCertPair(t, directory, 365*24*time.Hour)

	tlsValues := tlscert.TLSValues{
		Mode:         tlscert.TLSModeCertFile,
		CertFile:     certPath,
		KeyFile:      keyPath,
		ClientCAFile: "/nonexistent/ca.pem",
	}

	_, _, err := NewServerAdapterFromTLSConfig(context.Background(), tlsValues, serverPurposeMain, nil)
	assert.Error(t, err)
}

func TestNewServerAdapterFromTLSConfig_HealthPurpose(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	certPath, keyPath := generateTestCertPair(t, directory, 365*24*time.Hour)

	tlsValues := tlscert.TLSValues{
		Mode:     tlscert.TLSModeCertFile,
		CertFile: certPath,
		KeyFile:  keyPath,
	}

	adapter, cleanup, err := NewServerAdapterFromTLSConfig(context.Background(), tlsValues, serverPurposeHealth, nil)
	require.NoError(t, err)

	a, ok := adapter.(*driverHTTPServerAdapter)
	require.True(t, ok)
	assert.Equal(t, serverPurposeHealth, a.purpose)

	assert.NoError(t, cleanup())
}
