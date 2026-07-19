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
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	registry_querier_sqlite "piko.sh/piko/internal/registry/registry_dal/querier_sqlite"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/tests/testutil"

	_ "github.com/mattn/go-sqlite3"
)

func TestSQLiteConformance(t *testing.T) {
	RunStoreSuite(t, Config{
		NewStore:                           newSQLiteConformanceStore,
		SupportsArtefactLocker:             true,
		SupportsRegistryInspector:          true,
		SupportsRollback:                   true,
		SupportsNestedTransactionRejection: true,
		SupportsGCHints:                    true,
		SupportsSRIHashPersistence:         true,
		SupportsReleasePublisher:           true,
	})
}

func newSQLiteConformanceStore(t *testing.T) registry_domain.MetadataStore {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "conformance.db")
	require.NoError(t, testutil.RunRegistryMigrations(dbPath), "running registry migrations")

	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=10000&_foreign_keys=true", dbPath)
	database, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err, "opening SQLite database")
	database.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = database.Close() })

	return registry_querier_sqlite.New(database)
}
