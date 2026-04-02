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

package crypto_adapters

import (
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateProviderFromBase64Key(t *testing.T) {
	t.Parallel()

	t.Run("valid 32-byte key returns provider without error", func(t *testing.T) {
		t.Parallel()

		key := make([]byte, 32)
		_, err := rand.Read(key)
		require.NoError(t, err)

		encoded := base64.StdEncoding.EncodeToString(key)

		provider, err := CreateProviderFromBase64Key(encoded, "test-key-id")
		require.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("invalid base64 string returns decode error", func(t *testing.T) {
		t.Parallel()

		provider, err := CreateProviderFromBase64Key("not-valid-base64!!!", "test-key")
		require.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "failed to decode base64 key")
	})

	t.Run("16-byte key returns error about 32 bytes", func(t *testing.T) {
		t.Parallel()

		key := make([]byte, 16)
		_, err := rand.Read(key)
		require.NoError(t, err)

		encoded := base64.StdEncoding.EncodeToString(key)

		provider, err := CreateProviderFromBase64Key(encoded, "short-key")
		require.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "32 bytes")
		assert.Contains(t, err.Error(), "16 bytes")
	})

	t.Run("48-byte key returns error about 32 bytes", func(t *testing.T) {
		t.Parallel()

		key := make([]byte, 48)
		_, err := rand.Read(key)
		require.NoError(t, err)

		encoded := base64.StdEncoding.EncodeToString(key)

		provider, err := CreateProviderFromBase64Key(encoded, "long-key")
		require.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "32 bytes")
		assert.Contains(t, err.Error(), "48 bytes")
	})

	t.Run("empty key ID defaults to local-default", func(t *testing.T) {
		t.Parallel()

		key := make([]byte, 32)
		_, err := rand.Read(key)
		require.NoError(t, err)

		encoded := base64.StdEncoding.EncodeToString(key)

		provider, err := CreateProviderFromBase64Key(encoded, "")
		require.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("explicit key ID is accepted", func(t *testing.T) {
		t.Parallel()

		key := make([]byte, 32)
		_, err := rand.Read(key)
		require.NoError(t, err)

		encoded := base64.StdEncoding.EncodeToString(key)

		provider, err := CreateProviderFromBase64Key(encoded, "my-custom-key")
		require.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("empty base64 string returns error about 32 bytes", func(t *testing.T) {
		t.Parallel()

		encoded := base64.StdEncoding.EncodeToString([]byte{})

		provider, err := CreateProviderFromBase64Key(encoded, "empty-key")
		require.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "32 bytes")
	})
}
