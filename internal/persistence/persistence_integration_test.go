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
	"os"
	"path/filepath"
	"testing"
	"time"

	"piko.sh/piko/internal/orchestrator/orchestrator_dal"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_dal"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/wal/wal_domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_PersistenceRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "piko-persistence-test-*")
	require.NoError(t, err, "failed to create temp directory")
	defer func() { _ = os.RemoveAll(tempDir) }()

	ctx := context.Background()
	now := time.Now().Truncate(time.Millisecond)

	testArtefact := &registry_dto.ArtefactMeta{
		ID:         "artefact-integration-test",
		SourcePath: "/images/test.jpg",
		Status:     registry_dto.VariantStatusReady,
		CreatedAt:  now,
		UpdatedAt:  now,
		ActualVariants: []registry_dto.Variant{
			{
				VariantID:  "variant-1",
				StorageKey: "storage/variant-1.webp",
				MimeType:   "image/webp",
				Status:     registry_dto.VariantStatusReady,
				SizeBytes:  12345,
				CreatedAt:  now,
			},
		},
	}

	testTask := &orchestrator_domain.Task{
		ID:         "task-integration-test",
		WorkflowID: "workflow-1",
		Executor:   "image.process",
		Status:     orchestrator_domain.StatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
		ExecuteAt:  now,
		Config: orchestrator_domain.TaskConfig{
			Priority: orchestrator_domain.PriorityNormal,
		},
	}

	func() {
		provider := NewProvider(Config{
			RegistryCapacity:     1000,
			OrchestratorCapacity: 1000,
			Persistence: &PersistenceProviderConfig{
				Enabled:           true,
				WALDir:            filepath.Join(tempDir, "wal"),
				SyncMode:          wal_domain.SyncModeEveryWrite,
				SnapshotThreshold: 100,
			},
		})

		require.NoError(t, provider.Connect(ctx), "Connect failed")

		registryFactory, err := provider.RegistryDALFactory()
		require.NoError(t, err, "RegistryDALFactory failed")

		registryDAL, err := registryFactory.NewRegistryDAL()
		require.NoError(t, err, "NewRegistryDAL failed")

		dal, ok := registryDAL.(registry_dal.RegistryDAL)
		require.True(t, ok)
		err = dal.AtomicUpdate(ctx, []registry_dto.AtomicAction{
			{
				Type:     registry_dto.ActionTypeUpsertArtefact,
				Artefact: testArtefact,
			},
		})
		require.NoError(t, err, "AtomicUpdate failed")

		orchestratorFactory, err := provider.OrchestratorDALFactory()
		require.NoError(t, err, "OrchestratorDALFactory failed")

		orchestratorDAL, err := orchestratorFactory.NewOrchestratorDAL()
		require.NoError(t, err, "NewOrchestratorDAL failed")

		orchDAL, ok := orchestratorDAL.(orchestrator_dal.OrchestratorDAL)
		require.True(t, ok)
		require.NoError(t, orchDAL.CreateTask(ctx, testTask), "CreateTask failed")

		require.NoError(t, provider.Close(ctx), "Close failed")
	}()

	func() {
		provider := NewProvider(Config{
			RegistryCapacity:     1000,
			OrchestratorCapacity: 1000,
			Persistence: &PersistenceProviderConfig{
				Enabled:           true,
				WALDir:            filepath.Join(tempDir, "wal"),
				SyncMode:          wal_domain.SyncModeEveryWrite,
				SnapshotThreshold: 100,
			},
		})

		require.NoError(t, provider.Connect(ctx), "Connect (recovery) failed")
		defer func() { _ = provider.Close(ctx) }()

		registryFactory, err := provider.RegistryDALFactory()
		require.NoError(t, err, "RegistryDALFactory (recovery) failed")

		registryDAL, err := registryFactory.NewRegistryDAL()
		require.NoError(t, err, "NewRegistryDAL (recovery) failed")

		dal, ok := registryDAL.(registry_dal.RegistryDAL)
		require.True(t, ok)
		recovered, err := dal.GetArtefact(ctx, testArtefact.ID)
		require.NoError(t, err, "GetArtefact failed")

		assert.Equal(t, testArtefact.ID, recovered.ID,
			"artefact ID mismatch: got %q, want %q", recovered.ID, testArtefact.ID)
		assert.Equal(t, testArtefact.SourcePath, recovered.SourcePath,
			"artefact SourcePath mismatch: got %q, want %q", recovered.SourcePath, testArtefact.SourcePath)
		assert.Len(t, recovered.ActualVariants, len(testArtefact.ActualVariants),
			"artefact ActualVariants length mismatch: got %d, want %d",
			len(recovered.ActualVariants), len(testArtefact.ActualVariants))

		foundByKey, err := dal.FindArtefactByVariantStorageKey(ctx, "storage/variant-1.webp")
		require.NoError(t, err, "FindArtefactByVariantStorageKey failed")
		assert.Equal(t, testArtefact.ID, foundByKey.ID,
			"artefact found by storage key mismatch: got %q, want %q", foundByKey.ID, testArtefact.ID)

		orchestratorFactory, err := provider.OrchestratorDALFactory()
		require.NoError(t, err, "OrchestratorDALFactory (recovery) failed")

		orchestratorDAL, err := orchestratorFactory.NewOrchestratorDAL()
		require.NoError(t, err, "NewOrchestratorDAL (recovery) failed")

		orchDAL, ok := orchestratorDAL.(orchestrator_dal.OrchestratorDAL)
		require.True(t, ok)
		tasks, err := orchDAL.FetchAndMarkDueTasks(ctx, orchestrator_domain.PriorityNormal, 10)
		require.NoError(t, err, "FetchAndMarkDueTasks failed")

		var foundTask *orchestrator_domain.Task
		for _, task := range tasks {
			if task.ID == testTask.ID {
				foundTask = task
				break
			}
		}

		require.NotNil(t, foundTask, "task not recovered from WAL")
		assert.Equal(t, testTask.WorkflowID, foundTask.WorkflowID,
			"task WorkflowID mismatch: got %q, want %q", foundTask.WorkflowID, testTask.WorkflowID)
		assert.Equal(t, testTask.Executor, foundTask.Executor,
			"task Executor mismatch: got %q, want %q", foundTask.Executor, testTask.Executor)
	}()
}

func TestIntegration_PersistenceDisabled(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	provider := NewProvider(Config{
		RegistryCapacity:     1000,
		OrchestratorCapacity: 1000,
	})

	require.NoError(t, provider.Connect(ctx), "Connect failed")
	defer func() { _ = provider.Close(ctx) }()

	registryFactory, err := provider.RegistryDALFactory()
	require.NoError(t, err, "RegistryDALFactory failed")

	registryDAL, err := registryFactory.NewRegistryDAL()
	require.NoError(t, err, "NewRegistryDAL failed")

	dal, ok := registryDAL.(registry_dal.RegistryDAL)
	require.True(t, ok)

	now := time.Now()
	testArtefact := &registry_dto.ArtefactMeta{
		ID:         "test-no-persistence",
		SourcePath: "/test.jpg",
		Status:     registry_dto.VariantStatusReady,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	err = dal.AtomicUpdate(ctx, []registry_dto.AtomicAction{
		{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: testArtefact},
	})
	require.NoError(t, err, "AtomicUpdate failed")

	recovered, err := dal.GetArtefact(ctx, testArtefact.ID)
	require.NoError(t, err, "GetArtefact failed")

	assert.Equal(t, testArtefact.ID, recovered.ID,
		"artefact ID mismatch: got %q, want %q", recovered.ID, testArtefact.ID)
}

func TestIntegration_SearchAfterRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "piko-search-recovery-test-*")
	require.NoError(t, err, "failed to create temp directory")
	defer func() { _ = os.RemoveAll(tempDir) }()

	ctx := context.Background()
	now := time.Now().Truncate(time.Millisecond)

	variantTags := registry_dto.Tags{}
	variantTags.SetByName("format", "webp")
	variantTags.SetByName("width", "800")

	testArtefact := &registry_dto.ArtefactMeta{
		ID:         "searchable-artefact",
		SourcePath: "/images/tagged.jpg",
		Status:     registry_dto.VariantStatusReady,
		CreatedAt:  now,
		UpdatedAt:  now,
		ActualVariants: []registry_dto.Variant{
			{
				VariantID:    "variant-tagged",
				StorageKey:   "storage/tagged.webp",
				MimeType:     "image/webp",
				Status:       registry_dto.VariantStatusReady,
				SizeBytes:    5000,
				CreatedAt:    now,
				MetadataTags: variantTags,
			},
		},
	}

	func() {
		provider := NewProvider(Config{
			RegistryCapacity:     1000,
			OrchestratorCapacity: 1000,
			Persistence: &PersistenceProviderConfig{
				Enabled:           true,
				WALDir:            filepath.Join(tempDir, "wal"),
				SyncMode:          wal_domain.SyncModeEveryWrite,
				SnapshotThreshold: 100,
			},
		})

		require.NoError(t, provider.Connect(ctx), "Connect failed")

		registryFactory, err := provider.RegistryDALFactory()
		require.NoError(t, err)

		registryDAL, _ := registryFactory.NewRegistryDAL()
		dal, ok := registryDAL.(registry_dal.RegistryDAL)
		require.True(t, ok)

		err = dal.AtomicUpdate(ctx, []registry_dto.AtomicAction{
			{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: testArtefact},
		})
		require.NoError(t, err, "AtomicUpdate failed")

		results, err := dal.SearchArtefacts(ctx, registry_domain.SearchQuery{
			SimpleTagQuery: map[string]string{"format": "webp"},
		})
		require.NoError(t, err, "SearchArtefacts before close failed")
		require.Len(t, results, 1, "expected 1 result before close, got %d", len(results))

		_ = provider.Close(ctx)
	}()

	func() {
		provider := NewProvider(Config{
			RegistryCapacity:     1000,
			OrchestratorCapacity: 1000,
			Persistence: &PersistenceProviderConfig{
				Enabled:           true,
				WALDir:            filepath.Join(tempDir, "wal"),
				SyncMode:          wal_domain.SyncModeEveryWrite,
				SnapshotThreshold: 100,
			},
		})

		require.NoError(t, provider.Connect(ctx), "Connect (recovery) failed")
		defer func() { _ = provider.Close(ctx) }()

		registryFactory, err := provider.RegistryDALFactory()
		require.NoError(t, err)

		registryDAL, _ := registryFactory.NewRegistryDAL()
		dal, ok := registryDAL.(registry_dal.RegistryDAL)
		require.True(t, ok)

		results, err := dal.SearchArtefacts(ctx, registry_domain.SearchQuery{
			SimpleTagQuery: map[string]string{"format": "webp"},
		})
		require.NoError(t, err, "SearchArtefacts after recovery failed")
		require.Len(t, results, 1, "expected 1 result after recovery, got %d", len(results))
		assert.Equal(t, testArtefact.ID, results[0].ID,
			"wrong artefact found: got %q, want %q", results[0].ID, testArtefact.ID)

		results, err = dal.SearchArtefacts(ctx, registry_domain.SearchQuery{
			SimpleTagQuery: map[string]string{"width": "800"},
		})
		require.NoError(t, err, "SearchArtefacts by width failed")
		require.Len(t, results, 1, "expected 1 result for width search, got %d", len(results))
	}()
}
