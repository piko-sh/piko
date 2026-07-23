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
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

type Config struct {
	NewStore                           func(t *testing.T) registry_domain.MetadataStore
	SupportsArtefactLocker             bool
	SupportsRegistryInspector          bool
	SupportsRollback                   bool
	SupportsNestedTransactionRejection bool
	SupportsGCHints                    bool
	SupportsSRIHashPersistence         bool
	SupportsReleasePublisher           bool
}

func RunStoreSuite(t *testing.T, config Config) {
	t.Helper()

	t.Run("CoreOps", func(t *testing.T) { runCoreOps(t, config) })
	t.Run("Atomic", func(t *testing.T) { runAtomic(t, config) })
	t.Run("Transactions", func(t *testing.T) { runTransactions(t, config) })
	t.Run("RefCount", func(t *testing.T) { runRefCount(t, config) })
	t.Run("GCHints", func(t *testing.T) { runGCHints(t, config) })
	t.Run("Search", func(t *testing.T) { runSearch(t, config) })
	t.Run("ReverseLookup", func(t *testing.T) { runReverseLookup(t, config) })
	t.Run("Variants", func(t *testing.T) { runVariants(t, config) })
	t.Run("Fidelity", func(t *testing.T) { runFidelity(t, config) })
	t.Run("Releases", func(t *testing.T) { runReleases(t, config) })
}

var (
	fixedTime = time.Date(2026, 3, 14, 15, 9, 26, 0, time.UTC)
)

func buildArtefact(id, storageKey string) *registry_dto.ArtefactMeta {
	artefact := &registry_dto.ArtefactMeta{
		ID:         id,
		SourcePath: "src/" + id + ".txt",
		CreatedAt:  fixedTime,
		UpdatedAt:  fixedTime,
		ActualVariants: []registry_dto.Variant{
			buildVariant("source", storageKey),
		},
	}
	artefact.Status = artefact.ComputeStatus()
	return artefact
}

func buildVariant(variantID, storageKey string) registry_dto.Variant {
	return registry_dto.Variant{
		VariantID:        variantID,
		StorageKey:       storageKey,
		StorageBackendID: "local_disk_cache",
		MimeType:         "text/plain",
		SizeBytes:        42,
		Status:           registry_dto.VariantStatusReady,
		ContentHash:      "hash-" + storageKey,
		CreatedAt:        fixedTime,
		MetadataTags:     registry_dto.Tags{},
	}
}

func buildChunk(chunkID, storageKey string, sequence int) registry_dto.VariantChunk {
	return registry_dto.VariantChunk{
		ChunkID:          chunkID,
		StorageKey:       storageKey,
		StorageBackendID: "local_disk_cache",
		MimeType:         "application/octet-stream",
		SizeBytes:        16,
		SequenceNumber:   sequence,
		ContentHash:      "chunk-hash-" + storageKey,
		CreatedAt:        fixedTime,
	}
}

func upsert(t *testing.T, store registry_domain.MetadataStore, artefact *registry_dto.ArtefactMeta) {
	t.Helper()
	err := store.AtomicUpdate(t.Context(), []registry_dto.AtomicAction{{
		Type:       registry_dto.ActionTypeUpsertArtefact,
		ArtefactID: artefact.ID,
		Artefact:   artefact,
	}})
	require.NoError(t, err, "upserting artefact %q", artefact.ID)
}

func addGCHints(t *testing.T, store registry_domain.MetadataStore, hints []registry_dto.GCHint) {
	t.Helper()
	err := store.AtomicUpdate(t.Context(), []registry_dto.AtomicAction{{
		Type:    registry_dto.ActionTypeAddGCHints,
		GCHints: hints,
	}})
	require.NoError(t, err, "adding GC hints")
}

func incrementRef(t *testing.T, store registry_domain.MetadataStore, key string) int {
	t.Helper()
	count, err := store.IncrementBlobRefCount(t.Context(), registry_domain.BlobReference{
		StorageKey:       key,
		StorageBackendID: "local_disk_cache",
		ContentHash:      "hash-" + key,
		MimeType:         "text/plain",
		SizeBytes:        42,
		CreatedAt:        fixedTime,
		LastReferencedAt: fixedTime,
	})
	require.NoError(t, err, "incrementing ref count for %q", key)
	return count
}

func ctx(t *testing.T) context.Context {
	return t.Context()
}
