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

package db_engine_mysql

import (
	"piko.sh/piko/internal/querier/querier_dto"
)

func buildFunctionCatalogue(extraFunctions func(*FunctionCatalogueBuilder)) *querier_dto.FunctionCatalogue {
	builder := &FunctionCatalogueBuilder{
		Catalogue: &querier_dto.FunctionCatalogue{
			Functions: make(map[string][]*querier_dto.FunctionSignature),
		},
		integer:   querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int"},
		bigint:    querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "bigint"},
		float:     querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "float"},
		double:    querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "double"},
		decimal:   querier_dto.SQLType{Category: querier_dto.TypeCategoryDecimal, EngineName: "decimal"},
		text:      querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
		varchar:   querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "varchar"},
		boolean:   querier_dto.SQLType{Category: querier_dto.TypeCategoryBoolean, EngineName: "tinyint"},
		bytea:     querier_dto.SQLType{Category: querier_dto.TypeCategoryBytea, EngineName: "blob"},
		date:      querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "date"},
		time:      querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "time"},
		datetime:  querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "datetime"},
		timestamp: querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "timestamp"},
		json:      querier_dto.SQLType{Category: querier_dto.TypeCategoryJSON, EngineName: "json"},
		geometry:  querier_dto.SQLType{Category: querier_dto.TypeCategoryGeometric, EngineName: "geometry"},
		any:       querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: ""},
	}

	builder.registerMathFunctions()
	builder.registerStringFunctions()
	builder.registerDateTimeFunctions()
	builder.registerJSONFunctions()
	builder.registerAggregateFunctions()
	builder.registerWindowFunctions()
	builder.registerConditionalFunctions()
	builder.registerSystemFunctions()
	builder.registerTypeConversionFunctions()

	if extraFunctions != nil {
		extraFunctions(builder)
	}

	return builder.Catalogue
}

// FunctionCatalogueBuilder builds a MySQL function catalogue. It is exported
// so that flavour option functions (such as MariaDB) can register additional
// functions via WithExtraFunctions.
type FunctionCatalogueBuilder struct {
	Catalogue *querier_dto.FunctionCatalogue

	integer querier_dto.SQLType

	bigint querier_dto.SQLType

	float querier_dto.SQLType

	double querier_dto.SQLType

	decimal querier_dto.SQLType

	text querier_dto.SQLType

	varchar querier_dto.SQLType

	boolean querier_dto.SQLType

	bytea querier_dto.SQLType

	date querier_dto.SQLType

	time querier_dto.SQLType

	datetime querier_dto.SQLType

	timestamp querier_dto.SQLType

	json querier_dto.SQLType

	geometry querier_dto.SQLType

	any querier_dto.SQLType
}

// Integer returns the MySQL INT type.
func (b *FunctionCatalogueBuilder) Integer() querier_dto.SQLType { return b.integer }

// Bigint returns the MySQL BIGINT type.
func (b *FunctionCatalogueBuilder) Bigint() querier_dto.SQLType { return b.bigint }

// Float returns the MySQL FLOAT type.
func (b *FunctionCatalogueBuilder) Float() querier_dto.SQLType { return b.float }

// Double returns the MySQL DOUBLE type.
func (b *FunctionCatalogueBuilder) Double() querier_dto.SQLType { return b.double }

// Decimal returns the MySQL DECIMAL type.
func (b *FunctionCatalogueBuilder) Decimal() querier_dto.SQLType { return b.decimal }

// Text returns the MySQL TEXT type.
func (b *FunctionCatalogueBuilder) Text() querier_dto.SQLType { return b.text }

// Varchar returns the MySQL VARCHAR type.
func (b *FunctionCatalogueBuilder) Varchar() querier_dto.SQLType { return b.varchar }

// Boolean returns the MySQL BOOLEAN (TINYINT) type.
func (b *FunctionCatalogueBuilder) Boolean() querier_dto.SQLType { return b.boolean }

// Bytea returns the MySQL BLOB type.
func (b *FunctionCatalogueBuilder) Bytea() querier_dto.SQLType { return b.bytea }

// Date returns the MySQL DATE type.
func (b *FunctionCatalogueBuilder) Date() querier_dto.SQLType { return b.date }

// Time returns the MySQL TIME type.
func (b *FunctionCatalogueBuilder) Time() querier_dto.SQLType { return b.time }

// Datetime returns the MySQL DATETIME type.
func (b *FunctionCatalogueBuilder) Datetime() querier_dto.SQLType { return b.datetime }

// Timestamp returns the MySQL TIMESTAMP type.
func (b *FunctionCatalogueBuilder) Timestamp() querier_dto.SQLType { return b.timestamp }

// JSON returns the MySQL JSON type.
func (b *FunctionCatalogueBuilder) JSON() querier_dto.SQLType { return b.json }

// Geometry returns the MySQL GEOMETRY type.
func (b *FunctionCatalogueBuilder) Geometry() querier_dto.SQLType { return b.geometry }

// Add registers a function signature under the given name. Built-in functions
// are always marked as read-only since none of them modify data.
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

// Aggregate registers an aggregate function.
func (b *FunctionCatalogueBuilder) Aggregate(name string, arguments []querier_dto.FunctionArgument, returnType querier_dto.SQLType) {
	b.Add(name, &querier_dto.FunctionSignature{
		Arguments:         arguments,
		ReturnType:        returnType,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
}

// Window registers a window function.
func (b *FunctionCatalogueBuilder) Window(name string, arguments []querier_dto.FunctionArgument, returnType querier_dto.SQLType) {
	b.Add(name, &querier_dto.FunctionSignature{
		Arguments:         arguments,
		ReturnType:        returnType,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})
}

// Variadic registers a variadic function.
func (b *FunctionCatalogueBuilder) Variadic(name string, arguments []querier_dto.FunctionArgument, minArguments int, returnType querier_dto.SQLType) {
	b.Add(name, &querier_dto.FunctionSignature{
		Arguments:         arguments,
		ReturnType:        returnType,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      minArguments,
	})
}

// registerMathFunctions registers mathematical functions.
func (b *FunctionCatalogueBuilder) registerMathFunctions() {
	b.NullOnNull("abs", b.Args(paramNameX, b.integer), b.integer)
	b.NullOnNull("ceil", b.Args(paramNameX, b.double), b.integer)
	b.NullOnNull("ceiling", b.Args(paramNameX, b.double), b.integer)
	b.NullOnNull("floor", b.Args(paramNameX, b.double), b.integer)
	b.NullOnNull("round", b.Args(paramNameX, b.double), b.double)
	b.NullOnNull("round", b.Args(paramNameX, b.double, "d", b.integer), b.double)
	b.NullOnNull("truncate", b.Args(paramNameX, b.double, "d", b.integer), b.double)
	b.NullOnNull("sign", b.Args(paramNameX, b.double), b.integer)
	b.NullOnNull("sqrt", b.Args(paramNameX, b.double), b.double)
	b.NullOnNull("power", b.Args(paramNameX, b.double, paramNameY, b.double), b.double)
	b.NullOnNull("pow", b.Args(paramNameX, b.double, paramNameY, b.double), b.double)
	b.NullOnNull("exp", b.Args(paramNameX, b.double), b.double)
	b.NullOnNull("ln", b.Args(paramNameX, b.double), b.double)
	b.NullOnNull("log", b.Args(paramNameX, b.double), b.double)
	b.NullOnNull("log", b.Args("base", b.double, paramNameX, b.double), b.double)
	b.NullOnNull("log2", b.Args(paramNameX, b.double), b.double)
	b.NullOnNull("log10", b.Args(paramNameX, b.double), b.double)
	b.NullOnNull("mod", b.Args(paramNameX, b.double, paramNameY, b.double), b.integer)
	b.NeverNull("pi", nil, b.double)
	b.NeverNull("rand", nil, b.double)
	b.NullOnNull("rand", b.Args("seed", b.integer), b.double)
	b.NullOnNull("degrees", b.Args(paramNameX, b.double), b.double)
	b.NullOnNull("radians", b.Args(paramNameX, b.double), b.double)
	b.NullOnNull("crc32", b.Args(paramNameExpression, b.text), b.integer)
}

// registerStringFunctions registers string manipulation functions.
func (b *FunctionCatalogueBuilder) registerStringFunctions() {
	b.registerStringVariadicFunctions()
	b.registerStringLengthFunctions()
	b.registerStringCaseFunctions()
	b.registerStringTrimFunctions()
	b.registerStringPaddingFunctions()
	b.registerStringTransformFunctions()
	b.registerStringSearchFunctions()
	b.registerStringMiscFunctions()
}

func (b *FunctionCatalogueBuilder) registerStringVariadicFunctions() {
	b.Add("concat", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameValue, b.any),
		ReturnType:        b.text,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      1,
	})
	b.Add("concat_ws", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameSeparator, b.text, paramNameValue, b.any),
		ReturnType:        b.text,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      2,
	})
}

func (b *FunctionCatalogueBuilder) registerStringLengthFunctions() {
	b.NullOnNull("length", b.Args(paramNameString, b.text), b.integer)
	b.NullOnNull("char_length", b.Args(paramNameString, b.text), b.integer)
	b.NullOnNull("character_length", b.Args(paramNameString, b.text), b.integer)
	b.NullOnNull("octet_length", b.Args(paramNameString, b.text), b.integer)
}

func (b *FunctionCatalogueBuilder) registerStringCaseFunctions() {
	b.NullOnNull("lower", b.Args(paramNameString, b.text), b.text)
	b.NullOnNull("lcase", b.Args(paramNameString, b.text), b.text)
	b.NullOnNull("upper", b.Args(paramNameString, b.text), b.text)
	b.NullOnNull("ucase", b.Args(paramNameString, b.text), b.text)
}

func (b *FunctionCatalogueBuilder) registerStringTrimFunctions() {
	b.NullOnNull("trim", b.Args(paramNameString, b.text), b.text)
	b.NullOnNull("ltrim", b.Args(paramNameString, b.text), b.text)
	b.NullOnNull("rtrim", b.Args(paramNameString, b.text), b.text)
}

func (b *FunctionCatalogueBuilder) registerStringPaddingFunctions() {
	b.NullOnNull("lpad", b.Args(paramNameString, b.text, paramNameLength, b.integer, "pad", b.text), b.text)
	b.NullOnNull("lpad", b.Args(paramNameString, b.text, paramNameLength, b.integer), b.text)
	b.NullOnNull("rpad", b.Args(paramNameString, b.text, paramNameLength, b.integer, "pad", b.text), b.text)
	b.NullOnNull("rpad", b.Args(paramNameString, b.text, paramNameLength, b.integer), b.text)
}

func (b *FunctionCatalogueBuilder) registerStringTransformFunctions() {
	b.NullOnNull("repeat", b.Args(paramNameString, b.text, paramNameCount, b.integer), b.text)
	b.NullOnNull("replace", b.Args(paramNameString, b.text, "from_str", b.text, "to_str", b.text), b.text)
	b.NullOnNull("reverse", b.Args(paramNameString, b.text), b.text)
	b.NullOnNull("substring", b.Args(paramNameString, b.text, paramNameStart, b.integer), b.text)
	b.NullOnNull("substring", b.Args(paramNameString, b.text, paramNameStart, b.integer, paramNameLength, b.integer), b.text)
	b.NullOnNull("substr", b.Args(paramNameString, b.text, paramNameStart, b.integer), b.text)
	b.NullOnNull("substr", b.Args(paramNameString, b.text, paramNameStart, b.integer, paramNameLength, b.integer), b.text)
	b.NullOnNull("left", b.Args(paramNameString, b.text, paramNameLength, b.integer), b.text)
	b.NullOnNull("right", b.Args(paramNameString, b.text, paramNameLength, b.integer), b.text)
}

func (b *FunctionCatalogueBuilder) registerStringSearchFunctions() {
	b.NullOnNull("locate", b.Args(paramNameSubstring, b.text, paramNameString, b.text), b.integer)
	b.NullOnNull("locate", b.Args(paramNameSubstring, b.text, paramNameString, b.text, paramNameStart, b.integer), b.integer)
	b.NullOnNull("instr", b.Args(paramNameString, b.text, paramNameSubstring, b.text), b.integer)
}

func (b *FunctionCatalogueBuilder) registerStringMiscFunctions() {
	b.NullOnNull("ascii", b.Args(paramNameString, b.text), b.integer)
	b.NullOnNull("hex", b.Args(paramNameX, b.any), b.text)
	b.NullOnNull("unhex", b.Args(paramNameString, b.text), b.bytea)
	b.NullOnNull("space", b.Args(paramNameN, b.integer), b.text)
	b.NullOnNull("format", b.Args(paramNameX, b.double, paramNameD, b.integer), b.text)
	b.NullOnNull("format", b.Args(paramNameX, b.double, paramNameD, b.integer, paramNameLocale, b.text), b.text)
}

// registerDateTimeFunctions registers date and time functions.
func (b *FunctionCatalogueBuilder) registerDateTimeFunctions() {
	b.registerDateTimeCurrentFunctions()
	b.registerDateTimeExtractionFunctions()
	b.registerDateTimeFormatFunctions()
	b.registerDateTimeArithmeticFunctions()
}

func (b *FunctionCatalogueBuilder) registerDateTimeCurrentFunctions() {
	b.NeverNull("now", nil, b.datetime)
	b.NeverNull("curdate", nil, b.date)
	b.NeverNull("current_date", nil, b.date)
	b.NeverNull("curtime", nil, b.time)
	b.NeverNull("current_time", nil, b.time)
}

func (b *FunctionCatalogueBuilder) registerDateTimeExtractionFunctions() {
	b.NullOnNull("date", b.Args(paramNameExpression, b.datetime), b.date)
	b.NullOnNull("time", b.Args(paramNameExpression, b.datetime), b.time)
	b.NullOnNull("year", b.Args(paramNameDate, b.date), b.integer)
	b.NullOnNull("month", b.Args(paramNameDate, b.date), b.integer)
	b.NullOnNull("day", b.Args(paramNameDate, b.date), b.integer)
	b.NullOnNull("dayofmonth", b.Args(paramNameDate, b.date), b.integer)
	b.NullOnNull("hour", b.Args(paramNameTime, b.time), b.integer)
	b.NullOnNull("minute", b.Args(paramNameTime, b.time), b.integer)
	b.NullOnNull("second", b.Args(paramNameTime, b.time), b.integer)
	b.NullOnNull("dayofweek", b.Args(paramNameDate, b.date), b.integer)
	b.NullOnNull("dayofyear", b.Args(paramNameDate, b.date), b.integer)
	b.NullOnNull("week", b.Args(paramNameDate, b.date), b.integer)
	b.NullOnNull("week", b.Args(paramNameDate, b.date, "mode", b.integer), b.integer)
	b.NullOnNull("quarter", b.Args(paramNameDate, b.date), b.integer)
	b.NullOnNull("last_day", b.Args(paramNameDate, b.date), b.date)
}

func (b *FunctionCatalogueBuilder) registerDateTimeFormatFunctions() {
	b.NullOnNull("date_format", b.Args(paramNameDate, b.datetime, paramNameFormat, b.text), b.text)
	b.NullOnNull("str_to_date", b.Args(paramNameString, b.text, paramNameFormat, b.text), b.datetime)
	b.NullOnNull("from_unixtime", b.Args(paramNameTimestamp, b.bigint), b.datetime)
	b.NullOnNull("from_unixtime", b.Args(paramNameTimestamp, b.bigint, paramNameFormat, b.text), b.text)
	b.CalledOnNull("unix_timestamp", nil, b.bigint)
	b.NullOnNull("unix_timestamp", b.Args(paramNameDate, b.datetime), b.bigint)
}

func (b *FunctionCatalogueBuilder) registerDateTimeArithmeticFunctions() {
	b.NullOnNull("datediff", b.Args(paramNameExpr1, b.date, paramNameExpr2, b.date), b.integer)
	b.NullOnNull("timediff", b.Args(paramNameExpr1, b.time, paramNameExpr2, b.time), b.time)
	b.NullOnNull("timestampdiff", b.Args(paramNameUnit, b.text, paramNameExpr1, b.datetime, paramNameExpr2, b.datetime), b.bigint)
	b.NullOnNull("timestampadd", b.Args(paramNameUnit, b.text, paramNameInterval, b.integer, paramNameDatetime, b.datetime), b.datetime)
	b.NullOnNull("date_add", b.Args(paramNameDate, b.datetime, "interval", b.any), b.datetime)
	b.NullOnNull("date_sub", b.Args(paramNameDate, b.datetime, "interval", b.any), b.datetime)
}

// registerJSONFunctions registers JSON functions.
func (b *FunctionCatalogueBuilder) registerJSONFunctions() {
	b.registerJSONAccessFunctions()
	b.registerJSONMutationFunctions()
	b.registerJSONBuildFunctions()
	b.registerJSONIntrospectionFunctions()
}

func (b *FunctionCatalogueBuilder) registerJSONAccessFunctions() {
	b.NullOnNull("json_extract", b.Args(paramNameJSON, b.json, paramNamePath, b.text), b.json)
	b.NullOnNull("json_unquote", b.Args(paramNameJSON, b.json), b.text)
}

func (b *FunctionCatalogueBuilder) registerJSONMutationFunctions() {
	b.NullOnNull("json_set", b.Args(paramNameJSON, b.json, paramNamePath, b.text, paramNameValue, b.any), b.json)
	b.NullOnNull("json_insert", b.Args(paramNameJSON, b.json, paramNamePath, b.text, paramNameValue, b.any), b.json)
	b.NullOnNull("json_replace", b.Args(paramNameJSON, b.json, paramNamePath, b.text, paramNameValue, b.any), b.json)
	b.NullOnNull("json_remove", b.Args(paramNameJSON, b.json, paramNamePath, b.text), b.json)
}

func (b *FunctionCatalogueBuilder) registerJSONBuildFunctions() {
	b.Add("json_object", &querier_dto.FunctionSignature{
		Arguments:         b.Args("key", b.text, paramNameValue, b.any),
		ReturnType:        b.json,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
		IsVariadic:        true,
		MinArguments:      0,
	})
	b.Add("json_array", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameValue, b.any),
		ReturnType:        b.json,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
		IsVariadic:        true,
		MinArguments:      0,
	})
}

func (b *FunctionCatalogueBuilder) registerJSONIntrospectionFunctions() {
	b.NullOnNull("json_contains", b.Args(paramNameTarget, b.json, "candidate", b.json), b.boolean)
	b.NullOnNull("json_contains", b.Args(paramNameTarget, b.json, "candidate", b.json, paramNamePath, b.text), b.boolean)
	b.NullOnNull("json_contains_path", b.Args(paramNameJSON, b.json, "one_or_all", b.text, paramNamePath, b.text), b.boolean)
	b.NullOnNull("json_type", b.Args(paramNameJSON, b.json), b.text)
	b.NullOnNull("json_valid", b.Args(paramNameValue, b.any), b.boolean)
	b.NullOnNull("json_length", b.Args(paramNameJSON, b.json), b.integer)
	b.NullOnNull("json_length", b.Args(paramNameJSON, b.json, paramNamePath, b.text), b.integer)
	b.NullOnNull("json_keys", b.Args(paramNameJSON, b.json), b.json)
	b.NullOnNull("json_keys", b.Args(paramNameJSON, b.json, paramNamePath, b.text), b.json)
}

// registerAggregateFunctions registers aggregate functions.
func (b *FunctionCatalogueBuilder) registerAggregateFunctions() {
	b.Add("count", &querier_dto.FunctionSignature{
		ReturnType:        b.bigint,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})
	b.Add("count", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameX, b.any),
		ReturnType:        b.bigint,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})

	b.Aggregate("sum", b.Args(paramNameX, b.any), b.decimal)
	b.Aggregate("avg", b.Args(paramNameX, b.any), b.double)
	b.Aggregate("min", b.Args(paramNameX, b.any), b.any)
	b.Aggregate("max", b.Args(paramNameX, b.any), b.any)

	b.Add("group_concat", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameExpression, b.any),
		ReturnType:        b.text,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})

	b.Aggregate("bit_and", b.Args(paramNameX, b.any), b.any)
	b.Aggregate("bit_or", b.Args(paramNameX, b.any), b.any)
	b.Aggregate("bit_xor", b.Args(paramNameX, b.any), b.any)

	b.Aggregate("std", b.Args(paramNameX, b.any), b.double)
	b.Aggregate("stddev", b.Args(paramNameX, b.any), b.double)
}

// registerWindowFunctions registers window functions.
func (b *FunctionCatalogueBuilder) registerWindowFunctions() {
	b.Window("row_number", nil, b.bigint)
	b.Window("rank", nil, b.bigint)
	b.Window("dense_rank", nil, b.bigint)
	b.Window("ntile", b.Args(paramNameN, b.integer), b.integer)

	windowValueArgs := b.Args(paramNameExpression, b.any, "offset", b.integer, "default", b.any)

	b.Add("lag", &querier_dto.FunctionSignature{
		Arguments:         windowValueArgs,
		ReturnType:        b.any,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		MinArguments:      1,
	})
	b.Add("lead", &querier_dto.FunctionSignature{
		Arguments:         windowValueArgs,
		ReturnType:        b.any,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		MinArguments:      1,
	})

	b.CalledOnNull("first_value", b.Args(paramNameExpression, b.any), b.any)
	b.CalledOnNull("last_value", b.Args(paramNameExpression, b.any), b.any)
	b.CalledOnNull("nth_value", b.Args(paramNameExpression, b.any, paramNameN, b.integer), b.any)
}

// registerConditionalFunctions registers conditional expression functions.
func (b *FunctionCatalogueBuilder) registerConditionalFunctions() {
	b.CalledOnNull("if", b.Args(paramNameCondition, b.boolean, paramNameThen, b.any, paramNameElse, b.any), b.text)
	b.CalledOnNull("ifnull", b.Args(paramNameExpr1, b.any, paramNameExpr2, b.any), b.any)
	b.CalledOnNull("nullif", b.Args(paramNameExpr1, b.any, paramNameExpr2, b.any), b.any)

	b.Add("coalesce", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameValue, b.any),
		ReturnType:        b.any,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      1,
	})

	b.Add("greatest", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameValue, b.any),
		ReturnType:        b.any,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      1,
	})
	b.Add("least", &querier_dto.FunctionSignature{
		Arguments:         b.Args(paramNameValue, b.any),
		ReturnType:        b.any,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      1,
	})
}

// registerSystemFunctions registers system information functions.
func (b *FunctionCatalogueBuilder) registerSystemFunctions() {
	b.NeverNull("version", nil, b.text)
	b.NeverNull("database", nil, b.text)
	b.NeverNull("user", nil, b.text)
	b.NeverNull("current_user", nil, b.text)
	b.CalledOnNull("last_insert_id", nil, b.bigint)
	b.NeverNull("uuid", nil, b.text)
	b.NeverNull("connection_id", nil, b.bigint)
}

// registerTypeConversionFunctions registers type conversion functions.
func (b *FunctionCatalogueBuilder) registerTypeConversionFunctions() {
	b.NullOnNull("inet_aton", b.Args(paramNameExpression, b.text), b.bigint)
	b.NullOnNull("inet_ntoa", b.Args(paramNameExpression, b.bigint), b.text)
}
