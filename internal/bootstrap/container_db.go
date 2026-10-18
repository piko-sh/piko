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

// This file contains the otter in-memory backend setup for the default persistence path
// (when no SQL database is registered via AddDatabase).

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/persistence"
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/internal/wal/wal_domain"
)

// getOtterProvider returns the default otter persistence provider, creating and
// connecting it on first call. The otter provider manages in-memory caches with WAL
// persistence for both registry and orchestrator data.
//
// Returns *persistence.Provider which is the connected otter provider.
// Returns error when the provider fails to connect.
func (c *Container) getOtterProvider() (*persistence.Provider, error) {
	c.dbProviderOnce.Do(func() {
		_, l := logger_domain.From(c.GetAppContext(), log)
		l.Internal("Initialising otter persistence with WAL")

		walDir := filepath.Join(
			deref(c.serverConfig.Paths.BaseDir, "."),
			config.PikoInternalPath,
			"wal", "persistence",
		)

		provider := persistence.NewProvider(persistence.Config{
			RegistryCapacity:     100_000,
			OrchestratorCapacity: 100_000,
			Persistence: &persistence.PersistenceProviderConfig{
				Enabled:           true,
				WALDir:            walDir,
				SyncMode:          wal_domain.SyncModeBatched,
				SnapshotThreshold: 10_000,
			},
			Clock: nil,
		})

		ctx := c.GetAppContext()

		if err := provider.Connect(ctx); err != nil {
			_ = provider.Close(ctx)
			c.dbProviderErr = err
			return
		}

		c.dbProvider = provider

		shutdown.Register(ctx, "OtterPersistence", func(ctx context.Context) error {
			return provider.Close(ctx)
		})

		l.Internal("Otter persistence initialised successfully")
	})

	if c.dbProviderErr != nil {
		return nil, c.dbProviderErr
	}

	return c.dbProvider, nil
}

// FlushPersistenceSnapshot flushes build blobs and snapshots the active otter persistence
// provider.
//
// Returns error when the snapshot flush fails.
func (c *Container) FlushPersistenceSnapshot(ctx context.Context) error {
	var errs []error
	for _, store := range c.deferredBlobStores {
		if err := store.Flush(ctx); err != nil {
			errs = append(errs, fmt.Errorf("syncing build blobs: %w", err))
		}
	}
	if c.dbProvider != nil {
		if err := c.dbProvider.Checkpoint(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// blobFlusher is a blob store that defers durability until Flush.
type blobFlusher interface {
	// Flush makes every blob written so far durable.
	//
	// Returns error when flushing fails.
	Flush(ctx context.Context) error
}

// createOtterRegistryDAL creates a registry DAL from the otter provider.
//
// Returns any which is the registry DAL instance.
// Returns error when the otter provider or DAL factory fails.
func (c *Container) createOtterRegistryDAL() (any, error) {
	provider, err := c.getOtterProvider()
	if err != nil {
		return nil, err
	}

	factory, factoryErr := provider.RegistryDALFactory()
	if factoryErr != nil {
		return nil, factoryErr
	}

	return factory.NewRegistryDAL()
}

// createOtterOrchestratorDAL creates an orchestrator DAL from the otter provider.
//
// Returns any which is the orchestrator DAL instance.
// Returns error when the otter provider or DAL factory fails.
func (c *Container) createOtterOrchestratorDAL() (any, error) {
	provider, err := c.getOtterProvider()
	if err != nil {
		return nil, err
	}

	factory, factoryErr := provider.OrchestratorDALFactory()
	if factoryErr != nil {
		return nil, factoryErr
	}

	return factory.NewOrchestratorDAL()
}
