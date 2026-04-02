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
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
)

// findPropsStruct finds the Props struct definition in a Go AST file.
//
// Takes file (*ast.File) which is the parsed Go source file to search.
//
// Returns *ast.TypeSpec which is the type specification for the Props struct.
// Returns *ast.StructType which is the struct type definition.
// Returns bool which shows whether the Props struct was found.
func findPropsStruct(file *ast.File) (*ast.TypeSpec, *ast.StructType, bool) {
	for _, declaration := range file.Decls {
		genDecl, ok := declaration.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != "Props" {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			return typeSpec, structType, true
		}
	}
	return nil, nil, false
}

// findFieldInStruct searches for a field by name in a struct type.
//
// Takes structType (*ast.StructType) which is the struct to search in.
// Takes fieldName (string) which is the name of the field to find.
//
// Returns *ast.Field which is the matching field, or nil if not found.
// Returns int which is the index of the field in the fields list, or -1 if
// not found.
// Returns bool which is true when the field was found.
func findFieldInStruct(structType *ast.StructType, fieldName string) (*ast.Field, int, bool) {
	for i, field := range structType.Fields.List {
		for _, name := range field.Names {
			if name.Name == fieldName {
				return field, i, true
			}
		}
	}
	return nil, -1, false
}

// addCoerceTagToField adds coerce:"true" to a struct field's tag,
// creating a new tag if none exists or appending to an existing one.
//
// Takes field (*ast.Field) which is the struct field to modify in
// place.
func addCoerceTagToField(field *ast.Field) {
	if field.Tag == nil {
		field.Tag = &ast.BasicLit{
			Kind:  token.STRING,
			Value: "`coerce:\"true\"`",
		}
		return
	}

	tagValue := field.Tag.Value
	if len(tagValue) < 2 {
		field.Tag.Value = "`coerce:\"true\"`"
		return
	}

	tagContent := tagValue[1 : len(tagValue)-1]

	if tagContent == "" {
		field.Tag.Value = "`coerce:\"true\"`"
	} else {
		field.Tag.Value = fmt.Sprintf("`%s coerce:\"true\"`", tagContent)
	}
}

// findImportDecl finds the import declaration in a Go AST file.
//
// Takes file (*ast.File) which is the parsed Go source file to search.
//
// Returns *ast.GenDecl which is the import declaration if found.
// Returns bool which is true when an import declaration was found.
func findImportDecl(file *ast.File) (*ast.GenDecl, bool) {
	for _, declaration := range file.Decls {
		genDecl, ok := declaration.(*ast.GenDecl)
		if ok && genDecl.Tok == token.IMPORT {
			return genDecl, true
		}
	}
	return nil, false
}

// addImportToAST adds an import statement to a Go AST file.
//
// If an import declaration exists, it appends to it. Otherwise, it creates
// a new one at the start of the declarations list.
//
// Takes file (*ast.File) which is the AST to modify.
// Takes alias (string) which is the optional import alias, or empty for none.
// Takes importPath (string) which is the import path to add.
func addImportToAST(file *ast.File, alias, importPath string) {
	importSpec := &ast.ImportSpec{
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: strconv.Quote(importPath),
		},
	}

	if alias != "" {
		importSpec.Name = &ast.Ident{Name: alias}
	}

	importDecl, found := findImportDecl(file)
	if found {
		importDecl.Specs = append(importDecl.Specs, importSpec)
	} else {
		newImportDecl := &ast.GenDecl{
			Tok:   token.IMPORT,
			Specs: []ast.Spec{importSpec},
		}

		file.Decls = append([]ast.Decl{newImportDecl}, file.Decls...)
	}
}
