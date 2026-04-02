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

package compiler_domain

import (
	"fmt"
	"strings"

	parsejs "github.com/tdewolff/parse/v2/js"
	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/js_ast"
)

// ASTConverterService defines the interface for converting esbuild AST to
// tdewolff AST.
type ASTConverterService interface {
	// ConvertEsbuildToTdewolff converts an esbuild AST to a tdewolff AST.
	//
	// Takes esbuildAST (*js_ast.AST) which is the parsed esbuild syntax tree.
	// Takes registry (*RegistryContext) which provides the conversion context.
	//
	// Returns *parsejs.AST which is the equivalent tdewolff syntax tree.
	// Returns error when the conversion fails.
	ConvertEsbuildToTdewolff(esbuildAST *js_ast.AST, registry *RegistryContext) (*parsejs.AST, error)
}

// astConverterService implements ASTConverterService to convert between
// esbuild and tdewolff AST formats.
type astConverterService struct{}

var _ ASTConverterService = (*astConverterService)(nil)

// ConvertEsbuildToTdewolff implements the ASTConverterService interface.
//
// Takes esbuildAST (*js_ast.AST) which is the parsed AST from esbuild.
// Takes registry (*RegistryContext) which provides symbol lookup context.
//
// Returns *parsejs.AST which is the converted AST in tdewolff format.
// Returns error when the conversion fails.
func (*astConverterService) ConvertEsbuildToTdewolff(esbuildAST *js_ast.AST, registry *RegistryContext) (*parsejs.AST, error) {
	return ConvertEsbuildToTdewolff(esbuildAST, registry)
}

// ASTConverter converts an esbuild AST to a tdewolff AST.
// It uses the symbol table to resolve identifier names, import records to
// resolve module paths, and the registry context to look up names of
// identifiers created by hand.
type ASTConverter struct {
	// registry holds context for looking up identifier names and comments
	// from the parsed AST; nil turns off name resolution.
	registry *RegistryContext

	// symbols is the symbol table used to resolve reference indices to names.
	symbols []ast.Symbol

	// importRecords stores import records used to look up module paths by index.
	importRecords []ast.ImportRecord
}

// NewASTConverter creates a converter with access to the given symbol table,
// import records, and registry context.
//
// Takes symbols ([]ast.Symbol) which provides the symbol table for lookups.
// Takes importRecords ([]ast.ImportRecord) which lists the import statements.
// Takes registry (*RegistryContext) which provides the registry context.
//
// Returns *ASTConverter which is ready for converting AST nodes.
func NewASTConverter(symbols []ast.Symbol, importRecords []ast.ImportRecord, registry *RegistryContext) *ASTConverter {
	return &ASTConverter{
		registry:      registry,
		symbols:       symbols,
		importRecords: importRecords,
	}
}

// resolveRef finds the original name for a symbol reference.
//
// Takes ref (ast.Ref) which identifies the symbol to look up.
//
// Returns string which is the original name of the symbol, or empty if the
// symbol cannot be found.
func (c *ASTConverter) resolveRef(ref ast.Ref) string {
	if c.symbols == nil {
		return ""
	}
	index := int(ref.InnerIndex)
	if index < 0 || index >= len(c.symbols) {
		return ""
	}
	return c.symbols[index].OriginalName
}

// NewASTConverterService creates a new AST converter service.
//
// Returns ASTConverterService which provides methods for converting AST nodes.
func NewASTConverterService() ASTConverterService {
	return &astConverterService{}
}

// ConvertEsbuildToTdewolff converts an esbuild AST to a tdewolff AST,
// using esbuild for TypeScript parsing while keeping tdewolff-based code
// generation, which does not need symbol tables.
//
// When esbuildAST is nil, returns an empty AST without error.
//
// Takes esbuildAST (*js_ast.AST) which is the parsed esbuild syntax tree.
// Takes registry (*RegistryContext) which looks up names of manually-created
// identifiers.
//
// Returns *parsejs.AST which is the converted tdewolff syntax tree.
// Returns error when a statement cannot be converted.
func ConvertEsbuildToTdewolff(esbuildAST *js_ast.AST, registry *RegistryContext) (*parsejs.AST, error) {
	if esbuildAST == nil {
		return &parsejs.AST{}, nil
	}

	converter := NewASTConverter(esbuildAST.Symbols, esbuildAST.ImportRecords, registry)
	tdewolffAST := &parsejs.AST{}

	for partIndex := range esbuildAST.Parts {
		for statementIndex := range esbuildAST.Parts[partIndex].Stmts {
			statement := esbuildAST.Parts[partIndex].Stmts[statementIndex]
			convertedStmt, err := converter.convertStatement(statement)
			if err != nil {
				return nil, fmt.Errorf("converting statement: %w", err)
			}
			if convertedStmt != nil {
				tdewolffAST.List = append(tdewolffAST.List, convertedStmt)
			}
		}
	}

	return tdewolffAST, nil
}

// PrintExpr converts a JavaScript AST expression to source code.
//
// Takes expression (js_ast.Expr) which is the expression to convert.
// Takes registry (*RegistryContext) which looks up names for
// identifiers that were created by hand.
//
// Returns string which is the JavaScript source code, or an empty string if
// the expression is nil or conversion fails.
func PrintExpr(expression js_ast.Expr, registry *RegistryContext) string {
	if expression.Data == nil {
		return ""
	}

	converter := NewASTConverter(nil, nil, registry)
	tdewolffExpr, err := converter.convertExpression(expression)
	if err != nil || tdewolffExpr == nil {
		return ""
	}

	var builder strings.Builder
	tdewolffExpr.JS(&builder)
	return builder.String()
}

// PrintStatement converts a single JavaScript AST statement to source code.
//
// Takes statement (js_ast.Stmt) which is the statement to convert.
// Takes registry (*RegistryContext) which provides name lookup for identifiers
// that were created by hand.
//
// Returns string which contains the JavaScript source code. Returns an empty
// string if the statement is nil or conversion fails.
func PrintStatement(statement js_ast.Stmt, registry *RegistryContext) string {
	if statement.Data == nil {
		return ""
	}

	converter := NewASTConverter(nil, nil, registry)
	tdewolffStmt, err := converter.convertStatement(statement)
	if err != nil || tdewolffStmt == nil {
		return ""
	}

	var builder strings.Builder
	tdewolffStmt.JS(&builder)
	return builder.String()
}
