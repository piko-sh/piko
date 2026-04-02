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
