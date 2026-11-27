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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/daemon/daemon_frontend"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/templater/templater_domain"
)

type presignConfiguredStorage struct {
	storage_domain.Service
	config storage_domain.PresignConfig
}

func (s *presignConfiguredStorage) GetPresignConfig() storage_domain.PresignConfig {
	return s.config
}

func newPresignConfig(secret string, perMinute int) storage_domain.PresignConfig {
	config := storage_domain.DefaultPresignConfig()
	config.Secret = []byte(secret)
	config.RateLimitPerMinute = perMinute
	return config
}

func TestPresignConfigFor(t *testing.T) {
	configured := newPresignConfig("secret", 30)

	testCases := []struct {
		storage storage_domain.Service
		name    string
		want    storage_domain.PresignConfig
	}{
		{name: "uses the storage service's own configuration", storage: &presignConfiguredStorage{config: configured}, want: configured},
		{name: "falls back to the defaults", storage: &stubStorageService{}, want: storage_domain.DefaultPresignConfig()},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, presignConfigFor(tc.storage))
		})
	}
}

func TestNewPresignHandlers(t *testing.T) {
	testCases := []struct {
		storage         storage_domain.Service
		rateLimit       security_domain.RateLimitService
		name            string
		wantUpload      bool
		wantPublic      bool
		storageUnusable bool
	}{
		{name: "no storage service disables every handler", storageUnusable: true},
		{
			name:       "no presign secret leaves only public downloads",
			storage:    &presignConfiguredStorage{config: newPresignConfig("", 0)},
			wantPublic: true,
		},
		{
			name:       "a presign secret enables every handler",
			storage:    &presignConfiguredStorage{config: newPresignConfig("secret", 0)},
			rateLimit:  &security_domain.MockRateLimitService{},
			wantUpload: true,
			wantPublic: true,
		},
		{
			name:       "a presign rate limit without a service is reported but does not disable uploads",
			storage:    &presignConfiguredStorage{config: newPresignConfig("secret", 30)},
			wantUpload: true,
			wantPublic: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewContainer()
			if tc.storageUnusable {
				c.storageOnce.Do(func() {})
				c.storageErr = errors.New("storage unavailable")
			} else {
				c.SetStorageService(tc.storage)
			}

			upload, download, public := newPresignHandlers(context.Background(), c, tc.rateLimit)

			assert.Equal(t, tc.wantUpload, upload != nil, "upload handler")
			assert.Equal(t, tc.wantUpload, download != nil, "download handler")
			assert.Equal(t, tc.wantPublic, public != nil, "public download handler")
		})
	}
}

func TestNewActionResponseCache(t *testing.T) {
	t.Run("builds a weighted cache from the cache service", func(t *testing.T) {
		c := NewContainer()

		cache := newActionResponseCache(context.Background(), c)

		require.NotNil(t, cache)
		t.Cleanup(func() { _ = cache.Close(context.Background()) })
		require.NoError(t, cache.SetWithTTL(context.Background(), "response", []byte("{}"), time.Minute))
		value, found, err := cache.GetIfPresent(context.Background(), "response")
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, []byte("{}"), value)
	})

	t.Run("disables caching without a cache service", func(t *testing.T) {
		c := NewContainer()
		c.cacheOnce.Do(func() {})
		c.cacheErr = errors.New("cache unavailable")

		assert.Nil(t, newActionResponseCache(context.Background(), c))
	})
}

func TestCreateManifestProvider(t *testing.T) {
	testCases := []struct {
		configure func(t *testing.T, container *Container)
		name      string
		wantErr   bool
	}{
		{
			name: "an embedded manifest is served from memory",
			configure: func(_ *testing.T, container *Container) {
				container.embeddedManifest = []byte("manifest")
			},
		},
		{
			name: "an embedded piko folder without an embedded manifest is rejected",
			configure: func(_ *testing.T, container *Container) {
				container.embeddedPikoFS = fstest.MapFS{}
			},
			wantErr: true,
		},
		{
			name: "a missing dist directory is reported instead of reading without a sandbox",
			configure: func(t *testing.T, container *Container) {
				container.serverConfig.Paths.BaseDir = new(filepath.Join(t.TempDir(), "absent"))
			},
			wantErr: true,
		},
		{
			name: "an existing dist directory is read through a sandbox",
			configure: func(t *testing.T, container *Container) {
				baseDir := t.TempDir()
				require.NoError(t, os.Mkdir(filepath.Join(baseDir, distDirName), 0o750))
				container.serverConfig.Paths.BaseDir = new(baseDir)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			container := NewContainer()
			testCase.configure(t, container)

			provider, err := createManifestProvider(context.Background(), container)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Nil(t, provider)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, provider)
		})
	}
}

func TestCacheWeighers(t *testing.T) {
	testCases := []struct {
		weigh func() uint32
		name  string
		want  uint32
	}{
		{
			name:  "an action response weighs its key and payload",
			weigh: func() uint32 { return actionResponseWeigher("key", []byte("payload")) },
			want:  10,
		},
		{
			name:  "an empty action response weighs nothing",
			weigh: func() uint32 { return actionResponseWeigher("", nil) },
			want:  0,
		},
		{
			name: "a hybrid collection entry weighs its key, blobs and entity tags",
			weigh: func() uint32 {
				value := collection_domain.HybridCacheValue{}
				value.CurrentBlob = []byte("current")
				value.SnapshotBlob = []byte("snap")
				value.CurrentETag = "etag-1"
				value.SnapshotETag = "e2"
				return hybridCacheValueWeigher("key", value)
			},
			want: 22,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, testCase.weigh())
		})
	}
}

func TestRouterResolveServicesFailsWhenCSRFServiceFails(t *testing.T) {
	t.Parallel()

	setupErr := errors.New("csrf setup failed")
	c := NewContainer()
	c.registryServiceOverride = &stubRegistryService{}
	c.capabilityServiceOverride = &stubCapabilityService{}
	c.renderRegistryOverride = &stubRenderRegistryPort{}
	c.csrfOnce.Do(func() { c.csrfErr = setupErr })

	operation := &routerOperation{container: c}

	err := operation.resolveServices(t.Context())

	require.ErrorIs(t, err, setupErr)
	assert.Nil(t, operation.csrfService)
}

func TestBuildCacheMiddlewareConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                     string
		cspConfig                security_dto.CSPRuntimeConfig
		persistenceDisabled      bool
		wantCSPUsesRequestTokens bool
		coreRuntimeBuild         daemon_frontend.CoreRuntimeBuild
	}{
		{
			name:                     "turns the static page cache off under the strict policy, which uses request tokens",
			cspConfig:                security_domain.NewCSPBuilder().WithStrictPolicy().RuntimeConfig(),
			wantCSPUsesRequestTokens: true,
		},
		{
			name:      "keeps the static page cache under the Piko default policy",
			cspConfig: security_domain.NewCSPBuilder().WithPikoDefaults().RuntimeConfig(),
		},
		{
			name:                "keeps the static page cache under a raw policy string",
			cspConfig:           security_dto.CSPRuntimeConfig{Policy: "default-src 'self'"},
			persistenceDisabled: true,
		},
		{
			name:             "keys cached pages by the core runtime build the server sends",
			cspConfig:        security_domain.NewCSPBuilder().WithPikoDefaults().RuntimeConfig(),
			coreRuntimeBuild: daemon_frontend.CoreRuntimeBuildProduction,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			config := buildCacheMiddlewareConfig(testCase.cspConfig, testCase.persistenceDisabled, testCase.coreRuntimeBuild)

			assert.Equal(t, testCase.wantCSPUsesRequestTokens, config.CSPUsesRequestTokens)
			assert.Equal(t, testCase.coreRuntimeBuild, config.CoreRuntimeBuild)
			assert.Equal(t, testCase.persistenceDisabled, config.PersistenceDisabled)
			assert.Equal(t, defaultCacheWriteConcurrency, config.CacheWriteConcurrency)
		})
	}
}

func TestStaticPageCacheTurnedOffByCSP(t *testing.T) {
	t.Parallel()

	storeOf := func(hasCachePolicy ...bool) *templater_domain.MockManifestStoreView {
		keys := make([]string, len(hasCachePolicy))
		entries := make(map[string]templater_domain.PageEntryView, len(hasCachePolicy))
		for index, has := range hasCachePolicy {
			keys[index] = fmt.Sprintf("pages/page%d.pk", index)
			entries[keys[index]] = &templater_domain.MockPageEntryView{
				GetHasCachePolicyFunc: func() bool { return has },
			}
		}
		return &templater_domain.MockManifestStoreView{
			GetKeysFunc: func() []string { return keys },
			GetPageEntryFunc: func(key string) (templater_domain.PageEntryView, bool) {
				entry, found := entries[key]
				return entry, found
			},
		}
	}
	strict := security_domain.NewCSPBuilder().WithStrictPolicy().RuntimeConfig()

	testCases := []struct {
		store     templater_domain.ManifestStoreView
		name      string
		cspConfig security_dto.CSPRuntimeConfig
		want      bool
	}{
		{name: "a token policy and a page with a cache policy", cspConfig: strict, store: storeOf(false, true), want: true},
		{name: "a token policy and no page with a cache policy", cspConfig: strict, store: storeOf(false, false), want: false},
		{name: "a token policy and no pages", cspConfig: strict, store: storeOf(), want: false},
		{name: "a token policy and no manifest", cspConfig: strict, store: nil, want: false},
		{
			name:      "a policy without tokens",
			cspConfig: security_domain.NewCSPBuilder().WithPikoDefaults().RuntimeConfig(),
			store:     storeOf(true),
			want:      false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.want, staticPageCacheTurnedOffByCSP(testCase.cspConfig, testCase.store))
		})
	}
}

func TestBuildRouterConfig_SendsTheCoreRuntimeBuildOfTheContainer(t *testing.T) {
	t.Parallel()

	builds := []daemon_frontend.CoreRuntimeBuild{
		daemon_frontend.CoreRuntimeBuildDevelopment,
		daemon_frontend.CoreRuntimeBuildProduction,
	}

	for _, build := range builds {
		t.Run(build.CoreFileName(), func(t *testing.T) {
			t.Parallel()

			operation := &routerOperation{container: NewContainer(WithCoreRuntimeBuild(build))}

			assert.Equal(t, build, operation.buildRouterConfig(t.Context()).CoreRuntimeBuild)
		})
	}
}
