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
	"strings"
	"testing"

	"piko.sh/piko/internal/config"
	email_mock "piko.sh/piko/internal/email/email_adapters/provider_mock"
	storage_mock "piko.sh/piko/internal/storage/storage_adapters/provider_mock"
)

func TestValidateProviderConfiguration_NoProvidersNoDefaults(t *testing.T) {
	t.Parallel()
	c := NewContainer(config.NewConfigProvider())

	err := c.ValidateProviderConfiguration()
	if err != nil {
		t.Errorf("expected no error with empty configuration, got: %v", err)
	}
}

func TestValidateProviderConfiguration_EmailDefaultWithoutProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer(config.NewConfigProvider())
	c.SetEmailDefaultProvider("missing-provider")

	err := c.ValidateProviderConfiguration()
	if err == nil {
		t.Fatal("expected error when default provider is not registered")
	}
	if !strings.Contains(err.Error(), "email default provider") {
		t.Errorf("error should mention email provider, got: %v", err)
	}
	if !strings.Contains(err.Error(), "missing-provider") {
		t.Errorf("error should mention the missing provider name, got: %v", err)
	}
}

func TestValidateProviderConfiguration_EmailDefaultWithProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer(config.NewConfigProvider())
	c.AddEmailProvider("my-email", email_mock.NewMockEmailProvider())
	c.SetEmailDefaultProvider("my-email")

	err := c.ValidateProviderConfiguration()
	if err != nil {
		t.Errorf("expected no error when default matches registered provider, got: %v", err)
	}
}

func TestValidateProviderConfiguration_StorageDefaultWithoutProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer(config.NewConfigProvider())
	c.SetStorageDefaultProvider("missing-storage")

	err := c.ValidateProviderConfiguration()
	if err == nil {
		t.Fatal("expected error when default provider is not registered")
	}
	if !strings.Contains(err.Error(), "storage default provider") {
		t.Errorf("error should mention storage provider, got: %v", err)
	}
}

func TestValidateProviderConfiguration_StorageDefaultWithProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer(config.NewConfigProvider())
	c.AddStorageProvider("my-storage", storage_mock.NewMockStorageProvider())
	c.SetStorageDefaultProvider("my-storage")

	err := c.ValidateProviderConfiguration()
	if err != nil {
		t.Errorf("expected no error when default matches registered provider, got: %v", err)
	}
}

func TestValidateProviderConfiguration_CacheDefaultWithoutProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer(config.NewConfigProvider())
	c.SetCacheDefaultProvider("missing-cache")

	err := c.ValidateProviderConfiguration()
	if err == nil {
		t.Fatal("expected error when default provider is not registered")
	}
	if !strings.Contains(err.Error(), "cache default provider") {
		t.Errorf("error should mention cache provider, got: %v", err)
	}
}

func TestValidateProviderConfiguration_CryptoDefaultWithoutProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer(config.NewConfigProvider())
	c.SetCryptoDefaultProvider("missing-crypto")

	err := c.ValidateProviderConfiguration()
	if err == nil {
		t.Fatal("expected error when default provider is not registered")
	}
	if !strings.Contains(err.Error(), "crypto default provider") {
		t.Errorf("error should mention crypto provider, got: %v", err)
	}
}

func TestValidateProviderConfiguration_NotificationDefaultWithoutProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer(config.NewConfigProvider())
	c.SetNotificationDefaultProvider("missing-notification")

	err := c.ValidateProviderConfiguration()
	if err == nil {
		t.Fatal("expected error when default provider is not registered")
	}
	if !strings.Contains(err.Error(), "notification default provider") {
		t.Errorf("error should mention notification provider, got: %v", err)
	}
}

func TestValidateProviderConfiguration_MultipleErrors(t *testing.T) {
	t.Parallel()
	c := NewContainer(config.NewConfigProvider())
	c.SetEmailDefaultProvider("missing-email")
	c.SetStorageDefaultProvider("missing-storage")

	err := c.ValidateProviderConfiguration()
	if err == nil {
		t.Fatal("expected error when multiple defaults are missing")
	}
	if !strings.Contains(err.Error(), "email default provider") {
		t.Errorf("error should mention email provider, got: %v", err)
	}
	if !strings.Contains(err.Error(), "storage default provider") {
		t.Errorf("error should mention storage provider, got: %v", err)
	}
}

func TestValidateProviderConfiguration_DefaultPointsToWrongProvider(t *testing.T) {
	t.Parallel()
	c := NewContainer(config.NewConfigProvider())
	c.AddEmailProvider("provider-a", email_mock.NewMockEmailProvider())
	c.SetEmailDefaultProvider("provider-b")

	err := c.ValidateProviderConfiguration()
	if err == nil {
		t.Fatal("expected error when default points to unregistered provider")
	}
	if !strings.Contains(err.Error(), "provider-b") {
		t.Errorf("error should mention the missing provider name, got: %v", err)
	}
	if !strings.Contains(err.Error(), "provider-a") {
		t.Errorf("error should list available providers, got: %v", err)
	}
}
