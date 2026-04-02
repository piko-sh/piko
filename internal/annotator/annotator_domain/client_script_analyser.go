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

package annotator_domain

// Parses JavaScript and TypeScript client scripts to extract exported function metadata.
// Enables compile-time validation of event handlers by identifying available functions and their async status.

import (
	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/esbuild/js_ast"
	"piko.sh/piko/internal/esbuild/js_parser"
	"piko.sh/piko/internal/esbuild/logger"
)

// ClientScriptExports holds information about exported functions from a client
// script. It is used for compile-time validation of PK event handlers.
type ClientScriptExports struct {
	// ExportedFunctions maps function names to their export details.
	ExportedFunctions map[string]ExportedFunction

	// ScriptContent is the original script content, kept for reference.
	ScriptContent string

	// SourcePath is the file path to the PK file.
	SourcePath string
}

// ExportedFunction holds details about a function that is made available to
// client scripts.
type ExportedFunction struct {
	// Name is the exported name of the function.
	Name string

	// Params holds parameter type information extracted from TypeScript
	// annotations. Empty when no type annotations are present.
	Params []ParamInfo

	// IsAsync indicates whether the function is declared with the
	// async keyword.
	IsAsync bool
}

// ParamInfo holds type information for a single function parameter.
type ParamInfo struct {
	// Name is the parameter name (e.g. "age").
	Name string

	// TypeName is the raw TypeScript type annotation (e.g. "number",
	// "string[]", "SomeInterface").
	TypeName string

	// Category is the simplified type category: "string", "number",
	// "boolean", "object", or "any".
	Category string

	// Optional is true when the parameter has a ? marker.
	Optional bool

	// IsRest is true when the parameter has a ... spread operator.
	IsRest bool
}

// HasExport checks if a function with the given name is exported.
//
// Takes name (string) which specifies the function name to check.
//
// Returns bool which is true if the function is in the exports.
func (c *ClientScriptExports) HasExport(name string) bool {
	if c == nil || c.ExportedFunctions == nil {
		return false
	}
	_, exists := c.ExportedFunctions[name]
	return exists
}

// ExportNames returns a slice of all exported function names.
//
// Returns []string which contains the names of all exported functions, or nil
// if the receiver or its ExportedFunctions map is nil.
func (c *ClientScriptExports) ExportNames() []string {
	if c == nil || c.ExportedFunctions == nil {
		return nil
	}
	names := make([]string, 0, len(c.ExportedFunctions))
	for name := range c.ExportedFunctions {
		names = append(names, name)
	}
	return names
}

// AnalyseClientScript parses a TypeScript or JavaScript client script and
// extracts information about exported functions, enabling the system to check
// p-on event handlers at compile time.
//
// Takes content (string) which is the script source code to parse.
// Takes sourcePath (string) which is the file path used for error messages.
//
// Returns *ClientScriptExports which holds the extracted function details.
// Returns nil if the script is empty or cannot be parsed.
func AnalyseClientScript(content string, sourcePath string) *ClientScriptExports {
	if content == "" {
		return nil
	}

	paramsByFunc := ExtractFunctionParams(content)

	parseLog := logger.NewDeferLog(logger.DeferLogAll, nil)

	tree, ok := js_parser.Parse(
		parseLog,
		logger.Source{
			Index:          0,
			KeyPath:        logger.Path{Text: sourcePath},
			PrettyPaths:    logger.PrettyPaths{Rel: sourcePath, Abs: sourcePath},
			Contents:       content,
			IdentifierName: sourcePath,
		},
		clientScriptParserOptions(),
	)

	if !ok {
		return nil
	}

	exports := &ClientScriptExports{
		ExportedFunctions: make(map[string]ExportedFunction),
		ScriptContent:     content,
		SourcePath:        sourcePath,
	}

	extractExportsFromClientScript(&tree, exports)

	for name, function := range exports.ExportedFunctions {
		if params, ok := paramsByFunc[name]; ok {
			function.Params = params
			exports.ExportedFunctions[name] = function
		}
	}

	return exports
}

// clientScriptParserOptions creates parser settings for TypeScript code.
//
// Returns js_parser.Options which holds the parser settings.
func clientScriptParserOptions() js_parser.Options {
	return js_parser.OptionsFromConfig(&config.Options{
		TS: config.TSOptions{
			Parse: true,
		},
	})
}

// extractExportsFromClientScript walks the AST and populates the exports map.
//
// Takes tree (*js_ast.AST) which is the parsed JavaScript AST to scan.
// Takes exports (*ClientScriptExports) which is populated with found exports.
func extractExportsFromClientScript(tree *js_ast.AST, exports *ClientScriptExports) {
	for i := range tree.Parts {
		for _, statement := range tree.Parts[i].Stmts {
			extractExportFromStatement(statement, tree.Symbols, exports)
		}
	}
}

// extractExportFromStatement extracts export details from a single JavaScript
// AST statement.
//
// Takes statement (js_ast.Stmt) which is the statement to check for exports.
// Takes symbols ([]ast.Symbol) which provides symbol names for lookups.
// Takes exports (*ClientScriptExports) which collects the found exports.
func extractExportFromStatement(statement js_ast.Stmt, symbols []ast.Symbol, exports *ClientScriptExports) {
	switch s := statement.Data.(type) {
	case *js_ast.SFunction:
		extractFunctionExport(s, symbols, exports)
	case *js_ast.SExportClause:
		extractClauseExport(s, exports)
	case *js_ast.SExportDefault:
		extractDefaultExport(s, symbols, exports)
	case *js_ast.SLocal:
		extractLocalExport(s, symbols, exports)
	}
}

// extractFunctionExport handles top-level function declarations.
//
// Both exported and non-exported functions are captured because the PK
// transformer makes all top-level functions available as event handlers.
// The export keyword is optional in PK client scripts.
//
// Takes s (*js_ast.SFunction) which is the function statement to analyse.
// Takes symbols ([]ast.Symbol) which provides symbol lookup for names.
// Takes exports (*ClientScriptExports) which collects the found functions.
func extractFunctionExport(s *js_ast.SFunction, symbols []ast.Symbol, exports *ClientScriptExports) {
	if s.Fn.Name == nil {
		return
	}
	name := getClientScriptSymbolName(s.Fn.Name.Ref, symbols)
	if name == "" {
		return
	}
	exports.ExportedFunctions[name] = ExportedFunction{
		Name:    name,
		IsAsync: s.Fn.IsAsync,
	}
}

// extractClauseExport handles export clauses such as `export { foo, bar }`.
//
// Takes s (*js_ast.SExportClause) which contains the parsed export clause.
// Takes exports (*ClientScriptExports) which collects the exported functions.
func extractClauseExport(s *js_ast.SExportClause, exports *ClientScriptExports) {
	for _, item := range s.Items {
		if item.Alias != "" {
			exports.ExportedFunctions[item.Alias] = ExportedFunction{
				Name:    item.Alias,
				IsAsync: false,
			}
		}
	}
}

// extractDefaultExport handles `export default function` declarations.
//
// Takes s (*js_ast.SExportDefault) which is the default export statement to
// process.
// Takes symbols ([]ast.Symbol) which provides symbol data for name lookup.
// Takes exports (*ClientScriptExports) which stores the found function exports.
func extractDefaultExport(s *js_ast.SExportDefault, symbols []ast.Symbol, exports *ClientScriptExports) {
	fnStmt, ok := s.Value.Data.(*js_ast.SFunction)
	if !ok || fnStmt.Fn.Name == nil {
		return
	}
	name := getClientScriptSymbolName(fnStmt.Fn.Name.Ref, symbols)
	if name == "" {
		return
	}
	exports.ExportedFunctions[name] = ExportedFunction{
		Name:    name,
		IsAsync: fnStmt.Fn.IsAsync,
	}
}

// extractLocalExport finds functions in const, let, or var declarations.
//
// It checks each declaration to see if it is an arrow function or function
// expression. Both exported and non-exported declarations are captured because
// the PK transformer makes all top-level functions available as event handlers.
// The export keyword is not required in PK client scripts.
//
// Takes s (*js_ast.SLocal) which is the local statement to check.
// Takes symbols ([]ast.Symbol) which provides symbol data for binding names.
// Takes exports (*ClientScriptExports) which collects the found functions.
func extractLocalExport(s *js_ast.SLocal, symbols []ast.Symbol, exports *ClientScriptExports) {
	for _, declaration := range s.Decls {
		name := extractClientScriptBindingName(declaration.Binding, symbols)
		if name == "" {
			continue
		}
		if !isArrowOrFunctionExpr(declaration.ValueOrNil) {
			continue
		}
		isAsync := isAsyncArrowOrFunction(declaration.ValueOrNil)
		exports.ExportedFunctions[name] = ExportedFunction{
			Name:    name,
			IsAsync: isAsync,
		}
	}
}

// getClientScriptSymbolName finds the original name of a symbol by its
// reference.
//
// Takes ref (ast.Ref) which identifies the symbol to look up.
// Takes symbols ([]ast.Symbol) which contains the symbols to search.
//
// Returns string which is the original name, or empty if the reference is
// out of bounds.
func getClientScriptSymbolName(ref ast.Ref, symbols []ast.Symbol) string {
	if int(ref.InnerIndex) < len(symbols) {
		return symbols[ref.InnerIndex].OriginalName
	}
	return ""
}

// extractClientScriptBindingName gets the name from a binding pattern.
//
// Takes binding (js_ast.Binding) which is the binding pattern to extract a
// name from.
// Takes symbols ([]ast.Symbol) which provides the symbol table for looking up
// names.
//
// Returns string which is the name found, or empty if the binding is not an
// identifier.
func extractClientScriptBindingName(binding js_ast.Binding, symbols []ast.Symbol) string {
	if id, ok := binding.Data.(*js_ast.BIdentifier); ok {
		return getClientScriptSymbolName(id.Ref, symbols)
	}
	return ""
}

// isArrowOrFunctionExpr checks whether an expression is an arrow function or
// a function expression.
//
// Takes expression (js_ast.Expr) which is the expression to check.
//
// Returns bool which is true if the expression is an arrow function or
// function expression, false otherwise.
func isArrowOrFunctionExpr(expression js_ast.Expr) bool {
	switch expression.Data.(type) {
	case *js_ast.EArrow, *js_ast.EFunction:
		return true
	}
	return false
}

// isAsyncArrowOrFunction checks whether an expression is an async arrow or
// async function.
//
// Takes expression (js_ast.Expr) which is the expression to check.
//
// Returns bool which is true if the expression is async.
func isAsyncArrowOrFunction(expression js_ast.Expr) bool {
	switch e := expression.Data.(type) {
	case *js_ast.EArrow:
		return e.IsAsync
	case *js_ast.EFunction:
		return e.Fn.IsAsync
	}
	return false
}
