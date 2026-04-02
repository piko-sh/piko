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

package lsp_domain

import (
	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/esbuild/js_ast"
	"piko.sh/piko/internal/esbuild/js_parser"
	"piko.sh/piko/internal/esbuild/logger"
)

// extractClientScriptExports parses the TypeScript or JavaScript client
// script block and returns the names of exported functions. This is used for
// LSP completion suggestions when the user types p-on:click="".
//
// Returns []string which contains the exported function names, or nil if
// parsing fails or no client script exists.
func (d *document) extractClientScriptExports() []string {
	tree, ok := d.parseClientScript()
	if !ok {
		return nil
	}

	return extractExportsFromAST(tree)
}

// parseClientScript parses the client script block and returns the resulting
// AST. This is shared between export extraction and function extraction.
//
// Returns *js_ast.AST which is the parsed AST, or nil if parsing fails.
// Returns bool which indicates whether the parsing succeeded.
func (d *document) parseClientScript() (*js_ast.AST, bool) {
	if len(d.Content) == 0 {
		return nil, false
	}

	sfcResult := d.getSFCResult()
	if sfcResult == nil {
		return nil, false
	}

	clientScript, found := sfcResult.ClientScript()
	if !found || clientScript.Content == "" {
		return nil, false
	}

	parseLog := logger.NewDeferLog(logger.DeferLogAll, nil)

	tree, ok := js_parser.Parse(
		parseLog,
		logger.Source{
			Index:          0,
			KeyPath:        logger.Path{Text: d.URI.Filename()},
			PrettyPaths:    logger.PrettyPaths{Rel: d.URI.Filename(), Abs: d.URI.Filename()},
			Contents:       clientScript.Content,
			IdentifierName: d.URI.Filename(),
		},
		parserOptions(),
	)

	if !ok {
		return nil, false
	}

	return &tree, true
}

// clientScriptFunction holds a function name and whether it is exported.
type clientScriptFunction struct {
	// Name is the function name.
	Name string

	// Exported indicates whether the function is exported from the script.
	Exported bool
}

// extractClientScriptFunctions parses the client script block and returns all
// top-level function names, both exported and non-exported. This is used for
// event handler completion in templates.
//
// Returns []clientScriptFunction which contains the function names, or nil if
// parsing fails or no client script exists.
func (d *document) extractClientScriptFunctions() []clientScriptFunction {
	tree, ok := d.parseClientScript()
	if !ok {
		return nil
	}

	return extractAllFunctionsFromAST(tree)
}

// parserOptions creates parser settings for TypeScript parsing.
//
// Returns js_parser.Options which holds the parser settings.
func parserOptions() js_parser.Options {
	return js_parser.OptionsFromConfig(&config.Options{
		TS: config.TSOptions{
			Parse: true,
		},
	})
}

// extractExportsFromAST walks the AST and collects exported function names.
//
// Takes tree (*js_ast.AST) which is the parsed JavaScript AST to walk.
//
// Returns []string which contains the unique exported function names found.
func extractExportsFromAST(tree *js_ast.AST) []string {
	exports := make([]string, 0)
	seen := make(map[string]bool)

	for i := range tree.Parts {
		part := &tree.Parts[i]
		for _, statement := range part.Stmts {
			names := extractExportNamesFromStmt(statement, tree.Symbols)
			for _, name := range names {
				if !seen[name] {
					seen[name] = true
					exports = append(exports, name)
				}
			}
		}
	}

	return exports
}

// extractExportNamesFromStmt extracts export names from a single statement.
//
// Takes statement (js_ast.Stmt) which is the JavaScript AST statement to check.
// Takes symbols ([]ast.Symbol) which provides symbol data for lookups.
//
// Returns []string which contains the extracted export names, or nil if the
// statement is not an export.
func extractExportNamesFromStmt(statement js_ast.Stmt, symbols []ast.Symbol) []string {
	switch s := statement.Data.(type) {
	case *js_ast.SFunction:
		return extractExportedFunctionName(s, symbols)
	case *js_ast.SExportClause:
		return extractExportClauseNames(s)
	case *js_ast.SExportDefault:
		return extractDefaultExportName(s, symbols)
	case *js_ast.SLocal:
		return extractLocalExportNames(s, symbols)
	default:
		return nil
	}
}

// extractExportedFunctionName gets the name from an exported function statement.
//
// Takes s (*js_ast.SFunction) which is the function statement to check.
// Takes symbols ([]ast.Symbol) which provides the symbol table for name lookup.
//
// Returns []string which contains the function name, or nil if the function
// is not exported or has no name.
func extractExportedFunctionName(s *js_ast.SFunction, symbols []ast.Symbol) []string {
	if !s.IsExport || s.Fn.Name == nil {
		return nil
	}
	name := getSymbolName(s.Fn.Name.Ref, symbols)
	if name == "" {
		return nil
	}
	return []string{name}
}

// extractExportClauseNames gets all names from an export clause
// (e.g. export { foo, bar }).
//
// Takes s (*js_ast.SExportClause) which is the export clause to process.
//
// Returns []string which holds the alias names from the export clause.
func extractExportClauseNames(s *js_ast.SExportClause) []string {
	names := make([]string, 0, len(s.Items))
	for _, item := range s.Items {
		if item.Alias != "" {
			names = append(names, item.Alias)
		}
	}
	return names
}

// extractDefaultExportName gets the name from a default export statement.
//
// Takes s (*js_ast.SExportDefault) which is the default export statement.
// Takes symbols ([]ast.Symbol) which provides the symbol table for name lookup.
//
// Returns []string which contains the function name, or nil if the export is
// not a named function.
func extractDefaultExportName(s *js_ast.SExportDefault, symbols []ast.Symbol) []string {
	fnStmt, ok := s.Value.Data.(*js_ast.SFunction)
	if !ok || fnStmt.Fn.Name == nil {
		return nil
	}
	name := getSymbolName(fnStmt.Fn.Name.Ref, symbols)
	if name == "" {
		return nil
	}
	return []string{name}
}

// extractLocalExportNames extracts names from exported local declarations
// (export const foo = ...).
//
// Takes s (*js_ast.SLocal) which is the local declaration statement to check.
// Takes symbols ([]ast.Symbol) which provides symbol data for bindings.
//
// Returns []string which contains the exported names, or nil if not exported.
func extractLocalExportNames(s *js_ast.SLocal, symbols []ast.Symbol) []string {
	if !s.IsExport {
		return nil
	}
	names := make([]string, 0, len(s.Decls))
	for _, declaration := range s.Decls {
		name := extractBindingName(declaration.Binding, symbols)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

// extractAllFunctionsFromAST walks the AST and collects all top-level function
// names, both exported and non-exported.
//
// Takes tree (*js_ast.AST) which is the parsed JavaScript AST to walk.
//
// Returns []clientScriptFunction which contains the unique function names found.
func extractAllFunctionsFromAST(tree *js_ast.AST) []clientScriptFunction {
	functions := make([]clientScriptFunction, 0)
	seen := make(map[string]bool)

	for i := range tree.Parts {
		part := &tree.Parts[i]
		for _, statement := range part.Stmts {
			fns := extractFunctionNamesFromStmt(statement, tree.Symbols)
			for _, function := range fns {
				if !seen[function.Name] {
					seen[function.Name] = true
					functions = append(functions, function)
				}
			}
		}
	}

	return functions
}

// extractFunctionNamesFromStmt extracts function names from a single statement,
// regardless of export status.
//
// Takes statement (js_ast.Stmt) which is the JavaScript AST statement to check.
// Takes symbols ([]ast.Symbol) which provides symbol data for lookups.
//
// Returns []clientScriptFunction which contains the extracted function names.
func extractFunctionNamesFromStmt(statement js_ast.Stmt, symbols []ast.Symbol) []clientScriptFunction { //nolint:revive // dispatch table
	switch s := statement.Data.(type) {
	case *js_ast.SFunction:
		if s.Fn.Name == nil {
			return nil
		}
		name := getSymbolName(s.Fn.Name.Ref, symbols)
		if name == "" {
			return nil
		}
		return []clientScriptFunction{{Name: name, Exported: s.IsExport}}

	case *js_ast.SLocal:
		fns := make([]clientScriptFunction, 0, len(s.Decls))
		for _, declaration := range s.Decls {
			name := extractBindingName(declaration.Binding, symbols)
			if name != "" {
				fns = append(fns, clientScriptFunction{Name: name, Exported: s.IsExport})
			}
		}
		return fns

	case *js_ast.SExportDefault:
		fnStmt, ok := s.Value.Data.(*js_ast.SFunction)
		if !ok || fnStmt.Fn.Name == nil {
			return nil
		}
		name := getSymbolName(fnStmt.Fn.Name.Ref, symbols)
		if name == "" {
			return nil
		}
		return []clientScriptFunction{{Name: name, Exported: true}}

	case *js_ast.SExportClause:
		fns := make([]clientScriptFunction, 0, len(s.Items))
		for _, item := range s.Items {
			if item.Alias != "" {
				fns = append(fns, clientScriptFunction{Name: item.Alias, Exported: true})
			}
		}
		return fns

	default:
		return nil
	}
}

// getSymbolName finds the original name of a symbol from its reference.
//
// Takes ref (ast.Ref) which identifies the symbol to look up.
// Takes symbols ([]ast.Symbol) which contains the list of known symbols.
//
// Returns string which is the original name, or empty if the reference index
// is out of range.
func getSymbolName(ref ast.Ref, symbols []ast.Symbol) string {
	if int(ref.InnerIndex) < len(symbols) {
		return symbols[ref.InnerIndex].OriginalName
	}
	return ""
}

// extractBindingName gets the name from a binding pattern when it is a simple
// identifier.
//
// Takes binding (js_ast.Binding) which is the binding pattern to extract the
// name from.
// Takes symbols ([]ast.Symbol) which provides the symbol table for name lookup.
//
// Returns string which is the name, or empty if the binding is not a simple
// identifier.
func extractBindingName(binding js_ast.Binding, symbols []ast.Symbol) string {
	if id, ok := binding.Data.(*js_ast.BIdentifier); ok {
		return getSymbolName(id.Ref, symbols)
	}
	return ""
}
