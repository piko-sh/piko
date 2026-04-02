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

package db_engine_mariadb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestNewMariaDBEngine(t *testing.T) {
	t.Parallel()

	engine := NewMariaDBEngine()

	require.NotNil(t, engine, "engine should not be nil")
	assert.Equal(t, "mariadb", engine.Dialect(), "dialect should be mariadb")
}

func TestMariaDB_SupportsReturning(t *testing.T) {
	t.Parallel()

	engine := NewMariaDBEngine()

	assert.True(t, engine.SupportsReturning(), "MariaDB should support RETURNING clauses")
}

func TestMariaDB_ParameterStyle(t *testing.T) {
	t.Parallel()

	engine := NewMariaDBEngine()

	assert.Equal(t, querier_dto.ParameterStyleQuestion, engine.ParameterStyle(),
		"MariaDB should use question-mark parameter style")
}

func TestMariaDB_DefaultSchema(t *testing.T) {
	t.Parallel()

	engine := NewMariaDBEngine()

	assert.Equal(t, "", engine.DefaultSchema(),
		"MariaDB should have an empty default schema")
}

func TestMariaDB_BuiltinFunctions_ExtraFunctions(t *testing.T) {
	t.Parallel()

	engine := NewMariaDBEngine()

	catalogue := engine.BuiltinFunctions()
	require.NotNil(t, catalogue, "function catalogue should not be nil")

	expectedFunctions := []string{
		"uuid",
		"sys_guid",
		"inet_aton",
		"inet_ntoa",
		"inet6_aton",
		"inet6_ntoa",
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
