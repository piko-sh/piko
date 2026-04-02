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

package service_test

import (
	"encoding/json"
	"fmt"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

type mockEngineConfig struct {
	BuiltinFunctions map[string][]mockFunctionSignature `json:"builtinFunctions"`
	BuiltinTypes     map[string]mockSQLType             `json:"builtinTypes"`
	Statements       []mockStatement                    `json:"statements"`
}

type mockFunctionSignature struct {
	Arguments         []mockFunctionArgument `json:"arguments"`
	ReturnType        mockSQLType            `json:"returnType"`
	IsAggregate       bool                   `json:"isAggregate"`
	ReturnsSet        bool                   `json:"returnsSet"`
	NullableBehaviour string                 `json:"nullableBehaviour"`
	Schema            string                 `json:"schema"`
}

type mockFunctionArgument struct {
	Name string      `json:"name"`
	Type mockSQLType `json:"type"`
}

type mockSQLType struct {
	Category   string       `json:"category"`
	EngineName string       `json:"engineName"`
	Schema     string       `json:"schema"`
	Precision  *int         `json:"precision,omitempty"`
	Scale      *int         `json:"scale,omitempty"`
	Length     *int         `json:"length,omitempty"`
	EnumValues []string     `json:"enumValues,omitempty"`
	Element    *mockSQLType `json:"element,omitempty"`
}

type mockStatement struct {
	SQL         string           `json:"sql"`
	Mutation    *json.RawMessage `json:"mutation,omitempty"`
	RawAnalysis *json.RawMessage `json:"rawAnalysis,omitempty"`
}

type mockMutation struct {
	Kind              string                 `json:"kind"`
	SchemaName        string                 `json:"schemaName"`
	TableName         string                 `json:"tableName"`
	Columns           []mockColumn           `json:"columns,omitempty"`
	ColumnName        string                 `json:"columnName,omitempty"`
	EnumName          string                 `json:"enumName,omitempty"`
	EnumValues        []string               `json:"enumValues,omitempty"`
	FunctionSignature *mockFunctionSignature `json:"functionSignature,omitempty"`
	NewName           string                 `json:"newName,omitempty"`
	PrimaryKey        []string               `json:"primaryKey,omitempty"`
}

type mockColumn struct {
	Name            string      `json:"name"`
	SQLType         mockSQLType `json:"sqlType"`
	Nullable        bool        `json:"nullable"`
	HasDefault      bool        `json:"hasDefault"`
	IsGenerated     bool        `json:"isGenerated"`
	IsArray         bool        `json:"isArray"`
	ArrayDimensions int         `json:"arrayDimensions"`
	Comment         string      `json:"comment"`
}

type mockRawAnalysis struct {
	OutputColumns       []mockRawOutputColumn       `json:"outputColumns"`
	ParameterReferences []mockRawParameterReference `json:"parameterReferences"`
	FromTables          []mockTableReference        `json:"fromTables"`
	JoinClauses         []mockJoinClause            `json:"joinClauses"`
	CTEDefinitions      []mockCTEDefinition         `json:"cteDefinitions"`
	DerivedTables       []mockDerivedTable          `json:"derivedTables"`
	HasReturning        bool                        `json:"hasReturning"`
	GroupByColumns      []mockColumnReference       `json:"groupByColumns"`
}

type mockDerivedTable struct {
	Alias    string             `json:"alias"`
	Columns  []mockScopedColumn `json:"columns"`
	Source   string             `json:"source"`
	JoinKind string             `json:"joinKind"`
}

type mockScopedColumn struct {
	Name     string      `json:"name"`
	SQLType  mockSQLType `json:"sqlType"`
	Nullable bool        `json:"nullable"`
}

type mockRawOutputColumn struct {
	Name       string           `json:"name"`
	TableAlias string           `json:"tableAlias"`
	ColumnName string           `json:"columnName"`
	IsStar     bool             `json:"isStar"`
	Expression *json.RawMessage `json:"expression,omitempty"`
}

type mockRawParameterReference struct {
	Number          int                  `json:"number"`
	Context         string               `json:"context"`
	ColumnReference *mockColumnReference `json:"columnReference,omitempty"`
	CastType        *mockSQLType         `json:"castType,omitempty"`
}

type mockColumnReference struct {
	TableAlias string `json:"tableAlias"`
	ColumnName string `json:"columnName"`
}

type mockTableReference struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
	Alias  string `json:"alias"`
}

type mockJoinClause struct {
	Kind  string             `json:"kind"`
	Table mockTableReference `json:"table"`
}

type mockCTEDefinition struct {
	Name          string                `json:"name"`
	OutputColumns []mockRawOutputColumn `json:"outputColumns"`
	FromTables    []mockTableReference  `json:"fromTables"`
	IsRecursive   bool                  `json:"isRecursive"`
}

type mockStatementIndex int

func (mockStatementIndex) IsParsedStatement() {}

type mockEngine struct {
	config            mockEngineConfig
	recordedCatalogue *querier_dto.Catalogue
}

func newMockEngine(config mockEngineConfig) *mockEngine {
	return &mockEngine{config: config}
}

func (engine *mockEngine) ParseStatements(sql string) ([]querier_dto.ParsedStatement, error) {
	normalised := normaliseSQLForMatching(sql)
	var results []querier_dto.ParsedStatement

	for index, statement := range engine.config.Statements {
		statementNormalised := normaliseSQLForMatching(statement.SQL)
		if strings.Contains(normalised, statementNormalised) {
			results = append(results, querier_dto.ParsedStatement{
				Raw:      mockStatementIndex(index),
				Location: 0,
				Length:   len(sql),
			})
		}
	}

	return results, nil
}

func (engine *mockEngine) ApplyDDL(statement querier_dto.ParsedStatement) (*querier_dto.CatalogueMutation, error) {
	rawIndex, ok := statement.Raw.(mockStatementIndex)
	if !ok {
		return nil, fmt.Errorf("mock engine: unexpected statement raw type %T", statement.Raw)
	}
	index := int(rawIndex)

	if index < 0 || index >= len(engine.config.Statements) {
		return nil, fmt.Errorf("mock engine: statement index %d out of range", index)
	}

	entry := engine.config.Statements[index]
	if entry.Mutation == nil {
		return nil, nil
	}

	var mutation mockMutation
	if err := json.Unmarshal(*entry.Mutation, &mutation); err != nil {
		return nil, fmt.Errorf("mock engine: unmarshalling mutation: %w", err)
	}

	return convertMutation(mutation), nil
}

func (engine *mockEngine) AnalyseQuery(
	catalogue *querier_dto.Catalogue,
	statement querier_dto.ParsedStatement,
) (*querier_dto.RawQueryAnalysis, error) {
	engine.recordedCatalogue = catalogue

	rawIndex, ok := statement.Raw.(mockStatementIndex)
	if !ok {
		return nil, fmt.Errorf("mock engine: unexpected statement raw type %T", statement.Raw)
	}
	index := int(rawIndex)

	if index < 0 || index >= len(engine.config.Statements) {
		return nil, fmt.Errorf("mock engine: statement index %d out of range", index)
	}

	entry := engine.config.Statements[index]
	if entry.RawAnalysis == nil {
		return &querier_dto.RawQueryAnalysis{}, nil
	}

	var analysis mockRawAnalysis
	if err := json.Unmarshal(*entry.RawAnalysis, &analysis); err != nil {
		return nil, fmt.Errorf("mock engine: unmarshalling raw analysis: %w", err)
	}

	return convertRawAnalysis(analysis), nil
}

func (engine *mockEngine) BuiltinFunctions() *querier_dto.FunctionCatalogue {
	catalogue := &querier_dto.FunctionCatalogue{
		Functions: make(map[string][]*querier_dto.FunctionSignature),
	}

	for name, signatures := range engine.config.BuiltinFunctions {
		for _, signature := range signatures {
			catalogue.Functions[name] = append(catalogue.Functions[name], convertFunctionSignature(name, signature))
		}
	}

	return catalogue
}

func (engine *mockEngine) BuiltinTypes() *querier_dto.TypeCatalogue {
	catalogue := &querier_dto.TypeCatalogue{
		Types: make(map[string]querier_dto.SQLType),
	}

	for name, sqlType := range engine.config.BuiltinTypes {
		catalogue.Types[name] = convertSQLType(sqlType)
	}

	return catalogue
}

func (engine *mockEngine) NormaliseTypeName(name string, _ ...int) querier_dto.SQLType {
	lowered := strings.ToLower(name)
	if sqlType, exists := engine.config.BuiltinTypes[lowered]; exists {
		return convertSQLType(sqlType)
	}
	return querier_dto.SQLType{
		Category:   querier_dto.TypeCategoryUnknown,
		EngineName: name,
	}
}

func (*mockEngine) ParameterStyle() querier_dto.ParameterStyle {
	return querier_dto.ParameterStyleDollar
}

func (*mockEngine) SupportedDirectivePrefixes() []querier_dto.DirectiveParameterPrefix {
	return []querier_dto.DirectiveParameterPrefix{
		{Prefix: '$', IsNamed: false},
	}
}

func (*mockEngine) SupportsReturning() bool {
	return true
}

func (*mockEngine) Dialect() string {
	return "mock"
}

func (*mockEngine) SupportedExpressions() querier_dto.SQLExpressionFeature {
	return querier_dto.SQLFeaturesAll
}

func (*mockEngine) DefaultSchema() string {
	return "public"
}

func (*mockEngine) PromoteType(left querier_dto.SQLType, right querier_dto.SQLType) querier_dto.SQLType {
	numericRank := map[string]int{
		querier_dto.CanonicalInt2:    1,
		querier_dto.CanonicalInt4:    2,
		querier_dto.CanonicalInt8:    3,
		querier_dto.CanonicalFloat4:  4,
		querier_dto.CanonicalFloat8:  5,
		querier_dto.CanonicalNumeric: 6,
	}

	textRank := map[string]int{
		querier_dto.CanonicalChar:    1,
		querier_dto.CanonicalVarchar: 2,
		querier_dto.CanonicalText:    3,
	}

	temporalRank := map[string]int{
		querier_dto.CanonicalDate:        1,
		querier_dto.CanonicalTime:        2,
		querier_dto.CanonicalTimestamp:   3,
		querier_dto.CanonicalTimestampTZ: 4,
	}

	switch left.Category {
	case querier_dto.TypeCategoryInteger, querier_dto.TypeCategoryFloat, querier_dto.TypeCategoryDecimal:
		leftRank := numericRank[strings.ToLower(left.EngineName)]
		rightRank := numericRank[strings.ToLower(right.EngineName)]
		if rightRank > leftRank {
			return right
		}
		return left

	case querier_dto.TypeCategoryText:
		leftRank := textRank[strings.ToLower(left.EngineName)]
		rightRank := textRank[strings.ToLower(right.EngineName)]
		if rightRank > leftRank {
			return right
		}
		return left

	case querier_dto.TypeCategoryTemporal:
		leftRank := temporalRank[strings.ToLower(left.EngineName)]
		rightRank := temporalRank[strings.ToLower(right.EngineName)]
		if rightRank > leftRank {
			return right
		}
		return left
	}

	return left
}

func (*mockEngine) CanImplicitCast(from querier_dto.SQLTypeCategory, to querier_dto.SQLTypeCategory) bool {
	switch {
	case from == querier_dto.TypeCategoryInteger && to == querier_dto.TypeCategoryDecimal:
		return true
	case from == querier_dto.TypeCategoryInteger && to == querier_dto.TypeCategoryFloat:
		return true
	case from == querier_dto.TypeCategoryDecimal && to == querier_dto.TypeCategoryFloat:
		return true
	case from == querier_dto.TypeCategoryText && to == querier_dto.TypeCategoryText:
		return true
	default:
		return false
	}
}

func (*mockEngine) CommentStyle() querier_dto.CommentStyle {
	return querier_dto.DefaultSQLCommentStyle()
}

func (*mockEngine) TableValuedFunctionColumns(_ string) []querier_dto.ScopedColumn {
	return nil
}

func normaliseSQLForMatching(sql string) string {
	lines := strings.Split(sql, "\n")
	var filtered []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		if trimmed == "" {
			continue
		}
		filtered = append(filtered, trimmed)
	}
	joined := strings.Join(filtered, " ")

	var builder strings.Builder
	previousSpace := false
	for _, character := range joined {
		if character == ' ' || character == '\t' || character == '\n' || character == '\r' {
			if !previousSpace {
				builder.WriteByte(' ')
				previousSpace = true
			}
		} else {
			builder.WriteRune(character)
			previousSpace = false
		}
	}
	return strings.TrimSpace(builder.String())
}

func convertSQLType(source mockSQLType) querier_dto.SQLType {
	result := querier_dto.SQLType{
		Category:   parseSQLTypeCategory(source.Category),
		EngineName: source.EngineName,
		Schema:     source.Schema,
		Precision:  source.Precision,
		Scale:      source.Scale,
		Length:     source.Length,
		EnumValues: source.EnumValues,
	}
	if source.Element != nil {
		result.ElementType = new(convertSQLType(*source.Element))
	}
	return result
}

func convertColumn(source mockColumn) querier_dto.Column {
	return querier_dto.Column{
		Name:            source.Name,
		SQLType:         convertSQLType(source.SQLType),
		Nullable:        source.Nullable,
		HasDefault:      source.HasDefault,
		IsGenerated:     source.IsGenerated,
		IsArray:         source.IsArray,
		ArrayDimensions: source.ArrayDimensions,
		Comment:         source.Comment,
	}
}

func convertMutation(source mockMutation) *querier_dto.CatalogueMutation {
	mutation := &querier_dto.CatalogueMutation{
		Kind:       parseMutationKind(source.Kind),
		SchemaName: source.SchemaName,
		TableName:  source.TableName,
		ColumnName: source.ColumnName,
		EnumName:   source.EnumName,
		EnumValues: source.EnumValues,
		NewName:    source.NewName,
		PrimaryKey: source.PrimaryKey,
	}

	for _, column := range source.Columns {
		mutation.Columns = append(mutation.Columns, convertColumn(column))
	}

	if source.FunctionSignature != nil {
		mutation.FunctionSignature = convertFunctionSignature(source.TableName, *source.FunctionSignature)
	}

	return mutation
}

func convertFunctionSignature(name string, source mockFunctionSignature) *querier_dto.FunctionSignature {
	signature := &querier_dto.FunctionSignature{
		Name:              name,
		Schema:            source.Schema,
		ReturnType:        convertSQLType(source.ReturnType),
		ReturnsSet:        source.ReturnsSet,
		IsAggregate:       source.IsAggregate,
		NullableBehaviour: parseNullableBehaviour(source.NullableBehaviour),
	}

	for _, argument := range source.Arguments {
		signature.Arguments = append(signature.Arguments, querier_dto.FunctionArgument{
			Name: argument.Name,
			Type: convertSQLType(argument.Type),
		})
	}

	return signature
}

func convertRawAnalysis(source mockRawAnalysis) *querier_dto.RawQueryAnalysis {
	analysis := &querier_dto.RawQueryAnalysis{
		HasReturning: source.HasReturning,
	}

	for _, column := range source.OutputColumns {
		rawColumn := querier_dto.RawOutputColumn{
			Name:       column.Name,
			TableAlias: column.TableAlias,
			ColumnName: column.ColumnName,
			IsStar:     column.IsStar,
		}
		if column.Expression != nil {
			var raw any
			if err := json.Unmarshal(*column.Expression, &raw); err == nil {
				rawColumn.Expression = deserialiseExpression(raw)
			}
		}
		analysis.OutputColumns = append(analysis.OutputColumns, rawColumn)
	}

	for _, parameter := range source.ParameterReferences {
		reference := querier_dto.RawParameterReference{
			Number:  parameter.Number,
			Context: parseParameterContext(parameter.Context),
		}
		if parameter.ColumnReference != nil {
			reference.ColumnReference = &querier_dto.ColumnReference{
				TableAlias: parameter.ColumnReference.TableAlias,
				ColumnName: parameter.ColumnReference.ColumnName,
			}
		}
		if parameter.CastType != nil {
			reference.CastType = new(convertSQLType(*parameter.CastType))
		}
		analysis.ParameterReferences = append(analysis.ParameterReferences, reference)
	}

	for _, table := range source.FromTables {
		analysis.FromTables = append(analysis.FromTables, querier_dto.TableReference{
			Schema: table.Schema,
			Name:   table.Name,
			Alias:  table.Alias,
		})
	}

	for _, join := range source.JoinClauses {
		analysis.JoinClauses = append(analysis.JoinClauses, querier_dto.JoinClause{
			Kind: parseJoinKind(join.Kind),
			Table: querier_dto.TableReference{
				Schema: join.Table.Schema,
				Name:   join.Table.Name,
				Alias:  join.Table.Alias,
			},
		})
	}

	for _, cte := range source.CTEDefinitions {
		definition := querier_dto.RawCTEDefinition{
			Name:        cte.Name,
			IsRecursive: cte.IsRecursive,
		}
		for _, column := range cte.OutputColumns {
			rawColumn := querier_dto.RawOutputColumn{
				Name:       column.Name,
				TableAlias: column.TableAlias,
				ColumnName: column.ColumnName,
				IsStar:     column.IsStar,
			}
			if column.Expression != nil {
				var raw any
				if err := json.Unmarshal(*column.Expression, &raw); err == nil {
					rawColumn.Expression = deserialiseExpression(raw)
				}
			}
			definition.OutputColumns = append(definition.OutputColumns, rawColumn)
		}
		for _, table := range cte.FromTables {
			definition.FromTables = append(definition.FromTables, querier_dto.TableReference{
				Schema: table.Schema,
				Name:   table.Name,
				Alias:  table.Alias,
			})
		}
		analysis.CTEDefinitions = append(analysis.CTEDefinitions, definition)
	}

	for _, derived := range source.DerivedTables {
		reference := querier_dto.DerivedTableReference{
			Alias:    derived.Alias,
			Source:   parseDerivedTableSource(derived.Source),
			JoinKind: parseJoinKind(derived.JoinKind),
		}
		for _, column := range derived.Columns {
			reference.Columns = append(reference.Columns, querier_dto.ScopedColumn{
				Name:     column.Name,
				SQLType:  convertSQLType(column.SQLType),
				Nullable: column.Nullable,
			})
		}
		analysis.DerivedTables = append(analysis.DerivedTables, reference)
	}

	for _, groupByColumn := range source.GroupByColumns {
		analysis.GroupByColumns = append(analysis.GroupByColumns, querier_dto.ColumnReference{
			TableAlias: groupByColumn.TableAlias,
			ColumnName: groupByColumn.ColumnName,
		})
	}

	return analysis
}

func parseDerivedTableSource(source string) querier_dto.DerivedTableSource {
	switch strings.ToLower(source) {
	case "unnest":
		return querier_dto.DerivedSourceUnnest
	case "flatten":
		return querier_dto.DerivedSourceFlatten
	case "table_function":
		return querier_dto.DerivedSourceTableFunction
	case "subquery":
		return querier_dto.DerivedSourceSubquery
	default:
		return querier_dto.DerivedSourceSubquery
	}
}

func deserialiseExpression(raw any) querier_dto.Expression {
	if raw == nil {
		return nil
	}

	expressionMap, ok := raw.(map[string]any)
	if !ok {
		return &querier_dto.UnknownExpression{}
	}

	kind, _ := expressionMap["kind"].(string)
	switch kind {
	case "column_ref":
		tableAlias, _ := expressionMap["tableAlias"].(string)
		columnName, _ := expressionMap["columnName"].(string)
		return &querier_dto.ColumnRefExpression{TableAlias: tableAlias, ColumnName: columnName}

	case "function_call":
		functionName, _ := expressionMap["functionName"].(string)
		schema, _ := expressionMap["schema"].(string)
		return &querier_dto.FunctionCallExpression{
			FunctionName: functionName,
			Schema:       schema,
			Arguments:    deserialiseExpressionList(expressionMap["arguments"]),
		}

	case "coalesce":
		return &querier_dto.CoalesceExpression{
			Arguments: deserialiseExpressionList(expressionMap["arguments"]),
		}

	case "cast":
		typeName, _ := expressionMap["typeName"].(string)
		return &querier_dto.CastExpression{
			TypeName: typeName,
			Inner:    deserialiseExpression(expressionMap["expression"]),
		}

	case "literal":
		typeName, _ := expressionMap["typeName"].(string)
		return &querier_dto.LiteralExpression{TypeName: typeName}

	case "binary_op":
		operator, _ := expressionMap["operator"].(string)
		return &querier_dto.BinaryOpExpression{
			Operator: operator,
			Left:     deserialiseExpression(expressionMap["left"]),
			Right:    deserialiseExpression(expressionMap["right"]),
		}

	case "comparison":
		operator, _ := expressionMap["operator"].(string)
		return &querier_dto.ComparisonExpression{
			Operator: operator,
			Left:     deserialiseExpression(expressionMap["left"]),
			Right:    deserialiseExpression(expressionMap["right"]),
		}

	case "is_null":
		negated, _ := expressionMap["negated"].(bool)
		return &querier_dto.IsNullExpression{
			Inner:   deserialiseExpression(expressionMap["expression"]),
			Negated: negated,
		}

	case "in_list":
		return &querier_dto.InListExpression{
			Inner:  deserialiseExpression(expressionMap["expression"]),
			Values: deserialiseExpressionList(expressionMap["values"]),
		}

	case "between":
		return &querier_dto.BetweenExpression{
			Inner: deserialiseExpression(expressionMap["expression"]),
			Low:   deserialiseExpression(expressionMap["low"]),
			High:  deserialiseExpression(expressionMap["high"]),
		}

	case "logical_op":
		operator, _ := expressionMap["operator"].(string)
		return &querier_dto.LogicalOpExpression{
			Operator: operator,
			Operands: deserialiseExpressionList(expressionMap["operands"]),
		}

	case "unary_op":
		operator, _ := expressionMap["operator"].(string)
		return &querier_dto.UnaryOpExpression{
			Operator: operator,
			Operand:  deserialiseExpression(expressionMap["operand"]),
		}

	case "case_when":
		expression := &querier_dto.CaseWhenExpression{}
		if rawBranches, ok := expressionMap["branches"].([]any); ok {
			for _, rawBranch := range rawBranches {
				if branch, ok := rawBranch.(map[string]any); ok {
					expression.Branches = append(expression.Branches, querier_dto.CaseWhenBranch{
						Condition: deserialiseExpression(branch["condition"]),
						Result:    deserialiseExpression(branch["result"]),
					})
				}
			}
		}
		if elseResult, hasElse := expressionMap["elseResult"]; hasElse && elseResult != nil {
			expression.ElseResult = deserialiseExpression(elseResult)
		}
		return expression

	case "exists":
		return &querier_dto.ExistsExpression{}

	case "window_function":
		innerFunction := deserialiseExpression(expressionMap["function"])
		if functionCall, ok := innerFunction.(*querier_dto.FunctionCallExpression); ok {
			return &querier_dto.WindowFunctionExpression{Function: functionCall}
		}
		return &querier_dto.UnknownExpression{}

	case "array_subscript":
		return &querier_dto.ArraySubscriptExpression{
			Array: deserialiseExpression(expressionMap["array"]),
			Index: deserialiseExpression(expressionMap["index"]),
		}

	default:
		return &querier_dto.UnknownExpression{}
	}
}

func deserialiseExpressionList(raw any) []querier_dto.Expression {
	rawList, ok := raw.([]any)
	if !ok {
		return nil
	}

	result := make([]querier_dto.Expression, len(rawList))
	for i, item := range rawList {
		result[i] = deserialiseExpression(item)
	}
	return result
}

func parseSQLTypeCategory(category string) querier_dto.SQLTypeCategory {
	switch strings.ToLower(category) {
	case "integer":
		return querier_dto.TypeCategoryInteger
	case "float":
		return querier_dto.TypeCategoryFloat
	case "decimal":
		return querier_dto.TypeCategoryDecimal
	case "boolean":
		return querier_dto.TypeCategoryBoolean
	case "text":
		return querier_dto.TypeCategoryText
	case "bytea":
		return querier_dto.TypeCategoryBytea
	case "temporal":
		return querier_dto.TypeCategoryTemporal
	case "json":
		return querier_dto.TypeCategoryJSON
	case "uuid":
		return querier_dto.TypeCategoryUUID
	case "network":
		return querier_dto.TypeCategoryNetwork
	case "geometric":
		return querier_dto.TypeCategoryGeometric
	case "enum":
		return querier_dto.TypeCategoryEnum
	case "composite":
		return querier_dto.TypeCategoryComposite
	case "array":
		return querier_dto.TypeCategoryArray
	case "range":
		return querier_dto.TypeCategoryRange
	default:
		return querier_dto.TypeCategoryUnknown
	}
}

func parseMutationKind(kind string) querier_dto.MutationKind {
	switch strings.ToLower(kind) {
	case "createtable":
		return querier_dto.MutationCreateTable
	case "droptable":
		return querier_dto.MutationDropTable
	case "altertableaddcolumn":
		return querier_dto.MutationAlterTableAddColumn
	case "altertabledropcolumn":
		return querier_dto.MutationAlterTableDropColumn
	case "altertablealtercolumn":
		return querier_dto.MutationAlterTableAlterColumn
	case "altertablerenamecolumn":
		return querier_dto.MutationAlterTableRenameColumn
	case "altertablerenametable":
		return querier_dto.MutationAlterTableRenameTable
	case "altertablesetschema":
		return querier_dto.MutationAlterTableSetSchema
	case "createenum":
		return querier_dto.MutationCreateEnum
	case "alterenumaddvalue":
		return querier_dto.MutationAlterEnumAddValue
	case "alterenumrenamevalue":
		return querier_dto.MutationAlterEnumRenameValue
	case "dropenum":
		return querier_dto.MutationDropEnum
	case "createcompositetype":
		return querier_dto.MutationCreateCompositeType
	case "droptype":
		return querier_dto.MutationDropType
	case "createfunction":
		return querier_dto.MutationCreateFunction
	case "dropfunction":
		return querier_dto.MutationDropFunction
	case "createschema":
		return querier_dto.MutationCreateSchema
	case "dropschema":
		return querier_dto.MutationDropSchema
	case "createview":
		return querier_dto.MutationCreateView
	case "dropview":
		return querier_dto.MutationDropView
	case "createindex":
		return querier_dto.MutationCreateIndex
	case "dropindex":
		return querier_dto.MutationDropIndex
	case "createextension":
		return querier_dto.MutationCreateExtension
	case "comment":
		return querier_dto.MutationComment
	default:
		return querier_dto.MutationCreateTable
	}
}

func parseJoinKind(kind string) querier_dto.JoinKind {
	switch strings.ToLower(kind) {
	case "inner":
		return querier_dto.JoinInner
	case "left":
		return querier_dto.JoinLeft
	case "right":
		return querier_dto.JoinRight
	case "full":
		return querier_dto.JoinFull
	case "cross":
		return querier_dto.JoinCross
	default:
		return querier_dto.JoinInner
	}
}

func parseParameterContext(context string) querier_dto.ParameterContext {
	switch strings.ToLower(context) {
	case "comparison":
		return querier_dto.ParameterContextComparison
	case "assignment":
		return querier_dto.ParameterContextAssignment
	case "functionargument":
		return querier_dto.ParameterContextFunctionArgument
	case "cast":
		return querier_dto.ParameterContextCast
	case "inlist":
		return querier_dto.ParameterContextInList
	case "limit":
		return querier_dto.ParameterContextLimit
	case "offset":
		return querier_dto.ParameterContextOffset
	default:
		return querier_dto.ParameterContextUnknown
	}
}

func parseNullableBehaviour(behaviour string) querier_dto.FunctionNullableBehaviour {
	switch strings.ToLower(behaviour) {
	case "calledonnull":
		return querier_dto.FunctionNullableCalledOnNull
	case "returnsnullonnull":
		return querier_dto.FunctionNullableReturnsNullOnNull
	case "nevernull":
		return querier_dto.FunctionNullableNeverNull
	default:
		return querier_dto.FunctionNullableCalledOnNull
	}
}
