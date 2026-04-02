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
	"piko.sh/piko/internal/querier/querier_dto"
)

func buildFunctionCatalogue(extraFunctions func(*FunctionCatalogueBuilder)) *querier_dto.FunctionCatalogue {
	builder := &FunctionCatalogueBuilder{
		Catalogue: &querier_dto.FunctionCatalogue{
			Functions: make(map[string][]*querier_dto.FunctionSignature),
		},
		Integer:     querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"},
		Bigint:      querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int8"},
		Smallint:    querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int2"},
		Float4:      querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "float4"},
		Float8:      querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "float8"},
		Numeric:     querier_dto.SQLType{Category: querier_dto.TypeCategoryDecimal, EngineName: "numeric"},
		Boolean:     querier_dto.SQLType{Category: querier_dto.TypeCategoryBoolean, EngineName: "bool"},
		Text:        querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
		Bytea:       querier_dto.SQLType{Category: querier_dto.TypeCategoryBytea, EngineName: "bytea"},
		Timestamp:   querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "timestamp"},
		Timestamptz: querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "timestamptz"},
		Date:        querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "date"},
		Time:        querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "time"},
		Interval:    querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "interval"},
		JSON:        querier_dto.SQLType{Category: querier_dto.TypeCategoryJSON, EngineName: "json"},
		JSONB:       querier_dto.SQLType{Category: querier_dto.TypeCategoryJSON, EngineName: "jsonb"},
		UUID:        querier_dto.SQLType{Category: querier_dto.TypeCategoryUUID, EngineName: "uuid"},
		Any:         querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: ""},
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

	if extraFunctions != nil {
		extraFunctions(builder)
	}

	return builder.Catalogue
}

// FunctionCatalogueBuilder builds a PostgreSQL function catalogue. It is
// exported so that flavour option functions can register additional functions
// via WithExtraFunctions.
type FunctionCatalogueBuilder struct {
	Catalogue *querier_dto.FunctionCatalogue

	Integer querier_dto.SQLType

	Bigint querier_dto.SQLType

	Smallint querier_dto.SQLType

	Float4 querier_dto.SQLType

	Float8 querier_dto.SQLType

	Numeric querier_dto.SQLType

	Boolean querier_dto.SQLType

	Text querier_dto.SQLType

	Bytea querier_dto.SQLType

	Timestamp querier_dto.SQLType

	Timestamptz querier_dto.SQLType

	Date querier_dto.SQLType

	Time querier_dto.SQLType

	Interval querier_dto.SQLType

	JSON querier_dto.SQLType

	JSONB querier_dto.SQLType

	UUID querier_dto.SQLType

	Any querier_dto.SQLType
}

// Add registers a function signature under the given name.
func (b *FunctionCatalogueBuilder) Add(name string, signature *querier_dto.FunctionSignature) {
	signature.Name = name
	signature.DataAccess = querier_dto.DataAccessReadOnly
	b.Catalogue.Functions[name] = append(b.Catalogue.Functions[name], signature)
}

// Args builds a slice of FunctionArgument from alternating (name, type) pairs.
func (*FunctionCatalogueBuilder) Args(pairs ...any) []querier_dto.FunctionArgument {
	arguments := make([]querier_dto.FunctionArgument, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		name, nameOk := pairs[i].(string)
		sqlType, typeOk := pairs[i+1].(querier_dto.SQLType)
		if nameOk && typeOk {
			arguments = append(arguments, querier_dto.FunctionArgument{Name: name, Type: sqlType})
		}
	}
	return arguments
}

// NullOnNull registers a function that returns NULL when any argument is NULL.
func (b *FunctionCatalogueBuilder) NullOnNull(name string, arguments []querier_dto.FunctionArgument, returnType querier_dto.SQLType) {
	b.Add(name, &querier_dto.FunctionSignature{
		Arguments:         arguments,
		ReturnType:        returnType,
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
	})
}

// NeverNull registers a function that never returns NULL.
func (b *FunctionCatalogueBuilder) NeverNull(name string, arguments []querier_dto.FunctionArgument, returnType querier_dto.SQLType) {
	b.Add(name, &querier_dto.FunctionSignature{
		Arguments:         arguments,
		ReturnType:        returnType,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})
}

// CalledOnNull registers a function that is called even when arguments are
// NULL. The result may or may not be NULL depending on the function.
func (b *FunctionCatalogueBuilder) CalledOnNull(name string, arguments []querier_dto.FunctionArgument, returnType querier_dto.SQLType) {
	b.Add(name, &querier_dto.FunctionSignature{
		Arguments:         arguments,
		ReturnType:        returnType,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
}

// registerMathFunctions registers mathematical functions.
func (b *FunctionCatalogueBuilder) registerMathFunctions() {
	b.NullOnNull("abs", b.Args(paramNameX, b.Numeric), b.Numeric)
	b.NullOnNull("ceil", b.Args(paramNameX, b.Numeric), b.Numeric)
	b.NullOnNull("ceiling", b.Args(paramNameX, b.Numeric), b.Numeric)
	b.NullOnNull("floor", b.Args(paramNameX, b.Numeric), b.Numeric)
	b.NullOnNull("round", b.Args(paramNameX, b.Numeric), b.Numeric)
	b.NullOnNull("round", b.Args(paramNameX, b.Numeric, "s", b.Integer), b.Numeric)
	b.NullOnNull("trunc", b.Args(paramNameX, b.Numeric), b.Numeric)
	b.NullOnNull("trunc", b.Args(paramNameX, b.Numeric, "s", b.Integer), b.Numeric)
	b.NullOnNull("sign", b.Args(paramNameX, b.Numeric), b.Numeric)
	b.NullOnNull("sqrt", b.Args(paramNameX, b.Float8), b.Float8)
	b.NullOnNull("cbrt", b.Args(paramNameX, b.Float8), b.Float8)
	b.NullOnNull("power", b.Args("a", b.Numeric, "b", b.Numeric), b.Numeric)
	b.NullOnNull("pow", b.Args(paramNameA, b.Numeric, paramNameB, b.Numeric), b.Numeric)
	b.NullOnNull("exp", b.Args(paramNameX, b.Numeric), b.Numeric)
	b.NullOnNull("ln", b.Args(paramNameX, b.Numeric), b.Numeric)
	b.NullOnNull("log", b.Args(paramNameX, b.Numeric), b.Numeric)
	b.NullOnNull("log", b.Args("base", b.Numeric, paramNameX, b.Numeric), b.Numeric)
	b.NullOnNull("log10", b.Args(paramNameX, b.Numeric), b.Numeric)
	b.NullOnNull("mod", b.Args(paramNameY, b.Numeric, paramNameX, b.Numeric), b.Numeric)
	b.NullOnNull("div", b.Args(paramNameY, b.Numeric, paramNameX, b.Numeric), b.Numeric)
	b.NullOnNull("degrees", b.Args(paramNameX, b.Float8), b.Float8)
	b.NullOnNull("radians", b.Args(paramNameX, b.Float8), b.Float8)
	b.NeverNull("pi", nil, b.Float8)
	b.NeverNull("random", nil, b.Float8)
	b.NullOnNull("setseed", b.Args("seed", b.Float8), b.Float8)
	b.NullOnNull("factorial", b.Args(paramNameX, b.Bigint), b.Numeric)
	b.NullOnNull("gcd", b.Args(paramNameA, b.Bigint, paramNameB, b.Bigint), b.Bigint)
	b.NullOnNull("lcm", b.Args(paramNameA, b.Bigint, paramNameB, b.Bigint), b.Bigint)
	b.NullOnNull("width_bucket", b.Args("operand", b.Numeric, "low", b.Numeric, "high", b.Numeric, paramNameCount, b.Integer), b.Integer)
}

// registerTrigonometricFunctions registers trigonometric functions (radian and
// degree variants).
func (b *FunctionCatalogueBuilder) registerTrigonometricFunctions() {
	b.NullOnNull("acos", b.Args(paramNameX, b.Float8), b.Float8)
	b.NullOnNull("asin", b.Args(paramNameX, b.Float8), b.Float8)
	b.NullOnNull("atan", b.Args(paramNameX, b.Float8), b.Float8)
	b.NullOnNull("atan2", b.Args(paramNameY, b.Float8, paramNameX, b.Float8), b.Float8)
	b.NullOnNull("cos", b.Args(paramNameX, b.Float8), b.Float8)
	b.NullOnNull("sin", b.Args(paramNameX, b.Float8), b.Float8)
	b.NullOnNull("tan", b.Args(paramNameX, b.Float8), b.Float8)
	b.NullOnNull("cot", b.Args(paramNameX, b.Float8), b.Float8)

	b.NullOnNull("acosd", b.Args(paramNameX, b.Float8), b.Float8)
	b.NullOnNull("asind", b.Args(paramNameX, b.Float8), b.Float8)
	b.NullOnNull("atand", b.Args(paramNameX, b.Float8), b.Float8)
	b.NullOnNull("atan2d", b.Args(paramNameY, b.Float8, paramNameX, b.Float8), b.Float8)
	b.NullOnNull("cosd", b.Args(paramNameX, b.Float8), b.Float8)
	b.NullOnNull("sind", b.Args(paramNameX, b.Float8), b.Float8)
	b.NullOnNull("tand", b.Args(paramNameX, b.Float8), b.Float8)
	b.NullOnNull("cotd", b.Args(paramNameX, b.Float8), b.Float8)
}

// registerStringFunctions registers string manipulation functions.
func (b *FunctionCatalogueBuilder) registerStringFunctions() {
	b.registerStringBasicFunctions()
	b.registerStringVariadicFunctions()
	b.registerStringMiscFunctions()
}

func (b *FunctionCatalogueBuilder) registerStringBasicFunctions() {
	b.NullOnNull("length", b.Args(paramNameX, b.Text), b.Integer)
	b.NullOnNull("char_length", b.Args(paramNameX, b.Text), b.Integer)
	b.NullOnNull("octet_length", b.Args(paramNameX, b.Text), b.Integer)
	b.NullOnNull("bit_length", b.Args(paramNameX, b.Text), b.Integer)
	b.NullOnNull("lower", b.Args(paramNameX, b.Text), b.Text)
	b.NullOnNull("upper", b.Args(paramNameX, b.Text), b.Text)
	b.NullOnNull("trim", b.Args(paramNameX, b.Text), b.Text)
	b.NullOnNull("ltrim", b.Args(paramNameX, b.Text), b.Text)
	b.NullOnNull("ltrim", b.Args(paramNameX, b.Text, "characters", b.Text), b.Text)
	b.NullOnNull("rtrim", b.Args(paramNameX, b.Text), b.Text)
	b.NullOnNull("rtrim", b.Args(paramNameX, b.Text, "characters", b.Text), b.Text)
	b.NullOnNull("btrim", b.Args(paramNameX, b.Text), b.Text)
	b.NullOnNull("btrim", b.Args(paramNameX, b.Text, "characters", b.Text), b.Text)
	b.NullOnNull("lpad", b.Args(paramNameString, b.Text, paramNameLength, b.Integer), b.Text)
	b.NullOnNull("lpad", b.Args(paramNameString, b.Text, paramNameLength, b.Integer, "fill", b.Text), b.Text)
	b.NullOnNull("rpad", b.Args(paramNameString, b.Text, paramNameLength, b.Integer), b.Text)
	b.NullOnNull("rpad", b.Args(paramNameString, b.Text, paramNameLength, b.Integer, "fill", b.Text), b.Text)
	b.NullOnNull("repeat", b.Args(paramNameString, b.Text, "number", b.Integer), b.Text)
	b.NullOnNull("replace", b.Args(paramNameString, b.Text, "from", b.Text, "to", b.Text), b.Text)
	b.NullOnNull("reverse", b.Args(paramNameX, b.Text), b.Text)
	b.NullOnNull("substr", b.Args(paramNameString, b.Text, paramNameStart, b.Integer), b.Text)
	b.NullOnNull("substr", b.Args(paramNameString, b.Text, paramNameStart, b.Integer, paramNameCount, b.Integer), b.Text)
	b.NullOnNull("substring", b.Args(paramNameString, b.Text, paramNameStart, b.Integer), b.Text)
	b.NullOnNull("substring", b.Args(paramNameString, b.Text, paramNameStart, b.Integer, paramNameCount, b.Integer), b.Text)
	b.NullOnNull("left", b.Args(paramNameString, b.Text, paramNameN, b.Integer), b.Text)
	b.NullOnNull("right", b.Args(paramNameString, b.Text, paramNameN, b.Integer), b.Text)
}

func (b *FunctionCatalogueBuilder) registerStringVariadicFunctions() {
	b.Add("concat", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameValue, b.Any),
		ReturnType:        b.Text,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
		IsVariadic:        true,
		MinArguments:      1,
	})
	b.Add("concat_ws", &querier_dto.FunctionSignature{
		Arguments:         b.Args("separator", b.Text, paramNameValue, b.Any),
		ReturnType:        b.Text,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
		IsVariadic:        true,
		MinArguments:      2,
	})
	b.Add("format", &querier_dto.FunctionSignature{
		Arguments:         b.Args("formatstr", b.Text, paramNameValue, b.Any),
		ReturnType:        b.Text,
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		IsVariadic:        true,
		MinArguments:      1,
	})
}

func (b *FunctionCatalogueBuilder) registerStringMiscFunctions() {
	b.NullOnNull("initcap", b.Args(paramNameX, b.Text), b.Text)
	b.NullOnNull("translate", b.Args(paramNameString, b.Text, "from", b.Text, "to", b.Text), b.Text)
	b.NullOnNull("ascii", b.Args(paramNameX, b.Text), b.Integer)
	b.NullOnNull("chr", b.Args(paramNameX, b.Integer), b.Text)
	b.NullOnNull("encode", b.Args("data", b.Bytea, paramNameFormat, b.Text), b.Text)
	b.NullOnNull("decode", b.Args(paramNameString, b.Text, paramNameFormat, b.Text), b.Bytea)
	b.NullOnNull("md5", b.Args(paramNameX, b.Text), b.Text)
	b.NullOnNull("position", b.Args(paramNameSubstring, b.Text, paramNameString, b.Text), b.Integer)
	b.NullOnNull("strpos", b.Args(paramNameString, b.Text, paramNameSubstring, b.Text), b.Integer)
	b.NullOnNull("starts_with", b.Args(paramNameString, b.Text, "prefix", b.Text), b.Boolean)
	b.NullOnNull("split_part", b.Args(paramNameString, b.Text, paramNameDelimiter, b.Text, paramNameN, b.Integer), b.Text)
	b.NullOnNull("quote_ident", b.Args(paramNameX, b.Text), b.Text)
	b.NullOnNull("quote_literal", b.Args(paramNameX, b.Text), b.Text)
	b.CalledOnNull("quote_nullable", b.Args(paramNameX, b.Text), b.Text)
}

// registerDateTimeFunctions registers date and time functions.
func (b *FunctionCatalogueBuilder) registerDateTimeFunctions() {
	b.NeverNull("now", nil, b.Timestamptz)
	b.NeverNull("clock_timestamp", nil, b.Timestamptz)
	b.NeverNull("statement_timestamp", nil, b.Timestamptz)
	b.NeverNull("transaction_timestamp", nil, b.Timestamptz)
	b.NeverNull("timeofday", nil, b.Text)

	b.NullOnNull("age", b.Args(paramNameTimestamp, b.Timestamptz, paramNameTimestamp, b.Timestamptz), b.Interval)
	b.NullOnNull("age", b.Args(paramNameTimestamp, b.Timestamptz), b.Interval)

	b.NullOnNull("date_part", b.Args(paramNameField, b.Text, paramNameSource, b.Timestamptz), b.Float8)
	b.NullOnNull("date_part", b.Args(paramNameField, b.Text, paramNameSource, b.Interval), b.Float8)
	b.NullOnNull("date_trunc", b.Args(paramNameField, b.Text, paramNameSource, b.Timestamptz), b.Timestamptz)
	b.NullOnNull("date_trunc", b.Args(paramNameField, b.Text, paramNameSource, b.Interval), b.Interval)
	b.NullOnNull("date_trunc", b.Args(paramNameField, b.Text, paramNameSource, b.Timestamptz, "timezone", b.Text), b.Timestamptz)

	b.NullOnNull("extract", b.Args(paramNameField, b.Text, paramNameSource, b.Timestamptz), b.Numeric)
	b.NullOnNull("extract", b.Args(paramNameField, b.Text, paramNameSource, b.Interval), b.Numeric)

	b.NullOnNull("make_date", b.Args(paramNameYear, b.Integer, paramNameMonth, b.Integer, paramNameDay, b.Integer), b.Date)
	b.NullOnNull("make_time", b.Args(paramNameHour, b.Integer, paramNameMin, b.Integer, paramNameSec, b.Float8), b.Time)
	dateTimeArgs := b.Args(
		paramNameYear, b.Integer, paramNameMonth, b.Integer, paramNameDay, b.Integer,
		paramNameHour, b.Integer, paramNameMin, b.Integer, paramNameSec, b.Float8,
	)
	b.NullOnNull("make_timestamp", dateTimeArgs, b.Timestamp)
	b.NullOnNull("make_timestamptz", dateTimeArgs, b.Timestamptz)
	dateTimeWithTimezoneArgs := b.Args(
		paramNameYear, b.Integer, paramNameMonth, b.Integer, paramNameDay, b.Integer,
		paramNameHour, b.Integer, paramNameMin, b.Integer, paramNameSec, b.Float8,
		"timezone", b.Text,
	)
	b.NullOnNull("make_timestamptz", dateTimeWithTimezoneArgs, b.Timestamptz)
	b.NullOnNull("make_interval", b.Args("years", b.Integer, "months", b.Integer, "weeks", b.Integer, "days", b.Integer, "hours", b.Integer, "mins", b.Integer, "secs", b.Float8), b.Interval)

	b.NullOnNull("to_timestamp", b.Args("epoch", b.Float8), b.Timestamptz)
	b.NullOnNull("to_timestamp", b.Args(paramNameText, b.Text, paramNameFormat, b.Text), b.Timestamptz)
	b.NullOnNull("to_date", b.Args(paramNameText, b.Text, paramNameFormat, b.Text), b.Date)
	b.NullOnNull(funcNameToChar, b.Args(paramNameTimestamp, b.Timestamptz, paramNameFormat, b.Text), b.Text)
	b.NullOnNull(funcNameToChar, b.Args("interval", b.Interval, paramNameFormat, b.Text), b.Text)
	b.NullOnNull(funcNameToChar, b.Args(paramNameValue, b.Numeric, paramNameFormat, b.Text), b.Text)

	b.NullOnNull("isfinite", b.Args(paramNameValue, b.Date), b.Boolean)
	b.NullOnNull("isfinite", b.Args(paramNameValue, b.Timestamptz), b.Boolean)
	b.NullOnNull("isfinite", b.Args(paramNameValue, b.Interval), b.Boolean)

	b.NullOnNull("justify_days", b.Args(paramNameValue, b.Interval), b.Interval)
	b.NullOnNull("justify_hours", b.Args(paramNameValue, b.Interval), b.Interval)
	b.NullOnNull("justify_interval", b.Args(paramNameValue, b.Interval), b.Interval)
}

// registerJSONFunctions registers JSON and JSONB functions.
func (b *FunctionCatalogueBuilder) registerJSONFunctions() {
	b.registerJSONBuildFunctions()
	b.registerJSONScalarFunctions()
	b.registerJSONPathFunctions()
	b.registerJSONBMutationFunctions()
}

func (b *FunctionCatalogueBuilder) registerJSONBuildFunctions() {
	b.Add("json_build_object", &querier_dto.FunctionSignature{
		Arguments:         b.Args("key", b.Any, paramNameValue, b.Any),
		ReturnType:        b.JSON,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
		IsVariadic:        true,
		MinArguments:      0,
	})
	b.Add("json_build_array", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameValue, b.Any),
		ReturnType:        b.JSON,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
		IsVariadic:        true,
		MinArguments:      0,
	})
	b.Add("jsonb_build_object", &querier_dto.FunctionSignature{
		Arguments:         b.Args("key", b.Any, paramNameValue, b.Any),
		ReturnType:        b.JSONB,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
		IsVariadic:        true,
		MinArguments:      0,
	})
	b.Add("jsonb_build_array", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameValue, b.Any),
		ReturnType:        b.JSONB,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
		IsVariadic:        true,
		MinArguments:      0,
	})
}

func (b *FunctionCatalogueBuilder) registerJSONScalarFunctions() {
	b.NullOnNull("to_json", b.Args(paramNameValue, b.Any), b.JSON)
	b.NullOnNull("to_jsonb", b.Args(paramNameValue, b.Any), b.JSONB)
	b.NullOnNull("row_to_json", b.Args("record", b.Any), b.JSON)

	b.NullOnNull("json_array_length", b.Args(paramNameJSON, b.JSON), b.Integer)
	b.NullOnNull("jsonb_array_length", b.Args(paramNameJSON, b.JSONB), b.Integer)

	b.NullOnNull("json_typeof", b.Args(paramNameJSON, b.JSON), b.Text)
	b.NullOnNull("jsonb_typeof", b.Args(paramNameJSON, b.JSONB), b.Text)
}

func (b *FunctionCatalogueBuilder) registerJSONPathFunctions() {
	b.Add("json_extract_path", &querier_dto.FunctionSignature{
		Arguments:         b.Args("from_json", b.JSON, "path_elem", b.Text),
		ReturnType:        b.JSON,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      2,
	})
	b.Add("json_extract_path_text", &querier_dto.FunctionSignature{
		Arguments:         b.Args("from_json", b.JSON, "path_elem", b.Text),
		ReturnType:        b.Text,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      2,
	})
	b.Add("jsonb_extract_path", &querier_dto.FunctionSignature{
		Arguments:         b.Args("from_json", b.JSONB, "path_elem", b.Text),
		ReturnType:        b.JSONB,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      2,
	})
	b.Add("jsonb_extract_path_text", &querier_dto.FunctionSignature{
		Arguments:         b.Args("from_json", b.JSONB, "path_elem", b.Text),
		ReturnType:        b.Text,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      2,
	})

	b.NullOnNull("jsonb_path_exists", b.Args(paramNameTarget, b.JSONB, paramNamePath, b.Text), b.Boolean)
	b.NullOnNull("jsonb_path_match", b.Args(paramNameTarget, b.JSONB, paramNamePath, b.Text), b.Boolean)
}

func (b *FunctionCatalogueBuilder) registerJSONBMutationFunctions() {
	b.NullOnNull("jsonb_set", b.Args(paramNameTarget, b.JSONB, paramNamePath, b.Text, paramNameNewValue, b.JSONB), b.JSONB)
	b.NullOnNull("jsonb_set", b.Args(paramNameTarget, b.JSONB, paramNamePath, b.Text, paramNameNewValue, b.JSONB, "create_if_missing", b.Boolean), b.JSONB)
	b.NullOnNull("jsonb_insert", b.Args(paramNameTarget, b.JSONB, paramNamePath, b.Text, paramNameNewValue, b.JSONB), b.JSONB)
	b.NullOnNull("jsonb_insert", b.Args(paramNameTarget, b.JSONB, paramNamePath, b.Text, paramNameNewValue, b.JSONB, "insert_after", b.Boolean), b.JSONB)
	b.NullOnNull("jsonb_strip_nulls", b.Args(paramNameJSON, b.JSONB), b.JSONB)
	b.NullOnNull("jsonb_pretty", b.Args(paramNameJSON, b.JSONB), b.Text)
}

// registerArrayScalarFunctions registers array scalar functions.
func (b *FunctionCatalogueBuilder) registerArrayScalarFunctions() {
	b.NullOnNull("array_append", b.Args(paramNameArray, b.Any, paramNameElement, b.Any), b.Any)
	b.NullOnNull("array_cat", b.Args("array1", b.Any, "array2", b.Any), b.Any)
	b.NullOnNull("array_dims", b.Args(paramNameArray, b.Any), b.Text)
	b.NullOnNull("array_fill", b.Args(paramNameValue, b.Any, "dimensions", b.Any), b.Any)
	b.NullOnNull("array_length", b.Args(paramNameArray, b.Any, "dimension", b.Integer), b.Integer)
	b.NullOnNull("array_lower", b.Args(paramNameArray, b.Any, "dimension", b.Integer), b.Integer)
	b.NullOnNull("array_upper", b.Args(paramNameArray, b.Any, "dimension", b.Integer), b.Integer)
	b.NullOnNull("array_ndims", b.Args(paramNameArray, b.Any), b.Integer)
	b.NullOnNull("array_position", b.Args(paramNameArray, b.Any, paramNameElement, b.Any), b.Integer)
	b.NullOnNull("array_position", b.Args(paramNameArray, b.Any, paramNameElement, b.Any, paramNameStart, b.Integer), b.Integer)
	b.NullOnNull("array_positions", b.Args(paramNameArray, b.Any, paramNameElement, b.Any), b.Any)
	b.NullOnNull("array_prepend", b.Args(paramNameElement, b.Any, paramNameArray, b.Any), b.Any)
	b.NullOnNull("array_remove", b.Args(paramNameArray, b.Any, paramNameElement, b.Any), b.Any)
	b.NullOnNull("array_replace", b.Args(paramNameArray, b.Any, "from", b.Any, "to", b.Any), b.Any)
	b.NullOnNull("array_to_string", b.Args(paramNameArray, b.Any, paramNameDelimiter, b.Text), b.Text)
	b.NullOnNull("array_to_string", b.Args(paramNameArray, b.Any, paramNameDelimiter, b.Text, "null_string", b.Text), b.Text)
	b.NullOnNull("string_to_array", b.Args(paramNameString, b.Text, paramNameDelimiter, b.Text), b.Any)
	b.NullOnNull("string_to_array", b.Args(paramNameString, b.Text, paramNameDelimiter, b.Text, "null_string", b.Text), b.Any)
	b.NullOnNull("cardinality", b.Args(paramNameArray, b.Any), b.Integer)
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

func (b *FunctionCatalogueBuilder) registerCountAggregates() {
	b.Add("count", &querier_dto.FunctionSignature{
		ReturnType:        b.Bigint,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})
	b.Add("count", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameX, b.Any),
		ReturnType:        b.Bigint,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})
}

func (b *FunctionCatalogueBuilder) registerBooleanAndBitwiseAggregates() { //nolint:dupl // structurally similar aggregates
	b.Add("bool_and", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameX, b.Boolean),
		ReturnType:        b.Boolean,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("bool_or", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameX, b.Boolean),
		ReturnType:        b.Boolean,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("every", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameX, b.Boolean),
		ReturnType:        b.Boolean,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})

	b.Add("bit_and", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameX, b.Any),
		ReturnType:        b.Any,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("bit_or", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameX, b.Any),
		ReturnType:        b.Any,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("bit_xor", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameX, b.Any),
		ReturnType:        b.Any,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
}

func (b *FunctionCatalogueBuilder) registerCollectionAggregates() {
	b.Add("string_agg", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameExpression, b.Text, paramNameDelimiter, b.Text),
		ReturnType:        b.Text,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("array_agg", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameExpression, b.Any),
		ReturnType:        b.Any,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
}

func (b *FunctionCatalogueBuilder) registerJSONAggregates() {
	b.Add("json_agg", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameExpression, b.Any),
		ReturnType:        b.JSON,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("jsonb_agg", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameExpression, b.Any),
		ReturnType:        b.JSONB,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("json_object_agg", &querier_dto.FunctionSignature{
		Arguments:         b.Args("key", b.Any, paramNameValue, b.Any),
		ReturnType:        b.JSON,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("jsonb_object_agg", &querier_dto.FunctionSignature{
		Arguments:         b.Args("key", b.Any, paramNameValue, b.Any),
		ReturnType:        b.JSONB,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
}

func (b *FunctionCatalogueBuilder) registerOrderedSetAggregates() {
	b.Add("percentile_cont", &querier_dto.FunctionSignature{
		Arguments:         b.Args("fraction", b.Float8),
		ReturnType:        b.Float8,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("percentile_disc", &querier_dto.FunctionSignature{
		Arguments:         b.Args("fraction", b.Float8),
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

func (b *FunctionCatalogueBuilder) registerStatisticalAggregates() { //nolint:dupl // structurally similar aggregates
	b.Add("stddev", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameX, b.Any),
		ReturnType:        b.Numeric,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("stddev_pop", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameX, b.Any),
		ReturnType:        b.Numeric,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("stddev_samp", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameX, b.Any),
		ReturnType:        b.Numeric,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("variance", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameX, b.Any),
		ReturnType:        b.Numeric,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("var_pop", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameX, b.Any),
		ReturnType:        b.Numeric,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("var_samp", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameX, b.Any),
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
	b.NeverNull("ntile", b.Args(paramNameN, b.Integer), b.Integer)
	b.NeverNull("cume_dist", nil, b.Float8)
	b.NeverNull("percent_rank", nil, b.Float8)

	windowValueArgs := b.Args(paramNameExpression, b.Any, "offset", b.Integer, "default", b.Any)

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

	b.CalledOnNull("first_value", b.Args(paramNameExpression, b.Any), b.Any)
	b.CalledOnNull("last_value", b.Args(paramNameExpression, b.Any), b.Any)
	b.CalledOnNull("nth_value", b.Args(paramNameExpression, b.Any, paramNameN, b.Integer), b.Any)
}

// registerConditionalFunctions registers conditional expression functions.
func (b *FunctionCatalogueBuilder) registerConditionalFunctions() {
	b.CalledOnNull("nullif", b.Args("value1", b.Any, "value2", b.Any), b.Any)
	b.Add("greatest", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameValue, b.Any),
		ReturnType:        b.Any,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      1,
	})
	b.Add("least", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameValue, b.Any),
		ReturnType:        b.Any,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      1,
	})
}

// registerTypeConversionFunctions registers type conversion functions.
func (b *FunctionCatalogueBuilder) registerTypeConversionFunctions() {
	b.NullOnNull(funcNameToChar, b.Args(paramNameValue, b.Integer, paramNameFormat, b.Text), b.Text)
	b.NullOnNull("to_number", b.Args(paramNameText, b.Text, paramNameFormat, b.Text), b.Numeric)
}

// registerSystemFunctions registers system information functions.
func (b *FunctionCatalogueBuilder) registerSystemFunctions() {
	b.NeverNull("current_database", nil, b.Text)
	b.NeverNull("current_schema", nil, b.Text)
	b.NeverNull("current_user", nil, b.Text)
	b.NeverNull("session_user", nil, b.Text)
	b.NeverNull("version", nil, b.Text)

	b.NullOnNull("pg_typeof", b.Args(paramNameValue, b.Any), b.Text)
	b.NullOnNull("pg_column_size", b.Args(paramNameValue, b.Any), b.Integer)
	b.NullOnNull("pg_table_size", b.Args("table", b.Text), b.Bigint)
	b.NullOnNull("pg_total_relation_size", b.Args("table", b.Text), b.Bigint)
	b.CalledOnNull("obj_description", b.Args("oid", b.Integer, "catalog", b.Text), b.Text)
	b.CalledOnNull("obj_description", b.Args("oid", b.Integer), b.Text)
}
