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

package otter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

func newTestDAL(t *testing.T) *DAL {
	t.Helper()
	dal, err := NewOtterDAL(Config{Capacity: 1000})
	require.NoError(t, err)
	t.Cleanup(func() { _ = dal.Close() })
	d, ok := dal.(*DAL)
	require.True(t, ok)
	return d
}

func makeArtefact(id string, variantKeys ...string) *registry_dto.ArtefactMeta {
	now := time.Now()
	variants := make([]registry_dto.Variant, len(variantKeys))
	for i, key := range variantKeys {
		variants[i] = registry_dto.Variant{
			VariantID:  "v-" + key,
			StorageKey: key,
			MimeType:   "image/png",
			CreatedAt:  now,
			Status:     registry_dto.VariantStatusReady,
		}
	}
	return &registry_dto.ArtefactMeta{
		ID:             id,
		SourcePath:     "/test/" + id,
		CreatedAt:      now,
		UpdatedAt:      now,
		ActualVariants: variants,
	}
}

func TestRunAtomic_CommitPreservesMutations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(t *testing.T, dal *DAL)
		action func(ctx context.Context, store registry_domain.MetadataStore) error
		verify func(t *testing.T, dal *DAL)
	}{
		{
			name: "upsert artefact persists after commit",
			action: func(ctx context.Context, store registry_domain.MetadataStore) error {
				return store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
					{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: makeArtefact("art-1", "sk-1")},
				})
			},
			verify: func(t *testing.T, dal *DAL) {
				art, err := dal.GetArtefact(context.Background(), "art-1")
				require.NoError(t, err)
				assert.Equal(t, "art-1", art.ID)
			},
		},
		{
			name: "delete artefact persists after commit",
			setup: func(t *testing.T, dal *DAL) {
				ctx := context.Background()
				dal.mu.Lock()
				dal.upsertArtefactLocked(ctx, makeArtefact("art-del", "sk-del"))
				dal.mu.Unlock()
			},
			action: func(ctx context.Context, store registry_domain.MetadataStore) error {
				return store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
					{Type: registry_dto.ActionTypeDeleteArtefact, ArtefactID: "art-del"},
				})
			},
			verify: func(t *testing.T, dal *DAL) {
				_, err := dal.GetArtefact(context.Background(), "art-del")
				assert.ErrorIs(t, err, registry_domain.ErrArtefactNotFound)
			},
		},
		{
			name: "increment blob ref count persists after commit",
			action: func(ctx context.Context, store registry_domain.MetadataStore) error {
				count, err := store.IncrementBlobRefCount(ctx, registry_domain.BlobReference{
					StorageKey: "blob-1",
				})
				if err != nil {
					return err
				}
				if count != 1 {
					return errors.New("expected ref count 1")
				}
				return nil
			},
			verify: func(t *testing.T, dal *DAL) {
				count, err := dal.GetBlobRefCount(context.Background(), "blob-1")
				require.NoError(t, err)
				assert.Equal(t, 1, count)
			},
		},
		{
			name: "decrement blob ref count persists after commit",
			setup: func(t *testing.T, dal *DAL) {
				_, err := dal.IncrementBlobRefCount(context.Background(), registry_domain.BlobReference{
					StorageKey: "blob-dec",
				})
				require.NoError(t, err)
				_, err = dal.IncrementBlobRefCount(context.Background(), registry_domain.BlobReference{
					StorageKey: "blob-dec",
				})
				require.NoError(t, err)
			},
			action: func(ctx context.Context, store registry_domain.MetadataStore) error {
				count, _, err := store.DecrementBlobRefCount(ctx, "blob-dec")
				if err != nil {
					return err
				}
				if count != 1 {
					return errors.New("expected ref count 1")
				}
				return nil
			},
			verify: func(t *testing.T, dal *DAL) {
				count, err := dal.GetBlobRefCount(context.Background(), "blob-dec")
				require.NoError(t, err)
				assert.Equal(t, 1, count)
			},
		},
		{
			name: "add gc hints persists after commit",
			action: func(ctx context.Context, store registry_domain.MetadataStore) error {
				return store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
					{
						Type: registry_dto.ActionTypeAddGCHints,
						GCHints: []registry_dto.GCHint{
							{BackendID: "local", StorageKey: "gc-1"},
						},
					},
				})
			},
			verify: func(t *testing.T, dal *DAL) {
				hints, err := dal.PopGCHints(context.Background(), 10)
				require.NoError(t, err)
				require.Len(t, hints, 1)
				assert.Equal(t, "gc-1", hints[0].StorageKey)
			},
		},
		{
			name: "variant key index updated after commit",
			action: func(ctx context.Context, store registry_domain.MetadataStore) error {
				return store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
					{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: makeArtefact("art-vk", "vk-1")},
				})
			},
			verify: func(t *testing.T, dal *DAL) {
				art, err := dal.FindArtefactByVariantStorageKey(context.Background(), "vk-1")
				require.NoError(t, err)
				assert.Equal(t, "art-vk", art.ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dal := newTestDAL(t)
			ctx := context.Background()

			if tt.setup != nil {
				tt.setup(t, dal)
			}

			err := dal.RunAtomic(ctx, tt.action)
			require.NoError(t, err)
			tt.verify(t, dal)
		})
	}
}

func TestRunAtomic_RollbackRestoresState(t *testing.T) {
	t.Parallel()

	errRollback := errors.New("deliberate rollback")

	tests := []struct {
		name   string
		setup  func(t *testing.T, dal *DAL)
		action func(ctx context.Context, store registry_domain.MetadataStore) error
		verify func(t *testing.T, dal *DAL)
	}{
		{
			name: "upsert artefact rolled back",
			action: func(ctx context.Context, store registry_domain.MetadataStore) error {
				_ = store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
					{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: makeArtefact("art-rb", "sk-rb")},
				})
				return errRollback
			},
			verify: func(t *testing.T, dal *DAL) {
				_, err := dal.GetArtefact(context.Background(), "art-rb")
				assert.ErrorIs(t, err, registry_domain.ErrArtefactNotFound)
			},
		},
		{
			name: "delete artefact rolled back",
			setup: func(t *testing.T, dal *DAL) {
				ctx := context.Background()
				dal.mu.Lock()
				dal.upsertArtefactLocked(ctx, makeArtefact("art-keep", "sk-keep"))
				dal.mu.Unlock()
			},
			action: func(ctx context.Context, store registry_domain.MetadataStore) error {
				_ = store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
					{Type: registry_dto.ActionTypeDeleteArtefact, ArtefactID: "art-keep"},
				})
				return errRollback
			},
			verify: func(t *testing.T, dal *DAL) {
				art, err := dal.GetArtefact(context.Background(), "art-keep")
				require.NoError(t, err)
				assert.Equal(t, "art-keep", art.ID)
			},
		},
		{
			name: "increment blob ref count rolled back",
			action: func(ctx context.Context, store registry_domain.MetadataStore) error {
				_, _ = store.IncrementBlobRefCount(ctx, registry_domain.BlobReference{
					StorageKey: "blob-rb",
				})
				return errRollback
			},
			verify: func(t *testing.T, dal *DAL) {
				count, err := dal.GetBlobRefCount(context.Background(), "blob-rb")
				require.NoError(t, err)
				assert.Equal(t, 0, count)
			},
		},
		{
			name: "decrement blob ref count rolled back",
			setup: func(t *testing.T, dal *DAL) {
				_, err := dal.IncrementBlobRefCount(context.Background(), registry_domain.BlobReference{
					StorageKey: "blob-dec-rb",
				})
				require.NoError(t, err)
				_, err = dal.IncrementBlobRefCount(context.Background(), registry_domain.BlobReference{
					StorageKey: "blob-dec-rb",
				})
				require.NoError(t, err)
			},
			action: func(ctx context.Context, store registry_domain.MetadataStore) error {
				_, _, _ = store.DecrementBlobRefCount(ctx, "blob-dec-rb")
				return errRollback
			},
			verify: func(t *testing.T, dal *DAL) {
				count, err := dal.GetBlobRefCount(context.Background(), "blob-dec-rb")
				require.NoError(t, err)
				assert.Equal(t, 2, count)
			},
		},
		{
			name: "decrement to zero rolled back restores blob",
			setup: func(t *testing.T, dal *DAL) {
				_, err := dal.IncrementBlobRefCount(context.Background(), registry_domain.BlobReference{
					StorageKey: "blob-zero-rb",
				})
				require.NoError(t, err)
			},
			action: func(ctx context.Context, store registry_domain.MetadataStore) error {
				_, shouldDelete, _ := store.DecrementBlobRefCount(ctx, "blob-zero-rb")
				if !shouldDelete {
					return errors.New("expected shouldDelete=true")
				}
				return errRollback
			},
			verify: func(t *testing.T, dal *DAL) {
				count, err := dal.GetBlobRefCount(context.Background(), "blob-zero-rb")
				require.NoError(t, err)
				assert.Equal(t, 1, count)
			},
		},
		{
			name: "gc hints rolled back",
			action: func(ctx context.Context, store registry_domain.MetadataStore) error {
				_ = store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
					{
						Type: registry_dto.ActionTypeAddGCHints,
						GCHints: []registry_dto.GCHint{
							{BackendID: "local", StorageKey: "gc-rb"},
						},
					},
				})
				return errRollback
			},
			verify: func(t *testing.T, dal *DAL) {
				hints, err := dal.PopGCHints(context.Background(), 10)
				require.NoError(t, err)
				assert.Empty(t, hints)
			},
		},
		{
			name: "variant key index rolled back after upsert",
			action: func(ctx context.Context, store registry_domain.MetadataStore) error {
				_ = store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
					{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: makeArtefact("art-vk-rb", "vk-rb")},
				})
				return errRollback
			},
			verify: func(t *testing.T, dal *DAL) {
				_, err := dal.FindArtefactByVariantStorageKey(context.Background(), "vk-rb")
				assert.ErrorIs(t, err, registry_domain.ErrArtefactNotFound)
			},
		},
		{
			name: "variant key index rolled back after delete",
			setup: func(t *testing.T, dal *DAL) {
				ctx := context.Background()
				dal.mu.Lock()
				dal.upsertArtefactLocked(ctx, makeArtefact("art-vk-del", "vk-del"))
				dal.mu.Unlock()
			},
			action: func(ctx context.Context, store registry_domain.MetadataStore) error {
				_ = store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
					{Type: registry_dto.ActionTypeDeleteArtefact, ArtefactID: "art-vk-del"},
				})
				return errRollback
			},
			verify: func(t *testing.T, dal *DAL) {
				art, err := dal.FindArtefactByVariantStorageKey(context.Background(), "vk-del")
				require.NoError(t, err)
				assert.Equal(t, "art-vk-del", art.ID)
			},
		},
		{
			name: "multiple mutations rolled back atomically",
			setup: func(t *testing.T, dal *DAL) {
				ctx := context.Background()
				dal.mu.Lock()
				dal.upsertArtefactLocked(ctx, makeArtefact("art-multi", "sk-multi"))
				dal.mu.Unlock()
				_, err := dal.IncrementBlobRefCount(ctx, registry_domain.BlobReference{
					StorageKey: "blob-multi",
				})
				require.NoError(t, err)
			},
			action: func(ctx context.Context, store registry_domain.MetadataStore) error {

				_ = store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
					{Type: registry_dto.ActionTypeDeleteArtefact, ArtefactID: "art-multi"},
				})

				_ = store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
					{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: makeArtefact("art-new", "sk-new")},
				})

				_, _ = store.IncrementBlobRefCount(ctx, registry_domain.BlobReference{
					StorageKey: "blob-new",
				})
				_, _, _ = store.DecrementBlobRefCount(ctx, "blob-multi")

				_ = store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
					{
						Type:    registry_dto.ActionTypeAddGCHints,
						GCHints: []registry_dto.GCHint{{BackendID: "local", StorageKey: "gc-multi"}},
					},
				})
				return errRollback
			},
			verify: func(t *testing.T, dal *DAL) {
				ctx := context.Background()

				art, err := dal.GetArtefact(ctx, "art-multi")
				require.NoError(t, err)
				assert.Equal(t, "art-multi", art.ID)

				_, err = dal.GetArtefact(ctx, "art-new")
				assert.ErrorIs(t, err, registry_domain.ErrArtefactNotFound)

				count, err := dal.GetBlobRefCount(ctx, "blob-multi")
				require.NoError(t, err)
				assert.Equal(t, 1, count)

				count, err = dal.GetBlobRefCount(ctx, "blob-new")
				require.NoError(t, err)
				assert.Equal(t, 0, count)

				hints, err := dal.PopGCHints(ctx, 10)
				require.NoError(t, err)
				assert.Empty(t, hints)

				_, err = dal.FindArtefactByVariantStorageKey(ctx, "sk-multi")
				require.NoError(t, err)
				_, err = dal.FindArtefactByVariantStorageKey(ctx, "sk-new")
				assert.ErrorIs(t, err, registry_domain.ErrArtefactNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dal := newTestDAL(t)
			ctx := context.Background()

			if tt.setup != nil {
				tt.setup(t, dal)
			}

			err := dal.RunAtomic(ctx, tt.action)
			assert.Error(t, err)
			tt.verify(t, dal)
		})
	}
}

func TestRunAtomic_ReadsReflectMutationsWithinTransaction(t *testing.T) {
	t.Parallel()

	dal := newTestDAL(t)
	ctx := context.Background()

	err := dal.RunAtomic(ctx, func(ctx context.Context, store registry_domain.MetadataStore) error {

		_ = store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
			{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: makeArtefact("art-read", "sk-read")},
		})

		art, err := store.GetArtefact(ctx, "art-read")
		require.NoError(t, err)
		assert.Equal(t, "art-read", art.ID)

		_, err = store.IncrementBlobRefCount(ctx, registry_domain.BlobReference{
			StorageKey: "blob-read",
		})
		require.NoError(t, err)

		count, err := store.GetBlobRefCount(ctx, "blob-read")
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		return nil
	})
	require.NoError(t, err)
}

func TestRunAtomic_NestedTransactionReturnsError(t *testing.T) {
	t.Parallel()

	dal := newTestDAL(t)
	ctx := context.Background()

	err := dal.RunAtomic(ctx, func(ctx context.Context, store registry_domain.MetadataStore) error {
		return store.RunAtomic(ctx, func(ctx context.Context, _ registry_domain.MetadataStore) error {
			return nil
		})
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nested transactions are not supported")
}

func TestRunAtomic_EmptyTransactionIsNoOp(t *testing.T) {
	t.Parallel()

	dal := newTestDAL(t)
	ctx := context.Background()

	dal.mu.Lock()
	dal.upsertArtefactLocked(ctx, makeArtefact("art-empty", "sk-empty"))
	dal.mu.Unlock()

	err := dal.RunAtomic(ctx, func(_ context.Context, _ registry_domain.MetadataStore) error {
		return nil
	})
	require.NoError(t, err)

	art, err := dal.GetArtefact(ctx, "art-empty")
	require.NoError(t, err)
	assert.Equal(t, "art-empty", art.ID)
}

func TestRunAtomic_TransactionCloseDoesNotCloseParent(t *testing.T) {
	t.Parallel()

	dal := newTestDAL(t)
	ctx := context.Background()

	err := dal.RunAtomic(ctx, func(_ context.Context, store registry_domain.MetadataStore) error {
		return store.Close()
	})
	require.NoError(t, err)

	_, err = dal.GetArtefact(ctx, "nonexistent")
	assert.ErrorIs(t, err, registry_domain.ErrArtefactNotFound)
}
