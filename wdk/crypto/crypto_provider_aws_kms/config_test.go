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

package crypto_provider_aws_kms

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/crypto"
)

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			"valid config",
			Config{KeyID: "alias/my-key", Region: "us-east-1"},
			"",
		},
		{
			"valid with MaxRetries=0",
			Config{KeyID: "alias/my-key", Region: "us-east-1", MaxRetries: 0},
			"",
		},
		{
			"missing KeyID",
			Config{Region: "us-east-1"},
			"KeyID is required",
		},
		{
			"missing Region",
			Config{KeyID: "alias/my-key"},
			"Region is required",
		},
		{
			"negative MaxRetries",
			Config{KeyID: "alias/my-key", Region: "us-east-1", MaxRetries: -1},
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

		config := Config{KeyID: "alias/my-key", Region: "us-east-1"}
		result := config.WithDefaults()
		assert.Equal(t, 3, result.MaxRetries)
	})

	t.Run("non-zero MaxRetries preserved", func(t *testing.T) {
		t.Parallel()

		config := Config{KeyID: "alias/my-key", Region: "us-east-1", MaxRetries: 5}
		result := config.WithDefaults()
		assert.Equal(t, 5, result.MaxRetries)
	})
}

func TestMapKeyState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		state types.KeyState
		want  crypto.KeyStatus
	}{
		{name: "enabled", state: types.KeyStateEnabled, want: crypto.KeyStatusActive},
		{name: "pending deletion", state: types.KeyStatePendingDeletion, want: crypto.KeyStatusDestroyed},
		{name: "disabled", state: types.KeyStateDisabled, want: crypto.KeyStatusDisabled},
		{name: "unknown state", state: types.KeyState("SomeUnknownState"), want: crypto.KeyStatusDisabled},
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
