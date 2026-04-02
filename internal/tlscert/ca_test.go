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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/safedisk"
)

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

func TestLoadClientCAs_ValidFile(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	generateTestCA(t, directory)

	sandbox, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadOnly)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	pool, err := LoadClientCAs(sandbox, "ca.pem")
	require.NoError(t, err)
	require.NotNil(t, pool)
}

func TestLoadClientCAs_InvalidFile_ReturnsError(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadOnly)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	_, err = LoadClientCAs(sandbox, "nonexistent.pem")
	assert.Error(t, err)
}

func TestLoadClientCAs_NoCertsInFile_ReturnsError(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "empty.pem"), []byte("not a certificate"), 0600))

	sandbox, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadOnly)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	_, err = LoadClientCAs(sandbox, "empty.pem")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid certificates")
}
