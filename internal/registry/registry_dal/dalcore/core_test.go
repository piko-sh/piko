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
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/registry/registry_dal"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/registry/registry_schema"
	"piko.sh/piko/wdk/clock"
)

type fakeConnector struct{}

func (fakeConnector) Connect(_ context.Context) (driver.Conn, error) { return &fakeConn{}, nil }
func (fakeConnector) Driver() driver.Driver                          { return fakeSQLDriver{} }

type fakeSQLDriver struct{}

func (fakeSQLDriver) Open(_ string) (driver.Conn, error) { return &fakeConn{}, nil }

type fakeConn struct{}

func (*fakeConn) Prepare(_ string) (driver.Stmt, error) { return nil, errors.New("not supported") }
func (*fakeConn) Close() error                          { return nil }
func (*fakeConn) Begin() (driver.Tx, error)             { return fakeTx{}, nil }
func (*fakeConn) BeginTx(_ context.Context, _ driver.TxOptions) (driver.Tx, error) {
	return fakeTx{}, nil
}

type fakeTx struct{}

func (fakeTx) Commit() error   { return nil }
func (fakeTx) Rollback() error { return nil }

func newFakeDB(t *testing.T) *sql.DB {
	t.Helper()
	database := sql.OpenDB(fakeConnector{})
	t.Cleanup(func() { _ = database.Close() })
	return database
}

type stubDriver struct {
	getArtefactDataFunc                  func(ctx context.Context, artefactID string) ([]byte, error)
	getMultipleArtefactsDataFunc         func(ctx context.Context, artefactIDs []string) ([][]byte, error)
	listAllArtefactsDataFunc             func(ctx context.Context) ([][]byte, error)
	listRecentArtefactsDataFunc          func(ctx context.Context, limit int) ([][]byte, error)
	listAllArtefactIDsFunc               func(ctx context.Context) ([]string, error)
	findArtefactIDsByTagFunc             func(ctx context.Context, tagKey, tagValue string) ([]string, error)
	findArtefactIDsByTagValuesFunc       func(ctx context.Context, tagKey string, tagValues []string) ([]string, error)
	findArtefactIDByVariantStorageKeyFn  func(ctx context.Context, storageKey string) (string, error)
	listVariantStatusCountsFunc          func(ctx context.Context) ([]VariantStatusCount, error)
	incrementBlobRefCountFunc            func(ctx context.Context, params IncrementBlobRefCountParams) (int, error)
	decrementBlobRefCountFunc            func(ctx context.Context, storageKey string, lastReferencedAt int64) (int, error)
	deleteBlobReferenceIfZeroFunc        func(ctx context.Context, storageKey string) error
	getBlobRefCountFunc                  func(ctx context.Context, storageKey string) (int, error)
	popGCHintsFunc                       func(ctx context.Context, limit int) ([]GCHintRow, error)
	deleteGCHintsFunc                    func(ctx context.Context, ids []int64) error
	addGCHintFunc                        func(ctx context.Context, backendID, storageKey string, createdAt int64) error
	upsertArtefactFunc                   func(ctx context.Context, params UpsertArtefactParams) error
	deleteArtefactFunc                   func(ctx context.Context, artefactID string) error
	deleteVariantTagsForArtefactFunc     func(ctx context.Context, artefactID string) error
	deleteChunksForVariantFunc           func(ctx context.Context, artefactID, variantID string) error
	deleteVariantsForArtefactFunc        func(ctx context.Context, artefactID string) error
	deleteDesiredProfilesForArtefactFunc func(ctx context.Context, artefactID string) error
	insertVariantFunc                    func(ctx context.Context, params InsertVariantParams) error
	insertVariantTagFunc                 func(ctx context.Context, artefactID, variantID, tagKey, tagValue string) error
	insertVariantChunkFunc               func(ctx context.Context, params InsertVariantChunkParams) error
	insertDesiredProfileFunc             func(ctx context.Context, params InsertDesiredProfileParams) error
	calls          *[]string
	popGCHintLimit *int
	deletedGCHints *[]int64
}

func (s *stubDriver) record(name string) {
	if s.calls != nil {
		*s.calls = append(*s.calls, name)
	}
}

func (s *stubDriver) WithTx(_ *sql.Tx) Driver { return s }

func (s *stubDriver) GetArtefactData(ctx context.Context, artefactID string) ([]byte, error) {
	if s.getArtefactDataFunc != nil {
		return s.getArtefactDataFunc(ctx, artefactID)
	}
	return nil, nil
}

func (s *stubDriver) GetMultipleArtefactsData(ctx context.Context, artefactIDs []string) ([][]byte, error) {
	if s.getMultipleArtefactsDataFunc != nil {
		return s.getMultipleArtefactsDataFunc(ctx, artefactIDs)
	}
	return nil, nil
}

func (s *stubDriver) ListAllArtefactsData(ctx context.Context) ([][]byte, error) {
	if s.listAllArtefactsDataFunc != nil {
		return s.listAllArtefactsDataFunc(ctx)
	}
	return nil, nil
}

func (s *stubDriver) ListRecentArtefactsData(ctx context.Context, limit int) ([][]byte, error) {
	if s.listRecentArtefactsDataFunc != nil {
		return s.listRecentArtefactsDataFunc(ctx, limit)
	}
	return nil, nil
}

func (s *stubDriver) ListAllArtefactIDs(ctx context.Context) ([]string, error) {
	if s.listAllArtefactIDsFunc != nil {
		return s.listAllArtefactIDsFunc(ctx)
	}
	return nil, nil
}

func (s *stubDriver) FindArtefactIDsByTag(ctx context.Context, tagKey, tagValue string) ([]string, error) {
	if s.findArtefactIDsByTagFunc != nil {
		return s.findArtefactIDsByTagFunc(ctx, tagKey, tagValue)
	}
	return nil, nil
}

func (s *stubDriver) FindArtefactIDsByTagValues(ctx context.Context, tagKey string, tagValues []string) ([]string, error) {
	if s.findArtefactIDsByTagValuesFunc != nil {
		return s.findArtefactIDsByTagValuesFunc(ctx, tagKey, tagValues)
	}
	return nil, nil
}

func (s *stubDriver) FindArtefactIDByVariantStorageKey(ctx context.Context, storageKey string) (string, error) {
	if s.findArtefactIDByVariantStorageKeyFn != nil {
		return s.findArtefactIDByVariantStorageKeyFn(ctx, storageKey)
	}
	return "", nil
}

func (s *stubDriver) ListVariantStatusCounts(ctx context.Context) ([]VariantStatusCount, error) {
	if s.listVariantStatusCountsFunc != nil {
		return s.listVariantStatusCountsFunc(ctx)
	}
	return nil, nil
}

func (s *stubDriver) IncrementBlobRefCount(ctx context.Context, params IncrementBlobRefCountParams) (int, error) {
	if s.incrementBlobRefCountFunc != nil {
		return s.incrementBlobRefCountFunc(ctx, params)
	}
	return 0, nil
}

func (s *stubDriver) DecrementBlobRefCount(ctx context.Context, storageKey string, lastReferencedAt int64) (int, error) {
	if s.decrementBlobRefCountFunc != nil {
		return s.decrementBlobRefCountFunc(ctx, storageKey, lastReferencedAt)
	}
	return 0, nil
}

func (s *stubDriver) DeleteBlobReferenceIfZero(ctx context.Context, storageKey string) error {
	s.record("DeleteBlobReferenceIfZero")
	if s.deleteBlobReferenceIfZeroFunc != nil {
		return s.deleteBlobReferenceIfZeroFunc(ctx, storageKey)
	}
	return nil
}

func (s *stubDriver) GetBlobRefCount(ctx context.Context, storageKey string) (int, error) {
	if s.getBlobRefCountFunc != nil {
		return s.getBlobRefCountFunc(ctx, storageKey)
	}
	return 0, nil
}

func (s *stubDriver) PopGCHints(ctx context.Context, limit int) ([]GCHintRow, error) {
	if s.popGCHintLimit != nil {
		*s.popGCHintLimit = limit
	}
	if s.popGCHintsFunc != nil {
		return s.popGCHintsFunc(ctx, limit)
	}
	return nil, nil
}

func (s *stubDriver) DeleteGCHints(ctx context.Context, ids []int64) error {
	s.record("DeleteGCHints")
	if s.deletedGCHints != nil {
		*s.deletedGCHints = ids
	}
	if s.deleteGCHintsFunc != nil {
		return s.deleteGCHintsFunc(ctx, ids)
	}
	return nil
}

func (s *stubDriver) AddGCHint(ctx context.Context, backendID, storageKey string, createdAt int64) error {
	s.record("AddGCHint")
	if s.addGCHintFunc != nil {
		return s.addGCHintFunc(ctx, backendID, storageKey, createdAt)
	}
	return nil
}

func (s *stubDriver) UpsertArtefact(ctx context.Context, params UpsertArtefactParams) error {
	s.record("UpsertArtefact")
	if s.upsertArtefactFunc != nil {
		return s.upsertArtefactFunc(ctx, params)
	}
	return nil
}

func (s *stubDriver) DeleteArtefact(ctx context.Context, artefactID string) error {
	s.record("DeleteArtefact")
	if s.deleteArtefactFunc != nil {
		return s.deleteArtefactFunc(ctx, artefactID)
	}
	return nil
}

func (s *stubDriver) DeleteVariantTagsForArtefact(ctx context.Context, artefactID string) error {
	s.record("DeleteVariantTagsForArtefact")
	if s.deleteVariantTagsForArtefactFunc != nil {
		return s.deleteVariantTagsForArtefactFunc(ctx, artefactID)
	}
	return nil
}

func (s *stubDriver) DeleteChunksForVariant(ctx context.Context, artefactID, variantID string) error {
	s.record("DeleteChunksForVariant")
	if s.deleteChunksForVariantFunc != nil {
		return s.deleteChunksForVariantFunc(ctx, artefactID, variantID)
	}
	return nil
}

func (s *stubDriver) DeleteVariantsForArtefact(ctx context.Context, artefactID string) error {
	s.record("DeleteVariantsForArtefact")
	if s.deleteVariantsForArtefactFunc != nil {
		return s.deleteVariantsForArtefactFunc(ctx, artefactID)
	}
	return nil
}

func (s *stubDriver) DeleteDesiredProfilesForArtefact(ctx context.Context, artefactID string) error {
	s.record("DeleteDesiredProfilesForArtefact")
	if s.deleteDesiredProfilesForArtefactFunc != nil {
		return s.deleteDesiredProfilesForArtefactFunc(ctx, artefactID)
	}
	return nil
}

func (s *stubDriver) InsertVariant(ctx context.Context, params InsertVariantParams) error {
	s.record("InsertVariant")
	if s.insertVariantFunc != nil {
		return s.insertVariantFunc(ctx, params)
	}
	return nil
}

func (s *stubDriver) InsertVariantTag(ctx context.Context, artefactID, variantID, tagKey, tagValue string) error {
	s.record("InsertVariantTag")
	if s.insertVariantTagFunc != nil {
		return s.insertVariantTagFunc(ctx, artefactID, variantID, tagKey, tagValue)
	}
	return nil
}

func (s *stubDriver) InsertVariantChunk(ctx context.Context, params InsertVariantChunkParams) error {
	s.record("InsertVariantChunk")
	if s.insertVariantChunkFunc != nil {
		return s.insertVariantChunkFunc(ctx, params)
	}
	return nil
}

func (s *stubDriver) InsertDesiredProfile(ctx context.Context, params InsertDesiredProfileParams) error {
	s.record("InsertDesiredProfile")
	if s.insertDesiredProfileFunc != nil {
		return s.insertDesiredProfileFunc(ctx, params)
	}
	return nil
}

var (
	_ Driver = (*stubDriver)(nil)
)

func newStubCore(database *sql.DB, stub *stubDriver) *core {
	return &core{sqlDB: database, driver: stub, clock: clock.RealClock()}
}

func buildArtefactBlob(t *testing.T, artefactID string) []byte {
	t.Helper()
	now := time.Unix(1_700_000_000, 0).UTC()
	artefact := &registry_dto.ArtefactMeta{
		ID:         artefactID,
		SourcePath: "source/" + artefactID + ".png",
		CreatedAt:  now,
		UpdatedAt:  now,
		Status:     registry_dto.VariantStatusReady,
		ActualVariants: []registry_dto.Variant{
			{
				VariantID:        "source",
				StorageKey:       "blob/" + artefactID,
				StorageBackendID: "local",
				MimeType:         "image/png",
				Status:           registry_dto.VariantStatusReady,
				SizeBytes:        1024,
				CreatedAt:        now,
			},
		},
	}
	blob := registry_schema.BuildArtefactMeta(artefact)
	require.NotEmpty(t, blob, "BuildArtefactMeta must return a non-empty blob")
	return blob
}

func TestIntersectIDSets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		current  *map[string]struct{}
		expected map[string]struct{}
		name     string
		ids      []string
	}{
		{
			name:     "nil current creates a new set from ids",
			current:  nil,
			ids:      []string{"a", "b", "c"},
			expected: map[string]struct{}{"a": {}, "b": {}, "c": {}},
		},
		{
			name:     "non-nil current keeps only the intersection",
			current:  &map[string]struct{}{"a": {}, "b": {}, "c": {}},
			ids:      []string{"b", "c", "d"},
			expected: map[string]struct{}{"b": {}, "c": {}},
		},
		{
			name:     "disjoint sets produce an empty intersection",
			current:  &map[string]struct{}{"a": {}},
			ids:      []string{"z"},
			expected: map[string]struct{}{},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			result := intersectIDSets(testCase.current, testCase.ids)
			require.NotNil(t, result, "intersectIDSets must never return nil")
			assert.Equal(t, testCase.expected, *result, "intersection must match expectation")
		})
	}
}

func TestConvertHintsToDTO(t *testing.T) {
	t.Parallel()

	dbHints := []GCHintRow{
		{BackendID: "local", StorageKey: "blob/one", ID: 1},
		{BackendID: "s3", StorageKey: "blob/two", ID: 2},
	}

	hints, idsToDelete := convertHintsToDTO(dbHints)

	require.Equal(t, []registry_dto.GCHint{
		{BackendID: "local", StorageKey: "blob/one"},
		{BackendID: "s3", StorageKey: "blob/two"},
	}, hints, "converted hints must map backend and storage key")
	require.Equal(t, []int64{1, 2}, idsToDelete, "deletion IDs must preserve order")
}

func TestProcessAtomicAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		action      registry_dto.AtomicAction
		expectError string
		expectCalls []string
	}{
		{
			name: "upsert artefact fans out to inserts",
			action: registry_dto.AtomicAction{
				Type:     registry_dto.ActionTypeUpsertArtefact,
				Artefact: &registry_dto.ArtefactMeta{ID: "art-1"},
			},
			expectCalls: []string{
				"UpsertArtefact",
				"DeleteVariantTagsForArtefact",
				"DeleteVariantsForArtefact",
				"DeleteDesiredProfilesForArtefact",
			},
		},
		{
			name: "delete artefact calls DeleteArtefact",
			action: registry_dto.AtomicAction{
				Type:       registry_dto.ActionTypeDeleteArtefact,
				ArtefactID: "art-9",
			},
			expectCalls: []string{"DeleteArtefact"},
		},
		{
			name: "add GC hints calls AddGCHint per hint",
			action: registry_dto.AtomicAction{
				Type: registry_dto.ActionTypeAddGCHints,
				GCHints: []registry_dto.GCHint{
					{BackendID: "local", StorageKey: "blob/a"},
					{BackendID: "local", StorageKey: "blob/b"},
				},
			},
			expectCalls: []string{"AddGCHint", "AddGCHint"},
		},
		{
			name:        "unrecognised action type returns an error",
			action:      registry_dto.AtomicAction{Type: registry_dto.ActionType("BOGUS")},
			expectError: "unrecognised atomic action type",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var calls []string
			stub := &stubDriver{calls: &calls}

			err := processAtomicAction(t.Context(), stub, testCase.action, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))

			if testCase.expectError != "" {
				require.ErrorContains(t, err, testCase.expectError, "unexpected error")
				return
			}
			require.NoError(t, err, "action must succeed")
			assert.Equal(t, testCase.expectCalls, calls, "recorded driver calls must match")
		})
	}
}

func TestGetArtefact(t *testing.T) {
	t.Parallel()

	t.Run("found returns parsed artefact", func(t *testing.T) {
		t.Parallel()
		blob := buildArtefactBlob(t, "art-found")
		stub := &stubDriver{
			getArtefactDataFunc: func(_ context.Context, _ string) ([]byte, error) {
				return blob, nil
			},
		}
		core := newStubCore(nil, stub)

		artefact, err := core.GetArtefact(t.Context(), "art-found")

		require.NoError(t, err, "GetArtefact must succeed for a valid blob")
		require.NotNil(t, artefact, "artefact must be parsed")
		assert.Equal(t, "art-found", artefact.ID, "parsed ID must match")
		assert.Len(t, artefact.ActualVariants, 1, "variant must round-trip")
	})

	t.Run("no rows maps to ErrArtefactNotFound", func(t *testing.T) {
		t.Parallel()
		stub := &stubDriver{
			getArtefactDataFunc: func(_ context.Context, _ string) ([]byte, error) {
				return nil, sql.ErrNoRows
			},
		}
		core := newStubCore(nil, stub)

		artefact, err := core.GetArtefact(t.Context(), "missing")

		require.ErrorIs(t, err, registry_domain.ErrArtefactNotFound, "missing artefact must map to sentinel")
		assert.Nil(t, artefact, "artefact must be nil when not found")
	})

	t.Run("driver error is wrapped", func(t *testing.T) {
		t.Parallel()
		sentinel := errors.New("boom")
		stub := &stubDriver{
			getArtefactDataFunc: func(_ context.Context, _ string) ([]byte, error) {
				return nil, sentinel
			},
		}
		core := newStubCore(nil, stub)

		_, err := core.GetArtefact(t.Context(), "art")

		require.ErrorIs(t, err, sentinel, "driver error must be wrapped")
	})

	t.Run("empty blob is a parse error", func(t *testing.T) {
		t.Parallel()
		stub := &stubDriver{
			getArtefactDataFunc: func(_ context.Context, _ string) ([]byte, error) {
				return []byte{}, nil
			},
		}
		core := newStubCore(nil, stub)

		_, err := core.GetArtefact(t.Context(), "art")

		require.ErrorContains(t, err, "corrupted or empty data", "empty blob must produce a parse error")
	})
}

func TestGetMultipleArtefacts(t *testing.T) {
	t.Parallel()

	t.Run("empty input returns an empty slice without calling the driver", func(t *testing.T) {
		t.Parallel()
		driverCalled := false
		stub := &stubDriver{
			getMultipleArtefactsDataFunc: func(_ context.Context, _ []string) ([][]byte, error) {
				driverCalled = true
				return nil, nil
			},
		}
		core := newStubCore(nil, stub)

		results, err := core.GetMultipleArtefacts(t.Context(), nil)

		require.NoError(t, err, "empty input must not error")
		assert.Empty(t, results, "empty input must yield an empty result")
		assert.False(t, driverCalled, "driver must not be called for empty input")
	})

	t.Run("preserves order and drops missing artefacts", func(t *testing.T) {
		t.Parallel()
		first := buildArtefactBlob(t, "art-1")
		third := buildArtefactBlob(t, "art-3")
		stub := &stubDriver{
			getMultipleArtefactsDataFunc: func(_ context.Context, _ []string) ([][]byte, error) {
				return [][]byte{third, first}, nil
			},
		}
		core := newStubCore(nil, stub)

		results, err := core.GetMultipleArtefacts(t.Context(), []string{"art-1", "art-2", "art-3"})

		require.NoError(t, err, "retrieval must succeed")
		require.Len(t, results, 2, "missing artefact must be dropped")
		assert.Equal(t, "art-1", results[0].ID, "requested order must be preserved")
		assert.Equal(t, "art-3", results[1].ID, "requested order must be preserved")
	})

	t.Run("driver error is wrapped", func(t *testing.T) {
		t.Parallel()
		sentinel := errors.New("db down")
		stub := &stubDriver{
			getMultipleArtefactsDataFunc: func(_ context.Context, _ []string) ([][]byte, error) {
				return nil, sentinel
			},
		}
		core := newStubCore(nil, stub)

		_, err := core.GetMultipleArtefacts(t.Context(), []string{"art-1"})

		require.ErrorIs(t, err, sentinel, "driver error must be wrapped")
	})
}

func TestFindArtefactByVariantStorageKey(t *testing.T) {
	t.Parallel()

	t.Run("found delegates to GetArtefact", func(t *testing.T) {
		t.Parallel()
		blob := buildArtefactBlob(t, "art-key")
		stub := &stubDriver{
			findArtefactIDByVariantStorageKeyFn: func(_ context.Context, _ string) (string, error) {
				return "art-key", nil
			},
			getArtefactDataFunc: func(_ context.Context, _ string) ([]byte, error) {
				return blob, nil
			},
		}
		core := newStubCore(nil, stub)

		artefact, err := core.FindArtefactByVariantStorageKey(t.Context(), "blob/art-key")

		require.NoError(t, err, "lookup must succeed")
		assert.Equal(t, "art-key", artefact.ID, "resolved artefact must match")
	})

	t.Run("no rows maps to ErrArtefactNotFound", func(t *testing.T) {
		t.Parallel()
		stub := &stubDriver{
			findArtefactIDByVariantStorageKeyFn: func(_ context.Context, _ string) (string, error) {
				return "", sql.ErrNoRows
			},
		}
		core := newStubCore(nil, stub)

		_, err := core.FindArtefactByVariantStorageKey(t.Context(), "missing")

		require.ErrorIs(t, err, registry_domain.ErrArtefactNotFound, "missing key must map to sentinel")
	})
}

func TestListAllArtefactIDs(t *testing.T) {
	t.Parallel()

	stub := &stubDriver{
		listAllArtefactIDsFunc: func(_ context.Context) ([]string, error) {
			return []string{"a", "b"}, nil
		},
	}
	core := newStubCore(nil, stub)

	ids, err := core.ListAllArtefactIDs(t.Context())

	require.NoError(t, err, "listing IDs must succeed")
	assert.Equal(t, []string{"a", "b"}, ids, "IDs must pass through")
}

func TestListArtefactSummary(t *testing.T) {
	t.Parallel()

	ready := buildArtefactBlob(t, "art-ready")
	stub := &stubDriver{
		listAllArtefactsDataFunc: func(_ context.Context) ([][]byte, error) {
			return [][]byte{ready, {}, nil}, nil
		},
	}
	core := newStubCore(nil, stub)

	summary, err := core.ListArtefactSummary(t.Context())

	require.NoError(t, err, "summary must succeed")
	require.Len(t, summary, 1, "corrupt blobs must be skipped")
	assert.Equal(t, string(registry_dto.VariantStatusReady), summary[0].Status, "status must be aggregated")
	assert.Equal(t, int64(1), summary[0].Count, "count must reflect valid blobs")
}

func TestListArtefactSummaryDriverError(t *testing.T) {
	t.Parallel()

	stub := &stubDriver{
		listAllArtefactsDataFunc: func(_ context.Context) ([][]byte, error) {
			return nil, errors.New("db error")
		},
	}
	core := newStubCore(nil, stub)

	_, err := core.ListArtefactSummary(t.Context())

	require.Error(t, err, "driver error must surface")
}

func TestListVariantSummary(t *testing.T) {
	t.Parallel()

	stub := &stubDriver{
		listVariantStatusCountsFunc: func(_ context.Context) ([]VariantStatusCount, error) {
			return []VariantStatusCount{{Status: "READY", Count: 3}, {Status: "STALE", Count: 1}}, nil
		},
	}
	core := newStubCore(nil, stub)

	summary, err := core.ListVariantSummary(t.Context())

	require.NoError(t, err, "variant summary must succeed")
	require.Len(t, summary, 2, "each status must map to a summary entry")
	assert.Equal(t, "READY", summary[0].Status, "status must map through")
	assert.Equal(t, int64(3), summary[0].Count, "count must map through")
}

func TestListVariantSummaryDriverError(t *testing.T) {
	t.Parallel()

	stub := &stubDriver{
		listVariantStatusCountsFunc: func(_ context.Context) ([]VariantStatusCount, error) {
			return nil, errors.New("db error")
		},
	}
	core := newStubCore(nil, stub)

	_, err := core.ListVariantSummary(t.Context())

	require.Error(t, err, "driver error must surface")
}

func TestListRecentArtefacts(t *testing.T) {
	t.Parallel()

	blob := buildArtefactBlob(t, "art-recent")
	stub := &stubDriver{
		listRecentArtefactsDataFunc: func(_ context.Context, _ int) ([][]byte, error) {
			return [][]byte{blob, nil}, nil
		},
	}
	core := newStubCore(nil, stub)

	items, err := core.ListRecentArtefacts(t.Context(), 10)

	require.NoError(t, err, "listing recent artefacts must succeed")
	require.Len(t, items, 1, "corrupt blobs must be skipped")
	assert.Equal(t, "art-recent", items[0].ID, "ID must map through")
	assert.Equal(t, int64(1), items[0].VariantCount, "variant count must be computed")
	assert.Equal(t, int64(1024), items[0].TotalSize, "total size must sum variant sizes")
}

func TestListRecentArtefactsDriverError(t *testing.T) {
	t.Parallel()

	stub := &stubDriver{
		listRecentArtefactsDataFunc: func(_ context.Context, _ int) ([][]byte, error) {
			return nil, errors.New("db error")
		},
	}
	core := newStubCore(nil, stub)

	_, err := core.ListRecentArtefacts(t.Context(), 10)

	require.Error(t, err, "driver error must surface")
}

func TestSearchArtefacts(t *testing.T) {
	t.Parallel()

	t.Run("raw RediSearch query is unsupported", func(t *testing.T) {
		t.Parallel()
		core := newStubCore(nil, &stubDriver{})

		_, err := core.SearchArtefacts(t.Context(), registry_domain.SearchQuery{RawRediSearchQuery: "@x:{y}"})

		require.ErrorIs(t, err, registry_dal.ErrSearchUnsupported, "raw query must be rejected")
	})

	t.Run("empty query is rejected", func(t *testing.T) {
		t.Parallel()
		core := newStubCore(nil, &stubDriver{})

		_, err := core.SearchArtefacts(t.Context(), registry_domain.SearchQuery{})

		require.ErrorIs(t, err, errSearchQueryEmpty, "empty query must be rejected")
	})

	t.Run("tag query resolves and fetches artefacts", func(t *testing.T) {
		t.Parallel()
		blob := buildArtefactBlob(t, "art-tag")
		stub := &stubDriver{
			findArtefactIDsByTagFunc: func(_ context.Context, _, _ string) ([]string, error) {
				return []string{"art-tag"}, nil
			},
			getMultipleArtefactsDataFunc: func(_ context.Context, _ []string) ([][]byte, error) {
				return [][]byte{blob}, nil
			},
		}
		core := newStubCore(nil, stub)

		results, err := core.SearchArtefacts(t.Context(), registry_domain.SearchQuery{
			SimpleTagQuery: map[string]string{"category": "images"},
		})

		require.NoError(t, err, "tag search must succeed")
		require.Len(t, results, 1, "matching artefact must be returned")
		assert.Equal(t, "art-tag", results[0].ID, "result must match tag query")
	})

	t.Run("tag query with no matches returns empty", func(t *testing.T) {
		t.Parallel()
		stub := &stubDriver{
			findArtefactIDsByTagFunc: func(_ context.Context, _, _ string) ([]string, error) {
				return nil, nil
			},
		}
		core := newStubCore(nil, stub)

		results, err := core.SearchArtefacts(t.Context(), registry_domain.SearchQuery{
			SimpleTagQuery: map[string]string{"category": "none"},
		})

		require.NoError(t, err, "empty match must not error")
		assert.Empty(t, results, "no matches must yield an empty result")
	})

	t.Run("tag query driver error is wrapped", func(t *testing.T) {
		t.Parallel()
		stub := &stubDriver{
			findArtefactIDsByTagFunc: func(_ context.Context, _, _ string) ([]string, error) {
				return nil, errors.New("tag lookup failed")
			},
		}
		core := newStubCore(nil, stub)

		_, err := core.SearchArtefacts(t.Context(), registry_domain.SearchQuery{
			SimpleTagQuery: map[string]string{"category": "images"},
		})

		require.ErrorContains(t, err, "processing tag queries", "driver error must be wrapped")
	})
}

func TestSearchArtefactsByTagValues(t *testing.T) {
	t.Parallel()

	t.Run("empty values returns empty", func(t *testing.T) {
		t.Parallel()
		core := newStubCore(nil, &stubDriver{})

		results, err := core.SearchArtefactsByTagValues(t.Context(), "category", nil)

		require.NoError(t, err, "empty values must not error")
		assert.Empty(t, results, "empty values must yield an empty result")
	})

	t.Run("no matching IDs returns empty", func(t *testing.T) {
		t.Parallel()
		stub := &stubDriver{
			findArtefactIDsByTagValuesFunc: func(_ context.Context, _ string, _ []string) ([]string, error) {
				return nil, nil
			},
		}
		core := newStubCore(nil, stub)

		results, err := core.SearchArtefactsByTagValues(t.Context(), "category", []string{"images"})

		require.NoError(t, err, "no matches must not error")
		assert.Empty(t, results, "no matches must yield an empty result")
	})

	t.Run("matching IDs are fetched", func(t *testing.T) {
		t.Parallel()
		blob := buildArtefactBlob(t, "art-values")
		stub := &stubDriver{
			findArtefactIDsByTagValuesFunc: func(_ context.Context, _ string, _ []string) ([]string, error) {
				return []string{"art-values"}, nil
			},
			getMultipleArtefactsDataFunc: func(_ context.Context, _ []string) ([][]byte, error) {
				return [][]byte{blob}, nil
			},
		}
		core := newStubCore(nil, stub)

		results, err := core.SearchArtefactsByTagValues(t.Context(), "category", []string{"images"})

		require.NoError(t, err, "value search must succeed")
		require.Len(t, results, 1, "matching artefact must be returned")
		assert.Equal(t, "art-values", results[0].ID, "result must match")
	})

	t.Run("driver error is wrapped", func(t *testing.T) {
		t.Parallel()
		stub := &stubDriver{
			findArtefactIDsByTagValuesFunc: func(_ context.Context, _ string, _ []string) ([]string, error) {
				return nil, errors.New("db error")
			},
		}
		core := newStubCore(nil, stub)

		_, err := core.SearchArtefactsByTagValues(t.Context(), "category", []string{"images"})

		require.Error(t, err, "driver error must surface")
	})
}

func TestIncrementBlobRefCount(t *testing.T) {
	t.Parallel()

	t.Run("returns the new count", func(t *testing.T) {
		t.Parallel()
		stub := &stubDriver{
			incrementBlobRefCountFunc: func(_ context.Context, params IncrementBlobRefCountParams) (int, error) {
				assert.Equal(t, "blob/x", params.StorageKey, "storage key must pass through")
				return 5, nil
			},
		}
		core := newStubCore(nil, stub)

		count, err := core.IncrementBlobRefCount(t.Context(), registry_domain.BlobReference{StorageKey: "blob/x"})

		require.NoError(t, err, "increment must succeed")
		assert.Equal(t, 5, count, "new count must be returned")
	})

	t.Run("driver error is wrapped", func(t *testing.T) {
		t.Parallel()
		stub := &stubDriver{
			incrementBlobRefCountFunc: func(_ context.Context, _ IncrementBlobRefCountParams) (int, error) {
				return 0, errors.New("db error")
			},
		}
		core := newStubCore(nil, stub)

		_, err := core.IncrementBlobRefCount(t.Context(), registry_domain.BlobReference{StorageKey: "blob/x"})

		require.Error(t, err, "driver error must surface")
	})
}

func TestDecrementBlobRefCount(t *testing.T) {
	t.Parallel()

	t.Run("zero count triggers deletion", func(t *testing.T) {
		t.Parallel()
		var calls []string
		stub := &stubDriver{
			calls: &calls,
			decrementBlobRefCountFunc: func(_ context.Context, _ string, _ int64) (int, error) {
				return 0, nil
			},
		}
		core := newStubCore(nil, stub)

		count, shouldDelete, err := core.DecrementBlobRefCount(t.Context(), "blob/x")

		require.NoError(t, err, "decrement must succeed")
		assert.Equal(t, 0, count, "count must be zero")
		assert.True(t, shouldDelete, "zero count must request deletion")
		assert.Equal(t, []string{"DeleteBlobReferenceIfZero"}, calls, "deletion must be attempted")
	})

	t.Run("positive count does not trigger deletion", func(t *testing.T) {
		t.Parallel()
		var calls []string
		stub := &stubDriver{
			calls: &calls,
			decrementBlobRefCountFunc: func(_ context.Context, _ string, _ int64) (int, error) {
				return 2, nil
			},
		}
		core := newStubCore(nil, stub)

		count, shouldDelete, err := core.DecrementBlobRefCount(t.Context(), "blob/x")

		require.NoError(t, err, "decrement must succeed")
		assert.Equal(t, 2, count, "count must be positive")
		assert.False(t, shouldDelete, "positive count must not request deletion")
		assert.Empty(t, calls, "deletion must not be attempted")
	})

	t.Run("no rows maps to ErrBlobReferenceNotFound", func(t *testing.T) {
		t.Parallel()
		stub := &stubDriver{
			decrementBlobRefCountFunc: func(_ context.Context, _ string, _ int64) (int, error) {
				return 0, sql.ErrNoRows
			},
		}
		core := newStubCore(nil, stub)

		_, _, err := core.DecrementBlobRefCount(t.Context(), "missing")

		require.ErrorIs(t, err, registry_domain.ErrBlobReferenceNotFound, "missing blob must map to sentinel")
	})

	t.Run("driver error is wrapped", func(t *testing.T) {
		t.Parallel()
		stub := &stubDriver{
			decrementBlobRefCountFunc: func(_ context.Context, _ string, _ int64) (int, error) {
				return 0, errors.New("db error")
			},
		}
		core := newStubCore(nil, stub)

		_, _, err := core.DecrementBlobRefCount(t.Context(), "blob/x")

		require.Error(t, err, "driver error must surface")
	})
}

func TestGetBlobRefCount(t *testing.T) {
	t.Parallel()

	t.Run("returns the count", func(t *testing.T) {
		t.Parallel()
		stub := &stubDriver{
			getBlobRefCountFunc: func(_ context.Context, _ string) (int, error) {
				return 7, nil
			},
		}
		core := newStubCore(nil, stub)

		count, err := core.GetBlobRefCount(t.Context(), "blob/x")

		require.NoError(t, err, "lookup must succeed")
		assert.Equal(t, 7, count, "count must pass through")
	})

	t.Run("no rows returns zero without error", func(t *testing.T) {
		t.Parallel()
		stub := &stubDriver{
			getBlobRefCountFunc: func(_ context.Context, _ string) (int, error) {
				return 0, sql.ErrNoRows
			},
		}
		core := newStubCore(nil, stub)

		count, err := core.GetBlobRefCount(t.Context(), "missing")

		require.NoError(t, err, "missing blob must not error")
		assert.Equal(t, 0, count, "missing blob must return zero")
	})

	t.Run("driver error is wrapped", func(t *testing.T) {
		t.Parallel()
		stub := &stubDriver{
			getBlobRefCountFunc: func(_ context.Context, _ string) (int, error) {
				return 0, errors.New("db error")
			},
		}
		core := newStubCore(nil, stub)

		_, err := core.GetBlobRefCount(t.Context(), "blob/x")

		require.Error(t, err, "driver error must surface")
	})
}

func TestPopGCHints(t *testing.T) {
	t.Parallel()

	t.Run("empty result returns empty slice", func(t *testing.T) {
		t.Parallel()
		var calls []string
		stub := &stubDriver{
			calls: &calls,
			popGCHintsFunc: func(_ context.Context, _ int) ([]GCHintRow, error) {
				return nil, nil
			},
		}
		core := newStubCore(newFakeDB(t), stub)

		hints, err := core.PopGCHints(t.Context(), 10)

		require.NoError(t, err, "empty pop must succeed")
		assert.Empty(t, hints, "no hints must yield an empty slice")
		assert.NotContains(t, calls, "DeleteGCHints", "empty pop must not delete")
	})

	t.Run("non-empty result converts and deletes", func(t *testing.T) {
		t.Parallel()
		var calls []string
		var deleted []int64
		stub := &stubDriver{
			calls:          &calls,
			deletedGCHints: &deleted,
			popGCHintsFunc: func(_ context.Context, _ int) ([]GCHintRow, error) {
				return []GCHintRow{
					{BackendID: "local", StorageKey: "blob/a", ID: 1},
					{BackendID: "s3", StorageKey: "blob/b", ID: 2},
				}, nil
			},
		}
		core := newStubCore(newFakeDB(t), stub)

		hints, err := core.PopGCHints(t.Context(), 10)

		require.NoError(t, err, "pop must succeed")
		require.Len(t, hints, 2, "all hints must be converted")
		assert.Contains(t, calls, "DeleteGCHints", "popped hints must be deleted")
		assert.Equal(t, []int64{1, 2}, deleted, "deleted IDs must match popped rows")
	})

	t.Run("non-positive limit uses the default", func(t *testing.T) {
		t.Parallel()
		var seenLimit int
		stub := &stubDriver{
			popGCHintLimit: &seenLimit,
			popGCHintsFunc: func(_ context.Context, _ int) ([]GCHintRow, error) {
				return nil, nil
			},
		}
		core := newStubCore(newFakeDB(t), stub)

		_, err := core.PopGCHints(t.Context(), 0)

		require.NoError(t, err, "pop must succeed")
		assert.Equal(t, defaultGCHintLimit, seenLimit, "non-positive limit must default")
	})

	t.Run("oversized limit is clamped", func(t *testing.T) {
		t.Parallel()
		var seenLimit int
		stub := &stubDriver{
			popGCHintLimit: &seenLimit,
			popGCHintsFunc: func(_ context.Context, _ int) ([]GCHintRow, error) {
				return nil, nil
			},
		}
		core := newStubCore(newFakeDB(t), stub)

		_, err := core.PopGCHints(t.Context(), maxGCHintLimit+1000)

		require.NoError(t, err, "pop must succeed")
		assert.Equal(t, maxGCHintLimit, seenLimit, "oversized limit must be clamped")
	})

	t.Run("driver error is wrapped", func(t *testing.T) {
		t.Parallel()
		stub := &stubDriver{
			popGCHintsFunc: func(_ context.Context, _ int) ([]GCHintRow, error) {
				return nil, errors.New("db error")
			},
		}
		core := newStubCore(newFakeDB(t), stub)

		_, err := core.PopGCHints(t.Context(), 10)

		require.Error(t, err, "driver error must surface")
	})
}

func TestAtomicUpdate(t *testing.T) {
	t.Parallel()

	t.Run("dispatches each action in order", func(t *testing.T) {
		t.Parallel()
		var calls []string
		stub := &stubDriver{calls: &calls}
		core := newStubCore(newFakeDB(t), stub)

		err := core.AtomicUpdate(t.Context(), []registry_dto.AtomicAction{
			{Type: registry_dto.ActionTypeDeleteArtefact, ArtefactID: "art-1"},
			{Type: registry_dto.ActionTypeAddGCHints, GCHints: []registry_dto.GCHint{{StorageKey: "blob/a"}}},
		})

		require.NoError(t, err, "atomic update must succeed")
		assert.Equal(t, []string{"DeleteArtefact", "AddGCHint"}, calls, "actions must run in order")
	})

	t.Run("action failure aborts the update", func(t *testing.T) {
		t.Parallel()
		stub := &stubDriver{
			deleteArtefactFunc: func(_ context.Context, _ string) error {
				return errors.New("delete failed")
			},
		}
		core := newStubCore(newFakeDB(t), stub)

		err := core.AtomicUpdate(t.Context(), []registry_dto.AtomicAction{
			{Type: registry_dto.ActionTypeDeleteArtefact, ArtefactID: "art-1"},
		})

		require.Error(t, err, "action failure must surface")
	})
}

func TestUpsertArtefactFanOut(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	duration := 2.5
	artefact := &registry_dto.ArtefactMeta{
		ID:         "art-full",
		SourcePath: "source/art-full.png",
		CreatedAt:  now,
		UpdatedAt:  now,
		Status:     registry_dto.VariantStatusReady,
		DesiredProfiles: []registry_dto.NamedProfile{
			{
				Name: "thumbnail",
				Profile: registry_dto.DesiredProfile{
					Priority:       registry_dto.PriorityNeed,
					CapabilityName: "resize",
				},
			},
		},
		ActualVariants: []registry_dto.Variant{
			{
				VariantID:        "source",
				StorageKey:       "blob/art-full",
				StorageBackendID: "local",
				MimeType:         "image/png",
				Status:           registry_dto.VariantStatusReady,
				SizeBytes:        2048,
				CreatedAt:        now,
				Chunks: []registry_dto.VariantChunk{
					{
						ChunkID:          "chunk-0",
						StorageKey:       "blob/art-full/0",
						StorageBackendID: "local",
						MimeType:         "image/png",
						SizeBytes:        1024,
						SequenceNumber:   0,
						CreatedAt:        now,
						DurationSeconds:  &duration,
					},
				},
			},
		},
	}
	artefact.ActualVariants[0].MetadataTags.SetByName("etag", "abc123")

	var calls []string
	stub := &stubDriver{calls: &calls}

	err := upsertArtefact(t.Context(), stub, artefact)

	require.NoError(t, err, "upsert fan-out must succeed")
	assert.Equal(t, []string{
		"UpsertArtefact",
		"DeleteVariantTagsForArtefact",
		"DeleteChunksForVariant",
		"DeleteVariantsForArtefact",
		"DeleteDesiredProfilesForArtefact",
		"InsertVariant",
		"InsertVariantTag",
		"InsertVariantChunk",
		"InsertDesiredProfile",
	}, calls, "upsert must fan out to the full write sequence")
}

func TestRunAtomicNestedIsRejected(t *testing.T) {
	t.Parallel()

	core := &core{sqlDB: newFakeDB(t), driver: &stubDriver{}, clock: clock.RealClock(), inTransaction: true}

	err := core.RunAtomic(t.Context(), func(_ context.Context, _ registry_domain.MetadataStore) error {
		return nil
	})

	require.Error(t, err, "nested transactions must be rejected")
}

func TestRunAtomicCommits(t *testing.T) {
	t.Parallel()

	core := newStubCore(newFakeDB(t), &stubDriver{})

	var received registry_domain.MetadataStore
	err := core.RunAtomic(t.Context(), func(_ context.Context, store registry_domain.MetadataStore) error {
		received = store
		return nil
	})

	require.NoError(t, err, "atomic block must commit")
	require.NotNil(t, received, "transactional store must be provided")
}

func TestRunAtomicRollsBackOnError(t *testing.T) {
	t.Parallel()

	core := newStubCore(newFakeDB(t), &stubDriver{})

	sentinel := errors.New("user error")
	err := core.RunAtomic(t.Context(), func(_ context.Context, _ registry_domain.MetadataStore) error {
		return sentinel
	})

	require.ErrorIs(t, err, sentinel, "user error must surface")
}

func TestRunInTransactionNotInitialised(t *testing.T) {
	t.Parallel()

	core := &core{driver: &stubDriver{}, clock: clock.RealClock()}

	err := core.runInTransaction(t.Context(), func(_ context.Context, _ Driver) error {
		return nil
	})

	require.ErrorIs(t, err, errDALNotInitialised, "missing sql.DB must be reported")
}

func TestHealthCheckNilDB(t *testing.T) {
	t.Parallel()

	core := &core{driver: &stubDriver{}, clock: clock.RealClock()}

	require.NoError(t, core.HealthCheck(t.Context()), "nil DB health check must be a no-op")
}

func TestClose(t *testing.T) {
	t.Parallel()

	core := newStubCore(nil, &stubDriver{})

	require.NoError(t, core.Close(), "Close must be a no-op")
}

func TestNew(t *testing.T) {
	t.Parallel()

	dal := New(newFakeDB(t), &stubDriver{})

	require.NotNil(t, dal, "New must return a configured DAL")
}

func TestIncrementBlobRefCountMockClock(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	var seen IncrementBlobRefCountParams
	stub := &stubDriver{
		incrementBlobRefCountFunc: func(_ context.Context, params IncrementBlobRefCountParams) (int, error) {
			seen = params
			return 1, nil
		},
	}
	core := &core{driver: stub, clock: clock.NewMockClock(fixedTime)}

	count, err := core.IncrementBlobRefCount(t.Context(), registry_domain.BlobReference{StorageKey: "blob/x"})

	require.NoError(t, err, "increment must succeed")
	assert.Equal(t, 1, count, "new count must be returned")
	assert.Equal(t, fixedTime.Unix(), seen.CreatedAt, "CreatedAt must equal the injected mock clock time")
	assert.Equal(t, fixedTime.Unix(), seen.LastReferencedAt, "LastReferencedAt must equal the injected mock clock time")
}
