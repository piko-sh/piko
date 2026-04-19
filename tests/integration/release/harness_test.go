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

package release_test

import (
	"database/sql"
	"embed"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"

	"piko.sh/piko/internal/querier/querier_adapters/migration_sql"
	"piko.sh/piko/internal/querier/querier_domain"
	registry_otter "piko.sh/piko/internal/registry/registry_dal/otter"
	registry_querier_postgres "piko.sh/piko/internal/registry/registry_dal/querier_postgres"
	registry_querier_sqlite "piko.sh/piko/internal/registry/registry_dal/querier_sqlite"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/wdk/db/db_schema_registry_postgres"
	"piko.sh/piko/wdk/db/db_schema_registry_sqlite"
)

var (
	seedTime = time.Unix(1_700_000_000, 0).UTC()
	postgresSchemaCounter atomic.Int64
)

type backendCase struct {
	make func(t *testing.T) registry_domain.MetadataStore
	name string
}

func forEachPublisherBackend(t *testing.T, fn func(t *testing.T, store registry_domain.MetadataStore)) {
	t.Helper()
	t.Run("otter", func(t *testing.T) {
		t.Skip("otter does not implement ReleasePublisher")
	})
	cases := []backendCase{
		{name: "sqlite", make: newSQLiteStore},
		{name: "postgres", make: newPostgresStore},
	}
	for _, backend := range cases {
		t.Run(backend.name, func(t *testing.T) {
			if backend.name == "postgres" && postgresConnectionString == "" {
				t.Skipf("Postgres unavailable, so this backend is untested: %s "+
					"(set %s, or %s to make this fatal)",
					postgresUnavailableReason, postgresDSNEnv, postgresRequiredEnv)
			}
			fn(t, backend.make(t))
		})
	}
}

func newOtterStore(t *testing.T) registry_domain.MetadataStore {
	t.Helper()
	dal, err := registry_otter.NewOtterDAL(registry_otter.Config{})
	require.NoError(t, err, "creating otter registry DAL")
	t.Cleanup(func() { _ = dal.Close() })
	return dal
}

func newSQLiteStore(t *testing.T) registry_domain.MetadataStore {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "registry.db") +
		"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(10000)"
	database, err := sql.Open("sqlite", dsn)
	require.NoError(t, err, "opening SQLite database")
	database.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = database.Close() })

	runRegistryMigrations(t, database, migration_sql.SQLiteDialect(), db_schema_registry_sqlite.Migrations)
	return registry_querier_sqlite.New(database)
}

func newPostgresStore(t *testing.T) registry_domain.MetadataStore {
	t.Helper()
	schema := fmt.Sprintf("rel_test_%d_%d", os.Getpid(), postgresSchemaCounter.Add(1))
	database, err := sql.Open("pgx", postgresDSNWithSearchPath(postgresConnectionString, schema))
	require.NoError(t, err, "opening Postgres database")
	database.SetMaxOpenConns(1)

	_, err = database.Exec("CREATE SCHEMA " + schema)
	require.NoError(t, err, "creating test schema %q", schema)
	t.Cleanup(func() {
		_, _ = database.Exec("DROP SCHEMA " + schema + " CASCADE")
		_ = database.Close()
	})

	runRegistryMigrations(t, database, migration_sql.PostgresDialect(), db_schema_registry_postgres.Migrations)
	return registry_querier_postgres.New(database)
}

func postgresDSNWithSearchPath(dsn, schema string) string {
	optionsValue := "-csearch_path=" + schema
	if !strings.Contains(dsn, "://") {
		return dsn + " options='" + optionsValue + "'"
	}
	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}
	return dsn + separator + "options=" + url.QueryEscape(optionsValue)
}

func runRegistryMigrations(t *testing.T, database *sql.DB, dialect migration_sql.DialectConfig, migrations embed.FS) {
	t.Helper()
	executor := migration_sql.NewExecutor(database, dialect)
	reader := migration_sql.NewFSFileReader(migrations)
	service := querier_domain.NewMigrationService(executor, reader, "migrations")
	_, err := service.Up(t.Context())
	require.NoError(t, err, "running registry migrations")
}

func seedArtefact(artefactID, variantID, storageKey string) *registry_dto.ArtefactMeta {
	return &registry_dto.ArtefactMeta{
		ID:         artefactID,
		SourcePath: "pages/" + artefactID,
		CreatedAt:  seedTime,
		UpdatedAt:  seedTime,
		ActualVariants: []registry_dto.Variant{{
			VariantID:        variantID,
			StorageKey:       storageKey,
			StorageBackendID: "overlay",
			MimeType:         "image/webp",
			SizeBytes:        128,
			ContentHash:      "hash-" + storageKey,
			Status:           registry_dto.VariantStatusReady,
			CreatedAt:        seedTime,
		}},
	}
}

func mustPublish(
	t *testing.T,
	store registry_domain.MetadataStore,
	releaseID string,
	at time.Time,
	artefacts ...*registry_dto.ArtefactMeta,
) string {
	t.Helper()
	digest := registry_domain.SeedDigest(artefacts)
	outcome, err := registry_domain.PublishRelease(t.Context(), store, releaseID, digest, artefacts, at)
	require.NoError(t, err, "publishing release %q", releaseID)
	require.Equal(t, registry_domain.PublishOutcomePublished, outcome,
		"release %q must publish as the first claimant", releaseID)
	return digest
}

func requirePublisher(t *testing.T, store registry_domain.MetadataStore) registry_domain.ReleasePublisher {
	t.Helper()
	publisher, ok := store.(registry_domain.ReleasePublisher)
	require.True(t, ok, "this backend must implement registry_domain.ReleasePublisher")
	return publisher
}

func releaseLease(t *testing.T, store registry_domain.MetadataStore, releaseID string) (registry_domain.ReleaseLease, bool) {
	t.Helper()
	lease, exists, err := requirePublisher(t, store).GetRelease(t.Context(), releaseID)
	require.NoError(t, err, "reading release lease %q", releaseID)
	return lease, exists
}

func requireStorageKeyResolves(t *testing.T, store registry_domain.MetadataStore, storageKey, wantArtefactID string) {
	t.Helper()
	artefact, err := store.FindArtefactByVariantStorageKey(t.Context(), storageKey)
	require.NoError(t, err, "storage key %q must resolve", storageKey)
	require.NotNil(t, artefact, "storage key %q must resolve to an artefact", storageKey)
	require.Equal(t, wantArtefactID, artefact.ID, "storage key %q must belong to artefact %q", storageKey, wantArtefactID)
}

func requireStorageKeyGone(t *testing.T, store registry_domain.MetadataStore, storageKey string) {
	t.Helper()
	artefact, err := store.FindArtefactByVariantStorageKey(t.Context(), storageKey)
	require.ErrorIs(t, err, registry_domain.ErrArtefactNotFound, "storage key %q must no longer resolve", storageKey)
	require.Nil(t, artefact, "no artefact must be returned for the retired storage key %q", storageKey)
}

func blobRefCount(t *testing.T, store registry_domain.MetadataStore, storageKey string) int {
	t.Helper()
	count, err := store.GetBlobRefCount(t.Context(), storageKey)
	require.NoError(t, err, "reading blob reference count for %q", storageKey)
	return count
}
