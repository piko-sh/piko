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

package db_engine_sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_adapters/engine_shared"
	"piko.sh/piko/internal/querier/querier_dto"
)

func TestArgsBuildsWellFormedArguments(t *testing.T) {
	t.Parallel()

	builder := &FunctionCatalogueBuilder{CatalogueBuilder: engine_shared.NewCatalogueBuilder()}
	textType := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}
	integerType := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "integer"}

	arguments := builder.Args(Arg{Name: paramNameStr, Type: textType}, Arg{Name: "start", Type: integerType})
	require.Len(t, arguments, 2)
	assert.Equal(t, paramNameStr, arguments[0].Name)
	assert.Equal(t, textType, arguments[0].Type)
	assert.Equal(t, "start", arguments[1].Name)
	assert.Equal(t, integerType, arguments[1].Type)
}

func TestBuildFunctionCatalogueRegistersDualArityOverloads(t *testing.T) {
	t.Parallel()

	catalogue := buildFunctionCatalogue()

	wantArities := map[string][]int{
		"substr":    {2, 3},
		"substring": {2, 3},
		"round":     {1, 2},
		"trim":      {1, 2},
		"ltrim":     {1, 2},
		"rtrim":     {1, 2},
	}

	for name, arities := range wantArities {
		overloads, found := catalogue.Functions[name]
		require.True(t, found, "function %q is not registered", name)
		registered := make(map[int]bool, len(overloads))
		for _, signature := range overloads {
			registered[len(signature.Arguments)] = true
		}
		for _, arity := range arities {
			assert.Truef(t, registered[arity],
				"function %q is missing the %d-argument overload", name, arity)
		}
	}
}

func TestBuildFunctionCatalogueOverloadsAreConsistent(t *testing.T) {
	t.Parallel()

	catalogue := buildFunctionCatalogue()
	require.NotEmpty(t, catalogue.Functions)

	for name, overloads := range catalogue.Functions {
		require.NotEmpty(t, overloads, "function %q registered with no overloads", name)
		for index, signature := range overloads {
			require.NotNil(t, signature, "function %q overload %d is nil", name, index)
			assert.Equal(t, name, signature.Name, "function %q overload %d carries the wrong name", name, index)
			if signature.MinArguments > 0 {
				assert.LessOrEqual(t, signature.MinArguments, len(signature.Arguments),
					"function %q overload %d declares MinArguments beyond its argument count", name, index)
			}
			for argumentIndex, argument := range signature.Arguments {
				assert.NotEmpty(t, argument.Name,
					"function %q overload %d argument %d has an empty name", name, index, argumentIndex)
			}
		}
	}
}
