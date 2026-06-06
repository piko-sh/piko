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

// registerBitmapFunctions covers the RoaringBitmap helpers exposed by ClickHouse.
//
// All bitmap values are modelled with the unknown Dynamic type because the catalogue does
// not carry a dedicated bitmap SQLTypeCategory; downstream consumers track the bitmap
// shape via engine-specific metadata. Registration delegates to topical helpers to keep
// each registration helper within the linter budget.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerBitmapFunctions(b *FunctionCatalogueBuilder) {
	registerBitmapConstructors(b)
	registerBitmapPredicates(b)
	registerBitmapSetOperations(b)
	registerBitmapAggregates(b)
}

// registerBitmapConstructors covers the constructors and subset extractors: bitmapBuild,
// bitmapToArray, bitmapSubsetInRange, bitmapSubsetLimit, subBitmap, and bitmapTransform.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerBitmapConstructors(b *FunctionCatalogueBuilder) {
	b.Register("bitmapBuild", bitmapType(b), arrayOf(b.uint64Type))
	b.Register("bitmapToArray", arrayOf(b.uint64Type), bitmapType(b))
	b.Register("bitmapSubsetInRange", bitmapType(b), bitmapType(b), b.uint64Type, b.uint64Type)
	b.Register("bitmapSubsetLimit", bitmapType(b), bitmapType(b), b.uint64Type, b.uint64Type)
	b.Register("subBitmap", bitmapType(b), bitmapType(b), b.uint64Type, b.uint64Type)
	b.Register("bitmapTransform", bitmapType(b), bitmapType(b), arrayOf(b.uint64Type), arrayOf(b.uint64Type))
}

// registerBitmapPredicates covers membership, containment, and the scalar min, max, and
// cardinality accessors.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerBitmapPredicates(b *FunctionCatalogueBuilder) {
	b.Register("bitmapContains", b.boolType, bitmapType(b), b.uint64Type)
	b.Register("bitmapHasAll", b.boolType, bitmapType(b), bitmapType(b))
	b.Register("bitmapHasAny", b.boolType, bitmapType(b), bitmapType(b))
	b.Register("bitmapMin", b.uint64Type, bitmapType(b))
	b.Register("bitmapMax", b.uint64Type, bitmapType(b))
	b.Register("bitmapCardinality", b.uint64Type, bitmapType(b))
}

// registerBitmapSetOperations covers the pairwise set operators and the cardinality-
// returning counterparts that skip materialising the intermediate bitmap.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerBitmapSetOperations(b *FunctionCatalogueBuilder) {
	b.Register("bitmapAnd", bitmapType(b), bitmapType(b), bitmapType(b))
	b.Register("bitmapOr", bitmapType(b), bitmapType(b), bitmapType(b))
	b.Register("bitmapXor", bitmapType(b), bitmapType(b), bitmapType(b))
	b.Register("bitmapAndnot", bitmapType(b), bitmapType(b), bitmapType(b))
	b.Register("bitmapAndCardinality", b.uint64Type, bitmapType(b), bitmapType(b))
	b.Register("bitmapOrCardinality", b.uint64Type, bitmapType(b), bitmapType(b))
	b.Register("bitmapXorCardinality", b.uint64Type, bitmapType(b), bitmapType(b))
	b.Register("bitmapAndnotCardinality", b.uint64Type, bitmapType(b), bitmapType(b))
}

// registerBitmapAggregates covers the group-bitmap aggregates which accumulate UInt64
// values or pre-built bitmaps into a single aggregate-state bitmap.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerBitmapAggregates(b *FunctionCatalogueBuilder) {
	b.RegisterAggregate("groupBitmap", b.uint64Type, b.uint64Type)
	b.RegisterAggregate("groupBitmapAnd", b.uint64Type, bitmapType(b))
	b.RegisterAggregate("groupBitmapOr", b.uint64Type, bitmapType(b))
	b.RegisterAggregate("groupBitmapXor", b.uint64Type, bitmapType(b))
}

// bitmapType returns the SQLType used to model RoaringBitmap values across the bitmap
// helper family.
//
// The catalogue carries no dedicated bitmap category; downstream consumers identify
// bitmaps via engine-specific metadata, so the helpers fall back to the Dynamic unknown
// type to keep the catalogue consistent.
//
// Takes b (*FunctionCatalogueBuilder) which supplies the shared unknown type descriptor.
//
// Returns querier_dto.SQLType which is the Dynamic unknown type modelling a bitmap.
func bitmapType(b *FunctionCatalogueBuilder) querier_dto.SQLType {
	return b.unknownType
}
