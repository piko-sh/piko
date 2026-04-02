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
	"context"
	"encoding/base64"
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/crypto/crypto_domain"
)

type stubEncryptionProvider struct {
	crypto_domain.EncryptionProvider
}

func TestSelectCryptoProvider(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		wantProvider     crypto_domain.EncryptionProvider
		providers        map[string]crypto_domain.EncryptionProvider
		name             string
		defaultProvider  string
		wantName         string
		wantActiveKeyID  string
		wantErrSubstring string
		securityConfig   config.SecurityConfig
		wantErr          bool
	}{
		{
			name:            "custom provider with explicit default",
			providers:       map[string]crypto_domain.EncryptionProvider{"aws": &stubEncryptionProvider{}},
			defaultProvider: "aws",
			wantName:        "aws",
			wantActiveKeyID: "default",
		},
		{
			name:             "custom provider with missing default",
			providers:        map[string]crypto_domain.EncryptionProvider{"aws": &stubEncryptionProvider{}},
			defaultProvider:  "missing",
			wantErr:          true,
			wantErrSubstring: "not registered",
		},
		{
			name:            "custom providers with no default picks first",
			providers:       map[string]crypto_domain.EncryptionProvider{"local": &stubEncryptionProvider{}},
			wantName:        "local",
			wantActiveKeyID: "default",
		},
		{
			name:            "no custom providers with empty key returns disabled",
			securityConfig:  config.SecurityConfig{EncryptionKey: new("")},
			wantName:        "disabled",
			wantActiveKeyID: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := NewContainer(config.NewConfigProvider())

			if len(tc.providers) > 0 {
				c.cryptoProviders = make(map[string]crypto_domain.EncryptionProvider, len(tc.providers))
				maps.Copy(c.cryptoProviders, tc.providers)
			}
			c.cryptoDefaultProvider = tc.defaultProvider

			if tc.wantProvider == nil && !tc.wantErr && len(tc.providers) > 0 {
				tc.wantProvider = tc.providers[tc.wantName]
			}

			gotName, gotProvider, gotKeyID, err := c.selectCryptoProvider(context.Background(), &tc.securityConfig)

			if tc.wantErr {
				require.Error(t, err)
				if tc.wantErrSubstring != "" {
					assert.Contains(t, err.Error(), tc.wantErrSubstring)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantName, gotName)
			assert.Equal(t, tc.wantActiveKeyID, gotKeyID)
			if tc.wantProvider != nil {
				assert.Same(t, tc.wantProvider, gotProvider)
			}
		})
	}
}

func TestCreateProviderFromConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		wantName         string
		wantErrSubstring string
		securityConfig   config.SecurityConfig
		wantErr          bool
	}{
		{
			name:           "nil provider type defaults to local_aes_gcm and returns disabled with no key",
			securityConfig: config.SecurityConfig{},
			wantName:       "disabled",
		},
		{
			name:           "explicit local_aes_gcm with no key returns disabled",
			securityConfig: config.SecurityConfig{CryptoProvider: new("local_aes_gcm")},
			wantName:       "disabled",
		},
		{
			name:             "aws_kms returns cloud config error",
			securityConfig:   config.SecurityConfig{CryptoProvider: new("aws_kms")},
			wantErr:          true,
			wantErrSubstring: "cannot be configured via config file",
		},
		{
			name:             "gcp_kms returns cloud config error",
			securityConfig:   config.SecurityConfig{CryptoProvider: new("gcp_kms")},
			wantErr:          true,
			wantErrSubstring: "cannot be configured via config file",
		},
		{
			name:             "unknown provider returns error",
			securityConfig:   config.SecurityConfig{CryptoProvider: new("bogus")},
			wantErr:          true,
			wantErrSubstring: "unknown crypto provider",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := NewContainer(config.NewConfigProvider())

			gotName, _, _, err := c.createProviderFromConfig(context.Background(), &tc.securityConfig)

			if tc.wantErr {
				require.Error(t, err)
				if tc.wantErrSubstring != "" {
					assert.Contains(t, err.Error(), tc.wantErrSubstring)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantName, gotName)
		})
	}
}

func TestCreateLocalAESGCMProvider(t *testing.T) {
	t.Parallel()

	validKey := base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901"))

	testCases := []struct {
		name             string
		encryptionKey    string
		wantName         string
		wantActiveKeyID  string
		wantErrSubstring string
		wantNilProvider  bool
		wantErr          bool
	}{
		{
			name:            "empty key returns disabled",
			encryptionKey:   "",
			wantName:        "disabled",
			wantNilProvider: true,
		},
		{
			name:            "valid base64 key creates provider",
			encryptionKey:   validKey,
			wantName:        "local_aes_gcm",
			wantActiveKeyID: "piko-default-key",
		},
		{
			name:             "invalid key returns error",
			encryptionKey:    "not-valid-base64-!!!",
			wantErr:          true,
			wantErrSubstring: "failed to create local encryption provider",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := &Container{}
			securityConfig := &config.SecurityConfig{EncryptionKey: new(tc.encryptionKey)}

			gotName, gotProvider, gotKeyID, err := c.createLocalAESGCMProvider(securityConfig)

			if tc.wantErr {
				require.Error(t, err)
				if tc.wantErrSubstring != "" {
					assert.Contains(t, err.Error(), tc.wantErrSubstring)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantName, gotName)

			if tc.wantNilProvider {
				assert.Nil(t, gotProvider)
			} else {
				assert.NotNil(t, gotProvider)
				assert.Equal(t, tc.wantActiveKeyID, gotKeyID)
			}
		})
	}
}
