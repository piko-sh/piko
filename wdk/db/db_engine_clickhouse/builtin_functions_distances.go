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

// registerDistanceAndNormFunctions covers the vector-distance and vector-norm functions
// beyond the base L1, L2, Linf, and cosine variants.
//
// The set includes squared distance and norm, Lp variants, normalisers, and the Hamming-
// distance family across tuples, bytes, arrays, and integers. Registration delegates to
// topical helpers to keep each registration helper within the linter budget.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerDistanceAndNormFunctions(b *FunctionCatalogueBuilder) {
	registerVectorDistances(b)
	registerVectorNormalisers(b)
	registerHammingDistances(b)
}

// registerVectorDistances covers the Lp distance and norm helpers alongside the squared
// L2 and the transposed-matrix variants used by vector-similarity search.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerVectorDistances(b *FunctionCatalogueBuilder) {
	vector := vectorFloat64Type(b)
	b.Register("LpDistance", b.float64Type, vector, vector, b.float64Type)
	b.Register("LpNorm", b.float64Type, vector, b.float64Type)
	b.Register("L2SquaredDistance", b.float64Type, vector, vector)
	b.Register("L2SquaredNorm", b.float64Type, vector)
	b.Register("L2DistanceTransposed", b.float64Type, vector, vector, b.float64Type)
	b.Register("cosineDistanceTransposed", b.float64Type, vector, vector, b.float64Type)
}

// registerVectorNormalisers covers the L1, L2, Linf, and Lp normalisers which rescale a
// vector to unit length under their respective norms.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerVectorNormalisers(b *FunctionCatalogueBuilder) {
	vector := vectorFloat64Type(b)
	b.Register("L1Normalize", vector, vector)
	b.Register("L2Normalize", vector, vector)
	b.Register("LinfNormalize", vector, vector)
	b.Register("LpNormalize", vector, vector, b.float64Type)
}

// registerHammingDistances covers the Hamming-distance helpers across the tuple, array,
// and integer encodings.
//
// Tuples and arrays accept Dynamic element types because the distance is defined on any
// comparable element. byteHammingDistance lives in registerStringSimilarityFunctions
// because it operates on String inputs and groups with the rest of the string similarity
// family.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerHammingDistances(b *FunctionCatalogueBuilder) {
	b.Register("tupleHammingDistance", b.uint64Type, b.unknownType, b.unknownType)
	b.Register("arrayHammingDistance", b.uint64Type, arrayOf(b.unknownType), arrayOf(b.unknownType))
	b.Register("bitHammingDistance", b.uint64Type, b.int64Type, b.int64Type)
}

// vectorFloat64Type returns the canonical Array(Float64) shape used by every vector
// distance and norm helper.
//
// Centralised so both the distance and normaliser registration sites share one source of
// truth for the vector element type.
//
// Takes b (*FunctionCatalogueBuilder) which supplies the shared float64 type descriptor.
//
// Returns querier_dto.SQLType which is the Array(Float64) vector type.
func vectorFloat64Type(b *FunctionCatalogueBuilder) querier_dto.SQLType {
	return arrayOf(b.float64Type)
}
