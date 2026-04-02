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

package monitoring_transport_grpc

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/tlscert"
	"piko.sh/piko/wdk/safedisk"
)

func testSandboxFactory(t *testing.T, allowedPaths ...string) safedisk.Factory {
	t.Helper()
	f, err := safedisk.NewFactory(safedisk.FactoryConfig{
		AllowedPaths: allowedPaths,
		Enabled:      true,
	})
	require.NoError(t, err)
	return f
}

func generateTestCertPair(t *testing.T, directory string, validity time.Duration) (certPath, keyPath string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(validity),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	certPath = filepath.Join(directory, "cert.pem")
	certFile, err := os.Create(certPath)
	require.NoError(t, err)
	require.NoError(t, pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}))
	require.NoError(t, certFile.Close())

	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	keyPath = filepath.Join(directory, "key.pem")
	keyFile, err := os.Create(keyPath)
	require.NoError(t, err)
	require.NoError(t, pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))
	require.NoError(t, keyFile.Close())

	return certPath, keyPath
}

func generateTestCA(t *testing.T, directory string) string {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(100),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	caPath := filepath.Join(directory, "ca.pem")
	caFile, err := os.Create(caPath)
	require.NoError(t, err)
	require.NoError(t, pem.Encode(caFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}))
	require.NoError(t, caFile.Close())

	return caPath
}

func TestBuildGRPCTLSCredentials_ValidCerts(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	certPath, keyPath := generateTestCertPair(t, directory, 365*24*time.Hour)

	tlsValues := tlscert.TLSValues{
		Mode:       tlscert.TLSModeCertFile,
		CertFile:   certPath,
		KeyFile:    keyPath,
		MinVersion: tls.VersionTLS12,
	}

	creds, cleanup, err := buildGRPCTLSCredentials(context.Background(), tlsValues, testSandboxFactory(t))
	require.NoError(t, err)
	require.NotNil(t, creds)
	require.NotNil(t, cleanup)

	assert.NoError(t, cleanup())
}

func TestBuildGRPCTLSCredentials_WithClientCA(t *testing.T) {
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

	creds, cleanup, err := buildGRPCTLSCredentials(context.Background(), tlsValues, testSandboxFactory(t, directory))
	require.NoError(t, err)
	require.NotNil(t, creds)

	assert.NoError(t, cleanup())
}

func TestBuildGRPCTLSCredentials_InvalidCert_ReturnsError(t *testing.T) {
	t.Parallel()

	tlsValues := tlscert.TLSValues{
		Mode:     tlscert.TLSModeCertFile,
		CertFile: "/nonexistent/cert.pem",
		KeyFile:  "/nonexistent/key.pem",
	}

	_, _, err := buildGRPCTLSCredentials(context.Background(), tlsValues, testSandboxFactory(t))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "loading TLS certificates")
}

func TestBuildGRPCTLSCredentials_InvalidClientCA_ReturnsError(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	certPath, keyPath := generateTestCertPair(t, directory, 365*24*time.Hour)

	tlsValues := tlscert.TLSValues{
		Mode:         tlscert.TLSModeCertFile,
		CertFile:     certPath,
		KeyFile:      keyPath,
		ClientCAFile: "/nonexistent/ca.pem",
	}

	_, _, err := buildGRPCTLSCredentials(context.Background(), tlsValues, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "loading client CA")
}
