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

func runSearch(t *testing.T, config Config) {
	t.Run("finds artefacts by tag value", func(t *testing.T) {
		store := config.NewStore(t)
		upsert(t, store, taggedArtefact("tag-1", "source/t1", "tagName", "alpha"))
		upsert(t, store, taggedArtefact("tag-2", "source/t2", "tagName", "beta"))
		upsert(t, store, taggedArtefact("tag-3", "source/t3", "tagName", "alpha"))

		got, err := store.SearchArtefactsByTagValues(ctx(t), "tagName", []string{"alpha"})
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"tag-1", "tag-3"}, artefactIDs(got),
			"exactly the artefacts carrying the value must match")
	})

	t.Run("unknown tag value returns nothing, not an error", func(t *testing.T) {
		store := config.NewStore(t)
		upsert(t, store, taggedArtefact("tag-4", "source/t4", "tagName", "gamma"))

		got, err := store.SearchArtefactsByTagValues(ctx(t), "tagName", []string{"absent"})
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("a changed tag set does not leave the old tag searchable", func(t *testing.T) {
		store := config.NewStore(t)
		upsert(t, store, taggedArtefact("tag-5", "source/t5", "tagName", "before"))
		upsert(t, store, taggedArtefact("tag-5", "source/t5", "tagName", "after"))

		stale, err := store.SearchArtefactsByTagValues(ctx(t), "tagName", []string{"before"})
		require.NoError(t, err)
		assert.Empty(t, stale, "the previous tag value must not survive a re-upsert")

		fresh, err := store.SearchArtefactsByTagValues(ctx(t), "tagName", []string{"after"})
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"tag-5"}, artefactIDs(fresh))
	})
}

func taggedArtefact(id, storageKey, tagKey, tagValue string) *registry_dto.ArtefactMeta {
	artefact := buildArtefact(id, storageKey)
	artefact.ActualVariants[0].MetadataTags.SetByName(tagKey, tagValue)
	return artefact
}

func artefactIDs(artefacts []*registry_dto.ArtefactMeta) []string {
	ids := make([]string, len(artefacts))
	for i := range artefacts {
		ids[i] = artefacts[i].ID
	}
	return ids
}
