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

package db_engine_cockroachdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestNewCockroachDBEngine(t *testing.T) {
	t.Parallel()

	engine := NewCockroachDBEngine()

	require.NotNil(t, engine, "engine should not be nil")
	assert.Equal(t, "cockroachdb", engine.Dialect(), "dialect should be cockroachdb")
}

func TestCockroachDB_NormaliseTypeName_ExtraTypes(t *testing.T) {
	t.Parallel()

	engine := NewCockroachDBEngine()

	catalogue := engine.BuiltinTypes()
	require.NotNil(t, catalogue, "type catalogue should not be nil")

	t.Run("string resolves to text", func(t *testing.T) {
		t.Parallel()

		sqlType, exists := catalogue.Types["string"]
		require.True(t, exists, "type alias 'string' should exist in the type catalogue")
		assert.Equal(t, querier_dto.TypeCategoryText, sqlType.Category,
			"'string' should resolve to the text category")
		assert.Equal(t, "text", sqlType.EngineName,
			"'string' should resolve to engine name 'text'")
	})

	t.Run("bytes resolves to bytea", func(t *testing.T) {
		t.Parallel()

		sqlType, exists := catalogue.Types["bytes"]
		require.True(t, exists, "type alias 'bytes' should exist in the type catalogue")
		assert.Equal(t, querier_dto.TypeCategoryBytea, sqlType.Category,
			"'bytes' should resolve to the bytea category")
		assert.Equal(t, "bytea", sqlType.EngineName,
			"'bytes' should resolve to engine name 'bytea'")
	})
}

func TestCockroachDB_SupportsReturning(t *testing.T) {
	t.Parallel()

	engine := NewCockroachDBEngine()

	assert.True(t, engine.SupportsReturning(), "CockroachDB should support RETURNING clauses")
}

func TestCockroachDB_ParameterStyle(t *testing.T) {
	t.Parallel()

	engine := NewCockroachDBEngine()

	assert.Equal(t, querier_dto.ParameterStyleDollar, engine.ParameterStyle(),
		"CockroachDB should use dollar-sign parameter style")
}

func TestCockroachDB_DefaultSchema(t *testing.T) {
	t.Parallel()

	engine := NewCockroachDBEngine()

	assert.Equal(t, "public", engine.DefaultSchema(),
		"CockroachDB should have 'public' as the default schema")
}

func TestCockroachDB_BuiltinFunctions_ExtraFunctions(t *testing.T) {
	t.Parallel()

	engine := NewCockroachDBEngine()

	catalogue := engine.BuiltinFunctions()
	require.NotNil(t, catalogue, "function catalogue should not be nil")

	expectedFunctions := []string{
		"unique_rowid",
		"cluster_logical_timestamp",
		"crdb_internal.cluster_id",
		"gateway_region",
		"rehome_row",
		"crdb_internal.node_id",
		"crdb_internal.is_admin",
		"crdb_internal.locality_value",
		"from_ip",
		"to_ip",
		"experimental_strftime",
		"experimental_strptime",
	}

	for _, functionName := range expectedFunctions {
		t.Run(functionName, func(t *testing.T) {
			t.Parallel()

			signatures, exists := catalogue.Functions[functionName]
			assert.True(t, exists, "function %q should exist in the catalogue", functionName)
			assert.NotEmpty(t, signatures, "function %q should have at least one signature", functionName)
		})
	}
}
