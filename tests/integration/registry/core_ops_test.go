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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

func runCoreOps(t *testing.T, config Config) {
	t.Run("get missing returns ErrArtefactNotFound", func(t *testing.T) {
		store := config.NewStore(t)
		_, err := store.GetArtefact(ctx(t), "does-not-exist")
		require.Error(t, err, "a missing artefact must error")
		require.ErrorIs(t, err, registry_domain.ErrArtefactNotFound,
			"the error must satisfy errors.Is(ErrArtefactNotFound); callers branch on it")
	})

	t.Run("get after upsert returns the artefact", func(t *testing.T) {
		store := config.NewStore(t)
		upsert(t, store, buildArtefact("art-1", "source/1"))

		got, err := store.GetArtefact(ctx(t), "art-1")
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "art-1", got.ID)
		require.Len(t, got.ActualVariants, 1)
		assert.Equal(t, "source", got.ActualVariants[0].VariantID)
	})

	t.Run("get returns an independent copy", func(t *testing.T) {
		store := config.NewStore(t)
		upsert(t, store, buildArtefact("art-2", "source/2"))

		first, err := store.GetArtefact(ctx(t), "art-2")
		require.NoError(t, err)
		first.ActualVariants[0].StorageKey = "mutated"
		first.SourcePath = "mutated"

		second, err := store.GetArtefact(ctx(t), "art-2")
		require.NoError(t, err)
		assert.Equal(t, "source/2", second.ActualVariants[0].StorageKey,
			"mutating a returned artefact must not mutate the stored one")
		assert.NotEqual(t, "mutated", second.SourcePath)
	})

	t.Run("get multiple returns only present, no nils", func(t *testing.T) {
		store := config.NewStore(t)
		upsert(t, store, buildArtefact("art-a", "source/a"))
		upsert(t, store, buildArtefact("art-b", "source/b"))

		got, err := store.GetMultipleArtefacts(ctx(t), []string{"art-a", "missing", "art-b"})
		require.NoError(t, err, "a partial hit must not error")
		require.Len(t, got, 2, "only the present artefacts are returned")
		for _, artefact := range got {
			require.NotNil(t, artefact, "no nil entries in the result")
		}
	})

	t.Run("get multiple with empty input returns empty", func(t *testing.T) {
		store := config.NewStore(t)
		got, err := store.GetMultipleArtefacts(ctx(t), nil)
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("list all reflects upserts", func(t *testing.T) {
		store := config.NewStore(t)
		empty, err := store.ListAllArtefactIDs(ctx(t))
		require.NoError(t, err)
		assert.Empty(t, empty, "a fresh store lists nothing")

		upsert(t, store, buildArtefact("art-x", "source/x"))
		upsert(t, store, buildArtefact("art-y", "source/y"))

		ids, err := store.ListAllArtefactIDs(ctx(t))
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"art-x", "art-y"}, ids, "every ID appears exactly once")
	})

	t.Run("upsert overwrites", func(t *testing.T) {
		store := config.NewStore(t)
		upsert(t, store, buildArtefact("art-o", "source/old"))

		updated := buildArtefact("art-o", "source/new")
		updated.SourcePath = "src/updated.txt"
		upsert(t, store, updated)

		got, err := store.GetArtefact(ctx(t), "art-o")
		require.NoError(t, err)
		assert.Equal(t, "src/updated.txt", got.SourcePath)
		assert.Equal(t, "source/new", got.ActualVariants[0].StorageKey)

		ids, err := store.ListAllArtefactIDs(ctx(t))
		require.NoError(t, err)
		assert.Len(t, ids, 1, "an overwrite must not create a second row")
	})
}

func runReverseLookup(t *testing.T, config Config) {
	t.Run("finds the owning artefact for a variant key", func(t *testing.T) {
		store := config.NewStore(t)
		upsert(t, store, buildArtefact("art-r", "source/r"))

		got, err := store.FindArtefactByVariantStorageKey(ctx(t), "source/r")
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "art-r", got.ID)
	})

	t.Run("unknown key returns ErrArtefactNotFound", func(t *testing.T) {
		store := config.NewStore(t)
		_, err := store.FindArtefactByVariantStorageKey(ctx(t), "source/never")
		require.Error(t, err)
		require.ErrorIs(t, err, registry_domain.ErrArtefactNotFound)
	})

	t.Run("a replaced key stops resolving", func(t *testing.T) {
		store := config.NewStore(t)
		upsert(t, store, buildArtefact("art-r2", "source/first"))
		upsert(t, store, buildArtefact("art-r2", "source/second"))

		_, err := store.FindArtefactByVariantStorageKey(ctx(t), "source/first")
		require.ErrorIs(t, err, registry_domain.ErrArtefactNotFound,
			"the old storage key must not resolve after it is replaced")

		got, err := store.FindArtefactByVariantStorageKey(ctx(t), "source/second")
		require.NoError(t, err)
		assert.Equal(t, "art-r2", got.ID)
	})

	t.Run("a deleted artefact's keys stop resolving", func(t *testing.T) {
		store := config.NewStore(t)
		upsert(t, store, buildArtefact("art-r3", "source/r3"))

		err := store.AtomicUpdate(ctx(t), []registry_dto.AtomicAction{{
			Type:       registry_dto.ActionTypeDeleteArtefact,
			ArtefactID: "art-r3",
		}})
		require.NoError(t, err)

		_, err = store.FindArtefactByVariantStorageKey(ctx(t), "source/r3")
		require.ErrorIs(t, err, registry_domain.ErrArtefactNotFound)
	})
}

func requireNotFound(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	require.True(t, errors.Is(err, registry_domain.ErrArtefactNotFound),
		"expected ErrArtefactNotFound, got %v", err)
}
