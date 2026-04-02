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

package tlscert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"sync/atomic"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestNewCertificateLoader_LoadsValidCert(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	certPath, keyPath := generateTestCertPair(t, directory, 365*24*time.Hour)

	loader, err := NewCertificateLoader(context.Background(), certPath, keyPath, false)
	require.NoError(t, err)
	defer func() { _ = loader.Close() }()

	cert := loader.cert.Load()
	require.NotNil(t, cert)
	assert.NotEmpty(t, cert.Certificate)
}

func TestNewCertificateLoader_InvalidCertFile_ReturnsError(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	_, keyPath := generateTestCertPair(t, directory, 365*24*time.Hour)

	_, err := NewCertificateLoader(context.Background(), filepath.Join(directory, "nonexistent.pem"), keyPath, false)
	assert.Error(t, err)
}

func TestNewCertificateLoader_InvalidKeyFile_ReturnsError(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	certPath, _ := generateTestCertPair(t, directory, 365*24*time.Hour)

	_, err := NewCertificateLoader(context.Background(), certPath, filepath.Join(directory, "nonexistent.pem"), false)
	assert.Error(t, err)
}

func TestNewCertificateLoader_MismatchedPair_ReturnsError(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	certPath, _ := generateTestCertPair(t, directory, 365*24*time.Hour)

	dir2 := t.TempDir()
	_, keyPath2 := generateTestCertPair(t, dir2, 365*24*time.Hour)

	_, err := NewCertificateLoader(context.Background(), certPath, keyPath2, false)
	assert.Error(t, err)
}

func TestGetCertificate_ReturnsLoadedCert(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	certPath, keyPath := generateTestCertPair(t, directory, 365*24*time.Hour)

	loader, err := NewCertificateLoader(context.Background(), certPath, keyPath, false)
	require.NoError(t, err)
	defer func() { _ = loader.Close() }()

	cert, err := loader.GetCertificate(nil)
	require.NoError(t, err)
	require.NotNil(t, cert)
	assert.NotEmpty(t, cert.Certificate)
}

func TestGetCertificate_NilCert_ReturnsError(t *testing.T) {
	t.Parallel()

	loader := &CertificateLoader{}

	_, err := loader.GetCertificate(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no TLS certificate loaded")
}

func TestHotReload_UpdatesCert(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping hot-reload test in short mode")
	}
	t.Parallel()

	directory := t.TempDir()
	certPath, keyPath := generateTestCertPair(t, directory, 365*24*time.Hour)

	loader, err := NewCertificateLoader(context.Background(), certPath, keyPath, true)
	require.NoError(t, err)
	defer func() { _ = loader.Close() }()

	initialCert, err := loader.GetCertificate(nil)
	require.NoError(t, err)

	generateTestCertPair(t, directory, 730*24*time.Hour)

	time.Sleep(CertReloadDebounce + 500*time.Millisecond)

	newCert, err := loader.GetCertificate(nil)
	require.NoError(t, err)

	assert.NotEqual(t, initialCert.Certificate[0], newCert.Certificate[0])
}

func TestHotReload_InvalidNewCert_KeepsOld(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping hot-reload test in short mode")
	}
	t.Parallel()

	directory := t.TempDir()
	certPath, keyPath := generateTestCertPair(t, directory, 365*24*time.Hour)

	loader, err := NewCertificateLoader(context.Background(), certPath, keyPath, true)
	require.NoError(t, err)
	defer func() { _ = loader.Close() }()

	initialCert, err := loader.GetCertificate(nil)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(certPath, []byte("not a certificate"), 0600))

	time.Sleep(CertReloadDebounce + 500*time.Millisecond)

	currentCert, err := loader.GetCertificate(nil)
	require.NoError(t, err)
	assert.Equal(t, initialCert.Certificate[0], currentCert.Certificate[0])
}

func TestCertificateLoader_Close_NoWatcher(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	certPath, keyPath := generateTestCertPair(t, directory, 365*24*time.Hour)

	loader, err := NewCertificateLoader(context.Background(), certPath, keyPath, false)
	require.NoError(t, err)

	err = loader.Close()
	assert.NoError(t, err)
}

func TestCertificateLoader_Close_WithWatcher(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	certPath, keyPath := generateTestCertPair(t, directory, 365*24*time.Hour)

	loader, err := NewCertificateLoader(context.Background(), certPath, keyPath, true)
	require.NoError(t, err)

	err = loader.Close()
	assert.NoError(t, err)
}

func TestLoaderOption_OnReload_Called(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping hot-reload test in short mode")
	}
	t.Parallel()

	directory := t.TempDir()
	certPath, keyPath := generateTestCertPair(t, directory, 365*24*time.Hour)

	var called atomic.Bool
	loader, err := NewCertificateLoader(context.Background(), certPath, keyPath, true,
		WithOnReload(func() { called.Store(true) }),
	)
	require.NoError(t, err)
	defer func() { _ = loader.Close() }()

	generateTestCertPair(t, directory, 730*24*time.Hour)

	time.Sleep(CertReloadDebounce + 500*time.Millisecond)

	assert.True(t, called.Load(), "expected OnReload callback to be called")
}

func TestLoaderOption_OnError_Called(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping hot-reload test in short mode")
	}
	t.Parallel()

	directory := t.TempDir()
	certPath, keyPath := generateTestCertPair(t, directory, 365*24*time.Hour)

	var called atomic.Bool
	loader, err := NewCertificateLoader(context.Background(), certPath, keyPath, true,
		WithOnError(func(_ error) { called.Store(true) }),
	)
	require.NoError(t, err)
	defer func() { _ = loader.Close() }()

	require.NoError(t, os.WriteFile(certPath, []byte("not a certificate"), 0600))

	time.Sleep(CertReloadDebounce + 500*time.Millisecond)

	assert.True(t, called.Load(), "expected OnError callback to be called")
}
