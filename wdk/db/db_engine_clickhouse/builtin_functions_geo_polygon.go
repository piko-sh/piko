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

// registerGeoPolygonFunctions covers the WKT / WKB readers, the canonical wkt() writer,
// and the Cartesian / Spherical polygon arithmetic helpers.
//
// The WKT readers each emit the corresponding ClickHouse geometry type so downstream
// consumers receive the precise shape rather than an opaque blob. Delegates to topical
// helpers to keep each function within the linter budget.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered geometry functions.
func registerGeoPolygonFunctions(b *FunctionCatalogueBuilder) {
	registerWKTReaders(b)
	registerWKBReaders(b)
	registerWKTWriter(b)
	registerPolygonArithmetic(b)
	registerPolygonMetrics(b)
}

// registerWKTReaders covers the Well-Known Text parsers, each mapping a String input to
// the corresponding ClickHouse geometry value.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered reader functions.
func registerWKTReaders(b *FunctionCatalogueBuilder) {
	b.Register("readWKTPoint", geoType("Point"), b.textType)
	b.Register("readWKTLineString", geoType("LineString"), b.textType)
	b.Register("readWKTPolygon", geoType("Polygon"), b.textType)
	b.Register("readWKTMultiPolygon", geoType("MultiPolygon"), b.textType)
	b.Register("readWKTRing", geoType("Ring"), b.textType)
}

// registerWKBReaders covers the Well-Known Binary parsers.
//
// WKB inputs are FixedString-like blobs but ClickHouse accepts String here, so the
// catalogue records the parser inputs as String.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered reader functions.
func registerWKBReaders(b *FunctionCatalogueBuilder) {
	b.Register("readWKBPoint", geoType("Point"), b.textType)
	b.Register("readWKBLineString", geoType("LineString"), b.textType)
	b.Register("readWKBPolygon", geoType("Polygon"), b.textType)
	b.Register("readWKBMultiPolygon", geoType("MultiPolygon"), b.textType)
	b.Register("readWKBMultiLineString", geoType("MultiLineString"), b.textType)
}

// registerWKTWriter covers the Well-Known Text serialiser, which accepts any geometry
// value (modelled as Dynamic for catalogue purposes) and returns a String.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered writer function.
func registerWKTWriter(b *FunctionCatalogueBuilder) {
	b.Register("wkt", b.textType, b.unknownType)
}

// registerPolygonArithmetic covers the polygon set operators in both Cartesian and
// Spherical projections.
//
// Each pair has an intersection geometry result, union, symmetric difference and the
// predicate helpers for equality, containment and pairwise intersection.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered operator functions.
func registerPolygonArithmetic(b *FunctionCatalogueBuilder) {
	multiPolygon := geoType("MultiPolygon")
	for _, name := range []string{
		"polygonsIntersectionCartesian", "polygonsIntersectionSpherical",
		"polygonsUnionCartesian", "polygonsUnionSpherical",
		"polygonsSymDifferenceCartesian", "polygonsSymDifferenceSpherical",
	} {
		b.Register(name, multiPolygon, b.unknownType, b.unknownType)
	}
	for _, name := range []string{
		"polygonsIntersectCartesian", "polygonsIntersectSpherical",
		"polygonsWithinCartesian", "polygonsWithinSpherical",
		"polygonsEqualsCartesian",
	} {
		b.Register(name, b.boolType, b.unknownType, b.unknownType)
	}

	b.Register("polygonConvexHullCartesian", geoType("Polygon"), b.unknownType)
}

// registerPolygonMetrics covers the polygon scalar metrics (area, perimeter, pairwise
// distance) in both Cartesian and Spherical projections.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered metric functions.
func registerPolygonMetrics(b *FunctionCatalogueBuilder) {
	for _, name := range []string{
		"polygonAreaCartesian", "polygonAreaSpherical",
		"polygonPerimeterCartesian", "polygonPerimeterSpherical",
	} {
		b.Register(name, b.float64Type, b.unknownType)
	}
	for _, name := range []string{"polygonsDistanceCartesian", "polygonsDistanceSpherical"} {
		b.Register(name, b.float64Type, b.unknownType, b.unknownType)
	}
}

// geoType constructs a ClickHouse Geometric SQLType for the supplied engine name.
//
// Centralised so each geometry family (Point, Ring, LineString, MultiLineString, Polygon,
// MultiPolygon) shares one constructor and the category cannot drift between sites.
//
// Takes engineName (string) which is the ClickHouse geometry type name.
//
// Returns querier_dto.SQLType which is the geometric type for that name.
func geoType(engineName string) querier_dto.SQLType {
	return querier_dto.SQLType{Category: querier_dto.TypeCategoryGeometric, EngineName: engineName}
}
