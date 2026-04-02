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

package querier_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestScanMigrationReadOnlyOverrides(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		content       string
		commentPrefix string
		expected      map[string]*bool
	}{
		{
			name:          "no directives returns empty map",
			content:       "SELECT 1;",
			commentPrefix: "--",
			expected:      map[string]*bool{},
		},
		{
			name: "piko.readonly before CREATE FUNCTION returns map with function name true",
			content: `-- piko.readonly
CREATE FUNCTION my_func() RETURNS void AS $$ BEGIN END; $$ LANGUAGE plpgsql;`,
			commentPrefix: "--",
			expected: map[string]*bool{
				"my_func": new(true),
			},
		},
		{
			name: "piko.readonly(false) before CREATE FUNCTION returns map with function name false",
			content: `-- piko.readonly(false)
CREATE FUNCTION my_writer() RETURNS void AS $$ BEGIN END; $$ LANGUAGE plpgsql;`,
			commentPrefix: "--",
			expected: map[string]*bool{
				"my_writer": new(false),
			},
		},
		{
			name: "directive not followed by CREATE FUNCTION is ignored",
			content: `-- piko.readonly
SELECT 1;
CREATE FUNCTION after_select() RETURNS void AS $$ BEGIN END; $$ LANGUAGE plpgsql;`,
			commentPrefix: "--",
			expected:      map[string]*bool{},
		},
		{
			name: "multiple functions with overrides returns all",
			content: `-- piko.readonly
CREATE FUNCTION reader() RETURNS void AS $$ BEGIN END; $$ LANGUAGE plpgsql;
-- piko.readonly(false)
CREATE FUNCTION writer() RETURNS void AS $$ BEGIN END; $$ LANGUAGE plpgsql;`,
			commentPrefix: "--",
			expected: map[string]*bool{
				"reader": new(true),
				"writer": new(false),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := scanMigrationReadOnlyOverrides(tt.content, tt.commentPrefix)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractCreateFunctionName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{
			name:     "CREATE FUNCTION myFunc returns lowercase name",
			line:     "CREATE FUNCTION myFunc() RETURNS void",
			expected: "myfunc",
		},
		{
			name:     "CREATE OR REPLACE FUNCTION myFunc returns lowercase name",
			line:     "CREATE OR REPLACE FUNCTION myFunc() RETURNS void",
			expected: "myfunc",
		},
		{
			name:     "schema-qualified name strips schema prefix",
			line:     "CREATE FUNCTION myschema.myFunc() RETURNS void",
			expected: "myfunc",
		},
		{
			name:     "not a CREATE FUNCTION statement returns empty string",
			line:     "SELECT * FROM users",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := extractCreateFunctionName(tt.line)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCleanFunctionName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		raw      string
		expected string
	}{
		{
			name:     "name with parens strips trailing parens content",
			raw:      "my_func(integer,text)",
			expected: "my_func",
		},
		{
			name:     "schema-qualified name strips schema prefix",
			raw:      "myschema.my_func",
			expected: "my_func",
		},
		{
			name:     "quoted name strips quotes",
			raw:      `"MyFunc"`,
			expected: "myfunc",
		},
		{
			name:     "simple name returns lowercase",
			raw:      "UpperCase",
			expected: "uppercase",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := cleanFunctionName(tt.raw)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestApplyMigrationReadOnlyOverride(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mutation   *querier_dto.CatalogueMutation
		overrides  map[string]*bool
		assertions func(t *testing.T, mutation *querier_dto.CatalogueMutation)
	}{
		{
			name: "nil mutation does not panic",
			mutation: &querier_dto.CatalogueMutation{
				Kind: querier_dto.MutationCreateTable,
			},
			overrides: map[string]*bool{},
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				t.Helper()

				assert.Nil(t, mutation.FunctionSignature)
			},
		},
		{
			name: "non-CreateFunction mutation kind is a no-op",
			mutation: &querier_dto.CatalogueMutation{
				Kind: querier_dto.MutationCreateTable,
				FunctionSignature: &querier_dto.FunctionSignature{
					Name:       "irrelevant",
					DataAccess: querier_dto.DataAccessUnknown,
				},
			},
			overrides: map[string]*bool{
				"irrelevant": new(true),
			},
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				t.Helper()

				assert.Equal(t, querier_dto.DataAccessUnknown, mutation.FunctionSignature.DataAccess)
			},
		},
		{
			name: "CreateFunction with override true sets DataAccessReadOnly",
			mutation: &querier_dto.CatalogueMutation{
				Kind: querier_dto.MutationCreateFunction,
				FunctionSignature: &querier_dto.FunctionSignature{
					Name:       "my_reader",
					DataAccess: querier_dto.DataAccessUnknown,
				},
			},
			overrides: map[string]*bool{
				"my_reader": new(true),
			},
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				t.Helper()

				assert.Equal(t, querier_dto.DataAccessReadOnly, mutation.FunctionSignature.DataAccess)
			},
		},
		{
			name: "CreateFunction with override false sets DataAccessModifiesData",
			mutation: &querier_dto.CatalogueMutation{
				Kind: querier_dto.MutationCreateFunction,
				FunctionSignature: &querier_dto.FunctionSignature{
					Name:       "my_writer",
					DataAccess: querier_dto.DataAccessUnknown,
				},
			},
			overrides: map[string]*bool{
				"my_writer": new(false),
			},
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				t.Helper()

				assert.Equal(t, querier_dto.DataAccessModifiesData, mutation.FunctionSignature.DataAccess)
			},
		},
		{
			name: "CreateFunction with no override remains unchanged",
			mutation: &querier_dto.CatalogueMutation{
				Kind: querier_dto.MutationCreateFunction,
				FunctionSignature: &querier_dto.FunctionSignature{
					Name:       "no_override",
					DataAccess: querier_dto.DataAccessUnknown,
				},
			},
			overrides: map[string]*bool{},
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				t.Helper()

				assert.Equal(t, querier_dto.DataAccessUnknown, mutation.FunctionSignature.DataAccess)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			applyMigrationReadOnlyOverride(tt.mutation, tt.overrides)
			tt.assertions(t, tt.mutation)
		})
	}
}
