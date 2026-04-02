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

package db_engine_sqlite

import (
	"piko.sh/piko/internal/querier/querier_dto"
)

func buildFunctionCatalogue() *querier_dto.FunctionCatalogue {
	builder := &functionCatalogueBuilder{
		catalogue: &querier_dto.FunctionCatalogue{
			Functions: make(map[string][]*querier_dto.FunctionSignature),
		},
		integer: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "integer"},
		text:    querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
		real:    querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "real"},
		blob:    querier_dto.SQLType{Category: querier_dto.TypeCategoryBytea, EngineName: "blob"},
		any:     querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: ""},
	}

	builder.registerCoreFunctions()
	builder.registerMathFunctions()
	builder.registerDateTimeFunctions()
	builder.registerStringFunctions()
	builder.registerJSONScalarFunctions()
	builder.registerJSONAggregateFunctions()
	builder.registerAggregateFunctions()
	builder.registerWindowRankingFunctions()
	builder.registerWindowValueFunctions()
	builder.registerFTS5Functions()
	builder.registerRTreeFunctions()

	return builder.catalogue
}

type functionCatalogueBuilder struct {
	catalogue *querier_dto.FunctionCatalogue

	integer querier_dto.SQLType

	text querier_dto.SQLType

	real querier_dto.SQLType

	blob querier_dto.SQLType

	any querier_dto.SQLType
}

func (b *functionCatalogueBuilder) add(name string, signature *querier_dto.FunctionSignature) {
	signature.Name = name
	signature.DataAccess = querier_dto.DataAccessReadOnly
	b.catalogue.Functions[name] = append(b.catalogue.Functions[name], signature)
}

func (*functionCatalogueBuilder) args(pairs ...any) []querier_dto.FunctionArgument {
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

func (b *functionCatalogueBuilder) nullOnNull(name string, arguments []querier_dto.FunctionArgument, returnType querier_dto.SQLType) {
	b.add(name, &querier_dto.FunctionSignature{
		Arguments:         arguments,
		ReturnType:        returnType,
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
	})
}

func (b *functionCatalogueBuilder) neverNull(name string, arguments []querier_dto.FunctionArgument, returnType querier_dto.SQLType) {
	b.add(name, &querier_dto.FunctionSignature{
		Arguments:         arguments,
		ReturnType:        returnType,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})
}

func (b *functionCatalogueBuilder) calledOnNull(name string, arguments []querier_dto.FunctionArgument, returnType querier_dto.SQLType) {
	b.add(name, &querier_dto.FunctionSignature{
		Arguments:         arguments,
		ReturnType:        returnType,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
}

// registerCoreFunctions registers core scalar functions present since early
// SQLite versions.
func (b *functionCatalogueBuilder) registerCoreFunctions() {
	b.nullOnNull("abs", b.args(paramNameX, b.any), b.any)
	b.nullOnNull("length", b.args(paramNameX, b.any), b.integer)
	b.nullOnNull("lower", b.args(paramNameX, b.text), b.text)
	b.nullOnNull("upper", b.args(paramNameX, b.text), b.text)
	b.nullOnNull("trim", b.args(paramNameX, b.text), b.text)
	b.nullOnNull("ltrim", b.args(paramNameX, b.text), b.text)
	b.nullOnNull("rtrim", b.args(paramNameX, b.text), b.text)
	b.nullOnNull("replace", b.args(paramNameStr, b.text, "from", b.text, "to", b.text), b.text)
	b.nullOnNull("substr", b.args(paramNameStr, b.text, "start", b.integer, "length", b.integer), b.text)
	b.nullOnNull("substring", b.args(paramNameStr, b.text, "start", b.integer, "length", b.integer), b.text)
	b.nullOnNull("hex", b.args(paramNameX, b.any), b.text)
	b.nullOnNull("unhex", b.args(paramNameX, b.text), b.blob)
	b.nullOnNull("instr", b.args(paramNameStr, b.text, "substr", b.text), b.integer)
	b.nullOnNull("unicode", b.args(paramNameX, b.text), b.integer)
	b.nullOnNull("zeroblob", b.args("n", b.integer), b.blob)
	b.nullOnNull("round", b.args(paramNameX, b.real), b.real)
	b.nullOnNull("sign", b.args(paramNameX, b.any), b.integer)

	b.neverNull("typeof", b.args(paramNameX, b.any), b.text)
	b.neverNull("quote", b.args(paramNameX, b.any), b.text)
	b.neverNull("char", nil, b.text)
	b.neverNull("printf", b.args(paramNameFormat, b.text), b.text)
	b.neverNull(paramNameFormat, b.args(paramNameFormat, b.text), b.text)
	b.neverNull("random", nil, b.integer)
	b.neverNull("changes", nil, b.integer)
	b.neverNull("last_insert_rowid", nil, b.integer)
	b.neverNull("total_changes", nil, b.integer)

	b.calledOnNull("nullif", b.args(paramNameX, b.any, paramNameY, b.any), b.any)
	b.calledOnNull("ifnull", b.args(paramNameX, b.any, paramNameY, b.any), b.any)
	b.calledOnNull("iif", b.args("condition", b.any, paramNameX, b.any, paramNameY, b.any), b.any)
	b.calledOnNull("max", b.args(paramNameX, b.any, paramNameY, b.any), b.any)
	b.calledOnNull("min", b.args(paramNameX, b.any, paramNameY, b.any), b.any)
	b.calledOnNull("likelihood", b.args(paramNameX, b.any, "probability", b.real), b.any)
	b.calledOnNull("likely", b.args(paramNameX, b.any), b.any)
	b.calledOnNull("unlikely", b.args(paramNameX, b.any), b.any)
}

// registerMathFunctions registers math functions added in SQLite 3.35+.
func (b *functionCatalogueBuilder) registerMathFunctions() {
	b.nullOnNull("acos", b.args(paramNameX, b.real), b.real)
	b.nullOnNull("asin", b.args(paramNameX, b.real), b.real)
	b.nullOnNull("atan", b.args(paramNameX, b.real), b.real)
	b.nullOnNull("atan2", b.args(paramNameY, b.real, paramNameX, b.real), b.real)
	b.nullOnNull("cos", b.args(paramNameX, b.real), b.real)
	b.nullOnNull("sin", b.args(paramNameX, b.real), b.real)
	b.nullOnNull("tan", b.args(paramNameX, b.real), b.real)
	b.nullOnNull("ceil", b.args(paramNameX, b.real), b.integer)
	b.nullOnNull("ceiling", b.args(paramNameX, b.real), b.integer)
	b.nullOnNull("floor", b.args(paramNameX, b.real), b.integer)
	b.nullOnNull("trunc", b.args(paramNameX, b.real), b.integer)
	b.nullOnNull("sqrt", b.args(paramNameX, b.real), b.real)
	b.nullOnNull("exp", b.args(paramNameX, b.real), b.real)
	b.nullOnNull("ln", b.args(paramNameX, b.real), b.real)
	b.nullOnNull("log2", b.args(paramNameX, b.real), b.real)
	b.nullOnNull("log10", b.args(paramNameX, b.real), b.real)
	b.nullOnNull("pow", b.args(paramNameX, b.real, paramNameY, b.real), b.real)
	b.nullOnNull("power", b.args(paramNameX, b.real, paramNameY, b.real), b.real)
	b.nullOnNull("mod", b.args(paramNameX, b.any, paramNameY, b.any), b.any)
	b.nullOnNull("degrees", b.args(paramNameX, b.real), b.real)
	b.nullOnNull("radians", b.args(paramNameX, b.real), b.real)

	b.nullOnNull("log", b.args(paramNameX, b.real), b.real)
	b.nullOnNull("log", b.args("base", b.real, paramNameX, b.real), b.real)

	b.neverNull("pi", nil, b.real)
}

// registerDateTimeFunctions registers SQLite date and time functions.
func (b *functionCatalogueBuilder) registerDateTimeFunctions() {
	timeArgs := b.args("timestring", b.text, "modifier", b.text)

	b.add("date", &querier_dto.FunctionSignature{Arguments: timeArgs, ReturnType: b.text, NullableBehaviour: querier_dto.FunctionNullableCalledOnNull, IsVariadic: true, MinArguments: 1})
	b.add("time", &querier_dto.FunctionSignature{Arguments: timeArgs, ReturnType: b.text, NullableBehaviour: querier_dto.FunctionNullableCalledOnNull, IsVariadic: true, MinArguments: 1})
	b.add("datetime", &querier_dto.FunctionSignature{Arguments: timeArgs, ReturnType: b.text, NullableBehaviour: querier_dto.FunctionNullableCalledOnNull, IsVariadic: true, MinArguments: 1})
	b.add("julianday", &querier_dto.FunctionSignature{Arguments: timeArgs, ReturnType: b.real, NullableBehaviour: querier_dto.FunctionNullableCalledOnNull, IsVariadic: true, MinArguments: 1})
	b.add("unixepoch", &querier_dto.FunctionSignature{Arguments: timeArgs, ReturnType: b.integer, NullableBehaviour: querier_dto.FunctionNullableCalledOnNull, IsVariadic: true, MinArguments: 1})

	b.add("strftime", &querier_dto.FunctionSignature{
		Arguments:         b.args(paramNameFormat, b.text, "timestring", b.text, "modifier", b.text),
		ReturnType:        b.text,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      2,
	})

	b.calledOnNull("timediff", b.args("time1", b.text, "time2", b.text), b.text)
}

// registerStringFunctions registers additional string functions beyond the
// core set (lower, upper, trim, etc. are in registerCoreFunctions).
func (b *functionCatalogueBuilder) registerStringFunctions() {
	b.nullOnNull("glob", b.args("pattern", b.text, "string", b.text), b.integer)
	b.nullOnNull("like", b.args("pattern", b.text, "string", b.text), b.integer)
	b.nullOnNull("like", b.args("pattern", b.text, "string", b.text, "escape", b.text), b.integer)
	b.nullOnNull("soundex", b.args(paramNameX, b.text), b.text)
}

// registerJSONScalarFunctions registers JSON scalar functions.
func (b *functionCatalogueBuilder) registerJSONScalarFunctions() {
	b.nullOnNull(paramNameJSON, b.args(paramNameX, b.text), b.text)
	b.nullOnNull("json_extract", b.args(paramNameJSON, b.text, "path", b.text), b.any)
	b.nullOnNull("json_quote", b.args(paramNameValue, b.any), b.text)
	b.nullOnNull("json_patch", b.args("json1", b.text, "json2", b.text), b.text)

	b.neverNull("json_array", nil, b.text)
	b.neverNull("json_object", nil, b.text)
	b.neverNull("json_type", b.args(paramNameJSON, b.text), b.text)
	b.neverNull("json_valid", b.args(paramNameX, b.text), b.integer)

	mutatorArgs := b.args(paramNameJSON, b.text, "path", b.text, paramNameValue, b.any)
	for _, name := range []string{"json_set", "json_insert", "json_replace"} {
		b.add(name, &querier_dto.FunctionSignature{
			Arguments:         mutatorArgs,
			ReturnType:        b.text,
			NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
			IsVariadic:        true,
			MinArguments:      3,
		})
	}

	b.add("json_remove", &querier_dto.FunctionSignature{
		Arguments:         b.args(paramNameJSON, b.text, "path", b.text),
		ReturnType:        b.text,
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		IsVariadic:        true,
		MinArguments:      2,
	})
}

// registerJSONAggregateFunctions registers JSON aggregate functions.
func (b *functionCatalogueBuilder) registerJSONAggregateFunctions() {
	b.add("json_group_array", &querier_dto.FunctionSignature{
		Arguments:         b.args(paramNameValue, b.any),
		ReturnType:        b.text,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})
	b.add("json_group_object", &querier_dto.FunctionSignature{
		Arguments:         b.args("name", b.text, paramNameValue, b.any),
		ReturnType:        b.text,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})
}

// registerAggregateFunctions registers core aggregate functions.
func (b *functionCatalogueBuilder) registerAggregateFunctions() {
	b.add("count", &querier_dto.FunctionSignature{ReturnType: b.integer, IsAggregate: true, NullableBehaviour: querier_dto.FunctionNullableNeverNull})
	b.add("count", &querier_dto.FunctionSignature{Arguments: b.args(paramNameX, b.any), ReturnType: b.integer, IsAggregate: true, NullableBehaviour: querier_dto.FunctionNullableNeverNull})
	b.add("total", &querier_dto.FunctionSignature{Arguments: b.args(paramNameX, b.any), ReturnType: b.real, IsAggregate: true, NullableBehaviour: querier_dto.FunctionNullableNeverNull})

	b.add("avg", &querier_dto.FunctionSignature{Arguments: b.args(paramNameX, b.any), ReturnType: b.real, IsAggregate: true, NullableBehaviour: querier_dto.FunctionNullableCalledOnNull})
	b.add("sum", &querier_dto.FunctionSignature{Arguments: b.args(paramNameX, b.any), ReturnType: b.any, IsAggregate: true, NullableBehaviour: querier_dto.FunctionNullableCalledOnNull})
	b.add("group_concat", &querier_dto.FunctionSignature{Arguments: b.args(paramNameX, b.any), ReturnType: b.text, IsAggregate: true, NullableBehaviour: querier_dto.FunctionNullableCalledOnNull})

	b.add("max", &querier_dto.FunctionSignature{Arguments: b.args(paramNameX, b.any), ReturnType: b.any, IsAggregate: true, NullableBehaviour: querier_dto.FunctionNullableCalledOnNull})
	b.add("min", &querier_dto.FunctionSignature{Arguments: b.args(paramNameX, b.any), ReturnType: b.any, IsAggregate: true, NullableBehaviour: querier_dto.FunctionNullableCalledOnNull})
	b.add("group_concat", &querier_dto.FunctionSignature{
		Arguments:         b.args(paramNameX, b.any, "separator", b.text),
		ReturnType:        b.text,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
}

// registerWindowRankingFunctions registers window ranking functions
// (row_number, rank, dense_rank, etc.).
func (b *functionCatalogueBuilder) registerWindowRankingFunctions() {
	b.neverNull("row_number", nil, b.integer)
	b.neverNull("rank", nil, b.integer)
	b.neverNull("dense_rank", nil, b.integer)
	b.neverNull("ntile", b.args("n", b.integer), b.integer)
	b.neverNull("cume_dist", nil, b.real)
	b.neverNull("percent_rank", nil, b.real)
}

// registerWindowValueFunctions registers window value-access functions
// (lag, lead, first_value, last_value, nth_value).
func (b *functionCatalogueBuilder) registerWindowValueFunctions() {
	windowValueArgs := b.args(paramNameExpression, b.any, "offset", b.integer, "default", b.any)

	b.add("lag", &querier_dto.FunctionSignature{Arguments: windowValueArgs, ReturnType: b.any, NullableBehaviour: querier_dto.FunctionNullableCalledOnNull, MinArguments: 1})
	b.add("lead", &querier_dto.FunctionSignature{Arguments: windowValueArgs, ReturnType: b.any, NullableBehaviour: querier_dto.FunctionNullableCalledOnNull, MinArguments: 1})

	b.calledOnNull("first_value", b.args(paramNameExpression, b.any), b.any)
	b.calledOnNull("last_value", b.args(paramNameExpression, b.any), b.any)
	b.calledOnNull("nth_value", b.args(paramNameExpression, b.any, "n", b.integer), b.any)
}

// registerFTS5Functions registers FTS5 full-text search auxiliary functions.
func (b *functionCatalogueBuilder) registerFTS5Functions() {
	b.calledOnNull("highlight", b.args(paramNameTable, b.text, "column_index", b.integer, "open_tag", b.text, "close_tag", b.text), b.text)
	b.calledOnNull("snippet", b.args(paramNameTable, b.text, "column_index", b.integer, "open_tag", b.text, "close_tag", b.text, "ellipsis", b.text, "max_tokens", b.integer), b.text)

	b.add("bm25", &querier_dto.FunctionSignature{
		Arguments:         b.args(paramNameTable, b.text, "weight", b.real),
		ReturnType:        b.real,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      1,
	})

	b.calledOnNull("matchinfo", b.args(paramNameTable, b.text), b.blob)
	b.calledOnNull("matchinfo", b.args(paramNameTable, b.text, paramNameFormat, b.text), b.blob)
}

// registerRTreeFunctions registers R-Tree diagnostic functions.
func (b *functionCatalogueBuilder) registerRTreeFunctions() {
	b.calledOnNull("rtreecheck", b.args(paramNameTable, b.text), b.text)
	b.calledOnNull("rtreecheck", b.args("schema", b.text, paramNameTable, b.text), b.text)
	b.calledOnNull("rtreenode", b.args("pageno", b.integer, "data", b.blob), b.text)
}
