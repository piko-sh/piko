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
	"piko.sh/piko/internal/querier/querier_adapters/engine_shared"
	"piko.sh/piko/internal/querier/querier_dto"
)

// buildFunctionCatalogue assembles the MySQL function catalogue and applies any
// flavour-specific extensions.
//
// Takes extraFunctions (func(*FunctionCatalogueBuilder)) which is an optional callback
// that registers additional functions.
//
// Returns *querier_dto.FunctionCatalogue which is the populated catalogue.
func buildFunctionCatalogue(extraFunctions func(*FunctionCatalogueBuilder)) *querier_dto.FunctionCatalogue {
	builder := &FunctionCatalogueBuilder{
		CatalogueBuilder: engine_shared.NewCatalogueBuilder(),
		integer:          querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int"},
		bigint:           querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "bigint"},
		float:            querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "float"},
		double:           querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "double"},
		decimal:          querier_dto.SQLType{Category: querier_dto.TypeCategoryDecimal, EngineName: "decimal"},
		text:             querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
		varchar:          querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "varchar"},
		boolean:          querier_dto.SQLType{Category: querier_dto.TypeCategoryBoolean, EngineName: "tinyint"},
		bytea:            querier_dto.SQLType{Category: querier_dto.TypeCategoryBytea, EngineName: "blob"},
		date:             querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "date"},
		time:             querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "time"},
		datetime:         querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "datetime"},
		timestamp:        querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "timestamp"},
		json:             querier_dto.SQLType{Category: querier_dto.TypeCategoryJSON, EngineName: "json"},
		geometry:         querier_dto.SQLType{Category: querier_dto.TypeCategoryGeometric, EngineName: "geometry"},
		any:              querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: ""},
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

// FunctionCatalogueBuilder builds a MySQL function catalogue. It is exported so that
// flavour option functions (such as MariaDB) can register additional functions via
// WithExtraFunctions.
type FunctionCatalogueBuilder struct {
	// CatalogueBuilder provides the shared registration mechanism (Add, Args, NullOnNull,
	// NeverNull, CalledOnNull) and the Catalogue being assembled.
	*engine_shared.CatalogueBuilder

	// integer holds the cached MySQL INT type.
	integer querier_dto.SQLType

	// bigint holds the cached MySQL BIGINT type.
	bigint querier_dto.SQLType

	// float holds the cached MySQL FLOAT type.
	float querier_dto.SQLType

	// double holds the cached MySQL DOUBLE type.
	double querier_dto.SQLType

	// decimal holds the cached MySQL DECIMAL type.
	decimal querier_dto.SQLType

	// text holds the cached MySQL TEXT type.
	text querier_dto.SQLType

	// varchar holds the cached MySQL VARCHAR type.
	varchar querier_dto.SQLType

	// boolean holds the cached MySQL BOOLEAN (TINYINT) type.
	boolean querier_dto.SQLType

	// bytea holds the cached MySQL BLOB type.
	bytea querier_dto.SQLType

	// date holds the cached MySQL DATE type.
	date querier_dto.SQLType

	// time holds the cached MySQL TIME type.
	time querier_dto.SQLType

	// datetime holds the cached MySQL DATETIME type.
	datetime querier_dto.SQLType

	// timestamp holds the cached MySQL TIMESTAMP type.
	timestamp querier_dto.SQLType

	// json holds the cached MySQL JSON type.
	json querier_dto.SQLType

	// geometry holds the cached MySQL GEOMETRY type.
	geometry querier_dto.SQLType

	// any holds the wildcard type used for polymorphic argument slots.
	any querier_dto.SQLType
}

// Arg names a single function argument and its SQL type. It aliases the shared toolkit's
// Arg so existing call sites keep using the bare name; the registration methods (Add,
// Args, NullOnNull, NeverNull, CalledOnNull) are promoted from the embedded shared
// builder.
type Arg = engine_shared.Arg

// Integer returns the MySQL INT type.
//
// Returns querier_dto.SQLType which is the cached INT descriptor.
func (b *FunctionCatalogueBuilder) Integer() querier_dto.SQLType { return b.integer }

// Bigint returns the MySQL BIGINT type.
//
// Returns querier_dto.SQLType which is the cached BIGINT descriptor.
func (b *FunctionCatalogueBuilder) Bigint() querier_dto.SQLType { return b.bigint }

// Float returns the MySQL FLOAT type.
//
// Returns querier_dto.SQLType which is the cached FLOAT descriptor.
func (b *FunctionCatalogueBuilder) Float() querier_dto.SQLType { return b.float }

// Double returns the MySQL DOUBLE type.
//
// Returns querier_dto.SQLType which is the cached DOUBLE descriptor.
func (b *FunctionCatalogueBuilder) Double() querier_dto.SQLType { return b.double }

// Decimal returns the MySQL DECIMAL type.
//
// Returns querier_dto.SQLType which is the cached DECIMAL descriptor.
func (b *FunctionCatalogueBuilder) Decimal() querier_dto.SQLType { return b.decimal }

// Text returns the MySQL TEXT type.
//
// Returns querier_dto.SQLType which is the cached TEXT descriptor.
func (b *FunctionCatalogueBuilder) Text() querier_dto.SQLType { return b.text }

// Varchar returns the MySQL VARCHAR type.
//
// Returns querier_dto.SQLType which is the cached VARCHAR descriptor.
func (b *FunctionCatalogueBuilder) Varchar() querier_dto.SQLType { return b.varchar }

// Boolean returns the MySQL BOOLEAN (TINYINT) type.
//
// Returns querier_dto.SQLType which is the cached BOOLEAN descriptor.
func (b *FunctionCatalogueBuilder) Boolean() querier_dto.SQLType { return b.boolean }

// Bytea returns the MySQL BLOB type.
//
// Returns querier_dto.SQLType which is the cached BLOB descriptor.
func (b *FunctionCatalogueBuilder) Bytea() querier_dto.SQLType { return b.bytea }

// Date returns the MySQL DATE type.
//
// Returns querier_dto.SQLType which is the cached DATE descriptor.
func (b *FunctionCatalogueBuilder) Date() querier_dto.SQLType { return b.date }

// Time returns the MySQL TIME type.
//
// Returns querier_dto.SQLType which is the cached TIME descriptor.
func (b *FunctionCatalogueBuilder) Time() querier_dto.SQLType { return b.time }

// Datetime returns the MySQL DATETIME type.
//
// Returns querier_dto.SQLType which is the cached DATETIME descriptor.
func (b *FunctionCatalogueBuilder) Datetime() querier_dto.SQLType { return b.datetime }

// Timestamp returns the MySQL TIMESTAMP type.
//
// Returns querier_dto.SQLType which is the cached TIMESTAMP descriptor.
func (b *FunctionCatalogueBuilder) Timestamp() querier_dto.SQLType { return b.timestamp }

// JSON returns the MySQL JSON type.
//
// Returns querier_dto.SQLType which is the cached JSON descriptor.
func (b *FunctionCatalogueBuilder) JSON() querier_dto.SQLType { return b.json }

// Geometry returns the MySQL GEOMETRY type.
//
// Returns querier_dto.SQLType which is the cached GEOMETRY descriptor.
func (b *FunctionCatalogueBuilder) Geometry() querier_dto.SQLType { return b.geometry }

// Aggregate registers an aggregate function.
//
// Takes name (string) which is the function name.
// Takes arguments ([]querier_dto.FunctionArgument) which is the argument list.
// Takes returnType (querier_dto.SQLType) which is the function's return type.
func (b *FunctionCatalogueBuilder) Aggregate(name string, arguments []querier_dto.FunctionArgument, returnType querier_dto.SQLType) {
	b.Add(name, &querier_dto.FunctionSignature{
		Arguments:         arguments,
		ReturnType:        returnType,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
}

// Window registers a window function.
//
// Takes name (string) which is the function name.
// Takes arguments ([]querier_dto.FunctionArgument) which is the argument list.
// Takes returnType (querier_dto.SQLType) which is the function's return type.
func (b *FunctionCatalogueBuilder) Window(name string, arguments []querier_dto.FunctionArgument, returnType querier_dto.SQLType) {
	b.Add(name, &querier_dto.FunctionSignature{
		Arguments:         arguments,
		ReturnType:        returnType,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})
}

// Variadic registers a variadic function.
//
// Takes name (string) which is the function name.
// Takes arguments ([]querier_dto.FunctionArgument) which is the argument list, with the
// trailing slot acting as the variadic template.
// Takes minArguments (int) which is the minimum number of arguments required.
// Takes returnType (querier_dto.SQLType) which is the function's return type.
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
	b.NullOnNull("abs", b.Args(Arg{Name: paramNameX, Type: b.integer}), b.integer)
	b.NullOnNull("ceil", b.Args(Arg{Name: paramNameX, Type: b.double}), b.integer)
	b.NullOnNull("ceiling", b.Args(Arg{Name: paramNameX, Type: b.double}), b.integer)
	b.NullOnNull("floor", b.Args(Arg{Name: paramNameX, Type: b.double}), b.integer)
	b.NullOnNull("round", b.Args(Arg{Name: paramNameX, Type: b.double}), b.double)
	b.NullOnNull("round", b.Args(Arg{Name: paramNameX, Type: b.double}, Arg{Name: "d", Type: b.integer}), b.double)
	b.NullOnNull("truncate", b.Args(Arg{Name: paramNameX, Type: b.double}, Arg{Name: "d", Type: b.integer}), b.double)
	b.NullOnNull("sign", b.Args(Arg{Name: paramNameX, Type: b.double}), b.integer)
	b.NullOnNull("sqrt", b.Args(Arg{Name: paramNameX, Type: b.double}), b.double)
	b.NullOnNull("power", b.Args(Arg{Name: paramNameX, Type: b.double}, Arg{Name: paramNameY, Type: b.double}), b.double)
	b.NullOnNull("pow", b.Args(Arg{Name: paramNameX, Type: b.double}, Arg{Name: paramNameY, Type: b.double}), b.double)
	b.NullOnNull("exp", b.Args(Arg{Name: paramNameX, Type: b.double}), b.double)
	b.NullOnNull("ln", b.Args(Arg{Name: paramNameX, Type: b.double}), b.double)
	b.NullOnNull("log", b.Args(Arg{Name: paramNameX, Type: b.double}), b.double)
	b.NullOnNull("log", b.Args(Arg{Name: "base", Type: b.double}, Arg{Name: paramNameX, Type: b.double}), b.double)
	b.NullOnNull("log2", b.Args(Arg{Name: paramNameX, Type: b.double}), b.double)
	b.NullOnNull("log10", b.Args(Arg{Name: paramNameX, Type: b.double}), b.double)
	b.NullOnNull("mod", b.Args(Arg{Name: paramNameX, Type: b.double}, Arg{Name: paramNameY, Type: b.double}), b.integer)
	b.NeverNull("pi", nil, b.double)
	b.NeverNull("rand", nil, b.double)
	b.NullOnNull("rand", b.Args(Arg{Name: "seed", Type: b.integer}), b.double)
	b.NullOnNull("degrees", b.Args(Arg{Name: paramNameX, Type: b.double}), b.double)
	b.NullOnNull("radians", b.Args(Arg{Name: paramNameX, Type: b.double}), b.double)
	b.NullOnNull("crc32", b.Args(Arg{Name: paramNameExpression, Type: b.text}), b.integer)
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

// registerStringVariadicFunctions registers CONCAT and CONCAT_WS.
func (b *FunctionCatalogueBuilder) registerStringVariadicFunctions() {
	b.Add("concat", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameValue, Type: b.any}),
		ReturnType:        b.text,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      1,
	})
	b.Add("concat_ws", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameSeparator, Type: b.text}, Arg{Name: paramNameValue, Type: b.any}),
		ReturnType:        b.text,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      2,
	})
}

// registerStringLengthFunctions registers the string-length functions.
//
//nolint:dupl // structurally similar single-string functions
func (b *FunctionCatalogueBuilder) registerStringLengthFunctions() {
	b.NullOnNull("length", b.Args(Arg{Name: paramNameString, Type: b.text}), b.integer)
	b.NullOnNull("char_length", b.Args(Arg{Name: paramNameString, Type: b.text}), b.integer)
	b.NullOnNull("character_length", b.Args(Arg{Name: paramNameString, Type: b.text}), b.integer)
	b.NullOnNull("octet_length", b.Args(Arg{Name: paramNameString, Type: b.text}), b.integer)
}

// registerStringCaseFunctions registers case-conversion functions.
//
//nolint:dupl // structurally similar single-string functions
func (b *FunctionCatalogueBuilder) registerStringCaseFunctions() {
	b.NullOnNull("lower", b.Args(Arg{Name: paramNameString, Type: b.text}), b.text)
	b.NullOnNull("lcase", b.Args(Arg{Name: paramNameString, Type: b.text}), b.text)
	b.NullOnNull("upper", b.Args(Arg{Name: paramNameString, Type: b.text}), b.text)
	b.NullOnNull("ucase", b.Args(Arg{Name: paramNameString, Type: b.text}), b.text)
}

// registerStringTrimFunctions registers TRIM, LTRIM and RTRIM.
func (b *FunctionCatalogueBuilder) registerStringTrimFunctions() {
	b.NullOnNull("trim", b.Args(Arg{Name: paramNameString, Type: b.text}), b.text)
	b.NullOnNull("ltrim", b.Args(Arg{Name: paramNameString, Type: b.text}), b.text)
	b.NullOnNull("rtrim", b.Args(Arg{Name: paramNameString, Type: b.text}), b.text)
}

// registerStringPaddingFunctions registers LPAD and RPAD overloads.
func (b *FunctionCatalogueBuilder) registerStringPaddingFunctions() {
	b.NullOnNull("lpad", b.Args(Arg{Name: paramNameString, Type: b.text}, Arg{Name: paramNameLength, Type: b.integer}, Arg{Name: "pad", Type: b.text}), b.text)
	b.NullOnNull("lpad", b.Args(Arg{Name: paramNameString, Type: b.text}, Arg{Name: paramNameLength, Type: b.integer}), b.text)
	b.NullOnNull("rpad", b.Args(Arg{Name: paramNameString, Type: b.text}, Arg{Name: paramNameLength, Type: b.integer}, Arg{Name: "pad", Type: b.text}), b.text)
	b.NullOnNull("rpad", b.Args(Arg{Name: paramNameString, Type: b.text}, Arg{Name: paramNameLength, Type: b.integer}), b.text)
}

// registerStringTransformFunctions registers REPEAT, REPLACE, REVERSE, SUBSTRING, LEFT
// and RIGHT.
func (b *FunctionCatalogueBuilder) registerStringTransformFunctions() {
	b.NullOnNull("repeat", b.Args(Arg{Name: paramNameString, Type: b.text}, Arg{Name: paramNameCount, Type: b.integer}), b.text)
	b.NullOnNull("replace", b.Args(Arg{Name: paramNameString, Type: b.text}, Arg{Name: "from_str", Type: b.text}, Arg{Name: "to_str", Type: b.text}), b.text)
	b.NullOnNull("reverse", b.Args(Arg{Name: paramNameString, Type: b.text}), b.text)
	b.NullOnNull("substring", b.Args(Arg{Name: paramNameString, Type: b.text}, Arg{Name: paramNameStart, Type: b.integer}), b.text)
	b.NullOnNull("substring", b.Args(Arg{Name: paramNameString, Type: b.text}, Arg{Name: paramNameStart, Type: b.integer}, Arg{Name: paramNameLength, Type: b.integer}), b.text)
	b.NullOnNull("substr", b.Args(Arg{Name: paramNameString, Type: b.text}, Arg{Name: paramNameStart, Type: b.integer}), b.text)
	b.NullOnNull("substr", b.Args(Arg{Name: paramNameString, Type: b.text}, Arg{Name: paramNameStart, Type: b.integer}, Arg{Name: paramNameLength, Type: b.integer}), b.text)
	b.NullOnNull("left", b.Args(Arg{Name: paramNameString, Type: b.text}, Arg{Name: paramNameLength, Type: b.integer}), b.text)
	b.NullOnNull("right", b.Args(Arg{Name: paramNameString, Type: b.text}, Arg{Name: paramNameLength, Type: b.integer}), b.text)
}

// registerStringSearchFunctions registers LOCATE and INSTR.
func (b *FunctionCatalogueBuilder) registerStringSearchFunctions() {
	b.NullOnNull("locate", b.Args(Arg{Name: paramNameSubstring, Type: b.text}, Arg{Name: paramNameString, Type: b.text}), b.integer)
	b.NullOnNull("locate", b.Args(Arg{Name: paramNameSubstring, Type: b.text}, Arg{Name: paramNameString, Type: b.text}, Arg{Name: paramNameStart, Type: b.integer}), b.integer)
	b.NullOnNull("instr", b.Args(Arg{Name: paramNameString, Type: b.text}, Arg{Name: paramNameSubstring, Type: b.text}), b.integer)
}

// registerStringMiscFunctions registers miscellaneous string functions.
func (b *FunctionCatalogueBuilder) registerStringMiscFunctions() {
	b.NullOnNull("ascii", b.Args(Arg{Name: paramNameString, Type: b.text}), b.integer)
	b.NullOnNull("hex", b.Args(Arg{Name: paramNameX, Type: b.any}), b.text)
	b.NullOnNull("unhex", b.Args(Arg{Name: paramNameString, Type: b.text}), b.bytea)
	b.NullOnNull("space", b.Args(Arg{Name: paramNameN, Type: b.integer}), b.text)
	b.NullOnNull("format", b.Args(Arg{Name: paramNameX, Type: b.double}, Arg{Name: paramNameD, Type: b.integer}), b.text)
	b.NullOnNull("format", b.Args(Arg{Name: paramNameX, Type: b.double}, Arg{Name: paramNameD, Type: b.integer}, Arg{Name: paramNameLocale, Type: b.text}), b.text)
}

// registerDateTimeFunctions registers date and time functions.
func (b *FunctionCatalogueBuilder) registerDateTimeFunctions() {
	b.registerDateTimeCurrentFunctions()
	b.registerDateTimeExtractionFunctions()
	b.registerDateTimeFormatFunctions()
	b.registerDateTimeArithmeticFunctions()
}

// registerDateTimeCurrentFunctions registers current date/time accessors.
func (b *FunctionCatalogueBuilder) registerDateTimeCurrentFunctions() {
	b.NeverNull("now", nil, b.datetime)
	b.NeverNull("curdate", nil, b.date)
	b.NeverNull("current_date", nil, b.date)
	b.NeverNull("curtime", nil, b.time)
	b.NeverNull("current_time", nil, b.time)
}

// registerDateTimeExtractionFunctions registers date/time component extraction functions.
func (b *FunctionCatalogueBuilder) registerDateTimeExtractionFunctions() {
	b.NullOnNull("date", b.Args(Arg{Name: paramNameExpression, Type: b.datetime}), b.date)
	b.NullOnNull("time", b.Args(Arg{Name: paramNameExpression, Type: b.datetime}), b.time)
	b.NullOnNull("year", b.Args(Arg{Name: paramNameDate, Type: b.date}), b.integer)
	b.NullOnNull("month", b.Args(Arg{Name: paramNameDate, Type: b.date}), b.integer)
	b.NullOnNull("day", b.Args(Arg{Name: paramNameDate, Type: b.date}), b.integer)
	b.NullOnNull("dayofmonth", b.Args(Arg{Name: paramNameDate, Type: b.date}), b.integer)
	b.NullOnNull("hour", b.Args(Arg{Name: paramNameTime, Type: b.time}), b.integer)
	b.NullOnNull("minute", b.Args(Arg{Name: paramNameTime, Type: b.time}), b.integer)
	b.NullOnNull("second", b.Args(Arg{Name: paramNameTime, Type: b.time}), b.integer)
	b.NullOnNull("dayofweek", b.Args(Arg{Name: paramNameDate, Type: b.date}), b.integer)
	b.NullOnNull("dayofyear", b.Args(Arg{Name: paramNameDate, Type: b.date}), b.integer)
	b.NullOnNull("week", b.Args(Arg{Name: paramNameDate, Type: b.date}), b.integer)
	b.NullOnNull("week", b.Args(Arg{Name: paramNameDate, Type: b.date}, Arg{Name: "mode", Type: b.integer}), b.integer)
	b.NullOnNull("quarter", b.Args(Arg{Name: paramNameDate, Type: b.date}), b.integer)
	b.NullOnNull("last_day", b.Args(Arg{Name: paramNameDate, Type: b.date}), b.date)
}

// registerDateTimeFormatFunctions registers date/time formatting and parsing functions.
func (b *FunctionCatalogueBuilder) registerDateTimeFormatFunctions() {
	b.NullOnNull("date_format", b.Args(Arg{Name: paramNameDate, Type: b.datetime}, Arg{Name: paramNameFormat, Type: b.text}), b.text)
	b.NullOnNull("str_to_date", b.Args(Arg{Name: paramNameString, Type: b.text}, Arg{Name: paramNameFormat, Type: b.text}), b.datetime)
	b.NullOnNull("from_unixtime", b.Args(Arg{Name: paramNameTimestamp, Type: b.bigint}), b.datetime)
	b.NullOnNull("from_unixtime", b.Args(Arg{Name: paramNameTimestamp, Type: b.bigint}, Arg{Name: paramNameFormat, Type: b.text}), b.text)
	b.CalledOnNull("unix_timestamp", nil, b.bigint)
	b.NullOnNull("unix_timestamp", b.Args(Arg{Name: paramNameDate, Type: b.datetime}), b.bigint)
}

// registerDateTimeArithmeticFunctions registers date/time arithmetic functions.
func (b *FunctionCatalogueBuilder) registerDateTimeArithmeticFunctions() {
	b.NullOnNull("datediff", b.Args(Arg{Name: paramNameExpr1, Type: b.date}, Arg{Name: paramNameExpr2, Type: b.date}), b.integer)
	b.NullOnNull("timediff", b.Args(Arg{Name: paramNameExpr1, Type: b.time}, Arg{Name: paramNameExpr2, Type: b.time}), b.time)
	b.NullOnNull("timestampdiff", b.Args(Arg{Name: paramNameUnit, Type: b.text}, Arg{Name: paramNameExpr1, Type: b.datetime}, Arg{Name: paramNameExpr2, Type: b.datetime}), b.bigint)
	b.NullOnNull("timestampadd", b.Args(Arg{Name: paramNameUnit, Type: b.text}, Arg{Name: paramNameInterval, Type: b.integer}, Arg{Name: paramNameDatetime, Type: b.datetime}), b.datetime)
	b.NullOnNull("date_add", b.Args(Arg{Name: paramNameDate, Type: b.datetime}, Arg{Name: "interval", Type: b.any}), b.datetime)
	b.NullOnNull("date_sub", b.Args(Arg{Name: paramNameDate, Type: b.datetime}, Arg{Name: "interval", Type: b.any}), b.datetime)
}

// registerJSONFunctions registers JSON functions.
func (b *FunctionCatalogueBuilder) registerJSONFunctions() {
	b.registerJSONAccessFunctions()
	b.registerJSONMutationFunctions()
	b.registerJSONBuildFunctions()
	b.registerJSONIntrospectionFunctions()
}

// registerJSONAccessFunctions registers JSON accessor functions.
func (b *FunctionCatalogueBuilder) registerJSONAccessFunctions() {
	b.NullOnNull("json_extract", b.Args(Arg{Name: paramNameJSON, Type: b.json}, Arg{Name: paramNamePath, Type: b.text}), b.json)
	b.NullOnNull("json_unquote", b.Args(Arg{Name: paramNameJSON, Type: b.json}), b.text)
}

// registerJSONMutationFunctions registers JSON mutation functions.
func (b *FunctionCatalogueBuilder) registerJSONMutationFunctions() {
	b.NullOnNull("json_set", b.Args(Arg{Name: paramNameJSON, Type: b.json}, Arg{Name: paramNamePath, Type: b.text}, Arg{Name: paramNameValue, Type: b.any}), b.json)
	b.NullOnNull("json_insert", b.Args(Arg{Name: paramNameJSON, Type: b.json}, Arg{Name: paramNamePath, Type: b.text}, Arg{Name: paramNameValue, Type: b.any}), b.json)
	b.NullOnNull("json_replace", b.Args(Arg{Name: paramNameJSON, Type: b.json}, Arg{Name: paramNamePath, Type: b.text}, Arg{Name: paramNameValue, Type: b.any}), b.json)
	b.NullOnNull("json_remove", b.Args(Arg{Name: paramNameJSON, Type: b.json}, Arg{Name: paramNamePath, Type: b.text}), b.json)
}

// registerJSONBuildFunctions registers JSON construction functions.
func (b *FunctionCatalogueBuilder) registerJSONBuildFunctions() {
	b.Add("json_object", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: "key", Type: b.text}, Arg{Name: paramNameValue, Type: b.any}),
		ReturnType:        b.json,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
		IsVariadic:        true,
		MinArguments:      0,
	})
	b.Add("json_array", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameValue, Type: b.any}),
		ReturnType:        b.json,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
		IsVariadic:        true,
		MinArguments:      0,
	})
}

// registerJSONIntrospectionFunctions registers JSON introspection functions.
func (b *FunctionCatalogueBuilder) registerJSONIntrospectionFunctions() {
	b.NullOnNull("json_contains", b.Args(Arg{Name: paramNameTarget, Type: b.json}, Arg{Name: "candidate", Type: b.json}), b.boolean)
	b.NullOnNull("json_contains", b.Args(Arg{Name: paramNameTarget, Type: b.json}, Arg{Name: "candidate", Type: b.json}, Arg{Name: paramNamePath, Type: b.text}), b.boolean)
	b.NullOnNull("json_contains_path", b.Args(Arg{Name: paramNameJSON, Type: b.json}, Arg{Name: "one_or_all", Type: b.text}, Arg{Name: paramNamePath, Type: b.text}), b.boolean)
	b.NullOnNull("json_type", b.Args(Arg{Name: paramNameJSON, Type: b.json}), b.text)
	b.NullOnNull("json_valid", b.Args(Arg{Name: paramNameValue, Type: b.any}), b.boolean)
	b.NullOnNull("json_length", b.Args(Arg{Name: paramNameJSON, Type: b.json}), b.integer)
	b.NullOnNull("json_length", b.Args(Arg{Name: paramNameJSON, Type: b.json}, Arg{Name: paramNamePath, Type: b.text}), b.integer)
	b.NullOnNull("json_keys", b.Args(Arg{Name: paramNameJSON, Type: b.json}), b.json)
	b.NullOnNull("json_keys", b.Args(Arg{Name: paramNameJSON, Type: b.json}, Arg{Name: paramNamePath, Type: b.text}), b.json)
}

// registerAggregateFunctions registers aggregate functions.
func (b *FunctionCatalogueBuilder) registerAggregateFunctions() {
	b.Add("count", &querier_dto.FunctionSignature{
		ReturnType:        b.bigint,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})
	b.Add("count", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.any}),
		ReturnType:        b.bigint,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})

	b.Aggregate("sum", b.Args(Arg{Name: paramNameX, Type: b.any}), b.decimal)
	b.Aggregate("avg", b.Args(Arg{Name: paramNameX, Type: b.any}), b.double)
	b.Aggregate("min", b.Args(Arg{Name: paramNameX, Type: b.any}), b.any)
	b.Aggregate("max", b.Args(Arg{Name: paramNameX, Type: b.any}), b.any)

	b.Add("group_concat", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameExpression, Type: b.any}),
		ReturnType:        b.text,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})

	b.Aggregate("bit_and", b.Args(Arg{Name: paramNameX, Type: b.any}), b.any)
	b.Aggregate("bit_or", b.Args(Arg{Name: paramNameX, Type: b.any}), b.any)
	b.Aggregate("bit_xor", b.Args(Arg{Name: paramNameX, Type: b.any}), b.any)

	b.Aggregate("std", b.Args(Arg{Name: paramNameX, Type: b.any}), b.double)
	b.Aggregate("stddev", b.Args(Arg{Name: paramNameX, Type: b.any}), b.double)
}

// registerWindowFunctions registers window functions.
func (b *FunctionCatalogueBuilder) registerWindowFunctions() {
	b.Window("row_number", nil, b.bigint)
	b.Window("rank", nil, b.bigint)
	b.Window("dense_rank", nil, b.bigint)
	b.Window("ntile", b.Args(Arg{Name: paramNameN, Type: b.integer}), b.integer)

	windowValueArgs := b.Args(Arg{Name: paramNameExpression, Type: b.any}, Arg{Name: "offset", Type: b.integer}, Arg{Name: "default", Type: b.any})

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

	b.CalledOnNull("first_value", b.Args(Arg{Name: paramNameExpression, Type: b.any}), b.any)
	b.CalledOnNull("last_value", b.Args(Arg{Name: paramNameExpression, Type: b.any}), b.any)
	b.CalledOnNull("nth_value", b.Args(Arg{Name: paramNameExpression, Type: b.any}, Arg{Name: paramNameN, Type: b.integer}), b.any)
}

// registerConditionalFunctions registers conditional expression functions.
func (b *FunctionCatalogueBuilder) registerConditionalFunctions() {
	b.CalledOnNull("if", b.Args(Arg{Name: paramNameCondition, Type: b.boolean}, Arg{Name: paramNameThen, Type: b.any}, Arg{Name: paramNameElse, Type: b.any}), b.text)
	b.CalledOnNull("ifnull", b.Args(Arg{Name: paramNameExpr1, Type: b.any}, Arg{Name: paramNameExpr2, Type: b.any}), b.any)
	b.CalledOnNull("nullif", b.Args(Arg{Name: paramNameExpr1, Type: b.any}, Arg{Name: paramNameExpr2, Type: b.any}), b.any)

	b.Add("coalesce", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameValue, Type: b.any}),
		ReturnType:        b.any,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      1,
	})

	b.Add("greatest", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameValue, Type: b.any}),
		ReturnType:        b.any,
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		IsVariadic:        true,
		MinArguments:      1,
	})
	b.Add("least", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameValue, Type: b.any}),
		ReturnType:        b.any,
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
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
//
// INET6_ATON / INET6_NTOA are standard MySQL functions since 5.6.3, so they live in the
// base catalogue rather than the MariaDB-specific set; on plain MySQL they would
// otherwise resolve untyped.
func (b *FunctionCatalogueBuilder) registerTypeConversionFunctions() {
	varbinary := querier_dto.SQLType{Category: querier_dto.TypeCategoryBytea, EngineName: "varbinary"}

	b.NullOnNull("inet_aton", b.Args(Arg{Name: paramNameExpression, Type: b.text}), b.bigint)
	b.NullOnNull("inet_ntoa", b.Args(Arg{Name: paramNameExpression, Type: b.bigint}), b.text)
	b.NullOnNull("inet6_aton", b.Args(Arg{Name: "address", Type: b.varchar}), varbinary)
	b.NullOnNull("inet6_ntoa", b.Args(Arg{Name: paramNameValue, Type: varbinary}), b.varchar)
}
