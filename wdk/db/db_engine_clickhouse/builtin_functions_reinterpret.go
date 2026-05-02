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

const (
	// reinterpretFixedStringBytes is the canonical byte length used when reinterpreting
	// arbitrary values as FixedString. UUIDs and IPv6 values both occupy sixteen bytes, so
	// reinterpretAsFixedString is modelled with this width.
	reinterpretFixedStringBytes = 16

	// formatRowMinArgs is the minimum argument count for formatRow: at least a format-name
	// string plus one column.
	formatRowMinArgs = 2
)

// registerReinterpretFunctions covers the reinterpretAs* family, formatRow /
// formatRowNoNewline, toStringOrNull and the BFloat16 / DateTime32 conversion helpers.
// These functions all reinterpret the underlying byte representation rather than
// performing a value conversion, so they share a single registration block.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered signatures.
func registerReinterpretFunctions(b *FunctionCatalogueBuilder) {
	registerReinterpretCore(b)
	registerReinterpretInteger(b)
	registerReinterpretFloat(b)
	registerFormatRowAndStringNullableHelpers(b)
	registerBFloat16Conversions(b)
}

// registerReinterpretCore covers the bare reinterpret function plus the named
// reinterpretAs* helpers targeting String, FixedString, Date, DateTime and UUID.
//
// The bare reinterpret form takes a value and a type name (as a String) and is registered
// as a two-argument helper returning Dynamic so callers can express it without a
// dedicated signature per target type.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered signatures.
func registerReinterpretCore(b *FunctionCatalogueBuilder) {
	b.Register("reinterpret", b.unknownType, b.unknownType, b.textType)
	b.Register("reinterpretAsString", b.textType, b.unknownType)
	b.Register("reinterpretAsFixedString", fixedStringType(reinterpretFixedStringBytes), b.unknownType)
	b.Register("reinterpretAsDate", b.dateType, b.unknownType)
	b.Register("reinterpretAsDateTime", b.dateTimeType, b.unknownType)
	b.Register("reinterpretAsUUID", b.uuidType, b.unknownType)
}

// registerReinterpretInteger registers reinterpretAsInt8 through Int256 and the matching
// unsigned widths. Each width keeps the promoted return category (Int64 / UInt64) because
// the catalogue stores a single promoted type per signature; the precise width is carried
// through the EngineName for downstream emitters.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered signatures.
func registerReinterpretInteger(b *FunctionCatalogueBuilder) {
	signedWidths := []string{"8", "16", "32", "64", "128", "256"}
	for _, width := range signedWidths {
		b.Register("reinterpretAsInt"+width, b.int64Type, b.unknownType)
		b.Register("reinterpretAsUInt"+width, b.uint64Type, b.unknownType)
	}
}

// registerReinterpretFloat registers reinterpretAsFloat32 and reinterpretAsFloat64. Float
// reinterpretation is widely used when migrating between integer fixed-point
// representations and floating-point storage.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered signatures.
func registerReinterpretFloat(b *FunctionCatalogueBuilder) {
	b.Register("reinterpretAsFloat32", b.float32Type, b.unknownType)
	b.Register("reinterpretAsFloat64", b.float64Type, b.unknownType)
}

// registerFormatRowAndStringNullableHelpers covers formatRow, formatRowNoNewline
// (variadic over columns) and toStringOrNull.
//
// formatRow returns the row formatted using a named output format (e.g. "CSV", "JSON"),
// so the catalogue treats the format-name argument as a regular text and the trailing
// columns as a variadic Dynamic tail.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered signatures.
func registerFormatRowAndStringNullableHelpers(b *FunctionCatalogueBuilder) {
	b.RegisterVariadic("formatRow", b.textType, formatRowMinArgs, b.textType, b.unknownType)
	b.RegisterVariadic("formatRowNoNewline", b.textType, formatRowMinArgs, b.textType, b.unknownType)
	b.Register("toStringOrNull", b.textType, b.unknownType)
}

// registerBFloat16Conversions covers the toBFloat16 family.
//
// BFloat16 is the brain floating point format ClickHouse uses for memory-efficient ML
// workloads. Each variant is modelled with the wider Float32 promoted category so
// emitters can downcast based on the EngineName.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered signatures.
func registerBFloat16Conversions(b *FunctionCatalogueBuilder) {
	bfloat16Type := querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "BFloat16"}
	b.Register("toBFloat16", bfloat16Type, b.unknownType)
	for _, suffix := range safetyConversionSuffixes {
		b.Register("toBFloat16"+suffix, bfloat16Type, b.unknownType)
	}
}

// registerExtendedConversionFunctions covers the toDateTime32 family, the parseDateTime /
// parseDateTime64 safety variants, the Joda syntax sub-second parsers and
// fromUnixTimestamp64Second. These extend the existing date-time conversion catalogue
// with the fine-grained variants ClickHouse exposes for sub-second precision.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered signatures.
func registerExtendedConversionFunctions(b *FunctionCatalogueBuilder) {
	registerDateTime32Conversions(b)
	registerParseDateTimeVariants(b)
	registerParseDateTime64Variants(b)
	registerParseDateTime64JodaVariants(b)
	registerFromUnixTimestampSecond(b)
}

// registerDateTime32Conversions covers toDateTime32 and its safety variants. ClickHouse
// keeps DateTime32 as an explicit alias for the 32-bit DateTime so callers can opt into
// the legacy precision.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered signatures.
func registerDateTime32Conversions(b *FunctionCatalogueBuilder) {
	b.Register("toDateTime32", b.dateTimeType, b.unknownType)
	for _, suffix := range safetyConversionSuffixes {
		b.Register("toDateTime32"+suffix, b.dateTimeType, b.unknownType)
	}
}

// registerParseDateTimeVariants covers parseDateTimeOrNull and parseDateTimeOrZero. The
// base parseDateTime is already registered in registerCoreDateTime; the safety variants
// land here so the catalogue covers every failure-mode spelling.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered signatures.
func registerParseDateTimeVariants(b *FunctionCatalogueBuilder) {
	b.Register("parseDateTimeOrNull", b.dateTimeType, b.textType, b.textType)
	b.Register("parseDateTimeOrZero", b.dateTimeType, b.textType, b.textType)
}

// registerParseDateTime64Variants covers parseDateTime64 with its scale, optional format
// and optional timezone arguments, plus the OrNull and OrZero safety variants. Each arity
// is registered separately so signature matching can pick the right overload.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered signatures.
func registerParseDateTime64Variants(b *FunctionCatalogueBuilder) {
	b.Register("parseDateTime64", b.dateTime64Type, b.textType, b.uint64Type)
	b.Register("parseDateTime64", b.dateTime64Type, b.textType, b.uint64Type, b.textType)
	b.Register("parseDateTime64", b.dateTime64Type, b.textType, b.uint64Type, b.textType, b.textType)
	for _, suffix := range nullZeroSafetyConversionSuffixes {
		b.Register("parseDateTime64"+suffix, b.dateTime64Type, b.textType, b.uint64Type)
		b.Register("parseDateTime64"+suffix, b.dateTime64Type, b.textType, b.uint64Type, b.textType)
		b.Register("parseDateTime64"+suffix, b.dateTime64Type, b.textType, b.uint64Type, b.textType, b.textType)
	}
}

// registerParseDateTime64JodaVariants covers the Joda-syntax parsers for DateTime64 with
// the optional timezone argument and the OrNull / OrZero safety variants.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered signatures.
func registerParseDateTime64JodaVariants(b *FunctionCatalogueBuilder) {
	b.Register("parseDateTime64InJodaSyntax", b.dateTime64Type, b.textType, b.uint64Type, b.textType)
	b.Register("parseDateTime64InJodaSyntax", b.dateTime64Type, b.textType, b.uint64Type, b.textType, b.textType)
	for _, suffix := range nullZeroSafetyConversionSuffixes {
		b.Register("parseDateTime64InJodaSyntax"+suffix, b.dateTime64Type, b.textType, b.uint64Type, b.textType)
		b.Register("parseDateTime64InJodaSyntax"+suffix, b.dateTime64Type, b.textType, b.uint64Type, b.textType, b.textType)
	}
}

// registerFromUnixTimestampSecond covers fromUnixTimestamp64Second which mirrors the
// existing Milli / Micro / Nano helpers but at the whole-second grain.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered signatures.
func registerFromUnixTimestampSecond(b *FunctionCatalogueBuilder) {
	b.Register("fromUnixTimestamp64Second", b.dateTimeType, b.int64Type)
}
