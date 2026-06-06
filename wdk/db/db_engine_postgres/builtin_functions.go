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

package db_engine_postgres

import (
	"piko.sh/piko/internal/querier/querier_adapters/engine_shared"
	"piko.sh/piko/internal/querier/querier_dto"
)

// Arg names a single function argument and its SQL type. It aliases the shared toolkit
// type so call sites keep using the bare Arg name while the registration mechanism stays
// shared.
type Arg = engine_shared.Arg

// buildFunctionCatalogue assembles the PostgreSQL function catalogue.
//
// Takes extraFunctions (func(*FunctionCatalogueBuilder)) which adds extra signatures
// after the builtins have been registered.
//
// Returns *querier_dto.FunctionCatalogue which lists every registered function signature.
func buildFunctionCatalogue(extraFunctions func(*FunctionCatalogueBuilder)) *querier_dto.FunctionCatalogue {
	builder := &FunctionCatalogueBuilder{
		CatalogueBuilder: engine_shared.NewCatalogueBuilder(),
		Integer:          querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"},
		Bigint:           querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int8"},
		Smallint:         querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int2"},
		Float4:           querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "float4"},
		Float8:           querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "float8"},
		Numeric:          querier_dto.SQLType{Category: querier_dto.TypeCategoryDecimal, EngineName: "numeric"},
		Boolean:          querier_dto.SQLType{Category: querier_dto.TypeCategoryBoolean, EngineName: "bool"},
		Text:             querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
		Bytea:            querier_dto.SQLType{Category: querier_dto.TypeCategoryBytea, EngineName: "bytea"},
		Timestamp:        querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "timestamp"},
		Timestamptz:      querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "timestamptz"},
		Date:             querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "date"},
		Time:             querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "time"},
		Interval:         querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "interval"},
		JSON:             querier_dto.SQLType{Category: querier_dto.TypeCategoryJSON, EngineName: "json"},
		JSONB:            querier_dto.SQLType{Category: querier_dto.TypeCategoryJSON, EngineName: "jsonb"},
		UUID:             querier_dto.SQLType{Category: querier_dto.TypeCategoryUUID, EngineName: "uuid"},
		Any:              querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: ""},
	}

	builder.registerMathFunctions()
	builder.registerTrigonometricFunctions()
	builder.registerStringFunctions()
	builder.registerDateTimeFunctions()
	builder.registerJSONFunctions()
	builder.registerArrayScalarFunctions()
	builder.registerAggregateFunctions()
	builder.registerWindowFunctions()
	builder.registerConditionalFunctions()
	builder.registerTypeConversionFunctions()
	builder.registerSystemFunctions()
	builder.registerSequenceFunctions()
	builder.registerFullTextSearchFunctions()

	if extraFunctions != nil {
		extraFunctions(builder)
	}

	return builder.Catalogue
}

// FunctionCatalogueBuilder builds a PostgreSQL function catalogue. It is exported so that
// flavour option functions can register additional functions via WithExtraFunctions.
type FunctionCatalogueBuilder struct {
	// CatalogueBuilder provides the shared registration mechanism (Add, Args, NullOnNull,
	// NeverNull, CalledOnNull) and the Catalogue being assembled.
	*engine_shared.CatalogueBuilder

	// Integer is the canonical 4-byte integer type (int4).
	Integer querier_dto.SQLType

	// Bigint is the canonical 8-byte integer type (int8).
	Bigint querier_dto.SQLType

	// Smallint is the canonical 2-byte integer type (int2).
	Smallint querier_dto.SQLType

	// Float4 is the canonical single-precision floating point type.
	Float4 querier_dto.SQLType

	// Float8 is the canonical double-precision floating point type.
	Float8 querier_dto.SQLType

	// Numeric is the canonical arbitrary-precision decimal type.
	Numeric querier_dto.SQLType

	// Boolean is the canonical boolean type.
	Boolean querier_dto.SQLType

	// Text is the canonical variable-length text type.
	Text querier_dto.SQLType

	// Bytea is the canonical raw byte string type.
	Bytea querier_dto.SQLType

	// Timestamp is the canonical timestamp without time zone type.
	Timestamp querier_dto.SQLType

	// Timestamptz is the canonical timestamp with time zone type.
	Timestamptz querier_dto.SQLType

	// Date is the canonical calendar date type.
	Date querier_dto.SQLType

	// Time is the canonical time-of-day type.
	Time querier_dto.SQLType

	// Interval is the canonical interval duration type.
	Interval querier_dto.SQLType

	// JSON is the canonical textual JSON type.
	JSON querier_dto.SQLType

	// JSONB is the canonical binary JSON type.
	JSONB querier_dto.SQLType

	// UUID is the canonical universally unique identifier type.
	UUID querier_dto.SQLType

	// Any is the wildcard placeholder type used for polymorphic arguments.
	Any querier_dto.SQLType
}

// registerMathFunctions registers mathematical functions.
func (b *FunctionCatalogueBuilder) registerMathFunctions() {
	b.NullOnNull("abs", b.Args(Arg{Name: paramNameX, Type: b.Numeric}), b.Numeric)
	b.NullOnNull("ceil", b.Args(Arg{Name: paramNameX, Type: b.Numeric}), b.Numeric)
	b.NullOnNull("ceiling", b.Args(Arg{Name: paramNameX, Type: b.Numeric}), b.Numeric)
	b.NullOnNull("floor", b.Args(Arg{Name: paramNameX, Type: b.Numeric}), b.Numeric)
	b.NullOnNull("round", b.Args(Arg{Name: paramNameX, Type: b.Numeric}), b.Numeric)
	b.NullOnNull("round", b.Args(Arg{Name: paramNameX, Type: b.Numeric}, Arg{Name: "s", Type: b.Integer}), b.Numeric)
	b.NullOnNull("trunc", b.Args(Arg{Name: paramNameX, Type: b.Numeric}), b.Numeric)
	b.NullOnNull("trunc", b.Args(Arg{Name: paramNameX, Type: b.Numeric}, Arg{Name: "s", Type: b.Integer}), b.Numeric)
	b.NullOnNull("sign", b.Args(Arg{Name: paramNameX, Type: b.Numeric}), b.Numeric)
	b.NullOnNull("sqrt", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NullOnNull("cbrt", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NullOnNull("power", b.Args(Arg{Name: "a", Type: b.Numeric}, Arg{Name: "b", Type: b.Numeric}), b.Numeric)
	b.NullOnNull("pow", b.Args(Arg{Name: paramNameA, Type: b.Numeric}, Arg{Name: paramNameB, Type: b.Numeric}), b.Numeric)
	b.NullOnNull("exp", b.Args(Arg{Name: paramNameX, Type: b.Numeric}), b.Numeric)
	b.NullOnNull("ln", b.Args(Arg{Name: paramNameX, Type: b.Numeric}), b.Numeric)
	b.NullOnNull("log", b.Args(Arg{Name: paramNameX, Type: b.Numeric}), b.Numeric)
	b.NullOnNull("log", b.Args(Arg{Name: "base", Type: b.Numeric}, Arg{Name: paramNameX, Type: b.Numeric}), b.Numeric)
	b.NullOnNull("log10", b.Args(Arg{Name: paramNameX, Type: b.Numeric}), b.Numeric)
	b.NullOnNull("mod", b.Args(Arg{Name: paramNameY, Type: b.Numeric}, Arg{Name: paramNameX, Type: b.Numeric}), b.Numeric)
	b.NullOnNull("div", b.Args(Arg{Name: paramNameY, Type: b.Numeric}, Arg{Name: paramNameX, Type: b.Numeric}), b.Numeric)
	b.NullOnNull("degrees", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NullOnNull("radians", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NeverNull("pi", nil, b.Float8)
	b.NeverNull("random", nil, b.Float8)
	b.NullOnNull("setseed", b.Args(Arg{Name: "seed", Type: b.Float8}), b.Float8)
	b.NullOnNull("factorial", b.Args(Arg{Name: paramNameX, Type: b.Bigint}), b.Numeric)
	b.NullOnNull("gcd", b.Args(Arg{Name: paramNameA, Type: b.Bigint}, Arg{Name: paramNameB, Type: b.Bigint}), b.Bigint)
	b.NullOnNull("lcm", b.Args(Arg{Name: paramNameA, Type: b.Bigint}, Arg{Name: paramNameB, Type: b.Bigint}), b.Bigint)
	b.NullOnNull("width_bucket", b.Args(
		Arg{Name: "operand", Type: b.Numeric},
		Arg{Name: "low", Type: b.Numeric},
		Arg{Name: "high", Type: b.Numeric},
		Arg{Name: paramNameCount, Type: b.Integer},
	), b.Integer)
}

// registerTrigonometricFunctions registers trigonometric functions (radian and degree
// variants).
func (b *FunctionCatalogueBuilder) registerTrigonometricFunctions() {
	b.NullOnNull("acos", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NullOnNull("asin", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NullOnNull("atan", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NullOnNull("atan2", b.Args(Arg{Name: paramNameY, Type: b.Float8}, Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NullOnNull("cos", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NullOnNull("sin", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NullOnNull("tan", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NullOnNull("cot", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)

	b.NullOnNull("acosd", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NullOnNull("asind", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NullOnNull("atand", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NullOnNull("atan2d", b.Args(Arg{Name: paramNameY, Type: b.Float8}, Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NullOnNull("cosd", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NullOnNull("sind", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NullOnNull("tand", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NullOnNull("cotd", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
}

// registerStringFunctions registers string manipulation functions.
func (b *FunctionCatalogueBuilder) registerStringFunctions() {
	b.registerStringBasicFunctions()
	b.registerStringVariadicFunctions()
	b.registerStringMiscFunctions()
}

// registerStringBasicFunctions registers basic string scalar functions.
func (b *FunctionCatalogueBuilder) registerStringBasicFunctions() {
	b.NullOnNull("length", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Integer)
	b.NullOnNull("char_length", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Integer)
	b.NullOnNull("octet_length", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Integer)
	b.NullOnNull("bit_length", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Integer)
	b.NullOnNull("lower", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Text)
	b.NullOnNull("upper", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Text)
	b.NullOnNull("trim", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Text)
	b.NullOnNull("ltrim", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Text)
	b.NullOnNull("ltrim", b.Args(Arg{Name: paramNameX, Type: b.Text}, Arg{Name: "characters", Type: b.Text}), b.Text)
	b.NullOnNull("rtrim", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Text)
	b.NullOnNull("rtrim", b.Args(Arg{Name: paramNameX, Type: b.Text}, Arg{Name: "characters", Type: b.Text}), b.Text)
	b.NullOnNull("btrim", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Text)
	b.NullOnNull("btrim", b.Args(Arg{Name: paramNameX, Type: b.Text}, Arg{Name: "characters", Type: b.Text}), b.Text)
	b.NullOnNull("lpad", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: paramNameLength, Type: b.Integer}), b.Text)
	b.NullOnNull("lpad", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: paramNameLength, Type: b.Integer}, Arg{Name: "fill", Type: b.Text}), b.Text)
	b.NullOnNull("rpad", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: paramNameLength, Type: b.Integer}), b.Text)
	b.NullOnNull("rpad", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: paramNameLength, Type: b.Integer}, Arg{Name: "fill", Type: b.Text}), b.Text)
	b.NullOnNull("repeat", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: "number", Type: b.Integer}), b.Text)
	b.NullOnNull("replace", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: "from", Type: b.Text}, Arg{Name: "to", Type: b.Text}), b.Text)
	b.NullOnNull("reverse", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Text)
	b.NullOnNull("substr", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: paramNameStart, Type: b.Integer}), b.Text)
	b.NullOnNull("substr", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: paramNameStart, Type: b.Integer}, Arg{Name: paramNameCount, Type: b.Integer}), b.Text)
	b.NullOnNull("substring", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: paramNameStart, Type: b.Integer}), b.Text)
	b.NullOnNull("substring", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: paramNameStart, Type: b.Integer}, Arg{Name: paramNameCount, Type: b.Integer}), b.Text)
	b.NullOnNull("left", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: paramNameN, Type: b.Integer}), b.Text)
	b.NullOnNull("right", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: paramNameN, Type: b.Integer}), b.Text)
}

// registerStringVariadicFunctions registers variadic string functions.
func (b *FunctionCatalogueBuilder) registerStringVariadicFunctions() {
	b.Add("concat", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameValue, Type: b.Any}),
		ReturnType:        b.Text,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
		IsVariadic:        true,
		MinArguments:      1,
	})
	b.Add("concat_ws", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: "separator", Type: b.Text}, Arg{Name: paramNameValue, Type: b.Any}),
		ReturnType:        b.Text,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
		IsVariadic:        true,
		MinArguments:      2,
	})
	b.Add("format", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: "formatstr", Type: b.Text}, Arg{Name: paramNameValue, Type: b.Any}),
		ReturnType:        b.Text,
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		IsVariadic:        true,
		MinArguments:      1,
	})
}

// registerStringMiscFunctions registers miscellaneous string functions.
func (b *FunctionCatalogueBuilder) registerStringMiscFunctions() {
	b.NullOnNull("initcap", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Text)
	b.NullOnNull("translate", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: "from", Type: b.Text}, Arg{Name: "to", Type: b.Text}), b.Text)
	b.NullOnNull("ascii", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Integer)
	b.NullOnNull("chr", b.Args(Arg{Name: paramNameX, Type: b.Integer}), b.Text)
	b.NullOnNull("encode", b.Args(Arg{Name: "data", Type: b.Bytea}, Arg{Name: paramNameFormat, Type: b.Text}), b.Text)
	b.NullOnNull("decode", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: paramNameFormat, Type: b.Text}), b.Bytea)
	b.NullOnNull("md5", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Text)
	b.NullOnNull("position", b.Args(Arg{Name: paramNameSubstring, Type: b.Text}, Arg{Name: paramNameString, Type: b.Text}), b.Integer)
	b.NullOnNull("strpos", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: paramNameSubstring, Type: b.Text}), b.Integer)
	b.NullOnNull("starts_with", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: "prefix", Type: b.Text}), b.Boolean)
	b.NullOnNull("split_part", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: paramNameDelimiter, Type: b.Text}, Arg{Name: paramNameN, Type: b.Integer}), b.Text)
	b.NullOnNull("quote_ident", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Text)
	b.NullOnNull("quote_literal", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Text)
	b.CalledOnNull("quote_nullable", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Text)
}

// registerDateTimeFunctions registers date and time functions.
func (b *FunctionCatalogueBuilder) registerDateTimeFunctions() {
	b.NeverNull("now", nil, b.Timestamptz)
	b.NeverNull("clock_timestamp", nil, b.Timestamptz)
	b.NeverNull("statement_timestamp", nil, b.Timestamptz)
	b.NeverNull("transaction_timestamp", nil, b.Timestamptz)
	b.NeverNull("timeofday", nil, b.Text)

	b.NullOnNull("age", b.Args(Arg{Name: paramNameTimestamp, Type: b.Timestamptz}, Arg{Name: paramNameTimestamp, Type: b.Timestamptz}), b.Interval)
	b.NullOnNull("age", b.Args(Arg{Name: paramNameTimestamp, Type: b.Timestamptz}), b.Interval)

	b.NullOnNull("date_part", b.Args(Arg{Name: paramNameField, Type: b.Text}, Arg{Name: paramNameSource, Type: b.Timestamptz}), b.Float8)
	b.NullOnNull("date_part", b.Args(Arg{Name: paramNameField, Type: b.Text}, Arg{Name: paramNameSource, Type: b.Interval}), b.Float8)
	b.NullOnNull("date_trunc", b.Args(Arg{Name: paramNameField, Type: b.Text}, Arg{Name: paramNameSource, Type: b.Timestamptz}), b.Timestamptz)
	b.NullOnNull("date_trunc", b.Args(Arg{Name: paramNameField, Type: b.Text}, Arg{Name: paramNameSource, Type: b.Interval}), b.Interval)
	b.NullOnNull("date_trunc", b.Args(Arg{Name: paramNameField, Type: b.Text}, Arg{Name: paramNameSource, Type: b.Timestamptz}, Arg{Name: "timezone", Type: b.Text}), b.Timestamptz)

	b.NullOnNull("extract", b.Args(Arg{Name: paramNameField, Type: b.Text}, Arg{Name: paramNameSource, Type: b.Timestamptz}), b.Numeric)
	b.NullOnNull("extract", b.Args(Arg{Name: paramNameField, Type: b.Text}, Arg{Name: paramNameSource, Type: b.Interval}), b.Numeric)

	b.registerMakeDateTimeFunctions()

	b.NullOnNull("to_timestamp", b.Args(Arg{Name: "epoch", Type: b.Float8}), b.Timestamptz)
	b.NullOnNull("to_timestamp", b.Args(Arg{Name: paramNameText, Type: b.Text}, Arg{Name: paramNameFormat, Type: b.Text}), b.Timestamptz)
	b.NullOnNull("to_date", b.Args(Arg{Name: paramNameText, Type: b.Text}, Arg{Name: paramNameFormat, Type: b.Text}), b.Date)
	b.NullOnNull(funcNameToChar, b.Args(Arg{Name: paramNameTimestamp, Type: b.Timestamptz}, Arg{Name: paramNameFormat, Type: b.Text}), b.Text)
	b.NullOnNull(funcNameToChar, b.Args(Arg{Name: "interval", Type: b.Interval}, Arg{Name: paramNameFormat, Type: b.Text}), b.Text)
	b.NullOnNull(funcNameToChar, b.Args(Arg{Name: paramNameValue, Type: b.Numeric}, Arg{Name: paramNameFormat, Type: b.Text}), b.Text)

	b.NullOnNull("isfinite", b.Args(Arg{Name: paramNameValue, Type: b.Date}), b.Boolean)
	b.NullOnNull("isfinite", b.Args(Arg{Name: paramNameValue, Type: b.Timestamptz}), b.Boolean)
	b.NullOnNull("isfinite", b.Args(Arg{Name: paramNameValue, Type: b.Interval}), b.Boolean)

	b.NullOnNull("justify_days", b.Args(Arg{Name: paramNameValue, Type: b.Interval}), b.Interval)
	b.NullOnNull("justify_hours", b.Args(Arg{Name: paramNameValue, Type: b.Interval}), b.Interval)
	b.NullOnNull("justify_interval", b.Args(Arg{Name: paramNameValue, Type: b.Interval}), b.Interval)
}

// registerMakeDateTimeFunctions registers the make_* date and time constructors.
func (b *FunctionCatalogueBuilder) registerMakeDateTimeFunctions() {
	b.NullOnNull("make_date", b.Args(Arg{Name: paramNameYear, Type: b.Integer}, Arg{Name: paramNameMonth, Type: b.Integer}, Arg{Name: paramNameDay, Type: b.Integer}), b.Date)
	b.NullOnNull("make_time", b.Args(Arg{Name: paramNameHour, Type: b.Integer}, Arg{Name: paramNameMin, Type: b.Integer}, Arg{Name: paramNameSec, Type: b.Float8}), b.Time)
	dateTimeArgs := b.Args(
		Arg{Name: paramNameYear, Type: b.Integer},
		Arg{Name: paramNameMonth, Type: b.Integer},
		Arg{Name: paramNameDay, Type: b.Integer},
		Arg{Name: paramNameHour, Type: b.Integer},
		Arg{Name: paramNameMin, Type: b.Integer},
		Arg{Name: paramNameSec, Type: b.Float8},
	)
	b.NullOnNull("make_timestamp", dateTimeArgs, b.Timestamp)
	b.NullOnNull("make_timestamptz", dateTimeArgs, b.Timestamptz)
	dateTimeWithTimezoneArgs := b.Args(
		Arg{Name: paramNameYear, Type: b.Integer},
		Arg{Name: paramNameMonth, Type: b.Integer},
		Arg{Name: paramNameDay, Type: b.Integer},
		Arg{Name: paramNameHour, Type: b.Integer},
		Arg{Name: paramNameMin, Type: b.Integer},
		Arg{Name: paramNameSec, Type: b.Float8},
		Arg{Name: "timezone", Type: b.Text},
	)
	b.NullOnNull("make_timestamptz", dateTimeWithTimezoneArgs, b.Timestamptz)
	b.NullOnNull("make_interval", b.Args(
		Arg{Name: "years", Type: b.Integer},
		Arg{Name: "months", Type: b.Integer},
		Arg{Name: "weeks", Type: b.Integer},
		Arg{Name: "days", Type: b.Integer},
		Arg{Name: "hours", Type: b.Integer},
		Arg{Name: "mins", Type: b.Integer},
		Arg{Name: "secs", Type: b.Float8},
	), b.Interval)
}

// registerJSONFunctions registers JSON and JSONB functions.
func (b *FunctionCatalogueBuilder) registerJSONFunctions() {
	b.registerJSONBuildFunctions()
	b.registerJSONScalarFunctions()
	b.registerJSONPathFunctions()
	b.registerJSONBMutationFunctions()
}

// registerJSONBuildFunctions registers JSON and JSONB construction functions.
func (b *FunctionCatalogueBuilder) registerJSONBuildFunctions() {
	b.Add("json_build_object", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: "key", Type: b.Any}, Arg{Name: paramNameValue, Type: b.Any}),
		ReturnType:        b.JSON,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
		IsVariadic:        true,
		MinArguments:      0,
	})
	b.Add("json_build_array", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameValue, Type: b.Any}),
		ReturnType:        b.JSON,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
		IsVariadic:        true,
		MinArguments:      0,
	})
	b.Add("jsonb_build_object", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: "key", Type: b.Any}, Arg{Name: paramNameValue, Type: b.Any}),
		ReturnType:        b.JSONB,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
		IsVariadic:        true,
		MinArguments:      0,
	})
	b.Add("jsonb_build_array", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameValue, Type: b.Any}),
		ReturnType:        b.JSONB,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
		IsVariadic:        true,
		MinArguments:      0,
	})
}

// registerJSONScalarFunctions registers scalar JSON conversion functions.
func (b *FunctionCatalogueBuilder) registerJSONScalarFunctions() {
	b.NullOnNull("to_json", b.Args(Arg{Name: paramNameValue, Type: b.Any}), b.JSON)
	b.NullOnNull("to_jsonb", b.Args(Arg{Name: paramNameValue, Type: b.Any}), b.JSONB)
	b.NullOnNull("row_to_json", b.Args(Arg{Name: "record", Type: b.Any}), b.JSON)

	b.NullOnNull("json_array_length", b.Args(Arg{Name: paramNameJSON, Type: b.JSON}), b.Integer)
	b.NullOnNull("jsonb_array_length", b.Args(Arg{Name: paramNameJSON, Type: b.JSONB}), b.Integer)

	b.NullOnNull("json_typeof", b.Args(Arg{Name: paramNameJSON, Type: b.JSON}), b.Text)
	b.NullOnNull("jsonb_typeof", b.Args(Arg{Name: paramNameJSON, Type: b.JSONB}), b.Text)
}

// registerJSONPathFunctions registers JSON path extraction functions.
func (b *FunctionCatalogueBuilder) registerJSONPathFunctions() {
	b.Add("json_extract_path", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: "from_json", Type: b.JSON}, Arg{Name: "path_elem", Type: b.Text}),
		ReturnType:        b.JSON,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      2,
	})
	b.Add("json_extract_path_text", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: "from_json", Type: b.JSON}, Arg{Name: "path_elem", Type: b.Text}),
		ReturnType:        b.Text,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      2,
	})
	b.Add("jsonb_extract_path", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: "from_json", Type: b.JSONB}, Arg{Name: "path_elem", Type: b.Text}),
		ReturnType:        b.JSONB,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      2,
	})
	b.Add("jsonb_extract_path_text", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: "from_json", Type: b.JSONB}, Arg{Name: "path_elem", Type: b.Text}),
		ReturnType:        b.Text,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      2,
	})

	b.NullOnNull("jsonb_path_exists", b.Args(Arg{Name: paramNameTarget, Type: b.JSONB}, Arg{Name: paramNamePath, Type: b.Text}), b.Boolean)
	b.NullOnNull("jsonb_path_match", b.Args(Arg{Name: paramNameTarget, Type: b.JSONB}, Arg{Name: paramNamePath, Type: b.Text}), b.Boolean)
}

// registerJSONBMutationFunctions registers JSONB mutation functions.
func (b *FunctionCatalogueBuilder) registerJSONBMutationFunctions() {
	b.NullOnNull("jsonb_set", b.Args(Arg{Name: paramNameTarget, Type: b.JSONB}, Arg{Name: paramNamePath, Type: b.Text}, Arg{Name: paramNameNewValue, Type: b.JSONB}), b.JSONB)
	b.NullOnNull("jsonb_set", b.Args(
		Arg{Name: paramNameTarget, Type: b.JSONB},
		Arg{Name: paramNamePath, Type: b.Text},
		Arg{Name: paramNameNewValue, Type: b.JSONB},
		Arg{Name: "create_if_missing", Type: b.Boolean},
	), b.JSONB)
	b.NullOnNull("jsonb_insert", b.Args(Arg{Name: paramNameTarget, Type: b.JSONB}, Arg{Name: paramNamePath, Type: b.Text}, Arg{Name: paramNameNewValue, Type: b.JSONB}), b.JSONB)
	b.NullOnNull("jsonb_insert", b.Args(
		Arg{Name: paramNameTarget, Type: b.JSONB},
		Arg{Name: paramNamePath, Type: b.Text},
		Arg{Name: paramNameNewValue, Type: b.JSONB},
		Arg{Name: "insert_after", Type: b.Boolean},
	), b.JSONB)
	b.NullOnNull("jsonb_strip_nulls", b.Args(Arg{Name: paramNameJSON, Type: b.JSONB}), b.JSONB)
	b.NullOnNull("jsonb_pretty", b.Args(Arg{Name: paramNameJSON, Type: b.JSONB}), b.Text)
}

// registerArrayScalarFunctions registers array scalar functions.
func (b *FunctionCatalogueBuilder) registerArrayScalarFunctions() {
	b.NullOnNull("array_append", b.Args(Arg{Name: paramNameArray, Type: b.Any}, Arg{Name: paramNameElement, Type: b.Any}), b.Any)
	b.NullOnNull("array_cat", b.Args(Arg{Name: "array1", Type: b.Any}, Arg{Name: "array2", Type: b.Any}), b.Any)
	b.NullOnNull("array_dims", b.Args(Arg{Name: paramNameArray, Type: b.Any}), b.Text)
	b.NullOnNull("array_fill", b.Args(Arg{Name: paramNameValue, Type: b.Any}, Arg{Name: "dimensions", Type: b.Any}), b.Any)
	b.NullOnNull("array_length", b.Args(Arg{Name: paramNameArray, Type: b.Any}, Arg{Name: "dimension", Type: b.Integer}), b.Integer)
	b.NullOnNull("array_lower", b.Args(Arg{Name: paramNameArray, Type: b.Any}, Arg{Name: "dimension", Type: b.Integer}), b.Integer)
	b.NullOnNull("array_upper", b.Args(Arg{Name: paramNameArray, Type: b.Any}, Arg{Name: "dimension", Type: b.Integer}), b.Integer)
	b.NullOnNull("array_ndims", b.Args(Arg{Name: paramNameArray, Type: b.Any}), b.Integer)
	b.NullOnNull("array_position", b.Args(Arg{Name: paramNameArray, Type: b.Any}, Arg{Name: paramNameElement, Type: b.Any}), b.Integer)
	b.NullOnNull("array_position", b.Args(Arg{Name: paramNameArray, Type: b.Any}, Arg{Name: paramNameElement, Type: b.Any}, Arg{Name: paramNameStart, Type: b.Integer}), b.Integer)
	b.NullOnNull("array_positions", b.Args(Arg{Name: paramNameArray, Type: b.Any}, Arg{Name: paramNameElement, Type: b.Any}), b.Any)
	b.NullOnNull("array_prepend", b.Args(Arg{Name: paramNameElement, Type: b.Any}, Arg{Name: paramNameArray, Type: b.Any}), b.Any)
	b.NullOnNull("array_remove", b.Args(Arg{Name: paramNameArray, Type: b.Any}, Arg{Name: paramNameElement, Type: b.Any}), b.Any)
	b.NullOnNull("array_replace", b.Args(Arg{Name: paramNameArray, Type: b.Any}, Arg{Name: "from", Type: b.Any}, Arg{Name: "to", Type: b.Any}), b.Any)
	b.NullOnNull("array_to_string", b.Args(Arg{Name: paramNameArray, Type: b.Any}, Arg{Name: paramNameDelimiter, Type: b.Text}), b.Text)
	b.NullOnNull("array_to_string", b.Args(Arg{Name: paramNameArray, Type: b.Any}, Arg{Name: paramNameDelimiter, Type: b.Text}, Arg{Name: "null_string", Type: b.Text}), b.Text)
	b.NullOnNull("string_to_array", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: paramNameDelimiter, Type: b.Text}), b.Any)
	b.NullOnNull("string_to_array", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: paramNameDelimiter, Type: b.Text}, Arg{Name: "null_string", Type: b.Text}), b.Any)
	b.NullOnNull("cardinality", b.Args(Arg{Name: paramNameArray, Type: b.Any}), b.Integer)
}

// registerAggregateFunctions registers aggregate functions.
func (b *FunctionCatalogueBuilder) registerAggregateFunctions() {
	b.registerCountAggregates()
	b.registerBooleanAndBitwiseAggregates()
	b.registerCollectionAggregates()
	b.registerJSONAggregates()
	b.registerOrderedSetAggregates()
	b.registerStatisticalAggregates()
}

// registerCountAggregates registers count aggregate functions.
func (b *FunctionCatalogueBuilder) registerCountAggregates() {
	b.Add("count", &querier_dto.FunctionSignature{
		ReturnType:        b.Bigint,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})
	b.Add("count", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.Any}),
		ReturnType:        b.Bigint,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})
}

// registerBooleanAndBitwiseAggregates registers boolean and bitwise aggregates.
//
//nolint:dupl // structurally similar aggregates
func (b *FunctionCatalogueBuilder) registerBooleanAndBitwiseAggregates() {
	b.Add("bool_and", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.Boolean}),
		ReturnType:        b.Boolean,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("bool_or", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.Boolean}),
		ReturnType:        b.Boolean,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("every", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.Boolean}),
		ReturnType:        b.Boolean,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})

	b.Add("bit_and", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.Any}),
		ReturnType:        b.Any,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("bit_or", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.Any}),
		ReturnType:        b.Any,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("bit_xor", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.Any}),
		ReturnType:        b.Any,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
}

// registerCollectionAggregates registers string and array collection aggregates.
func (b *FunctionCatalogueBuilder) registerCollectionAggregates() {
	b.Add("string_agg", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameExpression, Type: b.Text}, Arg{Name: paramNameDelimiter, Type: b.Text}),
		ReturnType:        b.Text,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("array_agg", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameExpression, Type: b.Any}),
		ReturnType:        b.Any,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
}

// registerJSONAggregates registers JSON and JSONB aggregate functions.
func (b *FunctionCatalogueBuilder) registerJSONAggregates() {
	b.Add("json_agg", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameExpression, Type: b.Any}),
		ReturnType:        b.JSON,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("jsonb_agg", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameExpression, Type: b.Any}),
		ReturnType:        b.JSONB,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("json_object_agg", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: "key", Type: b.Any}, Arg{Name: paramNameValue, Type: b.Any}),
		ReturnType:        b.JSON,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("jsonb_object_agg", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: "key", Type: b.Any}, Arg{Name: paramNameValue, Type: b.Any}),
		ReturnType:        b.JSONB,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
}

// registerOrderedSetAggregates registers ordered-set aggregate functions.
func (b *FunctionCatalogueBuilder) registerOrderedSetAggregates() {
	b.Add("percentile_cont", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: "fraction", Type: b.Float8}),
		ReturnType:        b.Float8,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("percentile_disc", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: "fraction", Type: b.Float8}),
		ReturnType:        b.Any,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("mode", &querier_dto.FunctionSignature{
		ReturnType:        b.Any,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
}

// registerStatisticalAggregates registers statistical aggregate functions.
//
//nolint:dupl // structurally similar aggregates
func (b *FunctionCatalogueBuilder) registerStatisticalAggregates() {
	b.Add("stddev", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.Any}),
		ReturnType:        b.Numeric,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("stddev_pop", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.Any}),
		ReturnType:        b.Numeric,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("stddev_samp", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.Any}),
		ReturnType:        b.Numeric,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("variance", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.Any}),
		ReturnType:        b.Numeric,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("var_pop", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.Any}),
		ReturnType:        b.Numeric,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("var_samp", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.Any}),
		ReturnType:        b.Numeric,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
}

// registerWindowFunctions registers window functions.
func (b *FunctionCatalogueBuilder) registerWindowFunctions() {
	b.NeverNull("row_number", nil, b.Bigint)
	b.NeverNull("rank", nil, b.Bigint)
	b.NeverNull("dense_rank", nil, b.Bigint)
	b.NeverNull("ntile", b.Args(Arg{Name: paramNameN, Type: b.Integer}), b.Integer)
	b.NeverNull("cume_dist", nil, b.Float8)
	b.NeverNull("percent_rank", nil, b.Float8)

	windowValueArgs := b.Args(Arg{Name: paramNameExpression, Type: b.Any}, Arg{Name: "offset", Type: b.Integer}, Arg{Name: "default", Type: b.Any})

	b.Add("lag", &querier_dto.FunctionSignature{
		Arguments:         windowValueArgs,
		ReturnType:        b.Any,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		MinArguments:      1,
	})
	b.Add("lead", &querier_dto.FunctionSignature{
		Arguments:         windowValueArgs,
		ReturnType:        b.Any,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		MinArguments:      1,
	})

	b.CalledOnNull("first_value", b.Args(Arg{Name: paramNameExpression, Type: b.Any}), b.Any)
	b.CalledOnNull("last_value", b.Args(Arg{Name: paramNameExpression, Type: b.Any}), b.Any)
	b.CalledOnNull("nth_value", b.Args(Arg{Name: paramNameExpression, Type: b.Any}, Arg{Name: paramNameN, Type: b.Integer}), b.Any)
}

// registerConditionalFunctions registers conditional expression functions.
func (b *FunctionCatalogueBuilder) registerConditionalFunctions() {
	b.CalledOnNull("nullif", b.Args(Arg{Name: "value1", Type: b.Any}, Arg{Name: "value2", Type: b.Any}), b.Any)
	b.Add("greatest", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameValue, Type: b.Any}),
		ReturnType:        b.Any,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      1,
	})
	b.Add("least", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameValue, Type: b.Any}),
		ReturnType:        b.Any,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      1,
	})
}

// registerTypeConversionFunctions registers type conversion functions.
func (b *FunctionCatalogueBuilder) registerTypeConversionFunctions() {
	b.NullOnNull(funcNameToChar, b.Args(Arg{Name: paramNameValue, Type: b.Integer}, Arg{Name: paramNameFormat, Type: b.Text}), b.Text)
	b.NullOnNull("to_number", b.Args(Arg{Name: paramNameText, Type: b.Text}, Arg{Name: paramNameFormat, Type: b.Text}), b.Numeric)
}

// registerFullTextSearchFunctions registers the PostgreSQL full-text search helpers used
// to build, normalise, score, and excerpt tsvector / tsquery values. The catalogue treats
// tsvector and tsquery as text-categorised opaque types; ts_rank / ts_rank_cd /
// similarity return float4.
func (b *FunctionCatalogueBuilder) registerFullTextSearchFunctions() {
	tsvectorType := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "tsvector"}
	tsqueryType := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "tsquery"}

	b.NullOnNull("to_tsvector", b.Args(Arg{Name: paramNameX, Type: b.Text}), tsvectorType)
	b.NullOnNull("to_tsvector", b.Args(Arg{Name: paramNameConfig, Type: b.Text}, Arg{Name: paramNameX, Type: b.Text}), tsvectorType)
	b.NullOnNull("to_tsquery", b.Args(Arg{Name: paramNameX, Type: b.Text}), tsqueryType)
	b.NullOnNull("to_tsquery", b.Args(Arg{Name: paramNameConfig, Type: b.Text}, Arg{Name: paramNameX, Type: b.Text}), tsqueryType)
	b.NullOnNull("plainto_tsquery", b.Args(Arg{Name: paramNameX, Type: b.Text}), tsqueryType)
	b.NullOnNull("plainto_tsquery", b.Args(Arg{Name: paramNameConfig, Type: b.Text}, Arg{Name: paramNameX, Type: b.Text}), tsqueryType)
	b.NullOnNull("phraseto_tsquery", b.Args(Arg{Name: paramNameX, Type: b.Text}), tsqueryType)
	b.NullOnNull("phraseto_tsquery", b.Args(Arg{Name: paramNameConfig, Type: b.Text}, Arg{Name: paramNameX, Type: b.Text}), tsqueryType)
	b.NullOnNull("websearch_to_tsquery", b.Args(Arg{Name: paramNameX, Type: b.Text}), tsqueryType)
	b.NullOnNull("websearch_to_tsquery", b.Args(Arg{Name: paramNameConfig, Type: b.Text}, Arg{Name: paramNameX, Type: b.Text}), tsqueryType)

	b.NullOnNull("ts_rank", b.Args(Arg{Name: paramNameVector, Type: tsvectorType}, Arg{Name: paramNameQuery, Type: tsqueryType}), b.Float4)
	b.NullOnNull("ts_rank", b.Args(Arg{Name: paramNameVector, Type: tsvectorType}, Arg{Name: paramNameQuery, Type: tsqueryType}, Arg{Name: "normalisation", Type: b.Integer}), b.Float4)
	b.NullOnNull("ts_rank_cd", b.Args(Arg{Name: paramNameVector, Type: tsvectorType}, Arg{Name: paramNameQuery, Type: tsqueryType}), b.Float4)
	b.NullOnNull("ts_rank_cd", b.Args(Arg{Name: paramNameVector, Type: tsvectorType}, Arg{Name: paramNameQuery, Type: tsqueryType}, Arg{Name: "normalisation", Type: b.Integer}), b.Float4)

	b.NullOnNull(funcNameTSHeadline, b.Args(Arg{Name: paramNameDocument, Type: b.Text}, Arg{Name: paramNameQuery, Type: tsqueryType}), b.Text)
	b.NullOnNull(funcNameTSHeadline, b.Args(Arg{Name: paramNameConfig, Type: b.Text}, Arg{Name: paramNameDocument, Type: b.Text}, Arg{Name: paramNameQuery, Type: tsqueryType}), b.Text)
	b.NullOnNull(funcNameTSHeadline, b.Args(
		Arg{Name: paramNameConfig, Type: b.Text},
		Arg{Name: paramNameDocument, Type: b.Text},
		Arg{Name: paramNameQuery, Type: tsqueryType},
		Arg{Name: "options", Type: b.Text},
	), b.Text)
	b.NullOnNull(funcNameTSHeadline, b.Args(Arg{Name: paramNameDocument, Type: b.Text}, Arg{Name: paramNameQuery, Type: tsqueryType}, Arg{Name: "options", Type: b.Text}), b.Text)

	b.NullOnNull("setweight", b.Args(Arg{Name: paramNameVector, Type: tsvectorType}, Arg{Name: "weight", Type: b.Text}), tsvectorType)
	b.NullOnNull("strip", b.Args(Arg{Name: paramNameVector, Type: tsvectorType}), tsvectorType)
	b.NullOnNull("length", b.Args(Arg{Name: paramNameVector, Type: tsvectorType}), b.Integer)
	b.NullOnNull("numnode", b.Args(Arg{Name: paramNameQuery, Type: tsqueryType}), b.Integer)
	b.NullOnNull("querytree", b.Args(Arg{Name: paramNameQuery, Type: tsqueryType}), b.Text)

	b.NullOnNull("similarity", b.Args(Arg{Name: paramNameA, Type: b.Text}, Arg{Name: paramNameB, Type: b.Text}), b.Float4)
	b.NullOnNull("word_similarity", b.Args(Arg{Name: paramNameA, Type: b.Text}, Arg{Name: paramNameB, Type: b.Text}), b.Float4)
	b.NullOnNull("strict_word_similarity", b.Args(Arg{Name: paramNameA, Type: b.Text}, Arg{Name: paramNameB, Type: b.Text}), b.Float4)
}

// registerSystemFunctions registers system information functions.
func (b *FunctionCatalogueBuilder) registerSystemFunctions() {
	b.NeverNull("current_database", nil, b.Text)
	b.NeverNull("current_schema", nil, b.Text)
	b.NeverNull("current_user", nil, b.Text)
	b.NeverNull("session_user", nil, b.Text)
	b.NeverNull("version", nil, b.Text)

	b.NullOnNull("pg_typeof", b.Args(Arg{Name: paramNameValue, Type: b.Any}), b.Text)
	b.NullOnNull("pg_column_size", b.Args(Arg{Name: paramNameValue, Type: b.Any}), b.Integer)
	b.NullOnNull("pg_table_size", b.Args(Arg{Name: "table", Type: b.Text}), b.Bigint)
	b.NullOnNull("pg_total_relation_size", b.Args(Arg{Name: "table", Type: b.Text}), b.Bigint)
	b.CalledOnNull("obj_description", b.Args(Arg{Name: "oid", Type: b.Integer}, Arg{Name: "catalog", Type: b.Text}), b.Text)
	b.CalledOnNull("obj_description", b.Args(Arg{Name: "oid", Type: b.Integer}), b.Text)
}

// registerSequenceFunctions registers the sequence manipulation functions.
//
// nextval and setval advance or reset sequence state, so they write database state and
// are declared via ModifiesData to route a query projecting them to the writer
// connection. currval and lastval only read session state with no write, so they stay
// read-only. The sequence argument is a regclass in Postgres but is passed as a
// text/unknown literal such as nextval('s'), so a text-typed argument resolves the call.
func (b *FunctionCatalogueBuilder) registerSequenceFunctions() {
	b.ModifiesData("nextval", b.Args(Arg{Name: paramNameSequence, Type: b.Text}), b.Bigint)
	b.ModifiesData("setval", b.Args(
		Arg{Name: paramNameSequence, Type: b.Text},
		Arg{Name: paramNameValue, Type: b.Bigint},
	), b.Bigint)
	b.ModifiesData("setval", b.Args(
		Arg{Name: paramNameSequence, Type: b.Text},
		Arg{Name: paramNameValue, Type: b.Bigint},
		Arg{Name: "is_called", Type: b.Boolean},
	), b.Bigint)
	b.NeverNull("currval", b.Args(Arg{Name: paramNameSequence, Type: b.Text}), b.Bigint)
	b.NeverNull("lastval", nil, b.Bigint)
}
