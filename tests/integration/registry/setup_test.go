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

package registry_test_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/registry/registry_adapters"
	registry_querier_adapter "piko.sh/piko/internal/registry/registry_dal/querier_adapter"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/storage/storage_adapters/provider_disk"
	"piko.sh/piko/internal/storage/storage_adapters/registry_blob_adapter"
	"piko.sh/piko/internal/testutil/leakcheck"
	"piko.sh/piko/tests/testutil"
	"piko.sh/piko/wdk/logger"

	_ "github.com/mattn/go-sqlite3"
)

type spyMetadataStore struct {
	mock.Mock
	realStore  registry_domain.MetadataStore
	failureMap map[string]error
	parentSpy  *spyMetadataStore
}

func (m *spyMetadataStore) mockTarget() *spyMetadataStore {
	if m.parentSpy != nil {
		return m.parentSpy
	}
	return m
}

func newSpyMetadataStore(realStore registry_domain.MetadataStore) *spyMetadataStore {
	return &spyMetadataStore{
		realStore:  realStore,
		failureMap: make(map[string]error),
	}
}

func (m *spyMetadataStore) SetFailure(methodName string, err error) {
	m.failureMap[methodName] = err
}

func (m *spyMetadataStore) ClearFailures() {
	m.failureMap = make(map[string]error)
}

func (m *spyMetadataStore) GetArtefact(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
	if m.parentSpy == nil {
		m.Called(ctx, artefactID)
	}
	return m.realStore.GetArtefact(ctx, artefactID)
}

func (m *spyMetadataStore) GetMultipleArtefacts(ctx context.Context, artefactIDs []string) ([]*registry_dto.ArtefactMeta, error) {
	if m.parentSpy == nil {
		m.Called(ctx, artefactIDs)
	}
	return m.realStore.GetMultipleArtefacts(ctx, artefactIDs)
}

func (m *spyMetadataStore) AtomicUpdate(ctx context.Context, actions []registry_dto.AtomicAction) error {
	m.mockTarget().Called(ctx, actions)

	if err, ok := m.failureMap["AtomicUpdate"]; ok {
		return err
	}
	return m.realStore.AtomicUpdate(ctx, actions)
}

func (m *spyMetadataStore) ListAllArtefactIDs(ctx context.Context) ([]string, error) {
	if m.parentSpy == nil {
		m.Called(ctx)
	}
	return m.realStore.ListAllArtefactIDs(ctx)
}
func (m *spyMetadataStore) SearchArtefacts(ctx context.Context, query registry_domain.SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
	if m.parentSpy == nil {
		m.Called(ctx, query)
	}
	return m.realStore.SearchArtefacts(ctx, query)
}
func (m *spyMetadataStore) FindArtefactByVariantStorageKey(ctx context.Context, storageKey string) (*registry_dto.ArtefactMeta, error) {
	if m.parentSpy == nil {
		m.Called(ctx, storageKey)
	}
	return m.realStore.FindArtefactByVariantStorageKey(ctx, storageKey)
}
func (m *spyMetadataStore) PopGCHints(ctx context.Context, limit int) ([]registry_dto.GCHint, error) {
	if m.parentSpy == nil {
		m.Called(ctx, limit)
	}
	return m.realStore.PopGCHints(ctx, limit)
}

func (m *spyMetadataStore) SearchArtefactsByTagValues(ctx context.Context, tagKey string, tagValues []string) ([]*registry_dto.ArtefactMeta, error) {
	if m.parentSpy == nil {
		m.Called(ctx, tagKey, tagValues)
	}
	return m.realStore.SearchArtefactsByTagValues(ctx, tagKey, tagValues)
}

func (m *spyMetadataStore) IncrementBlobRefCount(ctx context.Context, blob registry_domain.BlobReference) (int, error) {
	m.mockTarget().Called(ctx, blob)

	if err, ok := m.failureMap["IncrementBlobRefCount"]; ok {
		return 0, err
	}
	return m.realStore.IncrementBlobRefCount(ctx, blob)
}

func (m *spyMetadataStore) DecrementBlobRefCount(ctx context.Context, storageKey string) (int, bool, error) {
	m.mockTarget().Called(ctx, storageKey)

	if err, ok := m.failureMap["DecrementBlobRefCount"]; ok {
		return 0, false, err
	}
	return m.realStore.DecrementBlobRefCount(ctx, storageKey)
}

func (m *spyMetadataStore) GetBlobRefCount(ctx context.Context, storageKey string) (int, error) {
	if m.parentSpy == nil {
		m.Called(ctx, storageKey)
	}
	return m.realStore.GetBlobRefCount(ctx, storageKey)
}

func (m *spyMetadataStore) RunAtomic(ctx context.Context, fn func(ctx context.Context, transactionStore registry_domain.MetadataStore) error) error {
	return m.realStore.RunAtomic(ctx, func(ctx context.Context, transactionStore registry_domain.MetadataStore) error {
		txSpy := &spyMetadataStore{
			realStore:  transactionStore,
			failureMap: m.failureMap,
			parentSpy:  m,
		}
		return fn(ctx, txSpy)
	})
}

func (m *spyMetadataStore) Close() error {
	return m.realStore.Close()
}

type testFixture struct {
	service   registry_domain.RegistryService
	spyStore  *spyMetadataStore
	blobStore registry_domain.BlobStore
	dbConn    *sql.DB
	cleanup   func()
}

func setupIntegrationTest(t *testing.T, withCache bool, fixtureFile string) testFixture {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "registry-integration-test-")
	require.NoError(t, err)

	dbPath := filepath.Join(tempDir, "metadata.db")
	err = testutil.RunRegistryMigrations(dbPath)
	require.NoError(t, err)

	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=10000", dbPath)
	dbConn, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	dbConn.SetMaxOpenConns(1)

	if fixtureFile != "" {
		loadFixture(t, dbConn, fixtureFile)
	}

	realStore := registry_querier_adapter.NewDAL(dbConn)
	spyStore := newSpyMetadataStore(realStore)

	var cache registry_domain.MetadataCache
	if withCache {
		otterCache, cacheErr := provider_otter.OtterProviderFactory(cache_dto.Options[string, *registry_dto.ArtefactMeta]{
			MaximumWeight: 10 * 1024 * 1024,
			Weigher:       registry_adapters.ArtefactMetaWeigher,
		})
		require.NoError(t, cacheErr)
		cache = registry_adapters.NewMetadataCache(otterCache)
	}

	blobDir := filepath.Join(tempDir, "blobs")
	diskProvider, err := provider_disk.NewDiskProvider(provider_disk.Config{
		BaseDirectory: blobDir,
	})
	require.NoError(t, err)
	blobStore, err := registry_blob_adapter.NewBlobStoreAdapter(registry_blob_adapter.Config{
		Provider:   diskProvider,
		Repository: "",
	})
	require.NoError(t, err)
	blobStores := map[string]registry_domain.BlobStore{"test_disk": blobStore}

	service := registry_domain.NewRegistryService(spyStore, blobStores, nil, cache)

	cleanup := func() {
		if cache != nil {
			_ = cache.Close(context.Background())
		}
		_ = dbConn.Close()
		_ = os.RemoveAll(tempDir)
	}

	return testFixture{
		service:   service,
		spyStore:  spyStore,
		blobStore: blobStore,
		dbConn:    dbConn,
		cleanup:   cleanup,
	}
}

func loadFixture(t *testing.T, db *sql.DB, fixtureName string) {
	t.Helper()

	switch fixtureName {
	case "base_scenario.sql":
		loadBaseScenarioFixture(t, db)
	case "complex_dependencies.sql":
		loadComplexDependenciesFixture(t, db)
	default:
		t.Fatalf("unknown fixture: %s", fixtureName)
	}
}

func TestMain(m *testing.M) {
	logger.AddPrettyOutput()
	leakcheck.VerifyTestMain(m)
}
