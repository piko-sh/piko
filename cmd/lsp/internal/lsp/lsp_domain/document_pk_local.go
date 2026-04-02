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
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/sfcparser"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// pkFileExtension is the file extension for Piko template files.
	pkFileExtension = ".pk"

	// fieldNamePadWidth is the width used to pad field names in type previews.
	fieldNamePadWidth = 20
)

// findLocalSymbolDefinition searches for any symbol (type, field, variable)
// in the original .pk script. This is the main entry point for .pk-local
// definition lookup.
//
// Takes symbolName (string) which is the name of the symbol to locate.
//
// Returns *protocol.Location which is the location of the symbol definition,
// or nil if the symbol is not found or the document is not a .pk file.
func (d *document) findLocalSymbolDefinition(ctx context.Context, symbolName string) *protocol.Location {
	_, l := logger_domain.From(ctx, log)

	if !strings.HasSuffix(d.URI.Filename(), pkFileExtension) {
		return nil
	}

	scriptResult, err := d.parseOriginalScriptBlock()
	if err != nil || scriptResult == nil {
		return nil
	}

	if typeLocation := findTypeDefinitionInAST(scriptResult.AST, scriptResult.Fset, symbolName); typeLocation != nil {
		adjustedLine := scriptResult.Offset.Line + typeLocation.Line - 1
		l.Trace("findLocalSymbolDefinition: Found as type definition",
			logger_domain.String("symbolName", symbolName),
			logger_domain.Int("line", adjustedLine),
			logger_domain.Int("column", typeLocation.Column))
		return buildProtocolLocation(d.URI, symbolName, adjustedLine, typeLocation.Column)
	}

	if fieldLocation := findAnyFieldDefinition(scriptResult.AST, scriptResult.Fset, symbolName); fieldLocation != nil {
		adjustedLine := scriptResult.Offset.Line + fieldLocation.Line - 1
		l.Trace("findLocalSymbolDefinition: Found as field definition",
			logger_domain.String("symbolName", symbolName),
			logger_domain.Int("line", adjustedLine),
			logger_domain.Int("column", fieldLocation.Column))
		return buildProtocolLocation(d.URI, symbolName, adjustedLine, fieldLocation.Column)
	}

	if funcLocation := findFunctionDefinitionInAST(scriptResult.AST, scriptResult.Fset, symbolName); funcLocation != nil {
		adjustedLine := scriptResult.Offset.Line + funcLocation.Line - 1
		l.Trace("findLocalSymbolDefinition: Found as function definition",
			logger_domain.String("symbolName", symbolName),
			logger_domain.Int("line", adjustedLine),
			logger_domain.Int("column", funcLocation.Column))
		return buildProtocolLocation(d.URI, symbolName, adjustedLine, funcLocation.Column)
	}

	return nil
}

// findLocalPKTypeDefinition attempts to find a type definition within the
// current .pk file's script block. This is necessary because the inspector
// analyses rewritten code with wrong positions, so we parse the original
// script content and find definitions directly.
//
// Takes typeName (string) which specifies the type to search for.
//
// Returns *protocol.Location which provides the adjusted position of the type
// definition, or nil if the type is not defined locally.
func (d *document) findLocalPKTypeDefinition(ctx context.Context, typeName string) *protocol.Location {
	_, l := logger_domain.From(ctx, log)

	if !strings.HasSuffix(d.URI.Filename(), pkFileExtension) {
		return nil
	}

	scriptResult, err := d.parseOriginalScriptBlock()
	if err != nil || scriptResult == nil {
		l.Trace("findLocalPKTypeDefinition: Could not parse script block",
			logger_domain.Error(err))
		return nil
	}

	location := findTypeDefinitionInAST(scriptResult.AST, scriptResult.Fset, typeName)
	if location == nil {
		return nil
	}

	adjustedLine := scriptResult.Offset.Line + location.Line - 1

	l.Trace("findLocalPKTypeDefinition: Found local type definition",
		logger_domain.String("typeName", typeName),
		logger_domain.Int("scriptLine", location.Line),
		logger_domain.Int("scriptColumn", location.Column),
		logger_domain.Int("realLine", adjustedLine),
		logger_domain.Int("realColumn", location.Column))

	return buildProtocolLocation(d.URI, typeName, adjustedLine, location.Column)
}

// scriptParseResult holds the result of parsing a .pk script block.
type scriptParseResult struct {
	// AST holds the parsed Go syntax tree for the script block.
	AST *ast.File

	// Fset is the file set used to find source positions in the AST.
	Fset *token.FileSet

	// Offset is the position in the source file where the parsed script begins.
	Offset *sfcparser.Location
}

// parseOriginalScriptBlock reads the current .pk file content and extracts
// the Go script block AST.
//
// Returns *scriptParseResult which contains the parsed AST, FileSet, and the
// script's start location offset within the .pk file.
// Returns error when the content is empty, SFC parsing fails, no Go script
// block is found, or the script content cannot be parsed as Go code.
func (d *document) parseOriginalScriptBlock() (*scriptParseResult, error) {
	if len(d.Content) == 0 {
		return nil, errors.New("no content available for .pk file")
	}

	sfcResult := d.getSFCResult()
	if sfcResult == nil {
		return nil, errors.New("failed to parse SFC")
	}

	goScript, found := sfcResult.GoScript()
	if !found || goScript.Content == "" {
		return nil, errors.New("no Go script block found")
	}

	fset := token.NewFileSet()
	scriptAST, err := parser.ParseFile(fset, "script.go", goScript.Content, parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("failed to parse script content: %w", err)
	}

	return &scriptParseResult{
		AST:    scriptAST,
		Fset:   fset,
		Offset: &goScript.ContentLocation,
	}, nil
}

// findStateTypeDefinition handles the special "state" identifier.
// It finds the Render function's first return type and jumps to that type
// definition.
//
// Returns *protocol.Location which points to the state type definition,
// or nil if the file is not a .pk file or the type cannot be found.
func (d *document) findStateTypeDefinition(ctx context.Context) *protocol.Location {
	_, l := logger_domain.From(ctx, log)

	if !strings.HasSuffix(d.URI.Filename(), pkFileExtension) {
		return nil
	}

	scriptResult, err := d.parseOriginalScriptBlock()
	if err != nil || scriptResult == nil {
		return nil
	}

	returnTypeName := findRenderReturnType(scriptResult.AST)
	if returnTypeName == "" {
		l.Trace("findStateTypeDefinition: Could not find Render return type")
		return nil
	}

	l.Trace("findStateTypeDefinition: Found Render return type",
		logger_domain.String("typeName", returnTypeName))

	return d.findLocalPKTypeDefinition(ctx, returnTypeName)
}

// typeDefLocation holds the location of a type or field definition within a
// parsed AST.
type typeDefLocation struct {
	// Name is the name of the type definition.
	Name string

	// Line is the 1-based line number within the script block.
	Line int

	// Column is the one-based column position of the symbol definition.
	Column int
}

// findTypeDefinitionInAST searches a Go AST for a type definition by name.
//
// Takes fileAST (*ast.File) which is the parsed Go source file to search.
// Takes fset (*token.FileSet) which provides position data for the AST.
// Takes typeName (string) which is the name of the type to find.
//
// Returns *typeDefLocation which contains the position of the type name in
// the AST, or nil if the type is not found.
func findTypeDefinitionInAST(fileAST *ast.File, fset *token.FileSet, typeName string) *typeDefLocation {
	for _, declaration := range fileAST.Decls {
		genDecl, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name == nil {
				continue
			}

			if typeSpec.Name.Name == typeName {
				position := fset.Position(typeSpec.Name.Pos())
				return &typeDefLocation{
					Line:   position.Line,
					Column: position.Column,
					Name:   typeSpec.Name.Name,
				}
			}
		}
	}
	return nil
}

// findFunctionDefinitionInAST searches for a function declaration with the
// given name in the file AST.
//
// Takes fileAST (*ast.File) which contains the parsed Go source file.
// Takes fset (*token.FileSet) which provides position data for the AST.
// Takes functionName (string) which is the name of the function to find.
//
// Returns *typeDefLocation which contains the position of the function name in
// the AST, or nil if the function is not found.
func findFunctionDefinitionInAST(fileAST *ast.File, fset *token.FileSet, functionName string) *typeDefLocation {
	for _, declaration := range fileAST.Decls {
		funcDecl, ok := declaration.(*ast.FuncDecl)
		if !ok || funcDecl.Name == nil {
			continue
		}

		if funcDecl.Name.Name == functionName {
			position := fset.Position(funcDecl.Name.Pos())
			return &typeDefLocation{
				Line:   position.Line,
				Column: position.Column,
				Name:   funcDecl.Name.Name,
			}
		}
	}
	return nil
}

// findAnyFieldDefinition searches all type definitions in a file for a field
// with the given name. Use it when the field name is known but not which
// type it belongs to.
//
// Takes fileAST (*ast.File) which contains the parsed Go source file.
// Takes fset (*token.FileSet) which provides position information.
// Takes fieldName (string) which specifies the field to search for.
//
// Returns *typeDefLocation which contains the field's location, or nil if not
// found.
func findAnyFieldDefinition(fileAST *ast.File, fset *token.FileSet, fieldName string) *typeDefLocation {
	for _, declaration := range fileAST.Decls {
		genDecl, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			if location := findFieldDefinitionInType(typeSpec, fset, fieldName); location != nil {
				return location
			}
		}
	}
	return nil
}

// findFieldDefinitionInType searches for a field within a given type.
//
// Takes typeSpec (*ast.TypeSpec) which defines the type to search in.
// Takes fset (*token.FileSet) which provides position data.
// Takes fieldName (string) which specifies the field name to find.
//
// Returns *typeDefLocation which contains the field's position, or nil if
// the type is not a struct or the field is not found.
func findFieldDefinitionInType(typeSpec *ast.TypeSpec, fset *token.FileSet, fieldName string) *typeDefLocation {
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok || structType.Fields == nil {
		return nil
	}

	for _, field := range structType.Fields.List {
		for _, fieldIdent := range field.Names {
			if fieldIdent.Name == fieldName {
				position := fset.Position(fieldIdent.Pos())
				return &typeDefLocation{
					Line:   position.Line,
					Column: position.Column,
					Name:   fieldIdent.Name,
				}
			}
		}
	}
	return nil
}

// findRenderReturnType finds the first return type of the Render function.
//
// Takes fileAST (*ast.File) which contains the parsed Go source file to search.
//
// Returns string which is the type name (e.g. "Response") or an empty string
// if no Render function is found or it has no return values.
func findRenderReturnType(fileAST *ast.File) string {
	for _, declaration := range fileAST.Decls {
		funcDecl, ok := declaration.(*ast.FuncDecl)
		if !ok || funcDecl.Name == nil || funcDecl.Name.Name != "Render" {
			continue
		}

		if funcDecl.Type == nil || funcDecl.Type.Results == nil || len(funcDecl.Type.Results.List) == 0 {
			return ""
		}

		firstResult := funcDecl.Type.Results.List[0]
		if identifier, ok := firstResult.Type.(*ast.Ident); ok {
			return identifier.Name
		}
	}
	return ""
}

// findRenderPropsType finds the props type from the Render function's second
// parameter.
//
// Takes fileAST (*ast.File) which contains the parsed Go source file to search.
//
// Returns *ast_domain.ResolvedTypeInfo which contains the resolved props type,
// or nil if the Render function is not found or has fewer than two parameters.
func findRenderPropsType(fileAST *ast.File) *ast_domain.ResolvedTypeInfo {
	for _, declaration := range fileAST.Decls {
		funcDecl, ok := declaration.(*ast.FuncDecl)
		if !ok || funcDecl.Name == nil || funcDecl.Name.Name != "Render" {
			continue
		}

		if funcDecl.Type == nil || funcDecl.Type.Params == nil || funcDecl.Type.Params.NumFields() < 2 {
			return nil
		}

		propsParam := funcDecl.Type.Params.List[1]
		return buildResolvedTypeFromExpr(propsParam.Type)
	}
	return nil
}

// buildResolvedTypeFromExpr turns an AST type expression into a
// ResolvedTypeInfo struct.
//
// Takes typeExpr (ast.Expr) which is the type expression to convert.
//
// Returns *ast_domain.ResolvedTypeInfo which holds the type expression and
// package alias if one exists, or nil if the expression is nil.
func buildResolvedTypeFromExpr(typeExpr ast.Expr) *ast_domain.ResolvedTypeInfo {
	if typeExpr == nil {
		return nil
	}

	switch t := typeExpr.(type) {
	case *ast.Ident:
		return &ast_domain.ResolvedTypeInfo{
			TypeExpression:       t,
			PackageAlias:         "",
			CanonicalPackagePath: "",
		}
	case *ast.SelectorExpr:
		pkgAlias := ""
		if identifier, ok := t.X.(*ast.Ident); ok {
			pkgAlias = identifier.Name
		}
		return &ast_domain.ResolvedTypeInfo{
			TypeExpression:       t,
			PackageAlias:         pkgAlias,
			CanonicalPackagePath: "",
		}
	case *ast.StarExpr:
		return buildResolvedTypeFromExpr(t.X)
	default:
		return &ast_domain.ResolvedTypeInfo{
			TypeExpression:       typeExpr,
			PackageAlias:         "",
			CanonicalPackagePath: "",
		}
	}
}

// formatTypeExpr converts an ast.Expr to its string form.
//
// Takes expression (ast.Expr) which is the type expression
// to format.
//
// Returns string which is the formatted type name, or "?"
// if the expression type is not recognised.
func formatTypeExpr(expression ast.Expr) string {
	switch t := expression.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if pkg, ok := t.X.(*ast.Ident); ok {
			return pkg.Name + "." + t.Sel.Name
		}
	case *ast.StarExpr:
		return "*" + formatTypeExpr(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + formatTypeExpr(t.Elt)
		}
		return "[...]" + formatTypeExpr(t.Elt)
	case *ast.MapType:
		return "map[" + formatTypeExpr(t.Key) + "]" + formatTypeExpr(t.Value)
	}
	return "?"
}

// getLocalTypePreview generates a struct preview from a .pk script's local type
// definition.
//
// Takes fileAST (*ast.File) which is the parsed Go source file to search.
// Takes typeName (string) which is the name of the type to preview.
// Takes maxFields (int) which limits how many fields to show.
//
// Returns string which is the formatted struct preview, or an empty
// string if the type is not found or is not a struct.
func getLocalTypePreview(fileAST *ast.File, _ *token.FileSet, typeName string, maxFields int) string {
	typeSpec := findTypeSpecByName(fileAST, typeName)
	if typeSpec == nil {
		return ""
	}

	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok || structType.Fields == nil {
		return ""
	}

	return formatStructPreview(typeName, structType, maxFields)
}

// findTypeSpecByName searches a parsed file for a type with the given name.
//
// Takes fileAST (*ast.File) which is the parsed file to search within.
// Takes typeName (string) which is the name of the type to find.
//
// Returns *ast.TypeSpec which is the matching type, or nil if not found.
func findTypeSpecByName(fileAST *ast.File, typeName string) *ast.TypeSpec {
	for _, declaration := range fileAST.Decls {
		genDecl, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name == nil {
				continue
			}

			if typeSpec.Name.Name == typeName {
				return typeSpec
			}
		}
	}
	return nil
}

// formatStructPreview formats a struct type as a preview string.
//
// The preview shows the struct definition with up to maxFields fields. If the
// struct has more fields than the limit, it adds a count of hidden fields.
//
// Takes typeName (string) which is the name of the struct type.
// Takes structType (*ast.StructType) which is the AST node for the struct.
// Takes maxFields (int) which limits how many fields to show.
//
// Returns string which is the formatted struct preview.
func formatStructPreview(typeName string, structType *ast.StructType, maxFields int) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "type %s struct {", typeName)

	fieldsShown := formatStructFields(&b, structType, maxFields)
	totalFields := countStructFields(structType)

	if totalFields > maxFields {
		_, _ = fmt.Fprintf(&b, "\n    ... (%d more fields)", totalFields-fieldsShown)
	}

	b.WriteString("\n}")
	return b.String()
}

// formatStructFields writes up to maxFields struct fields to the builder.
//
// Takes b (*strings.Builder) which receives the formatted field lines.
// Takes structType (*ast.StructType) which provides the fields to format.
// Takes maxFields (int) which limits how many fields are written.
//
// Returns int which is the number of fields actually written.
func formatStructFields(b *strings.Builder, structType *ast.StructType, maxFields int) int {
	fieldsShown := 0
	for _, field := range structType.Fields.List {
		for _, fieldName := range field.Names {
			if fieldsShown >= maxFields {
				return fieldsShown
			}
			b.WriteString("\n    ")
			b.WriteString(formatLocalFieldLine(field, fieldName.Name))
			fieldsShown++
		}
	}
	return fieldsShown
}

// countStructFields counts the total number of named fields in a struct type.
//
// Takes structType (*ast.StructType) which is the struct type to count.
//
// Returns int which is the count of named fields.
func countStructFields(structType *ast.StructType) int {
	total := 0
	for _, field := range structType.Fields.List {
		total += len(field.Names)
	}
	return total
}

// formatLocalFieldLine formats a struct field line for a .pk local AST type
// preview with proper alignment.
//
// Takes field (*ast.Field) which provides the field's type and optional tag.
// Takes fieldName (string) which specifies the field name to display.
//
// Returns string which contains the formatted field line with the name padded
// to a fixed width, followed by the type and tag if present.
func formatLocalFieldLine(field *ast.Field, fieldName string) string {
	paddedName := fieldName
	if len(paddedName) < fieldNamePadWidth {
		paddedName = paddedName + strings.Repeat(" ", fieldNamePadWidth-len(paddedName))
	}

	typeString := formatTypeExpr(field.Type)

	if field.Tag != nil {
		tag := field.Tag.Value
		if len(tag) >= 2 && tag[0] == '`' && tag[len(tag)-1] == '`' {
			tag = tag[1 : len(tag)-1]
		}
		return fmt.Sprintf("%s %s `%s`", paddedName, typeString, tag)
	}

	return fmt.Sprintf("%s %s", paddedName, typeString)
}

// buildProtocolLocation creates a protocol.Location for a symbol at the given
// position. The line and column are 1-based (as returned by token.Position).
//
// Takes uri (protocol.DocumentURI) which specifies the document containing the
// symbol.
// Takes symbolName (string) which determines the range length for the location.
// Takes line (int) which is the 1-based line number of the symbol.
// Takes column (int) which is the 1-based column number of the symbol.
//
// Returns *protocol.Location which contains the 0-based position range.
func buildProtocolLocation(uri protocol.DocumentURI, symbolName string, line, column int) *protocol.Location {
	return &protocol.Location{
		URI: uri,
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      safeconv.IntToUint32(line - 1),
				Character: safeconv.IntToUint32(column - 1),
			},
			End: protocol.Position{
				Line:      safeconv.IntToUint32(line - 1),
				Character: safeconv.IntToUint32(column - 1 + len(symbolName)),
			},
		},
	}
}
