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

	"github.com/stretchr/testify/require"
)

func TestIntegration_PersistenceRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "piko-persistence-test-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
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

		if err := provider.Connect(ctx); err != nil {
			t.Fatalf("Connect failed: %v", err)
		}

		registryFactory, err := provider.RegistryDALFactory()
		if err != nil {
			t.Fatalf("RegistryDALFactory failed: %v", err)
		}

		registryDAL, err := registryFactory.NewRegistryDAL()
		if err != nil {
			t.Fatalf("NewRegistryDAL failed: %v", err)
		}

		dal, ok := registryDAL.(registry_dal.RegistryDAL)
		require.True(t, ok)
		err = dal.AtomicUpdate(ctx, []registry_dto.AtomicAction{
			{
				Type:     registry_dto.ActionTypeUpsertArtefact,
				Artefact: testArtefact,
			},
		})
		if err != nil {
			t.Fatalf("AtomicUpdate failed: %v", err)
		}

		orchestratorFactory, err := provider.OrchestratorDALFactory()
		if err != nil {
			t.Fatalf("OrchestratorDALFactory failed: %v", err)
		}

		orchestratorDAL, err := orchestratorFactory.NewOrchestratorDAL()
		if err != nil {
			t.Fatalf("NewOrchestratorDAL failed: %v", err)
		}

		orchDAL, ok := orchestratorDAL.(orchestrator_dal.OrchestratorDAL)
		require.True(t, ok)
		if err := orchDAL.CreateTask(ctx, testTask); err != nil {
			t.Fatalf("CreateTask failed: %v", err)
		}

		if err := provider.Close(ctx); err != nil {
			t.Fatalf("Close failed: %v", err)
		}
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

		if err := provider.Connect(ctx); err != nil {
			t.Fatalf("Connect (recovery) failed: %v", err)
		}
		defer func() { _ = provider.Close(ctx) }()

		registryFactory, err := provider.RegistryDALFactory()
		if err != nil {
			t.Fatalf("RegistryDALFactory (recovery) failed: %v", err)
		}

		registryDAL, err := registryFactory.NewRegistryDAL()
		if err != nil {
			t.Fatalf("NewRegistryDAL (recovery) failed: %v", err)
		}

		dal, ok := registryDAL.(registry_dal.RegistryDAL)
		require.True(t, ok)
		recovered, err := dal.GetArtefact(ctx, testArtefact.ID)
		if err != nil {
			t.Fatalf("GetArtefact failed: %v", err)
		}

		if recovered.ID != testArtefact.ID {
			t.Errorf("artefact ID mismatch: got %q, want %q", recovered.ID, testArtefact.ID)
		}
		if recovered.SourcePath != testArtefact.SourcePath {
			t.Errorf("artefact SourcePath mismatch: got %q, want %q", recovered.SourcePath, testArtefact.SourcePath)
		}
		if len(recovered.ActualVariants) != len(testArtefact.ActualVariants) {
			t.Errorf("artefact ActualVariants length mismatch: got %d, want %d",
				len(recovered.ActualVariants), len(testArtefact.ActualVariants))
		}

		foundByKey, err := dal.FindArtefactByVariantStorageKey(ctx, "storage/variant-1.webp")
		if err != nil {
			t.Fatalf("FindArtefactByVariantStorageKey failed: %v", err)
		}
		if foundByKey.ID != testArtefact.ID {
			t.Errorf("artefact found by storage key mismatch: got %q, want %q", foundByKey.ID, testArtefact.ID)
		}

		orchestratorFactory, err := provider.OrchestratorDALFactory()
		if err != nil {
			t.Fatalf("OrchestratorDALFactory (recovery) failed: %v", err)
		}

		orchestratorDAL, err := orchestratorFactory.NewOrchestratorDAL()
		if err != nil {
			t.Fatalf("NewOrchestratorDAL (recovery) failed: %v", err)
		}

		orchDAL, ok := orchestratorDAL.(orchestrator_dal.OrchestratorDAL)
		require.True(t, ok)
		tasks, err := orchDAL.FetchAndMarkDueTasks(ctx, orchestrator_domain.PriorityNormal, 10)
		if err != nil {
			t.Fatalf("FetchAndMarkDueTasks failed: %v", err)
		}

		var foundTask *orchestrator_domain.Task
		for _, task := range tasks {
			if task.ID == testTask.ID {
				foundTask = task
				break
			}
		}

		if foundTask == nil {
			t.Fatalf("task not recovered from WAL")
		}
		if foundTask.WorkflowID != testTask.WorkflowID {
			t.Errorf("task WorkflowID mismatch: got %q, want %q", foundTask.WorkflowID, testTask.WorkflowID)
		}
		if foundTask.Executor != testTask.Executor {
			t.Errorf("task Executor mismatch: got %q, want %q", foundTask.Executor, testTask.Executor)
		}
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

	if err := provider.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer func() { _ = provider.Close(ctx) }()

	registryFactory, err := provider.RegistryDALFactory()
	if err != nil {
		t.Fatalf("RegistryDALFactory failed: %v", err)
	}

	registryDAL, err := registryFactory.NewRegistryDAL()
	if err != nil {
		t.Fatalf("NewRegistryDAL failed: %v", err)
	}

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
	if err != nil {
		t.Fatalf("AtomicUpdate failed: %v", err)
	}

	recovered, err := dal.GetArtefact(ctx, testArtefact.ID)
	if err != nil {
		t.Fatalf("GetArtefact failed: %v", err)
	}

	if recovered.ID != testArtefact.ID {
		t.Errorf("artefact ID mismatch: got %q, want %q", recovered.ID, testArtefact.ID)
	}
}

func TestIntegration_SearchAfterRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "piko-search-recovery-test-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
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

		if err := provider.Connect(ctx); err != nil {
			t.Fatalf("Connect failed: %v", err)
		}

		registryFactory, err := provider.RegistryDALFactory()
		require.NoError(t, err)

		registryDAL, _ := registryFactory.NewRegistryDAL()
		dal, ok := registryDAL.(registry_dal.RegistryDAL)
		require.True(t, ok)

		err = dal.AtomicUpdate(ctx, []registry_dto.AtomicAction{
			{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: testArtefact},
		})
		if err != nil {
			t.Fatalf("AtomicUpdate failed: %v", err)
		}

		results, err := dal.SearchArtefacts(ctx, registry_domain.SearchQuery{
			SimpleTagQuery: map[string]string{"format": "webp"},
		})
		if err != nil {
			t.Fatalf("SearchArtefacts before close failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result before close, got %d", len(results))
		}

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

		if err := provider.Connect(ctx); err != nil {
			t.Fatalf("Connect (recovery) failed: %v", err)
		}
		defer func() { _ = provider.Close(ctx) }()

		registryFactory, err := provider.RegistryDALFactory()
		require.NoError(t, err)

		registryDAL, _ := registryFactory.NewRegistryDAL()
		dal, ok := registryDAL.(registry_dal.RegistryDAL)
		require.True(t, ok)

		results, err := dal.SearchArtefacts(ctx, registry_domain.SearchQuery{
			SimpleTagQuery: map[string]string{"format": "webp"},
		})
		if err != nil {
			t.Fatalf("SearchArtefacts after recovery failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result after recovery, got %d", len(results))
		}
		if results[0].ID != testArtefact.ID {
			t.Errorf("wrong artefact found: got %q, want %q", results[0].ID, testArtefact.ID)
		}

		results, err = dal.SearchArtefacts(ctx, registry_domain.SearchQuery{
			SimpleTagQuery: map[string]string{"width": "800"},
		})
		if err != nil {
			t.Fatalf("SearchArtefacts by width failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result for width search, got %d", len(results))
		}
	}()
}
