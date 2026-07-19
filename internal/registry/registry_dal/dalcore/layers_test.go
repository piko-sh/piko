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

package dalcore

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/registry/registry_schema"
)

func buildLayerBlob(t *testing.T, artefactID, sourcePath, variantID, storageKey string, updatedAt time.Time) []byte {
	t.Helper()
	artefact := &registry_dto.ArtefactMeta{
		ID:         artefactID,
		SourcePath: sourcePath,
		CreatedAt:  updatedAt,
		UpdatedAt:  updatedAt,
		Status:     registry_dto.VariantStatusReady,
		ActualVariants: []registry_dto.Variant{{
			VariantID:        variantID,
			StorageKey:       storageKey,
			StorageBackendID: "local",
			MimeType:         "image/png",
			Status:           registry_dto.VariantStatusReady,
			SizeBytes:        1024,
			CreatedAt:        updatedAt,
		}},
	}
	data := registry_schema.BuildArtefactMeta(artefact)
	require.NotEmpty(t, data, "the layer blob must serialise")
	return data
}

func TestGetArtefact_MergesLayersWithRuntimePrimary(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	runtimeBlob := buildLayerBlob(t, "art", "runtime/source.png", "runtime-variant", "key-runtime", now)
	releaseBlob := buildLayerBlob(t, "art", "release/source.png", "release-variant", "key-release", now.Add(time.Hour))

	stub := &stubDriver{
		getArtefactLayersFunc: func(_ context.Context, _ string) ([]ArtefactLayerData, error) {
			return []ArtefactLayerData{
				{ID: "art", ReleaseID: "", Data: runtimeBlob},
				{ID: "art", ReleaseID: "rel-1", Data: releaseBlob},
			}, nil
		},
	}
	core := newStubCore(nil, stub)

	artefact, err := core.GetArtefact(t.Context(), "art")

	require.NoError(t, err, "the merged read must succeed")
	assert.Equal(t, "", artefact.ReleaseID, "the runtime layer must be the primary, proven by the merged ReleaseID")
	assert.Equal(t, "runtime/source.png", artefact.SourcePath, "scalars must come from the runtime layer")
	variantIDs := make(map[string]bool)
	for _, v := range artefact.ActualVariants {
		variantIDs[v.VariantID] = true
	}
	assert.True(t, variantIDs["runtime-variant"], "the runtime layer's variant must survive")
	assert.True(t, variantIDs["release-variant"], "the release layer's variant must survive")
}

func TestGetArtefact_WarnsAndServesOnUnparseableLayer(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	goodBlob := buildLayerBlob(t, "art", "runtime/source.png", "runtime-variant", "key-runtime", now)

	stub := &stubDriver{
		getArtefactLayersFunc: func(_ context.Context, _ string) ([]ArtefactLayerData, error) {
			return []ArtefactLayerData{
				{ID: "art", ReleaseID: "", Data: goodBlob},
				{ID: "art", ReleaseID: "rel-corrupt", Data: []byte("not a flatbuffer")},
			}, nil
		},
	}
	core := newStubCore(nil, stub)

	artefact, err := core.GetArtefact(t.Context(), "art")

	require.NoError(t, err, "the plain read must serve what parses")
	assert.Equal(t, "runtime/source.png", artefact.SourcePath, "the good layer must still serve")
	assert.Len(t, artefact.ActualVariants, 1, "only the good layer's records survive")
}

func TestGetArtefactForUpdate_FailsOnUnparseableLayer(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	goodBlob := buildLayerBlob(t, "art", "runtime/source.png", "runtime-variant", "key-runtime", now)

	stub := &stubDriver{
		getArtefactLayersFunc: func(_ context.Context, _ string) ([]ArtefactLayerData, error) {
			return []ArtefactLayerData{
				{ID: "art", ReleaseID: "", Data: goodBlob},
				{ID: "art", ReleaseID: "rel-corrupt", Data: []byte("not a flatbuffer")},
			}, nil
		},
	}
	core := newStubCore(nil, stub)

	_, err := core.GetArtefactForUpdate(t.Context(), "art")

	require.Error(t, err, "a read-modify-write over partial layers must be refused")
	assert.Contains(t, err.Error(), "failed to parse", "the error must explain the refusal")
}

func TestInspector_DeduplicatesMultiLayerArtefacts(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	older := buildLayerBlob(t, "art", "old/source.png", "v1", "key-1", now)
	newer := buildLayerBlob(t, "art", "new/source.png", "v2", "key-2", now.Add(time.Hour))

	stub := &stubDriver{
		listAllArtefactsDataFunc: func(_ context.Context) ([][]byte, error) {
			return [][]byte{older, newer}, nil
		},
		listRecentArtefactsDataFunc: func(_ context.Context, _ int) ([][]byte, error) {
			return [][]byte{newer, older}, nil
		},
	}
	core := newStubCore(nil, stub)

	summary, err := core.ListArtefactSummary(t.Context())
	require.NoError(t, err, "the summary must succeed")
	var total int64
	for _, entry := range summary {
		total += entry.Count
	}
	assert.Equal(t, int64(1), total, "two layers of one artefact must count once")

	recent, err := core.ListRecentArtefacts(t.Context(), 10)
	require.NoError(t, err, "the recent listing must succeed")
	require.Len(t, recent, 1, "two layers of one artefact must list once")
	assert.Equal(t, "new/source.png", recent[0].SourcePath, "the newest layer must win the listing")
}

func TestAtomicUpdate_RuntimeUpsertStripsReleaseProvidedRecords(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	releaseBlob := buildLayerBlob(t, "art", "release/source.png", "shared-variant", "key-shared", now)

	var persisted []byte
	stub := &stubDriver{
		getArtefactLayersFunc: func(_ context.Context, _ string) ([]ArtefactLayerData, error) {
			return []ArtefactLayerData{{ID: "art", ReleaseID: "rel-1", Data: releaseBlob}}, nil
		},
		upsertArtefactFunc: func(_ context.Context, params UpsertArtefactParams) error {
			persisted = params.DataFbs
			return nil
		},
	}
	core := newStubCore(newFakeDB(t), stub)

	incoming := &registry_dto.ArtefactMeta{
		ID:         "art",
		SourcePath: "runtime/source.png",
		CreatedAt:  now,
		UpdatedAt:  now,
		ActualVariants: []registry_dto.Variant{
			{
				VariantID:        "shared-variant",
				StorageKey:       "key-shared",
				StorageBackendID: "local",
				MimeType:         "image/png",
				Status:           registry_dto.VariantStatusReady,
				SizeBytes:        1024,
				CreatedAt:        now,
			},
			{
				VariantID:        "runtime-only",
				StorageKey:       "key-runtime",
				StorageBackendID: "local",
				MimeType:         "image/png",
				Status:           registry_dto.VariantStatusReady,
				SizeBytes:        2048,
				CreatedAt:        now,
			},
		},
	}
	err := core.AtomicUpdate(t.Context(), []registry_dto.AtomicAction{{
		Type:     registry_dto.ActionTypeUpsertArtefact,
		Artefact: incoming,
	}})

	require.NoError(t, err, "the runtime upsert must succeed")
	require.NotNil(t, persisted, "the artefact row must still be written")
	written := registry_schema.ParseArtefactMeta(persisted)
	require.NotNil(t, written, "the persisted payload must parse")
	require.Len(t, written.ActualVariants, 1, "the release-provided record must be stripped")
	assert.Equal(t, "runtime-only", written.ActualVariants[0].VariantID, "only the runtime delta persists")
	assert.Equal(t, "runtime/source.png", written.SourcePath, "the runtime layer keeps its scalar intent")
}

func TestAtomicUpdate_RuntimeUpsertKeepsDifferingStorageKey(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	releaseBlob := buildLayerBlob(t, "art", "release/source.png", "shared-variant", "key-release", now)

	var persisted []byte
	stub := &stubDriver{
		getArtefactLayersFunc: func(_ context.Context, _ string) ([]ArtefactLayerData, error) {
			return []ArtefactLayerData{{ID: "art", ReleaseID: "rel-1", Data: releaseBlob}}, nil
		},
		upsertArtefactFunc: func(_ context.Context, params UpsertArtefactParams) error {
			persisted = params.DataFbs
			return nil
		},
	}
	core := newStubCore(newFakeDB(t), stub)

	incoming := &registry_dto.ArtefactMeta{
		ID:         "art",
		SourcePath: "runtime/source.png",
		CreatedAt:  now,
		UpdatedAt:  now,
		ActualVariants: []registry_dto.Variant{{
			VariantID:        "shared-variant",
			StorageKey:       "key-runtime-differs",
			StorageBackendID: "local",
			MimeType:         "image/png",
			Status:           registry_dto.VariantStatusReady,
			SizeBytes:        1024,
			CreatedAt:        now,
		}},
	}
	err := core.AtomicUpdate(t.Context(), []registry_dto.AtomicAction{{
		Type:     registry_dto.ActionTypeUpsertArtefact,
		Artefact: incoming,
	}})

	require.NoError(t, err, "the runtime upsert must succeed")
	written := registry_schema.ParseArtefactMeta(persisted)
	require.NotNil(t, written, "the persisted payload must parse")
	require.Len(t, written.ActualVariants, 1, "a record with different bytes is NOT release-provided")
	assert.Equal(t, "key-runtime-differs", written.ActualVariants[0].StorageKey, "the differing record must persist")
}
