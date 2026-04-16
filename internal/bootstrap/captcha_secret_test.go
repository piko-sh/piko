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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/safedisk"
)

func TestDoResolveCaptchaSecret_UsesConfigWhenProvided(t *testing.T) {
	configSecret := "my-explicit-captcha-secret"

	result := doResolveCaptchaSecret(configSecret)

	assert.Equal(t, []byte(configSecret), result)
}

func TestDoResolveCaptchaSecret_WithTestCaptchaSandboxCreator(t *testing.T) {
	t.Run("reads existing secret from injected mock sandbox", func(t *testing.T) {
		sandbox := safedisk.NewMockSandbox("/tmp", safedisk.ModeReadWrite)
		expectedSecret := []byte("test-secret-12345678901234567890")
		encoded := make([]byte, hex.EncodedLen(len(expectedSecret)))
		hex.Encode(encoded, expectedSecret)
		require.NoError(t, sandbox.WriteFile(captchaSecretFileName, encoded, 0600))

		oldCreator := testCaptchaSandboxCreator
		testCaptchaSandboxCreator = func() (safedisk.Sandbox, error) {
			return sandbox, nil
		}
		defer func() { testCaptchaSandboxCreator = oldCreator }()

		result := doResolveCaptchaSecret("")

		assert.Equal(t, expectedSecret, result)
	})

	t.Run("creates and persists new secret when file not found", func(t *testing.T) {
		sandbox := safedisk.NewMockSandbox("/tmp", safedisk.ModeReadWrite)

		oldCreator := testCaptchaSandboxCreator
		testCaptchaSandboxCreator = func() (safedisk.Sandbox, error) {
			return sandbox, nil
		}
		defer func() { testCaptchaSandboxCreator = oldCreator }()

		result := doResolveCaptchaSecret("")

		assert.Len(t, result, captchaSecretLength)

		data, err := sandbox.ReadFile(captchaSecretFileName)
		require.NoError(t, err)
		assert.NotEmpty(t, data)
	})

	t.Run("generates ephemeral secret when sandbox creation fails", func(t *testing.T) {
		oldCreator := testCaptchaSandboxCreator
		testCaptchaSandboxCreator = func() (safedisk.Sandbox, error) {
			return nil, errors.New("sandbox creation failed")
		}
		defer func() { testCaptchaSandboxCreator = oldCreator }()

		result := doResolveCaptchaSecret("")

		assert.Len(t, result, captchaSecretLength)
	})

	t.Run("generates ephemeral secret when read fails with invalid hex", func(t *testing.T) {
		sandbox := safedisk.NewMockSandbox("/tmp", safedisk.ModeReadWrite)

		require.NoError(t, sandbox.WriteFile(captchaSecretFileName, []byte("not-valid-hex!!!"), 0600))

		oldCreator := testCaptchaSandboxCreator
		testCaptchaSandboxCreator = func() (safedisk.Sandbox, error) {
			return sandbox, nil
		}
		defer func() { testCaptchaSandboxCreator = oldCreator }()

		result := doResolveCaptchaSecret("")

		assert.Len(t, result, captchaSecretLength)
	})
}
