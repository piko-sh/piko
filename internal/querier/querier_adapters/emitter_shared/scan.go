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

package emitter_shared

import (
	"go/ast"
	"go/token"
	"strings"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_dto"
)

// BuildMethodParams constructs the parameter field list for a query method. Always starts
// with ctx context.Context.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the parameters.
// Takes mappings (*querier_dto.TypeMappingTable) for type resolution.
// Takes tracker (*ImportTracker) for import collection.
//
// Returns *ast.FieldList which is the parameter list.
func BuildMethodParams(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
) *ast.FieldList {
	fields := []*ast.Field{
		goastutil.Field(IdentCtx, goastutil.SelectorExpr("context", "Context")),
	}

	switch {
	case len(query.Parameters) == 0:
	case hasInlineableSingleParameter(query):
		goType := singleParameterGoType(query.Parameters[0], mappings)
		typeExpr := tracker.AddType(goType)
		if query.Parameters[0].IsSlice {
			typeExpr = &ast.ArrayType{Elt: typeExpr}
		}
		fields = append(fields, goastutil.Field(
			SnakeToCamelCase(query.Parameters[0].Name),
			typeExpr,
		))
	default:
		fields = append(fields, goastutil.Field(
			IdentParams,
			goastutil.CachedIdent(query.Name+"Params"),
		))
	}

	return goastutil.FieldList(fields...)
}

// singleParameterGoType resolves the Go type for an inlined parameter. A pagination bound
// (LIMIT/OFFSET) is emitted as a plain int, matching the multi-parameter Params-struct
// path (BuildFieldStruct), so a LIMIT argument has the same Go type whether or not the
// query has other parameters (no arity-dependent narrowing at the call site).
//
// Takes parameter (querier_dto.QueryParameter) which is the sole query parameter.
// Takes mappings (*querier_dto.TypeMappingTable) which drives non-pagination resolution.
//
// Returns querier_dto.GoType which is the resolved or pagination-overridden Go type.
func singleParameterGoType(
	parameter querier_dto.QueryParameter,
	mappings *querier_dto.TypeMappingTable,
) querier_dto.GoType {
	if parameter.IsPaginationBound() {
		return querier_dto.GoType{Name: "int"}
	}

	return ResolveGoType(parameter.SQLType, parameter.Nullable, mappings)
}

// hasInlineableSingleParameter reports whether the query's parameters can be a single
// bare argument on the generated method signature rather than via a Params struct.
//
// The single bare argument also drives the corresponding builder entry point. The
// condition is exactly one parameter that is not a slice. The check is centralised so the
// method-signature emitter and the runtime-builder entry composite agree on when to drop
// the struct wrapper.
//
// Takes query (*querier_dto.AnalysedQuery) which is the query under emission.
//
// Returns bool which is true when the single-parameter shortcut applies.
func hasInlineableSingleParameter(query *querier_dto.AnalysedQuery) bool {
	return len(query.Parameters) == 1 && !HasSliceParameter(query)
}

// BuildQueryArgs constructs the argument expressions for a database call.
//
// Engines whose wire protocol retains numbered placeholders (`?N`, `$N`) need one
// argument per unique parameter because the driver reuses bound values when an index
// repeats. For engines whose driver only accepts anonymous `?` placeholders the SQL is
// rewritten with the index stripped, so each occurrence becomes a fresh placeholder that
// requires its own argument and the helper emits the parameter access once per occurrence
// in source order.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the parameters.
// Takes strategy (MethodStrategy) which exposes placeholder behaviour so the helper can
// pick the right occurrence model.
//
// Returns []ast.Expr which contains ctx, sql constant, and parameter expressions.
func BuildQueryArgs(query *querier_dto.AnalysedQuery, strategy MethodStrategy) []ast.Expr {
	arguments := []ast.Expr{
		goastutil.CachedIdent(IdentCtx),
		goastutil.CachedIdent(SnakeToCamelCase(query.Name)),
	}
	if len(query.Parameters) == 0 {
		return arguments
	}
	if len(query.Parameters) == 1 && !HasSliceParameter(query) {
		return appendDirectParamArgs(arguments, query, strategy)
	}
	if strategy != nil && !strategy.PreservesPlaceholderIndices() {
		if ordered, ok := appendPlaceholderOrderArgs(arguments, query, strategy); ok {
			return ordered
		}
	}
	return appendParamsStructArgs(arguments, query, strategy)
}

// wrapParameterAccess returns the strategy-wrapped access expression for a parameter, or
// the bare expression when no strategy is set.
//
// Takes strategy (MethodStrategy) which controls per-parameter wrapping.
// Takes access (ast.Expr) which is the bare access expression.
// Takes paramName (string) which is the parameter's canonical name for the wrapper to
// embed.
//
// Returns ast.Expr which is the (possibly wrapped) access expression.
func wrapParameterAccess(strategy MethodStrategy, access ast.Expr, paramName string) ast.Expr {
	if strategy == nil {
		return access
	}
	return strategy.WrapParameterAccess(access, paramName)
}

// appendDirectParamArgs emits one access expression per occurrence of the query's single,
// non-slice parameter. The method signature in this case accepts the parameter directly
// as a bare identifier.
//
// Takes arguments ([]ast.Expr) which is the accumulating arg list.
// Takes query (*querier_dto.AnalysedQuery) which provides the parameter.
// Takes strategy (MethodStrategy) which drives per-arg wrapping.
//
// Returns []ast.Expr which extends arguments with the access exprs.
func appendDirectParamArgs(arguments []ast.Expr, query *querier_dto.AnalysedQuery, strategy MethodStrategy) []ast.Expr {
	access := goastutil.CachedIdent(SnakeToCamelCase(query.Parameters[0].Name))
	occurrences := parameterOccurrenceCount(query, query.Parameters[0].Number, strategy)
	for range occurrences {
		arguments = append(arguments, wrapParameterAccess(strategy, access, query.Parameters[0].Name))
	}
	return arguments
}

// appendPlaceholderOrderArgs walks the SQL string in source order and emits one access
// expression per placeholder occurrence. Used by engines whose driver collapses `?N`
// placeholders to anonymous `?`, where each placeholder occurrence requires its own
// argument.
//
// Every placeholder number in the analysed SQL is expected to resolve to a declared
// parameter, so parameterIndexByNumber returning -1 signals an analyser invariant break
// that would mismatch the bind count. The occurrence is skipped to keep codegen
// panic-free; the mismatch surfaces as a driver argument-count error at the call site if
// such an invalid SQL ever reaches this path.
//
// Takes arguments ([]ast.Expr) which is the accumulating arg list.
// Takes query (*querier_dto.AnalysedQuery) which provides the SQL and parameters.
// Takes strategy (MethodStrategy) which drives per-arg wrapping.
//
// Returns []ast.Expr which is the extended argument slice when the SQL contained
// recognisable placeholders, or the original slice when no placeholder order could be
// derived.
// Returns bool which is true when placeholders were found, and false when the caller must
// fall back to the struct-field order.
func appendPlaceholderOrderArgs(arguments []ast.Expr, query *querier_dto.AnalysedQuery, strategy MethodStrategy) ([]ast.Expr, bool) {
	ordering := PlaceholderOccurrenceOrder(query.SQL)
	if len(ordering) == 0 {
		return arguments, false
	}
	for _, number := range ordering {
		index := parameterIndexByNumber(query, number)
		if index < 0 {
			continue
		}
		arguments = append(arguments,
			wrapParameterAccess(strategy,
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentParams), SnakeToPascalCase(query.Parameters[index].Name)),
				query.Parameters[index].Name),
		)
	}
	return arguments, true
}

// appendParamsStructArgs emits one access expression per declared parameter in
// declaration order, selecting from the generated `params` struct. Used by engines that
// bind by indexed placeholder (SQLite, pgx) where each parameter has a single binding
// regardless of how many times its placeholder appears in the SQL.
//
// Takes arguments ([]ast.Expr), query, strategy as above.
//
// Returns []ast.Expr extended with one access per parameter.
func appendParamsStructArgs(arguments []ast.Expr, query *querier_dto.AnalysedQuery, strategy MethodStrategy) []ast.Expr {
	for index := range query.Parameters {
		arguments = append(arguments,
			wrapParameterAccess(strategy,
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentParams), SnakeToPascalCase(query.Parameters[index].Name)),
				query.Parameters[index].Name),
		)
	}

	return arguments
}

// parameterIndexByNumber returns the slice index of the parameter with the bind number.
//
// It returns -1 when no parameter carries the requested number.
//
// Takes query (*querier_dto.AnalysedQuery) which provides the parameter list to search.
// Takes number (int) which is the bind number to look up.
//
// Returns int which is the parameter slice index or -1.
func parameterIndexByNumber(query *querier_dto.AnalysedQuery, number int) int {
	for index := range query.Parameters {
		if query.Parameters[index].Number == number {
			return index
		}
	}
	return -1
}

// parameterOccurrenceCount returns the placeholder occurrence count for a bind number.
//
// For engines that preserve placeholder indices the driver dedupes by index so the count
// is always 1. For engines that strip the index each occurrence becomes its own
// placeholder so the helper walks the SQL to compute the count.
//
// Takes query (*querier_dto.AnalysedQuery) which holds the SQL and parameter metadata.
// Takes number (int) which is the bind number to count.
// Takes strategy (MethodStrategy) which exposes placeholder behaviour.
//
// Returns int which is the number of occurrences (minimum 1).
func parameterOccurrenceCount(query *querier_dto.AnalysedQuery, number int, strategy MethodStrategy) int {
	if strategy == nil || strategy.PreservesPlaceholderIndices() {
		return 1
	}
	ordering := PlaceholderOccurrenceOrder(query.SQL)
	if len(ordering) == 0 {
		return 1
	}
	count := 0
	for _, occurrenceNumber := range ordering {
		if occurrenceNumber == number {
			count++
		}
	}
	if count == 0 {
		return 1
	}
	return count
}

// BuildScanArgs constructs &row.Field expressions for rows.Scan calls. When embeds are
// present, embedded columns scan into &row.Embed.Field.
//
// Field names are resolved through scanFieldNames so a column whose Go name would collide
// with an earlier column in the same struct (for example "foo_bar" and "foo__bar" both
// folding to "FooBar") scans into the same disambiguated field the struct emitter
// declared, keeping the scan targets and the row struct in lockstep.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the output columns.
//
// Returns []ast.Expr which contains the address-of field expressions.
func BuildScanArgs(query *querier_dto.AnalysedQuery, strategy MethodStrategy, mappings *querier_dto.TypeMappingTable) []ast.Expr {
	fieldNames := scanFieldNames(query.OutputColumns)
	wrapped := WrappedArrayColumns(query, strategy, mappings)
	scanArguments := make([]ast.Expr, 0, len(query.OutputColumns))
	for index := range query.OutputColumns {
		column := &query.OutputColumns[index]
		var target ast.Expr
		if column.IsEmbedded {
			target = goastutil.AddressExpr(
				goastutil.SelectorExprFrom(
					goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentRow), SnakeToPascalCase(column.EmbedTable)),
					fieldNames[index],
				),
			)
		} else {
			target = goastutil.AddressExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentRow), fieldNames[index]),
			)
		}
		if wrapped[index] {
			target = goastutil.CallExpr(goastutil.SelectorExpr("dbjson", "ScanInto"), target)
		}
		scanArguments = append(scanArguments, target)
	}
	return scanArguments
}

// scanFieldNames resolves the Go field name each output column scans into.
//
// The result is parallel to the supplied column slice. Flat (non-embedded) columns share
// one disambiguation namespace and each embed table shares its own, mirroring exactly how
// the struct emitters build the row struct and the per-embed structs. The shared
// DisambiguateGoFieldNames call over the same ordered names guarantees the scan target
// matches the declared field.
//
// Takes columns ([]querier_dto.OutputColumn) which are the output columns in order.
//
// Returns []string which are the resolved Go field names in the same order.
func scanFieldNames(columns []querier_dto.OutputColumn) []string {
	flatNames := make([]string, 0, len(columns))
	embedNames := make(map[string][]string)
	for index := range columns {
		if columns[index].IsEmbedded {
			embedNames[columns[index].EmbedTable] = append(embedNames[columns[index].EmbedTable], columns[index].Name)
			continue
		}
		flatNames = append(flatNames, columns[index].Name)
	}

	flatResolved := DisambiguateGoFieldNames(flatNames)
	embedResolved := make(map[string][]string, len(embedNames))
	for table, names := range embedNames {
		embedResolved[table] = DisambiguateGoFieldNames(names)
	}

	fieldNames := make([]string, len(columns))
	flatCursor := 0
	embedCursor := make(map[string]int, len(embedNames))
	for index := range columns {
		if columns[index].IsEmbedded {
			table := columns[index].EmbedTable
			fieldNames[index] = embedResolved[table][embedCursor[table]]
			embedCursor[table]++
			continue
		}
		fieldNames[index] = flatResolved[flatCursor]
		flatCursor++
	}
	return fieldNames
}

// BuildEmbedPreAllocStatements generates allocation statements for outer-join embed
// pointers before scanning (e.g. row.User = &GetOrderUser{}).
//
// Takes query (*querier_dto.AnalysedQuery) which defines the output columns.
//
// Returns []ast.Stmt which contains the allocation statements, or nil when there are no
// outer-join embeds.
func BuildEmbedPreAllocStatements(query *querier_dto.AnalysedQuery) []ast.Stmt {
	if !HasEmbeddedColumns(query) {
		return nil
	}

	_, embedGroups := GroupColumnsByEmbed(query.OutputColumns)
	var statements []ast.Stmt

	for _, group := range embedGroups {
		if !group.IsOuter {
			continue
		}
		fieldName := SnakeToPascalCase(group.TableName)
		structName := EmbedStructName(query.Name, group.TableName)
		statements = append(statements,
			goastutil.AssignStmt(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentRow), fieldName),
				goastutil.AddressExpr(goastutil.CompositeLit(goastutil.CachedIdent(structName))),
			),
		)
	}

	return statements
}

// BuildEmbedNilCheckStatements generates nil checks for outer-join embeds after scanning.
//
// When the sentinel column of the embed scanned as nil, the embed pointer is set to nil
// (e.g. if row.User.ID == nil then row.User = nil).
//
// The sentinel must be a column whose resolved Go type is nilable, otherwise the
// generated `field == nil` comparison fails to compile. The analyser normally forces
// every outer-join column nullable (which maps to a pointer/interface/slice type), but an
// explicit user directive (piko.column(col, nullable: false), catalogue or query-level
// NullableOverride=false) can leave a column resolved to a value type after EmbedIsOuter
// was set. The first nilable column in the group is therefore chosen for the comparison.
// A group with no nilable column emits no nil check (the pre-allocated embed pointer
// stays non-nil).
//
// Takes query (*querier_dto.AnalysedQuery) which defines the output columns.
//
// Returns []ast.Stmt which contains the nil-check statements, or nil when there are no
// outer-join embeds.
func BuildEmbedNilCheckStatements(query *querier_dto.AnalysedQuery) []ast.Stmt {
	if !HasEmbeddedColumns(query) {
		return nil
	}

	_, embedGroups := GroupColumnsByEmbed(query.OutputColumns)
	var statements []ast.Stmt

	for _, group := range embedGroups {
		if !group.IsOuter || len(group.Columns) == 0 {
			continue
		}
		sentinelField, found := nilableSentinelField(group.Columns)
		if !found {
			continue
		}
		fieldName := SnakeToPascalCase(group.TableName)
		statements = append(statements,
			&ast.IfStmt{
				Cond: &ast.BinaryExpr{
					X: goastutil.SelectorExprFrom(
						goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentRow), fieldName),
						sentinelField,
					),
					Op: token.EQL,
					Y:  goastutil.CachedIdent(IdentNil),
				},
				Body: goastutil.BlockStmt(
					goastutil.AssignStmt(
						goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentRow), fieldName),
						goastutil.CachedIdent(IdentNil),
					),
				),
			},
		)
	}

	return statements
}

// nilableSentinelField returns the field name of the first nil-comparable group column.
//
// The returned name is the disambiguated Go field name of the first column whose resolved
// Go type can be compared against nil, together with whether such a column exists. The
// field name is resolved through the same per-group disambiguation the embed struct
// emitter uses so the comparison targets the declared field even when two columns fold
// onto the same Go identifier.
//
// Takes columns ([]querier_dto.OutputColumn) which are the group's columns in order.
//
// Returns string which is the disambiguated sentinel field name.
// Returns bool which is true when a nilable column was found.
func nilableSentinelField(columns []querier_dto.OutputColumn) (string, bool) {
	sourceNames := make([]string, len(columns))
	for index := range columns {
		sourceNames[index] = columns[index].Name
	}
	fieldNames := DisambiguateGoFieldNames(sourceNames)
	for index := range columns {
		if columnFieldIsNilable(&columns[index]) {
			return fieldNames[index], true
		}
	}
	return "", false
}

// columnFieldIsNilable reports whether a column's struct field is comparable against nil.
//
// A nullable column always resolves to a nilable Go type (pointer, []byte,
// json.RawMessage, []any, or any), and a non-nullable column with a go_type override that
// names a pointer, slice, map, channel, function, or interface type is likewise
// nil-comparable.
//
// Takes column (*querier_dto.OutputColumn) which is the column to inspect.
//
// Returns bool which is true when the column's field can be compared against nil.
func columnFieldIsNilable(column *querier_dto.OutputColumn) bool {
	if column.Nullable {
		return true
	}
	if column.GoTypeOverride != nil {
		return goTypeNameIsNilable(column.GoTypeOverride.Name)
	}
	return false
}

// goTypeNameIsNilable reports whether a Go type name denotes a nil-comparable type.
//
// Takes name (string) which is the Go type name (for example "*string" or "[]byte").
//
// Returns bool which is true when a value of the type may be compared against nil.
func goTypeNameIsNilable(name string) bool {
	switch {
	case name == "any":
		return true
	case strings.HasPrefix(name, "*"):
		return true
	case strings.HasPrefix(name, "[]"):
		return true
	case strings.HasPrefix(name, "map["):
		return true
	case strings.HasPrefix(name, "chan "):
		return true
	case strings.HasPrefix(name, "func("):
		return true
	case strings.HasPrefix(name, "interface{"):
		return true
	default:
		return false
	}
}

// BuildErrCheck constructs an if err != nil { return ..., err } statement.
//
// Takes zeroValues ([]ast.Expr) which are the zero values to return alongside the error.
//
// Returns *ast.IfStmt which is the error check statement.
func BuildErrCheck(zeroValues ...ast.Expr) *ast.IfStmt {
	returnValues := make([]ast.Expr, 0, len(zeroValues)+1)
	returnValues = append(returnValues, zeroValues...)
	returnValues = append(returnValues, goastutil.CachedIdent(IdentErr))

	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  goastutil.CachedIdent(IdentErr),
			Op: token.NEQ,
			Y:  goastutil.CachedIdent(IdentNil),
		},
		Body: goastutil.BlockStmt(
			goastutil.ReturnStmt(returnValues...),
		),
	}
}

// ConnectionField returns the DBTX field name to use for a query, selecting the reader
// for read-only queries and the writer otherwise.
//
// Takes query (*querier_dto.AnalysedQuery) which is the query to inspect.
//
// Returns string which is the field name ("reader" or "writer").
func ConnectionField(query *querier_dto.AnalysedQuery) string {
	if query.ReadOnly {
		return IdentReader
	}
	return IdentWriter
}
