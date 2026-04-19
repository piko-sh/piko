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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	email_mock "piko.sh/piko/internal/email/email_adapters/provider_mock"
	storage_mock "piko.sh/piko/internal/storage/storage_adapters/provider_mock"
)

func TestValidateProviderConfiguration_NoProvidersNoDefaults(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	err := c.ValidateProviderConfiguration()
	assert.NoError(t, err, "expected no error with empty configuration")
}

func TestValidateProviderConfiguration_EmailDefaultWithoutProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer()
	c.SetEmailDefaultProvider("missing-provider")

	err := c.ValidateProviderConfiguration()
	require.Error(t, err, "expected error when default provider is not registered")
	assert.Contains(t, err.Error(), "email default provider", "error should mention email provider")
	assert.Contains(t, err.Error(), "missing-provider", "error should mention the missing provider name")
}

func TestValidateProviderConfiguration_EmailDefaultWithProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer()
	c.AddEmailProvider("my-email", email_mock.NewMockEmailProvider())
	c.SetEmailDefaultProvider("my-email")

	err := c.ValidateProviderConfiguration()
	assert.NoError(t, err, "expected no error when default matches registered provider")
}

func TestValidateProviderConfiguration_StorageDefaultWithoutProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer()
	c.SetStorageDefaultProvider("missing-storage")

	err := c.ValidateProviderConfiguration()
	require.Error(t, err, "expected error when default provider is not registered")
	assert.Contains(t, err.Error(), "storage default provider", "error should mention storage provider")
}

func TestValidateProviderConfiguration_StorageDefaultWithProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer()
	c.AddStorageProvider("my-storage", storage_mock.NewMockStorageProvider())
	c.SetStorageDefaultProvider("my-storage")

	err := c.ValidateProviderConfiguration()
	assert.NoError(t, err, "expected no error when default matches registered provider")
}

func TestValidateProviderConfiguration_CacheDefaultWithoutProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer()
	c.SetCacheDefaultProvider("missing-cache")

	err := c.ValidateProviderConfiguration()
	require.Error(t, err, "expected error when default provider is not registered")
	assert.Contains(t, err.Error(), "cache default provider", "error should mention cache provider")
}

func TestValidateProviderConfiguration_CryptoDefaultWithoutProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer()
	c.SetCryptoDefaultProvider("missing-crypto")

	err := c.ValidateProviderConfiguration()
	require.Error(t, err, "expected error when default provider is not registered")
	assert.Contains(t, err.Error(), "crypto default provider", "error should mention crypto provider")
}

func TestValidateProviderConfiguration_NotificationDefaultWithoutProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer()
	c.SetNotificationDefaultProvider("missing-notification")

	err := c.ValidateProviderConfiguration()
	require.Error(t, err, "expected error when default provider is not registered")
	assert.Contains(t, err.Error(), "notification default provider", "error should mention notification provider")
}

func TestValidateProviderConfiguration_MultipleErrors(t *testing.T) {
	t.Parallel()
	c := NewContainer()
	c.SetEmailDefaultProvider("missing-email")
	c.SetStorageDefaultProvider("missing-storage")

	err := c.ValidateProviderConfiguration()
	require.Error(t, err, "expected error when multiple defaults are missing")
	assert.Contains(t, err.Error(), "email default provider", "error should mention email provider")
	assert.Contains(t, err.Error(), "storage default provider", "error should mention storage provider")
}

func TestValidateProviderConfiguration_DefaultPointsToWrongProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer()
	c.AddEmailProvider("provider-a", email_mock.NewMockEmailProvider())
	c.SetEmailDefaultProvider("provider-b")

	err := c.ValidateProviderConfiguration()
	require.Error(t, err, "expected error when default points to unregistered provider")
	assert.Contains(t, err.Error(), "provider-b", "error should mention the missing provider name")
	assert.Contains(t, err.Error(), "provider-a", "error should list available providers")
}
