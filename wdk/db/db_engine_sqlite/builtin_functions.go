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
	"piko.sh/piko/internal/querier/querier_adapters/engine_shared"
	"piko.sh/piko/internal/querier/querier_dto"
)

// buildFunctionCatalogue assembles every built-in SQLite function signature.
//
// Returns *querier_dto.FunctionCatalogue which contains every supported scalar,
// aggregate, window, FTS5, and R-Tree function.
func buildFunctionCatalogue() *querier_dto.FunctionCatalogue {
	builder := &FunctionCatalogueBuilder{
		CatalogueBuilder: engine_shared.NewCatalogueBuilder(),
		Integer:          querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "integer"},
		Bigint:           querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int8"},
		Text:             querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
		Real:             querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "real"},
		Blob:             querier_dto.SQLType{Category: querier_dto.TypeCategoryBytea, EngineName: "blob"},
		Any:              querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: ""},
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

	return builder.Catalogue
}

// FunctionCatalogueBuilder builds a SQLite function catalogue. It embeds the shared
// builder for the registration mechanism and keeps only its dialect type fields.
type FunctionCatalogueBuilder struct {
	// CatalogueBuilder provides the shared registration mechanism (Add, Args, NullOnNull,
	// NeverNull, CalledOnNull) and the Catalogue being assembled.
	*engine_shared.CatalogueBuilder

	// Integer is the SQL integer type used by signatures. It resolves to a 32-bit Go int,
	// matching the declared width of an ordinary INTEGER column.
	Integer querier_dto.SQLType

	// Bigint is the 64-bit SQL integer type.
	//
	// It applies to signatures whose result is genuinely a signed 64-bit value: count and
	// the window-ranking functions (which can exceed 2.1B over a large result set), the
	// 64-bit connection counters (changes/total_changes), and last_insert_rowid (the rowid
	// is a 64-bit value). SQLite returns all of these via the sqlite3 64-bit C API, matching
	// the int64 the other engines emit for COUNT and friends.
	Bigint querier_dto.SQLType

	// Text is the SQL text type used by signatures.
	Text querier_dto.SQLType

	// Real is the SQL real type used by signatures.
	Real querier_dto.SQLType

	// Blob is the SQL blob type used by signatures.
	Blob querier_dto.SQLType

	// Any is the SQL unknown type used by polymorphic signatures.
	Any querier_dto.SQLType
}

// Arg names a single function argument and its SQL type. It aliases the shared toolkit's
// Arg so call sites keep using the bare name; the registration methods (Add, Args,
// NullOnNull, NeverNull, CalledOnNull) are promoted from the embedded shared builder.
type Arg = engine_shared.Arg

// registerCoreFunctions registers core scalar functions present since early SQLite
// versions.
func (b *FunctionCatalogueBuilder) registerCoreFunctions() {
	b.NullOnNull("abs", b.Args(Arg{Name: paramNameX, Type: b.Any}), b.Any)
	b.NullOnNull("length", b.Args(Arg{Name: paramNameX, Type: b.Any}), b.Integer)
	b.NullOnNull("lower", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Text)
	b.NullOnNull("upper", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Text)
	b.NullOnNull("trim", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Text)
	b.NullOnNull("trim", b.Args(Arg{Name: paramNameX, Type: b.Text}, Arg{Name: paramNameY, Type: b.Text}), b.Text)
	b.NullOnNull("ltrim", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Text)
	b.NullOnNull("ltrim", b.Args(Arg{Name: paramNameX, Type: b.Text}, Arg{Name: paramNameY, Type: b.Text}), b.Text)
	b.NullOnNull("rtrim", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Text)
	b.NullOnNull("rtrim", b.Args(Arg{Name: paramNameX, Type: b.Text}, Arg{Name: paramNameY, Type: b.Text}), b.Text)
	b.NullOnNull("replace", b.Args(Arg{Name: paramNameStr, Type: b.Text}, Arg{Name: "from", Type: b.Text}, Arg{Name: "to", Type: b.Text}), b.Text)
	b.NullOnNull("substr", b.Args(Arg{Name: paramNameStr, Type: b.Text}, Arg{Name: "start", Type: b.Integer}), b.Text)
	b.NullOnNull("substr", b.Args(Arg{Name: paramNameStr, Type: b.Text}, Arg{Name: "start", Type: b.Integer}, Arg{Name: "length", Type: b.Integer}), b.Text)
	b.NullOnNull("substring", b.Args(Arg{Name: paramNameStr, Type: b.Text}, Arg{Name: "start", Type: b.Integer}), b.Text)
	b.NullOnNull("substring", b.Args(Arg{Name: paramNameStr, Type: b.Text}, Arg{Name: "start", Type: b.Integer}, Arg{Name: "length", Type: b.Integer}), b.Text)
	b.NullOnNull("hex", b.Args(Arg{Name: paramNameX, Type: b.Any}), b.Text)
	b.NullOnNull("unhex", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Blob)
	b.NullOnNull("instr", b.Args(Arg{Name: paramNameStr, Type: b.Text}, Arg{Name: "substr", Type: b.Text}), b.Integer)
	b.NullOnNull("unicode", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Integer)
	b.NullOnNull("zeroblob", b.Args(Arg{Name: "n", Type: b.Integer}), b.Blob)
	b.NullOnNull("round", b.Args(Arg{Name: paramNameX, Type: b.Real}), b.Real)
	b.NullOnNull("round", b.Args(Arg{Name: paramNameX, Type: b.Real}, Arg{Name: paramNameY, Type: b.Integer}), b.Real)
	b.NullOnNull("sign", b.Args(Arg{Name: paramNameX, Type: b.Any}), b.Integer)

	b.NeverNull("typeof", b.Args(Arg{Name: paramNameX, Type: b.Any}), b.Text)
	b.NeverNull("quote", b.Args(Arg{Name: paramNameX, Type: b.Any}), b.Text)
	b.NeverNull("char", nil, b.Text)
	b.NeverNull("printf", b.Args(Arg{Name: paramNameFormat, Type: b.Text}), b.Text)
	b.NeverNull(paramNameFormat, b.Args(Arg{Name: paramNameFormat, Type: b.Text}), b.Text)
	b.NeverNull("random", nil, b.Bigint)
	b.NeverNull("changes", nil, b.Bigint)
	b.NeverNull("last_insert_rowid", nil, b.Bigint)
	b.NeverNull("total_changes", nil, b.Bigint)

	b.CalledOnNull("nullif", b.Args(Arg{Name: paramNameX, Type: b.Any}, Arg{Name: paramNameY, Type: b.Any}), b.Any)
	b.CalledOnNull("ifnull", b.Args(Arg{Name: paramNameX, Type: b.Any}, Arg{Name: paramNameY, Type: b.Any}), b.Any)
	b.CalledOnNull("iif", b.Args(Arg{Name: "condition", Type: b.Any}, Arg{Name: paramNameX, Type: b.Any}, Arg{Name: paramNameY, Type: b.Any}), b.Any)
	b.CalledOnNull("max", b.Args(Arg{Name: paramNameX, Type: b.Any}, Arg{Name: paramNameY, Type: b.Any}), b.Any)
	b.CalledOnNull("min", b.Args(Arg{Name: paramNameX, Type: b.Any}, Arg{Name: paramNameY, Type: b.Any}), b.Any)
	b.CalledOnNull("likelihood", b.Args(Arg{Name: paramNameX, Type: b.Any}, Arg{Name: "probability", Type: b.Real}), b.Any)
	b.CalledOnNull("likely", b.Args(Arg{Name: paramNameX, Type: b.Any}), b.Any)
	b.CalledOnNull("unlikely", b.Args(Arg{Name: paramNameX, Type: b.Any}), b.Any)
}

// registerMathFunctions registers math functions added in SQLite 3.35+.
func (b *FunctionCatalogueBuilder) registerMathFunctions() {
	b.NullOnNull("acos", b.Args(Arg{Name: paramNameX, Type: b.Real}), b.Real)
	b.NullOnNull("asin", b.Args(Arg{Name: paramNameX, Type: b.Real}), b.Real)
	b.NullOnNull("atan", b.Args(Arg{Name: paramNameX, Type: b.Real}), b.Real)
	b.NullOnNull("atan2", b.Args(Arg{Name: paramNameY, Type: b.Real}, Arg{Name: paramNameX, Type: b.Real}), b.Real)
	b.NullOnNull("cos", b.Args(Arg{Name: paramNameX, Type: b.Real}), b.Real)
	b.NullOnNull("sin", b.Args(Arg{Name: paramNameX, Type: b.Real}), b.Real)
	b.NullOnNull("tan", b.Args(Arg{Name: paramNameX, Type: b.Real}), b.Real)
	b.NullOnNull("ceil", b.Args(Arg{Name: paramNameX, Type: b.Real}), b.Integer)
	b.NullOnNull("ceiling", b.Args(Arg{Name: paramNameX, Type: b.Real}), b.Integer)
	b.NullOnNull("floor", b.Args(Arg{Name: paramNameX, Type: b.Real}), b.Integer)
	b.NullOnNull("trunc", b.Args(Arg{Name: paramNameX, Type: b.Real}), b.Integer)
	b.NullOnNull("sqrt", b.Args(Arg{Name: paramNameX, Type: b.Real}), b.Real)
	b.NullOnNull("exp", b.Args(Arg{Name: paramNameX, Type: b.Real}), b.Real)
	b.NullOnNull("ln", b.Args(Arg{Name: paramNameX, Type: b.Real}), b.Real)
	b.NullOnNull("log2", b.Args(Arg{Name: paramNameX, Type: b.Real}), b.Real)
	b.NullOnNull("log10", b.Args(Arg{Name: paramNameX, Type: b.Real}), b.Real)
	b.NullOnNull("pow", b.Args(Arg{Name: paramNameX, Type: b.Real}, Arg{Name: paramNameY, Type: b.Real}), b.Real)
	b.NullOnNull("power", b.Args(Arg{Name: paramNameX, Type: b.Real}, Arg{Name: paramNameY, Type: b.Real}), b.Real)
	b.NullOnNull("mod", b.Args(Arg{Name: paramNameX, Type: b.Any}, Arg{Name: paramNameY, Type: b.Any}), b.Any)
	b.NullOnNull("degrees", b.Args(Arg{Name: paramNameX, Type: b.Real}), b.Real)
	b.NullOnNull("radians", b.Args(Arg{Name: paramNameX, Type: b.Real}), b.Real)

	b.NullOnNull("log", b.Args(Arg{Name: paramNameX, Type: b.Real}), b.Real)
	b.NullOnNull("log", b.Args(Arg{Name: "base", Type: b.Real}, Arg{Name: paramNameX, Type: b.Real}), b.Real)

	b.NeverNull("pi", nil, b.Real)
}

// registerDateTimeFunctions registers SQLite date and time functions.
func (b *FunctionCatalogueBuilder) registerDateTimeFunctions() {
	timeArgs := b.Args(Arg{Name: "timestring", Type: b.Text}, Arg{Name: "modifier", Type: b.Text})

	b.Add("date", &querier_dto.FunctionSignature{Arguments: timeArgs, ReturnType: b.Text, NullableBehaviour: querier_dto.FunctionNullableCalledOnNull, IsVariadic: true, MinArguments: 1})
	b.Add("time", &querier_dto.FunctionSignature{Arguments: timeArgs, ReturnType: b.Text, NullableBehaviour: querier_dto.FunctionNullableCalledOnNull, IsVariadic: true, MinArguments: 1})
	b.Add("datetime", &querier_dto.FunctionSignature{Arguments: timeArgs, ReturnType: b.Text, NullableBehaviour: querier_dto.FunctionNullableCalledOnNull, IsVariadic: true, MinArguments: 1})
	b.Add("julianday", &querier_dto.FunctionSignature{Arguments: timeArgs, ReturnType: b.Real, NullableBehaviour: querier_dto.FunctionNullableCalledOnNull, IsVariadic: true, MinArguments: 1})
	b.Add("unixepoch", &querier_dto.FunctionSignature{Arguments: timeArgs, ReturnType: b.Integer, NullableBehaviour: querier_dto.FunctionNullableCalledOnNull, IsVariadic: true, MinArguments: 1})

	b.Add("strftime", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameFormat, Type: b.Text}, Arg{Name: "timestring", Type: b.Text}, Arg{Name: "modifier", Type: b.Text}),
		ReturnType:        b.Text,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      2,
	})

	b.CalledOnNull("timediff", b.Args(Arg{Name: "time1", Type: b.Text}, Arg{Name: "time2", Type: b.Text}), b.Text)
}

// registerStringFunctions registers additional string functions beyond the core set
// (lower, upper, trim, etc. are in registerCoreFunctions).
func (b *FunctionCatalogueBuilder) registerStringFunctions() {
	b.NullOnNull("glob", b.Args(Arg{Name: "pattern", Type: b.Text}, Arg{Name: "string", Type: b.Text}), b.Integer)
	b.NullOnNull("like", b.Args(Arg{Name: "pattern", Type: b.Text}, Arg{Name: "string", Type: b.Text}), b.Integer)
	b.NullOnNull("like", b.Args(Arg{Name: "pattern", Type: b.Text}, Arg{Name: "string", Type: b.Text}, Arg{Name: "escape", Type: b.Text}), b.Integer)
	b.NullOnNull("soundex", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Text)
}

// registerJSONScalarFunctions registers JSON scalar functions.
func (b *FunctionCatalogueBuilder) registerJSONScalarFunctions() {
	b.NullOnNull(paramNameJSON, b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Text)
	b.NullOnNull("json_extract", b.Args(Arg{Name: paramNameJSON, Type: b.Text}, Arg{Name: "path", Type: b.Text}), b.Any)
	b.NullOnNull("json_quote", b.Args(Arg{Name: paramNameValue, Type: b.Any}), b.Text)
	b.NullOnNull("json_patch", b.Args(Arg{Name: "json1", Type: b.Text}, Arg{Name: "json2", Type: b.Text}), b.Text)

	b.NeverNull("json_array", nil, b.Text)
	b.NeverNull("json_object", nil, b.Text)
	b.NeverNull("json_type", b.Args(Arg{Name: paramNameJSON, Type: b.Text}), b.Text)
	b.NeverNull("json_valid", b.Args(Arg{Name: paramNameX, Type: b.Text}), b.Integer)

	mutatorArgs := b.Args(Arg{Name: paramNameJSON, Type: b.Text}, Arg{Name: "path", Type: b.Text}, Arg{Name: paramNameValue, Type: b.Any})
	for _, name := range []string{"json_set", "json_insert", "json_replace"} {
		b.Add(name, &querier_dto.FunctionSignature{
			Arguments:         mutatorArgs,
			ReturnType:        b.Text,
			NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
			IsVariadic:        true,
			MinArguments:      3,
		})
	}

	b.Add("json_remove", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameJSON, Type: b.Text}, Arg{Name: "path", Type: b.Text}),
		ReturnType:        b.Text,
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		IsVariadic:        true,
		MinArguments:      2,
	})
}

// registerJSONAggregateFunctions registers JSON aggregate functions.
func (b *FunctionCatalogueBuilder) registerJSONAggregateFunctions() {
	b.Add("json_group_array", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameValue, Type: b.Any}),
		ReturnType:        b.Text,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})
	b.Add("json_group_object", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: "name", Type: b.Text}, Arg{Name: paramNameValue, Type: b.Any}),
		ReturnType:        b.Text,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})
}

// registerAggregateFunctions registers core aggregate functions.
func (b *FunctionCatalogueBuilder) registerAggregateFunctions() {
	b.Add("count", &querier_dto.FunctionSignature{ReturnType: b.Bigint, IsAggregate: true, NullableBehaviour: querier_dto.FunctionNullableNeverNull})
	b.Add("count", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.Any}),
		ReturnType:        b.Bigint,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})
	b.Add("total", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.Any}),
		ReturnType:        b.Real,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})

	b.Add("avg", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.Any}),
		ReturnType:        b.Real,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("sum", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.Any}),
		ReturnType:        b.Any,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("group_concat", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.Any}),
		ReturnType:        b.Text,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})

	b.Add("max", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.Any}),
		ReturnType:        b.Any,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("min", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.Any}),
		ReturnType:        b.Any,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
	b.Add("group_concat", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameX, Type: b.Any}, Arg{Name: "separator", Type: b.Text}),
		ReturnType:        b.Text,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
}

// registerWindowRankingFunctions registers window ranking functions (row_number, rank,
// dense_rank, etc.).
func (b *FunctionCatalogueBuilder) registerWindowRankingFunctions() {
	b.NeverNull("row_number", nil, b.Bigint)
	b.NeverNull("rank", nil, b.Bigint)
	b.NeverNull("dense_rank", nil, b.Bigint)
	b.NeverNull("ntile", b.Args(Arg{Name: "n", Type: b.Integer}), b.Bigint)
	b.NeverNull("cume_dist", nil, b.Real)
	b.NeverNull("percent_rank", nil, b.Real)
}

// registerWindowValueFunctions registers window value-access functions (lag, lead,
// first_value, last_value, nth_value).
func (b *FunctionCatalogueBuilder) registerWindowValueFunctions() {
	windowValueArgs := b.Args(Arg{Name: paramNameExpression, Type: b.Any}, Arg{Name: "offset", Type: b.Integer}, Arg{Name: "default", Type: b.Any})

	b.Add("lag", &querier_dto.FunctionSignature{Arguments: windowValueArgs, ReturnType: b.Any, NullableBehaviour: querier_dto.FunctionNullableCalledOnNull, MinArguments: 1})
	b.Add("lead", &querier_dto.FunctionSignature{Arguments: windowValueArgs, ReturnType: b.Any, NullableBehaviour: querier_dto.FunctionNullableCalledOnNull, MinArguments: 1})

	b.CalledOnNull("first_value", b.Args(Arg{Name: paramNameExpression, Type: b.Any}), b.Any)
	b.CalledOnNull("last_value", b.Args(Arg{Name: paramNameExpression, Type: b.Any}), b.Any)
	b.CalledOnNull("nth_value", b.Args(Arg{Name: paramNameExpression, Type: b.Any}, Arg{Name: "n", Type: b.Integer}), b.Any)
}

// registerFTS5Functions registers FTS5 full-text search auxiliary functions.
func (b *FunctionCatalogueBuilder) registerFTS5Functions() {
	b.CalledOnNull("highlight", b.Args(
		Arg{Name: paramNameTable, Type: b.Text},
		Arg{Name: "column_index", Type: b.Integer},
		Arg{Name: "open_tag", Type: b.Text},
		Arg{Name: "close_tag", Type: b.Text},
	), b.Text)
	b.CalledOnNull("snippet", b.Args(
		Arg{Name: paramNameTable, Type: b.Text},
		Arg{Name: "column_index", Type: b.Integer},
		Arg{Name: "open_tag", Type: b.Text},
		Arg{Name: "close_tag", Type: b.Text},
		Arg{Name: "ellipsis", Type: b.Text},
		Arg{Name: "max_tokens", Type: b.Integer},
	), b.Text)

	b.Add("bm25", &querier_dto.FunctionSignature{
		Arguments:         b.Args(Arg{Name: paramNameTable, Type: b.Text}, Arg{Name: "weight", Type: b.Real}),
		ReturnType:        b.Real,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsVariadic:        true,
		MinArguments:      1,
	})

	b.CalledOnNull("matchinfo", b.Args(Arg{Name: paramNameTable, Type: b.Text}), b.Blob)
	b.CalledOnNull("matchinfo", b.Args(Arg{Name: paramNameTable, Type: b.Text}, Arg{Name: paramNameFormat, Type: b.Text}), b.Blob)
}

// registerRTreeFunctions registers R-Tree diagnostic functions.
func (b *FunctionCatalogueBuilder) registerRTreeFunctions() {
	b.CalledOnNull("rtreecheck", b.Args(Arg{Name: paramNameTable, Type: b.Text}), b.Text)
	b.CalledOnNull("rtreecheck", b.Args(Arg{Name: "schema", Type: b.Text}, Arg{Name: paramNameTable, Type: b.Text}), b.Text)
	b.CalledOnNull("rtreenode", b.Args(Arg{Name: "pageno", Type: b.Integer}, Arg{Name: "data", Type: b.Blob}), b.Text)
}
