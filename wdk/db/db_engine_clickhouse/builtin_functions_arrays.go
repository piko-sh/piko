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

// registerArraySetOperationFunctions covers array set-operations, rotations, shifts,
// similarity measures, machine learning curve helpers, and the dense and unique
// enumeration variants.
//
// Splitting by topical group keeps each helper function within the linter budget.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerArraySetOperationFunctions(b *FunctionCatalogueBuilder) {
	registerArraySetOperations(b)
	registerArrayShuffleAndSample(b)
	registerArrayPartialSortAndShuffle(b)
	registerArrayFillAndRotate(b)
	registerArrayRemoveAndElementOrNull(b)
	registerArrayEnumerateVariants(b)
	registerArraySimilarityHelpers(b)
	registerArrayLevenshteinHelpers(b)
	registerArrayMLCurveHelpers(b)
	registerArraySplitAndReduceInRanges(b)
}

// registerArraySetOperations covers union, except, and symmetric difference.
//
// The implementations preserve the input element type so the catalogue returns the array
// shape with the element retained.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerArraySetOperations(b *FunctionCatalogueBuilder) {
	b.Register("arrayUnion", arrayOf(b.unknownType), arrayOf(b.unknownType), arrayOf(b.unknownType))
	b.Register("arrayExcept", arrayOf(b.unknownType), arrayOf(b.unknownType), arrayOf(b.unknownType))
	b.Register("arraySymmetricDifference", arrayOf(b.unknownType), arrayOf(b.unknownType), arrayOf(b.unknownType))
}

// registerArrayShuffleAndSample covers arrayShuffle (with optional seed),
// arrayRandomSample, and arrayShingles.
//
// arrayShingles slices the input array into overlapping windows of the given length.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerArrayShuffleAndSample(b *FunctionCatalogueBuilder) {
	b.Register("arrayShuffle", arrayOf(b.unknownType), arrayOf(b.unknownType))
	b.Register("arrayShuffle", arrayOf(b.unknownType), arrayOf(b.unknownType), b.uint64Type)
	b.Register("arrayRandomSample", arrayOf(b.unknownType), arrayOf(b.unknownType), b.uint64Type)
	b.Register("arrayShingles", arrayOf(arrayOf(b.unknownType)), arrayOf(b.unknownType), b.uint64Type)
}

// registerArrayPartialSortAndShuffle covers the partial sort, partial reverse-sort, and
// partial shuffle helpers.
//
// Each takes the array and the cut count; the sort helpers accept an optional lambda
// comparator that the catalogue records as a Dynamic placeholder.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerArrayPartialSortAndShuffle(b *FunctionCatalogueBuilder) {
	b.Register("arrayPartialSort", arrayOf(b.unknownType), arrayOf(b.unknownType), b.uint64Type)
	b.Register("arrayPartialSort", arrayOf(b.unknownType), b.unknownType, arrayOf(b.unknownType), b.uint64Type)
	b.Register("arrayPartialReverseSort", arrayOf(b.unknownType), arrayOf(b.unknownType), b.uint64Type)
	b.Register("arrayPartialReverseSort", arrayOf(b.unknownType), b.unknownType, arrayOf(b.unknownType), b.uint64Type)
	b.Register("arrayPartialShuffle", arrayOf(b.unknownType), arrayOf(b.unknownType), b.uint64Type)
	b.Register("arrayPartialShuffle", arrayOf(b.unknownType), arrayOf(b.unknownType), b.uint64Type, b.uint64Type)
}

// registerArrayFillAndRotate covers the fill and reverse-fill helpers (which propagate
// the previous value through a condition mask) and the rotate and shift helpers (which
// move elements by a given count).
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerArrayFillAndRotate(b *FunctionCatalogueBuilder) {
	b.Register("arrayFill", arrayOf(b.unknownType), b.unknownType, arrayOf(b.unknownType))
	b.Register("arrayReverseFill", arrayOf(b.unknownType), b.unknownType, arrayOf(b.unknownType))
	b.Register("arrayRotateLeft", arrayOf(b.unknownType), arrayOf(b.unknownType), b.int64Type)
	b.Register("arrayRotateRight", arrayOf(b.unknownType), arrayOf(b.unknownType), b.int64Type)
	b.Register("arrayShiftLeft", arrayOf(b.unknownType), arrayOf(b.unknownType), b.int64Type)
	b.Register("arrayShiftRight", arrayOf(b.unknownType), arrayOf(b.unknownType), b.int64Type)
}

// registerArrayRemoveAndElementOrNull covers arrayRemove (drop all occurrences of a
// value) and arrayElementOrNull (positional access that returns NULL on out-of-bounds).
//
// These complement the existing arrayElement and arrayPushBack helpers.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerArrayRemoveAndElementOrNull(b *FunctionCatalogueBuilder) {
	b.Register("arrayRemove", arrayOf(b.unknownType), arrayOf(b.unknownType), b.unknownType)
	b.Register("arrayElementOrNull", b.unknownType, arrayOf(b.unknownType), b.int64Type)
}

// registerArrayEnumerateVariants covers arrayEnumerateDense, arrayEnumerateDenseRanked,
// arrayEnumerateUniq, and arrayEnumerateUniqRanked.
//
// Each returns a UInt64 array indexing the input; the ranked variants accept additional
// grouping arrays.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerArrayEnumerateVariants(b *FunctionCatalogueBuilder) {
	b.Register("arrayEnumerateDense", arrayOf(b.uint64Type), arrayOf(b.unknownType))
	b.RegisterVariadic("arrayEnumerateDenseRanked", arrayOf(b.uint64Type), 1, arrayOf(b.unknownType))
	b.RegisterVariadic("arrayEnumerateUniq", arrayOf(b.uint64Type), 1, arrayOf(b.unknownType))
	b.RegisterVariadic("arrayEnumerateUniqRanked", arrayOf(b.uint64Type), 1, arrayOf(b.unknownType))
}

// registerArraySimilarityHelpers covers the dot product, Jaccard index, similarity,
// autocorrelation, and non-negative cumulative sum helpers.
//
// Each returns a Float64 because the inputs may be either numeric or boolean.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerArraySimilarityHelpers(b *FunctionCatalogueBuilder) {
	b.Register("arrayDotProduct", b.float64Type, arrayOf(b.float64Type), arrayOf(b.float64Type))
	b.Register("arrayJaccardIndex", b.float64Type, arrayOf(b.unknownType), arrayOf(b.unknownType))
	b.Register("arraySimilarity", b.float64Type, arrayOf(b.unknownType), arrayOf(b.unknownType))
	b.Register("arrayAutocorrelation", b.float64Type, arrayOf(b.float64Type))
	b.Register("arrayCumSumNonNegative", arrayOf(b.float64Type), arrayOf(b.float64Type))
}

// registerArrayLevenshteinHelpers covers the Levenshtein distance helpers across plain
// and weighted variants.
//
// The weighted form accepts an additional weight array.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerArrayLevenshteinHelpers(b *FunctionCatalogueBuilder) {
	b.Register("arrayLevenshteinDistance", b.uint64Type, arrayOf(b.unknownType), arrayOf(b.unknownType))
	b.Register("arrayLevenshteinDistanceWeighted", b.float64Type, arrayOf(b.unknownType), arrayOf(b.unknownType), arrayOf(b.float64Type))
}

// registerArrayMLCurveHelpers covers the area-under-curve helpers used for binary
// classifier evaluation.
//
// These are precision-recall AUC, ROC AUC, and the normalised Gini coefficient.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerArrayMLCurveHelpers(b *FunctionCatalogueBuilder) {
	b.Register("arrayAUCPR", b.float64Type, arrayOf(b.float64Type), arrayOf(b.unknownType))
	b.Register("arrayROCAUC", b.float64Type, arrayOf(b.float64Type), arrayOf(b.unknownType))
	b.Register("arrayNormalizedGini", b.float64Type, arrayOf(b.float64Type), arrayOf(b.unknownType))
}

// registerArraySplitAndReduceInRanges covers arraySplit, arrayReverseSplit,
// arrayReduceInRanges, and arrayTranspose.
//
// arrayReduceInRanges takes an aggregate name, an array of ranges, and the input;
// arrayTranspose returns the transposed matrix shape.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerArraySplitAndReduceInRanges(b *FunctionCatalogueBuilder) {
	b.Register("arraySplit", arrayOf(arrayOf(b.unknownType)), b.unknownType, arrayOf(b.unknownType))
	b.Register("arrayReverseSplit", arrayOf(arrayOf(b.unknownType)), b.unknownType, arrayOf(b.unknownType))
	b.Register("arrayReduceInRanges", arrayOf(b.unknownType), b.textType, arrayOf(b.unknownType), arrayOf(b.unknownType))
	b.Register("arrayTranspose", arrayOf(arrayOf(b.unknownType)), arrayOf(arrayOf(b.unknownType)))
}
