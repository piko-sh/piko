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

func runVariants(t *testing.T, config Config) {
	t.Run("chunks round-trip in order", func(t *testing.T) {
		store := config.NewStore(t)
		artefact := buildArtefact("art-v1", "source/v1")
		chunked := buildVariant("video", "video/v1")
		chunked.Chunks = []registry_dto.VariantChunk{
			buildChunk("c0", "chunk/0", 0),
			buildChunk("c1", "chunk/1", 1),
			buildChunk("c2", "chunk/2", 2),
		}
		artefact.ActualVariants = append(artefact.ActualVariants, chunked)
		upsert(t, store, artefact)

		got, err := store.GetArtefact(ctx(t), "art-v1")
		require.NoError(t, err)
		video := findVariant(got, "video")
		require.NotNil(t, video, "the chunked variant must survive the round-trip")
		require.Len(t, video.Chunks, 3)
		for i := range video.Chunks {
			assert.Equal(t, i, video.Chunks[i].SequenceNumber, "chunk order must be preserved")
		}
	})

	t.Run("shrinking the variant set removes dropped variants and their chunks", func(t *testing.T) {
		store := config.NewStore(t)

		full := buildArtefact("art-v2", "source/v2")
		keep := buildVariant("keep", "keep/v2")
		drop := buildVariant("drop", "drop/v2")
		drop.Chunks = []registry_dto.VariantChunk{
			buildChunk("d0", "dropchunk/0", 0),
			buildChunk("d1", "dropchunk/1", 1),
		}
		full.ActualVariants = append(full.ActualVariants, keep, drop)
		upsert(t, store, full)

		shrunk := buildArtefact("art-v2", "source/v2")
		shrunk.ActualVariants = append(shrunk.ActualVariants, keep)
		upsert(t, store, shrunk)

		got, err := store.GetArtefact(ctx(t), "art-v2")
		require.NoError(t, err)
		assert.Nil(t, findVariant(got, "drop"), "the dropped variant must be gone")
		require.NotNil(t, findVariant(got, "keep"), "the kept variant must remain")

		_, err = store.FindArtefactByVariantStorageKey(ctx(t), "dropchunk/0")
		requireNotFound(t, err)
	})

	t.Run("re-upserting an unchanged variant set is stable", func(t *testing.T) {
		store := config.NewStore(t)
		artefact := buildArtefact("art-v3", "source/v3")
		artefact.ActualVariants = append(artefact.ActualVariants, buildVariant("thumb", "thumb/v3"))

		upsert(t, store, artefact)
		upsert(t, store, artefact)
		upsert(t, store, artefact)

		got, err := store.GetArtefact(ctx(t), "art-v3")
		require.NoError(t, err)
		assert.Len(t, got.ActualVariants, 2, "repeated identical upserts must not duplicate variants")
	})
}

func findVariant(artefact *registry_dto.ArtefactMeta, variantID string) *registry_dto.Variant {
	for i := range artefact.ActualVariants {
		if artefact.ActualVariants[i].VariantID == variantID {
			return &artefact.ActualVariants[i]
		}
	}
	return nil
}
