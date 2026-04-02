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
	"fmt"

	"piko.sh/piko/internal/querier/querier_dto"
)

// SQLiteEngine implements the querier EnginePort for SQLite.
type SQLiteEngine struct {
	functions *querier_dto.FunctionCatalogue

	types *querier_dto.TypeCatalogue
}

// NewSQLiteEngine creates a new SQLite engine adapter.
func NewSQLiteEngine() *SQLiteEngine {
	return &SQLiteEngine{
		functions: buildFunctionCatalogue(),
		types:     buildTypeCatalogue(),
	}
}

// ParseStatements tokenises and classifies SQL statements for the SQLite dialect.
func (*SQLiteEngine) ParseStatements(sql string) ([]querier_dto.ParsedStatement, error) {
	tokens, tokeniseError := tokenise(sql)
	if tokeniseError != nil {
		return nil, fmt.Errorf("tokenising SQL: %w", tokeniseError)
	}

	statementTokens := splitStatements(tokens)
	results := make([]querier_dto.ParsedStatement, 0, len(statementTokens))

	for _, stmtTokens := range statementTokens {
		kind := classifyStatement(stmtTokens)
		results = append(results, querier_dto.ParsedStatement{
			Raw:      &parsedStatement{tokens: stmtTokens, kind: kind},
			Location: stmtTokens[0].position,
			Length:   len(sql),
		})
	}

	return results, nil
}

// ApplyDDL applies a DDL statement to the catalogue for the SQLite dialect.
func (engine *SQLiteEngine) ApplyDDL(statement querier_dto.ParsedStatement) (*querier_dto.CatalogueMutation, error) {
	parsed, ok := statement.Raw.(*parsedStatement)
	if !ok {
		return nil, fmt.Errorf("unexpected statement type %T", statement.Raw)
	}

	p := newParser(parsed.tokens)

	switch parsed.kind {
	case statementKindCreateTable:
		return p.parseCreateTable(engine)
	case statementKindDropTable:
		return p.parseDropTable()
	case statementKindAlterTable:
		return p.parseAlterTable(engine)
	case statementKindCreateView:
		return p.parseCreateView()
	case statementKindDropView:
		return p.parseDropView()
	case statementKindCreateIndex:
		return p.parseCreateIndex()
	case statementKindDropIndex:
		return p.parseDropIndex()
	case statementKindCreateVirtualTable:
		return p.parseCreateVirtualTable(engine)
	case statementKindCreateTrigger:
		return p.parseCreateTrigger()
	case statementKindDropTrigger:
		return p.parseDropTrigger()
	default:
		return nil, nil
	}
}

// AnalyseQuery performs structural analysis of a DML statement for the SQLite dialect.
func (*SQLiteEngine) AnalyseQuery(
	_ *querier_dto.Catalogue,
	statement querier_dto.ParsedStatement,
) (*querier_dto.RawQueryAnalysis, error) {
	parsed, ok := statement.Raw.(*parsedStatement)
	if !ok {
		return nil, fmt.Errorf("unexpected statement type %T", statement.Raw)
	}

	p := newParser(parsed.tokens)

	switch parsed.kind {
	case statementKindSelect:
		return p.analyseSelect()
	case statementKindInsert:
		return p.analyseInsert()
	case statementKindUpdate:
		return p.analyseUpdate()
	case statementKindDelete:
		return p.analyseDelete()
	case statementKindValues:
		return p.analyseValues()
	default:
		return &querier_dto.RawQueryAnalysis{}, nil
	}
}

// BuiltinFunctions returns the SQLite built-in function catalogue.
func (engine *SQLiteEngine) BuiltinFunctions() *querier_dto.FunctionCatalogue {
	return engine.functions
}

// BuiltinTypes returns the SQLite built-in type catalogue.
func (engine *SQLiteEngine) BuiltinTypes() *querier_dto.TypeCatalogue {
	return engine.types
}

// NormaliseTypeName resolves a raw type name to a structured SQLType
// using SQLite affinity rules.
func (*SQLiteEngine) NormaliseTypeName(name string, modifiers ...int) querier_dto.SQLType {
	return normaliseTypeName(name, modifiers...)
}

// ParameterStyle returns the question-mark parameter style used by SQLite.
func (*SQLiteEngine) ParameterStyle() querier_dto.ParameterStyle {
	return querier_dto.ParameterStyleQuestion
}

// SupportedDirectivePrefixes returns the parameter prefixes valid in SQLite directives.
func (*SQLiteEngine) SupportedDirectivePrefixes() []querier_dto.DirectiveParameterPrefix {
	return []querier_dto.DirectiveParameterPrefix{
		{Prefix: '?', IsNamed: false},
		{Prefix: ':', IsNamed: true},
		{Prefix: '@', IsNamed: true},
		{Prefix: '$', IsNamed: true},
	}
}

// SupportsReturning reports that SQLite supports RETURNING clauses.
func (*SQLiteEngine) SupportsReturning() bool {
	return true
}

// Dialect returns "sqlite".
func (*SQLiteEngine) Dialect() string {
	return "sqlite"
}

// SupportedExpressions returns the expression features supported by SQLite.
func (*SQLiteEngine) SupportedExpressions() querier_dto.SQLExpressionFeature {
	return querier_dto.SQLFeaturesBase |
		querier_dto.SQLFeatureWindowFunction |
		querier_dto.SQLFeatureJSONOp |
		querier_dto.SQLFeatureScalarSubquery |
		querier_dto.SQLFeatureBitwiseOp
}

// DefaultSchema returns the default schema name for SQLite.
func (*SQLiteEngine) DefaultSchema() string {
	return "main"
}

// TableValuedFunctionColumns returns the output column schema for a known
// table-valued function, or nil if the function is not recognised.
func (*SQLiteEngine) TableValuedFunctionColumns(functionName string) []querier_dto.ScopedColumn {
	columns, exists := tableValuedFunctionColumns[functionName]
	if !exists {
		return nil
	}
	result := make([]querier_dto.ScopedColumn, len(columns))
	copy(result, columns)
	return result
}

// PromoteType returns the wider type within the same category. SQLite has
// only four storage classes (INTEGER, REAL, TEXT, BLOB), so same-category
// operands are always the same type - the left operand is returned unchanged.
func (*SQLiteEngine) PromoteType(left querier_dto.SQLType, _ querier_dto.SQLType) querier_dto.SQLType {
	return left
}

// CanImplicitCast reports whether SQLite allows implicit conversion between
// the given type categories.
func (*SQLiteEngine) CanImplicitCast(from querier_dto.SQLTypeCategory, to querier_dto.SQLTypeCategory) bool {
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

// CommentStyle returns the standard SQL comment style for SQLite.
func (*SQLiteEngine) CommentStyle() querier_dto.CommentStyle {
	return querier_dto.DefaultSQLCommentStyle()
}
