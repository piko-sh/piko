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
	"path/filepath"
	"strconv"
	"strings"

	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/js_ast"
	"piko.sh/piko/internal/esbuild/js_parser"
	"piko.sh/piko/internal/esbuild/logger"
	"piko.sh/piko/internal/sfcparser"
)

// pkcFileExtension is the file extension for Piko Component files.
const pkcFileExtension = ".pkc"

// pkcStateProperty holds type and location information for a single state
// property in a PKC component's script block.
type pkcStateProperty struct {
	// Name is the property name as it appears in the source code.
	Name string

	// JSType is the JavaScript type: "string", "number", "boolean", "array",
	// "object", or "any".
	JSType string

	// ElementType specifies the type of elements for array properties.
	ElementType string

	// KeyType is the key type for Map<K,V> properties; empty for non-map types.
	KeyType string

	// ValueType is the value type for maps; for Map<K, V> this holds V.
	ValueType string

	// InitialValue holds the JavaScript expression as a string for use as the
	// default value.
	InitialValue string

	// Line is the 0-based line number of the property in the document.
	Line int

	// Column is the 0-based column of the property name in the document.
	Column int

	// IsNullable indicates whether this is a union type containing null or
	// undefined.
	IsNullable bool
}

// pkcFunction holds information about a top-level function in a PKC script
// block.
type pkcFunction struct {
	// Name is the function name.
	Name string

	// ParamNames holds the names of the function's parameters.
	ParamNames []string

	// Line is the 0-based line number of the function keyword in the document.
	Line int

	// Column is the 0-based column of the function name in the document.
	Column int

	// Exported indicates whether the function is exported.
	Exported bool
}

// pkcImport holds information about a JavaScript import statement in a PKC
// script block.
type pkcImport struct {
	// Path is the import path as written (e.g. "@/utils.js" or "./helpers.js").
	Path string

	// Alias is the local name for the import.
	Alias string

	// Line is the 0-based line number of the import in the document.
	Line int

	// Column is the 0-based column of the import path in the document.
	Column int
}

// pkcMetadata holds all introspection data extracted from a PKC file. This is
// cached on the document and lazily initialised on first access.
type pkcMetadata struct {
	// StateProperties maps property names to their type metadata.
	StateProperties map[string]*pkcStateProperty

	// Functions maps function names to their metadata.
	Functions map[string]*pkcFunction

	// CSSClasses maps class names to their definition positions.
	CSSClasses map[string]cssClassDefinition

	// Imports lists all import statements from the script block.
	Imports []pkcImport

	// ComponentName is the component tag name from <script name="...">.
	ComponentName string

	// LifecycleHooks lists detected lifecycle hooks (e.g. "onConnected").
	LifecycleHooks []string

	// Refs lists _ref="..." attribute values from the template.
	Refs []string
}

// isPKCFile reports whether this document is a .pkc component file.
//
// Returns bool which is true if the document URI ends with ".pkc".
func (d *document) isPKCFile() bool {
	return strings.HasSuffix(string(d.URI), pkcFileExtension)
}

// getPKCMetadata returns the cached PKC metadata, extracting it on first access.
// Returns nil if the file is not a PKC file or extraction fails.
//
// Returns *pkcMetadata which holds all extracted introspection data.
func (d *document) getPKCMetadata() *pkcMetadata {
	if !d.isPKCFile() {
		return nil
	}

	d.pkcOnce.Do(func() {
		sfcResult := d.getSFCResult()
		if sfcResult == nil {
			return
		}
		d.pkcMeta = extractPKCMetadata(d.URI.Filename(), sfcResult)
	})

	return d.pkcMeta
}

// extractPKCMetadata builds the full metadata from a parsed SFC result.
//
// Takes filename (string) which identifies the source file.
// Takes sfc (*sfcparser.ParseResult) which holds the parsed SFC blocks.
//
// Returns *pkcMetadata which holds all extracted metadata.
func extractPKCMetadata(filename string, sfc *sfcparser.ParseResult) *pkcMetadata {
	meta := &pkcMetadata{
		StateProperties: make(map[string]*pkcStateProperty),
		Functions:       make(map[string]*pkcFunction),
		CSSClasses:      make(map[string]cssClassDefinition),
	}

	extractPKCScriptMetadata(filename, sfc, meta)
	extractPKCStyleMetadata(sfc, meta)
	extractPKCTemplateMetadata(sfc, meta)

	return meta
}

// extractPKCScriptMetadata extracts state properties, functions, imports, and
// lifecycle hooks from the PKC script block.
//
// Takes filename (string) which identifies the source file being processed.
// Takes sfc (*sfcparser.ParseResult) which contains the parsed single-file
// component data.
// Takes meta (*pkcMetadata) which receives the extracted metadata.
func extractPKCScriptMetadata(filename string, sfc *sfcparser.ParseResult, meta *pkcMetadata) {
	clientScript, found := sfc.ClientScript()
	if !found || clientScript.Content == "" {
		return
	}

	if name, ok := sfc.TemplateAttributes["name"]; ok && name != "" {
		meta.ComponentName = name
	} else if filename != "" {
		base := filepath.Base(filename)
		meta.ComponentName = strings.TrimSuffix(base, filepath.Ext(base))
	}

	typeAssertions := compiler_domain.ExtractTypeAssertions(clientScript.Content)

	parseLog := logger.NewDeferLog(logger.DeferLogAll, nil)

	tree, ok := js_parser.Parse(
		parseLog,
		logger.Source{
			Index:          0,
			KeyPath:        logger.Path{Text: filename},
			PrettyPaths:    logger.PrettyPaths{Rel: filename, Abs: filename},
			Contents:       clientScript.Content,
			IdentifierName: filename,
		},
		parserOptions(),
	)

	if !ok {
		return
	}

	baseLineOffset := clientScript.ContentLocation.Line - 1
	baseColOffset := clientScript.ContentLocation.Column - 1

	extractPKCStateFromAST(&tree, typeAssertions, clientScript.Content, baseLineOffset, baseColOffset, meta)
	extractPKCFunctionsFromAST(&tree, clientScript.Content, baseLineOffset, baseColOffset, meta)
	extractPKCImportsFromAST(&tree, clientScript.Content, baseLineOffset, baseColOffset, meta)
	extractPKCLifecycleHooks(&tree, meta)
}

// extractPKCStateFromAST walks the AST to find the state object and extract
// property metadata with document-absolute locations.
//
// Takes tree (*js_ast.AST) which is the parsed JavaScript AST to search.
// Takes typeAssertions (map[string]compiler_domain.TypeAssertion) which maps
// property names to their type assertions.
// Takes source (string) which is the original source code for location data.
// Takes baseLineOffset (int) which is the line offset to add to positions.
// Takes baseColOffset (int) which is the column offset to add to positions.
// Takes meta (*pkcMetadata) which receives the extracted property metadata.
func extractPKCStateFromAST(
	tree *js_ast.AST,
	typeAssertions map[string]compiler_domain.TypeAssertion,
	source string,
	baseLineOffset int,
	baseColOffset int,
	meta *pkcMetadata,
) {
	for partIndex := range tree.Parts {
		for _, statement := range tree.Parts[partIndex].Stmts {
			local, ok := statement.Data.(*js_ast.SLocal)
			if !ok {
				continue
			}

			for _, declaration := range local.Decls {
				if !isStateBindingInAST(declaration.Binding, tree.Symbols) {
					continue
				}
				extractPKCStateProperties(declaration, tree, typeAssertions, source, baseLineOffset, baseColOffset, meta)
			}
		}
	}
}

// isStateBindingInAST checks whether a binding is the "state" identifier.
//
// Takes binding (js_ast.Binding) which is the AST binding to check.
// Takes symbols ([]ast.Symbol) which provides the symbol table for lookup.
//
// Returns bool which is true if the binding refers to a "state" identifier.
func isStateBindingInAST(binding js_ast.Binding, symbols []ast.Symbol) bool {
	identifier, ok := binding.Data.(*js_ast.BIdentifier)
	if !ok {
		return false
	}
	if int(identifier.Ref.InnerIndex) >= len(symbols) {
		return false
	}
	return symbols[identifier.Ref.InnerIndex].OriginalName == "state"
}

// extractPKCStateProperties extracts each property from the state object
// with its type and location.
//
// Takes declaration (js_ast.Decl) which is the declaration
// containing the state object.
// Takes tree (*js_ast.AST) which provides symbol information for name resolution.
// Takes typeAssertions (map[string]compiler_domain.TypeAssertion) which holds
// explicit type annotations for properties.
// Takes source (string) which is the source code for position calculation.
// Takes baseLineOffset (int) which is the line offset for absolute positioning.
// Takes baseColOffset (int) which is the column offset for absolute positioning.
// Takes meta (*pkcMetadata) which receives the extracted state properties.
func extractPKCStateProperties(
	declaration js_ast.Decl,
	tree *js_ast.AST,
	typeAssertions map[string]compiler_domain.TypeAssertion,
	source string,
	baseLineOffset int,
	baseColOffset int,
	meta *pkcMetadata,
) {
	if declaration.ValueOrNil.Data == nil {
		return
	}

	jsObject, ok := declaration.ValueOrNil.Data.(*js_ast.EObject)
	if !ok {
		return
	}

	for i := range jsObject.Properties {
		prop := &jsObject.Properties[i]
		propName := getPropertyNameFromExpr(prop.Key, tree.Symbols)
		if propName == "" {
			continue
		}

		relLine, relCol := byteOffsetToLineColumn(source, int(prop.Key.Loc.Start))
		absLine, absCol := adjustToDocumentPosition(relLine, relCol, baseLineOffset, baseColOffset)

		stateProp := &pkcStateProperty{
			Name:   propName,
			JSType: "any",
			Line:   absLine,
			Column: absCol,
		}

		if assertion, found := typeAssertions[propName]; found {
			parsed := compiler_domain.ParseTypeString(assertion.TypeString)
			stateProp.JSType = parsed.JSType
			stateProp.ElementType = parsed.ElementType
			stateProp.KeyType = parsed.KeyType
			stateProp.ValueType = parsed.ValueType
			stateProp.IsNullable = parsed.IsNullable
		} else {
			inferPKCPropertyType(prop.ValueOrNil, stateProp)
		}

		stateProp.InitialValue = expressionToInitialValue(prop.ValueOrNil)

		meta.StateProperties[propName] = stateProp
	}
}

// getPropertyNameFromExpr extracts the name string from a property key
// expression.
//
// Takes key (js_ast.Expr) which is the property key expression to extract
// from.
// Takes symbols ([]ast.Symbol) which provides symbol lookup for identifier
// references.
//
// Returns string which is the extracted property name, or empty string if the
// key type is not supported or the symbol index is out of bounds.
func getPropertyNameFromExpr(key js_ast.Expr, symbols []ast.Symbol) string {
	switch k := key.Data.(type) {
	case *js_ast.EString:
		return stringFromUTF16(k.Value)
	case *js_ast.EIdentifier:
		if int(k.Ref.InnerIndex) < len(symbols) {
			return symbols[k.Ref.InnerIndex].OriginalName
		}
	}
	return ""
}

// stringFromUTF16 converts a UTF-16 encoded slice to a Go string.
//
// Takes value ([]uint16) which contains the UTF-16 encoded code units.
//
// Returns string which is the decoded Go string.
func stringFromUTF16(value []uint16) string {
	runes := make([]rune, 0, len(value))
	for _, v := range value {
		runes = append(runes, rune(v))
	}
	return string(runes)
}

// inferPKCPropertyType infers the JavaScript type from a literal expression.
//
// Takes expression (js_ast.Expr) which is the expression to analyse
// for type info.
// Takes prop (*pkcStateProperty) which receives the inferred type.
func inferPKCPropertyType(expression js_ast.Expr, prop *pkcStateProperty) {
	if expression.Data == nil {
		return
	}

	switch expression.Data.(type) {
	case *js_ast.ENumber:
		prop.JSType = "number"
	case *js_ast.EString:
		prop.JSType = "string"
	case *js_ast.EBoolean:
		prop.JSType = "boolean"
	case *js_ast.EArray:
		prop.JSType = "array"
	case *js_ast.EObject:
		prop.JSType = "object"
	case *js_ast.ENull, *js_ast.EUndefined:
		prop.JSType = "any"
	}
}

// expressionToInitialValue converts an expression to a string representation.
//
// Takes expression (js_ast.Expr) which is the AST expression to
// convert.
//
// Returns string which is the string form of the expression, or
// empty if the expression cannot be represented.
func expressionToInitialValue(expression js_ast.Expr) string {
	if expression.Data == nil {
		return ""
	}

	switch v := expression.Data.(type) {
	case *js_ast.ENumber:
		if v.Value == float64(int64(v.Value)) {
			return strconv.FormatInt(int64(v.Value), 10)
		}
		return strconv.FormatFloat(v.Value, 'f', -1, 64)
	case *js_ast.EString:
		return `"` + stringFromUTF16(v.Value) + `"`
	case *js_ast.EBoolean:
		if v.Value {
			return "true"
		}
		return "false"
	case *js_ast.EArray:
		if len(v.Items) == 0 {
			return "[]"
		}
		return "[...]"
	case *js_ast.EObject:
		if len(v.Properties) == 0 {
			return "{}"
		}
		return "{...}"
	case *js_ast.ENull:
		return "null"
	case *js_ast.EUndefined:
		return "undefined"
	default:
		return ""
	}
}

// extractPKCFunctionsFromAST walks the AST to find function declarations and
// arrow/function expressions assigned to const/let/var.
//
// Takes tree (*js_ast.AST) which is the parsed JavaScript AST to traverse.
// Takes source (string) which is the original source code for position mapping.
// Takes baseLineOffset (int) which adjusts line numbers for embedded scripts.
// Takes baseColOffset (int) which adjusts column numbers for embedded scripts.
// Takes meta (*pkcMetadata) which collects the extracted function information.
func extractPKCFunctionsFromAST(
	tree *js_ast.AST,
	source string,
	baseLineOffset int,
	baseColOffset int,
	meta *pkcMetadata,
) {
	for partIndex := range tree.Parts {
		for _, statement := range tree.Parts[partIndex].Stmts {
			switch s := statement.Data.(type) {
			case *js_ast.SFunction:
				extractPKCRegularFunction(s, tree, source, baseLineOffset, baseColOffset, meta)
			case *js_ast.SLocal:
				extractPKCLocalFunctions(s, tree, source, baseLineOffset, baseColOffset, meta)
			case *js_ast.SExportDefault:
				extractPKCDefaultExportFunction(s, tree, source, baseLineOffset, baseColOffset, meta)
			}
		}
	}
}

// extractPKCRegularFunction extracts a named function declaration
// (e.g. `function foo() {}`).
//
// Takes funcDecl (*js_ast.SFunction) which is the function declaration node.
// Takes tree (*js_ast.AST) which provides access to the symbol table.
// Takes source (string) which is the original source code for position mapping.
// Takes baseLineOffset (int) which is the line offset for absolute positioning.
// Takes baseColOffset (int) which is the column offset for absolute positioning.
// Takes meta (*pkcMetadata) which stores the extracted function metadata.
func extractPKCRegularFunction(
	funcDecl *js_ast.SFunction,
	tree *js_ast.AST,
	source string,
	baseLineOffset int,
	baseColOffset int,
	meta *pkcMetadata,
) {
	if funcDecl.Fn.Name == nil {
		return
	}

	nameRef := funcDecl.Fn.Name.Ref
	if int(nameRef.InnerIndex) >= len(tree.Symbols) {
		return
	}

	name := tree.Symbols[nameRef.InnerIndex].OriginalName
	if name == "" {
		return
	}

	relLine, relCol := byteOffsetToLineColumn(source, int(funcDecl.Fn.Name.Loc.Start))
	absLine, absCol := adjustToDocumentPosition(relLine, relCol, baseLineOffset, baseColOffset)

	meta.Functions[name] = &pkcFunction{
		Name:       name,
		ParamNames: extractFunctionParamNames(funcDecl.Fn.Args, tree.Symbols),
		Line:       absLine,
		Column:     absCol,
		Exported:   funcDecl.IsExport,
	}
}

// extractPKCLocalFunctions extracts functions from local declarations such as
// `const foo = () => {}` or `const foo = function() {}`.
//
// Takes local (*js_ast.SLocal) which contains the local variable declarations.
// Takes tree (*js_ast.AST) which provides the AST symbols for name resolution.
// Takes source (string) which is the source code for position calculation.
// Takes baseLineOffset (int) which is the line offset in the document.
// Takes baseColOffset (int) which is the column offset in the document.
// Takes meta (*pkcMetadata) which stores the extracted function metadata.
func extractPKCLocalFunctions(
	local *js_ast.SLocal,
	tree *js_ast.AST,
	source string,
	baseLineOffset int,
	baseColOffset int,
	meta *pkcMetadata,
) {
	for _, declaration := range local.Decls {
		identifier, ok := declaration.Binding.Data.(*js_ast.BIdentifier)
		if !ok {
			continue
		}

		if int(identifier.Ref.InnerIndex) >= len(tree.Symbols) {
			continue
		}

		name := tree.Symbols[identifier.Ref.InnerIndex].OriginalName
		if name == "" || name == "state" {
			continue
		}

		if declaration.ValueOrNil.Data == nil {
			continue
		}

		var arguments []js_ast.Arg
		switch v := declaration.ValueOrNil.Data.(type) {
		case *js_ast.EArrow:
			arguments = v.Args
		case *js_ast.EFunction:
			arguments = v.Fn.Args
		default:
			continue
		}

		relLine, relCol := byteOffsetToLineColumn(source, int(declaration.Binding.Loc.Start))
		absLine, absCol := adjustToDocumentPosition(relLine, relCol, baseLineOffset, baseColOffset)

		meta.Functions[name] = &pkcFunction{
			Name:       name,
			ParamNames: extractFunctionParamNames(arguments, tree.Symbols),
			Line:       absLine,
			Column:     absCol,
			Exported:   local.IsExport,
		}
	}
}

// extractPKCDefaultExportFunction extracts a named function from a default
// export statement (e.g. `export default function foo() {}`).
//
// Takes exportDefault (*js_ast.SExportDefault) which is the default export
// statement to extract from.
// Takes tree (*js_ast.AST) which provides the AST containing symbol information.
// Takes source (string) which is the original source code for position
// calculation.
// Takes baseLineOffset (int) which is the line offset for absolute positioning.
// Takes baseColOffset (int) which is the column offset for absolute positioning.
// Takes meta (*pkcMetadata) which stores the extracted function metadata.
func extractPKCDefaultExportFunction(
	exportDefault *js_ast.SExportDefault,
	tree *js_ast.AST,
	source string,
	baseLineOffset int,
	baseColOffset int,
	meta *pkcMetadata,
) {
	fnStmt, ok := exportDefault.Value.Data.(*js_ast.SFunction)
	if !ok || fnStmt.Fn.Name == nil {
		return
	}

	nameRef := fnStmt.Fn.Name.Ref
	if int(nameRef.InnerIndex) >= len(tree.Symbols) {
		return
	}

	name := tree.Symbols[nameRef.InnerIndex].OriginalName
	if name == "" {
		return
	}

	relLine, relCol := byteOffsetToLineColumn(source, int(fnStmt.Fn.Name.Loc.Start))
	absLine, absCol := adjustToDocumentPosition(relLine, relCol, baseLineOffset, baseColOffset)

	meta.Functions[name] = &pkcFunction{
		Name:       name,
		ParamNames: extractFunctionParamNames(fnStmt.Fn.Args, tree.Symbols),
		Line:       absLine,
		Column:     absCol,
		Exported:   true,
	}
}

// extractFunctionParamNames extracts parameter names from function arguments.
//
// Takes arguments ([]js_ast.Arg) which contains the function arguments to process.
// Takes symbols ([]ast.Symbol) which provides the symbol table for name lookup.
//
// Returns []string which contains the original names of the parameters.
func extractFunctionParamNames(arguments []js_ast.Arg, symbols []ast.Symbol) []string {
	paramNames := make([]string, 0, len(arguments))
	for _, argument := range arguments {
		if identifier, ok := argument.Binding.Data.(*js_ast.BIdentifier); ok {
			if int(identifier.Ref.InnerIndex) < len(symbols) {
				paramNames = append(paramNames, symbols[identifier.Ref.InnerIndex].OriginalName)
			}
		}
	}
	return paramNames
}

// extractPKCImportsFromAST walks the AST to find import statements.
//
// Takes tree (*js_ast.AST) which is the parsed JavaScript AST to search.
// Takes source (string) which is the original source text for position lookup.
// Takes baseLineOffset (int) which is the line offset for absolute positions.
// Takes baseColOffset (int) which is the column offset for absolute positions.
// Takes meta (*pkcMetadata) which collects the discovered imports.
func extractPKCImportsFromAST(
	tree *js_ast.AST,
	source string,
	baseLineOffset int,
	baseColOffset int,
	meta *pkcMetadata,
) {
	for partIndex := range tree.Parts {
		for _, statement := range tree.Parts[partIndex].Stmts {
			if imp, ok := extractSinglePKCImport(tree, statement, source, baseLineOffset, baseColOffset); ok {
				meta.Imports = append(meta.Imports, imp)
			}
		}
	}
}

// extractSinglePKCImport attempts to convert a single JS statement
// into a pkcImport. It returns the import and true when the
// statement is a valid import declaration, or a zero value and
// false otherwise.
//
// Takes tree (*js_ast.AST) which provides the import records
// and symbol table.
// Takes statement (js_ast.Stmt) which is the statement to inspect.
// Takes source (string) which is the source text for position
// mapping.
// Takes baseLineOffset (int) which is the line offset to add
// to positions.
// Takes baseColOffset (int) which is the column offset to add
// to positions.
//
// Returns pkcImport which holds the extracted import data.
// Returns bool which is true if the statement was an import.
func extractSinglePKCImport(
	tree *js_ast.AST,
	statement js_ast.Stmt,
	source string,
	baseLineOffset int,
	baseColOffset int,
) (pkcImport, bool) {
	importStmt, ok := statement.Data.(*js_ast.SImport)
	if !ok {
		return pkcImport{}, false
	}

	if int(importStmt.ImportRecordIndex) >= len(tree.ImportRecords) {
		return pkcImport{}, false
	}

	record := tree.ImportRecords[importStmt.ImportRecordIndex]
	importPath := record.Path.Text

	relLine, relCol := byteOffsetToLineColumn(source, int(statement.Loc.Start))
	absLine, absCol := adjustToDocumentPosition(relLine, relCol, baseLineOffset, baseColOffset)

	alias := resolveImportAlias(importStmt, tree.Symbols)

	return pkcImport{
		Path:   importPath,
		Alias:  alias,
		Line:   absLine,
		Column: absCol,
	}, true
}

// resolveImportAlias determines the local alias name for an
// import statement by checking the default name or the first
// named import item.
//
// Takes importStmt (*js_ast.SImport) which is the import
// statement to inspect.
// Takes symbols ([]ast.Symbol) which provides the symbol table
// for name lookup.
//
// Returns string which is the local alias, or empty if none
// is found.
func resolveImportAlias(importStmt *js_ast.SImport, symbols []ast.Symbol) string {
	if importStmt.DefaultName != nil {
		if int(importStmt.DefaultName.Ref.InnerIndex) < len(symbols) {
			return symbols[importStmt.DefaultName.Ref.InnerIndex].OriginalName
		}
	} else if importStmt.Items != nil && len(*importStmt.Items) > 0 {
		return (*importStmt.Items)[0].Alias
	}
	return ""
}

// extractPKCLifecycleHooks detects lifecycle hook assignments in the AST.
// Looks for patterns like `this.onConnected = ...`.
//
// Takes tree (*js_ast.AST) which is the parsed JavaScript syntax tree to scan.
// Takes meta (*pkcMetadata) which receives the detected lifecycle hook names.
func extractPKCLifecycleHooks(tree *js_ast.AST, meta *pkcMetadata) {
	knownHooks := map[string]bool{
		"onConnected":    true,
		"onDisconnected": true,
		"onUpdated":      true,
		"getValue":       true,
		"getName":        true,
	}

	for partIndex := range tree.Parts {
		for _, statement := range tree.Parts[partIndex].Stmts {
			if name, ok := extractThisAssignmentName(statement); ok && knownHooks[name] {
				meta.LifecycleHooks = append(meta.LifecycleHooks, name)
			}
		}
	}
}

// extractThisAssignmentName returns the property name when a
// statement is an assignment of the form `this.<name> = ...`,
// along with true. It returns an empty string and false for
// all other statement shapes.
//
// Takes statement (js_ast.Stmt) which is the statement to inspect.
//
// Returns string which is the property name, or empty if
// the statement is not a this-assignment.
// Returns bool which is true when the statement matches
// the this-assignment pattern.
func extractThisAssignmentName(statement js_ast.Stmt) (string, bool) {
	expressionStatement, ok := statement.Data.(*js_ast.SExpr)
	if !ok {
		return "", false
	}

	assign, ok := expressionStatement.Value.Data.(*js_ast.EBinary)
	if !ok {
		return "", false
	}

	dot, ok := assign.Left.Data.(*js_ast.EDot)
	if !ok {
		return "", false
	}

	if _, isThis := dot.Target.Data.(*js_ast.EThis); !isThis {
		return "", false
	}

	return dot.Name, true
}

// extractPKCStyleMetadata scans style blocks for CSS class definitions.
//
// Takes sfc (*sfcparser.ParseResult) which contains the parsed SFC blocks.
// Takes meta (*pkcMetadata) which receives the extracted CSS class definitions.
func extractPKCStyleMetadata(sfc *sfcparser.ParseResult, meta *pkcMetadata) {
	for _, block := range sfc.Styles {
		if block.Content == "" {
			continue
		}
		collectClassDefinitions(block, meta.CSSClasses)
	}
}

// extractPKCTemplateMetadata scans the template for _ref attributes.
//
// Takes sfc (*sfcparser.ParseResult) which contains the parsed SFC template.
// Takes meta (*pkcMetadata) which stores the extracted ref names.
func extractPKCTemplateMetadata(sfc *sfcparser.ParseResult, meta *pkcMetadata) {
	if sfc.Template == "" {
		return
	}

	for _, prefix := range []string{`_ref="`, `p-ref="`} {
		index := 0
		for index < len(sfc.Template) {
			position := strings.Index(sfc.Template[index:], prefix)
			if position == -1 {
				break
			}

			valueStart := index + position + len(prefix)
			valueEnd := strings.IndexByte(sfc.Template[valueStart:], '"')
			if valueEnd == -1 {
				break
			}

			refName := sfc.Template[valueStart : valueStart+valueEnd]
			if refName != "" {
				meta.Refs = append(meta.Refs, refName)
			}

			index = valueStart + valueEnd + 1
		}
	}
}

// byteOffsetToLineColumn converts a 0-based byte offset in source text to a
// 0-based line and column.
//
// Takes source (string) which is the text to scan.
// Takes offset (int) which is the byte offset to convert.
//
// Returns int which is the 0-based line number.
// Returns int which is the 0-based column number.
func byteOffsetToLineColumn(source string, offset int) (line int, column int) {
	for i := 0; i < offset && i < len(source); i++ {
		if source[i] == '\n' {
			line++
			column = 0
		} else {
			column++
		}
	}

	return line, column
}

// adjustToDocumentPosition adds the script block's content offset to a
// relative position to produce a document-absolute position.
//
// Takes relLine (int) which is the 0-based line within the script content.
// Takes relCol (int) which is the 0-based column within the script content.
// Takes baseLineOffset (int) which is the 0-based start line of the script
// content in the document.
// Takes baseColOffset (int) which is the 0-based start column of the script
// content in the document.
//
// Returns int which is the 0-based line in the document.
// Returns int which is the 0-based column in the document.
func adjustToDocumentPosition(relLine, relCol, baseLineOffset, baseColOffset int) (absLine int, absCol int) {
	absLine = baseLineOffset + relLine
	absCol = relCol
	if relLine == 0 {
		absCol += baseColOffset
	}
	return absLine, absCol
}

// getPKCTypeString formats a pkcStateProperty's type as a human-readable string.
//
// Takes prop (*pkcStateProperty) which holds the type metadata to format.
//
// Returns string which is the formatted type (e.g. "boolean", "Array<string>",
// "Map<string, number>").
func getPKCTypeString(prop *pkcStateProperty) string {
	switch prop.JSType {
	case "array":
		if prop.ElementType != "" {
			return "Array<" + prop.ElementType + ">"
		}
		return "Array"
	case "object":
		if prop.KeyType != "" && prop.ValueType != "" {
			return "Map<" + prop.KeyType + ", " + prop.ValueType + ">"
		}
		return "Object"
	default:
		return prop.JSType
	}
}
