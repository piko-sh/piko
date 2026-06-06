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
	"strconv"
	"strings"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// backtickQuote is the Go raw-string-literal delimiter used when emitting SQL constants.
	// Kept as a named constant so the literal does not appear inline as `"\\`"` in multiple
	// places.
	backtickQuote = "`"

	// carriageReturn is the CR control character. A Go raw-string literal silently drops any
	// carriage return it contains (per the Go spec), so its presence forces the emitter onto
	// the strconv.Quote path that preserves it.
	carriageReturn = "\r"
)

// TypedField describes a named field with a SQL type and nullability for struct
// generation.
type TypedField struct {
	// Name holds the field name in snake_case.
	Name string

	// GoTypeOverride, when non-empty, bypasses type-mapping resolution and uses this Go type
	// directly. Used for limit/offset parameters that must be plain int regardless of the
	// underlying SQL type.
	GoTypeOverride *querier_dto.GoType

	// SQLType holds the SQL type of the field.
	SQLType querier_dto.SQLType

	// Nullable holds whether the field accepts null values.
	Nullable bool

	// IsSlice indicates the parameter expands to multiple values (piko.slice). When true,
	// the generated Go type is wrapped in a slice ([]T).
	IsSlice bool

	// EmitJSONTag, when true, attaches a `json:"<Name>"` struct tag to the generated field.
	//
	// Used for output-column row structs so they remain structurally identical to the
	// table-level model structs (which also carry json tags), enabling Go's native struct
	// conversion at DAL boundaries when a SELECT * row needs to flow into a model-typed
	// helper.
	EmitJSONTag bool
}

// EmbedGroup describes a set of output columns that belong to one embedded table, along
// with whether the table was introduced via an outer join.
type EmbedGroup struct {
	// TableName holds the name of the embedded table.
	TableName string

	// Columns holds the output columns belonging to this embed group.
	Columns []querier_dto.OutputColumn

	// IsOuter holds whether the table was joined via an outer join.
	IsOuter bool
}

// GroupQueriesByFilename groups queries by their source SQL filename.
//
// Takes queries ([]*querier_dto.AnalysedQuery) which are the queries to group.
//
// Returns map[string][]*querier_dto.AnalysedQuery which maps filename to queries.
func GroupQueriesByFilename(queries []*querier_dto.AnalysedQuery) map[string][]*querier_dto.AnalysedQuery {
	grouped := make(map[string][]*querier_dto.AnalysedQuery)
	for _, query := range queries {
		grouped[query.Filename] = append(grouped[query.Filename], query)
	}
	return grouped
}

// HasParams reports whether a query has any parameters.
//
// Takes query (*querier_dto.AnalysedQuery) which is the query to check.
//
// Returns bool which is true when the query has at least one parameter.
func HasParams(query *querier_dto.AnalysedQuery) bool {
	return len(query.Parameters) > 0
}

// HasSliceParameter reports whether any parameter in the query uses piko.slice.
//
// Takes query (*querier_dto.AnalysedQuery) which is the query to check.
//
// Returns bool which is true when at least one parameter is a slice.
func HasSliceParameter(query *querier_dto.AnalysedQuery) bool {
	for index := range query.Parameters {
		if query.Parameters[index].IsSlice {
			return true
		}
	}
	return false
}

// HasOutputColumns reports whether a query produces result rows.
//
// Takes query (*querier_dto.AnalysedQuery) which is the query to check.
//
// Returns bool which is true when the query has output columns and uses a row-returning
// command.
func HasOutputColumns(query *querier_dto.AnalysedQuery) bool {
	if len(query.OutputColumns) == 0 {
		return false
	}
	switch query.Command {
	case querier_dto.QueryCommandOne, querier_dto.QueryCommandMany, querier_dto.QueryCommandStream, querier_dto.QueryCommandBatch:
		return true
	default:
		return false
	}
}

// BuildSQLConstant constructs a const declaration for the query's SQL text with directive
// comments stripped.
//
// Engines whose driver retains numbered placeholders (`?N`, `$N`) keep the indices in the
// emitted SQL so the driver can dedupe repeated references. Engines whose driver only
// accepts anonymous `?` cannot reuse a number, so the SQL is rewritten with each indexed
// placeholder collapsed to a bare `?`; the matching argument list is expanded by
// BuildQueryArgs to preserve one bind value per occurrence.
//
// Takes query (*querier_dto.AnalysedQuery) which provides the SQL and name.
// Takes strategy (MethodStrategy) which exposes placeholder behaviour, or nil when the
// caller has no strategy (the SQL is left unmodified).
//
// Returns ast.Decl which is the const declaration.
func BuildSQLConstant(query *querier_dto.AnalysedQuery, strategy MethodStrategy, mappings *querier_dto.TypeMappingTable) ast.Decl {
	sourceSQL := query.SQL
	if strategy != nil {
		if wrapFunc := strategy.ArrayJSONWrapFunc(); wrapFunc != "" {
			sourceSQL, _ = WrapArrayColumnsAsJSON(sourceSQL, query.OutputColumns, wrapFunc, strategy.QuoteIdentifier, mappings)
		}
	}

	strippedSQL := StripDirectiveComments(sourceSQL)
	strippedSQL = RewriteNamedParameters(strippedSQL, query.Parameters)

	if query.IsDynamic {
		excluded := make(map[int]bool)
		hasSortable := false
		for index := range query.Parameters {
			if query.Parameters[index].Kind == querier_dto.ParameterDirectiveSortable {
				excluded[query.Parameters[index].Number] = true
				hasSortable = true
			}
		}
		if hasSortable {
			strippedSQL = StripOrderByClause(strippedSQL)
			strippedSQL = RenumberParametersExcluding(strippedSQL, excluded)
		}
	}

	strippedSQL = rewriteBracedPlaceholders(strippedSQL, query, strategy)

	if strategy != nil && !strategy.PreservesPlaceholderIndices() && !HasSliceParameter(query) {
		strippedSQL = stripIndexedQuestionMarkPlaceholders(strippedSQL)
	}

	return &ast.GenDecl{
		Tok: token.CONST,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names:  []*ast.Ident{goastutil.CachedIdent(SnakeToCamelCase(query.Name))},
				Values: []ast.Expr{sqlStringLiteral(strippedSQL)},
			},
		},
	}
}

// rewriteBracedPlaceholders applies the ClickHouse `{name:Type}` placeholder rewrite for
// strategies that render braced named placeholders, returning the SQL unchanged
// otherwise.
//
// A dynamic-runtime query is rewritten to positional `$N` so its builder can append
// further predicates; a static query keeps named placeholders but prefixes each name so a
// parameter named after a reserved keyword cannot reach the server's SQL parser verbatim.
//
// Takes sql (string) which is the SQL to rewrite.
// Takes query (*querier_dto.AnalysedQuery) which provides the parameters and runtime
// mode.
// Takes strategy (MethodStrategy) which reports whether braced placeholders are used.
//
// Returns string which is the rewritten SQL.
func rewriteBracedPlaceholders(sql string, query *querier_dto.AnalysedQuery, strategy MethodStrategy) string {
	if strategy == nil || !strategy.UsesBracedNamedPlaceholders() {
		return sql
	}
	if query.DynamicRuntime {
		return RewriteBracedNamedToPositional(sql, query.Parameters)
	}
	return PrefixBracedNamedParameters(sql, query.Parameters)
}

// stripIndexedQuestionMarkPlaceholders rewrites every `?N` placeholder in SQL to a bare
// `?`.
//
// SQL string literals and line or block comments are skipped so embedded `?N` tokens
// inside a quoted value or comment remain verbatim. BuildSQLConstant uses this for
// MySQL/MariaDB emitters where the driver only accepts anonymous placeholders.
//
// Takes sql (string) which is the SQL text to rewrite.
//
// Returns string which is the SQL with `?N` collapsed to `?`.
func stripIndexedQuestionMarkPlaceholders(sql string) string {
	var builder strings.Builder
	builder.Grow(len(sql))
	position := 0
	for position < len(sql) {
		if end, ok := copySQLNoise(sql, position, &builder); ok {
			position = end
			continue
		}
		if end, ok := collapseIndexedPlaceholder(sql, position, &builder); ok {
			position = end
			continue
		}
		builder.WriteByte(sql[position])
		position++
	}
	return builder.String()
}

// collapseIndexedPlaceholder detects a `?N` token at position and writes a bare `?` to
// builder, returning the index just past the trailing digits.
// Returns (position, false) when the cursor is not on an indexed placeholder so the
// caller can fall through to its default copy.
//
// Takes sql (string) which is the SQL being scanned.
// Takes position (int) which is the current cursor.
// Takes builder (*strings.Builder) which receives the bare `?`.
//
// Returns nextPosition (int) which is the index past the placeholder.
// Returns consumed (bool) which is true when an indexed placeholder was rewritten.
func collapseIndexedPlaceholder(sql string, position int, builder *strings.Builder) (nextPosition int, consumed bool) {
	if sql[position] != '?' || position+1 >= len(sql) || sql[position+1] < '1' || sql[position+1] > '9' {
		return position, false
	}
	builder.WriteByte('?')
	position++
	for position < len(sql) && sql[position] >= '0' && sql[position] <= '9' {
		position++
	}
	return position, true
}

// CountSQLConstName returns the name of the package-level constant that holds the
// pre-derived `SELECT COUNT(*) ...` form of the runtime-builder query. Centralised so the
// const emission and the builder's Count() method agree on the identifier.
//
// Takes query (*querier_dto.AnalysedQuery) which provides the query name.
//
// Returns string which is the Go identifier for the count-SQL constant.
func CountSQLConstName(query *querier_dto.AnalysedQuery) string {
	return SnakeToCamelCase(query.Name) + "CountSQL"
}

// BaseSQLConstantName returns the Go identifier of the package-level constant that
// BuildSQLConstant emits for a query. Centralised so external callers (notably the OTel
// attribution map) can reference the same identifier without duplicating the
// SnakeToCamelCase convention.
//
// Takes queryName (string) which is the analysed query's logical name.
//
// Returns string which is the Go identifier for the SQL constant.
func BaseSQLConstantName(queryName string) string {
	return SnakeToCamelCase(queryName)
}

// BuildCountSQLConstant constructs the `<query>CountSQL` const declaration used by the
// runtime builder's .Count(ctx) terminal. Emitted only for piko.dynamic: runtime queries
// that have a non-empty CountSQL on the analysed query.
//
// Takes query (*querier_dto.AnalysedQuery) which carries the pre-derived count SQL.
//
// Returns ast.Decl which is the const declaration.
func BuildCountSQLConstant(query *querier_dto.AnalysedQuery) ast.Decl {
	return &ast.GenDecl{
		Tok: token.CONST,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names:  []*ast.Ident{goastutil.CachedIdent(CountSQLConstName(query))},
				Values: []ast.Expr{sqlStringLiteral(query.CountSQL)},
			},
		},
	}
}

// sqlStringLiteral builds the Go string-literal node for an embedded SQL constant.
//
// SQL is normally emitted as a raw-string (backtick-delimited) literal so the quoting
// stays readable, but a Go raw string cannot itself contain a backtick and silently drops
// any carriage return it contains. When the SQL contains a backtick or a carriage return
// the literal is emitted as a strconv.Quote-d interpreted string instead, so the
// generated constant is always valid Go and preserves the SQL text verbatim rather than a
// broken or CR-stripped raw literal.
//
// Takes sql (string) which is the SQL text to embed.
//
// Returns *ast.BasicLit which is the string-literal node.
func sqlStringLiteral(sql string) *ast.BasicLit {
	if strings.Contains(sql, backtickQuote) || strings.Contains(sql, carriageReturn) {
		return &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(sql)}
	}
	return &ast.BasicLit{Kind: token.STRING, Value: backtickQuote + sql + backtickQuote}
}

// BuildFieldStruct constructs a struct type declaration from query parameters.
//
// Takes structName (string) which is the name for the generated struct.
// Takes parameters ([]querier_dto.QueryParameter) which define the fields.
// Takes mappings (*querier_dto.TypeMappingTable) for type resolution.
// Takes tracker (*ImportTracker) for import collection.
//
// Returns ast.Decl which is the type declaration.
func BuildFieldStruct(
	structName string,
	parameters []querier_dto.QueryParameter,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
) ast.Decl {
	fields := make([]TypedField, len(parameters))
	for index := range parameters {
		field := TypedField{
			Name:     parameters[index].Name,
			SQLType:  parameters[index].SQLType,
			Nullable: parameters[index].Nullable,
			IsSlice:  parameters[index].IsSlice,
		}

		if parameters[index].IsPaginationBound() {
			field.GoTypeOverride = &querier_dto.GoType{Name: "int"}
		}

		fields[index] = field
	}
	return BuildStructDecl(structName, fields, mappings, tracker)
}

// requiresRowType reports whether the query command expects a Row type to scan into,
// regardless of whether output columns were successfully resolved.
//
// Used to decide whether to emit a stub Row struct when the analyser could not resolve
// any output columns. The stub keeps the generated code compileable so downstream DAL
// packages can stub the affected methods rather than failing the whole hexagon build.
//
// Takes query (*querier_dto.AnalysedQuery) which provides the query command.
//
// Returns bool which is true when the command needs a Row type.
func requiresRowType(query *querier_dto.AnalysedQuery) bool {
	switch query.Command {
	case querier_dto.QueryCommandOne, querier_dto.QueryCommandMany, querier_dto.QueryCommandStream, querier_dto.QueryCommandBatch:
		return true
	default:
		return false
	}
}

// buildEmptyRowStruct emits a no-field Row stub.
//
// Used when the analyser could not resolve any output columns but the query command still
// requires a Row type for the generated Scan call site.
//
// Takes structName (string) which is the name for the generated stub struct.
//
// Returns ast.Decl which is the empty struct type declaration.
func buildEmptyRowStruct(structName string) ast.Decl {
	return goastutil.GenDeclType(structName, goastutil.StructType())
}

// BuildColumnStruct constructs a struct type declaration from output columns.
//
// Takes structName (string) which is the name for the generated struct.
// Takes columns ([]querier_dto.OutputColumn) which define the fields.
// Takes mappings (*querier_dto.TypeMappingTable) for type resolution.
// Takes tracker (*ImportTracker) for import collection.
//
// Returns ast.Decl which is the type declaration.
func BuildColumnStruct(
	structName string,
	columns []querier_dto.OutputColumn,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
) ast.Decl {
	fields := make([]TypedField, len(columns))
	for index := range columns {
		fields[index] = TypedField{
			Name:           columns[index].Name,
			SQLType:        columns[index].SQLType,
			Nullable:       columns[index].Nullable,
			GoTypeOverride: applyNullableGoTypeOverride(columns[index].GoTypeOverride, columns[index].Nullable),
			EmitJSONTag:    true,
		}
	}
	return BuildStructDecl(structName, fields, mappings, tracker)
}

// applyNullableGoTypeOverride returns the GoType override, wrapping the name in a pointer
// when the column is nullable.
//
// A nullable column declared with `go_type: github.com/google/uuid.UUID` becomes
// `*uuid.UUID`.
//
// Takes override (*querier_dto.GoType) which is the declared override, or nil when none
// was declared.
// Takes nullable (bool) which is true when the column permits NULL.
//
// Returns *querier_dto.GoType which is the resolved override, or nil when no override was
// declared.
func applyNullableGoTypeOverride(override *querier_dto.GoType, nullable bool) *querier_dto.GoType {
	if override == nil {
		return nil
	}
	if !nullable {
		return override
	}
	if strings.HasPrefix(override.Name, "*") {
		return override
	}
	return &querier_dto.GoType{
		Package: override.Package,
		Name:    "*" + override.Name,
	}
}

// HasEmbeddedColumns reports whether any output column is part of an embed.
//
// Takes query (*querier_dto.AnalysedQuery) which is the query to check.
//
// Returns bool which is true when at least one output column is embedded.
func HasEmbeddedColumns(query *querier_dto.AnalysedQuery) bool {
	for index := range query.OutputColumns {
		if query.OutputColumns[index].IsEmbedded {
			return true
		}
	}
	return false
}

// GroupColumnsByEmbed separates output columns into flat (non-embedded) columns and embed
// groups, preserving order.
//
// Takes columns ([]querier_dto.OutputColumn) which are the columns to separate.
//
// Returns []querier_dto.OutputColumn which contains the non-embedded columns.
// Returns []EmbedGroup which contains the grouped embedded columns.
func GroupColumnsByEmbed(columns []querier_dto.OutputColumn) ([]querier_dto.OutputColumn, []EmbedGroup) {
	var flatColumns []querier_dto.OutputColumn
	groupMap := make(map[string]*EmbedGroup)
	var groupOrder []string

	for index := range columns {
		if !columns[index].IsEmbedded {
			flatColumns = append(flatColumns, columns[index])
			continue
		}
		group, exists := groupMap[columns[index].EmbedTable]
		if !exists {
			group = &EmbedGroup{
				TableName: columns[index].EmbedTable,
				IsOuter:   columns[index].EmbedIsOuter,
			}
			groupMap[columns[index].EmbedTable] = group
			groupOrder = append(groupOrder, columns[index].EmbedTable)
		}
		group.Columns = append(group.Columns, columns[index])
	}

	groups := make([]EmbedGroup, len(groupOrder))
	for index, tableName := range groupOrder {
		groups[index] = *groupMap[tableName]
	}

	return flatColumns, groups
}

// EmbedStructName returns the name for an embed struct: "{QueryName}{TablePascal}".
//
// Takes queryName (string) which is the query name prefix.
// Takes tableName (string) which is the table name to convert to PascalCase.
//
// Returns string which is the combined struct name.
func EmbedStructName(queryName, tableName string) string {
	return queryName + SnakeToPascalCase(tableName)
}

// GroupByKeyTable extracts the table name from the first group_by key (e.g., "orders.id"
// -> "orders").
//
// Takes query (*querier_dto.AnalysedQuery) which is the query to inspect.
//
// Returns string which is the key table name, or empty string if no group_by is set.
func GroupByKeyTable(query *querier_dto.AnalysedQuery) string {
	if len(query.GroupByKey) == 0 {
		return ""
	}
	parts := strings.SplitN(query.GroupByKey[0], ".", 2)
	if len(parts) == 2 {
		return parts[0]
	}
	return ""
}

// IsGroupByDetailEmbed reports whether a given embed group is a "detail" group in a
// group_by query (i.e., not the key table). Detail embeds become slices in the row
// struct.
//
// Takes group (EmbedGroup) which is the embed group to check.
// Takes keyTable (string) which is the group_by key table name.
//
// Returns bool which is true when the group is a detail embed.
func IsGroupByDetailEmbed(group EmbedGroup, keyTable string) bool {
	return keyTable != "" && !strings.EqualFold(group.TableName, keyTable)
}

// BuildOutputStructs generates the row struct and any per-embed structs for a query.
//
// When no embeds are present, produces a single row struct. When embeds exist, produces
// per-embed structs followed by a main row struct containing nested embed fields. For
// group_by queries, non-key embed fields become slices.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the output columns.
// Takes mappings (*querier_dto.TypeMappingTable) for type resolution.
// Takes tracker (*ImportTracker) for import collection.
//
// Returns []ast.Decl which contains the struct declarations.
func BuildOutputStructs(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
) []ast.Decl {
	if !HasEmbeddedColumns(query) {
		return []ast.Decl{BuildColumnStruct(query.Name+"Row", query.OutputColumns, mappings, tracker)}
	}

	flatColumns, embedGroups := GroupColumnsByEmbed(query.OutputColumns)
	var declarations []ast.Decl

	for _, group := range embedGroups {
		structName := EmbedStructName(query.Name, group.TableName)
		declarations = append(declarations, BuildColumnStruct(structName, group.Columns, mappings, tracker))
	}

	keyTable := GroupByKeyTable(query)
	declarations = append(declarations, BuildEmbedRowStruct(query.Name+"Row", flatColumns, embedGroups, query.Name, keyTable, mappings, tracker))

	return declarations
}

// BuildEmbedRowStruct constructs the main row struct with flat fields and nested embed
// fields.
//
// Inner-join embeds are value types; outer-join embeds are pointer types. In group_by
// queries, non-key embeds become slice fields.
//
// Takes structName (string) which is the name for the generated struct.
// Takes flatColumns ([]querier_dto.OutputColumn) which are the non-embedded columns.
// Takes embedGroups ([]EmbedGroup) which are the grouped embedded columns.
// Takes queryName (string) which is the query name used for embed struct names.
// Takes keyTable (string) which is the group_by key table name.
// Takes mappings (*querier_dto.TypeMappingTable) for type resolution.
// Takes tracker (*ImportTracker) for import collection.
//
// Returns ast.Decl which is the struct type declaration.
func BuildEmbedRowStruct(
	structName string,
	flatColumns []querier_dto.OutputColumn,
	embedGroups []EmbedGroup,
	queryName string,
	keyTable string,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
) ast.Decl {
	astFields := make([]*ast.Field, 0, len(flatColumns)+len(embedGroups))

	flatSourceNames := make([]string, len(flatColumns))
	for index := range flatColumns {
		flatSourceNames[index] = flatColumns[index].Name
	}
	flatFieldNames := DisambiguateGoFieldNames(flatSourceNames)

	for index := range flatColumns {
		goType := ResolveGoType(flatColumns[index].SQLType, flatColumns[index].Nullable, mappings)
		typeExpression := tracker.AddType(goType)
		astFields = append(astFields, &ast.Field{
			Names: []*ast.Ident{goastutil.CachedIdent(flatFieldNames[index])},
			Type:  typeExpression,
			Tag:   jsonStructTag(flatColumns[index].Name),
		})
	}

	for index := range embedGroups {
		fieldName := SnakeToPascalCase(embedGroups[index].TableName)
		embedType := goastutil.CachedIdent(EmbedStructName(queryName, embedGroups[index].TableName))
		var fieldType ast.Expr
		if IsGroupByDetailEmbed(embedGroups[index], keyTable) {
			fieldType = &ast.ArrayType{Elt: embedType}
		} else if embedGroups[index].IsOuter {
			fieldType = goastutil.StarExpr(embedType)
		} else {
			fieldType = embedType
		}
		astFields = append(astFields, &ast.Field{
			Names: []*ast.Ident{goastutil.CachedIdent(fieldName)},
			Type:  fieldType,
		})
	}

	return goastutil.GenDeclType(structName, goastutil.StructType(astFields...))
}

// BuildStructDecl constructs a type declaration from typed fields.
//
// Takes structName (string) which is the name for the generated struct.
// Takes fields ([]TypedField) which define the struct fields.
// Takes mappings (*querier_dto.TypeMappingTable) for type resolution.
// Takes tracker (*ImportTracker) for import collection.
//
// Returns ast.Decl which is the type declaration.
func BuildStructDecl(
	structName string,
	fields []TypedField,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
) ast.Decl {
	astFields := make([]*ast.Field, 0, len(fields))

	sourceNames := make([]string, len(fields))
	for index := range fields {
		sourceNames[index] = fields[index].Name
	}
	fieldNames := DisambiguateGoFieldNames(sourceNames)

	for index := range fields {
		var goType querier_dto.GoType
		if fields[index].GoTypeOverride != nil {
			goType = *fields[index].GoTypeOverride
		} else {
			goType = ResolveGoType(fields[index].SQLType, fields[index].Nullable, mappings)
		}
		typeExpression := tracker.AddType(goType)
		if fields[index].IsSlice {
			typeExpression = &ast.ArrayType{Elt: typeExpression}
		}

		astField := &ast.Field{
			Names: []*ast.Ident{goastutil.CachedIdent(fieldNames[index])},
			Type:  typeExpression,
		}
		if fields[index].EmitJSONTag {
			astField.Tag = jsonStructTag(fields[index].Name)
		}
		astFields = append(astFields, astField)
	}

	return goastutil.GenDeclType(structName, goastutil.StructType(astFields...))
}

// jsonStructTag constructs a `json:"<name>"` struct tag as an *ast.BasicLit. Centralised
// so the BasicLit construction (raw backticks around the tag, embedded double-quotes
// around the field name) lives in one place; without the helper the inline form triggers
// gocritic's tag-format lint and forces a nolint directive on every call site.
//
// Takes fieldName (string) which is the SQL column or parameter name.
//
// Returns *ast.BasicLit holding the formatted tag.
func jsonStructTag(fieldName string) *ast.BasicLit {
	return &ast.BasicLit{
		Kind:  token.STRING,
		Value: backtickQuote + `json:"` + fieldName + `"` + backtickQuote,
	}
}
