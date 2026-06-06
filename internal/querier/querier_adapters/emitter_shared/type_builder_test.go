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

package emitter_shared

import (
	"go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestImportTrackerBuiltinType(t *testing.T) {
	tracker := NewImportTracker()
	expression := tracker.AddType(querier_dto.GoType{Name: "string"})

	require.NotNil(t, expression)
	assert.Empty(t, tracker.imports)
}

func TestImportTrackerExternalPackage(t *testing.T) {
	tracker := NewImportTracker()
	expression := tracker.AddType(querier_dto.GoType{Package: "time", Name: "Time"})

	require.NotNil(t, expression)
	assert.Contains(t, tracker.imports, "time")
}

func TestImportTrackerPointerType(t *testing.T) {
	tracker := NewImportTracker()
	expression := tracker.AddType(querier_dto.GoType{Package: "time", Name: "*Time"})

	require.NotNil(t, expression)
	assert.Contains(t, tracker.imports, "time")
}

func TestImportTrackerSliceType(t *testing.T) {
	tracker := NewImportTracker()
	expression := tracker.AddType(querier_dto.GoType{Name: "[]byte"})

	require.NotNil(t, expression)
	assert.Empty(t, tracker.imports)
}

func TestResolveGoTypeArrayPreservesElementPackage(t *testing.T) {
	mappings := &querier_dto.TypeMappingTable{Mappings: []querier_dto.TypeMapping{
		{
			SQLCategory: querier_dto.TypeCategoryText,
			SQLName:     "uuid",
			NotNull:     querier_dto.GoType{Package: "github.com/google/uuid", Name: "UUID"},
			Nullable:    querier_dto.GoType{Package: "github.com/google/uuid", Name: "UUID"},
		},
	}}
	arrayType := querier_dto.SQLType{
		Category:    querier_dto.TypeCategoryArray,
		ElementType: &querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "uuid"},
	}

	resolved := ResolveGoType(arrayType, false, mappings)
	assert.Equal(t, "[]UUID", resolved.Name)
	assert.Equal(t, "github.com/google/uuid", resolved.Package, "element package must be preserved")

	tracker := NewImportTracker()
	expression := tracker.AddType(resolved)
	require.NotNil(t, expression)
	assert.Contains(t, tracker.imports, "github.com/google/uuid",
		"AddType must register the element package for a typed-slice array")
}

func TestResolveGoTypeExactMatch(t *testing.T) {
	mappings := &querier_dto.TypeMappingTable{
		Mappings: []querier_dto.TypeMapping{
			{
				SQLCategory: querier_dto.TypeCategoryInteger,
				SQLName:     "int8",
				NotNull:     querier_dto.GoType{Name: "int64"},
				Nullable:    querier_dto.GoType{Name: "*int64"},
			},
		},
	}

	result := ResolveGoType(
		querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int8"},
		false,
		mappings,
	)
	assert.Equal(t, "int64", result.Name)
}

func TestResolveGoTypeCategoryFallback(t *testing.T) {
	mappings := &querier_dto.TypeMappingTable{
		Mappings: []querier_dto.TypeMapping{
			{
				SQLCategory: querier_dto.TypeCategoryInteger,
				NotNull:     querier_dto.GoType{Name: "int32"},
				Nullable:    querier_dto.GoType{Name: "*int32"},
			},
		},
	}

	result := ResolveGoType(
		querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "unknown_int"},
		false,
		mappings,
	)
	assert.Equal(t, "int32", result.Name)
}

func TestResolveGoTypeNullable(t *testing.T) {
	mappings := &querier_dto.TypeMappingTable{
		Mappings: []querier_dto.TypeMapping{
			{
				SQLCategory: querier_dto.TypeCategoryText,
				NotNull:     querier_dto.GoType{Name: "string"},
				Nullable:    querier_dto.GoType{Name: "*string"},
			},
		},
	}

	result := ResolveGoType(
		querier_dto.SQLType{Category: querier_dto.TypeCategoryText},
		true,
		mappings,
	)
	assert.Equal(t, "*string", result.Name)
}

func TestResolveGoTypeUnknownFallsBackToAny(t *testing.T) {
	mappings := &querier_dto.TypeMappingTable{
		Mappings: []querier_dto.TypeMapping{},
	}

	result := ResolveGoType(
		querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
		false,
		mappings,
	)
	assert.Equal(t, "any", result.Name)
}

func TestResolveGoTypePrefersCaseSensitiveName(t *testing.T) {
	mappings := &querier_dto.TypeMappingTable{
		Mappings: []querier_dto.TypeMapping{

			{SQLCategory: querier_dto.TypeCategoryInteger, SQLName: "int8", NotNull: querier_dto.GoType{Name: "int64"}, Nullable: querier_dto.GoType{Name: "*int64"}},

			{SQLCategory: querier_dto.TypeCategoryInteger, SQLName: "Int8", NotNull: querier_dto.GoType{Name: "int8"}, Nullable: querier_dto.GoType{Name: "*int8"}},
			{SQLCategory: querier_dto.TypeCategoryInteger, SQLName: "Int64", NotNull: querier_dto.GoType{Name: "int64"}, Nullable: querier_dto.GoType{Name: "*int64"}},
			{SQLCategory: querier_dto.TypeCategoryInteger, SQLName: "UInt64", NotNull: querier_dto.GoType{Name: "uint64"}, Nullable: querier_dto.GoType{Name: "*uint64"}},
		},
	}

	cases := []struct {
		engineName string
		want       string
	}{
		{engineName: "int8", want: "int64"},
		{engineName: "Int8", want: "int8"},
		{engineName: "Int64", want: "int64"},
		{engineName: "UInt64", want: "uint64"},
	}
	for _, testCase := range cases {
		result := ResolveGoType(
			querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: testCase.engineName},
			false,
			mappings,
		)
		assert.Equalf(t, testCase.want, result.Name, "engine name %q", testCase.engineName)
	}
}

func TestImportTracker_DistinctPackagesSamePathBaseGetAliased(t *testing.T) {
	t.Parallel()

	tracker := NewImportTracker()
	firstExpression := tracker.AddType(querier_dto.GoType{Package: "github.com/google/uuid", Name: "UUID"})
	secondExpression := tracker.AddType(querier_dto.GoType{Package: "myapp/uuid", Name: "UUID"})

	firstSelector, firstOk := firstExpression.(*ast.SelectorExpr)
	require.True(t, firstOk, "first AddType should return SelectorExpr")
	firstAlias := firstSelector.X.(*ast.Ident).Name

	secondSelector, secondOk := secondExpression.(*ast.SelectorExpr)
	require.True(t, secondOk, "second AddType should return SelectorExpr")
	secondAlias := secondSelector.X.(*ast.Ident).Name

	assert.NotEqual(t, firstAlias, secondAlias, "colliding path.Base must produce distinct aliases")
	assert.Equal(t, "uuid", firstAlias, "first registrant keeps the natural alias")
	assert.Equal(t, "uuid2", secondAlias, "second registrant gets a numeric-suffixed alias")
}

func TestImportTracker_SamePackageReturnsSameAlias(t *testing.T) {
	t.Parallel()

	tracker := NewImportTracker()
	firstExpression := tracker.AddType(querier_dto.GoType{Package: "github.com/google/uuid", Name: "UUID"})
	secondExpression := tracker.AddType(querier_dto.GoType{Package: "github.com/google/uuid", Name: "Nil"})

	firstAlias := firstExpression.(*ast.SelectorExpr).X.(*ast.Ident).Name
	secondAlias := secondExpression.(*ast.SelectorExpr).X.(*ast.Ident).Name
	assert.Equal(t, firstAlias, secondAlias, "two types in the same package share one alias")
}

func TestImportTracker_NumericAliasContinuesIncrementing(t *testing.T) {
	t.Parallel()

	tracker := NewImportTracker()
	tracker.AddType(querier_dto.GoType{Package: "a/uuid", Name: "UUID"})
	tracker.AddType(querier_dto.GoType{Package: "b/uuid", Name: "UUID"})
	thirdExpression := tracker.AddType(querier_dto.GoType{Package: "c/uuid", Name: "UUID"})

	thirdAlias := thirdExpression.(*ast.SelectorExpr).X.(*ast.Ident).Name
	assert.Equal(t, "uuid3", thirdAlias, "third collision keeps incrementing")
}
