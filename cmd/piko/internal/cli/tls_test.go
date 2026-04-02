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

package cli

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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/safedisk"
)

func writeTestCA(t *testing.T, directory string) {
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

	certPath := filepath.Join(directory, caCertFilename)
	certFile, err := os.Create(certPath)
	require.NoError(t, err)
	require.NoError(t, pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}))
	require.NoError(t, certFile.Close())
}

func writeTestClientCert(t *testing.T, directory string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test client"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	certPath := filepath.Join(directory, clientCertFilename)
	certFile, err := os.Create(certPath)
	require.NoError(t, err)
	require.NoError(t, pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}))
	require.NoError(t, certFile.Close())

	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	keyPath := filepath.Join(directory, clientKeyFilename)
	keyFile, err := os.Create(keyPath)
	require.NoError(t, err)
	require.NoError(t, pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))
	require.NoError(t, keyFile.Close())
}

func TestLoadTLSCredentials_CAOnly_Succeeds(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	writeTestCA(t, directory)

	factory, err := safedisk.NewCLIFactory(directory)
	require.NoError(t, err)

	creds, err := loadTLSCredentials(factory, directory)
	require.NoError(t, err)
	assert.NotNil(t, creds)
}

func TestLoadTLSCredentials_WithClientCert_LoadsMTLS(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	writeTestCA(t, directory)
	writeTestClientCert(t, directory)

	factory, err := safedisk.NewCLIFactory(directory)
	require.NoError(t, err)

	creds, err := loadTLSCredentials(factory, directory)
	require.NoError(t, err)
	assert.NotNil(t, creds)
}

func TestLoadTLSCredentials_MissingCA_ReturnsError(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()

	factory, err := safedisk.NewCLIFactory(directory)
	require.NoError(t, err)

	_, err = loadTLSCredentials(factory, directory)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reading CA certificate")
}

func TestLoadTLSCredentials_InvalidCA_ReturnsError(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, caCertFilename), []byte("not a cert"), 0600))

	factory, err := safedisk.NewCLIFactory(directory)
	require.NoError(t, err)

	_, err = loadTLSCredentials(factory, directory)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid certificates")
}

func TestLoadTLSCredentials_InvalidClientCert_ReturnsError(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	writeTestCA(t, directory)
	require.NoError(t, os.WriteFile(filepath.Join(directory, clientCertFilename), []byte("bad cert"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(directory, clientKeyFilename), []byte("bad key"), 0600))

	factory, err := safedisk.NewCLIFactory(directory)
	require.NoError(t, err)

	_, err = loadTLSCredentials(factory, directory)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "loading client certificate")
}

func TestFileExists_ExistingFile(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "test.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0600))

	assert.True(t, fileExists(path))
}

func TestFileExists_NonExistent(t *testing.T) {
	t.Parallel()

	assert.False(t, fileExists("/nonexistent/file.txt"))
}

func TestFileExists_Directory(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	assert.False(t, fileExists(directory))
}
