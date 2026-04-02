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

package registry_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/registry/registry_adapters"
	registry_querier_adapter "piko.sh/piko/internal/registry/registry_dal/querier_adapter"
	registry_db "piko.sh/piko/internal/registry/registry_dal/querier_sqlite/db"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/tests/testutil"

	_ "modernc.org/sqlite"
)

func createBenchmarkCache(b *testing.B, maxWeight uint64) registry_domain.MetadataCache {
	b.Helper()
	otterCache, err := provider_otter.OtterProviderFactory(cache_dto.Options[string, *registry_dto.ArtefactMeta]{
		MaximumWeight: maxWeight,
		Weigher:       registry_adapters.ArtefactMetaWeigher,
	})
	require.NoError(b, err)
	return registry_adapters.NewMetadataCache(otterCache)
}

var blackholeServiceArtefact *registry_dto.ArtefactMeta
var blackholeServiceArtefacts []*registry_dto.ArtefactMeta

func setupRegistryServiceBenchmark(b *testing.B, numArtefacts, variantsPer, tagsPer, profilesPer int) (registry_domain.RegistryService, []string, func()) {
	b.Helper()

	store, artefactIDs, _, _, cleanupPersistent := setupServiceSQLiteBatchBenchmark(b, numArtefacts, variantsPer, tagsPer, profilesPer)

	cache := createBenchmarkCache(b, 512*1024*1024)

	service := registry_domain.NewRegistryService(store, nil, nil, cache)

	cleanup := func() {
		cleanupPersistent()
		_ = cache.Close(context.Background())
	}

	return service, artefactIDs, cleanup
}

func BenchmarkRegistryService_GetArtefact(b *testing.B) {
	b.Run("CacheMiss_Then_CacheHit", func(b *testing.B) {
		service, artefactIDs, cleanup := setupRegistryServiceBenchmark(b, 1, 10, 10, 8)
		defer cleanup()
		artefactID := artefactIDs[0]
		ctx := context.Background()

		_, err := service.GetArtefact(ctx, artefactID)
		require.NoError(b, err)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			artefact, err := service.GetArtefact(ctx, artefactID)
			if err != nil {
				b.Fatalf("GetArtefact failed: %v", err)
			}
			blackholeServiceArtefact = artefact
		}
	})

	b.Run("AlwaysMiss_FromLargeDB", func(b *testing.B) {
		store, _, dbConn, querier, cleanupPersistent := setupServiceSQLiteBatchBenchmark(b, 0, 0, 0, 0)
		defer cleanupPersistent()

		for i := range 1000 {
			seedServiceBenchmarkDataWithID(b, dbConn, querier, fmt.Sprintf("other_artefact_%d", i), 2, 2, 2)
		}
		targetArtefactID := seedServiceBenchmarkDataWithID(b, dbConn, querier, "bench/test-artefact.css", 4, 5, 4)

		cache := createBenchmarkCache(b, 512*1024*1024)
		defer func() { _ = cache.Close(context.Background()) }()

		service := registry_domain.NewRegistryService(store, nil, nil, cache)
		ctx := context.Background()

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {

			cache.Delete(ctx, targetArtefactID)

			artefact, err := service.GetArtefact(ctx, targetArtefactID)
			if err != nil {
				b.Fatalf("GetArtefact failed: %v", err)
			}
			blackholeServiceArtefact = artefact
		}
	})
}

func BenchmarkRegistryService_GetMultipleArtefacts(b *testing.B) {
	b.Run("LargeBatch_AlwaysMiss", func(b *testing.B) {
		store, targetArtefactIDs, _, _, cleanupPersistent := setupServiceSQLiteBatchBenchmark(b, 100, 2, 2, 2)
		defer cleanupPersistent()

		cache := createBenchmarkCache(b, 512*1024*1024)
		defer func() { _ = cache.Close(context.Background()) }()

		service := registry_domain.NewRegistryService(store, nil, nil, cache)
		ctx := context.Background()

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {

			for _, id := range targetArtefactIDs {
				cache.Delete(ctx, id)
			}

			artefacts, err := service.GetMultipleArtefacts(ctx, targetArtefactIDs)
			if err != nil {
				b.Fatalf("GetMultipleArtefacts failed: %v", err)
			}
			if len(artefacts) != 100 {
				b.Fatalf("Expected 100 artefacts, got %d", len(artefacts))
			}
			blackholeServiceArtefacts = artefacts
		}
	})

	b.Run("LargeBatch_AllHits", func(b *testing.B) {
		service, targetArtefactIDs, cleanup := setupRegistryServiceBenchmark(b, 100, 2, 2, 2)
		defer cleanup()
		ctx := context.Background()

		_, err := service.GetMultipleArtefacts(ctx, targetArtefactIDs)
		require.NoError(b, err)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			artefacts, err := service.GetMultipleArtefacts(ctx, targetArtefactIDs)
			if err != nil {
				b.Fatalf("GetMultipleArtefacts failed: %v", err)
			}
			if len(artefacts) != 100 {
				b.Fatalf("Expected 100 artefacts, got %d", len(artefacts))
			}
			blackholeServiceArtefacts = artefacts
		}
	})
}

func newServiceTestDB(t testing.TB, path string) *sql.DB {
	t.Helper()
	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(10000)&_pragma=foreign_keys(ON)&_pragma=cache_size(-20000)&_pragma=temp_store(MEMORY)&_pragma=mmap_size(67108864)",
		path,
	)

	conn, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)

	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxIdleTime(5 * time.Minute)

	err = conn.Ping()
	require.NoError(t, err)

	return conn
}

func setupServiceSQLiteBatchBenchmark(b *testing.B, numArtefacts, variantsPer, tagsPer, profilesPer int) (registry_domain.MetadataStore, []string, *sql.DB, *registry_db.Queries, func()) {
	b.Helper()
	tempDir, err := os.MkdirTemp("", "piko-bench-batch-")
	require.NoError(b, err)
	dbPath := filepath.Join(tempDir, "metadata.db")

	err = testutil.RunRegistryMigrations(dbPath)
	require.NoError(b, err)

	dbConn := newServiceTestDB(b, dbPath)
	querier := registry_db.New(dbConn)

	artefactIDs := make([]string, numArtefacts)
	for i := range numArtefacts {
		artefactID := fmt.Sprintf("bench/artefact-%d.css", i)
		seedServiceBenchmarkDataWithID(b, dbConn, querier, artefactID, variantsPer, tagsPer, profilesPer)
		artefactIDs[i] = artefactID
	}

	store := registry_querier_adapter.NewDAL(dbConn)

	cleanup := func() {
		_ = dbConn.Close()
		_ = os.RemoveAll(tempDir)
	}
	return store, artefactIDs, dbConn, querier, cleanup
}

func seedServiceBenchmarkDataWithID(t testing.TB, dbConn *sql.DB, q *registry_db.Queries, artefactID string, numVariants, numTagsPerVariant, numProfiles int) string {
	t.Helper()
	now := time.Now().UTC()
	tx, err := dbConn.Begin()
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	qtx := q.WithTx(tx)
	err = qtx.UpsertArtefact(context.Background(), registry_db.UpsertArtefactParams{
		P1: artefactID,
		P2: "source/" + artefactID,
		P3: int32(now.Unix()),
		P4: int32(now.Unix()),
	})
	require.NoError(t, err)
	for i := range numVariants {
		variantID := fmt.Sprintf("variant_%d", i)
		err := qtx.InsertVariant(context.Background(), registry_db.InsertVariantParams{
			P1: artefactID,
			P2: variantID,
			P3: uuid.NewString(),
			P4: "local_disk_cache",
			P5: "text/css",
			P6: 1024,
			P7: string(registry_dto.VariantStatusReady),
			P8: int32(now.Unix()),
		})
		require.NoError(t, err)
		for j := range numTagsPerVariant {
			err := qtx.InsertVariantTag(context.Background(), registry_db.InsertVariantTagParams{
				P1: artefactID,
				P2: variantID,
				P3: fmt.Sprintf("tag_key_%d", j),
				P4: fmt.Sprintf("tag_value_%d", j),
			})
			require.NoError(t, err)
		}
	}
	for i := range numProfiles {
		err := qtx.InsertDesiredProfile(context.Background(), registry_db.InsertDesiredProfileParams{
			P1: artefactID,
			P2: fmt.Sprintf("profile_%d", i),
			P3: "capability.minify",
			P4: string(registry_dto.PriorityWant),
			P5: "{}",
			P6: "{}",
			P7: "[\"source\"]",
		})
		require.NoError(t, err)
	}
	err = tx.Commit()
	require.NoError(t, err)
	return artefactID
}
