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

// This file contains registry service related container methods.

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/persistence"
	"piko.sh/piko/internal/registry/registry_adapters"
	"piko.sh/piko/internal/registry/registry_dal"
	registry_otter "piko.sh/piko/internal/registry/registry_dal/otter"
	registry_querier_adapter "piko.sh/piko/internal/registry/registry_dal/querier_adapter"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/render/render_adapters"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/internal/storage/storage_adapters/provider_disk"
	"piko.sh/piko/internal/storage/storage_adapters/provider_fs"
	"piko.sh/piko/internal/storage/storage_adapters/registry_blob_adapter"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// defaultRegistryCapacity is the default maximum number of registry
	// artefacts to store in the embedded cache.
	defaultRegistryCapacity = 100_000
)

// GetRegistryService returns the template registry service, creating it if
// necessary.
//
// Returns registry_domain.RegistryService which provides template registry
// operations.
// Returns error when the service could not be created.
func (c *Container) GetRegistryService() (registry_domain.RegistryService, error) {
	c.registryOnce.Do(func() {
		_, l := logger_domain.From(c.GetAppContext(), log)
		if c.registryServiceOverride != nil {
			l.Internal("Using provided RegistryService override.")
			c.registryService = c.registryServiceOverride
			return
		}
		c.createDefaultRegistryService()
	})
	return c.registryService, c.registryErr
}

// createDefaultRegistryService sets up the default registry service.
//
// Sets c.registryErr and returns early when the database provider is not
// available, the database type is not supported, or blob storage setup fails.
func (c *Container) createDefaultRegistryService() {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Creating default RegistryService...")

	metaStore, err := c.createRegistryMetadataStore()
	if err != nil {
		c.registryErr = err
		return
	}

	blobStores, err := c.createRegistryBlobStores(metaStore)
	if err != nil {
		c.registryErr = err
		return
	}

	metadataCache := c.createRegistryMetadataCache()

	c.registryMetaStore = metaStore
	c.registryMetaCache = metadataCache
	c.registerRegistryShutdownHandlers()

	c.registryService = registry_domain.NewRegistryService(metaStore, blobStores, c.GetEventBus(), metadataCache)
}

// createRegistryMetadataStore creates the metadata store using the provider's
// factory. This approach avoids importing driver-specific packages into the
// bootstrap layer, so users only download dependencies for drivers they
// explicitly import.
//
// Returns registry_domain.MetadataStore which provides metadata storage
// operations.
// Returns error when the database provider is unavailable or the factory fails.
func (c *Container) createRegistryMetadataStore() (registry_domain.MetadataStore, error) {
	if c.dbRegistrations != nil {
		if _, registered := c.dbRegistrations[DatabaseNameRegistry]; registered {
			return c.createQuerierRegistryDAL()
		}
	}

	return c.createProviderRegistryDAL()
}

// createQuerierRegistryDAL creates a registry DAL from a querier-managed
// database connection registered via AddDatabase(DatabaseNameRegistry, ...).
//
// Returns registry_domain.MetadataStore which is the querier-backed metadata store.
// Returns error when the database connection cannot be obtained.
func (c *Container) createQuerierRegistryDAL() (registry_domain.MetadataStore, error) {
	if err := c.runMigrationsIfConfigured(DatabaseNameRegistry); err != nil {
		return nil, fmt.Errorf("failed to migrate registry database: %w", err)
	}

	database, err := c.GetDatabaseConnection(DatabaseNameRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to get registry database connection: %w", err)
	}

	dal := registry_querier_adapter.NewDAL(database)
	c.registryInspector = dal

	return dal, nil
}

// createProviderRegistryDAL creates a registry DAL from the default otter
// in-memory backend with WAL persistence.
//
// Returns registry_domain.MetadataStore which is the otter-backed metadata store.
// Returns error when the otter DAL cannot be created or does not implement
// RegistryDALWithTx.
func (c *Container) createProviderRegistryDAL() (registry_domain.MetadataStore, error) {
	if c.embeddedPikoFS != nil {
		return c.loadEmbeddedRegistryDAL()
	}

	dalAny, err := c.createOtterRegistryDAL()
	if err != nil {
		return nil, fmt.Errorf("failed to create otter registry DAL: %w", err)
	}

	dal, ok := dalAny.(registry_dal.RegistryDALWithTx)
	if !ok {
		return nil, errors.New("otter registry DAL does not implement RegistryDALWithTx")
	}

	if inspector, ok := dalAny.(registry_domain.RegistryInspector); ok {
		c.registryInspector = inspector
	}

	return dal, nil
}

// loadEmbeddedRegistryDAL loads registry metadata from the embedded .piko
// filesystem and creates an otter-backed DAL.
func (c *Container) loadEmbeddedRegistryDAL() (registry_domain.MetadataStore, error) {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Creating registry DAL from embedded .piko filesystem")

	registryCache, err := persistence.LoadRegistryCacheFromFS(
		c.GetAppContext(), c.embeddedPikoFS, defaultRegistryCapacity)
	if err != nil {
		return nil, fmt.Errorf("loading registry cache from embedded fs: %w", err)
	}

	dal, dalErr := registry_otter.NewOtterDAL(
		registry_otter.Config{},
		registry_otter.WithCache(registryCache),
	)
	if dalErr != nil {
		return nil, fmt.Errorf("creating embedded registry DAL: %w", dalErr)
	}

	c.captureRegistryInspector(dal)
	return dal, nil
}

// captureRegistryInspector extracts the RegistryInspector from a DAL if
// the underlying type implements it.
func (c *Container) captureRegistryInspector(dal registry_dal.RegistryDALWithTx) {
	var dalAny any = dal
	if inspector, ok := dalAny.(registry_domain.RegistryInspector); ok {
		c.registryInspector = inspector
	}
}

// createRegistryBlobStores creates the blob stores for the registry service.
// Closes metaStore on error to prevent resource leaks.
//
// Takes metaStore (registry_domain.MetadataStore) which is closed on error.
//
// Returns map[string]registry_domain.BlobStore which contains blob stores keyed
// by storage backend ID.
// Returns error when blob provider or adapter creation fails.
func (c *Container) createRegistryBlobStores(metaStore registry_domain.MetadataStore) (map[string]registry_domain.BlobStore, error) {
	blobProvider, err := c.getRegistryBlobProvider()
	if err != nil {
		_ = metaStore.Close()
		return nil, fmt.Errorf("failed to get blob provider: %w", err)
	}

	blobAdapter, err := registry_blob_adapter.NewBlobStoreAdapter(registry_blob_adapter.Config{
		Provider:   blobProvider,
		Repository: "",
	})
	if err != nil {
		_ = metaStore.Close()
		return nil, fmt.Errorf("failed to create blob store adapter: %w", err)
	}

	return map[string]registry_domain.BlobStore{"local_disk_cache": blobAdapter}, nil
}

// createRegistryMetadataCache creates the metadata cache from config or
// provider. When a config is present, the cache is built via the cache
// hexagon's builder using the "artefact-metadata" factory blueprint registered
// by registry_adapters.
//
// Returns registry_domain.MetadataCache which may be nil if no cache is
// configured.
func (c *Container) createRegistryMetadataCache() registry_domain.MetadataCache {
	_, l := logger_domain.From(c.GetAppContext(), log)

	if c.registryMetadataCacheConfig == nil {
		return c.metadataCacheProvider()
	}

	cacheService, err := c.GetCacheService()
	if err != nil {
		l.Error("Failed to get cache service for registry metadata cache", logger_domain.Error(err))
		return c.metadataCacheProvider()
	}

	cacheConfig := c.registryMetadataCacheConfig

	builder := cache_domain.NewCacheBuilder[string, *registry_dto.ArtefactMeta](cacheService).
		FactoryBlueprint("artefact-metadata").
		Namespace("registry-metadata").
		MaximumWeight(cacheConfig.MaxWeight).
		Weigher(registry_adapters.ArtefactMetaWeigher)

	if cacheConfig.TTL > 0 {
		builder = builder.AccessExpiration(cacheConfig.TTL)
	}

	typedCache, err := builder.Build(c.GetAppContext())
	if err != nil {
		l.Error("Failed to build registry metadata cache", logger_domain.Error(err))
		return c.metadataCacheProvider()
	}

	metaCache := registry_adapters.NewMetadataCache(typedCache)

	l.Internal("Created registry metadata cache via cache hexagon",
		logger_domain.Uint64("maxWeight", cacheConfig.MaxWeight),
		logger_domain.Duration("ttl", cacheConfig.TTL))
	return metaCache
}

// registerRegistryShutdownHandlers registers shutdown handlers for registry
// resources.
func (c *Container) registerRegistryShutdownHandlers() {
	shutdown.Register(c.GetAppContext(), "RegistryMetadataStore", func(_ context.Context) error {
		return c.registryMetaStore.Close()
	})
	if c.registryMetaCache != nil {
		shutdown.Register(c.GetAppContext(), "RegistryMetadataCache", func(shutdownCtx context.Context) error {
			return c.registryMetaCache.Close(shutdownCtx)
		})
	}
}

// getRegistryBlobProvider returns the storage provider for registry blob
// storage using a priority order: "system" -> "default" -> built-in disk at
// .piko/blobs/.
//
// Returns storage_domain.StorageProviderPort which provides blob storage
// access.
// Returns error when the built-in disk provider cannot be created.
func (c *Container) getRegistryBlobProvider() (storage_domain.StorageProviderPort, error) {
	_, l := logger_domain.From(c.GetAppContext(), log)
	if p, ok := c.storageProviders[storage_dto.StorageProviderSystem]; ok {
		l.Internal("Using 'system' storage provider for registry blobs")
		return p, nil
	}

	if p, ok := c.storageProviders[storage_dto.StorageProviderDefault]; ok {
		l.Internal("Using 'default' storage provider for registry blobs (no 'system' configured)")
		return p, nil
	}

	if c.embeddedPikoFS != nil {
		l.Internal("Using embedded fs.FS for registry blobs")
		blobSubFS, err := fs.Sub(c.embeddedPikoFS, "blobs")
		if err != nil {
			return nil, fmt.Errorf("failed to create blob sub-fs: %w", err)
		}
		fsProvider, providerErr := provider_fs.NewFSProvider(blobSubFS)
		if providerErr != nil {
			return nil, fmt.Errorf("failed to create embedded fs blob provider: %w", providerErr)
		}
		return fsProvider, nil
	}

	l.Internal("Using built-in disk provider for registry blobs (no providers configured)")
	blobDir := filepath.Join(deref(c.config.ServerConfig.Paths.BaseDir, "."), config.PikoInternalPath, "blobs")
	blobSandbox, sandboxErr := c.createSandbox("registry-blob-storage", blobDir, safedisk.ModeReadWrite)
	if sandboxErr != nil {
		return nil, fmt.Errorf("failed to create blob storage sandbox: %w", sandboxErr)
	}
	diskProvider, err := provider_disk.NewDiskProvider(provider_disk.Config{
		BaseDirectory: blobDir,
		Sandbox:       blobSandbox,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create built-in disk provider for blobs: %w", err)
	}
	return diskProvider, nil
}

// GetRenderRegistry returns the component render registry, creating it if
// necessary.
//
// Returns render_domain.RegistryPort which provides access to render
// components.
func (c *Container) GetRenderRegistry() render_domain.RegistryPort {
	c.renderRegOnce.Do(func() {
		_, l := logger_domain.From(c.GetAppContext(), log)
		if c.renderRegistryOverride != nil {
			l.Internal("Using provided RenderRegistry override.")
			c.renderRegistry = c.renderRegistryOverride
			return
		}
		c.createDefaultRenderRegistry()
	})
	return c.renderRegistry
}

// SetRenderRegistryOverride sets a custom render registry to bypass the default
// creation which requires database connectivity. Use it for LSP and other
// lightweight tools that don't need full render capabilities.
//
// Takes registry (RegistryPort) which provides the custom render registry.
func (c *Container) SetRenderRegistryOverride(registry render_domain.RegistryPort) {
	c.renderRegistryOverride = registry
}

// createDefaultRenderRegistry sets up the default render registry using a
// DataLoaderRegistryAdapter.
func (c *Container) createDefaultRenderRegistry() {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Creating default RenderRegistry (DataLoaderRegistryAdapter)...")
	registryService, err := c.GetRegistryService()
	if err != nil {
		l.Panic("Failed to get registry service, cannot create render registry", logger_domain.Error(err))
	}
	c.renderRegistry = render_adapters.NewDataLoaderRegistryAdapter(registryService, &render_adapters.DataLoaderAdapterConfig{}, deref(c.config.ServerConfig.Paths.ArtefactServePath, "/_piko/assets"))

	if closer, ok := c.renderRegistry.(interface{ Close() }); ok {
		shutdown.Register(c.GetAppContext(), "RenderRegistry", func(_ context.Context) error {
			closer.Close()
			return nil
		})
	}
}
