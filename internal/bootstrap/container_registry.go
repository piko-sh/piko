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
	"os"
	"path/filepath"
	"time"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/persistence"
	"piko.sh/piko/internal/registry/registry_adapters"
	"piko.sh/piko/internal/registry/registry_dal"
	registry_otter "piko.sh/piko/internal/registry/registry_dal/otter"
	registry_querier_postgres "piko.sh/piko/internal/registry/registry_dal/querier_postgres"
	registry_querier_sqlite "piko.sh/piko/internal/registry/registry_dal/querier_sqlite"
	"piko.sh/piko/internal/registry/registry_dal/snapshot"
	"piko.sh/piko/internal/registry/registry_dal/union"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/render/render_adapters"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/internal/storage/storage_adapters/provider_disk"
	"piko.sh/piko/internal/storage/storage_adapters/provider_fs"
	"piko.sh/piko/internal/storage/storage_adapters/provider_union"
	"piko.sh/piko/internal/storage/storage_adapters/registry_blob_adapter"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// defaultRegistryCapacity is the default maximum number of registry artefacts to store
	// in the embedded cache.
	defaultRegistryCapacity = 100_000

	// registryReleaseHeartbeatInterval is how often a node advances its release's lease
	// heartbeat so a reaper on a shared backend knows the release is still live.
	registryReleaseHeartbeatInterval = 60 * time.Second

	// registryReleaseTTL is how stale a release lease's heartbeat must become before a
	// reaper retires it.
	registryReleaseTTL = 30 * time.Minute
)

// GetRegistryService returns the template registry service, creating it if necessary.
//
// Returns registry_domain.RegistryService which provides template registry operations.
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
// Sets c.registryErr and returns early when the database provider is not available, the
// database type is not supported, or blob storage setup fails.
//
// Concurrency: invoked once under registryOnce; spawns publishReleaseLayer on a goroutine
// when a release overlay and seed are present.
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

	eventBus, err := c.GetEventBus()
	if err != nil {
		c.registryErr = fmt.Errorf("getting event bus for registry service: %w", err)
		return
	}

	variantOrigin := registry_dto.VariantOriginRuntime
	serviceOptions := make([]registry_domain.RegistryServiceOption, 0, 2)
	if c.isBuildTime {
		variantOrigin = registry_dto.VariantOriginBuild
		release, hash := buildReleaseIdentity(c.releaseIDOverride)
		if release == unversionedReleaseID {
			_, l := logger_domain.From(c.GetAppContext(), log)
			l.Warn("Build has no release identifier (no WithReleaseID and no VCS revision); a canary or A/B rollout needs a distinct release per build, so set WithReleaseID")
		}
		serviceOptions = append(serviceOptions, registry_domain.WithDefaultBuildIdentity(release, hash))
	}
	serviceOptions = append(serviceOptions, registry_domain.WithDefaultVariantOrigin(variantOrigin))
	c.registryService = registry_domain.NewRegistryService(
		metaStore, blobStores, eventBus, metadataCache,
		serviceOptions...,
	)

	if c.registryReleaseOverlay != nil && c.registryReleaseSeed != nil {
		replicator := c.buildReleaseBlobReplicator(blobStores, c.registryReleaseOverlay)
		c.startReleasePublish(c.registryReleaseOverlay, c.registryReleaseSeed, replicator)
	}
}

// createRegistryMetadataStore creates the metadata store using the provider's factory.
// This approach avoids importing driver-specific packages into the bootstrap layer, so
// users only download dependencies for drivers they explicitly import.
//
// Returns registry_domain.MetadataStore which provides metadata storage operations.
// Returns error when the database provider is unavailable or the factory fails.
func (c *Container) createRegistryMetadataStore() (registry_domain.MetadataStore, error) {
	if c.isBuildTime {
		return c.createProviderRegistryDAL()
	}

	if c.embeddedPikoFS != nil {
		return c.createEmbeddedUnionRegistryStore()
	}

	if c.dbRegistrations != nil {
		if _, registered := c.dbRegistrations[DatabaseNameRegistry]; registered {
			store, err := c.createQuerierRegistryDAL()
			if err != nil {
				return nil, err
			}

			seed, err := persistence.LoadRegistryArtefactsFromFS(c.GetAppContext(), c.registrySeedFS())
			if err != nil {
				return nil, fmt.Errorf("loading registry seed to publish into SQL backend: %w", err)
			}
			c.registryReleaseOverlay = store
			c.registryReleaseSeed = seed
			return store, nil
		}
	}

	return c.createProviderRegistryDAL()
}

// createEmbeddedUnionRegistryStore builds the union registry store for a single-binary
// deploy.
//
// Returns registry_domain.MetadataStore which is the composed union store.
// Returns error when the seed cannot be loaded or the overlay cannot be built.
func (c *Container) createEmbeddedUnionRegistryStore() (registry_domain.MetadataStore, error) {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Creating union registry store from the embedded base and a writable overlay")

	seed, err := persistence.LoadRegistryArtefactsFromFS(c.GetAppContext(), c.embeddedPikoFS)
	if err != nil {
		return nil, fmt.Errorf("loading embedded registry seed: %w", err)
	}
	base := snapshot.New(seed)

	overlay, err := c.buildEmbeddedRegistryOverlay()
	if err != nil {
		return nil, err
	}

	store := union.New(base, overlay)
	if inspector, ok := store.(registry_domain.RegistryInspector); ok {
		c.registryInspector = inspector
	}

	c.registryReleaseOverlay = overlay
	c.registryReleaseSeed = seed

	return store, nil
}

// startReleasePublish spawns the async release publish joined to shutdown, so the
// goroutine is cancelled and awaited before the registry stores it writes into close.
//
// Takes overlay (registry_domain.MetadataStore) which is the writable overlay to publish
// into.
// Takes seed ([]*registry_dto.ArtefactMeta) which is the embedded base's artefact set.
// Takes replicator (registry_domain.BlobReplicator) which copies bytes into the shared
// blob store ahead of each record, or nil.
//
// Concurrency: spawns one background goroutine that a registered shutdown handler cancels
// and joins.
func (c *Container) startReleasePublish(overlay registry_domain.MetadataStore, seed []*registry_dto.ArtefactMeta, replicator registry_domain.BlobReplicator) {
	ctx, cancel := context.WithCancel(c.GetAppContext())
	done := make(chan struct{})
	shutdown.Register(c.GetAppContext(), "RegistryReleasePublish", func(_ context.Context) error {
		cancel()
		<-done
		return nil
	})
	go func() {
		defer close(done)
		c.publishReleaseLayer(ctx, overlay, seed, replicator)
	}()
}

// publishReleaseLayer publishes this binary's base seed into a shared overlay as an
// immutable release layer, then starts the lease heartbeat, so a canary or rolling deploy
// can coexist with other releases on one shared database and every node can serve the
// other releases' assets by storage key.
//
// Takes overlay (registry_domain.MetadataStore) which is the writable overlay to publish
// into.
// Takes seed ([]*registry_dto.ArtefactMeta) which is the embedded base's artefact set.
// Takes replicator (registry_domain.BlobReplicator) which copies bytes into the shared
// blob store ahead of each record, or nil when the deployment's bytes are already shared.
func (c *Container) publishReleaseLayer(ctx context.Context, overlay registry_domain.MetadataStore, seed []*registry_dto.ArtefactMeta, replicator registry_domain.BlobReplicator) {
	defer goroutine.RecoverPanic(ctx, "bootstrap.publishReleaseLayer")

	if _, ok := overlay.(registry_domain.ReleasePublisher); !ok {
		return
	}

	ctx, l := logger_domain.From(ctx, log)
	release, _ := buildReleaseIdentity(c.releaseIDOverride)
	if release == unversionedReleaseID {
		l.Warn("Publishing as 'unversioned' (no WithReleaseID and no VCS revision); concurrent canary releases would collide on one release id, so set WithReleaseID per build")
	}
	digest := registry_domain.SeedDigest(seed)

	publishOptions := make([]registry_domain.PublishOption, 0, 1)
	if replicator != nil {
		publishOptions = append(publishOptions, registry_domain.WithBlobReplicator(replicator))
	}
	outcome, err := registry_domain.PublishRelease(ctx, overlay, release, digest, seed, time.Now(), publishOptions...)
	if errors.Is(err, registry_domain.ErrReleaseDigestConflict) {
		l.Error("Release digest conflict: two different builds share one release id, so this "+
			"node's assets cannot be published for cross-release serving; set WithReleaseID per "+
			"build (or retire the conflicting release) to fix the deploy",
			logger_domain.String("release", release), logger_domain.Error(err))
		return
	}
	if err != nil {
		l.Warn("Publishing release layers into the shared overlay failed; this node still serves "+
			"its own base, but cross-release serving is degraded until publish succeeds",
			logger_domain.String("release", release), logger_domain.Error(err))
		return
	}
	l.Internal("Registry release publish complete",
		logger_domain.String("release", release),
		logger_domain.String("outcome", outcome.String()))

	if shouldStartReleaseHeartbeat(outcome) {
		c.startRegistryReleaseHeartbeat(overlay, release)
	}
}

// shouldStartReleaseHeartbeat reports whether a publish outcome means this node owns (or
// co-owns) a published release whose lease it must keep alive.
//
// Takes outcome (registry_domain.PublishOutcome) which is the publish result.
//
// Returns bool which is true when the heartbeat loop should start.
func shouldStartReleaseHeartbeat(outcome registry_domain.PublishOutcome) bool {
	return outcome == registry_domain.PublishOutcomePublished ||
		outcome == registry_domain.PublishOutcomeAlreadyPublished
}

// startRegistryReleaseHeartbeat runs a background loop that advances this release's lease
// heartbeat and reaps releases whose nodes have all gone away, for the lifetime of the
// app context. The reaper excludes this node's own release and every retire is
// idempotent, so all live nodes may run it concurrently without coordination.
//
// Takes overlay (registry_domain.MetadataStore) which is the shared writable overlay.
// Takes releaseID (string) which is this node's release, heartbeated and never reaped by
// it.
//
// Concurrency: safe for concurrent use; spawns a background goroutine that runs until the
// registered shutdown handler cancels its context and joins it.
func (c *Container) startRegistryReleaseHeartbeat(overlay registry_domain.MetadataStore, releaseID string) {
	ctx, cancel := context.WithCancel(c.GetAppContext())
	ctx, l := logger_domain.From(ctx, log)
	done := make(chan struct{})
	shutdown.Register(c.GetAppContext(), "RegistryReleaseHeartbeat", func(_ context.Context) error {
		cancel()
		<-done
		return nil
	})
	go func() {
		defer close(done)
		defer goroutine.RecoverPanic(ctx, "bootstrap.registryReleaseHeartbeat")
		ticker := time.NewTicker(registryReleaseHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := registry_domain.HeartbeatRelease(ctx, overlay, releaseID, time.Now()); err != nil {
					l.Warn("Release heartbeat failed; other nodes will reap this release once the "+
						"lease heartbeat passes the TTL",
						logger_domain.String("release", releaseID), logger_domain.Error(err))
				}
				if _, err := registry_domain.ReapExpiredReleases(ctx, overlay, releaseID, registryReleaseTTL, time.Now()); err != nil {
					l.Warn("Release reaper failed", logger_domain.String("release", releaseID), logger_domain.Error(err))
				}
			}
		}
	}()
}

// buildEmbeddedRegistryOverlay builds the writable overlay layer for the union registry
// store.
//
// It is a registered SQL backend when one is configured, otherwise an in-memory otter
// store. The overlay deliberately avoids the WAL-backed provider, which requires a
// writable .piko and would fail a read-only distroless boot; runtime data that must
// survive a restart belongs in a configured backend, not the ephemeral in-memory overlay.
//
// Returns registry_domain.MetadataStore which is the writable overlay.
// Returns error when the overlay backend cannot be built.
func (c *Container) buildEmbeddedRegistryOverlay() (registry_domain.MetadataStore, error) {
	if c.dbRegistrations != nil {
		if _, registered := c.dbRegistrations[DatabaseNameRegistry]; registered {
			return c.createQuerierRegistryDAL()
		}
	}

	dal, err := registry_otter.NewOtterDAL(registry_otter.Config{})
	if err != nil {
		return nil, fmt.Errorf("creating in-memory registry overlay: %w", err)
	}
	return dal, nil
}

// registrySeedFS returns the filesystem holding the build registry seed snapshot.
//
// The source is the embedded seed for a single-binary deploy, otherwise the on-disk .piko
// where generation wrote the snapshot for a file-based deploy. The returned FS always
// loads: LoadRegistryArtefactsFromFS no-ops when the snapshot is absent, so a project
// that never produced a snapshot publishes nothing.
//
// Returns fs.FS which holds the build registry seed snapshot.
func (c *Container) registrySeedFS() fs.FS {
	if c.embeddedPikoFS != nil {
		return c.embeddedPikoFS
	}
	return os.DirFS(filepath.Join(deref(c.serverConfig.Paths.BaseDir, "."), config.PikoInternalPath))
}

// RetireRegistryRelease removes a retired release's artefact layers from the registry
// backend.
//
// It deletes every `(id, release)` layer belonging to the release, leaving every other
// release's layers untouched, and drops the release's publish lease.
//
// Takes release which is the release id to retire.
//
// Returns error when the registry service cannot be built or the retire fails.
func (c *Container) RetireRegistryRelease(ctx context.Context, release string) error {
	if _, err := c.GetRegistryService(); err != nil {
		return fmt.Errorf("building registry service to retire release %q: %w", release, err)
	}
	store := c.registryReleaseOverlay
	if store == nil {
		store = c.registryMetaStore
	}
	return registry_domain.RetireRelease(ctx, store, release)
}

// createQuerierRegistryDAL creates a registry DAL from a querier-managed database
// connection registered via AddDatabase(DatabaseNameRegistry, ...).
//
// Returns registry_domain.MetadataStore which is the querier-backed metadata store.
// Returns error when the database connection cannot be obtained.
func (c *Container) createQuerierRegistryDAL() (registry_domain.MetadataStore, error) {
	database, driver, err := c.resolveQuerierDatabase(DatabaseNameRegistry, "registry")
	if err != nil {
		return nil, err
	}

	if isPostgresDriver(driver) {
		dal := registry_querier_postgres.New(database)
		if inspector, ok := dal.(registry_domain.RegistryInspector); ok {
			c.registryInspector = inspector
		}
		return dal, nil
	}

	dal := registry_querier_sqlite.New(database)
	if inspector, ok := dal.(registry_domain.RegistryInspector); ok {
		c.registryInspector = inspector
	}

	return dal, nil
}

// createProviderRegistryDAL creates a registry DAL from the default otter in-memory
// backend with WAL persistence.
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

// loadEmbeddedRegistryDAL loads registry metadata from the embedded .piko filesystem and
// creates an otter-backed DAL.
//
// Returns registry_domain.MetadataStore which is the embedded otter-backed metadata
// store.
// Returns error when the cache or DAL cannot be created.
func (c *Container) loadEmbeddedRegistryDAL() (registry_domain.MetadataStore, error) {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Creating registry DAL from embedded .piko filesystem")

	registryCache, err := persistence.LoadRegistryCacheFromFS(
		c.GetAppContext(), c.embeddedPikoFS, defaultRegistryCapacity)
	if err != nil {
		return nil, fmt.Errorf("loading registry cache from embedded fs: %w", err)
	}

	shutdown.Register(c.GetAppContext(), "EmbeddedRegistryCache", func(ctx context.Context) error {
		return registryCache.Close(ctx)
	})

	dal, dalErr := registry_otter.NewOtterDAL(
		registry_otter.Config{},
		registry_otter.WithCache(registryCache),
	)
	if dalErr != nil {
		return nil, fmt.Errorf("creating embedded registry DAL: %w", dalErr)
	}

	if rebuilder, ok := dal.(interface{ RebuildIndexes(context.Context) }); ok {
		rebuilder.RebuildIndexes(c.GetAppContext())
	}

	c.captureRegistryInspector(dal)
	return dal, nil
}

// captureRegistryInspector stores the RegistryInspector from a DAL if the underlying type
// implements it.
//
// Takes dal (registry_dal.RegistryDALWithTx) which may also implement RegistryInspector.
func (c *Container) captureRegistryInspector(dal registry_dal.RegistryDALWithTx) {
	var dalAny any = dal
	if inspector, ok := dalAny.(registry_domain.RegistryInspector); ok {
		c.registryInspector = inspector
	}
}

// createRegistryBlobStores creates the blob stores for the registry service. Closes
// metaStore on error to prevent resource leaks.
//
// Takes metaStore (registry_domain.MetadataStore) which is closed on error.
//
// Returns map[string]registry_domain.BlobStore which contains blob stores keyed by
// storage backend ID.
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

// createRegistryMetadataCache creates the metadata cache from config or provider. When a
// config is present, the cache is built via the cache hexagon's builder using the
// "artefact-metadata" factory blueprint registered by registry_adapters.
//
// Returns registry_domain.MetadataCache which may be nil if no cache is configured.
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

// registerRegistryShutdownHandlers registers shutdown handlers for registry resources.
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

// getRegistryBlobProvider returns the storage provider for registry blob storage using a
// priority order: "system" -> the explicitly registered provider (configured default
// name, "default", or the sole one) -> embedded fs.FS -> built-in disk at .piko/blobs/.
//
// Returns storage_domain.StorageProviderPort which provides blob storage access.
// Returns error when a configured provider is missing or the built-in disk provider
// cannot be created.
func (c *Container) getRegistryBlobProvider() (storage_domain.StorageProviderPort, error) {
	_, l := logger_domain.From(c.GetAppContext(), log)

	base, err := c.buildEmbeddedBlobBase()
	if err != nil {
		return nil, err
	}

	overlay, err := c.buildBlobOverlay(base != nil)
	if err != nil {
		return nil, err
	}

	if base != nil {
		l.Internal("Using union blob provider: embedded base with an overlay",
			logger_domain.Bool("writable", overlay != nil))
		c.registryBlobOverlay = overlay
		return provider_union.New(base, overlay), nil
	}
	return overlay, nil
}

// buildReleaseBlobReplicator builds the byte-replication half of the publish protocol: a
// function that copies one variant's bytes (and its chunks' bytes) from this node's blob
// store into the writable shared overlay, so a foreign-release node that resolves a
// published record can also serve its bytes.
//
// Takes blobStores (map[string]registry_domain.BlobStore) which resolve a variant's
// backend to this node's composed store (base-first reads).
// Takes overlayMeta (registry_domain.MetadataStore) which is the shared metadata overlay
// whose reference counts gate the copy.
//
// Returns registry_domain.BlobReplicator, or nil when this deployment has no shared
// writable blob store to replicate into.
func (c *Container) buildReleaseBlobReplicator(blobStores map[string]registry_domain.BlobStore, overlayMeta registry_domain.MetadataStore) registry_domain.BlobReplicator {
	if c.embeddedPikoFS == nil || c.registryBlobOverlay == nil {
		return nil
	}

	overlayStore, err := registry_blob_adapter.NewBlobStoreAdapter(registry_blob_adapter.Config{
		Provider:   c.registryBlobOverlay,
		Repository: "",
	})
	if err != nil {
		_, l := logger_domain.From(c.GetAppContext(), log)
		l.Warn("Building the release blob replicator failed; published records will not have "+
			"their bytes replicated into the shared store, so foreign-release nodes may not "+
			"resolve this release's assets",
			logger_domain.Error(err))
		return nil
	}

	replicator := &releaseBlobReplicator{
		sourceStores: blobStores,
		overlayMeta:  overlayMeta,
		overlayStore: overlayStore,
	}
	return replicator.replicate
}

// releaseBlobReplicator copies a published variant's bytes into the shared blob overlay
// ahead of its record, so a foreign-release node that resolves the record can also serve
// the bytes.
type releaseBlobReplicator struct {
	// sourceStores resolve a variant's backend id to this node's composed blob store.
	sourceStores map[string]registry_domain.BlobStore

	// overlayMeta is the shared metadata overlay whose reference counts gate the copy.
	overlayMeta registry_domain.MetadataStore

	// overlayStore is the writable shared blob store the bytes are copied into.
	overlayStore *registry_blob_adapter.BlobStoreAdapter
}

// replicate copies a variant's own bytes and every chunk's bytes into the shared overlay.
//
// Takes variant (*registry_dto.Variant) whose bytes are replicated.
//
// Returns error when a blob cannot be read from this node's store or written to the
// shared one.
func (r *releaseBlobReplicator) replicate(ctx context.Context, variant *registry_dto.Variant) error {
	if err := r.copyKey(ctx, variant.StorageBackendID, variant.StorageKey); err != nil {
		return err
	}
	for i := range variant.Chunks {
		chunk := &variant.Chunks[i]
		if err := r.copyKey(ctx, chunk.StorageBackendID, chunk.StorageKey); err != nil {
			return err
		}
	}
	return nil
}

// copyKey copies one blob into the shared overlay unless it is already replicated.
//
// Takes backendID (string) which selects this node's source blob store.
// Takes key (string) which is the content-addressed blob key.
//
// Returns error when the source store is unknown, the read fails, or the write fails.
func (r *releaseBlobReplicator) copyKey(ctx context.Context, backendID, key string) error {
	if r.alreadyReplicated(ctx, key) {
		return nil
	}
	source, ok := r.sourceStores[backendID]
	if !ok {
		return fmt.Errorf("no blob store for backend '%s'", backendID)
	}
	reader, err := source.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("reading blob '%s' for replication: %w", key, err)
	}
	defer func() { _ = reader.Close() }()
	if err := r.overlayStore.Put(ctx, key, reader); err != nil {
		return fmt.Errorf("writing blob '%s' to the shared store: %w", key, err)
	}
	return nil
}

// alreadyReplicated reports whether the shared overlay already holds a blob's bytes.
//
// It is replicated when its reference count is positive (references are written only
// after bytes) or when the bytes exist in the shared store from a crash between the byte
// write and the reference. An unreadable reference count or existence probe is treated as
// not replicated so the copy proceeds, since Put is idempotent and a redundant copy is
// harmless.
//
// Takes key (string) which is the content-addressed blob key.
//
// Returns bool which is true when the bytes are already in the shared overlay.
func (r *releaseBlobReplicator) alreadyReplicated(ctx context.Context, key string) bool {
	count, err := r.overlayMeta.GetBlobRefCount(ctx, key)
	if err == nil && count > 0 {
		return true
	}
	exists, err := r.overlayStore.Exists(ctx, key)
	return err == nil && exists
}

// buildEmbeddedBlobBase builds the read-only base blob provider from the embedded .piko
// blobs.
//
// It returns nil when the binary ships no embedded filesystem, so a non-embedded
// deployment has no base layer and serves entirely from its overlay.
//
// Returns storage_domain.StorageProviderPort which is the embedded base provider, or nil.
// Returns error when the embedded blobs subtree cannot be opened.
func (c *Container) buildEmbeddedBlobBase() (storage_domain.StorageProviderPort, error) {
	if c.embeddedPikoFS == nil {
		return nil, nil
	}
	_, l := logger_domain.From(c.GetAppContext(), log)
	if _, statErr := fs.Stat(c.embeddedPikoFS, "blobs"); statErr != nil {
		l.Warn("Embedded .piko has no 'blobs' subtree; asset reads will fall through to the overlay",
			logger_domain.Error(statErr))
	}
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

// buildBlobOverlay builds the writable overlay blob provider.
//
// It is the "system" provider, then any explicitly registered provider, and otherwise the
// built-in disk provider. When a base is present and no external provider is registered
// the overlay is nil, so a single-binary deploy with no object store serves read-only
// from the embed rather than trying to open a writable disk directory it may not have.
//
// Takes hasBase (bool) which reports whether an embedded base layer is present.
//
// Returns storage_domain.StorageProviderPort which is the writable overlay, or nil.
// Returns error when a configured provider is missing or the disk provider cannot be
// created.
func (c *Container) buildBlobOverlay(hasBase bool) (storage_domain.StorageProviderPort, error) {
	_, l := logger_domain.From(c.GetAppContext(), log)

	if p, ok := c.storageProviders[storage_dto.StorageProviderSystem]; ok {
		l.Internal("Using 'system' storage provider as the registry blob overlay")
		return p, nil
	}

	if len(c.storageProviders) > 0 {
		name, provider, err := c.selectExplicitStorageProvider()
		if err != nil {
			return nil, err
		}
		l.Internal("Using explicitly registered storage provider as the registry blob overlay",
			logger_domain.String("provider", name))
		return provider, nil
	}

	if hasBase {
		return nil, nil
	}

	l.Internal("Using built-in disk provider for registry blobs (no providers configured)")
	blobDir := filepath.Join(deref(c.serverConfig.Paths.BaseDir, "."), config.PikoInternalPath, "blobs")
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

// GetRenderRegistry returns the component render registry, creating it if necessary.
//
// Returns render_domain.RegistryPort which provides access to render components.
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

// SetRenderRegistryOverride sets a custom render registry to bypass the default creation
// which requires database connectivity. Use it for LSP and other lightweight tools that
// don't need full render capabilities.
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
	c.renderRegistry = render_adapters.NewDataLoaderRegistryAdapter(registryService, &render_adapters.DataLoaderAdapterConfig{}, deref(c.serverConfig.Paths.ArtefactServePath, "/_piko/assets"))

	if closer, ok := c.renderRegistry.(interface{ Close() }); ok {
		shutdown.Register(c.GetAppContext(), "RenderRegistry", func(_ context.Context) error {
			closer.Close()
			return nil
		})
	}

	shutdown.Register(c.GetAppContext(), "SpriteSheetCache", func(_ context.Context) error {
		render_domain.ShutdownSpriteSheetCache()
		return nil
	})
}
