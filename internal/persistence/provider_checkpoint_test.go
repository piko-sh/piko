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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_dal"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/wal/wal_domain"
)

func persistenceProvider(t *testing.T, walDir string) *Provider {
	t.Helper()
	return NewProvider(Config{
		RegistryCapacity:     1000,
		OrchestratorCapacity: 1000,
		Persistence: &PersistenceProviderConfig{
			Enabled:           true,
			WALDir:            walDir,
			SyncMode:          wal_domain.SyncModeEveryWrite,
			SnapshotThreshold: 100,
		},
	})
}

func writeCheckpointArtefact(t *testing.T, provider *Provider, ctx context.Context, id string) {
	t.Helper()
	registryFactory, err := provider.RegistryDALFactory()
	require.NoError(t, err, "the registry DAL factory must be available")
	registryDAL, err := registryFactory.NewRegistryDAL()
	require.NoError(t, err, "a registry DAL must be constructable")
	dal, ok := registryDAL.(registry_dal.RegistryDAL)
	require.True(t, ok, "the registry DAL must satisfy registry_dal.RegistryDAL")
	require.NoError(t, dal.AtomicUpdate(ctx, []registry_dto.AtomicAction{{
		Type:     registry_dto.ActionTypeUpsertArtefact,
		Artefact: &registry_dto.ArtefactMeta{ID: id, SourcePath: id + ".png"},
	}}), "writing the artefact must succeed")
}

func TestProviderCheckpointPersistsAndRecovers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	walDir := filepath.Join(t.TempDir(), "wal")

	provider := persistenceProvider(t, walDir)
	require.NoError(t, provider.Connect(ctx), "the provider must connect")
	writeCheckpointArtefact(t, provider, ctx, "art-checkpoint")
	require.NoError(t, provider.Checkpoint(ctx), "the checkpoint must flush every persistent cache")
	require.NoError(t, provider.Close(ctx), "the provider must close")

	recovered := persistenceProvider(t, walDir)
	require.NoError(t, recovered.Connect(ctx), "the recovery provider must connect")
	defer func() { _ = recovered.Close(ctx) }()

	registryFactory, err := recovered.RegistryDALFactory()
	require.NoError(t, err, "the recovery registry DAL factory must be available")
	registryDAL, err := registryFactory.NewRegistryDAL()
	require.NoError(t, err, "a recovery registry DAL must be constructable")
	dal, ok := registryDAL.(registry_dal.RegistryDAL)
	require.True(t, ok, "the recovery registry DAL must satisfy registry_dal.RegistryDAL")

	artefact, err := dal.GetArtefact(ctx, "art-checkpoint")
	require.NoError(t, err, "the checkpointed artefact must be recovered")
	assert.Equal(t, "art-checkpoint", artefact.ID, "the recovered artefact must be the checkpointed one")
}

func TestProviderCheckpointBeforeConnectIsNoOp(t *testing.T) {
	t.Parallel()

	provider := persistenceProvider(t, filepath.Join(t.TempDir(), "wal"))
	require.NoError(t, provider.Checkpoint(context.Background()),
		"a checkpoint before connect must be a harmless no-op")
}
