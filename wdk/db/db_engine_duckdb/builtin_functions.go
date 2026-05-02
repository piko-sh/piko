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

package db_engine_duckdb

import (
	"piko.sh/piko/internal/querier/querier_adapters/engine_shared"
	"piko.sh/piko/internal/querier/querier_dto"
)

// buildFunctionCatalogue assembles the DuckDB builtin function catalogue.
//
// Takes extraFunctions (func(*FunctionCatalogueBuilder)) which optionally registers
// additional flavour-specific functions on the builder.
//
// Returns *querier_dto.FunctionCatalogue which holds every registered function signature
// keyed by name.
func buildFunctionCatalogue(extraFunctions func(*FunctionCatalogueBuilder)) *querier_dto.FunctionCatalogue {
	builder := &FunctionCatalogueBuilder{
		CatalogueBuilder: engine_shared.NewCatalogueBuilder(),
		Integer:          querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"},
		Bigint:           querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int8"},
		Smallint:         querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int2"},
		Hugeint:          querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "hugeint"},
		Float4:           querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "float4"},
		Float8:           querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "float8"},
		Numeric:          querier_dto.SQLType{Category: querier_dto.TypeCategoryDecimal, EngineName: "numeric"},
		Boolean:          querier_dto.SQLType{Category: querier_dto.TypeCategoryBoolean, EngineName: "bool"},
		Text:             querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "varchar"},
		Bytea:            querier_dto.SQLType{Category: querier_dto.TypeCategoryBytea, EngineName: "blob"},
		Timestamp:        querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "timestamp"},
		Timestamptz:      querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "timestamptz"},
		Date:             querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "date"},
		Time:             querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "time"},
		Interval:         querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "interval"},
		JSON:             querier_dto.SQLType{Category: querier_dto.TypeCategoryJSON, EngineName: "json"},
		UUID:             querier_dto.SQLType{Category: querier_dto.TypeCategoryUUID, EngineName: "uuid"},
		Any:              querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: ""},
	}

	builder.registerMathFunctions()
	builder.registerTrigonometricFunctions()
	builder.registerStringFunctions()
	builder.registerDateTimeFunctions()
	builder.registerJSONFunctions()
	builder.registerListFunctions()
	builder.registerAggregateFunctions()
	builder.registerWindowFunctions()
	builder.registerConditionalFunctions()
	builder.registerTypeConversionFunctions()
	builder.registerSystemFunctions()
	builder.registerDuckDBSpecificFunctions()

	if extraFunctions != nil {
		extraFunctions(builder)
	}

	return builder.Catalogue
}

// FunctionCatalogueBuilder builds a DuckDB function catalogue. It is exported so that
// flavour option functions can register additional functions via WithExtraFunctions.
type FunctionCatalogueBuilder struct {
	// CatalogueBuilder provides the shared registration mechanism (Add, Args, NullOnNull,
	// NeverNull, CalledOnNull) and the Catalogue being assembled.
	*engine_shared.CatalogueBuilder

	// Integer is the 32-bit signed integer SQL type.
	Integer querier_dto.SQLType

	// Bigint is the 64-bit signed integer SQL type.
	Bigint querier_dto.SQLType

	// Smallint is the 16-bit signed integer SQL type.
	Smallint querier_dto.SQLType

	// Hugeint is the 128-bit signed integer SQL type.
	Hugeint querier_dto.SQLType

	// Float4 is the single-precision floating-point SQL type.
	Float4 querier_dto.SQLType

	// Float8 is the double-precision floating-point SQL type.
	Float8 querier_dto.SQLType

	// Numeric is the arbitrary-precision decimal SQL type.
	Numeric querier_dto.SQLType

	// Boolean is the boolean SQL type.
	Boolean querier_dto.SQLType

	// Text is the variable-length text SQL type.
	Text querier_dto.SQLType

	// Bytea is the binary blob SQL type.
	Bytea querier_dto.SQLType

	// Timestamp is the timestamp without time zone SQL type.
	Timestamp querier_dto.SQLType

	// Timestamptz is the timestamp with time zone SQL type.
	Timestamptz querier_dto.SQLType

	// Date is the calendar date SQL type.
	Date querier_dto.SQLType

	// Time is the time-of-day SQL type.
	Time querier_dto.SQLType

	// Interval is the time interval SQL type.
	Interval querier_dto.SQLType

	// JSON is the JSON document SQL type.
	JSON querier_dto.SQLType

	// UUID is the universally unique identifier SQL type.
	UUID querier_dto.SQLType

	// Any is the wildcard SQL type used for polymorphic arguments.
	Any querier_dto.SQLType
}

// Arg names a single function argument and its SQL type. It aliases the shared toolkit's
// Arg so existing call sites keep using the bare name; the registration methods (Add,
// Args, NullOnNull, NeverNull, CalledOnNull) are promoted from the embedded shared
// builder.
type Arg = engine_shared.Arg

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
	b.NullOnNull("log2", b.Args(Arg{Name: paramNameX, Type: b.Numeric}), b.Numeric)
	b.NullOnNull("mod", b.Args(Arg{Name: paramNameY, Type: b.Numeric}, Arg{Name: paramNameX, Type: b.Numeric}), b.Numeric)
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
	b.NullOnNull("even", b.Args(Arg{Name: paramNameX, Type: b.Numeric}), b.Numeric)
	b.NullOnNull("isnan", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Boolean)
	b.NullOnNull("isinf", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Boolean)
	b.NullOnNull("isfinite", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Boolean)
	b.NullOnNull("bit_count", b.Args(Arg{Name: paramNameX, Type: b.Any}), b.Bigint)
}

// registerTrigonometricFunctions registers trigonometric functions.
func (b *FunctionCatalogueBuilder) registerTrigonometricFunctions() {
	b.NullOnNull("acos", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NullOnNull("asin", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NullOnNull("atan", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NullOnNull("atan2", b.Args(Arg{Name: paramNameY, Type: b.Float8}, Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NullOnNull("cos", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NullOnNull("sin", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NullOnNull("tan", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
	b.NullOnNull("cot", b.Args(Arg{Name: paramNameX, Type: b.Float8}), b.Float8)
}

// registerStringFunctions registers string manipulation functions.
func (b *FunctionCatalogueBuilder) registerStringFunctions() {
	b.registerStringBasicFunctions()
	b.registerStringVariadicFunctions()
	b.registerStringMiscFunctions()
}

// registerStringBasicFunctions registers fixed-arity string functions.
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
	b.Add("printf", &querier_dto.FunctionSignature{
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
	b.NullOnNull("ends_with", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: "suffix", Type: b.Text}), b.Boolean)
	b.NullOnNull("contains", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: "search_string", Type: b.Text}), b.Boolean)
	b.NullOnNull("split_part", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: paramNameDelimiter, Type: b.Text}, Arg{Name: paramNameN, Type: b.Integer}), b.Text)
	b.NullOnNull("regexp_extract", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: "pattern", Type: b.Text}), b.Text)
	b.NullOnNull("regexp_replace", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: "pattern", Type: b.Text}, Arg{Name: "replacement", Type: b.Text}), b.Text)
	b.NullOnNull("regexp_full_match", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: "pattern", Type: b.Text}), b.Boolean)
}

// registerDateTimeFunctions registers date and time functions.
func (b *FunctionCatalogueBuilder) registerDateTimeFunctions() {
	b.NeverNull("now", nil, b.Timestamptz)
	b.NeverNull("current_timestamp", nil, b.Timestamptz)
	b.NeverNull("current_date", nil, b.Date)
	b.NeverNull("current_time", nil, b.Time)

	b.NullOnNull("age", b.Args(Arg{Name: paramNameTimestamp, Type: b.Timestamptz}, Arg{Name: paramNameTimestamp, Type: b.Timestamptz}), b.Interval)
	b.NullOnNull("age", b.Args(Arg{Name: paramNameTimestamp, Type: b.Timestamptz}), b.Interval)

	b.NullOnNull("date_part", b.Args(Arg{Name: paramNameField, Type: b.Text}, Arg{Name: paramNameSource, Type: b.Timestamptz}), b.Float8)
	b.NullOnNull("date_part", b.Args(Arg{Name: paramNameField, Type: b.Text}, Arg{Name: paramNameSource, Type: b.Interval}), b.Float8)
	b.NullOnNull("date_trunc", b.Args(Arg{Name: paramNameField, Type: b.Text}, Arg{Name: paramNameSource, Type: b.Timestamptz}), b.Timestamptz)
	b.NullOnNull("date_trunc", b.Args(Arg{Name: paramNameField, Type: b.Text}, Arg{Name: paramNameSource, Type: b.Interval}), b.Interval)

	b.NullOnNull("extract", b.Args(Arg{Name: paramNameField, Type: b.Text}, Arg{Name: paramNameSource, Type: b.Timestamptz}), b.Numeric)
	b.NullOnNull("extract", b.Args(Arg{Name: paramNameField, Type: b.Text}, Arg{Name: paramNameSource, Type: b.Interval}), b.Numeric)

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

	b.NullOnNull("to_timestamp", b.Args(Arg{Name: "epoch", Type: b.Float8}), b.Timestamptz)
	b.NullOnNull("to_timestamp", b.Args(Arg{Name: paramNameText, Type: b.Text}, Arg{Name: paramNameFormat, Type: b.Text}), b.Timestamptz)
	b.NullOnNull("to_date", b.Args(Arg{Name: paramNameText, Type: b.Text}, Arg{Name: paramNameFormat, Type: b.Text}), b.Date)
	b.NullOnNull(funcNameToChar, b.Args(Arg{Name: paramNameTimestamp, Type: b.Timestamptz}, Arg{Name: paramNameFormat, Type: b.Text}), b.Text)
	b.NullOnNull(funcNameToChar, b.Args(Arg{Name: "interval", Type: b.Interval}, Arg{Name: paramNameFormat, Type: b.Text}), b.Text)
	b.NullOnNull(funcNameToChar, b.Args(Arg{Name: paramNameValue, Type: b.Numeric}, Arg{Name: paramNameFormat, Type: b.Text}), b.Text)

	b.NullOnNull("strftime", b.Args(Arg{Name: paramNameFormat, Type: b.Text}, Arg{Name: paramNameTimestamp, Type: b.Timestamptz}), b.Text)
	b.NullOnNull("strftime", b.Args(Arg{Name: paramNameFormat, Type: b.Text}, Arg{Name: paramNameTimestamp, Type: b.Timestamp}), b.Text)
	b.NullOnNull("strptime", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: paramNameFormat, Type: b.Text}), b.Timestamp)

	b.NullOnNull("epoch_ms", b.Args(Arg{Name: paramNameValue, Type: b.Bigint}), b.Timestamp)
	b.NullOnNull("epoch_us", b.Args(Arg{Name: paramNameValue, Type: b.Bigint}), b.Timestamp)
	b.NullOnNull("epoch_ns", b.Args(Arg{Name: paramNameValue, Type: b.Bigint}), b.Timestamp)

	b.NullOnNull("last_day", b.Args(Arg{Name: paramNameValue, Type: b.Date}), b.Date)
	b.NullOnNull("dayname", b.Args(Arg{Name: paramNameValue, Type: b.Date}), b.Text)
	b.NullOnNull("monthname", b.Args(Arg{Name: paramNameValue, Type: b.Date}), b.Text)
}

// registerJSONFunctions registers JSON functions.
func (b *FunctionCatalogueBuilder) registerJSONFunctions() {
	b.NullOnNull("to_json", b.Args(Arg{Name: paramNameValue, Type: b.Any}), b.JSON)
	b.NullOnNull("json_array_length", b.Args(Arg{Name: paramNameJSON, Type: b.JSON}), b.Integer)
	b.NullOnNull("json_type", b.Args(Arg{Name: paramNameJSON, Type: b.JSON}), b.Text)
	b.NullOnNull("json_valid", b.Args(Arg{Name: paramNameJSON, Type: b.Text}), b.Boolean)

	b.NullOnNull("json_extract", b.Args(Arg{Name: paramNameJSON, Type: b.JSON}, Arg{Name: paramNamePath, Type: b.Text}), b.JSON)
	b.NullOnNull("json_extract_string", b.Args(Arg{Name: paramNameJSON, Type: b.JSON}, Arg{Name: paramNamePath, Type: b.Text}), b.Text)

	b.Add("json_object", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: "key", Type: b.Any}, Arg{Name: paramNameValue, Type: b.Any}),
		ReturnType:        b.JSON,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
		IsVariadic:        true,
		MinArguments:      0,
	})
	b.Add("json_array", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameValue, Type: b.Any}),
		ReturnType:        b.JSON,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
		IsVariadic:        true,
		MinArguments:      0,
	})

	b.NullOnNull("json_merge_patch", b.Args(Arg{Name: paramNameTarget, Type: b.JSON}, Arg{Name: "patch", Type: b.JSON}), b.JSON)
	b.NullOnNull("json_keys", b.Args(Arg{Name: paramNameJSON, Type: b.JSON}), b.Any)
}

// registerListFunctions registers DuckDB list (array) functions.
func (b *FunctionCatalogueBuilder) registerListFunctions() {
	b.Add("list_value", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameValue, Type: b.Any}),
		ReturnType:        b.Any,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
		IsVariadic:        true,
		MinArguments:      0,
	})
	b.NullOnNull("list_sort", b.Args(Arg{Name: paramNameList, Type: b.Any}), b.Any)
	b.NullOnNull("list_reverse_sort", b.Args(Arg{Name: paramNameList, Type: b.Any}), b.Any)
	b.NullOnNull("list_distinct", b.Args(Arg{Name: paramNameList, Type: b.Any}), b.Any)
	b.NullOnNull("list_unique", b.Args(Arg{Name: paramNameList, Type: b.Any}), b.Bigint)
	b.NullOnNull("list_contains", b.Args(Arg{Name: paramNameList, Type: b.Any}, Arg{Name: paramNameElement, Type: b.Any}), b.Boolean)
	b.NullOnNull("list_aggregate", b.Args(Arg{Name: paramNameList, Type: b.Any}, Arg{Name: "name", Type: b.Text}), b.Any)

	b.NullOnNull("array_length", b.Args(Arg{Name: paramNameArray, Type: b.Any}), b.Integer)
	b.NullOnNull("array_append", b.Args(Arg{Name: paramNameArray, Type: b.Any}, Arg{Name: paramNameElement, Type: b.Any}), b.Any)
	b.NullOnNull("array_cat", b.Args(Arg{Name: "array1", Type: b.Any}, Arg{Name: "array2", Type: b.Any}), b.Any)
	b.NullOnNull("array_position", b.Args(Arg{Name: paramNameArray, Type: b.Any}, Arg{Name: paramNameElement, Type: b.Any}), b.Integer)
	b.NullOnNull("array_to_string", b.Args(Arg{Name: paramNameArray, Type: b.Any}, Arg{Name: paramNameDelimiter, Type: b.Text}), b.Text)
	b.NullOnNull("string_to_array", b.Args(Arg{Name: paramNameString, Type: b.Text}, Arg{Name: paramNameDelimiter, Type: b.Text}), b.Any)
	b.NullOnNull("cardinality", b.Args(Arg{Name: paramNameArray, Type: b.Any}), b.Integer)
	b.NullOnNull("flatten", b.Args(Arg{Name: paramNameList, Type: b.Any}), b.Any)
	b.NullOnNull("unnest", b.Args(Arg{Name: paramNameList, Type: b.Any}), b.Any)
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

// registerCountAggregates registers the count aggregate variants.
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

// registerCollectionAggregates registers list and string collection aggregates.
func (b *FunctionCatalogueBuilder) registerCollectionAggregates() {
	b.Add("string_agg", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameExpression, Type: b.Text}, Arg{Name: paramNameDelimiter, Type: b.Text}),
		ReturnType:        b.Text,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("list", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameExpression, Type: b.Any}),
		ReturnType:        b.Any,
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

// registerJSONAggregates registers JSON aggregate constructors.
func (b *FunctionCatalogueBuilder) registerJSONAggregates() {
	b.Add("json_group_array", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameExpression, Type: b.Any}),
		ReturnType:        b.JSON,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("json_group_object", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: "key", Type: b.Any}, Arg{Name: paramNameValue, Type: b.Any}),
		ReturnType:        b.JSON,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
}

// registerOrderedSetAggregates registers percentile and ordered-set aggregates.
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
	b.Add("median", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.Any}),
		ReturnType:        b.Any,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("quantile_cont", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.Any}, Arg{Name: "fraction", Type: b.Float8}),
		ReturnType:        b.Any,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("quantile_disc", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.Any}, Arg{Name: "fraction", Type: b.Float8}),
		ReturnType:        b.Any,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
}

// registerStatisticalAggregates registers variance and stddev aggregates.
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
	b.Add("ifnull", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameExpression, Type: b.Any}, Arg{Name: "alternative", Type: b.Any}),
		ReturnType:        b.Any,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
}

// registerTypeConversionFunctions registers type conversion functions.
func (b *FunctionCatalogueBuilder) registerTypeConversionFunctions() {
	b.NullOnNull(funcNameToChar, b.Args(Arg{Name: paramNameValue, Type: b.Integer}, Arg{Name: paramNameFormat, Type: b.Text}), b.Text)
	b.NullOnNull("to_number", b.Args(Arg{Name: paramNameText, Type: b.Text}, Arg{Name: paramNameFormat, Type: b.Text}), b.Numeric)
}

// registerSystemFunctions registers system information functions.
func (b *FunctionCatalogueBuilder) registerSystemFunctions() {
	b.NeverNull("current_database", nil, b.Text)
	b.NeverNull("current_schema", nil, b.Text)
	b.NeverNull("current_user", nil, b.Text)
	b.NeverNull("version", nil, b.Text)
}

// registerDuckDBSpecificFunctions registers DuckDB-specific utility functions.
func (b *FunctionCatalogueBuilder) registerDuckDBSpecificFunctions() {
	b.NullOnNull("typeof", b.Args(Arg{Name: paramNameValue, Type: b.Any}), b.Text)
	b.NullOnNull("hash", b.Args(Arg{Name: paramNameValue, Type: b.Any}), b.Bigint)

	b.ModifiesData("nextval", b.Args(Arg{Name: paramNameSequenceName, Type: b.Text}), b.Bigint)

	b.Add("generate_series", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameStart, Type: b.Bigint}, Arg{Name: paramNameStop, Type: b.Bigint}, Arg{Name: paramNameStep, Type: b.Bigint}),
		ReturnType:        b.Bigint,
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		ReturnsSet:        true,
		MinArguments:      2,
	})
	b.Add("range", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameStart, Type: b.Bigint}, Arg{Name: paramNameStop, Type: b.Bigint}, Arg{Name: paramNameStep, Type: b.Bigint}),
		ReturnType:        b.Bigint,
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		ReturnsSet:        true,
		MinArguments:      2,
	})
}
