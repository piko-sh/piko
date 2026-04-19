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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_dto"
)

func runAtomic(t *testing.T, config Config) {
	t.Run("a batch of upserts all apply", func(t *testing.T) {
		store := config.NewStore(t)
		err := store.AtomicUpdate(ctx(t), []registry_dto.AtomicAction{
			upsertAction(buildArtefact("batch-1", "source/b1")),
			upsertAction(buildArtefact("batch-2", "source/b2")),
			upsertAction(buildArtefact("batch-3", "source/b3")),
		})
		require.NoError(t, err)

		ids, err := store.ListAllArtefactIDs(ctx(t))
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"batch-1", "batch-2", "batch-3"}, ids)
	})

	t.Run("an unknown action type fails the batch and mutates nothing", func(t *testing.T) {
		store := config.NewStore(t)
		err := store.AtomicUpdate(ctx(t), []registry_dto.AtomicAction{
			upsertAction(buildArtefact("batch-ok", "source/ok")),
			{Type: registry_dto.ActionType("NONSENSE"), ArtefactID: "bad"},
		})
		require.Error(t, err, "an unknown action type must fail the whole batch")

		ids, err := store.ListAllArtefactIDs(ctx(t))
		require.NoError(t, err)
		assert.NotContains(t, ids, "batch-ok",
			"the valid action in a failed batch must not be visible; the batch is all-or-nothing")
	})

	t.Run("upsert then delete within one batch ends absent", func(t *testing.T) {
		store := config.NewStore(t)
		err := store.AtomicUpdate(ctx(t), []registry_dto.AtomicAction{
			upsertAction(buildArtefact("seq", "source/seq")),
			{Type: registry_dto.ActionTypeDeleteArtefact, ArtefactID: "seq"},
		})
		require.NoError(t, err)

		_, err = store.GetArtefact(ctx(t), "seq")
		requireNotFound(t, err)
	})
}

func upsertAction(artefact *registry_dto.ArtefactMeta) registry_dto.AtomicAction {
	return registry_dto.AtomicAction{
		Type:       registry_dto.ActionTypeUpsertArtefact,
		ArtefactID: artefact.ID,
		Artefact:   artefact,
	}
}
