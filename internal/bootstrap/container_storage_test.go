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
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/config"
	storage_mock "piko.sh/piko/internal/storage/storage_adapters/provider_mock"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

func TestSelectStorageBaseProvider(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		wantProvider     storage_domain.StorageProviderPort
		providers        map[string]storage_domain.StorageProviderPort
		name             string
		defaultProvider  string
		wantName         string
		wantErrSubstring string
		wantErr          bool
	}{
		{
			name:            "explicit default that exists",
			providers:       map[string]storage_domain.StorageProviderPort{"s3": storage_mock.NewMockStorageProvider()},
			defaultProvider: "s3",
			wantName:        "s3",
		},
		{
			name:             "explicit default that does not exist",
			providers:        map[string]storage_domain.StorageProviderPort{"s3": storage_mock.NewMockStorageProvider()},
			defaultProvider:  "gcs",
			wantErr:          true,
			wantErrSubstring: "not registered",
		},
		{
			name:      "no explicit default but well-known default key exists",
			providers: map[string]storage_domain.StorageProviderPort{storage_dto.StorageProviderDefault: storage_mock.NewMockStorageProvider()},
			wantName:  storage_dto.StorageProviderDefault,
		},
		{
			name:      "no explicit default and no well-known key picks first",
			providers: map[string]storage_domain.StorageProviderPort{"custom": storage_mock.NewMockStorageProvider()},
			wantName:  "custom",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := NewContainer(config.NewConfigProvider())

			if len(tc.providers) > 0 {
				c.storageProviders = make(map[string]storage_domain.StorageProviderPort, len(tc.providers))
				maps.Copy(c.storageProviders, tc.providers)
			}
			c.storageDefaultProvider = tc.defaultProvider

			if tc.wantProvider == nil && !tc.wantErr && len(tc.providers) > 0 {
				tc.wantProvider = tc.providers[tc.wantName]
			}

			gotName, gotProvider, err := c.selectStorageBaseProvider()

			if tc.wantErr {
				require.Error(t, err)
				if tc.wantErrSubstring != "" {
					assert.Contains(t, err.Error(), tc.wantErrSubstring)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantName, gotName)
			if tc.wantProvider != nil {
				assert.Same(t, tc.wantProvider, gotProvider)
			}
		})
	}
}

func TestCreateStorageDispatcher(t *testing.T) {
	t.Parallel()

	baseProvider := storage_mock.NewMockStorageProvider()

	testCases := []struct {
		dispatcherConfig *storage_domain.DispatcherConfig
		name             string
		hasDispatcher    bool
		wantNil          bool
	}{
		{
			name:          "dispatcher not enabled",
			hasDispatcher: false,
			wantNil:       true,
		},
		{
			name:          "dispatcher enabled with default config",
			hasDispatcher: true,
			wantNil:       false,
		},
		{
			name:          "dispatcher enabled with custom config",
			hasDispatcher: true,
			dispatcherConfig: func() *storage_domain.DispatcherConfig {
				return new(storage_domain.DefaultDispatcherConfig())
			}(),
			wantNil: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := NewContainer(config.NewConfigProvider())
			c.hasStorageDispatcher = tc.hasDispatcher
			c.storageDispatcherConfig = tc.dispatcherConfig

			result, err := c.createStorageDispatcher(context.Background(), baseProvider, "test")

			require.NoError(t, err)

			if tc.wantNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
			}
		})
	}
}
