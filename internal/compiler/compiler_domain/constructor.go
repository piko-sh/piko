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
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/helpers"
	"piko.sh/piko/internal/esbuild/js_ast"
	"piko.sh/piko/internal/esbuild/logger"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safeconv"
)

// EnsureStandardConstructor checks that a class has a standard constructor and
// creates one if needed. A standard constructor has super() and this.init()
// calls.
//
// Takes classDecl (*js_ast.Class) which is the class to check.
// Takes registry (*RegistryContext) which provides the build context.
//
// Returns *js_ast.EFunction which is the standard constructor function.
// Returns error when classDecl is nil.
func EnsureStandardConstructor(
	ctx context.Context,
	classDecl *js_ast.Class,
	registry *RegistryContext,
) (*js_ast.EFunction, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "EnsureStandardConstructor")
	defer span.End()

	if classDecl == nil {
		err := errors.New("EnsureStandardConstructor: provided classDecl is nil")
		l.ReportError(span, err, "Nil class declaration provided")
		return nil, err
	}

	className := ""
	l = l.With(logger_domain.String(propClassName, className))
	span.SetAttributes(attribute.String(propClassName, className))

	constructor := findConstructorMethod(classDecl, registry)
	if constructor == nil {
		return createNewConstructor(ctx, span, classDecl, registry)
	}

	return standardiseExistingConstructor(ctx, span, constructor)
}

// EnsureConnectedCallback verifies the class has a connectedCallback lifecycle
// method.
//
// Takes classDecl (*js_ast.Class) which is the class to check.
// Takes registry (*RegistryContext) which provides the build context.
//
// Returns *js_ast.EFunction which is the existing or newly created callback.
// Returns error when classDecl is nil.
func EnsureConnectedCallback(ctx context.Context, classDecl *js_ast.Class, registry *RegistryContext) (*js_ast.EFunction, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "EnsureConnectedCallback")
	defer span.End()

	if classDecl == nil {
		err := errors.New("EnsureConnectedCallback: provided classDecl is nil")
		l.ReportError(span, err, "Nil class declaration provided")
		return nil, err
	}

	l = l.With(logger_domain.String(propClassName, ""))
	span.SetAttributes(attribute.String(propClassName, ""))

	if existing := findConnectedCallback(classDecl, registry); existing != nil {
		l.Trace("Found existing connectedCallback.")
		return existing, nil
	}

	return createConnectedCallback(ctx, span, classDecl, registry)
}

// InjectInitIntoConnectedCallback adds startup logic to the connectedCallback
// method of a web component. It inserts an init call and a super callback at
// the start of the method body.
//
// Takes connectedCallback (*js_ast.EFunction) which is the callback method to
// modify.
//
// Returns error when connectedCallback is nil or when parsing the init
// statement fails.
func InjectInitIntoConnectedCallback(ctx context.Context, connectedCallback *js_ast.EFunction) error {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "InjectInitIntoConnectedCallback")
	defer span.End()

	if connectedCallback == nil {
		err := errors.New("InjectInitIntoConnectedCallback: provided connectedCallback is nil")
		l.ReportError(span, err, "Nil connectedCallback provided")
		return err
	}

	startTime := time.Now()

	initStmt, err := parseSnippetAsStatement("this.init(instance.call(this, this));")
	if err != nil {
		l.ReportError(span, err, "Failed to parse init statement snippet")
		return fmt.Errorf("failed to parse init statement snippet: %w", err)
	}

	superCallStmt, err := parseSnippetAsStatement("super.connectedCallback();")
	if err != nil {
		l.ReportError(span, err, "Failed to parse super.connectedCallback snippet")
		return fmt.Errorf("failed to parse super.connectedCallback snippet: %w", err)
	}

	originalBody := connectedCallback.Fn.Body.Block.Stmts
	newBody := make([]js_ast.Stmt, 0, 2+len(originalBody))
	newBody = append(newBody, initStmt, superCallStmt)
	newBody = append(newBody, originalBody...)
	connectedCallback.Fn.Body.Block.Stmts = newBody

	span.SetAttributes(attribute.Int64("durationMs", time.Since(startTime).Milliseconds()))
	ConnectedCallbackInjectionCount.Add(ctx, 1)
	l.Trace("Injected init into connectedCallback")
	span.SetStatus(codes.Ok, "Successfully injected init into connectedCallback")

	return nil
}

// createNewConstructor creates a new constructor function with a super() call
// for a class that does not have one.
//
// Takes span (trace.Span) which provides tracing context for error reporting.
// Takes classDecl (*js_ast.Class) which is the class to receive the new
// constructor.
// Takes registry (*RegistryContext) which provides identifier creation.
//
// Returns *js_ast.EFunction which is the newly created constructor function.
// Returns error when the super() statement cannot be parsed.
func createNewConstructor(ctx context.Context, span trace.Span, classDecl *js_ast.Class, registry *RegistryContext) (*js_ast.EFunction, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("No constructor found, creating a new standard one.")

	superStmt, err := parseSnippetAsStatement("super();")
	if err != nil {
		l.ReportError(span, err, "Error creating super() statement")
		return nil, fmt.Errorf("internal error creating super() statement: %w", err)
	}

	constructor := &js_ast.EFunction{
		Fn: js_ast.Fn{
			Body: js_ast.FnBody{Block: js_ast.SBlock{Stmts: []js_ast.Stmt{superStmt}}},
		},
	}

	constructorProp := js_ast.Property{
		Key:        registry.MakeIdentifierExpr("constructor"),
		ValueOrNil: js_ast.Expr{Data: constructor},
		Kind:       js_ast.PropertyMethod,
	}
	classDecl.Properties = append([]js_ast.Property{constructorProp}, classDecl.Properties...)

	ConstructorCreationCount.Add(ctx, 1)
	l.Trace("Created new constructor")
	return constructor, nil
}

// standardiseExistingConstructor updates an existing constructor to use a
// standard format. It adds a super() call at the start of the body and keeps
// the original statements, but removes any existing super() or init instance
// calls.
//
// Takes span (trace.Span) which provides the tracing context for error
// reporting.
// Takes constructor (*js_ast.EFunction) which is the constructor
// function to update.
//
// Returns *js_ast.EFunction which is the updated constructor.
// Returns error when the super() statement cannot be parsed.
func standardiseExistingConstructor(ctx context.Context, span trace.Span, constructor *js_ast.EFunction) (*js_ast.EFunction, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Existing constructor found, standardising.")

	superCallStmt, err := parseSnippetAsStatement("super();")
	if err != nil {
		l.ReportError(span, err, "Error creating super() statement for standardisation")
		return nil, fmt.Errorf("internal error creating super() statement for standardisation: %w", err)
	}

	newBody := []js_ast.Stmt{superCallStmt}
	for _, statement := range constructor.Fn.Body.Block.Stmts {
		if isSuperCall(statement) || isSpecificInitInstanceCall(statement) {
			continue
		}
		newBody = append(newBody, statement)
	}

	constructor.Fn.Body.Block.Stmts = newBody
	ConstructorStandardisationCount.Add(ctx, 1)
	l.Trace("Standardised existing constructor")
	return constructor, nil
}

// findConnectedCallback searches for an existing connectedCallback method in a
// class.
//
// Takes classDecl (*js_ast.Class) which is the class to search within.
// Takes registry (*RegistryContext) which provides context for property lookup.
//
// Returns *js_ast.EFunction which is the method if found, or nil if not
// present.
func findConnectedCallback(classDecl *js_ast.Class, registry *RegistryContext) *js_ast.EFunction {
	for i := range classDecl.Properties {
		prop := &classDecl.Properties[i]
		if prop.Kind != js_ast.PropertyMethod {
			continue
		}

		keyName := getPropertyKeyName(prop.Key, registry)
		if keyName == "connectedCallback" {
			if jsFunction, ok := prop.ValueOrNil.Data.(*js_ast.EFunction); ok {
				return jsFunction
			}
		}
	}
	return nil
}

// getPropertyKeyName extracts the name from a property key expression.
//
// Takes key (js_ast.Expr) which is the property key expression to get the
// name from.
// Takes registry (*RegistryContext) which provides lookup for identifiers
// made by hand. If nil, uses the global registry instead.
//
// Returns string which is the property name, or an empty string if the key
// type is not supported.
func getPropertyKeyName(key js_ast.Expr, registry *RegistryContext) string {
	switch k := key.Data.(type) {
	case *js_ast.EString:
		return helpers.UTF16ToString(k.Value)
	case *js_ast.EIdentifier:
		if registry != nil {
			return registry.LookupIdentifierName(k)
		}
		return lookupIdentifierName(k)
	default:
		return ""
	}
}

// createConnectedCallback creates a new connectedCallback method for a class.
//
// Takes span (trace.Span) which records the operation status.
// Takes classDecl (*js_ast.Class) which receives the new callback property.
// Takes registry (*RegistryContext) which provides identifier creation.
//
// Returns *js_ast.EFunction which is the new callback function.
// Returns error when creation fails.
func createConnectedCallback(ctx context.Context, span trace.Span, classDecl *js_ast.Class, registry *RegistryContext) (*js_ast.EFunction, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("No connectedCallback found, creating new one.")

	ccb := &js_ast.EFunction{
		Fn: js_ast.Fn{
			Body: js_ast.FnBody{Block: js_ast.SBlock{Stmts: []js_ast.Stmt{}}},
		},
	}

	ccbProp := js_ast.Property{
		Key:        registry.MakeIdentifierExpr("connectedCallback"),
		ValueOrNil: js_ast.Expr{Data: ccb},
		Kind:       js_ast.PropertyMethod,
	}
	classDecl.Properties = append(classDecl.Properties, ccbProp)

	ConnectedCallbackCreationCount.Add(ctx, 1)
	l.Trace("Created new connectedCallback")
	span.SetStatus(codes.Ok, "Connected callback created successfully")
	return ccb, nil
}

// findConstructorMethod finds the constructor method within a class.
//
// Takes classDecl (*js_ast.Class) which is the class to search.
// Takes registry (*RegistryContext) which provides context for key lookup.
//
// Returns *js_ast.EFunction which is the constructor function, or nil if not
// found or classDecl is nil.
func findConstructorMethod(classDecl *js_ast.Class, registry *RegistryContext) *js_ast.EFunction {
	if classDecl == nil {
		return nil
	}
	for i := range classDecl.Properties {
		prop := &classDecl.Properties[i]
		if prop.Kind != js_ast.PropertyMethod {
			continue
		}

		if getPropertyKeyName(prop.Key, registry) == "constructor" {
			if jsFunction, ok := prop.ValueOrNil.Data.(*js_ast.EFunction); ok {
				return jsFunction
			}
		}
	}
	return nil
}

// parseSnippetAsStatement parses a code snippet and returns the first
// statement from the parsed AST.
//
// The function wraps the snippet in braces to form a block, then extracts the
// first statement. It adds a semicolon if the snippet does not end with one,
// a closing brace, or an opening brace.
//
// Takes snippet (string) which contains the JavaScript or TypeScript code to
// parse.
//
// Returns js_ast.Stmt which is the first parsed statement from the snippet.
// Returns error when parsing fails or the snippet produces no statements.
func parseSnippetAsStatement(snippet string) (js_ast.Stmt, error) {
	trimmedSnippet := strings.TrimSpace(snippet)
	if !strings.HasSuffix(trimmedSnippet, jsSemicolon) && !strings.HasSuffix(trimmedSnippet, jsCloseBrace) && !strings.HasSuffix(trimmedSnippet, jsOpenBrace) {
		trimmedSnippet += jsSemicolon
	}

	blockSnippet := fmt.Sprintf("{ %s }", trimmedSnippet)
	parser := NewTypeScriptParser()
	parsedAST, err := parser.ParseTypeScript(blockSnippet, "snippet.ts")

	if err != nil {
		return js_ast.Stmt{}, fmt.Errorf("parsing error for snippet block '%s': %w", blockSnippet, err)
	}
	if parsedAST == nil || len(parsedAST.Parts) == 0 {
		return js_ast.Stmt{}, fmt.Errorf("no top-level AST node found in snippet block: %q (original: %q)", blockSnippet, snippet)
	}

	var blockStmt *js_ast.SBlock
	for partIndex := range parsedAST.Parts {
		if len(parsedAST.Parts[partIndex].Stmts) > 0 {
			block, ok := parsedAST.Parts[partIndex].Stmts[0].Data.(*js_ast.SBlock)
			if ok && len(block.Stmts) > 0 {
				blockStmt = block
				break
			}
		}
	}

	if blockStmt == nil {
		return js_ast.Stmt{}, fmt.Errorf("expected a block statement with content from snippet: '%s' (original: %q)", snippet, snippet)
	}

	statement := blockStmt.Stmts[0]

	registerNamesFromSnippet(statement, parsedAST.Symbols)

	return statement, nil
}

// parseSnippetAsBlock parses a TypeScript snippet and returns all statements
// as a block. Use this when you need every statement from the snippet, not just
// the first one.
//
// Takes snippet (string) which contains the TypeScript code to parse.
//
// Returns *js_ast.SBlock which contains all parsed statements from the snippet.
// Returns error when parsing fails or no valid block statement is found.
func parseSnippetAsBlock(snippet string) (*js_ast.SBlock, error) {
	blockSnippet := prepareBlockSnippet(snippet)
	parser := NewTypeScriptParser()
	parsedAST, err := parser.ParseTypeScript(blockSnippet, "snippet.ts")

	if err != nil {
		return nil, fmt.Errorf("parsing error for snippet block '%s': %w", blockSnippet, err)
	}
	if parsedAST == nil || len(parsedAST.Parts) == 0 {
		return nil, fmt.Errorf("no top-level AST node found in snippet block: %q (original: %q)", blockSnippet, snippet)
	}

	block := findBlockInParts(parsedAST.Parts)
	if block == nil {
		return nil, fmt.Errorf("expected a block statement with content from snippet: '%s' (original: %q)", snippet, snippet)
	}

	for _, statement := range block.Stmts {
		registerNamesFromSnippet(statement, parsedAST.Symbols)
	}
	return block, nil
}

// prepareBlockSnippet wraps a code snippet in braces for block parsing.
//
// Takes snippet (string) which is the raw code to wrap.
//
// Returns string which is the snippet wrapped in braces. If the snippet does
// not end with a semicolon, closing brace, or opening brace, a semicolon is
// added before wrapping.
func prepareBlockSnippet(snippet string) string {
	trimmed := strings.TrimSpace(snippet)
	if !strings.HasSuffix(trimmed, jsSemicolon) && !strings.HasSuffix(trimmed, jsCloseBrace) && !strings.HasSuffix(trimmed, jsOpenBrace) {
		trimmed += jsSemicolon
	}
	return fmt.Sprintf("{ %s }", trimmed)
}

// findBlockInParts searches through AST parts to find the first block statement
// that contains statements.
//
// Takes parts ([]js_ast.Part) which contains the parsed JavaScript AST parts.
//
// Returns *js_ast.SBlock which is the first block with statements, or nil if
// none is found.
func findBlockInParts(parts []js_ast.Part) *js_ast.SBlock {
	for partIndex := range parts {
		if len(parts[partIndex].Stmts) == 0 {
			continue
		}
		block, ok := parts[partIndex].Stmts[0].Data.(*js_ast.SBlock)
		if ok && len(block.Stmts) > 0 {
			return block
		}
	}
	return nil
}

// parseModuleLevelStatement parses a module-level statement such as import or
// export. Unlike parseSnippetAsStatement, this does not wrap the snippet in a
// block, which is needed for import statements that cannot appear inside
// blocks.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes snippet (string) which contains the TypeScript statement to parse.
//
// Returns js_ast.Stmt which is the parsed statement.
// Returns *js_ast.AST which is the full AST for further processing.
// Returns error when parsing fails or no statement is found in the snippet.
func parseModuleLevelStatement(ctx context.Context, snippet string) (js_ast.Stmt, *js_ast.AST, error) {
	trimmedSnippet := strings.TrimSpace(snippet)

	parser := NewTypeScriptParser()
	parsedAST, err := parser.ParseTypeScript(trimmedSnippet, "module.ts")

	if err != nil {
		return js_ast.Stmt{}, nil, fmt.Errorf("parsing error for module statement '%s': %w", snippet, err)
	}
	if parsedAST == nil || len(parsedAST.Parts) == 0 {
		return js_ast.Stmt{}, nil, fmt.Errorf("no top-level AST node found in module statement: %q", snippet)
	}

	if statement, found := findStatementInParts(ctx, parsedAST); found {
		return statement, parsedAST, nil
	}

	if statement, found := buildImportFromRecords(parsedAST, trimmedSnippet); found {
		return statement, parsedAST, nil
	}

	return js_ast.Stmt{}, nil, fmt.Errorf("no statement found in module snippet: %q", snippet)
}

// findStatementInParts searches through all parts for any statement with data.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes parsedAST (*js_ast.AST) which contains the parsed JavaScript AST to
// search.
//
// Returns js_ast.Stmt which is the first statement found with non-nil data.
// Returns bool which indicates whether a statement was found.
func findStatementInParts(ctx context.Context, parsedAST *js_ast.AST) (js_ast.Stmt, bool) {
	_, l := logger_domain.From(ctx, log)
	for partIndex := range parsedAST.Parts {
		for statementIndex := range parsedAST.Parts[partIndex].Stmts {
			statement := parsedAST.Parts[partIndex].Stmts[statementIndex]
			if statement.Data == nil {
				continue
			}
			l.Trace("parseModuleLevelStatement found statement",
				logger_domain.Int("partIndex", partIndex),
				logger_domain.Int("statementIndex", statementIndex),
				logger_domain.String("stmtType", fmt.Sprintf("%T", statement.Data)))
			registerNamesFromSnippet(statement, parsedAST.Symbols)
			return statement, true
		}
	}
	return js_ast.Stmt{}, false
}

// buildImportFromRecords builds an import statement from import records when
// the import is not found in Parts.
//
// Takes parsedAST (*js_ast.AST) which provides the parsed AST with import
// records and symbols.
// Takes trimmedSnippet (string) which is the source snippet used to find
// default imports.
//
// Returns js_ast.Stmt which contains the built import statement.
// Returns bool which shows whether the import was built with success.
func buildImportFromRecords(parsedAST *js_ast.AST, trimmedSnippet string) (js_ast.Stmt, bool) {
	if len(parsedAST.ImportRecords) == 0 {
		return js_ast.Stmt{}, false
	}

	importSymbols := extractImportSymbols(parsedAST.Symbols)
	if len(importSymbols) == 0 {
		return js_ast.Stmt{}, false
	}

	if isNamespaceImportSnippet(trimmedSnippet) && len(importSymbols) == 1 {
		return buildNamespaceImportFromRecords(parsedAST.Symbols, importSymbols[0]), true
	}

	isDefaultImport := isDefaultImportSnippet(trimmedSnippet)
	aliasMap := parseImportAliases(trimmedSnippet)
	simport := buildSImportFromSymbols(parsedAST.Symbols, importSymbols, isDefaultImport, aliasMap)

	return js_ast.Stmt{Data: simport}, true
}

// parseImportAliases extracts alias mappings from an import statement string.
// For `import { add as addNumbers, foo }`, returns {"addNumbers": "add", "foo":
// "foo"}.
//
// Takes snippet (string) which is the import statement to parse.
//
// Returns map[string]string where key is local name and value is exported name.
func parseImportAliases(snippet string) map[string]string {
	aliases := make(map[string]string)

	startIndex := strings.Index(snippet, "{")
	endIndex := strings.Index(snippet, "}")
	if startIndex == -1 || endIndex == -1 || endIndex <= startIndex {
		return aliases
	}

	content := snippet[startIndex+1 : endIndex]

	for item := range strings.SplitSeq(content, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		const asKeywordPartsMinimum = 3
		parts := strings.Fields(item)
		if len(parts) >= asKeywordPartsMinimum && parts[1] == "as" {
			exportName := parts[0]
			localName := parts[2]
			aliases[localName] = exportName
		} else if len(parts) == 1 {
			aliases[parts[0]] = parts[0]
		}
	}

	return aliases
}

// extractImportSymbols filters symbols and returns only those with the
// SymbolImport kind.
//
// Takes symbols ([]ast.Symbol) which contains the symbols to filter.
//
// Returns []ast.Symbol which contains only import symbols.
func extractImportSymbols(symbols []ast.Symbol) []ast.Symbol {
	var importSymbols []ast.Symbol
	for _, symbol := range symbols {
		if symbol.Kind == ast.SymbolImport {
			importSymbols = append(importSymbols, symbol)
		}
	}
	return importSymbols
}

// isDefaultImportSnippet checks whether the given import snippet is a default
// import rather than a named import.
//
// Takes snippet (string) which is the import statement text to check.
//
// Returns bool which is true if the snippet is a default import.
func isDefaultImportSnippet(snippet string) bool {
	if isNamespaceImportSnippet(snippet) {
		return false
	}
	fromIndex := strings.Index(snippet, " from ")
	braceIndex := strings.Index(snippet, "{")
	return braceIndex == -1 || (fromIndex != -1 && braceIndex > fromIndex)
}

// isNamespaceImportSnippet checks whether the given import snippet uses
// namespace import syntax (import * as X from '...').
//
// Takes snippet (string) which is the import statement text to check.
//
// Returns bool which is true if the snippet is a namespace import.
func isNamespaceImportSnippet(snippet string) bool {
	return strings.Contains(snippet, "* as ")
}

// buildNamespaceImportFromRecords builds an SImport with namespace import
// fields set (StarNameLoc and NamespaceRef) from the given symbol.
//
// Takes allSymbols ([]ast.Symbol) which holds all symbols for index lookup.
// Takes namespaceSymbol (ast.Symbol) which is the namespace binding symbol.
//
// Returns js_ast.Stmt which contains the namespace import statement.
func buildNamespaceImportFromRecords(allSymbols []ast.Symbol, namespaceSymbol ast.Symbol) js_ast.Stmt {
	symbolIndex := findSymbolIndex(allSymbols, namespaceSymbol.OriginalName)
	loc := logger.Loc{}
	simport := &js_ast.SImport{
		StarNameLoc:       &loc,
		NamespaceRef:      ast.Ref{InnerIndex: safeconv.IntToUint32(symbolIndex)},
		ImportRecordIndex: 0,
		IsSingleLine:      true,
	}
	return js_ast.Stmt{Data: simport}
}

// buildSImportFromSymbols builds an SImport from the given import symbols.
//
// Takes allSymbols ([]ast.Symbol) which holds all symbols for index lookup.
// Takes importSymbols ([]ast.Symbol) which lists the symbols to import.
// Takes isDefault (bool) which shows whether this is a default import.
// Takes aliasMap (map[string]string) which maps local names to export names.
//
// Returns *js_ast.SImport which is the built import statement.
func buildSImportFromSymbols(allSymbols []ast.Symbol, importSymbols []ast.Symbol, isDefault bool, aliasMap map[string]string) *js_ast.SImport {
	simport := &js_ast.SImport{
		ImportRecordIndex: 0,
		IsSingleLine:      true,
	}

	if isDefault && len(importSymbols) == 1 {
		symbolName := importSymbols[0].OriginalName
		symbolIndex := findSymbolIndex(allSymbols, symbolName)
		simport.DefaultName = &ast.LocRef{
			Ref: ast.Ref{InnerIndex: safeconv.IntToUint32(symbolIndex)},
		}
	} else {
		items := make([]js_ast.ClauseItem, 0, len(importSymbols))
		for _, symbol := range importSymbols {
			localName := symbol.OriginalName
			exportName := localName
			if mappedExport, ok := aliasMap[localName]; ok {
				exportName = mappedExport
			}
			items = append(items, js_ast.ClauseItem{
				Alias:        exportName,
				OriginalName: localName,
			})
		}
		simport.Items = &items
	}

	return simport
}

// registerNamesFromSnippet extracts and registers names from a parsed
// statement. It walks the AST to find class names, function names, and other
// identifiers based on the statement type.
//
// Takes statement (js_ast.Stmt) which is the statement to extract names from.
// Takes symbols ([]ast.Symbol) which is the symbol table to register names
// into.
func registerNamesFromSnippet(statement js_ast.Stmt, symbols []ast.Symbol) {
	switch s := statement.Data.(type) {
	case *js_ast.SClass:
		registerClassNames(s, symbols)
	case *js_ast.SFunction:
		registerFunctionName(s, symbols)
	case *js_ast.SExpr:
		registerExprIdentifiers(s.Value, symbols)
	case *js_ast.SImport:
		registerImportNames(s, symbols)
	case *js_ast.SReturn:
		registerReturnStatement(s, symbols)
	case *js_ast.SBlock:
		registerBlockStatement(s, symbols)
	case *js_ast.SLocal:
		registerLocalStatement(s, symbols)
	case *js_ast.SIf:
		registerIfStatement(s, symbols)
	case *js_ast.SFor:
		registerForStatement(s, symbols)
	case *js_ast.SThrow:
		registerExprIdentifiers(s.Value, symbols)
	}
}

// registerReturnStatement records names from a return statement.
//
// Takes s (*js_ast.SReturn) which is the return statement to process.
// Takes symbols ([]ast.Symbol) which is the list to add names to.
func registerReturnStatement(s *js_ast.SReturn, symbols []ast.Symbol) {
	if s.ValueOrNil.Data != nil {
		registerExprIdentifiers(s.ValueOrNil, symbols)
	}
}

// registerBlockStatement records identifiers from a block statement.
//
// Takes s (*js_ast.SBlock) which contains the statements to process.
// Takes symbols ([]ast.Symbol) which receives the recorded identifiers.
func registerBlockStatement(s *js_ast.SBlock, symbols []ast.Symbol) {
	for _, innerStmt := range s.Stmts {
		registerNamesFromSnippet(innerStmt, symbols)
	}
}

// registerLocalStatement registers identifiers from a local variable
// statement (var, let, or const).
//
// Takes s (*js_ast.SLocal) which is the local statement to process.
// Takes symbols ([]ast.Symbol) which is the symbol table to add identifiers to.
func registerLocalStatement(s *js_ast.SLocal, symbols []ast.Symbol) {
	for _, declaration := range s.Decls {
		registerBindingIdentifiers(declaration.Binding, symbols)
		if declaration.ValueOrNil.Data != nil {
			registerExprIdentifiers(declaration.ValueOrNil, symbols)
		}
	}
}

// registerIfStatement gathers identifiers from an if statement.
//
// Takes s (*js_ast.SIf) which is the if statement to process.
// Takes symbols ([]ast.Symbol) which collects the found identifiers.
func registerIfStatement(s *js_ast.SIf, symbols []ast.Symbol) {
	registerExprIdentifiers(s.Test, symbols)
	registerNamesFromSnippet(s.Yes, symbols)
	if s.NoOrNil.Data != nil {
		registerNamesFromSnippet(s.NoOrNil, symbols)
	}
}

// registerForStatement records names found in a for statement.
//
// Takes s (*js_ast.SFor) which is the for statement to process.
// Takes symbols ([]ast.Symbol) which is the symbol table to add names to.
func registerForStatement(s *js_ast.SFor, symbols []ast.Symbol) {
	if s.InitOrNil.Data != nil {
		registerNamesFromSnippet(s.InitOrNil, symbols)
	}
	if s.TestOrNil.Data != nil {
		registerExprIdentifiers(s.TestOrNil, symbols)
	}
	if s.UpdateOrNil.Data != nil {
		registerExprIdentifiers(s.UpdateOrNil, symbols)
	}
	registerNamesFromSnippet(s.Body, symbols)
}

// registerClassNames records names from a class statement into the symbol
// table.
//
// Takes s (*js_ast.SClass) which provides the class statement to process.
// Takes symbols ([]ast.Symbol) which holds the symbol table for registration.
func registerClassNames(s *js_ast.SClass, symbols []ast.Symbol) {
	if s.Class.Name != nil {
		registerLocRefIfValid(s.Class.Name, symbols)
	}
	registerExprIdentifiers(s.Class.ExtendsOrNil, symbols)
	for _, prop := range s.Class.Properties {
		registerExprIdentifiers(prop.Key, symbols)
		registerExprIdentifiers(prop.ValueOrNil, symbols)
	}
}

// registerFunctionName registers the name from a function statement.
//
// Takes s (*js_ast.SFunction) which contains the function statement to check.
// Takes symbols ([]ast.Symbol) which provides the symbol table for lookup.
func registerFunctionName(s *js_ast.SFunction, symbols []ast.Symbol) {
	if s.Fn.Name != nil {
		registerLocRefIfValid(s.Fn.Name, symbols)
	}
}

// findSymbolIndex returns the index of a symbol with the given name.
//
// Takes symbols ([]ast.Symbol) which is the slice to search.
// Takes name (string) which is the symbol name to find.
//
// Returns int which is the index of the matching symbol, or -1 if not found.
func findSymbolIndex(symbols []ast.Symbol, name string) int {
	for i, symbol := range symbols {
		if symbol.OriginalName == name {
			return i
		}
	}
	return -1
}

// registerImportNames records names from an import statement in the symbol
// table.
//
// Takes s (*js_ast.SImport) which contains the import statement to process.
// Takes symbols ([]ast.Symbol) which holds the symbol table to update.
func registerImportNames(s *js_ast.SImport, symbols []ast.Symbol) {
	if s.DefaultName != nil {
		registerRefIfValid(s.DefaultName.Ref, symbols)
	}
	if s.Items != nil {
		for _, item := range *s.Items {
			registerRefIfValid(item.Name.Ref, symbols)
		}
	}
	if s.NamespaceRef.InnerIndex != 0 {
		registerRefIfValid(s.NamespaceRef, symbols)
	}
}

// registerLocRefIfValid registers a location reference name if its index is
// valid within the symbol table.
//
// Takes locRef (*ast.LocRef) which is the location reference to register.
// Takes symbols ([]ast.Symbol) which provides the symbol table for name lookup.
func registerLocRefIfValid(locRef *ast.LocRef, symbols []ast.Symbol) {
	index := int(locRef.Ref.InnerIndex)
	if index >= 0 && index < len(symbols) {
		registerLocRefName(locRef, symbols[index].OriginalName)
	}
}

// registerRefIfValid registers an identifier from a reference if the index is
// within bounds of the symbol table.
//
// Takes ref (ast.Ref) which contains the index to check.
// Takes symbols ([]ast.Symbol) which is the symbol table to look up.
func registerRefIfValid(ref ast.Ref, symbols []ast.Symbol) {
	index := int(ref.InnerIndex)
	if index >= 0 && index < len(symbols) {
		identifier := &js_ast.EIdentifier{Ref: ref}
		registerIdentifierName(identifier, symbols[index].OriginalName)
	}
}

// registerExprIdentifiers walks a JavaScript expression tree and records all
// identifiers it finds in the given symbol table.
//
// Takes expression (js_ast.Expr) which is the expression to walk.
// Takes symbols ([]ast.Symbol) which is the symbol table to
// record identifiers in.
func registerExprIdentifiers(expression js_ast.Expr, symbols []ast.Symbol) {
	if expression.Data == nil {
		return
	}

	switch e := expression.Data.(type) {
	case *js_ast.EIdentifier:
		registerIdentifierExpr(e, symbols)
	case *js_ast.ECall:
		registerCallExprIdents(e, symbols)
	case *js_ast.EDot:
		registerExprIdentifiers(e.Target, symbols)
	case *js_ast.EIndex:
		registerIndexExprIdents(e, symbols)
	case *js_ast.EBinary:
		registerBinaryExprIdents(e, symbols)
	case *js_ast.EUnary:
		registerExprIdentifiers(e.Value, symbols)
	case *js_ast.EArray:
		registerArrayExprIdents(e, symbols)
	case *js_ast.EObject:
		registerObjectExprIdents(e, symbols)
	case *js_ast.EArrow:
		registerArrowExprIdents(e, symbols)
	case *js_ast.EFunction:
		registerFunctionExprIdents(e, symbols)
	}
}

// registerIdentifierExpr records an identifier expression by looking up its
// original name in the symbol table.
//
// Takes e (*js_ast.EIdentifier) which is the identifier expression to record.
// Takes symbols ([]ast.Symbol) which is the symbol table for name lookup.
func registerIdentifierExpr(e *js_ast.EIdentifier, symbols []ast.Symbol) {
	index := int(e.Ref.InnerIndex)
	if index >= 0 && index < len(symbols) {
		registerIdentifierName(e, symbols[index].OriginalName)
	}
}

// registerCallExprIdents registers identifiers found in a call expression.
//
// Takes e (*js_ast.ECall) which is the call expression to process.
// Takes symbols ([]ast.Symbol) which is the list of symbols to register.
func registerCallExprIdents(e *js_ast.ECall, symbols []ast.Symbol) {
	registerExprIdentifiers(e.Target, symbols)
	for _, argument := range e.Args {
		registerExprIdentifiers(argument, symbols)
	}
}

// registerIndexExprIdents registers identifiers found in an index expression.
//
// Takes e (*js_ast.EIndex) which contains the target and index expressions.
// Takes symbols ([]ast.Symbol) which receives the registered identifiers.
func registerIndexExprIdents(e *js_ast.EIndex, symbols []ast.Symbol) {
	registerExprIdentifiers(e.Target, symbols)
	registerExprIdentifiers(e.Index, symbols)
}

// registerBinaryExprIdents registers identifiers found in a binary expression.
//
// Takes e (*js_ast.EBinary) which is the binary expression to process.
// Takes symbols ([]ast.Symbol) which is the list to add identifiers to.
func registerBinaryExprIdents(e *js_ast.EBinary, symbols []ast.Symbol) {
	registerExprIdentifiers(e.Left, symbols)
	registerExprIdentifiers(e.Right, symbols)
}

// registerArrayExprIdents registers identifiers found in an array expression.
//
// Takes e (*js_ast.EArray) which contains the array items to process.
// Takes symbols ([]ast.Symbol) which holds the symbols to register.
func registerArrayExprIdents(e *js_ast.EArray, symbols []ast.Symbol) {
	for _, item := range e.Items {
		registerExprIdentifiers(item, symbols)
	}
}

// registerObjectExprIdents scans an object expression and records any
// identifiers found within its properties.
//
// Takes e (*js_ast.EObject) which is the object expression to scan.
// Takes symbols ([]ast.Symbol) which collects the identifiers found.
func registerObjectExprIdents(e *js_ast.EObject, symbols []ast.Symbol) {
	for _, prop := range e.Properties {
		registerExprIdentifiers(prop.Key, symbols)
		registerExprIdentifiers(prop.ValueOrNil, symbols)
	}
}

// registerArrowExprIdents registers identifiers found in an arrow function.
//
// Takes e (*js_ast.EArrow) which is the arrow function to process.
// Takes symbols ([]ast.Symbol) which is the symbol table for registration.
func registerArrowExprIdents(e *js_ast.EArrow, symbols []ast.Symbol) {
	for _, argument := range e.Args {
		registerBindingIdentifiers(argument.Binding, symbols)
	}
	for _, statement := range e.Body.Block.Stmts {
		registerNamesFromSnippet(statement, symbols)
	}
}

// registerFunctionExprIdents registers identifiers found in a function
// expression. It scans the function arguments and body statements.
//
// Takes e (*js_ast.EFunction) which is the function expression to scan.
// Takes symbols ([]ast.Symbol) which collects the found identifiers.
func registerFunctionExprIdents(e *js_ast.EFunction, symbols []ast.Symbol) {
	for _, argument := range e.Fn.Args {
		registerBindingIdentifiers(argument.Binding, symbols)
	}
	for _, statement := range e.Fn.Body.Block.Stmts {
		registerNamesFromSnippet(statement, symbols)
	}
}

// registerBindingIdentifiers walks a binding structure recursively and
// registers each identifier it finds with its original name.
//
// Takes binding (js_ast.Binding) which is the binding to extract identifiers
// from.
// Takes symbols ([]ast.Symbol) which provides the symbol table for name lookup.
func registerBindingIdentifiers(binding js_ast.Binding, symbols []ast.Symbol) {
	if binding.Data == nil {
		return
	}

	switch b := binding.Data.(type) {
	case *js_ast.BIdentifier:
		index := int(b.Ref.InnerIndex)
		if index >= 0 && index < len(symbols) {
			registerBindingName(b, symbols[index].OriginalName)
		}
	case *js_ast.BArray:
		for _, item := range b.Items {
			registerBindingIdentifiers(item.Binding, symbols)
		}
	case *js_ast.BObject:
		for _, prop := range b.Properties {
			registerBindingIdentifiers(prop.Value, symbols)
		}
	}
}

// isSuperCall checks whether a statement is a super() constructor call.
//
// Takes statement (js_ast.Stmt) which is the statement to check.
//
// Returns bool which is true if the statement is a super() call.
func isSuperCall(statement js_ast.Stmt) bool {
	expressionStatement, ok := statement.Data.(*js_ast.SExpr)
	if !ok {
		return false
	}
	callExpr, ok := expressionStatement.Value.Data.(*js_ast.ECall)
	if !ok {
		return false
	}
	_, ok = callExpr.Target.Data.(*js_ast.ESuper)
	return ok
}

// getExprData extracts string data from a JavaScript AST
// expression.
//
// Takes expression (js_ast.Expr) which is the expression to
// extract data from.
//
// Returns []byte which contains the string data, or nil for
// unsupported expression types.
func getExprData(expression js_ast.Expr) []byte {
	switch e := expression.Data.(type) {
	case *js_ast.EString:
		return []byte(helpers.UTF16ToString(e.Value))
	case *js_ast.EIdentifier:
		return []byte("identifier")
	default:
		return nil
	}
}

// isSpecificInitInstanceCall reports whether the statement matches the pattern
// this.init(instance.call(this, this)).
//
// Takes statement (js_ast.Stmt) which is the statement to check.
//
// Returns bool which is true if the statement matches the pattern.
func isSpecificInitInstanceCall(statement js_ast.Stmt) bool {
	expressionStatement, ok := statement.Data.(*js_ast.SExpr)
	if !ok {
		return false
	}

	outerCallExpr, ok := expressionStatement.Value.Data.(*js_ast.ECall)
	if !ok {
		return false
	}

	outerDotExpr, ok := outerCallExpr.Target.Data.(*js_ast.EDot)
	if !ok {
		return false
	}

	if _, ok := outerDotExpr.Target.Data.(*js_ast.EThis); !ok {
		return false
	}

	if len(outerCallExpr.Args) != 1 {
		return false
	}

	innerCallExpr, ok := outerCallExpr.Args[0].Data.(*js_ast.ECall)
	if !ok {
		return false
	}

	_, ok = innerCallExpr.Target.Data.(*js_ast.EDot)
	if !ok {
		return false
	}

	if len(innerCallExpr.Args) != 2 {
		return false
	}

	_, ok1 := innerCallExpr.Args[0].Data.(*js_ast.EThis)
	_, ok2 := innerCallExpr.Args[1].Data.(*js_ast.EThis)

	return ok1 && ok2
}
