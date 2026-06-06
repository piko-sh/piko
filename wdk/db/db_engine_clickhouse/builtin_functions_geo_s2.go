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

package db_engine_clickhouse

import (
	"piko.sh/piko/internal/querier/querier_dto"
)

// registerGeoS2Functions covers the S2 spatial index helpers.
//
// The S2 library encodes cells as UInt64 identifiers. The cap and rect operators return
// Tuple(UInt64, Float64) shapes for the cap and rect family, and the cell-conversion
// helpers return Tuple(Float64, Float64) longitude and latitude pairs.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerGeoS2Functions(b *FunctionCatalogueBuilder) {
	posType := tupleFloat64Pos(b)
	capType := tupleUInt64Float64(b)

	rectType := tupleUInt64Pair(b)
	b.Register("geoToS2", b.uint64Type, b.float64Type, b.float64Type)
	b.Register("s2ToGeo", posType, b.uint64Type)
	b.Register("s2GetNeighbors", arrayOf(b.uint64Type), b.uint64Type)
	b.Register("s2CellsIntersect", b.boolType, b.uint64Type, b.uint64Type)
	b.Register("s2CapContains", b.boolType, b.uint64Type, b.float64Type, b.uint64Type)
	b.Register("s2CapUnion", capType, b.uint64Type, b.float64Type, b.uint64Type, b.float64Type)
	b.Register("s2RectAdd", rectType, b.uint64Type, b.uint64Type, b.uint64Type)
	b.Register("s2RectContains", b.boolType, b.uint64Type, b.uint64Type, b.uint64Type)
	b.Register("s2RectUnion", rectType, b.uint64Type, b.uint64Type, b.uint64Type, b.uint64Type)
	b.Register("s2RectIntersection", rectType, b.uint64Type, b.uint64Type, b.uint64Type, b.uint64Type)
}

// tupleUInt64Pair constructs a Tuple(UInt64, UInt64) SQLType used by the s2 rectangle
// helpers which report a (lo, hi) pair of S2 cell identifiers.
//
// Takes b (*FunctionCatalogueBuilder) which supplies the element SQL types.
//
// Returns querier_dto.SQLType which is the constructed tuple type.
func tupleUInt64Pair(b *FunctionCatalogueBuilder) querier_dto.SQLType {
	return querier_dto.SQLType{
		Category:   querier_dto.TypeCategoryStruct,
		EngineName: "Tuple",
		StructFields: []querier_dto.StructField{
			{Name: "lo", SQLType: b.uint64Type},
			{Name: "hi", SQLType: b.uint64Type},
		},
	}
}

// tupleUInt64Float64 constructs a Tuple(UInt64, Float64) SQLType used by the s2 cap-union
// helper which reports a centre cell identifier plus a radius in degrees.
//
// Takes b (*FunctionCatalogueBuilder) which supplies the element SQL types.
//
// Returns querier_dto.SQLType which is the constructed tuple type.
func tupleUInt64Float64(b *FunctionCatalogueBuilder) querier_dto.SQLType {
	return querier_dto.SQLType{
		Category:   querier_dto.TypeCategoryStruct,
		EngineName: "Tuple",
		StructFields: []querier_dto.StructField{
			{Name: "centre", SQLType: b.uint64Type},
			{Name: "radius", SQLType: b.float64Type},
		},
	}
}

// tupleFloat64Pos constructs a Tuple(Float64, Float64) SQLType used by the s2 geographic
// decode helper which reports a (longitude, latitude) pair.
//
// It is shared with builtin_functions_h3_extended.go where the H3 boundary helpers return
// an array of the same tuple shape, and kept here because the s2 family is the primary
// consumer.
//
// Takes b (*FunctionCatalogueBuilder) which supplies the element SQL types.
//
// Returns querier_dto.SQLType which is the constructed tuple type.
func tupleFloat64Pos(b *FunctionCatalogueBuilder) querier_dto.SQLType {
	return querier_dto.SQLType{
		Category:   querier_dto.TypeCategoryStruct,
		EngineName: "Tuple",
		StructFields: []querier_dto.StructField{
			{Name: "lon", SQLType: b.float64Type},
			{Name: "lat", SQLType: b.float64Type},
		},
	}
}
