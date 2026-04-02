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
	_ "modernc.org/sqlite"
	registry_querier_adapter "piko.sh/piko/internal/registry/registry_dal/querier_adapter"
	registry_db "piko.sh/piko/internal/registry/registry_dal/querier_sqlite/db"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/registry/registry_schema"
	"piko.sh/piko/tests/testutil"
)

var blackholeArtefact *registry_dto.ArtefactMeta
var blackholeArtefacts []*registry_dto.ArtefactMeta

func newTestDB(t testing.TB, path string) *sql.DB {
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

func setupSQLiteBenchmark(t testing.TB, numVariants, numTagsPerVariant, numProfiles int) (registry_domain.MetadataStore, string, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "piko-bench-registry-")
	require.NoError(t, err)

	sqlitePath := filepath.Join(tempDir, "metadata.db")

	err = testutil.RunRegistryMigrations(sqlitePath)
	require.NoError(t, err)

	dbConn := newTestDB(t, sqlitePath)
	querier := registry_db.New(dbConn)

	store := registry_querier_adapter.NewDAL(dbConn)

	artefactID := seedBenchmarkData(t, dbConn, querier, numVariants, numTagsPerVariant, numProfiles)

	cleanup := func() {
		_ = dbConn.Close()
		_ = os.RemoveAll(tempDir)
	}

	return store, artefactID, cleanup
}

func setupSQLiteBatchBenchmark(b *testing.B, numArtefacts, variantsPer, tagsPer, profilesPer int) (registry_domain.MetadataStore, []string, *sql.DB, *registry_db.Queries, func()) {
	b.Helper()
	tempDir, err := os.MkdirTemp("", "piko-bench-batch-")
	require.NoError(b, err)
	dbPath := filepath.Join(tempDir, "metadata.db")

	err = testutil.RunRegistryMigrations(dbPath)
	require.NoError(b, err)

	dbConn := newTestDB(b, dbPath)
	querier := registry_db.New(dbConn)

	artefactIDs := make([]string, numArtefacts)
	for i := range numArtefacts {
		artefactID := fmt.Sprintf("bench/artefact-%d.css", i)
		seedBenchmarkDataWithID(b, dbConn, querier, artefactID, variantsPer, tagsPer, profilesPer)
		artefactIDs[i] = artefactID
	}

	store := registry_querier_adapter.NewDAL(dbConn)

	cleanup := func() {
		_ = dbConn.Close()
		_ = os.RemoveAll(tempDir)
	}
	return store, artefactIDs, dbConn, querier, cleanup
}

func seedBenchmarkData(t testing.TB, dbConn *sql.DB, q *registry_db.Queries, numVariants, numTagsPerVariant, numProfiles int) string {
	t.Helper()
	return seedBenchmarkDataWithID(t, dbConn, q, "bench/test-artefact.css", numVariants, numTagsPerVariant, numProfiles)
}

func BenchmarkSQLiteMetadataStore_GetArtefact(b *testing.B) {
	b.Run("SimpleArtefact_4Variants_5TagsEach_4Profiles", func(b *testing.B) {
		store, artefactID, cleanup := setupSQLiteBenchmark(b, 4, 5, 4)
		defer cleanup()
		ctx := context.Background()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			artefact, err := store.GetArtefact(ctx, artefactID)
			if err != nil {
				b.Fatalf("GetArtefact failed: %v", err)
			}
			blackholeArtefact = artefact
		}
	})

	b.Run("ComplexArtefact_10Variants_10TagsEach_8Profiles", func(b *testing.B) {
		store, artefactID, cleanup := setupSQLiteBenchmark(b, 10, 10, 8)
		defer cleanup()
		ctx := context.Background()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			artefact, err := store.GetArtefact(ctx, artefactID)
			if err != nil {
				b.Fatalf("GetArtefact failed: %v", err)
			}
			blackholeArtefact = artefact
		}
	})

	b.Run("SimpleArtefact_With1000OtherArtefacts", func(b *testing.B) {
		store, targetArtefactID, dbConn, querier, cleanup := setupSQLiteBatchBenchmark(b, 1, 4, 5, 4)
		defer cleanup()
		for i := range 1000 {
			seedBenchmarkDataWithID(b, dbConn, querier, fmt.Sprintf("other_artefact_%d", i), 2, 2, 2)
		}
		ctx := context.Background()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			artefact, err := store.GetArtefact(ctx, targetArtefactID[0])
			if err != nil {
				b.Fatalf("GetArtefact failed: %v", err)
			}
			blackholeArtefact = artefact
		}
	})
}

func BenchmarkSQLiteMetadataStore_GetMultipleArtefacts(b *testing.B) {
	b.Run("SmallBatch_ComplexArtefacts", func(b *testing.B) {
		store, targetArtefactIDs, _, _, cleanup := setupSQLiteBatchBenchmark(b, 10, 10, 10, 8)
		defer cleanup()
		ctx := context.Background()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			artefacts, err := store.GetMultipleArtefacts(ctx, targetArtefactIDs)
			if err != nil {
				b.Fatalf("GetMultipleArtefacts failed: %v", err)
			}
			if len(artefacts) != 10 {
				b.Fatalf("Expected 10 artefacts, got %d", len(artefacts))
			}
			blackholeArtefacts = artefacts
		}
	})

	b.Run("LargeBatch_SimpleArtefacts", func(b *testing.B) {
		store, targetArtefactIDs, _, _, cleanup := setupSQLiteBatchBenchmark(b, 100, 2, 2, 2)
		defer cleanup()
		ctx := context.Background()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			artefacts, err := store.GetMultipleArtefacts(ctx, targetArtefactIDs)
			if err != nil {
				b.Fatalf("GetMultipleArtefacts failed: %v", err)
			}
			if len(artefacts) != 100 {
				b.Fatalf("Expected 100 artefacts, got %d", len(artefacts))
			}
			blackholeArtefacts = artefacts
		}
	})

	b.Run("SmallBatch_FromLargeDB", func(b *testing.B) {
		store, targetArtefactIDs, dbConn, querier, cleanup := setupSQLiteBatchBenchmark(b, 10, 4, 5, 4)
		defer cleanup()
		for i := range 1000 {
			seedBenchmarkDataWithID(b, dbConn, querier, fmt.Sprintf("other_artefact_%d", i), 2, 2, 2)
		}
		ctx := context.Background()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			artefacts, err := store.GetMultipleArtefacts(ctx, targetArtefactIDs)
			if err != nil {
				b.Fatalf("GetMultipleArtefacts failed: %v", err)
			}
			if len(artefacts) != 10 {
				b.Fatalf("Expected 10 artefacts, got %d", len(artefacts))
			}
			blackholeArtefacts = artefacts
		}
	})
}

func seedBenchmarkDataWithID(t testing.TB, dbConn *sql.DB, q *registry_db.Queries, artefactID string, numVariants, numTagsPerVariant, numProfiles int) string {
	t.Helper()
	now := time.Now().UTC()

	art := &registry_dto.ArtefactMeta{
		ID:              artefactID,
		SourcePath:      "source/" + artefactID,
		CreatedAt:       now,
		UpdatedAt:       now,
		ActualVariants:  make([]registry_dto.Variant, numVariants),
		DesiredProfiles: make([]registry_dto.NamedProfile, 0, numProfiles),
	}

	for i := range numVariants {
		var tags registry_dto.Tags
		for j := range numTagsPerVariant {
			tags.SetByName(fmt.Sprintf("tag_key_%d", j), fmt.Sprintf("tag_value_%d", j))
		}
		art.ActualVariants[i] = registry_dto.Variant{
			VariantID:        fmt.Sprintf("variant_%d", i),
			StorageKey:       uuid.NewString(),
			StorageBackendID: "local_disk_cache",
			MimeType:         "text/css",
			SizeBytes:        1024,
			Status:           registry_dto.VariantStatusReady,
			CreatedAt:        now,
			MetadataTags:     tags,
		}
	}

	for i := range numProfiles {
		var deps registry_dto.Dependencies
		deps.Add("source")
		art.DesiredProfiles = append(art.DesiredProfiles, registry_dto.NamedProfile{
			Name: fmt.Sprintf("profile_%d", i),
			Profile: registry_dto.DesiredProfile{
				CapabilityName: "capability.minify",
				Priority:       registry_dto.PriorityWant,
				DependsOn:      deps,
			},
		})
	}

	fbsData := registry_schema.BuildArtefactMeta(art)

	tx, err := dbConn.Begin()
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	qtx := q.WithTx(tx)

	err = qtx.UpsertArtefact(context.Background(), registry_db.UpsertArtefactParams{
		P1: art.ID,
		P2: art.SourcePath,
		P3: int32(art.CreatedAt.Unix()),
		P4: int32(art.UpdatedAt.Unix()),
		P5: fbsData,
	})
	require.NoError(t, err)

	for i := range art.ActualVariants {
		v := &art.ActualVariants[i]
		err := qtx.InsertVariant(context.Background(), registry_db.InsertVariantParams{
			P1: art.ID,
			P2: v.VariantID,
			P3: v.StorageKey,
			P4: v.StorageBackendID,
			P5: v.MimeType,
			P6: int32(v.SizeBytes),
			P7: string(v.Status),
			P8: int32(v.CreatedAt.Unix()),
		})
		require.NoError(t, err)
		for key, value := range v.MetadataTags.All() {
			err := qtx.InsertVariantTag(context.Background(), registry_db.InsertVariantTagParams{
				P1: art.ID,
				P2: v.VariantID,
				P3: key,
				P4: value,
			})
			require.NoError(t, err)
		}
	}

	for i := range art.DesiredProfiles {
		np := &art.DesiredProfiles[i]
		err := qtx.InsertDesiredProfile(context.Background(), registry_db.InsertDesiredProfileParams{
			P1: art.ID,
			P2: np.Name,
			P3: np.Profile.CapabilityName,
			P4: string(np.Profile.Priority),
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
