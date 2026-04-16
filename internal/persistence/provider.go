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

package persistence

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	cache_adapters_otter "piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	orchestrator_otter "piko.sh/piko/internal/orchestrator/orchestrator_dal/otter"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	registry_otter "piko.sh/piko/internal/registry/registry_dal/otter"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/wal/wal_domain"
)

const (
	// defaultRegistryCapacity is the default capacity for the registry cache.
	defaultRegistryCapacity = 100_000

	// defaultOrchestratorCapacity is the default capacity for the orchestrator.
	defaultOrchestratorCapacity = 100_000

	// defaultSnapshotThreshold is the default number of operations before a cache
	// snapshot is triggered.
	defaultSnapshotThreshold = 10_000
)

var (
	// errProviderNotConnected is returned when an operation is attempted on an
	// otter persistence provider that has not been connected.
	errProviderNotConnected = errors.New("otter provider not connected")

	// log is the package-level logger for the persistence package.
	log = logger_domain.GetLogger("piko/internal/persistence")
)

// Config holds settings for the otter persistence provider.
type Config struct {
	// Persistence sets up optional WAL-based persistence.
	// When nil or Enabled is false, the cache works only in memory.
	Persistence *PersistenceProviderConfig

	// RegistryCapacity is the maximum number of registry artefacts to store.
	// Defaults to 100,000 if zero or negative.
	RegistryCapacity int64

	// OrchestratorCapacity is the maximum number of orchestrator tasks to store.
	// Defaults to 100,000 if zero or negative.
	OrchestratorCapacity int64
}

// PersistenceProviderConfig configures WAL-based persistence for the provider.
type PersistenceProviderConfig struct {
	// WALDir is the folder path for write-ahead log and snapshot files.
	// Defaults to ".piko/wal/persistence" if empty.
	WALDir string

	// SnapshotThreshold is the number of WAL entries before triggering a snapshot.
	// Defaults to 10,000 if zero.
	SnapshotThreshold int

	// Enabled controls whether persistence is active. When false, the caches
	// operate purely in memory.
	Enabled bool

	// SyncMode controls how writes are saved to disk. Defaults to
	// SyncModeBatched for good speed with data safety.
	SyncMode wal_domain.SyncMode
}

// Provider implements the persistence port using an in-memory otter cache.
type Provider struct {
	// registryDAL provides data access for the registry.
	registryDAL any

	// orchestratorDAL provides data access for orchestration operations.
	orchestratorDAL any

	// registryDALFactory provides registry data access layer instances.
	registryDALFactory *otterRegistryDALFactory

	// orchestratorDALFactory is the orchestrator DAL factory; initialised by Connect.
	orchestratorDALFactory *otterOrchestratorDALFactory

	// persistentCaches holds caches created with WAL persistence. These must
	// be closed separately from the DALs because DALs that receive injected
	// caches (ownsCache=false) do not close them.
	persistentCaches []interface{ Close(context.Context) error }

	// config holds the provider settings including cache capacities and
	// persistence options.
	config Config

	// mu guards access to connected and cache fields.
	mu sync.RWMutex

	// connected indicates whether the provider has an active connection.
	connected bool
}

// NewProvider creates a new Otter persistence provider.
//
// Takes config (Config) which specifies cache settings.
//
// Returns *Provider which is the uninitialised provider. Call Connect to
// establish the caches.
func NewProvider(config Config) *Provider {
	return &Provider{
		config: config,
	}
}

// Connect initialises the in-memory storage.
//
// Returns error when the cache cannot be created.
//
// Safe for concurrent use. Uses a mutex to protect connection state.
func (p *Provider) Connect(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.connected {
		return nil
	}

	registryCapacity := valueOrDefault(p.config.RegistryCapacity, defaultRegistryCapacity)
	orchestratorCapacity := valueOrDefault(p.config.OrchestratorCapacity, defaultOrchestratorCapacity)

	var registryOpts []registry_otter.Option
	var orchestratorOpts []orchestrator_otter.Option

	if p.persistenceEnabled() {
		registryCache, orchCache, err := p.createPersistentCaches(registryCapacity, orchestratorCapacity)
		if err != nil {
			return fmt.Errorf("creating persistent caches: %w", err)
		}
		registryOpts = append(registryOpts, registry_otter.WithCache(registryCache))
		orchestratorOpts = append(orchestratorOpts, orchestrator_otter.WithCache(orchCache))
		p.persistentCaches = []interface{ Close(context.Context) error }{registryCache, orchCache}
	}

	registryDAL, err := registry_otter.NewOtterDAL(registry_otter.Config{
		Capacity: registryCapacity,
	}, registryOpts...)
	if err != nil {
		return fmt.Errorf("creating registry DAL: %w", err)
	}
	p.registryDAL = registryDAL

	if p.persistenceEnabled() {
		rebuildIndexes(ctx, registryDAL)
	}

	orchestratorDAL, err := orchestrator_otter.NewOtterDAL(orchestrator_otter.Config{
		Capacity: orchestratorCapacity,
	}, orchestratorOpts...)
	if err != nil {
		return fmt.Errorf("creating orchestrator DAL: %w", err)
	}
	p.orchestratorDAL = orchestratorDAL

	if p.persistenceEnabled() {
		rebuildIndexes(ctx, orchestratorDAL)
	}

	p.registryDALFactory = &otterRegistryDALFactory{dal: registryDAL}
	p.orchestratorDALFactory = &otterOrchestratorDALFactory{dal: orchestratorDAL}

	p.connected = true

	_, l := logger_domain.From(ctx, log)
	l.Internal("Otter persistence provider connected",
		logger_domain.Int64("registry_capacity", registryCapacity),
		logger_domain.Int64("orchestrator_capacity", orchestratorCapacity),
		logger_domain.Bool("persistence_enabled", p.persistenceEnabled()))

	return nil
}

// Close releases resources held by the provider.
//
// Returns error which is always nil for in-memory storage.
//
// Safe for concurrent use.
func (p *Provider) Close(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.connected {
		return nil
	}

	_, l := logger_domain.From(ctx, log)

	if closer, ok := p.registryDAL.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			l.Warn("Error closing registry DAL", logger_domain.Error(err))
		}
	}

	if closer, ok := p.orchestratorDAL.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			l.Warn("Error closing orchestrator DAL", logger_domain.Error(err))
		}
	}

	for _, cache := range p.persistentCaches {
		_ = cache.Close(context.WithoutCancel(ctx))
	}
	p.persistentCaches = nil

	p.registryDAL = nil
	p.orchestratorDAL = nil
	p.connected = false

	l.Internal("Otter persistence provider closed")
	return nil
}

// GetDatabaseType returns the type of database backend in use.
//
// Returns DatabaseType which is always DatabaseTypeOtter for this provider.
func (*Provider) GetDatabaseType() DatabaseType {
	return DatabaseTypeOtter
}

// HealthCheck verifies the provider is operational.
//
// Returns error which is always nil for in-memory storage.
//
// Safe for concurrent use.
func (p *Provider) HealthCheck(_ context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.connected {
		return errProviderNotConnected
	}
	return nil
}

// GetHealthDetails returns provider-specific health metrics.
//
// Returns map[string]any which contains health information including
// provider name, connection state, and cache capacities.
//
// Safe for concurrent use. Protected by a read lock on the provider mutex.
func (p *Provider) GetHealthDetails(_ context.Context) map[string]any {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]any{
		"provider":              "otter",
		"connected":             p.connected,
		"registry_capacity":     p.config.RegistryCapacity,
		"orchestrator_capacity": p.config.OrchestratorCapacity,
	}
}

// RunMigrations does nothing for in-memory storage.
//
// Returns error which is always nil.
func (*Provider) RunMigrations(_ context.Context) error {
	return nil
}

// RegistryDALFactory returns a factory for creating Registry data access layers.
//
// Returns RegistryDALFactory which provides registry DAL instances.
// Returns error when called before Connect has been called.
//
// Safe for concurrent use.
func (p *Provider) RegistryDALFactory() (RegistryDALFactory, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.registryDALFactory == nil {
		return nil, errors.New("otter: RegistryDALFactory called before Connect")
	}
	return p.registryDALFactory, nil
}

// OrchestratorDALFactory returns a factory for creating Orchestrator data
// access layers.
//
// Returns OrchestratorDALFactory which provides orchestrator DAL instances.
// Returns error when called before Connect has been called.
//
// Safe for concurrent use.
func (p *Provider) OrchestratorDALFactory() (OrchestratorDALFactory, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.orchestratorDALFactory == nil {
		return nil, errors.New("otter: OrchestratorDALFactory called before Connect")
	}
	return p.orchestratorDALFactory, nil
}

// persistenceEnabled reports whether WAL-based persistence is configured
// and active.
//
// Returns bool which is true when persistence is both configured and enabled.
func (p *Provider) persistenceEnabled() bool {
	return p.config.Persistence != nil && p.config.Persistence.Enabled
}

// createPersistentCaches creates caches with WAL persistence enabled.
//
// Takes registryCapacity (int64) which sets the maximum registry cache
// size.
// Takes orchestratorCapacity (int64) which sets the maximum orchestrator
// cache size.
//
// Returns cache_domain.ProviderPort[string, *registry_dto.ArtefactMeta] which
// is the WAL-backed artefact metadata cache.
// Returns cache_domain.ProviderPort[string, *orchestrator_domain.Task] which
// is the WAL-backed task cache.
// Returns error when cache creation fails.
func (p *Provider) createPersistentCaches(registryCapacity, orchestratorCapacity int64) (
	registryCache cache_domain.ProviderPort[string, *registry_dto.ArtefactMeta],
	orchCache cache_domain.ProviderPort[string, *orchestrator_domain.Task],
	err error,
) {
	config := p.config.Persistence

	walDir := config.WALDir
	if walDir == "" {
		walDir = ".piko/wal/persistence"
	}

	syncMode := config.SyncMode
	if syncMode == 0 {
		syncMode = wal_domain.SyncModeBatched
	}

	snapshotThreshold := valueOrDefault(config.SnapshotThreshold, defaultSnapshotThreshold)

	registryOpts := cache_dto.Options[string, *registry_dto.ArtefactMeta]{
		MaximumSize: int(registryCapacity),
		ProviderSpecific: cache_adapters_otter.PersistenceConfig[string, *registry_dto.ArtefactMeta]{
			Enabled:    true,
			WALConfig:  buildWALConfig(walDir, "registry", syncMode, snapshotThreshold),
			KeyCodec:   StringKeyCodec{},
			ValueCodec: ArtefactMetaCodec{},
		},
	}

	registryCache, err = cache_adapters_otter.OtterProviderFactory(registryOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("creating registry cache with WAL: %w", err)
	}

	orchestratorOpts := cache_dto.Options[string, *orchestrator_domain.Task]{
		MaximumSize: int(orchestratorCapacity),
		ProviderSpecific: cache_adapters_otter.PersistenceConfig[string, *orchestrator_domain.Task]{
			Enabled:    true,
			WALConfig:  buildWALConfig(walDir, "orchestrator", syncMode, snapshotThreshold),
			KeyCodec:   StringKeyCodec{},
			ValueCodec: TaskCodec{},
		},
	}

	orchCache, err = cache_adapters_otter.OtterProviderFactory(orchestratorOpts)
	if err != nil {
		_ = registryCache.Close(context.Background())
		return nil, nil, fmt.Errorf("creating orchestrator cache with WAL: %w", err)
	}

	return registryCache, orchCache, nil
}

// otterRegistryDALFactory creates registry data access layers from a shared
// otter DAL. It implements the RegistryDALFactory interface.
type otterRegistryDALFactory struct {
	// dal holds the registry data access layer instance.
	dal any
}

// NewRegistryDAL returns the shared otter registry DAL.
//
// Returns any which is the otter registry DAL.
// Returns error which is always nil.
func (f *otterRegistryDALFactory) NewRegistryDAL() (any, error) {
	return f.dal, nil
}

// otterOrchestratorDALFactory creates orchestrator DALs from a shared otter DAL.
// It implements OrchestratorDALFactory.
type otterOrchestratorDALFactory struct {
	// dal holds the cached OrchestratorDAL instance.
	dal any
}

// NewOrchestratorDAL returns the shared otter orchestrator DAL.
//
// Returns any which is the otter orchestrator DAL instance.
// Returns error which is always nil.
func (f *otterOrchestratorDALFactory) NewOrchestratorDAL() (any, error) {
	return f.dal, nil
}

// valueOrDefault returns v if positive, otherwise fallback.
//
// Takes v (T) which is the value to check.
// Takes fallback (T) which is the default value to use when v is not positive.
//
// Returns T which is v if greater than zero, otherwise fallback.
func valueOrDefault[T int | int64](v, fallback T) T {
	if v <= 0 {
		return fallback
	}
	return v
}

// rebuildIndexes calls RebuildIndexes on the given DAL if it implements
// the method.
//
// Takes ctx (context.Context) which carries logging context.
// Takes dal (any) which is checked for a RebuildIndexes method.
func rebuildIndexes(ctx context.Context, dal any) {
	if rebuilder, ok := dal.(interface{ RebuildIndexes(context.Context) }); ok {
		rebuilder.RebuildIndexes(ctx)
	}
}

// buildWALConfig creates a WAL configuration for a given subdirectory.
//
// Takes walDir (string) which is the base directory for WAL files.
// Takes subDir (string) which is the subdirectory name within walDir.
// Takes syncMode (wal_domain.SyncMode) which controls how writes are synced.
// Takes snapshotThreshold (int) which sets when snapshots are triggered.
//
// Returns wal_domain.Config which is the complete configuration with defaults
// applied.
func buildWALConfig(walDir, subDir string, syncMode wal_domain.SyncMode, snapshotThreshold int) wal_domain.Config {
	return wal_domain.Config{
		Dir:               filepath.Join(walDir, subDir),
		SyncMode:          syncMode,
		SnapshotThreshold: snapshotThreshold,
	}.WithDefaults()
}
