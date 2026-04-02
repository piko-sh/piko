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

package wasm_domain

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"piko.sh/piko/internal/inspector/inspector_dto"
)

// hoverTarget represents an identifier found at a cursor position.
type hoverTarget struct {
	// identifier is the identifier under the cursor; always set.
	identifier *ast.Ident

	// selector holds the selector expression when hovering over a qualified
	// identifier such as time.Time; nil if hovering over a simple identifier.
	selector *ast.SelectorExpr
}

// findHoverTarget finds the hover target at the given position.
// It detects both simple identifiers and selector expressions.
//
// Takes file (*ast.File) which contains the parsed source to search.
// Takes position (token.Pos) which specifies the position to find the target at.
//
// Returns *hoverTarget which is the found target, or nil if no target exists
// at the position.
func findHoverTarget(file *ast.File, position token.Pos) *hoverTarget {
	var target *hoverTarget

	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return false
		}

		if t := matchSelectorExpr(n, position); t != nil {
			target = t
			return false
		}

		if t := matchSimpleIdent(n, position, target); t != nil {
			target = t
			return false
		}

		return true
	})

	return target
}

// matchSelectorExpr checks if a node is a selector expression at the given
// position.
//
// Takes n (ast.Node) which is the node to check.
// Takes position (token.Pos) which is the position to match against.
//
// Returns *hoverTarget which contains the matched identifier and selector, or
// nil if the node is not a selector expression or the position does not fall
// within it.
func matchSelectorExpr(n ast.Node, position token.Pos) *hoverTarget {
	selectorExpression, ok := n.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	if selectorExpression.Sel.Pos() <= position && position <= selectorExpression.Sel.End() {
		return &hoverTarget{
			identifier: selectorExpression.Sel,
			selector:   selectorExpression,
		}
	}

	if xIdent, ok := selectorExpression.X.(*ast.Ident); ok {
		if xIdent.Pos() <= position && position <= xIdent.End() {
			return &hoverTarget{
				identifier: xIdent,
				selector:   nil,
			}
		}
	}

	return nil
}

// matchSimpleIdent checks if the node is an identifier at the given position.
//
// Takes n (ast.Node) which is the node to check.
// Takes position (token.Pos) which is the position to match against.
// Takes existing (*hoverTarget) which must be nil for a match to be returned.
//
// Returns *hoverTarget which holds the matched identifier, or nil if the node
// is not an identifier, the position is outside the identifier, or existing is
// not nil.
func matchSimpleIdent(n ast.Node, position token.Pos, existing *hoverTarget) *hoverTarget {
	identifier, ok := n.(*ast.Ident)
	if !ok {
		return nil
	}

	if identifier.Pos() <= position && position <= identifier.End() && existing == nil {
		return &hoverTarget{
			identifier: identifier,
			selector:   nil,
		}
	}

	return nil
}

// buildImportMap builds a map from import alias or name to package path.
// For example: {"fmt": "fmt", "t": "time"} where "t" is an alias for "time".
//
// Takes file (*ast.File) which contains the parsed Go source file.
//
// Returns map[string]string which maps import names to their full package paths.
func buildImportMap(file *ast.File) map[string]string {
	imports := make(map[string]string)

	for _, imp := range file.Imports {
		if imp.Path == nil {
			continue
		}

		packagePath := strings.Trim(imp.Path.Value, `"`)

		var name string
		if imp.Name != nil {
			name = imp.Name.Name
			if name == "_" || name == "." {
				continue
			}
		} else {
			parts := strings.Split(packagePath, "/")
			name = parts[len(parts)-1]
		}

		imports[name] = packagePath
	}

	return imports
}

// getQualifiedHoverContent returns hover content for a qualified name
// (pkg.Name).
//
// Takes pkgAlias (string) which is the package alias used in the source file.
// Takes name (string) which is the symbol name within that package.
// Takes file (*ast.File) which provides the import declarations for resolving
// the alias.
// Takes stdlibData (*inspector_dto.TypeData) which contains type and function
// definitions for standard library packages.
//
// Returns string which contains the formatted hover content, or an empty
// string when the package or symbol cannot be found.
func getQualifiedHoverContent(pkgAlias, name string, file *ast.File, stdlibData *inspector_dto.TypeData) string {
	if stdlibData == nil {
		return ""
	}

	imports := buildImportMap(file)
	packagePath, ok := imports[pkgAlias]
	if !ok {
		return ""
	}

	inspectedPackage, ok := stdlibData.Packages[packagePath]
	if !ok {
		return ""
	}

	if typ, ok := inspectedPackage.NamedTypes[name]; ok {
		return formatTypeHover(packagePath, typ)
	}

	if inspectedFunction, ok := inspectedPackage.Funcs[name]; ok {
		return formatFunctionHover(packagePath, inspectedFunction)
	}

	return ""
}

// getPackageHoverContent returns hover content for a package name.
//
// Takes pkgAlias (string) which is the package alias used in the import.
// Takes file (*ast.File) which provides the AST to extract import mappings.
// Takes stdlibData (*inspector_dto.TypeData) which contains standard library
// type information.
//
// Returns string which contains the formatted hover content, or an empty
// string when the package cannot be found.
func getPackageHoverContent(pkgAlias string, file *ast.File, stdlibData *inspector_dto.TypeData) string {
	if stdlibData == nil {
		return ""
	}

	imports := buildImportMap(file)
	packagePath, ok := imports[pkgAlias]
	if !ok {
		return ""
	}

	inspectedPackage, ok := stdlibData.Packages[packagePath]
	if !ok {
		return fmt.Sprintf("```go\npackage %s // %s\n```", pkgAlias, packagePath)
	}

	typeCount := len(inspectedPackage.NamedTypes)
	funcCount := len(inspectedPackage.Funcs)

	return fmt.Sprintf("```go\npackage %s // %s\n```\n\n%d types, %d functions",
		inspectedPackage.Name, packagePath, typeCount, funcCount)
}

// formatTypeHover builds hover content for a type.
//
// Takes packagePath (string) which specifies the package path for display.
// Takes typ (*inspector_dto.Type) which provides the type information.
//
// Returns string which contains the formatted hover content with the type
// definition and method count.
func formatTypeHover(packagePath string, typ *inspector_dto.Type) string {
	var builder strings.Builder
	builder.WriteString("```go\n")
	_, _ = fmt.Fprintf(&builder, "// %s\n", packagePath)
	_, _ = fmt.Fprintf(&builder, "type %s %s\n", typ.Name, typ.UnderlyingTypeString)
	builder.WriteString("```")

	if len(typ.Methods) > 0 {
		_, _ = fmt.Fprintf(&builder, "\n\n%d methods", len(typ.Methods))
	}

	return builder.String()
}

// formatFunctionHover formats hover content for a function.
//
// Takes packagePath (string) which specifies the package import path.
// Takes inspectedFunction (*inspector_dto.Function) which provides
// the function details.
//
// Returns string which contains the formatted hover content as a Markdown
// code block.
func formatFunctionHover(packagePath string, inspectedFunction *inspector_dto.Function) string {
	var builder strings.Builder
	builder.WriteString("```go\n")
	_, _ = fmt.Fprintf(&builder, "// %s\n", packagePath)
	_, _ = fmt.Fprintf(&builder, "func %s%s\n", inspectedFunction.Name, inspectedFunction.TypeString)
	builder.WriteString("```")
	return builder.String()
}

// getHoverContent returns hover content for a simple identifier.
//
// Takes name (string) which specifies the identifier to look up.
// Takes file (*ast.File) which provides the AST for local type and function
// lookup.
// Takes stdlibData (*inspector_dto.TypeData) which provides standard library
// type information.
//
// Returns string which contains the hover content, or an empty string if no
// content is found.
func getHoverContent(name string, file *ast.File, stdlibData *inspector_dto.TypeData) string {
	if content := findLocalTypeContent(name, file); content != "" {
		return content
	}

	if content := findLocalFunctionContent(name, file); content != "" {
		return content
	}

	if content := getPackageHoverContent(name, file, stdlibData); content != "" {
		return content
	}

	return findStdlibTypeContent(name, stdlibData)
}

// findLocalTypeContent searches for a local type declaration with the given
// name in an AST file.
//
// Takes name (string) which specifies the type name to find.
// Takes file (*ast.File) which provides the AST to search.
//
// Returns string which contains a Markdown code block with the type
// declaration, or an empty string if not found.
func findLocalTypeContent(name string, file *ast.File) string {
	for _, declaration := range file.Decls {
		genDecl, ok := declaration.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if typeSpec.Name.Name == name {
				return fmt.Sprintf("```go\ntype %s %s\n```", name, "...")
			}
		}
	}
	return ""
}

// findLocalFunctionContent searches for a local function declaration with the
// given name.
//
// Takes name (string) which specifies the function name to find.
// Takes file (*ast.File) which provides the parsed AST to search.
//
// Returns string which contains a formatted code block with the function
// signature if found, or an empty string if no match exists.
func findLocalFunctionContent(name string, file *ast.File) string {
	for _, declaration := range file.Decls {
		funcDecl, ok := declaration.(*ast.FuncDecl)
		if !ok || funcDecl.Recv != nil {
			continue
		}
		if funcDecl.Name.Name == name {
			return fmt.Sprintf("```go\nfunc %s(...)\n```", name)
		}
	}
	return ""
}

// findStdlibTypeContent searches stdlib packages for a type matching the name.
//
// Takes name (string) which specifies the type name to search for.
// Takes stdlibData (*inspector_dto.TypeData) which provides the stdlib type
// data to search through.
//
// Returns string which contains the formatted hover content for the matching
// type, or an empty string if stdlibData is nil or no match is found.
func findStdlibTypeContent(name string, stdlibData *inspector_dto.TypeData) string {
	if stdlibData == nil {
		return ""
	}
	for packagePath, inspectedPackage := range stdlibData.Packages {
		if typ, ok := inspectedPackage.NamedTypes[name]; ok {
			return formatTypeHover(packagePath, typ)
		}
	}
	return ""
}
