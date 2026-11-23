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
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/email/email_adapters/provider_mock"
	"piko.sh/piko/internal/storage/storage_dto"
)

func TestInitialiseForTesting_InstallsInMemoryServices(t *testing.T) {
	installGlobalContainer(t, nil)
	runGlobalShutdownOnCleanup(t)

	container := InitialiseForTesting(provider_mock.NewMockEmailProvider(), "mock-email")

	registry, err := container.GetRegistryService()
	require.NoError(t, err)
	assert.NotNil(t, registry)

	email, err := container.GetEmailService()
	require.NoError(t, err)
	assert.NotNil(t, email)

	cacheService, err := container.GetCacheService()
	require.NoError(t, err)
	assert.NotNil(t, cacheService)

	image, err := container.GetImageService()
	require.NoError(t, err)
	assert.NotNil(t, image)

	assert.Equal(t, storage_dto.StorageProviderDefault, container.storageDefaultProvider)

	rendered, err := container.GetEmailTemplateService().Render(context.Background(), httptest.NewRequest("GET", "/", nil), "welcome", nil, nil, false)
	require.NoError(t, err)
	assert.Equal(t, "Mock Email", rendered.PlainText)
}

func TestInitialiseForTesting_WithoutAnEmailProviderStillInstallsTheTemplateService(t *testing.T) {
	installGlobalContainer(t, nil)
	runGlobalShutdownOnCleanup(t)

	container := InitialiseForTesting(nil, "")

	assert.Empty(t, container.emailProviders)
	assert.NotNil(t, container.GetEmailTemplateService())
}
