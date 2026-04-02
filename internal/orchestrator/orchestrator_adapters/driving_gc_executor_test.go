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

package orchestrator_adapters

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

func TestNewGCExecutor(t *testing.T) {
	t.Parallel()

	executor := NewGCExecutor(nil, nil)
	require.NotNil(t, executor)

	gc, ok := executor.(*gcExecutor)
	require.True(t, ok)
	assert.Nil(t, gc.registryService)
	assert.Nil(t, gc.orchestratorService)
}

func TestExecutorNameBlobGC(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "blob.gc", ExecutorNameBlobGC)
}

func TestGCExecutor_Execute_UnknownMode(t *testing.T) {
	t.Parallel()

	registry := &registry_domain.MockRegistryService{}
	orcService := &orchestrator_domain.MockOrchestratorService{}
	executor := NewGCExecutor(registry, orcService)

	_, err := executor.Execute(context.Background(), map[string]any{
		"mode": "invalid",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown GC mode")
}

func TestGCExecutor_Hints_DeletesBlobs(t *testing.T) {
	t.Parallel()

	blobStore := &registry_domain.MockBlobStore{}
	var deletedKeys []string
	blobStore.DeleteFunc = func(_ context.Context, key string) error {
		deletedKeys = append(deletedKeys, key)
		return nil
	}

	registry := &registry_domain.MockRegistryService{
		PopGCHintsFunc: func(_ context.Context, limit int) ([]registry_dto.GCHint, error) {
			return []registry_dto.GCHint{
				{BackendID: "local_disk_cache", StorageKey: "source/abc.pkc"},
				{BackendID: "local_disk_cache", StorageKey: "generated/def.js"},
			}, nil
		},
		GetBlobStoreFunc: func(backendID string) (registry_domain.BlobStore, error) {
			return blobStore, nil
		},
	}

	orcService := &orchestrator_domain.MockOrchestratorService{}
	executor := NewGCExecutor(registry, orcService)

	result, err := executor.Execute(context.Background(), map[string]any{
		"mode":               "hints",
		"batch_size":         100,
		"reschedule_seconds": 30,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, result["deleted_count"])
	assert.Equal(t, "hints", result["mode"])
	assert.Equal(t, []string{"source/abc.pkc", "generated/def.js"}, deletedKeys)
}

func TestGCExecutor_Hints_EmptyBatch(t *testing.T) {
	t.Parallel()

	registry := &registry_domain.MockRegistryService{
		PopGCHintsFunc: func(_ context.Context, _ int) ([]registry_dto.GCHint, error) {
			return nil, nil
		},
	}

	orcService := &orchestrator_domain.MockOrchestratorService{}
	executor := NewGCExecutor(registry, orcService)

	result, err := executor.Execute(context.Background(), map[string]any{
		"mode":               "hints",
		"reschedule_seconds": 60,
	})

	require.NoError(t, err)
	assert.Equal(t, 0, result["deleted_count"])
	assert.Equal(t, int64(1), orcService.ScheduleCallCount)
}

func TestGCExecutor_Hints_FullBatch_ReschedulesImmediately(t *testing.T) {
	t.Parallel()

	batchSize := 3
	hints := make([]registry_dto.GCHint, batchSize)
	for i := range hints {
		hints[i] = registry_dto.GCHint{BackendID: "local_disk_cache", StorageKey: "blob" + string(rune('0'+i))}
	}

	blobStore := &registry_domain.MockBlobStore{}
	registry := &registry_domain.MockRegistryService{
		PopGCHintsFunc: func(_ context.Context, _ int) ([]registry_dto.GCHint, error) {
			return hints, nil
		},
		GetBlobStoreFunc: func(_ string) (registry_domain.BlobStore, error) {
			return blobStore, nil
		},
	}

	var scheduledAt time.Time
	orcService := &orchestrator_domain.MockOrchestratorService{
		ScheduleFunc: func(_ context.Context, _ *orchestrator_domain.Task, executeAt time.Time) (*orchestrator_domain.WorkflowReceipt, error) {
			scheduledAt = executeAt
			return nil, nil
		},
	}

	executor := NewGCExecutor(registry, orcService)

	_, err := executor.Execute(context.Background(), map[string]any{
		"mode":               "hints",
		"batch_size":         batchSize,
		"reschedule_seconds": 300,
	})

	require.NoError(t, err)

	assert.WithinDuration(t, time.Now(), scheduledAt, 1*time.Second)
}

func TestGCExecutor_Hints_DeleteError_ContinuesProcessing(t *testing.T) {
	t.Parallel()

	deleteCount := 0
	blobStore := &registry_domain.MockBlobStore{
		DeleteFunc: func(_ context.Context, key string) error {
			if key == "bad_key" {
				return errors.New("disk error")
			}
			deleteCount++
			return nil
		},
	}

	registry := &registry_domain.MockRegistryService{
		PopGCHintsFunc: func(_ context.Context, _ int) ([]registry_dto.GCHint, error) {
			return []registry_dto.GCHint{
				{BackendID: "local_disk_cache", StorageKey: "good_key"},
				{BackendID: "local_disk_cache", StorageKey: "bad_key"},
				{BackendID: "local_disk_cache", StorageKey: "another_good_key"},
			}, nil
		},
		GetBlobStoreFunc: func(_ string) (registry_domain.BlobStore, error) {
			return blobStore, nil
		},
	}

	orcService := &orchestrator_domain.MockOrchestratorService{}
	executor := NewGCExecutor(registry, orcService)

	result, err := executor.Execute(context.Background(), map[string]any{
		"mode": "hints",
	})

	require.NoError(t, err)
	assert.Equal(t, 2, result["deleted_count"])
}

func TestGCExecutor_Hints_BlobStoreNotFound_Skips(t *testing.T) {
	t.Parallel()

	registry := &registry_domain.MockRegistryService{
		PopGCHintsFunc: func(_ context.Context, _ int) ([]registry_dto.GCHint, error) {
			return []registry_dto.GCHint{
				{BackendID: "nonexistent", StorageKey: "blob.pkc"},
			}, nil
		},
		GetBlobStoreFunc: func(_ string) (registry_domain.BlobStore, error) {
			return nil, errors.New("backend not found")
		},
	}

	orcService := &orchestrator_domain.MockOrchestratorService{}
	executor := NewGCExecutor(registry, orcService)

	result, err := executor.Execute(context.Background(), map[string]any{
		"mode": "hints",
	})

	require.NoError(t, err)
	assert.Equal(t, 0, result["deleted_count"])
}

func TestGCExecutor_Orphans_DeletesUnreferencedBlobs(t *testing.T) {
	t.Parallel()

	blobStore := &registry_domain.MockBlobStore{
		ListKeysFunc: func(_ context.Context) ([]string, error) {
			return []string{
				"source/referenced.pkc",
				"source/orphan1.pkc",
				"generated/orphan2.js",
			}, nil
		},
	}
	var deletedKeys []string
	blobStore.DeleteFunc = func(_ context.Context, key string) error {
		deletedKeys = append(deletedKeys, key)
		return nil
	}

	registry := &registry_domain.MockRegistryService{
		ListAllArtefactIDsFunc: func(_ context.Context) ([]string, error) {
			return []string{"art1"}, nil
		},
		GetMultipleArtefactsFunc: func(_ context.Context, _ []string) ([]*registry_dto.ArtefactMeta, error) {
			return []*registry_dto.ArtefactMeta{
				{
					ID: "art1",
					ActualVariants: []registry_dto.Variant{
						{StorageKey: "source/referenced.pkc"},
					},
				},
			}, nil
		},
		ListBlobStoreIDsFunc: func() []string {
			return []string{"local_disk_cache"}
		},
		GetBlobStoreFunc: func(_ string) (registry_domain.BlobStore, error) {
			return blobStore, nil
		},
	}

	orcService := &orchestrator_domain.MockOrchestratorService{}
	executor := NewGCExecutor(registry, orcService)

	result, err := executor.Execute(context.Background(), map[string]any{
		"mode":               "orphans",
		"reschedule_seconds": 3600,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, result["deleted_count"])
	assert.Contains(t, deletedKeys, "source/orphan1.pkc")
	assert.Contains(t, deletedKeys, "generated/orphan2.js")
	assert.NotContains(t, deletedKeys, "source/referenced.pkc")
}

func TestGCExecutor_Orphans_SkipsTmpPrefix(t *testing.T) {
	t.Parallel()

	blobStore := &registry_domain.MockBlobStore{
		ListKeysFunc: func(_ context.Context) ([]string, error) {
			return []string{
				"tmp/upload-in-progress.pkc",
				"source/orphan.pkc",
			}, nil
		},
	}
	var deletedKeys []string
	blobStore.DeleteFunc = func(_ context.Context, key string) error {
		deletedKeys = append(deletedKeys, key)
		return nil
	}

	registry := &registry_domain.MockRegistryService{
		ListAllArtefactIDsFunc: func(_ context.Context) ([]string, error) {
			return nil, nil
		},
		GetMultipleArtefactsFunc: func(_ context.Context, _ []string) ([]*registry_dto.ArtefactMeta, error) {
			return nil, nil
		},
		ListBlobStoreIDsFunc: func() []string {
			return []string{"local_disk_cache"}
		},
		GetBlobStoreFunc: func(_ string) (registry_domain.BlobStore, error) {
			return blobStore, nil
		},
	}

	orcService := &orchestrator_domain.MockOrchestratorService{}
	executor := NewGCExecutor(registry, orcService)

	result, err := executor.Execute(context.Background(), map[string]any{
		"mode": "orphans",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result["deleted_count"])
	assert.Equal(t, []string{"source/orphan.pkc"}, deletedKeys)
}

func TestGCExecutor_Orphans_ListKeysUnsupported_SkipsBackend(t *testing.T) {
	t.Parallel()

	blobStore := &registry_domain.MockBlobStore{
		ListKeysFunc: func(_ context.Context) ([]string, error) {
			return nil, errors.New("listing not supported")
		},
	}

	registry := &registry_domain.MockRegistryService{
		ListAllArtefactIDsFunc: func(_ context.Context) ([]string, error) {
			return nil, nil
		},
		GetMultipleArtefactsFunc: func(_ context.Context, _ []string) ([]*registry_dto.ArtefactMeta, error) {
			return nil, nil
		},
		ListBlobStoreIDsFunc: func() []string {
			return []string{"local_disk_cache"}
		},
		GetBlobStoreFunc: func(_ string) (registry_domain.BlobStore, error) {
			return blobStore, nil
		},
	}

	orcService := &orchestrator_domain.MockOrchestratorService{}
	executor := NewGCExecutor(registry, orcService)

	result, err := executor.Execute(context.Background(), map[string]any{
		"mode": "orphans",
	})

	require.NoError(t, err)
	assert.Equal(t, 0, result["deleted_count"])
}

func TestGCExecutor_Orphans_ChunkKeysAlsoReferenced(t *testing.T) {
	t.Parallel()

	blobStore := &registry_domain.MockBlobStore{
		ListKeysFunc: func(_ context.Context) ([]string, error) {
			return []string{
				"source/main.pkc",
				"generated/chunk1.br",
				"generated/chunk2.gz",
				"generated/orphan.br",
			}, nil
		},
	}
	var deletedKeys []string
	blobStore.DeleteFunc = func(_ context.Context, key string) error {
		deletedKeys = append(deletedKeys, key)
		return nil
	}

	registry := &registry_domain.MockRegistryService{
		ListAllArtefactIDsFunc: func(_ context.Context) ([]string, error) {
			return []string{"art1"}, nil
		},
		GetMultipleArtefactsFunc: func(_ context.Context, _ []string) ([]*registry_dto.ArtefactMeta, error) {
			return []*registry_dto.ArtefactMeta{
				{
					ID: "art1",
					ActualVariants: []registry_dto.Variant{
						{
							StorageKey: "source/main.pkc",
							Chunks: []registry_dto.VariantChunk{
								{StorageKey: "generated/chunk1.br"},
								{StorageKey: "generated/chunk2.gz"},
							},
						},
					},
				},
			}, nil
		},
		ListBlobStoreIDsFunc: func() []string {
			return []string{"local_disk_cache"}
		},
		GetBlobStoreFunc: func(_ string) (registry_domain.BlobStore, error) {
			return blobStore, nil
		},
	}

	orcService := &orchestrator_domain.MockOrchestratorService{}
	executor := NewGCExecutor(registry, orcService)

	result, err := executor.Execute(context.Background(), map[string]any{
		"mode": "orphans",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result["deleted_count"])
	assert.Equal(t, []string{"generated/orphan.br"}, deletedKeys)
}

func TestGCPayloadString(t *testing.T) {
	t.Parallel()

	t.Run("returns value when present", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "hints", gcPayloadString(map[string]any{"mode": "hints"}, "mode", "default"))
	})

	t.Run("returns fallback when missing", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "default", gcPayloadString(map[string]any{}, "mode", "default"))
	})

	t.Run("returns fallback when wrong type", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "default", gcPayloadString(map[string]any{"mode": 123}, "mode", "default"))
	})
}

func TestGCPayloadInt(t *testing.T) {
	t.Parallel()

	t.Run("returns int value", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 50, gcPayloadInt(map[string]any{"batch_size": 50}, "batch_size", 100))
	})

	t.Run("returns float64 value as int", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 50, gcPayloadInt(map[string]any{"batch_size": float64(50)}, "batch_size", 100))
	})

	t.Run("returns fallback when missing", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 100, gcPayloadInt(map[string]any{}, "batch_size", 100))
	})

	t.Run("returns fallback when wrong type", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 100, gcPayloadInt(map[string]any{"batch_size": "fifty"}, "batch_size", 100))
	})
}

func TestGCExecutor_Reschedule_SetsDeduplicationKey(t *testing.T) {
	t.Parallel()

	registry := &registry_domain.MockRegistryService{
		PopGCHintsFunc: func(_ context.Context, _ int) ([]registry_dto.GCHint, error) {
			return nil, nil
		},
	}

	var scheduledTask *orchestrator_domain.Task
	orcService := &orchestrator_domain.MockOrchestratorService{
		ScheduleFunc: func(_ context.Context, task *orchestrator_domain.Task, _ time.Time) (*orchestrator_domain.WorkflowReceipt, error) {
			scheduledTask = task
			return nil, nil
		},
	}

	executor := NewGCExecutor(registry, orcService)

	_, err := executor.Execute(context.Background(), map[string]any{
		"mode": "hints",
	})

	require.NoError(t, err)
	require.NotNil(t, scheduledTask)
	assert.Equal(t, "blob.gc.hints", scheduledTask.DeduplicationKey)
	assert.Equal(t, orchestrator_domain.PriorityLow, scheduledTask.Config.Priority)
	assert.Equal(t, ExecutorNameBlobGC, scheduledTask.Executor)
}
