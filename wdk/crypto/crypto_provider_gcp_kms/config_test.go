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

package crypto_provider_gcp_kms

import (
	"testing"

	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/crypto"
)

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	validConfig := Config{
		ProjectID: "my-project",
		Location:  "global",
		KeyRing:   "my-ring",
		KeyName:   "my-key",
	}

	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{name: "valid config", config: validConfig, wantErr: ""},
		{
			"missing ProjectID",
			Config{Location: "global", KeyRing: "my-ring", KeyName: "my-key"},
			"ProjectID is required",
		},
		{
			"missing Location",
			Config{ProjectID: "my-project", KeyRing: "my-ring", KeyName: "my-key"},
			"Location is required",
		},
		{
			"missing KeyRing",
			Config{ProjectID: "my-project", Location: "global", KeyName: "my-key"},
			"KeyRing is required",
		},
		{
			"missing KeyName",
			Config{ProjectID: "my-project", Location: "global", KeyRing: "my-ring"},
			"KeyName is required",
		},
		{
			"negative MaxRetries",
			Config{ProjectID: "my-project", Location: "global", KeyRing: "my-ring", KeyName: "my-key", MaxRetries: -1},
			"maxRetries cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestConfig_WithDefaults(t *testing.T) {
	t.Parallel()

	t.Run("zero MaxRetries gets default", func(t *testing.T) {
		t.Parallel()

		config := Config{ProjectID: "p", Location: "l", KeyRing: "r", KeyName: "k"}
		result := config.WithDefaults()
		assert.Equal(t, 3, result.MaxRetries)
	})

	t.Run("non-zero MaxRetries preserved", func(t *testing.T) {
		t.Parallel()

		config := Config{ProjectID: "p", Location: "l", KeyRing: "r", KeyName: "k", MaxRetries: 7}
		result := config.WithDefaults()
		assert.Equal(t, 7, result.MaxRetries)
	})
}

func TestConfig_KeyResourceName(t *testing.T) {
	t.Parallel()

	config := Config{
		ProjectID: "my-project",
		Location:  "us-central1",
		KeyRing:   "my-ring",
		KeyName:   "my-key",
	}

	got := config.KeyResourceName()
	assert.Equal(t, "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key", got)
}

func TestMapKeyState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		state kmspb.CryptoKeyVersion_CryptoKeyVersionState
		want  crypto.KeyStatus
	}{
		{name: "enabled", state: kmspb.CryptoKeyVersion_ENABLED, want: crypto.KeyStatusActive},
		{name: "destroyed", state: kmspb.CryptoKeyVersion_DESTROYED, want: crypto.KeyStatusDestroyed},
		{name: "disabled", state: kmspb.CryptoKeyVersion_DISABLED, want: crypto.KeyStatusDisabled},
		{name: "unknown", state: kmspb.CryptoKeyVersion_CRYPTO_KEY_VERSION_STATE_UNSPECIFIED, want: crypto.KeyStatusDisabled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := mapKeyState(tt.state)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestZeroBytes(t *testing.T) {
	t.Parallel()

	t.Run("zeroes non-empty slice", func(t *testing.T) {
		t.Parallel()

		data := []byte{1, 2, 3, 4, 5}
		zeroBytes(data)
		assert.Equal(t, []byte{0, 0, 0, 0, 0}, data)
	})

	t.Run("empty slice no-op", func(t *testing.T) {
		t.Parallel()

		data := []byte{}
		zeroBytes(data)
		assert.Empty(t, data)
	})

	t.Run("nil slice no-op", func(t *testing.T) {
		t.Parallel()

		zeroBytes(nil)
	})
}
