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
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/safedisk"
)

func TestDoResolveCSRFSecret_UsesConfigWhenProvided(t *testing.T) {
	configSecret := "my-explicit-secret"

	result := doResolveCSRFSecret(configSecret)

	assert.Equal(t, []byte(configSecret), result)
}

func TestReadCSRFSecretFromSandbox_ReadsValidFile(t *testing.T) {

	tempDir := t.TempDir()
	expectedSecret := []byte("secret-from-file-1234567890123456")

	encoded := make([]byte, hex.EncodedLen(len(expectedSecret)))
	hex.Encode(encoded, expectedSecret)
	err := os.WriteFile(filepath.Join(tempDir, csrfSecretFileName), encoded, 0o600)
	require.NoError(t, err)

	sandbox := createTestSandbox(t, tempDir)
	defer func() { _ = sandbox.Close() }()

	result, err := readCSRFSecretFromSandbox(sandbox)

	require.NoError(t, err)
	assert.Equal(t, expectedSecret, result)
}

func TestReadCSRFSecretFromSandbox_ReturnsErrorForMissingFile(t *testing.T) {
	tempDir := t.TempDir()
	sandbox := createTestSandbox(t, tempDir)
	defer func() { _ = sandbox.Close() }()

	_, err := readCSRFSecretFromSandbox(sandbox)

	assert.Error(t, err)
}

func TestReadCSRFSecretFromSandbox_ReturnsErrorForEmptyFile(t *testing.T) {
	tempDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tempDir, csrfSecretFileName), []byte{}, 0o600)
	require.NoError(t, err)

	sandbox := createTestSandbox(t, tempDir)
	defer func() { _ = sandbox.Close() }()

	_, err = readCSRFSecretFromSandbox(sandbox)

	assert.Error(t, err)
}

func TestReadCSRFSecretFromSandbox_ReturnsErrorForCorruptedFile(t *testing.T) {
	tempDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tempDir, csrfSecretFileName), []byte("not-valid-hex-data!!"), 0o600)
	require.NoError(t, err)

	sandbox := createTestSandbox(t, tempDir)
	defer func() { _ = sandbox.Close() }()

	_, err = readCSRFSecretFromSandbox(sandbox)

	assert.Error(t, err)
}

func TestWriteCSRFSecretToSandbox_WritesAndReads(t *testing.T) {
	tempDir := t.TempDir()
	sandbox := createTestSandbox(t, tempDir)
	defer func() { _ = sandbox.Close() }()

	secret := generateCSRFSecret()

	err := writeCSRFSecretToSandbox(sandbox, secret)
	require.NoError(t, err)

	secretPath := filepath.Join(tempDir, csrfSecretFileName)
	_, err = os.Stat(secretPath)
	require.NoError(t, err)

	readBack, err := readCSRFSecretFromSandbox(sandbox)
	require.NoError(t, err)
	assert.Equal(t, secret, readBack)
}

func TestWriteCSRFSecretToSandbox_CreatesFileWithCorrectPermissions(t *testing.T) {
	tempDir := t.TempDir()
	sandbox := createTestSandbox(t, tempDir)
	defer func() { _ = sandbox.Close() }()

	secret := generateCSRFSecret()

	err := writeCSRFSecretToSandbox(sandbox, secret)
	require.NoError(t, err)

	secretPath := filepath.Join(tempDir, csrfSecretFileName)
	info, err := os.Stat(secretPath)
	require.NoError(t, err)

	mode := info.Mode().Perm()
	assert.Equal(t, os.FileMode(0o600), mode, "File should have 0600 permissions")
}

func TestGenerateCSRFSecret_ReturnsCorrectLength(t *testing.T) {
	secret := generateCSRFSecret()

	assert.Len(t, secret, csrfSecretLength)
}

func TestGenerateFallbackSecret_ReturnsNonEmptySecret(t *testing.T) {
	secret := generateFallbackSecret()

	assert.Len(t, secret, csrfSecretLength)
}

func TestGenerateFallbackSecret_GeneratesUniqueValues(t *testing.T) {

	secret1 := generateFallbackSecret()

	time.Sleep(time.Nanosecond * 100)
	secret2 := generateFallbackSecret()

	assert.NotEqual(t, secret1, secret2, "Fallback secrets should differ across calls")
}

func TestGetCSRFSecretPath_ReturnsFileName(t *testing.T) {
	path := getCSRFSecretPath()

	assert.Equal(t, csrfSecretFileName, path)
}

func TestCreateTempSandbox_CreatesSandbox(t *testing.T) {
	sandbox, err := createTempSandbox()

	require.NoError(t, err)
	require.NotNil(t, sandbox)
	defer func() { _ = sandbox.Close() }()

	assert.Equal(t, safedisk.ModeReadWrite, sandbox.Mode())
}

func TestWithCSRFSecret_OverridesAutoResolution(t *testing.T) {

	customSecret := []byte("my-custom-override-secret")

	container := &Container{
		csrfSecretKeyProvider: func() []byte {
			return []byte("auto-resolved-secret")
		},
	}

	opt := WithCSRFSecret(customSecret)
	opt(container)

	result := container.csrfSecretKeyProvider()
	assert.Equal(t, customSecret, result, "WithCSRFSecret should override auto-resolved secret")
}

func createTestSandbox(t *testing.T, directory string) safedisk.Sandbox {
	t.Helper()

	factory, err := safedisk.NewFactory(safedisk.FactoryConfig{
		CWD:          directory,
		AllowedPaths: []string{directory},
		Enabled:      true,
	})
	require.NoError(t, err)

	sandbox, err := factory.Create("test-csrf", directory, safedisk.ModeReadWrite)
	require.NoError(t, err)

	return sandbox
}

func TestDoResolveCSRFSecret_WithTestSandboxCreator(t *testing.T) {
	t.Run("reads existing secret from injected mock sandbox", func(t *testing.T) {
		sandbox := safedisk.NewMockSandbox("/tmp", safedisk.ModeReadWrite)
		expectedSecret := []byte("test-secret-12345678901234567890")
		encoded := make([]byte, hex.EncodedLen(len(expectedSecret)))
		hex.Encode(encoded, expectedSecret)
		require.NoError(t, sandbox.WriteFile(csrfSecretFileName, encoded, 0600))

		oldCreator := testSandboxCreator
		testSandboxCreator = func() (safedisk.Sandbox, error) {
			return sandbox, nil
		}
		defer func() { testSandboxCreator = oldCreator }()

		result := doResolveCSRFSecret("")

		assert.Equal(t, expectedSecret, result)
	})

	t.Run("generates ephemeral secret when sandbox creation fails", func(t *testing.T) {

		oldCreator := testSandboxCreator
		testSandboxCreator = func() (safedisk.Sandbox, error) {
			return nil, errors.New("sandbox creation failed")
		}
		defer func() { testSandboxCreator = oldCreator }()

		result := doResolveCSRFSecret("")

		assert.Len(t, result, csrfSecretLength)
	})

	t.Run("creates and persists new secret when file not found", func(t *testing.T) {
		sandbox := safedisk.NewMockSandbox("/tmp", safedisk.ModeReadWrite)

		oldCreator := testSandboxCreator
		testSandboxCreator = func() (safedisk.Sandbox, error) {
			return sandbox, nil
		}
		defer func() { testSandboxCreator = oldCreator }()

		result := doResolveCSRFSecret("")

		assert.Len(t, result, csrfSecretLength)

		data, err := sandbox.ReadFile(csrfSecretFileName)
		require.NoError(t, err)
		assert.NotEmpty(t, data)
	})

	t.Run("generates ephemeral secret when write fails", func(t *testing.T) {
		sandbox := safedisk.NewMockSandbox("/tmp", safedisk.ModeReadWrite)
		sandbox.WriteFileErr = errors.New("disk write error")

		oldCreator := testSandboxCreator
		testSandboxCreator = func() (safedisk.Sandbox, error) {
			return sandbox, nil
		}
		defer func() { testSandboxCreator = oldCreator }()

		result := doResolveCSRFSecret("")

		assert.Len(t, result, csrfSecretLength)
	})

	t.Run("generates ephemeral secret when read fails with error", func(t *testing.T) {
		sandbox := safedisk.NewMockSandbox("/tmp", safedisk.ModeReadWrite)

		require.NoError(t, sandbox.WriteFile(csrfSecretFileName, []byte("not-valid-hex!!!"), 0600))

		oldCreator := testSandboxCreator
		testSandboxCreator = func() (safedisk.Sandbox, error) {
			return sandbox, nil
		}
		defer func() { testSandboxCreator = oldCreator }()

		result := doResolveCSRFSecret("")

		assert.Len(t, result, csrfSecretLength)
	})
}
