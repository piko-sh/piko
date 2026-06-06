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

package db_engine_cockroachdb_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/db/db_engine_cockroachdb"
)

func TestCockroachDB_CrdbInternalFunctions_Resolve(t *testing.T) {
	t.Parallel()

	const queryContent = "-- piko.query(name: NodeInfo, command: one)\n" +
		"SELECT crdb_internal.node_id() AS node, crdb_internal.cluster_id() AS cluster;\n"

	caseDirectory := writeResolutionFixture(t, "-- no schema needed\n", queryContent)

	queries, diagnostics := analyseResolutionFixture(t, caseDirectory)

	for _, diagnostic := range diagnostics {
		assert.NotEqual(t, querier_dto.CodeUnknownFunction, diagnostic.Code,
			"crdb_internal builtins must resolve; got unknown-function diagnostic: %s", diagnostic.Message)
	}

	require.Len(t, queries, 1, "exactly one analysed query expected")
	columns := queries[0].OutputColumns
	require.Len(t, columns, 2, "two output columns expected")

	nodeColumn := findColumn(t, columns, "node")
	assert.Equal(t, querier_dto.TypeCategoryInteger, nodeColumn.SQLType.Category,
		"crdb_internal.node_id() should resolve to an integer column, not Unknown/any")
	assert.Equal(t, "int8", nodeColumn.SQLType.EngineName,
		"crdb_internal.node_id() should resolve to int8")

	clusterColumn := findColumn(t, columns, "cluster")
	assert.Equal(t, querier_dto.TypeCategoryUUID, clusterColumn.SQLType.Category,
		"crdb_internal.cluster_id() should resolve to a uuid column, not Unknown/any")
	assert.Equal(t, "uuid", clusterColumn.SQLType.EngineName,
		"crdb_internal.cluster_id() should resolve to uuid")
}

func writeResolutionFixture(t *testing.T, migrationContent string, queryContent string) string {
	t.Helper()

	caseDirectory := t.TempDir()
	migrationDirectory := filepath.Join(caseDirectory, "migrations")
	queryDirectory := filepath.Join(caseDirectory, "queries")
	require.NoError(t, os.MkdirAll(migrationDirectory, 0o755))
	require.NoError(t, os.MkdirAll(queryDirectory, 0o755))

	require.NoError(t, os.WriteFile(
		filepath.Join(migrationDirectory, "001_schema.up.sql"), []byte(migrationContent), 0o644))
	require.NoError(t, os.WriteFile(
		filepath.Join(queryDirectory, "queries.sql"), []byte(queryContent), 0o644))

	return caseDirectory
}

func analyseResolutionFixture(
	t *testing.T,
	caseDirectory string,
) ([]*querier_dto.AnalysedQuery, []querier_dto.SourceError) {
	t.Helper()

	engine := db_engine_cockroachdb.NewCockroachDBEngine()
	emitter := &recordingCodeEmitter{}

	service, serviceError := querier_domain.NewQuerierService(querier_domain.QuerierPorts{
		Engine:     engine,
		Emitter:    emitter,
		FileReader: &realFileReader{},
	})
	require.NoError(t, serviceError)

	result, generateError := service.GenerateDatabase(context.Background(), "test", &querier_dto.DatabaseConfig{
		MigrationDirectory: filepath.Join(caseDirectory, "migrations"),
		QueryDirectory:     filepath.Join(caseDirectory, "queries"),
	})
	require.NoError(t, generateError)
	require.NotNil(t, result)

	return emitter.queries, result.Diagnostics
}

func findColumn(t *testing.T, columns []querier_dto.OutputColumn, name string) querier_dto.OutputColumn {
	t.Helper()

	for _, column := range columns {
		if column.Name == name {
			return column
		}
	}

	t.Fatalf("output column %q not found", name)
	return querier_dto.OutputColumn{}
}
