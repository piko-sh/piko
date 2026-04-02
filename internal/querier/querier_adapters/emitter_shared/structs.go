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

// TypedField describes a named field with a SQL type and nullability for struct
// generation.
type TypedField struct {
	// Name holds the field name in snake_case.
	Name string

	// GoTypeOverride, when non-empty, bypasses type-mapping resolution and uses
	// this Go type directly. Used for limit/offset parameters that must be plain
	// int regardless of the underlying SQL type.
	GoTypeOverride *querier_dto.GoType

	// SQLType holds the SQL type of the field.
	SQLType querier_dto.SQLType

	// Nullable holds whether the field accepts null values.
	Nullable bool

	// IsSlice indicates the parameter expands to multiple values (piko.slice).
	// When true, the generated Go type is wrapped in a slice ([]T).
	IsSlice bool
}

// EmbedGroup describes a set of output columns that belong to one embedded
// table, along with whether the table was introduced via an outer join.
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
	for i := range query.Parameters {
		if query.Parameters[i].IsSlice {
			return true
		}
	}
	return false
}

// HasOutputColumns reports whether a query produces result rows.
//
// Takes query (*querier_dto.AnalysedQuery) which is the query to check.
//
// Returns bool which is true when the query has output columns and uses a
// row-returning command.
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

// BuildSQLConstant constructs a const declaration for the query's SQL text
// with directive comments stripped.
//
// Takes query (*querier_dto.AnalysedQuery) which provides the SQL and name.
//
// Returns ast.Decl which is the const declaration.
func BuildSQLConstant(query *querier_dto.AnalysedQuery) ast.Decl {
	strippedSQL := StripDirectiveComments(query.SQL)
	strippedSQL = RewriteNamedParameters(strippedSQL, query.Parameters)

	if query.IsDynamic {
		excluded := make(map[int]bool)
		hasSortable := false
		for i := range query.Parameters {
			if query.Parameters[i].Kind == querier_dto.ParameterDirectiveSortable {
				excluded[query.Parameters[i].Number] = true
				hasSortable = true
			}
		}
		if hasSortable {
			strippedSQL = StripOrderByClause(strippedSQL)
			strippedSQL = RenumberParametersExcluding(strippedSQL, excluded)
		}
	}

	return &ast.GenDecl{
		Tok: token.CONST,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names:  []*ast.Ident{goastutil.CachedIdent(SnakeToCamelCase(query.Name))},
				Values: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: "`" + strippedSQL + "`"}},
			},
		},
	}
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

		if parameters[index].Kind == querier_dto.ParameterDirectiveLimit ||
			parameters[index].Kind == querier_dto.ParameterDirectiveOffset {
			field.GoTypeOverride = &querier_dto.GoType{Name: "int"}
		}

		fields[index] = field
	}
	return BuildStructDecl(structName, fields, mappings, tracker)
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
			Name:     columns[index].Name,
			SQLType:  columns[index].SQLType,
			Nullable: columns[index].Nullable,
		}
	}
	return BuildStructDecl(structName, fields, mappings, tracker)
}

// HasEmbeddedColumns reports whether any output column is part of an embed.
//
// Takes query (*querier_dto.AnalysedQuery) which is the query to check.
//
// Returns bool which is true when at least one output column is embedded.
func HasEmbeddedColumns(query *querier_dto.AnalysedQuery) bool {
	for i := range query.OutputColumns {
		if query.OutputColumns[i].IsEmbedded {
			return true
		}
	}
	return false
}

// GroupColumnsByEmbed separates output columns into flat (non-embedded) columns
// and embed groups, preserving order.
//
// Takes columns ([]querier_dto.OutputColumn) which are the columns to separate.
//
// Returns []querier_dto.OutputColumn which contains the non-embedded columns.
// Returns []EmbedGroup which contains the grouped embedded columns.
func GroupColumnsByEmbed(columns []querier_dto.OutputColumn) ([]querier_dto.OutputColumn, []EmbedGroup) {
	var flatColumns []querier_dto.OutputColumn
	groupMap := make(map[string]*EmbedGroup)
	var groupOrder []string

	for i := range columns {
		if !columns[i].IsEmbedded {
			flatColumns = append(flatColumns, columns[i])
			continue
		}
		group, exists := groupMap[columns[i].EmbedTable]
		if !exists {
			group = &EmbedGroup{
				TableName: columns[i].EmbedTable,
				IsOuter:   columns[i].EmbedIsOuter,
			}
			groupMap[columns[i].EmbedTable] = group
			groupOrder = append(groupOrder, columns[i].EmbedTable)
		}
		group.Columns = append(group.Columns, columns[i])
	}

	groups := make([]EmbedGroup, len(groupOrder))
	for i, tableName := range groupOrder {
		groups[i] = *groupMap[tableName]
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

// GroupByKeyTable extracts the table name from the first group_by key (e.g.,
// "orders.id" -> "orders").
//
// Takes query (*querier_dto.AnalysedQuery) which is the query to inspect.
//
// Returns string which is the key table name, or empty string if no group_by
// is set.
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

// IsGroupByDetailEmbed reports whether a given embed group is a "detail" group
// in a group_by query (i.e., not the key table). Detail embeds become slices
// in the row struct.
//
// Takes group (EmbedGroup) which is the embed group to check.
// Takes keyTable (string) which is the group_by key table name.
//
// Returns bool which is true when the group is a detail embed.
func IsGroupByDetailEmbed(group EmbedGroup, keyTable string) bool {
	return keyTable != "" && !strings.EqualFold(group.TableName, keyTable)
}

// BuildOutputStructs generates the row struct and any per-embed structs for a
// query.
//
// When no embeds are present, produces a single row struct. When embeds exist,
// produces per-embed structs followed by a main row struct containing nested
// embed fields. For group_by queries, non-key embed fields become slices.
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

// BuildEmbedRowStruct constructs the main row struct with flat fields and
// nested embed fields.
//
// Inner-join embeds are value types; outer-join embeds are pointer types. In
// group_by queries, non-key embeds become slice fields.
//
// Takes structName (string) which is the name for the generated struct.
// Takes flatColumns ([]querier_dto.OutputColumn) which are the non-embedded
// columns.
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

	for i := range flatColumns {
		goType := ResolveGoType(flatColumns[i].SQLType, flatColumns[i].Nullable, mappings)
		typeExpression := tracker.AddType(goType)
		astFields = append(astFields, &ast.Field{
			Names: []*ast.Ident{goastutil.CachedIdent(SnakeToPascalCase(flatColumns[i].Name))},
			Type:  typeExpression,
		})
	}

	for i := range embedGroups {
		fieldName := SnakeToPascalCase(embedGroups[i].TableName)
		embedType := goastutil.CachedIdent(EmbedStructName(queryName, embedGroups[i].TableName))
		var fieldType ast.Expr
		if IsGroupByDetailEmbed(embedGroups[i], keyTable) {
			fieldType = &ast.ArrayType{Elt: embedType}
		} else if embedGroups[i].IsOuter {
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
			Names: []*ast.Ident{goastutil.CachedIdent(SnakeToPascalCase(fields[index].Name))},
			Type:  typeExpression,
		}
		astFields = append(astFields, astField)
	}

	return goastutil.GenDeclType(structName, goastutil.StructType(astFields...))
}
