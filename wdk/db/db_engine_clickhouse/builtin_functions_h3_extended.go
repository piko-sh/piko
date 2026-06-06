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

// registerH3ExtendedFunctions covers the broader H3 family beyond the historic core.
//
// It covers edge and cell metrics, hierarchy navigation, predicates, boundary lookup,
// line and distance helpers, polygon-to-cell filling and the unidirectional edge family,
// delegating to topical helpers to keep each function within the linter budget.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerH3ExtendedFunctions(b *FunctionCatalogueBuilder) {
	registerH3EdgeMetrics(b)
	registerH3CellMetrics(b)
	registerH3Hierarchy(b)
	registerH3Predicates(b)
	registerH3LinesAndDistance(b)
	registerH3UnidirectionalEdges(b)
}

// registerH3EdgeMetrics covers the H3 edge angular and length metrics across the metric,
// kilometre and radian unit variants.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerH3EdgeMetrics(b *FunctionCatalogueBuilder) {
	for _, name := range []string{
		"h3EdgeAngle", "h3EdgeLengthM", "h3EdgeLengthKm",
		"h3ExactEdgeLengthM", "h3ExactEdgeLengthKm", "h3ExactEdgeLengthRads",
	} {
		b.Register(name, b.float64Type, b.uint64Type)
	}
}

// registerH3CellMetrics covers the H3 cell area and count helpers plus the base-cell and
// face accessors.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerH3CellMetrics(b *FunctionCatalogueBuilder) {
	for _, name := range []string{"h3HexAreaM2", "h3HexAreaKm2", "h3CellAreaM2", "h3CellAreaRads2"} {
		b.Register(name, b.float64Type, b.uint64Type)
	}
	b.Register("h3NumHexagons", b.uint64Type, b.uint64Type)
	b.Register("h3GetBaseCell", b.uint64Type, b.uint64Type)
	b.Register("h3GetFaces", arrayOf(b.uint64Type), b.uint64Type)
}

// registerH3Hierarchy covers the H3 hierarchical navigation helpers for descending or
// ascending the resolution tree, plus the string conversions and the resolution-0 and
// pentagon index enumerators.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerH3Hierarchy(b *FunctionCatalogueBuilder) {
	b.Register("h3ToChildren", arrayOf(b.uint64Type), b.uint64Type, b.uint64Type)
	b.Register("h3ToCenterChild", b.uint64Type, b.uint64Type, b.uint64Type)
	b.Register("h3ToString", b.textType, b.uint64Type)
	b.Register("stringToH3", b.uint64Type, b.textType)
	b.Register("h3GetRes0Indexes", arrayOf(b.uint64Type))
	b.Register("h3GetPentagonIndexes", arrayOf(b.uint64Type), b.uint64Type)
	b.Register("h3ToGeoBoundary", arrayOf(tupleFloat64Pos(b)), b.uint64Type)
}

// registerH3Predicates covers the H3 boolean predicates for orientation, pentagon
// detection and neighbour testing.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerH3Predicates(b *FunctionCatalogueBuilder) {
	b.Register("h3IsResClassIII", b.boolType, b.uint64Type)
	b.Register("h3IsPentagon", b.boolType, b.uint64Type)
	b.Register("h3IndexesAreNeighbors", b.boolType, b.uint64Type, b.uint64Type)
}

// registerH3LinesAndDistance covers the H3 line generation, distance metric, ring and
// great-circle helpers plus polygon-to-cell coverage of a given resolution.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerH3LinesAndDistance(b *FunctionCatalogueBuilder) {
	b.Register("h3Line", arrayOf(b.uint64Type), b.uint64Type, b.uint64Type)
	b.Register("h3Distance", b.uint64Type, b.uint64Type, b.uint64Type)
	b.Register("h3HexRing", arrayOf(b.uint64Type), b.uint64Type, b.uint64Type)
	b.Register("h3PointDistM", b.float64Type, b.float64Type, b.float64Type, b.float64Type, b.float64Type)
	b.Register("h3PointDistKm", b.float64Type, b.float64Type, b.float64Type, b.float64Type, b.float64Type)
	b.Register("h3PointDistRads", b.float64Type, b.float64Type, b.float64Type, b.float64Type, b.float64Type)
	b.Register("h3PolygonToCells", arrayOf(b.uint64Type), b.unknownType, b.uint64Type)
	b.Register("h3PolygonToCellsWithContainment", arrayOf(b.uint64Type), b.unknownType, b.uint64Type, b.uint64Type)
}

// registerH3UnidirectionalEdges covers the seven-strong unidirectional edge family which
// lets callers represent and manipulate directed transitions between neighbouring cells.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerH3UnidirectionalEdges(b *FunctionCatalogueBuilder) {
	b.Register("h3GetUnidirectionalEdge", b.uint64Type, b.uint64Type, b.uint64Type)
	b.Register("h3UnidirectionalEdgeIsValid", b.boolType, b.uint64Type)
	b.Register("h3GetOriginIndexFromUnidirectionalEdge", b.uint64Type, b.uint64Type)
	b.Register("h3GetDestinationIndexFromUnidirectionalEdge", b.uint64Type, b.uint64Type)
	b.Register("h3GetIndexesFromUnidirectionalEdge", arrayOf(b.uint64Type), b.uint64Type)
	b.Register("h3GetUnidirectionalEdgesFromHexagon", arrayOf(b.uint64Type), b.uint64Type)
	b.Register("h3GetUnidirectionalEdgeBoundary", arrayOf(tupleFloat64Pos(b)), b.uint64Type)
}
