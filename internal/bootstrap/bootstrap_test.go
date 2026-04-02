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

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/config"
)

func TestBuildersDispatchTable(t *testing.T) {

	expectedModes := []string{runModeProd, runModeDev, runModeDevInterpreted}

	for _, mode := range expectedModes {
		t.Run("mode_"+mode, func(t *testing.T) {
			builder, ok := builders[mode]
			assert.True(t, ok, "expected builder for mode %q to be registered", mode)
			assert.NotNil(t, builder, "builder for mode %q should not be nil", mode)
		})
	}

	assert.Len(t, builders, len(expectedModes), "unexpected number of registered builders")
}

func TestRunModeConstants(t *testing.T) {

	assert.Equal(t, "dev", runModeDev)
	assert.Equal(t, "dev-i", runModeDevInterpreted)
	assert.Equal(t, "prod", runModeProd)
}

func TestManifestConstants(t *testing.T) {

	assert.Equal(t, "json", manifestFormatJSON)
	assert.Equal(t, "flatbuffers", manifestFormatFlatbuffers)
	assert.Equal(t, "manifest.json", manifestFilenameJSON)
	assert.Equal(t, "manifest.bin", manifestFilenameBinary)
}

func TestWithCSRFSecret(t *testing.T) {
	container := &Container{}
	secret := []byte("my-secret-key")

	opt := WithCSRFSecret(secret)
	opt(container)

	require.NotNil(t, container.csrfSecretKeyProvider)
	result := container.csrfSecretKeyProvider()
	assert.Equal(t, secret, result)
}

func TestWithMemoryRegistryCache(t *testing.T) {
	container := &Container{}

	opt := WithMemoryRegistryCache()
	opt(container)

	require.NotNil(t, container.registryMetadataCacheConfig)
	assert.Equal(t, uint64(defaultRegistryCacheSizeMB*megabyte), container.registryMetadataCacheConfig.MaxWeight)
	assert.Equal(t, int64(defaultRegistryCacheTTLMinutes*60*1e9), int64(container.registryMetadataCacheConfig.TTL))
	assert.True(t, container.registryMetadataCacheConfig.StatsEnabled)
}

func TestWithServerConfigDefaults(t *testing.T) {
	container := &Container{}
	defaults := &config.ServerConfig{}

	opt := WithServerConfigDefaults(defaults)
	opt(container)

	assert.Same(t, defaults, container.configServerDefaults)
}

func TestWithValidator(t *testing.T) {
	container := &Container{}
	customValidator := &testStructValidator{}

	opt := WithValidator(customValidator)
	opt(container)

	assert.Same(t, customValidator, container.validatorOverride)
}

type testStructValidator struct{}

func (*testStructValidator) Struct(any) error { return nil }

func TestNewContainer(t *testing.T) {
	configProvider := config.NewConfigProvider()

	container := NewContainer(configProvider)

	assert.NotNil(t, container)
	assert.Same(t, configProvider, container.config)
	assert.NotNil(t, container.metadataCacheProvider)

	assert.Nil(t, container.typeDataProvider)
	assert.NotNil(t, container.csrfSecretKeyProvider)
}

func TestNewContainerWithOptions(t *testing.T) {
	configProvider := config.NewConfigProvider()
	secret := []byte("test-secret")

	container := NewContainer(configProvider,
		WithCSRFSecret(secret),
		WithMemoryRegistryCache(),
	)

	assert.NotNil(t, container)
	assert.Equal(t, secret, container.csrfSecretKeyProvider())
	assert.NotNil(t, container.registryMetadataCacheConfig)
}

func TestBuildOperationHasPublicEntryPoint(t *testing.T) {
	tests := []struct {
		name        string
		entryPoints []annotator_dto.EntryPoint
		expected    bool
	}{
		{
			name:        "empty entry points",
			entryPoints: []annotator_dto.EntryPoint{},
			expected:    false,
		},
		{
			name: "only private partials",
			entryPoints: []annotator_dto.EntryPoint{
				{Path: "partials/header.pk", IsPage: false, IsPublic: false},
				{Path: "partials/footer.pk", IsPage: false, IsPublic: false},
			},
			expected: false,
		},
		{
			name: "has public page",
			entryPoints: []annotator_dto.EntryPoint{
				{Path: "pages/index.pk", IsPage: true, IsPublic: true},
			},
			expected: true,
		},
		{
			name: "has explicitly public partial",
			entryPoints: []annotator_dto.EntryPoint{
				{Path: "partials/card.pk", IsPage: false, IsPublic: true},
			},
			expected: true,
		},
		{
			name: "mixed public page and private partial",
			entryPoints: []annotator_dto.EntryPoint{
				{Path: "partials/header.pk", IsPage: false, IsPublic: false},
				{Path: "pages/index.pk", IsPage: true, IsPublic: true},
			},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			op := &buildOperation{
				entryPoints: tc.entryPoints,
			}
			result := op.hasPublicEntryPoint()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestServiceProbesTable(t *testing.T) {

	expectedServices := []string{
		"RegistryService",
		"CollectionService",
		"OrchestratorService",
		"EventsProvider",
		"StorageService",
		"EmailService",
		"CryptoService",
		"CacheService",
		"SEOService",
		"ImageService",
		"VideoService",
		"LLMService",
		"DatabaseService",
	}

	serviceNames := make(map[string]bool)
	for _, probe := range serviceProbes {
		serviceNames[probe.name] = true
		assert.NotNil(t, probe.getter, "getter for %q should not be nil", probe.name)
	}

	for _, expected := range expectedServices {
		assert.True(t, serviceNames[expected], "expected service %q in serviceProbes table", expected)
	}

	assert.Len(t, serviceProbes, len(expectedServices), "unexpected number of service probes")
}
