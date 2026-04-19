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

//go:build integration

package registry_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

var (
	errRollback = errors.New("conformance: forced rollback")
)

func runTransactions(t *testing.T, config Config) {
	t.Run("a nil-returning transaction commits", func(t *testing.T) {
		store := config.NewStore(t)
		err := store.RunAtomic(ctx(t), func(ctx context.Context, tx registry_domain.MetadataStore) error {
			return tx.AtomicUpdate(ctx, []registry_dto.AtomicAction{upsertAction(buildArtefact("tx-1", "source/tx1"))})
		})
		require.NoError(t, err)

		got, err := store.GetArtefact(ctx(t), "tx-1")
		require.NoError(t, err)
		assert.Equal(t, "tx-1", got.ID)
	})

	t.Run("reads inside the transaction see its own writes", func(t *testing.T) {
		store := config.NewStore(t)
		err := store.RunAtomic(ctx(t), func(ctx context.Context, tx registry_domain.MetadataStore) error {
			if err := tx.AtomicUpdate(ctx, []registry_dto.AtomicAction{upsertAction(buildArtefact("tx-2", "source/tx2"))}); err != nil {
				return err
			}
			got, err := tx.GetArtefact(ctx, "tx-2")
			if err != nil {
				return err
			}
			assert.Equal(t, "tx-2", got.ID)
			return nil
		})
		require.NoError(t, err)
	})

	if config.SupportsRollback {
		t.Run("an error rolls back every mutation", func(t *testing.T) {
			store := config.NewStore(t)
			upsert(t, store, buildArtefact("tx-keep", "source/keep"))

			err := store.RunAtomic(ctx(t), func(ctx context.Context, tx registry_domain.MetadataStore) error {
				if err := tx.AtomicUpdate(ctx, []registry_dto.AtomicAction{upsertAction(buildArtefact("tx-gone", "source/gone"))}); err != nil {
					return err
				}
				if _, err := tx.IncrementBlobRefCount(ctx, registry_domain.BlobReference{
					StorageKey: "blob/tx", StorageBackendID: "local_disk_cache", MimeType: "text/plain", SizeBytes: 1,
				}); err != nil {
					return err
				}
				return errRollback
			})
			require.ErrorIs(t, err, errRollback)

			_, err = store.GetArtefact(ctx(t), "tx-gone")
			requireNotFound(t, err)

			count, err := store.GetBlobRefCount(ctx(t), "blob/tx")
			require.NoError(t, err)
			assert.Equal(t, 0, count, "the ref-count increment must roll back with the transaction")

			got, err := store.GetArtefact(ctx(t), "tx-keep")
			require.NoError(t, err)
			assert.Equal(t, "tx-keep", got.ID, "a pre-existing artefact must survive the rollback")
		})
	} else {
		t.Run("rollback unsupported", func(t *testing.T) {
			t.Skip("backend does not advertise SupportsRollback")
		})
	}

	if config.SupportsNestedTransactionRejection {
		t.Run("a nested transaction is rejected", func(t *testing.T) {
			store := config.NewStore(t)
			err := store.RunAtomic(ctx(t), func(outerCtx context.Context, tx registry_domain.MetadataStore) error {
				return tx.RunAtomic(outerCtx, func(ctx context.Context, _ registry_domain.MetadataStore) error {
					return nil
				})
			})
			require.Error(t, err, "a RunAtomic inside a RunAtomic must be rejected, not silently flattened")
		})
	}

	t.Run("locked read agrees with a plain read", func(t *testing.T) {
		store := config.NewStore(t)
		upsert(t, store, buildArtefact("tx-lock", "source/lock"))

		err := store.RunAtomic(ctx(t), func(ctx context.Context, tx registry_domain.MetadataStore) error {
			locked, err := registry_domain.ReadArtefactForLockedUpdate(ctx, tx, "tx-lock")
			if err != nil {
				return err
			}
			assert.Equal(t, "tx-lock", locked.ID, "the locked read must return the same artefact as a plain get")
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("locked read of a missing artefact is ErrArtefactNotFound", func(t *testing.T) {
		store := config.NewStore(t)
		err := store.RunAtomic(ctx(t), func(ctx context.Context, tx registry_domain.MetadataStore) error {
			_, readErr := registry_domain.ReadArtefactForLockedUpdate(ctx, tx, "tx-absent")
			requireNotFound(t, readErr)
			return nil
		})
		require.NoError(t, err)
	})
}
